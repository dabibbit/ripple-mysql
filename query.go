package mysql

import (
	"database/sql"
	"fmt"
	"github.com/rubblelabs/ripple/data"
	"github.com/rubblelabs/ripple/storage"
	"math"
	"strconv"
	"strings"
	"time"
)

var txTypes = map[string]data.TransactionType{
	"payment":       data.PAYMENT,
	"accountset":    data.ACCOUNT_SET,
	"setregularkey": data.SET_REGULAR_KEY,
	"offercreate":   data.OFFER_CREATE,
	"offercancel":   data.OFFER_CANCEL,
	"trustset":      data.TRUST_SET,
	"amendment":     data.AMENDMENT,
	"setfee":        data.SET_FEE,
}

type LedgerQuery struct {
	Hash      *data.Hash256 `json:",omitempty"`
	Ledger    *uint32       `json:",omitempty"`
	MinLedger *uint32       `json:",omitempty"`
	MaxLedger *uint32       `json:",omitempty"`
}

type TransactionQuery struct {
	*LedgerQuery
	Account         *data.Account         `json:",omitempty"`
	AccountId       *uint32               `json:",omitempty"`
	DestinationId   *uint32               `json:",omitempty"`
	TransactionType *data.TransactionType `json:",omitempty"`
	Limit           uint32                `json:",omitempty"`
}

type TransactionRow struct {
	*data.TransactionWithMetaData
	Account   uint32
	CloseTime data.RippleTime
}

type QueryExecution struct {
	Time      time.Duration
	Statement string
	Params    []interface{}
}

type QueryResult struct {
	Query        TransactionQuery  `json:",omitempty"`
	Ledgers      []*data.Ledger    `json:",omitempty"`
	Transactions []*TransactionRow `json:",omitempty"`
	First, Last  uint32
	Queries      []QueryExecution
}

func (result *QueryResult) ExecuteQuery(tx *sql.Tx, query string, params []interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := tx.Query(query, params...)
	result.Queries = append(result.Queries, QueryExecution{time.Since(start), query, params})
	return rows, err
}

func (q QueryResult) MinLedger() uint32 {
	switch {
	case len(q.Transactions) == 0 && len(q.Ledgers) == 0:
		return 0
	case len(q.Ledgers) > 0:
		return q.Ledgers[0].LedgerSequence
	default:
		return q.Transactions[0].LedgerSequence
	}
}

func (q QueryResult) MaxLedger() uint32 {
	switch {
	case len(q.Transactions) == 0 && len(q.Ledgers) == 0:
		return 0
	case len(q.Ledgers) > 0:
		return q.Ledgers[len(q.Ledgers)-1].LedgerSequence
	default:
		return q.Transactions[len(q.Transactions)-1].LedgerSequence
	}
}

func (q QueryResult) Previous() uint32 {
	return q.MinLedger() - 1
}

func (q QueryResult) Next() uint32 {
	return q.MaxLedger() + 1
}

func (q QueryResult) Empty() bool {
	return len(q.Ledgers) == 0 && len(q.Transactions) == 0
}

func (q *LedgerQuery) Clone() *LedgerQuery {
	return &LedgerQuery{
		Hash:      q.Hash,
		MinLedger: q.MinLedger,
		MaxLedger: q.MaxLedger,
	}
}

func (q *TransactionQuery) Clone() *TransactionQuery {
	return &TransactionQuery{
		LedgerQuery:     q.LedgerQuery.Clone(),
		AccountId:       q.AccountId,
		DestinationId:   q.DestinationId,
		TransactionType: q.TransactionType,
		Limit:           q.Limit,
	}
}

func NewTransactionQuery(db IndexedDB, params map[string]string) (*TransactionQuery, error) {
	var err error
	q := &TransactionQuery{
		LedgerQuery: &LedgerQuery{},
		Limit:       100,
	}
	if l, ok := params["Ledger"]; ok {
		ledger, err := strconv.ParseUint(l, 10, 64)
		if err != nil {
			return nil, err
		}
		v := uint32(ledger)
		q.Ledger = &v
		q.Limit = math.MaxUint32
	}
	if min, ok := params["MinLedger"]; ok {
		minLedger, err := strconv.ParseUint(min, 10, 64)
		if err != nil {
			return nil, err
		}
		v := uint32(minLedger)
		q.MinLedger = &v
	}
	if max, ok := params["MaxLedger"]; ok {
		maxLedger, err := strconv.ParseUint(max, 10, 64)
		if err != nil {
			return nil, err
		}
		v := uint32(maxLedger)
		q.MaxLedger = &v
	}
	if txType, ok := txTypes[strings.ToLower(params["TransactionType"])]; ok {
		q.TransactionType = &txType
	}
	if account, ok := params["Account"]; ok {
		q.Account, err = data.NewAccountFromAddress(account)
		if err != nil {
			return nil, fmt.Errorf("Bad Account: %s", account)
		}
		if accountId, err := db.LookupAccount(q.Account); err != nil {
			return nil, fmt.Errorf("Account does not exist: %s", account)
		} else {
			q.AccountId = &accountId
			// q.DestinationId = &accountId
		}
	}
	if hash, ok := params["Hash"]; ok {
		if q.Hash, err = data.NewHash256(hash); err != nil {
			return nil, err
		}
	}
	return q, nil
}

func getLedgerRange(tx *sql.Tx, result *QueryResult) error {
	if result.First == 0 && result.Last == 0 {
		return tx.QueryRow(queries["GetLedgerRange"]).Scan(&result.First, &result.Last)
	}
	return nil
}

func (q *LedgerQuery) Rows(tx *sql.Tx, result *QueryResult) error {
	result.Query.LedgerQuery = q.Clone()
	if err := getLedgerRange(tx, result); err != nil {
		return err
	}
	var predicates []interface{}
	subQuery := `SELECT * FROM Ledger WHERE `
	switch {
	case q.Hash != nil:
		subQuery += `Hash=? `
		predicates = append(predicates, q.Hash.Bytes())
	case q.Ledger != nil:
		subQuery += `LedgerSequence=? ORDER BY LedgerSequence`
		predicates = append(predicates, q.Ledger)
	case q.MinLedger != nil:
		subQuery += `LedgerSequence>=? ORDER BY LedgerSequence`
		predicates = append(predicates, q.MinLedger)
	case q.MaxLedger != nil:
		subQuery += `LedgerSequence<=? ORDER BY LedgerSequence DESC`
		predicates = append(predicates, q.MaxLedger)
	default:
		return fmt.Errorf("Invalid Query: %+v", q)
	}
	sql := fmt.Sprintf("SELECT * FROM (%s LIMIT 10)l ORDER BY LedgerSequence;", subQuery)
	rows, err := result.ExecuteQuery(tx, sql, predicates)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var ledger data.Ledger
		if err := rows.Scan(LedgerColumns(&ledger)...); err != nil {
			return err
		}
		result.Ledgers = append(result.Ledgers, &ledger)
	}
	switch {
	case rows.Err() != nil:
		return rows.Err()
	case len(result.Ledgers) == 0:
		return storage.ErrNotFound
	default:
		return nil
	}
}

func (q *TransactionQuery) Where() (string, string, []interface{}) {
	var (
		where      []string
		predicates []interface{}
		order      string
	)
	if q.Hash != nil {
		where = append(where, `Hash=?`)
		predicates = append(predicates, q.Hash.Bytes())
	}
	if q.Ledger != nil {
		where = append(where, `LedgerSequence=?`)
		predicates = append(predicates, q.Ledger)
	}
	if q.MinLedger != nil {
		where = append(where, `LedgerSequence>=?`)
		predicates = append(predicates, q.MinLedger)
		order = "ORDER BY LedgerSequence,TransactionIndex"
	}
	if q.MaxLedger != nil {
		where = append(where, `LedgerSequence<=?`)
		predicates = append(predicates, q.MaxLedger)
		order = "ORDER BY LedgerSequence DESC,TransactionIndex DESC"
	}
	if q.TransactionType != nil {
		where = append(where, `TransactionType=?`)
		predicates = append(predicates, q.TransactionType)
	}
	if q.AccountId != nil {
		where = append(where, `Account=?`)
		predicates = append(predicates, q.AccountId)
	}
	return strings.Join(where, " AND "), order, predicates
}

func (q *TransactionQuery) Rows(tx *sql.Tx, result *QueryResult) error {
	result.Query = *q
	if err := getLedgerRange(tx, result); err != nil {
		return err
	}
	var (
		txQueries []*TransactionQuery
	)
	where, order, predicates := q.Where()
	subQuery := fmt.Sprintf("SELECT LedgerSequence,TransactionIndex,TransactionType FROM Transaction WHERE %s %s", where, order)
	ranges := fmt.Sprintf("SELECT TransactionType,MIN(LedgerSequence),MAX(LedgerSequence) FROM (%s LIMIT ?)t GROUP BY TransactionType ", subQuery)
	rows, err := result.ExecuteQuery(tx, ranges, append(predicates, q.Limit))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		sub := q.Clone()
		sub.TransactionType = new(data.TransactionType)
		sub.MinLedger = new(uint32)
		sub.MaxLedger = new(uint32)
		sub.Limit = math.MaxUint32
		if err := rows.Scan(sub.TransactionType, sub.MinLedger, sub.MaxLedger); err != nil {
			return err
		}
		txQueries = append(txQueries, sub)
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	for _, txQuery := range txQueries {
		where, _, predicates := txQuery.Where()
		sql := fmt.Sprintf("SELECT v.* FROM %sView v WHERE %s", txQuery.TransactionType, where)
		rows, err := result.ExecuteQuery(tx, sql, predicates)
		if err != nil {
			return err
		}
		defer rows.Close()
		for i := 0; rows.Next(); i++ {
			txm := &TransactionRow{
				TransactionWithMetaData: data.NewTransactionWithMetadata(*txQuery.TransactionType),
			}
			if err = rows.Scan(TxmColumns(txm)...); err != nil {
				return err
			}
			result.Transactions = append(result.Transactions, txm)
		}
		if rows.Err() != nil {
			return rows.Err()
		}
	}
	// result.Transactions.Sort()
	return nil
}

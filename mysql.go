package mysql

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/rubblelabs/ripple/data"
	"github.com/rubblelabs/ripple/storage"
)

type sqldb struct {
	*sql.DB
	name        string
	accounts    *AccountLookup
	regularKeys *RegularKeyLookup
	publicKeys  *PublicKeyLookup
	currencies  *CurrencyLookup
}

func NewMySqlDB(conn string, drop bool) (IndexedDB, error) {
	inner, err := sql.Open("mysql", conn)
	if err != nil {
		return nil, err
	}
	db := &sqldb{DB: inner}
	if err := db.QueryRow("SELECT DATABASE();").Scan(&db.name); err != nil {
		return nil, err
	}
	if drop {
		if _, err := db.Exec("DROP DATABASE " + db.name + ";"); err != nil {
			return nil, err
		}
		if _, err := db.Exec("CREATE DATABASE " + db.name + ";"); err != nil {
			return nil, err
		}
		if _, err := db.Exec("USE " + db.name + ";"); err != nil {
			return nil, err
		}
	}
	if err := db.execSchema(); err != nil {
		return nil, err
	}
	if db.accounts, err = NewAddressLookup(db); err != nil {
		return nil, err
	}
	if db.regularKeys, err = NewRegularKeyLookup(db); err != nil {
		return nil, err
	}
	if db.publicKeys, err = NewPublicKeyLookup(db); err != nil {
		return nil, err
	}
	if db.currencies, err = NewCurrencyLookup(db); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *sqldb) Insert(v data.Storer) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	switch item := v.(type) {
	case *data.Ledger:
		err = db.insertLedger(item, tx)
	case *data.TransactionWithMetaData:
		err = db.insertTransactionWithMetadata(item, tx)
	default:
		err = fmt.Errorf("Item %+v cannot be inserted into database", item)
	}
	if err != nil {
		if errRollBack := tx.Rollback(); errRollBack != nil {
			return fmt.Errorf("%s:%s", err.Error(), errRollBack.Error())
		}
		return err
	}
	return tx.Commit()
}

func (db *sqldb) insertLedger(l *data.Ledger, tx *sql.Tx) error {
	_, err := tx.Exec(statements["InsertLedger"],
		l.LedgerSequence,
		l.TotalXRP,
		l.PreviousLedger.Bytes(),
		l.TransactionHash.Bytes(),
		l.StateHash.Bytes(),
		l.ParentCloseTime.Uint32(),
		l.CloseTime.Uint32(),
		l.CloseResolution,
		l.CloseFlags,
		l.GetHash().Bytes(),
	)
	return err
}

func (db *sqldb) insertTransactionWithMetadata(t *data.TransactionWithMetaData, tx *sql.Tx) error {
	base := t.GetBase()
	_, err := tx.Exec(statements["InsertTransaction"],
		t.LedgerSequence,
		t.MetaData.TransactionIndex,
		t.MetaData.TransactionResult,
		base.TransactionType,
		base.Flags,
		base.SourceTag,
		&Account{&base.Account, db},
		base.Sequence,
		base.LastLedgerSequence,
		base.Fee.Bytes(),
		&PublicKey{base.SigningPubKey, db},
		base.TxnSignature.Bytes(),
		base.Hash.Bytes(),
	)
	if err != nil {
		return err
	}
	for i, memo := range base.Memos {
		_, err = tx.Exec(statements["InsertMemo"],
			t.LedgerSequence,
			t.MetaData.TransactionIndex,
			i,
			memo.Memo.MemoType.Bytes(),
			memo.Memo.MemoData.Bytes(),
		)
		if err != nil {
			return err
		}
	}
	for pos, effect := range t.MetaData.AffectedNodes {
		node, current, previous, state := effect.AffectedNode()
		_, err = tx.Exec(statements["InsertLedgerEntry"],
			t.LedgerSequence,
			t.MetaData.TransactionIndex,
			pos,
			node.LedgerEntryType,
			state,
			node.LedgerIndex.Bytes(),
			node.PreviousTxnID.Bytes(),
		)
		if err != nil {
			return err
		}
		// out, _ := json.MarshalIndent(node, "", "\t")
		// fmt.Println(state, string(out))
		switch e := current.(type) {
		case *data.AccountRoot:
			err = db.insertAccountRoot(pos, t, e, previous.(*data.AccountRoot), tx)
		case *data.RippleState:
			err = db.insertRippleState(pos, t, e, previous.(*data.RippleState), tx)
		case *data.Offer:
			err = db.insertOffer(pos, t, e, previous.(*data.Offer), tx)
		case *data.Directory:
			err = db.insertDirectory(pos, t, e, previous.(*data.Directory), tx)
		case *data.FeeSettings:
			err = db.insertFeeSetting(pos, t, e, previous.(*data.FeeSettings), tx)
		default:
			return fmt.Errorf("Unknown LedgerEntryType: %+v", e)
		}
		if err != nil {
			return err
		}
	}
	switch item := t.Transaction.(type) {
	case *data.Payment:
		return db.insertPayment(item, t, tx)
	case *data.OfferCreate:
		return db.insertOfferCreate(item, t, tx)
	case *data.OfferCancel:
		return db.insertOfferCancel(item, t, tx)
	case *data.AccountSet:
		return db.insertAccountSet(item, t, tx)
	case *data.SetRegularKey:
		return db.insertSetRegularKey(item, t, tx)
	case *data.TrustSet:
		return db.insertTrustSet(item, t, tx)
	case *data.SetFee:
		return db.insertSetFee(item, t, tx)
	case *data.Amendment:
		return db.insertAmendment(item, t, tx)
	default:
		return fmt.Errorf("Unknown Transaction type")
	}
}

func (db *sqldb) insertAccountRoot(pos int, txm *data.TransactionWithMetaData, current, previous *data.AccountRoot, tx *sql.Tx) error {
	_, err := tx.Exec(statements["InsertAccountRoot"],
		txm.LedgerSequence,
		txm.MetaData.TransactionIndex,
		pos,
		getOrDefault(current.Flags),
		&Account{current.Account, db},
		current.Sequence,
		current.Balance.Bytes(),
		getOrDefault(current.OwnerCount),
		&RegularKey{current.RegularKey, db},
		current.EmailHash.Bytes(),
		current.WalletLocator.Bytes(),
		current.WalletSize,
		current.MessageKey.Bytes(),
		current.Domain.Bytes(),
		current.TransferRate,
		previous.Flags,
		previous.Sequence,
		previous.Balance.Bytes(),
		previous.OwnerCount,
		&RegularKey{previous.RegularKey, db},
		previous.EmailHash.Bytes(),
		previous.WalletLocator.Bytes(),
		previous.WalletSize,
		previous.MessageKey.Bytes(),
		previous.Domain.Bytes(),
		previous.TransferRate,
	)
	return err
}

func (db *sqldb) insertRippleState(pos int, txm *data.TransactionWithMetaData, current, previous *data.RippleState, tx *sql.Tx) error {
	if (current.Balance.Currency != current.LowLimit.Currency) ||
		(current.Balance.Currency != current.HighLimit.Currency) {
		return fmt.Errorf("Bad assumptions!")
	}
	var (
		previousBalance   = NewAmount(previous.Balance)
		previousLowLimit  = NewAmount(previous.LowLimit)
		previousHighLimit = NewAmount(previous.HighLimit)
	)
	if err := previousBalance.Lookup(db); err != nil {
		return err
	}
	if err := previousLowLimit.Lookup(db); err != nil {
		return err
	}
	if err := previousHighLimit.Lookup(db); err != nil {
		return err
	}
	_, err := tx.Exec(statements["InsertRippleState"],
		txm.LedgerSequence,
		txm.MetaData.TransactionIndex,
		pos,
		current.Flags,
		current.Balance.Value.Bytes(),
		&Currency{&current.Balance.Currency, db},
		current.LowLimit.Value.Bytes(),
		&Account{&current.LowLimit.Issuer, db},
		current.HighLimit.Value.Bytes(),
		&Account{&current.HighLimit.Issuer, db},
		current.LowNode,
		current.HighNode,
		current.LowQualityIn,
		current.LowQualityOut,
		current.HighQualityIn,
		current.HighQualityOut,
		previous.Flags,
		previousBalance.Value,
		previousBalance.Currency,
		previousLowLimit.Value,
		previousLowLimit.Issuer,
		previousHighLimit.Value,
		previousHighLimit.Issuer,
		previous.LowNode,
		previous.HighNode,
		previous.LowQualityIn,
		previous.LowQualityOut,
		previous.HighQualityIn,
		previous.HighQualityOut,
	)
	return err
}

func (db *sqldb) insertOffer(pos int, txm *data.TransactionWithMetaData, current, previous *data.Offer, tx *sql.Tx) error {
	var (
		previousTakerPays = NewAmount(previous.TakerPays)
		previousTakerGets = NewAmount(previous.TakerGets)
	)
	if err := previousTakerPays.Lookup(db); err != nil {
		return err
	}
	if err := previousTakerGets.Lookup(db); err != nil {
		return err
	}
	_, err := tx.Exec(statements["InsertOffer"],
		txm.LedgerSequence,
		txm.MetaData.TransactionIndex,
		pos,
		getOrDefault(current.Flags),
		&Account{current.Account, db},
		current.Sequence,
		current.TakerPays.Value.Bytes(),
		&Currency{&current.TakerPays.Currency, db},
		&Account{&current.TakerPays.Issuer, db},
		current.TakerGets.Value.Bytes(),
		&Currency{&current.TakerGets.Currency, db},
		&Account{&current.TakerGets.Issuer, db},
		current.Expiration,
		current.BookDirectory.Bytes(),
		current.BookNode,
		current.OwnerNode,
		previous.Flags,
		previous.Sequence,
		previousTakerPays.Value,
		previousTakerPays.Currency,
		previousTakerPays.Issuer,
		previousTakerGets.Value,
		previousTakerGets.Currency,
		previousTakerGets.Issuer,
		previous.Expiration,
		previous.BookDirectory,
		previous.BookNode,
		previous.OwnerNode,
	)
	return err
}

func (db *sqldb) insertDirectory(pos int, txm *data.TransactionWithMetaData, current, previous *data.Directory, tx *sql.Tx) error {
	_, err := tx.Exec(statements["InsertDirectory"],
		txm.LedgerSequence,
		txm.MetaData.TransactionIndex,
		pos,
		current.RootIndex.Bytes(),
		current.Indexes,
		&Account{current.Owner, db},
		&Currency{current.TakerPaysCurrency.Currency(), db},
		&Account{current.TakerPaysIssuer.Account(), db},
		&Currency{current.TakerGetsCurrency.Currency(), db},
		&Account{current.TakerGetsIssuer.Account(), db},
		current.ExchangeRate.Bytes(),
		current.IndexNext,
		current.IndexPrevious,
		previous.RootIndex,
		previous.Indexes,
		&Account{previous.Owner, db},
		&Currency{previous.TakerPaysCurrency.Currency(), db},
		&Account{previous.TakerPaysIssuer.Account(), db},
		&Currency{previous.TakerGetsCurrency.Currency(), db},
		&Account{previous.TakerGetsIssuer.Account(), db},
		previous.ExchangeRate.Bytes(),
		previous.IndexNext,
		previous.IndexPrevious,
	)
	return err
}

func (db *sqldb) insertFeeSetting(pos int, txm *data.TransactionWithMetaData, current, previous *data.FeeSettings, tx *sql.Tx) error {
	_, err := tx.Exec(statements["InsertFeeSettings"],
		txm.LedgerSequence,
		txm.MetaData.TransactionIndex,
		pos,
		getOrDefault(current.Flags),
		current.BaseFee,
		current.ReferenceFeeUnits,
		current.ReserveBase,
		current.ReserveIncrement,
		current.Flags,
		previous.BaseFee,
		previous.ReferenceFeeUnits,
		previous.ReserveBase,
		previous.ReserveIncrement,
	)
	return err
}

func (db *sqldb) insertPayment(payment *data.Payment, t *data.TransactionWithMetaData, tx *sql.Tx) error {
	amount := NewAmount(&payment.Amount)
	if err := amount.Lookup(db); err != nil {
		return err
	}
	delivered := NewAmount(t.MetaData.DeliveredAmount)
	if err := delivered.Lookup(db); err != nil {
		return err
	}
	sendmax := NewAmount(payment.SendMax)
	if err := sendmax.Lookup(db); err != nil {
		return err
	}
	_, err := tx.Exec(statements["InsertPayment"],
		t.LedgerSequence,
		t.MetaData.TransactionIndex,
		&Account{&payment.Destination, db},
		amount.Value,
		amount.Currency,
		amount.Issuer,
		delivered.Value,
		delivered.Currency,
		delivered.Issuer,
		sendmax.Value,
		sendmax.Currency,
		sendmax.Issuer,
		payment.DestinationTag,
		getOrDefault(payment.InvoiceID),
	)
	if err != nil {
		return err
	}
	if payment.Paths == nil {
		return nil
	}
	for i := range *payment.Paths {
		for j, path := range (*payment.Paths)[i] {
			_, err := tx.Exec(statements["InsertPath"],
				t.LedgerSequence,
				t.MetaData.TransactionIndex,
				i,
				j,
				&Account{path.Account, db},
				&Currency{path.Currency, db},
				&Account{path.Issuer, db},
			)
			if err != nil {
				return err
			}

		}
	}
	return err
}

func (db *sqldb) insertOfferCreate(offer *data.OfferCreate, t *data.TransactionWithMetaData, tx *sql.Tx) error {
	takerPays := NewAmount(&offer.TakerPays)
	if err := takerPays.Lookup(db); err != nil {
		return err
	}
	takerGets := NewAmount(&offer.TakerGets)
	if err := takerGets.Lookup(db); err != nil {
		return err
	}
	_, err := tx.Exec(statements["InsertOfferCreate"],
		t.LedgerSequence,
		t.MetaData.TransactionIndex,
		offer.OfferSequence,
		takerPays.Value,
		takerPays.Currency,
		takerPays.Issuer,
		takerGets.Value,
		takerGets.Currency,
		takerGets.Issuer,
		offer.Expiration,
	)
	return err
}

func (db *sqldb) insertOfferCancel(offer *data.OfferCancel, t *data.TransactionWithMetaData, tx *sql.Tx) error {
	_, err := tx.Exec(statements["InsertOfferCancel"],
		t.LedgerSequence,
		t.MetaData.TransactionIndex,
		offer.OfferSequence,
	)
	return err
}

func (db *sqldb) insertAccountSet(accountset *data.AccountSet, t *data.TransactionWithMetaData, tx *sql.Tx) error {
	_, err := tx.Exec(statements["InsertAccountSet"],
		t.LedgerSequence,
		t.MetaData.TransactionIndex,
		NullHash128{&accountset.EmailHash},
		NullHash256{&accountset.WalletLocator},
		accountset.WalletSize,
		NullVariableLength{&accountset.MessageKey},
		NullVariableLength{&accountset.Domain},
		accountset.TransferRate,
		accountset.SetFlag,
		accountset.ClearFlag,
	)
	return err
}

func (db *sqldb) insertSetRegularKey(keyset *data.SetRegularKey, t *data.TransactionWithMetaData, tx *sql.Tx) error {
	_, err := tx.Exec(statements["InsertSetRegularKey"],
		t.LedgerSequence,
		t.MetaData.TransactionIndex,
		&RegularKey{keyset.RegularKey, db},
	)
	return err
}

func (db *sqldb) insertTrustSet(trustset *data.TrustSet, t *data.TransactionWithMetaData, tx *sql.Tx) error {
	limit := NewAmount(&trustset.LimitAmount)
	if err := limit.Lookup(db); err != nil {
		return err
	}
	_, err := tx.Exec(statements["InsertTrustSet"],
		t.LedgerSequence,
		t.MetaData.TransactionIndex,
		limit.Value,
		limit.Currency,
		limit.Issuer,
		trustset.QualityIn,
		trustset.QualityOut,
	)
	return err
}

func (db *sqldb) insertSetFee(setFee *data.SetFee, t *data.TransactionWithMetaData, tx *sql.Tx) error {
	_, err := tx.Exec(statements["InsertSetFee"],
		t.LedgerSequence,
		t.MetaData.TransactionIndex,
		setFee.BaseFee,
		setFee.ReferenceFeeUnits,
		setFee.ReserveBase,
		setFee.ReserveIncrement,
	)
	return err
}

func (db *sqldb) insertAmendment(amendment *data.Amendment, t *data.TransactionWithMetaData, tx *sql.Tx) error {
	_, err := tx.Exec(statements["InsertAmendment"],
		t.LedgerSequence,
		t.MetaData.TransactionIndex,
		amendment.Amendment.Bytes(),
	)
	return err
}

func (db *sqldb) InsertLookup(stmnt string, item *LookupItem) error {
	result, err := db.Exec(statements[stmnt], item.Id, item.Value, item.Human)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n > 1 {
		return fmt.Errorf("Duplicate Lookup inserted: %s", item.Human, item.Id)
	}
	return nil
}

func (db *sqldb) GetLookups(stmnt string) ([]LookupItem, error) {
	rows, err := db.DB.Query(statements[stmnt])
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []LookupItem
	for rows.Next() {
		var item LookupItem
		if err := rows.Scan(&item.Id, &item.Value, &item.Human); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (db *sqldb) GetAccount(n uint32) *data.Account {
	return db.accounts.Get(n)
}

func (db *sqldb) LookupAccount(a *data.Account) (uint32, error) {
	return db.accounts.Lookup(a)
}

func (db *sqldb) LookupCurrency(c *data.Currency) (uint32, error) {
	return db.currencies.Lookup(c)
}

func (db *sqldb) LookupRegularKey(r *data.RegularKey) (uint32, error) {
	return db.regularKeys.Lookup(r)
}

func (db *sqldb) LookupPublicKey(p *data.PublicKey) (uint32, error) {
	return db.publicKeys.Lookup(p)
}

func (db *sqldb) SearchAccounts(s string) ([]string, error) {
	rows, err := db.DB.Query(`SELECT Human FROM Account WHERE Human LIKE ? ORDER BY Human LIMIT 10;`, "%"+s+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []string{}
	for rows.Next() {
		var result string
		if err := rows.Scan(&result); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if rows.Err() != nil {
		return nil, err
	}
	return results, nil
}

func (db *sqldb) Query(q Query, result *QueryResult) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	return q.Rows(tx, result)
}

func (db *sqldb) Get(hash data.Hash256) (data.Storer, error) {
	result := &QueryResult{}
	query, err := NewTransactionQuery(db, map[string]string{"Hash": hash.String()})
	if err != nil {
		return nil, err
	}
	err = db.Query(query.LedgerQuery, result)
	switch {
	case err == storage.ErrNotFound:
		break
	case err != nil:
		return nil, err
	case len(result.Ledgers) > 1:
		return nil, fmt.Errorf("More than one Ledger found for %s", hash)
	case len(result.Ledgers) == 1:
		return result.Ledgers[0], nil
	}
	err = db.Query(query, result)
	switch {
	case err != nil:
		return nil, err
	case len(result.Transactions) == 0:
		return nil, storage.ErrNotFound
	case len(result.Transactions) > 1:
		return nil, fmt.Errorf("More than one Transaction found for %s", hash)
	default:
		return result.Transactions[0], nil
	}
}

func (db *sqldb) MissingLedgers(start, end uint32) ([]uint32, error) {
	stmnt := "SELECT s.seq  FROM seq_%d_to_%d s LEFT OUTER JOIN Ledger l ON s.seq=l.LedgerSequence WHERE l.LedgerSequence IS NULL;"
	rows, err := db.DB.Query(fmt.Sprintf(stmnt, start, end))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ledgers []uint32
	for rows.Next() {
		var ledgerSequence uint32
		if err := rows.Scan(&ledgerSequence); err != nil {
			return nil, err
		}
		ledgers = append(ledgers, ledgerSequence)
	}
	return ledgers, rows.Err()
}

func (db *sqldb) Stats() string {
	return ""
}

func (db *sqldb) Ledger() (*data.LedgerSet, error) { return nil, nil }

func (db *sqldb) execSchema() error {
	for _, sql := range schema {
		if _, err := db.Exec(sql); err != nil {
			return fmt.Errorf("%s\n%s", err, sql)
		}
	}
	return nil
}

package mysql

import (
	"database/sql"
	"github.com/rubblelabs/ripple/data"
	"github.com/rubblelabs/ripple/storage"
)

type Query interface {
	Rows(*sql.Tx, *QueryResult) error
}

type IndexedDB interface {
	storage.DB
	Query(Query, *QueryResult) error
	InsertLookup(string, *LookupItem) error
	GetLookups(string) ([]LookupItem, error)
	GetAccount(uint32) *data.Account
	LookupAccount(*data.Account) (uint32, error)
	LookupCurrency(*data.Currency) (uint32, error)
	LookupRegularKey(*data.RegularKey) (uint32, error)
	LookupPublicKey(*data.PublicKey) (uint32, error)
	SearchAccounts(s string) ([]string, error)
	MissingLedgers(start, end uint32) ([]uint32, error)
}

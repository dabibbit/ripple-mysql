package mysql

import (
	"bytes"
	"database/sql/driver"
	"fmt"
	"github.com/rubblelabs/ripple/data"
	"reflect"
)

type Path struct {
	Account, Currency, Issuer *uint32
}

type Value struct {
	*data.Value
}

type Amount struct {
	Amount   *data.Amount
	Value    []byte
	Currency *uint32
	Issuer   *uint32
}

func getOrDefault(value interface{}) interface{} {
	switch v := value.(type) {
	case *uint32:
		if v == nil {
			return 0
		}
		return *v
	case *data.LedgerEntryFlag:
		if v == nil {
			return 0
		}
		return *v
	case *data.Hash256:
		if v == nil {
			return []byte(nil)
		}
		return v.Bytes()
	case *data.Hash128:
		if v == nil {
			return []byte(nil)
		}
		return v.Bytes()
	case *data.VariableLength:
		if v == nil {
			return []byte(nil)
		}
		return v.Bytes()
	default:
		panic(fmt.Sprintf("No defined default for: %s", reflect.TypeOf(value)))
	}
}

func NewAmount(a *data.Amount) *Amount {
	return &Amount{Amount: a}
}

func (a *Amount) Lookup(db IndexedDB) error {
	if a.Amount == nil {
		return nil
	}
	a.Value = a.Amount.Value.Bytes()
	currency, err := db.LookupCurrency(&a.Amount.Currency)
	if err != nil {
		return err
	}
	a.Currency = &currency
	issuer, err := db.LookupAccount(&a.Amount.Issuer)
	if err != nil {
		return err
	}
	a.Issuer = &issuer
	return nil
}

type RippleTime struct {
	*data.RippleTime
}

type Hash256 struct {
	*data.Hash256
}

type Account struct {
	*data.Account
	DB IndexedDB
}

type Currency struct {
	*data.Currency
	DB IndexedDB
}

type RegularKey struct {
	*data.RegularKey
	DB IndexedDB
}

type PublicKey struct {
	*data.PublicKey
	DB IndexedDB
}

type NullAmount struct {
	Amount **data.Amount
}

type NullPublicKey struct {
	PublicKey **data.PublicKey
}

type NullHash256 struct {
	Hash256 **data.Hash256
}

type NullHash128 struct {
	Hash128 **data.Hash128
}

type NullVariableLength struct {
	VariableLength **data.VariableLength
}

type NullUint32 struct {
	Uint32 **uint32
}

type NullRegularKey struct {
	RegularKey **data.RegularKey
}

func (a *Amount) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	b, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan %+v into Amount", src)
	}
	return a.Amount.Unmarshal(bytes.NewReader(b))
}

func (a *NullAmount) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	b, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan %+v into Amount", src)
	}
	var amount data.Amount
	if err := amount.Unmarshal(bytes.NewReader(b)); err != nil {
		return err
	}
	*a.Amount = &amount
	return nil
}

func (v *Value) Scan(src interface{}) error {
	b, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan %+v into Value", src)
	}
	return v.Unmarshal(bytes.NewReader(b))
}

func (t *RippleTime) Scan(src interface{}) error {
	v, ok := src.(int64)
	if !ok {
		return fmt.Errorf("Cannot scan %+v into RippleTime", src)
	}
	t.T = uint32(v)
	return nil
}

func (h *Hash256) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	return scanBytes(h.Hash256[:], src, "Hash256")
}

func (a *Account) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	return scanBytes(a.Account[:], src, "Account")
}

func (a *Account) Value() (driver.Value, error) {
	if a.DB == nil {
		return nil, fmt.Errorf("Need an IndexedDB")
	}
	if a.Account == nil {
		return nil, nil
	}
	account, err := a.DB.LookupAccount(a.Account)
	if err != nil {
		return nil, err
	}
	return int64(account), nil
}

func (c *Currency) Value() (driver.Value, error) {
	if c.DB == nil {
		return nil, fmt.Errorf("Need an IndexedDB")
	}
	if c.Currency == nil {
		return nil, nil
	}
	currency, err := c.DB.LookupCurrency(c.Currency)
	if err != nil {
		return nil, err
	}
	return int64(currency), nil
}

func (r *RegularKey) Value() (driver.Value, error) {
	if r.DB == nil {
		return nil, fmt.Errorf("Need an IndexedDB")
	}
	if r.RegularKey == nil {
		return nil, nil
	}
	regKey, err := r.DB.LookupRegularKey(r.RegularKey)
	if err != nil {
		return nil, err
	}
	return int64(regKey), nil
}

func (a *PublicKey) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	return scanBytes(a.PublicKey[:], src, "PublicKey")
}

func (p *PublicKey) Value() (driver.Value, error) {
	if p.DB == nil {
		return nil, fmt.Errorf("Need an IndexedDB")
	}
	if p.PublicKey == nil {
		return nil, nil
	}
	pubKey, err := p.DB.LookupPublicKey(p.PublicKey)
	if err != nil {
		return nil, err
	}
	return int64(pubKey), nil
}

func (p *NullPublicKey) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	var pubKey data.PublicKey
	if err := scanBytes(pubKey[:], src, "NullPublicKey"); err != nil {
		return err
	}
	*p.PublicKey = &pubKey
	return nil
}

func (r *NullRegularKey) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	var regKey data.RegularKey
	if err := scanBytes(regKey[:], src, "NullRegularKey"); err != nil {
		return err
	}
	*r.RegularKey = &regKey
	return nil
}

func (h NullHash256) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	var hash data.Hash256
	if err := scanBytes(hash[:], src, "NullHash256"); err != nil {
		return nil
	}
	*h.Hash256 = &hash
	return nil
}

func (h NullHash256) Value() (driver.Value, error) {
	if *h.Hash256 == nil {
		return nil, nil
	}
	return (*h.Hash256).Bytes(), nil
}

func (h NullHash128) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	var hash data.Hash128
	if err := scanBytes(hash[:], src, "NullHash128"); err != nil {
		return nil
	}
	*h.Hash128 = &hash
	return nil
}

func (h NullHash128) Value() (driver.Value, error) {
	if *h.Hash128 == nil {
		return nil, nil
	}
	return (*h.Hash128).Bytes(), nil
}

func (v NullVariableLength) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	b, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan %+v into a NullVariableLength", src)
	}
	vl := make(data.VariableLength, len(b))
	copy(vl, b)
	*v.VariableLength = &vl
	return nil
}

func (v NullVariableLength) Value() (driver.Value, error) {
	if *v.VariableLength == nil {
		return nil, nil
	}
	return (*v.VariableLength).Bytes(), nil
}

func (n NullUint32) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	v, ok := src.(int64)
	if !ok {
		return fmt.Errorf("NullUint32: Cannot scan: %+v", src)
	}
	u := uint32(v)
	*n.Uint32 = &u
	return nil
}

func scanBytes(dest []byte, src interface{}, typ string) error {
	b, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan %+v into a %s", src, typ)
	}
	copy(dest, b)
	return nil
}

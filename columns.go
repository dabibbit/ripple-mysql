package mysql

import (
	"github.com/rubblelabs/ripple/data"
)

func LedgerColumns(ledger *data.Ledger) []interface{} {
	return []interface{}{
		&ledger.LedgerSequence,
		&ledger.TotalXRP,
		&Hash256{&ledger.PreviousLedger},
		&Hash256{&ledger.TransactionHash},
		&Hash256{&ledger.StateHash},
		&RippleTime{&ledger.ParentCloseTime},
		&RippleTime{&ledger.CloseTime},
		&ledger.CloseResolution,
		&ledger.CloseFlags,
		&Hash256{&ledger.Hash},
	}
}

func TxmColumns(txm *TransactionRow) []interface{} {
	base := txm.GetBase()
	items := []interface{}{
		&txm.LedgerSequence,
		&RippleTime{&txm.CloseTime},
		&txm.MetaData.TransactionIndex,
		&txm.MetaData.TransactionResult,
		&base.TransactionType,
		&base.Flags,
		&base.SourceTag,
		&txm.Account,
		&Account{&base.Account, nil},
		&base.Sequence,
		&base.LastLedgerSequence,
		&Value{&base.Fee},
		&NullPublicKey{&base.SigningPubKey},
		&base.TxnSignature,
		&Hash256{&base.Hash},
	}
	switch v := txm.Transaction.(type) {
	case *data.Payment:
		items = append(items,
			&Account{&v.Destination, nil},
			NewAmount(&v.Amount),
			&NullAmount{&txm.MetaData.DeliveredAmount},
			&NullAmount{&v.SendMax},
			&NullUint32{&v.DestinationTag},
			&NullHash256{&v.InvoiceID},
		)
	case *data.OfferCreate:
		items = append(items,
			&NullUint32{&v.OfferSequence},
			NewAmount(&v.TakerPays),
			NewAmount(&v.TakerGets),
			&NullUint32{&v.Expiration},
		)
	case *data.OfferCancel:
		items = append(items,
			&v.OfferSequence,
		)
	case *data.AccountSet:
		items = append(items,
			&NullHash128{&v.EmailHash},
			&NullHash256{&v.WalletLocator},
			&NullUint32{&v.WalletSize},
			&v.MessageKey,
			&v.Domain,
			&NullUint32{&v.TransferRate},
			&NullUint32{&v.SetFlag},
			&NullUint32{&v.ClearFlag},
		)
	case *data.TrustSet:
		items = append(items,
			NewAmount(&v.LimitAmount),
			&NullUint32{&v.QualityIn},
			&NullUint32{&v.QualityOut},
		)
	case *data.SetRegularKey:
		items = append(items,
			&NullRegularKey{&v.RegularKey},
		)
	case *data.SetFee:
		items = append(items,
			&v.BaseFee,
			&v.ReferenceFeeUnits,
			&v.ReserveBase,
			&v.ReserveIncrement,
		)
	}
	return items
}

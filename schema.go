package mysql

var (
	kernel = `SELECT t.LedgerSequence,
  t.TransactionIndex,
  t.TransactionType 
  FROM Transaction t
  WHERE (? IS NULL OR t.Hash=?)
  AND (? IS NULL OR t.LedgerSequence=?)
  AND (? IS NULL OR t.LedgerSequence>=?)
  AND (? IS NULL OR t.LedgerSequence<=?)
  AND (? IS NULL OR t.Account=?)
  AND (? IS NULL OR t.TransactionType=?)`

	kernelJoin = ` INNER JOIN (` + kernel + `)k 
  ON v.LedgerSequence=k.LedgerSequence AND v.TransactionIndex=k.TransactionIndex;`
)

var queries = map[string]string{
	"GetLedgerRange":    `SELECT MIN(LedgerSequence),MAX(LedgerSequence) FROM Ledger;`,
	"GetRanges":         `SELECT TransactionType,min(LedgerSequence),max(LedgerSequence) FROM(` + kernel + ` ORDER BY LedgerSequence %s,TransactionIndex %s LIMIT ?)t GROUP BY TransactionType`,
	"GetTransactions":   `SELECT v.* FROM TransactiontView v` + kernelJoin,
	"GetPayments":       `SELECT v.* FROM PaymentView v` + kernelJoin,
	"GetOfferCreates":   `SELECT v.* FROM OfferCreateView v` + kernelJoin,
	"GetOfferCancels":   `SELECT v.* FROM OfferCancelView v` + kernelJoin,
	"GetAccountSets":    `SELECT v.* FROM AccountSetView v` + kernelJoin,
	"GetSetRegularKeys": `SELECT v.* FROM SetRegularKeyView v` + kernelJoin,
	"GetTrustSets":      `SELECT v.* FROM TrustSetView v` + kernelJoin,
	"GetSetFees":        `SELECT v.* FROM SetFeeView v` + kernelJoin,
	"GetAmendments":     `SELECT v.* FROM AmendmentView v` + kernelJoin,
}

var statements = map[string]string{
	"InsertLedger":        `REPLACE INTO Ledger VALUES(?,?,?,?,?,?,?,?,?,?);`,
	"InsertTransaction":   `REPLACE INTO Transaction VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?);`,
	"InsertPayment":       `REPLACE INTO Payment VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?);`,
	"InsertOfferCreate":   `REPLACE INTO OfferCreate VALUES(?,?,?,?,?,?,?,?,?,?);`,
	"InsertOfferCancel":   `REPLACE INTO OfferCancel VALUES(?,?,?);`,
	"InsertAccountSet":    `REPLACE INTO AccountSet VALUES(?,?,?,?,?,?,?,?,?,?);`,
	"InsertSetRegularKey": `REPLACE INTO SetRegularKey VALUES(?,?,?);`,
	"InsertTrustSet":      `REPLACE INTO TrustSet VALUES(?,?,?,?,?,?,?);`,
	"InsertSetFee":        `REPLACE INTO SetFee VALUES(?,?,?,?,?,?)`,
	"InsertAmendment":     `REPLACE INTO Amendment VALUES(?,?,?)`,
	"InsertPath":          `REPLACE INTO Path VALUES(?,?,?,?,?,?,?)`,
	"InsertMemo":          `REPLACE INTO Memo VALUES(?,?,?,?,?);`,
	"InsertLedgerEntry":   `REPLACE INTO LedgerEntry Values(?,?,?,?,?,?,?)`,
	"InsertAccountRoot":   `REPLACE INTO AccountRoot VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?);`,
	"InsertRippleState":   `REPLACE INTO RippleState VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?);`,
	"InsertOffer":         `REPLACE INTO Offer VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?);`,
	"InsertFeeSettings":   `REPLACE INTO FeeSettings VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?);`,
	"InsertDirectory":     `REPLACE INTO Directory VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?);`,

	"GetAccounts":      `SELECT Id,Account,Human FROM Account;`,
	"InsertAccount":    `REPLACE INTO Account VALUES(?,?,?);`,
	"GetRegularKeys":   `SELECT Id,RegularKey,Human FROM RegularKey;`,
	"InsertRegularKey": `REPLACE INTO RegularKey VALUES(?,?,?);`,
	"GetPublicKeys":    `SELECT Id,PublicKey,Human FROM PublicKey;`,
	"InsertPublicKey":  `REPLACE INTO PublicKey VALUES(?,?,?);`,
	"GetCurrencies":    `SELECT Id,Currency,Human FROM Currency;`,
	"InsertCurrency":   `REPLACE INTO Currency VALUES(?,?,?);`,
}

var schema = []string{`
CREATE TABLE IF NOT EXISTS Currency(
  Id INT UNSIGNED NOT NULL,
  Currency BINARY(20) NOT NULL,
  Human VARCHAR(3) NOT NULL,
  PRIMARY KEY(Id)
);
`, `
CREATE TABLE IF NOT EXISTS Account(
  Id INT UNSIGNED NOT NULL,
  Account BINARY(20) NOT NULL,
  Human VARCHAR(35) NOT NULL,
  PRIMARY KEY(Id),
  KEY(Human)
);
`, `
CREATE TABLE IF NOT EXISTS RegularKey(
  Id INT UNSIGNED NOT NULL,
  RegularKey BINARY(20) NOT NULL,
  Human VARCHAR(35) NOT NULL,
  PRIMARY KEY(Id)
);
`, `
CREATE TABLE IF NOT EXISTS PublicKey(
  Id INT UNSIGNED NOT NULL,
  PublicKey BINARY(33) NOT NULL,
  Human VARCHAR(66) NOT NULL,
  PRIMARY KEY(Id)
);
`, `
CREATE TABLE IF NOT EXISTS Ledger (
  LedgerSequence INT UNSIGNED NOT NULL,
  TotalXRP BIGINT UNSIGNED NOT NULL,
  PreviousLedger BINARY(32) NOT NULL,
  TransactionHash BINARY(32) NOT NULL,
  StateHash BINARY(32) NOT NULL,
  ParentCloseTime INT UNSIGNED NOT NULL,
  CloseTime INT UNSIGNED NOT NULL,
  CloseResolution TINYINT UNSIGNED NOT NULL,
  CloseFlags TINYINT UNSIGNED NOT NULL,
  Hash BINARY(32) NOT NULL,
  PRIMARY KEY(LedgerSequence),KEY(Hash)
);
`, `
CREATE TABLE IF NOT EXISTS Transaction(
  LedgerSequence INT UNSIGNED NOT NULL,
  TransactionIndex INT UNSIGNED NOT NULL,
  TransactionResult TINYINT UNSIGNED NOT NULL,
  TransactionType MEDIUMINT UNSIGNED NOT NULL,
  Flags INT UNSIGNED NULL,
  SourceTag INT UNSIGNED  NULL,
  Account INT UNSIGNED NOT NULL,
  Sequence INT UNSIGNED NOT NULL,
  LastLedgerSequence INT UNSIGNED NULL,
  Fee BINARY(8),
  SigningPubKey INT UNSIGNED NOT NULL,
  TxnSignature VARBINARY(72) NULL,
  Hash BINARY(32) NOT NULL,
  PRIMARY KEY(LedgerSequence,TransactionIndex),
  KEY(Hash),
  KEY(Account,TransactionType),
  KEY(TransactionType)
);
`, `
CREATE OR REPLACE VIEW TransactionView AS
SELECT t.LedgerSequence,
  COALESCE(l.CloseTime,0),
  t.TransactionIndex,
  t.TransactionResult,
  t.TransactionType,
  t.Flags,
  t.SourceTag,
  t.Account,
  a.Account AS HumanAccount,
  t.Sequence,
  t.LastLedgerSequence,
  t.Fee,
  p.PublicKey,
  t.TxnSignature,
  t.Hash
FROM Transaction t
LEFT OUTER JOIN Ledger l ON t.LedgerSequence=l.LedgerSequence
INNER JOIN Account a ON t.Account=a.Id
INNER JOIN PublicKey p ON t.SigningPubKey=p.Id;
`, `
CREATE TABLE IF NOT EXISTS Payment(
  LedgerSequence INT UNSIGNED NOT NULL,
  TransactionIndex INT UNSIGNED NOT NULL,
  Destination INT UNSIGNED NOT NULL,
  Amount BINARY(8) NOT NULL,
  AmountCurrency INT UNSIGNED NOT NULL,
  AmountIssuer INT UNSIGNED NOT NULL,
  DeliveredAmount BINARY(8) NULL,
  DeliveredCurrency INT UNSIGNED NULL,
  DeliveredIssuer INT UNSIGNED NULL,
  SendMax BINARY(8) NULL,
  SendMaxCurrency INT UNSIGNED NULL,
  SendMaxIssuer INT UNSIGNED NULL,
  DestinationTag INT UNSIGNED NULL,
  InvoiceID BINARY(32) NULL,
  PRIMARY KEY(LedgerSequence,TransactionIndex),KEY(Destination)
);
`, `
CREATE OR REPLACE VIEW PaymentView AS
SELECT t.*, 
  dest.Account AS Destination,
  CONCAT(p.Amount,ac.Currency,aa.Account) AS Amount,
  CONCAT(p.DeliveredAmount,dc.Currency,di.Account) AS DeliveredAmount,
  CONCAT(p.SendMax,sc.Currency,si.Account) AS SendMax,
  p.DestinationTag,
  p.InvoiceID
FROM TransactionView t
INNER JOIN Payment p           ON t.LedgerSequence=p.LedgerSequence AND t.TransactionIndex=p.TransactionIndex
INNER JOIN Account dest        ON p.Destination=dest.Id
INNER JOIN Currency ac         ON p.AmountCurrency=ac.Id
INNER JOIN Account aa          ON p.AmountIssuer=aa.Id
LEFT OUTER JOIN Currency dc    ON p.DeliveredCurrency=dc.Id
LEFT OUTER JOIN Account di     ON p.DeliveredIssuer=di.Id
LEFT OUTER JOIN Currency sc    ON p.SendMaxCurrency=sc.Id
LEFT OUTER JOIN Account si     ON p.SendMaxIssuer=si.Id;
`, `
CREATE TABLE IF NOT EXISTS OfferCreate (
  LedgerSequence INT UNSIGNED NOT NULL,
  TransactionIndex INT UNSIGNED NOT NULL,
  OfferSequence INT UNSIGNED,
  TakerPays BINARY(8) NOT NULL,
  TakerPaysCurrency INT UNSIGNED NOT NULL,
  TakerPaysIssuer INT UNSIGNED NOT NULL,
  TakerGets BINARY(8) NOT NULL,
  TakerGetsCurrency INT UNSIGNED NOT NULL,
  TakerGetsIssuer INT UNSIGNED NOT NULL,
  Expiration INT UNSIGNED NULL,
  PRIMARY KEY(LedgerSequence,TransactionIndex)
);
`, `
CREATE OR REPLACE VIEW OfferCreateView AS
SELECT t.*, 
  o.OfferSequence AS OfferSequence,
  CONCAT(o.TakerPays,pc.Currency,pa.Account) AS TakerPays,
  CONCAT(o.TakerGets,gc.Currency,ga.Account) AS TakerGets,
  o.Expiration AS Expiration
FROM TransactionView t
INNER JOIN OfferCreate o       ON t.LedgerSequence=o.LedgerSequence AND t.TransactionIndex=o.TransactionIndex
INNER JOIN Currency pc         ON o.TakerPaysCurrency=pc.Id
INNER JOIN Account pa          ON o.TakerPaysIssuer=pa.Id
INNER JOIN Currency gc         ON o.TakerGetsCurrency=gc.Id
INNER JOIN Account ga          ON o.TakerGetsIssuer=ga.Id;
`, `
CREATE TABLE IF NOT EXISTS OfferCancel (
  LedgerSequence INT UNSIGNED NOT NULL,
  TransactionIndex INT UNSIGNED NOT NULL,
  OfferSequence INT UNSIGNED NOT NULL,
  PRIMARY KEY(LedgerSequence,TransactionIndex)
);
`, `
CREATE OR REPLACE VIEW OfferCancelView AS
SELECT t.*, 
  o.OfferSequence AS OfferSequence
FROM TransactionView t
INNER JOIN OfferCancel o       ON t.LedgerSequence=o.LedgerSequence AND t.TransactionIndex=o.TransactionIndex;
`, `
CREATE TABLE IF NOT EXISTS SetRegularKey (
  LedgerSequence INT UNSIGNED NOT NULL,
  TransactionIndex INT UNSIGNED NOT NULL,
  RegularKey INT UNSIGNED NULL,
  PRIMARY KEY(LedgerSequence,TransactionIndex)
);
`, `
CREATE OR REPLACE VIEW SetRegularKeyView AS
SELECT t.*, 
  k.RegularKey
FROM TransactionView t
INNER JOIN SetRegularKey r    ON t.LedgerSequence=r.LedgerSequence AND t.TransactionIndex=r.TransactionIndex
LEFT OUTER JOIN RegularKey k  ON r.RegularKey=k.Id;
`, `
CREATE TABLE IF NOT EXISTS SetFee(
  LedgerSequence INT UNSIGNED NOT NULL,
  TransactionIndex INT UNSIGNED NOT NULL,
  BaseFee BIGINT UNSIGNED NULL,
  ReferenceFeeUnits BIGINT UNSIGNED NULL,
  ReserveBase  BIGINT UNSIGNED NULL,
  ReserveIncrement BIGINT UNSIGNED NULL,
  PRIMARY KEY(LedgerSequence,TransactionIndex)
);
`, `
CREATE OR REPLACE VIEW SetFeeView AS
SELECT t.*, 
  f.BaseFee,
  f.ReferenceFeeUnits,
  f.ReserveBase,
  f.ReserveIncrement
FROM TransactionView t
INNER JOIN SetFee f  ON t.LedgerSequence=f.LedgerSequence AND t.TransactionIndex=f.TransactionIndex;
`, `
CREATE TABLE IF NOT EXISTS TrustSet (
  LedgerSequence INT UNSIGNED NOT NULL,
  TransactionIndex INT UNSIGNED NOT NULL,
  LimitAmount BINARY(8) NOT NULL,
  LimitAmountCurrency INT UNSIGNED NOT NULL,
  LimitAmountIssuer INT UNSIGNED NOT NULL,
  QualityIn INT UNSIGNED NULL,
  QualityOut INT UNSIGNED NULL,
  PRIMARY KEY(LedgerSequence,TransactionIndex)
);
`, `
CREATE OR REPLACE VIEW TrustSetView AS
SELECT t.*, 
  CONCAT(ts.LimitAmount,lc.Currency,la.Account) AS LimitAmount,
  ts.QualityIn,
  ts.QualityOut
FROM TransactionView t
INNER JOIN TrustSet ts         ON t.LedgerSequence=ts.LedgerSequence AND t.TransactionIndex=ts.TransactionIndex
INNER JOIN Currency lc         ON ts.LimitAmountCurrency=lc.Id
INNER JOIN Account la          ON ts.LimitAmountIssuer=la.Id
`, `
CREATE TABLE IF NOT EXISTS AccountSet (
  LedgerSequence INT UNSIGNED NOT NULL,
  TransactionIndex INT UNSIGNED NOT NULL,
  EmailHash BINARY(16) NULL,
  WalletLocator BINARY(32) NULL,
  WalletSize INT UNSIGNED NULL,
  MessageKey BINARY(33) NULL,
  Domain TINYBLOB NULL,
  TransferRate INT UNSIGNED NULL,
  SetFlag INT UNSIGNED NULL,
  ClearFlag INT UNSIGNED NULL,
  PRIMARY KEY(LedgerSequence,TransactionIndex)
);
`, `
CREATE OR REPLACE VIEW AccountSetView AS
SELECT t.*, 
  a.EmailHash,
  a.WalletLocator,
  a.WalletSize,
  a.MessageKey,
  a.Domain,
  a.TransferRate,
  a.SetFlag,
  a.ClearFlag
FROM TransactionView t
INNER JOIN AccountSet a ON t.LedgerSequence=a.LedgerSequence AND t.TransactionIndex=a.TransactionIndex
`, `
CREATE TABLE IF NOT EXISTS Amendment (
  LedgerSequence INT UNSIGNED NOT NULL,
  TransactionIndex INT UNSIGNED NOT NULL,
  Amendment BINARY(32),
  PRIMARY KEY(LedgerSequence,TransactionIndex)
);
`, `
CREATE OR REPLACE VIEW AmendmentView AS
SELECT t.*, 
  a.Amendment
FROM TransactionView t
INNER JOIN Amendment a ON t.LedgerSequence=a.LedgerSequence AND t.TransactionIndex=a.TransactionIndex
`, `
CREATE TABLE IF NOT EXISTS Path(
  LedgerSequence INT UNSIGNED NOT NULL,
  TransactionIndex INT UNSIGNED NOT NULL,
  PathSet MEDIUMINT UNSIGNED NOT NULL,
  Position MEDIUMINT UNSIGNED NOT NULL,
  Account INT UNSIGNED NULL,
  Currency INT UNSIGNED NULL,
  Issuer INT UNSIGNED NULL,
  PRIMARY KEY(LedgerSequence,TransactionIndex,PathSet,Position)
);
`, `
CREATE TABLE IF NOT EXISTS Memo(
  LedgerSequence INT UNSIGNED NOT NULL,
  TransactionIndex INT UNSIGNED NOT NULL,
  Position MEDIUMINT UNSIGNED NOT NULL,
  MemoType BLOB NULL,
  MemoData BLOB NULL,
  PRIMARY KEY(LedgerSequence,TransactionIndex,Position)
);`, `
CREATE TABLE IF NOT EXISTS LedgerEntry (
  LedgerSequence INT UNSIGNED NOT NULL,
  TransactionIndex INT UNSIGNED NOT NULL,
  Position MEDIUMINT UNSIGNED NOT NULL,
  LedgerEntryType MEDIUMINT UNSIGNED NOT NULL,
  LedgerEntryState SMALLINT UNSIGNED NOT NULL,
  LedgerIndex BINARY(32) NOT NULL,
  PreviousTxnID BINARY(32) NULL,
  PRIMARY KEY(LedgerSequence,TransactionIndex,Position)
);`, `
CREATE TABLE IF NOT EXISTS AccountRoot (
  LedgerSequence INT UNSIGNED NOT NULL,
  TransactionIndex INT UNSIGNED NOT NULL,
  Position MEDIUMINT UNSIGNED NOT NULL,
  Flags INT UNSIGNED NOT NULL,
  Account INT UNSIGNED NULL,
  Sequence INT UNSIGNED NULL,
  Balance BINARY(8) NULL,
  OwnerCount INT UNSIGNED NULL,
  RegularKey INT UNSIGNED NULL,
  EmailHash BINARY(16) NULL,
  WalletLocator BINARY(32) NULL,
  WalletSize INT UNSIGNED NULL,
  MessageKey BINARY(33) NULL,
  Domain TINYBLOB NULL,
  TransferRate INT UNSIGNED NULL,
  Previous_Flags INT UNSIGNED  NULL,
  Previous_Sequence INT UNSIGNED NULL,
  Previous_Balance BINARY(8) NULL,
  Previous_OwnerCount INT UNSIGNED NULL,
  Previous_RegularKey INT UNSIGNED NULL,
  Previous_EmailHash BINARY(16) NULL,
  Previous_WalletLocator BINARY(32) NULL,
  Previous_WalletSize INT UNSIGNED NULL,
  Previous_MessageKey BINARY(33) NULL,
  Previous_Domain TINYBLOB NULL,
  Previous_TransferRate INT UNSIGNED NULL,
  PRIMARY KEY(LedgerSequence,TransactionIndex,Position),
  KEY(Account)
);
`, `
CREATE TABLE IF NOT EXISTS Offer (
  LedgerSequence INT UNSIGNED NOT NULL,
  TransactionIndex INT UNSIGNED NOT NULL,
  Position MEDIUMINT UNSIGNED NOT NULL,
  Flags INT UNSIGNED NOT NULL,
  Account INT UNSIGNED NOT NULL,
  Sequence INT UNSIGNED NOT NULL,
  TakerPays BINARY(8) NOT NULL,
  TakerPaysCurrency INT UNSIGNED NOT NULL,
  TakerPaysIssuer INT UNSIGNED NOT NULL,
  TakerGets BINARY(8) NOT NULL,
  TakerGetsCurrency INT UNSIGNED NOT NULL,
  TakerGetsIssuer INT UNSIGNED NOT NULL,
  Expiration INT UNSIGNED,
  BookDirectory BINARY(32) NOT NULL,
  BookNode BIGINT NULL,
  OwnerNode BIGINT NULL,
  Previous_Flags INT UNSIGNED  NULL,
  Previous_Sequence INT UNSIGNED NULL,
  Previous_TakerPays BINARY(8) NULL,
  Previous_TakerPaysCurrency INT UNSIGNED NULL,
  Previous_TakerPaysIssuer INT UNSIGNED NULL,
  Previous_TakerGets BINARY(8) NULL,
  Previous_TakerGetsCurrency INT UNSIGNED NULL,
  Previous_TakerGetsIssuer INT UNSIGNED NULL,
  Previous_Expiration INT UNSIGNED,
  Previous_BookDirectory BINARY(32) NULL,
  Previous_BookNode BIGINT NULL,
  Previous_OwnerNode BIGINT NULL,
  PRIMARY KEY(LedgerSequence,TransactionIndex,Position)
);
`, `
CREATE TABLE IF NOT EXISTS RippleState (
  LedgerSequence INT UNSIGNED NOT NULL,
  TransactionIndex INT UNSIGNED NOT NULL,
  Position MEDIUMINT UNSIGNED NOT NULL,
  Flags INT UNSIGNED NULL,
  Balance BINARY(8) NOT NULL,
  Currency INT UNSIGNED NOT NULL,
  LowLimit BINARY(8) NOT NULL,
  LowLimitIssuer INT UNSIGNED NOT NULL,
  HighLimit BINARY(8) NOT NULL,
  HighLimitIssuer INT UNSIGNED NOT NULL,
  LowNode BIGINT UNSIGNED NULL,
  HighNode BIGINT UNSIGNED NULL,
  LowQualityIn INT UNSIGNED NULL,
  LowQualityOut INT UNSIGNED NULL,
  HighQualityIn INT UNSIGNED NULL,
  HighQualityOut INT UNSIGNED NULL,
  Previous_Flags INT UNSIGNED NULL,
  Previous_Balance BINARY(8) NULL,
  Previous_Currency INT UNSIGNED NULL,
  Previous_LowLimit BINARY(8) NULL,
  Previous_LowLimitIssuer INT UNSIGNED NULL,
  Previous_HighLimit BINARY(8) NULL,
  Previous_HighLimitIssuer INT UNSIGNED NULL,
  Previous_LowNode BIGINT UNSIGNED NULL,
  Previous_HighNode BIGINT UNSIGNED NULL,
  Previous_LowQualityIn INT UNSIGNED NULL,
  Previous_LowQualityOut INT UNSIGNED NULL,
  Previous_HighQualityIn INT UNSIGNED NULL,
  Previous_HighQualityOut INT UNSIGNED NULL,
  PRIMARY KEY(LedgerSequence,TransactionIndex,Position)
);
`, `
CREATE TABLE IF NOT EXISTS Directory (
  LedgerSequence INT UNSIGNED NOT NULL,
  TransactionIndex INT UNSIGNED NOT NULL,
  Position MEDIUMINT UNSIGNED NOT NULL,
  RootIndex BINARY(32) NOT NULL,
  Indexes BLOB NULL,
  Owner INT UNSIGNED NULL,
  TakerPaysCurrency INT UNSIGNED NULL,
  TakerPaysIssuer INT UNSIGNED NULL,
  TakerGetsCurrency INT UNSIGNED NULL,
  TakerGetsIssuer INT UNSIGNED NULL,
  ExchangeRate BINARY(8) NULL,
  IndexNext BIGINT UNSIGNED NULL,
  IndexPrevious BIGINT UNSIGNED NULL,
  Previous_RootIndex BINARY(32) NULL,
  Previous_Indexes BLOB NULL,
  Previous_Owner INT UNSIGNED NULL,
  Previous_TakerPaysCurrency INT UNSIGNED NULL,
  Previous_TakerPaysIssuer INT UNSIGNED NULL,
  Previous_TakerGetsCurrency INT UNSIGNED NULL,
  Previous_TakerGetsIssuer INT UNSIGNED NULL,
  Previous_ExchangeRate BINARY(8) NULL,
  Previous_IndexNext BIGINT UNSIGNED NULL,
  Previous_IndexPrevious BIGINT UNSIGNED NULL,
  PRIMARY KEY(LedgerSequence,TransactionIndex,Position)
);
`, `
CREATE TABLE IF NOT EXISTS FeeSettings (
  LedgerSequence INT UNSIGNED NOT NULL,
  TransactionIndex INT UNSIGNED NOT NULL,
  Position MEDIUMINT UNSIGNED NOT NULL,
  Flags INT UNSIGNED NOT NULL,
  BaseFee BIGINT UNSIGNED NULL,
  ReferenceFeeUnits BIGINT UNSIGNED NULL,
  ReserveBase  BIGINT UNSIGNED NULL,
  ReserveIncrement BIGINT UNSIGNED NULL,
  Previous_Flags INT UNSIGNED NOT NULL,
  Previous_BaseFee BIGINT UNSIGNED NULL,
  Previous_ReferenceFeeUnits BIGINT UNSIGNED NULL,
  Previous_ReserveBase  BIGINT UNSIGNED NULL,
  Previous_ReserveIncrement BIGINT UNSIGNED NULL,
  PRIMARY KEY(LedgerSequence,TransactionIndex,Position)
);
`}

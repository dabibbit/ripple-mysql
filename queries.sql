select `ledger`.`LedgerSequence` AS `LedgerSequence`,cast(conv(hex(`ledger`.`TotalXRP`),16,10) as unsigned) AS `XRP` from `ledger`
select * FROM(select x2.LedgerSequence+1,x1.XRP-x2.XRP as Fees from XRP x1 INNER JOIN XRP x2 on x1.LedgerSequence=x2.LedgerSequence-1)f where Fees<>0;

select LedgerSequence,count(*),sum(cast(conv(hex(Fee),16,10)as unsigned)&0xBFFFFFFFFFFFFFFF) as sum from Payment group by LedgerSequence having sum>=116056 order by sum;

-- MySQL voodoo to find missing transactions
REPLACE INTO MissingTransactions 
SELECT l.LedgerSequence
FROM Ledger l,
(SELECT @TotalFees=0,@LastXRP :=100000000000000000,@LastLedger:=0)v
WHERE (@TotalFees:=IF(l.LedgerSequence=@LastLedger+1,@LastXRP-l.TotalXRP,0)) IS NOT NULL
AND (@LastLedger:=l.LedgerSequence) IS NOT NULL
AND (@LastXRP:=l.TotalXRP) IS NOT NULL
AND Fees<>@TotalFees
ORDER BY l.LedgerSequence;

-- Get Missing transactions
SELECT l.LedgerSequence,
hex(l.TransactionHash)
FROM Ledger l
INNER JOIN MissingTransactions m
ON l.LedgerSequence=m.LedgerSequence
WHERE l.LedgerSequence BETWEEN 4000000 AND 5000000
LIMIT 100

SELECT p.LedgerSequence,
p.TransactionIndex,
p.Pathset,
p.Position,
a.Human AS Account,
c.Human AS Currency,
i.Human AS Issuer 
FROM Path p
LEFT OUTER JOIN Account a ON p.Account=a.Id
LEFT OUTER JOIN Currency c ON p.Currency=c.Id
LEFT OUTER JOIN Account i ON p.Issuer=i.Id;

Select Human,
MIN(LedgerSequence) as First,
Max(LedgerSequence) as Last,
Format(MAX(Balance)/1000000,2) as Max,
Format(MIN(Balance)/1000000,2) AS Min,
count(*)  
from Account,AccountRoot 
where Account.Id=AccountRoot.Account 
group by AccountRoot.Account 
order by max(Balance);

select Human,
SUM(IF(TransactionType=0,1,0)) AS Payment,
SUM(IF(TransactionType=3,1,0)) AS AccountSet,
SUM(IF(TransactionType=5,1,0)) AS SetRegularKey,
SUM(IF(TransactionType=7,1,0)) AS OfferCreate,
SUM(IF(TransactionType=8,1,0)) AS OfferCancel,
SUM(IF(TransactionType=20,1,0)) AS TrustSet,
SUM(IF(TransactionType=100,1,0)) AS Feature,
SUM(IF(TransactionType=101,1,0)) AS FeeSet,
Count(*) As Count 
from Transaction,Account
where Transaction.Account=Account.Id
group by Transaction.Account
order by count(*);

-- Get Node data
select concat(lpad(hex(LedgerSequence),8,'0'),lpad(hex(LedgerSequence),8,'0'),'04534E4400',hex(Raw),hex(Hash)) 
from Transaction 
order by LedgerSequence,TransactionIndex
limit 1000;

DELIMITER $
DROP FUNCTION IF EXISTS amount2$
CREATE FUNCTION amount2(a BINARY(8)) 
RETURNS DECIMAL(65,30) DETERMINISTIC
BEGIN
DECLARE raw BIGINT UNSIGNED;
DECLARE value DECIMAL(65,30);
DECLARE scale DECIMAL(65,30);
SET raw = CONV(HEX(a),16,10);
IF raw&0x3FFFFFFFFFFFFFFF=0 THEN
    RETURN 0;
END IF;
IF (raw>>63)= 1 THEN 
    SET value = raw&0x3FFFFFFFFFFFFF;
    SET scale = POW(10,CAST((raw>>54)&0xFF AS SIGNED)-97);
    SET value = value*scale;
ELSE
    SET value = raw&0x3FFFFFFFFFFFFFFF;
    SET value = value/1000000;
END IF;
IF (raw>>62 &1)=0 THEN 
    SET value=-value;
END IF;
RETURN value;
END$
DELIMITER ;
SELECT amount(UNHEX('D4838D7EA4C68000'));
SELECT amount(UNHEX('4000000029B92700'));
SELECT amount(UNHEX('8000000000000000'));
SELECT amount(UNHEX('CD860A24181E4000'));

DELIMITER $
DROP FUNCTION IF EXISTS amount2$
CREATE FUNCTION amount2(a BINARY(8)) 
RETURNS DECIMAL(65,30) DETERMINISTIC
BEGIN
DECLARE raw BIGINT UNSIGNED;
DECLARE value DECIMAL(65,30);
DECLARE scale INT SIGNED;
SET raw = CONV(HEX(a),16,10);
IF raw&0x3FFFFFFFFFFFFFFF=0 THEN
    RETURN 0;
END IF;
IF (raw>>63)= 1 THEN 
    SET value = raw&0x3FFFFFFFFFFFFF;
    SET scale = CAST((raw>>54)&0xFF AS SIGNED)-97;
    WHILE scale>0 DO
        SET value=value*10;
        SET scale=scale-1;
    END WHILE;
    WHILE scale<0 DO
        SET value=value/10;
        SET scale=scale+1;
    END WHILE;
ELSE
    SET value = raw&0x3FFFFFFFFFFFFFFF;
    SET value = value/1000000;
END IF;
IF (raw>>62 &1)=0 THEN 
    SET value=-value;
END IF;
RETURN value;
END$
DELIMITER ;
SELECT amount(UNHEX('D4838D7EA4C68000'));
SELECT amount(UNHEX('4000000029B92700'));
SELECT amount(UNHEX('8000000000000000'));
SELECT amount(UNHEX('CD860A24181E4000'));

000000000000000000000000000000

SELECT c.Human,i.Human,a.Sum,a.Count
FROM(
 SELECT AmountCurrency,
    AmountIssuer,
    sum_amount(Amount) AS Sum,
    COUNT(*) AS Count
    FROM Payment p
    INNER JOIN Transaction t
    ON p.LedgerSequence = t.LedgerSequence
    AND p.TransactionIndex = t.TransactionIndex
    WHERE t.TransactionResult = 0
    GROUP BY AmountCurrency,AmountIssuer 
    HAVING COUNT(*)>10
 )a
INNER JOIN Currency c
ON a.AmountCurrency =c.Id
INNER JOIN Account i
ON a.AmountIssuer = i.Id
ORDER BY a.Count;

-----

SELECT * 
FROM (
    SELECT HEX(Hash) AS Hash,
    LedgerSequence,
    TransactionIndex,
    TransactionType,
    FROM Transaction
    UNION ALL 
    SELECT HEX(Hash),
    LedgerSequence,
    NULL,
    NULL,
    FROM Ledger
)c ORDER BY LedgerSequence,TransactionIndex;


select Human,MAX(t.Account),MAX(Sequence)-count(*) from Transaction t ,Account a where t.Account=a.Id And t.Account<>0 group by Human having MAX(Sequence)<>COUNT(*) order by count(*);

SELECT LedgerSequence,
COUNT(*) AS Total,
SUM(IF(TransactionType=0,1,0)) AS Payment,
SUM(IF(TransactionType=3,1,0)) AS AccountSet,
SUM(IF(TransactionType=5,1,0)) AS SetRegularKey,
SUM(IF(TransactionType=7,1,0)) AS OfferCreate,
SUM(IF(TransactionType=8,1,0)) AS OfferCancel,
SUM(IF(TransactionType=20,1,0)) AS TrustSet,
SUM(IF(TransactionType=100,1,0)) AS Amendment,
SUM(IF(TransactionType=101,1,0)) AS FeeSet
FROM Transaction
GROUP BY LedgerSequence
HAVING Total>100;


--Time breakdown for Transaction Types
SELECT MIN(LedgerSequence),
MAX(LedgerSequence),
Date,
Hour,
TransactionType,
TransactionResult,
COUNT(*)
FROM
(
SELECT TransactionType,
TransactionResult,
FROM Transaction 
GROUP BY TransactionResult,TransactionType
ORDER BY TRansactionType,TransactionResult;

    SELECT l.LedgerSequence,
    DATE(FROM_UNIXTIME(CloseTime+946684800)) AS Date,
    HOUR(FROM_UNIXTIME(CloseTime+946684800)) AS Hour,
    t.TransactionType,
    t.TransactionResult
    FROM Ledger l
    LEFT OUTER JOIN Transaction t 
    ON l.LedgerSequence=t.LedgerSequence
)s
GROUP BY Date,TransactionType,TransactionResult











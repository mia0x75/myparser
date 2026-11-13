package types

import (
	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/myparser/sqlerr"
)

var (
	// ErrDataTooLong is returned when converts a string value that is longer than field type length.
	ErrDataTooLong = sqlerr.ClassTypes.New(codeDataTooLong, "Data Too Long")
	// ErrIllegalValueForType is returned when value of type is illegal.
	ErrIllegalValueForType = sqlerr.ClassTypes.New(codeIllegalValueForType, mysql.MySQLErrName[mysql.ErrIllegalValueForType])
	// ErrTruncated is returned when data has been truncated during conversion.
	ErrTruncated = sqlerr.ClassTypes.New(codeTruncated, "Data Truncated")
	// ErrTruncatedWrongVal is returned when data has been truncated during conversion.
	ErrTruncatedWrongVal = sqlerr.ClassTypes.New(codeTruncatedWrongValue, msgTruncatedWrongVal)
	// ErrOverflow is returned when data is out of range for a field type.
	ErrOverflow = sqlerr.ClassTypes.New(codeOverflow, msgOverflow)
	// ErrDivByZero is return when do division by 0.
	ErrDivByZero = sqlerr.ClassTypes.New(codeDivByZero, "Division by 0")
	// ErrTooBigDisplayWidth is return when display width out of range for column.
	ErrTooBigDisplayWidth = sqlerr.ClassTypes.New(codeTooBigDisplayWidth, "Too Big Display width")
	// ErrTooBigFieldLength is return when column length too big for column.
	ErrTooBigFieldLength = sqlerr.ClassTypes.New(codeTooBigFieldLength, "Too Big Field length")
	// ErrTooBigSet is returned when too many strings for column.
	ErrTooBigSet = sqlerr.ClassTypes.New(codeTooBigSet, "Too Big Set")
	// ErrTooBigScale is returned when type DECIMAL/NUMERIC scale is bigger than mysql.MaxDecimalScale.
	ErrTooBigScale = sqlerr.ClassTypes.New(codeTooBigScale, mysql.MySQLErrName[mysql.ErrTooBigScale])
	// ErrTooBigPrecision is returned when type DECIMAL/NUMERIC precision is bigger than mysql.MaxDecimalWidth
	ErrTooBigPrecision = sqlerr.ClassTypes.New(codeTooBigPrecision, mysql.MySQLErrName[mysql.ErrTooBigPrecision])
	// ErrWrongFieldSpec is return when incorrect column specifier for column.
	ErrWrongFieldSpec = sqlerr.ClassTypes.New(codeWrongFieldSpec, "Wrong Field Spec")
	// ErrBadNumber is return when parsing an invalid binary decimal number.
	ErrBadNumber = sqlerr.ClassTypes.New(codeBadNumber, "Bad Number")
	// ErrInvalidDefault is returned when meet a invalid default value.
	// ErrInvalidDefault = ErrInvalidDefault
	// ErrCastAsSignedOverflow is returned when positive out-of-range integer, and convert to it's negative complement.
	ErrCastAsSignedOverflow = sqlerr.ClassTypes.New(codeUnknown, msgCastAsSignedOverflow)
	// ErrCastNegIntAsUnsigned is returned when a negative integer be casted to an unsigned int.
	ErrCastNegIntAsUnsigned = sqlerr.ClassTypes.New(codeUnknown, msgCastNegIntAsUnsigned)
	// ErrMBiggerThanD is returned when precision less than the scale.
	ErrMBiggerThanD = sqlerr.ClassTypes.New(codeMBiggerThanD, mysql.MySQLErrName[mysql.ErrMBiggerThanD])
	// ErrWarnDataOutOfRange is returned when the value in a numeric column that is outside the permissible range of the column data type.
	// See https://dev.mysql.com/doc/refman/5.5/en/out-of-range-and-overflow.html for details
	ErrWarnDataOutOfRange = sqlerr.ClassTypes.New(codeDataOutOfRange, mysql.MySQLErrName[mysql.ErrWarnDataOutOfRange])
	// ErrDuplicatedValueInType is returned when enum column has duplicated value.
	ErrDuplicatedValueInType = sqlerr.ClassTypes.New(codeDuplicatedValueInType, mysql.MySQLErrName[mysql.ErrDuplicatedValueInType])
)

const (
	codeBadNumber sqlerr.ErrCode = 1

	codeDataTooLong           = sqlerr.ErrCode(mysql.ErrDataTooLong)
	codeIllegalValueForType   = sqlerr.ErrCode(mysql.ErrIllegalValueForType)
	codeTruncated             = sqlerr.ErrCode(mysql.WarnDataTruncated)
	codeOverflow              = sqlerr.ErrCode(mysql.ErrDataOutOfRange)
	codeDivByZero             = sqlerr.ErrCode(mysql.ErrDivisionByZero)
	codeTooBigDisplayWidth    = sqlerr.ErrCode(mysql.ErrTooBigDisplaywidth)
	codeTooBigFieldLength     = sqlerr.ErrCode(mysql.ErrTooBigFieldlength)
	codeTooBigSet             = sqlerr.ErrCode(mysql.ErrTooBigSet)
	codeTooBigScale           = sqlerr.ErrCode(mysql.ErrTooBigScale)
	codeTooBigPrecision       = sqlerr.ErrCode(mysql.ErrTooBigPrecision)
	codeWrongFieldSpec        = sqlerr.ErrCode(mysql.ErrWrongFieldSpec)
	codeTruncatedWrongValue   = sqlerr.ErrCode(mysql.ErrTruncatedWrongValue)
	codeUnknown               = sqlerr.ErrCode(mysql.ErrUnknown)
	codeInvalidDefault        = sqlerr.ErrCode(mysql.ErrInvalidDefault)
	codeMBiggerThanD          = sqlerr.ErrCode(mysql.ErrMBiggerThanD)
	codeDataOutOfRange        = sqlerr.ErrCode(mysql.ErrWarnDataOutOfRange)
	codeDuplicatedValueInType = sqlerr.ErrCode(mysql.ErrDuplicatedValueInType)
)

var (
	msgOverflow             = mysql.MySQLErrName[mysql.ErrDataOutOfRange]
	msgTruncatedWrongVal    = mysql.MySQLErrName[mysql.ErrTruncatedWrongValue]
	msgCastAsSignedOverflow = "Cast to signed converted positive out-of-range integer to it's negative complement"
	msgCastNegIntAsUnsigned = "Cast to unsigned converted negative integer to it's positive complement"
)

func init() {
	typesMySQLErrCodes := map[sqlerr.ErrCode]uint16{
		codeDataTooLong:           mysql.ErrDataTooLong,
		codeIllegalValueForType:   mysql.ErrIllegalValueForType,
		codeTruncated:             mysql.WarnDataTruncated,
		codeOverflow:              mysql.ErrDataOutOfRange,
		codeDivByZero:             mysql.ErrDivisionByZero,
		codeTooBigDisplayWidth:    mysql.ErrTooBigDisplaywidth,
		codeTooBigFieldLength:     mysql.ErrTooBigFieldlength,
		codeTooBigSet:             mysql.ErrTooBigSet,
		codeTooBigScale:           mysql.ErrTooBigScale,
		codeTooBigPrecision:       mysql.ErrTooBigPrecision,
		codeWrongFieldSpec:        mysql.ErrWrongFieldSpec,
		codeTruncatedWrongValue:   mysql.ErrTruncatedWrongValue,
		codeUnknown:               mysql.ErrUnknown,
		codeInvalidDefault:        mysql.ErrInvalidDefault,
		codeMBiggerThanD:          mysql.ErrMBiggerThanD,
		codeDataOutOfRange:        mysql.ErrWarnDataOutOfRange,
		codeDuplicatedValueInType: mysql.ErrDuplicatedValueInType,
	}
	sqlerr.ErrClassToMySQLCodes[sqlerr.ClassTypes] = typesMySQLErrCodes
}

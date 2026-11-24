package types

import (
	"github.com/mia0x75/myparser/mysql"
)

var (
	// ErrDataTooLong is returned when converts a string value that is longer than field type length.
	ErrDataTooLong = mysql.ClassTypes.New(codeDataTooLong, "Data Too Long")
	// ErrIllegalValueForType is returned when value of type is illegal.
	ErrIllegalValueForType = mysql.ClassTypes.New(codeIllegalValueForType, mysql.MySQLErrName[mysql.ErrIllegalValueForType])
	// ErrTruncated is returned when data has been truncated during conversion.
	ErrTruncated = mysql.ClassTypes.New(codeTruncated, "Data Truncated")
	// ErrTruncatedWrongVal is returned when data has been truncated during conversion.
	ErrTruncatedWrongVal = mysql.ClassTypes.New(codeTruncatedWrongValue, msgTruncatedWrongVal)
	// ErrOverflow is returned when data is out of range for a field type.
	ErrOverflow = mysql.ClassTypes.New(codeOverflow, msgOverflow)
	// ErrDivByZero is return when do division by 0.
	ErrDivByZero = mysql.ClassTypes.New(codeDivByZero, "Division by 0")
	// ErrTooBigDisplayWidth is return when display width out of range for column.
	ErrTooBigDisplayWidth = mysql.ClassTypes.New(codeTooBigDisplayWidth, "Too Big Display width")
	// ErrTooBigFieldLength is return when column length too big for column.
	ErrTooBigFieldLength = mysql.ClassTypes.New(codeTooBigFieldLength, "Too Big Field length")
	// ErrTooBigSet is returned when too many strings for column.
	ErrTooBigSet = mysql.ClassTypes.New(codeTooBigSet, "Too Big Set")
	// ErrTooBigScale is returned when type DECIMAL/NUMERIC scale is bigger than mysql.MaxDecimalScale.
	ErrTooBigScale = mysql.ClassTypes.New(codeTooBigScale, mysql.MySQLErrName[mysql.ErrTooBigScale])
	// ErrTooBigPrecision is returned when type DECIMAL/NUMERIC precision is bigger than mysql.MaxDecimalWidth
	ErrTooBigPrecision = mysql.ClassTypes.New(codeTooBigPrecision, mysql.MySQLErrName[mysql.ErrTooBigPrecision])
	// ErrWrongFieldSpec is return when incorrect column specifier for column.
	ErrWrongFieldSpec = mysql.ClassTypes.New(codeWrongFieldSpec, "Wrong Field Spec")
	// ErrBadNumber is return when parsing an invalid binary decimal number.
	ErrBadNumber = mysql.ClassTypes.New(codeBadNumber, "Bad Number")
	// ErrInvalidDefault is returned when meet a invalid default value.
	// ErrInvalidDefault = ErrInvalidDefault
	// ErrCastAsSignedOverflow is returned when positive out-of-range integer, and convert to it's negative complement.
	ErrCastAsSignedOverflow = mysql.ClassTypes.New(codeUnknown, msgCastAsSignedOverflow)
	// ErrCastNegIntAsUnsigned is returned when a negative integer be casted to an unsigned int.
	ErrCastNegIntAsUnsigned = mysql.ClassTypes.New(codeUnknown, msgCastNegIntAsUnsigned)
	// ErrMBiggerThanD is returned when precision less than the scale.
	ErrMBiggerThanD = mysql.ClassTypes.New(codeMBiggerThanD, mysql.MySQLErrName[mysql.ErrMBiggerThanD])
	// ErrWarnDataOutOfRange is returned when the value in a numeric column that is outside the permissible range of the column data type.
	// See https://dev.mysql.com/doc/refman/5.5/en/out-of-range-and-overflow.html for details
	ErrWarnDataOutOfRange = mysql.ClassTypes.New(codeDataOutOfRange, mysql.MySQLErrName[mysql.ErrWarnDataOutOfRange])
	// ErrDuplicatedValueInType is returned when enum column has duplicated value.
	ErrDuplicatedValueInType = mysql.ClassTypes.New(codeDuplicatedValueInType, mysql.MySQLErrName[mysql.ErrDuplicatedValueInType])
)

const (
	codeBadNumber mysql.ErrCode = 1

	codeDataTooLong           = mysql.ErrCode(mysql.ErrDataTooLong)
	codeIllegalValueForType   = mysql.ErrCode(mysql.ErrIllegalValueForType)
	codeTruncated             = mysql.ErrCode(mysql.WarnDataTruncated)
	codeOverflow              = mysql.ErrCode(mysql.ErrDataOutOfRange)
	codeDivByZero             = mysql.ErrCode(mysql.ErrDivisionByZero)
	codeTooBigDisplayWidth    = mysql.ErrCode(mysql.ErrTooBigDisplaywidth)
	codeTooBigFieldLength     = mysql.ErrCode(mysql.ErrTooBigFieldlength)
	codeTooBigSet             = mysql.ErrCode(mysql.ErrTooBigSet)
	codeTooBigScale           = mysql.ErrCode(mysql.ErrTooBigScale)
	codeTooBigPrecision       = mysql.ErrCode(mysql.ErrTooBigPrecision)
	codeWrongFieldSpec        = mysql.ErrCode(mysql.ErrWrongFieldSpec)
	codeTruncatedWrongValue   = mysql.ErrCode(mysql.ErrTruncatedWrongValue)
	codeUnknown               = mysql.ErrCode(mysql.ErrUnknown)
	codeInvalidDefault        = mysql.ErrCode(mysql.ErrInvalidDefault)
	codeMBiggerThanD          = mysql.ErrCode(mysql.ErrMBiggerThanD)
	codeDataOutOfRange        = mysql.ErrCode(mysql.ErrWarnDataOutOfRange)
	codeDuplicatedValueInType = mysql.ErrCode(mysql.ErrDuplicatedValueInType)
)

var (
	msgOverflow             = mysql.MySQLErrName[mysql.ErrDataOutOfRange]
	msgTruncatedWrongVal    = mysql.MySQLErrName[mysql.ErrTruncatedWrongValue]
	msgCastAsSignedOverflow = "Cast to signed converted positive out-of-range integer to it's negative complement"
	msgCastNegIntAsUnsigned = "Cast to unsigned converted negative integer to it's positive complement"
)

func init() {
	typesMySQLErrCodes := map[mysql.ErrCode]uint16{
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
	mysql.ErrClassToMySQLCodes[mysql.ClassTypes] = typesMySQLErrCodes
}

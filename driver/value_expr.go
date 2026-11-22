package driver

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mia0x75/myparser/hack"
	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/myparser/types"
)

// NewDecimal preserves MySQL truncation and overflow diagnostics.
func NewDecimal(str string) (any, error) {
	dec := new(types.MyDecimal)
	err := dec.FromString(hack.Slice(str))
	switch {
	case types.ErrTruncated.Equal(err):
		err = nil
	case types.ErrOverflow.Equal(err):
		truncated := dec.String()
		negative := strings.HasPrefix(strings.TrimLeft(str, " \t"), "-")
		dec = types.NewMaxOrMinDec(negative, mysql.MaxDecimalWidth, 0)
		err = mysql.NewErr(mysql.ErrTruncatedWrongValue, "DECIMAL", truncated)
	}
	return dec, err
}

// NewHexLiteral keeps dialect decoding explicit instead of globally registered.
func NewHexLiteral(str string) (any, error) {
	return types.NewHexLiteral(str)
}

// NewBitLiteral keeps dialect decoding explicit instead of globally registered.
func NewBitLiteral(str string) (any, error) {
	return types.NewBitLiteral(str)
}

// ValueExpr is a runtime execution value, not an AST node.
// It is a temporary bridge: it must be removed from the AST/Visitor boundary
// before Phase 3 is considered complete.
type ValueExpr struct {
	types.Datum
	Type             types.FieldType
	flag             uint64
	projectionOffset int
}

// SetType preserves legacy literal metadata outside the shared syntax core.
func (n *ValueExpr) SetType(tp *types.FieldType) { n.Type = *tp }

// GetType preserves the legacy literal accessor during AST migration.
func (n *ValueExpr) GetType() *types.FieldType { return &n.Type }

// SetFlag keeps the legacy syntax cache local to the dialect.
func (n *ValueExpr) SetFlag(flag uint64) { n.flag = flag }

// GetFlag keeps the legacy syntax cache local to the dialect.
func (n *ValueExpr) GetFlag() uint64 { return n.flag }

// GetDatumString preserves the dialect ValueExpr string accessor.
func (n *ValueExpr) GetDatumString() string {
	return n.GetString()
}

// Format renders the value to a writer for diagnostic use.
func (n *ValueExpr) Format(w io.Writer) {
	var s string
	switch n.Kind() {
	case types.KindNull:
		s = "NULL"
	case types.KindInt64:
		if n.Type.Flag&mysql.IsBooleanFlag != 0 {
			if n.GetInt64() > 0 {
				s = "TRUE"
			} else {
				s = "FALSE"
			}
		} else {
			s = strconv.FormatInt(n.GetInt64(), 10)
		}
	case types.KindUint64:
		s = strconv.FormatUint(n.GetUint64(), 10)
	case types.KindFloat32:
		s = strconv.FormatFloat(n.GetFloat64(), 'e', -1, 32)
	case types.KindFloat64:
		s = strconv.FormatFloat(n.GetFloat64(), 'e', -1, 64)
	case types.KindString, types.KindBytes:
		s = strconv.Quote(n.GetString())
	case types.KindMysqlDecimal:
		s = decimalString(n.GetMysqlDecimal())
	case types.KindBinaryLiteral:
		if n.Type.Flag&mysql.UnsignedFlag != 0 {
			s = fmt.Sprintf("x'%x'", n.GetBytes())
		} else {
			s = n.GetBinaryLiteral().ToBitLiteralString(true)
		}
		if n.Type.Flag&mysql.HasCharsetIntroducerFlag != 0 {
			s = "_" + n.Type.Charset + " " + s
		}
	default:
		var err error
		s, err = n.ToString()
		if err != nil {
			s = fmt.Sprint(n.GetValue())
		}
	}
	_, _ = fmt.Fprint(w, s)
}

func decimalString(dec *types.MyDecimal) string {
	str := dec.String()
	if dec.GetDigitsInt() != 0 || dec.GetDigitsFrac() <= 72 {
		return str
	}
	// A leading zero consumes one word and would truncate 73-81 digit fractions when reparsed.
	if dec.IsNegative() {
		return "-" + strings.TrimPrefix(str, "-0")
	}
	return strings.TrimPrefix(str, "0")
}

// NewValueExpr retains existing literal pointers and assigns dialect defaults.
func NewValueExpr(value any) *ValueExpr {
	if ve, ok := value.(*ValueExpr); ok {
		return ve
	}
	ve := &ValueExpr{}
	ve.SetValue(value)
	types.DefaultTypeForValue(value, &ve.Type)
	ve.projectionOffset = -1
	return ve
}

// SetProjectionOffset sets ValueExpr.projectionOffset for logical plan builder.
func (n *ValueExpr) SetProjectionOffset(offset int) {
	n.projectionOffset = offset
}

// GetProjectionOffset returns ValueExpr.projectionOffset.
func (n *ValueExpr) GetProjectionOffset() int {
	return n.projectionOffset
}

package driver

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/hack"
	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/myparser/types"
)

// The driver package injects value-expression constructors into ast so that
// ast does not depend on the concrete definition of types.Datum. The root
// package depends on ast but not on types; importing driver wires the two
// together.

func init() {
	ast.NewValueExpr = newValueExpr
	ast.NewParamMarkerExpr = newParamMarkerExpr
	ast.NewDecimal = func(str string) (any, error) {
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
	ast.NewHexLiteral = func(str string) (any, error) {
		h, err := types.NewHexLiteral(str)
		return h, err
	}
	ast.NewBitLiteral = func(str string) (any, error) {
		b, err := types.NewBitLiteral(str)
		return b, err
	}
}

var (
	_ ast.ParamMarkerExpr = &ParamMarkerExpr{}
	_ ast.ValueExpr       = &ValueExpr{}
)

// ValueExpr is the simple value expression.
type ValueExpr struct {
	ast.TexprNode
	types.Datum
	projectionOffset int
}

// Restore implements Node interface.
func (n *ValueExpr) Restore(ctx *format.RestoreCtx) error {
	switch n.Kind() {
	case types.KindNull:
		ctx.WriteKeyWord("NULL")
	case types.KindInt64:
		if n.Type.Flag&mysql.IsBooleanFlag != 0 {
			if n.GetInt64() > 0 {
				ctx.WriteKeyWord("TRUE")
			} else {
				ctx.WriteKeyWord("FALSE")
			}
		} else {
			ctx.WritePlain(strconv.FormatInt(n.GetInt64(), 10))
		}
	case types.KindUint64:
		ctx.WritePlain(strconv.FormatUint(n.GetUint64(), 10))
	case types.KindFloat32:
		ctx.WritePlain(strconv.FormatFloat(n.GetFloat64(), 'e', -1, 32))
	case types.KindFloat64:
		ctx.WritePlain(strconv.FormatFloat(n.GetFloat64(), 'e', -1, 64))
	case types.KindString:
		if n.Type.Charset != "" && n.Type.Charset != mysql.DefaultCharset {
			ctx.WritePlain("_")
			ctx.WriteKeyWord(n.Type.Charset)
		}
		ctx.WriteString(n.GetString())
	case types.KindBytes:
		ctx.WriteString(n.GetString())
	case types.KindMysqlDecimal:
		ctx.WritePlain(decimalString(n.GetMysqlDecimal()))
	case types.KindBinaryLiteral:
		if n.Type.Flag&mysql.HasCharsetIntroducerFlag != 0 {
			ctx.WritePlain("_")
			ctx.WriteKeyWord(n.Type.Charset)
			ctx.WritePlain(" ")
		}
		if n.Type.Flag&mysql.UnsignedFlag != 0 {
			ctx.WritePlainf("x'%x'", n.GetBytes())
		} else {
			ctx.WritePlain(n.GetBinaryLiteral().ToBitLiteralString(true))
		}
	case types.KindMysqlTime, types.KindMysqlDuration,
		types.KindMysqlEnum, types.KindMysqlSet:
		str, err := n.ToString()
		if err != nil {
			return errors.Trace(err)
		}
		ctx.WriteString(str)
	case types.KindMysqlBit:
		ctx.WritePlain(n.GetBinaryLiteral().ToBitLiteralString(true))
	case types.KindMysqlJSON:
		str, err := n.ToString()
		if err != nil {
			return errors.Trace(err)
		}
		ctx.WritePlain(str)
	case types.KindInterface, types.KindMinNotNull, types.KindMaxValue,
		types.KindRaw:
		// These kinds have no SQL literal form; they are internal sentinels
		// that should not appear in a restorable expression tree.
		return errors.New("not a restorable datum kind")
	default:
		return errors.New("can't format to string")
	}
	return nil
}

// GetDatumString implements the ast.ValueExpr interface.
func (n *ValueExpr) GetDatumString() string {
	return n.GetString()
}

// Format the ExprNode into a Writer.
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
		// Format cannot return an error; fall back to text instead of panicking.
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

// newValueExpr creates a ValueExpr with value, and sets default field type.
func newValueExpr(value any) ast.ValueExpr {
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

// Accept implements Node interface.
func (n *ValueExpr) Accept(v ast.Visitor) (ast.Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ValueExpr)
	return v.Leave(n)
}

// ParamMarkerExpr expression holds a place for another expression.
// Used in parsing prepare statement.
type ParamMarkerExpr struct {
	ValueExpr
	Offset int
	Order  int
}

// Restore implements Node interface.
func (n *ParamMarkerExpr) Restore(ctx *format.RestoreCtx) error {
	ctx.WritePlain("?")
	return nil
}

func newParamMarkerExpr(offset int) ast.ParamMarkerExpr {
	return &ParamMarkerExpr{
		Offset: offset,
	}
}

// Format the ExprNode into a Writer.
func (n *ParamMarkerExpr) Format(w io.Writer) {
	_, _ = fmt.Fprint(w, "?")
}

// Accept implements Node Accept interface.
func (n *ParamMarkerExpr) Accept(v ast.Visitor) (ast.Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ParamMarkerExpr)
	return v.Leave(n)
}

// SetOrder implements the ast.ParamMarkerExpr interface.
func (n *ParamMarkerExpr) SetOrder(order int) {
	n.Order = order
}

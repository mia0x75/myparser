package parser

import (
	"math"
	"strconv"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/opcode"
)

func sequenceSignedInt(l yyLexer, value interface{}, negative bool, span loc.Span) (ast.ExprNode, error) {
	var magnitude uint64
	switch v := value.(type) {
	case int64:
		if v < 0 {
			return nil, errors.New("expected an unsigned integer literal")
		}
		magnitude = uint64(v)
	case uint64:
		magnitude = v
	default:
		return nil, errors.New("expected an integer sequence value")
	}
	limit := uint64(math.MaxInt64)
	if negative {
		limit++
	}
	if magnitude > limit {
		return nil, errors.New("signed sequence integer out of range")
	}
	if negative {
		if magnitude == uint64(math.MaxInt64)+1 {
			return syntaxLiteral(l, pkast.LiteralInteger, strconv.FormatInt(math.MinInt64, 10), span), nil
		}
		return syntaxLiteral(l, pkast.LiteralInteger, strconv.FormatInt(-int64(magnitude), 10), span), nil
	}
	return syntaxLiteral(l, pkast.LiteralInteger, strconv.FormatUint(magnitude, 10), span), nil
}

func makeSequenceOption(tp ast.SequenceOptionType, value ast.ExprNode) *ast.SequenceOption {
	option := &ast.SequenceOption{Tp: tp, Value: value}
	negative := false
	if unary, ok := value.(*ast.UnaryOperationExpr); ok && (unary.Op == pkast.Operator(opcode.Plus) || unary.Op == pkast.Operator(opcode.Minus)) {
		negative = unary.Op == pkast.Operator(opcode.Minus)
		value = unary.Expr
	}
	if literal, ok := value.(*ast.LiteralExpr); ok && literal.Kind == pkast.LiteralInteger {
		if v, err := strconv.ParseInt(literal.Value, 10, 64); err == nil {
			option.Val = v
		} else if negative && literal.Value == "9223372036854775808" {
			option.Val = math.MinInt64
			return option
		}
	}
	if negative {
		option.Val = -option.Val
	}
	return option
}

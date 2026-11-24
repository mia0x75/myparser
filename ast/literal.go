package ast

import (
	"io"
	"strconv"
	"strings"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"

	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/mysql"
)

// LiteralExpr keeps literal syntax without an evaluated value or inferred type.
type LiteralExpr struct {
	pkast.BaseLiteralExpr
	Introducer lex.Ident
	Quote      lex.QuoteStyle
}

// Restore normalizes literal spelling without consulting connection metadata.
func (n *LiteralExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return nilRestoreError("LiteralExpr")
	}
	switch n.Kind {
	case pkast.LiteralNull:
		ctx.WriteKeyword("NULL")
	case pkast.LiteralBoolean:
		// Anything but the two boolean keywords must not be emitted as SQL.
		switch {
		case strings.EqualFold(n.Value, "true"):
			ctx.WriteKeyword("TRUE")
		case strings.EqualFold(n.Value, "false"):
			ctx.WriteKeyword("FALSE")
		default:
			return errors.Errorf("unsupported boolean literal value %q", n.Value)
		}
	case pkast.LiteralInteger, pkast.LiteralDecimal, pkast.LiteralFloat:
		if !validNumericLiteral(n.Value) {
			return errors.Errorf("invalid numeric literal %q", n.Value)
		}
		ctx.WritePlain(n.Value)
	case pkast.LiteralString, pkast.LiteralBytes:
		if err := validateBareIdentifier("charset introducer", n.Introducer); err != nil {
			return err
		}
		// Keep legacy canonical output; the explicit introducer remains in syntax.
		if n.Introducer.Present() && n.Introducer.Name != mysql.DefaultCharset {
			ctx.WritePlain("_")
			ctx.WriteKeyword(n.Introducer.Name)
		}
		ctx.WriteString(n.Value)
	case pkast.LiteralHex, pkast.LiteralBit:
		if err := validateBareIdentifier("charset introducer", n.Introducer); err != nil {
			return err
		}
		for _, digit := range n.Value {
			valid := digit == '0' || digit == '1'
			if n.Kind == pkast.LiteralHex {
				valid = digit >= '0' && digit <= '9' || digit >= 'a' && digit <= 'f' || digit >= 'A' && digit <= 'F'
			}
			if !valid {
				return errors.New("invalid binary literal digits")
			}
		}
		if n.Introducer.Present() {
			ctx.WritePlain("_")
			ctx.WriteKeyword(n.Introducer.Name)
			ctx.WritePlain(" ")
		}
		if n.Kind == pkast.LiteralHex {
			ctx.WritePlain("x'")
			if len(n.Value)%2 != 0 {
				ctx.WritePlain("0")
			}
			ctx.WritePlain(strings.ToLower(n.Value))
		} else {
			ctx.WritePlain("b'")
			value := strings.TrimLeft(n.Value, "0")
			if value == "" && n.Value != "" {
				value = "0"
			}
			ctx.WritePlain(value)
		}
		ctx.WritePlain("'")
	default:
		return errors.New("Unsupported syntax literal kind")
	}
	return ctx.Err()
}

func validNumericLiteral(value string) bool {
	if value == "" {
		return false
	}
	i := 0
	if value[i] == '+' || value[i] == '-' {
		i++
	}
	integerStart := i
	for i < len(value) && value[i] >= '0' && value[i] <= '9' {
		i++
	}
	integerDigits := i - integerStart
	fractionDigits := 0
	if i < len(value) && value[i] == '.' {
		i++
		fractionStart := i
		for i < len(value) && value[i] >= '0' && value[i] <= '9' {
			i++
		}
		fractionDigits = i - fractionStart
	}
	if integerDigits == 0 && fractionDigits == 0 {
		return false
	}
	if i < len(value) && (value[i] == 'e' || value[i] == 'E') {
		i++
		if i < len(value) && (value[i] == '+' || value[i] == '-') {
			i++
		}
		exponentStart := i
		for i < len(value) && value[i] >= '0' && value[i] <= '9' {
			i++
		}
		if i == exponentStart {
			return false
		}
	}
	return i == len(value)
}

// Format the ExprNode into a Writer.
func (n *LiteralExpr) Format(w io.Writer) {
	switch n.Kind {
	case pkast.LiteralString, pkast.LiteralBytes:
		_, _ = io.WriteString(w, strconv.Quote(n.Value))
	case pkast.LiteralNull, pkast.LiteralBoolean:
		// The legacy format prints these keywords upper case, while the
		// shared restore helper lower cases every keyword.
		ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, w)
		_ = n.Restore(ctx)
	default:
		formatViaRestore(n, w)
	}
}

func (n *LiteralExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, _ := v.Enter(n)
	outer, ok := entered.(*LiteralExpr)
	if !ok || outer == nil {
		return entered, false
	}
	return v.Leave(outer)
}

var _ ExprNode = (*LiteralExpr)(nil)

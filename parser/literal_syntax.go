package parser

import (
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/driver"
	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/myparser/types"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

// stringLiteralParts accumulates adjacent string fragments without copying
// the growing value on every grammar reduction.
type stringLiteralParts struct {
	parts []string
	span  loc.Span
}

func newStringLiteralParts(lit string, span loc.Span) *stringLiteralParts {
	return &stringLiteralParts{parts: []string{lit}, span: span}
}

func appendStringLiteralParts(parts *stringLiteralParts, lit string, span loc.Span) *stringLiteralParts {
	parts.parts = append(parts.parts, lit)
	parts.span = span
	return parts
}

func finalizeStringLiteral(l yyLexer, parts *stringLiteralParts) *ast.LiteralExpr {
	return stringLiteral(l, strings.Join(parts.parts, ""), parts.span)
}

// Raw borrows immutable input, never the scanner's reusable decoding buffer.
func syntaxLiteral(l yyLexer, kind pkast.LiteralKind, value string, span loc.Span) *ast.LiteralExpr {
	literal := &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: kind, Value: value}}
	literal.SetSpan(span)
	if src := yyGetParser(l).src; span.Within(len(src)) {
		literal.Raw = src[span.Start:span.End]
	}
	return literal
}

func stringLiteral(l yyLexer, value string, span loc.Span) *ast.LiteralExpr {
	literal := syntaxLiteral(l, pkast.LiteralString, value, span)
	if len(literal.Raw) != 0 {
		switch literal.Raw[0] {
		case '\'':
			literal.Quote = lex.QuoteSingle
		case '"':
			literal.Quote = lex.QuoteDouble
		case '`':
			literal.Quote = lex.QuoteBacktick
		}
	}
	return literal
}

func binaryLiteral(l yyLexer, kind pkast.LiteralKind, span loc.Span) *ast.LiteralExpr {
	literal := syntaxLiteral(l, kind, "", span)
	// The lexer has already validated the token, including quoted hex parity.
	if raw := literal.Raw; len(raw) >= 2 {
		if raw[0] == '0' {
			literal.Value = raw[2:]
		} else if len(raw) >= 3 {
			literal.Value = raw[2 : len(raw)-1]
			literal.Quote = lex.QuoteSingle
		}
	}
	return literal
}

func introduceLiteral(l yyLexer, literal *ast.LiteralExpr, name string, introducer, span loc.Span) *ast.LiteralExpr {
	literal.Introducer = syntaxIdent(l, name, introducer)
	if literal.Kind == pkast.LiteralString && name == "binary" {
		literal.Kind = pkast.LiteralBytes
	}
	literal.SetSpan(span)
	if src := yyGetParser(l).src; span.Within(len(src)) {
		literal.Raw = src[span.Start:span.End]
	}
	return literal
}

// literalStringValue reads the decoded string of a syntax literal.
func literalStringValue(expr ast.ExprNode) (string, bool) {
	literal, ok := expr.(*ast.LiteralExpr)
	if !ok || literal == nil {
		return "", false
	}
	if literal.Kind == pkast.LiteralString || literal.Kind == pkast.LiteralBytes {
		return literal.Value, true
	}
	return "", false
}

// Token values are discarded after their dialect-specific spelling is captured.
func numericLiteral(l yyLexer, value interface{}, span loc.Span) *ast.LiteralExpr {
	var kind pkast.LiteralKind
	var text string
	switch value := value.(type) {
	case int64:
		kind, text = pkast.LiteralInteger, strconv.FormatInt(value, 10)
	case uint64:
		kind, text = pkast.LiteralInteger, strconv.FormatUint(value, 10)
	case float64:
		kind, text = pkast.LiteralFloat, strconv.FormatFloat(value, 'e', -1, 64)
	case *types.MyDecimal:
		kind, text = pkast.LiteralDecimal, value.String()
		// Omitting the leading zero preserves the legacy 73–81 digit fraction round trip.
		if value.GetDigitsInt() == 0 && value.GetDigitsFrac() > 72 {
			if value.IsNegative() {
				text = "-" + strings.TrimPrefix(text, "-0")
			} else {
				text = strings.TrimPrefix(text, "0")
			}
		}
	default:
		yyAppendErr(l, yyErrorf(l, "unsupported numeric literal type %T", value))
	}
	return syntaxLiteral(l, kind, text, span)
}

func toInt(l *Scanner, lval *yySymType, str string) int {
	n, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		e := err.(*strconv.NumError)
		if e.Err == strconv.ErrRange {
			// Fall through to decimal: MySQL truncates out-of-range integer
			// literals to the max decimal value and emits a warning.
			return toDecimal(l, lval, str)
		}
		l.AppendError(l.Errorf("integer literal: %v", err))
		return int(unicode.ReplacementChar)
	}

	switch {
	case n <= math.MaxInt64:
		lval.item = int64(n)
	default:
		lval.item = n
	}
	return intLit
}

func toDecimal(l *Scanner, lval *yySymType, str string) int {
	dec, err := driver.NewDecimal(str)
	if err != nil {
		if sqlErr, ok := err.(*mysql.SQLError); ok && sqlErr.Code == mysql.ErrTruncatedWrongValue {
			l.warns = append(l.warns, err)
		} else {
			l.AppendError(l.Errorf("decimal literal: %v", err))
			return int(unicode.ReplacementChar)
		}
	}
	lval.item = dec
	return decLit
}

func toFloat(l *Scanner, lval *yySymType, str string) int {
	n, err := strconv.ParseFloat(str, 64)
	if err != nil {
		l.AppendError(mysql.NewErr(mysql.ErrIllegalValueForType, "double", str))
		return int(unicode.ReplacementChar)
	}

	lval.item = n
	return floatLit
}

// See https://dev.mysql.com/doc/refman/5.7/en/hexadecimal-literals.html
func toHex(l *Scanner, lval *yySymType, str string) int {
	h, err := driver.NewHexLiteral(str)
	if err != nil {
		l.AppendError(l.Errorf("hex literal: %v", err))
		return int(unicode.ReplacementChar)
	}
	lval.item = h
	return hexLit
}

// See https://dev.mysql.com/doc/refman/5.7/en/bit-type.html
func toBit(l *Scanner, lval *yySymType, str string) int {
	b, err := driver.NewBitLiteral(str)
	if err != nil {
		l.AppendError(l.Errorf("bit literal: %v", err))
		return int(unicode.ReplacementChar)
	}
	lval.item = b
	return bitLit
}

func getUint64FromNUM(num interface{}) uint64 {
	switch v := num.(type) {
	case int64:
		return uint64(v)
	case uint64:
		return v
	}
	return 0
}

func hexInteger(text string, allowQuoted bool) (uint64, error) {
	var digits string
	switch {
	case len(text) >= 3 && text[0] == '0' && text[1] == 'x':
		digits = text[2:]
	case allowQuoted && len(text) >= 3 && (text[0] == 'x' || text[0] == 'X') && text[1] == '\'' && text[len(text)-1] == '\'':
		digits = text[2 : len(text)-1]
	default:
		return 0, errors.New("expected a numeric 0x hexadecimal literal")
	}
	if digits == "" {
		return 0, nil
	}
	value, err := strconv.ParseUint(digits, 16, 64)
	if err != nil {
		return 0, errors.Annotate(err, "hexadecimal integer out of range")
	}
	return value, nil
}

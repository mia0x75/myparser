package myparser

import (
	"strconv"
	"strings"

	"github.com/mia0x75/myparser/ast"
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

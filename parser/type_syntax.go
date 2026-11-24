package parser

import (
	"strconv"
	"strings"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/mysql"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

// Grammar token categories do not survive as resolved type codes in the AST.
func newSyntaxType(kind byte) *ast.TypeName {
	var keyword string
	switch kind {
	case mysql.TypeTiny:
		keyword = "TINYINT"
	case mysql.TypeShort:
		keyword = "SMALLINT"
	case mysql.TypeInt24:
		keyword = "MEDIUMINT"
	case mysql.TypeLong:
		keyword = "INT"
	case mysql.TypeLonglong:
		keyword = "BIGINT"
	case mysql.TypeBit:
		keyword = "BIT"
	case mysql.TypeNewDecimal:
		keyword = "DECIMAL"
	case mysql.TypeFloat:
		keyword = "FLOAT"
	case mysql.TypeDouble:
		keyword = "DOUBLE"
	case mysql.TypeString, mysql.TypeVarString:
		keyword = "CHAR"
	case mysql.TypeVarchar:
		keyword = "VARCHAR"
	case mysql.TypeTinyBlob:
		keyword = "TINYTEXT"
	case mysql.TypeBlob:
		keyword = "TEXT"
	case mysql.TypeMediumBlob:
		keyword = "MEDIUMTEXT"
	case mysql.TypeLongBlob:
		keyword = "LONGTEXT"
	case mysql.TypeEnum:
		keyword = "ENUM"
	case mysql.TypeSet:
		keyword = "SET"
	case mysql.TypeJSON:
		keyword = "JSON"
	case mysql.TypeGeometry:
		keyword = "GEOMETRY"
	case mysql.TypeDate:
		keyword = "DATE"
	case mysql.TypeDatetime:
		keyword = "DATETIME"
	case mysql.TypeTimestamp:
		keyword = "TIMESTAMP"
	case mysql.TypeDuration:
		keyword = "TIME"
	case mysql.TypeYear:
		keyword = "YEAR"
	}
	return ast.NewTypeName(keyword)
}

func typeLiteral(l yyLexer, kind pkast.LiteralKind, value string, span loc.Span) *pkast.BaseLiteralExpr {
	literal := &pkast.BaseLiteralExpr{Kind: kind, Value: value}
	src := yyGetParser(l).src
	if span.Within(len(src)) && !span.Empty() {
		literal.SetSpan(span)
		literal.Raw = src[span.Start:span.End]
		if literal.Raw == value {
			literal.Value = literal.Raw
		}
	}
	return literal
}

func typeModifiers(values ...*pkast.BaseLiteralExpr) []ast.ExprNode {
	var modifiers []ast.ExprNode
	for _, value := range values {
		if value != nil {
			literal := &ast.LiteralExpr{BaseLiteralExpr: *value}
			if value.Kind == pkast.LiteralString && value.Raw != "" {
				switch value.Raw[0] {
				case '\'':
					literal.Quote = lex.QuoteSingle
				case '"':
					literal.Quote = lex.QuoteDouble
				}
			}
			modifiers = append(modifiers, literal)
		}
	}
	return modifiers
}

func typeLength(value *pkast.BaseLiteralExpr) int {
	if value == nil {
		return -1
	}
	length, err := strconv.Atoi(value.Value)
	if err != nil {
		return -1
	}
	return length
}

func syntaxTypeNode(l yyLexer, name *ast.TypeName, span loc.Span) *ast.TypeName {
	src := yyGetParser(l).src
	if span.Within(len(src)) && !span.Empty() {
		name.SetSpan(span)
		name.SetText(src[span.Start:span.End])
	}
	return name
}

func syntaxTypeIdent(l yyLexer, name string, span loc.Span) lex.Ident {
	ident := syntaxIdent(l, name, span)
	if ident.Raw != "" && ident.Raw[0] == '\'' {
		ident.Quote = lex.QuoteSingle
	}
	return ident
}

func charsetName(name lex.Ident) string { return strings.ToLower(name.Name) }

func typeStringValues(values []*pkast.BaseLiteralExpr) []string {
	strings := make([]string, len(values))
	for i, value := range values {
		strings[i] = value.Value
	}
	return strings
}

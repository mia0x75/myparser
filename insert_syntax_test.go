package myparser

import (
	"reflect"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

func TestSyntaxInsertColumns(t *testing.T) {
	if syntaxInsertColumns(nil) != nil {
		t.Fatal("an absent column list must remain nil")
	}
	if columns := syntaxInsertColumns([]ast.QualifiedName{}); columns == nil || len(columns) != 0 {
		t.Fatal("an explicit empty column list must remain present")
	}
	name := ast.QualifiedName{
		Parts: []lex.Ident{
			{Name: "a.b", Raw: "`a.b`", Quote: lex.QuoteBacktick, Span: loc.Span{Start: 2, End: 7}},
			{Name: "列", Raw: "`列`", Quote: lex.QuoteBacktick, Span: loc.Span{Start: 8, End: 13}},
		},
		Span: loc.Span{Start: 2, End: 13},
	}
	columns := syntaxInsertColumns([]ast.QualifiedName{name})
	column, ok := columns[0].(*ast.ColumnRef)
	if !ok || !reflect.DeepEqual(column.Name, name) || column.Span() != name.Span {
		t.Fatalf("qualified target metadata lost: %#v", columns[0])
	}
	if &column.Name.Parts[0] != &name.Parts[0] {
		t.Fatal("column conversion must not copy or decode identifier parts")
	}
}

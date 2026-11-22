package myparser

import (
	"slices"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	dformat "github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
)

func TestRenameTableVisitOnce(t *testing.T) {
	sql := "RENAME TABLE a TO b, c TO d"
	stmt, err := New().ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	_, ok := stmt.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		if table, ok := n.(*ast.TableName); ok {
			tables = append(tables, table.Name.Parts[0].Name)
			copy := *table
			copy.Name = ast.QualifiedName{Parts: []lex.Ident{{Name: table.Name.Parts[0].Name + "_new"}}}
			return &copy, false
		}
		return n, false
	}})
	if !ok || !slices.Equal(tables, []string{"a", "b", "c", "d"}) {
		t.Fatalf("tables visited = %q, ok=%v", tables, ok)
	}
	var out strings.Builder
	if err := stmt.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "RENAME TABLE `a_new` TO `b_new`, `c_new` TO `d_new`"; got != want {
		t.Fatalf("restore = %q, want %q", got, want)
	}
	if text, err := stmt.Span().Slice(sql); err != nil || text != sql {
		t.Fatalf("statement span = %q, error=%v", text, err)
	}
}

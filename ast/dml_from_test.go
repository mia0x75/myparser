package ast_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
)

func TestTableRefsClauseSingleSource(t *testing.T) {
	t.Parallel()
	source := &ast.TableName{BaseTableRef: pkast.BaseTableRef{
		Name: pkast.QualifiedName{Parts: []pkast.Ident{{Name: "mock_table"}}},
	}}
	clause := &ast.TableRefsClause{BaseFromClause: pkast.BaseFromClause{Table: source}}
	var out strings.Builder
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &out)
	ctx.JoinLevel = 3
	if err := clause.Restore(ctx); err != nil || out.String() != "`mock_table`" || ctx.JoinLevel != 3 {
		t.Fatalf("single source restore = %q, err=%v, level=%d", out.String(), err, ctx.JoinLevel)
	}
	var entered []ast.Node
	updated, ok := clause.Accept(operatorVisitor{
		enter: func(n ast.Node) (ast.Node, bool) { entered = append(entered, n); return n, false },
		leave: func(n ast.Node) (ast.Node, bool) { return n, true },
	})
	if !ok || updated != clause || !reflect.DeepEqual(entered, []ast.Node{clause, source}) {
		t.Fatalf("single source traversal lost identity or order: %v", entered)
	}
	for _, tc := range []struct {
		name        string
		replacement ast.Node
		wantOK      bool
	}{
		{name: "source", replacement: &ast.TableSource{BaseTableSource: pkast.BaseTableSource{Source: source}}, wantOK: true},
		{name: "expression", replacement: ast.NewParamMarkerExpr(0)},
		{name: "typed nil", replacement: (*ast.TableSource)(nil)},
		{name: "nil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clause.Table = source
			_, ok := clause.Accept(operatorVisitor{
				enter: func(n ast.Node) (ast.Node, bool) { return n, false },
				leave: func(n ast.Node) (ast.Node, bool) {
					if n == source {
						return tc.replacement, true
					}
					return n, true
				},
			})
			if ok != tc.wantOK || !ok && clause.Table != source || ok && clause.Table != tc.replacement {
				t.Fatalf("replacement %T: ok=%v, table=%T", tc.replacement, ok, clause.Table)
			}
		})
	}
	for _, source := range []ast.ResultSetNode{nil, (*ast.TableSource)(nil), (*ast.JoinExpr)(nil)} {
		clause.Table = source
		out.Reset()
		if err := clause.Restore(ctx); err == nil || out.Len() != 0 {
			t.Fatalf("missing source %T was rendered: %q, err=%v", source, out.String(), err)
		}
	}
}

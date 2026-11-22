package ast_test

import (
	"strings"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
)

func TestInsertRestoreRejectsNilSlots(t *testing.T) {
	for _, tc := range []struct {
		name  string
		clear func(*ast.InsertStmt)
	}{
		{"typed nil table", func(n *ast.InsertStmt) { n.Table = (*insertTestTable)(nil) }},
		{"typed nil query", func(n *ast.InsertStmt) { n.Query = (*insertTestQuery)(nil) }},
		{"typed nil column", func(n *ast.InsertStmt) { n.Columns[0] = (*ast.LiteralExpr)(nil) }},
		{"typed nil value", func(n *ast.InsertStmt) { n.Values[0][0] = (*ast.LiteralExpr)(nil) }},
		{"nil assignment", func(n *ast.InsertStmt) { n.Assignments[0] = nil }},
		{"nil conflict assignment", func(n *ast.InsertStmt) { n.OnDuplicate[0] = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, _ := newInsertTestTree()
			tc.clear(n)
			var out strings.Builder
			if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil {
				t.Fatal("Restore accepted nil slot")
			}
		})
	}
}

func TestQueryAndExecuteRestoreRejectsNilSlots(t *testing.T) {
	for _, tc := range []struct {
		name string
		node ast.Node
	}{
		{"nil VALUES", (*ast.ValuesStmt)(nil)},
		{"nil TABLE", (*ast.TableStmt)(nil)},
		{"typed nil TABLE reference", &ast.TableStmt{BaseTableStmt: pkast.BaseTableStmt{Table: (*insertTestTable)(nil)}}},
		{"typed nil VALUES row", &ast.ValuesStmt{BaseValuesStmt: pkast.BaseValuesStmt{Rows: []pkast.RowNode{(*ast.RowExpr)(nil)}}}},
		{"typed nil LIMIT count", &ast.Limit{BaseLimit: pkast.BaseLimit{Count: (*ast.LiteralExpr)(nil)}}},
		{"typed nil LIMIT offset", &ast.Limit{BaseLimit: pkast.BaseLimit{Count: ast.NewParamMarkerExpr(0), Offset: (*ast.LiteralExpr)(nil)}}},
		{"nil EXECUTE", (*ast.ExecuteStmt)(nil)},
		{"typed nil EXECUTE SQL", &ast.ExecuteStmt{SQLExpr: (*ast.LiteralExpr)(nil)}},
		{"nil EXECUTE variable", &ast.ExecuteStmt{UsingVars: []ast.ExprNode{nil}}},
		{"typed nil EXECUTE variable", &ast.ExecuteStmt{UsingVars: []ast.ExprNode{(*ast.LiteralExpr)(nil)}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := tc.node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil {
				t.Fatal("Restore accepted nil slot")
			}
		})
	}
}

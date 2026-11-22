package ast_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
)

func TestStatsAndMaintenanceRestoreAndVisitorBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name string
		new  func(*ast.TableName) (ast.Node, func() *ast.TableName)
	}{
		{"ANALYZE", func(table *ast.TableName) (ast.Node, func() *ast.TableName) {
			n := &ast.AnalyzeTableStmt{TableNames: []*ast.TableName{table}}
			return n, func() *ast.TableName { return n.TableNames[0] }
		}},
		{"DROP STATS", func(table *ast.TableName) (ast.Node, func() *ast.TableName) {
			n := &ast.DropStatsStmt{Table: table}
			return n, func() *ast.TableName { return n.Table }
		}},
		{"CHECK TABLE", func(table *ast.TableName) (ast.Node, func() *ast.TableName) {
			n := &ast.TableMaintenanceStmt{Tp: ast.TableMaintenanceCheck, Tables: []*ast.TableName{table}}
			return n, func() *ast.TableName { return n.Tables[0] }
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, _ := tc.new(nil)
			var out strings.Builder
			if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil {
				t.Fatal("Restore accepted nil table")
			}
			for _, replacement := range []ast.Node{nil, (*ast.TableName)(nil), &ast.ColumnPosition{}} {
				old := &ast.TableName{Name: ast.QualifiedName{Parts: []lex.Ident{{Name: "t"}}}}
				n, slot := tc.new(old)
				_, ok := n.Accept(operatorVisitor{
					enter: func(node ast.Node) (ast.Node, bool) { return node, false },
					leave: func(node ast.Node) (ast.Node, bool) {
						if node == old {
							return replacement, true
						}
						return node, true
					},
				})
				if ok || slot() != old {
					t.Fatal("Accept stored a rejected table replacement")
				}
				marker := errors.New("MOCK_WRITER")
				if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, callBoundaryWriter{marker})); !errors.Is(err, marker) {
					t.Fatalf("lost writer error: %v", err)
				}
			}
		})
	}
}

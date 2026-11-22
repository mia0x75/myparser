package myparser

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/parserkit/loc"
)

func TestLimitSharedSyntax(t *testing.T) {
	for _, tc := range []struct {
		name, sql, clause, count, offset string
	}{
		{"select count", "SELECT 1 LIMIT 2", "LIMIT 2", "2", ""},
		{"select comma", "SELECT 1 LIMIT 2,3", "LIMIT 2,3", "3", "2"},
		{"select offset", "SELECT 1 LIMIT 3 OFFSET 2", "LIMIT 3 OFFSET 2", "3", "2"},
		{"set operation", "SELECT 1 UNION SELECT 2 LIMIT 2,3", "LIMIT 2,3", "3", "2"},
		{"update", "UPDATE t SET a=1 LIMIT 2", "LIMIT 2", "2", ""},
		{"delete", "DELETE FROM t WHERE a=1 LIMIT 2", "LIMIT 2", "2", ""},
		{"show errors", "SHOW ERRORS LIMIT 2,3", "LIMIT 2,3", "3", "2"},
		{"show profile", "SHOW PROFILE CPU FOR QUERY 1 LIMIT 2,3", "LIMIT 2,3", "3", "2"},
		{"handler", "HANDLER t READ FIRST LIMIT 3 OFFSET 2", "LIMIT 3 OFFSET 2", "3", "2"},
		{"aggregate", "SELECT GROUP_CONCAT(a LIMIT 3 OFFSET 2) FROM t", "LIMIT 3 OFFSET 2", "3", "2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := New()
			stmt, err := p.ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			var limits []*ast.Limit
			if _, ok := stmt.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
				if limit, ok := n.(*ast.Limit); ok {
					limits = append(limits, limit)
				}
				return n, false
			}}); !ok || len(limits) != 1 {
				t.Fatalf("LIMIT traversal: ok=%v, found=%d", ok, len(limits))
			}
			limit := limits[0]
			start := strings.Index(tc.sql, tc.clause)
			wantSpan := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(tc.clause))}
			if limit.Span() != wantSpan {
				t.Fatalf("LIMIT span = %v, want %v", limit.Span(), wantSpan)
			}
			for _, slot := range []struct {
				name string
				expr ast.ExprNode
				want string
			}{{"count", limit.Count, tc.count}, {"offset", limit.Offset, tc.offset}} {
				if slot.want == "" {
					if slot.expr != nil {
						t.Fatalf("unexpected %s: %T", slot.name, slot.expr)
					}
					continue
				}
				literal, ok := slot.expr.(*ast.LiteralExpr)
				if !ok || literal.Value != slot.want {
					t.Fatalf("%s = %+v, want %s", slot.name, slot.expr, slot.want)
				}
			}
			restored := restoreStmt(t, stmt)
			reparsed, err := p.ParseOneStmt(restored, "", "")
			if err != nil {
				t.Fatal(err)
			}
			if got := restoreStmt(t, reparsed); got != restored {
				t.Fatalf("LIMIT restore drifted: %q -> %q", restored, got)
			}
			if _, err := p.ParseOneStmt("SELECT ? LIMIT ?", "", ""); err != nil {
				t.Fatal(err)
			}
			if limit.Span() != wantSpan || restoreStmt(t, stmt) != restored {
				t.Fatal("parser reuse changed the previous LIMIT")
			}
		})
	}
}

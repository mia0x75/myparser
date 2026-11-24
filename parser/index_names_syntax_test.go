package parser

import (
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/parserkit/lex"
)

func TestIndexNamesSyntax(t *testing.T) {
	for _, tc := range []struct {
		name, sql, hint string
		want            []string
	}{
		{"hint", "SELECT * FROM t USE INDEX (`索``名`, plain)", "USE INDEX (`索``名`, plain)", []string{"索`名", "plain"}},
		{"hint primary", "SELECT * FROM t FORCE INDEX FOR JOIN (primary)", "FORCE INDEX FOR JOIN (primary)", []string{"primary"}},
		{"hint empty", "SELECT * FROM t USE INDEX ()", "USE INDEX ()", nil},
		{"analyze", "ANALYZE TABLE t INDEX `索``名`, plain", "", []string{"索`名", "plain"}},
		{"analyze partition", "ANALYZE TABLE t PARTITION p INDEX `索``名`, plain WITH 2 BUCKETS", "", []string{"索`名", "plain"}},
		{"alter analyze", "ALTER TABLE t ANALYZE PARTITION p INDEX `索``名`, plain WITH 2 BUCKETS", "", []string{"索`名", "plain"}},
		{"persistent", "ANALYZE TABLE t PERSISTENT FOR COLUMNS () INDEXES (`索``名`, plain)", "", []string{"索`名", "plain"}},
		{"persistent all", "ANALYZE TABLE t PERSISTENT FOR COLUMNS ALL INDEXES ALL", "", nil},
		{"persistent empty", "ANALYZE TABLE t PERSISTENT FOR COLUMNS () INDEXES ()", "", []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			stmt, err := p.ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			get := func(stmt ast.StmtNode) []lex.Ident {
				switch n := stmt.(type) {
				case *ast.SelectStmt:
					hint := n.From.FromClauseCore().Table.(*ast.JoinExpr).Left.(*ast.TableSource).IndexHints[0]
					return hint.IndexNames
				case *ast.AnalyzeTableStmt:
					return n.IndexNames
				default:
					t.Fatalf("unexpected statement %T", stmt)
					return nil
				}
			}
			ids := get(stmt)
			if (ids == nil) != (tc.want == nil) || len(ids) != len(tc.want) {
				t.Fatalf("index list = %+v, want %+v (nil=%v)", ids, tc.want, tc.want == nil)
			}
			for i, id := range ids {
				if id.Name != tc.want[i] || !id.Present() {
					t.Fatalf("index %d = %+v, want %q", i, id, tc.want[i])
				}
				if got, err := id.Span.Slice(tc.sql); err != nil || got != id.Raw {
					t.Fatalf("identifier selects %q (%v), raw=%q", got, err, id.Raw)
				}
				if strings.HasPrefix(id.Raw, "`") {
					if id.Quote != lex.QuoteBacktick {
						t.Fatalf("quoted identifier lost quote: %+v", id)
					}
				} else if id.Name != id.Raw || id.Quote != lex.QuoteUnquoted || unsafe.StringData(id.Raw) != unsafe.StringData(tc.sql[id.Span.Start:]) {
					t.Fatalf("unquoted identifier lost input slicing: %+v", id)
				}
			}
			if tc.hint != "" {
				hint := stmt.(*ast.SelectStmt).From.FromClauseCore().Table.(*ast.JoinExpr).Left.(*ast.TableSource).IndexHints[0]
				if selected, err := hint.Span.Slice(tc.sql); err != nil || selected != tc.hint {
					t.Fatalf("hint span selects %q (%v), want %q", selected, err, tc.hint)
				}
			}
			snapshot := append([]lex.Ident(nil), ids...)
			canonical := restoreStmt(t, stmt)
			reparsed, err := p.ParseOneStmt(canonical, "", "")
			if err != nil {
				t.Fatalf("reparse %q: %v", canonical, err)
			}
			next := get(reparsed)
			if (next == nil) != (ids == nil) || len(next) != len(ids) {
				t.Fatal("round trip changed index list presence")
			}
			for i := range ids {
				if next[i].Name != ids[i].Name {
					t.Fatalf("round trip changed index %d: %q -> %q", i, ids[i].Name, next[i].Name)
				}
			}
			if !reflect.DeepEqual(ids, snapshot) && len(ids) != 0 || restoreStmt(t, stmt) != canonical {
				t.Fatal("parser reuse changed retained index syntax")
			}
		})
	}
}

package parser

import (
	"github.com/mia0x75/myparser/ast"
	dformat "github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
	"slices"
	"strings"
	"testing"
)

func TestParserReuseAfterFailureKeepsParameterState(t *testing.T) {
	for _, failedSQL := range []string{"SELECT ? +", "SELECT /*! /*! ? */ */"} {
		t.Run(failedSQL, func(t *testing.T) {
			p := New()
			old, err := p.ParseOneStmt("SELECT ? + ?")
			if err != nil {
				t.Fatal(err)
			}
			var retained []*ast.ParamMarkerExpr
			if _, ok := old.Accept(parameterCollector{params: &retained}); !ok || len(retained) != 2 {
				t.Fatal("initial parameter traversal failed")
			}
			before := restoreStmt(t, old)
			for range 3 {
				if stmts, _, err := p.ParseWithWarnings(failedSQL); err == nil || stmts != nil {
					t.Fatal("invalid SQL produced statements or no error")
				}
				const sql = "SELECT '中文', ?; SELECT ? LIMIT ?"
				stmts, warns, err := p.ParseWithWarnings(sql)
				if err != nil || len(warns) != 0 || len(stmts) != 2 {
					t.Fatalf("Parse after failure: statements=%d warnings=%d err=%v", len(stmts), len(warns), err)
				}
				fresh, _, err := New().ParseWithWarnings(sql)
				if err != nil {
					t.Fatal(err)
				}
				for i, stmt := range stmts {
					var got, want []*ast.ParamMarkerExpr
					_, gotOK := stmt.Accept(parameterCollector{params: &got})
					_, wantOK := fresh[i].Accept(parameterCollector{params: &want})
					if !gotOK || !wantOK || len(got) != i+1 || len(got) != len(want) {
						t.Fatal("parameter traversal changed after failed parse")
					}
					for j, param := range got {
						raw, err := param.Span().Slice(sql)
						if err != nil || raw != "?" || param.Index != j || param.Span() != want[j].Span() {
							t.Fatalf("parameter %d state leaked: index=%d span=%v err=%v", j, param.Index, param.Span(), err)
						}
					}
				}
			}
			if restoreStmt(t, old) != before || retained[0].Index != 0 || retained[1].Index != 1 ||
				retained[0].Span() != (loc.Span{Start: 7, End: 8}) || retained[1].Span() != (loc.Span{Start: 11, End: 12}) {
				t.Fatal("later failed/successful parses changed the retained AST")
			}
		})
	}
}

func TestRenameTableVisitOnce(t *testing.T) {
	sql := "RENAME TABLE a TO b, c TO d"
	stmt, err := New().ParseOneStmt(sql)
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

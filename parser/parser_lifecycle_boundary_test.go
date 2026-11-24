package parser

import (
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/parserkit/loc"
)

func TestParserReuseAfterFailureKeepsParameterState(t *testing.T) {
	for _, failedSQL := range []string{"SELECT ? +", "SELECT /*! /*! ? */ */"} {
		t.Run(failedSQL, func(t *testing.T) {
			p := New()
			old, err := p.ParseOneStmt("SELECT ? + ?", "", "")
			if err != nil {
				t.Fatal(err)
			}
			var retained []*ast.ParamMarkerExpr
			if _, ok := old.Accept(parameterCollector{params: &retained}); !ok || len(retained) != 2 {
				t.Fatal("initial parameter traversal failed")
			}
			before := restoreStmt(t, old)
			for range 3 {
				if stmts, _, err := p.Parse(failedSQL, "", ""); err == nil || stmts != nil {
					t.Fatal("invalid SQL produced statements or no error")
				}
				const sql = "SELECT '中文', ?; SELECT ? LIMIT ?"
				stmts, warns, err := p.Parse(sql, "", "")
				if err != nil || len(warns) != 0 || len(stmts) != 2 {
					t.Fatalf("Parse after failure: statements=%d warnings=%d err=%v", len(stmts), len(warns), err)
				}
				fresh, _, err := New().Parse(sql, "", "")
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

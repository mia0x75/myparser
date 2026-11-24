package parser

import (
	"sort"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/parserkit/loc"
)

func TestParameterMarkersKeepIndexSeparateFromSpan(t *testing.T) {
	parser := New()
	parser.EnableWindowFunc(true)
	sql := "SELECT ?, SUM(a) OVER (ORDER BY a ROWS BETWEEN ? PRECEDING AND ? FOLLOWING) LIMIT ?"
	stmt, err := parser.ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}

	var params []*ast.ParamMarkerExpr
	stmt.Accept(parameterCollector{params: &params})
	if len(params) != 4 {
		t.Fatalf("got %d parameters, want 4", len(params))
	}
	firstParseParams := append([]*ast.ParamMarkerExpr(nil), params...)
	searchFrom := 0
	for i, param := range params {
		relative := strings.Index(sql[searchFrom:], "?")
		if relative < 0 {
			t.Fatalf("parameter %d has no source placeholder", i)
		}
		start := searchFrom + relative
		wantSpan := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + 1)}
		searchFrom = start + 1
		if param.Index != i {
			t.Errorf("parameter %d index = %d, want %d", i, param.Index, i)
		}
		if param.Span() != wantSpan {
			t.Errorf("parameter %d span = %v, want %v", i, param.Span(), wantSpan)
		}
	}

	stmt, err = parser.ParseOneStmt("SELECT ?", "", "")
	if err != nil {
		t.Fatal(err)
	}
	params = nil
	stmt.Accept(parameterCollector{params: &params})
	if len(params) != 1 || params[0].Index != 0 {
		t.Fatalf("parser reused parameter index: %#v", params)
	}
	if firstParseParams[0].Index != 0 || firstParseParams[0].Span() != (loc.Span{Start: 7, End: 8}) {
		t.Fatalf("later Parse changed an existing parameter: index=%d span=%v", firstParseParams[0].Index, firstParseParams[0].Span())
	}
}

func TestParameterMarkersAcrossStatementsAndSourceForms(t *testing.T) {
	for _, tc := range []struct {
		name string
		sql  string
	}{
		{"statement list", "SELECT ?; SELECT ? LIMIT ?"},
		{"limit offset", "SELECT ? LIMIT ? OFFSET ?"},
		{"update limit", "UPDATE t SET a=? LIMIT ?"},
		{"delete limit", "DELETE FROM t WHERE a=? LIMIT ?"},
		{"unicode and executable comment", "SELECT '中文', /*! ? + */ ?"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parser := New()
			stmts, _, err := parser.Parse(tc.sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			var params []*ast.ParamMarkerExpr
			for _, stmt := range stmts {
				var statementParams []*ast.ParamMarkerExpr
				stmt.Accept(parameterCollector{params: &statementParams})
				sort.Slice(statementParams, func(i, j int) bool {
					return statementParams[i].Span().Start < statementParams[j].Span().Start
				})
				for i, param := range statementParams {
					if param.Index != i {
						t.Errorf("statement parameter %d has index %d, want %d", i, param.Index, i)
					}
				}
				params = append(params, statementParams...)
			}
			sort.Slice(params, func(i, j int) bool { return params[i].Span().Start < params[j].Span().Start })
			if len(params) != strings.Count(tc.sql, "?") {
				t.Fatalf("got %d parameters", len(params))
			}
			for i, param := range params {
				raw, err := param.Span().Slice(tc.sql)
				if err != nil || raw != "?" {
					t.Fatalf("parameter %d: index=%d span=%v raw=%q err=%v", i, param.Index, param.Span(), raw, err)
				}
			}
		})
	}
}

type parameterCollector struct {
	params *[]*ast.ParamMarkerExpr
}

func (v parameterCollector) Enter(node ast.Node) (ast.Node, bool) {
	if param, ok := node.(*ast.ParamMarkerExpr); ok {
		*v.params = append(*v.params, param)
	}
	return node, false
}

func (parameterCollector) Leave(node ast.Node) (ast.Node, bool) {
	return node, true
}

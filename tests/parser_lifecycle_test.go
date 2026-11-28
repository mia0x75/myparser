package parser_test

import (
	"strings"
	"testing"

	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/parser"
)

func TestParseEmptyStatementList(t *testing.T) {
	for _, tc := range []struct {
		name string
		sql  string
	}{
		{name: "empty"},
		{name: "comments and separators", sql: ";; /* empty */ ;"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmts, err := parser.New().Parse(tc.sql)
			if err != nil || stmts != nil {
				t.Fatalf("Parse(%q) = %v, %v; want nil, nil", tc.sql, stmts, err)
			}
		})
	}
}

func TestParseBatchStatementLifecycle(t *testing.T) {
	const sql = "; /* lead */ SELECT ?, ?;; -- gap\n SELECT ? /* trailing */ ;"
	stmts, err := parser.New().Parse(sql)
	if err != nil || len(stmts) != 2 {
		t.Fatalf("Parse returned %d statements, error %v", len(stmts), err)
	}
	for i, tc := range []struct {
		query string
		text  string
		count int
	}{
		{query: "SELECT ?, ?", text: "; /* lead */ SELECT ?, ?;", count: 2},
		{query: "SELECT ?", text: "; -- gap\n SELECT ? /* trailing */ ;", count: 1},
	} {
		stmt, ok := stmts[i].(*ast.SelectStmt)
		if !ok {
			t.Fatalf("statement %d = %T, want *ast.SelectStmt", i, stmts[i])
		}
		start := strings.LastIndex(sql, tc.query)
		want := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(tc.query))}
		if stmt.Span() != want || stmt.Text() != tc.text {
			t.Errorf("statement %d span/text = %v/%q, want %v/%q", i, stmt.Span(), stmt.Text(), want, tc.text)
		}
		if len(stmt.Fields.Items) != tc.count {
			t.Fatalf("statement %d has %d fields, want %d", i, len(stmt.Fields.Items), tc.count)
		}
		for j, field := range stmt.Fields.Items {
			param, ok := field.Expr.(*ast.ParamMarkerExpr)
			if !ok {
				t.Fatalf("statement %d field %d = %T, want parameter", i, j, field.Expr)
			}
			if param.Index != j || !param.Span().Within(len(sql)) || sql[param.Span().Start:param.Span().End] != "?" {
				t.Errorf("statement %d parameter %d index/span = %d/%v", i, j, param.Index, param.Span())
			}
		}
	}
}

func TestParseRecoversAfterBatchError(t *testing.T) {
	p := parser.New()
	stmts, err := p.Parse("SELECT ?; SELECT FROM")
	if err == nil || stmts != nil {
		t.Fatalf("invalid batch returned %v, %v; want nil statements and error", stmts, err)
	}
	stmts, err = p.Parse("SELECT ?")
	if err != nil || len(stmts) != 1 {
		t.Fatalf("Parse after error returned %d statements, error %v", len(stmts), err)
	}
	stmt, ok := stmts[0].(*ast.SelectStmt)
	if !ok || stmt.Text() != "SELECT ?" || len(stmt.Fields.Items) != 1 {
		t.Fatalf("Parse after error returned unexpected statement: %#v", stmts[0])
	}
	param, ok := stmt.Fields.Items[0].Expr.(*ast.ParamMarkerExpr)
	if !ok || param.Index != 0 {
		t.Fatalf("Parse after error returned parameter %#v, want index zero", stmt.Fields.Items[0].Expr)
	}
}

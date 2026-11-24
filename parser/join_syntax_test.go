package parser_test

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/parser"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

func TestJoinSyntaxSpans(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, from, left, right string
		kind                    pkast.JoinType
	}{
		{"single", "`表`", "", "", pkast.JoinTypeUnknown},
		{"comma", "`表`,b", "`表`", "", pkast.JoinTypeInner},
		{"insert at left leaf", "`表` JOIN b LEFT JOIN c ON ?=?", "`表` JOIN b", "", pkast.JoinTypeLeft},
		{"explicit parentheses", "`表` JOIN ((b LEFT JOIN c ON ?=?))", "", "((b LEFT JOIN c ON ?=?))", pkast.JoinTypeInner},
		{"natural", "`表` NATURAL JOIN b", "", "", pkast.JoinTypeInner},
		{"straight", "`表` STRAIGHT_JOIN b ON ?=?", "", "", pkast.JoinTypeInner},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const prefix = "SELECT * FROM "
			sql := prefix + tc.from + " WHERE 1"
			stmt, err := parser.New().ParseOneStmt(sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			n := stmt.(*ast.SelectStmt).From.FromClauseCore().Table.(*ast.JoinExpr)
			if n.Type != tc.kind || n.Span() != joinSpan(len(prefix), len(tc.from)) {
				t.Fatalf("root type/span = %d %v, want %d over %q", n.Type, n.Span(), tc.kind, tc.from)
			}
			for _, child := range []struct {
				n    ast.ResultSetNode
				text string
			}{{n.Left, tc.left}, {n.Right, tc.right}} {
				if child.text == "" {
					continue
				}
				joined, ok := child.n.(*ast.JoinExpr)
				start := strings.Index(sql, child.text)
				if !ok || joined.Span() != joinSpan(start, len(child.text)) {
					t.Fatalf("child %T does not cover %q", child.n, child.text)
				}
			}
		})
	}
}

func TestJoinSyntaxNamesAndOnLifetime(t *testing.T) {
	t.Parallel()
	const sql = "SELECT * FROM (`库.点`.`表` AS `别` JOIN b USING (`列.点`)) LEFT JOIN c ON ?=?"
	p := parser.New()
	stmt, err := p.ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}
	root := stmt.(*ast.SelectStmt).From.FromClauseCore().Table.(*ast.JoinExpr)
	inner := root.Left.(*ast.JoinExpr)
	table := inner.Left.(*ast.TableSource)
	check := func() {
		t.Helper()
		if table.Alias.Raw != "`别`" || len(table.Source.(*ast.TableName).Name.Parts) != 2 || len(inner.Using) != 1 || len(inner.Using[0].Parts) != 1 {
			t.Fatal("lost alias or name segments")
		}
		id := inner.Using[0].Parts[0]
		start := strings.Index(sql, "`列.点`")
		if id.Name != "列.点" || id.Raw != "`列.点`" || id.Quote != lex.QuoteBacktick || id.Span != joinSpan(start, len(id.Raw)) {
			t.Fatalf("using identifier = %+v", id)
		}
		on := root.On.(*ast.BinaryOperationExpr)
		for i, p := range []*ast.ParamMarkerExpr{on.Left.(*ast.ParamMarkerExpr), on.Right.(*ast.ParamMarkerExpr)} {
			start := strings.Index(sql, "?") + 2*i
			if p.Index != i || p.Span() != joinSpan(start, 1) {
				t.Fatalf("ON parameter %d = index %d span %v", i, p.Index, p.Span())
			}
		}
	}
	check()
	if _, err := p.ParseOneStmt("SELECT ? FROM x JOIN (y JOIN z ON ?=?)", "", ""); err != nil {
		t.Fatal(err)
	}
	check()
}

func joinSpan(start, length int) loc.Span {
	return loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + length)}
}

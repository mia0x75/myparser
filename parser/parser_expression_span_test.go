package parser

import (
	"fmt"
	"slices"
	"testing"

	"github.com/mia0x75/myparser/ast"
)

type expressionSpanCollector struct {
	nodes *[]ast.Node
}

func (v expressionSpanCollector) Enter(n ast.Node) (ast.Node, bool) {
	*v.nodes = append(*v.nodes, n)
	return n, false
}

func (expressionSpanCollector) Leave(n ast.Node) (ast.Node, bool) { return n, true }

func TestExpressionReductionSpans(t *testing.T) {
	for _, tc := range []struct {
		name string
		expr string
		want map[string][]string
	}{
		{name: "logic or", expr: "a OR b"},
		{name: "logic xor", expr: "a XOR b"},
		{name: "logic and", expr: "a AND b"},
		{name: "not", expr: "NOT a"},
		{name: "bit or", expr: "a | b"},
		{name: "bit and", expr: "a & b"},
		{name: "left shift", expr: "a << b"},
		{name: "right shift", expr: "a >> b"},
		{name: "add", expr: "a + b"},
		{name: "subtract", expr: "a - b"},
		{name: "multiply", expr: "a * b"},
		{name: "divide", expr: "a / b"},
		{name: "percent", expr: "a % b"},
		{name: "integer divide", expr: "a DIV b"},
		{name: "modulo", expr: "a MOD b"},
		{name: "bit xor", expr: "a ^ b"},
		{name: "negate", expr: "-a"},
		{name: "positive", expr: "+a"},
		{name: "bit negate", expr: "~a"},
		{name: "bang", expr: "!a"},
		{name: "compare", expr: "a >= 1"},
		{name: "is true", expr: "a IS TRUE"},
		{name: "is false", expr: "a IS NOT FALSE"},
		{name: "is null", expr: "a IS NULL"},
		{name: "is unknown", expr: "a IS NOT UNKNOWN"},
		{name: "is json", expr: "a IS JSON"},
		{name: "between", expr: "a NOT BETWEEN 1 AND 2"},
		{name: "in list", expr: "a IN (1,2)"},
		{name: "in query", expr: "a NOT IN (SELECT 1)", want: map[string][]string{"*ast.SubqueryExpr": {"(SELECT 1)"}, "*ast.SelectStmt": {"SELECT 1"}}},
		{name: "case", expr: "CASE a WHEN 1 THEN b WHEN 2 THEN c ELSE d END", want: map[string][]string{"*ast.WhenClause": {"WHEN 1 THEN b", "WHEN 2 THEN c"}}},
		{name: "searched case", expr: "CASE WHEN a BETWEEN 1 AND 2 THEN b ELSE c END", want: map[string][]string{"*ast.WhenClause": {"WHEN a BETWEEN 1 AND 2 THEN b"}, "*ast.BetweenExpr": {"a BETWEEN 1 AND 2"}}},
		{name: "any query", expr: "a > ANY (SELECT 1)", want: map[string][]string{"*ast.SubqueryExpr": {"(SELECT 1)"}, "*ast.SelectStmt": {"SELECT 1"}}},
		{name: "all values", expr: "a = ALL (VALUES ROW(1))", want: map[string][]string{"*ast.SubqueryExpr": {"(VALUES ROW(1))"}, "*ast.ValuesStmt": {"VALUES ROW(1)"}}},
		{name: "like", expr: "a LIKE 'x' ESCAPE '!'"},
		{name: "regexp", expr: "a NOT REGEXP 'x'"},
		{name: "member", expr: "a MEMBER OF (b)"},
		{name: "sounds like", expr: "a SOUNDS LIKE b"},
		{name: "parentheses", expr: "(a + b)", want: map[string][]string{"*ast.BinaryOperationExpr": {"a + b"}}},
		{name: "row", expr: "ROW(a,b)"},
		{name: "bare row", expr: "(a,b)"},
		{name: "exists", expr: "EXISTS (SELECT 1)", want: map[string][]string{"*ast.SubqueryExpr": {"(SELECT 1)"}, "*ast.SelectStmt": {"SELECT 1"}}},
		{name: "not exists", expr: "NOT EXISTS (SELECT 1)", want: map[string][]string{"*ast.SubqueryExpr": {"(SELECT 1)"}, "*ast.SelectStmt": {"SELECT 1"}}},
		{name: "binary cast", expr: "BINARY a", want: map[string][]string{"*ast.TypeName": {"BINARY"}}},
		{name: "cast", expr: "CAST(a AS SIGNED)", want: map[string][]string{"*ast.TypeName": {"SIGNED"}}},
		{name: "convert", expr: "CONVERT(a, CHAR(8))", want: map[string][]string{"*ast.TypeName": {"CHAR(8)"}}},
		{name: "default", expr: "DEFAULT(a)"},
		{name: "values", expr: "VALUES(a)"},
		{name: "value", expr: "VALUE(a)"},
		{name: "json extract", expr: "a->'$.x'"},
		{name: "variable assignment", expr: "@v := 1", want: map[string][]string{"*ast.VariableTarget": {"@v"}}},
		{name: "compare assignment", expr: "a = @v := 1", want: map[string][]string{"*ast.VariableExpr": {"@v := 1"}, "*ast.VariableTarget": {"@v"}}},
		{name: "unicode and comments", expr: "`列` /* gap */ BETWEEN 1 AND 2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			sql := "/* 字节前缀 */ SELECT " + tc.expr
			stmt, err := p.ParseOneStmt(sql)
			if err != nil {
				t.Fatal(err)
			}
			expr := stmt.(*ast.SelectStmt).Fields.Items[0].Expr
			span := expr.Span()
			if got, err := span.Slice(sql); err != nil || got != tc.expr {
				t.Fatalf("root span %v = %q, want %q", span, got, tc.expr)
			}
			var nodes []ast.Node
			if _, ok := expr.Accept(expressionSpanCollector{nodes: &nodes}); !ok {
				t.Fatal("expression visitor failed")
			}
			checkNodeSpanTexts(t, sql, nodes, tc.want)
			if _, err := p.ParseOneStmt("SELECT 2"); err != nil {
				t.Fatal(err)
			}
			if expr.Span() != span {
				t.Fatalf("span changed after reuse: %v -> %v", span, expr.Span())
			}
		})
	}
}

func checkNodeSpanTexts(t *testing.T, sql string, nodes []ast.Node, expected map[string][]string) {
	t.Helper()
	for kind, want := range expected {
		var got []string
		for _, node := range nodes {
			if fmt.Sprintf("%T", node) == kind {
				text, err := node.Span().Slice(sql)
				if err != nil {
					t.Fatalf("%s has invalid span %v", kind, node.Span())
				}
				got = append(got, text)
			}
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s spans = %q, want %q", kind, got, want)
		}
	}
}

func TestJSONUnquoteSyntheticExtractionSpan(t *testing.T) {
	sql := "SELECT a->>'$.x'"
	stmt, err := New().ParseOneStmt(sql)
	if err != nil {
		t.Fatal(err)
	}
	call := stmt.(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.FuncCallExpr)
	if got, err := call.Span().Slice(sql); err != nil || got != "a->>'$.x'" {
		t.Fatalf("operator span = %q, error=%v", got, err)
	}
	if got := call.Args[0].Span(); got.Valid() {
		t.Fatalf("synthetic extraction span = %v", got)
	}
}

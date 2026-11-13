package ast_test

import (
	"testing"

	"github.com/mia0x75/myparser"
	"github.com/mia0x75/myparser/ast"
)

func TestHasAggFlag(t *testing.T) {
	expr := &ast.BetweenExpr{}
	flagTests := []struct {
		flag   uint64
		hasAgg bool
	}{
		{ast.FlagHasAggregateFunc, true},
		{ast.FlagHasAggregateFunc | ast.FlagHasVariable, true},
		{ast.FlagHasVariable, false},
	}
	for _, tt := range flagTests {
		expr.SetFlag(tt.flag)
		if got := ast.HasAggFlag(expr); got != tt.hasAgg {
			t.Fatalf("got %#v, want %#v", got, tt.hasAgg)
		}
	}
}

func TestFlag(t *testing.T) {
	p := myparser.New()
	flagTests := []struct {
		expr string
		flag uint64
	}{
		{
			"1 between 0 and 2",
			ast.FlagConstant,
		},
		{
			"case 1 when 1 then 1 else 0 end",
			ast.FlagConstant,
		},
		{
			"case 1 when 1 then 1 else 0 end",
			ast.FlagConstant,
		},
		{
			"case 1 when a > 1 then 1 else 0 end",
			ast.FlagConstant | ast.FlagHasReference,
		},
		{
			"1 = ANY (select 1) OR exists (select 1)",
			ast.FlagHasSubquery,
		},
		{
			"1 in (1) or 1 is true or null is null or 'abc' like 'abc' or 'abc' rlike 'abc'",
			ast.FlagConstant,
		},
		{
			"row (1, 1) = row (1, 1)",
			ast.FlagConstant,
		},
		{
			"(1 + a) > ?",
			ast.FlagHasReference | ast.FlagHasParamMarker,
		},
		{
			"trim('abc ')",
			ast.FlagHasFunc,
		},
		{
			"INTERVAL ? DAY + @date",
			ast.FlagHasFunc | ast.FlagHasParamMarker | ast.FlagHasVariable,
		},
		{
			"now() + EXTRACT(YEAR FROM '2009-07-02') + CAST(1 AS UNSIGNED)",
			ast.FlagHasFunc,
		},
		{
			"substring('abc', 1)",
			ast.FlagHasFunc,
		},
		{
			"sum(a)",
			ast.FlagHasAggregateFunc | ast.FlagHasReference,
		},
		{
			"(select 1) as a",
			ast.FlagHasSubquery,
		},
		{
			"@auto_commit",
			ast.FlagHasVariable,
		},
		{
			"default(a)",
			ast.FlagHasDefault,
		},
		{
			"a is null",
			ast.FlagHasReference,
		},
		{
			"a is json",
			ast.FlagHasReference,
		},
		{
			"a member of (b)",
			ast.FlagHasReference,
		},
		{
			"a sounds like b",
			ast.FlagHasReference,
		},
		{
			"sum(a) is json",
			ast.FlagHasAggregateFunc | ast.FlagHasReference,
		},
		{
			"1 is true",
			ast.FlagConstant,
		},
		{
			"a in (1, count(*), 3)",
			ast.FlagConstant | ast.FlagHasReference | ast.FlagHasAggregateFunc,
		},
		{
			"'Michael!' REGEXP '.*'",
			ast.FlagConstant,
		},
		{
			"a REGEXP '.*'",
			ast.FlagHasReference,
		},
		{
			"-a",
			ast.FlagHasReference,
		},
	}
	for _, tt := range flagTests {
		stmt, err := p.ParseOneStmt("select "+tt.expr, "", "")
		if err != nil {
			t.Fatalf("For %s, got error %v, want nil", tt.expr, err)
		}
		selectStmt := stmt.(*ast.SelectStmt)
		ast.SetFlag(selectStmt)
		expr := selectStmt.Fields.Fields[0].Expr
		if got := expr.GetFlag(); got != tt.flag {
			t.Fatalf("For %s, got %#v, want %#v", tt.expr, got, tt.flag)
		}
	}
}

func TestAggregateTailFlags(t *testing.T) {
	for _, tt := range []struct {
		name string
		sql  string
		flag uint64
	}{
		{name: "order expression", sql: "SELECT GROUP_CONCAT(1 ORDER BY @v)", flag: ast.FlagHasAggregateFunc | ast.FlagHasVariable},
		{name: "limit count", sql: "SELECT GROUP_CONCAT(1 LIMIT ?)", flag: ast.FlagHasAggregateFunc | ast.FlagHasParamMarker},
		{name: "limit offset", sql: "SELECT JSON_ARRAYAGG(1 LIMIT ?,1)", flag: ast.FlagHasAggregateFunc | ast.FlagHasParamMarker},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stmt, err := myparser.New().ParseOneStmt(tt.sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			ast.SetFlag(stmt)
			expr := stmt.(*ast.SelectStmt).Fields.Fields[0].Expr
			if got := expr.GetFlag(); got != tt.flag {
				t.Fatalf("flags = %d, want %d", got, tt.flag)
			}
		})
	}
	separator := &ast.VariableExpr{Name: "separator"}
	agg := &ast.AggregateFuncExpr{F: "group_concat", Args: []ast.ExprNode{ast.NewValueExpr(1)}, Separator: separator}
	ast.SetFlag(agg)
	if got := agg.GetFlag(); got != ast.FlagHasAggregateFunc|ast.FlagHasVariable {
		t.Fatalf("separator flags = %d", got)
	}
}

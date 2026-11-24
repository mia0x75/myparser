package parser

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	pkast "github.com/mia0x75/parserkit/ast"
)

// literalIntegerKind is the shared kind of an integer literal.
const literalIntegerKind = pkast.LiteralInteger

// TestNumericLiteralKindsAndSpans locks the numeric literal syntax: the kind
// tells the numeric family, Raw keeps the source spelling, and Value keeps the
// canonical text.
func TestNumericLiteralKindsAndSpans(t *testing.T) {
	src := "select 007,1.50,1e5,18446744073709551615,18446744073709551616"
	stmt := mustParseOne(t, src).(*ast.SelectStmt)
	want := []struct {
		kind  pkast.LiteralKind
		raw   string
		value string
	}{
		{pkast.LiteralInteger, "007", "7"},
		{pkast.LiteralDecimal, "1.50", "1.50"},
		{pkast.LiteralFloat, "1e5", "1e+05"},
		{pkast.LiteralInteger, "18446744073709551615", "18446744073709551615"},
		{pkast.LiteralDecimal, "18446744073709551616", "18446744073709551616"},
	}
	fields := stmt.Fields.Items
	if len(fields) != len(want) {
		t.Fatalf("got %d fields, want %d", len(fields), len(want))
	}
	for i, w := range want {
		literal, ok := fields[i].Expr.(*ast.LiteralExpr)
		if !ok {
			t.Fatalf("field %d: got %T, want *ast.LiteralExpr", i, fields[i].Expr)
		}
		if literal.Kind != w.kind || literal.Raw != w.raw || literal.Value != w.value {
			t.Errorf("field %d: got kind=%v raw=%q value=%q, want kind=%v raw=%q value=%q",
				i, literal.Kind, literal.Raw, literal.Value, w.kind, w.raw, w.value)
		}
		span := literal.Span()
		start := strings.Index(src, w.raw)
		if start < 0 {
			t.Fatalf("field %d: %q is missing from the source", i, w.raw)
		}
		if int(span.Start) != start || int(span.End) != start+len(w.raw) {
			t.Errorf("field %d: got span %v, want [%d,%d)", i, span, start, start+len(w.raw))
		}
		if text := src[int(span.Start):int(span.End)]; text != w.raw {
			t.Errorf("field %d: got source %q, want %q", i, text, w.raw)
		}
	}
}

// TestNumericLiteralSurvivesParserReuse keeps a parsed literal alive when the
// same parser parses another statement afterwards.
func TestNumericLiteralSurvivesParserReuse(t *testing.T) {
	p := New()
	stmt, err := p.ParseOneStmt("select 007,1.50", "", "")
	if err != nil {
		t.Fatal(err)
	}
	fields := stmt.(*ast.SelectStmt).Fields.Items
	integer := fields[0].Expr.(*ast.LiteralExpr)
	decimal := fields[1].Expr.(*ast.LiteralExpr)
	if _, err := p.ParseOneStmt("select 999999999999999999999999999999", "", ""); err != nil {
		t.Fatal(err)
	}
	if integer.Raw != "007" || integer.Value != "7" {
		t.Errorf("integer literal changed after a later parse: raw=%q value=%q", integer.Raw, integer.Value)
	}
	if decimal.Raw != "1.50" || decimal.Value != "1.50" {
		t.Errorf("decimal literal changed after a later parse: raw=%q value=%q", decimal.Raw, decimal.Value)
	}
}

// TestNumericLiteralInOrderByAndLimit locks numeric literals in ORDER BY and
// LIMIT clauses.
func TestNumericLiteralInOrderByAndLimit(t *testing.T) {
	stmt := mustParseOne(t, "select a from t order by 1 desc, 2 limit 3").(*ast.SelectStmt)
	if stmt.OrderBy == nil || len(stmt.OrderBy.OrderByClauseCore().Items) != 2 {
		t.Fatalf("ORDER BY was lost: %#v", stmt.OrderBy)
	}
	for i, want := range []int{1, 2} {
		orderItem := stmt.OrderBy.OrderByClauseCore().Items[i]
		position, ok := orderItem.OrderItemCore().Expr.(*ast.PositionExpr)
		if !ok {
			t.Fatalf("ORDER BY item %d: got %T, want *ast.PositionExpr", i, orderItem.OrderItemCore().Expr)
		}
		if position.N != want {
			t.Errorf("ORDER BY item %d: got position %d, want %d", i, position.N, want)
		}
	}
	fraction := mustParseOne(t, "select a from t order by 1.5").(*ast.SelectStmt)
	orderItem := fraction.OrderBy.OrderByClauseCore().Items[0]
	literal, ok := orderItem.OrderItemCore().Expr.(*ast.LiteralExpr)
	if !ok {
		t.Fatalf("ORDER BY fraction: got %T, want *ast.LiteralExpr", orderItem.OrderItemCore().Expr)
	}
	if literal.Kind != pkast.LiteralDecimal || literal.Raw != "1.5" || literal.Value != "1.5" {
		t.Errorf("ORDER BY fraction: got kind=%v raw=%q value=%q, want kind=%v raw=%q value=%q",
			literal.Kind, literal.Raw, literal.Value, pkast.LiteralDecimal, "1.5", "1.5")
	}
	if stmt.Limit == nil {
		t.Fatal("LIMIT was lost")
	}
	count, ok := stmt.Limit.LimitCore().Count.(*ast.LiteralExpr)
	if !ok {
		t.Fatalf("LIMIT count: got %T, want *ast.LiteralExpr", stmt.Limit.LimitCore().Count)
	}
	if count.Kind != pkast.LiteralInteger || count.Value != "3" {
		t.Errorf("LIMIT count: got kind=%v value=%q, want kind=%v value=%q",
			count.Kind, count.Value, pkast.LiteralInteger, "3")
	}
}

// TestNumericLiteralInNowAndKill locks numeric literals in NOW() and KILL.
func TestNumericLiteralInNowAndKill(t *testing.T) {
	now := mustParseOne(t, "select now(3)").(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.FuncCallExpr)
	if len(now.Args) != 1 {
		t.Fatalf("NOW() argument was lost: %#v", now.Args)
	}
	literal, ok := now.Args[0].(*ast.LiteralExpr)
	if !ok {
		t.Fatalf("NOW() argument: got %T, want *ast.LiteralExpr", now.Args[0])
	}
	if literal.Kind != pkast.LiteralInteger || literal.Raw != "3" || literal.Value != "3" {
		t.Errorf("NOW() argument: got kind=%v raw=%q value=%q, want kind=%v raw=%q value=%q",
			literal.Kind, literal.Raw, literal.Value, pkast.LiteralInteger, "3", "3")
	}
	kill := mustParseOne(t, "kill 1").(*ast.KillStmt)
	if kill.ConnectionID != 1 || kill.Expr != nil {
		t.Fatalf("KILL connection id was lost: %#v", kill)
	}
}

// TestNumericLiteralRoundTrips keeps numeric literals in the default, LIMIT,
// window, and sequence clauses.
func TestNumericLiteralRoundTrips(t *testing.T) {
	cases := []struct {
		src      string
		restored string
	}{
		{"create table t(a int default 1)", "CREATE TABLE `t` (`a` INT DEFAULT 1)"},
		{"select * from t limit 1", "SELECT * FROM `t` LIMIT 1"},
		{
			"select sum(a) over (rows between 1 preceding and 2 following) from t",
			"SELECT SUM(`a`) OVER (ROWS BETWEEN 1 PRECEDING AND 2 FOLLOWING) FROM `t`",
		},
		{"create sequence s start with 1 minvalue 0", "CREATE SEQUENCE `s` START WITH 1 MINVALUE 0"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			RunRestoreTest(t, tc.src, tc.restored, true)
		})
	}
	p := New()
	p.EnableWindowFunc(true)
	parsed, err := p.ParseOneStmt("select sum(a) over (rows between 1 preceding and 2 following) from t", "", "")
	if err != nil {
		t.Fatal(err)
	}
	stmt := parsed.(*ast.SelectStmt)
	window := stmt.Fields.Items[0].Expr.(*ast.WindowFuncExpr)
	for _, bound := range []struct {
		name string
		expr ast.ExprNode
	}{
		{"start", window.Spec.Frame.(*ast.FrameClause).Extent.Start.WindowFrameBoundCore().Expr},
		{"end", window.Spec.Frame.(*ast.FrameClause).Extent.End.WindowFrameBoundCore().Expr},
	} {
		literal, ok := bound.expr.(*ast.LiteralExpr)
		if !ok {
			t.Fatalf("frame %s bound: got %T, want *ast.LiteralExpr", bound.name, bound.expr)
		}
		if literal.Kind != pkast.LiteralInteger {
			t.Errorf("frame %s bound: got kind=%v, want kind=%v", bound.name, literal.Kind, pkast.LiteralInteger)
		}
	}
}

func TestReplicationNumericLiteralMetadata(t *testing.T) {
	for _, tc := range []struct {
		name string
		sql  string
		want string
	}{
		{"hex", "change master to master_port=0x10", "0x10"},
		{"unsigned", "change master to master_port=0010", "0010"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmt := mustParseOne(t, tc.sql).(*ast.ChangeMasterStmt)
			literal, ok := stmt.Options[0].Expr.(*ast.LiteralExpr)
			if !ok {
				t.Fatalf("option value: got %T, want *ast.LiteralExpr", stmt.Options[0].Expr)
			}
			span := literal.Span()
			if literal.Raw != tc.want || !span.Within(len(tc.sql)) || tc.sql[span.Start:span.End] != tc.want {
				t.Fatalf("literal metadata: raw=%q span=%v, want raw=%q with source-width span", literal.Raw, span, tc.want)
			}
		})
	}
}

func TestSequenceSignedLiteralMetadata(t *testing.T) {
	for _, raw := range []string{"007", "+007", "-007", "-9223372036854775808"} {
		t.Run(raw, func(t *testing.T) {
			sql := "create sequence s increment by " + raw
			stmt := mustParseOne(t, sql).(*ast.CreateSequenceStmt)
			literal := stmt.Options[0].Value.(*ast.LiteralExpr)
			span := literal.Span()
			if literal.Raw != raw || !span.Within(len(sql)) || sql[span.Start:span.End] != raw {
				t.Fatalf("signed literal: raw=%q span=%v, want %q", literal.Raw, span, raw)
			}
		})
	}
}

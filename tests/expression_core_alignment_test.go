package parser_test

import (
	"testing"

	dialectast "github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/parser"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/loc"
)

func TestParseSharedLiteralCore(t *testing.T) {
	const prefix = "SELECT "
	for _, tc := range []struct {
		raw   string
		kind  pkast.LiteralKind
		value string
	}{
		{raw: "TrUe", kind: pkast.LiteralBoolean, value: "true"},
		{raw: "FaLsE", kind: pkast.LiteralBoolean, value: "false"},
		{raw: "NuLl", kind: pkast.LiteralNull, value: "null"},
		{raw: "001", kind: pkast.LiteralInteger, value: "1"},
		{raw: "1.25", kind: pkast.LiteralDecimal, value: "1.25"},
		{raw: "'a''b'", kind: pkast.LiteralString, value: "a'b"},
		{raw: "B'101'", kind: pkast.LiteralBit, value: "101"},
		{raw: "X'cafe'", kind: pkast.LiteralHex, value: "cafe"},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			sql := prefix + tc.raw + ", 0"
			stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)
			expr := stmt.Fields.Items[0].Expr
			lit, ok := expr.(*dialectast.LiteralExpr)
			if !ok {
				t.Fatalf("expression = %T, want a literal", expr)
			}
			core := lit.BaseLiteralExpr
			wantSpan := loc.Span{Start: loc.ByteOffset(len(prefix)), End: loc.ByteOffset(len(prefix) + len(tc.raw))}
			if core.Kind != tc.kind || core.Value != tc.value || core.Raw != tc.raw || core.Span() != wantSpan {
				t.Fatalf("literal core = %#v, want kind %v, value %q, raw %q, span %v", core, tc.kind, tc.value, tc.raw, wantSpan)
			}
		})
	}
	if _, err := parser.New().ParseOneStmt("SELECT 'unterminated"); err == nil {
		t.Fatal("accepted an unterminated literal")
	}
}

func TestParseSharedParameterCore(t *testing.T) {
	const sql = "SELECT ?, ?"
	stmt := mustParseOne(t, sql).(*dialectast.SelectStmt)
	for i := range 2 {
		expr := stmt.Fields.Items[i].Expr
		param, ok := expr.(*dialectast.ParamMarkerExpr)
		if !ok {
			t.Fatalf("expression %d = %T, want a parameter", i, expr)
		}
		core := param.BaseParameterExpr
		span := core.Span()
		if core.Index != i || !span.Within(len(sql)) || sql[span.Start:span.End] != "?" {
			t.Fatalf("parameter %d: index = %d, span = %v, want ?", i, core.Index, span)
		}
	}
	if _, err := parser.New().ParseOneStmt("SELECT ?[1]"); err == nil {
		t.Fatal("accepted a subscripted parameter")
	}
}

package myparser

import (
	"strings"
	"testing"
	"unsafe"

	"github.com/mia0x75/myparser/ast"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

func callSyntaxCore(t *testing.T, expr ast.ExprNode) *pkast.BaseCallExpr {
	t.Helper()
	switch expr := expr.(type) {
	case *ast.AggregateFuncExpr:
		return &expr.BaseCallExpr
	case *ast.WindowFuncExpr:
		return &expr.BaseCallExpr
	default:
		t.Fatalf("call type = %T", expr)
		return nil
	}
}

func TestAggregateWindowNamesAndSpans(t *testing.T) {
	for _, call := range []string{
		"AvG(DISTINCT x)", "BIT_AND(x)", "BIT_AND(ALL x)", "BIT_OR(x)", "BIT_OR(ALL x)",
		"BIT_XOR(x)", "BIT_XOR(ALL x)", "COUNT(DISTINCT x,y)", "COUNT(ALL x)", "COUNT(x)", "COUNT(*)",
		"GROUP_CONCAT(DISTINCT x,y ORDER BY x DESC SEPARATOR '|' LIMIT 2)", "JSON_ARRAYAGG(x)",
		"MAX(DISTINCT x)", "MIN(x)", "SUM(x)", "STDDEV_POP(x)", "STDDEV_SAMP(x)", "VARIANCE(x)", "VaR_SaMp(x)",
	} {
		for _, window := range []bool{false, true} {
			name := call
			if window {
				// GROUP_CONCAT's legacy window-tail loss is outside the name migration.
				if strings.HasPrefix(call, "GROUP_CONCAT") {
					name = "GROUP_CONCAT(x,y)"
				}
				name += " OVER (PARTITION BY x ORDER BY y)"
			}
			t.Run(name, func(t *testing.T) {
				checkCallNameSpan(t, name, name[:strings.IndexByte(name, '(')], window)
			})
		}
	}
	for _, call := range []string{
		"ROW_NUMBER() OVER ()", "RANK() OVER ()", "DENSE_RANK() OVER ()", "CUME_DIST() OVER ()", "PERCENT_RANK() OVER ()",
		"NTILE(4) OVER ()", "LEAD(x,2,3) IGNORE NULLS OVER ()", "LAG(x) OVER ()",
		"FIRST_VALUE(x) OVER ()", "LAST_VALUE(x) OVER ()", "NTH_VALUE(x,2) FROM LAST IGNORE NULLS OVER ()",
		"Custom(x) OVER ()", "函數(x) OVER ()", "$f(x) OVER ()", "123abc(x) OVER ()",
		"PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY x) OVER ()",
		"`函``数`(x) OVER ()", "``(x) OVER ()", "`a.b`(x) OVER ()", "`Mixed``Name`(x) WITHIN GROUP (ORDER BY y) OVER ()",
	} {
		t.Run(call, func(t *testing.T) {
			checkCallNameSpan(t, call, call[:strings.IndexByte(call, '(')], true)
		})
	}
}

func checkCallNameSpan(t *testing.T, call, rawName string, window bool) {
	t.Helper()
	p := New()
	p.EnableWindowFunc(true)
	sql := "SELECT /* 中文 */ " + call
	stmt, err := p.ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}
	expr := stmt.(*ast.SelectStmt).Fields.Items[0].Expr
	core := callSyntaxCore(t, expr)
	_, isWindow := expr.(*ast.WindowFuncExpr)
	if isWindow != window {
		t.Fatalf("window=%v, want %v", isWindow, window)
	}
	if len(core.Name.Parts) != 1 {
		t.Fatalf("name parts = %#v, want a single part", core.Name.Parts)
	}
	id := core.Name.Parts[0]
	if !id.Present() || id.Raw != rawName {
		t.Fatalf("name = %+v, want raw %q", id, rawName)
	}
	if id.Quote == lex.QuoteUnquoted && !strings.EqualFold(rawName, "variance") && id.Name != rawName {
		t.Fatalf("decoded name = %q, want original spelling %q", id.Name, rawName)
	}
	if got, err := core.Name.Span.Slice(sql); err != nil || got != rawName {
		t.Fatalf("name span selects %q (%v), want %q", got, err, rawName)
	}
	if core.Name.Span != id.Span {
		t.Fatalf("name span = %v, part span = %v", core.Name.Span, id.Span)
	}
	if unsafe.StringData(core.Name.Parts[0].Raw) != unsafe.StringData(sql[core.Name.Span.Start:core.Name.Span.End]) {
		t.Fatal("raw name does not share the input substring")
	}
	if id.Quote == lex.QuoteUnquoted && !strings.EqualFold(rawName, "variance") &&
		unsafe.StringData(core.Name.Parts[0].Name) != unsafe.StringData(core.Name.Parts[0].Raw) {
		t.Fatal("unquoted name does not share the input substring")
	}
	if got, err := expr.Span().Slice(sql); err != nil || got != call {
		t.Fatalf("call span selects %q (%v), want %q", got, err, call)
	}
	if rawName[0] == '`' {
		decoded := strings.ReplaceAll(rawName[1:len(rawName)-1], "``", "`")
		if id.Name != decoded || id.Quote != lex.QuoteBacktick {
			t.Fatalf("decoded name = %+v, want %q", id, decoded)
		}
	}
	id.Name, id.Raw = strings.Clone(id.Name), strings.Clone(id.Raw)
	canonical := restoreStmt(t, stmt)
	reparsed, err := p.ParseOneStmt(canonical, "", "")
	if err != nil {
		t.Fatalf("reparse %q: %v", canonical, err)
	}
	next := callSyntaxCore(t, reparsed.(*ast.SelectStmt).Fields.Items[0].Expr)
	if len(next.Name.Parts) != 1 {
		t.Fatalf("reparsed name parts = %#v, want a single part", next.Name.Parts)
	}
	if !strings.EqualFold(next.Name.Parts[0].Name, id.Name) || len(next.Args) != len(core.Args) {
		t.Fatal("round trip changed call name or arguments")
	}
	if _, err := p.ParseOneStmt("SELECT `不``同`(x) OVER ()", "", ""); err != nil {
		t.Fatal(err)
	}
	if len(core.Name.Parts) != 1 ||
		core.Name.Parts[0] != id ||
		core.Name.Span != id.Span ||
		restoreStmt(t, stmt) != canonical {
		t.Fatal("parser reuse changed retained call")
	}
}

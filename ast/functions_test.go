package ast_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mia0x75/myparser"
	. "github.com/mia0x75/myparser/ast"
	dformat "github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

func TestFunctionsVisitorCover(t *testing.T) {
	valueExpr := &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "42"}}
	stmts := []Node{
		&AggregateFuncExpr{BaseCallExpr: pkast.BaseCallExpr{Args: []ExprNode{valueExpr}}},
		&FuncCallExpr{BaseCallExpr: pkast.BaseCallExpr{Args: []ExprNode{valueExpr}}},
		&FuncCastExpr{Expr: valueExpr},
		&WindowFuncExpr{Spec: WindowSpec{}},
	}

	for _, stmt := range stmts {
		stmt.Accept(visitor{})
		stmt.Accept(visitor1{})
	}
}

func TestFuncCallExprRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"JSON_ARRAYAGG(attribute)", "JSON_ARRAYAGG(`attribute`)"},
		{"JSON_OBJECTAGG(attribute, value)", "JSON_OBJECTAGG(`attribute`, `value`)"},
		{"ABS(-1024)", "ABS(-1024)"},
		{"ACOS(3.14)", "ACOS(3.14)"},
		{"CONV('a',16,2)", "CONV('a', 16, 2)"},
		{"COS(PI())", "COS(PI())"},
		{"RAND()", "RAND()"},
		{"ADDDATE('2000-01-01', 1)", "ADDDATE('2000-01-01', INTERVAL 1 DAY)"},
		{"DATE_ADD('2000-01-01', INTERVAL 1 DAY)", "DATE_ADD('2000-01-01', INTERVAL 1 DAY)"},
		{"INTERVAL 1 DAY + '1997-12-31'", "INTERVAL 1 DAY + '1997-12-31'"},
		{"-INTERVAL 1 DAY + '1997-12-31'", "-INTERVAL 1 DAY + '1997-12-31'"},
		{"-(INTERVAL 1 DAY + '1997-12-31')", "-(INTERVAL 1 DAY + '1997-12-31')"},
		{"INTERVAL (1+1) DAY + '1997-12-31'", "INTERVAL (1+1) DAY + '1997-12-31'"},
		{"DATE_ADD('2000-01-01', INTERVAL '1 1:12:23.100000' DAY_MICROSECOND)", "DATE_ADD('2000-01-01', INTERVAL '1 1:12:23.100000' DAY_MICROSECOND)"},
		{"EXTRACT(DAY FROM '2000-01-01')", "EXTRACT(DAY FROM '2000-01-01')"},
		{"extract(day from '1999-01-01')", "EXTRACT(DAY FROM '1999-01-01')"},
		{"GET_FORMAT(DATE, 'EUR')", "GET_FORMAT(DATE, 'EUR')"},
		{"POSITION('a' IN 'abc')", "POSITION('a' IN 'abc')"},
		{"TRIM('  bar   ')", "TRIM('  bar   ')"},
		{"TRIM('a' FROM '  bar   ')", "TRIM('a' FROM '  bar   ')"},
		{"TRIM(LEADING FROM '  bar   ')", "TRIM(LEADING FROM '  bar   ')"},
		{"TRIM(BOTH FROM '  bar   ')", "TRIM(BOTH FROM '  bar   ')"},
		{"TRIM(TRAILING FROM '  bar   ')", "TRIM(TRAILING FROM '  bar   ')"},
		{"TRIM(LEADING 'x' FROM 'xxxyxxx')", "TRIM(LEADING 'x' FROM 'xxxyxxx')"},
		{"TRIM(BOTH 'x' FROM 'xxxyxxx')", "TRIM(BOTH 'x' FROM 'xxxyxxx')"},
		{"TRIM(TRAILING 'x' FROM 'xxxyxxx')", "TRIM(TRAILING 'x' FROM 'xxxyxxx')"},
		{"DATE_ADD('2008-01-02', INTERVAL INTERVAL(1, 0, 1) DAY)", "DATE_ADD('2008-01-02', INTERVAL INTERVAL(1, 0, 1) DAY)"},
		{"BENCHMARK(1000000, AES_ENCRYPT('text', UNHEX('F3229A0B371ED2D9441B830D21A390C3')))", "BENCHMARK(1000000, AES_ENCRYPT('text', UNHEX('F3229A0B371ED2D9441B830D21A390C3')))"},
		{"SUBSTRING('Quadratically', 5)", "SUBSTRING('Quadratically', 5)"},
		{"SUBSTRING('Quadratically' FROM 5)", "SUBSTRING('Quadratically', 5)"},
		{"SUBSTRING('Quadratically', 5, 6)", "SUBSTRING('Quadratically', 5, 6)"},
		{"SUBSTRING('Quadratically' FROM 5 FOR 6)", "SUBSTRING('Quadratically', 5, 6)"},
		{"MASTER_POS_WAIT(@log_name, @log_pos, @timeout, @channel_name)", "MASTER_POS_WAIT(@`log_name`, @`log_pos`, @`timeout`, @`channel_name`)"},
		{"JSON_TYPE('[123]')", "JSON_TYPE('[123]')"},
		{"bit_and(all c1)", "BIT_AND(`c1`)"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Items[0].Expr
	}
	RunNodeRestoreTest(t, testCases, "select %s", extractNodeFunc)
}

func TestFuncCallNameKeepsSourceSyntax(t *testing.T) {
	parser := myparser.New()
	for _, tc := range []struct {
		name         string
		sql          string
		functionName string
		raw          string
		quote        lex.QuoteStyle
	}{
		{name: "unquoted", sql: "select MiXeD_fn(1)", functionName: "MiXeD_fn", raw: "MiXeD_fn"},
		{name: "backtick", sql: "select `MiXeD_fn`(1)", functionName: "MiXeD_fn", raw: "`MiXeD_fn`", quote: lex.QuoteBacktick},
		{name: "keyword", sql: "SELECT NOW()", functionName: "NOW", raw: "NOW"},
		{name: "canonical alias", sql: "SELECT CURRENT_TIMESTAMP", functionName: "CURRENT_TIMESTAMP", raw: "CURRENT_TIMESTAMP"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmt, err := parser.ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			call := stmt.(*SelectStmt).Fields.Items[0].Expr.(*FuncCallExpr)
			if len(call.Name.Parts) != 1 {
				t.Fatalf("function name parts = %#v, want a single part", call.Name.Parts)
			}
			part := call.Name.Parts[0]
			if call.Name.Span != part.Span {
				t.Fatalf("function name span = %v, part span = %v", call.Name.Span, part.Span)
			}
			if part.Name != tc.functionName || part.Quote != tc.quote {
				t.Fatalf("function name = %#v", part)
			}
			if got, err := call.Name.Span.Slice(tc.sql); err != nil || got != tc.raw || part.Raw != got {
				t.Fatalf("function name syntax = raw %q, span %q, err %v", part.Raw, got, err)
			}
		})
	}
}

func TestFuncCallExprFormatRejectsMissingCharset(t *testing.T) {
	expr := &FuncCallExpr{
		BaseCallExpr: pkast.BaseCallExpr{
			Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "convert"}}},
			Args: []ExprNode{&LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralString, Value: "value"}}},
		},
	}

	var formatted strings.Builder
	expr.Format(&formatted)
	if got, want := formatted.String(), "<invalid function syntax>"; got != want {
		t.Fatalf("Format() = %q, want %q", got, want)
	}
}

func TestFuncCallExprRestoreRejectsInvalidCharset(t *testing.T) {
	expr := &FuncCallExpr{
		BaseCallExpr: pkast.BaseCallExpr{
			Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "convert"}}},
			Args: []ExprNode{&LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralString, Value: "value"}}},
		},
		Charset: lex.Ident{Name: "utf8; DROP TABLE users; --"},
	}
	var out strings.Builder
	err := expr.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out))
	if err == nil || out.Len() != 0 {
		t.Fatalf("Restore wrote %q, error = %v; want no output and an error", out.String(), err)
	}
}

func TestAggregateFuncStarRestore(t *testing.T) {
	restore := func(expr ExprNode) string {
		var out strings.Builder
		if err := expr.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); err != nil {
			t.Fatalf("Restore: %v", err)
		}
		return out.String()
	}
	star := &AggregateFuncExpr{
		BaseCallExpr: pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "count"}}}},
		Star:         true,
	}
	if got, want := restore(star), "COUNT(*)"; got != want {
		t.Fatalf("COUNT(*) = %q, want %q", got, want)
	}
	if len(star.Args) != 0 {
		t.Fatalf("star argument = %#v, want none", star.Args)
	}
	// Star is only valid for COUNT; other aggregates must reject it.
	aggStar := &AggregateFuncExpr{BaseCallExpr: pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "sum"}}}}, Star: true}
	if err := aggStar.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &strings.Builder{})); err == nil {
		t.Fatal("aggregate star restored without error")
	}
	for _, invalid := range []*AggregateFuncExpr{
		{BaseCallExpr: pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "count"}}}, Args: []ExprNode{&LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "1"}}}}, Star: true},
		{BaseCallExpr: pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "count"}}}}, Distinct: true, Star: true},
	} {
		if err := invalid.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &strings.Builder{})); err == nil {
			t.Fatal("invalid COUNT(*) combination restored without error")
		}
	}
}

func TestFuncCastExprRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"CONVERT('Müller' USING UtF8Mb4)", "CONVERT('Müller' USING UTF8MB4)"},
		{"CONVERT('Müller', CHAR(32) CHARACTER SET UtF8)", "CONVERT('Müller', CHAR(32) CHARACTER SET utf8)"},
		{"CAST('test' AS CHAR CHARACTER SET UtF8)", "CAST('test' AS CHAR CHARACTER SET utf8)"},
		{"BINARY 'New York'", "BINARY 'New York'"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Items[0].Expr
	}
	RunNodeRestoreTest(t, testCases, "select %s", extractNodeFunc)
}

func TestAggregateFuncExprRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"AVG(test_score)", "AVG(`test_score`)"},
		{"AVG(distinct test_score)", "AVG(DISTINCT `test_score`)"},
		{"BIT_AND(test_score)", "BIT_AND(`test_score`)"},
		{"BIT_OR(test_score)", "BIT_OR(`test_score`)"},
		{"BIT_XOR(test_score)", "BIT_XOR(`test_score`)"},
		{"COUNT(test_score)", "COUNT(`test_score`)"},
		{"COUNT(*)", "COUNT(*)"},
		{"COUNT(DISTINCT scores, results)", "COUNT(DISTINCT `scores`, `results`)"},
		{"GROUP_CONCAT(a,b)", "GROUP_CONCAT(`a`, `b`)"},
		{"GROUP_CONCAT(DISTINCT a,b ORDER BY a DESC,b SEPARATOR '|' LIMIT 2 OFFSET 1)", "GROUP_CONCAT(DISTINCT `a`, `b` ORDER BY `a` DESC,`b` SEPARATOR '|' LIMIT 1,2)"},
		{"GROUP_CONCAT(a ORDER BY b LIMIT 2)", "GROUP_CONCAT(`a` ORDER BY `b` LIMIT 2)"},
		{"JSON_ARRAYAGG(attribute)", "JSON_ARRAYAGG(`attribute`)"},
		{"JSON_ARRAYAGG(DISTINCT a ORDER BY b DESC LIMIT 1,2)", "JSON_ARRAYAGG(DISTINCT `a` ORDER BY `b` DESC LIMIT 1,2)"},
		{"MIN(test_score)", "MIN(`test_score`)"},
		{"MIN(DISTINCT test_score)", "MIN(DISTINCT `test_score`)"},
		{"MAX(test_score)", "MAX(`test_score`)"},
		{"MAX(DISTINCT test_score)", "MAX(DISTINCT `test_score`)"},
		{"STD(test_score)", "STD(`test_score`)"},
		{"STDDEV(test_score)", "STDDEV(`test_score`)"},
		{"STDDEV_POP(test_score)", "STDDEV_POP(`test_score`)"},
		{"STDDEV_SAMP(test_score)", "STDDEV_SAMP(`test_score`)"},
		{"SUM(test_score)", "SUM(`test_score`)"},
		{"SUM(DISTINCT test_score)", "SUM(DISTINCT `test_score`)"},
		{"VAR_POP(test_score)", "VAR_POP(`test_score`)"},
		{"VAR_SAMP(test_score)", "VAR_SAMP(`test_score`)"},
		{"VARIANCE(test_score)", "VAR_POP(`test_score`)"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Items[0].Expr
	}
	RunNodeRestoreTest(t, testCases, "select %s", extractNodeFunc)
}

func TestAggregateFuncExprTailVisitor(t *testing.T) {
	stmt, err := myparser.New().ParseOneStmt("SELECT GROUP_CONCAT(a,b ORDER BY c SEPARATOR '|' LIMIT 2,3)", "", "")
	if err != nil {
		t.Fatal(err)
	}
	agg := stmt.(*SelectStmt).Fields.Items[0].Expr.(*AggregateFuncExpr)
	if len(agg.Args) != 2 || agg.Separator == nil || agg.OrderBy == nil || agg.Limit == nil {
		t.Fatalf("aggregate tails = %#v", agg)
	}
	checker := &aggregateTailVisitor{seen: make(map[Node]bool)}
	if _, ok := agg.Accept(checker); !ok {
		t.Fatal("visitor aborted")
	}
	for _, node := range []Node{agg.Separator, agg.OrderBy, agg.OrderBy.Items[0].OrderItemCore().Expr, agg.Limit, agg.Limit.Count, agg.Limit.Offset} {
		if !checker.seen[node] {
			t.Fatalf("visitor missed %T", node)
		}
	}
}

type aggregateTailVisitor struct {
	seen map[Node]bool
}

func (v *aggregateTailVisitor) Enter(n Node) (Node, bool) {
	v.seen[n] = true
	return n, false
}

func (v *aggregateTailVisitor) Leave(n Node) (Node, bool) { return n, true }

func TestCallCarriersVisitorBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name string
		new  func(pkast.BaseCallExpr) ExprNode
	}{
		{"aggregate", func(call pkast.BaseCallExpr) ExprNode { return &AggregateFuncExpr{BaseCallExpr: call} }},
		{"window", func(call pkast.BaseCallExpr) ExprNode { return &WindowFuncExpr{BaseCallExpr: call} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			child := &sharedPureExpr{}
			root := tc.new(pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "f"}}}, Args: []ExprNode{child}})
			replacement := tc.new(pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "g"}}}, Args: []ExprNode{child}})
			var entered []Node
			got, ok := root.Accept(operatorVisitor{
				enter: func(n Node) (Node, bool) {
					entered = append(entered, n)
					if n == root {
						return replacement, false
					}
					return n, false
				},
				leave: func(n Node) (Node, bool) { return n, true },
			})
			if !ok || got != replacement || len(entered) < 2 || entered[0] != root || entered[1] != child {
				t.Fatalf("outer identity lost: %T %v %v", got, ok, entered)
			}
			if _, ok := root.Accept(&sharedVisitor{replacement: &BeginStmt{}}); ok {
				t.Fatal("statement replaced a call argument")
			}
			for _, bad := range []Node{&BeginStmt{}, nil} {
				if _, ok := root.Accept(operatorVisitor{
					enter: func(n Node) (Node, bool) { return bad, true },
					leave: func(n Node) (Node, bool) { return n, true },
				}); ok {
					t.Fatal("Enter accepted a different outer role")
				}
			}
			for _, control := range []struct {
				name string
				v    *sharedVisitor
				ok   bool
			}{
				{"skip", &sharedVisitor{skipRoot: true}, true},
				{"stop", &sharedVisitor{stop: true}, false},
			} {
				t.Run(control.name, func(t *testing.T) {
					if _, ok := root.Accept(control.v); ok != control.ok {
						t.Fatalf("Accept=%v, want %v", ok, control.ok)
					}
					if control.v.skipRoot && len(control.v.entered) != 1 {
						t.Fatal("skipped call still visited children")
					}
				})
			}
			if _, ok := root.Accept(&sharedVisitor{replacement: pureLiteral("7")}); !ok {
				t.Fatal("same-role expression replacement failed")
			}
			failing := tc.new(pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "f"}}}, Args: []ExprNode{&sharedPureExpr{restoreErr: true}}})
			var output strings.Builder
			var marker *pureRestoreError
			if err := failing.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &output)); !errors.As(err, &marker) {
				t.Fatalf("argument error chain lost: %v", err)
			}
			for _, empty := range []ExprNode{nil, (*LiteralExpr)(nil)} {
				missing := tc.new(pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "f"}}}, Args: []ExprNode{empty}})
				if err := missing.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &output)); err == nil {
					t.Fatal("nil argument restored without error")
				}
				if _, ok := missing.Accept(&sharedVisitor{}); ok {
					t.Fatal("nil argument visited without failure")
				}
			}
			root = tc.new(pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "f"}}}, Args: []ExprNode{child}})
			if _, ok := root.Accept(&sharedVisitor{replacement: (*LiteralExpr)(nil)}); ok {
				t.Fatal("typed-nil argument replacement was accepted")
			}
		})
	}
}

func TestAggregateTypedNilSeparator(t *testing.T) {
	root := &AggregateFuncExpr{BaseCallExpr: pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "group_concat"}}}}, Separator: (*LiteralExpr)(nil)}
	var output strings.Builder
	if err := root.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &output)); err == nil {
		t.Fatal("typed-nil separator restored without error")
	}
	if _, ok := root.Accept(&sharedVisitor{}); ok {
		t.Fatal("typed-nil separator visited without failure")
	}
}

func TestConvert(t *testing.T) {
	// Test case for CONVERT(expr USING transcoding_name).
	cases := []struct {
		SQL          string
		CharsetName  string
		ErrorMessage string
	}{
		{`SELECT CONVERT("abc" USING "latin1")`, "latin1", ""},
		{`SELECT CONVERT("abc" USING laTiN1)`, "latin1", ""},
		{`SELECT CONVERT("abc" USING "binary")`, "binary", ""},
		{`SELECT CONVERT("abc" USING biNaRy)`, "binary", ""},
		{`SELECT CONVERT(a USING a)`, "", `[parser:1115]Unknown character set: 'a'`},
		{`SELECT CONVERT("abc" USING CONCAT("utf", "8"))`, "", `[parser:1115]Unknown character set: 'CONCAT'`},
	}
	for _, testCase := range cases {
		stmt, err := myparser.New().ParseOneStmt(testCase.SQL, "", "")
		if testCase.ErrorMessage != "" {
			if err == nil || err.Error() != testCase.ErrorMessage {
				t.Fatalf("got error %v, want %v", err, testCase.ErrorMessage)
			}
			continue
		}
		if err != nil {
			t.Fatalf("got error %v, want nil", err)
		}

		st := stmt.(*SelectStmt)
		expr := st.Fields.Items[0].Expr.(*FuncCallExpr)
		if got := expr.Charset.Name; got != testCase.CharsetName {
			t.Fatalf("got %#v, want %#v", got, testCase.CharsetName)
		}
	}
}

func TestChar(t *testing.T) {
	// Test case for CHAR(N USING charset_name)
	cases := []struct {
		SQL          string
		CharsetName  string
		ErrorMessage string
	}{
		{`SELECT CHAR("abc" USING "latin1")`, "latin1", ""},
		{`SELECT CHAR("abc" USING laTiN1)`, "latin1", ""},
		{`SELECT CHAR("abc" USING "binary")`, "binary", ""},
		{`SELECT CHAR("abc" USING binary)`, "binary", ""},
		{`SELECT CHAR(a USING a)`, "", `[parser:1115]Unknown character set: 'a'`},
		{`SELECT CHAR("abc" USING CONCAT("utf", "8"))`, "", `[parser:1115]Unknown character set: 'CONCAT'`},
	}
	for _, testCase := range cases {
		stmt, err := myparser.New().ParseOneStmt(testCase.SQL, "", "")
		if testCase.ErrorMessage != "" {
			if err == nil || err.Error() != testCase.ErrorMessage {
				t.Fatalf("got error %v, want %v", err, testCase.ErrorMessage)
			}
			continue
		}
		if err != nil {
			t.Fatalf("got error %v, want nil", err)
		}

		st := stmt.(*SelectStmt)
		expr := st.Fields.Items[0].Expr.(*FuncCallExpr)
		charsetArg, ok := expr.Args[1].(*LiteralExpr)
		if !ok || charsetArg.Kind != pkast.LiteralString {
			t.Fatalf("charset argument = %T, want syntax string literal", expr.Args[1])
		}
		if got := charsetArg.Value; got != testCase.CharsetName {
			t.Fatalf("got %#v, want %#v", got, testCase.CharsetName)
		}
	}
}

func TestWindowFuncExprRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"RANK() OVER w", "RANK() OVER `w`"},
		{"JSON_ARRAYAGG(a) OVER (PARTITION BY b)", "JSON_ARRAYAGG(`a`) OVER (PARTITION BY `b`)"},
		{"RANK() OVER (PARTITION BY a)", "RANK() OVER (PARTITION BY `a`)"},
		{"MAX(DISTINCT a) OVER (PARTITION BY a)", "MAX(DISTINCT `a`) OVER (PARTITION BY `a`)"},
		{"MAX(DISTINCTROW a) OVER (PARTITION BY a)", "MAX(DISTINCT `a`) OVER (PARTITION BY `a`)"},
		{"MAX(DISTINCT ALL a) OVER (PARTITION BY a)", "MAX(DISTINCT `a`) OVER (PARTITION BY `a`)"},
		{"MAX(ALL a) OVER (PARTITION BY a)", "MAX(`a`) OVER (PARTITION BY `a`)"},
		{"COUNT(*) OVER (PARTITION BY a)", "COUNT(*) OVER (PARTITION BY `a`)"},
		{"FIRST_VALUE(val) IGNORE NULLS OVER (w)", "FIRST_VALUE(`val`) IGNORE NULLS OVER (`w`)"},
		{"FIRST_VALUE(val) RESPECT NULLS OVER w", "FIRST_VALUE(`val`) OVER `w`"},
		{"NTH_VALUE(val, 233) FROM LAST IGNORE NULLS OVER w", "NTH_VALUE(`val`, 233) FROM LAST IGNORE NULLS OVER `w`"},
		{"NTH_VALUE(val, 233) FROM FIRST IGNORE NULLS OVER (w)", "NTH_VALUE(`val`, 233) IGNORE NULLS OVER (`w`)"},
		{"GROUP_CONCAT(a ORDER BY b LIMIT 2) OVER w", "GROUP_CONCAT(`a` ORDER BY `b` LIMIT 2) OVER `w`"},
		{"GROUP_CONCAT(a,b ORDER BY b SEPARATOR '|' LIMIT 2,3) OVER w", "GROUP_CONCAT(`a`, `b` ORDER BY `b` SEPARATOR '|' LIMIT 2,3) OVER `w`"},
		{"COUNT(*) OVER w", "COUNT(*) OVER `w`"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Items[0].Expr
	}
	RunNodeRestoreTest(t, testCases, "select %s from t", extractNodeFunc)
}

// TestCallTailsRestoreAndAccept covers COUNT(*) and window aggregate tails
// independently of parser construction.
func TestCallTailsRestoreAndAccept(t *testing.T) {
	restore := func(expr ExprNode) string {
		var out strings.Builder
		if err := expr.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); err != nil {
			t.Fatalf("Restore: %v", err)
		}
		return out.String()
	}
	star := &AggregateFuncExpr{
		BaseCallExpr: pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "count"}}}},
		Star:         true,
	}
	if got, want := restore(star), "COUNT(*)"; got != want {
		t.Fatalf("COUNT(*) = %q, want %q", got, want)
	}
	if len(star.Args) != 0 {
		t.Fatalf("star argument = %#v, want none", star.Args)
	}
	aggStar := &AggregateFuncExpr{BaseCallExpr: pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "sum"}}}}, Star: true}
	if err := aggStar.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &strings.Builder{})); err == nil {
		t.Fatal("aggregate star restored without error")
	}
}

// TestWindowTailsRestoreAndAccept covers GROUP_CONCAT tails on window calls.
func TestWindowTailsRestoreAndAccept(t *testing.T) {
	tails := &WindowFuncExpr{
		BaseCallExpr: pkast.BaseCallExpr{
			Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "group_concat"}}},
			Args: []ExprNode{&LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralString, Value: "a"}}, &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralString, Value: "b"}}},
		},
		OrderBy:   &OrderByClause{BaseOrderByClause: pkast.BaseOrderByClause{Items: []OrderItemNode{&OrderItem{BaseOrderItem: pkast.BaseOrderItem{Expr: &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralString, Value: "b"}}}}}}},
		Separator: &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralString, Value: "|"}},
		Limit:     &Limit{BaseLimit: pkast.BaseLimit{Count: &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "2"}}}},
	}
	// Hand-built node with valid tails restores cleanly; ORDER BY / SEPARATOR
	// / LIMIT appear after the window function argument list and before OVER.
	var out strings.Builder
	if err := tails.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); err != nil {
		t.Fatalf("restore valid tails: %v", err)
	}
	if got, want := out.String(), "GROUP_CONCAT('a', 'b' ORDER BY 'b' SEPARATOR '|' LIMIT 2) OVER ()"; got != want {
		t.Fatalf("tails restore = %q, want %q", got, want)
	}
	// Same-role tail replacement through Accept keeps the structure valid.
	_, ok := tails.Accept(operatorVisitor{
		enter: func(n Node) (Node, bool) {
			switch n {
			case tails.Separator:
				return &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralString, Value: ";"}}, false
			case tails.Limit:
				return &Limit{BaseLimit: pkast.BaseLimit{Count: &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "5"}}}}, false
			case tails.OrderBy:
				return &OrderByClause{BaseOrderByClause: pkast.BaseOrderByClause{Items: []OrderItemNode{&OrderItem{BaseOrderItem: pkast.BaseOrderItem{Expr: &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralString, Value: "z"}}}}}}}, false
			}
			return n, false
		},
		leave: func(n Node) (Node, bool) { return n, true },
	})
	if !ok {
		t.Fatal("same-role tail replacement was rejected")
	}
	// A same-role visitor that replaces the outer call with a statement must fail.
	if _, ok := tails.Accept(operatorVisitor{
		enter: func(n Node) (Node, bool) {
			if n == tails {
				return &BeginStmt{}, true
			}
			return n, false
		},
		leave: func(n Node) (Node, bool) { return n, true },
	}); ok {
		t.Fatal("statement replaced a window call")
	}
}

package ast_test

import (
	"testing"

	"github.com/mia0x75/myparser"
	. "github.com/mia0x75/myparser/ast"
	driver "github.com/mia0x75/myparser/driver"
)

func TestFunctionsVisitorCover(t *testing.T) {
	valueExpr := NewValueExpr(42)
	stmts := []Node{
		&AggregateFuncExpr{Args: []ExprNode{valueExpr}},
		&FuncCallExpr{Args: []ExprNode{valueExpr}},
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
		return node.(*SelectStmt).Fields.Fields[0].Expr
	}
	RunNodeRestoreTest(t, testCases, "select %s", extractNodeFunc)
}

func TestFuncCastExprRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"CONVERT('Müller' USING UtF8Mb4)", "CONVERT('Müller' USING UTF8MB4)"},
		{"CONVERT('Müller', CHAR(32) CHARACTER SET UtF8)", "CONVERT('Müller', CHAR(32) CHARACTER SET utf8)"},
		{"CAST('test' AS CHAR CHARACTER SET UtF8)", "CAST('test' AS CHAR CHARACTER SET utf8)"},
		{"BINARY 'New York'", "BINARY 'New York'"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Fields[0].Expr
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
		{"COUNT(*)", "COUNT(1)"},
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
		return node.(*SelectStmt).Fields.Fields[0].Expr
	}
	RunNodeRestoreTest(t, testCases, "select %s", extractNodeFunc)
}

func TestAggregateFuncExprTailVisitor(t *testing.T) {
	stmt, err := myparser.New().ParseOneStmt("SELECT GROUP_CONCAT(a,b ORDER BY c SEPARATOR '|' LIMIT 2,3)", "", "")
	if err != nil {
		t.Fatal(err)
	}
	agg := stmt.(*SelectStmt).Fields.Fields[0].Expr.(*AggregateFuncExpr)
	if len(agg.Args) != 2 || agg.Separator == nil || agg.OrderBy == nil || agg.Limit == nil {
		t.Fatalf("aggregate tails = %#v", agg)
	}
	checker := &aggregateTailVisitor{seen: make(map[Node]bool)}
	if _, ok := agg.Accept(checker); !ok {
		t.Fatal("visitor aborted")
	}
	for _, node := range []Node{agg.Separator, agg.OrderBy, agg.OrderBy.Items[0].Expr, agg.Limit, agg.Limit.Count, agg.Limit.Offset} {
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
		expr := st.Fields.Fields[0].Expr.(*FuncCallExpr)
		charsetArg := expr.Args[1].(*driver.ValueExpr)
		if got := charsetArg.GetString(); got != testCase.CharsetName {
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
		expr := st.Fields.Fields[0].Expr.(*FuncCallExpr)
		charsetArg := expr.Args[1].(*driver.ValueExpr)
		if got := charsetArg.GetString(); got != testCase.CharsetName {
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
		{"FIRST_VALUE(val) IGNORE NULLS OVER (w)", "FIRST_VALUE(`val`) IGNORE NULLS OVER (`w`)"},
		{"FIRST_VALUE(val) RESPECT NULLS OVER w", "FIRST_VALUE(`val`) OVER `w`"},
		{"NTH_VALUE(val, 233) FROM LAST IGNORE NULLS OVER w", "NTH_VALUE(`val`, 233) FROM LAST IGNORE NULLS OVER `w`"},
		{"NTH_VALUE(val, 233) FROM FIRST IGNORE NULLS OVER (w)", "NTH_VALUE(`val`, 233) IGNORE NULLS OVER (`w`)"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Fields[0].Expr
	}
	RunNodeRestoreTest(t, testCases, "select %s from t", extractNodeFunc)
}

package parser_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/parser"
)

func TestJoinChainAndParenTailRoundTrip(t *testing.T) {
	sources := []string{
		// Nested ON join chains.
		"EXPLAIN EXTENDED\nSELECT /*+ JOIN_ORDER(alias11, alias8) */ 1\nFROM t4 AS alias4\n  LEFT JOIN t5 AS alias5 JOIN t6 AS alias6 ON alias5.f2 = alias6.f1\n  LEFT JOIN t7 AS alias7 JOIN t2 AS alias8 ON alias7.f1 = alias8.f1\n    ON alias5.f1 = alias8.f2 ON alias4.f2 = alias6.f1\n  JOIN t10 AS alias10 JOIN t11 AS alias11 ON alias10.f1 = alias11.f1\n    ON alias4.f2 = alias11.f2",
		"EXPLAIN EXTENDED\nSELECT /*+ JOIN_ORDER(alias11, alias10, alias8, alias7) */ 1\nFROM t4 AS alias4\n  LEFT JOIN t5 AS alias5 JOIN t6 AS alias6 ON alias5.f2 = alias6.f1\n  LEFT JOIN t7 AS alias7 JOIN t2 AS alias8 ON alias7.f1 = alias8.f1\n    ON alias5.f1 = alias8.f2 ON alias4.f2 = alias6.f1\n  JOIN t10 AS alias10 JOIN t11 AS alias11 ON alias10.f1 = alias11.f1\n    ON alias4.f2 = alias11.f2",
		"EXPLAIN EXTENDED SELECT /*+ JOIN_ORDER(alias8, alias6) */ 1\nFROM t1 AS alias1\n  LEFT JOIN t7 alias7\n    JOIN t2 AS alias2\n      LEFT JOIN t3 AS alias3 JOIN t4 AS alias4 ON alias4.f2 = alias3.f1\n      ON alias4.f1 =  alias2.f1\n    ON alias2.f2 = alias7.f1\n    JOIN t10 AS alias5\n      LEFT JOIN t6 AS alias6 JOIN t2 AS alias8 ON alias6.f1 = alias8.f2\n      ON alias6.f1 =  alias5.f1\n    ON alias5.f3 = alias7.f1\n  ON alias1.f2 = alias7.f1",
		"EXPLAIN\nSELECT /*+ JOIN_ORDER(alias11, alias8) */ 1\nFROM t4 AS alias4\n  LEFT JOIN t5 AS alias5 JOIN t6 AS alias6 ON alias5.f2 = alias6.f1\n  LEFT JOIN t7 AS alias7 JOIN t2 AS alias8 ON alias7.f1 = alias8.f1\n    ON alias5.f1 = alias8.f2 ON alias4.f2 = alias6.f1\n  JOIN t10 AS alias10 JOIN t11 AS alias11 ON alias10.f1 = alias11.f1\n    ON alias4.f2 = alias11.f2",
		"EXPLAIN\nSELECT /*+ JOIN_ORDER(alias11, alias10, alias8, alias7) */ 1\nFROM t4 AS alias4\n  LEFT JOIN t5 AS alias5 JOIN t6 AS alias6 ON alias5.f2 = alias6.f1\n  LEFT JOIN t7 AS alias7 JOIN t2 AS alias8 ON alias7.f1 = alias8.f1\n    ON alias5.f1 = alias8.f2 ON alias4.f2 = alias6.f1\n  JOIN t10 AS alias10 JOIN t11 AS alias11 ON alias10.f1 = alias11.f1\n    ON alias4.f2 = alias11.f2",
		"EXPLAIN SELECT /*+ JOIN_ORDER(alias8, alias6) */ 1\nFROM t1 AS alias1\n  LEFT JOIN t7 alias7\n    JOIN t2 AS alias2\n      LEFT JOIN t3 AS alias3 JOIN t4 AS alias4 ON alias4.f2 = alias3.f1\n      ON alias4.f1 =  alias2.f1\n    ON alias2.f2 = alias7.f1\n    JOIN t10 AS alias5\n      LEFT JOIN t6 AS alias6 JOIN t2 AS alias8 ON alias6.f1 = alias8.f2\n      ON alias6.f1 =  alias5.f1\n    ON alias5.f3 = alias7.f1\n  ON alias1.f2 = alias7.f1",
		"EXPLAIN SELECT * FROM t1 JOIN t2 JOIN t3 ON t2.b = t3.c ON t1.a = t2.b",
		"EXPLAIN SELECT * FROM t1 JOIN t2 JOIN t3 JOIN t4 ON t3.c = t4.d ON t2.b = t3.c",
		"EXPLAIN SELECT * FROM t1 JOIN t2 JOIN t3 JOIN t4 ON t3.c = t4.d ON t1.a = t2.b",
		// Parenthesized and UNION query tails.
		"(select * from t1) for update",
		"(select * from t1) union (select * from t1) for update",
		"SELECT 1 UNION SELECT 1 FROM DUAL INTO @var FOR UPDATE",
		"(SELECT 1 UNION SELECT 1 FROM DUAL INTO @var FOR UPDATE)",
		"SELECT 1 UNION SELECT 1 UNION SELECT 1 FROM DUAL INTO @var FOR UPDATE",
		"(SELECT 1 UNION SELECT 1 UNION SELECT 1 FROM DUAL INTO @var FOR UPDATE)",
		"(SELECT 1) LIMIT 1 INTO @var",
		"(SELECT 2 AS c) ORDER BY c INTO @var",
		"(SELECT 3 AS c) ORDER BY c LIMIT 1 INTO @var",
		"(SELECT 4) INTO @var",
	}
	parser := parser.New()
	for _, src := range sources {
		t.Run(src, func(t *testing.T) {
			stmt, err := parser.ParseOneStmt(src, "", "")
			if err != nil {
				t.Fatalf("parse %q: %v", src, err)
			}
			var sb strings.Builder
			if err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
				t.Fatalf("restore %q: %v", src, err)
			}
			restored := sb.String()
			again, err := parser.ParseOneStmt(restored, "", "")
			if err != nil {
				t.Fatalf("reparse %q: %v", restored, err)
			}
			cleanNodeText(stmt)
			cleanNodeText(again)
			if !reflect.DeepEqual(stmt, again) {
				t.Fatalf("round trip %q -> %q changed the tree", src, restored)
			}
		})
	}
}

func TestQueryTailPreservation(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"((SELECT 1) FOR UPDATE) LIMIT 1", "((SELECT 1) FOR UPDATE) LIMIT 1"},
		{"((SELECT * FROM t) FOR UPDATE OF t NOWAIT) LIMIT 1", "((SELECT * FROM `t`) FOR UPDATE OF `t` NOWAIT) LIMIT 1"},
		{"(SELECT 1) ORDER BY 1 LIMIT 2 FOR UPDATE INTO @a", "(SELECT 1) ORDER BY 1 LIMIT 2 FOR UPDATE INTO @`a`"},
		{"(SELECT 1) ORDER BY 1 LIMIT 2 INTO @a FOR UPDATE", "(SELECT 1) ORDER BY 1 LIMIT 2 FOR UPDATE INTO @`a`"},
		{"(SELECT 1) UNION (SELECT 2) ORDER BY 1 LIMIT 1 FOR UPDATE INTO @a", "(SELECT 1) UNION (SELECT 2) ORDER BY 1 LIMIT 1 FOR UPDATE INTO @`a`"},
		{"((SELECT 1) INTO @a)", "(SELECT 1) INTO @`a`"},
		{"(SELECT 1 INTO @a) LIMIT 1", "(SELECT 1 INTO @`a`) LIMIT 1"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			runRestoreTest(t, tc.src, tc.restored, true)
		})
	}
	assertParseErrors(t, []string{
		"((SELECT 1) INTO @a) LIMIT 1",
		"((SELECT 1) INTO @a) INTO @b",
		"SELECT ((SELECT 1) INTO @a)",
		"SELECT * FROM ((SELECT 1) INTO @a) d",
		"WITH d AS ((SELECT 1) INTO @a) SELECT * FROM d",
		"SELECT 1 UNION ((SELECT 1) INTO @a)",
		"((SELECT 1) INTO @a) UNION SELECT 1",
	})
}

func TestQueryTailVisitors(t *testing.T) {
	cases := []struct {
		src     string
		isChild func(ast.Node) bool
	}{
		{"(SELECT 1) INTO @a", func(n ast.Node) bool { _, ok := n.(*ast.VariableTarget); return ok }},
		{"((SELECT 1 FROM t) FOR UPDATE OF t NOWAIT) LIMIT 1", func(n ast.Node) bool { _, ok := n.(*ast.TableName); return ok }},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			stmt := mustParseOne(t, tc.src)
			assertVisitorControls(t, stmt, tc.isChild)
			visited := 0
			stmt.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
				if tc.isChild(n) {
					visited++
				}
				return n, false
			}})
			want := 1
			if strings.Contains(tc.src, " OF ") {
				want = 2
			}
			if visited != want {
				t.Fatalf("visited %d tail children, want %d", visited, want)
			}
		})
	}
}

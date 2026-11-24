package parser_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/myparser/parser"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

func TestQueryMiscAndCTERoundTrip(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"explain extended select * from t", "EXPLAIN EXTENDED SELECT * FROM `t`"},
		{"explain partitions select * from t", "EXPLAIN PARTITIONS SELECT * FROM `t`"},
		{"analyze select * from t", "ANALYZE SELECT * FROM `t`"},
		{"analyze format=json select * from t", "ANALYZE FORMAT = 'json' SELECT * FROM `t`"},
		{"select sql_calc_found_rows distinct high_priority 1", "SELECT HIGH_PRIORITY SQL_CALC_FOUND_ROWS DISTINCT 1"},
		{"select distinct distinct 1", "SELECT DISTINCT 1"},
		{"select all all 1", "SELECT 1"},
		{"select 1 where 1 group by 1 having 1", "SELECT 1 FROM DUAL WHERE 1 GROUP BY 1 HAVING 1"},
		{"select 1 from dual where 1 group by 1 having 1", "SELECT 1 FROM DUAL WHERE 1 GROUP BY 1 HAVING 1"},
		{"select _latin2'a', _koi8r 0xff, _binary x'41'", "SELECT _LATIN2'a',_KOI8R x'ff',_BINARY x'41'"},
		{"update t set a=default", "UPDATE `t` SET `a`=DEFAULT"},
		{"insert into t set a=default, b=1", "INSERT INTO `t` SET `a`=DEFAULT,`b`=1"},
		{"explain with c as (select 1) select * from c", "EXPLAIN FORMAT = 'row' WITH `c` AS (SELECT 1) SELECT * FROM `c`"},
		{"with c as (with d as (select 1) select * from d) select * from c", "WITH `c` AS (WITH `d` AS (SELECT 1) SELECT * FROM `d`) SELECT * FROM `c`"},
		{"select (with c as (select 1 as a) select a from c)", "SELECT (WITH `c` AS (SELECT 1 AS `a`) SELECT `a` FROM `c`)"},
		{"insert into t(a) with c as (select 1 as a) select a from c", "INSERT INTO `t` (`a`) WITH `c` AS (SELECT 1 AS `a`) SELECT `a` FROM `c`"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			runRestoreTest(t, tc.src, tc.restored, true)
		})
	}
	assertParseErrors(t, []string{
		"explain extended select", "analyze format=json",
	})
}

func TestQueryCTERoundTrip(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"with c as (select 1 as a) update t set a=(select a from c)", "WITH `c` AS (SELECT 1 AS `a`) UPDATE `t` SET `a`=(SELECT `a` FROM `c`)"},
		{"with c as (select 1 as a) delete from t where a in (select a from c)", "WITH `c` AS (SELECT 1 AS `a`) DELETE FROM `t` WHERE `a` IN (SELECT `a` FROM `c`)"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			runRestoreTest(t, tc.src, tc.restored, true)
		})
	}
	assertParseErrors(t, []string{
		"with c as (select 1) update", "with c as (select 1) delete",
	})
}

func TestManagementSessionAndPreparedRoundTrip(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"show create procedure db.p", "SHOW CREATE PROCEDURE `db`.`p`"},
		{"show create function f", "SHOW CREATE FUNCTION `f`"},
		{"show create trigger tr", "SHOW CREATE TRIGGER `tr`"},
		{"show create event ev", "SHOW CREATE EVENT `ev`"},
		{"show local variables like 'SQL_MODE'", "SHOW SESSION VARIABLES LIKE 'SQL_MODE'"},
		{"show local status", "SHOW SESSION STATUS"},
		{"set names default", "SET NAMES DEFAULT"},
		{"set character set default", "SET CHARACTER SET DEFAULT"},
		{"set @@global.max_allowed_packet:=@saved", "SET @@GLOBAL.`max_allowed_packet`=@`saved`"},
		{"set local sql_mode:=default", "SET @@SESSION.`sql_mode`=DEFAULT"},
		{"execute s using 1, 'x', @a", "EXECUTE `s` USING 1,'x',@`a`"},
		{"execute immediate concat('select ', ?) using default, 1", "EXECUTE IMMEDIATE CONCAT('select ', ?) USING DEFAULT,1"},
		{"analyze table t1,t2 persistent for all", "ANALYZE TABLE `t1`,`t2` PERSISTENT FOR ALL"},
		{"analyze local table t1 persistent for columns(a,b) indexes()", "ANALYZE NO_WRITE_TO_BINLOG TABLE `t1` PERSISTENT FOR COLUMNS (`a`,`b`) INDEXES ()"},
		{"set statement sql_log_bin=0 for call db.p()", "SET STATEMENT `sql_log_bin`=0 FOR CALL `db`.`p`()"},
		{"set statement sql_mode='NO_ENGINE_SUBSTITUTION', max_statement_time=2 for select * from t", "SET STATEMENT `sql_mode`='NO_ENGINE_SUBSTITUTION', `max_statement_time`=2 FOR SELECT * FROM `t`"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			runRestoreTest(t, tc.src, tc.restored, true)
		})
	}
	assertParseErrors(t, []string{
		"show create procedure",
		"set statement for select 1", "set statement x=1 for",
		"set statement x=1, for select 1", "set statement @@global.x=1 for select 1",
		"execute immediate 'select 1' using", "execute immediate 'select 1' using 1,",
		"analyze table t persistent for", "analyze table t persistent for columns(a)",
	})
}

func TestManagementShowReplicaRoundTrip(t *testing.T) {
	for _, src := range []string{"show replica status", "show replicas", "show binary log status", "show binary logs"} {
		t.Run(src, func(t *testing.T) {
			runRestoreTest(t, src, strings.ToUpper(src), true)
		})
	}
	assertParseErrors(t, []string{"show replica", "show replicas status", "show binary log", "show binary status"})
}

func TestQueryCTEViewAndRoutineRoundTrip(t *testing.T) {
	for _, src := range []string{
		"create view v as with c as (select 1 as a) select a from c",
		"alter view v as with c as (select 1 as a) select a from c",
		"create view v as select 1 union select 2",
		"create table t as with c as (select 1 as a) select a from c",
		"create procedure p() begin call db.p2(1); set statement x=1 for select 1; end",
		"create procedure p() with c as (select 1 as a) update t set a=(select a from c)",
	} {
		t.Run(src, func(t *testing.T) {
			stmt := mustParseOne(t, src)
			restored := restoreStmt(t, stmt)
			again := mustParseOne(t, restored)
			if got := restoreStmt(t, again); got != restored {
				t.Fatalf("unstable restore: %q -> %q", restored, got)
			}
			if _, ok := stmt.Accept(syntaxVisitor{}); !ok {
				t.Fatal("visitor aborted")
			}
		})
	}
}

func TestQueryExpressionTypeRoundTrip(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"select 'andre%' like 'andreй%' escape 'й'", "SELECT 'andre%' LIKE 'andreй%' ESCAPE 'й'"},
		{"select 'abc' like 'a%' escape 0x5c", "SELECT 'abc' LIKE 'a%' ESCAPE x'5c'"},
		{"select 'abc' like 'a%' escape concat('x')", "SELECT 'abc' LIKE 'a%' ESCAPE CONCAT('x')"},
		{"select percentile_cont(0.5) within group(order by score) over(partition by name) from t1", "SELECT PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY `score`) OVER (PARTITION BY `name`) FROM `t1`"},
		{"select foo(a) over(partition by b order by c) from t", "SELECT FOO(`a`) OVER (PARTITION BY `b` ORDER BY `c`) FROM `t`"},
		{"select var_pop(a) over(partition by b), var_samp(a) over() from t", "SELECT VAR_POP(`a`) OVER (PARTITION BY `b`),VAR_SAMP(`a`) OVER () FROM `t`"},
		{"select cast('1' as float), cast('1' as double), cast('1' as double precision), cast('1' as year)", "SELECT CAST('1' AS FLOAT),CAST('1' AS DOUBLE),CAST('1' AS DOUBLE),CAST('1' AS YEAR)"},
		{"select 'a' collate utf8mb4_uca1400_ai_ci", "SELECT 'a' COLLATE utf8mb4_uca1400_ai_ci"},
		{"select within from t", "SELECT `within` FROM `t`"},
		{"(select a, b as c from t1) order by c+1", "(SELECT `a`,`b` AS `c` FROM `t1`) ORDER BY `c`+1"},
		{"((select 1))", "SELECT 1"},
		{"(((select 1))) order by 1 limit 1", "(SELECT 1) ORDER BY 1 LIMIT 1"},
		{"select ((select 1))", "SELECT (SELECT 1)"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			runRestoreTest(t, tc.src, tc.restored, true)
		})
	}
	assertParseErrors(t, []string{
		"select 1 like 1 escape",
		"select cast(1 as year(4))",
		"select cast(1 as float unsigned)",
		"((select 1)",
		"(select 1))",
	})
}

func TestQueryWindowFunctionRoundTrip(t *testing.T) {
	for _, src := range []string{
		"select percentile_cont(0.5) within group(order by a,b) over()",
		"select percentile_cont(0.5) within group(order by a)",
		"select percentile_cont(0.5) within group() over()",
		"select foo(a) over",
	} {
		p := parser.New()
		p.EnableWindowFunc(true)
		if _, _, err := p.Parse(src, "", ""); err == nil {
			t.Errorf("expected parse error for %q", src)
		}
	}
}

func TestQueryWeightStringAndTimestampsRoundTrip(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"select weight_string('a')", "SELECT WEIGHT_STRING('a')"},
		{"select weight_string('a' as char(0x3) level 0)", "SELECT WEIGHT_STRING('a' AS CHAR(3) LEVEL 0)"},
		{"select weight_string('a' as char(x'03'))", "SELECT WEIGHT_STRING('a' AS CHAR(3))"},
		{"select weight_string('a' as binary(3))", "SELECT WEIGHT_STRING('a' AS BINARY(3))"},
		{"select weight_string('a' level 0x1 desc reverse,2 asc,9 reverse)", "SELECT WEIGHT_STRING('a' LEVEL 1 DESC REVERSE,2 ASC,9 REVERSE)"},
		{"select weight_string('a' level 0-9), weight_string('a' level 2-1)", "SELECT WEIGHT_STRING('a' LEVEL 0-9),WEIGHT_STRING('a' LEVEL 2-1)"},
		{"select weight_string('a',0x1,2.5,0)", "SELECT WEIGHT_STRING('a',1,2.5,0)"},
		{"select weight_string('a',x'01',2,0)", "SELECT WEIGHT_STRING('a',1,2,0)"},
		{"select weight_string('a',0xffffffffffffffff,2,0)", "SELECT WEIGHT_STRING('a',18446744073709551615,2,0)"},
		{"select weight_string('a',+1,2,0)", "SELECT WEIGHT_STRING('a',+1,2,0)"},
		{"select timestampadd(sql_tsi_second,1,'2000-01-01'), timestampdiff(sql_tsi_year,'2000-01-01','2001-01-01')", "SELECT TIMESTAMPADD(SQL_TSI_SECOND, 1, '2000-01-01'),TIMESTAMPDIFF(SQL_TSI_YEAR, '2000-01-01', '2001-01-01')"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			runRestoreTest(t, tc.src, tc.restored, true)
		})
	}
}

func TestQueryIntoVariablesRoundTrip(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"select 1,2,3,4 into r.a, `r.a`, `@a`, @a", "SELECT 1,2,3,4 INTO `r`.`a`, `r.a`, `@a`, @`a`"},
		{"select 1,2 into @'a.b', @`@a` from t", "SELECT 1,2 INTO @`a.b`, @`@a` FROM `t`"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			runRestoreTest(t, tc.src, tc.restored, true)
		})
	}
}

func TestQueryValuesSetRoundTrip(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"values row(1),row(2)", "VALUES ROW(1), ROW(2)"},
		{"values (1),(2)", "VALUES (1), (2)"},
		{"values row(1) union all values row(2) union select 3", "VALUES ROW(1) UNION ALL VALUES ROW(2) UNION SELECT 3"},
		{"values (1) union all values (2) union select 3", "VALUES (1) UNION ALL VALUES (2) UNION SELECT 3"},
		{"select 1 union values (2)", "SELECT 1 UNION VALUES (2)"},
		{"values row(1) except all values row(2) intersect values row(3)", "VALUES ROW(1) EXCEPT ALL VALUES ROW(2) INTERSECT VALUES ROW(3)"},
		{"values row(1) intersect values row(2) union values row(3)", "VALUES ROW(1) INTERSECT VALUES ROW(2) UNION VALUES ROW(3)"},
		{"select 1 union values row(2) intersect values row(3)", "SELECT 1 UNION VALUES ROW(2) INTERSECT VALUES ROW(3)"},
		{"values (1) union select 2 order by 1 limit 1", "VALUES (1) UNION SELECT 2 ORDER BY 1 LIMIT 1"},
		{"with recursive c(a) as (values (identifier) union all values (2) union select a+1 from c where a<3) select * from c", "WITH RECURSIVE `c` (`a`) AS (VALUES (`identifier`) UNION ALL VALUES (2) UNION SELECT `a`+1 FROM `c` WHERE `a`<3) SELECT * FROM `c`"},
		{"select * from (values row(1),row(2)) d(x)", "SELECT * FROM (VALUES ROW(1), ROW(2)) AS `d` (`x`)"},
		{"select * from (values (1),(2)) d(x)", "SELECT * FROM (VALUES (1), (2)) AS `d` (`x`)"},
		{"select * from (select 1 as a) d(x)", "SELECT * FROM (SELECT 1 AS `a`) AS `d` (`x`)"},
		{"select * from (with c as (select 1 as a) select a from c) d(x)", "SELECT * FROM (WITH `c` AS (SELECT 1 AS `a`) SELECT `a` FROM `c`) AS `d` (`x`)"},
		{"select * from t join (select * from u join v on u.a=v.a) d on t.a=d.a", "SELECT * FROM `t` JOIN (SELECT * FROM `u` JOIN `v` ON `u`.`a`=`v`.`a`) AS `d` ON `t`.`a`=`d`.`a`"},
		{"with recursive c(a) as (select 1 union select a+1 from c where a<3) cycle a restrict select * from c", "WITH RECURSIVE `c` (`a`) AS (SELECT 1 UNION SELECT `a`+1 FROM `c` WHERE `a`<3) CYCLE `a` RESTRICT SELECT * FROM `c`"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			runRestoreTest(t, tc.src, tc.restored, true)
		})
	}
}

func TestQueryRoutineAndViewRoundTrip(t *testing.T) {
	for _, src := range []string{
		"create procedure p() begin declare r row(a int,b varchar(8)) default (1,'x'); select 2 into r.a; set r.a=3; end",
		"create procedure p() begin declare r row type of db.t; select 1 into r.a; end",
		"create procedure p() begin declare a type of db.t.a default 1; end",
		"create procedure p() begin declare a type of t.a; end",
		"create procedure p() begin declare c cursor for select 1; begin declare r row type of c; fetch next from c into r; end; end",
		"create procedure p() begin declare `a.b` int; declare `@a` int; declare c cursor for select 1,2; fetch from c into `a.b`,`@a`; end",
		"create view v as select a from t with check option",
		"create view v as select a from t with cascaded check option",
		"alter view v as select a from t with local check option",
	} {
		t.Run(src, func(t *testing.T) {
			stmt := mustParseOne(t, src)
			restored := restoreStmt(t, stmt)
			again := mustParseOne(t, restored)
			if got := restoreStmt(t, again); got != restored {
				t.Fatalf("unstable restore: %q -> %q", restored, got)
			}
			if _, ok := stmt.Accept(syntaxVisitor{}); !ok {
				t.Fatal("visitor aborted")
			}
		})
	}
}

func TestQueryMiscAndQuerySyntaxErrors(t *testing.T) {
	// These boundaries were checked on the pinned MySQL 8.4.13/MariaDB 12.3.3
	// servers. Runtime definition errors (e.g. SEQUENCE 4085 or DEFAULT 1901)
	// are deliberately covered by the positive tests instead.
	assertParseErrors(t, []string{
		"select weight_string('a' as char(0))",
		"select weight_string('a' as char(-1))",
		"select weight_string('a' as char(+3))",
		"select weight_string('a' as char(1.5))",
		"select weight_string('a' as char)",
		"select weight_string('a' as varchar(3))",
		"select weight_string('a' as binary(3) level 1)",
		"select weight_string('a' level +1)",
		"select weight_string('a' level 1.5)",
		"select weight_string('a' level x'02')",
		"select weight_string('a' level 1 reverse desc)",
		"select weight_string('a' level 1-2 desc)",
		"select weight_string('a' level 1-2,3)",
		"select weight_string('a',-1,2,0)",
		"select weight_string('a',+0x1,2,0)",
		"values row(1),(2)",
		"values (1),row(2)",
		"create procedure p() begin declare x int; declare c cursor for select 1; fetch c into @x; end",
		"create procedure p() begin declare r row(a int); declare c cursor for select 1; fetch c into r.a; end",
		"create procedure p() begin declare r row(a int default 1); end",
		"select * from t as d(x)",
		"with c as (select 1 as a) cycle a restrict select * from c",
		"select timestampadd(sql_tsi_frac_second,1,'2000-01-01')",
		"select timestampadd(sql_tsi_microsecond,1,'2000-01-01')",
		"select timestampadd(sql_tsi_millisecond,1,'2000-01-01')",
		"select timestampadd(second_microsecond,1,'2000-01-01')",
		"select 1 order by 1 union select 2",
		"select 1 limit 1 union select 2",
		"select 1 for update union select 2",
		"(select 1 from t into @v) union select 2",
		"select 1 union (select 2 from dual into @v)",
		"select (select 1 from t into @v)",
		"select * from (select 1 from t into @v) d",
		"with c as (select 1 from t into @v) select * from c",
		"select now(-1)",
		"select now(+1)",
		"select now(1.0)",
		"select now(a)",
		"select json_objectagg(a)",
		"select json_objectagg(a,b,c)",
		"create table t(a double(2147483648,0))",
		"create table t(a double(1,2147483648))",
		"select 'unterminated",
		"select 1 like 1 escape",
	})
}

func TestQueryQueryTailBoundaryControls(t *testing.T) {
	// These retain legal query tails, runtime errors, and one-server dialect
	// differences adjacent to the rejected boundaries above.
	for _, src := range []string{
		"select 1 into @v",
		"select 1 into @v from t",
		"select 1 from t into @v",
		"(select 1 into @v)",
		"select 1 union select 2 into @v",
		"select 1 union (select 2 into @v)",
		"select 1 union (select 2 into @v from t)",
		"select 1 into @v union select 2",
		"(select 1 into @v) union select 2",
		"select (select 1 into @v)",
		"select * from (select 1 into @v from t) d",
		"with c as (select 1 into @v) select * from c",
		"(select 1 order by 1 limit 1) union select 2",
		"select 1 union (select 2 order by 1 limit 1)",
		"select 1 union select 2 for update",
		"(select 1 for update) union select 2",
		"select now()",
		"select now(0)",
		"select now(7)",
		"select json_objectagg(a,b) from t",
		"select json_arrayagg(a) from t",
	} {
		t.Run(src, func(t *testing.T) {
			stmt := mustParseOne(t, src)
			restored := restoreStmt(t, stmt)
			again := mustParseOne(t, restored)
			if got := restoreStmt(t, again); got != restored {
				t.Fatalf("unstable restore: %q -> %q", restored, got)
			}
			if _, ok := stmt.Accept(syntaxVisitor{}); !ok {
				t.Fatal("visitor aborted")
			}
		})
	}

	// Both servers reject duplicate INTO, including MySQL's dedicated error.
	if _, err := parser.New().ParseOneStmt("select 1 into @a from t into @b", "", ""); err == nil {
		t.Fatal("duplicate INTO was accepted")
	}
	before := mustParseOne(t, "select 1 into @v from t").(*ast.SelectStmt)
	trailing := mustParseOne(t, "select 1 from t into @v").(*ast.SelectStmt)
	if before.Into == nil || !before.Into.BeforeFrom || trailing.Into == nil || trailing.Into.BeforeFrom {
		t.Fatalf("INTO position was lost: before=%#v trailing=%#v", before.Into, trailing.Into)
	}
}

func TestQueryUnterminatedStringSQLModes(t *testing.T) {
	for _, tc := range []struct {
		mode mysql.SQLMode
		sql  string
	}{
		{sql: "select 'unterminated"},
		{mode: mysql.ModeNoBackslashEscapes, sql: "select 'unterminated"},
		{mode: mysql.ModeANSIQuotes, sql: `create procedure p() update t set c=''"`},
	} {
		p := parser.New()
		p.SetSQLMode(tc.mode)
		if _, err := p.ParseOneStmt(tc.sql, "", ""); err == nil {
			t.Fatalf("expected unterminated string error for %q in mode %d", tc.sql, tc.mode)
		}
	}
}

func TestQueryStatementVisitors(t *testing.T) {
	for _, src := range []string{
		"call p(1)",
		"set statement x=1 for call p(2)",
		"with c as (select 1 as a) update t set a=2",
		"with c as (select 1 as a) delete from t where a=2",
	} {
		t.Run(src, func(t *testing.T) {
			stmt := mustParseOne(t, src)
			values := 0
			_, ok := stmt.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
				switch n := n.(type) {
				case *ast.LiteralExpr:
					values++
					return replaceLiteral(n, "9"), false
				}
				return n, false
			}})
			if !ok || values == 0 {
				t.Fatalf("visitor did not reach expressions: ok=%v values=%d", ok, values)
			}
			if got := restoreStmt(t, stmt); !strings.Contains(got, "9") {
				t.Fatalf("visitor replacement lost: %s", got)
			}
			_, ok = stmt.Accept(syntaxVisitor{leave: func(n ast.Node) (ast.Node, bool) {
				return n, !isValueOrLiteral(n)
			}})
			if ok {
				t.Fatal("visitor abort was not propagated")
			}
			entered := 0
			_, ok = stmt.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
				entered++
				return n, true
			}})
			if !ok || entered != 1 {
				t.Fatalf("skipChildren: ok=%v entered=%d", ok, entered)
			}
		})
	}
}

func TestQueryLikeAndWindowExpressionASTAndVisitors(t *testing.T) {
	p := parser.New()
	p.EnableWindowFunc(true)
	stmt, err := p.ParseOneStmt("select 'a' like 'b' escape concat('x'), percentile_cont(0.5) within group(order by score+2) over(partition by grp+3) from t", "", "")
	if err != nil {
		t.Fatal(err)
	}
	sel := stmt.(*ast.SelectStmt)
	like := sel.Fields.Items[0].Expr.(*ast.PatternLikeExpr)
	if like.EscapeExpr == nil {
		t.Fatal("expression ESCAPE was not preserved in the AST")
	}
	window := sel.Fields.Items[1].Expr.(*ast.WindowFuncExpr)
	if window.WithinGroup == nil || len(window.WithinGroup.Items) != 1 {
		t.Fatalf("WITHIN GROUP was not preserved: %#v", window.WithinGroup)
	}

	values := 0
	_, ok := stmt.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch n.(type) {
		case *ast.LiteralExpr:
			values++
			return &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "9"}}, false
		}
		return n, false
	}})
	if !ok || values != 6 {
		t.Fatalf("visitor expression count: ok=%v values=%d, want 6", ok, values)
	}
	restored := restoreStmt(t, stmt)
	for _, want := range []string{"ESCAPE CONCAT(9)", "PERCENTILE_CONT(9)", "ORDER BY `score`+9", "PARTITION BY `grp`+9"} {
		if !strings.Contains(restored, want) {
			t.Fatalf("visitor replacement %q missing from %s", want, restored)
		}
	}

	_, ok = stmt.Accept(syntaxVisitor{leave: func(n ast.Node) (ast.Node, bool) {
		return n, !isValueOrLiteral(n)
	}})
	if ok {
		t.Fatal("visitor abort was not propagated from a LIKE or window expression")
	}
}

func TestQueryParenthesizedQueryVisitorsAndAST(t *testing.T) {
	outer := mustParseOne(t, "(select 1 order by 1 limit 3) order by 1 limit 2").(*ast.QueryTailStmt)
	inner := outer.Query.(*ast.SelectStmt)
	if outer.Tail.OrderBy == nil || outer.Tail.Limit == nil || inner.OrderBy == nil || inner.Limit == nil {
		t.Fatal("inner or outer ORDER BY/LIMIT lost")
	}
	outerLimit, outerIsLiteral := outer.Tail.Limit.LimitCore().Count.(*ast.LiteralExpr)
	innerLimit, innerIsLiteral := inner.Limit.LimitCore().Count.(*ast.LiteralExpr)
	if !outerIsLiteral || !innerIsLiteral || outerLimit.Value != "2" || innerLimit.Value != "3" {
		t.Fatal("outer LIMIT replaced the inner LIMIT")
	}
	orders, limits := 0, 0
	_, ok := outer.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch n.(type) {
		case *ast.OrderByClause:
			orders++
		case *ast.Limit:
			limits++
		}
		return n, false
	}})
	if !ok || orders != 2 || limits != 2 {
		t.Fatalf("visitor: ok=%v orders=%d limits=%d", ok, orders, limits)
	}
	assertVisitorControls(t, outer, func(n ast.Node) bool { _, ok := n.(*ast.Limit); return ok })

	set := mustParseOne(t, "((select 1)) union ((select 2))").(*ast.SetOprStmt)
	for i, node := range []ast.QueryNode{set.Left, set.Right} {
		sub, ok := node.(*ast.QueryTailStmt)
		if !ok || !sub.Parenthesized {
			t.Fatalf("unexpected grouped operand %d: %T", i, node)
		}
	}

	cte := mustParseOne(t, "with c as (select 1) (select * from c) order by 1").(*ast.QueryTailStmt)
	if cte.With == nil || cte.Tail.OrderBy == nil {
		t.Fatalf("WITH/ORDER BY shape: %#v", cte)
	}
	if _, ok := cte.Accept(syntaxVisitor{}); !ok {
		t.Fatal("CTE visitor aborted")
	}

	explain := mustParseOne(t, "explain ((select 1 order by 1 limit 3) order by 1 limit 2) order by 1 limit 1").(*ast.ExplainStmt)
	bare, ok := explain.Stmt.(*ast.QueryTailStmt)
	if !ok {
		t.Fatalf("EXPLAIN body unexpected type: %T", explain.Stmt)
	}
	if bare.Tail.OrderBy == nil || bare.Tail.Limit == nil {
		t.Fatalf("explain outer ORDER BY/LIMIT lost: %#v", bare)
	}
	orders, limits = 0, 0
	_, ok = explain.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch n.(type) {
		case *ast.OrderByClause:
			orders++
		case *ast.Limit:
			limits++
		}
		return n, false
	}})
	if !ok || orders != 3 || limits != 3 {
		t.Fatalf("EXPLAIN visitor: ok=%v orders=%d limits=%d", ok, orders, limits)
	}
}

func TestQueryAnalyseTimeZoneRoundTrip(t *testing.T) {
	// These forms were also checked against the pinned MySQL 8.4.13 and
	// MariaDB 12.3.3 servers in testdata/docker-compose.yml.
	cases := []struct{ src, restored string }{
		{"select 1 procedure analyse()", "SELECT 1 PROCEDURE ANALYSE()"},
		{"select a from t order by a limit 2 procedure analyse(1,2) for update", "SELECT `a` FROM `t` ORDER BY `a` LIMIT 2 PROCEDURE ANALYSE(1,2) FOR UPDATE"},
		{"select 1 union select 2 procedure analyse()", "SELECT 1 UNION SELECT 2 PROCEDURE ANALYSE()"},
		{"select cast(timestamp'2019-01-01' at time zone 'UTC' as datetime(6))", "SELECT CAST(TIMESTAMPLITERAL('2019-01-01') AT TIME ZONE 'UTC' AS DATETIME(6))"},
		{"select cast(timestamp'2019-01-01' at time zone interval '+00:00' as datetime)", "SELECT CAST(TIMESTAMPLITERAL('2019-01-01') AT TIME ZONE INTERVAL '+00:00' AS DATETIME)"},
		{"select cast(timestamp'2019-01-01' at local as datetime)", "SELECT CAST(TIMESTAMPLITERAL('2019-01-01') AT LOCAL AS DATETIME)"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			runRestoreTest(t, tc.src, tc.restored, true)
		})
	}
}

func TestQueryAnalyseTimeZoneAndLoopSyntaxErrors(t *testing.T) {
	// Each form below returns a syntax error on its corresponding pinned
	// server. AT LOCAL itself is intentionally not listed: MySQL parses it and
	// returns ER_NOT_SUPPORTED_YET instead of a syntax error.
	assertParseErrors(t, []string{
		"select (select 1 procedure analyse())",
		"select * from (select 1 procedure analyse()) d",
		"create table x as select 1 procedure analyse()",
		"select 1 into @x procedure analyse()",
		"select 1 procedure analyse() into @x",
		"select 1 procedure analyse() union select 2",
		"select cast(timestamp'2019-01-01' at time zone 'UTC' as date)",
		"select cast(timestamp'2019-01-01' at time zone @z as datetime)",
		"select cast(timestamp'2019-01-01' at time zone concat('U','TC') as datetime)",
		"select cast(timestamp'2019-01-01' at time zone _utf8mb4'UTC' as datetime)",
		"select cast(timestamp'2019-01-01' at time zone 'U' 'TC' as datetime)",
		"select cast(timestamp'2019-01-01' as datetime at time zone 'UTC')",
	})
}

func TestQueryCastVarcharAndValuesSetAST(t *testing.T) {
	cast := mustParseOne(t, "select cast('x' as varchar(10))").(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.FuncCastExpr)
	if cast.Type.Keyword() != "VARCHAR" || len(cast.Type.Modifiers) != 1 || cast.Type.Modifiers[0].(*ast.LiteralExpr).Value != "10" {
		t.Fatalf("CAST VARCHAR target was lost: %#v", cast.Type)
	}

	sql := "values (1,5) union values (2,6) union select 3,4"
	set := mustParseOne(t, sql).(*ast.SetOprStmt)
	left, ok := set.Left.(*ast.SetOprStmt)
	if !ok || left.Operator != ast.Union || set.Operator != ast.Union {
		t.Fatalf("VALUES set tree was lost: %#v", set)
	}
	first, ok := left.Left.(*ast.ValuesStmt)
	if !ok || len(first.Rows) == 0 {
		t.Fatalf("first VALUES operand was lost: %#v", left.Left)
	}
	firstRow, rowOK := first.Rows[0].(*ast.RowExpr)
	if !rowOK || !firstRow.OmitRowKeyword || first.ValuesKeywordOffset != 0 {
		t.Fatalf("first VALUES operand was lost: %#v", left.Left)
	}
	second, ok := left.Right.(*ast.ValuesStmt)
	if !ok || second.ValuesKeywordOffset != strings.Index(sql[1:], "values")+1 {
		t.Fatalf("second VALUES operand was lost: %#v", left.Right)
	}
	if _, ok := set.Right.(*ast.SelectStmt); !ok {
		t.Fatalf("SELECT set operand was lost: %#v", set.Right)
	}
	tailed := mustParseOne(t, "values (1) union select 2 order by 1 limit 1").(*ast.SetOprStmt)
	tailedLast := tailed.Right.(*ast.SelectStmt)
	if tailed.Tail.OrderBy == nil || tailed.Tail.Limit == nil || tailedLast.OrderBy != nil || tailedLast.Limit != nil {
		t.Fatalf("VALUES set tail was not attached to the set operation: %#v", tailed)
	}

	valuesNodes, literals := 0, 0
	if _, ok := set.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch n.(type) {
		case *ast.ValuesStmt:
			valuesNodes++
		case *ast.LiteralExpr:
			literals++
		}
		return n, false
	}}); !ok || valuesNodes != 2 || literals != 6 {
		t.Fatalf("VALUES visitor: ok=%v nodes=%d literals=%d", ok, valuesNodes, literals)
	}
}

func TestQueryDerivedAndCycleAST(t *testing.T) {
	const derivedSQL = "select * from (values (1),(2)) d(x)"
	sel := mustParseOne(t, derivedSQL).(*ast.SelectStmt)
	derived := sel.From.FromClauseCore().Table.(*ast.JoinExpr).Left.(*ast.TableSource)
	xStart := strings.Index(derivedSQL, "x")
	wantColumns := []lex.Ident{{Name: "x", Raw: "x", Span: loc.Span{Start: loc.ByteOffset(xStart), End: loc.ByteOffset(xStart + 1)}}}
	derivedValues := derived.Source.(*ast.ValuesStmt)
	derivedRow, rowOK := derivedValues.Rows[0].(*ast.RowExpr)
	if !reflect.DeepEqual(derived.Columns, wantColumns) || !rowOK || !derivedRow.OmitRowKeyword {
		t.Fatalf("derived VALUES or aliases were lost: %#v", derived)
	}
	assertVisitorControls(t, sel, isValueOrLiteral)
	const cycleSQL = "with recursive c(a) as (select 1) cycle a restrict select * from c"
	cte := mustParseOne(t, cycleSQL).(*ast.SelectStmt)
	aStart := strings.Index(cycleSQL, "cycle a") + len("cycle ")
	wantCycle := []lex.Ident{{Name: "a", Raw: "a", Span: loc.Span{Start: loc.ByteOffset(aStart), End: loc.ByteOffset(aStart + 1)}}}
	with := cte.With.(*ast.WithClause)
	if !with.IsRecursive || !reflect.DeepEqual(with.CycleColumns, wantCycle) {
		t.Fatalf("CYCLE clause was lost: %#v", cte.With)
	}
}

func TestQueryVariableTargetASTAndVisitors(t *testing.T) {
	for _, src := range []string{
		"select 1,2,3,4 into r.a, `r.a`, `@a`, @a",
		"select 1,2,3,4 into r.a, `r.a`, `@a`, @a from t",
	} {
		t.Run(src, func(t *testing.T) {
			sel := mustParseOne(t, src).(*ast.SelectStmt)
			wantParts := [][]string{{"r", "a"}, {"r.a"}, {"@a"}, {"a"}}
			if sel.Into == nil || len(sel.Into.Targets) != len(wantParts) {
				t.Fatalf("structured INTO targets were lost: %#v", sel.Into)
			}
			for i, target := range sel.Into.Targets {
				gotParts := make([]string, len(target.Name.Parts))
				for j, part := range target.Name.Parts {
					gotParts[j] = part.Name
				}
				if !reflect.DeepEqual(gotParts, wantParts[i]) || (target.Kind == ast.VariableTargetUser) != (i == 3) {
					t.Fatalf("target %d: %#v", i, target)
				}
			}
			visited := 0
			_, ok := sel.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
				if target, ok := n.(*ast.VariableTarget); ok {
					visited++
					kind, prefix := ast.VariableTargetLocal, ast.VariablePrefixNone
					if target.Kind == ast.VariableTargetUser {
						kind, prefix = ast.VariableTargetUser, ast.VariablePrefixAt
					}
					return &ast.VariableTarget{Name: ast.QualifiedName{Parts: []lex.Ident{{Name: "renamed"}}}, Kind: kind, Prefix: prefix}, false
				}
				return n, false
			}})
			if !ok || visited != 4 {
				t.Fatalf("INTO visitor: ok=%v visited=%d", ok, visited)
			}
			if got := restoreStmt(t, sel); !strings.Contains(got, "INTO `renamed`, `renamed`, `renamed`, @`renamed`") {
				t.Fatalf("INTO replacement lost: %s", got)
			}
			if got := restoreStmt(t, sel); strings.Count(got, " INTO ") != 1 {
				t.Fatalf("duplicate INTO restore: %s", got)
			}
			assertVisitorControls(t, sel.Into, func(n ast.Node) bool { _, ok := n.(*ast.VariableTarget); return ok })
		})
	}
}

func TestQueryExpressionFlags(t *testing.T) {
	sel := mustParseOne(t, "select weight_string(a),weight_string(@a,1,2,3),'a' like 'b' escape ?").(*ast.SelectStmt)
	want := []uint64{ast.FlagHasFunc | ast.FlagHasReference, ast.FlagHasFunc | ast.FlagHasVariable, ast.FlagHasParamMarker}
	for i, field := range sel.Fields.Items {
		if got := ast.ExprFlag(field.Expr); got != want[i] {
			t.Errorf("field %d flags: %d, want %d", i, got, want[i])
		}
	}
	cast := mustParseOne(t, "select cast(1 at time zone 'UTC' as datetime)").(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.FuncCastExpr)
	cast.TimeZone.Expr = ast.NewParamMarkerExprAt(loc.Span{Start: 0, End: 1}, 0)
	if got := ast.ExprFlag(cast); got != ast.FlagHasFunc|ast.FlagHasParamMarker {
		t.Fatalf("time-zone child flags were lost: %d", got)
	}
	weight := sel.Fields.Items[1].Expr.(*ast.WeightStringExpr)
	weight.Params[0] = ast.NewParamMarkerExprAt(loc.Span{Start: 0, End: 1}, 0)
	if got := ast.ExprFlag(weight); got != ast.FlagHasFunc|ast.FlagHasVariable|ast.FlagHasParamMarker {
		t.Fatalf("numeric parameter child flags were lost: %d", got)
	}

	// Window spec subtrees carry their own parameters and variables.
	windowParser := parser.New()
	windowParser.EnableWindowFunc(true)
	windowStmt, err := windowParser.ParseOneStmt("select row_number() over (partition by ? order by @v)", "", "")
	if err != nil {
		t.Fatalf("parse window spec: %v", err)
	}
	window := windowStmt.(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.WindowFuncExpr)
	if got := ast.ExprFlag(window); got != ast.FlagHasWindowFunc|ast.FlagHasParamMarker|ast.FlagHasVariable {
		t.Fatalf("window spec flags: %d, want window|param|variable", got)
	}
}

func TestQueryWeightStringASTAndVisitors(t *testing.T) {
	stmt := mustParseOne(t, "select weight_string(a as char(3) level 0 desc reverse,2 asc),weight_string('a' level 9-0),weight_string('a',1,2.5,0)")
	fields := stmt.(*ast.SelectStmt).Fields.Items
	weight := fields[0].Expr.(*ast.WeightStringExpr)
	wantLevels := []ast.WeightStringLevel{{Number: 0, Order: ast.WeightStringLevelOrderDesc, Reverse: true}, {Number: 2, Order: ast.WeightStringLevelOrderAsc}}
	if weight.CastType != ast.WeightStringCastChar || weight.Length != 3 || !reflect.DeepEqual(weight.Levels, wantLevels) {
		t.Fatalf("WEIGHT_STRING options were lost: %#v", weight)
	}
	ranged := fields[1].Expr.(*ast.WeightStringExpr)
	if ranged.LevelRange == nil || *ranged.LevelRange != (ast.WeightStringLevelRange{Start: 9, End: 0}) {
		t.Fatalf("WEIGHT_STRING range was lost: %#v", ranged)
	}
	params := fields[2].Expr.(*ast.WeightStringExpr)
	if len(params.Params) != 3 {
		t.Fatalf("numeric parameters were lost: %#v", params)
	}
	var formatted bytes.Buffer
	params.Format(&formatted)
	// Format follows the syntax literal's Go-style quoting, unlike SQL Restore.
	if got := formatted.String(); got != `WEIGHT_STRING("a",1,2.5,0)` {
		t.Fatalf("Format: %s", got)
	}

	values := 0
	_, ok := params.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch n.(type) {
		case *ast.LiteralExpr:
			values++
			return &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "9"}}, false
		}
		return n, false
	}})
	if !ok || values != 4 {
		t.Fatalf("WEIGHT_STRING visitor: ok=%v values=%d", ok, values)
	}
	if got := restoreStmt(t, stmt); !strings.Contains(got, "WEIGHT_STRING(9,9,9,9)") {
		t.Fatalf("WEIGHT_STRING replacement lost: %s", got)
	}
	assertVisitorControls(t, params, isValueOrLiteral)
}

package myparser

import (
	"strings"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/mysql"
)

// These forms are accepted by the pinned MySQL/MariaDB MTR servers. Missing
// tables, routines, privileges, and dynamic SQL errors are not parse errors.
func TestMTRStatementRoundTrip(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"call p", "CALL `p`()"},
		{"call db.p(1, @x, a+1)", "CALL `db`.`p`(1, @`x`, `a`+1)"},
		{"explain extended select * from t", "EXPLAIN EXTENDED SELECT * FROM `t`"},
		{"explain partitions select * from t", "EXPLAIN PARTITIONS SELECT * FROM `t`"},
		{"analyze select * from t", "ANALYZE SELECT * FROM `t`"},
		{"analyze format=json select * from t", "ANALYZE FORMAT = 'json' SELECT * FROM `t`"},
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
		{"select sql_calc_found_rows distinct high_priority 1", "SELECT HIGH_PRIORITY SQL_CALC_FOUND_ROWS DISTINCT 1"},
		{"select distinct distinct 1", "SELECT DISTINCT 1"},
		{"select all all 1", "SELECT 1"},
		{"select 1 where 1 group by 1 having 1", "SELECT 1 FROM DUAL WHERE 1 GROUP BY 1 HAVING 1"},
		{"select 1 from dual where 1 group by 1 having 1", "SELECT 1 FROM DUAL WHERE 1 GROUP BY 1 HAVING 1"},
		{"select _latin2'a', _koi8r 0xff, _binary x'41'", "SELECT _LATIN2'a',_KOI8R x'ff',_BINARY x'41'"},
		{"set statement sql_log_bin=0 for call db.p()", "SET STATEMENT `sql_log_bin`=0 FOR CALL `db`.`p`()"},
		{"set statement sql_mode='NO_ENGINE_SUBSTITUTION', max_statement_time=2 for select * from t", "SET STATEMENT `sql_mode`='NO_ENGINE_SUBSTITUTION', `max_statement_time`=2 FOR SELECT * FROM `t`"},
		{"update t set a=default", "UPDATE `t` SET `a`=DEFAULT"},
		{"insert into t set a=default, b=1", "INSERT INTO `t` SET `a`=DEFAULT,`b`=1"},
		{"with c as (select 1 as a) update t set a=(select a from c)", "WITH `c` AS (SELECT 1 AS `a`) UPDATE `t` SET `a`=(SELECT `a` FROM `c`)"},
		{"with c as (select 1 as a) delete from t where a in (select a from c)", "WITH `c` AS (SELECT 1 AS `a`) DELETE FROM `t` WHERE `a` IN (SELECT `a` FROM `c`)"},
		{"explain with c as (select 1) select * from c", "EXPLAIN FORMAT = 'row' WITH `c` AS (SELECT 1) SELECT * FROM `c`"},
		{"with c as (with d as (select 1) select * from d) select * from c", "WITH `c` AS (WITH `d` AS (SELECT 1) SELECT * FROM `d`) SELECT * FROM `c`"},
		{"select (with c as (select 1 as a) select a from c)", "SELECT (WITH `c` AS (SELECT 1 AS `a`) SELECT `a` FROM `c`)"},
		{"insert into t(a) with c as (select 1 as a) select a from c", "INSERT INTO `t` (`a`) WITH `c` AS (SELECT 1 AS `a`) SELECT `a` FROM `c`"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			RunRestoreTest(t, tc.src, tc.restored, true)
		})
	}
	assertExtensionParseErrors(t, []string{
		"call", "call p(,)", "call p(1,)", "call a.b.c()",
		"explain extended select", "analyze format=json", "show create procedure",
		"set statement for select 1", "set statement x=1 for",
		"set statement x=1, for select 1", "set statement @@global.x=1 for select 1",
		"execute immediate 'select 1' using", "execute immediate 'select 1' using 1,",
		"analyze table t persistent for", "analyze table t persistent for columns(a)",
		"with c as (select 1) update", "with c as (select 1) delete",
	})
}

func TestMTRCTEViewAndRoutineRoundTrip(t *testing.T) {
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
			if _, ok := stmt.Accept(extensionVisitor{}); !ok {
				t.Fatal("visitor aborted")
			}
		})
	}
}

func TestMTRParserPanicRegressions(t *testing.T) {
	for _, src := range []string{
		"create procedure p() begin declare a int; set a=1; end",
		"create procedure p() begin declare a, b double(10,3); set a=1; end",
		"create function f() returns longblob begin declare ret longblob; return ret; end",
		"select /*+ */ 1",
		"select /*+ */ (select /*+ */ 1)",
	} {
		t.Run(src, func(t *testing.T) {
			stmt := mustParseOne(t, src)
			if strings.TrimSpace(stmt.Text()) == "" {
				t.Fatal("statement text was lost")
			}
			if _, ok := stmt.Accept(extensionVisitor{}); !ok {
				t.Fatal("visitor aborted")
			}
			mustParseOne(t, restoreStmt(t, stmt))
		})
	}
	var scanner Scanner
	scanner.reset("")
	if got := scanner.stmtText(); got != "" {
		t.Fatalf("empty scanner text: %q", got)
	}
}

func TestMTRStatementVisitors(t *testing.T) {
	for _, src := range []string{
		"call p(1)",
		"set statement x=1 for call p(2)",
		"with c as (select 1 as a) update t set a=2",
		"with c as (select 1 as a) delete from t where a=2",
	} {
		t.Run(src, func(t *testing.T) {
			stmt := mustParseOne(t, src)
			values := 0
			_, ok := stmt.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
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
			_, ok = stmt.Accept(extensionVisitor{leave: func(n ast.Node) (ast.Node, bool) {
				return n, !isValueOrLiteral(n)
			}})
			if ok {
				t.Fatal("visitor abort was not propagated")
			}
			entered := 0
			_, ok = stmt.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
				entered++
				return n, true
			}})
			if !ok || entered != 1 {
				t.Fatalf("skipChildren: ok=%v entered=%d", ok, entered)
			}
		})
	}
}

func TestMTRPhase3RoundTrip(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"select 'andre%' like 'andreй%' escape 'й'", "SELECT 'andre%' LIKE 'andreй%' ESCAPE 'й'"},
		{"select 'abc' like 'a%' escape 0x5c", "SELECT 'abc' LIKE 'a%' ESCAPE x'5c'"},
		{"select 'abc' like 'a%' escape concat('x')", "SELECT 'abc' LIKE 'a%' ESCAPE CONCAT('x')"},
		{"select percentile_cont(0.5) within group(order by score) over(partition by name) from t1", "SELECT PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY `score`) OVER (PARTITION BY `name`) FROM `t1`"},
		{"select foo(a) over(partition by b order by c) from t", "SELECT FOO(`a`) OVER (PARTITION BY `b` ORDER BY `c`) FROM `t`"},
		{"select var_pop(a) over(partition by b), var_samp(a) over() from t", "SELECT VAR_POP(`a`) OVER (PARTITION BY `b`),VAR_SAMP(`a`) OVER () FROM `t`"},
		{"select cast('1' as float), cast('1' as double), cast('1' as double precision), cast('1' as year)", "SELECT CAST('1' AS FLOAT),CAST('1' AS DOUBLE),CAST('1' AS DOUBLE),CAST('1' AS YEAR)"},
		{"create table t(a year(2), b year(5))", "CREATE TABLE `t` (`a` YEAR(2),`b` YEAR(5))"},
		{"select 'a' collate utf8mb4_uca1400_ai_ci", "SELECT 'a' COLLATE utf8mb4_uca1400_ai_ci"},
		{"select within from t", "SELECT `within` FROM `t`"},
		{"(select a, b as c from t1) order by c+1", "(SELECT `a`,`b` AS `c` FROM `t1`) ORDER BY `c`+1"},
		{"((select 1))", "SELECT 1"},
		{"(((select 1))) order by 1 limit 1", "(SELECT 1) ORDER BY 1 LIMIT 1"},
		{"select ((select 1))", "SELECT (SELECT 1)"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			RunRestoreTest(t, tc.src, tc.restored, true)
		})
	}

	assertExtensionParseErrors(t, []string{
		"select 1 like 1 escape",
		"select cast(1 as year(4))",
		"select cast(1 as float unsigned)",
		"create table t(a year())",
		"((select 1)",
		"(select 1))",
	})
	for _, src := range []string{
		"select percentile_cont(0.5) within group(order by a,b) over()",
		"select percentile_cont(0.5) within group(order by a)",
		"select percentile_cont(0.5) within group() over()",
		"select foo(a) over",
	} {
		p := New()
		p.EnableWindowFunc(true)
		if _, _, err := p.Parse(src, "", ""); err == nil {
			t.Errorf("expected parse error for %q", src)
		}
	}
}

func TestMTRPhase3ExpressionASTAndVisitors(t *testing.T) {
	p := New()
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
	_, ok := stmt.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
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

	_, ok = stmt.Accept(extensionVisitor{leave: func(n ast.Node) (ast.Node, bool) {
		return n, !isValueOrLiteral(n)
	}})
	if ok {
		t.Fatal("visitor abort was not propagated from a phase-three expression")
	}
}

func TestMTRPhase3CastAST(t *testing.T) {
	stmt := mustParseOne(t, "select cast(1 as float), cast(2 as double), cast(3 as year)")
	fields := stmt.(*ast.SelectStmt).Fields.Items
	want := []string{"FLOAT", "DOUBLE", "YEAR"}
	for i, field := range fields {
		cast, ok := field.Expr.(*ast.FuncCastExpr)
		if !ok {
			t.Fatalf("field %d: got %T, want *ast.FuncCastExpr", i, field.Expr)
		}
		if cast.Type.Keyword() != want[i] {
			t.Fatalf("field %d: got type %s, want %s", i, cast.Type.Keyword(), want[i])
		}
	}
	if _, ok := stmt.Accept(extensionVisitor{}); !ok {
		t.Fatal("cast visitor aborted")
	}
	RunRestoreTest(t, restoreStmt(t, stmt), "SELECT CAST(1 AS FLOAT),CAST(2 AS DOUBLE),CAST(3 AS YEAR)", true)
}

func TestMTRUnsupportedNationalStringTypes(t *testing.T) {
	// National string types always use the deliberately unsupported utf8mb3.
	assertExtensionParseErrors(t, []string{
		"create table t(a nchar(10))",
		"create table t(a national char(10))",
		"create table t(a national character(10))",
		"create table t(a national varchar(10))",
		"create table t(a nvarchar(10))",
		"create table t(a nchar varchar(10))",
		"create table t(a national character varying(10))",
		"create table t(a nchar varying(10))",
		"alter table t add a national char(10)",
		"select cast(1 as nchar)",
		"select convert(1, nchar(10))",
	})
	for _, src := range []string{
		"create table t(a char(10), b varchar(10))",
		"create table national(a int)",
		"select cast(1 as char(10))",
	} {
		t.Run(src, func(t *testing.T) {
			stmt := mustParseOne(t, src)
			mustParseOne(t, restoreStmt(t, stmt))
		})
	}
}

func TestMTRTransactionCompletion(t *testing.T) {
	for _, command := range []string{"commit", "rollback"} {
		for _, options := range []string{
			"", "and chain", "and no chain", "release", "no release",
			"and chain no release", "and no chain release", "and no chain no release",
		} {
			src := command + " work"
			want := strings.ToUpper(command)
			if options != "" {
				src += " " + options
				want += " " + strings.ToUpper(options)
			}
			t.Run(src, func(t *testing.T) {
				RunRestoreTest(t, src, want, true)
				stmt := mustParseOne(t, src)
				var got ast.TransactionOptions
				switch s := stmt.(type) {
				case *ast.CommitStmt:
					got = s.TransactionOptions
				case *ast.RollbackStmt:
					got = s.TransactionOptions
				}
				if (got.Chain != nil) != strings.Contains(options, "chain") ||
					(got.Release != nil) != strings.Contains(options, "release") {
					t.Fatalf("explicit completion choices lost: %#v", got)
				}
				if got.Chain != nil && *got.Chain != !strings.Contains(options, "no chain") {
					t.Fatalf("NO CHAIN lost: %#v", got)
				}
				if got.Release != nil && *got.Release != !strings.Contains(options, "no release") {
					t.Fatalf("NO RELEASE lost: %#v", got)
				}
				if _, ok := stmt.Accept(extensionVisitor{}); !ok {
					t.Fatal("transaction visitor aborted")
				}
			})
		}
	}
	RunRestoreTest(t, "rollback work to savepoint s", "ROLLBACK TO SAVEPOINT `s`", true)
	assertExtensionParseErrors(t, []string{
		"commit and chain release", "rollback work and chain release",
		"commit chain", "rollback and no", "commit no chain",
		"commit release and chain", "rollback to s no release",
		"rollback and chain to savepoint s", "commit work work",
	})
}

func TestMTRHighFrequencyDDLRoundTrip(t *testing.T) {
	for _, tc := range []struct{ src, restored string }{
		{"create table t(a serial comment 'x', b int)", "CREATE TABLE `t` (`a` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT UNIQUE KEY COMMENT 'x',`b` INT)"},
		{"alter table t add a serial", "ALTER TABLE `t` ADD COLUMN `a` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT UNIQUE KEY"},
		{"create index i type btree on t(a)", "CREATE INDEX `i` ON `t` (`a`) USING BTREE"},
		{"create index i on t(a) type hash", "CREATE INDEX `i` ON `t` (`a`) USING HASH"},
		{"create index i using rtree on t(a)", "CREATE INDEX `i` ON `t` (`a`) USING RTREE"},
		{"alter table t convert to character set default", "ALTER TABLE `t` CONVERT TO CHARACTER SET DEFAULT"},
		{"alter table t convert to character set default collate latin1_bin", "ALTER TABLE `t` CONVERT TO CHARACTER SET DEFAULT COLLATE latin1_bin"},
		{"alter table t convert to charset latin1", "ALTER TABLE `t` CONVERT TO CHARACTER SET latin1"},
		{"alter table t default charset latin1 collate latin1_bin", "ALTER TABLE `t` DEFAULT CHARACTER SET = latin1 DEFAULT COLLATE = latin1_bin"},
	} {
		t.Run(tc.src, func(t *testing.T) {
			RunRestoreTest(t, tc.src, tc.restored, true)
		})
	}
	assertExtensionParseErrors(t, []string{
		"create table t(a serial as (1))", "alter table t add a serial as (1)",
		"create index i type on t(a)", "create index i type unknown on t(a)",
		"create table t(a int, key type btree (a))",
		"alter table t convert to character set", "alter table t convert to character set default collate",
	})
}

func TestMTRHighFrequencyDDLVisitors(t *testing.T) {
	stmt := mustParseOne(t, "create table t(a serial comment 'x')")
	col := stmt.(*ast.CreateTableStmt).Columns[0].(*ast.ColumnDef)
	if typ := col.Type.(*ast.TypeName); typ.Keyword() != "BIGINT" || !typ.Unsigned || len(col.Options) != 4 {
		t.Fatalf("SERIAL type or implicit options lost: %#v", col)
	}
	for i, tp := range []ast.ColumnOptionType{
		ast.ColumnOptionNotNull, ast.ColumnOptionAutoIncrement, ast.ColumnOptionUniqKey, ast.ColumnOptionComment,
	} {
		if col.Options[i].Tp != tp {
			t.Fatalf("SERIAL option %d = %v, want %v", i, col.Options[i].Tp, tp)
		}
	}
	values := 0
	_, ok := stmt.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch n := n.(type) {
		case *ast.LiteralExpr:
			values++
			return replaceLiteral(n, "changed"), false
		}
		return n, false
	}})
	if !ok || values != 1 || !strings.Contains(restoreStmt(t, stmt), "COMMENT 'changed'") {
		t.Fatalf("SERIAL visitor lost attributes: ok=%v values=%d", ok, values)
	}
	_, ok = stmt.Accept(extensionVisitor{leave: func(n ast.Node) (ast.Node, bool) {
		_, isLiteral := n.(*ast.LiteralExpr)
		return n, !isLiteral
	}})
	if ok {
		t.Fatal("SERIAL visitor abort was not propagated")
	}
}

func TestMTRHighFrequencyShowRoundTrip(t *testing.T) {
	for _, src := range []string{"show replica status", "show replicas", "show binary log status", "show binary logs"} {
		t.Run(src, func(t *testing.T) {
			RunRestoreTest(t, src, strings.ToUpper(src), true)
		})
	}
	assertExtensionParseErrors(t, []string{"show replica", "show replicas status", "show binary log", "show binary status"})
}

func TestMTRHighFrequencyDiagnosticsAndKillRoundTrip(t *testing.T) {
	for _, tc := range []struct{ src, restored string }{
		{"show warnings limit 1", "SHOW WARNINGS LIMIT 1"},
		{"show errors limit 1,2", "SHOW ERRORS LIMIT 1,2"},
		{"show warnings limit 2 offset 1", "SHOW WARNINGS LIMIT 1,2"},
		{"show errors limit ?,?", "SHOW ERRORS LIMIT ?,?"},
		{"kill query @id", "KILL QUERY @`id`"},
		{"kill 99999999+1", "KILL 99999999+1"},
		{"kill connection (99999999+1)", "KILL (99999999+1)"},
		{"kill query '99999999'", "KILL QUERY '99999999'"},
		{"kill query (select 99999999)", "KILL QUERY (SELECT 99999999)"},
		{"kill query null", "KILL QUERY NULL"},
		{"kill query ?", "KILL QUERY ?"},
	} {
		t.Run(tc.src, func(t *testing.T) {
			RunRestoreTest(t, tc.src, tc.restored, true)
		})
	}
	assertExtensionParseErrors(t, []string{
		"show warnings where Code=1", "show errors like '%'", "show warnings limit @n",
		"show errors limit 1+1", "show warnings limit -1", "show errors limit 1,",
		"kill query 99999999,99999998", "kill connection", "kill query 1+",
	})
}

func TestMTRHighFrequencyBuiltinTableNames(t *testing.T) {
	for _, name := range []string{
		"adddate", "bit_and", "bit_or", "bit_xor", "cast", "count", "curdate", "curtime",
		"date_add", "date_sub", "extract", "group_concat", "json_arrayagg", "max", "mid", "min",
		"now", "position", "session_user", "std", "stddev", "stddev_pop", "stddev_samp", "subdate",
		"substr", "substring", "sum", "sysdate", "system_user", "trim", "user", "variance", "var_pop", "var_samp",
	} {
		t.Run(name, func(t *testing.T) {
			adjacentAllowed := false
			switch name {
			case "adddate", "session_user", "subdate", "system_user", "user":
				adjacentAllowed = true
			}
			for _, mode := range []struct {
				name  string
				value mysql.SQLMode
			}{
				{name: "empty"},
				{name: "ignore space", value: mysql.ModeIgnoreSpace},
			} {
				t.Run(mode.name, func(t *testing.T) {
					parser := New()
					parser.SetSQLMode(mode.value)
					for _, tc := range []struct {
						name, sql string
						accepted  bool
					}{
						{name: "create adjacent", sql: "create table " + name + "(a int)", accepted: adjacentAllowed},
						{name: "insert adjacent", sql: "insert into " + name + "(a) values (1)", accepted: adjacentAllowed},
						{name: "space", sql: "create table " + name + " (a int)", accepted: adjacentAllowed || mode.value == 0},
						{name: "quoted", sql: "create table `" + name + "`(a int)", accepted: true},
						{name: "qualified", sql: "create table db." + name + "(a int)", accepted: true},
						{name: "no parentheses", sql: "create table " + name + " as select 1 as a", accepted: true},
					} {
						t.Run(tc.name, func(t *testing.T) {
							stmt, err := parser.ParseOneStmt(tc.sql, "", "")
							if !tc.accepted {
								if err == nil {
									t.Fatal("accepted an ambiguous built-in table name")
								}
								return
							}
							if err != nil {
								t.Fatal(err)
							}
							if restored := restoreStmt(t, stmt); !strings.Contains(restored, "`"+name+"`") {
								t.Fatalf("table name lost: %s", restored)
							}
						})
					}
				})
			}
		})
	}
	RunRestoreTest(t, "select count(*),cast(1 as signed),substring('abc',1)",
		"SELECT COUNT(*),CAST(1 AS SIGNED),SUBSTRING('abc', 1)", true)
}

func TestMTRHighFrequencyDiagnosticsAndKillVisitors(t *testing.T) {
	for _, tc := range []struct {
		src, restored string
		children      int
	}{
		{"show warnings limit 1,2", "SHOW WARNINGS LIMIT 9,9", 2},
		{"show profile limit 1,2", "SHOW PROFILE LIMIT 9,9", 2},
		{"kill connection (1+2)", "KILL (9+9)", 2},
		{"kill query (select 1)", "KILL QUERY (SELECT 9)", 1},
		{"kill query @id", "KILL QUERY 9", 1},
	} {
		t.Run(tc.src, func(t *testing.T) {
			stmt := mustParseOne(t, tc.src)
			children := 0
			isChild := func(n ast.Node) bool {
				switch n.(type) {
				case *ast.LiteralExpr, *ast.VariableExpr:
					return true
				}
				return false
			}
			_, ok := stmt.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
				switch n := n.(type) {
				case *ast.LiteralExpr:
					children++
					return replaceLiteral(n, "9"), true
				case *ast.VariableExpr:
					// Replace the variable with an integer literal so KILL
					// restores the plain connection-ID form.
					children++
					return &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "9"}}, true
				}
				return n, false
			}})
			if !ok || children != tc.children || restoreStmt(t, stmt) != tc.restored {
				t.Fatalf("visitor replacement: ok=%v children=%d sql=%s", ok, children, restoreStmt(t, stmt))
			}
			mustParseOne(t, tc.restored)
			assertMTRPhase5VisitorControls(t, stmt, isChild)
		})
	}
	show := mustParseOne(t, "show errors limit 2 offset 1").(*ast.ShowStmt)
	if show.Tp != ast.ShowErrors || show.Limit == nil || show.ShowProfileLimit != nil {
		t.Fatalf("diagnostic limit AST lost: %#v", show)
	}
	for _, tc := range []struct {
		sql string
		id  uint64
	}{
		{"kill 7", 7}, {"kill query 18446744073709551615", ^uint64(0)},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			kill := mustParseOne(t, tc.sql).(*ast.KillStmt)
			if kill.ConnectionID != tc.id || kill.Expr != nil {
				t.Fatalf("literal connection ID changed: %#v", kill)
			}
		})
	}
	RunRestoreTest(t, "select connection+1,query+1", "SELECT `connection`+1,`query`+1", true)
}

func TestMTRWindowFrameRoundTrip(t *testing.T) {
	for _, tc := range []struct{ name, src, restored string }{
		{"string start", "select sum(a) over (range 'abcd' preceding) from t", "SELECT SUM(`a`) OVER (RANGE BETWEEN 'abcd' PRECEDING AND CURRENT ROW) FROM `t`"},
		{"string end", "select sum(a) over (rows between current row and '3' following) from t", "SELECT SUM(`a`) OVER (ROWS BETWEEN CURRENT ROW AND '3' FOLLOWING) FROM `t`"},
		{"groups", "select sum(a) over (groups between 1 preceding and 1 following) from t", "SELECT SUM(`a`) OVER (GROUPS BETWEEN 1 PRECEDING AND 1 FOLLOWING) FROM `t`"},
		{"exclude current row", "select sum(a) over (rows current row exclude current row) from t", "SELECT SUM(`a`) OVER (ROWS BETWEEN CURRENT ROW AND CURRENT ROW EXCLUDE CURRENT ROW) FROM `t`"},
		{"exclude group", "select sum(a) over (rows current row exclude group) from t", "SELECT SUM(`a`) OVER (ROWS BETWEEN CURRENT ROW AND CURRENT ROW EXCLUDE GROUP) FROM `t`"},
		{"exclude ties", "select sum(a) over (rows current row exclude ties) from t", "SELECT SUM(`a`) OVER (ROWS BETWEEN CURRENT ROW AND CURRENT ROW EXCLUDE TIES) FROM `t`"},
		{"exclude no others", "select sum(a) over w from t window w as (rows current row exclude no others)", "SELECT SUM(`a`) OVER `w` FROM `t` WINDOW `w` AS (ROWS BETWEEN CURRENT ROW AND CURRENT ROW EXCLUDE NO OTHERS)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			RunRestoreTest(t, tc.src, tc.restored, true)
		})
	}
	p := New()
	p.EnableWindowFunc(true)
	for _, sql := range []string{
		"select sum(a) over (rows current row exclude current) from t",
		"select sum(a) over (rows current row exclude others) from t",
		"select sum(a) over (rows current row exclude no ties) from t",
		"select sum(a) over (rows current row exclude no others exclude ties) from t",
		"select sum(a) over (exclude no others) from t",
		"select sum(a) over (rows '2') from t",
	} {
		t.Run(sql, func(t *testing.T) {
			if _, err := p.ParseOneStmt(sql, "", ""); err == nil {
				t.Fatal("accepted invalid window frame")
			}
		})
	}
}

func TestMTRWindowFrameVisitors(t *testing.T) {
	p := New()
	p.EnableWindowFunc(true)
	stmt, err := p.ParseOneStmt("select sum(a) over (rows between '2' preceding and '3' following exclude ties)", "", "")
	if err != nil {
		t.Fatal(err)
	}
	values := 0
	_, ok := stmt.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch n.(type) {
		case *ast.LiteralExpr:
			values++
			return &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "4"}}, true
		}
		return n, false
	}})
	want := "SELECT SUM(`a`) OVER (ROWS BETWEEN 4 PRECEDING AND 4 FOLLOWING EXCLUDE TIES)"
	if !ok || values != 2 || restoreStmt(t, stmt) != want {
		t.Fatalf("frame visitor: ok=%v values=%d sql=%s", ok, values, restoreStmt(t, stmt))
	}
	for _, isChild := range []func(ast.Node) bool{
		func(n ast.Node) bool { _, ok := n.(*ast.FrameClause); return ok },
		func(n ast.Node) bool { _, ok := n.(*ast.FrameBound); return ok },
		isValueOrLiteral,
	} {
		assertMTRPhase5VisitorControls(t, stmt, isChild)
	}
}

func TestMTRParenthesizedQueryRoundTrip(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"(select 1)", "SELECT 1"},
		{"(select 1 order by 1 limit 1)", "SELECT 1 ORDER BY 1 LIMIT 1"},
		{"((select 1))", "SELECT 1"},
		{"((select 1)) order by 1 limit 1", "(SELECT 1) ORDER BY 1 LIMIT 1"},
		{"((select 1 order by 1 limit 1)) order by 1 limit 1", "(SELECT 1 ORDER BY 1 LIMIT 1) ORDER BY 1 LIMIT 1"},
		{"select ((select 1))", "SELECT (SELECT 1)"},
		{"select ((select 1 order by 1 limit 1))", "SELECT (SELECT 1 ORDER BY 1 LIMIT 1)"},
		{"select (((select 1)) order by 1 limit 1)", "SELECT ((SELECT 1) ORDER BY 1 LIMIT 1)"},
		{"select ((select 1) + 2)", "SELECT ((SELECT 1)+2)"},
		{"((select 1)) union ((select 2))", "(SELECT 1) UNION (SELECT 2)"},
		{"(select 1) union (select 2) order by 1", "(SELECT 1) UNION (SELECT 2) ORDER BY 1"},
		{"(select 1) union select 2 order by 1 limit 1", "(SELECT 1) UNION SELECT 2 ORDER BY 1 LIMIT 1"},
		{"(select 1) union select 2 order by 1", "(SELECT 1) UNION SELECT 2 ORDER BY 1"},
		{"(select 1) union select 2 limit 1", "(SELECT 1) UNION SELECT 2 LIMIT 1"},
		{"(select 1 union select 2) order by 1 limit 1", "(SELECT 1 UNION SELECT 2) ORDER BY 1 LIMIT 1"},
		{"((select 1 order by 1 limit 1) order by 1 limit 1) union ((select 2))", "((SELECT 1 ORDER BY 1 LIMIT 1) ORDER BY 1 LIMIT 1) UNION (SELECT 2)"},
		{"with c as (select 1) (select * from c)", "WITH `c` AS (SELECT 1) SELECT * FROM `c`"},
		{"with c as (select 1) (select * from c) order by 1", "WITH `c` AS (SELECT 1) (SELECT * FROM `c`) ORDER BY 1"},
		{"with c as (select 1) (select * from c) union select 2", "WITH `c` AS (SELECT 1) (SELECT * FROM `c`) UNION SELECT 2"},
		{"(with c as (select 1) select * from c) union select 2", "(WITH `c` AS (SELECT 1) SELECT * FROM `c`) UNION SELECT 2"},
		{"with c as (select 1) (with d as (select 2) select * from d)", "WITH `c` AS (SELECT 1) (WITH `d` AS (SELECT 2) SELECT * FROM `d`)"},
		{"with c as ((select 1) order by 1 limit 1) select * from c", "WITH `c` AS ((SELECT 1) ORDER BY 1 LIMIT 1) SELECT * FROM `c`"},
		{"select * from ((select 1) order by 1) d", "SELECT * FROM ((SELECT 1) ORDER BY 1) AS `d`"},
		{"select * from (((select 1) order by 1 limit 1) limit 2) d", "SELECT * FROM (((SELECT 1) ORDER BY 1 LIMIT 1) LIMIT 2) AS `d`"},
		{"select * from ((select 1) union (select 2)) d", "SELECT * FROM ((SELECT 1) UNION (SELECT 2)) AS `d`"},
		{"explain ((select 1 order by 1 limit 3) order by 1 limit 2) order by 1 limit 1", "EXPLAIN FORMAT = 'row' ((SELECT 1 ORDER BY 1 LIMIT 3) ORDER BY 1 LIMIT 2) ORDER BY 1 LIMIT 1"},
		{"create view v as (select 1 as a)", "CREATE ALGORITHM = UNDEFINED DEFINER = CURRENT_USER SQL SECURITY DEFINER VIEW `v` AS SELECT 1 AS `a`"},
		{"alter view v as (select 1 as a)", "ALTER ALGORITHM = UNDEFINED DEFINER = CURRENT_USER SQL SECURITY DEFINER VIEW `v` AS SELECT 1 AS `a`"},
		{"alter view v as ((select 1 as a))", "ALTER ALGORITHM = UNDEFINED DEFINER = CURRENT_USER SQL SECURITY DEFINER VIEW `v` AS SELECT 1 AS `a`"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			RunRestoreTest(t, tc.src, tc.restored, true)
		})
	}
	assertExtensionParseErrors(t, []string{
		"(select 1) order by", "(select 1) limit",
		"(select 1) order by 1 where 1", "select (1 union 2)",
		"select ((select 1) order by)", "select * from ((select 1) order by 1)",
	})
}

func TestMTRParenthesizedQueryVisitorsAndAST(t *testing.T) {
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
	_, ok := outer.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
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
	assertMTRPhase5VisitorControls(t, outer, func(n ast.Node) bool { _, ok := n.(*ast.Limit); return ok })

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
	if _, ok := cte.Accept(extensionVisitor{}); !ok {
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
	_, ok = explain.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
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

func TestMTRPhase4RoundTrip(t *testing.T) {
	// These forms were also checked against the pinned MySQL 8.4.13 and
	// MariaDB 12.3.3 servers in testdata/docker-compose.yml.
	cases := []struct{ src, restored string }{
		{"select 1 procedure analyse()", "SELECT 1 PROCEDURE ANALYSE()"},
		{"select a from t order by a limit 2 procedure analyse(1,2) for update", "SELECT `a` FROM `t` ORDER BY `a` LIMIT 2 PROCEDURE ANALYSE(1,2) FOR UPDATE"},
		{"select 1 union select 2 procedure analyse()", "SELECT 1 UNION SELECT 2 PROCEDURE ANALYSE()"},
		{"select cast(timestamp'2019-01-01' at time zone 'UTC' as datetime(6))", "SELECT CAST(TIMESTAMPLITERAL('2019-01-01') AT TIME ZONE 'UTC' AS DATETIME(6))"},
		{"select cast(timestamp'2019-01-01' at time zone interval '+00:00' as datetime)", "SELECT CAST(TIMESTAMPLITERAL('2019-01-01') AT TIME ZONE INTERVAL '+00:00' AS DATETIME)"},
		{"select cast(timestamp'2019-01-01' at local as datetime)", "SELECT CAST(TIMESTAMPLITERAL('2019-01-01') AT LOCAL AS DATETIME)"},
		{"for i in 1..31 do select i; end for", "FOR `i` IN 1..31 DO SELECT `i`; END FOR"},
		{"for i in 1.5..2.5 do select i; end for", "FOR `i` IN 1.5..2.5 DO SELECT `i`; END FOR"},
		{"create procedure p() begin outer_loop: for i in 1..2 do select i; end for outer_loop; end", "CREATE DEFINER = CURRENT_USER PROCEDURE `p`() BEGIN\n  `outer_loop`: FOR `i` IN 1..2 DO SELECT `i`; END FOR `outer_loop`;\nEND"},
		{"begin not atomic select 1; end", "BEGIN NOT ATOMIC\n  SELECT 1;\nEND"},
		{"create table g(a geometry, b point srid 0, c linestring, d polygon, e multipoint, f multilinestring, g multipolygon, h geometrycollection)", "CREATE TABLE `g` (`a` GEOMETRY,`b` POINT SRID 0,`c` LINESTRING,`d` POLYGON,`e` MULTIPOINT,`f` MULTILINESTRING,`g` MULTIPOLYGON,`h` GEOMETRYCOLLECTION)"},
		{"select point(1,2), polygon(linestring(point(0,0),point(1,1),point(0,0)))", "SELECT point(1, 2),polygon(linestring(point(0, 0), point(1, 1), point(0, 0)))"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			RunRestoreTest(t, tc.src, tc.restored, true)
		})
	}
}

func TestMTRPhase4ServerRejectedSyntax(t *testing.T) {
	// Each form below returns a syntax error on its corresponding pinned
	// server. AT LOCAL itself is intentionally not listed: MySQL parses it and
	// returns ER_NOT_SUPPORTED_YET instead of a syntax error.
	assertExtensionParseErrors(t, []string{
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
		"top_label: for i in 1..2 do select i; end for top_label",
	})
}

func TestMTRPhase4ASTAndVisitors(t *testing.T) {
	stmt := mustParseOne(t, "select 1 procedure analyse(2+3) for update")
	sel := stmt.(*ast.SelectStmt)
	if sel.ProcedureAnalyse == nil || len(sel.ProcedureAnalyse.Args) != 1 || sel.LockTp != ast.SelectLockForUpdate {
		t.Fatalf("PROCEDURE ANALYSE AST was not retained: %#v", sel)
	}
	values := 0
	_, ok := stmt.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch n := n.(type) {
		case *ast.LiteralExpr:
			values++
			return replaceLiteral(n, "9"), false
		}
		return n, false
	}})
	if !ok || values != 3 {
		t.Fatalf("PROCEDURE ANALYSE visitor: ok=%v values=%d, want 3", ok, values)
	}
	if got := restoreStmt(t, stmt); got != "SELECT 9 PROCEDURE ANALYSE(9+9) FOR UPDATE" {
		t.Fatalf("PROCEDURE ANALYSE visitor replacement lost: %s", got)
	}

	stmt = mustParseOne(t, "select cast(1 at time zone interval '+00:00' as datetime(3))")
	cast := stmt.(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.FuncCastExpr)
	if cast.TimeZone == nil || !cast.TimeZone.Interval || cast.TimeZone.Local || cast.TimeZone.Expr == nil ||
		cast.Type.Keyword() != "DATETIME" || len(cast.Type.Modifiers) != 1 || cast.Type.Modifiers[0].(*ast.LiteralExpr).Value != "3" {
		t.Fatalf("CAST time-zone AST was not retained: %#v", cast)
	}
	_, ok = stmt.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch n := n.(type) {
		case *ast.TypeName:
			// Type modifiers keep their own spelling.
			return n, true
		case *ast.LiteralExpr:
			if n.Kind == pkast.LiteralString || n.Kind == pkast.LiteralBytes {
				return replaceLiteral(n, "UTC"), false
			}
			return replaceLiteral(n, "9"), false
		}
		return n, false
	}})
	if !ok {
		t.Fatal("CAST time-zone visitor aborted")
	}
	if got := restoreStmt(t, stmt); got != "SELECT CAST(9 AT TIME ZONE INTERVAL 'UTC' AS DATETIME(3))" {
		t.Fatalf("CAST time-zone visitor replacement lost: %s", got)
	}
	mustParseOne(t, restoreStmt(t, stmt))

	stmt = mustParseOne(t, "for i in 1..2 do select 3; end for")
	loop := stmt.(*ast.ForStmt)
	if loop.Variable.Name != "i" || loop.LowerBound == nil || loop.UpperBound == nil || loop.Query != nil || len(loop.List) != 1 {
		t.Fatalf("FOR AST was not retained: %#v", loop)
	}
	values = 0
	_, ok = stmt.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch n := n.(type) {
		case *ast.LiteralExpr:
			values++
			return replaceLiteral(n, "9"), false
		}
		return n, false
	}})
	if !ok || values != 3 || restoreStmt(t, stmt) != "FOR `i` IN 9..9 DO SELECT 9; END FOR" {
		t.Fatalf("FOR visitor replacement lost: ok=%v values=%d sql=%s", ok, values, restoreStmt(t, stmt))
	}

	stmt = mustParseOne(t, "begin not atomic select 1; end")
	block := stmt.(*ast.BeginEndStmt)
	if !block.NotAtomic || len(block.List) != 1 {
		t.Fatalf("top-level compound statement AST was not retained: %#v", block)
	}

	stmt = mustParseOne(t, "create table g(a geometry, b point srid 4326, c multipolygon)")
	create := stmt.(*ast.CreateTableStmt)
	wantGeometry := []string{"GEOMETRY", "POINT", "MULTIPOLYGON"}
	for i, want := range wantGeometry {
		if typ := create.Columns[i].ColumnDefCore().Type.(*ast.TypeName); typ.Keyword() != want {
			t.Fatalf("column %d spatial type: %#v, want %s", i, typ, want)
		}
	}
	if opts := create.Columns[1].(*ast.ColumnDef).Options; len(opts) != 1 || opts[0].Tp != ast.ColumnOptionSRID {
		t.Fatalf("SRID option was not retained: %#v", opts)
	}
	_, ok = stmt.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch n := n.(type) {
		case *ast.LiteralExpr:
			return replaceLiteral(n, "0"), false
		}
		return n, false
	}})
	if !ok || !strings.Contains(restoreStmt(t, stmt), "POINT SRID 0") {
		t.Fatalf("SRID visitor replacement lost: ok=%v sql=%s", ok, restoreStmt(t, stmt))
	}
}

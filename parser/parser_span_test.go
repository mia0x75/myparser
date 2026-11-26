package parser

import (
	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/parserkit/loc"
	"strings"
	"testing"
)

func TestSpanGeneratedReductionAndReuse(t *testing.T) {
	var parser yyParserImpl
	for _, tc := range []struct {
		name, sql string
		span      loc.Span
	}{
		{"select", "SELECT 1", loc.Span{Start: 0, End: 8}},
		{"trailing punctuation", " SELECT 1; ", loc.Span{Start: 1, End: 10}},
		{"epsilon after select", "", loc.InvalidSpan()},
		{"skipped comment", "/* comment */ SELECT 1", loc.Span{Start: 14, End: 22}},
		{"epsilon comment", "/* comment */", loc.InvalidSpan()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := parseState{src: tc.sql}
			st.lexer.reset(tc.sql)
			st.lexer.parser = &st
			if result := parser.Parse(&st.lexer); result != 0 || len(st.lexer.errs) != 0 {
				t.Fatalf("parse = %d, errors = %v", result, st.lexer.errs)
			}
			// These short inputs stay in the fixed stack, whose first slot holds Start.
			if got := parser.stack[1].span; got != tc.span {
				t.Fatalf("Start reduction = %v, want %v", got, tc.span)
			}
		})
	}
}

func TestWindowFrameNodeSpans(t *testing.T) {
	sql := "SELECT SUM(a) OVER (ROWS BETWEEN 2 PRECEDING AND CURRENT ROW) FROM t"
	p := New()
	p.EnableWindowFunc(true)
	stmt, err := p.ParseOneStmt(sql)
	if err != nil {
		t.Fatal(err)
	}
	window := stmt.(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.WindowFuncExpr)
	frame := window.Spec.Frame.(*ast.FrameClause)
	for _, tc := range []struct {
		name string
		node ast.Node
		text string
	}{
		{"frame", frame, "ROWS BETWEEN 2 PRECEDING AND CURRENT ROW"},
		{"start bound", frame.Extent.Start, "2 PRECEDING"},
		{"end bound", frame.Extent.End, "CURRENT ROW"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			span := tc.node.Span()
			if !span.Within(len(sql)) {
				t.Fatalf("span %v is outside the source", span)
			}
			if got := sql[span.Start:span.End]; got != tc.text {
				t.Fatalf("span %v selects %q, want %q", span, got, tc.text)
			}
		})
	}
}

func TestSelectCoreNodeSpans(t *testing.T) {
	sql := "WITH c AS (SELECT 1) SELECT x FROM t GROUP BY x HAVING x > 0"
	stmt, err := New().ParseOneStmt(sql)
	if err != nil {
		t.Fatal(err)
	}
	selectStmt := stmt.(*ast.SelectStmt)
	for _, tc := range []struct {
		name string
		node ast.Node
		text string
	}{
		{"select", selectStmt, sql},
		{"from", selectStmt.From, "t"},
		{"having", selectStmt.Having, "x > 0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			span := tc.node.Span()
			if !span.Within(len(sql)) || sql[span.Start:span.End] != tc.text {
				t.Fatalf("span %v selects %q, want %q", span, sql[span.Start:span.End], tc.text)
			}
		})
	}
}

func TestAlterColumnDefinitionSpans(t *testing.T) {
	for _, clause := range []string{"ADD `名` INT DEFAULT 1", "MODIFY `名` INT DEFAULT 1"} {
		t.Run(clause, func(t *testing.T) {
			sql := "ALTER TABLE t " + clause
			stmt, err := New().ParseOneStmt(sql)
			if err != nil {
				t.Fatal(err)
			}
			col := stmt.(*ast.AlterTableStmt).Specs[0].NewColumns[0]
			span := col.Span()
			want := clause[strings.IndexByte(clause, '`'):]
			if !span.Within(len(sql)) || sql[span.Start:span.End] != want {
				t.Fatalf("column span %v, want %q", span, want)
			}
			if col.ColumnDefCore() != &col.BaseColumnDef || col.Name.Parts[0].Raw != "`名`" {
				t.Fatalf("shared column syntax lost: %+v", col.ColumnDefCore())
			}
		})
	}
}

func TestTableAndPartitionOptionSpans(t *testing.T) {
	sql := "CREATE TABLE t (a INT) ENGINE=InnoDB COMMENT='table' PARTITION BY RANGE (a) (PARTITION p VALUES LESS THAN MAXVALUE COMMENT='partition')"
	stmt, err := New().ParseOneStmt(sql)
	if err != nil {
		t.Fatal(err)
	}
	create := stmt.(*ast.CreateTableStmt)
	checkSpan := func(name string, node ast.Node, want string) {
		t.Helper()
		span := node.Span()
		if !span.Within(len(sql)) || sql[span.Start:span.End] != want {
			t.Errorf("%s span %v selects %q, want %q", name, span, sql[span.Start:span.End], want)
		}
	}
	checkSpan("engine option", create.Options[0], "ENGINE=InnoDB")
	checkSpan("table comment option", create.Options[1], "COMMENT='table'")
	checkSpan("partition options", create.Partition, "PARTITION BY RANGE (a) (PARTITION p VALUES LESS THAN MAXVALUE COMMENT='partition')")
	definition := create.Partition.Definitions[0]
	checkSpan("partition definition", definition, "PARTITION p VALUES LESS THAN MAXVALUE COMMENT='partition'")
	checkSpan("partition boundary", definition.LessThan[0], "MAXVALUE")
	checkSpan("partition comment option", definition.Options[0], "COMMENT='partition'")
}

func TestAlterTableCharsetOptionSpans(t *testing.T) {
	sql := "ALTER TABLE t CONVERT TO CHARACTER SET utf8 COLLATE utf8_bin"
	stmt, err := New().ParseOneStmt(sql)
	if err != nil {
		t.Fatal(err)
	}
	options := stmt.(*ast.AlterTableStmt).Specs[0].Options
	for i, want := range []string{"utf8", "COLLATE utf8_bin"} {
		span := options[i].Span()
		if !span.Within(len(sql)) || sql[span.Start:span.End] != want {
			t.Errorf("option %d span %v selects %q, want %q", i, span, sql[span.Start:span.End], want)
		}
	}
}

func TestTopLevelStatementSpans(t *testing.T) {
	sqls := []string{
		"ALTER EVENT ev ENABLE",
		"alter function f deterministic",
		"alter procedure p contains sql",
		"ALTER USER IF EXISTS `root`@`localhost` IDENTIFIED BY 'new-password'",
		"alter view v as select * from t",
		"ANALYZE TABLE t",
		"begin",
		"CALL sysdate()",
		"commit",
		"create event ev on schedule at '2020-01-01 00:00:00' do select 1",
		"create function f() returns int return 1",
		"create trigger trg before insert on t for each row set @a=1",
		"CREATE USER `ttt`@`%` REQUIRE X509",
		"CREATE ROLE `test-role`",
		"create view v as select * from t",
		"DO 1",
		"drop event ev",
		"drop function f",
		"drop index a on t",
		"DROP PROCEDURE p",
		"drop table xxx",
		"drop trigger trg",
		"DROP ROLE `role`",
		"DROP USER `root`@`localhost`",
		"flush no_write_to_binlog tables tbl1 with read lock",
		"GRANT 'role1' TO 'user1'@'localhost'",
		"kill 23123",
		"PURGE BINARY LOGS",
		"return 1",
		"REVOKE 'role1' FROM 'user1'@'localhost'",
		"SET DEFAULT ROLE administrator TO `joe`@`10.0.0.1`",
		"SET PASSWORD = 'password'",
		"set role `role1`",
		"SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ",
		"TRUNCATE TABLE t1",
		"BACKUP LOCK t",
		"SHUTDOWN",
	}
	for _, sql := range sqls {
		stmt, err := New().ParseOneStmt(sql)
		if err != nil {
			t.Errorf("parse %q: %v", sql, err)
			continue
		}
		if !stmt.Span().Valid() {
			t.Errorf("%q -> %T has invalid span %v", sql, stmt, stmt.Span())
		}
	}
}

func TestProgramReductionSpans(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want map[string][]string
	}{
		{name: "declare and set", body: "BEGIN DECLARE a INT DEFAULT 1; SET a=2; END", want: map[string][]string{
			"*ast.DeclareStmt": {"DECLARE a INT DEFAULT 1"}, "*ast.SetStmt": {"SET a=2"}, "*ast.VariableAssignment": {"a=2"},
		}},
		{name: "if", body: "BEGIN IF a THEN SET a=1; ELSEIF b THEN SET a=2; END IF; END", want: map[string][]string{
			"*ast.IfStmt": {"IF a THEN SET a=1; ELSEIF b THEN SET a=2; END IF"}, "*ast.IfClause": {"IF a THEN SET a=1;", "ELSEIF b THEN SET a=2;"},
		}},
		{name: "empty if", body: "BEGIN IF a THEN ELSEIF b THEN END IF; END", want: map[string][]string{
			"*ast.IfClause": {"IF a THEN", "ELSEIF b THEN"},
		}},
		{name: "case", body: "BEGIN CASE a WHEN 1 THEN SET a=1; END CASE; END", want: map[string][]string{
			"*ast.CaseStmt": {"CASE a WHEN 1 THEN SET a=1; END CASE"}, "*ast.CaseWhenClause": {"WHEN 1 THEN SET a=1;"},
		}},
		{name: "while label", body: "BEGIN lbl: WHILE a DO SET a=1; END WHILE lbl; END", want: map[string][]string{
			"*ast.WhileStmt": {"lbl: WHILE a DO SET a=1; END WHILE lbl"},
		}},
		{name: "repeat", body: "BEGIN REPEAT SET a=1; UNTIL a END REPEAT; END", want: map[string][]string{
			"*ast.RepeatStmt": {"REPEAT SET a=1; UNTIL a END REPEAT"},
		}},
		{name: "loop", body: "BEGIN lbl: LOOP ITERATE lbl; LEAVE lbl; END LOOP lbl; END", want: map[string][]string{
			"*ast.LoopStmt": {"lbl: LOOP ITERATE lbl; LEAVE lbl; END LOOP lbl"}, "*ast.IterateStmt": {"ITERATE lbl"}, "*ast.LeaveStmt": {"LEAVE lbl"},
		}},
		{name: "for", body: "BEGIN FOR i IN 1 .. 2 DO SET a=1; END FOR; END", want: map[string][]string{
			"*ast.ForStmt": {"FOR i IN 1 .. 2 DO SET a=1; END FOR"},
		}},
		{name: "cursor", body: "BEGIN DECLARE cur CURSOR FOR SELECT 1; OPEN cur; FETCH NEXT FROM cur INTO a; CLOSE cur; END", want: map[string][]string{
			"*ast.DeclareCursorStmt": {"DECLARE cur CURSOR FOR SELECT 1"}, "*ast.OpenStmt": {"OPEN cur"}, "*ast.FetchStmt": {"FETCH NEXT FROM cur INTO a"}, "*ast.CloseStmt": {"CLOSE cur"},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sql := "/* 源码 */ CREATE PROCEDURE p() " + tc.body
			stmt, err := New().ParseOneStmt(sql)
			if err != nil {
				t.Fatal(err)
			}
			if got, err := stmt.Span().Slice(sql); err != nil || got != "CREATE PROCEDURE p() "+tc.body {
				t.Fatalf("procedure span = %q, error=%v", got, err)
			}
			var nodes []ast.Node
			if _, ok := stmt.Accept(expressionSpanCollector{nodes: &nodes}); !ok {
				t.Fatal("procedure visitor failed")
			}
			checkNodeSpanTexts(t, sql, nodes, tc.want)
			checkNodeSpanTexts(t, sql, nodes, map[string][]string{"*ast.BeginEndStmt": {tc.body}})
		})
	}
}

func TestSetVariableReductionSpans(t *testing.T) {
	for _, tc := range []struct{ assignment, target, value string }{
		{assignment: "a=1", target: "a", value: "1"},
		{assignment: "r.a=1", target: "r.a", value: "1"},
		{assignment: "GLOBAL a=1", target: "GLOBAL a", value: "1"},
		{assignment: "SESSION a=1", target: "SESSION a", value: "1"},
		{assignment: "LOCAL a=1", target: "LOCAL a", value: "1"},
		{assignment: "@`名`=1", target: "@`名`", value: "1"},
		{assignment: "@@session.a=1", target: "@@session.a", value: "1"},
		{assignment: "NAMES utf8mb4", value: "utf8mb4"},
		{assignment: "NAMES utf8mb4 COLLATE utf8mb4_bin", value: "utf8mb4"},
		{assignment: "NAMES DEFAULT", value: "DEFAULT"},
	} {
		t.Run(tc.assignment, func(t *testing.T) {
			sql := "SET " + tc.assignment
			stmt, err := New().ParseOneStmt(sql)
			if err != nil {
				t.Fatal(err)
			}
			assignment := stmt.(*ast.SetStmt).Variables[0]
			if got, err := assignment.Span().Slice(sql); err != nil || got != tc.assignment {
				t.Fatalf("assignment span = %q, error=%v", got, err)
			}
			if tc.target != "" {
				if got, err := assignment.Target.Span().Slice(sql); err != nil || got != tc.target {
					t.Fatalf("target span = %q, error=%v", got, err)
				}
			}
			if got, err := assignment.Value.Span().Slice(sql); err != nil || got != tc.value {
				t.Fatalf("value span = %q, error=%v", got, err)
			}
		})
	}
}

func TestDatabaseReferenceStatementSpans(t *testing.T) {
	for _, sql := range []string{
		"use db",
		"explain t", "describe t c", "explain select 1", "explain extended select 1",
		"explain partitions select 1", "explain for connection 1",
		"explain format=json for connection 1", "explain format=json select 1",
		"explain analyze select 1", "explain analyze format=json select 1",
		"analyze select 1", "analyze format=json select 1",
		"show tables in db where 1", "show create view t", "show create database db",
		"show create table t", "show create procedure p", "show create function f",
		"show create trigger tr", "show create event ev", "show create user 'test'@'host'",
		"show grants", "show grants for 'test'@'host'", "show master status",
		"show replica status", "show replicas", "show binary log status", "show binary logs",
		"show full processlist", "show profiles", "show profile cpu for query 1 limit 2",
		"show privileges", "show warnings limit 2", "show errors limit 2",
	} {
		t.Run(sql, func(t *testing.T) {
			input := "/* prefix */ " + sql + " ;"
			stmt := mustParseOne(t, input)
			got, err := stmt.Span().Slice(input)
			if err != nil || got != sql {
				t.Fatalf("span=%v selects %q, err=%v, want %q", stmt.Span(), got, err, sql)
			}
		})
	}
}

func TestShowIndexKeepsOneDatabaseSource(t *testing.T) {
	const sql = "show index from t from `a.b`"
	show := mustParseOne(t, sql).(*ast.ShowStmt)
	if show.Database.Present() || len(show.Table.Name.Parts) != 2 {
		t.Fatalf("SHOW INDEX duplicated or lost database source: %+v", show)
	}
	ident := show.Table.Name.Parts[0]
	if ident.Name != "a.b" || ident.Raw != "`a.b`" {
		t.Fatalf("SHOW INDEX database=%+v", ident)
	}
	if got, err := ident.Span.Slice(sql); err != nil || got != "`a.b`" {
		t.Fatalf("database span selects %q, err=%v", got, err)
	}
}

func TestDatabaseReferenceMultiStatementSpans(t *testing.T) {
	const input = "/* start */ use dba; show events in dbb where 1; explain for schema dbc select 1;"
	stmts, _, err := New().ParseWithWarnings(input)
	if err != nil || len(stmts) != 3 {
		t.Fatalf("statements=%d err=%v", len(stmts), err)
	}
	for i, want := range []string{"use dba", "show events in dbb where 1", "explain for schema dbc select 1"} {
		got, err := stmts[i].Span().Slice(input)
		if err != nil || got != want || int(stmts[i].Span().Start) != strings.Index(input, want) {
			t.Fatalf("statement %d selects %q, err=%v, want %q", i, got, err, want)
		}
	}
}

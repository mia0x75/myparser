package myparser

import (
	"bytes"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/mysql"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

func TestMTRPhase5RoundTrip(t *testing.T) {
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
		{"create sequence s increment = -2 start = +3 minvalue = -4 maxvalue = 9 cache+1 cycle", "CREATE SEQUENCE `s` INCREMENT BY -2 START WITH 3 MINVALUE -4 MAXVALUE 9 CACHE 1 CYCLE"},
		{"create sequence s increment by -9223372036854775808", "CREATE SEQUENCE `s` INCREMENT BY -9223372036854775808"},
		{"create sequence s cache -1 minvalue -1.5", "CREATE SEQUENCE `s` CACHE -1 MINVALUE -1.5"},
		{"create sequence s as bigint unsigned start 18446744073709551615 maxvalue 18446744073709551615", "CREATE SEQUENCE `s` AS BIGINT UNSIGNED START WITH 18446744073709551615 MAXVALUE 18446744073709551615"},
		{"create sequence s minvalue -18446744073709551615", "CREATE SEQUENCE `s` MINVALUE -18446744073709551615"},
		{"select 1,2,3,4 into r.a, `r.a`, `@a`, @a", "SELECT 1,2,3,4 INTO `r`.`a`, `r.a`, `@a`, @`a`"},
		{"select 1,2 into @'a.b', @`@a` from t", "SELECT 1,2 INTO @`a.b`, @`@a` FROM `t`"},
		{"values row(1),row(2)", "VALUES ROW(1), ROW(2)"},
		{"values (1),(2)", "VALUES (1), (2)"},
		{"select cast('x' as varchar(10))", "SELECT CAST('x' AS VARCHAR(10))"},
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
		{"create table t(a int default (1+2), b int default abs(-1))", "CREATE TABLE `t` (`a` INT DEFAULT (1+2),`b` INT DEFAULT ABS(-1))"},
		{"create table t(a int default (select 1))", "CREATE TABLE `t` (`a` INT DEFAULT (SELECT 1))"},
		{"create table g(a point default (point(1,2)), spatial key idx(a))", "CREATE TABLE `g` (`a` POINT DEFAULT (point(1, 2)),SPATIAL `idx`(`a`))"},
		{"create table g(a point default point(1,2))", "CREATE TABLE `g` (`a` POINT DEFAULT point(1, 2))"},
		{"drop temporary table if exists t1,t2", "DROP TEMPORARY TABLE IF EXISTS `t1`, `t2`"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			RunRestoreTest(t, tc.src, tc.restored, true)
		})
	}
}

func TestMTRPhase5RoutineAndViewRoundTrip(t *testing.T) {
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
			if _, ok := stmt.Accept(extensionVisitor{}); !ok {
				t.Fatal("visitor aborted")
			}
		})
	}
}

func TestMTRPhase5ServerRejectedSyntax(t *testing.T) {
	// These boundaries were checked on the pinned MySQL 8.4.13/MariaDB 12.3.3
	// servers. Runtime definition errors (e.g. SEQUENCE 4085 or DEFAULT 1901)
	// are deliberately covered by the positive tests instead.
	assertExtensionParseErrors(t, []string{
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
		"create sequence s increment by = 1",
		"create sequence s start with = 1",
		"create sequence s increment 9223372036854775808",
		"create sequence s increment +18446744073709551615",
		"create sequence s cache 9223372036854775808",
		"create sequence s start -9223372036854775809",
		"create sequence s start 1.5",
		"create sequence s start 0x1",
		"create sequence s minvalue 0x1",
		"create sequence s as char start 1",
		"create sequence s as int(10) start 1",
		"values row(1),(2)",
		"values (1),row(2)",
		"create table t(a int default 1+2)",
		"create table t(a int default abs(-1)+2)",
		"create table t(a int default -abs(-1))",
		"create table t(a timestamp default now)",
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
		"alter event ev",
		"alter definer=current_user event ev",
		"create table t(a varchar(10) collate ascii_bin generated always as (null) collate ascii_bin)",
		"create table t(a varchar(10) generated always as (null) collate ascii_bin collate ascii_bin)",
		"create table t(a double(2147483648,0))",
		"create table t(a double(1,2147483648))",
		"select 'unterminated",
	})
}

func TestMTRPhase5BoundaryControls(t *testing.T) {
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
		"alter event ev enable",
		"alter event ev comment 'x'",
		"alter event ev do select 1",
		"create table t(a varchar(10) collate ascii_bin collate ascii_bin)",
		"create table t(a varchar(10) collate ascii_bin generated always as (null))",
		"create table t(a double(2147483647,0))",
		"create table t(a double(1,2147483647))",
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

	// Both servers reject duplicate INTO, including MySQL's dedicated error.
	if _, err := New().ParseOneStmt("select 1 into @a from t into @b", "", ""); err == nil {
		t.Fatal("duplicate INTO was accepted")
	}
	before := mustParseOne(t, "select 1 into @v from t").(*ast.SelectStmt)
	trailing := mustParseOne(t, "select 1 from t into @v").(*ast.SelectStmt)
	if before.Into == nil || !before.Into.BeforeFrom || trailing.Into == nil || trailing.Into.BeforeFrom {
		t.Fatalf("INTO position was lost: before=%#v trailing=%#v", before.Into, trailing.Into)
	}
}

func TestMTRPhase5UnterminatedStringSQLModes(t *testing.T) {
	for _, tc := range []struct {
		mode mysql.SQLMode
		sql  string
	}{
		{sql: "select 'unterminated"},
		{mode: mysql.ModeNoBackslashEscapes, sql: "select 'unterminated"},
		{mode: mysql.ModeANSIQuotes, sql: `create procedure p() update t set c=''"`},
	} {
		p := New()
		p.SetSQLMode(tc.mode)
		if _, err := p.ParseOneStmt(tc.sql, "", ""); err == nil {
			t.Fatalf("expected unterminated string error for %q in mode %d", tc.sql, tc.mode)
		}
	}
}

func TestMTRPhase5WeightStringASTAndVisitors(t *testing.T) {
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
	_, ok := params.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
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
	assertMTRPhase5VisitorControls(t, params, isValueOrLiteral)
}

func TestMTRPhase5VariableTargetASTAndVisitors(t *testing.T) {
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
			_, ok := sel.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
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
			assertMTRPhase5VisitorControls(t, sel.Into, func(n ast.Node) bool { _, ok := n.(*ast.VariableTarget); return ok })
		})
	}
	proc := mustParseOne(t, "create procedure p() begin declare `a.b` int; declare c cursor for select 1; fetch c into `a.b`; end").(*ast.CreateProcedureStmt)
	fetch := proc.Body.(*ast.BeginEndStmt).List[2].(*ast.FetchStmt)
	if len(fetch.Targets) != 1 || len(fetch.Targets[0].Name.Parts) != 1 || fetch.Targets[0].Name.Parts[0].Name != "a.b" || fetch.Targets[0].Kind == ast.VariableTargetUser {
		t.Fatalf("FETCH target was lost: %#v", fetch)
	}
	assertMTRPhase5VisitorControls(t, fetch, func(n ast.Node) bool { _, ok := n.(*ast.VariableTarget); return ok })
}

func TestMTRPhase5SequenceASTAndVisitors(t *testing.T) {
	stmt := mustParseOne(t, "create sequence s as bigint unsigned increment by -9223372036854775808 start 18446744073709551615 minvalue -18446744073709551615").(*ast.CreateSequenceStmt)
	if stmt.Options[0].Tp != ast.SequenceOptionAs || stmt.Options[0].Type.Keyword() != "BIGINT" || !stmt.Options[0].Type.Unsigned {
		t.Fatalf("sequence type was lost: %#v", stmt.Options[0])
	}
	start, isLiteral := stmt.Options[2].Value.(*ast.LiteralExpr)
	if stmt.Options[1].Val != math.MinInt64 || !isLiteral || start.Kind != pkast.LiteralInteger || start.Value != "18446744073709551615" {
		t.Fatalf("sequence integer boundaries were lost: %#v", stmt.Options)
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
	if !ok || values != 3 || !strings.Contains(restoreStmt(t, stmt), "START WITH 9 MINVALUE -9") {
		t.Fatalf("sequence visitor: ok=%v values=%d sql=%s", ok, values, restoreStmt(t, stmt))
	}
	assertMTRPhase5VisitorControls(t, stmt, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.LiteralExpr:
			return true
		}
		return false
	})
}

func TestMTRPhase5RowDefaultAndDerivedAST(t *testing.T) {
	proc := mustParseOne(t, "create procedure p() begin declare r row(a int,b varchar(8)) default (1,'x'); declare x type of db.t.a; declare y row type of db.t; set r.a=2; end").(*ast.CreateProcedureStmt)
	stmts := proc.Body.(*ast.BeginEndStmt).List
	row := stmts[0].(*ast.DeclareStmt)
	if len(row.RowType) != 2 || len(row.RowType[1].Type.TypeCore().Modifiers) != 1 ||
		row.RowType[1].Type.TypeCore().Modifiers[0].(*ast.LiteralExpr).Value != "8" || row.Default == nil {
		t.Fatalf("ROW type was lost: %#v", row)
	}
	ref := stmts[1].(*ast.DeclareStmt)
	if ref.TypeRef == nil || ref.RowTypeRef || len(ref.TypeRef.Parts) != 3 || ref.TypeRef.Parts[0].Name != "db" || ref.TypeRef.Parts[1].Name != "t" || ref.TypeRef.Parts[2].Name != "a" {
		t.Fatalf("TYPE OF reference was lost: %#v", ref)
	}
	if ref := stmts[2].(*ast.DeclareStmt); ref.TypeRef == nil || !ref.RowTypeRef {
		t.Fatalf("ROW TYPE OF reference was lost: %#v", ref)
	}
	assign := stmts[3].(*ast.SetStmt).Variables[0]
	if assign.Target == nil || len(assign.Target.Name.Parts) != 2 || assign.Target.Name.Parts[0].Name != "r" || assign.Target.Name.Parts[1].Name != "a" {
		t.Fatalf("ROW assignment was lost: %#v", assign)
	}
	assertMTRPhase5VisitorControls(t, proc, func(n ast.Node) bool { _, ok := n.(*ast.ColumnDef); return ok })

	create := mustParseOne(t, "create table g(a point default (point(1,2)), spatial key idx(a))").(*ast.CreateTableStmt)
	if _, ok := create.Columns[0].(*ast.ColumnDef).Options[0].Expr.(*ast.ParenthesesExpr); !ok || create.Constraints[0].(*ast.Constraint).Tp != ast.ConstraintSpatial {
		t.Fatalf("DEFAULT or SPATIAL constraint was lost: %#v", create)
	}
	assertMTRPhase5VisitorControls(t, create, isValueOrLiteral)

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
	assertMTRPhase5VisitorControls(t, sel, isValueOrLiteral)
	const cycleSQL = "with recursive c(a) as (select 1) cycle a restrict select * from c"
	cte := mustParseOne(t, cycleSQL).(*ast.SelectStmt)
	aStart := strings.Index(cycleSQL, "cycle a") + len("cycle ")
	wantCycle := []lex.Ident{{Name: "a", Raw: "a", Span: loc.Span{Start: loc.ByteOffset(aStart), End: loc.ByteOffset(aStart + 1)}}}
	with := cte.With.(*ast.WithClause)
	if !with.IsRecursive || !reflect.DeepEqual(with.CycleColumns, wantCycle) {
		t.Fatalf("CYCLE clause was lost: %#v", cte.With)
	}

	for _, withCheck := range []bool{false, true} {
		src := "create view v as select a from t"
		if withCheck {
			src += " with check option"
		}
		view := mustParseOne(t, src).(*ast.CreateViewStmt)
		if view.HasCheckOption != withCheck || strings.Contains(restoreStmt(t, view), " CHECK OPTION") != withCheck {
			t.Fatalf("CHECK OPTION presence was lost: %#v", view)
		}
	}
}

func TestMTRPhase5CastVarcharAndValuesSetAST(t *testing.T) {
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
	if _, ok := set.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
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

	insert := mustParseOne(t, "insert into t values (1),(2) on duplicate key update a=values(a)").(*ast.InsertStmt)
	if insert.Query != nil || len(insert.Values) != 2 {
		t.Fatalf("INSERT VALUES changed shape: %#v", insert)
	}
	if _, ok := insert.OnDuplicate[0].Expr.(*ast.ValuesExpr); !ok {
		t.Fatalf("VALUES(column) changed shape: %#v", insert.OnDuplicate[0].Expr)
	}
}

func TestMTRPhase5ExpressionFlags(t *testing.T) {
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
}

func assertMTRPhase5VisitorControls(t *testing.T, node ast.Node, isChild func(ast.Node) bool) {
	t.Helper()
	_, ok := node.Accept(extensionVisitor{leave: func(n ast.Node) (ast.Node, bool) {
		return n, !isChild(n)
	}})
	if ok {
		t.Fatal("visitor abort was not propagated")
	}
	entered := 0
	_, ok = node.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		entered++
		return n, true
	}})
	if !ok || entered != 1 {
		t.Fatalf("skipChildren: ok=%v entered=%d", ok, entered)
	}
}

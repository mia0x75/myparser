package parser_test

import (
	"math"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	pkast "github.com/mia0x75/parserkit/ast"
)

func TestDDLSerialAndIndexRoundTrip(t *testing.T) {
	for _, tc := range []struct{ src, restored string }{
		{"create table t(a serial comment 'x', b int)", "CREATE TABLE `t` (`a` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT UNIQUE KEY COMMENT 'x',`b` INT)"},
		{"alter table t add a serial", "ALTER TABLE `t` ADD COLUMN `a` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT UNIQUE KEY"},
		{"create index i type btree on t(a)", "CREATE INDEX `i` ON `t` (`a`) USING BTREE"},
		{"create index i on t(a) type hash", "CREATE INDEX `i` ON `t` (`a`) USING HASH"},
		{"create index i using rtree on t(a)", "CREATE INDEX `i` ON `t` (`a`) USING RTREE"},
	} {
		t.Run(tc.src, func(t *testing.T) {
			runRestoreTest(t, tc.src, tc.restored, true)
		})
	}
	assertParseErrors(t, []string{
		"create table t(a serial as (1))", "alter table t add a serial as (1)",
		"create index i type on t(a)", "create index i type unknown on t(a)",
		"create table t(a int, key type btree (a))",
	})
}

func TestDDLCharsetRoundTrip(t *testing.T) {
	for _, tc := range []struct{ src, restored string }{
		{"alter table t convert to character set default", "ALTER TABLE `t` CONVERT TO CHARACTER SET DEFAULT"},
		{"alter table t convert to character set default collate latin1_bin", "ALTER TABLE `t` CONVERT TO CHARACTER SET DEFAULT COLLATE latin1_bin"},
		{"alter table t convert to charset latin1", "ALTER TABLE `t` CONVERT TO CHARACTER SET latin1"},
		{"alter table t default charset latin1 collate latin1_bin", "ALTER TABLE `t` DEFAULT CHARACTER SET = latin1 DEFAULT COLLATE = latin1_bin"},
	} {
		t.Run(tc.src, func(t *testing.T) {
			runRestoreTest(t, tc.src, tc.restored, true)
		})
	}
	assertParseErrors(t, []string{
		"alter table t convert to character set", "alter table t convert to character set default collate",
	})
}

func TestDDLYearColumnTypeRoundTrip(t *testing.T) {
	for _, tc := range []struct{ src, restored string }{
		{"create table t(a year(2), b year(5))", "CREATE TABLE `t` (`a` YEAR(2),`b` YEAR(5))"},
	} {
		t.Run(tc.src, func(t *testing.T) {
			runRestoreTest(t, tc.src, tc.restored, true)
		})
	}
	assertParseErrors(t, []string{
		"create table t(a year())",
	})
}

func TestDDLNationalStringTypes(t *testing.T) {
	// National string types always use the deliberately unsupported utf8mb3.
	assertParseErrors(t, []string{
		"create table t(a nchar(10))",
		"create table t(a national char(10))",
		"create table t(a national character(10))",
		"create table t(a national varchar(10))",
		"create table t(a nvarchar(10))",
		"create table t(a nchar varchar(10))",
		"create table t(a national character varying(10))",
		"create table t(a nchar varying(10))",
		"alter table t add a national char(10)",
	})
	for _, src := range []string{
		"create table t(a char(10), b varchar(10))",
		"create table national(a int)",
	} {
		t.Run(src, func(t *testing.T) {
			stmt := mustParseOne(t, src)
			mustParseOne(t, restoreStmt(t, stmt))
		})
	}
}

func TestDDLSequenceRoundTrip(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"create sequence s increment = -2 start = +3 minvalue = -4 maxvalue = 9 cache+1 cycle", "CREATE SEQUENCE `s` INCREMENT BY -2 START WITH 3 MINVALUE -4 MAXVALUE 9 CACHE 1 CYCLE"},
		{"create sequence s increment by -9223372036854775808", "CREATE SEQUENCE `s` INCREMENT BY -9223372036854775808"},
		{"create sequence s cache -1 minvalue -1.5", "CREATE SEQUENCE `s` CACHE -1 MINVALUE -1.5"},
		{"create sequence s as bigint unsigned start 18446744073709551615 maxvalue 18446744073709551615", "CREATE SEQUENCE `s` AS BIGINT UNSIGNED START WITH 18446744073709551615 MAXVALUE 18446744073709551615"},
		{"create sequence s minvalue -18446744073709551615", "CREATE SEQUENCE `s` MINVALUE -18446744073709551615"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			runRestoreTest(t, tc.src, tc.restored, true)
		})
	}
}

func TestDDLColumnDefaultRoundTrip(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"create table t(a int default (1+2), b int default abs(-1))", "CREATE TABLE `t` (`a` INT DEFAULT (1+2),`b` INT DEFAULT ABS(-1))"},
		{"create table t(a int default (select 1))", "CREATE TABLE `t` (`a` INT DEFAULT (SELECT 1))"},
		{"create table g(a point default (point(1,2)), spatial key idx(a))", "CREATE TABLE `g` (`a` POINT DEFAULT (point(1, 2)),SPATIAL `idx`(`a`))"},
		{"create table g(a point default point(1,2))", "CREATE TABLE `g` (`a` POINT DEFAULT point(1, 2))"},
		{"drop temporary table if exists t1,t2", "DROP TEMPORARY TABLE IF EXISTS `t1`, `t2`"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			runRestoreTest(t, tc.src, tc.restored, true)
		})
	}
}

func TestDDLSequenceSyntaxErrors(t *testing.T) {
	assertParseErrors(t, []string{
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
	})
}

func TestDDLDefaultSyntaxErrors(t *testing.T) {
	assertParseErrors(t, []string{
		"create table t(a int default 1+2)",
		"create table t(a int default abs(-1)+2)",
		"create table t(a int default -abs(-1))",
		"create table t(a timestamp default now)",
	})
}

func TestDDLEventAndTableBoundaryControls(t *testing.T) {
	// These retain legal query tails, runtime errors, and one-server dialect
	// differences adjacent to the rejected boundaries above.
	for _, src := range []string{
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
			if _, ok := stmt.Accept(syntaxVisitor{}); !ok {
				t.Fatal("visitor aborted")
			}
		})
	}
}

func TestDDLSerialASTAndVisitors(t *testing.T) {
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
	_, ok := stmt.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
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
	_, ok = stmt.Accept(syntaxVisitor{leave: func(n ast.Node) (ast.Node, bool) {
		_, isLiteral := n.(*ast.LiteralExpr)
		return n, !isLiteral
	}})
	if ok {
		t.Fatal("SERIAL visitor abort was not propagated")
	}
}

func TestDDLDefaultAndSpatialAST(t *testing.T) {
	create := mustParseOne(t, "create table g(a point default (point(1,2)), spatial key idx(a))").(*ast.CreateTableStmt)
	if _, ok := create.Columns[0].(*ast.ColumnDef).Options[0].Expr.(*ast.ParenthesesExpr); !ok || create.Constraints[0].(*ast.Constraint).Tp != ast.ConstraintSpatial {
		t.Fatalf("DEFAULT or SPATIAL constraint was lost: %#v", create)
	}
	assertVisitorControls(t, create, isValueOrLiteral)
}

func TestDDLSequenceASTAndVisitors(t *testing.T) {
	stmt := mustParseOne(t, "create sequence s as bigint unsigned increment by -9223372036854775808 start 18446744073709551615 minvalue -18446744073709551615").(*ast.CreateSequenceStmt)
	if stmt.Options[0].Tp != ast.SequenceOptionAs || stmt.Options[0].Type.Keyword() != "BIGINT" || !stmt.Options[0].Type.Unsigned {
		t.Fatalf("sequence type was lost: %#v", stmt.Options[0])
	}
	start, isLiteral := stmt.Options[2].Value.(*ast.LiteralExpr)
	if stmt.Options[1].Val != math.MinInt64 || !isLiteral || start.Kind != pkast.LiteralInteger || start.Value != "18446744073709551615" {
		t.Fatalf("sequence integer boundaries were lost: %#v", stmt.Options)
	}
	values := 0
	_, ok := stmt.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
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
	assertVisitorControls(t, stmt, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.LiteralExpr:
			return true
		}
		return false
	})
}

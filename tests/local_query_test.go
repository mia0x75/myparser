package parser_test

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
)

// TestLocalWindowIdentifiers checks that window-related keywords stay valid
// identifiers in column, table and window-definition positions, with window
// functions enabled and disabled.
func TestLocalWindowIdentifiers(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		name := "window disabled"
		if enabled {
			name = "window enabled"
		}
		t.Run(name, func(t *testing.T) {
			for _, tc := range []struct{ src, restored string }{
				{"select id, window from door", "SELECT `id`,`window` FROM `door`"},
				{"select rank from t where window='x' order by rank", "SELECT `rank` FROM `t` WHERE `window`='x' ORDER BY `rank`"},
				{"select t.window, db.t.rank from t", "SELECT `t`.`window`,`db`.`t`.`rank` FROM `t`"},
				{"create table door(id int, window varchar(10), rank int)", "CREATE TABLE `door` (`id` INT,`window` VARCHAR(10),`rank` INT)"},
			} {
				t.Run(tc.src, func(t *testing.T) {
					localRunRestoreTest(t, tc.src, tc.restored, enabled)
				})
			}
		})
	}
	localRunRestoreTest(t, "select rank() over w as rank from t window w as (order by a)",
		"SELECT RANK() OVER `w` AS `rank` FROM `t` WINDOW `w` AS (ORDER BY `a`)", true)
}

// TestLocalValuesAndDeleteAST covers quantified VALUES row subqueries and
// multi-table DELETE ... USING tail clauses.
func TestLocalValuesAndDeleteAST(t *testing.T) {
	for _, quantifier := range []string{"all", "any", "some"} {
		t.Run(quantifier, func(t *testing.T) {
			src := "select 1 < " + quantifier + "(values row(2),row(cast(3 as decimal))) from dual"
			stmt := mustParseOne(t, src).(*ast.SelectStmt)
			cmp := stmt.Fields.Items[0].Expr.(*ast.CompareSubqueryExpr)
			sq := cmp.R.(*ast.SubqueryExpr)
			values, ok := sq.Query.(*ast.ValuesStmt)
			if !ok || len(values.Rows) != 2 {
				t.Fatalf("quantified VALUES source was lost: %#v", sq.Query)
			}
			firstRow, rowOK := values.Rows[0].(*ast.RowExpr)
			if !rowOK || firstRow.OmitRowKeyword || cmp.All != (quantifier == "all") {
				t.Fatalf("quantified VALUES query was lost: %#v", cmp)
			}
			localRunRestoreTest(t, src, restoreStmt(t, stmt), true)
			assertVisitorControls(t, cmp, isValueOrLiteral)
		})
	}
	stmt := mustParseOne(t, "with c as (select 1 as a) delete from t using t,c where t.a=c.a order by t.a limit 1").(*ast.DeleteStmt)
	if stmt.With == nil || !stmt.IsMultiTable || stmt.BeforeFrom || stmt.OrderBy == nil || stmt.Limit == nil {
		t.Fatalf("DELETE USING tail was lost: %#v", stmt)
	}
	localRunRestoreTest(t, stmt.Text(), restoreStmt(t, stmt), true)
	_, ok := stmt.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch n := n.(type) {
		case *ast.LiteralExpr:
			return replaceLiteral(n, "9"), false
		}
		return n, false
	}})
	if !ok || !strings.HasSuffix(restoreStmt(t, stmt), "LIMIT 9") {
		t.Fatal("DELETE USING limit visitor replacement was lost")
	}
	assertVisitorControls(t, stmt, func(n ast.Node) bool { _, ok := n.(*ast.Limit); return ok })
}

// TestLocalSyntaxBoundaries pins the statement shapes the parser must reject
// around periods, partition counts, ROW TYPE OF, BACKUP, ANALYZE, DELETE and
// character sets.
func TestLocalSyntaxBoundaries(t *testing.T) {
	assertParseErrors(t, []string{
		"create table t(s date,e date,period for p())",
		"create table t(s date,e date,period for p(s))",
		"create table t(s date,e date,period for p(s,e,s))",
		"create table t(s date,e date,period for p(t.s,e))",
		"create table t(s date,e date,period for p(s,db.t.e))",
		"create table t(s date,e date,period for p(s+1,e))",
		"create table t(a int,primary key((a+1)))",
		"create table t(a int,foreign key((a+1)) references u(a))",
		"create table t(a int,foreign key(a) references u((a+1)))",
		"create table t(a int,fulltext key((a+1)))",
		"create table t(a int,spatial key((a+1)))",
		"alter table t partition by key(a) (partition p values less than (1))",
		"alter table t partition by hash(a) partitions -1",
		"alter table t partition by hash(a) partitions 1.5",
		"create function f() returns row type of t",
		"create function f() returns row type of t.c.x return 1",
		"backup lock", "backup unlock t", "analyze tables", "analyze tables t,",
		"delete from t using t limit", "delete from t using t order by limit 1",
		"select 1 < all(values row(2),row())",
		"set names cp1250_latin2",
		"create table t(a int) charset cp1250_latin2",
		"create table t(a varchar(10) character set cp1250_latin2)",
		"select convert('x' using cp1250_latin2)",
		"set character set unknown_charset",
		"set character set cp1250_latin2 collate latin2_bin",
		"create procedure p() begin declare select int; end",
		"create procedure p() select: loop leave select; end loop",
	})
}

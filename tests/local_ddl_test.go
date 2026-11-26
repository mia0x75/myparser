package parser_test

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/parserkit/lex"
)

// lexIdent builds a decoded-only identifier for visitor replacement nodes.
func lexIdent(name string) lex.Ident {
	return lex.Ident{Name: name}
}

// localQualifiedName builds a syntax-only qualified name with invalid spans,
// mirroring how the production builder sets name spans from parts.
func localQualifiedName(name string) ast.QualifiedName {
	return ast.QualifiedName{Parts: []lex.Ident{lexIdent(name)}}
}

// TestLocalParenthesizedCTAS covers CREATE TABLE ... AS SELECT variants and
// rejects the forms that are not parenthesized CTAS.
func TestLocalParenthesizedCTAS(t *testing.T) {
	for _, tc := range []struct{ src, restored string }{
		{"create table t (select 1)", "CREATE TABLE `t`  AS (SELECT 1)"},
		{"create table t (select 1) union (select 2)", "CREATE TABLE `t`  AS (SELECT 1) UNION (SELECT 2)"},
		{"create table t (select 1 union select 2)", "CREATE TABLE `t`  AS (SELECT 1 UNION SELECT 2)"},
		{"create table t (with recursive c(a) as (select 1 union all select a+1 from c where a<3) select a from c)",
			"CREATE TABLE `t`  AS (WITH RECURSIVE `c` (`a`) AS (SELECT 1 UNION ALL SELECT `a`+1 FROM `c` WHERE `a`<3) SELECT `a` FROM `c`)"},
		{"create temporary table if not exists db.t (select 1)", "CREATE TEMPORARY TABLE IF NOT EXISTS `db`.`t`  AS (SELECT 1)"},
		{"create table t(a int) replace as (select 1)", "CREATE TABLE `t` (`a` INT) REPLACE AS (SELECT 1)"},
		{"create table t engine=innodb as (select 1)", "CREATE TABLE `t`  ENGINE = innodb AS (SELECT 1)"},
		{"create table t (like u)", "CREATE TABLE `t` LIKE `u`"},
	} {
		t.Run(tc.src, func(t *testing.T) {
			localRunRestoreTest(t, tc.src, tc.restored, true)
			stmt := mustParseOne(t, tc.src).(*ast.CreateTableStmt)
			if stmt.Query == nil && stmt.ReferTable == nil {
				t.Fatal("CTAS query was lost")
			}
		})
	}
	assertParseErrors(t, []string{
		"create table t (select)",
		"create table t (with c as (select 1))",
		"create table t (select 1) union",
		"create table t (a int, select 1)",
	})
}

// TestLocalPeriodASTAndVisitors checks PERIOD FOR declarations survive the
// parse and that visitors replace their name and column references.
func TestLocalPeriodASTAndVisitors(t *testing.T) {
	stmt := mustParseOne(t, "create table t(s date,e date,period for p(s,e))").(*ast.CreateTableStmt)
	if len(stmt.Periods) != 1 || stmt.Periods[0].Name.Name != "p" ||
		len(stmt.Periods[0].Start.Parts) != 1 || stmt.Periods[0].Start.Parts[0].Name != "s" ||
		len(stmt.Periods[0].End.Parts) != 1 || stmt.Periods[0].End.Parts[0].Name != "e" {
		t.Fatalf("period declaration was lost: %#v", stmt.Periods)
	}
	columns, periods := 0, 0
	_, ok := stmt.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch x := n.(type) {
		case *ast.PeriodDefinition:
			periods++
			copy := *x
			copy.Name = lexIdent("q")
			copy.Start = localQualifiedName("renamed")
			copy.End = localQualifiedName("renamed")
			return &copy, false
		case *ast.ColumnDef:
			columns++
			copy := *x
			copy.Name = localQualifiedName("renamed")
			return &copy, false
		}
		return n, false
	}})
	if !ok || columns != 2 || periods != 1 || stmt.Periods[0].Name.Name != "q" ||
		!strings.Contains(restoreStmt(t, stmt), "PERIOD FOR `q` (`renamed`,`renamed`)") {
		t.Fatalf("period visitor replacement was lost: ok=%v columns=%d", ok, columns)
	}
	assertVisitorControls(t, stmt, func(n ast.Node) bool { _, ok := n.(*ast.PeriodDefinition); return ok })
	assertVisitorControls(t, stmt.Periods[0], func(n ast.Node) bool { _, ok := n.(*ast.PeriodDefinition); return ok })
}

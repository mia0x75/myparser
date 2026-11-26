package parser_test

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/myparser/parser"
	pkast "github.com/mia0x75/parserkit/ast"
)

func TestManagementDiagnosticsAndKillRoundTrip(t *testing.T) {
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
			runRestoreTest(t, tc.src, tc.restored, true)
		})
	}
	assertParseErrors(t, []string{
		"show warnings where Code=1", "show errors like '%'", "show warnings limit @n",
		"show errors limit 1+1", "show warnings limit -1", "show errors limit 1,",
		"kill query 99999999,99999998", "kill connection", "kill query 1+",
	})
}

func TestManagementTransactionRoundTrip(t *testing.T) {
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
				runRestoreTest(t, src, want, true)
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
				if _, ok := stmt.Accept(syntaxVisitor{}); !ok {
					t.Fatal("transaction visitor aborted")
				}
			})
		}
	}
	runRestoreTest(t, "rollback work to savepoint s", "ROLLBACK TO SAVEPOINT `s`", true)
	assertParseErrors(t, []string{
		"commit and chain release", "rollback work and chain release",
		"commit chain", "rollback and no", "commit no chain",
		"commit release and chain", "rollback to s no release",
		"rollback and chain to savepoint s", "commit work work",
	})
}

func TestManagementBuiltinTableNames(t *testing.T) {
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
					parser := parser.New()
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
							stmt, err := parser.ParseOneStmt(tc.sql)
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
	runRestoreTest(t, "select count(*),cast(1 as signed),substring('abc',1)",
		"SELECT COUNT(*),CAST(1 AS SIGNED),SUBSTRING('abc', 1)", true)
}

func TestManagementDiagnosticsAndKillVisitors(t *testing.T) {
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
			_, ok := stmt.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
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
			assertVisitorControls(t, stmt, isChild)
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
	runRestoreTest(t, "select connection+1,query+1", "SELECT `connection`+1,`query`+1", true)
}

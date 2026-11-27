package parser_test

import (
	"reflect"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/myparser/parser"
)

func TestParseSharedPreparedStatementCore(t *testing.T) {
	stmt := mustParseOne(t, "PREPARE alignment_p FROM 'SELECT ?'").(*ast.PrepareStmt)
	if stmt.Name.Name != "alignment_p" || !stmt.SQLTextPresent || stmt.SQLText != "SELECT ?" || stmt.SQLVar != nil {
		t.Fatalf("PREPARE core = %#v", stmt)
	}
	exec := mustParseOne(t, "EXECUTE alignment_p USING @a").(*ast.ExecuteStmt)
	if exec.Name.Name != "alignment_p" || len(exec.UsingVars) != 1 || exec.SQLExpr != nil {
		t.Fatalf("EXECUTE core = %#v", exec)
	}
	dealloc := mustParseOne(t, "DEALLOCATE PREPARE alignment_p").(*ast.DeallocateStmt)
	if dealloc.Name.Name != "alignment_p" {
		t.Fatalf("DEALLOCATE core = %#v", dealloc)
	}
}

func TestParseSharedTransactionCore(t *testing.T) {
	for _, tc := range []struct {
		sql  string
		want ast.StmtNode
	}{
		{"BEGIN", &ast.BeginStmt{}},
		{"COMMIT WORK", &ast.CommitStmt{}},
		{"ROLLBACK", &ast.RollbackStmt{}},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			stmt := mustParseOne(t, tc.sql)
			if reflect.TypeOf(stmt) != reflect.TypeOf(tc.want) {
				t.Fatalf("statement = %T, want %T", stmt, tc.want)
			}
		})
	}
	save := mustParseOne(t, "SAVEPOINT alignment_s").(*ast.SavepointStmt)
	rollback := mustParseOne(t, "ROLLBACK TO SAVEPOINT alignment_s").(*ast.RollbackStmt)
	release := mustParseOne(t, "RELEASE SAVEPOINT alignment_s").(*ast.ReleaseSavepointStmt)
	if save.Name.Name != "alignment_s" || rollback.SavepointName.Name != save.Name.Name || release.Name.Name != save.Name.Name {
		t.Fatalf("savepoint names = %#v / %#v / %#v", save, rollback, release)
	}
}

func TestParseSharedGrantCore(t *testing.T) {
	stmt := mustParseOne(t, "GRANT SELECT ON alignment_t TO alignment_user WITH GRANT OPTION").(*ast.GrantStmt)
	if len(stmt.Privs) != 1 || stmt.Privs[0].Priv != mysql.SelectPriv || !stmt.WithGrant ||
		stmt.Object == nil || len(stmt.Object.Parts) != 1 || len(stmt.Users) != 1 {
		t.Fatalf("GRANT core = %#v", stmt)
	}
	if stmt.Object.Parts[0].Name.Name != "alignment_t" || stmt.Object.Parts[0].Wildcard {
		t.Fatalf("GRANT object = %#v", stmt.Object)
	}
}

func TestParseSharedMaintenanceCore(t *testing.T) {
	explain := mustParseOne(t, "EXPLAIN SELECT 1").(*ast.ExplainStmt)
	if _, ok := explain.Stmt.(*ast.SelectStmt); !ok {
		t.Fatalf("EXPLAIN body = %T", explain.Stmt)
	}
	analyze := mustParseOne(t, "ANALYZE TABLE alignment_t").(*ast.AnalyzeTableStmt)
	if len(analyze.TableNames) != 1 {
		t.Fatalf("ANALYZE targets = %#v", analyze.TableNames)
	}
	name := analyze.TableNames[0].TableRefCore().Name.Parts
	if len(name) != 1 || name[0].Name != "alignment_t" {
		t.Fatalf("ANALYZE target = %#v", name)
	}
}

func TestParseUtilityDialectBoundaries(t *testing.T) {
	for _, sql := range []string{
		"PREPARE alignment_p AS SELECT 1",
		"EXECUTE alignment_p (1)",
		"COPY alignment_t FROM STDIN",
		"LISTEN alignment_channel",
	} {
		t.Run(sql, func(t *testing.T) {
			if _, err := parser.New().ParseOneStmt(sql); err == nil {
				t.Fatal("accepted PostgreSQL utility syntax")
			}
		})
	}
}

func TestParseDialectImportCore(t *testing.T) {
	stmt := mustParseOne(t, "LOAD DATA LOCAL INFILE 'mock.csv' INTO TABLE alignment_t").(*ast.LoadDataStmt)
	if !stmt.IsLocal || stmt.Path != "mock.csv" || stmt.Table == nil {
		t.Fatalf("LOAD DATA core = %#v", stmt)
	}
	name := stmt.Table.TableRefCore().Name.Parts
	if len(name) != 1 || name[0].Name != "alignment_t" {
		t.Fatalf("LOAD DATA target = %#v", name)
	}
}

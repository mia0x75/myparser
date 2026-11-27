package parser_test

import (
	"testing"

	"github.com/mia0x75/myparser/ast"
)

func TestParseSharedCreateTableCore(t *testing.T) {
	stmt := mustParseOne(t, "CREATE TABLE alignment_t (id INT NOT NULL, name VARCHAR(20))").(*ast.CreateTableStmt)
	if stmt.Table == nil || len(stmt.Columns) != 2 {
		t.Fatalf("CREATE TABLE core = %#v", stmt)
	}
	for i, want := range []string{"id", "name"} {
		column := stmt.Columns[i].ColumnDefCore()
		if column == nil || len(column.Name.Parts) != 1 || column.Name.Parts[0].Name != want || column.Type == nil {
			t.Fatalf("column %d = %#v", i, stmt.Columns[i])
		}
	}
}

func TestParseSharedCreateTableAsCore(t *testing.T) {
	stmt := mustParseOne(t, "CREATE TABLE alignment_t AS SELECT 1 AS id").(*ast.CreateTableStmt)
	if stmt.Table == nil || stmt.Query == nil {
		t.Fatalf("CREATE TABLE AS core = %#v", stmt)
	}
}

func TestParseSharedAlterTableAddColumnCore(t *testing.T) {
	stmt := mustParseOne(t, "ALTER TABLE alignment_t ADD COLUMN extra INT").(*ast.AlterTableStmt)
	if stmt.Table == nil || len(stmt.Specs) != 1 || len(stmt.Specs[0].NewColumns) != 1 {
		t.Fatalf("ALTER TABLE core = %#v", stmt)
	}
	column := stmt.Specs[0].NewColumns[0]
	if len(column.Name.Parts) != 1 || column.Name.Parts[0].Name != "extra" || column.Type == nil {
		t.Fatalf("added column = %#v", column)
	}
}

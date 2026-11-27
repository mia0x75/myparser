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

func TestParseSharedTableConstraintCore(t *testing.T) {
	const sql = "CREATE TABLE align_child (id INT, parent_id INT, CONSTRAINT pk PRIMARY KEY (id), CONSTRAINT uq UNIQUE (parent_id), CONSTRAINT fk FOREIGN KEY (parent_id) REFERENCES align_parent (id) ON DELETE CASCADE, CONSTRAINT ck CHECK (id > 0))"
	stmt := mustParseOne(t, sql).(*ast.CreateTableStmt)
	if len(stmt.Constraints) != 4 {
		t.Fatalf("table constraints = %#v", stmt.Constraints)
	}
	wantTypes := []ast.ConstraintType{ast.ConstraintPrimaryKey, ast.ConstraintUniq, ast.ConstraintForeignKey, ast.ConstraintCheck}
	for i, want := range wantTypes {
		constraint, ok := stmt.Constraints[i].(*ast.Constraint)
		if !ok || constraint.Tp != want {
			t.Fatalf("constraint %d = %#v, want type %v", i, stmt.Constraints[i], want)
		}
		core := constraint.ConstraintCore()
		if core == nil {
			t.Fatalf("constraint %d has no shared core", i)
		}
	}
	primary := stmt.Constraints[0].(*ast.Constraint).ConstraintCore()
	unique := stmt.Constraints[1].(*ast.Constraint).ConstraintCore()
	if len(primary.Keys) != 1 || len(unique.Keys) != 1 {
		t.Fatalf("key constraint cores = %#v / %#v", primary, unique)
	}
	fkNode := stmt.Constraints[2].(*ast.Constraint)
	fk := fkNode.ConstraintCore()
	ref, ok := fk.Refer.(*ast.ReferenceDef)
	if !ok || ref.OnDelete == nil || ref.OnDelete.ReferOpt != ast.ReferOptionCascade {
		t.Fatalf("foreign key reference = %#v", fk.Refer)
	}
	if stmt.Constraints[3].(*ast.Constraint).ConstraintCore().Expr == nil {
		t.Fatalf("check constraint = %#v", stmt.Constraints[3])
	}
}

func TestParseSharedCreateIndexCore(t *testing.T) {
	stmt := mustParseOne(t, "CREATE UNIQUE INDEX align_idx ON align_child (parent_id)").(*ast.CreateIndexStmt)
	if !stmt.Unique || stmt.IndexName.Name != "align_idx" || stmt.Table == nil || len(stmt.Keys) != 1 {
		t.Fatalf("CREATE INDEX core = %#v", stmt)
	}
	key := stmt.Keys[0].IndexKeyPartCore()
	if key == nil || len(key.Column.Parts) != 1 || key.Column.Parts[0].Name != "parent_id" {
		t.Fatalf("CREATE INDEX key = %#v", key)
	}
}

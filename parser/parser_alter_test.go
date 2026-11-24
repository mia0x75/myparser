package parser

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	dformat "github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/loc"
)

func TestAlterColumnDefaultRestore(t *testing.T) {
	for _, tc := range []struct{ name, sql, want string }{
		{name: "set", sql: "ALTER TABLE t ALTER COLUMN a SET DEFAULT 1", want: "ALTER TABLE `t` ALTER COLUMN `a` SET DEFAULT 1"},
		{name: "drop", sql: "ALTER TABLE t ALTER COLUMN a DROP DEFAULT", want: "ALTER TABLE `t` ALTER COLUMN `a` DROP DEFAULT"},
		{name: "null", sql: "ALTER TABLE t ALTER COLUMN a SET DEFAULT NULL", want: "ALTER TABLE `t` ALTER COLUMN `a` SET DEFAULT NULL"},
		{name: "enable keys", sql: "ALTER TABLE t ENABLE KEYS", want: "ALTER TABLE `t` ENABLE KEYS"},
		{name: "disable keys", sql: "ALTER TABLE t DISABLE KEYS", want: "ALTER TABLE `t` DISABLE KEYS"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmt, err := New().ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			var out strings.Builder
			if err := stmt.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAlterReductionSpans(t *testing.T) {
	for _, spec := range []string{
		"ADD COLUMN a INT DEFAULT 1", "ADD (a INT, b INT)", "ADD PRIMARY KEY(a)",
		"DROP COLUMN a", "DROP PRIMARY KEY", "MODIFY COLUMN a INT FIRST", "CHANGE COLUMN a b INT AFTER c",
		"ALTER COLUMN a SET DEFAULT 1", "ALTER COLUMN a DROP DEFAULT", "RENAME TO u", "RENAME COLUMN a TO b",
		"LOCK=NONE", "ALGORITHM=INPLACE", "FORCE", "ENABLE KEYS", "DISABLE KEYS",
		"DROP PARTITION p", "TRUNCATE PARTITION p", "COALESCE PARTITION 2", "REMOVE PARTITIONING",
	} {
		t.Run(spec, func(t *testing.T) {
			sql := "/* 列 */ ALTER TABLE t " + spec
			stmt, err := New().ParseOneStmt(sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			if got, err := stmt.Span().Slice(sql); err != nil || got != "ALTER TABLE t "+spec {
				t.Fatalf("statement span = %q, error=%v", got, err)
			}
			alter := stmt.(*ast.AlterTableStmt)
			if got, err := alter.Specs[0].Span().Slice(sql); err != nil || got != spec {
				t.Fatalf("spec span = %q, error=%v", got, err)
			}
			for _, col := range alter.Specs[0].NewColumns {
				for _, option := range col.Options {
					if got, err := option.Span().Slice(sql); err != nil || got != "DEFAULT 1" {
						t.Fatalf("option span = %q, error=%v", got, err)
					}
				}
			}
			if position := alter.Specs[0].Position; position != nil {
				switch position.Tp {
				case ast.ColumnPositionNone:
					if position.Span() != loc.InvalidSpan() {
						t.Fatalf("missing position has span %v", position.Span())
					}
				case ast.ColumnPositionFirst, ast.ColumnPositionAfter:
					got, err := position.Span().Slice(sql)
					if err != nil || (got != "FIRST" && got != "AFTER c") {
						t.Fatalf("position span = %q, error=%v", got, err)
					}
				}
			}
		})
	}
}

func TestAlterDefaultSyntaxPayload(t *testing.T) {
	for _, tc := range []struct{ clause, value string }{
		{clause: "SET DEFAULT -2", value: "-2"},
		{clause: "SET DEFAULT NULL", value: "NULL"},
		{clause: "DROP DEFAULT"},
	} {
		t.Run(tc.clause, func(t *testing.T) {
			sql := "ALTER TABLE t ALTER `名` " + tc.clause
			stmt, err := New().ParseOneStmt(sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			spec := stmt.(*ast.AlterTableStmt).Specs[0]
			if len(spec.NewColumns) != 0 || spec.OldColumnName == nil {
				t.Fatal("default operation must not synthesize a column definition")
			}
			if got, err := spec.OldColumnName.Span.Slice(sql); err != nil || got != "`名`" {
				t.Fatalf("target span = %q, error=%v", got, err)
			}
			if tc.value == "" {
				if spec.Default != nil {
					t.Fatal("DROP DEFAULT retained a value")
				}
			} else {
				if spec.Default == nil {
					t.Fatal("SET DEFAULT lost its value")
				}
				if got, err := spec.Default.Span().Slice(sql); err != nil || got != tc.value {
					t.Fatalf("default span = %q, error=%v", got, err)
				}
			}
		})
	}
}

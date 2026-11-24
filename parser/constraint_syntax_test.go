package parser

import (
	"strings"
	"testing"
	"unsafe"

	"github.com/mia0x75/myparser/ast"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

func TestConstraintSyntaxNamesAndSpans(t *testing.T) {
	for _, tc := range []struct {
		name, clause string
		kind         pkast.ConstraintKind
		index        bool
	}{
		{"foreign", "FOREIGN KEY `索``名` (a) REFERENCES p(a)", pkast.ConstraintKindForeignKey, true},
		{"primary", "PRIMARY KEY `索``名` (a)", pkast.ConstraintKindPrimaryKey, true},
		{"unique", "UNIQUE KEY `索``名` (a)", pkast.ConstraintKindUnique, true},
		{"index", "KEY `索``名` (a)", pkast.ConstraintKindIndex, true},
		{"fulltext", "FULLTEXT KEY `索``名` (a)", pkast.ConstraintKindIndex, true},
		{"spatial", "SPATIAL KEY `索``名` (a)", pkast.ConstraintKindIndex, true},
		{"check", "CHECK (a > 0)", pkast.ConstraintKindCheck, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, prefix := range []string{"CREATE TABLE t (a INT, ", "ALTER TABLE t ADD "} {
				for _, symbol := range []string{"", "CONSTRAINT `符``名` ", "CONSTRAINT `` "} {
					clause := symbol + tc.clause
					sql := prefix + clause
					if strings.HasPrefix(prefix, "CREATE") {
						sql += ")"
					}
					p := New()
					stmt, err := p.ParseOneStmt(sql, "", "")
					if err != nil {
						t.Fatalf("parse %q: %v", sql, err)
					}
					var slot pkast.ConstraintNode
					switch s := stmt.(type) {
					case *ast.CreateTableStmt:
						slot = s.Constraints[0]
					case *ast.AlterTableStmt:
						slot = s.Specs[0].Constraint
					}
					c := slot.(*ast.Constraint)
					if slot.ConstraintKind() != tc.kind || slot.ConstraintCore() != &c.BaseConstraint {
						t.Fatalf("kind/core lost for %q", sql)
					}
					if got, err := c.Span().Slice(sql); err != nil || got != clause {
						t.Fatalf("constraint span selects %q (%v), want %q", got, err, clause)
					}
					if c.Name.Present() != (symbol != "") || c.IndexName.Present() != tc.index {
						t.Fatalf("names lost: symbol=%+v index=%+v", c.Name, c.IndexName)
					}
					for _, id := range []lex.Ident{c.Name, c.IndexName} {
						if !id.Present() {
							continue
						}
						if got, err := id.Span.Slice(sql); err != nil || got != id.Raw || id.Quote != lex.QuoteBacktick {
							t.Fatalf("identifier %+v selects %q (%v)", id, got, err)
						}
					}
					if tc.index && c.IndexName.Name != "索`名" {
						t.Fatalf("index name decoded as %q", c.IndexName.Name)
					}
					if symbol == "CONSTRAINT `符``名` " && c.Name.Name != "符`名" {
						t.Fatalf("symbol decoded as %q", c.Name.Name)
					}
					if tc.kind == pkast.ConstraintKindCheck {
						if got, err := c.Expr.Span().Slice(sql); err != nil || got != "a > 0" {
							t.Fatalf("CHECK span %q (%v)", got, err)
						}
					}
					before := restoreStmt(t, stmt)
					reparsed, err := p.ParseOneStmt(before, "", "")
					if err != nil {
						t.Fatalf("reparse %q: %v", before, err)
					}
					var next *ast.Constraint
					switch s := reparsed.(type) {
					case *ast.CreateTableStmt:
						next = s.Constraints[0].(*ast.Constraint)
					case *ast.AlterTableStmt:
						next = s.Specs[0].Constraint.(*ast.Constraint)
					}
					if next.Name.Name != c.Name.Name || next.Name.Present() != c.Name.Present() || next.IndexName.Name != c.IndexName.Name || next.IndexName.Present() != c.IndexName.Present() {
						t.Fatalf("reparse changed symbol/index roles: %+v / %+v -> %+v / %+v", c.Name, c.IndexName, next.Name, next.IndexName)
					}
					if restoreStmt(t, stmt) != before || tc.index && c.IndexName.Raw != "`索``名`" {
						t.Fatal("parser reuse changed retained names")
					}
					if symbol != "" && !strings.Contains(before, symbol) {
						t.Fatalf("restore lost symbol: %q", before)
					}
					if tc.index && !strings.Contains(before, "`索``名`") {
						t.Fatalf("restore lost index name: %q", before)
					}
				}
			}
		})
	}
}

func TestConstraintUnquotedNamesShareInput(t *testing.T) {
	const sql = "CREATE TABLE t (a INT, CONSTRAINT plain FOREIGN KEY idx (a) REFERENCES p(a))"
	c := mustParseOne(t, sql).(*ast.CreateTableStmt).Constraints[0].(*ast.Constraint)
	for _, id := range []lex.Ident{c.Name, c.IndexName} {
		if id.Name != id.Raw || id.Quote != lex.QuoteUnquoted || unsafe.StringData(id.Raw) != unsafe.StringData(sql[id.Span.Start:]) {
			t.Fatalf("unquoted identifier lost source slicing: %+v", id)
		}
	}
}

package parser

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
)

func TestReferenceSyntaxSpansAndReuse(t *testing.T) {
	for _, actions := range []string{
		"ON DELETE SET NULL ON UPDATE NO ACTION",
		"ON UPDATE NO ACTION ON DELETE SET NULL",
		"",
	} {
		t.Run(actions, func(t *testing.T) {
			p := New()
			refSQL := "REFERENCES `库`.`父`(`键`(2))"
			if actions != "" {
				refSQL += " " + actions
			}
			sql := "CREATE TABLE t (a INT, CONSTRAINT fk FOREIGN KEY (a) " + refSQL + ")"
			stmt, err := p.ParseOneStmt(sql)
			if err != nil {
				t.Fatal(err)
			}
			ref := stmt.(*ast.CreateTableStmt).Constraints[0].(*ast.Constraint).Refer.(*ast.ReferenceDef)
			for _, part := range []struct {
				node ast.Node
				want string
			}{
				{ref, refSQL}, {ref.Table, "`库`.`父`"}, {ref.Columns[0], "`键`(2)"},
			} {
				if got, err := part.node.Span().Slice(sql); err != nil || got != part.want {
					t.Fatalf("%T span selects %q (%v), want %q", part.node, got, err, part.want)
				}
			}
			for _, action := range []struct {
				node ast.Node
				want string
			}{{ref.OnDelete, "ON DELETE SET NULL"}, {ref.OnUpdate, "ON UPDATE NO ACTION"}} {
				if actions == "" {
					if action.node.Span().Valid() {
						t.Fatalf("epsilon action has a source span: %v", action.node.Span())
					}
				} else if got, err := action.node.Span().Slice(sql); err != nil || got != action.want {
					t.Fatalf("action span selects %q (%v), want %q", got, err, action.want)
				}
			}
			before := restoreStmt(t, stmt)
			if !strings.Contains(before, "REFERENCES `库`.`父`(`键`(2))") {
				t.Fatalf("reference lost its table or key: %s", before)
			}
			if _, err := p.ParseOneStmt("CREATE TABLE other (b TEXT)"); err != nil {
				t.Fatal(err)
			}
			if after := restoreStmt(t, stmt); after != before || ref.Table.TableRefCore().Name.Parts[0].Raw != "`库`" {
				t.Fatalf("parser reuse changed the old reference: %s -> %s", before, after)
			}
		})
	}
}

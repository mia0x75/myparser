package ast_test

import (
	"testing"

	. "github.com/mia0x75/myparser/ast"
	dformat "github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/model"
	pkast "github.com/mia0x75/parserkit/ast"
)

// byExtendedGroup proves same-role extensions survive clause traversal and output.
type byExtendedGroup struct{ pkast.BaseGroupItem }

func (n *byExtendedGroup) Accept(v Visitor) (Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}
func (n *byExtendedGroup) Restore(ctx *dformat.RestoreCtx) error {
	ctx.WritePlain("EXTENSION")
	return ctx.Err()
}

func TestGroupClauseSuffixAndExtensions(t *testing.T) {
	for _, tc := range []struct {
		name, want          string
		rollup, cube, check bool
		option              model.ViewCheckOption
	}{
		{"none", "GROUP BY 1 DESC", false, false, false, 0},
		{"rollup", "GROUP BY 1 DESC WITH ROLLUP", true, false, false, 0},
		{"cube", "GROUP BY 1 DESC WITH CUBE", false, true, false, 0},
		{"local", "GROUP BY 1 DESC WITH LOCAL CHECK OPTION", false, false, true, model.CheckOptionLocal},
		{"cascaded", "GROUP BY 1 DESC WITH CASCADED CHECK OPTION", false, false, true, model.CheckOptionCascaded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := &GroupByClause{BaseGroupByClause: pkast.BaseGroupByClause{Items: []GroupItemNode{&GroupItem{BaseGroupItem: pkast.BaseGroupItem{Expr: &PositionExpr{N: 1}}, Desc: true}}}, WithRollup: tc.rollup, WithCube: tc.cube, HasCheckOption: tc.check, CheckOption: tc.option}
			if got := restoreExpr(t, n, dformat.DefaultRestoreFlags); got != tc.want {
				t.Fatalf("restore=%q want=%q", got, tc.want)
			}
		})
	}
	for _, role := range []string{"order", "group"} {
		t.Run(role+"/extension", func(t *testing.T) {
			f := makeByClause(role)
			var replacement Node = &byExtendedGroup{}
			if role == "order" {
				replacement = &byExtendedOrder{}
			}
			_, ok := f.node.Accept(byClauseVisitor{leave: func(n Node) (Node, bool) {
				if n == f.items[0] {
					return replacement, true
				}
				return n, true
			}})
			if !ok || f.item(0) != replacement {
				t.Fatal("same-role dialect extension lost")
			}
			seen := false
			_, ok = f.node.Accept(byClauseVisitor{enter: func(n Node) (Node, bool) { seen = seen || n == replacement; return n, false }})
			if !ok || !seen {
				t.Fatal("extension outer identity lost")
			}
			want := "GROUP BY EXTENSION,2"
			if role == "order" {
				want = "ORDER BY EXTENSION,2"
			}
			if got := restoreExpr(t, f.node, dformat.DefaultRestoreFlags); got != want {
				t.Fatalf("extension restore=%q want=%q", got, want)
			}
		})
	}
}

type byExtendedOrder struct{ pkast.BaseOrderItem }

func (n *byExtendedOrder) Accept(v Visitor) (Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}
func (n *byExtendedOrder) Restore(ctx *dformat.RestoreCtx) error {
	ctx.WritePlain("EXTENSION")
	return ctx.Err()
}

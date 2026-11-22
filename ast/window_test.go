package ast_test

import (
	"io"
	"testing"

	. "github.com/mia0x75/myparser/ast"
	dformat "github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
)

func TestByRoleNilAndEmptyLists(t *testing.T) {
	for _, tc := range []struct {
		name string
		node Node
	}{
		{"order nil", &OrderByClause{}},
		{"group nil", &GroupByClause{}},
		{"partition nil", &PartitionByClause{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := tc.node.Accept(byClauseVisitor{}); !ok {
				t.Fatal("empty list traversal failed")
			}
			switch n := tc.node.(type) {
			case *OrderByClause:
				if n.Items != nil {
					t.Fatal("nil order list changed")
				}
			case *GroupByClause:
				if n.Items != nil {
					t.Fatal("nil group list changed")
				}
			case *PartitionByClause:
				if n.Items != nil {
					t.Fatal("nil partition list changed")
				}
			}
		})
	}
	order := &OrderByClause{BaseOrderByClause: pkast.BaseOrderByClause{Items: []OrderItemNode{}}}
	group := &GroupByClause{BaseGroupByClause: pkast.BaseGroupByClause{Items: []GroupItemNode{}}}
	partition := &PartitionByClause{BaseWindowPartitionClause: pkast.BaseWindowPartitionClause{Items: []pkast.WindowPartitionItemNode{}}}
	for _, n := range []Node{order, group, partition} {
		if _, ok := n.Accept(byClauseVisitor{}); !ok {
			t.Fatal("present empty list failed")
		}
	}
	if order.Items == nil || group.Items == nil || partition.Items == nil {
		t.Fatal("present empty list became nil")
	}
	for _, n := range []Node{
		&OrderByClause{BaseOrderByClause: pkast.BaseOrderByClause{Items: []OrderItemNode{nil}}},
		&GroupByClause{BaseGroupByClause: pkast.BaseGroupByClause{Items: []GroupItemNode{(*GroupItem)(nil)}}},
		&PartitionByClause{BaseWindowPartitionClause: pkast.BaseWindowPartitionClause{Items: []pkast.WindowPartitionItemNode{nil}}},
		&PartitionByClause{BaseWindowPartitionClause: pkast.BaseWindowPartitionClause{Items: []pkast.WindowPartitionItemNode{(*PartitionItem)(nil)}}},
	} {
		if _, ok := n.Accept(byClauseVisitor{}); ok {
			t.Fatalf("%T accepted nil item", n)
		}
		if err := n.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, io.Discard)); err == nil {
			t.Fatalf("%T restored nil item", n)
		}
	}
	if got := restoreExpr(t, &PartitionItem{Expr: &PositionExpr{N: 1}, Desc: true}, dformat.DefaultRestoreFlags); got != "1 DESC" {
		t.Fatalf("partition direction lost: %q", got)
	}
}

func TestWindowAcceptRejectsWrongReplacement(t *testing.T) {
	for _, node := range []Node{
		&WindowSpec{},
		&FrameClause{},
		&FrameBound{},
	} {
		got, ok := node.Accept(replaceWindowNodeVisitor{})
		if ok || got == nil {
			t.Fatalf("%T accepted replacement %T: ok=%v", node, got, ok)
		}
	}
	bound := &FrameBound{BaseWindowFrameBound: pkast.BaseWindowFrameBound{Expr: &LiteralExpr{}}}
	got, ok := bound.Accept(replaceWindowExprVisitor{})
	if ok || got == nil {
		t.Fatalf("FrameBound accepted expression replacement %T: ok=%v", got, ok)
	}
}

type replaceWindowNodeVisitor struct{}

func (replaceWindowNodeVisitor) Enter(node Node) (Node, bool) {
	switch node.(type) {
	case *WindowSpec, *FrameClause, *FrameBound:
		return &TableName{}, false
	default:
		return node, false
	}
}

func (replaceWindowNodeVisitor) Leave(node Node) (Node, bool) { return node, true }

type replaceWindowExprVisitor struct{}

func (replaceWindowExprVisitor) Enter(node Node) (Node, bool) {
	if _, ok := node.(*LiteralExpr); ok {
		return &TableName{}, false
	}
	return node, false
}

func (replaceWindowExprVisitor) Leave(node Node) (Node, bool) { return node, true }

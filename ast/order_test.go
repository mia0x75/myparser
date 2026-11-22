package ast_test

import (
	"errors"
	"io"
	"reflect"
	"testing"

	. "github.com/mia0x75/myparser/ast"
	dformat "github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
)

type byClauseVisitor struct {
	enter func(Node) (Node, bool)
	leave func(Node) (Node, bool)
}

func (v byClauseVisitor) Enter(n Node) (Node, bool) {
	if v.enter != nil {
		return v.enter(n)
	}
	return n, false
}

func (v byClauseVisitor) Leave(n Node) (Node, bool) {
	if v.leave != nil {
		return v.leave(n)
	}
	return n, true
}

type byClauseFixture struct {
	node     Node
	items    [2]Node
	exprs    [2]ExprNode
	item     func(int) Node
	typedNil Node
}

func makeByClause(role string) byClauseFixture {
	f := byClauseFixture{exprs: [2]ExprNode{&PositionExpr{N: 1}, &PositionExpr{N: 2}}}
	switch role {
	case "order":
		a := &OrderItem{BaseOrderItem: pkast.BaseOrderItem{Expr: f.exprs[0]}}
		b := &OrderItem{BaseOrderItem: pkast.BaseOrderItem{Expr: f.exprs[1]}}
		n := &OrderByClause{BaseOrderByClause: pkast.BaseOrderByClause{Items: []OrderItemNode{a, b}}, ForUnion: true}
		f.node, f.items, f.item, f.typedNil = n, [2]Node{a, b}, func(i int) Node { return n.Items[i] }, (*OrderItem)(nil)
	case "group":
		a := &GroupItem{BaseGroupItem: pkast.BaseGroupItem{Expr: f.exprs[0]}}
		b := &GroupItem{BaseGroupItem: pkast.BaseGroupItem{Expr: f.exprs[1]}}
		n := &GroupByClause{BaseGroupByClause: pkast.BaseGroupByClause{Items: []GroupItemNode{a, b}}}
		f.node, f.items, f.item, f.typedNil = n, [2]Node{a, b}, func(i int) Node { return n.Items[i] }, (*GroupItem)(nil)
	case "partition":
		a, b := &PartitionItem{Expr: f.exprs[0]}, &PartitionItem{Expr: f.exprs[1]}
		n := &PartitionByClause{BaseWindowPartitionClause: pkast.BaseWindowPartitionClause{Items: []pkast.WindowPartitionItemNode{a, b}}}
		f.node, f.items, f.item, f.typedNil = n, [2]Node{a, b}, func(i int) Node { return n.Items[i] }, (*PartitionItem)(nil)
	}
	return f
}

func TestByRoleClauseTraversal(t *testing.T) {
	for _, role := range []string{"order", "group", "partition"} {
		t.Run(role, func(t *testing.T) {
			f := makeByClause(role)
			var enters, leaves []Node
			out, ok := f.node.Accept(byClauseVisitor{
				enter: func(n Node) (Node, bool) { enters = append(enters, n); return n, false },
				leave: func(n Node) (Node, bool) { leaves = append(leaves, n); return n, true },
			})
			if !ok || out != f.node || !reflect.DeepEqual(enters, []Node{f.node, f.items[0], f.exprs[0], f.items[1], f.exprs[1]}) ||
				!reflect.DeepEqual(leaves, []Node{f.exprs[0], f.items[0], f.exprs[1], f.items[1], f.node}) {
				t.Fatalf("outer identity or traversal order lost: enters=%v leaves=%v", enters, leaves)
			}
			if n, ok := f.node.(*OrderByClause); ok && !n.ForUnion {
				t.Fatal("legacy ForUnion overwritten")
			}

			t.Run("skip clause", func(t *testing.T) {
				var seen []Node
				_, ok := f.node.Accept(byClauseVisitor{enter: func(n Node) (Node, bool) { seen = append(seen, n); return n, true }})
				if !ok || !reflect.DeepEqual(seen, []Node{f.node}) {
					t.Fatal("skip must not visit children")
				}
			})
			t.Run("same type Enter replacement", func(t *testing.T) {
				replacement := makeByClause(role)
				out, ok := f.node.Accept(byClauseVisitor{enter: func(n Node) (Node, bool) {
					if n == f.node {
						return replacement.node, false
					}
					return n, false
				}})
				if !ok || out != replacement.node {
					t.Fatal("clause Enter replacement lost")
				}
			})
			t.Run("item and expression replacement", func(t *testing.T) {
				f, replacement := makeByClause(role), makeByClause(role)
				newExpr := &PositionExpr{N: 3}
				_, ok := f.node.Accept(byClauseVisitor{leave: func(n Node) (Node, bool) {
					if n == f.items[0] {
						return replacement.items[0], true
					}
					if n == f.exprs[1] {
						return newExpr, true
					}
					return n, true
				}})
				if !ok || f.item(0) != replacement.items[0] {
					t.Fatal("same-role replacement not written back")
				}
				var found bool
				_, ok = f.item(1).Accept(byClauseVisitor{enter: func(n Node) (Node, bool) { found = found || n == newExpr; return n, false }})
				if !ok || !found {
					t.Fatal("expression replacement not written back")
				}
			})
			t.Run("stop without later visits", func(t *testing.T) {
				f := makeByClause(role)
				var seen []Node
				_, ok := f.node.Accept(byClauseVisitor{enter: func(n Node) (Node, bool) { seen = append(seen, n); return n, false }, leave: func(n Node) (Node, bool) { return n, n != f.exprs[0] }})
				if ok || !reflect.DeepEqual(seen, []Node{f.node, f.items[0], f.exprs[0]}) {
					t.Fatalf("stop propagation failed: %v", seen)
				}
			})
			for _, tc := range []struct {
				name        string
				replacement Node
			}{
				{"non-item", &FieldList{}}, {"nil", nil}, {"typed nil", f.typedNil},
				{"cross role", map[string]Node{"order": &GroupItem{}, "group": &OrderItem{}, "partition": &OrderItem{}}[role]},
			} {
				t.Run(tc.name, func(t *testing.T) {
					f := makeByClause(role)
					_, ok := f.node.Accept(byClauseVisitor{leave: func(n Node) (Node, bool) {
						if n == f.items[0] {
							return tc.replacement, true
						}
						return n, true
					}})
					if ok || f.item(0) != f.items[0] {
						t.Fatal("invalid role replacement must fail without changing the slot")
					}
				})
			}
			t.Run("wrong Enter type", func(t *testing.T) {
				_, ok := f.node.Accept(byClauseVisitor{enter: func(Node) (Node, bool) { return &FieldList{}, false }})
				if ok {
					t.Fatal("wrong Enter type must stop")
				}
			})
		})
	}
}

type byShortWriter struct{}

func (byShortWriter) Write([]byte) (int, error) { return 0, nil }

func TestByRoleClauseWriteErrors(t *testing.T) {
	for _, role := range []string{"order", "group", "partition"} {
		t.Run(role, func(t *testing.T) {
			f := makeByClause(role)
			err := f.node.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, byShortWriter{}))
			if !errors.Is(err, io.ErrShortWrite) {
				t.Fatalf("writer root cause lost: %v", err)
			}
		})
	}
}

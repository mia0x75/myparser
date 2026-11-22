package ast

import (
	"io"
	"reflect"
	"testing"

	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

type jsonBoundaryVisitor struct {
	enter func(Node) (Node, bool)
	leave func(Node) (Node, bool)
}

func (v jsonBoundaryVisitor) Enter(n Node) (Node, bool) {
	if v.enter != nil {
		return v.enter(n)
	}
	return n, false
}
func (v jsonBoundaryVisitor) Leave(n Node) (Node, bool) {
	if v.leave != nil {
		return v.leave(n)
	}
	return n, true
}

type jsonExtensionType struct{ pkast.BaseTypeName }

func (n *jsonExtensionType) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("INT")
	return ctx.Err()
}
func (n *jsonExtensionType) Accept(v Visitor) (Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func TestJSONTableBoundary(t *testing.T) {
	column := func() *JSONTableColumn {
		return &JSONTableColumn{Name: lex.Ident{Name: "c"}, Kind: JSONTableColumnPath, Type: NewTypeName("INT")}
	}
	for _, tc := range []struct {
		name string
		node Node
	}{
		{"nil table", (*JSONTable)(nil)},
		{"nil column", (*JSONTableColumn)(nil)},
		{"nil behavior", (*JSONTableBehavior)(nil)},
		{"missing expression", &JSONTable{Columns: []*JSONTableColumn{column()}}},
		{"typed nil expression", &JSONTable{Expr: (*LiteralExpr)(nil), Columns: []*JSONTableColumn{column()}}},
		{"missing columns", &JSONTable{Expr: jsonLitExpr(t, "1")}},
		{"typed nil type", &JSONTableColumn{Name: lex.Ident{Name: "c"}, Kind: JSONTableColumnPath, Type: (*jsonExtensionType)(nil)}},
		{"typed nil value", &JSONTableBehavior{Kind: JSONTableBehaviorDefault, On: JSONTableBehaviorOnEmpty, Value: (*LiteralExpr)(nil)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := tc.node.Accept(jsonBoundaryVisitor{}); ok {
				t.Fatal("invalid tree accepted")
			}
			if err := tc.node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, io.Discard)); err == nil {
				t.Fatal("invalid tree restored")
			}
		})
	}
	for _, root := range []Node{&JSONTable{}, column(), &JSONTableBehavior{}} {
		for _, replacement := range []Node{nil, (*JSONTable)(nil), (*JSONTableColumn)(nil), (*JSONTableBehavior)(nil), &GrantObject{}} {
			if _, ok := root.Accept(jsonBoundaryVisitor{enter: func(Node) (Node, bool) { return replacement, true }}); ok {
				t.Fatalf("%T accepted invalid skip replacement %T", root, replacement)
			}
		}
	}
	for _, replacement := range []Node{(*LiteralExpr)(nil), &GrantObject{}} {
		table := &JSONTable{Expr: jsonLitExpr(t, "1"), Columns: []*JSONTableColumn{column()}}
		if _, ok := table.Accept(jsonBoundaryVisitor{leave: func(n Node) (Node, bool) {
			if _, ok := n.(*LiteralExpr); ok {
				return replacement, true
			}
			return n, true
		}}); ok {
			t.Fatal("invalid expression replacement accepted")
		}
	}
	extension := &jsonExtensionType{}
	c := column()
	if _, ok := c.Accept(jsonBoundaryVisitor{leave: func(n Node) (Node, bool) {
		if _, ok := n.(*TypeName); ok {
			return extension, true
		}
		return n, true
	}}); !ok || c.Type != extension {
		t.Fatal("same-role type extension rejected")
	}
	if got := restoreNode(t, c); got != "`c` INT PATH ''" {
		t.Fatal(got)
	}
	var order []reflect.Type
	c.Behaviors = []*JSONTableBehavior{{Kind: JSONTableBehaviorDefault, On: JSONTableBehaviorOnEmpty, Value: jsonLitExpr(t, "1")}}
	if _, ok := c.Accept(jsonBoundaryVisitor{enter: func(n Node) (Node, bool) {
		order = append(order, reflect.TypeOf(n))
		return n, false
	}}); !ok {
		t.Fatal("valid tree rejected")
	}
	want := []reflect.Type{reflect.TypeOf(c), reflect.TypeOf(extension), reflect.TypeOf(c.Behaviors[0]), reflect.TypeOf(c.Behaviors[0].Value)}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("visit order %v, want %v", order, want)
	}
}

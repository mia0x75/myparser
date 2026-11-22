package ast_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

func TestProjectionNodesKeepSharedDispatch(t *testing.T) {
	t.Parallel()
	if reflect.TypeFor[ast.SelectItem]() != reflect.TypeFor[pkast.SelectItem]() ||
		reflect.TypeFor[ast.FieldList]() != reflect.TypeFor[pkast.FieldList]() ||
		reflect.TypeFor[ast.WildcardExpr]() != reflect.TypeFor[pkast.WildcardExpr]() {
		t.Fatalf("projection nodes must keep the shared concrete type identity")
	}

	expr := &ast.ColumnRef{BaseColumnRef: pkast.BaseColumnRef{
		Name: ast.QualifiedName{Parts: []lex.Ident{{Name: "total"}}},
	}}
	item := &ast.SelectItem{Expr: expr, Alias: lex.Ident{Name: "sum"}}
	visited := []string{}
	visitor := testVisitor{log: &visited}
	if _, ok := item.Accept(visitor); !ok {
		t.Fatalf("item dispatch failed: %v", visited)
	}
	if len(visited) != 2 || visited[0] != "SelectItem" || visited[1] != "ColumnRef" {
		t.Fatalf("item dispatch order = %v", visited)
	}

	wildcard := &ast.WildcardExpr{Qualifier: ast.QualifiedName{Parts: []lex.Ident{{Name: "t"}}}}
	if _, ok := wildcard.Accept(visitor); !ok {
		t.Fatalf("wildcard dispatch failed: %v", visited)
	}
	if !reflect.DeepEqual(visited, []string{"SelectItem", "ColumnRef", "WildcardExpr"}) {
		t.Fatalf("wildcard dispatch order = %v", visited)
	}
	visited = nil
	var out strings.Builder
	if err := wildcard.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
		t.Fatal(err)
	}
	if out.String() != "`t`.*" {
		t.Fatalf("wildcard restore = %q", out.String())
	}

	list := &ast.FieldList{Items: []*ast.SelectItem{item, {Expr: wildcard}}}
	if _, ok := list.Accept(visitor); !ok {
		t.Fatalf("field list dispatch failed: %v", visited)
	}
	want := []string{"FieldList", "SelectItem", "ColumnRef", "SelectItem", "WildcardExpr"}
	if !reflect.DeepEqual(visited, want) {
		t.Fatalf("field list dispatch order = %v, want %v", visited, want)
	}
	out.Reset()
	if err := list.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
		t.Fatal(err)
	}
	if out.String() != "`total` AS `sum`, `t`.*" {
		t.Fatalf("field list restore = %q", out.String())
	}
}

// testVisitor records the concrete type name of every entered node.
type testVisitor struct{ log *[]string }

func (v testVisitor) Enter(n ast.Node) (ast.Node, bool) {
	*v.log = append(*v.log, reflect.TypeOf(n).Elem().Name())
	return n, false
}

func (v testVisitor) Leave(n ast.Node) (ast.Node, bool) { return n, true }

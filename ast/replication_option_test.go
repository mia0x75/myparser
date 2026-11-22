package ast_test

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
)

func TestReplicationOptionNodeContract(t *testing.T) {
	if reflect.TypeFor[*ast.ReplicationOption]().Implements(reflect.TypeFor[ast.ExprNode]()) ||
		reflect.TypeFor[*ast.ReplicationOption]().Implements(reflect.TypeFor[ast.StmtNode]()) {
		t.Fatal("replication settings must be plain syntax nodes, not expressions or statements")
	}
	for _, tc := range []struct {
		name string
		node *ast.ReplicationOption
	}{
		{"nil", nil},
		{"missing name", &ast.ReplicationOption{Expr: &sharedPureExpr{}}},
		{"missing value", &ast.ReplicationOption{Name: lex.Ident{Name: "master_host"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := tc.node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil || out.Len() != 0 {
				t.Fatalf("incomplete option must fail before output: out=%q, err=%v", out.String(), err)
			}
		})
	}
	option := &ast.ReplicationOption{Name: lex.Ident{Name: "master_host", Raw: "master_host"}, Expr: &sharedPureExpr{}}
	var out strings.Builder
	if err := option.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil || out.String() != "MASTER_HOST=PURE" {
		t.Fatalf("keyword output = %q, err=%v", out.String(), err)
	}
	if err := option.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, failingWriter{})); err == nil {
		t.Fatal("writer failure must propagate")
	}
	option.Expr = &sharedPureExpr{restoreErr: true}
	var cause *pureRestoreError
	if err := option.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, io.Discard)); !errors.As(err, &cause) {
		t.Fatalf("expression error chain lost: %v", err)
	}
}

func TestReplicationOptionVisitor(t *testing.T) {
	originalExpr, replacementExpr, visitedExpr := &sharedPureExpr{}, &sharedPureExpr{}, &sharedPureExpr{}
	original := &ast.ReplicationOption{Name: lex.Ident{Name: "master_port"}, Expr: originalExpr}
	replacement := &ast.ReplicationOption{Name: lex.Ident{Name: "source_port"}, Expr: replacementExpr}
	var entered []ast.Node
	got, ok := original.Accept(operatorVisitor{
		enter: func(n ast.Node) (ast.Node, bool) {
			entered = append(entered, n)
			if n == original {
				return replacement, false
			}
			return n, false
		},
		leave: func(n ast.Node) (ast.Node, bool) {
			if n == replacementExpr {
				return visitedExpr, true
			}
			return n, true
		},
	})
	if !ok || got != replacement || replacement.Expr != visitedExpr || original.Expr != originalExpr ||
		!reflect.DeepEqual(entered, []ast.Node{original, replacementExpr}) {
		t.Fatalf("outer substitution must visit only the replacement value: got=%v, ok=%v, entered=%v", got, ok, entered)
	}
	for _, tc := range []struct {
		name string
		node ast.Node
	}{
		{"nil outer", nil},
		{"typed nil outer", (*ast.ReplicationOption)(nil)},
		{"wrong outer", &ast.Assignment{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			childVisited := false
			_, ok := original.Accept(operatorVisitor{
				enter: func(n ast.Node) (ast.Node, bool) {
					if n == original {
						return tc.node, false
					}
					childVisited = true
					return n, false
				},
				leave: func(n ast.Node) (ast.Node, bool) { return n, true },
			})
			if ok || childVisited {
				t.Fatal("invalid outer replacement must stop before visiting children")
			}
		})
	}
	for _, tc := range []struct {
		name string
		node ast.Node
	}{
		{"nil", nil},
		{"non expression", &ast.ReplicationOption{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			node := &ast.ReplicationOption{Expr: originalExpr}
			_, ok := node.Accept(operatorVisitor{
				enter: func(n ast.Node) (ast.Node, bool) { return n, false },
				leave: func(n ast.Node) (ast.Node, bool) {
					if n == originalExpr {
						return tc.node, true
					}
					return n, true
				},
			})
			if ok || node.Expr != originalExpr {
				t.Fatal("invalid child replacement must stop without changing the slot")
			}
		})
	}
	for _, tc := range []struct {
		name string
		skip bool
	}{{"stop", false}, {"skip", true}} {
		t.Run(tc.name, func(t *testing.T) {
			node := &ast.ReplicationOption{Expr: originalExpr}
			childEntered, parentLeft := false, false
			_, ok := node.Accept(operatorVisitor{
				enter: func(n ast.Node) (ast.Node, bool) {
					childEntered = childEntered || n == originalExpr
					return n, tc.skip
				},
				leave: func(n ast.Node) (ast.Node, bool) {
					parentLeft = parentLeft || n == node
					return n, n != originalExpr
				},
			})
			if ok != tc.skip || childEntered == tc.skip || parentLeft != tc.skip {
				t.Fatalf("skip/stop propagation: ok=%v, child=%v, parent=%v", ok, childEntered, parentLeft)
			}
		})
	}
}

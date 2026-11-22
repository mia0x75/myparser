package ast_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
)

func TestSequenceOptionalOperandsAndRestoreBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name string
		node *ast.SequenceOption
		want string
	}{
		{"AS without value", &ast.SequenceOption{Tp: ast.SequenceOptionAs, Type: ast.NewTypeName("INT")}, "AS INT"},
		{"Val fallback without type", &ast.SequenceOption{Tp: ast.SequenceOptionIncrement, Val: -2}, "INCREMENT BY -2"},
		{"CYCLE without operands", &ast.SequenceOption{Tp: ast.SequenceOptionCycle}, "CYCLE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := tc.node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil || out.String() != tc.want {
				t.Fatalf("Restore = %q, %v; want %q", out.String(), err, tc.want)
			}
			if got, ok := tc.node.Accept(&sharedVisitor{}); !ok || got != tc.node {
				t.Fatal("Accept required an absent optional operand")
			}
			marker := errors.New("MOCK_WRITER")
			if err := tc.node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, callBoundaryWriter{marker})); !errors.Is(err, marker) {
				t.Fatalf("lost writer error: %v", err)
			}
		})
	}
	for _, tc := range []struct {
		name string
		node ast.Node
	}{
		{"nil option", (*ast.SequenceOption)(nil)},
		{"AS missing type", &ast.SequenceOption{Tp: ast.SequenceOptionAs}},
		{"typed nil value", &ast.SequenceOption{Tp: ast.SequenceOptionIncrement, Value: (*ast.LiteralExpr)(nil)}},
		{"CREATE missing name", &ast.CreateSequenceStmt{}},
		{"CREATE nil option", &ast.CreateSequenceStmt{Name: &ast.TableName{Name: ast.QualifiedName{Parts: []lex.Ident{{Name: "s"}}}}, Options: []*ast.SequenceOption{nil}}},
		{"DROP nil name", &ast.DropSequenceStmt{Names: []*ast.TableName{nil}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := tc.node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil {
				t.Fatal("Restore accepted missing operand")
			}
		})
	}
}

func TestSequenceVisitorKeepsRejectedSlotsAndAllowsExpressionReplacement(t *testing.T) {
	for _, bad := range []ast.Node{nil, (*ast.LiteralExpr)(nil), &ast.ColumnPosition{}} {
		value, typ := ast.NewParamMarkerExpr(0), ast.NewTypeName("INT")
		for _, target := range []ast.Node{value, typ} {
			n := &ast.SequenceOption{Type: typ, Value: value}
			if _, ok := n.Accept(operatorVisitor{
				enter: func(node ast.Node) (ast.Node, bool) { return node, false },
				leave: func(node ast.Node) (ast.Node, bool) {
					if node == target {
						return bad, true
					}
					return node, true
				},
			}); ok || n.Type != typ || n.Value != value {
				t.Fatal("Accept accepted or stored an invalid replacement")
			}
		}
	}
	old, replacement := ast.NewParamMarkerExpr(0), &ast.DefaultExpr{}
	n := &ast.SequenceOption{Value: old}
	if _, ok := n.Accept(operatorVisitor{
		enter: func(node ast.Node) (ast.Node, bool) { return node, false },
		leave: func(node ast.Node) (ast.Node, bool) {
			if node == old {
				return replacement, true
			}
			return node, true
		},
	}); !ok || n.Value != replacement {
		t.Fatal("Accept rejected a legal expression replacement")
	}
}

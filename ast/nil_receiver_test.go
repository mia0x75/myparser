package ast

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/format"
)

type nilReceiverVisitor struct{ calls int }

func (v *nilReceiverVisitor) Enter(n Node) (Node, bool) {
	v.calls++
	return n, false
}

func (v *nilReceiverVisitor) Leave(n Node) (Node, bool) {
	v.calls++
	return n, true
}

func TestNilReceiversDoNotDispatchOrPanic(t *testing.T) {
	nodes := []Node{
		(*LiteralExpr)(nil),
		(*ParamMarkerExpr)(nil),
		(*VariableTarget)(nil),
		(*ChangeMasterStmt)(nil),
		(*ReplicaControlStmt)(nil),
		(*ExplainForStmt)(nil),
		(*SelectStmt)(nil),
		(*InsertStmt)(nil),
		(*CreateDatabaseStmt)(nil),
		(*ColumnRef)(nil),
		(*ValuesStmt)(nil),
		(*FrameBound)(nil),
		(*GroupItem)(nil),
		(*WindowSpec)(nil),
		(*TypeName)(nil),
		(*CreateUserStmt)(nil),
		(*SetStmt)(nil),
		(*WhileStmt)(nil),
	}
	for _, node := range nodes {
		v := &nilReceiverVisitor{}
		updated, ok := node.Accept(v)
		if updated != nil || ok || v.calls != 0 {
			t.Errorf("%T nil Accept = (%v, %v), visitor calls = %d", node, updated, ok, v.calls)
		}
		ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{})
		if err := node.Restore(ctx); err == nil {
			t.Errorf("%T nil Restore returned no error", node)
		}
	}
}

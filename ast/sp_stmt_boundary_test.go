package ast

import (
	"fmt"
	"testing"
)

type spBoundaryVisitor struct {
	enter func(Node) (Node, bool)
	leave func(Node) (Node, bool)
}

func (v spBoundaryVisitor) Enter(n Node) (Node, bool) {
	if v.enter != nil {
		return v.enter(n)
	}
	return n, false
}

func (v spBoundaryVisitor) Leave(n Node) (Node, bool) {
	if v.leave != nil {
		return v.leave(n)
	}
	return n, true
}

func TestIfBranchVisitorIdentity(t *testing.T) {
	n := &IfStmt{Conditions: []*IfClause{{Condition: NewParamMarkerExpr(0), List: []StmtNode{&BeginEndStmt{}}}}}
	var branches int
	_, ok := n.Accept(spBoundaryVisitor{enter: func(n Node) (Node, bool) {
		if fmt.Sprintf("%T", n) == "*ast.IfClause" {
			branches++
		}
		return n, false
	}})
	if !ok || branches != 1 {
		t.Fatalf("branch visits = %d, ok=%v", branches, ok)
	}
}

func TestStatementListRejectsInvalidReplacement(t *testing.T) {
	for _, tc := range []struct {
		name string
		node Node
	}{
		{name: "wrong role", node: NewParamMarkerExpr(0)},
		{name: "nil"},
		{name: "typed nil", node: (*BeginEndStmt)(nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if err := recover(); err != nil {
					t.Fatalf("invalid replacement panicked: %v", err)
				}
			}()
			list := []StmtNode{&BeginEndStmt{}}
			_, ok := acceptStmtList(spBoundaryVisitor{leave: func(Node) (Node, bool) { return tc.node, true }}, list)
			if ok {
				t.Fatal("accepted invalid statement replacement")
			}
		})
	}
}

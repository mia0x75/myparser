package ast

import (
	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
)

// CallStmt invokes a stored procedure, optionally qualified by a database name.
type CallStmt struct {
	stmtNode
	Name *TableName
	Args []ExprNode
}

// Restore implements Node interface.
func (n *CallStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Name) {
		return errors.New("restore CallStmt: missing name")
	}
	ctx.WriteKeyword("CALL ")
	if err := n.Name.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CallStmt.Name")
	}
	ctx.WritePlain("(")
	for i, arg := range n.Args {
		if !nodePresent(arg) {
			return errors.Errorf("restore CallStmt.Args[%d]: nil expression", i)
		}
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := arg.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore CallStmt.Args[%d]", i)
		}
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

// Accept implements Node interface.
func (n *CallStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	updated, ok := newNode.(*CallStmt)
	if !ok || updated == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(updated)
	}
	n = updated
	if !nodePresent(n.Name) {
		return n, false
	}
	name, ok := n.Name.Accept(v)
	if !ok {
		return n, false
	}
	updatedName, ok := name.(*TableName)
	if !ok || updatedName == nil {
		return n, false
	}
	n.Name = updatedName
	for i, arg := range n.Args {
		if !nodePresent(arg) {
			return n, false
		}
		node, ok := arg.Accept(v)
		if !ok {
			return n, false
		}
		updatedArg, ok := node.(ExprNode)
		if !ok || !nodePresent(updatedArg) {
			return n, false
		}
		n.Args[i] = updatedArg
	}
	return v.Leave(n)
}

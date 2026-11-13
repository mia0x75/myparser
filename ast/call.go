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
	ctx.WriteKeyWord("CALL ")
	if err := n.Name.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CallStmt.Name")
	}
	ctx.WritePlain("(")
	for i, arg := range n.Args {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := arg.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore CallStmt.Args[%d]", i)
		}
	}
	ctx.WritePlain(")")
	return nil
}

// Accept implements Node interface.
func (n *CallStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CallStmt)
	name, ok := n.Name.Accept(v)
	if !ok {
		return n, false
	}
	n.Name = name.(*TableName)
	for i, arg := range n.Args {
		node, ok := arg.Accept(v)
		if !ok {
			return n, false
		}
		n.Args[i] = node.(ExprNode)
	}
	return v.Leave(n)
}

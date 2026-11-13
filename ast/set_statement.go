package ast

import (
	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
)

// SetStatementStmt applies session variables to a single MariaDB statement.
type SetStatementStmt struct {
	stmtNode
	Variables []*VariableAssignment
	Stmt      StmtNode
}

// Restore implements Node interface.
func (n *SetStatementStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("SET STATEMENT ")
	for i, variable := range n.Variables {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		ctx.WriteName(variable.Name)
		ctx.WritePlain("=")
		if err := variable.Value.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore SetStatementStmt.Variables[%d]", i)
		}
	}
	ctx.WriteKeyWord(" FOR ")
	if err := n.Stmt.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore SetStatementStmt.Stmt")
	}
	return nil
}

// Accept implements Node interface.
func (n *SetStatementStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*SetStatementStmt)
	for i, variable := range n.Variables {
		node, ok := variable.Accept(v)
		if !ok {
			return n, false
		}
		n.Variables[i] = node.(*VariableAssignment)
	}
	stmt, ok := n.Stmt.Accept(v)
	if !ok {
		return n, false
	}
	n.Stmt = stmt.(StmtNode)
	return v.Leave(n)
}

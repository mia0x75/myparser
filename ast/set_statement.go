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
	if n == nil || !nodePresent(n.Stmt) {
		return errors.New("restore SetStatementStmt: missing statement")
	}
	ctx.WriteKeyword("SET STATEMENT ")
	for i, variable := range n.Variables {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := restoreSetStatementVariable(ctx, variable); err != nil {
			return errors.Annotatef(err, "An error occurred while restore SetStatementStmt.Variables[%d]", i)
		}
	}
	ctx.WriteKeyword(" FOR ")
	if err := n.Stmt.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore SetStatementStmt.Stmt")
	}
	return ctx.Err()
}

func restoreSetStatementVariable(ctx *format.RestoreCtx, variable *VariableAssignment) error {
	if variable == nil {
		return errors.New("restore SET STATEMENT variable: missing assignment")
	}
	if variable.Target == nil || variable.Target.Kind != VariableTargetSystem {
		return variable.Restore(ctx)
	}
	target := *variable.Target
	target.Kind = VariableTargetLocal
	target.Prefix = VariablePrefixNone
	target.Scope = VariableScopeImplicit
	target.ScopeExplicit = false
	copy := *variable
	copy.Target = &target
	return copy.Restore(ctx)
}

// Accept implements Node interface.
func (n *SetStatementStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*SetStatementStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	for i, variable := range n.Variables {
		updated, ok := acceptNode[*VariableAssignment](v, variable)
		if !ok {
			return n, false
		}
		n.Variables[i] = updated
	}
	stmt, ok := acceptNode[StmtNode](v, n.Stmt)
	if !ok {
		return n, false
	}
	n.Stmt = stmt
	return v.Leave(n)
}

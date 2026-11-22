package ast

import (
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"

	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
)

// ReplicationOption is a dialect keyword=value replication setting. Unlike a
// column assignment, its target is a fixed lexical keyword, not a writable
// target expression.
type ReplicationOption struct {
	pkast.BaseNode

	Name lex.Ident
	Expr ExprNode
}

var _ Node = (*ReplicationOption)(nil)

// Restore implements Node interface.
func (n *ReplicationOption) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !n.Name.Present() || n.Expr == nil {
		return errors.New("Incomplete replication option")
	}
	ctx.WriteKeyword(n.Name.Name)
	ctx.WritePlain("=")
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore ReplicationOption.Expr")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ReplicationOption) Accept(v Visitor) (Node, bool) {
	entered, skipChildren := v.Enter(n)
	outer, ok := entered.(*ReplicationOption)
	if !ok || outer == nil {
		return entered, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	if outer.Expr != nil {
		node, ok := outer.Expr.Accept(v)
		if !ok {
			return outer, false
		}
		expr, ok := node.(ExprNode)
		if !ok || expr == nil {
			return outer, false
		}
		outer.Expr = expr
	}
	return v.Leave(outer)
}

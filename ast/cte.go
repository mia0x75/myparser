package ast

import (
	"fmt"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"

	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
)

type CommonTableExpression = pkast.CommonTableExpression
type CTENode = pkast.CTENode

// WithClause retains MariaDB's CYCLE extension while sharing standard CTE data.
type WithClause struct {
	pkast.BaseWithClause
	// CycleColumns is MariaDB's optional `CYCLE column_list RESTRICT` clause.
	CycleColumns []lex.Ident
}

func (n *WithClause) WithClauseCore() *pkast.BaseWithClause {
	if n == nil {
		return nil
	}
	return &n.BaseWithClause
}

func (n *WithClause) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return fmt.Errorf("restore with clause: nil clause")
	}
	ctx.WriteKeyword("WITH ")
	if n.IsRecursive {
		ctx.WriteKeyword("RECURSIVE ")
	}
	for i, cte := range n.CTEs {
		if !nodePresent(cte) || cte.CTECore() == nil {
			return fmt.Errorf("restore with clause CTE %d: nil CTE", i)
		}
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := cte.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore WithClause.CTEs[%d]", i)
		}
	}
	if len(n.CycleColumns) > 0 {
		ctx.WriteKeyword(" CYCLE ")
		for i, column := range n.CycleColumns {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			ctx.WriteIdent(column)
		}
		ctx.WriteKeyword(" RESTRICT")
	}
	return ctx.Err()
}

func (n *WithClause) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	clause, ok := entered.(*WithClause)
	if !ok || clause == nil {
		return entered, false
	}
	if skip {
		return v.Leave(clause)
	}
	for i, cte := range clause.CTEs {
		if !nodePresent(cte) || cte.CTECore() == nil {
			return clause, false
		}
		updated, ok := cte.Accept(v)
		if !ok {
			return clause, false
		}
		replacement, ok := nodeAs[CTENode](updated)
		if !ok || replacement.CTECore() == nil {
			return clause, false
		}
		clause.CTEs[i] = replacement
	}
	return v.Leave(clause)
}

var _ Node = (*WithClause)(nil)

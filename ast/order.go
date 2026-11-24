package ast

import (
	"fmt"

	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
)

type OrderItem = pkast.OrderItem
type OrderItemNode = pkast.OrderItemNode

type OrderByClauseNode = pkast.OrderByClauseNode

// OrderByClause retains the outer clause identity and legacy ForUnion field.
// Set-operation tail ownership is represented by the containing query node.
type OrderByClause struct {
	pkast.BaseOrderByClause
	ForUnion bool
}

func (n *OrderByClause) OrderByClauseCore() *pkast.BaseOrderByClause {
	if n == nil {
		return nil
	}
	return &n.BaseOrderByClause
}

var _ OrderByClauseNode = (*OrderByClause)(nil)

func (n *OrderByClause) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return fmt.Errorf("restore order by: nil clause")
	}
	ctx.WriteKeyword("ORDER BY ")
	for i, item := range n.Items {
		if item == nil || item.OrderItemCore() == nil {
			return fmt.Errorf("restore order by item %d: nil item", i)
		}
		if i != 0 {
			ctx.WritePlain(",")
		}
		if err := item.Restore(ctx); err != nil {
			return fmt.Errorf("restore order by item %d: %w", i, err)
		}
	}
	return ctx.Err()
}

func (n *OrderByClause) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	clause, ok := entered.(*OrderByClause)
	if !ok || clause == nil {
		return entered, false
	}
	if skip {
		return v.Leave(clause)
	}
	for i, item := range clause.Items {
		if item == nil || item.OrderItemCore() == nil {
			return clause, false
		}
		updated, ok := item.Accept(v)
		if !ok {
			return clause, false
		}
		replacement, ok := updated.(OrderItemNode)
		if !ok || replacement == nil || replacement.OrderItemCore() == nil {
			return clause, false
		}
		clause.Items[i] = replacement
	}
	return v.Leave(clause)
}

var _ Node = (*OrderByClause)(nil)

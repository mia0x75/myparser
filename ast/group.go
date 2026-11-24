package ast

import (
	"fmt"

	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/model"
	pkast "github.com/mia0x75/parserkit/ast"
)

type GroupItemNode = pkast.GroupItemNode

// GroupItem keeps MySQL's grouping direction outside the shared grouping core.
type GroupItem struct {
	pkast.BaseGroupItem
	Desc bool
}

func (n *GroupItem) GroupItemCore() *pkast.BaseGroupItem {
	if n == nil {
		return nil
	}
	return &n.BaseGroupItem
}

func (n *GroupItem) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Expr == nil {
		return fmt.Errorf("restore group item: missing expression")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return fmt.Errorf("restore group item expression: %w", err)
	}
	if n.Desc {
		ctx.WriteKeyword(" DESC")
	}
	return ctx.Err()
}

func (n *GroupItem) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	item, ok := entered.(*GroupItem)
	if !ok || item == nil {
		return entered, false
	}
	if skip {
		return v.Leave(item)
	}
	if item.Expr != nil {
		updated, ok := item.Expr.Accept(v)
		if !ok {
			return item, false
		}
		expr, ok := updated.(ExprNode)
		if !ok || !nodePresent(expr) {
			return item, false
		}
		item.Expr = expr
	}
	return v.Leave(item)
}

// GroupByClause retains MySQL's suffixes, including view CHECK OPTION syntax.
type GroupByClause struct {
	pkast.BaseGroupByClause
	// WithRollup and WithCube preserve the frozen grammar's suffix choices.
	WithRollup bool
	WithCube   bool
	// CheckOption is meaningful only when a view CHECK OPTION suffix is present.
	HasCheckOption bool
	CheckOption    model.ViewCheckOption
}

func (n *GroupByClause) GroupByClauseCore() *pkast.BaseGroupByClause {
	if n == nil {
		return nil
	}
	return &n.BaseGroupByClause
}

func (n *GroupByClause) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return fmt.Errorf("restore group by: nil clause")
	}
	ctx.WriteKeyword("GROUP BY ")
	for i, item := range n.Items {
		if item == nil || item.GroupItemCore() == nil {
			return fmt.Errorf("restore group by item %d: nil item", i)
		}
		if i != 0 {
			ctx.WritePlain(",")
		}
		if err := item.Restore(ctx); err != nil {
			return fmt.Errorf("restore group by item %d: %w", i, err)
		}
	}
	if n.WithRollup {
		ctx.WriteKeyword(" WITH ROLLUP")
	}
	if n.WithCube {
		ctx.WriteKeyword(" WITH CUBE")
	}
	if n.HasCheckOption {
		if n.CheckOption == model.CheckOptionLocal {
			ctx.WriteKeyword(" WITH LOCAL CHECK OPTION")
		} else {
			ctx.WriteKeyword(" WITH CASCADED CHECK OPTION")
		}
	}
	return ctx.Err()
}

func (n *GroupByClause) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	clause, ok := entered.(*GroupByClause)
	if !ok || clause == nil {
		return entered, false
	}
	if skip {
		return v.Leave(clause)
	}
	for i, item := range clause.Items {
		if item == nil || item.GroupItemCore() == nil {
			return clause, false
		}
		updated, ok := item.Accept(v)
		if !ok {
			return clause, false
		}
		replacement, ok := updated.(GroupItemNode)
		if !ok || replacement == nil || replacement.GroupItemCore() == nil {
			return clause, false
		}
		clause.Items[i] = replacement
	}
	return v.Leave(clause)
}

var (
	_ GroupItemNode = (*GroupItem)(nil)
	_ Node          = (*GroupByClause)(nil)
)

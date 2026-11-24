package ast

import (
	"fmt"

	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
)

// PartitionItem is a window partition expression. Desc preserves the modifier
// accepted by the frozen MySQL grammar; it does not make this an ordering item.
type PartitionItem struct {
	pkast.BaseWindowPartitionItem
	Desc bool
}

func (n *PartitionItem) WindowPartitionItemCore() *pkast.BaseWindowPartitionItem {
	if n == nil {
		return nil
	}
	return &n.BaseWindowPartitionItem
}

func (n *PartitionItem) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Expr == nil {
		return fmt.Errorf("restore partition item: missing expression")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return fmt.Errorf("restore partition item expression: %w", err)
	}
	if n.Desc {
		ctx.WriteKeyword(" DESC")
	}
	return ctx.Err()
}

func (n *PartitionItem) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	item, ok := entered.(*PartitionItem)
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

var _ Node = (*PartitionItem)(nil)
var _ pkast.WindowPartitionItemNode = (*PartitionItem)(nil)

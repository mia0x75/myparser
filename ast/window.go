package ast

import (
	"fmt"

	"github.com/mia0x75/myparser/errors"
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
	if item.Expr == nil {
		// Restore requires a partition expression; fail early so Accept
		// success still means the tree is structurally valid.
		return item, false
	}
	updated, ok := item.Expr.Accept(v)
	if !ok {
		return item, false
	}
	expr, ok := updated.(ExprNode)
	if !ok || !nodePresent(expr) {
		return item, false
	}
	item.Expr = expr
	return v.Leave(item)
}

var _ Node = (*PartitionItem)(nil)
var _ pkast.WindowPartitionItemNode = (*PartitionItem)(nil)

// WindowSpec is the specification of a window.
type WindowSpec struct {
	pkast.BaseWindowSpec
}

func (n *WindowSpec) WindowSpecCore() *pkast.BaseWindowSpec {
	if n == nil {
		return nil
	}
	return &n.BaseWindowSpec
}

// Restore implements Node interface.
func (n *WindowSpec) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return nilRestoreError("WindowSpec")
	}
	if n.Name.Present() {
		ctx.WriteIdent(n.Name)
		if n.OnlyAlias {
			return ctx.Err()
		}
		ctx.WriteKeyword(" AS ")
	}
	ctx.WritePlain("(")
	sep := ""
	if n.Ref.Present() {
		ctx.WriteIdent(n.Ref)
		sep = " "
	}
	if n.PartitionBy != nil {
		if n.PartitionBy.WindowPartitionClauseCore() == nil {
			return errors.New("restore window spec: nil partition clause")
		}
		ctx.WritePlain(sep)
		if err := n.PartitionBy.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore WindowSpec.PartitionBy")
		}
		sep = " "
	}
	if n.OrderBy != nil {
		if n.OrderBy.OrderByClauseCore() == nil {
			return errors.New("restore window spec: nil order by clause")
		}
		ctx.WritePlain(sep)
		if err := n.OrderBy.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore WindowSpec.OrderBy")
		}
		sep = " "
	}
	if n.Frame != nil {
		if n.Frame.WindowFrameClauseCore() == nil {
			return errors.New("restore window spec: nil frame clause")
		}
		ctx.WritePlain(sep)
		if err := n.Frame.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore WindowSpec.Frame")
		}
	}
	ctx.WritePlain(")")

	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *WindowSpec) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	var ok bool
	n, ok = newNode.(*WindowSpec)
	if !ok || n == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(n)
	}
	if n.PartitionBy != nil {
		if n.PartitionBy.WindowPartitionClauseCore() == nil {
			return n, false
		}
		node, ok := n.PartitionBy.Accept(v)
		if !ok {
			return n, false
		}
		partition, ok := node.(pkast.WindowPartitionClauseNode)
		if !ok || partition == nil || partition.WindowPartitionClauseCore() == nil {
			return n, false
		}
		n.PartitionBy = partition
	}
	if n.OrderBy != nil {
		if n.OrderBy.OrderByClauseCore() == nil {
			return n, false
		}
		node, ok := n.OrderBy.Accept(v)
		if !ok {
			return n, false
		}
		orderBy, ok := node.(pkast.OrderByClauseNode)
		if !ok || orderBy == nil || orderBy.OrderByClauseCore() == nil {
			return n, false
		}
		n.OrderBy = orderBy
	}
	if n.Frame != nil {
		if n.Frame.WindowFrameClauseCore() == nil {
			return n, false
		}
		node, ok := n.Frame.Accept(v)
		if !ok {
			return n, false
		}
		frame, ok := node.(pkast.WindowFrameClauseNode)
		if !ok || frame == nil || frame.WindowFrameClauseCore() == nil {
			return n, false
		}
		n.Frame = frame
	}
	return v.Leave(n)
}

// PartitionByClause represents partition by clause.
type PartitionByClause struct {
	pkast.BaseWindowPartitionClause
}

func (n *PartitionByClause) WindowPartitionClauseCore() *pkast.BaseWindowPartitionClause {
	if n == nil {
		return nil
	}
	return &n.BaseWindowPartitionClause
}

// Restore implements Node interface.
func (n *PartitionByClause) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore partition by: nil clause")
	}
	ctx.WriteKeyword("PARTITION BY ")
	for i, v := range n.Items {
		if v == nil || v.WindowPartitionItemCore() == nil {
			return errors.Errorf("restore partition by item %d: nil item", i)
		}
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore PartitionByClause.Items[%d]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *PartitionByClause) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	clause, ok := newNode.(*PartitionByClause)
	if !ok || clause == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(newNode)
	}
	n = clause
	for i, val := range n.Items {
		if val == nil || val.WindowPartitionItemCore() == nil {
			return n, false
		}
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := node.(pkast.WindowPartitionItemNode)
		if !ok || replacement == nil || replacement.WindowPartitionItemCore() == nil {
			return n, false
		}
		n.Items[i] = replacement
	}
	return v.Leave(n)
}

// FrameType is the type of window function frame.
type FrameType = pkast.WindowFrameType

// Window function frame types.
// MySQL only supports `ROWS` and `RANGES`.
const (
	Rows   = pkast.WindowFrameRows
	Ranges = pkast.WindowFrameRange
	Groups = pkast.WindowFrameGroups
)

// FrameClause represents frame clause.
type FrameClause struct {
	pkast.BaseWindowFrameClause
	// Source offsets locate dialect diagnostics without reinterpreting expressions.
	TypeOffset      int
	ExclusionOffset int
}

// Restore implements Node interface.
func (n *FrameClause) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return nilRestoreError("FrameClause")
	}
	if n.Extent.Start == nil || n.Extent.Start.WindowFrameBoundCore() == nil ||
		n.Extent.End == nil || n.Extent.End.WindowFrameBoundCore() == nil {
		return errors.New("window frame requires start and end bounds")
	}
	switch n.Type {
	case Rows:
		ctx.WriteKeyword("ROWS")
	case Ranges:
		ctx.WriteKeyword("RANGE")
	case Groups:
		ctx.WriteKeyword("GROUPS")
	default:
		return errors.New("Unsupported window function frame type")
	}
	ctx.WriteKeyword(" BETWEEN ")
	if err := n.Extent.Start.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore FrameClause.Extent.Start")
	}
	ctx.WriteKeyword(" AND ")
	if err := n.Extent.End.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore FrameClause.Extent.End")
	}
	switch n.Exclusion {
	case pkast.WindowFrameExcludeNone:
	case pkast.WindowFrameExcludeCurrentRow:
		ctx.WriteKeyword(" EXCLUDE CURRENT ROW")
	case pkast.WindowFrameExcludeGroup:
		ctx.WriteKeyword(" EXCLUDE GROUP")
	case pkast.WindowFrameExcludeTies:
		ctx.WriteKeyword(" EXCLUDE TIES")
	case pkast.WindowFrameExcludeNoOthers:
		ctx.WriteKeyword(" EXCLUDE NO OTHERS")
	default:
		return errors.New("Unsupported window function frame exclusion")
	}

	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *FrameClause) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	var ok bool
	n, ok = newNode.(*FrameClause)
	if !ok || n == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(n)
	}
	if n.Extent.Start != nil {
		if n.Extent.Start.WindowFrameBoundCore() == nil {
			return n, false
		}
		node, ok := n.Extent.Start.Accept(v)
		if !ok {
			return n, false
		}
		bound, ok := node.(pkast.WindowFrameBoundNode)
		if !ok || bound == nil || bound.WindowFrameBoundCore() == nil {
			return n, false
		}
		n.Extent.Start = bound
	}
	if n.Extent.End != nil {
		if n.Extent.End.WindowFrameBoundCore() == nil {
			return n, false
		}
		node, ok := n.Extent.End.Accept(v)
		if !ok {
			return n, false
		}
		bound, ok := node.(pkast.WindowFrameBoundNode)
		if !ok || bound == nil || bound.WindowFrameBoundCore() == nil {
			return n, false
		}
		n.Extent.End = bound
	}
	return v.Leave(n)
}

// FrameExtent represents frame extent.
type FrameExtent = pkast.WindowFrameExtent

// BoundType is the type of window function frame bound.
type BoundType = pkast.WindowFrameBoundType

// Frame bound types.
const (
	Following  = pkast.WindowFrameFollowing
	Preceding  = pkast.WindowFramePreceding
	CurrentRow = pkast.WindowFrameCurrentRow
)

// FrameBound represents frame bound.
type FrameBound struct {
	pkast.BaseWindowFrameBound
	ExprOffset int
}

func (n *FrameBound) WindowFrameBoundCore() *pkast.BaseWindowFrameBound {
	if n == nil {
		return nil
	}
	return &n.BaseWindowFrameBound
}

// Restore implements Node interface.
func (n *FrameBound) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return nilRestoreError("FrameBound")
	}
	if n.Type != CurrentRow && n.Type != Preceding && n.Type != Following {
		return errors.New("unsupported window frame bound direction")
	}
	if n.Unit.Present() && !validTimeUnit(n.Unit.Name) {
		return errors.New("invalid frame time unit")
	}
	if n.Unit.Present() && n.Expr == nil {
		return errors.New("frame time unit requires an expression")
	}
	if n.Unit.Present() && n.Type != Preceding && n.Type != Following {
		return errors.New("frame time unit requires a preceding or following bound")
	}
	if n.Unbounded {
		ctx.WriteKeyword("UNBOUNDED")
	}
	switch n.Type {
	case CurrentRow:
		ctx.WriteKeyword("CURRENT ROW")
	case Preceding, Following:
		if n.Unit.Present() {
			ctx.WriteKeyword("INTERVAL ")
		}
		if n.Expr != nil {
			if err := n.Expr.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore FrameBound.Expr")
			}
		}
		if n.Unit.Present() {
			ctx.WritePlain(" ")
			ctx.WriteKeyword(n.Unit.Name)
		}
		if n.Type == Preceding {
			ctx.WriteKeyword(" PRECEDING")
		} else {
			ctx.WriteKeyword(" FOLLOWING")
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *FrameBound) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	var ok bool
	n, ok = newNode.(*FrameBound)
	if !ok || n == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(n)
	}
	if n.Expr != nil {
		expr, ok := acceptNode[ExprNode](v, n.Expr)
		if !ok {
			return n, false
		}
		n.Expr = expr
	}
	return v.Leave(n)
}

var (
	_ pkast.WindowFrameClauseNode = (*FrameClause)(nil)
	_ pkast.WindowFrameBoundNode  = (*FrameBound)(nil)
)

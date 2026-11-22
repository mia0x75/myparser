package ast

import (
	"fmt"

	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
)

// JoinExprNode exposes join syntax shared with other dialects.
type JoinExprNode = pkast.JoinExprNode

// JoinExpr keeps MySQL's STRAIGHT_JOIN outside the standard shared relation.
type JoinExpr struct {
	pkast.BaseJoinExpr
	StraightJoin bool
}

// JoinExprCore returns the single shared value without replacing this outer.
func (n *JoinExpr) JoinExprCore() *pkast.BaseJoinExpr {
	if n == nil {
		return nil
	}
	return &n.BaseJoinExpr
}

// Restore preserves the existing join parentheses and the caller's nesting level.
func (n *JoinExpr) Restore(ctx *format.RestoreCtx) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil || !joinRelationPresent(n.Left) {
		return fmt.Errorf("restore join: missing left relation")
	}
	if n.Right != nil && !joinRelationPresent(n.Right) {
		return fmt.Errorf("restore join: missing right relation")
	}
	switch n.Type {
	case pkast.JoinTypeUnknown, pkast.JoinTypeInner, pkast.JoinTypeLeft, pkast.JoinTypeRight, pkast.JoinTypeCross:
	default:
		return fmt.Errorf("restore join: unsupported MySQL join type %d", n.Type)
	}
	level := ctx.JoinLevel
	defer func() {
		ctx.JoinLevel = level
		if level != 0 {
			ctx.WritePlain(")")
			if err == nil {
				err = ctx.Err()
			}
		}
	}()
	if level != 0 {
		ctx.WritePlain("(")
	}
	ctx.JoinLevel = level + 1
	if err := n.Left.Restore(ctx); err != nil {
		return fmt.Errorf("restore JoinExpr.Left: %w", err)
	}
	ctx.JoinLevel = level
	if n.Right == nil {
		return ctx.Err()
	}
	if n.Natural {
		ctx.WriteKeyword(" NATURAL")
	}
	switch n.Type {
	case pkast.JoinTypeLeft:
		ctx.WriteKeyword(" LEFT")
	case pkast.JoinTypeRight:
		ctx.WriteKeyword(" RIGHT")
	case pkast.JoinTypeCross:
		ctx.WriteKeyword(" CROSS")
	}
	if n.StraightJoin {
		ctx.WriteKeyword(" STRAIGHT_JOIN ")
	} else {
		ctx.WriteKeyword(" JOIN ")
	}
	ctx.JoinLevel = level + 1
	if err := n.Right.Restore(ctx); err != nil {
		return fmt.Errorf("restore JoinExpr.Right: %w", err)
	}
	ctx.JoinLevel = level
	if n.On != nil {
		ctx.WriteKeyword(" ON ")
		if err := n.On.Restore(ctx); err != nil {
			return fmt.Errorf("restore JoinExpr.On: %w", err)
		}
	}
	if len(n.Using) != 0 {
		ctx.WriteKeyword(" USING ")
		ctx.WritePlain("(")
		for i, name := range n.Using {
			if i != 0 {
				ctx.WritePlain(",")
			}
			if err := restoreQualifiedName(ctx, name); err != nil {
				return fmt.Errorf("restore JoinExpr.Using: %w", err)
			}
		}
		ctx.WritePlain(")")
	}
	return ctx.Err()
}

// Accept walks Left, Right, then the actual ON expression on this dialect outer.
func (n *JoinExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*JoinExpr)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	if !joinRelationPresent(outer.Left) {
		return outer, false
	}
	updated, ok := outer.Left.Accept(v)
	if !ok {
		return outer, false
	}
	left, ok := updated.(ResultSetNode)
	if !ok || !joinRelationPresent(left) {
		return outer, false
	}
	outer.Left = left
	if outer.Right != nil {
		if !joinRelationPresent(outer.Right) {
			return outer, false
		}
		updated, ok = outer.Right.Accept(v)
		if !ok {
			return outer, false
		}
		right, ok := updated.(ResultSetNode)
		if !ok || !joinRelationPresent(right) {
			return outer, false
		}
		outer.Right = right
	}
	if outer.On != nil {
		updated, ok = outer.On.Accept(v)
		if !ok {
			return outer, false
		}
		on, ok := updated.(ExprNode)
		if !ok || on == nil {
			return outer, false
		}
		outer.On = on
	}
	return v.Leave(outer)
}

var (
	_ ResultSetNode = (*JoinExpr)(nil)
	_ JoinExprNode  = (*JoinExpr)(nil)
)

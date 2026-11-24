package ast

import (
	"fmt"

	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/mysql"
	pkast "github.com/mia0x75/parserkit/ast"
)

// DeleteStmt preserves MySQL's target list and choice of multi-table syntax.
type DeleteStmt struct {
	pkast.BaseDeleteStmt
	With         *WithClause
	Targets      []*TableName
	Priority     mysql.PriorityEnum
	IgnoreErr    bool
	Quick        bool
	IsMultiTable bool
	BeforeFrom   bool
	TableHints   []*OptimizerHint
}

func (n *DeleteStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return fmt.Errorf("restore DeleteStmt: nil statement")
	}
	if n.IsMultiTable {
		if n.Table != nil || !resultSetPresent(n.Using) || len(n.Targets) == 0 {
			return fmt.Errorf("restore DeleteStmt: inconsistent multi-table mode")
		}
		for i, target := range n.Targets {
			if target == nil {
				return fmt.Errorf("restore DeleteStmt.Targets[%d]: nil target", i)
			}
		}
	} else if !resultSetPresent(n.Table) || n.Using != nil || len(n.Targets) != 0 || n.BeforeFrom {
		return fmt.Errorf("restore DeleteStmt: inconsistent single-table mode")
	}
	if n.With != nil {
		if err := n.With.Restore(ctx); err != nil {
			return fmt.Errorf("restore DeleteStmt.With: %w", err)
		}
		ctx.WritePlain(" ")
	}
	ctx.WriteKeyword("DELETE ")
	if len(n.TableHints) > 0 {
		ctx.WritePlain("/*+ ")
		for i, hint := range n.TableHints {
			if hint == nil {
				return fmt.Errorf("restore DeleteStmt.TableHints[%d]: nil hint", i)
			}
			if err := hint.Restore(ctx); err != nil {
				return fmt.Errorf("restore DeleteStmt.TableHints[%d]: %w", i, err)
			}
		}
		ctx.WritePlain("*/ ")
	}
	if err := n.Priority.Restore(ctx); err != nil {
		return err
	}
	if n.Priority != mysql.NoPriority {
		ctx.WritePlain(" ")
	}
	if n.Quick {
		ctx.WriteKeyword("QUICK ")
	}
	if n.IgnoreErr {
		ctx.WriteKeyword("IGNORE ")
	}
	if n.IsMultiTable {
		if !n.BeforeFrom {
			ctx.WriteKeyword("FROM ")
		}
		for i, target := range n.Targets {
			if i != 0 {
				ctx.WritePlain(",")
			}
			if err := target.Restore(ctx); err != nil {
				return fmt.Errorf("restore DeleteStmt.Targets[%d]: %w", i, err)
			}
		}
		if n.BeforeFrom {
			ctx.WriteKeyword(" FROM ")
		} else {
			ctx.WriteKeyword(" USING ")
		}
		if err := restoreDMLRelation(ctx, n.Using); err != nil {
			return fmt.Errorf("restore DeleteStmt.Using: %w", err)
		}
	} else {
		ctx.WriteKeyword("FROM ")
		if err := restoreDMLRelation(ctx, n.Table); err != nil {
			return fmt.Errorf("restore DeleteStmt.Table: %w", err)
		}
	}
	if n.Where != nil {
		ctx.WriteKeyword(" WHERE ")
		if err := n.Where.Restore(ctx); err != nil {
			return fmt.Errorf("restore DeleteStmt.Where: %w", err)
		}
	}
	if n.OrderBy != nil {
		if n.OrderBy.OrderByClauseCore() == nil {
			return fmt.Errorf("restore DeleteStmt.OrderBy: nil clause")
		}
		ctx.WritePlain(" ")
		if err := n.OrderBy.Restore(ctx); err != nil {
			return fmt.Errorf("restore DeleteStmt.OrderBy: %w", err)
		}
	}
	if n.Limit != nil {
		if n.Limit.LimitCore() == nil {
			return fmt.Errorf("restore DeleteStmt.Limit: nil clause")
		}
		ctx.WritePlain(" ")
		if err := n.Limit.Restore(ctx); err != nil {
			return fmt.Errorf("restore DeleteStmt.Limit: %w", err)
		}
	}
	if n.Returning != nil {
		ctx.WriteKeyword(" RETURNING ")
		if err := n.Returning.Restore(ctx); err != nil {
			return fmt.Errorf("restore DeleteStmt.Returning: %w", err)
		}
	}
	return ctx.Err()
}

func (n *DeleteStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*DeleteStmt)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	n = outer
	if n.With != nil {
		updated, ok := n.With.Accept(v)
		if !ok {
			return n, false
		}
		with, ok := updated.(*WithClause)
		if !ok || with == nil {
			return n, false
		}
		n.With = with
	}
	if !acceptOptimizerHints(v, n.TableHints) {
		return n, false
	}
	if n.Table != nil {
		if !resultSetPresent(n.Table) {
			return n, false
		}
		updated, ok := n.Table.Accept(v)
		if !ok {
			return n, false
		}
		table, ok := updated.(ResultSetNode)
		if !ok || !resultSetPresent(table) {
			return n, false
		}
		n.Table = table
	}
	if n.Using != nil {
		if !resultSetPresent(n.Using) {
			return n, false
		}
		updated, ok := n.Using.Accept(v)
		if !ok {
			return n, false
		}
		source, ok := updated.(ResultSetNode)
		if !ok || !resultSetPresent(source) {
			return n, false
		}
		n.Using = source
	}
	for i, target := range n.Targets {
		if target == nil {
			return n, false
		}
		updated, ok := target.Accept(v)
		if !ok {
			return n, false
		}
		table, ok := updated.(*TableName)
		if !ok || table == nil {
			return n, false
		}
		n.Targets[i] = table
	}
	if n.Where != nil {
		updated, ok := n.Where.Accept(v)
		if !ok {
			return n, false
		}
		expr, ok := updated.(ExprNode)
		if !ok || !nodePresent(expr) {
			return n, false
		}
		n.Where = expr
	}
	if n.OrderBy != nil {
		if n.OrderBy.OrderByClauseCore() == nil {
			return n, false
		}
		updated, ok := n.OrderBy.Accept(v)
		if !ok {
			return n, false
		}
		clause, ok := updated.(OrderByClauseNode)
		if !ok || clause == nil || clause.OrderByClauseCore() == nil {
			return n, false
		}
		n.OrderBy = clause
	}
	if n.Limit != nil {
		if n.Limit.LimitCore() == nil {
			return n, false
		}
		updated, ok := n.Limit.Accept(v)
		if !ok {
			return n, false
		}
		clause, ok := updated.(LimitNode)
		if !ok || clause == nil || clause.LimitCore() == nil {
			return n, false
		}
		n.Limit = clause
	}
	if n.Returning != nil {
		updated, ok := n.Returning.Accept(v)
		if !ok {
			return n, false
		}
		fields, ok := updated.(*FieldList)
		if !ok || fields == nil {
			return n, false
		}
		n.Returning = fields
	}
	return v.Leave(n)
}

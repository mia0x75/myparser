package ast

import (
	"fmt"

	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/mysql"
	pkast "github.com/mia0x75/parserkit/ast"
)

// UpdateStmt keeps MySQL modifiers outside the shared update syntax.
type UpdateStmt struct {
	pkast.BaseUpdateStmt
	With          *WithClause
	Priority      mysql.PriorityEnum
	IgnoreErr     bool
	MultipleTable bool
	TableHints    []*OptimizerHint
}

func (n *UpdateStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil || !resultSetPresent(n.Table) || len(n.Assignments) == 0 {
		return fmt.Errorf("restore UpdateStmt: missing table or assignments")
	}
	for i, assignment := range n.Assignments {
		if assignment == nil {
			return fmt.Errorf("restore UpdateStmt.Assignments[%d]: nil assignment", i)
		}
	}
	if n.With != nil {
		if err := n.With.Restore(ctx); err != nil {
			return fmt.Errorf("restore UpdateStmt.With: %w", err)
		}
		ctx.WritePlain(" ")
	}
	ctx.WriteKeyword("UPDATE ")
	if len(n.TableHints) > 0 {
		ctx.WritePlain("/*+ ")
		for i, hint := range n.TableHints {
			if hint == nil {
				return fmt.Errorf("restore UpdateStmt.TableHints[%d]: nil hint", i)
			}
			if err := hint.Restore(ctx); err != nil {
				return fmt.Errorf("restore UpdateStmt.TableHints[%d]: %w", i, err)
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
	if n.IgnoreErr {
		ctx.WriteKeyword("IGNORE ")
	}
	if err := restoreDMLRelation(ctx, n.Table); err != nil {
		return fmt.Errorf("restore UpdateStmt.Table: %w", err)
	}
	ctx.WriteKeyword(" SET ")
	for i, assignment := range n.Assignments {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := assignment.Restore(ctx); err != nil {
			return fmt.Errorf("restore UpdateStmt.Assignments[%d]: %w", i, err)
		}
	}
	if n.Where != nil {
		ctx.WriteKeyword(" WHERE ")
		if err := n.Where.Restore(ctx); err != nil {
			return fmt.Errorf("restore UpdateStmt.Where: %w", err)
		}
	}
	if n.OrderBy != nil {
		if n.OrderBy.OrderByClauseCore() == nil {
			return fmt.Errorf("restore UpdateStmt.OrderBy: nil clause")
		}
		ctx.WritePlain(" ")
		if err := n.OrderBy.Restore(ctx); err != nil {
			return fmt.Errorf("restore UpdateStmt.OrderBy: %w", err)
		}
	}
	if n.Limit != nil {
		if n.Limit.LimitCore() == nil {
			return fmt.Errorf("restore UpdateStmt.Limit: nil clause")
		}
		ctx.WritePlain(" ")
		if err := n.Limit.Restore(ctx); err != nil {
			return fmt.Errorf("restore UpdateStmt.Limit: %w", err)
		}
	}
	if n.Returning != nil {
		ctx.WriteKeyword(" RETURNING ")
		if err := n.Returning.Restore(ctx); err != nil {
			return fmt.Errorf("restore UpdateStmt.Returning: %w", err)
		}
	}
	return ctx.Err()
}

func (n *UpdateStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*UpdateStmt)
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
	for i, assignment := range n.Assignments {
		if assignment == nil {
			return n, false
		}
		updated, ok := assignment.Accept(v)
		if !ok {
			return n, false
		}
		value, ok := updated.(*Assignment)
		if !ok || value == nil {
			return n, false
		}
		n.Assignments[i] = value
	}
	if n.Where != nil {
		updated, ok := n.Where.Accept(v)
		if !ok {
			return n, false
		}
		expr, ok := updated.(ExprNode)
		if !ok || expr == nil {
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

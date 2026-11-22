package ast

import (
	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
)

// SetOprStmt is a MySQL set operation with optional dialect-specific tails.
type SetOprStmt struct {
	pkast.BaseSetOperation
	Tail pkast.BaseQueryTail

	With             *WithClause
	ProcedureAnalyse *ProcedureAnalyseClause
	LockTp           SelectLockType
	LockedTables     []*TableName
	LockWaitPolicy   SelectLockWaitPolicy
	Into             *SelectIntoStatement
}

// QueryTailStmtCore exposes the shared query-wrapper data.
func (n *QueryTailStmt) QueryTailStmtCore() *pkast.BaseQueryTailStmt {
	if n == nil {
		return nil
	}
	return &n.BaseQueryTailStmt
}

// SetOperationCore exposes the shared binary set-operation data.
func (n *SetOprStmt) SetOperationCore() *pkast.BaseSetOperation {
	if n == nil {
		return nil
	}
	return &n.BaseSetOperation
}

// SetOprType is the MySQL spelling of a portable set operation.
type SetOprType = pkast.SetOperationType

const (
	Union        = pkast.SetOperationUnion
	UnionAll     = pkast.SetOperationUnionAll
	Except       = pkast.SetOperationExcept
	ExceptAll    = pkast.SetOperationExceptAll
	Intersect    = pkast.SetOperationIntersect
	IntersectAll = pkast.SetOperationIntersectAll
)

// QueryTailStmt is a parenthesized query level with optional MySQL-only tail
// clauses. The shared ORDER BY and LIMIT data lives in parserkit's base.
type QueryTailStmt struct {
	pkast.BaseQueryTailStmt

	With             *WithClause
	ProcedureAnalyse *ProcedureAnalyseClause
	LockTp           SelectLockType
	LockedTables     []*TableName
	LockWaitPolicy   SelectLockWaitPolicy
	Into             *SelectIntoStatement
}

func restoreQueryTail(ctx *format.RestoreCtx, tail *pkast.BaseQueryTail) error {
	if tail == nil {
		return nil
	}
	if tail.OrderBy != nil {
		if tail.OrderBy.OrderByClauseCore() == nil {
			return errors.New("query tail contains an invalid ORDER BY clause")
		}
		ctx.WritePlain(" ")
		if err := tail.OrderBy.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore query tail ORDER BY")
		}
	}
	if tail.Limit != nil {
		if tail.Limit.LimitCore() == nil {
			return errors.New("query tail contains an invalid LIMIT clause")
		}
		ctx.WritePlain(" ")
		if err := tail.Limit.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore query tail LIMIT")
		}
	}
	return nil
}

func restoreQueryTailExtensions(ctx *format.RestoreCtx, procedure *ProcedureAnalyseClause, lockTp SelectLockType, lockedTables []*TableName, waitPolicy SelectLockWaitPolicy, into *SelectIntoStatement) error {
	if procedure != nil {
		ctx.WritePlain(" ")
		if err := procedure.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore query tail PROCEDURE ANALYSE")
		}
	}
	switch lockTp {
	case SelectLockInShareMode:
		ctx.WriteKeyword(" LOCK ")
		ctx.WriteKeyword(lockTp.String())
	case SelectLockForUpdate:
		ctx.WritePlain(" ")
		ctx.WriteKeyword("FOR UPDATE")
	case SelectLockForShare:
		ctx.WritePlain(" ")
		ctx.WriteKeyword("FOR SHARE")
	}
	if len(lockedTables) > 0 {
		ctx.WriteKeyword(" OF ")
		for i, table := range lockedTables {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if table == nil {
				return errors.Errorf("query tail locked table %d is nil", i)
			}
			if err := table.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore query tail locked table")
			}
		}
	}
	switch waitPolicy {
	case SelectLockWaitNoWait:
		ctx.WriteKeyword(" NOWAIT")
	case SelectLockWaitSkipLocked:
		ctx.WriteKeyword(" SKIP LOCKED")
	}
	if into != nil {
		ctx.WritePlain(" ")
		if err := into.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore query tail INTO")
		}
	}
	return ctx.Err()
}

// Restore implements Node interface.
func (n *QueryTailStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !resultSetPresent(n.Query) {
		return errors.New("An error occurred while restore QueryTailStmt.Query: nil query")
	}
	if n.With != nil {
		if err := n.With.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore QueryTailStmt.With")
		}
		ctx.WritePlain(" ")
	}
	if n.Parenthesized {
		ctx.WritePlain("(")
	}
	if err := n.Query.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore QueryTailStmt.Query")
	}
	if n.Parenthesized {
		ctx.WritePlain(")")
	}
	if err := restoreQueryTail(ctx, &n.Tail); err != nil {
		return err
	}
	if err := restoreQueryTailExtensions(ctx, n.ProcedureAnalyse, n.LockTp, n.LockedTables, n.LockWaitPolicy, n.Into); err != nil {
		return err
	}
	return ctx.Err()
}

// Accept implements Node interface.
func (n *QueryTailStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, ok := newNode.(*QueryTailStmt)
	if !ok || outer == nil {
		return newNode, false
	}
	n = outer
	if skipChildren {
		return v.Leave(n)
	}
	if n.With != nil {
		node, ok := n.With.Accept(v)
		if !ok {
			return n, false
		}
		with, ok := node.(*WithClause)
		if !ok || with == nil {
			return n, false
		}
		n.With = with
	}
	if !visitQueryOperand(v, &n.Query) || !visitQueryTail(v, &n.Tail) {
		return n, false
	}
	if n.ProcedureAnalyse != nil {
		node, ok := n.ProcedureAnalyse.Accept(v)
		if !ok {
			return n, false
		}
		clause, ok := node.(*ProcedureAnalyseClause)
		if !ok || clause == nil {
			return n, false
		}
		n.ProcedureAnalyse = clause
	}
	for i, table := range n.LockedTables {
		if table == nil {
			return n, false
		}
		node, ok := table.Accept(v)
		if !ok {
			return n, false
		}
		table, ok = node.(*TableName)
		if !ok || table == nil {
			return n, false
		}
		n.LockedTables[i] = table
	}
	if n.Into != nil {
		node, ok := n.Into.Accept(v)
		if !ok {
			return n, false
		}
		into, ok := node.(*SelectIntoStatement)
		if !ok || into == nil {
			return n, false
		}
		n.Into = into
	}
	return v.Leave(n)
}

func restoreSetOperationChild(ctx *format.RestoreCtx, child QueryNode) error {
	if !resultSetPresent(child) {
		return errors.New("set operation contains a nil operand")
	}
	if nested, ok := child.(pkast.SetOperationNode); ok {
		core := nested.SetOperationCore()
		if core == nil || core.Operator.String() == "" || core.Left == nil || core.Right == nil {
			return errors.New("set operation contains an incomplete nested operation")
		}
	}
	return child.Restore(ctx)
}

// Restore implements Node interface.
func (n *SetOprStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Left == nil || n.Right == nil {
		return errors.New("An error occurred while restore SetOprStmt: incomplete set operation")
	}
	operator := n.Operator.String()
	if operator == "" {
		return errors.Errorf("restore set operation: invalid operator %d", n.Operator)
	}
	if n.With != nil {
		if err := n.With.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SetOprStmt.With")
		}
		ctx.WritePlain(" ")
	}
	if err := restoreSetOperationChild(ctx, n.Left); err != nil {
		return errors.Annotate(err, "An error occurred while restore SetOprStmt.Left")
	}
	ctx.WriteKeyword(" " + operator + " ")
	if err := restoreSetOperationChild(ctx, n.Right); err != nil {
		return errors.Annotate(err, "An error occurred while restore SetOprStmt.Right")
	}
	if err := restoreQueryTail(ctx, &n.Tail); err != nil {
		return err
	}
	return restoreQueryTailExtensions(ctx, n.ProcedureAnalyse, n.LockTp, n.LockedTables, n.LockWaitPolicy, n.Into)
}

// Accept implements Node interface.
func (n *SetOprStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, ok := newNode.(*SetOprStmt)
	if !ok || outer == nil {
		return newNode, false
	}
	n = outer
	if skipChildren {
		return v.Leave(n)
	}
	if n.With != nil {
		node, ok := n.With.Accept(v)
		if !ok {
			return n, false
		}
		with, ok := node.(*WithClause)
		if !ok || with == nil {
			return n, false
		}
		n.With = with
	}
	if !visitQueryOperand(v, &n.Left) || !visitQueryOperand(v, &n.Right) || !visitQueryTail(v, &n.Tail) {
		return n, false
	}
	if n.ProcedureAnalyse != nil {
		node, ok := n.ProcedureAnalyse.Accept(v)
		if !ok {
			return n, false
		}
		clause, ok := node.(*ProcedureAnalyseClause)
		if !ok || clause == nil {
			return n, false
		}
		n.ProcedureAnalyse = clause
	}
	for i, table := range n.LockedTables {
		if table == nil {
			return n, false
		}
		node, ok := table.Accept(v)
		if !ok {
			return n, false
		}
		table, ok = node.(*TableName)
		if !ok || table == nil {
			return n, false
		}
		n.LockedTables[i] = table
	}
	if n.Into != nil {
		node, ok := n.Into.Accept(v)
		if !ok {
			return n, false
		}
		into, ok := node.(*SelectIntoStatement)
		if !ok || into == nil {
			return n, false
		}
		n.Into = into
	}
	return v.Leave(n)
}

func visitQueryOperand(v Visitor, operand *QueryNode) bool {
	if !resultSetPresent(*operand) {
		return false
	}
	node, ok := (*operand).Accept(v)
	if !ok {
		return false
	}
	query, ok := node.(QueryNode)
	if !ok || !resultSetPresent(query) {
		return false
	}
	*operand = query
	return true
}

func visitQueryTail(v Visitor, tail *pkast.BaseQueryTail) bool {
	if tail.OrderBy != nil {
		if tail.OrderBy.OrderByClauseCore() == nil {
			return false
		}
		node, ok := tail.OrderBy.Accept(v)
		if !ok {
			return false
		}
		clause, ok := node.(pkast.OrderByClauseNode)
		if !ok || clause == nil || clause.OrderByClauseCore() == nil {
			return false
		}
		tail.OrderBy = clause
	}
	if tail.Limit != nil {
		if tail.Limit.LimitCore() == nil {
			return false
		}
		node, ok := tail.Limit.Accept(v)
		if !ok {
			return false
		}
		clause, ok := node.(pkast.LimitNode)
		if !ok || clause == nil || clause.LimitCore() == nil {
			return false
		}
		tail.Limit = clause
	}
	return true
}

var _ DMLNode = (*SetOprStmt)(nil)
var _ QueryNode = (*QueryTailStmt)(nil)
var _ pkast.SetOperationNode = (*SetOprStmt)(nil)
var _ pkast.QueryTailNode = (*QueryTailStmt)(nil)

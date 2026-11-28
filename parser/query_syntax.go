package parser

import (
	"fmt"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/myparser/ast"
)

type setOprPart struct {
	query    ast.QueryNode
	operator ast.SetOprType
}

type setOprParts []setOprPart

// setOprOperandsPaged reports per-operand ORDER BY/LIMIT on a bare SELECT.
// Parenthesized operands arrive as QueryTailStmt, where paging is legal.
func setOprOperandsPaged(parts setOprParts) bool {
	for _, part := range parts {
		if sel, ok := part.query.(*ast.SelectStmt); ok && sel != nil && (sel.OrderBy != nil || sel.Limit != nil) {
			return true
		}
	}
	return false
}

func setOprPrecedence(operator ast.SetOprType) int {
	if operator == ast.Intersect || operator == ast.IntersectAll {
		return 2
	}
	return 1
}

func reduceSetOpr(nodes []ast.QueryNode, operators []ast.SetOprType) ([]ast.QueryNode, []ast.SetOprType) {
	right := nodes[len(nodes)-1]
	left := nodes[len(nodes)-2]
	operator := operators[len(operators)-1]
	nodes = nodes[:len(nodes)-2]
	operators = operators[:len(operators)-1]
	operation := &ast.SetOprStmt{BaseSetOperation: pkast.BaseSetOperation{Left: left, Right: right, Operator: operator}}
	operation.SetSpan(loc.Cover(left.Span(), right.Span()))
	nodes = append(nodes, operation)
	return nodes, operators
}

func buildSetOpr(parts setOprParts) *ast.SetOprStmt {
	if len(parts) < 2 || parts[0].query == nil {
		return nil
	}
	nodes := []ast.QueryNode{parts[0].query}
	operators := make([]ast.SetOprType, 0, len(parts)-1)
	for _, part := range parts[1:] {
		if part.query == nil || part.operator == pkast.SetOperationUnknown {
			return nil
		}
		for len(operators) > 0 && setOprPrecedence(operators[len(operators)-1]) >= setOprPrecedence(part.operator) {
			nodes, operators = reduceSetOpr(nodes, operators)
		}
		nodes = append(nodes, part.query)
		operators = append(operators, part.operator)
	}
	for len(operators) > 0 {
		nodes, operators = reduceSetOpr(nodes, operators)
	}
	if len(nodes) != 1 {
		return nil
	}
	operation, ok := nodes[0].(*ast.SetOprStmt)
	if !ok {
		return nil
	}
	return operation
}

type parenQueryTailShape struct {
	OrderBy *ast.OrderByClause
	Limit   *ast.Limit
	Lock    *ast.SelectLockInfo
	Into    *ast.SelectIntoStatement
}

func (t parenQueryTailShape) pagination() pkast.BaseQueryTail {
	var tail pkast.BaseQueryTail
	if t.OrderBy != nil {
		tail.OrderBy = t.OrderBy
		tail.SetSpan(t.OrderBy.Span())
	}
	if t.Limit != nil {
		tail.Limit = t.Limit
		tail.SetSpan(loc.Cover(tail.Span(), t.Limit.Span()))
	}
	return tail
}

// Each tail keeps its query level so an outer LIMIT never replaces an inner one.
func parenQueryTail(query ast.QueryNode, tail parenQueryTailShape) *ast.QueryTailStmt {
	out := &ast.QueryTailStmt{
		BaseQueryTailStmt: pkast.BaseQueryTailStmt{
			Query:         query,
			Tail:          tail.pagination(),
			Parenthesized: true,
		},
		Into: tail.Into,
	}
	if tail.Lock != nil {
		out.LockTp = tail.Lock.Type
		out.LockedTables = tail.Lock.Tables
		out.LockWaitPolicy = tail.Lock.WaitPolicy
	}
	return out
}

func moveSelectPagination(set *ast.SetOprStmt, last *ast.SelectStmt) error {
	var orderBy *ast.OrderByClause
	if last.OrderBy != nil {
		parsed, ok := last.OrderBy.(*ast.OrderByClause)
		if !ok {
			return fmt.Errorf("unexpected SELECT ORDER BY clause type %T", last.OrderBy)
		}
		orderBy = parsed
	}
	var limit *ast.Limit
	if last.Limit != nil {
		parsed, ok := last.Limit.(*ast.Limit)
		if !ok {
			return fmt.Errorf("unexpected SELECT LIMIT clause type %T", last.Limit)
		}
		limit = parsed
	}
	set.Tail = (parenQueryTailShape{OrderBy: orderBy, Limit: limit}).pagination()
	last.OrderBy, last.Limit = nil, nil
	return nil
}

// addCrossJoin inserts a conditionless join at the left-most leaf of the
// right join tree, without crossing explicit parentheses.
func (st *parseState) addCrossJoin(left, right ast.ResultSetNode) ast.ResultSetNode {
	if j, ok := right.(*ast.JoinExpr); ok && !st.joinGroups[j] {
		j.Left = st.addCrossJoin(left, j.Left)
		j.SetSpan(loc.Cover(left.Span(), j.Span()))
		return j
	}
	return newJoinExpr(loc.Cover(left.Span(), right.Span()), pkast.BaseJoinExpr{
		Left: left, Right: right, Type: pkast.JoinTypeInner,
	})
}

func newJoinExpr(span loc.Span, base pkast.BaseJoinExpr) *ast.JoinExpr {
	n := &ast.JoinExpr{BaseJoinExpr: base}
	n.SetSpan(span)
	return n
}

// withClauseSelect attaches the common table expression clause to the head
// query of a WITH statement without duplicating the clause on a nested level.
func withClauseSelect(clause *ast.WithClause, query ast.QueryNode) ast.QueryNode {
	switch target := query.(type) {
	case *ast.SelectStmt:
		if target.With == nil {
			target.With = clause
			return target
		}
	case *ast.SetOprStmt:
		if target.With == nil {
			target.With = clause
			return target
		}
	case *ast.QueryTailStmt:
		if target.With == nil {
			target.With = clause
			return target
		}
	}
	outer := parenQueryTail(query, parenQueryTailShape{})
	outer.With = clause
	return outer
}

// setOprQueryFromSubSelect preserves explicit grouping for a set-operation
// operand without using a collection node as a fake one-operand query.
func setOprQueryFromSubSelect(sub *ast.SubqueryExpr) ast.QueryNode {
	if sub == nil {
		return nil
	}
	query := sub.Query
	group := &ast.QueryTailStmt{
		BaseQueryTailStmt: pkast.BaseQueryTailStmt{Query: query, Parenthesized: true},
	}
	group.SetSpan(sub.Span())
	return group
}

func newSubqueryExpr(query ast.QueryNode, span loc.Span) *ast.SubqueryExpr {
	sub := &ast.SubqueryExpr{Query: query}
	sub.SetSpan(span)
	return sub
}

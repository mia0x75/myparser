package ast

import "github.com/mia0x75/myparser/format"

// Check known relation pointers without reflecting on dialect extensions.
func resultSetPresent(n ResultSetNode) bool {
	switch n := n.(type) {
	case nil:
		return false
	case *TableSource:
		return n != nil
	case *JoinExpr:
		return n != nil
	case *TableName:
		return n != nil
	case *SelectStmt:
		return n != nil
	case *SetOprStmt:
		return n != nil
	case *QueryTailStmt:
		return n != nil
	case *ValuesStmt:
		return n != nil
	case *TableStmt:
		return n != nil
	case *SubqueryExpr:
		return n != nil
	case TableSourceNode:
		return n.TableSourceCore() != nil
	case JoinExprNode:
		return n.JoinExprCore() != nil
	default:
		return true
	}
}

func joinRelationPresent(n ResultSetNode) bool {
	switch n := n.(type) {
	case nil:
		return false
	case *TableSource:
		return n != nil
	case *JoinExpr:
		return n != nil
	default:
		return resultSetPresent(n)
	}
}

func restoreDMLRelation(ctx *format.RestoreCtx, relation ResultSetNode) error {
	// A DML relation is a fresh join tree, like a SELECT TableRefsClause.
	level := ctx.JoinLevel
	ctx.JoinLevel = 0
	defer func() { ctx.JoinLevel = level }()
	return relation.Restore(ctx)
}

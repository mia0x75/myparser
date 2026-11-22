package myparser

import (
	"github.com/mia0x75/myparser/ast"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/loc"
)

// Only the final DML boundary sheds an empty root, never an internal join.
func dmlRelation(root ast.ResultSetNode, span loc.Span) ast.ResultSetNode {
	if join, ok := root.(*ast.JoinExpr); ok && join != nil && join.Left != nil &&
		join.Right == nil && join.Type == pkast.JoinTypeUnknown && join.On == nil && len(join.Using) == 0 &&
		!join.Natural && !join.StraightJoin {
		root = join.Left
	}
	if root != nil {
		root.SetSpan(span)
	}
	return root
}

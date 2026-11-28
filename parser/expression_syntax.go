package parser

import (
	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/opcode"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/loc"
)

func binaryExpr(op opcode.Op, left, right ast.ExprNode, span loc.Span) *ast.BinaryOperationExpr {
	expr := &ast.BinaryOperationExpr{BaseBinaryExpr: pkast.BaseBinaryExpr{Op: pkast.Operator(op), Left: left, Right: right}}
	expr.SetSpan(span)
	return expr
}

func isNullExpr(expr ast.ExprNode, not bool, span loc.Span) *ast.IsNullExpr {
	node := &ast.IsNullExpr{Expr: expr, Not: not}
	node.SetSpan(span)
	return node
}

func isTruthExpr(expr ast.ExprNode, not bool, truth int64, span loc.Span) *ast.IsTruthExpr {
	node := &ast.IsTruthExpr{Expr: expr, Not: not, True: truth}
	node.SetSpan(span)
	return node
}

func patternInExpr(expr ast.ExprNode, list []ast.ExprNode, subquery ast.ExprNode, not bool, span loc.Span) *ast.PatternInExpr {
	node := &ast.PatternInExpr{Expr: expr, List: list, Sel: subquery, Not: not}
	node.SetSpan(span)
	return node
}

func betweenExpr(expr, left, right ast.ExprNode, not bool, span loc.Span) *ast.BetweenExpr {
	node := &ast.BetweenExpr{Expr: expr, Left: left, Right: right, Not: not}
	node.SetSpan(span)
	return node
}

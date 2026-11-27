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

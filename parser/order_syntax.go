package parser

import (
	"github.com/mia0x75/myparser/ast"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/loc"
)

// syntaxByItem is a reduction value, not an AST node. The frozen ByList rule
// serves three roles; the enclosing clause chooses the final concrete node.
type syntaxByItem struct {
	expr ast.ExprNode
	desc bool
	span loc.Span
}

func syntaxOrderBy(items []*syntaxByItem, span loc.Span) *ast.OrderByClause {
	clause := &ast.OrderByClause{}
	clause.SetSpan(span)
	if items == nil {
		return clause
	}
	clause.Items = make([]ast.OrderItemNode, len(items))
	nodes := make([]ast.OrderItem, len(items))
	for i, item := range items {
		nodes[i].Expr = item.expr
		nodes[i].Desc = item.desc
		nodes[i].SetSpan(item.span)
		clause.Items[i] = &nodes[i]
	}
	return clause
}

func syntaxGroupBy(items []*syntaxByItem, span loc.Span) *ast.GroupByClause {
	clause := &ast.GroupByClause{}
	clause.SetSpan(span)
	if items == nil {
		return clause
	}
	clause.Items = make([]pkast.GroupItemNode, len(items))
	nodes := make([]ast.GroupItem, len(items))
	for i, item := range items {
		nodes[i].Expr = item.expr
		nodes[i].Desc = item.desc
		nodes[i].SetSpan(item.span)
		clause.Items[i] = &nodes[i]
	}
	return clause
}

func syntaxPartitionBy(items []*syntaxByItem, span loc.Span) *ast.PartitionByClause {
	clause := &ast.PartitionByClause{}
	clause.SetSpan(span)
	if items == nil {
		return clause
	}
	clause.Items = make([]pkast.WindowPartitionItemNode, len(items))
	nodes := make([]ast.PartitionItem, len(items))
	for i, item := range items {
		nodes[i].Expr = item.expr
		nodes[i].Desc = item.desc
		nodes[i].SetSpan(item.span)
		clause.Items[i] = &nodes[i]
	}
	return clause
}

package ast_test

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
)

func TestExpressionRestoreRejectsMissingOperands(t *testing.T) {
	for _, node := range []ast.Node{
		&ast.BetweenExpr{}, &ast.BinaryOperationExpr{}, &ast.WhenClause{},
		&ast.SubqueryExpr{}, &ast.CollateExpr{}, &ast.CompareSubqueryExpr{},
		&ast.ExistsSubqueryExpr{}, &ast.PatternInExpr{}, &ast.IsNullExpr{},
		&ast.IsTruthExpr{}, &ast.PatternLikeExpr{}, &ast.ParenthesesExpr{},
		&ast.PatternRegexpExpr{}, &ast.MemberOfExpr{}, &ast.IsJSONExpr{},
		&ast.MatchAgainst{}, &ast.SoundsLikeExpr{}, &ast.UnaryOperationExpr{},
		&ast.ValuesExpr{}, &ast.VariableExpr{},
	} {
		t.Run(fmt.Sprintf("%T", node), func(t *testing.T) {
			typedNil, ok := reflect.Zero(reflect.TypeOf(node)).Interface().(ast.Node)
			if !ok {
				t.Fatal("typed nil does not implement Node")
			}
			for _, invalid := range []ast.Node{node, typedNil} {
				var out strings.Builder
				if err := invalid.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil {
					t.Fatal("Restore accepted missing operand")
				}
			}
		})
	}
}

func TestExpressionRestoreRejectsTypedNilChildren(t *testing.T) {
	value := pureIntLiteral("1")
	var typedNil ast.ExprNode = (*ast.LiteralExpr)(nil)
	for _, tc := range []struct {
		name string
		node ast.Node
	}{
		{"between", &ast.BetweenExpr{Expr: value, Left: value, Right: typedNil}},
		{"binary", &ast.BinaryOperationExpr{BaseBinaryExpr: pkast.BaseBinaryExpr{Left: typedNil, Right: value}}},
		{"when", &ast.WhenClause{Expr: value, Result: typedNil}},
		{"case value", &ast.CaseExpr{Value: typedNil}},
		{"case else", &ast.CaseExpr{ElseClause: typedNil}},
		{"case branch", &ast.CaseExpr{WhenClauses: []*ast.WhenClause{nil}}},
		{"subquery", &ast.SubqueryExpr{Query: (*ast.SelectStmt)(nil)}},
		{"collate", &ast.CollateExpr{Expr: typedNil}},
		{"compare", &ast.CompareSubqueryExpr{L: value, R: typedNil}},
		{"exists", &ast.ExistsSubqueryExpr{Sel: typedNil}},
		{"in list", &ast.PatternInExpr{Expr: value, List: []ast.ExprNode{typedNil}}},
		{"in subquery", &ast.PatternInExpr{Expr: value, Sel: typedNil}},
		{"is null", &ast.IsNullExpr{Expr: typedNil}},
		{"is truth", &ast.IsTruthExpr{Expr: typedNil}},
		{"like escape", &ast.PatternLikeExpr{Expr: value, Pattern: value, EscapeExpr: typedNil}},
		{"parentheses", &ast.ParenthesesExpr{Expr: typedNil}},
		{"regexp", &ast.PatternRegexpExpr{Expr: value, Pattern: typedNil}},
		{"member", &ast.MemberOfExpr{Value: value, Array: typedNil}},
		{"is json", &ast.IsJSONExpr{Expr: typedNil}},
		{"match", &ast.MatchAgainst{Expr: typedNil}},
		{"sounds like", &ast.SoundsLikeExpr{Expr: value, Pattern: typedNil}},
		{"row", &ast.RowExpr{BaseRowExpr: pkast.BaseRowExpr{Items: []ast.ExprNode{typedNil}}}},
		{"unary", &ast.UnaryOperationExpr{BaseUnaryExpr: pkast.BaseUnaryExpr{Expr: typedNil}}},
		{"variable", &ast.VariableExpr{Target: &ast.VariableTarget{}, Value: typedNil}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := tc.node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil {
				t.Fatal("Restore accepted typed nil child")
			}
		})
	}
}

type expressionErrorWriter struct{ err error }

func (w expressionErrorWriter) Write([]byte) (int, error) { return 0, w.err }

func TestExpressionRestoreRetainsWriterErrors(t *testing.T) {
	want := errors.New("mock writer failure")
	value := pureIntLiteral("1")
	for _, node := range []ast.Node{
		&ast.BetweenExpr{Expr: value, Left: value, Right: value},
		&ast.WhenClause{Expr: value, Result: value},
		&ast.CaseExpr{WhenClauses: []*ast.WhenClause{{Expr: value, Result: value}}},
		&ast.IsNullExpr{Expr: value}, &ast.ParenthesesExpr{Expr: value},
	} {
		t.Run(fmt.Sprintf("%T", node), func(t *testing.T) {
			err := node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, expressionErrorWriter{err: want}))
			if !errors.Is(err, want) {
				t.Fatalf("Restore error = %v; want writer failure", err)
			}
		})
	}
}

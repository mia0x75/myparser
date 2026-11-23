package ast_test

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

func TestCallQualifiedNameRestore(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		parts []lex.Ident
		want  string
	}{
		{"two", []lex.Ident{{Name: "mock_schema"}, {Name: "mock_func"}}, "`mock_schema`.`mock_func`"},
		{"three", []lex.Ident{{Name: "mock_db"}, {Name: "mock_schema"}, {Name: "mock_func"}},
			"`mock_db`.`mock_schema`.`mock_func`"},
		{"quoted dot", []lex.Ident{{Name: "mock.func", Quote: lex.QuoteBacktick}}, "`mock.func`"},
		{"case sensitive qualifier", []lex.Ident{{Name: "Mock.Schema", Quote: lex.QuoteBacktick},
			{Name: "Mock.Func", Quote: lex.QuoteBacktick}}, "`Mock.Schema`.`Mock.Func`"},
		{"quoted empty", []lex.Ident{{Quote: lex.QuoteBacktick}}, "``"},
		{"qualified builtin", []lex.Ident{{Name: "mock_schema"}, {Name: "extract"}}, "`mock_schema`.`extract`"},
		{"quoted builtin", []lex.Ident{{Name: "extract", Quote: lex.QuoteBacktick}}, "`extract`"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			call := pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: tc.parts}}
			for _, kind := range []struct {
				name string
				expr ast.ExprNode
				tail string
			}{
				{"ordinary", &ast.FuncCallExpr{BaseCallExpr: call}, ""},
				{"aggregate", &ast.AggregateFuncExpr{BaseCallExpr: call}, ""},
				{"window", &ast.WindowFuncExpr{BaseCallExpr: call}, " OVER ()"},
			} {
				t.Run(kind.name, func(t *testing.T) {
					var out strings.Builder
					if err := kind.expr.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil || out.String() != tc.want+"()"+kind.tail {
						t.Fatalf("restore = %q, err=%v", out.String(), err)
					}
					out.Reset()
					ast.FormatExpr(kind.expr, &out)
					wantFormat := tc.want + "()" + strings.ToLower(kind.tail)
					if out.String() != wantFormat {
						t.Fatalf("format = %q, want %q", out.String(), wantFormat)
					}
				})
			}
		})
	}
}

func TestCallQualifiedNameValidation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		parts []lex.Ident
	}{
		{"absent", nil},
		{"missing qualifier", []lex.Ident{{}, {Name: "mock_func"}}},
		{"missing callee", []lex.Ident{{Name: "mock_schema"}, {}}},
		{"injected qualifier", []lex.Ident{{Name: "mock;schema"}, {Name: "mock_func"}}},
		{"injected callee", []lex.Ident{{Name: "mock_schema"}, {Name: "mock_func()"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			call := pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: tc.parts}}
			for _, expr := range []ast.ExprNode{
				&ast.FuncCallExpr{BaseCallExpr: call}, &ast.AggregateFuncExpr{BaseCallExpr: call},
				&ast.WindowFuncExpr{BaseCallExpr: call},
			} {
				var out strings.Builder
				if err := expr.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil || out.Len() != 0 {
					t.Fatalf("%T accepted malformed name: %q, err=%v", expr, out.String(), err)
				}
			}
		})
	}
	name := pkast.QualifiedName{Parts: []lex.Ident{{Name: "mock_schema"}, {Name: "count"}}}
	for _, expr := range []ast.ExprNode{
		&ast.AggregateFuncExpr{BaseCallExpr: pkast.BaseCallExpr{Name: name}, Star: true},
		&ast.WindowFuncExpr{BaseCallExpr: pkast.BaseCallExpr{Name: name}, Star: true},
	} {
		if err := expr.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{})); err == nil {
			t.Fatalf("%T treated a qualified callee as builtin COUNT", expr)
		}
	}
}

package ast_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

type extendedCTE struct {
	pkast.BaseCommonTableExpression
	materialized bool
}

func (n *extendedCTE) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *extendedCTE) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteIdent(n.Name)
	ctx.WriteKeyword(" AS ")
	if n.materialized {
		ctx.WriteKeyword("MATERIALIZED ")
	}
	ctx.WritePlain("(")
	if err := n.Query.Restore(ctx); err != nil {
		return err
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

var _ ast.CTENode = (*extendedCTE)(nil)

func TestWithClauseCTERoles(t *testing.T) {
	t.Parallel()
	original := &ast.CommonTableExpression{BaseCommonTableExpression: pkast.BaseCommonTableExpression{
		Name: lex.Ident{Name: "c"}, Query: &fakeCTESelect{},
	}}
	extended := &extendedCTE{BaseCommonTableExpression: original.BaseCommonTableExpression, materialized: true}
	for _, tc := range []struct {
		name        string
		replacement ast.Node
		wantOK      bool
	}{
		{"extension", extended, true},
		{"expression", ast.NewParamMarkerExpr(0), false},
		{"typed nil extension", (*extendedCTE)(nil), false},
		{"nil", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clause := &ast.WithClause{BaseWithClause: pkast.BaseWithClause{CTEs: []ast.CTENode{original}}}
			_, ok := clause.Accept(operatorVisitor{
				enter: func(n ast.Node) (ast.Node, bool) { return n, false },
				leave: func(n ast.Node) (ast.Node, bool) {
					if n == original {
						return tc.replacement, true
					}
					return n, true
				},
			})
			if ok != tc.wantOK || !ok && clause.CTEs[0] != original || ok && clause.CTEs[0] != tc.replacement {
				t.Fatalf("CTE replacement %T: ok=%v, stored=%T", tc.replacement, ok, clause.CTEs[0])
			}
			if !ok {
				return
			}
			var entered []ast.Node
			_, ok = clause.Accept(operatorVisitor{
				enter: func(n ast.Node) (ast.Node, bool) { entered = append(entered, n); return n, false },
				leave: func(n ast.Node) (ast.Node, bool) { return n, true },
			})
			if !ok || !reflect.DeepEqual(entered, []ast.Node{clause, extended}) ||
				clause.CTEs[0].CTECore() != &extended.BaseCommonTableExpression {
				t.Fatal("extension lost outer identity or shared storage")
			}
			var out strings.Builder
			if err := clause.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil || out.String() != "WITH `c` AS MATERIALIZED (SELECT 1)" {
				t.Fatalf("extension restore = %q, err=%v", out.String(), err)
			}
		})
	}
}

func TestWithClauseTypedNilCTE(t *testing.T) {
	t.Parallel()
	clause := &ast.WithClause{BaseWithClause: pkast.BaseWithClause{CTEs: []ast.CTENode{(*extendedCTE)(nil)}}}
	entered := 0
	v := operatorVisitor{
		enter: func(n ast.Node) (ast.Node, bool) { entered++; return n, false },
		leave: func(n ast.Node) (ast.Node, bool) { t.Fatal("invalid CTE must stop before Leave"); return n, true },
	}
	if _, ok := clause.Accept(v); ok || entered != 1 {
		t.Fatal("typed nil CTE must stop at its clause")
	}
	var out strings.Builder
	if err := clause.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil || !strings.Contains(err.Error(), "CTE 0") {
		t.Fatalf("typed nil CTE must fail with its index: %v", err)
	}
	entered = 0
	if n, ok := (*ast.WithClause)(nil).Accept(v); n != nil || ok || entered != 0 {
		t.Fatal("nil WITH must not visit")
	}
}

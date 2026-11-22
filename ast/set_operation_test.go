package ast_test

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
)

type invalidCoreOrder struct{ *dmlTestOrder }

func (*invalidCoreOrder) OrderByClauseCore() *pkast.BaseOrderByClause { return nil }

type invalidCoreLimit struct{ *dmlTestLimit }

func (*invalidCoreLimit) LimitCore() *pkast.BaseLimit { return nil }

func paginationOf(n ast.Node) *pkast.BaseQueryTail {
	switch n := n.(type) {
	case *ast.QueryTailStmt:
		return &n.Tail
	case *ast.SetOprStmt:
		return &n.Tail
	default:
		return nil
	}
}

func TestSetOperationRejectsInvalidOperators(t *testing.T) {
	for _, operator := range []pkast.SetOperationType{
		pkast.SetOperationUnknown,
		pkast.SetOperationType(99),
	} {
		t.Run(strconv.Itoa(int(operator)), func(t *testing.T) {
			left := dmlParse(t, "SELECT 1 FROM t").(ast.QueryNode)
			right := dmlParse(t, "SELECT 2 FROM t").(ast.QueryNode)
			n := &ast.SetOprStmt{
				BaseSetOperation: pkast.BaseSetOperation{
					Left:     left,
					Right:    right,
					Operator: operator,
				},
			}

			var restored strings.Builder
			if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &restored)); err == nil {
				t.Fatalf("Restore() accepted invalid operator %d", operator)
			}
		})
	}
}

func TestSetOperationRejectsInvalidNestedOperator(t *testing.T) {
	left := dmlParse(t, "SELECT 1 FROM t").(ast.QueryNode)
	right := dmlParse(t, "SELECT 2 FROM t").(ast.QueryNode)
	nested := &ast.SetOprStmt{
		BaseSetOperation: pkast.BaseSetOperation{
			Left:     left,
			Right:    right,
			Operator: pkast.SetOperationType(99),
		},
	}
	outer := &ast.SetOprStmt{
		BaseSetOperation: pkast.BaseSetOperation{
			Left:     nested,
			Right:    right,
			Operator: pkast.SetOperationUnion,
		},
	}

	var restored strings.Builder
	if err := outer.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &restored)); err == nil {
		t.Fatal("Restore() accepted a nested operation with an invalid operator")
	}
}

func TestQueryPaginationRejectsInvalidSharedClauses(t *testing.T) {
	cases := []struct {
		name string
		tail pkast.BaseQueryTail
	}{
		{name: "ORDER BY core", tail: pkast.BaseQueryTail{OrderBy: &invalidCoreOrder{&dmlTestOrder{}}}},
		{name: "LIMIT core", tail: pkast.BaseQueryTail{Limit: &invalidCoreLimit{&dmlTestLimit{}}}},
		{name: "typed nil ORDER BY", tail: pkast.BaseQueryTail{OrderBy: (*ast.OrderByClause)(nil)}},
		{name: "typed nil LIMIT", tail: pkast.BaseQueryTail{Limit: (*ast.Limit)(nil)}},
	}
	for _, sql := range []string{"SELECT 1 UNION SELECT 2", "(SELECT 1) LIMIT 2"} {
		for _, tc := range cases {
			t.Run(sql+"/"+tc.name, func(t *testing.T) {
				n := dmlParse(t, sql)
				*paginationOf(n) = tc.tail
				var restored strings.Builder
				if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &restored)); err == nil {
					t.Fatal("Restore() accepted a clause without a core")
				}
				if _, ok := n.Accept(byClauseVisitor{}); ok {
					t.Fatal("Accept() accepted a clause without a core")
				}
			})
		}
	}
}

func TestQueryTailVisitorRejectsInvalidReplacement(t *testing.T) {
	for _, sql := range []string{"SELECT 1 UNION SELECT 2", "(SELECT 1) LIMIT 2"} {
		t.Run(sql, func(t *testing.T) {
			n := dmlParse(t, sql)
			if _, ok := n.Accept(byClauseVisitor{
				enter: func(ast.Node) (ast.Node, bool) { return &ast.FieldList{}, true },
				leave: func(node ast.Node) (ast.Node, bool) { t.Error("invalid root reached Leave"); return node, true },
			}); ok {
				t.Fatal("accepted wrong root type with skipChildren")
			}
			if _, ok := n.Accept(byClauseVisitor{leave: func(node ast.Node) (ast.Node, bool) {
				if _, ok := node.(*ast.SelectStmt); ok {
					return (*ast.QueryTailStmt)(nil), true
				}
				return node, true
			}}); ok {
				t.Fatal("accepted a typed-nil query operand replacement")
			}
		})
	}
	if _, ok := (&ast.QueryTailStmt{}).Accept(byClauseVisitor{}); ok {
		t.Fatal("visited a query wrapper with no operand")
	}
}

func TestQueryTailRestorePreservesClauseError(t *testing.T) {
	cause := errors.New("clause restore failed")
	for _, sql := range []string{"SELECT 1 UNION SELECT 2", "(SELECT 1) LIMIT 2"} {
		t.Run(sql, func(t *testing.T) {
			n := dmlParse(t, sql)
			paginationOf(n).Limit = &dmlTestLimit{err: cause}
			var out strings.Builder
			err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out))
			if !errors.Is(err, cause) || !strings.Contains(err.Error(), "LIMIT") {
				t.Fatalf("Restore() lost the clause error or context: %v", err)
			}
		})
	}
}

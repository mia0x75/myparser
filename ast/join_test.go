package ast_test

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
)

type joinVisitor struct {
	enter func(ast.Node) (ast.Node, bool)
	leave func(ast.Node) (ast.Node, bool)
}

func (v joinVisitor) Enter(n ast.Node) (ast.Node, bool) { return v.enter(n) }
func (v joinVisitor) Leave(n ast.Node) (ast.Node, bool) { return v.leave(n) }

func joinFixture() *ast.JoinExpr {
	return &ast.JoinExpr{StraightJoin: true, BaseJoinExpr: pkast.BaseJoinExpr{
		Left: &ast.TableName{}, Right: &ast.TableName{}, On: ast.NewParamMarkerExpr(0), Type: pkast.JoinTypeInner,
	}}
}

func TestJoinExprVisitor(t *testing.T) {
	t.Parallel()
	t.Run("identity and order", func(t *testing.T) {
		n := joinFixture()
		var role ast.JoinExprNode = n
		if role.JoinExprCore() != &n.BaseJoinExpr || reflect.TypeFor[ast.JoinExpr]().NumField() != 2 {
			t.Fatal("outer must contain one shared core and its extension")
		}
		var enter, leave []ast.Node
		got, ok := n.Accept(joinVisitor{
			func(x ast.Node) (ast.Node, bool) { enter = append(enter, x); return x, false },
			func(x ast.Node) (ast.Node, bool) { leave = append(leave, x); return x, true },
		})
		if !ok || got != n || !n.StraightJoin || !reflect.DeepEqual(enter, []ast.Node{n, n.Left, n.Right, n.On}) ||
			!reflect.DeepEqual(leave, []ast.Node{n.Left, n.Right, n.On, n}) {
			t.Fatal("outer identity, dialect extension or L/R/ON order lost")
		}
	})
	t.Run("skip and replace outer", func(t *testing.T) {
		n, next := joinFixture(), joinFixture()
		got, ok := n.Accept(joinVisitor{
			func(x ast.Node) (ast.Node, bool) {
				if x != n {
					t.Fatal("visited a skipped child")
				}
				return next, true
			},
			func(x ast.Node) (ast.Node, bool) { return x, x == next },
		})
		if !ok || got != next {
			t.Fatal("skip or outer replacement lost")
		}
	})
	for _, tc := range []struct {
		name      string
		out       ast.Node
		keepGoing bool
		ok        bool
	}{
		{"stop", nil, false, false},
		{"wrong category", &ast.TableName{}, true, false},
		{"replace expression", ast.NewParamMarkerExpr(17), true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := joinFixture()
			original := n.On
			outerLeft := false
			_, ok := n.Accept(joinVisitor{
				func(x ast.Node) (ast.Node, bool) { return x, false },
				func(x ast.Node) (ast.Node, bool) {
					if x == original {
						return tc.out, tc.keepGoing
					}
					outerLeft = outerLeft || x == n
					return x, true
				},
			})
			if ok != tc.ok || outerLeft != tc.ok || (ok && n.On != tc.out) || (!ok && n.On != original) {
				t.Fatal("ON visitor control or assignment changed")
			}
		})
	}
}

type joinWriter struct {
	failOn string
	err    error
}

func (w joinWriter) Write(p []byte) (int, error) {
	if string(p) == w.failOn {
		return 0, w.err
	}
	return len(p), nil
}

func TestJoinExprRestoreLevel(t *testing.T) {
	t.Parallel()
	var original = errors.New("restore failure")
	for _, tc := range []struct {
		name, write string
		cause       error
	}{
		{"left", "l", original}, {"right", "r", original},
		{"on", "?", original}, {"closing parenthesis", ")", original},
		{"short write", ")", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := joinFixture()
			n.Left = &ast.TableName{BaseTableRef: pkast.BaseTableRef{Name: pkast.QualifiedName{Parts: []pkast.Ident{{Name: "l"}}}}}
			n.Right = &ast.TableName{BaseTableRef: pkast.BaseTableRef{Name: pkast.QualifiedName{Parts: []pkast.Ident{{Name: "r"}}}}}
			ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, joinWriter{tc.write, tc.cause})
			ctx.JoinLevel = 3
			want := tc.cause
			if want == nil {
				want = io.ErrShortWrite
			}
			if err := n.Restore(ctx); !errors.Is(err, want) || ctx.JoinLevel != 3 {
				t.Fatalf("err=%v level=%d, want %v at level 3", err, ctx.JoinLevel, want)
			}
		})
	}
	for _, invalid := range []ast.ResultSetNode{nil, (*ast.JoinExpr)(nil), (*ast.TableSource)(nil)} {
		n := joinFixture()
		n.Left = invalid
		var buf strings.Builder
		if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &buf)); err == nil || buf.Len() != 0 {
			t.Fatalf("invalid left %T was rendered: %q, err=%v", invalid, buf.String(), err)
		}
	}
}

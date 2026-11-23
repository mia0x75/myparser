package ast_test

import (
	"errors"
	"io"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	dformat "github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

type boundaryCall struct {
	name     string
	new      func(pkast.BaseCallExpr) ast.ExprNode
	typedNil ast.Node
}

func boundaryCalls() []boundaryCall {
	return []boundaryCall{
		{"aggregate", func(call pkast.BaseCallExpr) ast.ExprNode {
			return &ast.AggregateFuncExpr{BaseCallExpr: call}
		}, (*ast.AggregateFuncExpr)(nil)},
		{"window", func(call pkast.BaseCallExpr) ast.ExprNode {
			return &ast.WindowFuncExpr{BaseCallExpr: call}
		}, (*ast.WindowFuncExpr)(nil)},
	}
}

func TestCallChildLeaveBoundaries(t *testing.T) {
	cases := []struct {
		name string
		new  func() (ast.ExprNode, ast.Node, ast.Node)
	}{
		{"separator", func() (ast.ExprNode, ast.Node, ast.Node) {
			child := &sharedPureExpr{}
			return &ast.AggregateFuncExpr{Separator: child}, child, (*sharedPureExpr)(nil)
		}},
		{"order by", func() (ast.ExprNode, ast.Node, ast.Node) {
			child := &ast.OrderByClause{}
			return &ast.AggregateFuncExpr{OrderBy: child}, child, (*ast.OrderByClause)(nil)
		}},
		{"limit", func() (ast.ExprNode, ast.Node, ast.Node) {
			child := &ast.Limit{}
			return &ast.AggregateFuncExpr{Limit: child}, child, (*ast.Limit)(nil)
		}},
		{"within group", func() (ast.ExprNode, ast.Node, ast.Node) {
			child := &ast.OrderByClause{}
			return &ast.WindowFuncExpr{WithinGroup: child}, child, (*ast.OrderByClause)(nil)
		}},
		{"window spec", func() (ast.ExprNode, ast.Node, ast.Node) {
			root := &ast.WindowFuncExpr{}
			return root, &root.Spec, (*ast.WindowSpec)(nil)
		}},
	}
	for _, call := range boundaryCalls() {
		for index := range 2 {
			cases = append(cases, struct {
				name string
				new  func() (ast.ExprNode, ast.Node, ast.Node)
			}{call.name + "/arg " + strconv.Itoa(index), func() (ast.ExprNode, ast.Node, ast.Node) {
				args := []ast.ExprNode{&sharedPureExpr{}, &sharedPureExpr{}}
				return call.new(pkast.BaseCallExpr{Args: args}), args[index], (*sharedPureExpr)(nil)
			}})
		}
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, kind := range []string{"nil", "typed nil", "wrong role"} {
				t.Run(kind, func(t *testing.T) {
					root, target, typedNil := tc.new()
					var replacement ast.Node
					switch kind {
					case "typed nil":
						replacement = typedNil
					case "wrong role":
						replacement = &ast.BeginStmt{}
					}
					reached := false
					_, ok := root.Accept(operatorVisitor{
						enter: func(n ast.Node) (ast.Node, bool) { return n, n == target },
						leave: func(n ast.Node) (ast.Node, bool) {
							if n == target {
								reached = true
								return replacement, true
							}
							return n, true
						},
					})
					if !reached || ok {
						t.Fatalf("replacement reached=%v, Accept=%v; want true, false", reached, ok)
					}
				})
			}
		})
	}
}

func TestCallEnterAndTraversalBoundaries(t *testing.T) {
	for _, tc := range boundaryCalls() {
		t.Run(tc.name, func(t *testing.T) {
			oldChild, newChild := &sharedPureExpr{}, &sharedPureExpr{}
			root := tc.new(pkast.BaseCallExpr{Args: []ast.ExprNode{oldChild}})
			for _, bad := range []ast.Node{nil, tc.typedNil, &ast.BeginStmt{}} {
				for _, skip := range []bool{false, true} {
					if _, ok := root.Accept(operatorVisitor{
						enter: func(ast.Node) (ast.Node, bool) { return bad, skip },
						leave: func(n ast.Node) (ast.Node, bool) { return n, true },
					}); ok {
						t.Fatalf("accepted Enter=%T, skip=%v", bad, skip)
					}
				}
			}
			fresh := tc.new(pkast.BaseCallExpr{Args: []ast.ExprNode{newChild}})
			var entered []ast.Node
			got, ok := root.Accept(operatorVisitor{
				enter: func(n ast.Node) (ast.Node, bool) {
					entered = append(entered, n)
					if n == root {
						return fresh, false
					}
					return n, false
				},
				leave: func(n ast.Node) (ast.Node, bool) { return n, true },
			})
			if !ok || got != fresh || !slices.Contains(entered, ast.Node(newChild)) || slices.Contains(entered, ast.Node(oldChild)) {
				t.Fatal("Enter replacement did not traverse the new outer instance")
			}
		})
	}
	arg, separator, count := &sharedPureExpr{}, &sharedPureExpr{}, &sharedPureExpr{}
	agg := &ast.AggregateFuncExpr{
		BaseCallExpr: pkast.BaseCallExpr{Args: []ast.ExprNode{arg}}, Separator: separator,
		OrderBy: &ast.OrderByClause{}, Limit: &ast.Limit{BaseLimit: pkast.BaseLimit{Count: count}},
	}
	win := &ast.WindowFuncExpr{BaseCallExpr: pkast.BaseCallExpr{Args: []ast.ExprNode{arg}}, WithinGroup: &ast.OrderByClause{}}
	for _, tc := range []struct {
		name string
		root ast.ExprNode
		want []ast.Node
	}{
		{"aggregate", agg, []ast.Node{agg, arg, separator, agg.OrderBy, agg.Limit, count}},
		{"window", win, []ast.Node{win, arg, win.WithinGroup, &win.Spec}},
	} {
		for _, control := range []string{"order", "skip", "stop"} {
			t.Run(tc.name+"/"+control, func(t *testing.T) {
				var entered, left []ast.Node
				got, ok := tc.root.Accept(operatorVisitor{
					enter: func(n ast.Node) (ast.Node, bool) {
						entered = append(entered, n)
						return n, control == "skip"
					},
					leave: func(n ast.Node) (ast.Node, bool) {
						left = append(left, n)
						return n, control != "stop" || n != arg
					},
				})
				want := tc.want
				switch control {
				case "skip":
					want = want[:1]
				case "stop":
					want = want[:2]
					if len(left) != 1 || left[0] != arg {
						t.Fatal("stop visited later nodes or called Leave on the outer node")
					}
				}
				if got != tc.root || ok != (control != "stop") || !slices.Equal(entered, want) {
					t.Fatalf("Accept=%v, entered=%v; want %v", ok, entered, want)
				}
			})
		}
	}
}

type callBoundaryWriter struct{ err error }

func (w callBoundaryWriter) Write([]byte) (int, error) { return 0, w.err }

func TestCallRestoreBoundaries(t *testing.T) {
	for _, tc := range boundaryCalls() {
		t.Run(tc.name, func(t *testing.T) {
			for _, child := range []ast.ExprNode{nil, (*ast.LiteralExpr)(nil), (*sharedPureExpr)(nil)} {
				root := tc.new(pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "f"}}}, Args: []ast.ExprNode{child}})
				var out strings.Builder
				if err := root.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); err == nil {
					t.Fatalf("restored nil argument %T", child)
				}
				if _, ok := root.Accept(&sharedVisitor{}); ok {
					t.Fatalf("visited nil argument %T", child)
				}
			}
			var out strings.Builder
			root := tc.new(pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "f"}}}, Args: []ast.ExprNode{&sharedPureExpr{restoreErr: true}}})
			var marker *pureRestoreError
			if err := root.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); !errors.As(err, &marker) {
				t.Fatalf("lost child error chain: %v", err)
			}
			root = tc.new(pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "f"}}}})
			writeErr := errors.New("call writer failure")
			for _, err := range []error{writeErr, nil} {
				want := err
				if want == nil {
					want = io.ErrShortWrite
				}
				if got := root.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, callBoundaryWriter{err})); !errors.Is(got, want) {
					t.Fatalf("lost writer error: %v; want %v", got, want)
				}
			}
		})
	}
	agg := &ast.AggregateFuncExpr{
		BaseCallExpr: pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "group_concat"}}}},
		Separator:    &sharedPureExpr{restoreErr: true},
	}
	var out strings.Builder
	var marker *pureRestoreError
	if err := agg.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); !errors.As(err, &marker) {
		t.Fatalf("lost separator error chain: %v", err)
	}
}

func TestCallRestoreNameBoundaries(t *testing.T) {
	for _, call := range boundaryCalls() {
		t.Run(call.name, func(t *testing.T) {
			for _, name := range []string{"", "a;b", "a b", "a`b", "a(b)", "a.b", "a--b", "a/*b", "a\nb"} {
				root := call.new(pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: name}}}})
				var out strings.Builder
				if err := root.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); err == nil || out.Len() != 0 {
					t.Fatalf("name %q: output=%q, error=%v", name, out.String(), err)
				}
			}
			for _, tc := range []struct {
				name  lex.Ident
				flags dformat.RestoreFlags
				want  string
			}{
				{lex.Ident{Name: "函數"}, dformat.DefaultRestoreFlags, "函數()"},
				{lex.Ident{Name: "$f"}, dformat.DefaultRestoreFlags, "$F()"},
				{lex.Ident{Name: "123abc"}, dformat.DefaultRestoreFlags, "123ABC()"},
				{lex.Ident{Name: "MiXeD"}, dformat.RestoreKeyWordLowercase, "mixed()"},
				{lex.Ident{Name: "a`;b", Raw: "ignored raw SQL", Quote: lex.QuoteBacktick}, dformat.DefaultRestoreFlags, "`a``;b`()"},
				{lex.Ident{Name: "MiXeD", Raw: "`MiXeD`", Quote: lex.QuoteBacktick}, dformat.RestoreNameDoubleQuotes, "\"MiXeD\"()"},
				{lex.Ident{Quote: lex.QuoteBacktick}, dformat.DefaultRestoreFlags, "``()"},
			} {
				root := call.new(pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{tc.name}}})
				var out strings.Builder
				want := tc.want
				if call.name == "window" {
					tail := " OVER ()"
					if tc.flags&dformat.RestoreKeyWordLowercase != 0 {
						tail = " over ()"
					}
					want += tail
				}
				if err := root.Restore(dformat.NewRestoreCtx(tc.flags, &out)); err != nil || out.String() != want {
					t.Fatalf("name=%+v: output=%q, error=%v; want %q", tc.name, out.String(), err, want)
				}
			}
		})
	}
}

func TestSpecialCallRestoreAndVisitorBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name string
		new  func(ast.ExprNode) (ast.ExprNode, func() ast.ExprNode)
	}{
		{"function", func(arg ast.ExprNode) (ast.ExprNode, func() ast.ExprNode) {
			n := &ast.FuncCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "f"}}}, Args: []ast.ExprNode{arg}}
			return n, func() ast.ExprNode { return n.Args[0] }
		}},
		{"weight", func(arg ast.ExprNode) (ast.ExprNode, func() ast.ExprNode) {
			n := &ast.WeightStringExpr{Expr: arg}
			return n, func() ast.ExprNode { return n.Expr }
		}},
		{"binary", func(arg ast.ExprNode) (ast.ExprNode, func() ast.ExprNode) {
			n := &ast.FuncCastExpr{Expr: arg, FunctionType: ast.CastBinaryOperator}
			return n, func() ast.ExprNode { return n.Expr }
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, bad := range []ast.ExprNode{nil, (*ast.LiteralExpr)(nil)} {
				n, _ := tc.new(bad)
				var out strings.Builder
				if err := n.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); err == nil {
					t.Fatal("Restore accepted nil argument")
				}
			}
			for _, replacement := range []ast.Node{nil, (*ast.LiteralExpr)(nil), &ast.ColumnPosition{}, &ast.DefaultExpr{}} {
				old := ast.NewParamMarkerExpr(0)
				n, slot := tc.new(old)
				_, ok := n.Accept(operatorVisitor{
					enter: func(node ast.Node) (ast.Node, bool) { return node, false },
					leave: func(node ast.Node) (ast.Node, bool) {
						if node == old {
							return replacement, true
						}
						return node, true
					},
				})
				if expr, valid := replacement.(*ast.DefaultExpr); valid {
					if !ok || slot() != expr {
						t.Fatal("Accept rejected legal cross-expression replacement")
					}
				} else if ok || slot() != old {
					t.Fatal("Accept stored a rejected replacement")
				}
			}
			n, _ := tc.new(ast.NewParamMarkerExpr(0))
			marker := errors.New("MOCK_WRITER")
			if err := n.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, callBoundaryWriter{marker})); !errors.Is(err, marker) {
				t.Fatalf("lost writer error: %v", err)
			}
		})
	}
	for _, tc := range []struct {
		name string
		node ast.Node
	}{
		{"CAST missing type", &ast.FuncCastExpr{FunctionType: ast.CastFunction, Expr: ast.NewParamMarkerExpr(0)}},
		{"CAST missing zone", &ast.FuncCastExpr{FunctionType: ast.CastFunction, Expr: ast.NewParamMarkerExpr(0), Type: ast.NewTypeName("DATETIME"), TimeZone: &ast.CastTimeZone{}}},
		{"WEIGHT nil parameter", &ast.WeightStringExpr{Expr: ast.NewParamMarkerExpr(0), Params: []ast.ExprNode{(*ast.LiteralExpr)(nil)}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := tc.node.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); err == nil {
				t.Fatal("Restore accepted missing operand")
			}
		})
	}
}

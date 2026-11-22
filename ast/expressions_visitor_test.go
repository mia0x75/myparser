package ast_test

import (
	"reflect"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/myparser/ast"
)

type operatorVisitor struct {
	enter func(ast.Node) (ast.Node, bool)
	leave func(ast.Node) (ast.Node, bool)
}

func (v operatorVisitor) Enter(n ast.Node) (ast.Node, bool) { return v.enter(n) }
func (v operatorVisitor) Leave(n ast.Node) (ast.Node, bool) { return v.leave(n) }

func TestOperatorsPreserveOuterVisitorIdentity(t *testing.T) {
	for _, name := range []string{"binary", "unary", "row"} {
		t.Run(name, func(t *testing.T) {
			child := &sharedPureExpr{}
			var root, replacement ast.ExprNode
			switch name {
			case "unary":
				root = &ast.UnaryOperationExpr{BaseUnaryExpr: pkast.BaseUnaryExpr{Expr: child}}
				replacement = &ast.UnaryOperationExpr{BaseUnaryExpr: pkast.BaseUnaryExpr{Expr: child}}
			case "binary":
				root = &ast.BinaryOperationExpr{BaseBinaryExpr: pkast.BaseBinaryExpr{Left: child, Right: child}}
				replacement = &ast.BinaryOperationExpr{BaseBinaryExpr: pkast.BaseBinaryExpr{Left: child, Right: child}}
			case "row":
				root = &ast.RowExpr{BaseRowExpr: pkast.BaseRowExpr{Items: []ast.ExprNode{child, child}}}
				replacement = &ast.RowExpr{BaseRowExpr: pkast.BaseRowExpr{Items: []ast.ExprNode{child, child}}}
			}
			var entered, left []ast.Node
			v := operatorVisitor{
				enter: func(n ast.Node) (ast.Node, bool) {
					entered = append(entered, n)
					if n == root {
						return replacement, false
					}
					return n, false
				},
				leave: func(n ast.Node) (ast.Node, bool) {
					left = append(left, n)
					return n, true
				},
			}
			got, ok := root.Accept(v)
			if !ok || got != replacement {
				t.Fatalf("Enter replacement lost: got %T, ok=%v", got, ok)
			}
			wantEnter := []ast.Node{root, child}
			wantLeave := []ast.Node{child, replacement}
			if name != "unary" {
				wantEnter = []ast.Node{root, child, child}
				wantLeave = []ast.Node{child, child, replacement}
			}
			if !reflect.DeepEqual(entered, wantEnter) || !reflect.DeepEqual(left, wantLeave) {
				t.Fatalf("wrong visit order: entered=%v, left=%v", entered, left)
			}

			t.Run("skip", func(t *testing.T) {
				entered, left = nil, nil
				v.enter = func(n ast.Node) (ast.Node, bool) {
					entered = append(entered, n)
					return replacement, true
				}
				if got, ok := root.Accept(v); !ok || got != replacement {
					t.Fatal("skip lost the replacement")
				}
				if !reflect.DeepEqual(entered, []ast.Node{root}) || !reflect.DeepEqual(left, []ast.Node{replacement}) {
					t.Fatal("skip must still Leave the outer replacement exactly once")
				}
			})

			t.Run("stop", func(t *testing.T) {
				stopper := &sharedVisitor{stop: true}
				if _, ok := root.Accept(stopper); ok || len(stopper.entered) != 2 {
					t.Fatal("child stop must prevent visiting the next slot or leaving the root")
				}
			})
		})
	}
}

func pureIntLiteral(value string) ast.ExprNode {
	return &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: value}}
}

func TestOperatorsReplaceChildrenWithPureExpressions(t *testing.T) {
	replacement := &sharedPureExpr{}
	bin := &ast.BinaryOperationExpr{BaseBinaryExpr: pkast.BaseBinaryExpr{Left: &sharedPureExpr{}, Right: &sharedPureExpr{}}}
	unary := &ast.UnaryOperationExpr{BaseUnaryExpr: pkast.BaseUnaryExpr{Expr: &sharedPureExpr{}}}
	row := &ast.RowExpr{BaseRowExpr: pkast.BaseRowExpr{Items: []ast.ExprNode{&sharedPureExpr{}, &sharedPureExpr{}}}}
	for _, root := range []ast.ExprNode{bin, unary, row} {
		v := &sharedVisitor{replacement: replacement}
		if got, ok := root.Accept(v); !ok || got != root {
			t.Fatal("pure child replacement lost outer identity")
		}
	}
	if bin.Left != replacement || bin.Right != replacement || unary.Expr != replacement {
		t.Fatal("replacement was not written into every expression slot")
	}
	if row.Items[0] != replacement || row.Items[1] != replacement {
		t.Fatal("replacement was not written into every row item")
	}
}

func TestCollateStopsWhenChildVisitorStops(t *testing.T) {
	child := &sharedPureExpr{}
	root := &ast.CollateExpr{Expr: child, Collation: "utf8mb4_bin"}
	var entered, left []ast.Node
	v := operatorVisitor{
		enter: func(n ast.Node) (ast.Node, bool) {
			entered = append(entered, n)
			return n, false
		},
		leave: func(n ast.Node) (ast.Node, bool) {
			left = append(left, n)
			return n, false
		},
	}

	got, ok := root.Accept(v)
	if ok || got != root {
		t.Fatalf("child stop must return the original collate node: got=%T ok=%v", got, ok)
	}
	if !reflect.DeepEqual(entered, []ast.Node{root, child}) {
		t.Fatalf("wrong enter order: %v", entered)
	}
	if !reflect.DeepEqual(left, []ast.Node{child}) {
		t.Fatalf("outer Leave must not run after child stop: %v", left)
	}
}

func TestOperatorsRefuseWrongEnterTypes(t *testing.T) {
	for _, root := range []ast.ExprNode{&ast.BinaryOperationExpr{}, &ast.UnaryOperationExpr{}, &ast.RowExpr{}} {
		for _, skip := range []bool{false, true} {
			for _, wrong := range []ast.Node{nil, &sharedPureExpr{}, (*ast.BinaryOperationExpr)(nil), (*ast.UnaryOperationExpr)(nil), (*ast.RowExpr)(nil)} {
				left := false
				v := operatorVisitor{
					enter: func(ast.Node) (ast.Node, bool) { return wrong, skip },
					leave: func(n ast.Node) (ast.Node, bool) { left = true; return n, true },
				}
				if _, ok := root.Accept(v); ok || left {
					t.Fatal("wrong Enter type must stop before Leave, even when skipping")
				}
			}
		}
	}
}

func TestExpressionVisitorsRejectInvalidRootReplacements(t *testing.T) {
	roots := []struct {
		name     string
		node     ast.Node
		typedNil ast.Node
	}{
		{"between", &ast.BetweenExpr{}, (*ast.BetweenExpr)(nil)},
		{"binary", &ast.BinaryOperationExpr{}, (*ast.BinaryOperationExpr)(nil)},
		{"case", &ast.CaseExpr{}, (*ast.CaseExpr)(nil)},
		{"collate", &ast.CollateExpr{}, (*ast.CollateExpr)(nil)},
		{"column", &ast.ColumnRef{}, (*ast.ColumnRef)(nil)},
		{"compare-subquery", &ast.CompareSubqueryExpr{}, (*ast.CompareSubqueryExpr)(nil)},
		{"default", &ast.DefaultExpr{}, (*ast.DefaultExpr)(nil)},
		{"exists-subquery", &ast.ExistsSubqueryExpr{}, (*ast.ExistsSubqueryExpr)(nil)},
		{"is-json", &ast.IsJSONExpr{}, (*ast.IsJSONExpr)(nil)},
		{"is-null", &ast.IsNullExpr{}, (*ast.IsNullExpr)(nil)},
		{"is-truth", &ast.IsTruthExpr{}, (*ast.IsTruthExpr)(nil)},
		{"match-against", &ast.MatchAgainst{}, (*ast.MatchAgainst)(nil)},
		{"member-of", &ast.MemberOfExpr{}, (*ast.MemberOfExpr)(nil)},
		{"parentheses", &ast.ParenthesesExpr{}, (*ast.ParenthesesExpr)(nil)},
		{"pattern-in", &ast.PatternInExpr{}, (*ast.PatternInExpr)(nil)},
		{"pattern-like", &ast.PatternLikeExpr{}, (*ast.PatternLikeExpr)(nil)},
		{"pattern-regexp", &ast.PatternRegexpExpr{}, (*ast.PatternRegexpExpr)(nil)},
		{"position", &ast.PositionExpr{}, (*ast.PositionExpr)(nil)},
		{"row", &ast.RowExpr{}, (*ast.RowExpr)(nil)},
		{"sounds-like", &ast.SoundsLikeExpr{}, (*ast.SoundsLikeExpr)(nil)},
		{"subquery", &ast.SubqueryExpr{}, (*ast.SubqueryExpr)(nil)},
		{"unary", &ast.UnaryOperationExpr{}, (*ast.UnaryOperationExpr)(nil)},
		{"values", &ast.ValuesExpr{}, (*ast.ValuesExpr)(nil)},
		{"max-value", &ast.MaxValueExpr{}, (*ast.MaxValueExpr)(nil)},
		{"when-clause", &ast.WhenClause{}, (*ast.WhenClause)(nil)},
		{"parameter", ast.NewParamMarkerExpr(0), (*ast.ParamMarkerExpr)(nil)},
	}
	for _, root := range roots {
		for _, replacement := range []ast.Node{nil, pureIntLiteral("1"), root.typedNil} {
			for _, skip := range []bool{false, true} {
				t.Run(root.name, func(t *testing.T) {
					left := false
					visitor := operatorVisitor{
						enter: func(ast.Node) (ast.Node, bool) { return replacement, skip },
						leave: func(n ast.Node) (ast.Node, bool) { left = true; return n, true },
					}
					if _, ok := root.node.Accept(visitor); ok || left {
						t.Fatalf("%s accepted invalid root replacement or called Leave", root.name)
					}
				})
			}
		}
	}
}

func TestVariableExprRejectsInvalidReplacements(t *testing.T) {
	for _, replacement := range []ast.Node{nil, (*ast.VariableExpr)(nil)} {
		for _, skip := range []bool{false, true} {
			left := false
			visitor := operatorVisitor{
				enter: func(ast.Node) (ast.Node, bool) { return replacement, skip },
				leave: func(n ast.Node) (ast.Node, bool) { left = true; return n, true },
			}
			if _, ok := (&ast.VariableExpr{}).Accept(visitor); ok || left {
				t.Fatalf("accepted invalid replacement %T with skip=%v", replacement, skip)
			}
		}
	}
	for _, replacement := range []ast.Node{nil, pureIntLiteral("1"), (*ast.VariableExpr)(nil)} {
		left := false
		visitor := operatorVisitor{
			enter: func(ast.Node) (ast.Node, bool) { return replacement, false },
			leave: func(n ast.Node) (ast.Node, bool) { left = true; return n, true },
		}
		if _, ok := (&ast.VariableExpr{}).Accept(visitor); ok || left {
			t.Fatalf("accepted invalid outer replacement %T", replacement)
		}
	}
}

func TestVariableExprCanBeReplacedWhenSkippingChildren(t *testing.T) {
	replacement := pureIntLiteral("9")
	visitor := operatorVisitor{
		enter: func(ast.Node) (ast.Node, bool) { return replacement, true },
		leave: func(n ast.Node) (ast.Node, bool) { return n, true },
	}
	if got, ok := (&ast.VariableExpr{}).Accept(visitor); !ok || got != replacement {
		t.Fatalf("Accept = %T, %v; want skipped expression replacement", got, ok)
	}
}

func TestExpressionVisitorsRejectInvalidChildReplacements(t *testing.T) {
	shared := pureIntLiteral("1")
	selectStmt := &ast.SelectStmt{}
	cases := []struct {
		name string
		root ast.Node
		bad  ast.Node
	}{
		{"between", &ast.BetweenExpr{Expr: shared, Left: shared, Right: shared}, &ast.SelectStmt{}},
		{"binary", &ast.BinaryOperationExpr{BaseBinaryExpr: pkast.BaseBinaryExpr{Left: shared, Right: shared}}, &ast.SelectStmt{}},
		{"when-clause", &ast.WhenClause{Expr: shared, Result: shared}, &ast.SelectStmt{}},
		{"case", &ast.CaseExpr{Value: shared}, &ast.SelectStmt{}},
		{"subquery", &ast.SubqueryExpr{Query: selectStmt}, pureIntLiteral("2")},
		{"collate", &ast.CollateExpr{Expr: shared}, &ast.SelectStmt{}},
		{"compare-subquery", &ast.CompareSubqueryExpr{L: shared, R: shared}, &ast.SelectStmt{}},
		{"exists", &ast.ExistsSubqueryExpr{Sel: shared}, &ast.SelectStmt{}},
		{"pattern-in", &ast.PatternInExpr{Expr: shared, List: []ast.ExprNode{shared}}, &ast.SelectStmt{}},
		{"is-null", &ast.IsNullExpr{Expr: shared}, &ast.SelectStmt{}},
		{"is-truth", &ast.IsTruthExpr{Expr: shared}, &ast.SelectStmt{}},
		{"pattern-like", &ast.PatternLikeExpr{Expr: shared, Pattern: shared}, &ast.SelectStmt{}},
		{"parentheses", &ast.ParenthesesExpr{Expr: shared}, &ast.SelectStmt{}},
		{"position", &ast.PositionExpr{P: shared}, &ast.SelectStmt{}},
		{"regexp", &ast.PatternRegexpExpr{Expr: shared, Pattern: shared}, &ast.SelectStmt{}},
		{"member-of", &ast.MemberOfExpr{Value: shared, Array: shared}, &ast.SelectStmt{}},
		{"is-json", &ast.IsJSONExpr{Expr: shared}, &ast.SelectStmt{}},
		{"match-against", &ast.MatchAgainst{Expr: shared}, &ast.SelectStmt{}},
		{"sounds-like", &ast.SoundsLikeExpr{Expr: shared, Pattern: shared}, &ast.SelectStmt{}},
		{"row", &ast.RowExpr{BaseRowExpr: pkast.BaseRowExpr{Items: []ast.ExprNode{shared}}}, &ast.SelectStmt{}},
		{"unary", &ast.UnaryOperationExpr{BaseUnaryExpr: pkast.BaseUnaryExpr{Expr: shared}}, &ast.SelectStmt{}},
		{"values", &ast.ValuesExpr{Column: &ast.ColumnRef{}}, &ast.SelectStmt{}},
		{"variable", &ast.VariableExpr{Value: shared}, &ast.SelectStmt{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rootLeft bool
			visitor := operatorVisitor{
				enter: func(node ast.Node) (ast.Node, bool) { return node, false },
				leave: func(node ast.Node) (ast.Node, bool) {
					if node == tc.root {
						rootLeft = true
						return node, true
					}
					return tc.bad, true
				},
			}
			if _, ok := tc.root.Accept(visitor); ok || rootLeft {
				t.Fatal("invalid child replacement was accepted or root Leave ran")
			}
		})
	}
}

func TestRowRefusesNonExpressionChildren(t *testing.T) {
	for _, wrong := range []ast.Node{nil, &ast.SelectStmt{}} {
		child := pureIntLiteral("1")
		row := &ast.RowExpr{BaseRowExpr: pkast.BaseRowExpr{Items: []ast.ExprNode{nil, child, child}}}
		var entered, left []ast.Node
		v := operatorVisitor{
			enter: func(n ast.Node) (ast.Node, bool) {
				entered = append(entered, n)
				return n, false
			},
			leave: func(n ast.Node) (ast.Node, bool) {
				left = append(left, n)
				return wrong, true
			},
		}
		if got, ok := row.Accept(v); ok || got != row {
			t.Fatal("non-expression replacement must stop at the row")
		}
		if !reflect.DeepEqual(entered, []ast.Node{row, child}) || !reflect.DeepEqual(left, []ast.Node{child}) {
			t.Fatal("nil slots must be skipped and wrong replacements must stop before the next slot")
		}
		if row.Items[1] != child || row.Items[2] != child {
			t.Fatal("refused replacement changed the row items")
		}
	}
}

func TestRowVisitsEmptyAndNilSlots(t *testing.T) {
	for _, items := range [][]ast.ExprNode{nil, {nil, nil}} {
		row := &ast.RowExpr{BaseRowExpr: pkast.BaseRowExpr{Items: items}}
		var entered, left []ast.Node
		v := operatorVisitor{
			enter: func(n ast.Node) (ast.Node, bool) {
				entered = append(entered, n)
				return n, false
			},
			leave: func(n ast.Node) (ast.Node, bool) {
				left = append(left, n)
				return n, true
			},
		}
		if got, ok := row.Accept(v); !ok || got != row {
			t.Fatal("empty row traversal lost outer identity")
		}
		if !reflect.DeepEqual(entered, []ast.Node{row}) || !reflect.DeepEqual(left, []ast.Node{row}) {
			t.Fatal("empty rows must enter and leave only the outer node")
		}
	}
}

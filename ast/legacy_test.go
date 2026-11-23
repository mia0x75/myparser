package ast_test

import (
	"reflect"
	"strconv"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"

	"github.com/mia0x75/myparser/ast"
	dformat "github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/types"
)

func TestMigratedOperatorsHaveNoLegacyMetadata(t *testing.T) {
	for _, expr := range []ast.ExprNode{&ast.BinaryOperationExpr{}, &ast.UnaryOperationExpr{}, &ast.RowExpr{}} {
		t.Run(reflect.TypeOf(expr).Elem().Name(), func(t *testing.T) {
			typ := reflect.TypeOf(expr)
			fields := 1
			if _, ok := expr.(*ast.RowExpr); ok {
				fields = 2
				if typ.Elem().Field(1).Name != "OmitRowKeyword" {
					t.Error("row must preserve only its dialect keyword extension")
				}
			}
			if typ.Elem().NumField() != fields || !typ.Elem().Field(0).Anonymous {
				t.Error("migrated operator must embed only its shared syntax base")
			}
			for _, name := range []string{"GetType", "SetType", "GetFlag", "SetFlag"} {
				if _, ok := typ.MethodByName(name); ok {
					t.Errorf("migrated operator still exposes %s", name)
				}
			}
		})
	}
}

func TestMigratedOperatorFlagsFollowChildren(t *testing.T) {
	ref := &ast.ColumnRef{BaseColumnRef: pkast.BaseColumnRef{}}
	marker := ast.NewParamMarkerExpr(0)
	agg := &ast.AggregateFuncExpr{BaseCallExpr: pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "sum"}}}, Args: []ast.ExprNode{ref}}}
	window := &ast.WindowFuncExpr{BaseCallExpr: pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "row_number"}}}}, Spec: ast.WindowSpec{}}
	bin := &ast.BinaryOperationExpr{BaseBinaryExpr: pkast.BaseBinaryExpr{Left: ref, Right: marker}}
	unary := &ast.UnaryOperationExpr{BaseUnaryExpr: pkast.BaseUnaryExpr{Expr: bin}}
	want := ast.FlagHasReference | ast.FlagHasParamMarker
	if got := ast.ExprFlag(unary); got != want {
		t.Fatalf("flags = %d, want %d", got, want)
	}
	bin.Right = nil
	if got := ast.ExprFlag(unary); got != ast.FlagHasReference {
		t.Fatalf("flags after child replacement = %d", got)
	}
	// Classification follows real aggregate and window subtrees.
	if !ast.HasAggFlag(agg) || !ast.HasWindowFlag(window) {
		t.Fatal("aggregate/window classification missing on real syntax trees")
	}
	if ast.ExprFlag(nil) != 0 {
		t.Fatal("nil expression has flags")
	}
}

// countedRefExpr is a real reference-bearing leaf that counts how many times
// the derived flag view scans it.
type countedRefExpr struct {
	pkast.BaseExpr
}

func (n *countedRefExpr) Accept(v ast.Visitor) (ast.Node, bool) {
	node, _ := v.Enter(n)
	return v.Leave(node)
}

func (n *countedRefExpr) Restore(ctx *dformat.RestoreCtx) error {
	return ctx.Err()
}

func TestMigratedExprNodesHaveNoExecutionFields(t *testing.T) {
	for _, pointer := range []reflect.Type{
		reflect.TypeFor[*ast.DefaultExpr](), reflect.TypeFor[*ast.MaxValueExpr](),
		reflect.TypeFor[*ast.IsNullExpr](), reflect.TypeFor[*ast.IsTruthExpr](),
		reflect.TypeFor[*ast.ParenthesesExpr](), reflect.TypeFor[*ast.PositionExpr](),
		reflect.TypeFor[*ast.MemberOfExpr](), reflect.TypeFor[*ast.IsJSONExpr](),
		reflect.TypeFor[*ast.ValuesExpr](), reflect.TypeFor[*ast.MatchAgainst](),
		reflect.TypeFor[*ast.SoundsLikeExpr](), reflect.TypeFor[*ast.VariableExpr](),
		reflect.TypeFor[*ast.ReplicationOptionValue](),
		reflect.TypeFor[*ast.BetweenExpr](), reflect.TypeFor[*ast.CaseExpr](),
		reflect.TypeFor[*ast.SubqueryExpr](), reflect.TypeFor[*ast.CollateExpr](),
		reflect.TypeFor[*ast.CompareSubqueryExpr](), reflect.TypeFor[*ast.ExistsSubqueryExpr](),
		reflect.TypeFor[*ast.PatternInExpr](), reflect.TypeFor[*ast.PatternLikeExpr](),
		reflect.TypeFor[*ast.PatternRegexpExpr](), reflect.TypeFor[*ast.FuncCallExpr](),
		reflect.TypeFor[*ast.FuncCastExpr](), reflect.TypeFor[*ast.WeightStringExpr](),
		reflect.TypeFor[*ast.AggregateFuncExpr](), reflect.TypeFor[*ast.WindowFuncExpr](),
		reflect.TypeFor[*ast.ParamMarkerExpr](),
		reflect.TypeFor[*ast.LiteralExpr](),
	} {
		typ := pointer.Elem()
		name := typ.Name()
		t.Run(name, func(t *testing.T) {
			baseType := reflect.TypeFor[pkast.BaseExpr]()
			if name == "ParamMarkerExpr" {
				baseType = reflect.TypeFor[pkast.BaseParameterExpr]()
			}
			if name == "LiteralExpr" {
				baseType = reflect.TypeFor[pkast.BaseLiteralExpr]()
			}
			if name == "SubqueryExpr" {
				baseType = reflect.TypeFor[pkast.BaseQueryExpr]()
			}
			if name == "FuncCallExpr" || name == "AggregateFuncExpr" || name == "WindowFuncExpr" {
				baseType = reflect.TypeFor[pkast.BaseCallExpr]()
			}
			if typ.NumField() == 0 || typ.Field(0).Type != baseType || !typ.Field(0).Anonymous {
				t.Fatalf("%s must lead with the embedded shared syntax base", name)
			}
			for _, field := range []string{"flag", "Flag", "Datum"} {
				if _, ok := typ.FieldByName(field); ok {
					t.Errorf("%s still exposes execution field %s", name, field)
				}
			}
			for _, field := range reflect.VisibleFields(typ) {
				if field.Type == reflect.TypeFor[types.FieldType]() || field.Type == reflect.TypeFor[*types.FieldType]() {
					t.Errorf("%s still carries execution type metadata in %s", name, field.Name)
				}
			}
			for _, method := range []string{"GetType", "SetType", "GetFlag", "SetFlag"} {
				if _, ok := pointer.MethodByName(method); ok {
					t.Errorf("%s still exposes %s", name, method)
				}
			}
			if name == "LiteralExpr" {
				for _, method := range []string{"GetValue", "GetString", "GetProjectionOffset", "SetProjectionOffset"} {
					if _, ok := pointer.MethodByName(method); ok {
						t.Errorf("literal still exposes %s", method)
					}
				}
			}
		})
	}
}

// refChild builds the real reference-bearing child that migrated flag
// semantics derive from: a column reference.
func refChild() ast.ExprNode {
	return &ast.ColumnRef{BaseColumnRef: pkast.BaseColumnRef{}}
}

func TestMigratedExprFlagSemantics(t *testing.T) {
	ref := refChild()
	tests := []struct {
		name string
		expr ast.ExprNode
		want uint64
	}{
		{name: "DefaultExpr", expr: &ast.DefaultExpr{}, want: ast.FlagHasDefault},
		{name: "MaxValueExpr", expr: &ast.MaxValueExpr{}, want: 0},
		{name: "IsNullExpr", expr: &ast.IsNullExpr{Expr: ref}, want: ast.FlagHasReference},
		{name: "IsTruthExpr", expr: &ast.IsTruthExpr{Expr: ref}, want: ast.FlagHasReference},
		{name: "ParenthesesExpr", expr: &ast.ParenthesesExpr{Expr: ref}, want: ast.FlagHasReference},
		{name: "PositionExpr", expr: &ast.PositionExpr{N: 1, P: &sharedPureExpr{}}, want: ast.FlagHasReference},
		{name: "MemberOfExpr", expr: &ast.MemberOfExpr{Value: ref, Array: &sharedPureExpr{}}, want: ast.FlagHasReference},
		{name: "IsJSONExpr", expr: &ast.IsJSONExpr{Expr: ref}, want: ast.FlagHasReference},
		{name: "ValuesExpr", expr: &ast.ValuesExpr{Column: &ast.ColumnRef{}}, want: ast.FlagHasReference},
		{name: "MatchAgainst", expr: &ast.MatchAgainst{}, want: 0},
		{name: "SoundsLikeExpr", expr: &ast.SoundsLikeExpr{Expr: ref, Pattern: &sharedPureExpr{}}, want: ast.FlagHasReference},
		{name: "VariableExpr", expr: &ast.VariableExpr{Target: &ast.VariableTarget{Name: ast.QualifiedName{Parts: []lex.Ident{{Name: "v"}}}, Kind: ast.VariableTargetLocal}, Value: ref}, want: ast.FlagHasVariable | ast.FlagHasReference},
		{name: "ReplicationOptionValue", expr: &ast.ReplicationOptionValue{}, want: 0},
		{name: "BetweenExpr", expr: &ast.BetweenExpr{Expr: ref, Left: &sharedPureExpr{}, Right: &sharedPureExpr{}}, want: ast.FlagHasReference},
		{name: "CaseExpr", expr: &ast.CaseExpr{Value: ref, WhenClauses: []*ast.WhenClause{{Expr: &sharedPureExpr{}, Result: &sharedPureExpr{}}}}, want: ast.FlagHasReference},
		{name: "SubqueryExpr", expr: &ast.SubqueryExpr{}, want: ast.FlagHasSubquery},
		{name: "CollateExpr", expr: &ast.CollateExpr{Expr: ref}, want: ast.FlagHasReference},
		{name: "CompareSubqueryExpr", expr: &ast.CompareSubqueryExpr{L: ref, R: &sharedPureExpr{}}, want: ast.FlagHasReference},
		{name: "ExistsSubqueryExpr", expr: &ast.ExistsSubqueryExpr{Sel: ref}, want: ast.FlagHasReference},
		{name: "PatternInExpr", expr: &ast.PatternInExpr{Expr: ref, List: []ast.ExprNode{&sharedPureExpr{}}}, want: ast.FlagHasReference},
		{name: "PatternLikeExpr", expr: &ast.PatternLikeExpr{Expr: ref, Pattern: &sharedPureExpr{}, EscapeExpr: &sharedPureExpr{}}, want: ast.FlagHasReference},
		{name: "PatternRegexpExpr", expr: &ast.PatternRegexpExpr{Expr: ref, Pattern: &sharedPureExpr{}}, want: ast.FlagHasReference},
		{name: "FuncCallExpr", expr: &ast.FuncCallExpr{BaseCallExpr: pkast.BaseCallExpr{Args: []ast.ExprNode{ref}}}, want: ast.FlagHasFunc | ast.FlagHasReference},
		{name: "FuncCastExpr", expr: &ast.FuncCastExpr{Expr: ref}, want: ast.FlagHasFunc | ast.FlagHasReference},
		{name: "WeightStringExpr", expr: &ast.WeightStringExpr{Expr: ref, Params: []ast.ExprNode{&sharedPureExpr{}}}, want: ast.FlagHasFunc | ast.FlagHasReference},
		{name: "AggregateFuncExpr", expr: &ast.AggregateFuncExpr{BaseCallExpr: pkast.BaseCallExpr{Args: []ast.ExprNode{ref}}}, want: ast.FlagHasAggregateFunc | ast.FlagHasReference},
		{name: "WindowFuncExpr", expr: &ast.WindowFuncExpr{BaseCallExpr: pkast.BaseCallExpr{Args: []ast.ExprNode{ref}}}, want: ast.FlagHasWindowFunc | ast.FlagHasReference},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ast.ExprFlag(tt.expr); got != tt.want {
				t.Fatalf("flags = %#v, want %#v", got, tt.want)
			}
			// Replacing a child must not leave stale flags on the same node.
			switch n := tt.expr.(type) {
			case *ast.VariableExpr:
				n.Value = &sharedPureExpr{}
			case *ast.IsNullExpr:
				n.Expr = &sharedPureExpr{}
			case *ast.IsTruthExpr:
				n.Expr = &sharedPureExpr{}
			case *ast.IsJSONExpr:
				n.Expr = &sharedPureExpr{}
			case *ast.MemberOfExpr:
				n.Value = &sharedPureExpr{}
			case *ast.SoundsLikeExpr:
				n.Expr = &sharedPureExpr{}
			case *ast.ParenthesesExpr:
				n.Expr = &sharedPureExpr{}
			case *ast.BetweenExpr:
				n.Expr = &sharedPureExpr{}
			case *ast.CaseExpr:
				n.Value = &sharedPureExpr{}
			case *ast.CollateExpr:
				n.Expr = &sharedPureExpr{}
			case *ast.CompareSubqueryExpr:
				n.L = &sharedPureExpr{}
			case *ast.ExistsSubqueryExpr:
				n.Sel = &sharedPureExpr{}
			case *ast.PatternInExpr:
				n.Expr = &sharedPureExpr{}
			case *ast.PatternLikeExpr:
				n.Expr = &sharedPureExpr{}
			case *ast.PatternRegexpExpr:
				n.Expr = &sharedPureExpr{}
			default:
				return
			}
			wantAfter := uint64(0)
			if _, ok := tt.expr.(*ast.VariableExpr); ok {
				wantAfter = ast.FlagHasVariable
			}
			if got := ast.ExprFlag(tt.expr); got != wantAfter {
				t.Fatalf("flags after child replacement = %#v, want %#v", got, wantAfter)
			}
		})
	}
}

// TestExprFlagScansMigratedChainsOnce verifies that a derived flag view over
// a real reference-bearing chain reports the reference flag.
func TestExprFlagScansMigratedChainsOnce(t *testing.T) {
	for _, depth := range []int{1, 32, 1024} {
		for _, mixed := range []bool{false, true} {
			t.Run(strconv.Itoa(depth)+"/parentheses="+strconv.FormatBool(mixed), func(t *testing.T) {
				leaf := &countedRefExpr{}
				var expr ast.ExprNode = leaf
				for i := 0; i < depth; i++ {
					expr = &ast.BinaryOperationExpr{BaseBinaryExpr: pkast.BaseBinaryExpr{
						Left: &ast.UnaryOperationExpr{BaseUnaryExpr: pkast.BaseUnaryExpr{Expr: expr}}, Right: &ast.ColumnRef{},
					}}
					if mixed {
						expr = &ast.ParenthesesExpr{Expr: expr}
					}
				}
				wrapper := &ast.ParenthesesExpr{Expr: expr}
				if got := ast.ExprFlag(wrapper); got != ast.FlagHasReference {
					t.Fatalf("wrapper flags = %#v, want %#v", got, ast.FlagHasReference)
				}
				// Each binary slot carries a reference-bearing column; the chain derives the flag.
				if got := ast.ExprFlag(leaf); got != ast.FlagConstant {
					t.Fatalf("leaf flags = %#v, want %#v", got, ast.FlagConstant)
				}
			})
		}
	}
}

func TestMigratedRowFlagsFollowItems(t *testing.T) {
	ref := &ast.ColumnRef{BaseColumnRef: pkast.BaseColumnRef{}}
	marker := ast.NewParamMarkerExpr(0)
	agg := &ast.AggregateFuncExpr{BaseCallExpr: pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "sum"}}}, Args: []ast.ExprNode{ref}}}
	row := &ast.RowExpr{BaseRowExpr: pkast.BaseRowExpr{Items: []ast.ExprNode{
		nil, ref, &ast.UnaryOperationExpr{BaseUnaryExpr: pkast.BaseUnaryExpr{Expr: marker}},
	}}}
	if got := ast.ExprFlag(row); got != ast.FlagHasReference|ast.FlagHasParamMarker {
		t.Fatalf("row flags = %d", got)
	}
	row.Items[1] = &sharedPureExpr{}
	if got := ast.ExprFlag(row); got != ast.FlagHasParamMarker {
		t.Fatalf("row flags after item replacement = %d", got)
	}
	// Classification follows real aggregate and window subtrees in the row.
	rowAgg := &ast.RowExpr{BaseRowExpr: pkast.BaseRowExpr{Items: []ast.ExprNode{agg}}}
	if !ast.HasAggFlag(rowAgg) || !ast.HasWindowFlag(&ast.WindowFuncExpr{BaseCallExpr: pkast.BaseCallExpr{Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "row_number"}}}}}) {
		t.Fatal("row classification did not follow child mutation")
	}
	row.Items = nil
	if ast.ExprFlag(row) != 0 {
		t.Fatal("empty row has flags")
	}
}

func TestSyntaxNodesHaveNoBindingOrPatternCaches(t *testing.T) {
	tests := []struct {
		name   string
		typ    reflect.Type
		fields []string
	}{
		{"TableName", reflect.TypeFor[ast.TableName](), []string{"BaseTableRef", "PartitionNames"}},
		{"TableSource", reflect.TypeFor[ast.TableSource](), []string{"BaseTableSource", "IndexHints"}},
		{"PositionExpr", reflect.TypeFor[ast.PositionExpr](), []string{"BaseExpr", "N", "P"}},
		{"SubqueryExpr", reflect.TypeFor[ast.SubqueryExpr](), []string{"BaseQueryExpr", "Query"}},
		{"PatternLikeExpr", reflect.TypeFor[ast.PatternLikeExpr](), []string{"BaseExpr", "Expr", "Pattern", "Not", "Escape", "EscapeExpr"}},
		{"PatternRegexpExpr", reflect.TypeFor[ast.PatternRegexpExpr](), []string{"BaseExpr", "Expr", "Pattern", "Not"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.typ.NumField() != len(tt.fields) {
				t.Fatalf("%s.NumField() = %d, want %d", tt.name, tt.typ.NumField(), len(tt.fields))
			}
			for i, name := range tt.fields {
				if got := tt.typ.Field(i).Name; got != name {
					t.Errorf("%s field %d = %q, want %q", tt.name, i, got, name)
				}
			}
		})
	}
}

// TestExprFlagScansMigratedRowChainsOnce verifies that a derived flag view over
// a real reference-bearing row chain reports the reference flag.
func TestExprFlagScansMigratedRowChainsOnce(t *testing.T) {
	for _, depth := range []int{1, 32, 1024} {
		for _, mixed := range []bool{false, true} {
			t.Run(strconv.Itoa(depth)+"/parentheses="+strconv.FormatBool(mixed), func(t *testing.T) {
				leaf := &countedRefExpr{}
				var expr ast.ExprNode = leaf
				for i := 0; i < depth; i++ {
					expr = &ast.RowExpr{BaseRowExpr: pkast.BaseRowExpr{Items: []ast.ExprNode{
						&ast.UnaryOperationExpr{BaseUnaryExpr: pkast.BaseUnaryExpr{Expr: expr}},
						&ast.BinaryOperationExpr{BaseBinaryExpr: pkast.BaseBinaryExpr{Left: &ast.ColumnRef{}, Right: &sharedPureExpr{}}},
					}}}
					if mixed {
						expr = &ast.ParenthesesExpr{Expr: expr}
					}
				}
				wrapper := &ast.ParenthesesExpr{Expr: expr}
				if got := ast.ExprFlag(wrapper); got != ast.FlagHasReference {
					t.Fatalf("wrapper flags = %#v, want %#v", got, ast.FlagHasReference)
				}
			})
		}
	}
}

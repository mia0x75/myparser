package ast

import (
	"fmt"
	"io"
	"strings"

	"github.com/mia0x75/myparser/format"
)

// ExprFlag derives flags for migrated syntax nodes. Pure syntax nodes never
// cache execution-layer flags.
func ExprFlag(expr ExprNode) uint64 {
	switch expr := expr.(type) {
	case *ParamMarkerExpr:
		return FlagHasParamMarker
	case *ColumnRef:
		return FlagHasReference
	case *DefaultExpr:
		return FlagHasDefault
	case *PositionExpr, *ValuesExpr:
		return FlagHasReference
	case *VariableExpr:
		return FlagHasVariable | ExprFlag(expr.Value)
	case *IsNullExpr:
		return ExprFlag(expr.Expr)
	case *IsTruthExpr:
		return ExprFlag(expr.Expr)
	case *IsJSONExpr:
		return ExprFlag(expr.Expr)
	case *ParenthesesExpr:
		return ExprFlag(expr.Expr)
	case *MemberOfExpr:
		return ExprFlag(expr.Value) | ExprFlag(expr.Array)
	case *SoundsLikeExpr:
		return ExprFlag(expr.Expr) | ExprFlag(expr.Pattern)
	case *BetweenExpr:
		return ExprFlag(expr.Expr) | ExprFlag(expr.Left) | ExprFlag(expr.Right)
	case *CaseExpr:
		var flag uint64
		flag |= ExprFlag(expr.Value)
		for _, clause := range expr.WhenClauses {
			if clause != nil {
				flag |= ExprFlag(clause.Expr) | ExprFlag(clause.Result)
			}
		}
		return flag | ExprFlag(expr.ElseClause)
	case *SubqueryExpr:
		return FlagHasSubquery
	case *CollateExpr:
		return ExprFlag(expr.Expr)
	case *CompareSubqueryExpr:
		return ExprFlag(expr.L) | ExprFlag(expr.R)
	case *ExistsSubqueryExpr:
		return ExprFlag(expr.Sel)
	case *PatternInExpr:
		flag := ExprFlag(expr.Expr)
		for _, item := range expr.List {
			flag |= ExprFlag(item)
		}
		return flag | ExprFlag(expr.Sel)
	case *PatternLikeExpr:
		return ExprFlag(expr.Expr) | ExprFlag(expr.Pattern) | ExprFlag(expr.EscapeExpr)
	case *PatternRegexpExpr:
		return ExprFlag(expr.Expr) | ExprFlag(expr.Pattern)
	case *BinaryOperationExpr:
		return ExprFlag(expr.Left) | ExprFlag(expr.Right)
	case *UnaryOperationExpr:
		return ExprFlag(expr.Expr)
	case *RowExpr:
		var flag uint64
		for _, item := range expr.Items {
			flag |= ExprFlag(item)
		}
		return flag
	case *FuncCallExpr:
		flag := FlagHasFunc
		for _, arg := range expr.Args {
			flag |= ExprFlag(arg)
		}
		return flag
	case *FuncCastExpr:
		flag := FlagHasFunc | ExprFlag(expr.Expr)
		if expr.TimeZone != nil {
			flag |= ExprFlag(expr.TimeZone.Expr)
		}
		return flag
	case *WeightStringExpr:
		flag := FlagHasFunc | ExprFlag(expr.Expr)
		for _, param := range expr.Params {
			flag |= ExprFlag(param)
		}
		return flag
	case *AggregateFuncExpr:
		flag := FlagHasAggregateFunc
		for _, arg := range expr.Args {
			flag |= ExprFlag(arg)
		}
		return flag | callTailFlags(expr.OrderBy, expr.Separator, expr.Limit)
	case *WindowFuncExpr:
		flag := FlagHasWindowFunc
		for _, arg := range expr.Args {
			flag |= ExprFlag(arg)
		}
		return flag | callTailFlags(expr.OrderBy, expr.Separator, expr.Limit)
	}
	return 0
}

func callTailFlags(order *OrderByClause, separator ExprNode, limit *Limit) uint64 {
	flag := ExprFlag(separator)
	if order != nil {
		for _, item := range order.Items {
			if item != nil {
				if core := item.OrderItemCore(); core != nil {
					flag |= ExprFlag(core.Expr)
				}
			}
		}
	}
	if limit != nil {
		flag |= ExprFlag(limit.Count) | ExprFlag(limit.Offset)
	}
	return flag
}

// FormatExpr preserves legacy rendering without widening the shared ExprNode.
// Use Restore when output errors must be reported.
func FormatExpr(expr ExprNode, w io.Writer) {
	if legacy, ok := expr.(interface{ Format(io.Writer) }); ok {
		legacy.Format(w)
		return
	}
	var out strings.Builder
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &out)
	if err := expr.Restore(ctx); err != nil {
		_, _ = fmt.Fprint(w, "<unsupported>")
		return
	}
	_, _ = io.WriteString(w, out.String())
}

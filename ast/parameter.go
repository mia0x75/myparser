package ast

import (
	"io"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/myparser/format"
)

// ParamMarkerExpr represents a parameter placeholder without execution data.
type ParamMarkerExpr struct {
	pkast.BaseParameterExpr
}

// NewParamMarkerExpr creates a parameter with an unknown source span.
func NewParamMarkerExpr(index int) *ParamMarkerExpr {
	return NewParamMarkerExprAt(loc.InvalidSpan(), index)
}

// NewParamMarkerExprAt creates a parameter with independent source and index data.
func NewParamMarkerExprAt(span loc.Span, index int) *ParamMarkerExpr {
	n := &ParamMarkerExpr{BaseParameterExpr: pkast.BaseParameterExpr{Index: index}}
	n.SetSpan(span)
	return n
}

// Restore implements Node interface.
func (*ParamMarkerExpr) Restore(ctx *format.RestoreCtx) error {
	ctx.WritePlain("?")
	return ctx.Err()
}

// Format implements ExprNode interface.
func (*ParamMarkerExpr) Format(w io.Writer) {
	_, _ = io.WriteString(w, "?")
}

// SetOrder preserves the existing API; the syntax index is canonical.
func (n *ParamMarkerExpr) SetOrder(index int) { n.Index = index }

// Accept implements Node interface.
func (n *ParamMarkerExpr) Accept(v Visitor) (Node, bool) {
	entered, _ := v.Enter(n)
	marker, ok := entered.(*ParamMarkerExpr)
	if !ok || marker == nil {
		return entered, false
	}
	return v.Leave(marker)
}

var _ ExprNode = (*ParamMarkerExpr)(nil)

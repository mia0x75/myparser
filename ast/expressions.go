package ast

import (
	"fmt"
	"io"
	"strings"

	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/myparser/errors"

	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/opcode"
)

var (
	_ ExprNode      = &BetweenExpr{}
	_ ExprNode      = &BinaryOperationExpr{}
	_ ExprNode      = &CaseExpr{}
	_ ExprNode      = &CollateExpr{}
	_ ExprNode      = &ColumnRef{}
	_ ExprNode      = &CompareSubqueryExpr{}
	_ ExprNode      = &DefaultExpr{}
	_ ExprNode      = &ExistsSubqueryExpr{}
	_ ExprNode      = &IsJSONExpr{}
	_ ExprNode      = &IsNullExpr{}
	_ ExprNode      = &IsTruthExpr{}
	_ ExprNode      = &MatchAgainst{}
	_ ExprNode      = &MemberOfExpr{}
	_ ExprNode      = &ParenthesesExpr{}
	_ ExprNode      = &PatternInExpr{}
	_ ExprNode      = &PatternLikeExpr{}
	_ ExprNode      = &PatternRegexpExpr{}
	_ ExprNode      = &PositionExpr{}
	_ ExprNode      = &RowExpr{}
	_ pkast.RowNode = &RowExpr{}
	_ ExprNode      = &SoundsLikeExpr{}
	_ ExprNode      = &SubqueryExpr{}
	_ ExprNode      = &UnaryOperationExpr{}
	_ ExprNode      = &ValuesExpr{}
	_ ExprNode      = &VariableExpr{}

	_ Node                = &WhenClause{}
	_ pkast.ColumnRefNode = &ColumnRef{}
)

// BetweenExpr is for "between and" or "not between and" expression.
type BetweenExpr struct {
	pkast.BaseExpr
	// Expr is the expression to be checked.
	Expr ExprNode
	// Left is the expression for minimal value in the range.
	Left ExprNode
	// Right is the expression for maximum value in the range.
	Right ExprNode
	// Not is true, the expression is "not between and".
	Not bool
}

// Restore implements Node interface.
func (n *BetweenExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Expr) || !nodePresent(n.Left) || !nodePresent(n.Right) {
		return errors.New("restore BetweenExpr: missing operand")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore BetweenExpr.Expr")
	}
	if n.Not {
		ctx.WriteKeyword(" NOT BETWEEN ")
	} else {
		ctx.WriteKeyword(" BETWEEN ")
	}
	if err := n.Left.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore BetweenExpr.Left")
	}
	ctx.WriteKeyword(" AND ")
	if err := n.Right.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore BetweenExpr.Right ")
	}
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *BetweenExpr) Format(w io.Writer) {
	FormatExpr(n.Expr, w)
	if n.Not {
		_, _ = fmt.Fprint(w, " NOT BETWEEN ")
	} else {
		_, _ = fmt.Fprint(w, " BETWEEN ")
	}
	FormatExpr(n.Left, w)
	_, _ = fmt.Fprint(w, " AND ")
	FormatExpr(n.Right, w)
}

// Accept implements Node interface.
func (n *BetweenExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, valid := nodeAs[*BetweenExpr](newNode)
	if !valid {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	node, ok := acceptNode[ExprNode](v, n.Expr)
	if !ok {
		return n, false
	}
	n.Expr = node

	node, ok = acceptNode[ExprNode](v, n.Left)
	if !ok {
		return n, false
	}
	n.Left = node

	node, ok = acceptNode[ExprNode](v, n.Right)
	if !ok {
		return n, false
	}
	n.Right = node

	return v.Leave(n)
}

// BinaryOperationExpr is for binary operation like `1 + 1`, `1 - 1`, etc.
type BinaryOperationExpr struct {
	pkast.BaseBinaryExpr
}

// Restore implements Node interface.
func (n *BinaryOperationExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore BinaryOperationExpr: nil expression")
	}
	if !nodePresent(n.Left) {
		return errors.New("BinaryOperationExpr.Left is nil")
	}
	if !nodePresent(n.Right) {
		return errors.New("BinaryOperationExpr.Right is nil")
	}
	if err := n.Left.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred when restore BinaryOperationExpr.Left")
	}
	if ctx.Flags.HasSpacesAroundBinaryOperationFlag() {
		ctx.WritePlain(" ")
	}
	if err := opcode.Op(n.Op).Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred when restore BinaryOperationExpr.Op")
	}
	if ctx.Flags.HasSpacesAroundBinaryOperationFlag() {
		ctx.WritePlain(" ")
	}
	if err := n.Right.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred when restore BinaryOperationExpr.Right")
	}

	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *BinaryOperationExpr) Format(w io.Writer) {
	FormatExpr(n.Left, w)
	_, _ = fmt.Fprint(w, " ")
	opcode.Op(n.Op).Format(w)
	_, _ = fmt.Fprint(w, " ")
	FormatExpr(n.Right, w)
}

// Accept implements Node interface.
func (n *BinaryOperationExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skipChildren := v.Enter(n)
	enterNode, ok := nodeAs[*BinaryOperationExpr](entered)
	if !ok {
		return entered, false
	}
	if skipChildren {
		return v.Leave(enterNode)
	}

	if enterNode.Left != nil {
		next, ok := acceptNode[ExprNode](v, enterNode.Left)
		if !ok {
			return enterNode, false
		}
		enterNode.Left = next
	}

	if enterNode.Right != nil {
		next, ok := acceptNode[ExprNode](v, enterNode.Right)
		if !ok {
			return enterNode, false
		}
		enterNode.Right = next
	}

	return v.Leave(enterNode)
}

// WhenClause is the when clause in Case expression for "when condition then result".
type WhenClause struct {
	node
	// Expr is the condition expression in WhenClause.
	Expr ExprNode
	// Result is the result expression in WhenClause.
	Result ExprNode
}

// Restore implements Node interface.
func (n *WhenClause) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Expr) || !nodePresent(n.Result) {
		return errors.New("restore WhenClause: missing condition or result")
	}
	ctx.WriteKeyword("WHEN ")
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore WhenClauses.Expr")
	}
	ctx.WriteKeyword(" THEN ")
	if err := n.Result.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore WhenClauses.Result")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *WhenClause) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, valid := nodeAs[*WhenClause](newNode)
	if !valid {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	node, ok := acceptNode[ExprNode](v, n.Expr)
	if !ok {
		return n, false
	}
	n.Expr = node

	node, ok = acceptNode[ExprNode](v, n.Result)
	if !ok {
		return n, false
	}
	n.Result = node
	return v.Leave(n)
}

// CaseExpr is the case expression.
type CaseExpr struct {
	pkast.BaseExpr
	// Value is the compare value expression.
	Value ExprNode
	// WhenClauses is the condition check expression.
	WhenClauses []*WhenClause
	// ElseClause is the else result expression.
	ElseClause ExprNode
}

// Restore implements Node interface.
func (n *CaseExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore CaseExpr: nil expression")
	}
	ctx.WriteKeyword("CASE")
	if n.Value != nil {
		if !nodePresent(n.Value) {
			return errors.New("restore CaseExpr: nil value")
		}
		ctx.WritePlain(" ")
		if err := n.Value.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore CaseExpr.Value")
		}
	}
	for _, clause := range n.WhenClauses {
		if clause == nil {
			return errors.New("restore CaseExpr: nil WHEN clause")
		}
		ctx.WritePlain(" ")
		if err := clause.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore CaseExpr.WhenClauses")
		}
	}
	if n.ElseClause != nil {
		if !nodePresent(n.ElseClause) {
			return errors.New("restore CaseExpr: nil ELSE expression")
		}
		ctx.WriteKeyword(" ELSE ")
		if err := n.ElseClause.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore CaseExpr.ElseClause")
		}
	}
	ctx.WriteKeyword(" END")

	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *CaseExpr) Format(w io.Writer) {
	_, _ = fmt.Fprint(w, "CASE")
	// Because the presence of `case when` syntax, `Value` could be nil and we need check this.
	if n.Value != nil {
		_, _ = fmt.Fprint(w, " ")
		FormatExpr(n.Value, w)
	}
	for _, clause := range n.WhenClauses {
		_, _ = fmt.Fprint(w, " ")
		_, _ = fmt.Fprint(w, "WHEN ")
		FormatExpr(clause.Expr, w)
		_, _ = fmt.Fprint(w, " THEN ")
		FormatExpr(clause.Result, w)
	}
	if n.ElseClause != nil {
		_, _ = fmt.Fprint(w, " ELSE ")
		FormatExpr(n.ElseClause, w)
	}
	_, _ = fmt.Fprint(w, " END")
}

// Accept implements Node Accept interface.
func (n *CaseExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, valid := nodeAs[*CaseExpr](newNode)
	if !valid {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	if n.Value != nil {
		node, ok := acceptNode[ExprNode](v, n.Value)
		if !ok {
			return n, false
		}
		n.Value = node
	}
	for i, val := range n.WhenClauses {
		node, ok := acceptNode[*WhenClause](v, val)
		if !ok {
			return n, false
		}
		n.WhenClauses[i] = node
	}
	if n.ElseClause != nil {
		node, ok := acceptNode[ExprNode](v, n.ElseClause)
		if !ok {
			return n, false
		}
		n.ElseClause = node
	}
	return v.Leave(n)
}

// SubqueryExpr represents a subquery.
type SubqueryExpr struct {
	pkast.BaseQueryExpr
	// Query is the query SelectNode.
	Query QueryNode
}

// Restore implements Node interface.
func (n *SubqueryExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Query) {
		return errors.New("restore SubqueryExpr: query is missing")
	}
	ctx.WritePlain("(")
	if err := n.Query.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore SubqueryExpr.Query")
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

// formatViaRestore delegates a Format call to the node's Restore with the
// legacy Format flags: lowercase keywords, backquoted identifiers and
// double-quoted strings. Keywords stay lowercase because MySQL treats them
// as case-insensitive; Format output remains parseable.
func formatViaRestore(node Node, w io.Writer) {
	flags := format.RestoreKeyWordLowercase |
		format.RestoreNameBackQuotes |
		format.RestoreStringDoubleQuotes
	ctx := format.NewRestoreCtx(flags, w)
	if err := node.Restore(ctx); err != nil {
		_, _ = fmt.Fprint(w, "<unsupported>")
	}
}

// Format the ExprNode into a Writer.
func (n *SubqueryExpr) Format(w io.Writer) {
	formatViaRestore(n, w)
}

// Accept implements Node Accept interface.
func (n *SubqueryExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*SubqueryExpr](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(newNode)
	}
	n = outer
	if n.Query == nil {
		return n, false
	}
	node, ok := acceptNode[QueryNode](v, n.Query)
	if !ok {
		return n, false
	}
	n.Query = node
	return v.Leave(n)
}

// CollateExpr represents `expr COLLATE collation`, preserving the collation
// so restore and downstream semantics can use it.
type CollateExpr struct {
	pkast.BaseExpr
	Expr      ExprNode
	Collation string
}

// Restore implements Node interface.
func (n *CollateExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Expr) {
		return errors.New("restore CollateExpr: missing expression")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CollateExpr.Expr")
	}
	ctx.WriteKeyword(" COLLATE ")
	// Write the collation name verbatim: it is data, not a keyword.
	ctx.WritePlain(n.Collation)
	return ctx.Err()
}

// Format implements ExprNode interface.
func (n *CollateExpr) Format(w io.Writer) {
	var sb strings.Builder
	if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
		_, _ = fmt.Fprint(w, err)
		return
	}
	_, _ = fmt.Fprint(w, sb.String())
}

// Accept implements Node interface.
func (n *CollateExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, valid := nodeAs[*CollateExpr](newNode)
	if !valid {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	if n.Expr != nil {
		newExpr, ok := acceptNode[ExprNode](v, n.Expr)
		if !ok {
			return n, false
		}
		n.Expr = newExpr
	}
	return v.Leave(n)
}

// CompareSubqueryExpr is the expression for "expr cmp (select ...)".
// See https://dev.mysql.com/doc/refman/5.7/en/comparisons-using-subqueries.html
// See https://dev.mysql.com/doc/refman/5.7/en/any-in-some-subqueries.html
// See https://dev.mysql.com/doc/refman/5.7/en/all-subqueries.html
type CompareSubqueryExpr struct {
	pkast.BaseExpr
	// L is the left expression
	L ExprNode
	// Op is the comparison opcode.
	Op opcode.Op
	// R is the subquery for right expression, may be rewritten to other type of expression.
	R ExprNode
	// All is true, we should compare all records in subquery.
	All bool
}

// Restore implements Node interface.
func (n *CompareSubqueryExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.L) || !nodePresent(n.R) {
		return errors.New("restore CompareSubqueryExpr: missing operand")
	}
	if err := n.L.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CompareSubqueryExpr.L")
	}
	if err := n.Op.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CompareSubqueryExpr.Op")
	}
	if n.All {
		ctx.WriteKeyword("ALL ")
	} else {
		ctx.WriteKeyword("ANY ")
	}
	if err := n.R.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CompareSubqueryExpr.R")
	}
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *CompareSubqueryExpr) Format(w io.Writer) {
	formatViaRestore(n, w)
}

// Accept implements Node Accept interface.
func (n *CompareSubqueryExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, valid := nodeAs[*CompareSubqueryExpr](newNode)
	if !valid {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	node, ok := acceptNode[ExprNode](v, n.L)
	if !ok {
		return n, false
	}
	n.L = node
	node, ok = acceptNode[ExprNode](v, n.R)
	if !ok {
		return n, false
	}
	n.R = node
	return v.Leave(n)
}

// ColumnRef represents a column reference expression.
type ColumnRef struct {
	pkast.BaseColumnRef
}

// Restore implements Node interface.
func (n *ColumnRef) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return nilRestoreError("ColumnRef")
	}
	if err := restoreQualifiedName(ctx, n.Name); err != nil {
		return errors.Trace(err)
	}
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *ColumnRef) Format(w io.Writer) {
	formatQualifiedName(w, n.Name)
}

// Accept implements Node Accept interface.
func (n *ColumnRef) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*ColumnRef](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	return v.Leave(outer)
}

// DefaultExpr is the default expression using default value for a column.
type DefaultExpr struct {
	pkast.BaseExpr
	// Name is the optional column name for DEFAULT(name).
	Name QualifiedName
}

// Restore implements Node interface.
func (n *DefaultExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return nilRestoreError("DefaultExpr")
	}
	ctx.WriteKeyword("DEFAULT")
	if len(n.Name.Parts) != 0 {
		ctx.WritePlain("(")
		if err := restoreQualifiedName(ctx, n.Name); err != nil {
			return errors.Annotate(err, "An error occurred while restore DefaultExpr.Name")
		}
		ctx.WritePlain(")")
	}
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *DefaultExpr) Format(w io.Writer) {
	formatViaRestore(n, w)
}

// Accept implements Node Accept interface.
func (n *DefaultExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*DefaultExpr](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	return v.Leave(outer)
}

// ExistsSubqueryExpr is the expression for "exists (select ...)".
// See https://dev.mysql.com/doc/refman/5.7/en/exists-and-not-exists-subqueries.html
type ExistsSubqueryExpr struct {
	pkast.BaseExpr
	// Sel is the subquery, may be rewritten to other type of expression.
	Sel ExprNode
	// Not is true, the expression is "not exists".
	Not bool
}

// Restore implements Node interface.
func (n *ExistsSubqueryExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Sel) {
		return errors.New("restore ExistsSubqueryExpr: missing subquery")
	}
	if n.Not {
		ctx.WriteKeyword("NOT EXISTS ")
	} else {
		ctx.WriteKeyword("EXISTS ")
	}
	if err := n.Sel.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore ExistsSubqueryExpr.Sel")
	}
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *ExistsSubqueryExpr) Format(w io.Writer) {
	formatViaRestore(n, w)
}

// Accept implements Node Accept interface.
func (n *ExistsSubqueryExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, valid := nodeAs[*ExistsSubqueryExpr](newNode)
	if !valid {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	node, ok := acceptNode[ExprNode](v, n.Sel)
	if !ok {
		return n, false
	}
	n.Sel = node
	return v.Leave(n)
}

// PatternInExpr is the expression for in operator, like "expr in (1, 2, 3)" or "expr in (select c from t)".
type PatternInExpr struct {
	pkast.BaseExpr
	// Expr is the value expression to be compared.
	Expr ExprNode
	// List is the list expression in compare list.
	List []ExprNode
	// Not is true, the expression is "not in".
	Not bool
	// Sel is the subquery, may be rewritten to other type of expression.
	Sel ExprNode
}

// Restore implements Node interface.
func (n *PatternInExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Expr) {
		return errors.New("restore PatternInExpr: missing expression")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore PatternInExpr.Expr")
	}
	if n.Not {
		ctx.WriteKeyword(" NOT IN ")
	} else {
		ctx.WriteKeyword(" IN ")
	}
	if n.Sel != nil {
		if !nodePresent(n.Sel) {
			return errors.New("restore PatternInExpr: nil subquery")
		}
		if err := n.Sel.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore PatternInExpr.Sel")
		}
	} else {
		ctx.WritePlain("(")
		for i, expr := range n.List {
			if !nodePresent(expr) {
				return errors.Errorf("restore PatternInExpr.List[%d]: missing expression", i)
			}
			if i != 0 {
				ctx.WritePlain(",")
			}
			if err := expr.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore PatternInExpr.List[%d]", i)
			}
		}
		ctx.WritePlain(")")
	}
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *PatternInExpr) Format(w io.Writer) {
	FormatExpr(n.Expr, w)
	if n.Not {
		_, _ = fmt.Fprint(w, " NOT IN (")
	} else {
		_, _ = fmt.Fprint(w, " IN (")
	}
	for i, expr := range n.List {
		if i != 0 {
			_, _ = fmt.Fprint(w, ",")
		}
		FormatExpr(expr, w)
	}
	_, _ = fmt.Fprint(w, ")")
}

// Accept implements Node Accept interface.
func (n *PatternInExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, valid := nodeAs[*PatternInExpr](newNode)
	if !valid {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	node, ok := acceptNode[ExprNode](v, n.Expr)
	if !ok {
		return n, false
	}
	n.Expr = node
	for i, val := range n.List {
		node, ok = acceptNode[ExprNode](v, val)
		if !ok {
			return n, false
		}
		n.List[i] = node
	}
	if n.Sel != nil {
		node, ok = acceptNode[ExprNode](v, n.Sel)
		if !ok {
			return n, false
		}
		n.Sel = node
	}
	return v.Leave(n)
}

// IsNullExpr is the expression for null check.
type IsNullExpr struct {
	pkast.BaseExpr
	// Expr is the expression to be checked.
	Expr ExprNode
	// Not is true, the expression is "is not null".
	Not bool
}

// Restore implements Node interface.
func (n *IsNullExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Expr) {
		return errors.New("restore IsNullExpr: missing expression")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Trace(err)
	}
	if n.Not {
		ctx.WriteKeyword(" IS NOT NULL")
	} else {
		ctx.WriteKeyword(" IS NULL")
	}
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *IsNullExpr) Format(w io.Writer) {
	FormatExpr(n.Expr, w)
	if n.Not {
		_, _ = fmt.Fprint(w, " IS NOT NULL")
		return
	}
	_, _ = fmt.Fprint(w, " IS NULL")
}

// Accept implements Node Accept interface.
func (n *IsNullExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, valid := nodeAs[*IsNullExpr](newNode)
	if !valid {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	node, ok := acceptNode[ExprNode](v, n.Expr)
	if !ok {
		return n, false
	}
	n.Expr = node
	return v.Leave(n)
}

// IsTruthExpr is the expression for true/false check.
type IsTruthExpr struct {
	pkast.BaseExpr
	// Expr is the expression to be checked.
	Expr ExprNode
	// Not is true, the expression is "is not true/false".
	Not bool
	// True indicates checking true or false.
	True int64
}

// Restore implements Node interface.
func (n *IsTruthExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Expr) {
		return errors.New("restore IsTruthExpr: missing expression")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Trace(err)
	}
	if n.Not {
		ctx.WriteKeyword(" IS NOT")
	} else {
		ctx.WriteKeyword(" IS")
	}
	if n.True > 0 {
		ctx.WriteKeyword(" TRUE")
	} else {
		ctx.WriteKeyword(" FALSE")
	}
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *IsTruthExpr) Format(w io.Writer) {
	FormatExpr(n.Expr, w)
	if n.Not {
		_, _ = fmt.Fprint(w, " IS NOT")
	} else {
		_, _ = fmt.Fprint(w, " IS")
	}
	if n.True > 0 {
		_, _ = fmt.Fprint(w, " TRUE")
	} else {
		_, _ = fmt.Fprint(w, " FALSE")
	}
}

// Accept implements Node Accept interface.
func (n *IsTruthExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, valid := nodeAs[*IsTruthExpr](newNode)
	if !valid {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	node, ok := acceptNode[ExprNode](v, n.Expr)
	if !ok {
		return n, false
	}
	n.Expr = node
	return v.Leave(n)
}

// PatternLikeExpr is the expression for like operator, e.g, expr like "%123%"
type PatternLikeExpr struct {
	pkast.BaseExpr
	// Expr is the expression to be checked.
	Expr ExprNode
	// Pattern is the like expression.
	Pattern ExprNode
	// Not is true, the expression is "not like".
	Not bool

	Escape     byte
	EscapeExpr ExprNode
}

// Restore implements Node interface.
func (n *PatternLikeExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Expr) || !nodePresent(n.Pattern) {
		return errors.New("restore PatternLikeExpr: missing operand")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore PatternLikeExpr.Expr")
	}

	if n.Not {
		ctx.WriteKeyword(" NOT LIKE ")
	} else {
		ctx.WriteKeyword(" LIKE ")
	}

	if err := n.Pattern.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore PatternLikeExpr.Pattern")
	}

	if n.EscapeExpr != nil {
		if !nodePresent(n.EscapeExpr) {
			return errors.New("restore PatternLikeExpr: nil escape expression")
		}
		ctx.WriteKeyword(" ESCAPE ")
		if err := n.EscapeExpr.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore PatternLikeExpr.EscapeExpr")
		}
	} else if escape := string(n.Escape); escape != "" && escape != "\\" {
		ctx.WriteKeyword(" ESCAPE ")
		ctx.WriteString(escape)
	}
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *PatternLikeExpr) Format(w io.Writer) {
	FormatExpr(n.Expr, w)
	if n.Not {
		_, _ = fmt.Fprint(w, " NOT LIKE ")
	} else {
		_, _ = fmt.Fprint(w, " LIKE ")
	}
	FormatExpr(n.Pattern, w)
	if n.EscapeExpr != nil {
		// Escape expressions keep their syntax form; legacy datum nodes have no escape spelling.
		_, _ = fmt.Fprint(w, " ESCAPE ")
		if literal, ok := n.EscapeExpr.(*LiteralExpr); ok && literal.Kind == pkast.LiteralString {
			_, _ = fmt.Fprintf(w, "'%s'", strings.ReplaceAll(literal.Value, "'", "''"))
		} else {
			FormatExpr(n.EscapeExpr, w)
		}
	} else if n.Escape != 0 && n.Escape != '\\' {
		// A zero escape byte is the default, not a written ESCAPE clause.
		_, _ = fmt.Fprint(w, " ESCAPE ")
		_, _ = fmt.Fprintf(w, "'%c'", n.Escape)
	}
}

// Accept implements Node Accept interface.
func (n *PatternLikeExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, valid := nodeAs[*PatternLikeExpr](newNode)
	if !valid {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	if n.Expr != nil {
		node, ok := acceptNode[ExprNode](v, n.Expr)
		if !ok {
			return n, false
		}
		n.Expr = node
	}
	if n.Pattern != nil {
		node, ok := acceptNode[ExprNode](v, n.Pattern)
		if !ok {
			return n, false
		}
		n.Pattern = node
	}
	if n.EscapeExpr != nil {
		node, ok := acceptNode[ExprNode](v, n.EscapeExpr)
		if !ok {
			return n, false
		}
		n.EscapeExpr = node
	}
	return v.Leave(n)
}

// ParenthesesExpr is the parentheses expression.
type ParenthesesExpr struct {
	pkast.BaseExpr
	// Expr is the expression in parentheses.
	Expr ExprNode
}

// Restore implements Node interface.
func (n *ParenthesesExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Expr) {
		return errors.New("restore ParenthesesExpr: missing expression")
	}
	ctx.WritePlain("(")
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred when restore ParenthesesExpr.Expr")
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *ParenthesesExpr) Format(w io.Writer) {
	_, _ = fmt.Fprint(w, "(")
	FormatExpr(n.Expr, w)
	_, _ = fmt.Fprint(w, ")")
}

// Accept implements Node Accept interface.
func (n *ParenthesesExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, valid := nodeAs[*ParenthesesExpr](newNode)
	if !valid {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	if n.Expr != nil {
		node, ok := acceptNode[ExprNode](v, n.Expr)
		if !ok {
			return n, false
		}
		n.Expr = node
	}
	return v.Leave(n)
}

// PositionExpr is the expression for order by and group by position.
// MySQL use position expression started from 1, it looks a little confused inner.
// maybe later we will use 0 at first.
type PositionExpr struct {
	pkast.BaseExpr
	// N is the position, started from 1 now.
	N int
	// P is the parameterized position.
	P ExprNode
}

// Restore implements Node interface.
func (n *PositionExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return nilRestoreError("PositionExpr")
	}
	ctx.WritePlainf("%d", n.N)
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *PositionExpr) Format(w io.Writer) {
	formatViaRestore(n, w)
}

// Accept implements Node Accept interface.
func (n *PositionExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, valid := nodeAs[*PositionExpr](newNode)
	if !valid {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	if n.P != nil {
		node, ok := acceptNode[ExprNode](v, n.P)
		if !ok {
			return n, false
		}
		n.P = node
	}
	return v.Leave(n)
}

// PatternRegexpExpr is the pattern expression for pattern match.
type PatternRegexpExpr struct {
	pkast.BaseExpr
	// Expr is the expression to be checked.
	Expr ExprNode
	// Pattern is the expression for pattern.
	Pattern ExprNode
	// Not is true, the expression is "not rlike",
	Not bool
}

// Restore implements Node interface.
func (n *PatternRegexpExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Expr) || !nodePresent(n.Pattern) {
		return errors.New("restore PatternRegexpExpr: missing operand")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore PatternRegexpExpr.Expr")
	}

	if n.Not {
		ctx.WriteKeyword(" NOT REGEXP ")
	} else {
		ctx.WriteKeyword(" REGEXP ")
	}

	if err := n.Pattern.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore PatternRegexpExpr.Pattern")
	}

	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *PatternRegexpExpr) Format(w io.Writer) {
	FormatExpr(n.Expr, w)
	if n.Not {
		_, _ = fmt.Fprint(w, " NOT REGEXP ")
	} else {
		_, _ = fmt.Fprint(w, " REGEXP ")
	}
	FormatExpr(n.Pattern, w)
}

// Accept implements Node Accept interface.
func (n *PatternRegexpExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, valid := nodeAs[*PatternRegexpExpr](newNode)
	if !valid {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	node, ok := acceptNode[ExprNode](v, n.Expr)
	if !ok {
		return n, false
	}
	n.Expr = node
	node, ok = acceptNode[ExprNode](v, n.Pattern)
	if !ok {
		return n, false
	}
	n.Pattern = node
	return v.Leave(n)
}

// RowExpr is the expression for row constructor.
// See https://dev.mysql.com/doc/refman/5.7/en/row-subqueries.html
// MemberOfExpr is the JSON MEMBER OF predicate.
// See https://dev.mysql.com/doc/refman/8.0/en/json-search-functions.html#operator_member-of
type MemberOfExpr struct {
	pkast.BaseExpr
	// Value is the value to search for.
	Value ExprNode
	// Array is the JSON array expression.
	Array ExprNode
}

// Restore implements Node interface.
func (n *MemberOfExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Value) || !nodePresent(n.Array) {
		return errors.New("restore MemberOfExpr: missing operand")
	}
	if err := n.Value.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore MemberOfExpr.Value")
	}
	ctx.WriteKeyword(" MEMBER OF (")
	if err := n.Array.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore MemberOfExpr.Array")
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *MemberOfExpr) Format(w io.Writer) {
	FormatExpr(n.Value, w)
	_, _ = fmt.Fprint(w, " MEMBER OF (")
	FormatExpr(n.Array, w)
	_, _ = fmt.Fprint(w, ")")
}

// Accept implements Node Accept interface.
func (n *MemberOfExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, valid := nodeAs[*MemberOfExpr](newNode)
	if !valid {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	node, ok := acceptNode[ExprNode](v, n.Value)
	if !ok {
		return n, false
	}
	n.Value = node
	node, ok = acceptNode[ExprNode](v, n.Array)
	if !ok {
		return n, false
	}
	n.Array = node
	return v.Leave(n)
}

// IsJSONExpr is the IS [NOT] JSON predicate.
// See https://mariadb.com/docs/server/reference/sql-structure/operators/comparison-operators/is-json
type IsJSONExpr struct {
	pkast.BaseExpr
	// Expr is the expression to be checked.
	Expr ExprNode
	// Not is true, the expression is "is not json".
	Not bool
	// Type is the optional MariaDB 12.3 JSON type constraint: VALUE, ARRAY, OBJECT, SCALAR.
	Type string
	// UniqueKeys is empty, WITH UNIQUE KEYS, or WITHOUT UNIQUE KEYS.
	UniqueKeys string
}

// Restore implements Node interface.
func (n *IsJSONExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Expr) {
		return errors.New("restore IsJSONExpr: missing expression")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore IsJSONExpr.Expr")
	}
	if n.Not {
		ctx.WriteKeyword(" IS NOT JSON")
	} else {
		ctx.WriteKeyword(" IS JSON")
	}
	if n.Type != "" {
		ctx.WritePlain(" ")
		ctx.WriteKeyword(n.Type)
	}
	if n.UniqueKeys != "" {
		ctx.WritePlain(" ")
		ctx.WriteKeyword(n.UniqueKeys)
	}
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *IsJSONExpr) Format(w io.Writer) {
	ctx := format.NewRestoreCtx(format.RestoreKeyWordUppercase|format.RestoreNameBackQuotes|format.RestoreStringDoubleQuotes, w)
	FormatExpr(n.Expr, w)
	if n.Not {
		ctx.WriteKeyword(" IS NOT JSON")
	} else {
		ctx.WriteKeyword(" IS JSON")
	}
	if n.Type != "" {
		ctx.WritePlain(" ")
		ctx.WriteKeyword(n.Type)
	}
	if n.UniqueKeys != "" {
		ctx.WritePlain(" ")
		ctx.WriteKeyword(n.UniqueKeys)
	}
}

// Accept implements Node Accept interface.
func (n *IsJSONExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, valid := nodeAs[*IsJSONExpr](newNode)
	if !valid {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	node, ok := acceptNode[ExprNode](v, n.Expr)
	if !ok {
		return n, false
	}
	n.Expr = node
	return v.Leave(n)
}

// MatchAgainstModifier is the full-text search modifier of MATCH ... AGAINST.
type MatchAgainstModifier int

const (
	MatchAgainstNaturalLanguage MatchAgainstModifier = iota
	MatchAgainstBoolean
)

// MatchAgainst is the MATCH (col, ...) AGAINST ('expr') expression.
// See https://dev.mysql.com/doc/refman/8.0/en/fulltext-search.html
type MatchAgainst struct {
	pkast.BaseExpr

	Columns        []QualifiedName
	Expr           ExprNode
	Modifier       MatchAgainstModifier
	QueryExpansion bool
}

// Restore implements Node interface.
func (n *MatchAgainst) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Expr) {
		return errors.New("restore MatchAgainst: missing expression")
	}
	ctx.WriteKeyword("MATCH (")
	for i, col := range n.Columns {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := restoreQualifiedName(ctx, col); err != nil {
			return errors.Annotatef(err, "An error occurred while restore MatchAgainst.Columns[%d]", i)
		}
	}
	ctx.WritePlain(") ")
	ctx.WriteKeyword("AGAINST (")
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore MatchAgainst.Expr")
	}
	if n.Modifier == MatchAgainstBoolean {
		ctx.WriteKeyword(" IN BOOLEAN MODE")
	} else if n.QueryExpansion {
		ctx.WriteKeyword(" WITH QUERY EXPANSION")
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *MatchAgainst) Format(w io.Writer) {
	// Upper keyword case and double-quoted strings match the legacy Format
	// spelling pinned by the consumer tests (compat_extensions_test).
	ctx := format.NewRestoreCtx(format.RestoreKeyWordUppercase|format.RestoreNameBackQuotes|format.RestoreStringDoubleQuotes, w)
	ctx.WriteKeyword("MATCH (")
	for i, col := range n.Columns {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		ctx.WritePlain(originalQualifiedName(col))
	}
	ctx.WritePlain(") ")
	ctx.WriteKeyword("AGAINST (")
	FormatExpr(n.Expr, w)
	if n.Modifier == MatchAgainstBoolean {
		ctx.WriteKeyword(" IN BOOLEAN MODE")
	} else if n.QueryExpansion {
		ctx.WriteKeyword(" WITH QUERY EXPANSION")
	}
	ctx.WritePlain(")")
}

// Accept implements Node Accept interface.
func (n *MatchAgainst) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, valid := nodeAs[*MatchAgainst](newNode)
	if !valid {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	node, ok := acceptNode[ExprNode](v, n.Expr)
	if !ok {
		return n, false
	}
	n.Expr = node
	return v.Leave(n)
}

// SoundsLikeExpr is the expr SOUNDS LIKE expr predicate.
// See https://dev.mysql.com/doc/refman/8.0/en/string-functions.html#operator_sounds-like
type SoundsLikeExpr struct {
	pkast.BaseExpr

	Expr    ExprNode
	Pattern ExprNode
}

// Restore implements Node interface.
func (n *SoundsLikeExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Expr) || !nodePresent(n.Pattern) {
		return errors.New("restore SoundsLikeExpr: missing operand")
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore SoundsLikeExpr.Expr")
	}
	ctx.WriteKeyword(" SOUNDS LIKE ")
	if err := n.Pattern.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore SoundsLikeExpr.Pattern")
	}
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *SoundsLikeExpr) Format(w io.Writer) {
	ctx := format.NewRestoreCtx(format.RestoreKeyWordUppercase|format.RestoreNameBackQuotes|format.RestoreStringDoubleQuotes, w)
	FormatExpr(n.Expr, w)
	ctx.WriteKeyword(" SOUNDS LIKE ")
	FormatExpr(n.Pattern, w)
}

// Accept implements Node Accept interface.
func (n *SoundsLikeExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, valid := nodeAs[*SoundsLikeExpr](newNode)
	if !valid {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	node, ok := acceptNode[ExprNode](v, n.Expr)
	if !ok {
		return n, false
	}
	n.Expr = node
	node, ok = acceptNode[ExprNode](v, n.Pattern)
	if !ok {
		return n, false
	}
	n.Pattern = node
	return v.Leave(n)
}

type RowExpr struct {
	pkast.BaseRowExpr
	// OmitRowKeyword preserves MariaDB's `VALUES (expr, ...)` row syntax.
	OmitRowKeyword bool
}

// RowCore lets role-based parents check for a missing row before dispatch.
func (n *RowExpr) RowCore() *pkast.BaseRowExpr {
	if n == nil {
		return nil
	}
	return &n.BaseRowExpr
}

// Restore implements Node interface.
func (n *RowExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore RowExpr: nil row")
	}
	if !n.OmitRowKeyword {
		ctx.WriteKeyword("ROW")
	}
	ctx.WritePlain("(")
	for i, v := range n.Items {
		if i != 0 {
			ctx.WritePlain(",")
		}
		if !nodePresent(v) {
			return errors.Errorf("RowExpr.Items[%d] is nil", i)
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred when restore RowExpr.Items[%v]", i)
		}
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *RowExpr) Format(w io.Writer) {
	formatViaRestore(n, w)
}

// Accept implements Node Accept interface.
func (n *RowExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*RowExpr](entered)
	if !ok {
		return entered, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	for i, val := range outer.Items {
		if val == nil {
			continue
		}
		expr, ok := acceptNode[ExprNode](v, val)
		if !ok {
			return outer, false
		}
		outer.Items[i] = expr
	}
	return v.Leave(outer)
}

// UnaryOperationExpr is the expression for unary operator.
type UnaryOperationExpr struct {
	pkast.BaseUnaryExpr
}

// Restore implements Node interface.
func (n *UnaryOperationExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Expr) {
		return errors.New("UnaryOperationExpr.Expr is nil")
	}
	if err := opcode.Op(n.Op).Restore(ctx); err != nil {
		return errors.Trace(err)
	}
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Trace(err)
	}
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *UnaryOperationExpr) Format(w io.Writer) {
	opcode.Op(n.Op).Format(w)
	if n.Expr != nil {
		FormatExpr(n.Expr, w)
	}
}

// Accept implements Node Accept interface.
func (n *UnaryOperationExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skipChildren := v.Enter(n)
	enterNode, ok := nodeAs[*UnaryOperationExpr](entered)
	if !ok {
		return entered, false
	}
	if skipChildren {
		return v.Leave(enterNode)
	}
	if enterNode.Expr != nil {
		next, ok := acceptNode[ExprNode](v, enterNode.Expr)
		if !ok {
			return enterNode, false
		}
		enterNode.Expr = next
	}
	return v.Leave(enterNode)
}

// ValuesExpr is the expression used in INSERT VALUES.
type ValuesExpr struct {
	pkast.BaseExpr
	// Column is column name.
	Column *ColumnRef
	// Singular preserves MariaDB's VALUE(column) spelling.
	Singular bool
}

// Restore implements Node interface.
func (n *ValuesExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Column) {
		return errors.New("ValuesExpr.Column is nil")
	}
	if n.Singular {
		ctx.WriteKeyword("VALUE")
	} else {
		ctx.WriteKeyword("VALUES")
	}
	ctx.WritePlain("(")
	if err := n.Column.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore ValuesExpr.Column")
	}
	ctx.WritePlain(")")

	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *ValuesExpr) Format(w io.Writer) {
	if n.Singular {
		_, _ = fmt.Fprint(w, "VALUE(")
	} else {
		_, _ = fmt.Fprint(w, "VALUES(")
	}
	n.Column.Format(w)
	_, _ = fmt.Fprint(w, ")")
}

// Accept implements Node Accept interface.
func (n *ValuesExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	updated, ok := newNode.(*ValuesExpr)
	if !ok || updated == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(updated)
	}
	n = updated
	if !nodePresent(n.Column) {
		return n, false
	}
	node, ok := n.Column.Accept(v)
	if !ok {
		return n, false
	}
	column, ok := node.(*ColumnRef)
	if !ok || column == nil {
		return n, false
	}
	n.Column = column
	return v.Leave(n)
}

// VariableExpr is the expression for variable.
type VariableExpr struct {
	pkast.BaseExpr
	// Target is the variable reference. It is either a user or system target.
	Target *VariableTarget
	// Value is the variable value.
	Value ExprNode
}

// Restore implements Node interface.
func (n *VariableExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Target == nil {
		return errors.New("ast: variable expression has no target")
	}
	if err := n.Target.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore VariableExpr.Target")
	}

	if n.Value != nil {
		if !nodePresent(n.Value) {
			return errors.New("restore VariableExpr: nil value")
		}
		ctx.WritePlain(":=")
		if err := n.Value.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore VariableExpr.Value")
		}
	}

	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *VariableExpr) Format(w io.Writer) {
	formatViaRestore(n, w)
}

// Accept implements Node Accept interface.
func (n *VariableExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		replacement, ok := nodeAs[ExprNode](newNode)
		if !ok {
			return newNode, false
		}
		return v.Leave(replacement)
	}
	outer, valid := nodeAs[*VariableExpr](newNode)
	if !valid {
		return newNode, false
	}
	n = outer
	if n.Target != nil {
		node, ok := acceptNode[*VariableTarget](v, n.Target)
		if !ok {
			return n, false
		}
		n.Target = node
	}
	if n.Value != nil {
		node, ok := acceptNode[ExprNode](v, n.Value)
		if !ok {
			return n, false
		}
		n.Value = node
	}
	return v.Leave(n)
}

// MaxValueExpr is the expression for "maxvalue" used in partition.
type MaxValueExpr struct {
	pkast.BaseExpr
}

// Restore implements Node interface.
func (n *MaxValueExpr) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return nilRestoreError("MaxValueExpr")
	}
	ctx.WriteKeyword("MAXVALUE")
	return ctx.Err()
}

// Format the ExprNode into a Writer.
func (n *MaxValueExpr) Format(w io.Writer) {
	_, _ = fmt.Fprint(w, "MAXVALUE")
}

// Accept implements Node Accept interface.
func (n *MaxValueExpr) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*MaxValueExpr](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	return v.Leave(outer)
}

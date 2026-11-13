package ast

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/mia0x75/myparser/errors"

	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/model"
	"github.com/mia0x75/myparser/opcode"
)

var (
	_ ExprNode = &BetweenExpr{}
	_ ExprNode = &BinaryOperationExpr{}
	_ ExprNode = &CaseExpr{}
	_ ExprNode = &CollateExpr{}
	_ ExprNode = &ColumnNameExpr{}
	_ ExprNode = &CompareSubqueryExpr{}
	_ ExprNode = &DefaultExpr{}
	_ ExprNode = &ExistsSubqueryExpr{}
	_ ExprNode = &IsJSONExpr{}
	_ ExprNode = &IsNullExpr{}
	_ ExprNode = &IsTruthExpr{}
	_ ExprNode = &MatchAgainst{}
	_ ExprNode = &MemberOfExpr{}
	_ ExprNode = &ParenthesesExpr{}
	_ ExprNode = &PatternInExpr{}
	_ ExprNode = &PatternLikeExpr{}
	_ ExprNode = &PatternRegexpExpr{}
	_ ExprNode = &PositionExpr{}
	_ ExprNode = &RowExpr{}
	_ ExprNode = &SoundsLikeExpr{}
	_ ExprNode = &SubqueryExpr{}
	_ ExprNode = &UnaryOperationExpr{}
	_ ExprNode = &ValuesExpr{}
	_ ExprNode = &VariableExpr{}

	_ Node = &ColumnName{}
	_ Node = &WhenClause{}
)

// ValueExpr define a interface for ValueExpr.
type ValueExpr interface {
	ExprNode
	SetValue(val any)
	GetValue() any
	GetDatumString() string
	GetString() string
	GetProjectionOffset() int
	SetProjectionOffset(offset int)
}

// NewValueExpr creates a ValueExpr with value, and sets default field type.
var NewValueExpr func(any) ValueExpr

// NewParamMarkerExpr creates a ParamMarkerExpr.
var NewParamMarkerExpr func(offset int) ParamMarkerExpr

// BetweenExpr is for "between and" or "not between and" expression.
type BetweenExpr struct {
	exprNode
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
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore BetweenExpr.Expr")
	}
	if n.Not {
		ctx.WriteKeyWord(" NOT BETWEEN ")
	} else {
		ctx.WriteKeyWord(" BETWEEN ")
	}
	if err := n.Left.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore BetweenExpr.Left")
	}
	ctx.WriteKeyWord(" AND ")
	if err := n.Right.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore BetweenExpr.Right ")
	}
	return nil
}

// Format the ExprNode into a Writer.
func (n *BetweenExpr) Format(w io.Writer) {
	n.Expr.Format(w)
	if n.Not {
		_, _ = fmt.Fprint(w, " NOT BETWEEN ")
	} else {
		_, _ = fmt.Fprint(w, " BETWEEN ")
	}
	n.Left.Format(w)
	_, _ = fmt.Fprint(w, " AND ")
	n.Right.Format(w)
}

// Accept implements Node interface.
func (n *BetweenExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}

	n = newNode.(*BetweenExpr)
	node, ok := n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	n.Expr = node.(ExprNode)

	node, ok = n.Left.Accept(v)
	if !ok {
		return n, false
	}
	n.Left = node.(ExprNode)

	node, ok = n.Right.Accept(v)
	if !ok {
		return n, false
	}
	n.Right = node.(ExprNode)

	return v.Leave(n)
}

// BinaryOperationExpr is for binary operation like `1 + 1`, `1 - 1`, etc.
type BinaryOperationExpr struct {
	exprNode
	// Op is the operator code for BinaryOperation.
	Op opcode.Op
	// L is the left expression in BinaryOperation.
	L ExprNode
	// R is the right expression in BinaryOperation.
	R ExprNode
}

// Restore implements Node interface.
func (n *BinaryOperationExpr) Restore(ctx *format.RestoreCtx) error {
	if err := n.L.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred when restore BinaryOperationExpr.L")
	}
	if ctx.Flags.HasSpacesAroundBinaryOperationFlag() {
		ctx.WritePlain(" ")
	}
	if err := n.Op.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred when restore BinaryOperationExpr.Op")
	}
	if ctx.Flags.HasSpacesAroundBinaryOperationFlag() {
		ctx.WritePlain(" ")
	}
	if err := n.R.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred when restore BinaryOperationExpr.R")
	}

	return nil
}

// Format the ExprNode into a Writer.
func (n *BinaryOperationExpr) Format(w io.Writer) {
	n.L.Format(w)
	_, _ = fmt.Fprint(w, " ")
	n.Op.Format(w)
	_, _ = fmt.Fprint(w, " ")
	n.R.Format(w)
}

// Accept implements Node interface.
func (n *BinaryOperationExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}

	n = newNode.(*BinaryOperationExpr)
	node, ok := n.L.Accept(v)
	if !ok {
		return n, false
	}
	n.L = node.(ExprNode)

	node, ok = n.R.Accept(v)
	if !ok {
		return n, false
	}
	n.R = node.(ExprNode)

	return v.Leave(n)
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
	ctx.WriteKeyWord("WHEN ")
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore WhenClauses.Expr")
	}
	ctx.WriteKeyWord(" THEN ")
	if err := n.Result.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore WhenClauses.Result")
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *WhenClause) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}

	n = newNode.(*WhenClause)
	node, ok := n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	n.Expr = node.(ExprNode)

	node, ok = n.Result.Accept(v)
	if !ok {
		return n, false
	}
	n.Result = node.(ExprNode)
	return v.Leave(n)
}

// CaseExpr is the case expression.
type CaseExpr struct {
	exprNode
	// Value is the compare value expression.
	Value ExprNode
	// WhenClauses is the condition check expression.
	WhenClauses []*WhenClause
	// ElseClause is the else result expression.
	ElseClause ExprNode
}

// Restore implements Node interface.
func (n *CaseExpr) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("CASE")
	if n.Value != nil {
		ctx.WritePlain(" ")
		if err := n.Value.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore CaseExpr.Value")
		}
	}
	for _, clause := range n.WhenClauses {
		ctx.WritePlain(" ")
		if err := clause.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore CaseExpr.WhenClauses")
		}
	}
	if n.ElseClause != nil {
		ctx.WriteKeyWord(" ELSE ")
		if err := n.ElseClause.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore CaseExpr.ElseClause")
		}
	}
	ctx.WriteKeyWord(" END")

	return nil
}

// Format the ExprNode into a Writer.
func (n *CaseExpr) Format(w io.Writer) {
	_, _ = fmt.Fprint(w, "CASE")
	// Because the presence of `case when` syntax, `Value` could be nil and we need check this.
	if n.Value != nil {
		_, _ = fmt.Fprint(w, " ")
		n.Value.Format(w)
	}
	for _, clause := range n.WhenClauses {
		_, _ = fmt.Fprint(w, " ")
		_, _ = fmt.Fprint(w, "WHEN ")
		clause.Expr.Format(w)
		_, _ = fmt.Fprint(w, " THEN ")
		clause.Result.Format(w)
	}
	if n.ElseClause != nil {
		_, _ = fmt.Fprint(w, " ELSE ")
		n.ElseClause.Format(w)
	}
	_, _ = fmt.Fprint(w, " END")
}

// Accept implements Node Accept interface.
func (n *CaseExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}

	n = newNode.(*CaseExpr)
	if n.Value != nil {
		node, ok := n.Value.Accept(v)
		if !ok {
			return n, false
		}
		n.Value = node.(ExprNode)
	}
	for i, val := range n.WhenClauses {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.WhenClauses[i] = node.(*WhenClause)
	}
	if n.ElseClause != nil {
		node, ok := n.ElseClause.Accept(v)
		if !ok {
			return n, false
		}
		n.ElseClause = node.(ExprNode)
	}
	return v.Leave(n)
}

// SubqueryExpr represents a subquery.
type SubqueryExpr struct {
	exprNode
	// Query is the query SelectNode.
	Query      ResultSetNode
	Evaluated  bool
	Correlated bool
	MultiRows  bool
	Exists     bool
}

// Restore implements Node interface.
func (n *SubqueryExpr) Restore(ctx *format.RestoreCtx) error {
	ctx.WritePlain("(")
	if err := n.Query.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore SubqueryExpr.Query")
	}
	ctx.WritePlain(")")
	return nil
}

// formatViaRestore delegates a Format call to the node's Restore with flags
// that match the existing Format output style (lowercase keywords, backquoted
// identifiers, double-quoted strings) so that the 24 already-implemented
// Format bodies stay consistent with the stubs.
func formatViaRestore(node Node, w io.Writer) {
	flags := format.RestoreKeyWordLowercase |
		format.RestoreNameBackQuotes |
		format.RestoreStringDoubleQuotes
	ctx := format.NewRestoreCtx(flags, w)
	_ = node.Restore(ctx)
}

// Format the ExprNode into a Writer.
func (n *SubqueryExpr) Format(w io.Writer) {
	formatViaRestore(n, w)
}

// Accept implements Node Accept interface.
func (n *SubqueryExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*SubqueryExpr)
	node, ok := n.Query.Accept(v)
	if !ok {
		return n, false
	}
	n.Query = node.(ResultSetNode)
	return v.Leave(n)
}

// CollateExpr represents `expr COLLATE collation`, preserving the collation
// so restore and downstream semantics can use it.
type CollateExpr struct {
	exprNode
	Expr      ExprNode
	Collation string
}

// Restore implements Node interface.
func (n *CollateExpr) Restore(ctx *format.RestoreCtx) error {
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CollateExpr.Expr")
	}
	ctx.WriteKeyWord(" COLLATE ")
	// Write the collation name verbatim: it is data, not a keyword.
	ctx.WritePlain(n.Collation)
	return nil
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
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CollateExpr)
	if n.Expr != nil {
		newExpr, _ := n.Expr.Accept(v)
		n.Expr = newExpr.(ExprNode)
	}
	return v.Leave(n)
}

// CompareSubqueryExpr is the expression for "expr cmp (select ...)".
// See https://dev.mysql.com/doc/refman/5.7/en/comparisons-using-subqueries.html
// See https://dev.mysql.com/doc/refman/5.7/en/any-in-some-subqueries.html
// See https://dev.mysql.com/doc/refman/5.7/en/all-subqueries.html
type CompareSubqueryExpr struct {
	exprNode
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
	if err := n.L.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CompareSubqueryExpr.L")
	}
	if err := n.Op.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CompareSubqueryExpr.Op")
	}
	if n.All {
		ctx.WriteKeyWord("ALL ")
	} else {
		ctx.WriteKeyWord("ANY ")
	}
	if err := n.R.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CompareSubqueryExpr.R")
	}
	return nil
}

// Format the ExprNode into a Writer.
func (n *CompareSubqueryExpr) Format(w io.Writer) {
	formatViaRestore(n, w)
}

// Accept implements Node Accept interface.
func (n *CompareSubqueryExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CompareSubqueryExpr)
	node, ok := n.L.Accept(v)
	if !ok {
		return n, false
	}
	n.L = node.(ExprNode)
	node, ok = n.R.Accept(v)
	if !ok {
		return n, false
	}
	n.R = node.(ExprNode)
	return v.Leave(n)
}

// ColumnName represents column name.
type ColumnName struct {
	node
	Schema model.CIStr
	Table  model.CIStr
	Name   model.CIStr
}

// Restore implements Node interface.
func (n *ColumnName) Restore(ctx *format.RestoreCtx) error {
	if n.Schema.O != "" {
		ctx.WriteName(n.Schema.O)
		ctx.WritePlain(".")
	}
	if n.Table.O != "" {
		ctx.WriteName(n.Table.O)
		ctx.WritePlain(".")
	}
	ctx.WriteName(n.Name.O)
	return nil
}

// Accept implements Node Accept interface.
func (n *ColumnName) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ColumnName)
	return v.Leave(n)
}

// String implements Stringer interface.
func (n *ColumnName) String() string {
	result := n.Name.L
	if n.Table.L != "" {
		result = n.Table.L + "." + result
	}
	if n.Schema.L != "" {
		result = n.Schema.L + "." + result
	}
	return result
}

// OrigColName returns the full original column name.
func (n *ColumnName) OrigColName() (ret string) {
	ret = n.Name.O
	if n.Table.O == "" {
		return
	}
	ret = n.Table.O + "." + ret
	if n.Schema.O == "" {
		return
	}
	ret = n.Schema.O + "." + ret
	return
}

// ColumnNameExpr represents a column name expression.
type ColumnNameExpr struct {
	exprNode

	// Name is the referenced column name.
	Name *ColumnName

	// Refer is the result field the column name refers to.
	// The value of Refer.Expr is used as the value of the expression.
	Refer *ResultField
}

// Restore implements Node interface.
func (n *ColumnNameExpr) Restore(ctx *format.RestoreCtx) error {
	if err := n.Name.Restore(ctx); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// Format the ExprNode into a Writer.
func (n *ColumnNameExpr) Format(w io.Writer) {
	name := strings.ReplaceAll(n.Name.String(), ".", "`.`")
	_, _ = fmt.Fprintf(w, "`%s`", name)
}

// Accept implements Node Accept interface.
func (n *ColumnNameExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ColumnNameExpr)
	node, ok := n.Name.Accept(v)
	if !ok {
		return n, false
	}
	n.Name = node.(*ColumnName)
	return v.Leave(n)
}

// DefaultExpr is the default expression using default value for a column.
type DefaultExpr struct {
	exprNode
	// Name is the column name.
	Name *ColumnName
}

// Restore implements Node interface.
func (n *DefaultExpr) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("DEFAULT")
	if n.Name != nil {
		ctx.WritePlain("(")
		if err := n.Name.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DefaultExpr.Name")
		}
		ctx.WritePlain(")")
	}
	return nil
}

// Format the ExprNode into a Writer.
func (n *DefaultExpr) Format(w io.Writer) {
	formatViaRestore(n, w)
}

// Accept implements Node Accept interface.
func (n *DefaultExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*DefaultExpr)
	if n.Name != nil {
		node, ok := n.Name.Accept(v)
		if !ok {
			return n, false
		}
		n.Name = node.(*ColumnName)
	}
	return v.Leave(n)
}

// ExistsSubqueryExpr is the expression for "exists (select ...)".
// See https://dev.mysql.com/doc/refman/5.7/en/exists-and-not-exists-subqueries.html
type ExistsSubqueryExpr struct {
	exprNode
	// Sel is the subquery, may be rewritten to other type of expression.
	Sel ExprNode
	// Not is true, the expression is "not exists".
	Not bool
}

// Restore implements Node interface.
func (n *ExistsSubqueryExpr) Restore(ctx *format.RestoreCtx) error {
	if n.Not {
		ctx.WriteKeyWord("NOT EXISTS ")
	} else {
		ctx.WriteKeyWord("EXISTS ")
	}
	if err := n.Sel.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore ExistsSubqueryExpr.Sel")
	}
	return nil
}

// Format the ExprNode into a Writer.
func (n *ExistsSubqueryExpr) Format(w io.Writer) {
	formatViaRestore(n, w)
}

// Accept implements Node Accept interface.
func (n *ExistsSubqueryExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ExistsSubqueryExpr)
	node, ok := n.Sel.Accept(v)
	if !ok {
		return n, false
	}
	n.Sel = node.(ExprNode)
	return v.Leave(n)
}

// PatternInExpr is the expression for in operator, like "expr in (1, 2, 3)" or "expr in (select c from t)".
type PatternInExpr struct {
	exprNode
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
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore PatternInExpr.Expr")
	}
	if n.Not {
		ctx.WriteKeyWord(" NOT IN ")
	} else {
		ctx.WriteKeyWord(" IN ")
	}
	if n.Sel != nil {
		if err := n.Sel.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore PatternInExpr.Sel")
		}
	} else {
		ctx.WritePlain("(")
		for i, expr := range n.List {
			if i != 0 {
				ctx.WritePlain(",")
			}
			if err := expr.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore PatternInExpr.List[%d]", i)
			}
		}
		ctx.WritePlain(")")
	}
	return nil
}

// Format the ExprNode into a Writer.
func (n *PatternInExpr) Format(w io.Writer) {
	n.Expr.Format(w)
	if n.Not {
		_, _ = fmt.Fprint(w, " NOT IN (")
	} else {
		_, _ = fmt.Fprint(w, " IN (")
	}
	for i, expr := range n.List {
		if i != 0 {
			_, _ = fmt.Fprint(w, ",")
		}
		expr.Format(w)
	}
	_, _ = fmt.Fprint(w, ")")
}

// Accept implements Node Accept interface.
func (n *PatternInExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*PatternInExpr)
	node, ok := n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	n.Expr = node.(ExprNode)
	for i, val := range n.List {
		node, ok = val.Accept(v)
		if !ok {
			return n, false
		}
		n.List[i] = node.(ExprNode)
	}
	if n.Sel != nil {
		node, ok = n.Sel.Accept(v)
		if !ok {
			return n, false
		}
		n.Sel = node.(ExprNode)
	}
	return v.Leave(n)
}

// IsNullExpr is the expression for null check.
type IsNullExpr struct {
	exprNode
	// Expr is the expression to be checked.
	Expr ExprNode
	// Not is true, the expression is "is not null".
	Not bool
}

// Restore implements Node interface.
func (n *IsNullExpr) Restore(ctx *format.RestoreCtx) error {
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Trace(err)
	}
	if n.Not {
		ctx.WriteKeyWord(" IS NOT NULL")
	} else {
		ctx.WriteKeyWord(" IS NULL")
	}
	return nil
}

// Format the ExprNode into a Writer.
func (n *IsNullExpr) Format(w io.Writer) {
	n.Expr.Format(w)
	if n.Not {
		_, _ = fmt.Fprint(w, " IS NOT NULL")
		return
	}
	_, _ = fmt.Fprint(w, " IS NULL")
}

// Accept implements Node Accept interface.
func (n *IsNullExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*IsNullExpr)
	node, ok := n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	n.Expr = node.(ExprNode)
	return v.Leave(n)
}

// IsTruthExpr is the expression for true/false check.
type IsTruthExpr struct {
	exprNode
	// Expr is the expression to be checked.
	Expr ExprNode
	// Not is true, the expression is "is not true/false".
	Not bool
	// True indicates checking true or false.
	True int64
}

// Restore implements Node interface.
func (n *IsTruthExpr) Restore(ctx *format.RestoreCtx) error {
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Trace(err)
	}
	if n.Not {
		ctx.WriteKeyWord(" IS NOT")
	} else {
		ctx.WriteKeyWord(" IS")
	}
	if n.True > 0 {
		ctx.WriteKeyWord(" TRUE")
	} else {
		ctx.WriteKeyWord(" FALSE")
	}
	return nil
}

// Format the ExprNode into a Writer.
func (n *IsTruthExpr) Format(w io.Writer) {
	n.Expr.Format(w)
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
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*IsTruthExpr)
	node, ok := n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	n.Expr = node.(ExprNode)
	return v.Leave(n)
}

// PatternLikeExpr is the expression for like operator, e.g, expr like "%123%"
type PatternLikeExpr struct {
	exprNode
	// Expr is the expression to be checked.
	Expr ExprNode
	// Pattern is the like expression.
	Pattern ExprNode
	// Not is true, the expression is "not like".
	Not bool

	Escape     byte
	EscapeExpr ExprNode

	PatChars []byte
	PatTypes []byte
}

// Restore implements Node interface.
func (n *PatternLikeExpr) Restore(ctx *format.RestoreCtx) error {
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore PatternLikeExpr.Expr")
	}

	if n.Not {
		ctx.WriteKeyWord(" NOT LIKE ")
	} else {
		ctx.WriteKeyWord(" LIKE ")
	}

	if err := n.Pattern.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore PatternLikeExpr.Pattern")
	}

	if n.EscapeExpr != nil {
		ctx.WriteKeyWord(" ESCAPE ")
		if err := n.EscapeExpr.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore PatternLikeExpr.EscapeExpr")
		}
	} else if escape := string(n.Escape); escape != "\\" {
		ctx.WriteKeyWord(" ESCAPE ")
		ctx.WriteString(escape)
	}
	return nil
}

// Format the ExprNode into a Writer.
func (n *PatternLikeExpr) Format(w io.Writer) {
	n.Expr.Format(w)
	if n.Not {
		_, _ = fmt.Fprint(w, " NOT LIKE ")
	} else {
		_, _ = fmt.Fprint(w, " LIKE ")
	}
	n.Pattern.Format(w)
	if n.EscapeExpr != nil {
		_, _ = fmt.Fprint(w, " ESCAPE ")
		if value, ok := n.EscapeExpr.(ValueExpr); ok {
			if text, ok := value.GetValue().(string); ok {
				_, _ = fmt.Fprintf(w, "'%s'", strings.ReplaceAll(text, "'", "''"))
			} else {
				n.EscapeExpr.Format(w)
			}
		} else {
			n.EscapeExpr.Format(w)
		}
	} else if n.Escape != '\\' {
		_, _ = fmt.Fprint(w, " ESCAPE ")
		_, _ = fmt.Fprintf(w, "'%c'", n.Escape)
	}
}

// Accept implements Node Accept interface.
func (n *PatternLikeExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*PatternLikeExpr)
	if n.Expr != nil {
		node, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		n.Expr = node.(ExprNode)
	}
	if n.Pattern != nil {
		node, ok := n.Pattern.Accept(v)
		if !ok {
			return n, false
		}
		n.Pattern = node.(ExprNode)
	}
	if n.EscapeExpr != nil {
		node, ok := n.EscapeExpr.Accept(v)
		if !ok {
			return n, false
		}
		n.EscapeExpr = node.(ExprNode)
	}
	return v.Leave(n)
}

// ParamMarkerExpr expression holds a place for another expression.
// Used in parsing prepare statement.
type ParamMarkerExpr interface {
	ValueExpr
	SetOrder(int)
}

// ParenthesesExpr is the parentheses expression.
type ParenthesesExpr struct {
	exprNode
	// Expr is the expression in parentheses.
	Expr ExprNode
}

// Restore implements Node interface.
func (n *ParenthesesExpr) Restore(ctx *format.RestoreCtx) error {
	ctx.WritePlain("(")
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred when restore ParenthesesExpr.Expr")
	}
	ctx.WritePlain(")")
	return nil
}

// Format the ExprNode into a Writer.
func (n *ParenthesesExpr) Format(w io.Writer) {
	_, _ = fmt.Fprint(w, "(")
	n.Expr.Format(w)
	_, _ = fmt.Fprint(w, ")")
}

// Accept implements Node Accept interface.
func (n *ParenthesesExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ParenthesesExpr)
	if n.Expr != nil {
		node, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		n.Expr = node.(ExprNode)
	}
	return v.Leave(n)
}

// PositionExpr is the expression for order by and group by position.
// MySQL use position expression started from 1, it looks a little confused inner.
// maybe later we will use 0 at first.
type PositionExpr struct {
	exprNode
	// N is the position, started from 1 now.
	N int
	// P is the parameterized position.
	P ExprNode
	// Refer is the result field the position refers to.
	Refer *ResultField
}

// Restore implements Node interface.
func (n *PositionExpr) Restore(ctx *format.RestoreCtx) error {
	ctx.WritePlainf("%d", n.N)
	return nil
}

// Format the ExprNode into a Writer.
func (n *PositionExpr) Format(w io.Writer) {
	formatViaRestore(n, w)
}

// Accept implements Node Accept interface.
func (n *PositionExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*PositionExpr)
	if n.P != nil {
		node, ok := n.P.Accept(v)
		if !ok {
			return n, false
		}
		n.P = node.(ExprNode)
	}
	return v.Leave(n)
}

// PatternRegexpExpr is the pattern expression for pattern match.
type PatternRegexpExpr struct {
	exprNode
	// Expr is the expression to be checked.
	Expr ExprNode
	// Pattern is the expression for pattern.
	Pattern ExprNode
	// Not is true, the expression is "not rlike",
	Not bool

	// Re is the compiled regexp.
	Re *regexp.Regexp
	// Sexpr is the string for Expr expression.
	Sexpr *string
}

// Restore implements Node interface.
func (n *PatternRegexpExpr) Restore(ctx *format.RestoreCtx) error {
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore PatternRegexpExpr.Expr")
	}

	if n.Not {
		ctx.WriteKeyWord(" NOT REGEXP ")
	} else {
		ctx.WriteKeyWord(" REGEXP ")
	}

	if err := n.Pattern.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore PatternRegexpExpr.Pattern")
	}

	return nil
}

// Format the ExprNode into a Writer.
func (n *PatternRegexpExpr) Format(w io.Writer) {
	n.Expr.Format(w)
	if n.Not {
		_, _ = fmt.Fprint(w, " NOT REGEXP ")
	} else {
		_, _ = fmt.Fprint(w, " REGEXP ")
	}
	n.Pattern.Format(w)
}

// Accept implements Node Accept interface.
func (n *PatternRegexpExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*PatternRegexpExpr)
	node, ok := n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	n.Expr = node.(ExprNode)
	node, ok = n.Pattern.Accept(v)
	if !ok {
		return n, false
	}
	n.Pattern = node.(ExprNode)
	return v.Leave(n)
}

// RowExpr is the expression for row constructor.
// See https://dev.mysql.com/doc/refman/5.7/en/row-subqueries.html
// MemberOfExpr is the JSON MEMBER OF predicate.
// See https://dev.mysql.com/doc/refman/8.0/en/json-search-functions.html#operator_member-of
type MemberOfExpr struct {
	exprNode
	// Value is the value to search for.
	Value ExprNode
	// Array is the JSON array expression.
	Array ExprNode
}

// Restore implements Node interface.
func (n *MemberOfExpr) Restore(ctx *format.RestoreCtx) error {
	if err := n.Value.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore MemberOfExpr.Value")
	}
	ctx.WriteKeyWord(" MEMBER OF (")
	if err := n.Array.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore MemberOfExpr.Array")
	}
	ctx.WritePlain(")")
	return nil
}

// Format the ExprNode into a Writer.
func (n *MemberOfExpr) Format(w io.Writer) {
	n.Value.Format(w)
	_, _ = fmt.Fprint(w, " MEMBER OF (")
	n.Array.Format(w)
	_, _ = fmt.Fprint(w, ")")
}

// Accept implements Node Accept interface.
func (n *MemberOfExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*MemberOfExpr)
	node, ok := n.Value.Accept(v)
	if !ok {
		return n, false
	}
	n.Value = node.(ExprNode)
	node, ok = n.Array.Accept(v)
	if !ok {
		return n, false
	}
	n.Array = node.(ExprNode)
	return v.Leave(n)
}

// IsJSONExpr is the IS [NOT] JSON predicate.
// See https://mariadb.com/docs/server/reference/sql-structure/operators/comparison-operators/is-json
type IsJSONExpr struct {
	exprNode
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
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore IsJSONExpr.Expr")
	}
	if n.Not {
		ctx.WriteKeyWord(" IS NOT JSON")
	} else {
		ctx.WriteKeyWord(" IS JSON")
	}
	if n.Type != "" {
		ctx.WritePlain(" ")
		ctx.WriteKeyWord(n.Type)
	}
	if n.UniqueKeys != "" {
		ctx.WritePlain(" ")
		ctx.WriteKeyWord(n.UniqueKeys)
	}
	return nil
}

// Format the ExprNode into a Writer.
func (n *IsJSONExpr) Format(w io.Writer) {
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, w)
	n.Expr.Format(w)
	if n.Not {
		ctx.WriteKeyWord(" IS NOT JSON")
	} else {
		ctx.WriteKeyWord(" IS JSON")
	}
	if n.Type != "" {
		ctx.WriteKeyWord(" " + n.Type)
	}
	if n.UniqueKeys != "" {
		ctx.WriteKeyWord(" " + n.UniqueKeys)
	}
}

// Accept implements Node Accept interface.
func (n *IsJSONExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*IsJSONExpr)
	node, ok := n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	n.Expr = node.(ExprNode)
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
	exprNode

	Columns        []*ColumnName
	Expr           ExprNode
	Modifier       MatchAgainstModifier
	QueryExpansion bool
}

// Restore implements Node interface.
func (n *MatchAgainst) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("MATCH (")
	for i, col := range n.Columns {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := col.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore MatchAgainst.Columns[%d]", i)
		}
	}
	ctx.WritePlain(") ")
	ctx.WriteKeyWord("AGAINST (")
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore MatchAgainst.Expr")
	}
	if n.Modifier == MatchAgainstBoolean {
		ctx.WriteKeyWord(" IN BOOLEAN MODE")
	} else if n.QueryExpansion {
		ctx.WriteKeyWord(" WITH QUERY EXPANSION")
	}
	ctx.WritePlain(")")
	return nil
}

// Format the ExprNode into a Writer.
func (n *MatchAgainst) Format(w io.Writer) {
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, w)
	ctx.WritePlain("MATCH (")
	for i, col := range n.Columns {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if col.Schema.O != "" {
			ctx.WritePlain(col.Schema.O + ".")
		}
		if col.Table.O != "" {
			ctx.WritePlain(col.Table.O + ".")
		}
		ctx.WritePlain(col.Name.O)
	}
	ctx.WritePlain(") AGAINST (")
	n.Expr.Format(w)
	if n.Modifier == MatchAgainstBoolean {
		ctx.WriteKeyWord(" IN BOOLEAN MODE")
	} else if n.QueryExpansion {
		ctx.WriteKeyWord(" WITH QUERY EXPANSION")
	}
	ctx.WritePlain(")")
}

// Accept implements Node Accept interface.
func (n *MatchAgainst) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*MatchAgainst)
	for i, col := range n.Columns {
		node, ok := col.Accept(v)
		if !ok {
			return n, false
		}
		n.Columns[i] = node.(*ColumnName)
	}
	node, ok := n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	n.Expr = node.(ExprNode)
	return v.Leave(n)
}

// SoundsLikeExpr is the expr SOUNDS LIKE expr predicate.
// See https://dev.mysql.com/doc/refman/8.0/en/string-functions.html#operator_sounds-like
type SoundsLikeExpr struct {
	exprNode

	Expr    ExprNode
	Pattern ExprNode
}

// Restore implements Node interface.
func (n *SoundsLikeExpr) Restore(ctx *format.RestoreCtx) error {
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore SoundsLikeExpr.Expr")
	}
	ctx.WriteKeyWord(" SOUNDS LIKE ")
	if err := n.Pattern.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore SoundsLikeExpr.Pattern")
	}
	return nil
}

// Format the ExprNode into a Writer.
func (n *SoundsLikeExpr) Format(w io.Writer) {
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, w)
	n.Expr.Format(w)
	ctx.WriteKeyWord(" SOUNDS LIKE ")
	n.Pattern.Format(w)
}

// Accept implements Node Accept interface.
func (n *SoundsLikeExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*SoundsLikeExpr)
	node, ok := n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	n.Expr = node.(ExprNode)
	node, ok = n.Pattern.Accept(v)
	if !ok {
		return n, false
	}
	n.Pattern = node.(ExprNode)
	return v.Leave(n)
}

type RowExpr struct {
	exprNode

	Values []ExprNode
	// OmitRowKeyword preserves MariaDB's `VALUES (expr, ...)` row syntax.
	OmitRowKeyword bool
}

// Restore implements Node interface.
func (n *RowExpr) Restore(ctx *format.RestoreCtx) error {
	if !n.OmitRowKeyword {
		ctx.WriteKeyWord("ROW")
	}
	ctx.WritePlain("(")
	for i, v := range n.Values {
		if i != 0 {
			ctx.WritePlain(",")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred when restore RowExpr.Values[%v]", i)
		}
	}
	ctx.WritePlain(")")
	return nil
}

// Format the ExprNode into a Writer.
func (n *RowExpr) Format(w io.Writer) {
	formatViaRestore(n, w)
}

// Accept implements Node Accept interface.
func (n *RowExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*RowExpr)
	for i, val := range n.Values {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Values[i] = node.(ExprNode)
	}
	return v.Leave(n)
}

// UnaryOperationExpr is the expression for unary operator.
type UnaryOperationExpr struct {
	exprNode
	// Op is the operator opcode.
	Op opcode.Op
	// V is the unary expression.
	V ExprNode
}

// Restore implements Node interface.
func (n *UnaryOperationExpr) Restore(ctx *format.RestoreCtx) error {
	if err := n.Op.Restore(ctx); err != nil {
		return errors.Trace(err)
	}
	if err := n.V.Restore(ctx); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// Format the ExprNode into a Writer.
func (n *UnaryOperationExpr) Format(w io.Writer) {
	n.Op.Format(w)
	n.V.Format(w)
}

// Accept implements Node Accept interface.
func (n *UnaryOperationExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*UnaryOperationExpr)
	node, ok := n.V.Accept(v)
	if !ok {
		return n, false
	}
	n.V = node.(ExprNode)
	return v.Leave(n)
}

// ValuesExpr is the expression used in INSERT VALUES.
type ValuesExpr struct {
	exprNode
	// Column is column name.
	Column *ColumnNameExpr
	// Singular preserves MariaDB's VALUE(column) spelling.
	Singular bool
}

// Restore implements Node interface.
func (n *ValuesExpr) Restore(ctx *format.RestoreCtx) error {
	if n.Singular {
		ctx.WriteKeyWord("VALUE")
	} else {
		ctx.WriteKeyWord("VALUES")
	}
	ctx.WritePlain("(")
	if err := n.Column.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore ValuesExpr.Column")
	}
	ctx.WritePlain(")")

	return nil
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
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ValuesExpr)
	node, ok := n.Column.Accept(v)
	if !ok {
		return n, false
	}
	// `node` may be *ast.ValueExpr, so do the type assertion without
	// storing `ok`.
	n.Column, _ = node.(*ColumnNameExpr)
	return v.Leave(n)
}

// VariableExpr is the expression for variable.
type VariableExpr struct {
	exprNode
	// Name is the variable name.
	Name string
	// IsGlobal indicates whether this variable is global.
	IsGlobal bool
	// IsPersist indicates whether this variable is persisted.
	IsPersist bool
	// IsPersistOnly indicates whether this variable is persisted only.
	IsPersistOnly bool
	// IsSystem indicates whether this variable is a system variable in current session.
	IsSystem bool
	// ExplicitScope indicates whether this variable scope is set explicitly.
	ExplicitScope bool
	// Value is the variable value.
	Value ExprNode
}

// Restore implements Node interface.
func (n *VariableExpr) Restore(ctx *format.RestoreCtx) error {
	if n.IsSystem {
		ctx.WritePlain("@@")
		if n.ExplicitScope {
			switch {
			case n.IsPersistOnly:
				ctx.WriteKeyWord("PERSIST_ONLY")
			case n.IsPersist:
				ctx.WriteKeyWord("PERSIST")
			case n.IsGlobal:
				ctx.WriteKeyWord("GLOBAL")
			default:
				ctx.WriteKeyWord("SESSION")
			}
			ctx.WritePlain(".")
		}
	} else {
		ctx.WritePlain("@")
	}
	ctx.WriteName(n.Name)

	if n.Value != nil {
		ctx.WritePlain(":=")
		if err := n.Value.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore VariableExpr.Value")
		}
	}

	return nil
}

// Format the ExprNode into a Writer.
func (n *VariableExpr) Format(w io.Writer) {
	formatViaRestore(n, w)
}

// Accept implements Node Accept interface.
func (n *VariableExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*VariableExpr)
	if n.Value == nil {
		return v.Leave(n)
	}

	node, ok := n.Value.Accept(v)
	if !ok {
		return n, false
	}
	n.Value = node.(ExprNode)
	return v.Leave(n)
}

// MaxValueExpr is the expression for "maxvalue" used in partition.
type MaxValueExpr struct {
	exprNode
}

// Restore implements Node interface.
func (n *MaxValueExpr) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("MAXVALUE")
	return nil
}

// Format the ExprNode into a Writer.
func (n *MaxValueExpr) Format(w io.Writer) {
	_, _ = fmt.Fprint(w, "MAXVALUE")
}

// Accept implements Node Accept interface.
func (n *MaxValueExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(n)
}

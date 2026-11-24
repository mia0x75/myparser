package ast

import (
	"fmt"

	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
)

// LabeledStmt is implemented by compound statements that may carry a label.
type LabeledStmt interface {
	StmtNode
	// SetLabel sets the label of the statement.
	SetLabel(label lex.Ident)
}

func restoreStmtList(ctx *format.RestoreCtx, list []StmtNode) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for i, s := range list {
		if !nodePresent(s) {
			return errors.Errorf("restore statement list item %d: nil statement", i)
		}
		if err := s.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore statement list")
		}
		ctx.WritePlain(";")
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return ctx.Err()
}

func acceptStmtList(v Visitor, list []StmtNode) ([]StmtNode, bool) {
	for i, s := range list {
		if !nodePresent(s) {
			return list, false
		}
		node, ok := s.Accept(v)
		if !ok {
			return list, false
		}
		stmt, ok := node.(StmtNode)
		if !ok || !nodePresent(stmt) {
			return list, false
		}
		list[i] = stmt
	}
	return list, true
}

// BeginEndStmt is a compound statement: `[label:] BEGIN ... END [label]`.
// See https://dev.mysql.com/doc/refman/8.0/en/begin-end.html
type BeginEndStmt struct {
	stmtNode

	Label     lex.Ident
	NotAtomic bool
	List      []StmtNode
}

// SetLabel implements LabeledStmt interface.
func (n *BeginEndStmt) SetLabel(label lex.Ident) { n.Label = label }

// Restore implements Node interface.
func (n *BeginEndStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore BeginEndStmt: nil statement")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if n.Label.Present() {
		restoreStoredIdent(ctx, n.Label)
		ctx.WritePlain(": ")
	}
	ctx.WriteKeyword("BEGIN")
	if n.NotAtomic {
		ctx.WriteKeyword(" NOT ATOMIC")
	}
	if len(n.List) > 0 {
		ctx.WritePlain("\n")
		for _, s := range n.List {
			if !nodePresent(s) {
				return errors.New("restore begin-end statement: nil statement in list")
			}
			ctx.WritePlain("  ")
			if err := s.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore BeginEndStmt.List")
			}
			ctx.WritePlain(";")
			ctx.WritePlain("\n")
		}
	} else {
		ctx.WritePlain(" ")
	}
	ctx.WriteKeyword("END")
	if n.Label.Present() {
		ctx.WritePlain(" ")
		ctx.WriteIdent(n.Label)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *BeginEndStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*BeginEndStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	list, ok := acceptStmtList(v, n.List)
	if !ok {
		return n, false
	}
	n.List = list
	return v.Leave(n)
}

// DeclareStmt is a variable declaration: `DECLARE var [, var]... type [DEFAULT expr]`.
// See https://dev.mysql.com/doc/refman/8.0/en/declare-local-variable.html
type DeclareStmt struct {
	stmtNode

	Names   []lex.Ident
	Type    *TypeName
	RowType []*ColumnDef
	// TypeRef is the reference in a MariaDB TYPE OF or ROW TYPE OF declaration.
	TypeRef    *QualifiedName
	RowTypeRef bool
	Default    ExprNode
}

// Restore implements Node interface.
func (n *DeclareStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore DeclareStmt: nil statement")
	}
	ctx.WriteKeyword("DECLARE ")
	for i, name := range n.Names {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		ctx.WriteIdent(name)
	}
	ctx.WritePlain(" ")
	if n.TypeRef != nil {
		if n.RowTypeRef {
			ctx.WriteKeyword("ROW ")
		}
		ctx.WriteKeyword("TYPE OF ")
		if err := restoreQualifiedName(ctx, *n.TypeRef); err != nil {
			return errors.Annotate(err, "An error occurred while restore DeclareStmt.TypeRef")
		}
	} else if len(n.RowType) > 0 {
		ctx.WriteKeyword("ROW")
		ctx.WritePlain("(")
		for i, field := range n.RowType {
			if field == nil {
				return errors.Errorf("restore DeclareStmt.RowType[%d]: missing field", i)
			}
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := field.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore DeclareStmt.RowType[%d]", i)
			}
		}
		ctx.WritePlain(")")
	} else if n.Type != nil {
		if err := n.Type.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DeclareStmt.Type")
		}
	}
	if n.Default != nil {
		if !nodePresent(n.Default) {
			return errors.New("restore DeclareStmt: nil default")
		}
		ctx.WriteKeyword(" DEFAULT ")
		if err := n.Default.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DeclareStmt.Default")
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DeclareStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*DeclareStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	if n.Type != nil {
		updated, ok := acceptNode[*TypeName](v, n.Type)
		if !ok {
			return n, false
		}
		n.Type = updated
	}
	for i, field := range n.RowType {
		if !nodePresent(field) {
			return n, false
		}
		updated, ok := acceptNode[*ColumnDef](v, field)
		if !ok {
			return n, false
		}
		n.RowType[i] = updated
	}
	if n.Default != nil {
		updated, ok := acceptNode[ExprNode](v, n.Default)
		if !ok {
			return n, false
		}
		n.Default = updated
	}
	return v.Leave(n)
}

// HandlerConditionType is the type of a condition used by a handler or SIGNAL.
type HandlerConditionType int

// The types of handler conditions.
const (
	HandlerConditionSQLState HandlerConditionType = iota
	HandlerConditionErrorCode
	HandlerConditionName
	HandlerConditionSQLWarning
	HandlerConditionNotFound
	HandlerConditionSQLException
)

// HandlerCondition is a condition value used by a handler, SIGNAL or RESIGNAL.
type HandlerCondition struct {
	Type HandlerConditionType
	// Value is the condition value, either an errno or a condition name.
	Value lex.Ident
	// SQLState is the `SQLSTATE VALUE '...'` value.
	SQLState string
}

// Restore implements Node interface.
func (c *HandlerCondition) Restore(ctx *format.RestoreCtx) error {
	if c == nil {
		return errors.New("restore HandlerCondition: nil condition")
	}
	switch c.Type {
	case HandlerConditionSQLState:
		ctx.WriteKeyword("SQLSTATE VALUE ")
		ctx.WriteString(c.SQLState)
	case HandlerConditionErrorCode:
		ctx.WritePlain(c.Value.Name)
	case HandlerConditionName:
		ctx.WriteIdent(c.Value)
	case HandlerConditionSQLWarning:
		ctx.WriteKeyword("SQLWARNING")
	case HandlerConditionNotFound:
		ctx.WriteKeyword("NOT FOUND")
	case HandlerConditionSQLException:
		ctx.WriteKeyword("SQLEXCEPTION")
	}
	return ctx.Err()
}

// NewErrnoHandlerCondition builds a handler condition from a MySQL error code.
func NewErrnoHandlerCondition(v any) *HandlerCondition {
	return &HandlerCondition{Type: HandlerConditionErrorCode, Value: lex.Ident{Name: fmt.Sprintf("%v", v)}}
}

// DeclareConditionStmt is a condition declaration:
// `DECLARE name CONDITION FOR condition_value`.
type DeclareConditionStmt struct {
	stmtNode

	Name      lex.Ident
	Condition *HandlerCondition
}

// Restore implements Node interface.
func (n *DeclareConditionStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("DeclareConditionStmt")
	}
	if n == nil || n.Condition == nil {
		return errors.New("restore declare condition statement: missing condition")
	}
	ctx.WriteKeyword("DECLARE ")
	ctx.WriteIdent(n.Name)
	ctx.WriteKeyword(" CONDITION FOR ")
	return n.Condition.Restore(ctx)
}

// Accept implements Node Accept interface.
func (n *DeclareConditionStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*DeclareConditionStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	return v.Leave(stmt)
}

// DeclareCursorStmt is a cursor declaration: `DECLARE name CURSOR FOR select`.
type DeclareCursorStmt struct {
	stmtNode

	Name   lex.Ident
	Select QueryNode
}

// Restore implements Node interface.
func (n *DeclareCursorStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Select) {
		return errors.New("restore DeclareCursorStmt: missing query")
	}
	ctx.WriteKeyword("DECLARE ")
	ctx.WriteIdent(n.Name)
	ctx.WriteKeyword(" CURSOR FOR ")
	if n.Select == nil {
		return errors.New("An error occurred while restore DeclareCursorStmt.Select: nil query")
	}
	return n.Select.Restore(ctx)
}

// Accept implements Node Accept interface.
func (n *DeclareCursorStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*DeclareCursorStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	if n.Select != nil {
		updated, ok := acceptNode[QueryNode](v, n.Select)
		if !ok {
			return n, false
		}
		n.Select = updated
	}
	return v.Leave(n)
}

// HandlerType is the action of a handler declaration.
type HandlerType int

// The handler actions.
const (
	HandlerContinue HandlerType = iota
	HandlerExit
	HandlerUndo
)

// HandlerTypeString returns the SQL keyword of the handler action.
func HandlerTypeString(t HandlerType) string {
	switch t {
	case HandlerExit:
		return "EXIT"
	case HandlerUndo:
		return "UNDO"
	default:
		return "CONTINUE"
	}
}

// DeclareHandlerStmt is a handler declaration:
// `DECLARE {CONTINUE|EXIT|UNDO} HANDLER FOR condition [, condition]... stmt`.
type DeclareHandlerStmt struct {
	stmtNode

	Action     HandlerType
	Conditions []*HandlerCondition
	Stmt       StmtNode
}

// Restore implements Node interface.
func (n *DeclareHandlerStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("DeclareHandlerStmt")
	}
	if n == nil || len(n.Conditions) == 0 {
		return errors.New("restore declare handler statement: missing conditions")
	}
	if !nodePresent(n.Stmt) {
		return errors.New("restore declare handler statement: missing statement")
	}
	ctx.WriteKeyword("DECLARE ")
	ctx.WriteKeyword(HandlerTypeString(n.Action))
	ctx.WriteKeyword(" HANDLER FOR ")
	for i, c := range n.Conditions {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if c == nil {
			return errors.Errorf("restore declare handler statement: nil condition %d", i)
		}
		if err := c.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DeclareHandlerStmt.Conditions")
		}
	}
	ctx.WritePlain(" ")
	return n.Stmt.Restore(ctx)
}

// Accept implements Node Accept interface.
func (n *DeclareHandlerStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*DeclareHandlerStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	if !nodePresent(n.Stmt) {
		return n, false
	}
	updated, ok := acceptNode[StmtNode](v, n.Stmt)
	if !ok {
		return n, false
	}
	n.Stmt = updated
	return v.Leave(n)
}

// IfClause is a single branch of an IfStmt.
type IfClause struct {
	node
	Condition ExprNode
	List      []StmtNode
	ElseIf    bool
}

var _ Node = (*IfClause)(nil)

func (n *IfClause) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("IfClause")
	}
	if n == nil || !nodePresent(n.Condition) {
		return errors.New("restore if clause: missing condition")
	}
	if n.ElseIf {
		ctx.WriteKeyword("ELSEIF ")
	} else {
		ctx.WriteKeyword("IF ")
	}
	if err := n.Condition.Restore(ctx); err != nil {
		return errors.Annotate(err, "restore if clause condition")
	}
	ctx.WriteKeyword(" THEN ")
	if err := restoreStmtList(ctx, n.List); err != nil {
		return err
	}
	return ctx.Err()
}

func (n *IfClause) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	clause, ok := newNode.(*IfClause)
	if !ok || clause == nil {
		return newNode, false
	}
	n = clause
	if skipChildren {
		return v.Leave(n)
	}
	if !nodePresent(n.Condition) {
		return n, false
	}
	child, ok := n.Condition.Accept(v)
	if !ok {
		return n, false
	}
	expr, ok := child.(ExprNode)
	if !ok || !nodePresent(expr) {
		return n, false
	}
	n.Condition = expr
	list, ok := acceptStmtList(v, n.List)
	if !ok {
		return n, false
	}
	n.List = list
	return v.Leave(n)
}

// IfStmt is an `IF ... THEN ... [ELSEIF ...] [ELSE ...] END IF` statement.
// See https://dev.mysql.com/doc/refman/8.0/en/if.html
type IfStmt struct {
	stmtNode

	Conditions []*IfClause
	ElseList   []StmtNode
}

// Restore implements Node interface.
func (n *IfStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("IfStmt")
	}
	if n == nil || len(n.Conditions) == 0 {
		return errors.New("restore if statement: missing branches")
	}
	if n.Conditions[0] == nil || !nodePresent(n.Conditions[0].Condition) {
		return errors.New("restore if statement: missing first condition")
	}
	for i, clause := range n.Conditions {
		if clause == nil {
			return errors.New("restore if statement: nil branch")
		}
		if i > 0 {
			ctx.WritePlain(" ")
		}
		if err := clause.Restore(ctx); err != nil {
			return errors.Annotate(err, "restore if statement branch")
		}
	}
	if n.ElseList != nil {
		ctx.WriteKeyword(" ELSE ")
		if err := restoreStmtList(ctx, n.ElseList); err != nil {
			return err
		}
	}
	ctx.WriteKeyword(" END IF")
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *IfStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := newNode.(*IfStmt)
	if !ok || stmt == nil {
		return newNode, false
	}
	n = stmt
	if skipChildren {
		return v.Leave(n)
	}
	if len(n.Conditions) == 0 {
		return n, false
	}
	for i, c := range n.Conditions {
		if c == nil {
			return n, false
		}
		node, ok := c.Accept(v)
		if !ok {
			return n, false
		}
		clause, ok := node.(*IfClause)
		if !ok || clause == nil {
			return n, false
		}
		n.Conditions[i] = clause
	}
	if n.ElseList != nil {
		list, ok := acceptStmtList(v, n.ElseList)
		if !ok {
			return n, false
		}
		n.ElseList = list
	}
	return v.Leave(n)
}

type CaseWhenClause struct {
	node
	Expr ExprNode
	List []StmtNode
}

var _ Node = (*CaseWhenClause)(nil)

func (n *CaseWhenClause) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("CaseWhenClause")
	}
	if n == nil || !nodePresent(n.Expr) {
		return errors.New("restore case when clause: missing expression")
	}
	ctx.WriteKeyword("WHEN ")
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "restore case when clause expression")
	}
	ctx.WriteKeyword(" THEN ")
	return restoreStmtList(ctx, n.List)
}

func (n *CaseWhenClause) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	clause, ok := newNode.(*CaseWhenClause)
	if !ok || clause == nil {
		return newNode, false
	}
	n = clause
	if skipChildren {
		return v.Leave(n)
	}
	if !nodePresent(n.Expr) {
		return n, false
	}
	child, ok := n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	expr, ok := child.(ExprNode)
	if !ok || !nodePresent(expr) {
		return n, false
	}
	n.Expr = expr
	list, ok := acceptStmtList(v, n.List)
	if !ok {
		return n, false
	}
	n.List = list
	return v.Leave(n)
}

// CaseStmt is a `CASE ... WHEN ... THEN ... [ELSE ...] END CASE` statement.
// See https://dev.mysql.com/doc/refman/8.0/en/case.html
type CaseStmt struct {
	stmtNode

	Value    ExprNode
	Whens    []*CaseWhenClause
	ElseList []StmtNode
}

// Restore implements Node interface.
func (n *CaseStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("CaseStmt")
	}
	if n == nil || len(n.Whens) == 0 {
		return errors.New("restore case statement: missing branches")
	}
	ctx.WriteKeyword("CASE")
	if n.Value != nil {
		if !nodePresent(n.Value) {
			return errors.New("restore case statement: nil value")
		}
		ctx.WritePlain(" ")
		if err := n.Value.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore CaseStmt.Value")
		}
	}
	for _, w := range n.Whens {
		if w == nil {
			return errors.New("restore case statement: nil branch")
		}
		ctx.WritePlain(" ")
		if err := w.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore CaseStmt.Whens")
		}
	}
	if n.ElseList != nil {
		ctx.WriteKeyword(" ELSE ")
		if err := restoreStmtList(ctx, n.ElseList); err != nil {
			return err
		}
	}
	ctx.WriteKeyword(" END CASE")
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CaseStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := newNode.(*CaseStmt)
	if !ok || stmt == nil {
		return newNode, false
	}
	n = stmt
	if skipChildren {
		return v.Leave(n)
	}
	if len(n.Whens) == 0 {
		return n, false
	}
	if n.Value != nil {
		if !nodePresent(n.Value) {
			return n, false
		}
		node, ok := n.Value.Accept(v)
		if !ok {
			return n, false
		}
		expr, ok := node.(ExprNode)
		if !ok || !nodePresent(expr) {
			return n, false
		}
		n.Value = expr
	}
	for i, w := range n.Whens {
		if w == nil {
			return n, false
		}
		node, ok := w.Accept(v)
		if !ok {
			return n, false
		}
		clause, ok := node.(*CaseWhenClause)
		if !ok || clause == nil {
			return n, false
		}
		n.Whens[i] = clause
	}
	if n.ElseList != nil {
		list, ok := acceptStmtList(v, n.ElseList)
		if !ok {
			return n, false
		}
		n.ElseList = list
	}
	return v.Leave(n)
}

// WhileStmt is a `[label:] WHILE cond DO ... END WHILE [label]` statement.
// See https://dev.mysql.com/doc/refman/8.0/en/while.html
type WhileStmt struct {
	stmtNode

	Label     lex.Ident
	Condition ExprNode
	List      []StmtNode
}

// SetLabel implements LabeledStmt interface.
func (n *WhileStmt) SetLabel(label lex.Ident) { n.Label = label }

// Restore implements Node interface.
func (n *WhileStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("WhileStmt")
	}
	if n == nil || !nodePresent(n.Condition) {
		return errors.New("restore while statement: missing condition")
	}
	if n.Label.Present() {
		restoreStoredIdent(ctx, n.Label)
		ctx.WritePlain(": ")
	}
	ctx.WriteKeyword("WHILE ")
	if err := n.Condition.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore WhileStmt.Condition")
	}
	ctx.WriteKeyword(" DO ")
	if err := restoreStmtList(ctx, n.List); err != nil {
		return err
	}
	ctx.WriteKeyword(" END WHILE")
	if n.Label.Present() {
		ctx.WritePlain(" ")
		restoreStoredIdent(ctx, n.Label)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *WhileStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*WhileStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	if !nodePresent(n.Condition) {
		return n, false
	}
	updated, ok := acceptNode[ExprNode](v, n.Condition)
	if !ok {
		return n, false
	}
	n.Condition = updated
	list, ok := acceptStmtList(v, n.List)
	if !ok {
		return n, false
	}
	n.List = list
	return v.Leave(n)
}

// RepeatStmt is a `[label:] REPEAT ... UNTIL cond END REPEAT [label]` statement.
// See https://dev.mysql.com/doc/refman/8.0/en/repeat.html
type RepeatStmt struct {
	stmtNode

	Label     lex.Ident
	Condition ExprNode
	List      []StmtNode
}

// SetLabel implements LabeledStmt interface.
func (n *RepeatStmt) SetLabel(label lex.Ident) { n.Label = label }

// Restore implements Node interface.
func (n *RepeatStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("RepeatStmt")
	}
	if n == nil || !nodePresent(n.Condition) {
		return errors.New("restore repeat statement: missing condition")
	}
	if n.Label.Present() {
		restoreStoredIdent(ctx, n.Label)
		ctx.WritePlain(": ")
	}
	ctx.WriteKeyword("REPEAT ")
	if err := restoreStmtList(ctx, n.List); err != nil {
		return err
	}
	ctx.WriteKeyword(" UNTIL ")
	if err := n.Condition.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore RepeatStmt.Condition")
	}
	ctx.WriteKeyword(" END REPEAT")
	if n.Label.Present() {
		ctx.WritePlain(" ")
		ctx.WriteIdent(n.Label)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *RepeatStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*RepeatStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	list, ok := acceptStmtList(v, n.List)
	if !ok {
		return n, false
	}
	n.List = list
	updated, ok := acceptNode[ExprNode](v, n.Condition)
	if !ok {
		return n, false
	}
	n.Condition = updated
	return v.Leave(n)
}

// LoopStmt is a `[label:] LOOP ... END LOOP [label]` statement.
// See https://dev.mysql.com/doc/refman/8.0/en/loop.html
type LoopStmt struct {
	stmtNode

	Label lex.Ident
	List  []StmtNode
}

// SetLabel implements LabeledStmt interface.
func (n *LoopStmt) SetLabel(label lex.Ident) { n.Label = label }

// Restore implements Node interface.
func (n *LoopStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore LoopStmt: nil statement")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if n.Label.Present() {
		ctx.WriteIdent(n.Label)
		ctx.WritePlain(": ")
	}
	ctx.WriteKeyword("LOOP ")
	if err := restoreStmtList(ctx, n.List); err != nil {
		return err
	}
	ctx.WriteKeyword(" END LOOP")
	if n.Label.Present() {
		ctx.WritePlain(" ")
		ctx.WriteIdent(n.Label)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *LoopStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*LoopStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	list, ok := acceptStmtList(v, n.List)
	if !ok {
		return n, false
	}
	n.List = list
	return v.Leave(n)
}

// ForStmt is a MariaDB `[label:] FOR var IN ... DO ... END FOR [label]`
// statement. A loop iterates either over an expression range or over the rows
// produced by Query.
type ForStmt struct {
	stmtNode

	Label      lex.Ident
	Variable   lex.Ident
	LowerBound ExprNode
	UpperBound ExprNode
	Query      QueryNode
	List       []StmtNode
}

// SetLabel implements LabeledStmt interface.
func (n *ForStmt) SetLabel(label lex.Ident) { n.Label = label }

// Restore implements Node interface.
func (n *ForStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore ForStmt: nil statement")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if n.Query == nil &&
		(!nodePresent(n.LowerBound) || !nodePresent(n.UpperBound)) {
		return errors.New("restore for statement: missing range bounds")
	}
	if n.Label.Present() {
		ctx.WriteIdent(n.Label)
		ctx.WritePlain(": ")
	}
	ctx.WriteKeyword("FOR ")
	ctx.WriteIdent(n.Variable)
	ctx.WriteKeyword(" IN ")
	if n.Query != nil {
		if !nodePresent(n.Query) {
			return errors.New("restore ForStmt: nil query")
		}
		ctx.WritePlain("(")
		if err := n.Query.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore ForStmt.Query")
		}
		ctx.WritePlain(")")
	} else {
		if err := n.LowerBound.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore ForStmt.LowerBound")
		}
		ctx.WritePlain("..")
		if err := n.UpperBound.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore ForStmt.UpperBound")
		}
	}
	ctx.WriteKeyword(" DO ")
	if err := restoreStmtList(ctx, n.List); err != nil {
		return err
	}
	ctx.WriteKeyword(" END FOR")
	if n.Label.Present() {
		ctx.WritePlain(" ")
		ctx.WriteIdent(n.Label)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ForStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*ForStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	if n.Query != nil {
		updated, ok := acceptNode[QueryNode](v, n.Query)
		if !ok {
			return n, false
		}
		n.Query = updated
	} else {
		for _, bound := range []*ExprNode{&n.LowerBound, &n.UpperBound} {
			if *bound == nil {
				continue
			}
			updated, ok := acceptNode[ExprNode](v, *bound)
			if !ok {
				return n, false
			}
			*bound = updated
		}
	}
	list, ok := acceptStmtList(v, n.List)
	if !ok {
		return n, false
	}
	n.List = list
	return v.Leave(n)
}

// LeaveStmt is a `LEAVE label` statement.
// See https://dev.mysql.com/doc/refman/8.0/en/leave.html
type LeaveStmt struct {
	stmtNode

	Label lex.Ident
}

// Restore implements Node interface.
func (n *LeaveStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore LeaveStmt: nil statement")
	}
	ctx.WriteKeyword("LEAVE ")
	if n.Label.Present() {
		restoreStoredIdent(ctx, n.Label)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *LeaveStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*LeaveStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	return v.Leave(stmt)
}

// IterateStmt is an `ITERATE label` statement.
// See https://dev.mysql.com/doc/refman/8.0/en/iterate.html
type IterateStmt struct {
	stmtNode

	Label lex.Ident
}

// Restore implements Node interface.
func (n *IterateStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore IterateStmt: nil statement")
	}
	ctx.WriteKeyword("ITERATE ")
	if n.Label.Present() {
		restoreStoredIdent(ctx, n.Label)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *IterateStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*IterateStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	return v.Leave(stmt)
}

// OpenStmt is an `OPEN cursor_name` statement.
// See https://dev.mysql.com/doc/refman/8.0/en/open.html
type OpenStmt struct {
	stmtNode

	Name lex.Ident
}

// Restore implements Node interface.
func (n *OpenStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore OpenStmt: nil statement")
	}
	ctx.WriteKeyword("OPEN ")
	ctx.WriteIdent(n.Name)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *OpenStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*OpenStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	return v.Leave(stmt)
}

// CloseStmt is a `CLOSE cursor_name` statement.
// See https://dev.mysql.com/doc/refman/8.0/en/close.html
type CloseStmt struct {
	stmtNode

	Name lex.Ident
}

// Restore implements Node interface.
func (n *CloseStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore CloseStmt: nil statement")
	}
	ctx.WriteKeyword("CLOSE ")
	ctx.WriteIdent(n.Name)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CloseStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*CloseStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	return v.Leave(stmt)
}

// FetchStmt is a `FETCH [NEXT] [FROM] cursor_name INTO var [, var]...`
// statement. See https://dev.mysql.com/doc/refman/8.0/en/fetch.html
type FetchStmt struct {
	stmtNode

	Name    lex.Ident
	Targets []*VariableTarget
	// HasNext/HasFrom mark the optional FETCH NEXT FROM prefixes; both are
	// absent on a plain `FETCH cursor INTO ...`.
	HasNext bool
	HasFrom bool
}

// Restore implements Node interface.
func (n *FetchStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore FetchStmt: nil statement")
	}
	ctx.WriteKeyword("FETCH ")
	if n.HasNext {
		ctx.WriteKeyword("NEXT ")
	}
	if n.HasFrom {
		ctx.WriteKeyword("FROM ")
	}
	ctx.WriteIdent(n.Name)
	if len(n.Targets) == 0 {
		return errors.New("An error occurred while restore FetchStmt.Targets: no targets")
	}
	ctx.WriteKeyword(" INTO ")
	for i, target := range n.Targets {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if target == nil {
			return errors.New("An error occurred while restore FetchStmt.Targets: nil target")
		}
		if err := target.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore FetchStmt.Targets[%d]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node interface.
func (n *FetchStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*FetchStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	for i, target := range n.Targets {
		if !nodePresent(target) {
			return n, false
		}
		updated, ok := acceptNode[*VariableTarget](v, target)
		if !ok {
			return n, false
		}
		n.Targets[i] = updated
	}
	return v.Leave(n)
}

// SignalInfoItem is a `name = expr` item of a SIGNAL or RESIGNAL statement.
type SignalInfoItem struct {
	Name  lex.Ident
	Value ExprNode
}

// SignalStmt is a `SIGNAL condition [SET item = expr, ...]` statement.
// See https://dev.mysql.com/doc/refman/8.0/en/signal.html
type SignalStmt struct {
	stmtNode

	Condition *HandlerCondition
	InfoItems []*SignalInfoItem
}

// Restore implements Node interface.
func (n *SignalStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("SignalStmt")
	}
	if n == nil || n.Condition == nil {
		return errors.New("restore signal statement: missing condition")
	}
	ctx.WriteKeyword("SIGNAL ")
	if err := n.Condition.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore SignalStmt.Condition")
	}
	if len(n.InfoItems) > 0 {
		ctx.WriteKeyword(" SET ")
		for i, item := range n.InfoItems {
			if item == nil || !nodePresent(item.Value) {
				return errors.Errorf("restore SignalStmt.InfoItems[%d]: missing value", i)
			}
			if i > 0 {
				ctx.WritePlain(", ")
			}
			ctx.WriteIdent(item.Name)
			ctx.WritePlain(" = ")
			if err := item.Value.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore SignalStmt.InfoItems")
			}
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *SignalStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*SignalStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	for _, item := range n.InfoItems {
		if item == nil {
			return n, false
		}
		updated, ok := acceptNode[ExprNode](v, item.Value)
		if !ok {
			return n, false
		}
		item.Value = updated
	}
	return v.Leave(n)
}

// ResignalStmt is a `RESIGNAL [condition] [SET item = expr, ...]` statement.
// See https://dev.mysql.com/doc/refman/8.0/en/resignal.html
type ResignalStmt struct {
	stmtNode

	Condition *HandlerCondition
	InfoItems []*SignalInfoItem
}

// Restore implements Node interface.
func (n *ResignalStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore ResignalStmt: nil statement")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	ctx.WriteKeyword("RESIGNAL")
	if n.Condition != nil {
		ctx.WritePlain(" ")
		if err := n.Condition.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore ResignalStmt.Condition")
		}
	}
	if len(n.InfoItems) > 0 {
		ctx.WriteKeyword(" SET ")
		for i, item := range n.InfoItems {
			if item == nil || !nodePresent(item.Value) {
				return errors.Errorf("restore ResignalStmt.InfoItems[%d]: missing value", i)
			}
			if i > 0 {
				ctx.WritePlain(", ")
			}
			ctx.WriteIdent(item.Name)
			ctx.WritePlain(" = ")
			if err := item.Value.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore ResignalStmt.InfoItems")
			}
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ResignalStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*ResignalStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	for _, item := range n.InfoItems {
		if item == nil {
			return n, false
		}
		updated, ok := acceptNode[ExprNode](v, item.Value)
		if !ok {
			return n, false
		}
		item.Value = updated
	}
	return v.Leave(n)
}

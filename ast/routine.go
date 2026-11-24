package ast

import (
	"github.com/mia0x75/myparser/auth"
	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/model"
	"github.com/mia0x75/parserkit/lex"
)

// FuncParamDir is the direction of a stored routine parameter.
type FuncParamDir int

// Directions of a stored routine parameter.
const (
	FuncParamDirIn FuncParamDir = iota
	FuncParamDirOut
	FuncParamDirInOut
)

// String returns the SQL keyword of the parameter direction.
func (d FuncParamDir) String() string {
	switch d {
	case FuncParamDirOut:
		return "OUT"
	case FuncParamDirInOut:
		return "INOUT"
	default:
		return "IN"
	}
}

// FuncParam is a parameter of a stored function or procedure.
type FuncParam struct {
	Name lex.Ident
	Type *TypeName
	Dir  FuncParamDir
}

// Restore implements Node interface.
func (p *FuncParam) Restore(ctx *format.RestoreCtx) error {
	if p == nil || !p.Name.Present() {
		return errors.New("ast: missing routine parameter name")
	}
	if p.Dir != FuncParamDirIn {
		ctx.WriteKeyword(p.Dir.String())
		ctx.WritePlain(" ")
	}
	ctx.WriteIdent(p.Name)
	ctx.WritePlain(" ")
	if p.Type == nil {
		return nil
	}
	return p.Type.Restore(ctx)
}

// RoutineDataAccess describes the data-access characteristic of a routine.
type RoutineDataAccess int

// Data-access characteristics.
const (
	RoutineDataAccessNoSQL RoutineDataAccess = iota
	RoutineDataAccessContainsSQL
	RoutineDataAccessReadsSQLData
	RoutineDataAccessModifiesSQLData
)

// RoutineCharacteristics holds the optional characteristics of a stored
// function or procedure.
type RoutineCharacteristics struct {
	Language               string
	LanguageSpecified      bool
	Deterministic          bool
	DeterministicSpecified bool
	DataAccess             RoutineDataAccess
	DataAccessSpecified    bool
	Security               model.ViewSecurity
	SecuritySpecified      bool
	Comment                string
	CommentSpecified       bool
}

// HasAny reports whether at least one characteristic was specified.
func (c *RoutineCharacteristics) HasAny() bool {
	if c == nil {
		return false
	}
	return c.LanguageSpecified || c.DeterministicSpecified || c.DataAccessSpecified ||
		c.SecuritySpecified || c.CommentSpecified
}

// Restore implements Node interface. Each clause is prefixed with a leading
// space so that an empty characteristic list renders nothing.
func (c *RoutineCharacteristics) Restore(ctx *format.RestoreCtx) error {
	if c == nil {
		return ctx.Err()
	}
	if c.LanguageSpecified {
		ctx.WritePlain(" ")
		ctx.WriteKeyword("LANGUAGE ")
		ctx.WriteKeyword(c.Language)
	}
	if c.DeterministicSpecified {
		ctx.WritePlain(" ")
		if !c.Deterministic {
			ctx.WriteKeyword("NOT ")
		}
		ctx.WriteKeyword("DETERMINISTIC")
	}
	if c.DataAccessSpecified {
		ctx.WritePlain(" ")
		switch c.DataAccess {
		case RoutineDataAccessContainsSQL:
			ctx.WriteKeyword("CONTAINS SQL")
		case RoutineDataAccessReadsSQLData:
			ctx.WriteKeyword("READS SQL DATA")
		case RoutineDataAccessModifiesSQLData:
			ctx.WriteKeyword("MODIFIES SQL DATA")
		default:
			ctx.WriteKeyword("NO SQL")
		}
	}
	if c.SecuritySpecified {
		ctx.WritePlain(" ")
		ctx.WriteKeyword("SQL SECURITY ")
		ctx.WriteKeyword(c.Security.String())
	}
	if c.CommentSpecified {
		ctx.WritePlain(" ")
		ctx.WriteKeyword("COMMENT ")
		ctx.WriteString(c.Comment)
	}
	return ctx.Err()
}

// EventSchedule is the ON SCHEDULE clause of a CREATE/ALTER EVENT statement.
type EventSchedule struct {
	At       ExprNode
	Every    ExprNode
	Interval string
	Starts   ExprNode
	Ends     ExprNode
}

func (s *EventSchedule) accept(v Visitor) (*EventSchedule, bool) {
	if s == nil {
		return nil, true
	}
	for _, expr := range []*ExprNode{&s.At, &s.Every, &s.Starts, &s.Ends} {
		if *expr == nil {
			continue
		}
		updated, ok := acceptNode[ExprNode](v, *expr)
		if !ok {
			return s, false
		}
		*expr = updated
	}
	return s, true
}

// Restore implements Node interface.
func (s *EventSchedule) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil {
		return errors.New("restore event schedule: missing schedule")
	}
	ctx.WriteKeyword("ON SCHEDULE ")
	if s.At != nil {
		if !nodePresent(s.At) {
			return errors.New("restore EventSchedule: nil AT expression")
		}
		ctx.WriteKeyword("AT ")
		if err := s.At.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore EventSchedule.At")
		}
		return ctx.Err()
	}
	ctx.WriteKeyword("EVERY ")
	if !nodePresent(s.Every) {
		return errors.New("restore event schedule: missing interval expression")
	}
	if err := s.Every.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore EventSchedule.Every")
	}
	if s.Interval != "" {
		ctx.WritePlain(" ")
		ctx.WriteKeyword(s.Interval)
	}
	if s.Starts != nil {
		if !nodePresent(s.Starts) {
			return errors.New("restore EventSchedule: nil STARTS expression")
		}
		ctx.WriteKeyword(" STARTS ")
		if err := s.Starts.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore EventSchedule.Starts")
		}
	}
	if s.Ends != nil {
		if !nodePresent(s.Ends) {
			return errors.New("restore EventSchedule: nil ENDS expression")
		}
		ctx.WriteKeyword(" ENDS ")
		if err := s.Ends.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore EventSchedule.Ends")
		}
	}
	return ctx.Err()
}

// EventStatus is the [ENABLE | DISABLE | DISABLE ON SLAVE] clause of an event.
type EventStatus int

// Event statuses.
const (
	EventStatusDefault EventStatus = iota
	EventStatusEnable
	EventStatusDisable
	EventStatusDisableOnSlave
)

// CreateFunctionStmt is a statement to create a stored function or a user
// defined function (UDF).
// See https://dev.mysql.com/doc/refman/8.0/en/create-procedure.html
// See https://dev.mysql.com/doc/refman/8.0/en/create-function-udf.html
type CreateFunctionStmt struct {
	ddlNode

	OrReplace       bool
	IfNotExists     bool
	Aggregate       bool
	Definer         *auth.UserIdentity
	Name            *TableName
	Params          []*FuncParam
	ReturnType      *TypeName
	ReturnsRowType  *TableName
	Characteristics *RoutineCharacteristics
	Body            StmtNode
	// Soname is the shared library name of a UDF. When it is not empty the
	// statement is a `CREATE FUNCTION ... SONAME` UDF definition.
	Soname string
}

// Restore implements Node interface.
func (n *CreateFunctionStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("CreateFunctionStmt")
	}
	if n == nil || n.Name == nil {
		return errors.New("restore create function statement: missing name")
	}
	if !nodePresent(n.ReturnType) && !nodePresent(n.ReturnsRowType) {
		return errors.New("restore create function statement: missing return type")
	}
	if n.Soname != "" {
		if n.ReturnType == nil {
			return errors.New("restore CreateFunctionStmt: missing UDF return type")
		}
		ctx.WriteKeyword("CREATE ")
		if n.Aggregate {
			ctx.WriteKeyword("AGGREGATE ")
		}
		ctx.WriteKeyword("FUNCTION ")
		if n.IfNotExists {
			ctx.WriteKeyword("IF NOT EXISTS ")
		}
		if err := n.Name.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore CreateFunctionStmt.Name")
		}
		ctx.WriteKeyword(" RETURNS ")
		if err := n.ReturnType.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore CreateFunctionStmt.ReturnType")
		}
		ctx.WriteKeyword(" SONAME ")
		ctx.WriteString(n.Soname)
		return ctx.Err()
	}
	ctx.WriteKeyword("CREATE ")
	if n.OrReplace {
		ctx.WriteKeyword("OR REPLACE ")
	}
	ctx.WriteKeyword("DEFINER")
	ctx.WritePlain(" = ")
	if n.Definer == nil {
		return errors.New("restore create function statement: missing definer")
	}
	if err := n.Definer.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CreateFunctionStmt.Definer")
	}
	ctx.WriteKeyword(" FUNCTION ")
	if n.IfNotExists {
		ctx.WriteKeyword("IF NOT EXISTS ")
	}
	if err := n.Name.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CreateFunctionStmt.Name")
	}
	if err := restoreFuncParams(ctx, n.Params); err != nil {
		return err
	}
	ctx.WriteKeyword(" RETURNS ")
	if n.ReturnsRowType != nil {
		ctx.WriteKeyword("ROW TYPE OF ")
		if err := n.ReturnsRowType.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore CreateFunctionStmt.ReturnsRowType")
		}
	} else if err := n.ReturnType.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CreateFunctionStmt.ReturnType")
	}
	if err := n.Characteristics.Restore(ctx); err != nil {
		return err
	}
	if !nodePresent(n.Body) {
		return errors.New("restore create function statement: missing body")
	}
	ctx.WritePlain(" ")
	if err := n.Body.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CreateFunctionStmt.Body")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateFunctionStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*CreateFunctionStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	if !nodePresent(n.Name) {
		return n, false
	}
	updated, ok := acceptNode[*TableName](v, n.Name)
	if !ok {
		return n, false
	}
	n.Name = updated
	for _, param := range n.Params {
		if param == nil {
			// Restore rejects a missing parameter entry.
			return n, false
		}
		if param.Type == nil {
			continue
		}
		typeName, ok := acceptNode[*TypeName](v, param.Type)
		if !ok {
			return n, false
		}
		param.Type = typeName
	}
	if n.ReturnType != nil {
		typeName, ok := acceptNode[*TypeName](v, n.ReturnType)
		if !ok {
			return n, false
		}
		n.ReturnType = typeName
	}
	if n.ReturnsRowType != nil {
		rowType, ok := acceptNode[*TableName](v, n.ReturnsRowType)
		if !ok {
			return n, false
		}
		n.ReturnsRowType = rowType
	}
	if n.Body != nil {
		body, ok := acceptNode[StmtNode](v, n.Body)
		if !ok {
			return n, false
		}
		n.Body = body
	}
	return v.Leave(n)
}

// CreateProcedureStmt is a statement to create a stored procedure.
// See https://dev.mysql.com/doc/refman/8.0/en/create-procedure.html
type CreateProcedureStmt struct {
	ddlNode

	OrReplace       bool
	IfNotExists     bool
	Definer         *auth.UserIdentity
	Name            *TableName
	Params          []*FuncParam
	Characteristics *RoutineCharacteristics
	Body            StmtNode
}

// Restore implements Node interface.
func (n *CreateProcedureStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("CreateProcedureStmt")
	}
	if n == nil || n.Definer == nil {
		return errors.New("restore create procedure statement: missing definer")
	}
	if !nodePresent(n.Name) {
		return errors.New("restore create procedure statement: missing name")
	}
	if !nodePresent(n.Body) {
		return errors.New("restore create procedure statement: missing body")
	}
	ctx.WriteKeyword("CREATE ")
	if n.OrReplace {
		ctx.WriteKeyword("OR REPLACE ")
	}
	ctx.WriteKeyword("DEFINER")
	ctx.WritePlain(" = ")
	if err := n.Definer.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CreateProcedureStmt.Definer")
	}
	ctx.WriteKeyword(" PROCEDURE ")
	if n.IfNotExists {
		ctx.WriteKeyword("IF NOT EXISTS ")
	}
	if err := n.Name.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CreateProcedureStmt.Name")
	}
	if err := restoreFuncParams(ctx, n.Params); err != nil {
		return err
	}
	if err := n.Characteristics.Restore(ctx); err != nil {
		return err
	}
	ctx.WritePlain(" ")
	if err := n.Body.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CreateProcedureStmt.Body")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateProcedureStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*CreateProcedureStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	if !nodePresent(n.Name) {
		return n, false
	}
	updated, ok := acceptNode[*TableName](v, n.Name)
	if !ok {
		return n, false
	}
	n.Name = updated
	for _, param := range n.Params {
		if param == nil {
			// Restore rejects a missing parameter entry.
			return n, false
		}
		if param.Type == nil {
			continue
		}
		typeName, ok := acceptNode[*TypeName](v, param.Type)
		if !ok {
			return n, false
		}
		param.Type = typeName
	}
	if !nodePresent(n.Body) {
		return n, false
	}
	body, ok := acceptNode[StmtNode](v, n.Body)
	if !ok {
		return n, false
	}
	n.Body = body
	return v.Leave(n)
}

// AlterFunctionStmt is a statement to change the characteristics of a stored
// function.
// See https://dev.mysql.com/doc/refman/8.0/en/alter-function.html
type AlterFunctionStmt struct {
	ddlNode

	Name            *TableName
	Characteristics *RoutineCharacteristics
}

// Restore implements Node interface.
func (n *AlterFunctionStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Name == nil {
		return errors.New("restore AlterFunctionStmt: missing name")
	}
	ctx.WriteKeyword("ALTER FUNCTION ")
	if err := n.Name.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore AlterFunctionStmt.Name")
	}
	return n.Characteristics.Restore(ctx)
}

// Accept implements Node Accept interface.
func (n *AlterFunctionStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*AlterFunctionStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	updated, ok := acceptNode[*TableName](v, n.Name)
	if !ok {
		return n, false
	}
	n.Name = updated
	return v.Leave(n)
}

// AlterProcedureStmt is a statement to change the characteristics of a stored
// procedure.
// See https://dev.mysql.com/doc/refman/8.0/en/alter-procedure.html
type AlterProcedureStmt struct {
	ddlNode

	Name            *TableName
	Characteristics *RoutineCharacteristics
}

// Restore implements Node interface.
func (n *AlterProcedureStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Name == nil {
		return errors.New("restore AlterProcedureStmt: missing name")
	}
	ctx.WriteKeyword("ALTER PROCEDURE ")
	if err := n.Name.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore AlterProcedureStmt.Name")
	}
	return n.Characteristics.Restore(ctx)
}

// Accept implements Node Accept interface.
func (n *AlterProcedureStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*AlterProcedureStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	updated, ok := acceptNode[*TableName](v, n.Name)
	if !ok {
		return n, false
	}
	n.Name = updated
	return v.Leave(n)
}

// CreateTriggerStmt is a statement to create a trigger.
// See https://dev.mysql.com/doc/refman/8.0/en/create-trigger.html
type CreateTriggerStmt struct {
	ddlNode

	OrReplace        bool
	IfNotExists      bool
	Definer          *auth.UserIdentity
	Name             *TableName
	Time             string
	Event            string
	UpdateColumns    []QualifiedName
	Table            *TableName
	Order            string
	OtherTriggerName lex.Ident
	Body             StmtNode
}

// Restore implements Node interface.
func (n *CreateTriggerStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Name == nil || n.Table == nil || !nodePresent(n.Body) {
		return errors.New("restore CreateTriggerStmt: missing name, table or body")
	}
	ctx.WriteKeyword("CREATE ")
	if n.OrReplace {
		ctx.WriteKeyword("OR REPLACE ")
	}
	ctx.WriteKeyword("DEFINER")
	ctx.WritePlain(" = ")
	if n.Definer == nil {
		ctx.WriteKeyword("CURRENT_USER")
	} else if err := n.Definer.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CreateTriggerStmt.Definer")
	}
	ctx.WriteKeyword(" TRIGGER ")
	if n.IfNotExists {
		ctx.WriteKeyword("IF NOT EXISTS ")
	}
	if n.Name != nil {
		if err := n.Name.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore CreateTriggerStmt.Name")
		}
	}
	ctx.WriteKeyword(" " + n.Time + " " + n.Event)
	if n.Event == "UPDATE" && len(n.UpdateColumns) > 0 {
		ctx.WriteKeyword(" OF ")
		for i, col := range n.UpdateColumns {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := restoreQualifiedName(ctx, col); err != nil {
				return errors.Annotate(err, "An error occurred while restore CreateTriggerStmt.UpdateColumns")
			}
		}
	}
	ctx.WriteKeyword(" ON ")
	if n.Table != nil {
		if err := n.Table.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore CreateTriggerStmt.Table")
		}
	}
	ctx.WriteKeyword(" FOR EACH ROW ")
	if n.Order != "" && n.OtherTriggerName.Present() {
		ctx.WriteKeyword(n.Order)
		ctx.WritePlain(" ")
		ctx.WriteIdent(n.OtherTriggerName)
		ctx.WritePlain(" ")
	}
	if n.Body != nil {
		if err := n.Body.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore CreateTriggerStmt.Body")
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateTriggerStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*CreateTriggerStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	updated, ok := acceptNode[*TableName](v, n.Table)
	if !ok {
		return n, false
	}
	n.Table = updated
	updated, ok = acceptNode[*TableName](v, n.Name)
	if !ok {
		return n, false
	}
	n.Name = updated
	body, ok := acceptNode[StmtNode](v, n.Body)
	if !ok {
		return n, false
	}
	n.Body = body
	return v.Leave(n)
}

// CreateEventStmt is a statement to create an event.
// See https://dev.mysql.com/doc/refman/8.0/en/create-event.html
type CreateEventStmt struct {
	ddlNode

	Definer          *auth.UserIdentity
	IfNotExists      bool
	Name             *TableName
	Schedule         *EventSchedule
	OnCompletion     bool
	Preserve         bool
	Status           EventStatus
	Comment          string
	CommentSpecified bool
	Body             StmtNode
}

// Restore implements Node interface.
func (n *CreateEventStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("CreateEventStmt")
	}
	if n == nil || n.Definer == nil {
		return errors.New("restore create event statement: missing definer")
	}
	if !nodePresent(n.Name) {
		return errors.New("restore create event statement: missing name")
	}
	if !nodePresent(n.Body) {
		return errors.New("restore create event statement: missing body")
	}
	ctx.WriteKeyword("CREATE DEFINER")
	ctx.WritePlain(" = ")
	if err := n.Definer.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CreateEventStmt.Definer")
	}
	ctx.WriteKeyword(" EVENT ")
	if n.IfNotExists {
		ctx.WriteKeyword("IF NOT EXISTS ")
	}
	if err := n.Name.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CreateEventStmt.Name")
	}
	ctx.WritePlain(" ")
	if err := n.Schedule.Restore(ctx); err != nil {
		return err
	}
	if n.OnCompletion {
		ctx.WriteKeyword(" ON COMPLETION ")
		if !n.Preserve {
			ctx.WriteKeyword("NOT ")
		}
		ctx.WriteKeyword("PRESERVE")
	}
	restoreEventStatus(ctx, n.Status)
	if n.CommentSpecified {
		ctx.WriteKeyword(" COMMENT ")
		ctx.WriteString(n.Comment)
	}
	ctx.WriteKeyword(" DO ")
	return n.Body.Restore(ctx)
}

// Accept implements Node Accept interface.
func (n *CreateEventStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*CreateEventStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	updated, ok := acceptNode[*TableName](v, n.Name)
	if !ok {
		return n, false
	}
	n.Name = updated
	if n.Schedule != nil {
		schedule, ok := n.Schedule.accept(v)
		if !ok {
			return n, false
		}
		n.Schedule = schedule
	}
	body, ok := acceptNode[StmtNode](v, n.Body)
	if !ok {
		return n, false
	}
	n.Body = body
	return v.Leave(n)
}

// AlterEventStmt is a statement to alter an existing event.
// See https://dev.mysql.com/doc/refman/8.0/en/alter-event.html
type AlterEventStmt struct {
	ddlNode

	Definer          *auth.UserIdentity
	Name             *TableName
	Schedule         *EventSchedule
	OnCompletion     bool
	Preserve         bool
	RenameName       *TableName
	Status           EventStatus
	Comment          string
	CommentSpecified bool
	Body             StmtNode
}

// Restore implements Node interface.
func (n *AlterEventStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("AlterEventStmt")
	}
	if n == nil || n.Definer == nil {
		return errors.New("restore alter event statement: missing definer")
	}
	if !nodePresent(n.Name) {
		return errors.New("restore alter event statement: missing name")
	}
	ctx.WriteKeyword("ALTER DEFINER")
	ctx.WritePlain(" = ")
	if err := n.Definer.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore AlterEventStmt.Definer")
	}
	ctx.WriteKeyword(" EVENT ")
	if err := n.Name.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore AlterEventStmt.Name")
	}
	if n.Schedule != nil {
		ctx.WritePlain(" ")
		if err := n.Schedule.Restore(ctx); err != nil {
			return err
		}
	}
	if n.OnCompletion {
		ctx.WriteKeyword(" ON COMPLETION ")
		if !n.Preserve {
			ctx.WriteKeyword("NOT ")
		}
		ctx.WriteKeyword("PRESERVE")
	}
	if n.RenameName != nil {
		ctx.WriteKeyword(" RENAME TO ")
		if err := n.RenameName.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AlterEventStmt.RenameName")
		}
	}
	restoreEventStatus(ctx, n.Status)
	if n.CommentSpecified {
		ctx.WriteKeyword(" COMMENT ")
		ctx.WriteString(n.Comment)
	}
	if n.Body != nil {
		if !nodePresent(n.Body) {
			return errors.New("restore AlterEventStmt: nil body")
		}
		ctx.WriteKeyword(" DO ")
		if err := n.Body.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AlterEventStmt.Body")
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *AlterEventStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*AlterEventStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	updated, ok := acceptNode[*TableName](v, n.Name)
	if !ok {
		return n, false
	}
	n.Name = updated
	if n.Schedule != nil {
		schedule, ok := n.Schedule.accept(v)
		if !ok {
			return n, false
		}
		n.Schedule = schedule
	}
	if n.RenameName != nil {
		renameNode, ok := acceptNode[*TableName](v, n.RenameName)
		if !ok {
			return n, false
		}
		n.RenameName = renameNode
	}
	if n.Body != nil {
		body, ok := acceptNode[StmtNode](v, n.Body)
		if !ok {
			return n, false
		}
		n.Body = body
	}
	return v.Leave(n)
}

// ReturnStmt is the RETURN statement used in stored functions.
// See https://dev.mysql.com/doc/refman/8.0/en/return.html
type ReturnStmt struct {
	stmtNode

	Expr ExprNode
}

// Restore implements Node interface.
func (n *ReturnStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("ReturnStmt")
	}
	if n == nil || !nodePresent(n.Expr) {
		return errors.New("restore return statement: missing expression")
	}
	ctx.WriteKeyword("RETURN ")
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore ReturnStmt.Expr")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ReturnStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*ReturnStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	updated, ok := acceptNode[ExprNode](v, n.Expr)
	if !ok {
		return n, false
	}
	n.Expr = updated
	return v.Leave(n)
}

func restoreFuncParams(ctx *format.RestoreCtx, params []*FuncParam) error {
	ctx.WritePlain("(")
	for i, p := range params {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if p == nil {
			return errors.Errorf("restore routine parameter %d: nil parameter", i)
		}
		if err := p.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore routine parameters")
		}
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

func restoreEventStatus(ctx *format.RestoreCtx, status EventStatus) {
	switch status {
	case EventStatusEnable:
		ctx.WriteKeyword(" ENABLE")
	case EventStatusDisable:
		ctx.WriteKeyword(" DISABLE")
	case EventStatusDisableOnSlave:
		ctx.WriteKeyword(" DISABLE ON SLAVE")
	}
}

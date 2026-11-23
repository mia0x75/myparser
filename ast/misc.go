package ast

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/mia0x75/myparser/errors"

	"github.com/mia0x75/myparser/auth"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

var (
	_ StmtNode = &AlterUserStmt{}
	_ StmtNode = &BeginStmt{}
	_ StmtNode = &BinlogStmt{}
	_ StmtNode = &CommitStmt{}
	_ StmtNode = &CallStmt{}
	_ StmtNode = &CreateUserStmt{}
	_ StmtNode = &DeallocateStmt{}
	_ StmtNode = &DoStmt{}
	_ StmtNode = &ExecuteStmt{}
	_ StmtNode = &ExplainStmt{}
	_ StmtNode = &GrantStmt{}
	_ StmtNode = &GrantRoleStmt{}
	_ StmtNode = &PrepareStmt{}
	_ StmtNode = &RollbackStmt{}
	_ StmtNode = &SavepointStmt{}
	_ StmtNode = &ReleaseSavepointStmt{}
	_ StmtNode = &XAStmt{}
	_ StmtNode = &SetPwdStmt{}
	_ StmtNode = &SetRoleStmt{}
	_ StmtNode = &SetDefaultRoleStmt{}
	_ StmtNode = &SetStmt{}
	_ StmtNode = &SetStatementStmt{}
	_ StmtNode = &UseStmt{}
	_ StmtNode = &PurgeStmt{}
	_ StmtNode = &FlushStmt{}
	_ StmtNode = &KillStmt{}
	_ StmtNode = &LockTableStmt{}
	_ StmtNode = &UnlockTableStmt{}

	_ Node = &PrivElem{}
	_ Node = &VariableAssignment{}
)

// Isolation level constants.
const (
	ReadCommitted   = "READ-COMMITTED"
	ReadUncommitted = "READ-UNCOMMITTED"
	Serializable    = "SERIALIZABLE"
	RepeatableRead  = "REPEATABLE-READ"

	// Valid formats for explain statement.
	ExplainFormatROW = "row"
	ExplainFormatDOT = "dot"
)

var (
	// ExplainFormats stores the valid formats for explain statement, used by validator.
	ExplainFormats = []string{
		ExplainFormatROW,
		ExplainFormatDOT,
	}
)

// TypeOpt is used for parsing data type option from SQL.
type TypeOpt struct {
	IsUnsigned bool
	IsZerofill bool
}

// FloatOpt is used for parsing floating-point type option from SQL.
// See http://dev.mysql.com/doc/refman/5.7/en/floating-point-types.html
type FloatOpt struct {
	Flen    int
	Decimal int
}

// AuthOption is used for parsing create use statement.
type AuthOption struct {
	// ByAuthString set as true, if AuthString is used for authorization. Otherwise, authorization is done by HashString.
	ByAuthString bool
	AuthString   string
	HashString   string
	// HashStringPresent distinguishes AS '' from an absent credential.
	HashStringPresent bool
	// AuthPlugin keeps the name from IDENTIFIED WITH; empty means the legacy forms.
	AuthPlugin string
}

// Restore implements Node interface.
func (n *AuthOption) Restore(ctx *format.RestoreCtx) error {
	if n.AuthPlugin != "" {
		ctx.WriteKeyword("IDENTIFIED WITH ")
		ctx.WriteName(n.AuthPlugin)
		switch {
		case n.ByAuthString:
			ctx.WriteKeyword(" BY ")
			ctx.WriteString(n.AuthString)
		case n.HashStringPresent || n.HashString != "":
			ctx.WriteKeyword(" AS ")
			ctx.WriteString(n.HashString)
		}
		return ctx.Err()
	}
	ctx.WriteKeyword("IDENTIFIED BY ")
	if n.ByAuthString {
		ctx.WriteString(n.AuthString)
	} else {
		ctx.WriteKeyword("PASSWORD ")
		ctx.WriteString(n.HashString)
	}
	return ctx.Err()
}

// ExplainForStmt is a statement to provite information about how is SQL statement executeing
// in connection #ConnectionID
// See https://dev.mysql.com/doc/refman/5.7/en/explain.html
type ExplainForStmt struct {
	stmtNode

	Format       string
	ConnectionID uint64
}

// Restore implements Node interface.
func (n *ExplainForStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("EXPLAIN ")
	ctx.WriteKeyword("FORMAT ")
	ctx.WritePlain("= ")
	ctx.WriteString(n.Format)
	ctx.WritePlain(" ")
	ctx.WriteKeyword("FOR ")
	ctx.WriteKeyword("CONNECTION ")
	ctx.WritePlain(strconv.FormatUint(n.ConnectionID, 10))
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ExplainForStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*ExplainForStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// ExplainStmt is a statement to provide information about how is SQL statement executed
// or get columns information in a table.
// See https://dev.mysql.com/doc/refman/5.7/en/explain.html
type ExplainStmt struct {
	stmtNode

	Stmt       StmtNode
	Format     string
	Analyze    bool
	Extended   bool
	Partitions bool
	// AnalyzeOnly records MariaDB's standalone ANALYZE statement.
	AnalyzeOnly bool
	// Into records MySQL's EXPLAIN FORMAT=JSON INTO @var clause.
	Into *SelectIntoStatement
	// Schema records MySQL's EXPLAIN ... FOR SCHEMA db clause.
	Schema lex.Ident
}

// Restore implements Node interface.
func (n *ExplainStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Stmt) {
		return errors.New("ast: missing explained statement")
	}
	if showStmt, ok := n.Stmt.(*ShowStmt); ok {
		if showStmt.Table == nil {
			return errors.New("ast: missing described table")
		}
		ctx.WriteKeyword("DESC ")
		if err := showStmt.Table.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore ExplainStmt.ShowStmt.Table")
		}
		if showStmt.Column != nil {
			ctx.WritePlain(" ")
			if err := restoreQualifiedName(ctx, *showStmt.Column); err != nil {
				return errors.Annotate(err, "An error occurred while restore ExplainStmt.ShowStmt.Column")
			}
		}
		return ctx.Err()
	}
	if !n.AnalyzeOnly {
		ctx.WriteKeyword("EXPLAIN ")
	}
	if n.Analyze {
		ctx.WriteKeyword("ANALYZE ")
	}
	if n.Extended {
		ctx.WriteKeyword("EXTENDED ")
	} else if n.Partitions {
		ctx.WriteKeyword("PARTITIONS ")
	} else if !n.Analyze || n.Format != "row" {
		ctx.WriteKeyword("FORMAT ")
		ctx.WritePlain("= ")
		ctx.WriteString(n.Format)
		ctx.WritePlain(" ")
	}
	if n.Into != nil {
		if err := n.Into.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore ExplainStmt.Into")
		}
		ctx.WritePlain(" ")
	}
	if n.Schema.Present() {
		ctx.WriteKeyword("FOR ")
		ctx.WriteKeyword("SCHEMA ")
		ctx.WriteIdent(n.Schema)
		ctx.WritePlain(" ")
	}
	if err := n.Stmt.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore ExplainStmt.Stmt")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ExplainStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*ExplainStmt)
	if !ok || next == nil {
		return n, false
	}
	n = next
	if skipChildren {
		return v.Leave(n)
	}
	if n.Into != nil {
		node, ok := n.Into.Accept(v)
		if !ok {
			return n, false
		}
		into, ok := node.(*SelectIntoStatement)
		if !ok || into == nil {
			return n, false
		}
		n.Into = into
	}
	if !nodePresent(n.Stmt) {
		return n, false
	}
	node, ok := n.Stmt.Accept(v)
	if !ok {
		return n, false
	}
	stmt, ok := node.(StmtNode)
	if !ok || !nodePresent(stmt) {
		return n, false
	}
	n.Stmt = stmt
	return v.Leave(n)
}

// PrepareStmt is a statement to prepares a SQL statement which contains placeholders,
// and it is executed with ExecuteStmt and released with DeallocateStmt.
// See https://dev.mysql.com/doc/refman/5.7/en/prepare.html
type PrepareStmt struct {
	stmtNode

	Name    lex.Ident
	SQLText string
	// SQLTextPresent distinguishes an empty SQL string from a variable source.
	SQLTextPresent bool
	SQLVar         *VariableExpr
}

// Restore implements Node interface.
func (n *PrepareStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("PREPARE ")
	ctx.WriteIdent(n.Name)
	ctx.WriteKeyword(" FROM ")
	if n.SQLTextPresent || n.SQLText != "" {
		ctx.WriteString(n.SQLText)
		return ctx.Err()
	}
	if n.SQLVar != nil {
		if err := n.SQLVar.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore PrepareStmt.SQLVar")
		}
		return ctx.Err()
	}
	return errors.New("An error occurred while restore PrepareStmt")
}

// Accept implements Node Accept interface.
func (n *PrepareStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*PrepareStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	if n.SQLVar != nil {
		variable, ok := acceptNode[*VariableExpr](v, n.SQLVar)
		if !ok {
			return n, false
		}
		n.SQLVar = variable
	}
	return v.Leave(n)
}

// DeallocateStmt is a statement to release PreparedStmt.
// See https://dev.mysql.com/doc/refman/5.7/en/deallocate-prepare.html
type DeallocateStmt struct {
	stmtNode

	Name lex.Ident
}

// Restore implements Node interface.
func (n *DeallocateStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("DEALLOCATE PREPARE ")
	ctx.WriteIdent(n.Name)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DeallocateStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*DeallocateStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// ExecuteStmt is a statement to execute PreparedStmt.
// See https://dev.mysql.com/doc/refman/5.7/en/execute.html
type ExecuteStmt struct {
	stmtNode

	Name lex.Ident
	// SQLExpr is the SQL text expression in MariaDB EXECUTE IMMEDIATE.
	SQLExpr   ExprNode
	UsingVars []ExprNode
}

// Restore implements Node interface.
func (n *ExecuteStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore ExecuteStmt: nil statement")
	}
	ctx.WriteKeyword("EXECUTE ")
	if n.SQLExpr != nil {
		if !nodePresent(n.SQLExpr) {
			return errors.New("restore ExecuteStmt.SQLExpr: nil expression")
		}
		ctx.WriteKeyword("IMMEDIATE ")
		if err := n.SQLExpr.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore ExecuteStmt.SQLExpr")
		}
	} else {
		ctx.WriteIdent(n.Name)
	}
	if len(n.UsingVars) > 0 {
		ctx.WriteKeyword(" USING ")
		for i, val := range n.UsingVars {
			if !nodePresent(val) {
				return errors.Errorf("restore ExecuteStmt.UsingVars[%d]: missing expression", i)
			}
			if i != 0 {
				ctx.WritePlain(",")
			}
			if err := val.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore ExecuteStmt.UsingVars index %d", i)
			}
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ExecuteStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*ExecuteStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	if n.SQLExpr != nil {
		expr, ok := acceptNode[ExprNode](v, n.SQLExpr)
		if !ok {
			return n, false
		}
		n.SQLExpr = expr
	}
	for i, val := range n.UsingVars {
		expr, ok := acceptNode[ExprNode](v, val)
		if !ok {
			return n, false
		}
		n.UsingVars[i] = expr
	}
	return v.Leave(n)
}

// BeginStmt is a statement to start a new transaction.
// See https://dev.mysql.com/doc/refman/5.7/en/commit.html
type BeginStmt struct {
	stmtNode
}

// Restore implements Node interface.
func (n *BeginStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("START TRANSACTION")
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *BeginStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*BeginStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// BinlogStmt is an internal-use statement.
// We just parse and ignore it.
// See http://dev.mysql.com/doc/refman/5.7/en/binlog.html
type BinlogStmt struct {
	stmtNode
	Str string
}

// Restore implements Node interface.
func (n *BinlogStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("BINLOG ")
	ctx.WriteString(n.Str)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *BinlogStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*BinlogStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// BackupStmt is MariaDB's BACKUP LOCK / BACKUP UNLOCK statement.
type BackupStmt struct {
	stmtNode

	Table  *TableName
	Unlock bool
}

// Restore implements Node interface.
func (n *BackupStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("BACKUP ")
	if n.Unlock {
		ctx.WriteKeyword("UNLOCK")
		return ctx.Err()
	}
	ctx.WriteKeyword("LOCK ")
	return n.Table.Restore(ctx)
}

// Accept implements Node Accept interface.
func (n *BackupStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*BackupStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	if n.Table != nil {
		table, ok := acceptNode[*TableName](v, n.Table)
		if !ok {
			return n, false
		}
		n.Table = table
	}
	return v.Leave(n)
}

// TransactionOptions retains explicit completion choices; nil uses the session default.
type TransactionOptions struct {
	Chain   *bool
	Release *bool
}

func (n *TransactionOptions) restore(ctx *format.RestoreCtx) {
	if n.Chain != nil {
		ctx.WriteKeyword(" AND ")
		if !*n.Chain {
			ctx.WriteKeyword("NO ")
		}
		ctx.WriteKeyword("CHAIN")
	}
	if n.Release != nil {
		ctx.WritePlain(" ")
		if !*n.Release {
			ctx.WriteKeyword("NO ")
		}
		ctx.WriteKeyword("RELEASE")
	}
}

// CommitStmt is a statement to commit the current transaction.
// See https://dev.mysql.com/doc/refman/5.7/en/commit.html
type CommitStmt struct {
	stmtNode
	TransactionOptions
}

// Restore implements Node interface.
func (n *CommitStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("COMMIT")
	n.restore(ctx)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CommitStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*CommitStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// RollbackStmt is a statement to roll back the current transaction.
// See https://dev.mysql.com/doc/refman/5.7/en/commit.html
type RollbackStmt struct {
	stmtNode
	TransactionOptions

	// SavepointName is the target savepoint of `ROLLBACK TO [SAVEPOINT] name`.
	// Its zero value is absent for a full transaction rollback; an empty
	// backtick-quoted name is present and must not degrade to a full rollback.
	SavepointName lex.Ident
}

// Restore implements Node interface.
func (n *RollbackStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("ROLLBACK")
	if n.SavepointName.Present() {
		ctx.WriteKeyword(" TO SAVEPOINT ")
		ctx.WriteIdent(n.SavepointName)
	} else {
		n.restore(ctx)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *RollbackStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*RollbackStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// SavepointStmt is a `SAVEPOINT identifier` statement.
// See https://dev.mysql.com/doc/refman/8.0/en/savepoint.html
type SavepointStmt struct {
	stmtNode

	Name lex.Ident
}

// Restore implements Node interface.
func (n *SavepointStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("SAVEPOINT ")
	ctx.WriteIdent(n.Name)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *SavepointStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*SavepointStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// ReleaseSavepointStmt is a `RELEASE SAVEPOINT identifier` statement.
// See https://dev.mysql.com/doc/refman/8.0/en/savepoint.html
type ReleaseSavepointStmt struct {
	stmtNode

	Name lex.Ident
}

// Restore implements Node interface.
func (n *ReleaseSavepointStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("RELEASE SAVEPOINT ")
	ctx.WriteIdent(n.Name)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ReleaseSavepointStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*ReleaseSavepointStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// XAStmt is a statement to execute an XA transaction command.
// See https://dev.mysql.com/doc/refman/8.0/en/xa-statements.html
type XAStmt struct {
	stmtNode

	// Command is one of START, BEGIN, END, PREPARE, COMMIT, ROLLBACK, RECOVER.
	Command string
	// XID is the literal transaction identifier. It is empty for RECOVER.
	XID string
	// Extra holds trailing keywords such as JOIN, RESUME, SUSPEND, FOR MIGRATE
	// or ONE PHASE.
	Extra string
}

// Restore implements Node interface.
func (n *XAStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("XA ")
	ctx.WriteKeyword(n.Command)
	if n.XID != "" {
		ctx.WritePlain(" ")
		ctx.WriteString(n.XID)
	}
	if n.Extra != "" {
		ctx.WritePlain(" ")
		ctx.WriteKeyword(n.Extra)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *XAStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*XAStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// UseStmt selects the current database.
// See https://dev.mysql.com/doc/refman/5.7/en/use.html
type UseStmt struct {
	stmtNode

	Database lex.Ident
}

// Restore implements Node interface.
func (n *UseStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !n.Database.Present() {
		return errors.New("ast: missing use database")
	}
	ctx.WriteKeyword("USE ")
	ctx.WriteIdent(n.Database)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *UseStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, _ := v.Enter(n)
	next, ok := newNode.(*UseStmt)
	if !ok || next == nil {
		return n, false
	}
	return v.Leave(next)
}

// VariableAssignmentKind distinguishes variable assignments from the two
// SET forms that configure the connection character set.
type VariableAssignmentKind uint8

const (
	VariableAssignmentVariable VariableAssignmentKind = iota
	VariableAssignmentNames
	VariableAssignmentCharset
)

const (
	// SetNames identifies SET NAMES assignments.
	SetNames = "SetNAMES"
	// SetCharset identifies SET CHARACTER SET / SET CHARSET assignments.
	SetCharset = "SetCHARSET"
)

// VariableAssignment is a variable assignment struct.
type VariableAssignment struct {
	node
	Kind   VariableAssignmentKind
	Target *VariableTarget
	Value  ExprNode
	// ExtendValue is a way to store extended info.
	// VariableAssignment should be able to store information for SetCharset/SetPWD Stmt.
	// For SetCharsetStmt, Value is charset, ExtendValue is collation.
	ExtendValue ExprNode
}

// Restore implements Node interface.
func (n *VariableAssignment) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore VariableAssignment: missing assignment")
	}
	switch n.Kind {
	case VariableAssignmentNames:
		ctx.WriteKeyword("NAMES ")
	case VariableAssignmentCharset:
		ctx.WriteKeyword("CHARACTER SET ")
	case VariableAssignmentVariable:
		if n.Target == nil {
			return errors.New("ast: variable assignment has no target")
		}
		if err := n.Target.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore VariableAssignment.Target")
		}
		ctx.WritePlain("=")
	}
	if !nodePresent(n.Value) {
		return errors.New("ast: variable assignment has no value")
	}
	if err := n.Value.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore VariableAssignment.Value")
	}
	if n.ExtendValue != nil {
		if !nodePresent(n.ExtendValue) {
			return errors.New("restore VariableAssignment: nil collation value")
		}
		ctx.WriteKeyword(" COLLATE ")
		if err := n.ExtendValue.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore VariableAssignment.ExtendValue")
		}
	}
	return ctx.Err()
}

// Accept implements Node interface.
func (n *VariableAssignment) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*VariableAssignment](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	if n.Target != nil {
		target, ok := acceptNode[*VariableTarget](v, n.Target)
		if !ok {
			return n, false
		}
		n.Target = target
	}
	if n.Value != nil {
		expr, ok := acceptNode[ExprNode](v, n.Value)
		if !ok {
			return n, false
		}
		n.Value = expr
	}
	if n.ExtendValue != nil {
		updated, ok := acceptNode[ExprNode](v, n.ExtendValue)
		if !ok {
			return n, false
		}
		n.ExtendValue = updated
	}
	return v.Leave(n)
}

// FlushStmtType is the type for FLUSH statement.
type FlushStmtType int

// Flush statement types.
const (
	FlushNone FlushStmtType = iota
	FlushTables
	FlushPrivileges
	FlushStatus
)

// FlushStmt is a statement to flush tables/privileges/optimizer costs and so on.
type FlushStmt struct {
	stmtNode

	Tp              FlushStmtType // Privileges/Tables/...
	NoWriteToBinLog bool
	Tables          []*TableName // For FlushTableStmt, if Tables is empty, it means flush all tables.
	ReadLock        bool
}

// Restore implements Node interface.
func (n *FlushStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("FLUSH ")
	if n.NoWriteToBinLog {
		ctx.WriteKeyword("NO_WRITE_TO_BINLOG ")
	}
	switch n.Tp {
	case FlushTables:
		ctx.WriteKeyword("TABLES")
		for i, v := range n.Tables {
			if i == 0 {
				ctx.WritePlain(" ")
			} else {
				ctx.WritePlain(", ")
			}
			if err := v.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore FlushStmt.Tables[%d]", i)
			}
		}
		if n.ReadLock {
			ctx.WriteKeyword(" WITH READ LOCK")
		}
	case FlushPrivileges:
		ctx.WriteKeyword("PRIVILEGES")
	case FlushStatus:
		ctx.WriteKeyword("STATUS")
	default:
		return errors.New("Unsupported type of FlushTables")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *FlushStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*FlushStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	for i, table := range n.Tables {
		updated, ok := acceptNode[*TableName](v, table)
		if !ok {
			return n, false
		}
		n.Tables[i] = updated
	}
	return v.Leave(n)
}

// KillStmt is a statement to kill a query or connection.
type KillStmt struct {
	stmtNode

	// Query indicates whether terminate a single query on this connection or the whole connection.
	// If Query is true, terminates the statement the connection is currently executing, but leaves the connection itself intact.
	// If Query is false, terminates the connection associated with the given ConnectionID, after terminating any statement the connection is executing.
	Query        bool
	ConnectionID uint64
	// Expr is an expression-valued connection ID; nil uses ConnectionID.
	Expr ExprNode
}

// Restore implements Node interface.
func (n *KillStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("KILL")
	if n.Query {
		ctx.WriteKeyword(" QUERY")
	}
	if n.Expr != nil {
		ctx.WritePlain(" ")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore KillStmt.Expr")
		}
	} else {
		ctx.WritePlainf(" %d", n.ConnectionID)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *KillStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := nodeAs[*KillStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(stmt)
	}
	n = stmt
	if n.Expr != nil {
		exprNode, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		expr, ok := nodeAs[ExprNode](exprNode)
		if !ok {
			return n, false
		}
		n.Expr = expr
	}
	return v.Leave(n)
}

// SetStmt is the statement to set variables.
type SetStmt struct {
	stmtNode
	// Variables is the list of variable assignment.
	Variables []*VariableAssignment
}

// Restore implements Node interface.
func (n *SetStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("SET ")
	for i, v := range n.Variables {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore SetStmt.Variables[%d]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *SetStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*SetStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	for i, val := range n.Variables {
		variable, ok := acceptNode[*VariableAssignment](v, val)
		if !ok {
			return n, false
		}
		n.Variables[i] = variable
	}
	return v.Leave(n)
}

/*
// SetCharsetStmt is a statement to assign values to character and collation variables.
// See https://dev.mysql.com/doc/refman/5.7/en/set-statement.html
type SetCharsetStmt struct {
	stmtNode

	Charset string
	Collate string
}

// Accept implements Node Accept interface.
func (n *SetCharsetStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*SetCharsetStmt)
	return v.Leave(n)
}
*/

// SetPwdStmt is a statement to assign a password to user account.
// See https://dev.mysql.com/doc/refman/5.7/en/set-password.html
type SetPwdStmt struct {
	stmtNode

	User     *auth.UserIdentity
	Password string
}

// Restore implements Node interface.
func (n *SetPwdStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("SET PASSWORD")
	if n.User != nil {
		ctx.WriteKeyword(" FOR ")
		if err := n.User.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SetPwdStmt.User")
		}
	}
	ctx.WritePlain("=")
	ctx.WriteString(n.Password)
	return ctx.Err()
}

// SecureText implements SensitiveStatement interface.
func (n *SetPwdStmt) SecureText() string {
	return fmt.Sprintf("set password for user %s", n.User)
}

// Accept implements Node Accept interface.
func (n *SetPwdStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*SetPwdStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// SetRoleStmtType is the type for FLUSH statement.
type SetRoleStmtType int

// SetRole statement types.
const (
	SetRoleDefault SetRoleStmtType = iota
	SetRoleNone
	SetRoleAll
	SetRoleAllExcept
	SetRoleRegular
)

// SetRoleStmt TODO:
type SetRoleStmt struct {
	stmtNode

	SetRoleOpt SetRoleStmtType
	RoleList   []*auth.RoleIdentity
}

// Restore implements Node interface.
func (n *SetRoleStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("SET ROLE")
	switch n.SetRoleOpt {
	case SetRoleDefault:
		ctx.WriteKeyword(" DEFAULT")
	case SetRoleNone:
		ctx.WriteKeyword(" NONE")
	case SetRoleAll:
		ctx.WriteKeyword(" ALL")
	case SetRoleAllExcept:
		ctx.WriteKeyword(" ALL EXCEPT")
	}
	for i, role := range n.RoleList {
		ctx.WritePlain(" ")
		err := role.Restore(ctx)
		if err != nil {
			return errors.Annotate(err, "An error occurred while restore SetRoleStmt.RoleList")
		}
		if i != len(n.RoleList)-1 {
			ctx.WritePlain(",")
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *SetRoleStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*SetRoleStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// SetDefaultRoleStmt TODO:
type SetDefaultRoleStmt struct {
	stmtNode

	SetRoleOpt SetRoleStmtType
	RoleList   []*auth.RoleIdentity
	UserList   []*auth.UserIdentity
}

// Restore implements Node interface.
func (n *SetDefaultRoleStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("SET DEFAULT ROLE")
	switch n.SetRoleOpt {
	case SetRoleNone:
		ctx.WriteKeyword(" NONE")
	case SetRoleAll:
		ctx.WriteKeyword(" ALL")
	default:
	}
	for i, role := range n.RoleList {
		ctx.WritePlain(" ")
		err := role.Restore(ctx)
		if err != nil {
			return errors.Annotate(err, "An error occurred while restore SetDefaultRoleStmt.RoleList")
		}
		if i != len(n.RoleList)-1 {
			ctx.WritePlain(",")
		}
	}
	ctx.WritePlain(" TO")
	for i, user := range n.UserList {
		ctx.WritePlain(" ")
		err := user.Restore(ctx)
		if err != nil {
			return errors.Annotate(err, "An error occurred while restore SetDefaultRoleStmt.UserList")
		}
		if i != len(n.UserList)-1 {
			ctx.WritePlain(",")
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *SetDefaultRoleStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*SetDefaultRoleStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// UserSpec is used for parsing create user statement.
type UserSpec struct {
	User    *auth.UserIdentity
	AuthOpt *AuthOption
	IsRole  bool
}

// Restore implements Node interface.
func (n *UserSpec) Restore(ctx *format.RestoreCtx) error {
	if err := n.User.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore UserSpec.User")
	}
	if n.AuthOpt != nil {
		ctx.WritePlain(" ")
		if err := n.AuthOpt.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore UserSpec.AuthOpt")
		}
	}
	return ctx.Err()
}

// SecurityString formats the UserSpec without password information.
func (n *UserSpec) SecurityString() string {
	withPassword := false
	if opt := n.AuthOpt; opt != nil {
		if len(opt.AuthString) > 0 || len(opt.HashString) > 0 {
			withPassword = true
		}
	}
	if withPassword {
		return fmt.Sprintf("{%s password = ***}", n.User)
	}
	return n.User.String()
}

// EncodedPassword returns the encoded password (which is the real data mysql.user).
// The boolean value indicates input's password format is legal or not.
func (n *UserSpec) EncodedPassword() (string, bool) {
	if n.AuthOpt == nil {
		return "", true
	}

	opt := n.AuthOpt
	if opt.ByAuthString {
		return auth.EncodePassword(opt.AuthString), true
	}

	// Not a legal password string.
	if len(opt.HashString) != 41 || !strings.HasPrefix(opt.HashString, "*") {
		return "", false
	}
	return opt.HashString, true
}

// TODO:
const (
	TslNone = iota
	Ssl
	X509
	Cipher
	Issuer
	Subject
)

// TslOption TODO:
type TslOption struct {
	Type  int
	Value string
}

// Restore implements Node interface.
func (t *TslOption) Restore(ctx *format.RestoreCtx) error {
	switch t.Type {
	case TslNone:
		ctx.WriteKeyword("NONE")
	case Ssl:
		ctx.WriteKeyword("SSL")
	case X509:
		ctx.WriteKeyword("X509")
	case Cipher:
		ctx.WriteKeyword("CIPHER ")
		ctx.WriteString(t.Value)
	case Issuer:
		ctx.WriteKeyword("ISSUER ")
		ctx.WriteString(t.Value)
	case Subject:
		ctx.WriteKeyword("SUBJECT ")
		ctx.WriteString(t.Value)
	default:
		return errors.Errorf("Unsupported TslOption.Type %d", t.Type)
	}
	return ctx.Err()
}

// TODO:
const (
	MaxQueriesPerHour = iota + 1
	MaxUpdatesPerHour
	MaxConnectionsPerHour
	MaxUserConnections
)

// ResourceOption TODO:
type ResourceOption struct {
	Type  int
	Count int64
}

// Restore implements Node interface.
func (r *ResourceOption) Restore(ctx *format.RestoreCtx) error {
	switch r.Type {
	case MaxQueriesPerHour:
		ctx.WriteKeyword("MAX_QUERIES_PER_HOUR ")
	case MaxUpdatesPerHour:
		ctx.WriteKeyword("MAX_UPDATES_PER_HOUR ")
	case MaxConnectionsPerHour:
		ctx.WriteKeyword("MAX_CONNECTIONS_PER_HOUR ")
	case MaxUserConnections:
		ctx.WriteKeyword("MAX_USER_CONNECTIONS ")
	default:
		return errors.Errorf("Unsupported ResourceOption.Type %d", r.Type)
	}
	ctx.WritePlainf("%d", r.Count)
	return ctx.Err()
}

// TODO:
const (
	PasswordExpire = iota + 1
	PasswordExpireDefault
	PasswordExpireNever
	PasswordExpireInterval
	Lock
	Unlock
)

// PasswordOrLockOption TODO:
type PasswordOrLockOption struct {
	Type  int
	Count int64
}

// Restore implements Node interface.
func (p *PasswordOrLockOption) Restore(ctx *format.RestoreCtx) error {
	switch p.Type {
	case PasswordExpire:
		ctx.WriteKeyword("PASSWORD EXPIRE")
	case PasswordExpireDefault:
		ctx.WriteKeyword("PASSWORD EXPIRE DEFAULT")
	case PasswordExpireNever:
		ctx.WriteKeyword("PASSWORD EXPIRE NEVER")
	case PasswordExpireInterval:
		ctx.WriteKeyword("PASSWORD EXPIRE NEVER")
		ctx.WritePlainf(" %d", p.Count)
		ctx.WriteKeyword(" DAY")
	case Lock:
		ctx.WriteKeyword("ACCOUNT LOCK")
	case Unlock:
		ctx.WriteKeyword("ACCOUNT UNLOCK")
	default:
		return errors.Errorf("Unsupported PasswordOrLockOption.Type %d", p.Type)
	}
	return ctx.Err()
}

// CreateUserStmt creates user account.
// See https://dev.mysql.com/doc/refman/5.7/en/create-user.html
type CreateUserStmt struct {
	stmtNode

	IsCreateRole          bool
	IfNotExists           bool
	Specs                 []*UserSpec
	TslOptions            []*TslOption
	ResourceOptions       []*ResourceOption
	PasswordOrLockOptions []*PasswordOrLockOption
}

// Restore implements Node interface.
func (n *CreateUserStmt) Restore(ctx *format.RestoreCtx) error {
	if n.IsCreateRole {
		ctx.WriteKeyword("CREATE ROLE ")
	} else {
		ctx.WriteKeyword("CREATE USER ")
	}
	if n.IfNotExists {
		ctx.WriteKeyword("IF NOT EXISTS ")
	}
	for i, v := range n.Specs {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore CreateUserStmt.Specs[%d]", i)
		}
	}

	tslOptionLen := len(n.TslOptions)

	if tslOptionLen != 0 {
		ctx.WriteKeyword(" REQUIRE ")
	}

	// Restore `tslOptions` reversely to keep order the same with original sql
	for i := tslOptionLen; i > 0; i-- {
		if i != tslOptionLen {
			ctx.WriteKeyword(" AND ")
		}
		if err := n.TslOptions[i-1].Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore CreateUserStmt.TslOptions[%d]", i)
		}
	}

	if len(n.ResourceOptions) != 0 {
		ctx.WriteKeyword(" WITH")
	}

	for i, v := range n.ResourceOptions {
		ctx.WritePlain(" ")
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore CreateUserStmt.ResourceOptions[%d]", i)
		}
	}

	for i, v := range n.PasswordOrLockOptions {
		ctx.WritePlain(" ")
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore CreateUserStmt.PasswordOrLockOptions[%d]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateUserStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*CreateUserStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// SecureText implements SensitiveStatement interface.
func (n *CreateUserStmt) SecureText() string {
	var buf bytes.Buffer
	buf.WriteString("create user")
	for _, user := range n.Specs {
		buf.WriteString(" ")
		buf.WriteString(user.SecurityString())
	}
	return buf.String()
}

// AlterUserStmt modifies user account.
// See https://dev.mysql.com/doc/refman/5.7/en/alter-user.html
type AlterUserStmt struct {
	stmtNode

	IfExists    bool
	CurrentAuth *AuthOption
	Specs       []*UserSpec
}

// Restore implements Node interface.
func (n *AlterUserStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("ALTER USER ")
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}
	if n.CurrentAuth != nil {
		ctx.WriteKeyword("USER")
		ctx.WritePlain("() ")
		if err := n.CurrentAuth.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AlterUserStmt.CurrentAuth")
		}
	}
	for i, v := range n.Specs {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore AlterUserStmt.Specs[%d]", i)
		}
	}
	return ctx.Err()
}

// SecureText implements SensitiveStatement interface.
func (n *AlterUserStmt) SecureText() string {
	var buf bytes.Buffer
	buf.WriteString("alter user")
	for _, user := range n.Specs {
		buf.WriteString(" ")
		buf.WriteString(user.SecurityString())
	}
	return buf.String()
}

// Accept implements Node Accept interface.
func (n *AlterUserStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*AlterUserStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// DropUserStmt creates user account.
// See http://dev.mysql.com/doc/refman/5.7/en/drop-user.html
type DropUserStmt struct {
	stmtNode

	IfExists   bool
	IsDropRole bool
	UserList   []*auth.UserIdentity
}

// Restore implements Node interface.
func (n *DropUserStmt) Restore(ctx *format.RestoreCtx) error {
	if n.IsDropRole {
		ctx.WriteKeyword("DROP ROLE ")
	} else {
		ctx.WriteKeyword("DROP USER ")
	}
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}
	for i, v := range n.UserList {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore DropUserStmt.UserList[%d]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropUserStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*DropUserStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// DoStmt is the struct for DO statement.
type DoStmt struct {
	stmtNode

	Exprs []ExprNode
}

// Restore implements Node interface.
func (n *DoStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("DO ")
	for i, v := range n.Exprs {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore DoStmt.Exprs[%d]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DoStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*DoStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	for i, val := range n.Exprs {
		expr, ok := acceptNode[ExprNode](v, val)
		if !ok {
			return n, false
		}
		n.Exprs[i] = expr
	}
	return v.Leave(n)
}

// HandleRange represents a range where handle value >= Begin and < End.
type HandleRange struct {
	Begin int64
	End   int64
}

// PrivElem is the privilege type and optional column list.
type PrivElem struct {
	node

	Priv mysql.PrivilegeType
	Cols []QualifiedName
}

// Restore implements Node interface.
func (n *PrivElem) Restore(ctx *format.RestoreCtx) error {
	switch n.Priv {
	case 0:
		ctx.WritePlain("/* UNSUPPORTED TYPE */")
	case mysql.AllPriv:
		ctx.WriteKeyword("ALL")
	case mysql.AlterPriv:
		ctx.WriteKeyword("ALTER")
	case mysql.CreatePriv:
		ctx.WriteKeyword("CREATE")
	case mysql.CreateUserPriv:
		ctx.WriteKeyword("CREATE USER")
	case mysql.CreateRolePriv:
		ctx.WriteKeyword("CREATE ROLE")
	case mysql.TriggerPriv:
		ctx.WriteKeyword("TRIGGER")
	case mysql.DeletePriv:
		ctx.WriteKeyword("DELETE")
	case mysql.DropPriv:
		ctx.WriteKeyword("DROP")
	case mysql.ProcessPriv:
		ctx.WriteKeyword("PROCESS")
	case mysql.ExecutePriv:
		ctx.WriteKeyword("EXECUTE")
	case mysql.IndexPriv:
		ctx.WriteKeyword("INDEX")
	case mysql.InsertPriv:
		ctx.WriteKeyword("INSERT")
	case mysql.SelectPriv:
		ctx.WriteKeyword("SELECT")
	case mysql.SuperPriv:
		ctx.WriteKeyword("SUPER")
	case mysql.ShowDBPriv:
		ctx.WriteKeyword("SHOW DATABASES")
	case mysql.UpdatePriv:
		ctx.WriteKeyword("UPDATE")
	case mysql.GrantPriv:
		ctx.WriteKeyword("GRANT OPTION")
	case mysql.ReferencesPriv:
		ctx.WriteKeyword("REFERENCES")
	case mysql.CreateViewPriv:
		ctx.WriteKeyword("CREATE VIEW")
	case mysql.ShowViewPriv:
		ctx.WriteKeyword("SHOW VIEW")
	default:
		return errors.New("Undefined privilege type")
	}
	if n.Cols != nil {
		ctx.WritePlain(" (")
		for i, v := range n.Cols {
			if i != 0 {
				ctx.WritePlain(",")
			}
			if err := restoreQualifiedName(ctx, v); err != nil {
				return errors.Annotatef(err, "An error occurred while restore PrivElem.Cols[%d]", i)
			}
		}
		ctx.WritePlain(")")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *PrivElem) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*PrivElem](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// ObjectTypeType is the type for object type.
type ObjectTypeType int

const (
	// ObjectTypeNone is for empty object type.
	ObjectTypeNone ObjectTypeType = iota + 1
	// ObjectTypeTable means the following object is a table.
	ObjectTypeTable
)

// Restore implements Node interface.
func (n ObjectTypeType) Restore(ctx *format.RestoreCtx) error {
	switch n {
	case ObjectTypeNone:
		// do nothing
	case ObjectTypeTable:
		ctx.WriteKeyword("TABLE")
	default:
		return errors.New("Unsupported object type")
	}
	return ctx.Err()
}

// GrantObjectPart distinguishes a literal identifier from an unquoted star.
type GrantObjectPart struct {
	Name         lex.Ident
	WildcardSpan loc.Span
	Wildcard     bool
}

type GrantObject struct {
	node
	Parts []GrantObjectPart
}

func (n *GrantObject) validate() error {
	if n == nil || len(n.Parts) == 0 || len(n.Parts) > 2 {
		return errors.New("ast: invalid grant object")
	}
	for _, part := range n.Parts {
		if part.Name.Present() == part.Wildcard {
			return errors.New("ast: grant object part requires a name or wildcard")
		}
	}
	if len(n.Parts) == 2 && n.Parts[0].Wildcard && !n.Parts[1].Wildcard {
		return errors.New("ast: named object cannot follow a wildcard database")
	}
	return nil
}

func (n *GrantObject) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := n.validate(); err != nil {
		return err
	}
	for i, part := range n.Parts {
		if i != 0 {
			ctx.WritePlain(".")
		}
		if part.Wildcard {
			ctx.WritePlain("*")
		} else {
			ctx.WriteIdent(part.Name)
		}
	}
	return ctx.Err()
}

func (n *GrantObject) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	entered, skip := v.Enter(n)
	next, ok := entered.(*GrantObject)
	if !ok || next == nil {
		return n, false
	}
	if skip {
		return v.Leave(next)
	}
	return v.Leave(next)
}

// RevokeStmt is the struct for REVOKE statement.
type RevokeStmt struct {
	stmtNode

	Privs      []*PrivElem
	ObjectType ObjectTypeType
	Object     *GrantObject
	Users      []*UserSpec
}

// Restore implements Node interface.
func (n *RevokeStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil || n.Object == nil {
		return errors.New("ast: missing revoke object")
	}
	if err := n.Object.validate(); err != nil {
		return err
	}
	ctx.WriteKeyword("REVOKE ")
	for i, v := range n.Privs {
		if v == nil {
			return errors.New("ast: missing revoke privilege")
		}
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore RevokeStmt.Privs[%d]", i)
		}
	}
	ctx.WriteKeyword(" ON ")
	if n.ObjectType != ObjectTypeNone {
		if err := n.ObjectType.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore RevokeStmt.ObjectType")
		}
		ctx.WritePlain(" ")
	}
	if err := n.Object.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore RevokeStmt.Object")
	}
	ctx.WriteKeyword(" FROM ")
	for i, v := range n.Users {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if v == nil {
			return errors.New("ast: missing revoke user")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore RevokeStmt.Users[%d]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *RevokeStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*RevokeStmt)
	if !ok || next == nil {
		return n, false
	}
	n = next
	if skipChildren {
		return v.Leave(n)
	}
	for i, val := range n.Privs {
		if val == nil {
			return n, false
		}
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		priv, ok := node.(*PrivElem)
		if !ok || priv == nil {
			return n, false
		}
		n.Privs[i] = priv
	}
	if n.Object == nil {
		return n, false
	}
	object, ok := n.Object.Accept(v)
	if !ok {
		return n, false
	}
	nextObject, ok := object.(*GrantObject)
	if !ok || nextObject == nil {
		return n, false
	}
	n.Object = nextObject
	return v.Leave(n)
}

// RevokeRoleStmt is the struct for REVOKE statement.
type RevokeRoleStmt struct {
	stmtNode

	Roles []*auth.RoleIdentity
	Users []*auth.UserIdentity
}

// Restore implements Node interface.
func (n *RevokeRoleStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("REVOKE ")
	for i, role := range n.Roles {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := role.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore RevokeRoleStmt.Roles[%d]", i)
		}
	}
	ctx.WriteKeyword(" FROM ")
	for i, v := range n.Users {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore RevokeRoleStmt.Users[%d]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *RevokeRoleStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*RevokeRoleStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// GrantStmt is the struct for GRANT statement.
type GrantStmt struct {
	stmtNode

	Privs      []*PrivElem
	ObjectType ObjectTypeType
	Object     *GrantObject
	Users      []*UserSpec
	WithGrant  bool
}

// Restore implements Node interface.
func (n *GrantStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil || n.Object == nil {
		return errors.New("ast: missing grant object")
	}
	if err := n.Object.validate(); err != nil {
		return err
	}
	ctx.WriteKeyword("GRANT ")
	for i, v := range n.Privs {
		if v == nil {
			return errors.New("ast: missing grant privilege")
		}
		if i != 0 && v.Priv != 0 {
			ctx.WritePlain(", ")
		} else if v.Priv == 0 {
			ctx.WritePlain(" ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore GrantStmt.Privs[%d]", i)
		}
	}
	ctx.WriteKeyword(" ON ")
	if n.ObjectType != ObjectTypeNone {
		if err := n.ObjectType.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore GrantStmt.ObjectType")
		}
		ctx.WritePlain(" ")
	}
	if err := n.Object.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore GrantStmt.Object")
	}
	ctx.WriteKeyword(" TO ")
	for i, v := range n.Users {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if v == nil {
			return errors.New("ast: missing grant user")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore GrantStmt.Users[%d]", i)
		}
	}
	if n.WithGrant {
		ctx.WriteKeyword(" WITH GRANT OPTION")
	}
	return ctx.Err()
}

// SecureText implements SensitiveStatement interface.
func (n *GrantStmt) SecureText() string {
	text := n.Text()
	// Filter "identified by xxx" because it would expose password information.
	idx := strings.Index(strings.ToLower(text), "identified")
	if idx > 0 {
		text = text[:idx]
	}
	return text
}

// Accept implements Node Accept interface.
func (n *GrantStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*GrantStmt)
	if !ok || next == nil {
		return n, false
	}
	n = next
	if skipChildren {
		return v.Leave(n)
	}
	for i, val := range n.Privs {
		if val == nil {
			return n, false
		}
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		priv, ok := node.(*PrivElem)
		if !ok || priv == nil {
			return n, false
		}
		n.Privs[i] = priv
	}
	if n.Object == nil {
		return n, false
	}
	object, ok := n.Object.Accept(v)
	if !ok {
		return n, false
	}
	nextObject, ok := object.(*GrantObject)
	if !ok || nextObject == nil {
		return n, false
	}
	n.Object = nextObject
	return v.Leave(n)
}

// GrantRoleStmt is the struct for GRANT TO statement.
type GrantRoleStmt struct {
	stmtNode

	Roles []*auth.RoleIdentity
	Users []*auth.UserIdentity
}

// Accept implements Node Accept interface.
func (n *GrantRoleStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*GrantRoleStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// Restore implements Node interface.
func (n *GrantRoleStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("GRANT ")
	if len(n.Roles) > 0 {
		for i, role := range n.Roles {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if err := role.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore GrantRoleStmt.Roles[%d]", i)
			}
		}
	}
	ctx.WriteKeyword(" TO ")
	for i, v := range n.Users {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore GrantStmt.Users[%d]", i)
		}
	}
	return ctx.Err()
}

// SecureText implements SensitiveStatement interface.
func (n *GrantRoleStmt) SecureText() string {
	text := n.Text()
	// Filter "identified by xxx" because it would expose password information.
	idx := strings.Index(strings.ToLower(text), "identified")
	if idx > 0 {
		text = text[:idx]
	}
	return text
}

// LockTableStmt TODO:
type LockTableStmt struct {
	stmtNode
}

// Restore implements Node interface.
func (n *LockTableStmt) Restore(ctx *format.RestoreCtx) error {
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *LockTableStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*LockTableStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// UnlockTableStmt TODO:
type UnlockTableStmt struct {
	stmtNode
}

// Restore implements Node interface.
func (n *UnlockTableStmt) Restore(ctx *format.RestoreCtx) error {
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *UnlockTableStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*UnlockTableStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// PurgeStmt restores `PURGE BINARY LOGS` and `PURGE MASTER LOGS`.
type PurgeStmt struct {
	stmtNode
	Binary bool
}

// Restore implements Node interface.
func (n *PurgeStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("PURGE")
	ctx.WritePlain(" ")
	if n.Binary {
		ctx.WriteKeyword("BINARY")
	} else {
		ctx.WriteKeyword("MASTER")
	}
	ctx.WritePlain(" LOGS")
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *PurgeStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	n, ok := nodeAs[*PurgeStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(n)
}

// SelectStmtOpts wrap around select hints and switches
type SelectStmtOpts struct {
	SQLBigResult    bool
	SQLBufferResult bool
	SQLCache        bool
	SQLSmallResult  bool
	CalcFoundRows   bool
	StraightJoin    bool
	Priority        mysql.PriorityEnum
	TableHints      []*OptimizerHint
}

type OptimizerHintKind uint8

const (
	OptimizerHintUnknown OptimizerHintKind = iota
	OptimizerHintMaxExecutionTime
)

type OptimizerHint struct {
	node
	Kind             OptimizerHintKind
	MaxExecutionTime uint64
}

func (n *OptimizerHint) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil || n.Kind != OptimizerHintMaxExecutionTime {
		return errors.New("ast: unsupported optimizer hint")
	}
	ctx.WriteKeyword("MAX_EXECUTION_TIME")
	ctx.WritePlainf("(%d)", n.MaxExecutionTime)
	return ctx.Err()
}

func (n *OptimizerHint) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*OptimizerHint)
	if !ok || next == nil {
		return n, false
	}
	if skipChildren {
		return v.Leave(next)
	}
	return v.Leave(next)
}

func acceptOptimizerHints(v Visitor, hints []*OptimizerHint) bool {
	for i, hint := range hints {
		if hint == nil {
			return false
		}
		node, ok := hint.Accept(v)
		if !ok {
			return false
		}
		next, ok := node.(*OptimizerHint)
		if !ok || next == nil {
			return false
		}
		hints[i] = next
	}
	return true
}

package ast

import (
	"fmt"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/myparser/errors"

	"github.com/mia0x75/myparser/auth"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/mysql"
)

var (
	_ DMLNode = &DeleteStmt{}
	_ DMLNode = &InsertStmt{}
	_ DMLNode = &SetOprStmt{}
	_ DMLNode = &UpdateStmt{}
	_ DMLNode = &SelectStmt{}
	_ DMLNode = &ValuesStmt{}
	_ DMLNode = &TableStmt{}
	_ DMLNode = &ShowStmt{}
	_ DMLNode = &LoadDataStmt{}

	_ Node                    = &Assignment{}
	_ Node                    = &Limit{}
	_ Node                    = &TableName{}
	_ Node                    = &TableRefsClause{}
	_ Node                    = &TableSource{}
	_ pkast.SetOperationNode  = (*SetOprStmt)(nil)
	_ pkast.QueryTailNode     = (*QueryTailStmt)(nil)
	_ Node                    = &SelectItem{}
	_ Node                    = &FieldList{}
	_ ExprNode                = &WildcardExpr{}
	_ Node                    = &WindowSpec{}
	_ Node                    = &PartitionByClause{}
	_ Node                    = &FrameClause{}
	_ Node                    = &FrameBound{}
	_ pkast.FromClauseNode    = (*TableRefsClause)(nil)
	_ pkast.GroupByClauseNode = (*GroupByClause)(nil)
	_ pkast.WithClauseNode    = (*WithClause)(nil)
	_ pkast.WindowSpecNode    = (*WindowSpec)(nil)
)

// TableName represents a table name. It embeds the shared BaseTableRef;
// diagnostics read schema/bare name and spans from Name.Parts.
type TableName struct {
	pkast.BaseTableRef

	PartitionNames []lex.Ident
}

// TableRefCore lets role-based parents check for a missing table before dispatch.
func (n *TableName) TableRefCore() *pkast.BaseTableRef {
	if n == nil {
		return nil
	}
	return &n.BaseTableRef
}

// Restore implements Node interface.
func (n *TableName) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return nilRestoreError("TableName")
	}
	if err := restoreQualifiedName(ctx, n.Name); err != nil {
		return errors.Annotate(err, "An error occurred while restore TableName.Name")
	}
	if len(n.PartitionNames) > 0 {
		ctx.WriteKeyword(" PARTITION")
		ctx.WritePlain("(")
		for i, v := range n.PartitionNames {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			ctx.WriteIdent(v)
		}
		ctx.WritePlain(")")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *TableName) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	table, ok := newNode.(*TableName)
	if !ok || table == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(table)
}

// SelectLockType is the lock type for SelectStmt.
type SelectLockType int

// Select lock types.
const (
	SelectLockNone SelectLockType = iota
	SelectLockForUpdate
	SelectLockInShareMode
	SelectLockForShare
)

// String implements fmt.Stringer.
func (slt SelectLockType) String() string {
	switch slt {
	case SelectLockNone:
		return "none"
	case SelectLockForUpdate:
		return "for update"
	case SelectLockInShareMode:
		return "in share mode"
	case SelectLockForShare:
		return "for share"
	}
	return "unsupported select lock type"
}

// SelectLockWaitPolicy is the wait policy of a locking read.
type SelectLockWaitPolicy int

// Wait policies for locking reads.
const (
	SelectLockWaitDefault SelectLockWaitPolicy = iota
	SelectLockWaitNoWait
	SelectLockWaitSkipLocked
)

// SelectLockInfo is the parsed SELECT locking clause.
type SelectLockInfo struct {
	// Type is the locking read type.
	Type SelectLockType
	// Tables is the optional `OF tbl_name [, ...]` list.
	Tables []*TableName
	// WaitPolicy is the optional NOWAIT / SKIP LOCKED policy.
	WaitPolicy SelectLockWaitPolicy
}

// TableRefsClause represents table references clause in dml statement.
type TableRefsClause struct {
	pkast.BaseFromClause
}

func (n *TableRefsClause) FromClauseCore() *pkast.BaseFromClause {
	if n == nil {
		return nil
	}
	return &n.BaseFromClause
}

// Restore implements Node interface.
func (n *TableRefsClause) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Table) {
		return errors.New("restore TableRefsClause: missing relation")
	}
	// Each query has its own join tree. A derived query must not inherit the
	// nesting level of the outer JOIN that contains its TableSource.
	joinLevel := ctx.JoinLevel
	ctx.JoinLevel = 0
	defer func() { ctx.JoinLevel = joinLevel }()
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore TableRefsClause.Table")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *TableRefsClause) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	clause, ok := newNode.(*TableRefsClause)
	if !ok || clause == nil {
		return newNode, false
	}
	n = clause
	if skipChildren {
		return v.Leave(n)
	}
	if !nodePresent(n.Table) {
		return n, false
	}
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	table, ok := nodeAs[ResultSetNode](node)
	if !ok {
		return n, false
	}
	n.Table = table
	return v.Leave(n)
}

// ProcedureAnalyseClause is MariaDB's legacy `PROCEDURE ANALYSE(...)`
// query-tail clause.
type ProcedureAnalyseClause struct {
	node

	Args []ExprNode
}

// Restore implements Node interface.
func (n *ProcedureAnalyseClause) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("restore procedure analyse: nil clause")
	}
	ctx.WriteKeyword("PROCEDURE ANALYSE")
	ctx.WritePlain("(")
	for i, arg := range n.Args {
		if !nodePresent(arg) {
			return errors.Errorf("restore procedure analyse arg %d: nil expression", i)
		}
		if i > 0 {
			ctx.WritePlain(",")
		}
		if err := arg.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore ProcedureAnalyseClause.Args[%d]", i)
		}
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ProcedureAnalyseClause) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	clause, ok := newNode.(*ProcedureAnalyseClause)
	if !ok || clause == nil {
		return newNode, false
	}
	n = clause
	if skipChildren {
		return v.Leave(n)
	}
	for i, arg := range n.Args {
		if !nodePresent(arg) {
			return n, false
		}
		node, ok := arg.Accept(v)
		if !ok {
			return n, false
		}
		expr, ok := node.(ExprNode)
		if !ok || !nodePresent(expr) {
			return n, false
		}
		n.Args[i] = expr
	}
	return v.Leave(n)
}

// SelectStmt represents the select query node.
// See https://dev.mysql.com/doc/refman/5.7/en/select.html
type SelectStmt struct {
	pkast.BaseSelectStmt

	// SelectStmtOpts wraps around select hints and switches.
	*SelectStmtOpts
	// ProcedureAnalyse is MariaDB's optional legacy query-tail clause.
	ProcedureAnalyse *ProcedureAnalyseClause
	// LockTp is the lock type
	LockTp SelectLockType
	// LockedTables is the optional `OF tbl_name [, ...]` list of a locking read.
	LockedTables []*TableName
	// LockWaitPolicy is the wait policy of a locking read (NOWAIT / SKIP LOCKED).
	LockWaitPolicy SelectLockWaitPolicy
	// Into is the optional SELECT ... INTO variables / OUTFILE / DUMPFILE clause.
	Into *SelectIntoStatement
	// IsInBraces indicates whether it's a stmt in brace.
	IsInBraces bool
	// AfterSetOperator indicates the SelectStmt after which type of set operator
	// (UNION/EXCEPT/INTERSECT) it is. It is nil when the SelectStmt is the first
	// operand of a set operation or when it is not part of one.
	AfterSetOperator *SetOprType
}

// Restore implements Node interface.
func (n *SelectStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("restore select: nil statement")
	}
	if n.With != nil {
		if n.With.WithClauseCore() == nil {
			return errors.New("An error occurred while restore SelectStmt.With: nil clause")
		}
		if err := n.With.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.With")
		}
		ctx.WritePlain(" ")
	}
	ctx.WriteKeyword("SELECT ")

	if n.SelectStmtOpts != nil && n.Priority > 0 {
		ctx.WriteKeyword(mysql.Priority2Str[n.Priority])
		ctx.WritePlain(" ")
	}

	if n.SelectStmtOpts != nil && n.SQLSmallResult {
		ctx.WriteKeyword("SQL_SMALL_RESULT ")
	}

	if n.SelectStmtOpts != nil && n.SQLBigResult {
		ctx.WriteKeyword("SQL_BIG_RESULT ")
	}

	if n.SelectStmtOpts != nil && n.SQLBufferResult {
		ctx.WriteKeyword("SQL_BUFFER_RESULT ")
	}

	if n.SelectStmtOpts != nil && n.CalcFoundRows {
		ctx.WriteKeyword("SQL_CALC_FOUND_ROWS ")
	}

	if n.SelectStmtOpts != nil && !n.SQLCache {
		ctx.WriteKeyword("SQL_NO_CACHE ")
	}

	if n.SelectStmtOpts != nil && len(n.TableHints) != 0 {
		ctx.WritePlain("/*+ ")
		for i, tableHint := range n.TableHints {
			if tableHint == nil {
				return errors.Errorf("An error occurred while restore SelectStmt.TableHints[%d]: nil hint", i)
			}
			if err := tableHint.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore SelectStmt.TableHints[%d]", i)
			}
		}
		ctx.WritePlain("*/ ")
	}

	if n.Distinct {
		ctx.WriteKeyword("DISTINCT ")
	}
	if n.SelectStmtOpts != nil && n.StraightJoin {
		ctx.WriteKeyword("STRAIGHT_JOIN ")
	}
	// A select without a projection is not restorable SQL.
	if n.Fields == nil || len(n.Fields.Items) == 0 {
		return errors.New("restore select: missing projection")
	}
	for i, field := range n.Fields.Items {
		if i != 0 {
			ctx.WritePlain(",")
		}
		if err := field.Restore(ctx); err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				// Keep the first write error ahead of a later validation
				// failure from the same restore.
				return fmt.Errorf("%w: %w", ctxErr, errors.Annotatef(err, "An error occurred while restore SelectStmt.Fields[%d]", i))
			}
			return errors.Annotatef(err, "An error occurred while restore SelectStmt.Fields[%d]", i)
		}
	}

	if n.Into != nil && n.Into.BeforeFrom {
		ctx.WritePlain(" ")
		if err := n.Into.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.Into before FROM")
		}
	}

	if n.From != nil {
		if n.From.FromClauseCore() == nil {
			return errors.New("An error occurred while restore SelectStmt.From: nil clause")
		}
		ctx.WriteKeyword(" FROM ")
		if err := n.From.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.From")
		}
	}

	if n.From == nil && n.Where != nil {
		ctx.WriteKeyword(" FROM DUAL")
	}
	if n.Where != nil {
		if !nodePresent(n.Where) {
			return errors.New("restore select: nil where expression")
		}
		ctx.WriteKeyword(" WHERE ")
		if err := n.Where.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.Where")
		}
	}

	if n.GroupBy != nil {
		if n.GroupBy.GroupByClauseCore() == nil {
			return errors.New("An error occurred while restore SelectStmt.GroupBy: nil clause")
		}
		ctx.WritePlain(" ")
		if err := n.GroupBy.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.GroupBy")
		}
	}

	if n.Having != nil {
		if !nodePresent(n.Having) {
			return errors.New("restore select: nil having expression")
		}
		ctx.WriteKeyword(" HAVING ")
		if err := n.Having.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.Having")
		}
	}

	if n.WindowSpecs != nil {
		ctx.WriteKeyword(" WINDOW ")
		for i, windowsSpec := range n.WindowSpecs {
			if windowsSpec == nil || windowsSpec.WindowSpecCore() == nil {
				return errors.Errorf("An error occurred while restore SelectStmt.WindowSpec[%d]: nil clause", i)
			}
			if i != 0 {
				ctx.WritePlain(",")
			}
			if err := windowsSpec.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore SelectStmt.WindowSpec[%d]", i)
			}
		}
	}

	if n.OrderBy != nil {
		if n.OrderBy.OrderByClauseCore() == nil {
			return errors.New("An error occurred while restore SelectStmt.OrderBy: nil clause")
		}
		ctx.WritePlain(" ")
		if err := n.OrderBy.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.OrderBy")
		}
	}

	if n.Limit != nil {
		if n.Limit.LimitCore() == nil {
			return errors.New("An error occurred while restore SelectStmt.Limit: nil clause")
		}
		ctx.WritePlain(" ")
		if err := n.Limit.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.Limit")
		}
	}

	if n.ProcedureAnalyse != nil {
		ctx.WritePlain(" ")
		if err := n.ProcedureAnalyse.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.ProcedureAnalyse")
		}
	}

	switch n.LockTp {
	case SelectLockInShareMode:
		ctx.WriteKeyword(" LOCK ")
		ctx.WriteKeyword(n.LockTp.String())
	case SelectLockForUpdate:
		ctx.WritePlain(" ")
		ctx.WriteKeyword("FOR UPDATE")
	case SelectLockForShare:
		ctx.WritePlain(" ")
		ctx.WriteKeyword("FOR SHARE")
	}
	if len(n.LockedTables) > 0 {
		ctx.WriteKeyword(" OF ")
		for i, tbl := range n.LockedTables {
			if tbl == nil {
				return errors.Errorf("restore select locked table %d: nil table", i)
			}
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if err := tbl.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore SelectStmt.LockedTables")
			}
		}
	}
	switch n.LockWaitPolicy {
	case SelectLockWaitNoWait:
		ctx.WriteKeyword(" NOWAIT")
	case SelectLockWaitSkipLocked:
		ctx.WriteKeyword(" SKIP LOCKED")
	}
	if n.Into != nil && !n.Into.BeforeFrom {
		ctx.WritePlain(" ")
		if err := n.Into.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SelectStmt.Into")
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *SelectStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	selectStmt, ok := newNode.(*SelectStmt)
	if !ok || selectStmt == nil {
		return newNode, false
	}
	n = selectStmt
	if skipChildren {
		return v.Leave(n)
	}
	if n.With != nil {
		if n.With.WithClauseCore() == nil {
			return n, false
		}
		node, ok := n.With.Accept(v)
		if !ok {
			return n, false
		}
		clause, ok := node.(pkast.WithClauseNode)
		if !ok || clause == nil || clause.WithClauseCore() == nil {
			return n, false
		}
		n.With = clause
	}
	if n.SelectStmtOpts != nil && !acceptOptimizerHints(v, n.TableHints) {
		return n, false
	}

	if n.From != nil {
		if n.From.FromClauseCore() == nil {
			return n, false
		}
		node, ok := n.From.Accept(v)
		if !ok {
			return n, false
		}
		clause, ok := node.(pkast.FromClauseNode)
		if !ok || clause == nil || clause.FromClauseCore() == nil {
			return n, false
		}
		n.From = clause
	}

	if n.Where != nil {
		if !nodePresent(n.Where) {
			return n, false
		}
		node, ok := n.Where.Accept(v)
		if !ok {
			return n, false
		}
		where, ok := node.(ExprNode)
		if !ok || !nodePresent(where) {
			return n, false
		}
		n.Where = where
	}

	if n.Fields != nil {
		node, ok := n.Fields.Accept(v)
		if !ok {
			return n, false
		}
		fields, ok := node.(*FieldList)
		if !ok || fields == nil {
			return n, false
		}
		n.Fields = fields
	}

	if n.GroupBy != nil {
		if n.GroupBy.GroupByClauseCore() == nil {
			return n, false
		}
		node, ok := n.GroupBy.Accept(v)
		if !ok {
			return n, false
		}
		clause, ok := node.(pkast.GroupByClauseNode)
		if !ok || clause == nil || clause.GroupByClauseCore() == nil {
			return n, false
		}
		n.GroupBy = clause
	}

	if n.Having != nil {
		if !nodePresent(n.Having) {
			return n, false
		}
		node, ok := n.Having.Accept(v)
		if !ok {
			return n, false
		}
		expr, ok := node.(ExprNode)
		if !ok || !nodePresent(expr) {
			return n, false
		}
		n.Having = expr
	}

	for i, spec := range n.WindowSpecs {
		if spec == nil || spec.WindowSpecCore() == nil {
			return n, false
		}
		node, ok := spec.Accept(v)
		if !ok {
			return n, false
		}
		window, ok := node.(pkast.WindowSpecNode)
		if !ok || window == nil || window.WindowSpecCore() == nil {
			return n, false
		}
		n.WindowSpecs[i] = window
	}

	if n.OrderBy != nil {
		if n.OrderBy.OrderByClauseCore() == nil {
			return n, false
		}
		node, ok := n.OrderBy.Accept(v)
		if !ok {
			return n, false
		}
		clause, ok := node.(pkast.OrderByClauseNode)
		if !ok || clause == nil || clause.OrderByClauseCore() == nil {
			return n, false
		}
		n.OrderBy = clause
	}

	if n.Limit != nil {
		if n.Limit.LimitCore() == nil {
			return n, false
		}
		node, ok := n.Limit.Accept(v)
		if !ok {
			return n, false
		}
		clause, ok := node.(pkast.LimitNode)
		if !ok || clause == nil || clause.LimitCore() == nil {
			return n, false
		}
		n.Limit = clause
	}

	if n.ProcedureAnalyse != nil {
		node, ok := n.ProcedureAnalyse.Accept(v)
		if !ok {
			return n, false
		}
		clause, ok := node.(*ProcedureAnalyseClause)
		if !ok || clause == nil {
			return n, false
		}
		n.ProcedureAnalyse = clause
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

	for i, table := range n.LockedTables {
		if table == nil {
			return n, false
		}
		node, ok := table.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := node.(*TableName)
		if !ok || replacement == nil {
			return n, false
		}
		n.LockedTables[i] = replacement
	}

	return v.Leave(n)
}

// ValuesStmt is a table value constructor statement.
// See https://dev.mysql.com/doc/refman/8.0/en/values.html
type ValuesStmt struct {
	pkast.BaseValuesStmt

	// ValuesKeywordOffset is the UTF-8 byte offset of the VALUES keyword in
	// the parsed source. It is used for source-located dialect diagnostics.
	ValuesKeywordOffset int
	// AfterSetOperator identifies the set operator before this VALUES operand.
	AfterSetOperator *SetOprType
}

// Restore implements Node interface.
func (n *ValuesStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore ValuesStmt: nil statement")
	}
	ctx.WriteKeyword("VALUES ")
	for i, row := range n.Rows {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if !nodePresent(row) || row.RowCore() == nil {
			return errors.Errorf("ValuesStmt.Rows[%d] is nil", i)
		}
		if err := row.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while splicing ValuesStmt Row[%v]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ValuesStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	n, ok := newNode.(*ValuesStmt)
	if !ok || n == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(n)
	}
	for i, row := range n.Rows {
		if !nodePresent(row) || row.RowCore() == nil {
			return n, false
		}
		node, ok := row.Accept(v)
		if !ok {
			return n, false
		}
		updated, ok := node.(pkast.RowNode)
		if !ok || !nodePresent(updated) || updated.RowCore() == nil {
			return n, false
		}
		n.Rows[i] = updated
	}
	return v.Leave(n)
}

// TableStmt is a `TABLE tbl_name` statement, which is equivalent to
// `SELECT * FROM tbl_name`.
// See https://dev.mysql.com/doc/refman/8.0/en/table.html
type TableStmt struct {
	pkast.BaseTableStmt
}

// Restore implements Node interface.
func (n *TableStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Table) || n.Table.TableRefCore() == nil {
		return errors.New("An error occurred while restore TableStmt.Table: nil table")
	}
	ctx.WriteKeyword("TABLE ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while splicing TableStmt Table")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *TableStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	n, ok := newNode.(*TableStmt)
	if !ok || n == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(n)
	}
	if !nodePresent(n.Table) || n.Table.TableRefCore() == nil {
		return n, false
	}
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	table, ok := node.(pkast.TableRefNode)
	if !ok || !nodePresent(table) || table.TableRefCore() == nil {
		return n, false
	}
	n.Table = table
	return v.Leave(n)
}

// Assignment shares target/value syntax and traversal with other dialects.
type Assignment = pkast.Assignment

// ColumnNameOrUserVar is a column name or user variable in LOAD DATA.
type ColumnNameOrUserVar struct {
	ColumnName *QualifiedName
	UserVar    *VariableExpr
}

// LoadDataStmt is a statement to load data from a specified file, then insert this rows into an existing table.
// See https://dev.mysql.com/doc/refman/5.7/en/load-data.html
type LoadDataStmt struct {
	dmlNode

	IsLocal           bool
	Path              string
	OnDuplicate       OnDuplicateKeyHandlingType
	Table             *TableName
	Columns           []QualifiedName
	FieldsInfo        *FieldsClause
	LinesInfo         *LinesClause
	IgnoreLines       uint64
	ColumnAssignments []*Assignment

	ColumnsAndUserVars []*ColumnNameOrUserVar
}

// Restore implements Node interface.
func (n *LoadDataStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Table == nil {
		return errors.New("restore LoadDataStmt: missing table")
	}
	ctx.WriteKeyword("LOAD DATA ")
	if n.IsLocal {
		ctx.WriteKeyword("LOCAL ")
	}
	ctx.WriteKeyword("INFILE ")
	ctx.WriteString(n.Path)
	switch n.OnDuplicate {
	case OnDuplicateKeyHandlingReplace:
		ctx.WriteKeyword(" REPLACE")
	case OnDuplicateKeyHandlingIgnore:
		ctx.WriteKeyword(" IGNORE")
	}
	ctx.WriteKeyword(" INTO TABLE ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore LoadDataStmt.Table")
	}
	if n.FieldsInfo != nil {
		if err := n.FieldsInfo.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore LoadDataStmt.FieldsInfo")
		}
	}
	if n.LinesInfo != nil {
		if err := n.LinesInfo.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore LoadDataStmt.LinesInfo")
		}
	}
	if n.IgnoreLines != 0 {
		ctx.WriteKeyword(" IGNORE ")
		ctx.WritePlainf("%d", n.IgnoreLines)
		ctx.WriteKeyword(" LINES")
	}
	if len(n.ColumnsAndUserVars) != 0 {
		ctx.WritePlain(" (")
		for i, c := range n.ColumnsAndUserVars {
			if c == nil || (c.ColumnName == nil) == (c.UserVar == nil) {
				return errors.Errorf("restore LoadDataStmt.ColumnsAndUserVars[%d]: missing or ambiguous target", i)
			}
			if i != 0 {
				ctx.WritePlain(",")
			}
			if c.ColumnName != nil {
				if err := restoreQualifiedName(ctx, *c.ColumnName); err != nil {
					return errors.Annotate(err, "An error occurred while restore LoadDataStmt.ColumnsAndUserVars")
				}
			}
			if c.UserVar != nil {
				if err := c.UserVar.Restore(ctx); err != nil {
					return errors.Annotate(err, "An error occurred while restore LoadDataStmt.ColumnsAndUserVars")
				}
			}
		}
		ctx.WritePlain(")")
	}

	if n.ColumnAssignments != nil {
		ctx.WriteKeyword(" SET")
		for i, assign := range n.ColumnAssignments {
			if assign == nil {
				return errors.Errorf("restore LoadDataStmt.ColumnAssignments[%d]: missing assignment", i)
			}
			if i != 0 {
				ctx.WritePlain(",")
			}
			ctx.WritePlain(" ")
			if err := assign.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore LoadDataStmt.ColumnAssignments")
			}
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *LoadDataStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*LoadDataStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	table, ok := acceptNode[*TableName](v, n.Table)
	if !ok {
		return n, false
	}
	n.Table = table
	for i, assignment := range n.ColumnAssignments {
		updated, ok := acceptNode[*Assignment](v, assignment)
		if !ok {
			return n, false
		}
		n.ColumnAssignments[i] = updated
	}
	for _, column := range n.ColumnsAndUserVars {
		if column == nil || (column.ColumnName == nil) == (column.UserVar == nil) {
			return n, false
		}
		if column.UserVar != nil {
			variable, ok := acceptNode[*VariableExpr](v, column.UserVar)
			if !ok {
				return n, false
			}
			column.UserVar = variable
		}
	}
	return v.Leave(n)
}

const (
	Terminated = iota
	Enclosed
	Escaped
)

type FieldItem struct {
	Type  int
	Value string
}

// FieldsClause represents fields references clause in load data statement.
type FieldsClause struct {
	Terminated string
	Enclosed   byte
	Escaped    byte
}

// Restore for FieldsClause
func (n *FieldsClause) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return nilRestoreError("FieldsClause")
	}
	// Enclosed alone must trigger the clause: with the other options at
	// their defaults, the FIELDS keyword is what carries the enclosed char.
	if n.Terminated != "\t" || n.Escaped != '\\' || n.Enclosed != 0 {
		ctx.WriteKeyword(" FIELDS")
		if n.Terminated != "\t" {
			ctx.WriteKeyword(" TERMINATED BY ")
			ctx.WriteString(n.Terminated)
		}
		if n.Enclosed != 0 {
			ctx.WriteKeyword(" ENCLOSED BY ")
			ctx.WriteString(string(n.Enclosed))
		}
		if n.Escaped != '\\' {
			ctx.WriteKeyword(" ESCAPED BY ")
			if n.Escaped == 0 {
				ctx.WritePlain("''")
			} else {
				ctx.WriteString(string(n.Escaped))
			}
		}
	}
	return ctx.Err()
}

// LinesClause represents lines references clause in load data statement.
type LinesClause struct {
	Starting   string
	Terminated string
}

// Restore for LinesClause
func (n *LinesClause) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return nilRestoreError("LinesClause")
	}
	if n.Starting != "" || n.Terminated != "\n" {
		ctx.WriteKeyword(" LINES")
		if n.Starting != "" {
			ctx.WriteKeyword(" STARTING BY ")
			ctx.WriteString(n.Starting)
		}
		if n.Terminated != "\n" {
			ctx.WriteKeyword(" TERMINATED BY ")
			ctx.WriteString(n.Terminated)
		}
	}
	return ctx.Err()
}

// SelectIntoType is the type of a SELECT ... INTO clause.
type SelectIntoType int

// SELECT ... INTO clause types.
const (
	SelectIntoVarList SelectIntoType = iota + 1
	SelectIntoOutfile
	SelectIntoDumpfile
)

// SelectIntoStatement represents the INTO clause of a SELECT statement,
// covering INTO var_list, INTO OUTFILE and INTO DUMPFILE.
type SelectIntoStatement struct {
	node

	Tp SelectIntoType
	// BeforeFrom records the SELECT ... INTO ... FROM spelling. It lets
	// validation distinguish that MySQL form from a query-tail INTO; Restore
	// may still normalize both forms to a query tail.
	BeforeFrom bool
	Targets    []*VariableTarget
	FileName   string
	Charset    string
	Fields     *FieldsClause
	Lines      *LinesClause
}

// Restore implements Node interface.
func (n *SelectIntoStatement) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("restore select into: nil clause")
	}
	switch n.Tp {
	case SelectIntoVarList:
		if len(n.Targets) == 0 {
			return errors.New("restore select into: missing targets")
		}
		ctx.WriteKeyword("INTO ")
		for i, target := range n.Targets {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if target == nil {
				return errors.Errorf("SelectIntoStatement.Targets[%d] is nil", i)
			}
			if err := target.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore SelectIntoStatement.Targets[%d]", i)
			}
		}
	case SelectIntoOutfile:
		ctx.WriteKeyword("INTO OUTFILE ")
		ctx.WriteString(n.FileName)
		if n.Charset != "" {
			ctx.WriteKeyword(" CHARACTER SET ")
			ctx.WritePlain(n.Charset)
		}
		if n.Fields != nil {
			if err := n.Fields.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore SelectIntoStatement.Fields")
			}
		}
		if n.Lines != nil {
			if err := n.Lines.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore SelectIntoStatement.Lines")
			}
		}
	case SelectIntoDumpfile:
		ctx.WriteKeyword("INTO DUMPFILE ")
		ctx.WriteString(n.FileName)
	default:
		return errors.Errorf("restore select into: invalid type %d", n.Tp)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *SelectIntoStatement) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	into, ok := newNode.(*SelectIntoStatement)
	if !ok || into == nil {
		return newNode, false
	}
	n = into
	if skipChildren {
		return v.Leave(n)
	}
	for i, target := range n.Targets {
		if target == nil {
			return n, false
		}
		node, ok := target.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := node.(*VariableTarget)
		if !ok || replacement == nil {
			return n, false
		}
		n.Targets[i] = replacement
	}
	return v.Leave(n)
}

// InsertRowAlias is MySQL's row alias of INSERT ... VALUES and INSERT ... SET.
// It can be referenced in the ON DUPLICATE KEY UPDATE clause.
// See https://dev.mysql.com/doc/refman/8.4/en/insert.html
type InsertRowAlias struct {
	Name lex.Ident
	// Columns retain qualifiers accepted by the frozen grammar; Restore keeps
	// the dialect's historical last-part column-alias spelling.
	Columns []QualifiedName
	Span    loc.Span
}

// InsertStmt is a statement to insert new rows into an existing table.
// See https://dev.mysql.com/doc/refman/5.7/en/insert.html
type InsertStmt struct {
	pkast.BaseInsertStmt

	IsReplace bool
	IgnoreErr bool
	Priority  mysql.PriorityEnum
	// Assignments is the MySQL SET input, distinct from conflict handling.
	// A non-nil empty list retains the frozen grammar's empty SET form.
	Assignments []*Assignment
	OnDuplicate []*Assignment
	// RowAlias is the optional MySQL-only row alias.
	RowAlias *InsertRowAlias
}

// Restore implements Node interface.
func (n *InsertStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Table) {
		return errors.New("restore InsertStmt: missing table")
	}
	if n.IsReplace {
		ctx.WriteKeyword("REPLACE ")
	} else {
		ctx.WriteKeyword("INSERT ")
	}
	if err := n.Priority.Restore(ctx); err != nil {
		return errors.Trace(err)
	}
	if n.Priority != mysql.NoPriority {
		ctx.WritePlain(" ")
	}
	if n.IgnoreErr {
		ctx.WriteKeyword("IGNORE ")
	}
	ctx.WriteKeyword("INTO ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore InsertStmt.Table")
	}
	if n.Columns != nil {
		ctx.WritePlain(" (")
		for i, v := range n.Columns {
			if i != 0 {
				ctx.WritePlain(",")
			}
			if !nodePresent(v) {
				return errors.Errorf("restore InsertStmt.Columns[%d]: missing target", i)
			}
			if err := v.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore InsertStmt.Columns[%d]", i)
			}
		}
		ctx.WritePlain(")")
	}
	if n.Values != nil {
		ctx.WriteKeyword(" VALUES ")
		for i, row := range n.Values {
			if i != 0 {
				ctx.WritePlain(",")
			}
			ctx.WritePlain("(")
			for j, v := range row {
				if j != 0 {
					ctx.WritePlain(",")
				}
				if !nodePresent(v) {
					return errors.Errorf("restore InsertStmt.Values[%d][%d]: missing expression", i, j)
				}
				if err := v.Restore(ctx); err != nil {
					return errors.Annotatef(err, "An error occurred while restore InsertStmt.Values[%d][%d]", i, j)
				}
			}
			ctx.WritePlain(")")
		}
	}
	if n.Query != nil {
		if !nodePresent(n.Query) {
			return errors.New("restore InsertStmt.Query: nil query")
		}
		ctx.WritePlain(" ")
		if err := n.Query.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore InsertStmt.Query")
		}
	}
	if n.Assignments != nil {
		ctx.WriteKeyword(" SET ")
		for i, v := range n.Assignments {
			if v == nil {
				return errors.Errorf("restore InsertStmt.Assignments[%d]: missing assignment", i)
			}
			if i != 0 {
				ctx.WritePlain(",")
			}
			if err := v.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore InsertStmt.Assignments[%d]", i)
			}
		}
	}
	if n.RowAlias != nil {
		ctx.WriteKeyword(" AS ")
		ctx.WriteIdent(n.RowAlias.Name)
		if len(n.RowAlias.Columns) > 0 {
			ctx.WritePlain("(")
			for i, column := range n.RowAlias.Columns {
				if i != 0 {
					ctx.WritePlain(",")
				}
				if len(column.Parts) == 0 {
					return errors.New("restore insert row alias: empty column name")
				}
				ctx.WriteIdent(column.Parts[len(column.Parts)-1])
			}
			ctx.WritePlain(")")
		}
	}
	if n.OnDuplicate != nil {
		ctx.WriteKeyword(" ON DUPLICATE KEY UPDATE ")
		for i, v := range n.OnDuplicate {
			if v == nil {
				return errors.Errorf("restore InsertStmt.OnDuplicate[%d]: missing assignment", i)
			}
			if i != 0 {
				ctx.WritePlain(",")
			}
			if err := v.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore InsertStmt.OnDuplicate[%d]", i)
			}
		}
	}
	if n.Returning != nil {
		ctx.WriteKeyword(" RETURNING ")
		if err := n.Returning.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore InsertStmt.Returning")
		}
	}

	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *InsertStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, ok := newNode.(*InsertStmt)
	if !ok || outer == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}

	n = outer
	if n.Query != nil {
		query, ok := acceptNode[QueryNode](v, n.Query)
		if !ok {
			return n, false
		}
		n.Query = query
	}

	if n.Table != nil {
		table, ok := acceptNode[pkast.TableRefNode](v, n.Table)
		if !ok {
			return n, false
		}
		n.Table = table
	}

	for i, column := range n.Columns {
		if column == nil {
			return n, false
		}
		expr, ok := acceptNode[ExprNode](v, column)
		if !ok {
			return n, false
		}
		n.Columns[i] = expr
	}
	for i, list := range n.Values {
		for j, val := range list {
			if val == nil {
				return n, false
			}
			expr, ok := acceptNode[ExprNode](v, val)
			if !ok {
				return n, false
			}
			n.Values[i][j] = expr
		}
	}
	for i, val := range n.Assignments {
		assignment, ok := acceptNode[*Assignment](v, val)
		if !ok {
			return n, false
		}
		n.Assignments[i] = assignment
	}
	for i, val := range n.OnDuplicate {
		assignment, ok := acceptNode[*Assignment](v, val)
		if !ok {
			return n, false
		}
		n.OnDuplicate[i] = assignment
	}
	if n.Returning != nil {
		node, ok := n.Returning.Accept(v)
		if !ok {
			return n, false
		}
		fields, ok := node.(*FieldList)
		if !ok || fields == nil {
			return n, false
		}
		n.Returning = fields
	}
	return v.Leave(n)
}

// Limit is the limit clause.
type Limit struct {
	pkast.BaseLimit
}

type LimitNode = pkast.LimitNode

func (n *Limit) LimitCore() *pkast.BaseLimit {
	if n == nil {
		return nil
	}
	return &n.BaseLimit
}

var _ LimitNode = (*Limit)(nil)

// Restore implements Node interface.
func (n *Limit) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Count) {
		return errors.New("restore Limit: missing count")
	}
	ctx.WriteKeyword("LIMIT ")
	if n.Offset != nil {
		if !nodePresent(n.Offset) {
			return errors.New("restore Limit.Offset: nil offset")
		}
		if err := n.Offset.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore Limit.Offset")
		}
		ctx.WritePlain(",")
	}
	if err := n.Count.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore Limit.Count")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *Limit) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	outer, ok := newNode.(*Limit)
	if !ok || outer == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(newNode)
	}
	n = outer
	if n.Count != nil {
		expr, ok := acceptNode[ExprNode](v, n.Count)
		if !ok {
			return n, false
		}
		n.Count = expr
	}
	if n.Offset != nil {
		expr, ok := acceptNode[ExprNode](v, n.Offset)
		if !ok {
			return n, false
		}
		n.Offset = expr
	}

	return v.Leave(n)
}

// ShowStmtType is the type for SHOW statement.
type ShowStmtType int

// Show statement types.
const (
	ShowNone = iota
	ShowEngines
	ShowDatabases
	ShowTables
	ShowTableStatus
	ShowColumns
	ShowWarnings
	ShowCharset
	ShowVariables
	ShowStatus
	ShowCollation
	ShowCreateTable
	ShowCreateView
	ShowCreateUser
	ShowGrants
	ShowTriggers
	ShowProcedureStatus
	ShowIndex
	ShowProcessList
	ShowCreateDatabase
	ShowEvents
	ShowStatsMeta
	ShowStatsHistograms
	ShowStatsBuckets
	ShowStatsHealthy
	ShowPlugins
	ShowProfile
	ShowProfiles
	ShowMasterStatus
	ShowPrivileges
	ShowErrors
	ShowBindings
	ShowOpenTables
	ShowCreateProcedure
	ShowCreateFunction
	ShowCreateTrigger
	ShowCreateEvent
	ShowReplicaStatus
	ShowReplicas
	ShowBinaryLogStatus
	ShowBinaryLogs
)

const (
	ProfileTypeInvalid = iota
	ProfileTypeCPU
	ProfileTypeMemory
	ProfileTypeBlockIo
	ProfileTypeContextSwitch
	ProfileTypePageFaults
	ProfileTypeIpc
	ProfileTypeSwaps
	ProfileTypeSource
	ProfileTypeAll
)

// ShowStmt is a statement to provide information about databases, tables, columns and so on.
// See https://dev.mysql.com/doc/refman/5.7/en/show.html
type ShowStmt struct {
	dmlNode

	Tp          ShowStmtType // Databases/Tables/Columns/....
	Database    lex.Ident
	Table       *TableName     // Used for showing columns.
	Column      *QualifiedName // Used for `desc table column`.
	Flag        int            // Some flag parsed from sql, such as FULL.
	Full        bool
	User        *auth.UserIdentity   // Used for show grants/create user.
	Roles       []*auth.RoleIdentity // Used for show grants .. using
	IfNotExists bool                 // Used for `show create database if not exists`

	// GlobalScope is used by `show variables`
	GlobalScope bool
	Pattern     *PatternLikeExpr
	Where       ExprNode
	Limit       *Limit // Used for SHOW WARNINGS and SHOW ERRORS.

	ShowProfileTypes []int  // Used for `SHOW PROFILE` syntax
	ShowProfileArgs  *int64 // Used for `SHOW PROFILE` syntax
	ShowProfileLimit *Limit // Used for `SHOW PROFILE` syntax
}

// Restore implements Node interface.
func (n *ShowStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("ast: nil show statement")
	}
	switch n.Tp {
	case ShowCreateTable, ShowCreateView, ShowCreateProcedure, ShowCreateFunction,
		ShowCreateTrigger, ShowCreateEvent, ShowIndex, ShowColumns:
		if n.Table == nil {
			return errors.New("ast: missing show table")
		}
	case ShowCreateDatabase:
		if !n.Database.Present() {
			return errors.New("ast: missing show database")
		}
	case ShowCreateUser:
		if n.User == nil {
			return errors.New("ast: missing show user")
		}
	}
	restoreOptFull := func() {
		if n.Full {
			ctx.WriteKeyword("FULL ")
		}
	}
	restoreShowDatabaseNameOpt := func() {
		if n.Database.Present() {
			// FROM OR IN
			ctx.WriteKeyword(" IN ")
			ctx.WriteIdent(n.Database)
		}
	}
	restoreGlobalScope := func() {
		if n.GlobalScope {
			ctx.WriteKeyword("GLOBAL ")
		} else {
			ctx.WriteKeyword("SESSION ")
		}
	}
	restoreShowLikeOrWhereOpt := func() error {
		if n.Pattern != nil {
			if !nodePresent(n.Pattern.Pattern) {
				return errors.New("ast: missing show pattern")
			}
			ctx.WriteKeyword(" LIKE ")
			if err := n.Pattern.Pattern.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore ShowStmt.Pattern")
			}
		} else if n.Where != nil {
			if !nodePresent(n.Where) {
				return errors.New("ast: nil show where expression")
			}
			ctx.WriteKeyword(" WHERE ")
			if err := n.Where.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore ShowStmt.Where")
			}
		}
		return nil
	}

	ctx.WriteKeyword("SHOW ")
	switch n.Tp {
	case ShowCreateTable:
		ctx.WriteKeyword("CREATE TABLE ")
		if err := n.Table.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore ShowStmt.Table")
		}
	case ShowCreateView:
		ctx.WriteKeyword("CREATE VIEW ")
		if err := n.Table.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore ShowStmt.VIEW")
		}
	case ShowCreateProcedure, ShowCreateFunction, ShowCreateTrigger, ShowCreateEvent:
		ctx.WriteKeyword("CREATE ")
		switch n.Tp {
		case ShowCreateProcedure:
			ctx.WriteKeyword("PROCEDURE ")
		case ShowCreateFunction:
			ctx.WriteKeyword("FUNCTION ")
		case ShowCreateTrigger:
			ctx.WriteKeyword("TRIGGER ")
		case ShowCreateEvent:
			ctx.WriteKeyword("EVENT ")
		}
		if err := n.Table.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore ShowStmt.Table")
		}
	case ShowCreateDatabase:
		ctx.WriteKeyword("CREATE DATABASE ")
		if n.IfNotExists {
			ctx.WriteKeyword("IF NOT EXISTS ")
		}
		ctx.WriteIdent(n.Database)
	case ShowCreateUser:
		ctx.WriteKeyword("CREATE USER ")
		if err := n.User.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore ShowStmt.User")
		}
	case ShowGrants:
		ctx.WriteKeyword("GRANTS")
		if n.User != nil {
			ctx.WriteKeyword(" FOR ")
			if err := n.User.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore ShowStmt.User")
			}
		}
		if n.Roles != nil {
			ctx.WriteKeyword(" USING ")
			for i, r := range n.Roles {
				if r == nil {
					return errors.New("ast: nil show role")
				}
				if err := r.Restore(ctx); err != nil {
					return errors.Annotate(err, "An error occurred while restore ShowStmt.User")
				}
				if i != len(n.Roles)-1 {
					ctx.WritePlain(", ")
				}
			}
		}
	case ShowMasterStatus:
		ctx.WriteKeyword("MASTER STATUS")
	case ShowReplicaStatus:
		ctx.WriteKeyword("REPLICA STATUS")
	case ShowReplicas:
		ctx.WriteKeyword("REPLICAS")
	case ShowBinaryLogStatus:
		ctx.WriteKeyword("BINARY LOG STATUS")
	case ShowBinaryLogs:
		ctx.WriteKeyword("BINARY LOGS")
	case ShowProcessList:
		restoreOptFull()
		ctx.WriteKeyword("PROCESSLIST")
	case ShowStatsMeta:
		ctx.WriteKeyword("STATS_META")
		if err := restoreShowLikeOrWhereOpt(); err != nil {
			return err
		}
	case ShowStatsHistograms:
		ctx.WriteKeyword("STATS_HISTOGRAMS")
		if err := restoreShowLikeOrWhereOpt(); err != nil {
			return err
		}
	case ShowStatsBuckets:
		ctx.WriteKeyword("STATS_BUCKETS")
		if err := restoreShowLikeOrWhereOpt(); err != nil {
			return err
		}
	case ShowStatsHealthy:
		ctx.WriteKeyword("STATS_HEALTHY")
		if err := restoreShowLikeOrWhereOpt(); err != nil {
			return err
		}
	case ShowProfiles:
		ctx.WriteKeyword("PROFILES")
	case ShowProfile:
		ctx.WriteKeyword("PROFILE")
		if len(n.ShowProfileTypes) > 0 {
			for i, tp := range n.ShowProfileTypes {
				if i != 0 {
					ctx.WritePlain(",")
				}
				ctx.WritePlain(" ")
				switch tp {
				case ProfileTypeCPU:
					ctx.WriteKeyword("CPU")
				case ProfileTypeMemory:
					ctx.WriteKeyword("MEMORY")
				case ProfileTypeBlockIo:
					ctx.WriteKeyword("BLOCK IO")
				case ProfileTypeContextSwitch:
					ctx.WriteKeyword("CONTEXT SWITCHES")
				case ProfileTypeIpc:
					ctx.WriteKeyword("IPC")
				case ProfileTypePageFaults:
					ctx.WriteKeyword("PAGE FAULTS")
				case ProfileTypeSource:
					ctx.WriteKeyword("SOURCE")
				case ProfileTypeSwaps:
					ctx.WriteKeyword("SWAPS")
				case ProfileTypeAll:
					ctx.WriteKeyword("ALL")
				}
			}
		}
		if n.ShowProfileArgs != nil {
			ctx.WriteKeyword(" FOR QUERY ")
			ctx.WritePlainf("%d", *n.ShowProfileArgs)
		}
		if n.ShowProfileLimit != nil {
			ctx.WritePlain(" ")
			if err := n.ShowProfileLimit.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore ShowStmt.WritePlain")
			}
		}
	case ShowPrivileges:
		ctx.WriteKeyword("PRIVILEGES")
	case ShowWarnings, ShowErrors:
		if n.Tp == ShowWarnings {
			ctx.WriteKeyword("WARNINGS")
		} else {
			ctx.WriteKeyword("ERRORS")
		}
		if n.Limit != nil {
			ctx.WritePlain(" ")
			if err := n.Limit.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore ShowStmt.Limit")
			}
		}
	// ShowTargetFilterable
	default:
		switch n.Tp {
		case ShowEngines:
			ctx.WriteKeyword("ENGINES")
		case ShowDatabases:
			ctx.WriteKeyword("DATABASES")
		case ShowCharset:
			ctx.WriteKeyword("CHARSET")
		case ShowTables:
			restoreOptFull()
			ctx.WriteKeyword("TABLES")
			restoreShowDatabaseNameOpt()
		case ShowOpenTables:
			ctx.WriteKeyword("OPEN TABLES")
			restoreShowDatabaseNameOpt()
		case ShowTableStatus:
			ctx.WriteKeyword("TABLE STATUS")
			restoreShowDatabaseNameOpt()
		case ShowIndex:
			// here can be INDEX INDEXES KEYS
			// FROM or IN
			ctx.WriteKeyword("INDEX IN ")
			if err := n.Table.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while resotre ShowStmt.Table")
			}
		case ShowColumns: // equivalent to SHOW FIELDS
			restoreOptFull()
			ctx.WriteKeyword("COLUMNS")
			if n.Table != nil {
				// FROM or IN
				ctx.WriteKeyword(" IN ")
				if err := n.Table.Restore(ctx); err != nil {
					return errors.Annotate(err, "An error occurred while resotre ShowStmt.Table")
				}
			}
			restoreShowDatabaseNameOpt()
		case ShowVariables:
			restoreGlobalScope()
			ctx.WriteKeyword("VARIABLES")
		case ShowStatus:
			restoreGlobalScope()
			ctx.WriteKeyword("STATUS")
		case ShowCollation:
			ctx.WriteKeyword("COLLATION")
		case ShowTriggers:
			ctx.WriteKeyword("TRIGGERS")
			restoreShowDatabaseNameOpt()
		case ShowProcedureStatus:
			ctx.WriteKeyword("PROCEDURE STATUS")
		case ShowEvents:
			ctx.WriteKeyword("EVENTS")
			restoreShowDatabaseNameOpt()
		case ShowPlugins:
			ctx.WriteKeyword("PLUGINS")
		default:
			return errors.New("Unknown ShowStmt type")
		}
		if err := restoreShowLikeOrWhereOpt(); err != nil {
			return err
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ShowStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*ShowStmt)
	if !ok || next == nil {
		return n, false
	}
	n = next
	if skipChildren {
		return v.Leave(n)
	}
	if n.Table != nil {
		node, ok := n.Table.Accept(v)
		if !ok {
			return n, false
		}
		table, ok := node.(*TableName)
		if !ok || table == nil {
			return n, false
		}
		n.Table = table
	}
	if n.Pattern != nil {
		node, ok := n.Pattern.Accept(v)
		if !ok {
			return n, false
		}
		pattern, ok := node.(*PatternLikeExpr)
		if !ok || pattern == nil {
			return n, false
		}
		n.Pattern = pattern
	}
	if n.Where != nil {
		if !nodePresent(n.Where) {
			return n, false
		}
		node, ok := n.Where.Accept(v)
		if !ok {
			return n, false
		}
		where, ok := node.(ExprNode)
		if !ok || !nodePresent(where) {
			return n, false
		}
		n.Where = where
	}
	if n.Limit != nil {
		node, ok := n.Limit.Accept(v)
		if !ok {
			return n, false
		}
		limit, ok := node.(*Limit)
		if !ok || limit == nil {
			return n, false
		}
		n.Limit = limit
	}
	if n.ShowProfileLimit != nil {
		node, ok := n.ShowProfileLimit.Accept(v)
		if !ok {
			return n, false
		}
		limit, ok := node.(*Limit)
		if !ok || limit == nil {
			return n, false
		}
		n.ShowProfileLimit = limit
	}
	return v.Leave(n)
}

// DeleteStmt preserves MySQL's target list and choice of multi-table syntax.
type DeleteStmt struct {
	pkast.BaseDeleteStmt
	With         *WithClause
	Targets      []*TableName
	Priority     mysql.PriorityEnum
	IgnoreErr    bool
	Quick        bool
	IsMultiTable bool
	BeforeFrom   bool
	TableHints   []*OptimizerHint
}

func (n *DeleteStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return fmt.Errorf("restore DeleteStmt: nil statement")
	}
	if n.IsMultiTable {
		if n.Table != nil || !resultSetPresent(n.Using) || len(n.Targets) == 0 {
			return fmt.Errorf("restore DeleteStmt: inconsistent multi-table mode")
		}
		for i, target := range n.Targets {
			if target == nil {
				return fmt.Errorf("restore DeleteStmt.Targets[%d]: nil target", i)
			}
		}
	} else if !resultSetPresent(n.Table) || n.Using != nil || len(n.Targets) != 0 || n.BeforeFrom {
		return fmt.Errorf("restore DeleteStmt: inconsistent single-table mode")
	}
	if n.With != nil {
		if err := n.With.Restore(ctx); err != nil {
			return fmt.Errorf("restore DeleteStmt.With: %w", err)
		}
		ctx.WritePlain(" ")
	}
	ctx.WriteKeyword("DELETE ")
	if len(n.TableHints) > 0 {
		ctx.WritePlain("/*+ ")
		for i, hint := range n.TableHints {
			if hint == nil {
				return fmt.Errorf("restore DeleteStmt.TableHints[%d]: nil hint", i)
			}
			if err := hint.Restore(ctx); err != nil {
				return fmt.Errorf("restore DeleteStmt.TableHints[%d]: %w", i, err)
			}
		}
		ctx.WritePlain("*/ ")
	}
	if err := n.Priority.Restore(ctx); err != nil {
		return err
	}
	if n.Priority != mysql.NoPriority {
		ctx.WritePlain(" ")
	}
	if n.Quick {
		ctx.WriteKeyword("QUICK ")
	}
	if n.IgnoreErr {
		ctx.WriteKeyword("IGNORE ")
	}
	if n.IsMultiTable {
		if !n.BeforeFrom {
			ctx.WriteKeyword("FROM ")
		}
		for i, target := range n.Targets {
			if i != 0 {
				ctx.WritePlain(",")
			}
			if err := target.Restore(ctx); err != nil {
				return fmt.Errorf("restore DeleteStmt.Targets[%d]: %w", i, err)
			}
		}
		if n.BeforeFrom {
			ctx.WriteKeyword(" FROM ")
		} else {
			ctx.WriteKeyword(" USING ")
		}
		if err := restoreDMLRelation(ctx, n.Using); err != nil {
			return fmt.Errorf("restore DeleteStmt.Using: %w", err)
		}
	} else {
		ctx.WriteKeyword("FROM ")
		if err := restoreDMLRelation(ctx, n.Table); err != nil {
			return fmt.Errorf("restore DeleteStmt.Table: %w", err)
		}
	}
	if n.Where != nil {
		ctx.WriteKeyword(" WHERE ")
		if err := n.Where.Restore(ctx); err != nil {
			return fmt.Errorf("restore DeleteStmt.Where: %w", err)
		}
	}
	if n.OrderBy != nil {
		if n.OrderBy.OrderByClauseCore() == nil {
			return fmt.Errorf("restore DeleteStmt.OrderBy: nil clause")
		}
		ctx.WritePlain(" ")
		if err := n.OrderBy.Restore(ctx); err != nil {
			return fmt.Errorf("restore DeleteStmt.OrderBy: %w", err)
		}
	}
	if n.Limit != nil {
		if n.Limit.LimitCore() == nil {
			return fmt.Errorf("restore DeleteStmt.Limit: nil clause")
		}
		ctx.WritePlain(" ")
		if err := n.Limit.Restore(ctx); err != nil {
			return fmt.Errorf("restore DeleteStmt.Limit: %w", err)
		}
	}
	if n.Returning != nil {
		ctx.WriteKeyword(" RETURNING ")
		if err := n.Returning.Restore(ctx); err != nil {
			return fmt.Errorf("restore DeleteStmt.Returning: %w", err)
		}
	}
	return ctx.Err()
}

func (n *DeleteStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*DeleteStmt)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	n = outer
	if n.With != nil {
		updated, ok := n.With.Accept(v)
		if !ok {
			return n, false
		}
		with, ok := updated.(*WithClause)
		if !ok || with == nil {
			return n, false
		}
		n.With = with
	}
	if !acceptOptimizerHints(v, n.TableHints) {
		return n, false
	}
	if n.Table != nil {
		if !resultSetPresent(n.Table) {
			return n, false
		}
		updated, ok := n.Table.Accept(v)
		if !ok {
			return n, false
		}
		table, ok := updated.(ResultSetNode)
		if !ok || !resultSetPresent(table) {
			return n, false
		}
		n.Table = table
	}
	if n.Using != nil {
		if !resultSetPresent(n.Using) {
			return n, false
		}
		updated, ok := n.Using.Accept(v)
		if !ok {
			return n, false
		}
		source, ok := updated.(ResultSetNode)
		if !ok || !resultSetPresent(source) {
			return n, false
		}
		n.Using = source
	}
	for i, target := range n.Targets {
		if target == nil {
			return n, false
		}
		updated, ok := target.Accept(v)
		if !ok {
			return n, false
		}
		table, ok := updated.(*TableName)
		if !ok || table == nil {
			return n, false
		}
		n.Targets[i] = table
	}
	if n.Where != nil {
		updated, ok := n.Where.Accept(v)
		if !ok {
			return n, false
		}
		expr, ok := updated.(ExprNode)
		if !ok || !nodePresent(expr) {
			return n, false
		}
		n.Where = expr
	}
	if n.OrderBy != nil {
		if n.OrderBy.OrderByClauseCore() == nil {
			return n, false
		}
		updated, ok := n.OrderBy.Accept(v)
		if !ok {
			return n, false
		}
		clause, ok := updated.(OrderByClauseNode)
		if !ok || clause == nil || clause.OrderByClauseCore() == nil {
			return n, false
		}
		n.OrderBy = clause
	}
	if n.Limit != nil {
		if n.Limit.LimitCore() == nil {
			return n, false
		}
		updated, ok := n.Limit.Accept(v)
		if !ok {
			return n, false
		}
		clause, ok := updated.(LimitNode)
		if !ok || clause == nil || clause.LimitCore() == nil {
			return n, false
		}
		n.Limit = clause
	}
	if n.Returning != nil {
		updated, ok := n.Returning.Accept(v)
		if !ok {
			return n, false
		}
		fields, ok := updated.(*FieldList)
		if !ok || fields == nil {
			return n, false
		}
		n.Returning = fields
	}
	return v.Leave(n)
}

// UpdateStmt keeps MySQL modifiers outside the shared update syntax.
type UpdateStmt struct {
	pkast.BaseUpdateStmt
	With          *WithClause
	Priority      mysql.PriorityEnum
	IgnoreErr     bool
	MultipleTable bool
	TableHints    []*OptimizerHint
}

func (n *UpdateStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("UpdateStmt")
	}
	if n == nil || !resultSetPresent(n.Table) || len(n.Assignments) == 0 {
		return fmt.Errorf("restore UpdateStmt: missing table or assignments")
	}
	for i, assignment := range n.Assignments {
		if assignment == nil {
			return fmt.Errorf("restore UpdateStmt.Assignments[%d]: nil assignment", i)
		}
	}
	if n.With != nil {
		if err := n.With.Restore(ctx); err != nil {
			return fmt.Errorf("restore UpdateStmt.With: %w", err)
		}
		ctx.WritePlain(" ")
	}
	ctx.WriteKeyword("UPDATE ")
	if len(n.TableHints) > 0 {
		ctx.WritePlain("/*+ ")
		for i, hint := range n.TableHints {
			if hint == nil {
				return fmt.Errorf("restore UpdateStmt.TableHints[%d]: nil hint", i)
			}
			if err := hint.Restore(ctx); err != nil {
				return fmt.Errorf("restore UpdateStmt.TableHints[%d]: %w", i, err)
			}
		}
		ctx.WritePlain("*/ ")
	}
	if err := n.Priority.Restore(ctx); err != nil {
		return err
	}
	if n.Priority != mysql.NoPriority {
		ctx.WritePlain(" ")
	}
	if n.IgnoreErr {
		ctx.WriteKeyword("IGNORE ")
	}
	if err := restoreDMLRelation(ctx, n.Table); err != nil {
		return fmt.Errorf("restore UpdateStmt.Table: %w", err)
	}
	ctx.WriteKeyword(" SET ")
	for i, assignment := range n.Assignments {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := assignment.Restore(ctx); err != nil {
			return fmt.Errorf("restore UpdateStmt.Assignments[%d]: %w", i, err)
		}
	}
	if n.Where != nil {
		ctx.WriteKeyword(" WHERE ")
		if err := n.Where.Restore(ctx); err != nil {
			return fmt.Errorf("restore UpdateStmt.Where: %w", err)
		}
	}
	if n.OrderBy != nil {
		if n.OrderBy.OrderByClauseCore() == nil {
			return fmt.Errorf("restore UpdateStmt.OrderBy: nil clause")
		}
		ctx.WritePlain(" ")
		if err := n.OrderBy.Restore(ctx); err != nil {
			return fmt.Errorf("restore UpdateStmt.OrderBy: %w", err)
		}
	}
	if n.Limit != nil {
		if n.Limit.LimitCore() == nil {
			return fmt.Errorf("restore UpdateStmt.Limit: nil clause")
		}
		ctx.WritePlain(" ")
		if err := n.Limit.Restore(ctx); err != nil {
			return fmt.Errorf("restore UpdateStmt.Limit: %w", err)
		}
	}
	if n.Returning != nil {
		ctx.WriteKeyword(" RETURNING ")
		if err := n.Returning.Restore(ctx); err != nil {
			return fmt.Errorf("restore UpdateStmt.Returning: %w", err)
		}
	}
	return ctx.Err()
}

func (n *UpdateStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*UpdateStmt)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	n = outer
	if n.With != nil {
		updated, ok := n.With.Accept(v)
		if !ok {
			return n, false
		}
		with, ok := updated.(*WithClause)
		if !ok || with == nil {
			return n, false
		}
		n.With = with
	}
	if !acceptOptimizerHints(v, n.TableHints) {
		return n, false
	}
	if !resultSetPresent(n.Table) {
		return n, false
	}
	updated, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	table, ok := updated.(ResultSetNode)
	if !ok || !resultSetPresent(table) {
		return n, false
	}
	n.Table = table
	for i, assignment := range n.Assignments {
		if assignment == nil {
			return n, false
		}
		updated, ok := assignment.Accept(v)
		if !ok {
			return n, false
		}
		value, ok := updated.(*Assignment)
		if !ok || value == nil {
			return n, false
		}
		n.Assignments[i] = value
	}
	if n.Where != nil {
		updated, ok := n.Where.Accept(v)
		if !ok {
			return n, false
		}
		expr, ok := updated.(ExprNode)
		if !ok || !nodePresent(expr) {
			return n, false
		}
		n.Where = expr
	}
	if n.OrderBy != nil {
		if n.OrderBy.OrderByClauseCore() == nil {
			return n, false
		}
		updated, ok := n.OrderBy.Accept(v)
		if !ok {
			return n, false
		}
		clause, ok := updated.(OrderByClauseNode)
		if !ok || clause == nil || clause.OrderByClauseCore() == nil {
			return n, false
		}
		n.OrderBy = clause
	}
	if n.Limit != nil {
		if n.Limit.LimitCore() == nil {
			return n, false
		}
		updated, ok := n.Limit.Accept(v)
		if !ok {
			return n, false
		}
		clause, ok := updated.(LimitNode)
		if !ok || clause == nil || clause.LimitCore() == nil {
			return n, false
		}
		n.Limit = clause
	}
	if n.Returning != nil {
		updated, ok := n.Returning.Accept(v)
		if !ok {
			return n, false
		}
		fields, ok := updated.(*FieldList)
		if !ok || fields == nil {
			return n, false
		}
		n.Returning = fields
	}
	return v.Leave(n)
}

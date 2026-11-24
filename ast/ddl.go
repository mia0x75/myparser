package ast

import (
	"strconv"

	"github.com/mia0x75/myparser/auth"
	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/model"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

var (
	_ DDLNode = &AlterTableStmt{}
	_ DDLNode = &AlterDatabaseStmt{}
	_ DDLNode = &CreateDatabaseStmt{}
	_ DDLNode = &CreateIndexStmt{}
	_ DDLNode = &CreateTableStmt{}
	_ DDLNode = &CreateViewStmt{}
	_ DDLNode = &AlterViewStmt{}
	_ DDLNode = &CreateFunctionStmt{}
	_ DDLNode = &CreateProcedureStmt{}
	_ DDLNode = &CreateTriggerStmt{}
	_ DDLNode = &CreateEventStmt{}
	_ DDLNode = &AlterFunctionStmt{}
	_ DDLNode = &AlterProcedureStmt{}
	_ DDLNode = &AlterEventStmt{}
	_ DDLNode = &DropDatabaseStmt{}
	_ DDLNode = &DropIndexStmt{}
	_ DDLNode = &DropTableStmt{}
	_ DDLNode = &DropFunctionStmt{}
	_ DDLNode = &DropProcedureStmt{}
	_ DDLNode = &DropTriggerStmt{}
	_ DDLNode = &DropEventStmt{}
	_ DDLNode = &RenameTableStmt{}
	_ DDLNode = &TruncateTableStmt{}

	_ Node                   = &AlterTableSpec{}
	_ Node                   = &PeriodDefinition{}
	_ Node                   = &PartitionOptions{}
	_ Node                   = &PartitionDefinition{}
	_ pkast.ColumnDefNode    = &ColumnDef{}
	_ Node                   = &ColumnOption{}
	_ Node                   = &ColumnPosition{}
	_ Node                   = &Constraint{}
	_ pkast.IndexKeyPartNode = (*IndexKeyPart)(nil)
	_ pkast.ReferenceDefNode = (*ReferenceDef)(nil)
)

// CharsetOpt is used for parsing charset option from SQL.
type CharsetOpt struct {
	Chs string
	Col string
}

// DatabaseOptionType is the type for database options.
type DatabaseOptionType int

// Database option types.
const (
	DatabaseOptionNone DatabaseOptionType = iota
	DatabaseOptionCharset
	DatabaseOptionCollate
)

// DatabaseOption represents database option.
type DatabaseOption struct {
	Tp    DatabaseOptionType
	Value string
}

// Restore implements Node interface.
func (n *DatabaseOption) Restore(ctx *format.RestoreCtx) error {
	switch n.Tp {
	case DatabaseOptionCharset:
		ctx.WriteKeyword("CHARACTER SET")
		ctx.WritePlain(" = ")
		ctx.WritePlain(n.Value)
	case DatabaseOptionCollate:
		ctx.WriteKeyword("COLLATE")
		ctx.WritePlain(" = ")
		ctx.WritePlain(n.Value)
	default:
		return errors.Errorf("invalid DatabaseOptionType: %d", n.Tp)
	}
	return ctx.Err()
}

// CreateDatabaseStmt is a statement to create a database.
// See https://dev.mysql.com/doc/refman/5.7/en/create-database.html
type CreateDatabaseStmt struct {
	ddlNode

	IfNotExists bool
	Name        lex.Ident
	Options     []*DatabaseOption
}

// Restore implements Node interface.
func (n *CreateDatabaseStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("CREATE DATABASE ")
	if n.IfNotExists {
		ctx.WriteKeyword("IF NOT EXISTS ")
	}
	ctx.WriteIdent(n.Name)
	for i, option := range n.Options {
		if option == nil {
			return errors.Errorf("restore create database: nil option at %d", i)
		}
		ctx.WritePlain(" ")
		err := option.Restore(ctx)
		if err != nil {
			return errors.Annotatef(err, "An error occurred while splicing CreateDatabaseStmt DatabaseOption: [%v]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateDatabaseStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := newNode.(*CreateDatabaseStmt)
	if !ok || stmt == nil {
		return newNode, false
	}
	n = stmt
	if skipChildren {
		return v.Leave(n)
	}
	return v.Leave(n)
}

// AlterDatabaseStmt is a statement to alter a database.
// See https://dev.mysql.com/doc/refman/5.7/en/alter-database.html
type AlterDatabaseStmt struct {
	ddlNode

	Name                 lex.Ident
	AlterDefaultDatabase bool
	Options              []*DatabaseOption
}

// Restore implements Node interface.
func (n *AlterDatabaseStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("ALTER DATABASE")
	if !n.AlterDefaultDatabase {
		ctx.WritePlain(" ")
		ctx.WriteIdent(n.Name)
	}
	for i, option := range n.Options {
		if option == nil {
			return errors.Errorf("restore alter database: nil option at %d", i)
		}
		ctx.WritePlain(" ")
		err := option.Restore(ctx)
		if err != nil {
			return errors.Annotatef(err, "An error occurred while splicing AlterDatabaseStmt DatabaseOption: [%v]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *AlterDatabaseStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := newNode.(*AlterDatabaseStmt)
	if !ok || stmt == nil {
		return newNode, false
	}
	n = stmt
	if skipChildren {
		return v.Leave(n)
	}
	return v.Leave(n)
}

// DropDatabaseStmt is a statement to drop a database and all tables in the database.
// See https://dev.mysql.com/doc/refman/5.7/en/drop-database.html
type DropDatabaseStmt struct {
	ddlNode

	IfExists bool
	Name     lex.Ident
}

// Restore implements Node interface.
func (n *DropDatabaseStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("DROP DATABASE ")
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}
	ctx.WriteIdent(n.Name)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropDatabaseStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := newNode.(*DropDatabaseStmt)
	if !ok || stmt == nil {
		return newNode, false
	}
	n = stmt
	if skipChildren {
		return v.Leave(n)
	}
	return v.Leave(n)
}

// IndexKeyPart preserves a column key or a parenthesized expression key.
type IndexKeyPart struct {
	pkast.BaseIndexKeyPart
}

func (n *IndexKeyPart) IndexKeyPartCore() *pkast.BaseIndexKeyPart {
	if n == nil {
		return nil
	}
	return &n.BaseIndexKeyPart
}

// Restore implements Node interface.
func (n *IndexKeyPart) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore index key part: nil part")
	}
	if n.Expr != nil && (len(n.Column.Parts) != 0 || n.PrefixLength > 0) {
		return errors.New("restore index key part: expression conflicts with column or prefix length")
	}
	if n.Expr != nil {
		ctx.WritePlain("(")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotate(err, "restore IndexKeyPart expression")
		}
		ctx.WritePlain(")")
	} else {
		if err := restoreQualifiedName(ctx, n.Column); err != nil {
			return errors.Annotate(err, "restore IndexKeyPart column")
		}
		if n.PrefixLength > 0 {
			ctx.WritePlainf("(%d)", n.PrefixLength)
		}
	}
	if n.Desc {
		ctx.WriteKeyword(" DESC")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *IndexKeyPart) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	part, ok := newNode.(*IndexKeyPart)
	if !ok || part == nil {
		return newNode, false
	}
	n = part
	if skipChildren {
		return v.Leave(n)
	}
	if n.Expr != nil {
		node, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		expr, ok := node.(ExprNode)
		if !ok || expr == nil {
			return n, false
		}
		n.Expr = expr
	}
	return v.Leave(n)
}

// ReferenceDef is used for parsing foreign key reference option from SQL.
// See http://dev.mysql.com/doc/refman/5.7/en/create-table-foreign-keys.html
type ReferenceDef struct {
	pkast.BaseReferenceDef
	OnDelete *OnDeleteOpt
	OnUpdate *OnUpdateOpt
}

func (n *ReferenceDef) ReferenceDefCore() *pkast.BaseReferenceDef {
	if n == nil {
		return nil
	}
	return &n.BaseReferenceDef
}

// Restore implements Node interface.
func (n *ReferenceDef) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Table == nil || n.Table.TableRefCore() == nil {
		return errors.New("restore reference definition: missing table")
	}
	if len(n.Columns) == 0 {
		return errors.New("restore reference definition: missing columns")
	}
	ctx.WriteKeyword("REFERENCES ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while splicing ReferenceDef")
	}
	ctx.WritePlain("(")
	for i, column := range n.Columns {
		if column == nil || column.IndexKeyPartCore() == nil {
			return errors.Errorf("restore reference definition: nil column at %d", i)
		}
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := column.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restoring referenced column: [%v]", i)
		}
	}
	ctx.WritePlain(")")
	if n.OnDelete != nil && n.OnDelete.ReferOpt != ReferOptionNoOption {
		ctx.WritePlain(" ")
		if err := n.OnDelete.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing OnDelete")
		}
	}
	if n.OnUpdate != nil && n.OnUpdate.ReferOpt != ReferOptionNoOption {
		ctx.WritePlain(" ")
		if err := n.OnUpdate.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing OnUpdate")
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ReferenceDef) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	ref, ok := newNode.(*ReferenceDef)
	if !ok || ref == nil {
		return newNode, false
	}
	n = ref
	if skipChildren {
		return v.Leave(n)
	}
	if n.Table == nil || n.Table.TableRefCore() == nil {
		return n, false
	}
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	table, ok := node.(pkast.TableRefNode)
	if !ok || table == nil || table.TableRefCore() == nil {
		return n, false
	}
	n.Table = table
	for i, val := range n.Columns {
		if val == nil || val.IndexKeyPartCore() == nil {
			return n, false
		}
		node, ok = val.Accept(v)
		if !ok {
			return n, false
		}
		column, ok := node.(pkast.IndexKeyPartNode)
		if !ok || column == nil || column.IndexKeyPartCore() == nil {
			return n, false
		}
		n.Columns[i] = column
	}
	if n.OnDelete != nil {
		updated, ok := n.OnDelete.Accept(v)
		if !ok {
			return n, false
		}
		action, ok := updated.(*OnDeleteOpt)
		if !ok || action == nil {
			return n, false
		}
		n.OnDelete = action
	}
	if n.OnUpdate != nil {
		updated, ok := n.OnUpdate.Accept(v)
		if !ok {
			return n, false
		}
		action, ok := updated.(*OnUpdateOpt)
		if !ok || action == nil {
			return n, false
		}
		n.OnUpdate = action
	}
	return v.Leave(n)
}

// ReferOptionType is the type for refer options.
type ReferOptionType int

// Refer option types.
const (
	ReferOptionNoOption ReferOptionType = iota
	ReferOptionRestrict
	ReferOptionCascade
	ReferOptionSetNull
	ReferOptionNoAction
)

// String implements fmt.Stringer interface.
func (r ReferOptionType) String() string {
	switch r {
	case ReferOptionRestrict:
		return "RESTRICT"
	case ReferOptionCascade:
		return "CASCADE"
	case ReferOptionSetNull:
		return "SET NULL"
	case ReferOptionNoAction:
		return "NO ACTION"
	}
	return ""
}

// OnDeleteOpt is used for optional on delete clause.
type OnDeleteOpt struct {
	node
	ReferOpt ReferOptionType
}

// Restore implements Node interface.
func (n *OnDeleteOpt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.ReferOpt != ReferOptionNoOption && n.ReferOpt.String() == "" {
		return errors.New("restore ON DELETE: invalid action")
	}
	if n.ReferOpt != ReferOptionNoOption {
		ctx.WriteKeyword("ON DELETE ")
		ctx.WriteKeyword(n.ReferOpt.String())
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *OnDeleteOpt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, _ := v.Enter(n)
	action, ok := newNode.(*OnDeleteOpt)
	if !ok || action == nil {
		return newNode, false
	}
	return v.Leave(action)
}

// OnUpdateOpt is used for optional on update clause.
type OnUpdateOpt struct {
	node
	ReferOpt ReferOptionType
}

// Restore implements Node interface.
func (n *OnUpdateOpt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.ReferOpt != ReferOptionNoOption && n.ReferOpt.String() == "" {
		return errors.New("restore ON UPDATE: invalid action")
	}
	if n.ReferOpt != ReferOptionNoOption {
		ctx.WriteKeyword("ON UPDATE ")
		ctx.WriteKeyword(n.ReferOpt.String())
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *OnUpdateOpt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, _ := v.Enter(n)
	action, ok := newNode.(*OnUpdateOpt)
	if !ok || action == nil {
		return newNode, false
	}
	return v.Leave(action)
}

// ColumnOptionType is the type for ColumnOption.
type ColumnOptionType int

// ColumnOption types.
const (
	ColumnOptionNoOption ColumnOptionType = iota
	ColumnOptionPrimaryKey
	ColumnOptionNotNull
	ColumnOptionAutoIncrement
	ColumnOptionDefaultValue
	ColumnOptionUniqKey
	ColumnOptionNull
	ColumnOptionOnUpdate // For Timestamp and Datetime only.
	ColumnOptionFulltext
	ColumnOptionComment
	ColumnOptionGenerated
	ColumnOptionReference
	ColumnOptionCollate
	ColumnOptionCheck
	ColumnOptionVisible
	ColumnOptionInvisible
	ColumnOptionSRID
)

var (
	invalidOptionForGeneratedColumn = map[ColumnOptionType]struct{}{
		ColumnOptionAutoIncrement: {},
		ColumnOptionOnUpdate:      {},
		ColumnOptionDefaultValue:  {},
	}
)

// ColumnOption is used for parsing column constraint info from SQL.
type ColumnOption struct {
	node

	Tp ColumnOptionType
	// Expr is used for ColumnOptionDefaultValue/ColumnOptionOnUpdate,
	// ColumnOptionGenerated, and ColumnOptionSRID.
	// For ColumnOptionDefaultValue or ColumnOptionOnUpdate, it's the target value.
	// For ColumnOptionGenerated, it's the target expression.
	Expr ExprNode
	// Stored is only for ColumnOptionGenerated, default is false.
	Stored bool
	// Refer is used for foreign key.
	Refer    pkast.ReferenceDefNode
	StrValue string
}

// Restore implements Node interface.
func (n *ColumnOption) Restore(ctx *format.RestoreCtx) error {
	switch n.Tp {
	case ColumnOptionNoOption:
		return ctx.Err()
	case ColumnOptionPrimaryKey:
		ctx.WriteKeyword("PRIMARY KEY")
	case ColumnOptionNotNull:
		ctx.WriteKeyword("NOT NULL")
	case ColumnOptionAutoIncrement:
		ctx.WriteKeyword("AUTO_INCREMENT")
	case ColumnOptionDefaultValue:
		ctx.WriteKeyword("DEFAULT ")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing ColumnOption DefaultValue Expr")
		}
	case ColumnOptionUniqKey:
		ctx.WriteKeyword("UNIQUE KEY")
	case ColumnOptionNull:
		ctx.WriteKeyword("NULL")
	case ColumnOptionOnUpdate:
		ctx.WriteKeyword("ON UPDATE ")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing ColumnOption ON UPDATE Expr")
		}
	case ColumnOptionFulltext:
		return errors.New("Parser ignore the `ColumnOptionFulltext` type now")
	case ColumnOptionComment:
		ctx.WriteKeyword("COMMENT ")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing ColumnOption COMMENT Expr")
		}
	case ColumnOptionGenerated:
		ctx.WriteKeyword("GENERATED ALWAYS AS")
		ctx.WritePlain("(")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing ColumnOption GENERATED ALWAYS Expr")
		}
		ctx.WritePlain(")")
		if n.Stored {
			ctx.WriteKeyword(" STORED")
		} else {
			ctx.WriteKeyword(" VIRTUAL")
		}
	case ColumnOptionReference:
		if n.Refer == nil || n.Refer.ReferenceDefCore() == nil {
			return errors.New("restore column option: missing REFERENCES clause")
		}
		if err := n.Refer.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing ColumnOption ReferenceDef")
		}
	case ColumnOptionCollate:
		if n.StrValue == "" {
			return errors.New("Empty ColumnOption COLLATE")
		}
		ctx.WriteKeyword("COLLATE ")
		ctx.WritePlain(n.StrValue)
	case ColumnOptionCheck:
		ctx.WriteKeyword("CHECK (")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing ColumnOption CHECK Expr")
		}
		ctx.WritePlain(")")
	case ColumnOptionVisible:
		ctx.WriteKeyword("VISIBLE")
	case ColumnOptionInvisible:
		ctx.WriteKeyword("INVISIBLE")
	case ColumnOptionSRID:
		ctx.WriteKeyword("SRID ")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing ColumnOption SRID Expr")
		}
	default:
		return errors.New("An error occurred while splicing ColumnOption")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ColumnOption) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	option, ok := newNode.(*ColumnOption)
	if !ok || option == nil {
		return newNode, false
	}
	n = option
	if skipChildren {
		return v.Leave(n)
	}
	if n.Expr != nil {
		if !nodePresent(n.Expr) {
			return n, false
		}
		node, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		expr, ok := node.(ExprNode)
		if !ok || !nodePresent(expr) {
			return n, false
		}
		n.Expr = expr
	}
	if n.Refer != nil {
		if n.Refer.ReferenceDefCore() == nil {
			return n, false
		}
		node, ok := n.Refer.Accept(v)
		if !ok {
			return n, false
		}
		ref, ok := node.(pkast.ReferenceDefNode)
		if !ok || ref == nil || ref.ReferenceDefCore() == nil {
			return n, false
		}
		n.Refer = ref
	}
	return v.Leave(n)
}

// IndexOption is the index options.
//
//	  KEY_BLOCK_SIZE [=] value
//	| index_type
//	| WITH PARSER parser_name
//	| COMMENT 'string'
//
// See http://dev.mysql.com/doc/refman/5.7/en/create-table.html
type IndexOption struct {
	node

	KeyBlockSize uint64
	Tp           model.IndexType
	Comment      string
}

// Restore implements Node interface.
func (n *IndexOption) Restore(ctx *format.RestoreCtx) error {
	hasPrevOption := false
	if n.KeyBlockSize > 0 {
		ctx.WriteKeyword("KEY_BLOCK_SIZE")
		ctx.WritePlainf("=%d", n.KeyBlockSize)
		hasPrevOption = true
	}

	if n.Tp != model.IndexTypeInvalid {
		if hasPrevOption {
			ctx.WritePlain(" ")
		}
		ctx.WriteKeyword("USING ")
		ctx.WritePlain(n.Tp.String())
		hasPrevOption = true
	}

	if n.Comment != "" {
		if hasPrevOption {
			ctx.WritePlain(" ")
		}
		ctx.WriteKeyword("COMMENT ")
		ctx.WriteString(n.Comment)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *IndexOption) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	option, ok := newNode.(*IndexOption)
	if !ok || option == nil {
		return newNode, false
	}
	n = option
	if skipChildren {
		return v.Leave(n)
	}
	return v.Leave(n)
}

// ColumnDef is used for parsing column definition from SQL.
type ColumnDef struct {
	pkast.BaseColumnDef
	Options []*ColumnOption
}

// ColumnDefCore exposes the shared column declaration.
func (n *ColumnDef) ColumnDefCore() *pkast.BaseColumnDef {
	if n == nil {
		return nil
	}
	return &n.BaseColumnDef
}

// Restore implements Node interface.
func (n *ColumnDef) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore ColumnDef: nil column")
	}
	if err := restoreQualifiedName(ctx, n.Name); err != nil {
		return errors.Annotate(err, "An error occurred while splicing ColumnDef Name")
	}
	if n.Type != nil {
		if n.Type.TypeCore() == nil {
			return errors.New("restore ColumnDef: nil type")
		}
		ctx.WritePlain(" ")
		if err := n.Type.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing ColumnDef Type")
		}
	}
	for i, options := range n.Options {
		if options == nil {
			return errors.Errorf("restore ColumnDef: nil option at %d", i)
		}
		ctx.WritePlain(" ")
		if err := options.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while splicing ColumnDef ColumnOption: [%v]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ColumnDef) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	column, ok := newNode.(*ColumnDef)
	if !ok || column == nil {
		return newNode, false
	}
	n = column
	if skipChildren {
		return v.Leave(n)
	}
	if n.Type != nil {
		if n.Type.TypeCore() == nil {
			return n, false
		}
		node, ok := n.Type.Accept(v)
		if !ok {
			return n, false
		}
		typ, ok := node.(pkast.TypeNode)
		if !ok || typ == nil || typ.TypeCore() == nil {
			return n, false
		}
		n.Type = typ
	}
	for i, val := range n.Options {
		if val == nil {
			return n, false
		}
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		option, ok := node.(*ColumnOption)
		if !ok || option == nil {
			return n, false
		}
		n.Options[i] = option
	}
	return v.Leave(n)
}

// Validate checks if a column definition is legal.
// For example, generated column definitions that contain such
// column options as `ON UPDATE`, `AUTO_INCREMENT`, `DEFAULT`
// are illegal.
func (n *ColumnDef) Validate() bool {
	generatedCol := false
	illegalOpt4gc := false
	collateCnt := 0
	for _, opt := range n.Options {
		if opt.Tp == ColumnOptionGenerated {
			generatedCol = true
		}
		if opt.Tp == ColumnOptionCollate {
			collateCnt++
		}
		_, found := invalidOptionForGeneratedColumn[opt.Tp]
		illegalOpt4gc = illegalOpt4gc || found
	}
	// Both MySQL and MariaDB reject a generated column that carries more than
	// one COLLATE option. A duplicate COLLATE on a plain column is accepted by
	// MariaDB, so it must stay accepted for the shared dialect surface.
	if generatedCol && collateCnt > 1 {
		return false
	}
	return !generatedCol || !illegalOpt4gc
}

// PeriodDefinition declares MariaDB application-time or system-time columns.
type PeriodDefinition struct {
	node
	Name  lex.Ident
	Start QualifiedName
	End   QualifiedName
}

// Restore implements Node interface.
func (n *PeriodDefinition) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("PERIOD FOR ")
	ctx.WriteIdent(n.Name)
	ctx.WritePlain(" (")
	if err := restoreQualifiedName(ctx, n.Start); err != nil {
		return errors.Annotate(err, "An error occurred while restore PeriodDefinition.Start")
	}
	ctx.WritePlain(",")
	if err := restoreQualifiedName(ctx, n.End); err != nil {
		return errors.Annotate(err, "An error occurred while restore PeriodDefinition.End")
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *PeriodDefinition) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	period, ok := newNode.(*PeriodDefinition)
	if !ok || period == nil {
		return newNode, false
	}
	n = period
	if skipChildren {
		return v.Leave(n)
	}
	return v.Leave(n)
}

// CreateTableStmt is a statement to create a table.
// See https://dev.mysql.com/doc/refman/5.7/en/create-table.html
type CreateTableStmt struct {
	pkast.BaseCreateTableStmt

	OrReplace   bool
	ReferTable  *TableName
	Periods     []*PeriodDefinition
	Options     []*TableOption
	Partition   *PartitionOptions
	OnDuplicate OnDuplicateKeyHandlingType
}

// Restore implements Node interface.
func (n *CreateTableStmt) Restore(ctx *format.RestoreCtx) error {
	if ctx == nil {
		return errors.New("restore CreateTableStmt: nil context")
	}
	if n == nil || n.Table == nil || n.Table.TableRefCore() == nil {
		return errors.New("restore CreateTableStmt: missing table")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	ctx.WriteKeyword("CREATE ")
	if n.OrReplace {
		ctx.WriteKeyword("OR REPLACE ")
	}
	if n.Temporary {
		ctx.WriteKeyword("TEMPORARY ")
	}
	ctx.WriteKeyword("TABLE ")
	if n.IfNotExists {
		ctx.WriteKeyword("IF NOT EXISTS ")
	}
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while splicing CreateTableStmt Table")
	}
	ctx.WritePlain(" ")
	if n.ReferTable != nil {
		ctx.WriteKeyword("LIKE ")
		if err := n.ReferTable.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing CreateTableStmt ReferTable")
		}
	}

	lenCols := len(n.Columns)
	lenConstraints := len(n.Constraints)
	if lenCols+lenConstraints+len(n.Periods) > 0 {
		ctx.WritePlain("(")
		for i, col := range n.Columns {
			if col == nil || col.ColumnDefCore() == nil {
				return errors.Errorf("restore CreateTableStmt: nil column at %d", i)
			}
			if i > 0 {
				ctx.WritePlain(",")
			}
			if err := col.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while splicing CreateTableStmt ColumnDef: [%v]", i)
			}
		}
		for i, constraint := range n.Constraints {
			if constraint == nil || constraint.ConstraintCore() == nil {
				return errors.Errorf("restore create table: nil constraint at %d", i)
			}
			if i > 0 || lenCols >= 1 {
				ctx.WritePlain(",")
			}
			if err := constraint.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while splicing CreateTableStmt Constraints: [%v]", i)
			}
		}
		for i, period := range n.Periods {
			if period == nil {
				return errors.Errorf("restore create table: nil period at %d", i)
			}
			if i > 0 || lenCols+lenConstraints > 0 {
				ctx.WritePlain(",")
			}
			if err := period.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while splicing CreateTableStmt Periods: [%v]", i)
			}
		}
		ctx.WritePlain(")")
	}

	for i, option := range n.Options {
		if option == nil {
			return errors.Errorf("restore CreateTableStmt: nil table option at %d", i)
		}
		ctx.WritePlain(" ")
		if err := option.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while splicing CreateTableStmt TableOption: [%v]", i)
		}
	}

	if n.Partition != nil {
		ctx.WritePlain(" ")
		if err := n.Partition.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing CreateTableStmt Partition")
		}
	}

	if n.Query != nil {
		if !resultSetPresent(n.Query) {
			return errors.New("restore CreateTableStmt: nil query")
		}
		switch n.OnDuplicate {
		case OnDuplicateKeyHandlingError:
			ctx.WriteKeyword(" AS ")
		case OnDuplicateKeyHandlingIgnore:
			ctx.WriteKeyword(" IGNORE AS ")
		case OnDuplicateKeyHandlingReplace:
			ctx.WriteKeyword(" REPLACE AS ")
		}

		if err := n.Query.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore CreateTableStmt.Query")
		}
	}

	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateTableStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := newNode.(*CreateTableStmt)
	if !ok || stmt == nil {
		return newNode, false
	}
	n = stmt
	if skipChildren {
		return v.Leave(n)
	}
	if n.Table == nil || n.Table.TableRefCore() == nil {
		return n, false
	}
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	table, ok := node.(pkast.TableRefNode)
	if !ok || table == nil || table.TableRefCore() == nil {
		return n, false
	}
	n.Table = table
	if n.ReferTable != nil {
		node, ok = n.ReferTable.Accept(v)
		if !ok {
			return n, false
		}
		referTable, ok := node.(*TableName)
		if !ok || referTable == nil {
			return n, false
		}
		n.ReferTable = referTable
	}
	for i, val := range n.Columns {
		if val == nil || val.ColumnDefCore() == nil {
			return n, false
		}
		node, ok = val.Accept(v)
		if !ok {
			return n, false
		}
		col, ok := node.(pkast.ColumnDefNode)
		if !ok || col == nil || col.ColumnDefCore() == nil {
			return n, false
		}
		n.Columns[i] = col
	}
	for i, val := range n.Constraints {
		if val == nil || val.ConstraintCore() == nil {
			return n, false
		}
		node, ok = val.Accept(v)
		if !ok {
			return n, false
		}
		constraint, ok := node.(pkast.ConstraintNode)
		if !ok || constraint == nil || constraint.ConstraintCore() == nil {
			return n, false
		}
		n.Constraints[i] = constraint
	}
	for i, val := range n.Periods {
		if val == nil {
			return n, false
		}
		node, ok = val.Accept(v)
		if !ok {
			return n, false
		}
		period, ok := node.(*PeriodDefinition)
		if !ok || period == nil {
			return n, false
		}
		n.Periods[i] = period
	}
	if !acceptTableOptions(v, n.Options) {
		return n, false
	}
	if n.Partition != nil {
		node, ok = n.Partition.Accept(v)
		if !ok {
			return n, false
		}
		partition, ok := node.(*PartitionOptions)
		if !ok || partition == nil {
			return n, false
		}
		n.Partition = partition
	}
	if n.Query != nil && !visitQueryOperand(v, &n.Query) {
		return n, false
	}

	return v.Leave(n)
}

// DropTableStmt is a statement to drop one or more tables.
// See https://dev.mysql.com/doc/refman/5.7/en/drop-table.html
type DropTableStmt struct {
	ddlNode

	IfExists  bool
	Tables    []*TableName
	IsView    bool
	Temporary bool
}

// Restore implements Node interface.
func (n *DropTableStmt) Restore(ctx *format.RestoreCtx) error {
	if ctx == nil {
		return errors.New("restore drop table: nil context")
	}
	if n == nil {
		return errors.New("restore drop table: nil statement")
	}
	if n.IsView {
		ctx.WriteKeyword("DROP VIEW ")
	} else {
		ctx.WriteKeyword("DROP ")
		if n.Temporary {
			ctx.WriteKeyword("TEMPORARY ")
		}
		ctx.WriteKeyword("TABLE ")
	}
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}

	for index, table := range n.Tables {
		if table == nil {
			return errors.Errorf("restore drop table: nil table at %d", index)
		}
		if index != 0 {
			ctx.WritePlain(", ")
		}
		if err := table.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore DropTableStmt.Tables "+strconv.Itoa(index))
		}
	}

	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropTableStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := newNode.(*DropTableStmt)
	if !ok || stmt == nil {
		return newNode, false
	}
	n = stmt
	if skipChildren {
		return v.Leave(n)
	}
	for i, val := range n.Tables {
		if val == nil {
			return n, false
		}
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		table, ok := node.(*TableName)
		if !ok || table == nil {
			return n, false
		}
		n.Tables[i] = table
	}
	return v.Leave(n)
}

// DropFunctionStmt is a statement to drop a stored function.
// See https://dev.mysql.com/doc/refman/8.0/en/drop-function.html
type DropFunctionStmt struct {
	ddlNode

	IfExists bool
	Name     *TableName
}

// Restore implements Node interface.
func (n *DropFunctionStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Name == nil {
		return errors.New("restore drop function: missing name")
	}
	ctx.WriteKeyword("DROP FUNCTION ")
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}
	if err := n.Name.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore DropFunctionStmt.Name")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropFunctionStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := newNode.(*DropFunctionStmt)
	if !ok || stmt == nil {
		return newNode, false
	}
	n = stmt
	if skipChildren {
		return v.Leave(n)
	}
	if n.Name == nil {
		return n, false
	}
	node, ok := n.Name.Accept(v)
	if !ok {
		return n, false
	}
	name, ok := node.(*TableName)
	if !ok || name == nil {
		return n, false
	}
	n.Name = name
	return v.Leave(n)
}

// DropProcedureStmt is a statement to drop a stored procedure.
// See https://dev.mysql.com/doc/refman/8.0/en/drop-procedure.html
type DropProcedureStmt struct {
	ddlNode

	IfExists bool
	Name     *TableName
}

// Restore implements Node interface.
func (n *DropProcedureStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Name == nil {
		return errors.New("restore drop procedure: missing name")
	}
	ctx.WriteKeyword("DROP PROCEDURE ")
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}
	if err := n.Name.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore DropProcedureStmt.Name")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropProcedureStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := newNode.(*DropProcedureStmt)
	if !ok || stmt == nil {
		return newNode, false
	}
	n = stmt
	if skipChildren {
		return v.Leave(n)
	}
	if n.Name == nil {
		return n, false
	}
	node, ok := n.Name.Accept(v)
	if !ok {
		return n, false
	}
	name, ok := node.(*TableName)
	if !ok || name == nil {
		return n, false
	}
	n.Name = name
	return v.Leave(n)
}

// DropTriggerStmt is a statement to drop a trigger.
// See https://dev.mysql.com/doc/refman/8.0/en/drop-trigger.html
type DropTriggerStmt struct {
	ddlNode

	IfExists bool
	Name     *TableName
}

// Restore implements Node interface.
func (n *DropTriggerStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Name == nil {
		return errors.New("restore drop trigger: missing name")
	}
	ctx.WriteKeyword("DROP TRIGGER ")
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}
	if err := n.Name.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore DropTriggerStmt.Name")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropTriggerStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := newNode.(*DropTriggerStmt)
	if !ok || stmt == nil {
		return newNode, false
	}
	n = stmt
	if skipChildren {
		return v.Leave(n)
	}
	if n.Name == nil {
		return n, false
	}
	node, ok := n.Name.Accept(v)
	if !ok {
		return n, false
	}
	name, ok := node.(*TableName)
	if !ok || name == nil {
		return n, false
	}
	n.Name = name
	return v.Leave(n)
}

// DropEventStmt is a statement to drop an event.
// See https://dev.mysql.com/doc/refman/8.0/en/drop-event.html
type DropEventStmt struct {
	ddlNode

	IfExists bool
	Name     *TableName
}

// Restore implements Node interface.
func (n *DropEventStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Name == nil {
		return errors.New("restore drop event: missing name")
	}
	ctx.WriteKeyword("DROP EVENT ")
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}
	if err := n.Name.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore DropEventStmt.Name")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropEventStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := newNode.(*DropEventStmt)
	if !ok || stmt == nil {
		return newNode, false
	}
	n = stmt
	if skipChildren {
		return v.Leave(n)
	}
	if n.Name == nil {
		return n, false
	}
	node, ok := n.Name.Accept(v)
	if !ok {
		return n, false
	}
	name, ok := node.(*TableName)
	if !ok || name == nil {
		return n, false
	}
	n.Name = name
	return v.Leave(n)
}

// RenameTableStmt is a statement to rename a table.
// See http://dev.mysql.com/doc/refman/5.7/en/rename-table.html
type RenameTableStmt struct {
	ddlNode
	Renames []*TableRename
}

// Restore implements Node interface.
func (n *RenameTableStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil || len(n.Renames) == 0 {
		return errors.New("restore rename table: missing renames")
	}
	ctx.WriteKeyword("RENAME TABLE ")
	for index, rename := range n.Renames {
		if index != 0 {
			ctx.WritePlain(", ")
		}
		if err := rename.Restore(ctx); err != nil {
			return errors.Annotatef(err, "restore rename table item %d", index)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *RenameTableStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := newNode.(*RenameTableStmt)
	if !ok || stmt == nil {
		return newNode, false
	}
	n = stmt
	if skipChildren {
		return v.Leave(n)
	}
	if len(n.Renames) == 0 {
		return n, false
	}
	for i, rename := range n.Renames {
		if rename == nil {
			return n, false
		}
		node, ok := rename.Accept(v)
		if !ok {
			return n, false
		}
		next, ok := node.(*TableRename)
		if !ok || next == nil {
			return n, false
		}
		n.Renames[i] = next
	}

	return v.Leave(n)
}

type TableRename struct {
	node
	Table    *TableName
	NewTable *TableName
}

// Restore implements Node interface.
func (n *TableRename) Restore(ctx *format.RestoreCtx) error {
	if ctx == nil {
		return errors.New("restore table rename: nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil || n.Table == nil || n.NewTable == nil {
		return errors.New("restore table rename: missing table")
	}
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "restore table rename source")
	}
	ctx.WriteKeyword(" TO ")
	if err := n.NewTable.Restore(ctx); err != nil {
		return errors.Annotate(err, "restore table rename destination")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *TableRename) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	rename, ok := newNode.(*TableRename)
	if !ok || rename == nil {
		return newNode, false
	}
	n = rename
	if skipChildren {
		return v.Leave(n)
	}
	if n.Table == nil || n.NewTable == nil {
		return n, false
	}
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	table, ok := node.(*TableName)
	if !ok || table == nil {
		return n, false
	}
	n.Table = table
	node, ok = n.NewTable.Accept(v)
	if !ok {
		return n, false
	}
	table, ok = node.(*TableName)
	if !ok || table == nil {
		return n, false
	}
	n.NewTable = table
	return v.Leave(n)
}

// CreateViewStmt is a statement to create a View.
// See https://dev.mysql.com/doc/refman/5.7/en/create-view.html
type CreateViewStmt struct {
	ddlNode

	OrReplace      bool
	ViewName       *TableName
	Columns        []lex.Ident
	Select         QueryNode
	Algorithm      model.ViewAlgorithm
	Definer        *auth.UserIdentity
	Security       model.ViewSecurity
	CheckOption    model.ViewCheckOption
	HasCheckOption bool
}

// Restore implements Node interface.
func (n *CreateViewStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.ViewName == nil || n.Select == nil || n.Definer == nil {
		return errors.New("restore create view: missing required field")
	}
	ctx.WriteKeyword("CREATE ")
	if n.OrReplace {
		ctx.WriteKeyword("OR REPLACE ")
	}
	ctx.WriteKeyword("ALGORITHM")
	ctx.WritePlain(" = ")
	ctx.WriteKeyword(n.Algorithm.String())
	ctx.WriteKeyword(" DEFINER")
	ctx.WritePlain(" = ")

	if err := n.Definer.Restore(ctx); err != nil {
		return errors.Annotate(err, "restore view definer")
	}

	ctx.WriteKeyword(" SQL SECURITY ")
	ctx.WriteKeyword(n.Security.String())
	ctx.WriteKeyword(" VIEW ")

	if err := n.ViewName.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while create CreateViewStmt.ViewName")
	}

	for i, col := range n.Columns {
		if i == 0 {
			ctx.WritePlain(" (")
		} else {
			ctx.WritePlain(",")
		}
		ctx.WriteIdent(col)
		if i == len(n.Columns)-1 {
			ctx.WritePlain(")")
		}
	}

	ctx.WriteKeyword(" AS ")

	if err := n.Select.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while create CreateViewStmt.Select")
	}

	if n.HasCheckOption || n.CheckOption != model.CheckOptionCascaded {
		ctx.WriteKeyword(" WITH ")
		ctx.WriteKeyword(n.CheckOption.String())
		ctx.WriteKeyword(" CHECK OPTION")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateViewStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := newNode.(*CreateViewStmt)
	if !ok || stmt == nil {
		return newNode, false
	}
	n = stmt
	if skipChildren {
		return v.Leave(n)
	}
	if n.ViewName == nil || n.Select == nil {
		return n, false
	}
	node, ok := n.ViewName.Accept(v)
	if !ok {
		return n, false
	}
	viewName, ok := node.(*TableName)
	if !ok || viewName == nil {
		return n, false
	}
	n.ViewName = viewName
	selnode, ok := n.Select.Accept(v)
	if !ok {
		return n, false
	}
	selectNode, ok := selnode.(QueryNode)
	if !ok || !resultSetPresent(selectNode) {
		return n, false
	}
	n.Select = selectNode
	return v.Leave(n)
}

// AlterViewStmt is a statement to alter an existing view.
// See https://dev.mysql.com/doc/refman/8.0/en/alter-view.html
type AlterViewStmt struct {
	ddlNode

	ViewName       *TableName
	Columns        []lex.Ident
	Select         QueryNode
	Algorithm      model.ViewAlgorithm
	Definer        *auth.UserIdentity
	Security       model.ViewSecurity
	CheckOption    model.ViewCheckOption
	HasCheckOption bool
}

// Restore implements Node interface.
func (n *AlterViewStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.ViewName == nil || n.Select == nil || n.Definer == nil {
		return errors.New("restore alter view: missing required field")
	}
	ctx.WriteKeyword("ALTER ALGORITHM")
	ctx.WritePlain(" = ")
	ctx.WriteKeyword(n.Algorithm.String())
	ctx.WriteKeyword(" DEFINER")
	ctx.WritePlain(" = ")

	if err := n.Definer.Restore(ctx); err != nil {
		return errors.Annotate(err, "restore view definer")
	}

	ctx.WriteKeyword(" SQL SECURITY ")
	ctx.WriteKeyword(n.Security.String())
	ctx.WriteKeyword(" VIEW ")

	if err := n.ViewName.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while alter AlterViewStmt.ViewName")
	}

	for i, col := range n.Columns {
		if i == 0 {
			ctx.WritePlain(" (")
		} else {
			ctx.WritePlain(",")
		}
		ctx.WriteIdent(col)
		if i == len(n.Columns)-1 {
			ctx.WritePlain(")")
		}
	}

	ctx.WriteKeyword(" AS ")

	if err := n.Select.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while alter AlterViewStmt.Select")
	}

	if n.HasCheckOption || n.CheckOption != model.CheckOptionCascaded {
		ctx.WriteKeyword(" WITH ")
		ctx.WriteKeyword(n.CheckOption.String())
		ctx.WriteKeyword(" CHECK OPTION")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *AlterViewStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := newNode.(*AlterViewStmt)
	if !ok || stmt == nil {
		return newNode, false
	}
	n = stmt
	if skipChildren {
		return v.Leave(n)
	}
	if n.ViewName == nil || n.Select == nil {
		return n, false
	}
	node, ok := n.ViewName.Accept(v)
	if !ok {
		return n, false
	}
	viewName, ok := node.(*TableName)
	if !ok || viewName == nil {
		return n, false
	}
	n.ViewName = viewName
	selnode, ok := n.Select.Accept(v)
	if !ok {
		return n, false
	}
	selectNode, ok := selnode.(QueryNode)
	if !ok || !resultSetPresent(selectNode) {
		return n, false
	}
	n.Select = selectNode
	return v.Leave(n)
}

// CreateIndexStmt is a statement to create an index.
// See https://dev.mysql.com/doc/refman/5.7/en/create-index.html
type CreateIndexStmt struct {
	ddlNode

	IndexName   lex.Ident
	Table       *TableName
	Unique      bool
	Keys        []pkast.IndexKeyPartNode
	IndexOption *IndexOption
}

// Restore implements Node interface.
func (n *CreateIndexStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Table == nil {
		return errors.New("restore create index: missing table")
	}
	if len(n.Keys) == 0 {
		return errors.New("restore create index: missing keys")
	}
	ctx.WriteKeyword("CREATE ")
	if n.Unique {
		ctx.WriteKeyword("UNIQUE ")
	}
	ctx.WriteKeyword("INDEX ")
	ctx.WriteIdent(n.IndexName)
	ctx.WriteKeyword(" ON ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CreateIndexStmt.Table")
	}

	ctx.WritePlain(" (")
	for i, key := range n.Keys {
		if key == nil || key.IndexKeyPartCore() == nil {
			return errors.Errorf("restore create index: nil key at %d", i)
		}
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := key.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore CreateIndexStmt.Keys: [%v]", i)
		}
	}
	ctx.WritePlain(")")

	if n.IndexOption != nil && (n.IndexOption.Tp != model.IndexTypeInvalid || n.IndexOption.KeyBlockSize > 0 || n.IndexOption.Comment != "") {
		ctx.WritePlain(" ")
		if err := n.IndexOption.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore CreateIndexStmt.IndexOption")
		}
	}

	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateIndexStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := newNode.(*CreateIndexStmt)
	if !ok || stmt == nil {
		return newNode, false
	}
	n = stmt
	if skipChildren {
		return v.Leave(n)
	}
	if n.Table == nil {
		return n, false
	}
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	table, ok := node.(*TableName)
	if !ok || table == nil {
		return n, false
	}
	n.Table = table
	for i, val := range n.Keys {
		if val == nil || val.IndexKeyPartCore() == nil {
			return n, false
		}
		node, ok = val.Accept(v)
		if !ok {
			return n, false
		}
		key, ok := node.(pkast.IndexKeyPartNode)
		if !ok || key == nil || key.IndexKeyPartCore() == nil {
			return n, false
		}
		n.Keys[i] = key
	}
	if n.IndexOption != nil {
		node, ok := n.IndexOption.Accept(v)
		if !ok {
			return n, false
		}
		option, ok := node.(*IndexOption)
		if !ok || option == nil {
			return n, false
		}
		n.IndexOption = option
	}
	return v.Leave(n)
}

// DropIndexStmt is a statement to drop the index.
// See https://dev.mysql.com/doc/refman/5.7/en/drop-index.html
type DropIndexStmt struct {
	ddlNode

	IfExists  bool
	IndexName lex.Ident
	Table     *TableName
}

// Restore implements Node interface.
func (n *DropIndexStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Table == nil {
		return errors.New("restore drop index: missing table")
	}
	ctx.WriteKeyword("DROP INDEX ")
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}
	ctx.WriteIdent(n.IndexName)
	ctx.WriteKeyword(" ON ")

	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while add index")
	}

	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropIndexStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := newNode.(*DropIndexStmt)
	if !ok || stmt == nil {
		return newNode, false
	}
	n = stmt
	if skipChildren {
		return v.Leave(n)
	}
	if n.Table == nil {
		return n, false
	}
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	table, ok := node.(*TableName)
	if !ok || table == nil {
		return n, false
	}
	n.Table = table
	return v.Leave(n)
}

// TableOptionType is the type for TableOption
type TableOptionType int

// TableOption types.
const (
	TableOptionNone TableOptionType = iota
	TableOptionEngine
	TableOptionCharset
	TableOptionCollate
	TableOptionAutoIncrement
	TableOptionComment
	TableOptionAvgRowLength
	TableOptionCheckSum
	TableOptionCompression
	TableOptionConnection
	TableOptionPassword
	TableOptionKeyBlockSize
	TableOptionMaxRows
	TableOptionMinRows
	TableOptionDelayKeyWrite
	TableOptionRowFormat
	TableOptionStatsPersistent
	TableOptionShardRowID
	TableOptionPackKeys
	TableOptionTablespace
)

// RowFormat types
const (
	RowFormatDefault uint64 = iota + 1
	RowFormatDynamic
	RowFormatFixed
	RowFormatCompressed
	RowFormatRedundant
	RowFormatCompact
	TokuDBRowFormatDefault
	TokuDBRowFormatFast
	TokuDBRowFormatSmall
	TokuDBRowFormatZlib
	TokuDBRowFormatQuickLZ
	TokuDBRowFormatLzma
	TokuDBRowFormatSnappy
	TokuDBRowFormatUncompressed
)

// OnDuplicateKeyHandlingType is the option that handle unique key values in 'CREATE TABLE ... SELECT' or `LOAD DATA`.
// See https://dev.mysql.com/doc/refman/5.7/en/create-table-select.html
type OnDuplicateKeyHandlingType int

// OnDuplicateKeyHandling types
const (
	OnDuplicateKeyHandlingError OnDuplicateKeyHandlingType = iota
	OnDuplicateKeyHandlingIgnore
	OnDuplicateKeyHandlingReplace
)

// TableOption is used for parsing table option from SQL.
type TableOption struct {
	node
	Tp        TableOptionType
	StrValue  string
	UintValue uint64
}

// Restore implements Node interface.
func (n *TableOption) Restore(ctx *format.RestoreCtx) error {
	if ctx == nil {
		return errors.New("restore table option: nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("restore table option: nil option")
	}
	switch n.Tp {
	case TableOptionEngine:
		ctx.WriteKeyword("ENGINE ")
		ctx.WritePlain("= ")
		if n.StrValue != "" {
			ctx.WritePlain(n.StrValue)
		} else {
			ctx.WritePlain("''")
		}
	case TableOptionCharset:
		ctx.WriteKeyword("DEFAULT CHARACTER SET ")
		ctx.WritePlain("= ")
		ctx.WritePlain(n.StrValue)
	case TableOptionCollate:
		ctx.WriteKeyword("DEFAULT COLLATE ")
		ctx.WritePlain("= ")
		ctx.WritePlain(n.StrValue)
	case TableOptionAutoIncrement:
		ctx.WriteKeyword("AUTO_INCREMENT ")
		ctx.WritePlain("= ")
		ctx.WritePlainf("%d", n.UintValue)
	case TableOptionComment:
		ctx.WriteKeyword("COMMENT ")
		ctx.WritePlain("= ")
		ctx.WriteString(n.StrValue)
	case TableOptionTablespace:
		ctx.WriteKeyword("TABLESPACE ")
		ctx.WritePlain("= ")
		ctx.WriteName(n.StrValue)
	case TableOptionAvgRowLength:
		ctx.WriteKeyword("AVG_ROW_LENGTH ")
		ctx.WritePlain("= ")
		ctx.WritePlainf("%d", n.UintValue)
	case TableOptionCheckSum:
		ctx.WriteKeyword("CHECKSUM ")
		ctx.WritePlain("= ")
		ctx.WritePlainf("%d", n.UintValue)
	case TableOptionCompression:
		ctx.WriteKeyword("COMPRESSION ")
		ctx.WritePlain("= ")
		ctx.WriteString(n.StrValue)
	case TableOptionConnection:
		ctx.WriteKeyword("CONNECTION ")
		ctx.WritePlain("= ")
		ctx.WriteString(n.StrValue)
	case TableOptionPassword:
		ctx.WriteKeyword("PASSWORD ")
		ctx.WritePlain("= ")
		ctx.WriteString(n.StrValue)
	case TableOptionKeyBlockSize:
		ctx.WriteKeyword("KEY_BLOCK_SIZE ")
		ctx.WritePlain("= ")
		ctx.WritePlainf("%d", n.UintValue)
	case TableOptionMaxRows:
		ctx.WriteKeyword("MAX_ROWS ")
		ctx.WritePlain("= ")
		ctx.WritePlainf("%d", n.UintValue)
	case TableOptionMinRows:
		ctx.WriteKeyword("MIN_ROWS ")
		ctx.WritePlain("= ")
		ctx.WritePlainf("%d", n.UintValue)
	case TableOptionDelayKeyWrite:
		ctx.WriteKeyword("DELAY_KEY_WRITE ")
		ctx.WritePlain("= ")
		ctx.WritePlainf("%d", n.UintValue)
	case TableOptionRowFormat:
		ctx.WriteKeyword("ROW_FORMAT ")
		ctx.WritePlain("= ")
		switch n.UintValue {
		case RowFormatDefault:
			ctx.WriteKeyword("DEFAULT")
		case RowFormatDynamic:
			ctx.WriteKeyword("DYNAMIC")
		case RowFormatFixed:
			ctx.WriteKeyword("FIXED")
		case RowFormatCompressed:
			ctx.WriteKeyword("COMPRESSED")
		case RowFormatRedundant:
			ctx.WriteKeyword("REDUNDANT")
		case RowFormatCompact:
			ctx.WriteKeyword("COMPACT")
		case TokuDBRowFormatDefault:
			ctx.WriteKeyword("TOKUDB_DEFAULT")
		case TokuDBRowFormatFast:
			ctx.WriteKeyword("TOKUDB_FAST")
		case TokuDBRowFormatSmall:
			ctx.WriteKeyword("TOKUDB_SMALL")
		case TokuDBRowFormatZlib:
			ctx.WriteKeyword("TOKUDB_ZLIB")
		case TokuDBRowFormatQuickLZ:
			ctx.WriteKeyword("TOKUDB_QUICKLZ")
		case TokuDBRowFormatLzma:
			ctx.WriteKeyword("TOKUDB_LZMA")
		case TokuDBRowFormatSnappy:
			ctx.WriteKeyword("TOKUDB_SNAPPY")
		case TokuDBRowFormatUncompressed:
			ctx.WriteKeyword("TOKUDB_UNCOMPRESSED")
		default:
			return errors.Errorf("invalid TableOption: TableOptionRowFormat: %d", n.UintValue)
		}
	case TableOptionStatsPersistent:
		ctx.WriteKeyword("STATS_PERSISTENT ")
		ctx.WritePlain("= ")
		if n.StrValue != "" {
			ctx.WriteKeyword(n.StrValue)
		} else {
			ctx.WritePlainf("%d", n.UintValue)
		}
	case TableOptionShardRowID:
		ctx.WriteKeyword("SHARD_ROW_ID_BITS ")
		ctx.WritePlain("= ")
		ctx.WritePlainf("%d", n.UintValue)
	case TableOptionPackKeys:
		ctx.WriteKeyword("PACK_KEYS ")
		ctx.WritePlain("= ")
		if n.StrValue != "" {
			ctx.WriteKeyword(n.StrValue)
		} else {
			ctx.WritePlainf("%d", n.UintValue)
		}
	default:
		return errors.Errorf("invalid TableOption: %d", n.Tp)
	}
	return ctx.Err()
}

func (n *TableOption) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, _ := v.Enter(n)
	option, ok := newNode.(*TableOption)
	if !ok || option == nil {
		return newNode, false
	}
	return v.Leave(option)
}

var _ Node = (*TableOption)(nil)

func acceptTableOptions(v Visitor, options []*TableOption) bool {
	for i, option := range options {
		if option == nil {
			return false
		}
		node, ok := option.Accept(v)
		if !ok {
			return false
		}
		replacement, ok := node.(*TableOption)
		if !ok || replacement == nil {
			return false
		}
		options[i] = replacement
	}
	return true
}

// ColumnPositionType is the type for ColumnPosition.
type ColumnPositionType int

// ColumnPosition Types
const (
	ColumnPositionNone ColumnPositionType = iota
	ColumnPositionFirst
	ColumnPositionAfter
)

// ColumnPosition represent the position of the newly added column
type ColumnPosition struct {
	node
	// Tp is either ColumnPositionNone, ColumnPositionFirst or ColumnPositionAfter.
	Tp ColumnPositionType
	// RelativeColumn is the column the newly added column after if type is ColumnPositionAfter
	RelativeColumn *QualifiedName
}

// Restore implements Node interface.
func (n *ColumnPosition) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore column position: nil position")
	}
	switch n.Tp {
	case ColumnPositionNone:
		// do nothing
	case ColumnPositionFirst:
		ctx.WriteKeyword("FIRST")
	case ColumnPositionAfter:
		if n.RelativeColumn == nil {
			return errors.New("restore column position: missing relative column")
		}
		ctx.WriteKeyword("AFTER ")
		if err := restoreQualifiedName(ctx, *n.RelativeColumn); err != nil {
			return errors.Annotate(err, "An error occurred while restore ColumnPosition.RelativeColumn")
		}
	default:
		return errors.Errorf("invalid ColumnPositionType: %d", n.Tp)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ColumnPosition) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	position, ok := newNode.(*ColumnPosition)
	if !ok || position == nil {
		return newNode, false
	}
	n = position
	if skipChildren {
		return v.Leave(n)
	}
	return v.Leave(n)
}

// AlterTableType is the type for AlterTableSpec.
type AlterTableType int

// AlterTable types.
const (
	AlterTableOption AlterTableType = iota + 1
	AlterTableAddColumns
	AlterTableAddConstraint
	AlterTableDropColumn
	AlterTableDropPrimaryKey
	AlterTableDropIndex
	AlterTableDropForeignKey
	AlterTableModifyColumn
	AlterTableChangeColumn
	AlterTableRenameTable
	AlterTableAlterColumn
	AlterTableLock
	AlterTableAlgorithm
	AlterTableRenameIndex
	AlterTableForce
	AlterTableAddPartitions
	AlterTableCoalescePartitions
	AlterTableDropPartition
	AlterTableTruncatePartition
	AlterTableRenameColumn
	AlterTableExchangePartition
	AlterTableReorganizePartition
	AlterTableRemovePartitioning
	AlterTableDiscardTablespace
	AlterTableImportTablespace
	AlterTablePartition
	AlterTableEnableKeys
	AlterTableDisableKeys

	// TODO: Add more actions
)

// LockType is the type for AlterTableSpec.
// See https://dev.mysql.com/doc/refman/5.7/en/alter-table.html#alter-table-concurrency
type LockType byte

func (n LockType) String() string {
	switch n {
	case LockTypeNone:
		return "NONE"
	case LockTypeDefault:
		return "DEFAULT"
	case LockTypeShared:
		return "SHARED"
	case LockTypeExclusive:
		return "EXCLUSIVE"
	}
	return ""
}

// Lock Types.
const (
	LockTypeNone LockType = iota + 1
	LockTypeDefault
	LockTypeShared
	LockTypeExclusive
)

// AlterAlgorithm is the algorithm of the DDL operations.
// See https://dev.mysql.com/doc/refman/8.0/en/alter-table.html#alter-table-performance.
type AlterAlgorithm byte

// DDL alter algorithms.
// For now, only supported inplace and instance algorithms. If the user specify `copy`,
// will get an error.
const (
	AlterAlgorithmDefault AlterAlgorithm = iota
	AlterAlgorithmCopy
	AlterAlgorithmInplace
	AlterAlgorithmInstant
)

func (a AlterAlgorithm) String() string {
	switch a {
	case AlterAlgorithmDefault:
		return "DEFAULT"
	case AlterAlgorithmCopy:
		return "COPY"
	case AlterAlgorithmInplace:
		return "INPLACE"
	case AlterAlgorithmInstant:
		return "INSTANT"
	default:
		return "DEFAULT"
	}
}

// AlterTableSpec represents alter table specification.
type AlterTableSpec struct {
	node

	Tp                AlterTableType
	Name              lex.Ident
	Constraint        pkast.ConstraintNode
	Options           []*TableOption
	CharsetConvert    bool
	NewTable          *TableName
	NewColumns        []*ColumnDef
	Default           ExprNode
	OldColumnName     *QualifiedName
	NewColumnName     *QualifiedName
	Position          *ColumnPosition
	LockType          LockType
	Algorithm         AlterAlgorithm
	Comment           string
	FromKey           lex.Ident
	ToKey             lex.Ident
	PartDefinitions   []*PartitionDefinition
	PartitionNames    []lex.Ident
	Partition         *PartitionOptions
	WithoutValidation bool
	IfExists          bool
	Num               uint64
}

// Restore implements Node interface.
func (n *AlterTableSpec) Restore(ctx *format.RestoreCtx) error {
	if ctx == nil {
		return errors.New("restore alter table spec: nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("restore alter table spec: nil spec")
	}
	switch n.Tp {
	case AlterTableOption:
		switch {
		case n.CharsetConvert:
			if len(n.Options) < 1 || len(n.Options) > 2 || n.Options[0] == nil || n.Options[0].Tp != TableOptionCharset {
				return errors.New("Invalid character set conversion options")
			}
			ctx.WriteKeyword("CONVERT TO CHARACTER SET ")
			ctx.WritePlain(n.Options[0].StrValue)
			if len(n.Options) == 2 {
				if n.Options[1] == nil || n.Options[1].Tp != TableOptionCollate {
					return errors.New("Invalid character set conversion collation")
				}
				ctx.WriteKeyword(" COLLATE ")
				ctx.WritePlain(n.Options[1].StrValue)
			}
		default:
			if len(n.Options) == 0 {
				return errors.New("restore alter table options: no options")
			}
			for i, opt := range n.Options {
				if opt == nil {
					return errors.Errorf("restore alter table options: nil option at %d", i)
				}
				if i != 0 {
					ctx.WritePlain(" ")
				}
				if err := opt.Restore(ctx); err != nil {
					return errors.Annotatef(err, "An error occurred while restore AlterTableSpec.Options[%d]", i)
				}
			}
		}
	case AlterTableAddColumns:
		if len(n.NewColumns) == 0 || n.Position != nil && len(n.NewColumns) != 1 {
			return errors.New("restore alter table add columns: invalid columns or position")
		}
		ctx.WriteKeyword("ADD COLUMN ")
		if n.Position != nil && len(n.NewColumns) == 1 {
			if err := n.NewColumns[0].Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore AlterTableSpec.NewColumns[%d]", 0)
			}
			if n.Position.Tp != ColumnPositionNone {
				ctx.WritePlain(" ")
			}
			if err := n.Position.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore AlterTableSpec.Position")
			}
		} else {
			ctx.WritePlain("(")
			for i, col := range n.NewColumns {
				if col == nil {
					return errors.Errorf("restore alter table add columns: nil column at %d", i)
				}
				if i != 0 {
					ctx.WritePlain(", ")
				}
				if err := col.Restore(ctx); err != nil {
					return errors.Annotatef(err, "An error occurred while restore AlterTableSpec.NewColumns[%d]", i)
				}
			}
			ctx.WritePlain(")")
		}
	case AlterTableAddConstraint:
		if n.Constraint == nil || n.Constraint.ConstraintCore() == nil {
			return errors.New("restore alter table: missing constraint")
		}
		ctx.WriteKeyword("ADD ")
		if err := n.Constraint.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AlterTableSpec.Constraint")
		}
	case AlterTableDropColumn:
		if n.OldColumnName == nil {
			return errors.New("restore alter table drop column: missing column name")
		}
		ctx.WriteKeyword("DROP COLUMN ")
		if err := restoreQualifiedName(ctx, *n.OldColumnName); err != nil {
			return errors.Annotate(err, "An error occurred while restore AlterTableSpec.OldColumnName")
		}
	// TODO: RestrictOrCascadeOpt not support
	case AlterTableDropPrimaryKey:
		ctx.WriteKeyword("DROP PRIMARY KEY")
	case AlterTableDropIndex:
		ctx.WriteKeyword("DROP INDEX ")
		ctx.WriteIdent(n.Name)
	case AlterTableDropForeignKey:
		ctx.WriteKeyword("DROP FOREIGN KEY ")
		if n.IfExists {
			ctx.WriteKeyword("IF EXISTS ")
		}
		ctx.WriteIdent(n.Name)
	case AlterTableModifyColumn:
		if len(n.NewColumns) != 1 || n.NewColumns[0] == nil || n.Position == nil {
			return errors.New("restore alter table modify column: missing column or position")
		}
		ctx.WriteKeyword("MODIFY COLUMN ")
		if err := n.NewColumns[0].Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AlterTableSpec.NewColumns[0]")
		}
		if n.Position.Tp != ColumnPositionNone {
			ctx.WritePlain(" ")
		}
		if err := n.Position.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AlterTableSpec.Position")
		}
	case AlterTableChangeColumn:
		if n.OldColumnName == nil || len(n.NewColumns) != 1 || n.NewColumns[0] == nil || n.Position == nil {
			return errors.New("restore alter table change column: missing target or position")
		}
		ctx.WriteKeyword("CHANGE COLUMN ")
		if n.OldColumnName != nil {
			if err := restoreQualifiedName(ctx, *n.OldColumnName); err != nil {
				return errors.Annotate(err, "An error occurred while restore AlterTableSpec.OldColumnName")
			}
		}
		ctx.WritePlain(" ")
		if err := n.NewColumns[0].Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AlterTableSpec.NewColumns[0]")
		}
		if n.Position.Tp != ColumnPositionNone {
			ctx.WritePlain(" ")
		}
		if err := n.Position.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AlterTableSpec.Position")
		}
	case AlterTableRenameTable:
		if n.NewTable == nil {
			return errors.New("restore alter table rename: missing table")
		}
		ctx.WriteKeyword("RENAME AS ")
		if err := n.NewTable.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AlterTableSpec.NewTable")
		}
	case AlterTableAlterColumn:
		if n.OldColumnName == nil || len(n.NewColumns) != 0 {
			return errors.New("restore alter column default: invalid target")
		}
		ctx.WriteKeyword("ALTER COLUMN ")
		if err := restoreQualifiedName(ctx, *n.OldColumnName); err != nil {
			return errors.Annotate(err, "restore alter column default target")
		}
		if n.Default != nil {
			if !nodePresent(n.Default) {
				return errors.New("restore alter column default: nil expression")
			}
			ctx.WriteKeyword(" SET DEFAULT ")
			if err := n.Default.Restore(ctx); err != nil {
				return errors.Annotate(err, "restore alter column default expression")
			}
		} else {
			ctx.WriteKeyword(" DROP DEFAULT")
		}
	case AlterTableLock:
		ctx.WriteKeyword("LOCK ")
		ctx.WritePlain("= ")
		ctx.WriteKeyword(n.LockType.String())
	case AlterTableAlgorithm:
		ctx.WriteKeyword("ALGORITHM ")
		ctx.WritePlain("= ")
		ctx.WriteKeyword(n.Algorithm.String())
	case AlterTableRenameIndex:
		ctx.WriteKeyword("RENAME INDEX ")
		ctx.WriteIdent(n.FromKey)
		ctx.WriteKeyword(" TO ")
		ctx.WriteIdent(n.ToKey)
	case AlterTableRenameColumn:
		if n.OldColumnName == nil || n.NewColumnName == nil {
			return errors.New("restore alter table rename column: missing name")
		}
		ctx.WriteKeyword("RENAME COLUMN ")
		if n.OldColumnName != nil {
			if err := restoreQualifiedName(ctx, *n.OldColumnName); err != nil {
				return errors.Annotate(err, "An error occurred while restore AlterTableSpec.OldColumnName")
			}
		}
		ctx.WriteKeyword(" TO ")
		if n.NewColumnName != nil {
			if err := restoreQualifiedName(ctx, *n.NewColumnName); err != nil {
				return errors.Annotate(err, "An error occurred while restore AlterTableSpec.NewColumnName")
			}
		}
	case AlterTableForce:
		// TODO: not support
		ctx.WriteKeyword("FORCE")
		ctx.WritePlain(" /* AlterTableForce is not supported */ ")
	case AlterTableAddPartitions:
		if n.PartDefinitions != nil && len(n.PartDefinitions) == 0 {
			return errors.New("restore alter table add partitions: empty definitions")
		}
		ctx.WriteKeyword("ADD PARTITION")
		if n.PartDefinitions != nil {
			ctx.WritePlain(" (")
			for i, def := range n.PartDefinitions {
				if def == nil {
					return errors.Errorf("restore alter table add partitions: nil definition at %d", i)
				}
				if i != 0 {
					ctx.WritePlain(", ")
				}
				if err := def.Restore(ctx); err != nil {
					return errors.Annotatef(err, "An error occurred while restore AlterTableSpec.PartDefinitions[%d]", i)
				}
			}
			ctx.WritePlain(")")
		} else if n.Num != 0 {
			ctx.WriteKeyword(" PARTITIONS ")
			ctx.WritePlainf("%d", n.Num)
		}
	case AlterTableCoalescePartitions:
		ctx.WriteKeyword("COALESCE PARTITION ")
		ctx.WritePlainf("%d", n.Num)
	case AlterTableDropPartition:
		ctx.WriteKeyword("DROP PARTITION ")
		ctx.WriteIdent(n.Name)
	case AlterTableTruncatePartition:
		ctx.WriteKeyword("TRUNCATE PARTITION ")
		ctx.WriteIdent(n.Name)
	case AlterTableExchangePartition:
		if n.NewTable == nil {
			return errors.New("restore alter table exchange partition: missing table")
		}
		ctx.WriteKeyword("EXCHANGE PARTITION ")
		ctx.WriteIdent(n.Name)
		ctx.WriteKeyword(" WITH TABLE ")
		if err := n.NewTable.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AlterTableSpec.NewTable")
		}
		if n.WithoutValidation {
			ctx.WriteKeyword(" WITHOUT VALIDATION")
		}
	case AlterTableReorganizePartition:
		if len(n.PartDefinitions) == 0 {
			return errors.New("restore alter table reorganize partition: missing definitions")
		}
		ctx.WriteKeyword("REORGANIZE PARTITION ")
		if len(n.PartitionNames) == 0 {
			ctx.WriteKeyword("ALL")
		} else {
			for i, name := range n.PartitionNames {
				if i != 0 {
					ctx.WritePlain(", ")
				}
				ctx.WriteIdent(name)
			}
		}
		ctx.WriteKeyword(" INTO (")
		for i, def := range n.PartDefinitions {
			if def == nil {
				return errors.Errorf("restore alter table reorganize partition: nil definition at %d", i)
			}
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if err := def.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore AlterTableSpec.PartDefinitions[%d]", i)
			}
		}
		ctx.WritePlain(")")
	case AlterTableRemovePartitioning:
		ctx.WriteKeyword("REMOVE PARTITIONING")
	case AlterTableDiscardTablespace:
		ctx.WriteKeyword("DISCARD TABLESPACE")
	case AlterTableImportTablespace:
		ctx.WriteKeyword("IMPORT TABLESPACE")
	case AlterTableEnableKeys:
		ctx.WriteKeyword("ENABLE KEYS")
	case AlterTableDisableKeys:
		ctx.WriteKeyword("DISABLE KEYS")
	case AlterTablePartition:
		if n.Partition == nil {
			return errors.New("Invalid partition options")
		}
		if err := n.Partition.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AlterTableSpec.Partition")
		}
	default:
		return errors.Errorf("restore alter table spec: invalid operation %d", n.Tp)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *AlterTableSpec) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	spec, ok := newNode.(*AlterTableSpec)
	if !ok || spec == nil {
		return newNode, false
	}
	n = spec
	if skipChildren {
		return v.Leave(n)
	}
	if !acceptTableOptions(v, n.Options) {
		return n, false
	}
	if n.Default != nil {
		if !nodePresent(n.Default) {
			return n, false
		}
		node, ok := n.Default.Accept(v)
		if !ok {
			return n, false
		}
		expr, ok := node.(ExprNode)
		if !ok || !nodePresent(expr) {
			return n, false
		}
		n.Default = expr
	}
	if n.Constraint != nil {
		if n.Constraint.ConstraintCore() == nil {
			return n, false
		}
		node, ok := n.Constraint.Accept(v)
		if !ok {
			return n, false
		}
		constraint, ok := node.(pkast.ConstraintNode)
		if !ok || constraint == nil || constraint.ConstraintCore() == nil {
			return n, false
		}
		n.Constraint = constraint
	}
	if n.NewTable != nil {
		if !nodePresent(n.NewTable) {
			return n, false
		}
		node, ok := n.NewTable.Accept(v)
		if !ok {
			return n, false
		}
		table, ok := node.(*TableName)
		if !ok || table == nil {
			return n, false
		}
		n.NewTable = table
	}
	for i, col := range n.NewColumns {
		if col == nil {
			return n, false
		}
		node, ok := col.Accept(v)
		if !ok {
			return n, false
		}
		column, ok := node.(*ColumnDef)
		if !ok || column == nil {
			return n, false
		}
		n.NewColumns[i] = column
	}
	if n.Position != nil {
		if !nodePresent(n.Position) {
			return n, false
		}
		node, ok := n.Position.Accept(v)
		if !ok {
			return n, false
		}
		position, ok := node.(*ColumnPosition)
		if !ok || position == nil {
			return n, false
		}
		n.Position = position
	}
	if n.Partition != nil {
		if !nodePresent(n.Partition) {
			return n, false
		}
		node, ok := n.Partition.Accept(v)
		if !ok {
			return n, false
		}
		partition, ok := node.(*PartitionOptions)
		if !ok || partition == nil {
			return n, false
		}
		n.Partition = partition
	}
	for i, def := range n.PartDefinitions {
		if def == nil {
			return n, false
		}
		node, ok := def.Accept(v)
		if !ok {
			return n, false
		}
		definition, ok := node.(*PartitionDefinition)
		if !ok || definition == nil {
			return n, false
		}
		n.PartDefinitions[i] = definition
	}
	return v.Leave(n)
}

// AlterTableStmt is a statement to change the structure of a table.
// See https://dev.mysql.com/doc/refman/5.7/en/alter-table.html
type AlterTableStmt struct {
	ddlNode

	Table *TableName
	Specs []*AlterTableSpec
}

// Restore implements Node interface.
func (n *AlterTableStmt) Restore(ctx *format.RestoreCtx) error {
	if ctx == nil {
		return errors.New("restore alter table: nil context")
	}
	if n == nil || n.Table == nil {
		return errors.New("restore alter table: missing table")
	}
	if len(n.Specs) == 0 {
		return errors.New("restore alter table: missing specifications")
	}
	ctx.WriteKeyword("ALTER TABLE ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore AlterTableStmt.Table")
	}
	for i, spec := range n.Specs {
		if spec == nil {
			return errors.Errorf("restore alter table: nil specification at %d", i)
		}
		if i == 0 {
			ctx.WritePlain(" ")
		} else {
			ctx.WritePlain(", ")
		}
		if err := spec.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore AlterTableStmt.Specs[%d]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *AlterTableStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := newNode.(*AlterTableStmt)
	if !ok || stmt == nil {
		return newNode, false
	}
	n = stmt
	if skipChildren {
		return v.Leave(n)
	}
	if n.Table == nil {
		return n, false
	}
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	table, ok := node.(*TableName)
	if !ok || table == nil {
		return n, false
	}
	n.Table = table
	for i, val := range n.Specs {
		if val == nil {
			return n, false
		}
		node, ok = val.Accept(v)
		if !ok {
			return n, false
		}
		spec, ok := node.(*AlterTableSpec)
		if !ok || spec == nil {
			return n, false
		}
		n.Specs[i] = spec
	}
	return v.Leave(n)
}

// TruncateTableStmt is a statement to empty a table completely.
// See https://dev.mysql.com/doc/refman/5.7/en/truncate-table.html
type TruncateTableStmt struct {
	ddlNode

	Table *TableName
}

// Restore implements Node interface.
func (n *TruncateTableStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("TRUNCATE TABLE ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore TruncateTableStmt.Table")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *TruncateTableStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	stmt, ok := newNode.(*TruncateTableStmt)
	if !ok || stmt == nil {
		return newNode, false
	}
	n = stmt
	if skipChildren {
		return v.Leave(n)
	}
	if n.Table == nil {
		return n, false
	}
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	table, ok := node.(*TableName)
	if !ok || table == nil {
		return n, false
	}
	n.Table = table
	return v.Leave(n)
}

// PartitionDefinition defines a single partition.
type PartitionDefinition struct {
	node
	Name                 lex.Ident
	LessThan             []ExprNode
	Options              []*TableOption
	OmitValueParentheses bool
	OmitPartitionKeyword bool
}

// Restore implements Node interface.
func (n *PartitionDefinition) Restore(ctx *format.RestoreCtx) error {
	if ctx == nil {
		return errors.New("restore partition definition: nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("restore partition definition: nil definition")
	}
	if n.OmitValueParentheses {
		if len(n.LessThan) != 1 {
			return errors.New("restore partition definition: bare boundary requires MAXVALUE")
		}
		maxValue, ok := n.LessThan[0].(*MaxValueExpr)
		if !ok || maxValue == nil {
			return errors.New("restore partition definition: bare boundary requires MAXVALUE")
		}
	}
	if n.LessThan != nil && len(n.LessThan) == 0 {
		return errors.New("restore partition definition: empty boundary")
	}
	if !n.OmitPartitionKeyword {
		ctx.WriteKeyword("PARTITION ")
	}
	ctx.WriteIdent(n.Name)
	if n.LessThan != nil {
		ctx.WriteKeyword(" VALUES LESS THAN ")
		if !n.OmitValueParentheses {
			ctx.WritePlain("(")
		}
		for k, less := range n.LessThan {
			if !nodePresent(less) {
				return errors.Errorf("restore partition boundary %d: nil expression", k)
			}
			if k != 0 {
				ctx.WritePlain(", ")
			}
			if err := less.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore PartitionDefinition.LessThan[%d]", k)
			}
		}
		if !n.OmitValueParentheses {
			ctx.WritePlain(")")
		}
	}
	for i, option := range n.Options {
		if option == nil {
			return errors.Errorf("restore partition definition: nil option at %d", i)
		}
		ctx.WritePlain(" ")
		if err := option.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore PartitionDefinition.Options[%d]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *PartitionDefinition) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	definition, ok := newNode.(*PartitionDefinition)
	if !ok || definition == nil {
		return newNode, false
	}
	n = definition
	if skipChildren {
		return v.Leave(n)
	}
	for i, expr := range n.LessThan {
		if !nodePresent(expr) {
			return n, false
		}
		node, ok := expr.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := node.(ExprNode)
		if !ok || !nodePresent(replacement) {
			return n, false
		}
		n.LessThan[i] = replacement
	}
	if !acceptTableOptions(v, n.Options) {
		return n, false
	}
	return v.Leave(n)
}

// PartitionOptions specifies the partition options.
type PartitionOptions struct {
	node
	Tp             model.PartitionType
	Expr           ExprNode
	ColumnNames    []QualifiedName
	Definitions    []*PartitionDefinition
	Num            uint64
	HasNum         bool
	Linear         bool
	Sub            *PartitionOptions
	IsSubPartition bool
}

// Restore implements Node interface.
func (n *PartitionOptions) Restore(ctx *format.RestoreCtx) error {
	if ctx == nil {
		return errors.New("restore partition options: nil context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("restore partition options: nil options")
	}
	if n.Expr != nil && (!nodePresent(n.Expr) || n.ColumnNames != nil) {
		return errors.New("restore partition options: invalid expression or column list")
	}
	switch n.Tp {
	case model.PartitionTypeHash:
		if !nodePresent(n.Expr) || n.ColumnNames != nil {
			return errors.New("restore hash partition: missing expression")
		}
	case model.PartitionTypeRange:
		if (n.Expr == nil) == (len(n.ColumnNames) == 0) {
			return errors.New("restore range partition: missing expression or columns")
		}
	case model.PartitionTypeKey:
		if n.Expr != nil {
			return errors.New("restore key partition: unexpected expression")
		}
	default:
		return errors.Errorf("restore partition options: invalid type %d", n.Tp)
	}
	if n.IsSubPartition {
		ctx.WriteKeyword("SUBPARTITION BY ")
	} else {
		ctx.WriteKeyword("PARTITION BY ")
	}
	if n.Linear {
		ctx.WriteKeyword("LINEAR ")
	}
	switch n.Tp {
	case model.PartitionTypeRange:
		ctx.WriteKeyword("RANGE ")
	case model.PartitionTypeHash:
		ctx.WriteKeyword("HASH ")
	case model.PartitionTypeKey:
		ctx.WriteKeyword("KEY ")
	default:
		return errors.Errorf("invalid model.PartitionType: %d", n.Tp)
	}

	if n.Expr != nil {
		ctx.WritePlain("(")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore PartitionOptions Expr")
		}
		ctx.WritePlain(")")
	}
	if n.Tp != model.PartitionTypeKey && len(n.ColumnNames) > 0 {
		ctx.WriteKeyword("COLUMNS")
		ctx.WritePlain("(")
		for i, col := range n.ColumnNames {
			if i > 0 {
				ctx.WritePlain(",")
			}
			if err := restoreQualifiedName(ctx, col); err != nil {
				return errors.Annotatef(err, "restore partition column %d", i)
			}
		}
		ctx.WritePlain(")")
	}
	if n.Tp == model.PartitionTypeKey {
		ctx.WritePlain("(")
		for i, col := range n.ColumnNames {
			if i > 0 {
				ctx.WritePlain(",")
			}
			if err := restoreQualifiedName(ctx, col); err != nil {
				return errors.Annotatef(err, "An error occurred while splicing PartitionOptions ColumnName: [%v]", i)
			}
		}
		ctx.WritePlain(")")
	}
	if n.HasNum || n.Num > 0 {
		if n.IsSubPartition {
			ctx.WriteKeyword(" SUBPARTITIONS ")
		} else {
			ctx.WriteKeyword(" PARTITIONS ")
		}
		ctx.WritePlainf("%d", n.Num)
	}
	if n.Sub != nil {
		ctx.WritePlain(" ")
		if err := n.Sub.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore PartitionOptions.Sub")
		}
	}

	if len(n.Definitions) > 0 {
		ctx.WritePlain(" (")
		for i, def := range n.Definitions {
			if def == nil {
				return errors.Errorf("restore partition options: nil definition at %d", i)
			}
			if i > 0 {
				ctx.WritePlain(",")
			}
			if err := def.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while splicing PartitionOptions Definitions: [%v]", i)
			}
		}
		ctx.WritePlain(")")
	}

	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *PartitionOptions) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	options, ok := newNode.(*PartitionOptions)
	if !ok || options == nil {
		return newNode, false
	}
	n = options
	if skipChildren {
		return v.Leave(n)
	}
	if n.Expr != nil {
		if !nodePresent(n.Expr) {
			return n, false
		}
		node, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		expr, ok := node.(ExprNode)
		if !ok || !nodePresent(expr) {
			return n, false
		}
		n.Expr = expr
	}
	if n.Sub != nil {
		node, ok := n.Sub.Accept(v)
		if !ok {
			return n, false
		}
		sub, ok := node.(*PartitionOptions)
		if !ok || sub == nil {
			return n, false
		}
		n.Sub = sub
	}
	for i, def := range n.Definitions {
		if def == nil {
			return n, false
		}
		node, ok := def.Accept(v)
		if !ok {
			return n, false
		}
		definition, ok := node.(*PartitionDefinition)
		if !ok || definition == nil {
			return n, false
		}
		n.Definitions[i] = definition
	}
	return v.Leave(n)
}

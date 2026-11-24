package ast

import (
	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
)

var (
	_ DDLNode = &CreateTablespaceStmt{}
	_ DDLNode = &AlterTablespaceStmt{}
	_ DDLNode = &DropTablespaceStmt{}
	_ Node    = &TablespaceOption{}
)

// TablespaceOptionType is the type for tablespace options.
type TablespaceOptionType int

// Tablespace option types.
const (
	TablespaceOptionNone TablespaceOptionType = iota
	TablespaceOptionDataFile
	TablespaceOptionFileBlockSize
	TablespaceOptionExtentSize
	TablespaceOptionInitialSize
	TablespaceOptionAutoextendSize
	TablespaceOptionMaxSize
	TablespaceOptionNodegroup
	TablespaceOptionWait
	TablespaceOptionNoWait
	TablespaceOptionComment
	TablespaceOptionEngine
)

// TablespaceOption represents a single option of a tablespace definition.
type TablespaceOption struct {
	node

	Tp     TablespaceOptionType
	Str    string
	Engine lex.Ident
	Val    uint64
}

// Restore implements Node interface.
func (n *TablespaceOption) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("ast: missing tablespace option")
	}
	if n.Tp == TablespaceOptionEngine {
		if !n.Engine.Present() || n.Str != "" || n.Val != 0 {
			return errors.New("ast: invalid tablespace engine")
		}
	} else if n.Engine.Present() {
		return errors.New("ast: unexpected tablespace engine")
	}
	switch n.Tp {
	case TablespaceOptionDataFile, TablespaceOptionComment:
		if n.Val != 0 {
			return errors.New("ast: unexpected numeric tablespace payload")
		}
	case TablespaceOptionWait, TablespaceOptionNoWait:
		if n.Str != "" || n.Val != 0 {
			return errors.New("ast: unexpected tablespace flag payload")
		}
	default:
		if n.Str != "" {
			return errors.New("ast: unexpected tablespace string payload")
		}
	}
	switch n.Tp {
	case TablespaceOptionDataFile:
		ctx.WriteKeyword("ADD DATAFILE ")
		ctx.WriteString(n.Str)
	case TablespaceOptionFileBlockSize:
		ctx.WriteKeyword("FILE_BLOCK_SIZE = ")
		ctx.WritePlainf("%d", n.Val)
	case TablespaceOptionExtentSize:
		ctx.WriteKeyword("EXTENT_SIZE = ")
		ctx.WritePlainf("%d", n.Val)
	case TablespaceOptionInitialSize:
		ctx.WriteKeyword("INITIAL_SIZE = ")
		ctx.WritePlainf("%d", n.Val)
	case TablespaceOptionAutoextendSize:
		ctx.WriteKeyword("AUTOEXTEND_SIZE = ")
		ctx.WritePlainf("%d", n.Val)
	case TablespaceOptionMaxSize:
		ctx.WriteKeyword("MAX_SIZE = ")
		ctx.WritePlainf("%d", n.Val)
	case TablespaceOptionNodegroup:
		ctx.WriteKeyword("NODEGROUP = ")
		ctx.WritePlainf("%d", n.Val)
	case TablespaceOptionWait:
		ctx.WriteKeyword("WAIT")
	case TablespaceOptionNoWait:
		ctx.WriteKeyword("NO WAIT")
	case TablespaceOptionComment:
		ctx.WriteKeyword("COMMENT = ")
		ctx.WriteString(n.Str)
	case TablespaceOptionEngine:
		ctx.WriteKeyword("ENGINE = ")
		ctx.WriteIdent(n.Engine)
	default:
		return errors.New("An error occurred while splicing TablespaceOption")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *TablespaceOption) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*TablespaceOption)
	if !ok || next == nil {
		return n, false
	}
	n = next
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(n)
}

// CreateTablespaceStmt is a statement to create a tablespace.
// See https://dev.mysql.com/doc/refman/8.0/en/create-tablespace.html
type CreateTablespaceStmt struct {
	ddlNode

	Undo    bool
	Name    lex.Ident
	Options []*TablespaceOption
}

// Restore implements Node interface.
func (n *CreateTablespaceStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("CreateTablespaceStmt")
	}
	if n == nil || !n.Name.Present() {
		return errors.New("ast: missing tablespace name")
	}
	ctx.WriteKeyword("CREATE ")
	if n.Undo {
		ctx.WriteKeyword("UNDO ")
	}
	ctx.WriteKeyword("TABLESPACE ")
	ctx.WriteIdent(n.Name)
	for _, opt := range n.Options {
		ctx.WritePlain(" ")
		if err := opt.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing CreateTablespaceStmt")
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateTablespaceStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*CreateTablespaceStmt)
	if !ok || next == nil {
		return n, false
	}
	n = next
	if skipChildren {
		return v.Leave(newNode)
	}
	for i, opt := range n.Options {
		if opt == nil {
			return n, false
		}
		node, ok := opt.Accept(v)
		if !ok {
			return n, false
		}
		nextOption, ok := node.(*TablespaceOption)
		if !ok || nextOption == nil {
			return n, false
		}
		n.Options[i] = nextOption
	}
	return v.Leave(n)
}

// AlterTablespaceStmt is a statement to alter a tablespace.
// See https://dev.mysql.com/doc/refman/8.0/en/alter-tablespace.html
type AlterTablespaceStmt struct {
	ddlNode

	Undo     bool
	Name     lex.Ident
	Options  []*TablespaceOption
	RenameTo lex.Ident
}

// Restore implements Node interface.
func (n *AlterTablespaceStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("AlterTablespaceStmt")
	}
	if n == nil || !n.Name.Present() || n.RenameTo.Present() && len(n.Options) != 0 {
		return errors.New("ast: invalid altered tablespace")
	}
	ctx.WriteKeyword("ALTER ")
	if n.Undo {
		ctx.WriteKeyword("UNDO ")
	}
	ctx.WriteKeyword("TABLESPACE ")
	ctx.WriteIdent(n.Name)
	for _, opt := range n.Options {
		ctx.WritePlain(" ")
		if err := opt.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing AlterTablespaceStmt")
		}
	}
	if n.RenameTo.Present() {
		ctx.WriteKeyword(" RENAME TO ")
		ctx.WriteIdent(n.RenameTo)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *AlterTablespaceStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*AlterTablespaceStmt)
	if !ok || next == nil {
		return n, false
	}
	n = next
	if skipChildren {
		return v.Leave(newNode)
	}
	for i, opt := range n.Options {
		if opt == nil {
			return n, false
		}
		node, ok := opt.Accept(v)
		if !ok {
			return n, false
		}
		nextOption, ok := node.(*TablespaceOption)
		if !ok || nextOption == nil {
			return n, false
		}
		n.Options[i] = nextOption
	}
	return v.Leave(n)
}

// DropTablespaceStmt is a statement to drop a tablespace.
// See https://dev.mysql.com/doc/refman/8.0/en/drop-tablespace.html
type DropTablespaceStmt struct {
	ddlNode

	Undo   bool
	Name   lex.Ident
	Option *TablespaceOption
}

// Restore implements Node interface.
func (n *DropTablespaceStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("DropTablespaceStmt")
	}
	if n == nil || !n.Name.Present() {
		return errors.New("ast: missing tablespace name")
	}
	ctx.WriteKeyword("DROP ")
	if n.Undo {
		ctx.WriteKeyword("UNDO ")
	}
	ctx.WriteKeyword("TABLESPACE ")
	ctx.WriteIdent(n.Name)
	if n.Option != nil {
		ctx.WritePlain(" ")
		if err := n.Option.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing DropTablespaceStmt")
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropTablespaceStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*DropTablespaceStmt)
	if !ok || next == nil {
		return n, false
	}
	n = next
	if skipChildren {
		return v.Leave(newNode)
	}
	if n.Option != nil {
		updated, ok := n.Option.Accept(v)
		if !ok {
			return n, false
		}
		option, ok := updated.(*TablespaceOption)
		if !ok || option == nil {
			return n, false
		}
		n.Option = option
	}
	return v.Leave(n)
}

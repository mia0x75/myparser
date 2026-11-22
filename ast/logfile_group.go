package ast

import (
	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
)

var (
	_ DDLNode = &CreateLogfileGroupStmt{}
	_ DDLNode = &AlterLogfileGroupStmt{}
	_ DDLNode = &DropLogfileGroupStmt{}
	_ Node    = &LogfileGroupOption{}
)

// LogfileGroupOptionType is the type for logfile group options.
type LogfileGroupOptionType int

// Logfile group option types.
const (
	LogfileGroupOptionNone LogfileGroupOptionType = iota
	LogfileGroupOptionInitialSize
	LogfileGroupOptionUndoBufferSize
	LogfileGroupOptionRedoBufferSize
	LogfileGroupOptionNodegroup
	LogfileGroupOptionWait
	LogfileGroupOptionComment
	LogfileGroupOptionEngine
)

// LogfileGroupOption represents a single option of a logfile group definition.
type LogfileGroupOption struct {
	node

	Tp     LogfileGroupOptionType
	Str    string
	Engine lex.Ident
	Val    uint64
}

// Restore implements Node interface.
func (n *LogfileGroupOption) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("ast: missing logfile group option")
	}
	if n.Tp == LogfileGroupOptionEngine {
		if !n.Engine.Present() || n.Str != "" || n.Val != 0 {
			return errors.New("ast: invalid logfile group engine")
		}
	} else if n.Engine.Present() {
		return errors.New("ast: unexpected logfile group engine")
	}
	switch n.Tp {
	case LogfileGroupOptionComment:
		if n.Val != 0 {
			return errors.New("ast: unexpected numeric logfile group payload")
		}
	case LogfileGroupOptionWait:
		if n.Str != "" || n.Val != 0 {
			return errors.New("ast: unexpected logfile group flag payload")
		}
	default:
		if n.Str != "" {
			return errors.New("ast: unexpected logfile group string payload")
		}
	}
	switch n.Tp {
	case LogfileGroupOptionInitialSize:
		ctx.WriteKeyword("INITIAL_SIZE = ")
		ctx.WritePlainf("%d", n.Val)
	case LogfileGroupOptionUndoBufferSize:
		ctx.WriteKeyword("UNDO_BUFFER_SIZE = ")
		ctx.WritePlainf("%d", n.Val)
	case LogfileGroupOptionRedoBufferSize:
		ctx.WriteKeyword("REDO_BUFFER_SIZE = ")
		ctx.WritePlainf("%d", n.Val)
	case LogfileGroupOptionNodegroup:
		ctx.WriteKeyword("NODEGROUP = ")
		ctx.WritePlainf("%d", n.Val)
	case LogfileGroupOptionWait:
		ctx.WriteKeyword("WAIT")
	case LogfileGroupOptionComment:
		ctx.WriteKeyword("COMMENT = ")
		ctx.WriteString(n.Str)
	case LogfileGroupOptionEngine:
		ctx.WriteKeyword("ENGINE = ")
		ctx.WriteIdent(n.Engine)
	default:
		return errors.New("An error occurred while splicing LogfileGroupOption")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *LogfileGroupOption) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*LogfileGroupOption)
	if !ok || next == nil {
		return n, false
	}
	n = next
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(n)
}

func restoreLogfileGroupOptions(ctx *format.RestoreCtx, options []*LogfileGroupOption) error {
	for _, opt := range options {
		ctx.WritePlain(" ")
		if err := opt.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing LogfileGroupOptions")
		}
	}
	return ctx.Err()
}

// CreateLogfileGroupStmt is a statement to create a logfile group.
// See https://dev.mysql.com/doc/refman/8.0/en/create-logfile-group.html
type CreateLogfileGroupStmt struct {
	ddlNode

	Name     lex.Ident
	UndoFile string
	Options  []*LogfileGroupOption
}

// Restore implements Node interface.
func (n *CreateLogfileGroupStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil || !n.Name.Present() {
		return errors.New("ast: missing logfile group name")
	}
	ctx.WriteKeyword("CREATE LOGFILE GROUP ")
	ctx.WriteIdent(n.Name)
	ctx.WriteKeyword(" ADD UNDOFILE ")
	ctx.WriteString(n.UndoFile)
	return restoreLogfileGroupOptions(ctx, n.Options)
}

// Accept implements Node Accept interface.
func (n *CreateLogfileGroupStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*CreateLogfileGroupStmt)
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
		nextOption, ok := node.(*LogfileGroupOption)
		if !ok || nextOption == nil {
			return n, false
		}
		n.Options[i] = nextOption
	}
	return v.Leave(n)
}

// AlterLogfileGroupStmt is a statement to alter a logfile group.
// See https://dev.mysql.com/doc/refman/8.0/en/alter-logfile-group.html
type AlterLogfileGroupStmt struct {
	ddlNode

	Name     lex.Ident
	UndoFile string
	Options  []*LogfileGroupOption
}

// Restore implements Node interface.
func (n *AlterLogfileGroupStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil || !n.Name.Present() {
		return errors.New("ast: missing logfile group name")
	}
	ctx.WriteKeyword("ALTER LOGFILE GROUP ")
	ctx.WriteIdent(n.Name)
	ctx.WriteKeyword(" ADD UNDOFILE ")
	ctx.WriteString(n.UndoFile)
	return restoreLogfileGroupOptions(ctx, n.Options)
}

// Accept implements Node Accept interface.
func (n *AlterLogfileGroupStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*AlterLogfileGroupStmt)
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
		nextOption, ok := node.(*LogfileGroupOption)
		if !ok || nextOption == nil {
			return n, false
		}
		n.Options[i] = nextOption
	}
	return v.Leave(n)
}

// DropLogfileGroupStmt is a statement to drop a logfile group.
// See https://dev.mysql.com/doc/refman/8.0/en/drop-logfile-group.html
type DropLogfileGroupStmt struct {
	ddlNode

	Name    lex.Ident
	Options []*LogfileGroupOption
}

// Restore implements Node interface.
func (n *DropLogfileGroupStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil || !n.Name.Present() {
		return errors.New("ast: missing logfile group name")
	}
	ctx.WriteKeyword("DROP LOGFILE GROUP ")
	ctx.WriteIdent(n.Name)
	return restoreLogfileGroupOptions(ctx, n.Options)
}

// Accept implements Node Accept interface.
func (n *DropLogfileGroupStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*DropLogfileGroupStmt)
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
		nextOption, ok := node.(*LogfileGroupOption)
		if !ok || nextOption == nil {
			return n, false
		}
		n.Options[i] = nextOption
	}
	return v.Leave(n)
}

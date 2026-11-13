package ast

import (
	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/types"
)

var (
	_ DDLNode = &CreateSequenceStmt{}
	_ DDLNode = &DropSequenceStmt{}
	_ Node    = &SequenceOption{}
)

// SequenceOptionType is the type for sequence options.
type SequenceOptionType int

// Sequence option types.
const (
	SequenceOptionNone SequenceOptionType = iota
	SequenceOptionIncrement
	SequenceOptionMinValue
	SequenceOptionNoMinValue
	SequenceOptionMaxValue
	SequenceOptionNoMaxValue
	SequenceOptionStart
	SequenceOptionCache
	SequenceOptionNoCache
	SequenceOptionCycle
	SequenceOptionNoCycle
	SequenceOptionAs
)

// SequenceOption represents a single option of a sequence definition.
type SequenceOption struct {
	node

	Tp SequenceOptionType
	// Val is retained for existing consumers of signed integer options.
	Val int64
	// Value preserves unsigned boundaries and fractional MINVALUE/MAXVALUE
	// literals. Restore prefers it to Val when it is populated.
	Value ExprNode
	// Type is the integer type of an AS option.
	Type *types.FieldType
}

// Restore implements Node interface.
func (n *SequenceOption) Restore(ctx *format.RestoreCtx) error {
	switch n.Tp {
	case SequenceOptionIncrement:
		ctx.WriteKeyWord("INCREMENT BY ")
	case SequenceOptionMinValue:
		ctx.WriteKeyWord("MINVALUE ")
	case SequenceOptionNoMinValue:
		ctx.WriteKeyWord("NO MINVALUE")
	case SequenceOptionMaxValue:
		ctx.WriteKeyWord("MAXVALUE ")
	case SequenceOptionNoMaxValue:
		ctx.WriteKeyWord("NO MAXVALUE")
	case SequenceOptionStart:
		ctx.WriteKeyWord("START WITH ")
	case SequenceOptionCache:
		ctx.WriteKeyWord("CACHE ")
	case SequenceOptionNoCache:
		ctx.WriteKeyWord("NOCACHE")
	case SequenceOptionCycle:
		ctx.WriteKeyWord("CYCLE")
	case SequenceOptionNoCycle:
		ctx.WriteKeyWord("NO CYCLE")
	case SequenceOptionAs:
		ctx.WriteKeyWord("AS ")
		if err := n.Type.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SequenceOption.Type")
		}
	default:
		return errors.New("An error occurred while splicing SequenceOption")
	}
	switch n.Tp {
	case SequenceOptionIncrement, SequenceOptionMinValue, SequenceOptionMaxValue, SequenceOptionStart, SequenceOptionCache:
		if n.Value != nil {
			if err := n.Value.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore SequenceOption.Value")
			}
		} else {
			ctx.WritePlainf("%d", n.Val)
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *SequenceOption) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*SequenceOption)
	if n.Value != nil {
		node, ok := n.Value.Accept(v)
		if !ok {
			return n, false
		}
		n.Value = node.(ExprNode)
	}
	return v.Leave(n)
}

// CreateSequenceStmt is a statement to create a sequence (MariaDB).
// See https://mariadb.com/kb/en/create-sequence/
type CreateSequenceStmt struct {
	ddlNode

	IfNotExists bool
	OrReplace   bool
	Temporary   bool
	Name        *TableName
	Options     []*SequenceOption
}

// Restore implements Node interface.
func (n *CreateSequenceStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("CREATE ")
	if n.OrReplace {
		ctx.WriteKeyWord("OR REPLACE ")
	}
	if n.Temporary {
		ctx.WriteKeyWord("TEMPORARY ")
	}
	ctx.WriteKeyWord("SEQUENCE ")
	if n.IfNotExists {
		ctx.WriteKeyWord("IF NOT EXISTS ")
	}
	if err := n.Name.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while splicing CreateSequenceStmt Name")
	}
	for _, option := range n.Options {
		ctx.WritePlain(" ")
		if err := option.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while splicing CreateSequenceStmt Option")
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *CreateSequenceStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CreateSequenceStmt)
	node, ok := n.Name.Accept(v)
	if !ok {
		return n, false
	}
	n.Name = node.(*TableName)
	for i, opt := range n.Options {
		node, ok := opt.Accept(v)
		if !ok {
			return n, false
		}
		n.Options[i] = node.(*SequenceOption)
	}
	return v.Leave(n)
}

// DropSequenceStmt is a statement to drop one or more sequences (MariaDB).
// See https://mariadb.com/kb/en/drop-sequence/
type DropSequenceStmt struct {
	ddlNode

	IfExists bool
	Names    []*TableName
}

// Restore implements Node interface.
func (n *DropSequenceStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("DROP SEQUENCE ")
	if n.IfExists {
		ctx.WriteKeyWord("IF EXISTS ")
	}
	for i, name := range n.Names {
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := name.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while splicing DropSequenceStmt Names[%d]", i)
		}
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *DropSequenceStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*DropSequenceStmt)
	for i, name := range n.Names {
		node, ok := name.Accept(v)
		if !ok {
			return n, false
		}
		n.Names[i] = node.(*TableName)
	}
	return v.Leave(n)
}

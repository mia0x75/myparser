package ast

import (
	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
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
	Type *TypeName
}

// Restore implements Node interface.
func (n *SequenceOption) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore SequenceOption: nil option")
	}
	switch n.Tp {
	case SequenceOptionIncrement:
		ctx.WriteKeyword("INCREMENT BY ")
	case SequenceOptionMinValue:
		ctx.WriteKeyword("MINVALUE ")
	case SequenceOptionNoMinValue:
		ctx.WriteKeyword("NO MINVALUE")
	case SequenceOptionMaxValue:
		ctx.WriteKeyword("MAXVALUE ")
	case SequenceOptionNoMaxValue:
		ctx.WriteKeyword("NO MAXVALUE")
	case SequenceOptionStart:
		ctx.WriteKeyword("START WITH ")
	case SequenceOptionCache:
		ctx.WriteKeyword("CACHE ")
	case SequenceOptionNoCache:
		ctx.WriteKeyword("NOCACHE")
	case SequenceOptionCycle:
		ctx.WriteKeyword("CYCLE")
	case SequenceOptionNoCycle:
		ctx.WriteKeyword("NO CYCLE")
	case SequenceOptionAs:
		if n.Type == nil {
			return errors.New("restore SequenceOption: missing AS type")
		}
		ctx.WriteKeyword("AS ")
		if err := n.Type.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore SequenceOption.Type")
		}
	default:
		return errors.New("An error occurred while splicing SequenceOption")
	}
	switch n.Tp {
	case SequenceOptionIncrement, SequenceOptionMinValue, SequenceOptionMaxValue, SequenceOptionStart, SequenceOptionCache:
		if n.Value != nil {
			if !nodePresent(n.Value) {
				return errors.New("restore SequenceOption: nil value")
			}
			if err := n.Value.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore SequenceOption.Value")
			}
		} else {
			ctx.WritePlainf("%d", n.Val)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *SequenceOption) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	entered, ok := nodeAs[*SequenceOption](newNode)
	if !ok {
		return n, false
	}
	if skipChildren {
		return v.Leave(entered)
	}
	n = entered
	if n.Type != nil {
		updatedType, ok := acceptNode[*TypeName](v, n.Type)
		if !ok {
			return n, false
		}
		n.Type = updatedType
	}
	if n.Value != nil {
		updated, ok := acceptNode[ExprNode](v, n.Value)
		if !ok {
			return n, false
		}
		n.Value = updated
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
	if n == nil || n.Name == nil {
		return errors.New("restore CreateSequenceStmt: missing name")
	}
	ctx.WriteKeyword("CREATE ")
	if n.OrReplace {
		ctx.WriteKeyword("OR REPLACE ")
	}
	if n.Temporary {
		ctx.WriteKeyword("TEMPORARY ")
	}
	ctx.WriteKeyword("SEQUENCE ")
	if n.IfNotExists {
		ctx.WriteKeyword("IF NOT EXISTS ")
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
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateSequenceStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	entered, ok := nodeAs[*CreateSequenceStmt](newNode)
	if !ok {
		return n, false
	}
	if skipChildren {
		return v.Leave(entered)
	}
	n = entered
	updated, ok := acceptNode[*TableName](v, n.Name)
	if !ok {
		return n, false
	}
	n.Name = updated
	for i, opt := range n.Options {
		updated, ok := acceptNode[*SequenceOption](v, opt)
		if !ok {
			return n, false
		}
		n.Options[i] = updated
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
	if n == nil {
		return errors.New("restore DropSequenceStmt: nil statement")
	}
	ctx.WriteKeyword("DROP SEQUENCE ")
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}
	for i, name := range n.Names {
		if name == nil {
			return errors.Errorf("restore DropSequenceStmt.Names[%d]: missing name", i)
		}
		if i > 0 {
			ctx.WritePlain(", ")
		}
		if err := name.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while splicing DropSequenceStmt Names[%d]", i)
		}
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropSequenceStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	entered, ok := nodeAs[*DropSequenceStmt](newNode)
	if !ok {
		return n, false
	}
	if skipChildren {
		return v.Leave(entered)
	}
	n = entered
	for i, name := range n.Names {
		updated, ok := acceptNode[*TableName](v, name)
		if !ok {
			return n, false
		}
		n.Names[i] = updated
	}
	return v.Leave(n)
}

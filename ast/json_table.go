package ast

import (
	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/model"
	"github.com/mia0x75/myparser/types"
)

// JSON_TABLE behavior kinds for missing values and conversion errors.
const (
	JSONTableBehaviorValue = iota
	JSONTableBehaviorDefault
	JSONTableBehaviorError
)

// JSONTable converts a JSON expression to a table using column path definitions.
type JSONTable struct {
	node

	Expr    ExprNode
	Path    string
	Columns []*JSONTableColumn
}

func (n *JSONTable) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("JSON_TABLE")
	ctx.WritePlain("(")
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore JSONTable.Expr")
	}
	ctx.WritePlain(", ")
	ctx.WriteString(n.Path)
	ctx.WritePlain(" COLUMNS (")
	if err := restoreJSONTableColumns(ctx, n.Columns); err != nil {
		return err
	}
	ctx.WritePlain("))")
	return nil
}

func (n *JSONTable) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*JSONTable)
	if n.Expr != nil {
		node, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		n.Expr = node.(ExprNode)
	}
	for i, column := range n.Columns {
		node, ok := column.Accept(v)
		if !ok {
			return n, false
		}
		n.Columns[i] = node.(*JSONTableColumn)
	}
	return v.Leave(n)
}

// JSONTableColumn is an ordinality, path, exists, or nested path column.
type JSONTableColumn struct {
	node

	Name       model.CIStr
	Tp         *types.FieldType
	Path       string
	Exists     bool
	Ordinality bool
	Nested     bool
	Columns    []*JSONTableColumn
	OnEmpty    *JSONTableBehavior
	OnError    *JSONTableBehavior
}

func (c *JSONTableColumn) Restore(ctx *format.RestoreCtx) error {
	switch {
	case c.Ordinality:
		ctx.WriteName(c.Name.O)
		ctx.WriteKeyWord(" FOR ORDINALITY")
	case c.Nested:
		ctx.WriteKeyWord("NESTED PATH ")
		ctx.WriteString(c.Path)
		ctx.WriteKeyWord(" COLUMNS (")
		if err := restoreJSONTableColumns(ctx, c.Columns); err != nil {
			return err
		}
		ctx.WritePlain(")")
	default:
		ctx.WriteName(c.Name.O)
		if c.Tp != nil {
			ctx.WritePlain(" ")
			if err := c.Tp.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore JSONTableColumn.Type")
			}
		}
		if c.Exists {
			ctx.WriteKeyWord(" EXISTS PATH ")
		} else {
			ctx.WriteKeyWord(" PATH ")
		}
		ctx.WriteString(c.Path)
		if err := c.restoreJSONTableBehaviors(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Accept visits nested columns and the default expressions of behavior clauses.
func (c *JSONTableColumn) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(c)
	if skipChildren {
		return v.Leave(newNode)
	}
	c = newNode.(*JSONTableColumn)
	for i, column := range c.Columns {
		node, ok := column.Accept(v)
		if !ok {
			return c, false
		}
		c.Columns[i] = node.(*JSONTableColumn)
	}
	for _, behavior := range []*JSONTableBehavior{c.OnEmpty, c.OnError} {
		if behavior != nil && behavior.Value != nil {
			node, ok := behavior.Value.Accept(v)
			if !ok {
				return c, false
			}
			behavior.Value = node.(ExprNode)
		}
	}
	return v.Leave(c)
}

func (c *JSONTableColumn) restoreJSONTableBehaviors(ctx *format.RestoreCtx) error {
	if c.OnEmpty != nil {
		if err := c.OnEmpty.Restore(ctx, true); err != nil {
			return err
		}
	}
	if c.OnError != nil {
		if err := c.OnError.Restore(ctx, false); err != nil {
			return err
		}
	}
	return nil
}

func restoreJSONTableColumns(ctx *format.RestoreCtx, cols []*JSONTableColumn) error {
	for i, col := range cols {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := col.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore JSONTable.Column[%d]", i)
		}
	}
	return nil
}

// JSONTableBehavior specifies NULL, ERROR, or DEFAULT on an empty path or error.
type JSONTableBehavior struct {
	Tp      int
	OnEmpty bool
	Value   ExprNode
}

func (b *JSONTableBehavior) Restore(ctx *format.RestoreCtx, onEmpty bool) error {
	ctx.WritePlain(" ")
	switch b.Tp {
	case JSONTableBehaviorError:
		ctx.WriteKeyWord("ERROR")
	case JSONTableBehaviorDefault:
		ctx.WriteKeyWord("DEFAULT ")
		if err := b.Value.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore JSONTableBehavior.Default")
		}
	default:
		if b.Value == nil {
			ctx.WriteKeyWord("NULL")
		} else if err := b.Value.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore JSONTableBehavior.Value")
		}
	}
	if onEmpty {
		ctx.WriteKeyWord(" ON EMPTY")
	} else {
		ctx.WriteKeyWord(" ON ERROR")
	}
	return nil
}

// NewJSONTableColumnBehavior constructs a column behavior clause.
func NewJSONTableColumnBehavior(tp int, onEmpty bool, value ExprNode) *JSONTableBehavior {
	return &JSONTableBehavior{Tp: tp, OnEmpty: onEmpty, Value: value}
}

// AssignJSONTableBehavior assigns a clause to its ON EMPTY or ON ERROR field.
func AssignJSONTableBehavior(col *JSONTableColumn, behavior *JSONTableBehavior) {
	if col == nil || behavior == nil {
		return
	}
	if behavior.OnEmpty {
		col.OnEmpty = behavior
		return
	}
	col.OnError = behavior
}

// AssignJSONTableBehaviors assigns the column's behavior clauses.
func AssignJSONTableBehaviors(col *JSONTableColumn, behaviors []*JSONTableBehavior) {
	if col == nil {
		return
	}
	for _, b := range behaviors {
		AssignJSONTableBehavior(col, b)
	}
}

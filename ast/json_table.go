package ast

import (
	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

// JSON_TABLE column roles keep mutually exclusive syntax separate.
type JSONTableColumnKind int

const (
	JSONTableColumnUnknown JSONTableColumnKind = iota
	JSONTableColumnOrdinality
	JSONTableColumnPath
	JSONTableColumnExists
	JSONTableColumnNested
)

// JSON_TABLE behavior kinds for missing values and conversion errors.
type JSONTableBehaviorKind int

const (
	JSONTableBehaviorUnknown JSONTableBehaviorKind = iota
	JSONTableBehaviorNull
	JSONTableBehaviorError
	JSONTableBehaviorDefault
)

// JSON_TABLE behavior targets.
type JSONTableBehaviorTarget int

const (
	JSONTableBehaviorOnUnknown JSONTableBehaviorTarget = iota
	JSONTableBehaviorOnEmpty
	JSONTableBehaviorOnError
)

// JSONTable converts a JSON expression to a table using column path definitions.
type JSONTable struct {
	resultSetNode

	Expr    ExprNode
	Path    string
	Columns []*JSONTableColumn
}

// JSONTable is a result-set source.
var _ ResultSetNode = (*JSONTable)(nil)

func (n *JSONTable) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("restore JSONTable: nil node")
	}
	if !nodePresent(n.Expr) || len(n.Columns) == 0 {
		return errors.New("restore JSONTable: missing expression")
	}
	ctx.WriteKeyword("JSON_TABLE")
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
	return ctx.Err()
}

func (n *JSONTable) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	table, ok := newNode.(*JSONTable)
	if !ok || table == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(newNode)
	}
	if !nodePresent(table.Expr) || len(table.Columns) == 0 {
		return table, false
	}
	expr, ok := table.Expr.Accept(v)
	if !ok {
		return table, false
	}
	updated, ok := expr.(ExprNode)
	if !ok || !nodePresent(updated) {
		return table, false
	}
	table.Expr = updated
	for i, column := range table.Columns {
		if column == nil {
			return table, false
		}
		node, ok := column.Accept(v)
		if !ok {
			return table, false
		}
		updated, ok := node.(*JSONTableColumn)
		if !ok || updated == nil {
			return table, false
		}
		table.Columns[i] = updated
	}
	return v.Leave(table)
}

// JSONTableColumn is an ordinality, path, exists, or nested path column.
type JSONTableColumn struct {
	node

	Name        lex.Ident
	Type        pkast.TypeNode
	Path        string
	Columns     []*JSONTableColumn
	Kind        JSONTableColumnKind
	PathKeyword bool
	Behaviors   []*JSONTableBehavior
}

var _ Node = (*JSONTableColumn)(nil)

func (c *JSONTableColumn) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c == nil {
		return errors.New("restore JSONTableColumn: nil node")
	}
	if err := c.validate(); err != nil {
		return err
	}
	switch c.Kind {
	case JSONTableColumnOrdinality:
		ctx.WriteIdent(c.Name)
		ctx.WriteKeyword(" FOR ORDINALITY")
		return ctx.Err()
	case JSONTableColumnNested:
		if c.PathKeyword {
			ctx.WriteKeyword("NESTED PATH ")
		} else {
			ctx.WriteKeyword("NESTED ")
		}
		ctx.WriteString(c.Path)
		ctx.WriteKeyword(" COLUMNS (")
		if err := restoreJSONTableColumns(ctx, c.Columns); err != nil {
			return err
		}
		ctx.WritePlain(")")
		return ctx.Err()
	default:
		ctx.WriteIdent(c.Name)
		ctx.WritePlain(" ")
		if err := c.Type.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore JSONTableColumn.Type")
		}
		if c.Kind == JSONTableColumnExists {
			ctx.WriteKeyword(" EXISTS PATH ")
		} else {
			ctx.WriteKeyword(" PATH ")
		}
		ctx.WriteString(c.Path)
		for _, behavior := range c.Behaviors {
			if err := behavior.Restore(ctx); err != nil {
				return err
			}
		}
		return ctx.Err()
	}
}

func (c *JSONTableColumn) validate() error {
	switch c.Kind {
	case JSONTableColumnOrdinality:
		if !c.Name.Present() || c.Type != nil || c.Path != "" || c.PathKeyword || len(c.Columns) != 0 || len(c.Behaviors) != 0 {
			return errors.New("ast: invalid JSON_TABLE ordinality column")
		}
	case JSONTableColumnPath, JSONTableColumnExists:
		if !c.Name.Present() || !nodePresent(c.Type) || c.Type.TypeCore() == nil || c.PathKeyword || len(c.Columns) != 0 {
			return errors.New("ast: invalid JSON_TABLE path column")
		}
	case JSONTableColumnNested:
		if c.Name.Present() || c.Type != nil || len(c.Behaviors) != 0 || len(c.Columns) == 0 {
			return errors.New("ast: invalid JSON_TABLE nested column")
		}
	default:
		return errors.New("ast: unknown JSON_TABLE column kind")
	}
	seen := [3]bool{}
	for _, behavior := range c.Behaviors {
		if behavior == nil || behavior.validate() != nil {
			return errors.New("ast: invalid JSON_TABLE behavior")
		}
		if seen[behavior.On] {
			return errors.New("ast: duplicate JSON_TABLE behavior target")
		}
		seen[behavior.On] = true
	}
	return nil
}

// Accept visits the type, nested columns, and behavior clauses in source order.
func (c *JSONTableColumn) Accept(v Visitor) (Node, bool) {
	if c == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(c)
	column, ok := newNode.(*JSONTableColumn)
	if !ok || column == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(newNode)
	}
	if column.validate() != nil {
		return column, false
	}
	if column.Type != nil {
		node, ok := column.Type.Accept(v)
		if !ok {
			return column, false
		}
		updated, ok := node.(pkast.TypeNode)
		if !ok || !nodePresent(updated) || updated.TypeCore() == nil {
			return column, false
		}
		column.Type = updated
	}
	for i, child := range column.Columns {
		if child == nil {
			return column, false
		}
		node, ok := child.Accept(v)
		if !ok {
			return column, false
		}
		updated, ok := node.(*JSONTableColumn)
		if !ok || updated == nil {
			return column, false
		}
		column.Columns[i] = updated
	}
	for i, behavior := range column.Behaviors {
		if behavior == nil {
			return column, false
		}
		node, ok := behavior.Accept(v)
		if !ok {
			return column, false
		}
		updated, ok := node.(*JSONTableBehavior)
		if !ok || updated == nil {
			return column, false
		}
		column.Behaviors[i] = updated
	}
	return v.Leave(column)
}

func restoreJSONTableColumns(ctx *format.RestoreCtx, cols []*JSONTableColumn) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(cols) == 0 {
		return errors.New("ast: missing JSON_TABLE columns")
	}
	for i, col := range cols {
		if col == nil {
			return errors.Errorf("restore JSONTable: nil column at %d", i)
		}
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := col.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore JSONTable.Column[%d]", i)
		}
	}
	return ctx.Err()
}

// JSONTableBehavior specifies NULL, ERROR, or DEFAULT on an empty path or error.
type JSONTableBehavior struct {
	node

	Kind  JSONTableBehaviorKind
	On    JSONTableBehaviorTarget
	Value ExprNode
}

var _ Node = (*JSONTableBehavior)(nil)

func (b *JSONTableBehavior) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if b == nil {
		return errors.New("restore JSONTableBehavior: nil node")
	}
	if err := b.validate(); err != nil {
		return err
	}
	switch b.Kind {
	case JSONTableBehaviorNull:
		ctx.WritePlain(" ")
		ctx.WriteKeyword("NULL")
	case JSONTableBehaviorError:
		ctx.WritePlain(" ")
		ctx.WriteKeyword("ERROR")
	case JSONTableBehaviorDefault:
		ctx.WritePlain(" ")
		ctx.WriteKeyword("DEFAULT ")
		if err := b.Value.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore JSONTableBehavior.Value")
		}
	default:
		return errors.Errorf("restore JSONTableBehavior: unknown kind %d", b.Kind)
	}
	switch b.On {
	case JSONTableBehaviorOnEmpty:
		ctx.WriteKeyword(" ON EMPTY")
	case JSONTableBehaviorOnError:
		ctx.WriteKeyword(" ON ERROR")
	default:
		return errors.Errorf("restore JSONTableBehavior: unknown target %d", b.On)
	}
	return ctx.Err()
}

func (b *JSONTableBehavior) validate() error {
	if b == nil {
		return errors.New("ast: nil JSON_TABLE behavior")
	}
	switch b.Kind {
	case JSONTableBehaviorNull, JSONTableBehaviorError:
		if b.Value != nil {
			return errors.New("ast: non-default JSON_TABLE behavior carries a value")
		}
	case JSONTableBehaviorDefault:
		if !nodePresent(b.Value) {
			return errors.New("ast: default JSON_TABLE behavior requires a value")
		}
	default:
		return errors.New("ast: unknown JSON_TABLE behavior kind")
	}
	if b.On != JSONTableBehaviorOnEmpty && b.On != JSONTableBehaviorOnError {
		return errors.New("ast: unknown JSON_TABLE behavior target")
	}
	return nil
}

func (b *JSONTableBehavior) Accept(v Visitor) (Node, bool) {
	if b == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(b)
	behavior, ok := newNode.(*JSONTableBehavior)
	if !ok || behavior == nil {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(newNode)
	}
	if behavior.validate() != nil {
		return behavior, false
	}
	if behavior.Value != nil {
		node, ok := behavior.Value.Accept(v)
		if !ok {
			return behavior, false
		}
		updated, ok := node.(ExprNode)
		if !ok || !nodePresent(updated) {
			return behavior, false
		}
		behavior.Value = updated
	}
	return v.Leave(behavior)
}

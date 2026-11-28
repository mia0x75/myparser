package ast

import (
	"strings"

	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
)

var (
	_ DDLNode = &CreateServerStmt{}
	_ DDLNode = &AlterServerStmt{}
	_ DDLNode = &DropServerStmt{}
	_ Node    = &ServerOption{}
)

// ServerOption represents a single option of a foreign server definition.
type ServerOption struct {
	node

	Name  lex.Ident
	Value string
}

// Restore implements Node interface.
func (n *ServerOption) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("ServerOption")
	}
	if n == nil || !n.Name.Present() {
		return errors.New("ast: missing server option name")
	}
	if n.Name.Quote == lex.QuoteUnquoted && strings.EqualFold(n.Name.Name, "DATABASE") {
		ctx.WriteKeyword("DATABASE")
	} else {
		ctx.WriteIdent(n.Name)
	}
	ctx.WritePlain(" ")
	ctx.WriteString(n.Value)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ServerOption) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*ServerOption)
	if !ok || next == nil {
		return n, false
	}
	n = next
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(n)
}

// CreateServerStmt is a statement to create a foreign server.
// See https://dev.mysql.com/doc/refman/8.0/en/create-server.html
type CreateServerStmt struct {
	ddlNode

	Name    lex.Ident
	Wrapper string
	Options []*ServerOption
}

// Restore implements Node interface.
func (n *CreateServerStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("CreateServerStmt")
	}
	if n == nil || !n.Name.Present() {
		return errors.New("ast: missing server name")
	}
	ctx.WriteKeyword("CREATE SERVER ")
	ctx.WriteIdent(n.Name)
	ctx.WriteKeyword(" FOREIGN DATA WRAPPER ")
	ctx.WriteString(n.Wrapper)
	if len(n.Options) > 0 {
		ctx.WriteKeyword(" OPTIONS(")
		for i, opt := range n.Options {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := opt.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while splicing CreateServerStmt")
			}
		}
		ctx.WritePlain(")")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateServerStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*CreateServerStmt)
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
		nextOption, ok := node.(*ServerOption)
		if !ok || nextOption == nil {
			return n, false
		}
		n.Options[i] = nextOption
	}
	return v.Leave(n)
}

// AlterServerStmt is a statement to alter a foreign server.
// See https://dev.mysql.com/doc/refman/8.0/en/alter-server.html
type AlterServerStmt struct {
	ddlNode

	Name    lex.Ident
	Options []*ServerOption
}

// Restore implements Node interface.
func (n *AlterServerStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("AlterServerStmt")
	}
	if n == nil || !n.Name.Present() {
		return errors.New("ast: missing server name")
	}
	ctx.WriteKeyword("ALTER SERVER ")
	ctx.WriteIdent(n.Name)
	if len(n.Options) > 0 {
		ctx.WriteKeyword(" OPTIONS(")
		for i, opt := range n.Options {
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := opt.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while splicing AlterServerStmt")
			}
		}
		ctx.WritePlain(")")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *AlterServerStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*AlterServerStmt)
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
		nextOption, ok := node.(*ServerOption)
		if !ok || nextOption == nil {
			return n, false
		}
		n.Options[i] = nextOption
	}
	return v.Leave(n)
}

// DropServerStmt is a statement to drop a foreign server.
// See https://dev.mysql.com/doc/refman/8.0/en/drop-server.html
type DropServerStmt struct {
	ddlNode

	IfExists bool
	Name     lex.Ident
}

// Restore implements Node interface.
func (n *DropServerStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("DropServerStmt")
	}
	if n == nil || !n.Name.Present() {
		return errors.New("ast: missing server name")
	}
	ctx.WriteKeyword("DROP SERVER ")
	if n.IfExists {
		ctx.WriteKeyword("IF EXISTS ")
	}
	ctx.WriteIdent(n.Name)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropServerStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*DropServerStmt)
	if !ok || next == nil {
		return n, false
	}
	n = next
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(n)
}

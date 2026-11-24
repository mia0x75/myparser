package ast

import (
	"strings"

	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
)

var (
	_ DDLNode = &CreateResourceGroupStmt{}
	_ DDLNode = &AlterResourceGroupStmt{}
	_ DDLNode = &DropResourceGroupStmt{}
	_ Node    = &ResourceGroupVCPU{}
)

// Resource group states.
const (
	ResourceGroupStateNone = iota
	ResourceGroupStateEnable
	ResourceGroupStateDisable
)

// ResourceGroupVCPU is a single VCPU range in a resource group definition.
type ResourceGroupVCPU struct {
	node

	From uint64
	To   uint64
}

// Restore implements Node interface.
func (n *ResourceGroupVCPU) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("ast: missing resource group VCPU")
	}
	if n.From == n.To {
		ctx.WritePlainf("%d", n.From)
	} else {
		ctx.WritePlainf("%d-%d", n.From, n.To)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ResourceGroupVCPU) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*ResourceGroupVCPU)
	if !ok || next == nil {
		return n, false
	}
	n = next
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(n)
}

func restoreVCPUList(ctx *format.RestoreCtx, vcpu []*ResourceGroupVCPU) error {
	ctx.WriteKeyword(" VCPU = ")
	for i, v := range vcpu {
		if i > 0 {
			ctx.WritePlain(",")
		}
		if err := v.Restore(ctx); err != nil {
			return err
		}
	}
	return ctx.Err()
}

func restoreResourceGroupOptional(ctx *format.RestoreCtx, tp *int, state int) {
	if tp != nil {
		ctx.WriteKeyword(" THREAD_PRIORITY = ")
		ctx.WritePlainf("%d", *tp)
	}
	switch state {
	case ResourceGroupStateEnable:
		ctx.WriteKeyword(" ENABLE")
	case ResourceGroupStateDisable:
		ctx.WriteKeyword(" DISABLE")
	}
}

// CreateResourceGroupStmt is a statement to create a resource group.
// See https://dev.mysql.com/doc/refman/8.0/en/create-resource-group.html
type CreateResourceGroupStmt struct {
	ddlNode

	Name           lex.Ident
	Type           lex.Ident
	VCPU           []*ResourceGroupVCPU
	ThreadPriority *int
	State          int
}

// Restore implements Node interface.
func (n *CreateResourceGroupStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("CreateResourceGroupStmt")
	}
	if n == nil || !n.Name.Present() || !n.Type.Present() {
		return errors.New("ast: missing resource group name or type")
	}
	if n.State < ResourceGroupStateNone || n.State > ResourceGroupStateDisable {
		return errors.New("ast: invalid resource group state")
	}
	ctx.WriteKeyword("CREATE RESOURCE GROUP ")
	ctx.WriteIdent(n.Name)
	ctx.WriteKeyword(" TYPE = ")
	if n.Type.Quote == lex.QuoteUnquoted && (strings.EqualFold(n.Type.Name, "SYSTEM") || strings.EqualFold(n.Type.Name, "USER")) {
		ctx.WriteKeyword(n.Type.Name)
	} else {
		ctx.WriteIdent(n.Type)
	}
	if len(n.VCPU) > 0 {
		if err := restoreVCPUList(ctx, n.VCPU); err != nil {
			return err
		}
	}
	restoreResourceGroupOptional(ctx, n.ThreadPriority, n.State)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *CreateResourceGroupStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*CreateResourceGroupStmt)
	if !ok || next == nil {
		return n, false
	}
	n = next
	if skipChildren {
		return v.Leave(newNode)
	}
	for i, vcpu := range n.VCPU {
		if vcpu == nil {
			return n, false
		}
		node, ok := vcpu.Accept(v)
		if !ok {
			return n, false
		}
		nextVCPU, ok := node.(*ResourceGroupVCPU)
		if !ok || nextVCPU == nil {
			return n, false
		}
		n.VCPU[i] = nextVCPU
	}
	return v.Leave(n)
}

// AlterResourceGroupStmt is a statement to alter a resource group.
// See https://dev.mysql.com/doc/refman/8.0/en/alter-resource-group.html
type AlterResourceGroupStmt struct {
	ddlNode

	Name           lex.Ident
	VCPU           []*ResourceGroupVCPU
	ThreadPriority *int
	State          int
}

// Restore implements Node interface.
func (n *AlterResourceGroupStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("AlterResourceGroupStmt")
	}
	if n == nil || !n.Name.Present() {
		return errors.New("ast: missing resource group name")
	}
	if n.State < ResourceGroupStateNone || n.State > ResourceGroupStateDisable {
		return errors.New("ast: invalid resource group state")
	}
	ctx.WriteKeyword("ALTER RESOURCE GROUP ")
	ctx.WriteIdent(n.Name)
	if len(n.VCPU) > 0 {
		if err := restoreVCPUList(ctx, n.VCPU); err != nil {
			return err
		}
	}
	restoreResourceGroupOptional(ctx, n.ThreadPriority, n.State)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *AlterResourceGroupStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*AlterResourceGroupStmt)
	if !ok || next == nil {
		return n, false
	}
	n = next
	if skipChildren {
		return v.Leave(newNode)
	}
	for i, vcpu := range n.VCPU {
		if vcpu == nil {
			return n, false
		}
		node, ok := vcpu.Accept(v)
		if !ok {
			return n, false
		}
		nextVCPU, ok := node.(*ResourceGroupVCPU)
		if !ok || nextVCPU == nil {
			return n, false
		}
		n.VCPU[i] = nextVCPU
	}
	return v.Leave(n)
}

// DropResourceGroupStmt is a statement to drop a resource group.
// See https://dev.mysql.com/doc/refman/8.0/en/drop-resource-group.html
type DropResourceGroupStmt struct {
	ddlNode

	Name  lex.Ident
	Force bool
}

// Restore implements Node interface.
func (n *DropResourceGroupStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return nilRestoreError("DropResourceGroupStmt")
	}
	if n == nil || !n.Name.Present() {
		return errors.New("ast: missing resource group name")
	}
	ctx.WriteKeyword("DROP RESOURCE GROUP ")
	ctx.WriteIdent(n.Name)
	if n.Force {
		ctx.WriteKeyword(" FORCE")
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropResourceGroupStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, skipChildren := v.Enter(n)
	next, ok := newNode.(*DropResourceGroupStmt)
	if !ok || next == nil {
		return n, false
	}
	n = next
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(n)
}

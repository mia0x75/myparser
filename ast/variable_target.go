package ast

import (
	"strings"

	"github.com/mia0x75/myparser/format"
)

// VariableTarget is a local variable, a MariaDB ROW field, or a user variable
// in SELECT INTO or FETCH. Parts preserves identifier boundaries: `r.a` is one
// part, whereas r.a has two. IsUser distinguishes @a from the local name `@a`.
type VariableTarget struct {
	node
	Parts  []string
	IsUser bool
}

// LegacyName returns the name used by the legacy string-only target fields.
// Those fields cannot represent quoted dots or distinguish quoted @ names.
func (n *VariableTarget) LegacyName() string {
	name := strings.Join(n.Parts, ".")
	if n.IsUser {
		return "@" + name
	}
	return name
}

// Restore implements Node interface.
func (n *VariableTarget) Restore(ctx *format.RestoreCtx) error {
	if n.IsUser {
		ctx.WritePlain("@")
	}
	for i, part := range n.Parts {
		if i > 0 {
			ctx.WritePlain(".")
		}
		ctx.WriteName(part)
	}
	return nil
}

// Accept implements Node interface.
func (n *VariableTarget) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	return v.Leave(newNode)
}

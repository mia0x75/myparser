package ast

import (
	"github.com/mia0x75/myparser/format"
)

// VariableTargetKind identifies the syntax role of a variable target.
type VariableTargetKind uint8

const (
	VariableTargetLocal VariableTargetKind = iota + 1
	VariableTargetUser
	VariableTargetSystem
	VariableTargetRowField
)

// VariableScope identifies an explicit system-variable scope. The zero value
// means that the input did not spell a scope.
type VariableScope uint8

const (
	VariableScopeImplicit VariableScope = iota
	VariableScopeSession
	VariableScopeGlobal
	VariableScopePersist
	VariableScopePersistOnly
	VariableScopeLocal
)

func (s VariableScope) keyword() string {
	switch s {
	case VariableScopeSession:
		return "SESSION"
	case VariableScopeGlobal:
		return "GLOBAL"
	case VariableScopePersist:
		return "PERSIST"
	case VariableScopePersistOnly:
		return "PERSIST_ONLY"
	case VariableScopeLocal:
		return "SESSION"
	default:
		return ""
	}
}

// VariablePrefix records whether a target was written with @ or @@. Bare
// system-variable assignments use VariablePrefixNone.
type VariablePrefix uint8

const (
	VariablePrefixNone VariablePrefix = iota
	VariablePrefixAt
	VariablePrefixDoubleAt
)

// VariableTarget is the single syntax representation for local variables,
// user variables, system variables, and MariaDB ROW fields.
type VariableTarget struct {
	node

	Name          QualifiedName
	Kind          VariableTargetKind
	Scope         VariableScope
	ScopeExplicit bool
	Prefix        VariablePrefix
}

// Restore implements Node interface.
func (n *VariableTarget) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return nilRestoreError("VariableTarget")
	}
	switch n.Kind {
	case VariableTargetUser:
		ctx.WritePlain("@")
	case VariableTargetSystem:
		switch n.Prefix {
		case VariablePrefixAt:
			ctx.WritePlain("@")
		case VariablePrefixDoubleAt:
			ctx.WritePlain("@@")
			if n.ScopeExplicit {
				ctx.WriteKeyword(n.Scope.keyword())
				ctx.WritePlain(".")
			}
		case VariablePrefixNone:
			ctx.WritePlain("@@")
			scope := n.Scope
			if scope == VariableScopeImplicit {
				scope = VariableScopeSession
			}
			ctx.WriteKeyword(scope.keyword())
			ctx.WritePlain(".")
		}
	case VariableTargetLocal, VariableTargetRowField:
		// No prefix.
	}
	for i, part := range n.Name.Parts {
		if i != 0 {
			ctx.WritePlain(".")
		}
		ctx.WriteIdent(part)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *VariableTarget) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, _ := v.Enter(n)
	target, ok := nodeAs[*VariableTarget](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(target)
}

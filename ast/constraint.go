package ast

import (
	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

type ConstraintType int

const (
	ConstraintNoConstraint ConstraintType = iota
	ConstraintPrimaryKey
	ConstraintKey
	ConstraintIndex
	ConstraintUniq
	ConstraintUniqKey
	ConstraintUniqIndex
	ConstraintForeignKey
	ConstraintFulltext
	ConstraintCheck
	ConstraintSpatial
)

// Constraint retains dialect spelling and index options outside the shared core.
type Constraint struct {
	pkast.BaseConstraint
	Tp        ConstraintType
	IndexName lex.Ident
	Option    *IndexOption
}

var _ pkast.ConstraintNode = (*Constraint)(nil)

func (n *Constraint) ConstraintCore() *pkast.BaseConstraint {
	if n == nil {
		return nil
	}
	return &n.BaseConstraint
}

func (n *Constraint) ConstraintKind() pkast.ConstraintKind {
	if n == nil {
		return pkast.ConstraintKindUnknown
	}
	switch n.Tp {
	case ConstraintPrimaryKey:
		return pkast.ConstraintKindPrimaryKey
	case ConstraintUniq, ConstraintUniqKey, ConstraintUniqIndex:
		return pkast.ConstraintKindUnique
	case ConstraintForeignKey:
		return pkast.ConstraintKindForeignKey
	case ConstraintCheck:
		return pkast.ConstraintKindCheck
	case ConstraintKey, ConstraintIndex, ConstraintFulltext, ConstraintSpatial:
		return pkast.ConstraintKindIndex
	default:
		return pkast.ConstraintKindUnknown
	}
}

func (n *Constraint) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore constraint: nil constraint")
	}
	if n.Tp == ConstraintNoConstraint {
		return ctx.Err()
	}
	var keyword string
	switch n.Tp {
	case ConstraintPrimaryKey:
		keyword = "PRIMARY KEY"
	case ConstraintKey:
		keyword = "KEY"
	case ConstraintIndex:
		keyword = "INDEX"
	case ConstraintUniq:
		keyword = "UNIQUE"
	case ConstraintUniqKey:
		keyword = "UNIQUE KEY"
	case ConstraintUniqIndex:
		keyword = "UNIQUE INDEX"
	case ConstraintForeignKey:
		keyword = "FOREIGN KEY"
	case ConstraintFulltext:
		keyword = "FULLTEXT"
	case ConstraintCheck:
		keyword = "CHECK"
	case ConstraintSpatial:
		keyword = "SPATIAL"
	default:
		return errors.Errorf("restore constraint: invalid type %d", n.Tp)
	}
	if n.Tp == ConstraintCheck {
		if n.Expr == nil {
			return errors.New("restore constraint: missing CHECK expression")
		}
		if len(n.Keys) != 0 || n.Refer != nil || n.IndexName.Present() || n.Option != nil {
			return errors.New("restore constraint: CHECK conflicts with index syntax")
		}
	} else {
		if len(n.Keys) == 0 {
			return errors.New("restore constraint: missing keys")
		}
		if n.Expr != nil {
			return errors.New("restore constraint: unexpected CHECK expression")
		}
		if n.Tp == ConstraintForeignKey {
			if n.Refer == nil || n.Refer.ReferenceDefCore() == nil {
				return errors.New("restore constraint: missing REFERENCES clause")
			}
		} else if n.Refer != nil {
			return errors.New("restore constraint: REFERENCES requires FOREIGN KEY")
		}
	}
	if n.Name.Present() || n.Tp == ConstraintForeignKey {
		ctx.WriteKeyword("CONSTRAINT ")
		if n.Name.Present() {
			ctx.WriteIdent(n.Name)
			ctx.WritePlain(" ")
		}
	}
	ctx.WriteKeyword(keyword)
	if n.IndexName.Present() {
		ctx.WritePlain(" ")
		ctx.WriteIdent(n.IndexName)
	}
	if n.Tp == ConstraintCheck || n.Tp == ConstraintForeignKey {
		ctx.WritePlain(" ")
	}
	ctx.WritePlain("(")
	if n.Tp == ConstraintCheck {
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotate(err, "restore constraint CHECK expression")
		}
	} else {
		for i, key := range n.Keys {
			if key == nil || key.IndexKeyPartCore() == nil {
				return errors.Errorf("restore constraint: nil key at %d", i)
			}
			if i > 0 {
				ctx.WritePlain(", ")
			}
			if err := key.Restore(ctx); err != nil {
				return errors.Annotatef(err, "restore constraint key %d", i)
			}
		}
	}
	ctx.WritePlain(")")
	if n.Refer != nil {
		ctx.WritePlain(" ")
		if err := n.Refer.Restore(ctx); err != nil {
			return errors.Annotate(err, "restore constraint REFERENCES")
		}
	}
	if n.Option != nil {
		ctx.WritePlain(" ")
		if err := n.Option.Restore(ctx); err != nil {
			return errors.Annotate(err, "restore constraint index option")
		}
	}
	return ctx.Err()
}

func (n *Constraint) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*Constraint)
	if !ok || outer == nil {
		return entered, false
	}
	n = outer
	if skip {
		return v.Leave(n)
	}
	for i, key := range n.Keys {
		if key == nil || key.IndexKeyPartCore() == nil {
			return n, false
		}
		updated, ok := key.Accept(v)
		if !ok {
			return n, false
		}
		replacement, ok := updated.(pkast.IndexKeyPartNode)
		if !ok || replacement == nil || replacement.IndexKeyPartCore() == nil {
			return n, false
		}
		n.Keys[i] = replacement
	}
	if n.Expr != nil {
		updated, ok := n.Expr.Accept(v)
		if !ok {
			return n, false
		}
		expr, ok := updated.(ExprNode)
		if !ok || expr == nil {
			return n, false
		}
		n.Expr = expr
	}
	if n.Refer != nil {
		if n.Refer.ReferenceDefCore() == nil {
			return n, false
		}
		updated, ok := n.Refer.Accept(v)
		if !ok {
			return n, false
		}
		ref, ok := updated.(pkast.ReferenceDefNode)
		if !ok || ref == nil || ref.ReferenceDefCore() == nil {
			return n, false
		}
		n.Refer = ref
	}
	if n.Option != nil {
		updated, ok := n.Option.Accept(v)
		if !ok {
			return n, false
		}
		option, ok := updated.(*IndexOption)
		if !ok || option == nil {
			return n, false
		}
		n.Option = option
	}
	return v.Leave(n)
}

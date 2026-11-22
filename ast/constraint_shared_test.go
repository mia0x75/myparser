package ast_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

func TestConstraintSharedKind(t *testing.T) {
	for _, tc := range []struct {
		tp   ast.ConstraintType
		kind pkast.ConstraintKind
	}{
		{ast.ConstraintNoConstraint, pkast.ConstraintKindUnknown},
		{ast.ConstraintPrimaryKey, pkast.ConstraintKindPrimaryKey},
		{ast.ConstraintUniq, pkast.ConstraintKindUnique},
		{ast.ConstraintUniqKey, pkast.ConstraintKindUnique},
		{ast.ConstraintUniqIndex, pkast.ConstraintKindUnique},
		{ast.ConstraintForeignKey, pkast.ConstraintKindForeignKey},
		{ast.ConstraintCheck, pkast.ConstraintKindCheck},
		{ast.ConstraintKey, pkast.ConstraintKindIndex},
		{ast.ConstraintIndex, pkast.ConstraintKindIndex},
		{ast.ConstraintFulltext, pkast.ConstraintKindIndex},
		{ast.ConstraintSpatial, pkast.ConstraintKindIndex},
		{ast.ConstraintType(999), pkast.ConstraintKindUnknown},
	} {
		c := &ast.Constraint{Tp: tc.tp}
		if c.ConstraintKind() != tc.kind || c.ConstraintCore() != &c.BaseConstraint {
			t.Fatalf("type %d maps to %d, want %d", tc.tp, c.ConstraintKind(), tc.kind)
		}
	}
	var missing *ast.Constraint
	if missing.ConstraintCore() != nil || missing.ConstraintKind() != pkast.ConstraintKindUnknown {
		t.Fatal("nil constraint must expose no core or kind")
	}
}

func TestConstraintSharedRestore(t *testing.T) {
	for _, tc := range []struct {
		name string
		node *ast.Constraint
	}{
		{"nil", nil},
		{"invalid type", &ast.Constraint{Tp: ast.ConstraintType(99)}},
		{"missing CHECK", &ast.Constraint{Tp: ast.ConstraintCheck}},
		{"missing keys", &ast.Constraint{Tp: ast.ConstraintPrimaryKey}},
		{"missing REFERENCES", &ast.Constraint{Tp: ast.ConstraintForeignKey, BaseConstraint: pkast.BaseConstraint{Keys: []pkast.IndexKeyPartNode{&ast.IndexKeyPart{}}}}},
		{"nil key", &ast.Constraint{Tp: ast.ConstraintIndex, BaseConstraint: pkast.BaseConstraint{Keys: []pkast.IndexKeyPartNode{nil}}}},
		{"CHECK with keys", &ast.Constraint{Tp: ast.ConstraintCheck, BaseConstraint: pkast.BaseConstraint{Expr: &indexKeyExpr{}, Keys: []pkast.IndexKeyPartNode{&ast.IndexKeyPart{}}}}},
		{"CHECK with index name", &ast.Constraint{Tp: ast.ConstraintCheck, IndexName: lex.Ident{Name: "idx"}, BaseConstraint: pkast.BaseConstraint{Expr: &indexKeyExpr{}}}},
		{"index with CHECK", &ast.Constraint{Tp: ast.ConstraintIndex, BaseConstraint: pkast.BaseConstraint{Keys: []pkast.IndexKeyPartNode{&ast.IndexKeyPart{}}, Expr: &indexKeyExpr{}}}},
		{"index with REFERENCES", &ast.Constraint{Tp: ast.ConstraintIndex, BaseConstraint: pkast.BaseConstraint{Keys: []pkast.IndexKeyPartNode{&ast.IndexKeyPart{}}, Refer: &ast.ReferenceDef{}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{})); err == nil {
				t.Fatal("invalid constraint restored without an error")
			}
		})
	}
	t.Run("child error and writer error remain visible", func(t *testing.T) {
		marker := errors.New("constraint child failed")
		c := &ast.Constraint{Tp: ast.ConstraintIndex, BaseConstraint: pkast.BaseConstraint{Keys: []pkast.IndexKeyPartNode{&sharedReferenceKeyPart{failRestore: marker}}}}
		if err := c.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{})); !errors.Is(err, marker) {
			t.Fatalf("lost child error: %v", err)
		}
		c.Keys[0] = &ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{Column: qualName("a")}}
		if err := c.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &indexKeyFailWriter{err: marker})); !errors.Is(err, marker) {
			t.Fatalf("lost writer error: %v", err)
		}
	})
}

func TestConstraintSharedVisitor(t *testing.T) {
	key := &ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{Column: qualName("a")}}
	newConstraint := func() *ast.Constraint {
		return &ast.Constraint{Tp: ast.ConstraintIndex, BaseConstraint: pkast.BaseConstraint{Keys: []pkast.IndexKeyPartNode{key}}}
	}
	t.Run("outer replacement follows new children", func(t *testing.T) {
		c := newConstraint()
		child := &ast.IndexKeyPart{}
		replacement := &ast.Constraint{BaseConstraint: pkast.BaseConstraint{Keys: []pkast.IndexKeyPartNode{child}}}
		visited := []pkast.Node{}
		updated, ok := c.Accept(joinVisitor{enter: func(n pkast.Node) (pkast.Node, bool) {
			visited = append(visited, n)
			if n == c {
				return replacement, false
			}
			return n, false
		}, leave: func(n pkast.Node) (pkast.Node, bool) { return n, true }})
		if !ok || updated != replacement || len(visited) != 2 || visited[1] != child {
			t.Fatal("outer replacement did not follow its new children")
		}
	})
	t.Run("skip and stop", func(t *testing.T) {
		c := newConstraint()
		count := 0
		_, ok := c.Accept(joinVisitor{enter: func(n pkast.Node) (pkast.Node, bool) { count++; return n, true }, leave: func(n pkast.Node) (pkast.Node, bool) { return n, false }})
		if ok || count != 1 {
			t.Fatal("skip/stop ignored")
		}
	})
	t.Run("CHECK replacement keeps expression role", func(t *testing.T) {
		expr := &indexKeyExpr{}
		c := &ast.Constraint{Tp: ast.ConstraintCheck, BaseConstraint: pkast.BaseConstraint{Expr: expr}}
		_, ok := c.Accept(joinVisitor{enter: func(n pkast.Node) (pkast.Node, bool) { return n, false }, leave: func(n pkast.Node) (pkast.Node, bool) {
			if n == expr {
				return &ast.TableName{}, true
			}
			return n, true
		}})
		if ok || c.Expr != expr {
			t.Fatal("wrong CHECK replacement polluted the expression")
		}
	})
	t.Run("REFERENCES replacement keeps reference role", func(t *testing.T) {
		ref := &ast.ReferenceDef{BaseReferenceDef: pkast.BaseReferenceDef{Table: &ast.TableName{}}}
		c := newConstraint()
		c.Refer = ref
		_, ok := c.Accept(joinVisitor{enter: func(n pkast.Node) (pkast.Node, bool) { return n, n == ref }, leave: func(n pkast.Node) (pkast.Node, bool) {
			if n == ref {
				return key, true
			}
			return n, true
		}})
		if ok || c.Refer != ref {
			t.Fatal("wrong REFERENCES replacement polluted the reference")
		}
	})
	t.Run("same key role accepted", func(t *testing.T) {
		c := newConstraint()
		custom := &sharedReferenceKeyPart{}
		_, ok := c.Accept(joinVisitor{enter: func(n pkast.Node) (pkast.Node, bool) { return n, false }, leave: func(n pkast.Node) (pkast.Node, bool) {
			if n == key {
				return custom, true
			}
			return n, true
		}})
		if !ok || c.Keys[0] != custom {
			t.Fatal("same-role key replacement rejected")
		}
	})
	t.Run("wrong key role rejected", func(t *testing.T) {
		c := newConstraint()
		_, ok := c.Accept(joinVisitor{enter: func(n pkast.Node) (pkast.Node, bool) { return n, false }, leave: func(n pkast.Node) (pkast.Node, bool) {
			if n == key {
				return &ast.TableName{}, true
			}
			return n, true
		}})
		if ok || c.Keys[0] != key {
			t.Fatal("wrong key role accepted or polluted slot")
		}
	})
	t.Run("parent accepts same constraint role", func(t *testing.T) {
		c := newConstraint()
		custom := &testConstraintRole{}
		stmt := &ast.CreateTableStmt{BaseCreateTableStmt: pkast.BaseCreateTableStmt{Table: &ast.TableName{}, Constraints: []pkast.ConstraintNode{c}}}
		_, ok := stmt.Accept(joinVisitor{enter: func(n pkast.Node) (pkast.Node, bool) { return n, false }, leave: func(n pkast.Node) (pkast.Node, bool) {
			if n == c {
				return custom, true
			}
			return n, true
		}})
		if !ok || stmt.Constraints[0] != custom {
			t.Fatal("parent rejected same constraint role")
		}
	})
	t.Run("parent rejects typed nil constraint", func(t *testing.T) {
		var c *ast.Constraint
		stmt := &ast.CreateTableStmt{BaseCreateTableStmt: pkast.BaseCreateTableStmt{Table: &ast.TableName{}, Constraints: []pkast.ConstraintNode{c}}}
		if _, ok := stmt.Accept(joinVisitor{enter: func(n pkast.Node) (pkast.Node, bool) { return n, false }, leave: func(n pkast.Node) (pkast.Node, bool) { return n, true }}); ok {
			t.Fatal("typed nil constraint accepted")
		}
	})
}

func TestConstraintParentRoles(t *testing.T) {
	c := &ast.Constraint{Tp: ast.ConstraintCheck, BaseConstraint: pkast.BaseConstraint{Expr: &indexKeyExpr{}}}
	for _, tc := range []struct {
		name string
		new  func(pkast.ConstraintNode) ast.Node
		get  func(ast.Node) pkast.ConstraintNode
	}{
		{"create", func(c pkast.ConstraintNode) ast.Node {
			return &ast.CreateTableStmt{BaseCreateTableStmt: pkast.BaseCreateTableStmt{Table: &ast.TableName{}, Constraints: []pkast.ConstraintNode{c}}}
		}, func(n ast.Node) pkast.ConstraintNode { return n.(*ast.CreateTableStmt).Constraints[0] }},
		{"alter", func(c pkast.ConstraintNode) ast.Node {
			return &ast.AlterTableSpec{Tp: ast.AlterTableAddConstraint, Constraint: c}
		}, func(n ast.Node) pkast.ConstraintNode { return n.(*ast.AlterTableSpec).Constraint }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, replacement := range []pkast.Node{&testConstraintRole{}, &ast.TableName{}, (*ast.Constraint)(nil), nil} {
				parent := tc.new(c)
				_, ok := parent.Accept(joinVisitor{enter: func(n pkast.Node) (pkast.Node, bool) { return n, false }, leave: func(n pkast.Node) (pkast.Node, bool) {
					if n == c {
						return replacement, true
					}
					return n, true
				}})
				custom, valid := replacement.(*testConstraintRole)
				if ok != valid || valid && tc.get(parent) != custom || !valid && tc.get(parent) != c {
					t.Fatalf("replacement %T: ok=%v, slot=%T", replacement, ok, tc.get(parent))
				}
			}
			for _, missing := range []pkast.ConstraintNode{nil, (*ast.Constraint)(nil)} {
				if err := tc.new(missing).Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{})); err == nil {
					t.Fatal("missing parent constraint restored without an error")
				}
			}
		})
	}
	ref := &ast.ReferenceDef{}
	for _, replacement := range []pkast.Node{&testReferenceRole{}, &ast.TableName{}, (*ast.ReferenceDef)(nil), nil} {
		option := &ast.ColumnOption{Tp: ast.ColumnOptionReference, Refer: ref}
		_, ok := option.Accept(joinVisitor{enter: func(n pkast.Node) (pkast.Node, bool) { return n, n == ref }, leave: func(n pkast.Node) (pkast.Node, bool) {
			if n == ref {
				return replacement, true
			}
			return n, true
		}})
		custom, valid := replacement.(*testReferenceRole)
		if ok != valid || valid && option.Refer != custom || !valid && option.Refer != ref {
			t.Fatalf("column REFERENCES replacement %T: ok=%v, slot=%T", replacement, ok, option.Refer)
		}
	}
	for _, missing := range []pkast.ReferenceDefNode{nil, (*ast.ReferenceDef)(nil)} {
		option := &ast.ColumnOption{Tp: ast.ColumnOptionReference, Refer: missing}
		if err := option.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{})); err == nil {
			t.Fatal("missing column REFERENCES restored without an error")
		}
	}
}

type testConstraintRole struct{ pkast.BaseConstraint }

func (n *testConstraintRole) ConstraintKind() pkast.ConstraintKind { return pkast.ConstraintKindCheck }
func (n *testConstraintRole) Accept(v pkast.Visitor) (pkast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}
func (n *testConstraintRole) Restore(ctx *format.RestoreCtx) error { return ctx.Err() }

type testReferenceRole struct{ pkast.BaseReferenceDef }

func (n *testReferenceRole) Accept(v pkast.Visitor) (pkast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}
func (n *testReferenceRole) Restore(ctx *format.RestoreCtx) error { return ctx.Err() }

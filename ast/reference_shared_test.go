package ast_test

import (
	"errors"
	"strings"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
)

// TestSharedReferenceDefAccept pins the fixed ReferenceDef traversal
// contract: Table -> Columns -> OnDelete -> OnUpdate order, an Enter
// replacement of the outer node re-enters the new children, skip and stop
// are honored, nil and typed-nil slots stop the walk, and an incompatible
// child Leave replacement is refused without polluting the slot.
func TestSharedReferenceDefAccept(t *testing.T) {
	t.Parallel()

	newColumn := func() *ast.IndexKeyPart {
		return &ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{Column: qualName("a")}}
	}
	newTable := func() *ast.TableName {
		return &ast.TableName{BaseTableRef: pkast.BaseTableRef{Name: qualName("t")}}
	}
	newDef := func() *ast.ReferenceDef {
		return &ast.ReferenceDef{
			BaseReferenceDef: pkast.BaseReferenceDef{Table: newTable(), Columns: []pkast.IndexKeyPartNode{newColumn()}},
			OnDelete:         &ast.OnDeleteOpt{ReferOpt: ast.ReferOptionCascade},
			OnUpdate:         &ast.OnUpdateOpt{ReferOpt: ast.ReferOptionRestrict},
		}
	}

	t.Run("fixed child order", func(t *testing.T) {
		t.Parallel()

		def := newDef()
		def.Columns = append(def.Columns, newColumn())
		visited := []pkast.Node{}
		updated, ok := def.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) {
				visited = append(visited, n)
				return n, false
			},
			leave: func(n pkast.Node) (pkast.Node, bool) { return n, true },
		})
		if !ok || updated != pkast.Node(def) {
			t.Fatalf("traversal stopped or lost the outer def: ok=%v got=%T", ok, updated)
		}
		if len(visited) != 6 {
			t.Fatalf("visited %d nodes, want 6 (def, table, 2 columns, ondelete, onupdate)", len(visited))
		}
		want := []pkast.Node{pkast.Node(def), def.Table, def.Columns[0], def.Columns[1], def.OnDelete, def.OnUpdate}
		for i := range visited {
			if visited[i] != want[i] {
				t.Fatalf("visit %d is %T, want %T", i, visited[i], want[i])
			}
		}
	})

	// Enter must hand back the same concrete type the parent dispatches, so
	// a custom same-role node cannot replace the outer def; the custom node
	// travels through a child slot instead, which the last subtest covers.
	t.Run("enter replacement re-enters its children", func(t *testing.T) {
		t.Parallel()

		def := newDef()
		replacement := newDef()
		visited := []pkast.Node{}
		updated, ok := def.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) {
				visited = append(visited, n)
				if _, isDef := n.(*ast.ReferenceDef); isDef {
					return replacement, false
				}
				return n, false
			},
			leave: func(n pkast.Node) (pkast.Node, bool) { return n, true },
		})
		if !ok || updated != pkast.Node(replacement) {
			t.Fatalf("enter replacement lost: ok=%v got=%T", ok, updated)
		}
		// The dispatch of the replaced outer node does not re-enter the
		// replacement itself: it visits the replacement's children
		// directly.
		if len(visited) != 5 {
			t.Fatalf("visited %d nodes, want 5 (def, table, column, ondelete, onupdate)", len(visited))
		}
		if _, isDef := visited[0].(*ast.ReferenceDef); !isDef {
			t.Fatalf("first visit is %T, want *ast.ReferenceDef", visited[0])
		}
		if visited[0] != pkast.Node(def) {
			t.Fatal("first visit is not the original def")
		}
		if _, isTable := visited[1].(*ast.TableName); !isTable {
			t.Fatalf("second visit is %T, want the replacement table", visited[1])
		}
		if visited[1] == pkast.Node(def.Table) {
			t.Fatal("the original table leaked into the walk")
		}
		if _, isKey := visited[2].(*ast.IndexKeyPart); !isKey {
			t.Fatalf("third visit is %T, want the replacement key column", visited[2])
		}
		if visited[2] == pkast.Node(def.Columns[0]) {
			t.Fatal("the original column leaked into the walk")
		}
	})

	t.Run("skip children", func(t *testing.T) {
		t.Parallel()

		def := newDef()
		visited := []pkast.Node{}
		updated, ok := def.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) {
				visited = append(visited, n)
				return n, n == pkast.Node(def)
			},
			leave: func(n pkast.Node) (pkast.Node, bool) { return n, true },
		})
		if !ok || updated != pkast.Node(def) {
			t.Fatalf("skip path lost the outer def: ok=%v", ok)
		}
		if len(visited) != 1 {
			t.Fatalf("visited %d nodes, want only the outer def", len(visited))
		}
	})

	t.Run("stop at the def", func(t *testing.T) {
		t.Parallel()

		def := newDef()
		visited := []pkast.Node{}
		updated, ok := def.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) {
				visited = append(visited, n)
				return n, false
			},
			leave: func(n pkast.Node) (pkast.Node, bool) {
				// Stop at the outer leave: the child walk is complete, the
				// outer def must not run its second leave.
				if n == pkast.Node(def) {
					return n, false
				}
				return n, true
			},
		})
		if ok || updated != pkast.Node(def) {
			t.Fatalf("stop at the def must end the walk: ok=%v got=%T", ok, updated)
		}
		// The outer leave stops the walk; the enters recorded before that
		// point must be exactly the def, table, column, ondelete, onupdate.
		if len(visited) != 5 {
			t.Fatalf("visited %d nodes, want 5 (def, table, column, ondelete, onupdate)", len(visited))
		}
		if visited[0] != pkast.Node(def) || visited[1] != def.Table || visited[2] != def.Columns[0] || visited[3] != def.OnDelete || visited[4] != def.OnUpdate {
			t.Fatalf("enter order = %v, want def, table, column, ondelete, onupdate", visited)
		}
	})

	t.Run("nil table stops the walk", func(t *testing.T) {
		t.Parallel()

		def := newDef()
		var table pkast.TableRefNode
		def.Table = table
		visited := []pkast.Node{}
		updated, ok := def.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) {
				visited = append(visited, n)
				return n, false
			},
			leave: func(n pkast.Node) (pkast.Node, bool) { return n, true },
		})
		if ok {
			t.Fatal("nil table must stop the walk")
		}
		if updated != pkast.Node(def) {
			t.Fatalf("nil table walk lost the outer def: got %T", updated)
		}
		if len(visited) != 1 {
			t.Fatalf("visited %d nodes, want only the def before the nil table", len(visited))
		}
	})

	t.Run("typed nil table core stops the walk", func(t *testing.T) {
		t.Parallel()

		def := newDef()
		var table *ast.TableName // typed nil: the interface is non-nil, the core is nil
		def.Table = table
		_, ok := def.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) {
				return n, false
			},
			leave: func(n pkast.Node) (pkast.Node, bool) { return n, true },
		})
		if ok {
			t.Fatal("a typed-nil table must stop the walk through the core guard")
		}
	})

	t.Run("nil column stops the walk", func(t *testing.T) {
		t.Parallel()

		def := newDef()
		def.Columns = append(def.Columns, nil)
		_, ok := def.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) {
				return n, false
			},
			leave: func(n pkast.Node) (pkast.Node, bool) { return n, true },
		})
		if ok {
			t.Fatal("a nil column slot must stop the walk")
		}
	})

	t.Run("nil action skips the action visit", func(t *testing.T) {
		t.Parallel()

		def := newDef()
		def.OnDelete = nil
		def.OnUpdate = nil
		visited := []pkast.Node{}
		updated, ok := def.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) {
				visited = append(visited, n)
				return n, false
			},
			leave: func(n pkast.Node) (pkast.Node, bool) { return n, true },
		})
		if !ok || updated != pkast.Node(def) {
			t.Fatalf("nil actions must not stop a valid def: ok=%v", ok)
		}
		if len(visited) != 3 {
			t.Fatalf("visited %d nodes, want 3 (def, table, column)", len(visited))
		}
	})

	t.Run("incompatible action replacement is refused", func(t *testing.T) {
		t.Parallel()

		def := newDef()
		original := def.OnDelete
		wrong := &ast.IndexKeyPart{}
		_, ok := def.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) { return n, false },
			leave: func(n pkast.Node) (pkast.Node, bool) {
				if n == pkast.Node(original) {
					return wrong, true
				}
				return n, true
			},
		})
		if ok {
			t.Fatal("a non-action OnDelete replacement must stop the walk")
		}
		if def.OnDelete != original {
			t.Fatal("the OnDelete slot was polluted by a refused replacement")
		}
	})

	t.Run("incompatible column replacement is refused", func(t *testing.T) {
		t.Parallel()

		def := newDef()
		original := def.Columns[0]
		wrong := &ast.OnDeleteOpt{}
		_, ok := def.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) { return n, false },
			leave: func(n pkast.Node) (pkast.Node, bool) {
				if n == pkast.Node(original) {
					return wrong, true
				}
				return n, true
			},
		})
		if ok {
			t.Fatal("a non-key-column replacement must stop the walk")
		}
		if def.Columns[0] != original {
			t.Fatal("the column slot was polluted by a refused replacement")
		}
	})

	t.Run("a custom same-role key node is accepted in the child slot", func(t *testing.T) {
		t.Parallel()

		def := newDef()
		original := def.Columns[0]
		custom := &sharedReferenceKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{Column: qualName("x")}}
		_, ok := def.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) { return n, false },
			leave: func(n pkast.Node) (pkast.Node, bool) {
				// The parent def accepts any IndexKeyPartNode in its column
				// slot; only the concrete node's own dispatch would reject
				// the custom node, so the replacement must happen on the
				// child leave, where the parent performs the role check.
				if n == pkast.Node(original) {
					return custom, true
				}
				return n, true
			},
		})
		if !ok {
			t.Fatal("a conforming custom key node must travel through the def")
		}
		if def.Columns[0] != pkast.IndexKeyPartNode(custom) {
			t.Fatal("the column slot must hold the custom key node")
		}
	})
}

// sharedReferenceKeyPart is a test-only custom key node that satisfies the
// shared role interface without being the concrete MySQL IndexKeyPart.
type sharedReferenceKeyPart struct {
	pkast.BaseIndexKeyPart
	failRestore error
}

func (n *sharedReferenceKeyPart) IndexKeyPartCore() *pkast.BaseIndexKeyPart {
	if n == nil {
		return nil
	}
	return &n.BaseIndexKeyPart
}

func (n *sharedReferenceKeyPart) Accept(v pkast.Visitor) (pkast.Node, bool) {
	entered, skip := v.Enter(n)
	outer, ok := entered.(*sharedReferenceKeyPart)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	return v.Leave(outer)
}

func (n *sharedReferenceKeyPart) Restore(ctx *format.RestoreCtx) error {
	if n.failRestore != nil {
		return n.failRestore
	}
	return ctx.Err()
}

// TestSharedReferenceDefRestore pins the shared Restore contract: a missing
// table or column reports an error without partial output, and child and
// writer errors propagate as annotated chains.
func TestSharedReferenceDefRestore(t *testing.T) {
	t.Parallel()

	newDef := func() *ast.ReferenceDef {
		return &ast.ReferenceDef{
			BaseReferenceDef: pkast.BaseReferenceDef{
				Table:   &ast.TableName{BaseTableRef: pkast.BaseTableRef{Name: qualName("p")}},
				Columns: []pkast.IndexKeyPartNode{&ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{Column: qualName("a")}}},
			},
		}
	}
	restored := func(n *ast.ReferenceDef) string {
		t.Helper()
		var sb strings.Builder
		if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
			t.Fatalf("restore: %v", err)
		}
		return sb.String()
	}

	t.Run("absent table errors", func(t *testing.T) {
		t.Parallel()

		var def *ast.ReferenceDef
		err := def.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{}))
		if err == nil || !strings.Contains(err.Error(), "table") {
			t.Fatalf("nil def Restore() = %v, want a missing-table error", err)
		}

		def = newDef()
		var table pkast.TableRefNode
		def.Table = table
		err = def.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{}))
		if err == nil || !strings.Contains(err.Error(), "table") {
			t.Fatalf("nil table Restore() = %v, want a missing-table error", err)
		}

		def = newDef()
		var typedNil *ast.TableName
		def.Table = typedNil
		err = def.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{}))
		if err == nil || !strings.Contains(err.Error(), "table") {
			t.Fatalf("typed-nil table Restore() = %v, want a missing-table error", err)
		}
	})

	t.Run("nil column errors", func(t *testing.T) {
		t.Parallel()

		def := newDef()
		def.Columns = []pkast.IndexKeyPartNode{&ast.IndexKeyPart{}, nil}
		err := def.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{}))
		if err == nil || !strings.Contains(err.Error(), "nil column") {
			t.Fatalf("Restore() = %v, want a nil-column error", err)
		}
	})

	t.Run("empty columns error", func(t *testing.T) {
		t.Parallel()

		def := newDef()
		def.Columns = nil
		err := def.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{}))
		if err == nil || !strings.Contains(err.Error(), "column") {
			t.Fatalf("Restore() = %v, want a missing-columns error", err)
		}
	})

	t.Run("child restore error chains through", func(t *testing.T) {
		t.Parallel()

		childErr := errors.New("child column failed")
		def := newDef()
		def.Columns[0] = &sharedReferenceKeyPart{failRestore: childErr}
		err := def.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{}))
		// The dialect's error package wraps non-Caused errors, so the
		// child error is reported by message; the annotation pins the slot.
		if err == nil || !strings.Contains(err.Error(), "child column failed") {
			t.Fatalf("Restore() = %v, want the child error chained", err)
		}
		if !strings.Contains(err.Error(), "column") {
			t.Fatalf("Restore() = %v, want the column annotation", err)
		}
	})

	t.Run("writer error propagates through the def", func(t *testing.T) {
		t.Parallel()

		def := newDef()
		writeErr := errors.New("write failed")
		failCtx := format.NewRestoreCtx(format.DefaultRestoreFlags, &indexKeyFailWriter{err: writeErr})
		err := def.Restore(failCtx)
		if !errors.Is(err, writeErr) {
			t.Fatalf("Restore() = %v, want the writer error", err)
		}
		if !errors.Is(failCtx.Err(), writeErr) {
			t.Fatalf("context error = %v, want the writer error", failCtx.Err())
		}
		_ = restored
	})
}

// TestSharedIndexKeyPartRestore pins the IndexKeyPart Restore branches:
// column spelling with optional prefix, the expression branch which drops
// the column, DESC always on both branches, and the nil-receiver contract.
// parser.y never emits "(expr) DESC", so that combination is a manual-AST
// semantic check only, not a syntax exception.
func TestSharedIndexKeyPartRestore(t *testing.T) {
	t.Parallel()

	t.Run("nil receiver reports an error", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("nil receiver Restore() panicked instead of returning an error: %v", r)
			}
		}()
		var part *ast.IndexKeyPart
		err := part.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{}))
		if err == nil || !strings.Contains(err.Error(), "nil part") {
			t.Errorf("nil receiver Restore() = %v, want a nil-part error", err)
		}
	})

	cases := []struct {
		name string
		part ast.IndexKeyPart
		want string
	}{
		{
			name: "column with prefix",
			part: ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{
				Column:       qualName("world"),
				PrefixLength: 2,
			}},
			want: "`world`(2)",
		},
		{
			name: "expression branch drops the column",
			part: ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{
				Expr: &sharedRefExpr{},
			}},
			want: "(REF-EXPRESSION)",
		},
		{
			// Manual-AST semantic: the writer emits DESC on either branch;
			// the grammar itself never produces this spelling.
			name: "desc survives the expression branch",
			part: ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{
				Expr: &sharedRefExpr{},
				Desc: true,
			}},
			want: "(REF-EXPRESSION) DESC",
		},
		{
			name: "expression conflicts with a column",
			part: ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{
				Column: qualName("a"),
				Expr:   &sharedRefExpr{},
			}},
			want: "restore index key part: expression conflicts with column or prefix length",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var sb strings.Builder
			err := (&tc.part).Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
			if tc.want == "restore index key part: expression conflicts with column or prefix length" {
				if err == nil || err.Error() != tc.want {
					t.Fatalf("Restore() = %v, want the conflict error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Restore() = %v, want no error", err)
			}
			if got := sb.String(); got != tc.want {
				t.Fatalf("Restore() = %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("child expression restore error is annotated", func(t *testing.T) {
		t.Parallel()

		childErr := errors.New("expr restore failed")
		var sb strings.Builder
		node := &ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{Expr: &sharedRefExpr{failRestore: childErr}}}
		err := node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
		// The dialect's error package wraps non-Caused errors, so the
		// child error is reported by message; the annotation pins the slot.
		if err == nil || !strings.Contains(err.Error(), "expr restore failed") || !strings.Contains(err.Error(), "IndexKeyPart expression") {
			t.Fatalf("Restore() = %v, want the annotated expression error", err)
		}
	})
}

// sharedRefExpr is a minimal shared leaf expression for key part tests.
type sharedRefExpr struct {
	pkast.BaseExpr
	failRestore error
}

func (n *sharedRefExpr) Accept(v pkast.Visitor) (pkast.Node, bool) {
	entered, skip := v.Enter(n)
	outer, ok := entered.(*sharedRefExpr)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	return v.Leave(outer)
}

func (n *sharedRefExpr) Restore(ctx *format.RestoreCtx) error {
	if n.failRestore != nil {
		return n.failRestore
	}
	ctx.WritePlain("REF-EXPRESSION")
	return ctx.Err()
}

var _ pkast.ExprNode = (*sharedRefExpr)(nil)

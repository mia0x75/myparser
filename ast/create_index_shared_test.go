package ast_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/model"
)

// TestSharedCreateIndexRestore pins the shared CreateIndexStmt Restore
// contract: a nil receiver, a missing table (including a typed-nil
// TableName) and an empty key list each report an error without emitting
// output; a nil IndexOption is silently omitted; child key restore errors
// chain through with their slot index; and writer errors propagate to the
// returned error and to the context.
func TestSharedCreateIndexRestore(t *testing.T) {
	t.Parallel()

	newKey := func() *ast.IndexKeyPart {
		return &ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{Column: qualName("a")}}
	}
	newTable := func() *ast.TableName {
		return &ast.TableName{BaseTableRef: pkast.BaseTableRef{Name: qualName("t")}}
	}
	newStmt := func() *ast.CreateIndexStmt {
		return &ast.CreateIndexStmt{Table: newTable(), Keys: []pkast.IndexKeyPartNode{newKey()}}
	}

	t.Run("nil receiver errors without panicking", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("nil receiver Restore() panicked instead of returning an error: %v", r)
			}
		}()
		var stmt *ast.CreateIndexStmt
		err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{}))
		if err == nil || !strings.Contains(err.Error(), "missing table") {
			t.Fatalf("nil receiver Restore() = %v, want a missing-table error", err)
		}
	})

	t.Run("nil and typed-nil table error before output", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		stmt.Table = nil
		var out strings.Builder
		err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out))
		if err == nil || !strings.Contains(err.Error(), "table") {
			t.Fatalf("nil table Restore() = %v, want a missing-table error", err)
		}
		if out.Len() != 0 {
			t.Fatalf("nil table emitted %q before the error", out.String())
		}

		stmt = newStmt()
		var typedNil *ast.TableName // typed nil: the interface is non-nil, the pointer is nil
		stmt.Table = typedNil
		err = stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{}))
		if err == nil || !strings.Contains(err.Error(), "table") {
			t.Fatalf("typed-nil table Restore() = %v, want a missing-table error", err)
		}
	})

	t.Run("empty keys error", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		stmt.Keys = nil
		err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{}))
		if err == nil || !strings.Contains(err.Error(), "keys") {
			t.Fatalf("Restore() = %v, want a missing-keys error", err)
		}
	})

	t.Run("nil key errors with the slot index", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		stmt.Keys = append(stmt.Keys, nil)
		err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{}))
		if err == nil || !strings.Contains(err.Error(), "nil key at 1") {
			t.Fatalf("Restore() = %v, want the nil key at index 1", err)
		}
	})

	t.Run("nil index option is omitted from the output", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		stmt.IndexOption = nil
		var out strings.Builder
		if err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
			t.Fatalf("Restore() = %v, want no error", err)
		}
		if got := out.String(); got != "CREATE INDEX `` ON `t` (`a`)" {
			t.Fatalf("Restore() = %q, want the option-less spelling", got)
		}
	})

	t.Run("invalid index option fields are omitted", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		stmt.IndexOption = &ast.IndexOption{} // all-invalid option: nothing may be written
		var out strings.Builder
		if err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
			t.Fatalf("Restore() = %v, want no error", err)
		}
		if got := out.String(); got != "CREATE INDEX `` ON `t` (`a`)" {
			t.Fatalf("Restore() = %q, want the option-less spelling", got)
		}
	})

	t.Run("populated index option is rendered", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		stmt.IndexOption = &ast.IndexOption{KeyBlockSize: 8, Tp: model.IndexTypeHash, Comment: "c"}
		var out strings.Builder
		if err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
			t.Fatalf("Restore() = %v, want no error", err)
		}
		if got := out.String(); got != "CREATE INDEX `` ON `t` (`a`) KEY_BLOCK_SIZE=8 USING HASH COMMENT 'c'" {
			t.Fatalf("Restore() = %q, want the option spelling", got)
		}
	})

	t.Run("unique flag is rendered", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		stmt.Unique = true
		var out strings.Builder
		if err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
			t.Fatalf("Restore() = %v, want no error", err)
		}
		if got := out.String(); got != "CREATE UNIQUE INDEX `` ON `t` (`a`)" {
			t.Fatalf("Restore() = %q, want the unique spelling", got)
		}
	})

	t.Run("child key restore error chains with the slot index", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		stmt.Keys[0] = &sharedCreateIndexFailKey{BaseIndexKeyPart: pkast.BaseIndexKeyPart{Column: qualName("a")}}
		err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{}))
		if err == nil {
			t.Fatalf("Restore() = %v, want an error", err)
		}
		if !strings.Contains(err.Error(), "create index key restore failed") {
			t.Fatalf("Restore() = %v, want the key error chained", err)
		}
		if !strings.Contains(err.Error(), "Keys: [0]") {
			t.Fatalf("Restore() = %v, want the slot index annotation", err)
		}
	})

	t.Run("writer error propagates through the stmt", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		writeErr := errors.New("write failed")
		ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &indexKeyFailWriter{err: writeErr})
		err := stmt.Restore(ctx)
		if !errors.Is(err, writeErr) {
			t.Fatalf("Restore() = %v, want the writer error", err)
		}
		if !errors.Is(ctx.Err(), writeErr) {
			t.Fatalf("context error = %v, want the writer error", ctx.Err())
		}
	})
}

// TestSharedOnOptionsRestore pins the OnDeleteOpt/OnUpdateOpt Restore
// contract: valid refer options render their clause, a nil receiver or an
// unknown option type reports an error without panicking, and writer errors
// propagate to the returned error and the context.
func TestSharedOnOptionsRestore(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		opt  *ast.OnDeleteOpt
	}{
		{name: "nil receiver", opt: nil},
		{name: "valid restrict", opt: &ast.OnDeleteOpt{ReferOpt: ast.ReferOptionRestrict}},
		{name: "valid cascade", opt: &ast.OnDeleteOpt{ReferOpt: ast.ReferOptionCascade}},
		{name: "valid set null", opt: &ast.OnDeleteOpt{ReferOpt: ast.ReferOptionSetNull}},
		{name: "valid no action", opt: &ast.OnDeleteOpt{ReferOpt: ast.ReferOptionNoAction}},
		{name: "invalid option type", opt: &ast.OnDeleteOpt{ReferOpt: ast.ReferOptionType(99)}},
	}
	for _, tc := range cases {
		t.Run("on delete "+tc.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Restore() panicked instead of returning an error: %v", r)
				}
			}()
			var out strings.Builder
			err := tc.opt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out))
			if tc.name == "nil receiver" || tc.name == "invalid option type" {
				if err == nil || !strings.Contains(err.Error(), "ON DELETE") {
					t.Fatalf("Restore() = %v, want an invalid ON DELETE action error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Restore() = %v, want no error", err)
			}
			if got := out.String(); got != "ON DELETE "+tc.opt.ReferOpt.String() {
				t.Fatalf("Restore() = %q, want the clause spelling", got)
			}
		})
	}

	t.Run("writer error propagates", func(t *testing.T) {
		t.Parallel()

		writeErr := errors.New("write failed")
		ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &indexKeyFailWriter{err: writeErr})
		err := (&ast.OnDeleteOpt{ReferOpt: ast.ReferOptionCascade}).Restore(ctx)
		if !errors.Is(err, writeErr) {
			t.Fatalf("Restore() = %v, want the writer error", err)
		}
		if !errors.Is(ctx.Err(), writeErr) {
			t.Fatalf("context error = %v, want the writer error", ctx.Err())
		}
	})

	t.Run("on update follows the on delete contract", func(t *testing.T) {
		t.Parallel()

		var out strings.Builder
		if err := (&ast.OnUpdateOpt{ReferOpt: ast.ReferOptionNoAction}).Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
			t.Fatalf("Restore() = %v, want no error", err)
		}
		if got := out.String(); got != "ON UPDATE NO ACTION" {
			t.Fatalf("Restore() = %q, want the clause spelling", got)
		}
		var opt *ast.OnUpdateOpt
		err := opt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{}))
		if err == nil || !strings.Contains(err.Error(), "ON UPDATE") {
			t.Fatalf("nil receiver Restore() = %v, want an invalid ON UPDATE action error", err)
		}
	})
}

// TestSharedCreateIndexAccept pins the shared CreateIndexStmt traversal
// contract: an Enter replacement of the outer node re-enters its children,
// a conforming same-role key node is accepted in a child slot, a bad table,
// key or option replacement is refused without polluting the slot, and
// skip and stop are honored.
func TestSharedCreateIndexAccept(t *testing.T) {
	t.Parallel()

	newKey := func() *ast.IndexKeyPart {
		return &ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{Column: qualName("a")}}
	}
	newTable := func() *ast.TableName {
		return &ast.TableName{BaseTableRef: pkast.BaseTableRef{Name: qualName("t")}}
	}
	newStmt := func() *ast.CreateIndexStmt {
		return &ast.CreateIndexStmt{
			Table: newTable(),
			Keys:  []pkast.IndexKeyPartNode{newKey(), newKey()},
		}
	}

	t.Run("fixed child order", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		stmt.IndexOption = &ast.IndexOption{}
		visited := []pkast.Node{}
		updated, ok := stmt.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) {
				visited = append(visited, n)
				return n, false
			},
			leave: func(n pkast.Node) (pkast.Node, bool) { return n, true },
		})
		if !ok || updated != pkast.Node(stmt) {
			t.Fatalf("traversal stopped or lost the outer stmt: ok=%v got=%T", ok, updated)
		}
		if len(visited) != 5 {
			t.Fatalf("visited %d nodes, want 5 (stmt, table, 2 keys, option)", len(visited))
		}
		want := []pkast.Node{pkast.Node(stmt), stmt.Table, stmt.Keys[0], stmt.Keys[1], stmt.IndexOption}
		for i := range visited {
			if visited[i] != want[i] {
				t.Fatalf("visit %d is %T, want %T", i, visited[i], want[i])
			}
		}
	})

	t.Run("enter replacement re-enters its children", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		replacement := newStmt()
		replacement.IndexOption = &ast.IndexOption{}
		visited := []pkast.Node{}
		updated, ok := stmt.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) {
				visited = append(visited, n)
				if _, isStmt := n.(*ast.CreateIndexStmt); isStmt {
					return replacement, false
				}
				return n, false
			},
			leave: func(n pkast.Node) (pkast.Node, bool) { return n, true },
		})
		if !ok || updated != pkast.Node(replacement) {
			t.Fatalf("enter replacement lost: ok=%v got=%T", ok, updated)
		}
		// The visitor sees the original stmt, then the replacement's
		// children directly (the replacement is never re-entered).
		// Original children must not appear in the walk.
		if len(visited) != 5 {
			t.Fatalf("visited %d nodes, want 5 (stmt, table, 2 keys, option)", len(visited))
		}
		if _, isStmt := visited[0].(*ast.CreateIndexStmt); !isStmt {
			t.Fatalf("first visit is %T, want *ast.CreateIndexStmt", visited[0])
		}
		if visited[0] != pkast.Node(stmt) {
			t.Fatal("first visit is not the original stmt")
		}
		if visited[1] == pkast.Node(stmt.Table) {
			t.Fatal("the original table leaked into the walk")
		}
		if visited[2] == pkast.Node(stmt.Keys[0]) {
			t.Fatal("the original key leaked into the walk")
		}
	})

	t.Run("custom same-role key is accepted in the child slot", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		original := stmt.Keys[0]
		custom := &sharedReferenceKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{Column: qualName("x")}}
		_, ok := stmt.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) { return n, false },
			leave: func(n pkast.Node) (pkast.Node, bool) {
				if n == pkast.Node(original) {
					return custom, true
				}
				return n, true
			},
		})
		if !ok {
			t.Fatal("a conforming custom key node must travel through the stmt")
		}
		if stmt.Keys[0] != pkast.IndexKeyPartNode(custom) {
			t.Fatal("the key slot must hold the custom key node")
		}
	})

	t.Run("bad table replacement is refused without polluting the slot", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		original := stmt.Table
		_, ok := stmt.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) { return n, false },
			leave: func(n pkast.Node) (pkast.Node, bool) {
				if n == pkast.Node(original) {
					// A TableRefNode that is not the concrete TableName.
					return &ast.TableRefsClause{}, true
				}
				return n, true
			},
		})
		if ok {
			t.Fatal("a non-TableName table replacement must stop the walk")
		}
		if stmt.Table != original {
			t.Fatal("the table slot was polluted by a refused replacement")
		}
	})

	t.Run("bad key replacement is refused without polluting the slot", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		original := stmt.Keys[0]
		wrong := &ast.OnDeleteOpt{ReferOpt: ast.ReferOptionCascade}
		_, ok := stmt.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) { return n, false },
			leave: func(n pkast.Node) (pkast.Node, bool) {
				if n == pkast.Node(original) {
					return wrong, true
				}
				return n, true
			},
		})
		if ok {
			t.Fatal("a non-key replacement must stop the walk")
		}
		if stmt.Keys[0] != original {
			t.Fatal("the key slot was polluted by a refused replacement")
		}
	})

	t.Run("bad option replacement is refused without polluting the slot", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		original := &ast.IndexOption{}
		stmt.IndexOption = original
		wrong := &ast.OnUpdateOpt{ReferOpt: ast.ReferOptionRestrict}
		_, ok := stmt.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) { return n, false },
			leave: func(n pkast.Node) (pkast.Node, bool) {
				if n == pkast.Node(original) {
					return wrong, true
				}
				return n, true
			},
		})
		if ok {
			t.Fatal("a non-option replacement must stop the walk")
		}
		if stmt.IndexOption != original {
			t.Fatal("the option slot was polluted by a refused replacement")
		}
	})

	t.Run("nil table stops the walk", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		stmt.Table = nil
		visited := []pkast.Node{}
		updated, ok := stmt.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) {
				visited = append(visited, n)
				return n, false
			},
		})
		if ok {
			t.Fatal("nil table must stop the walk")
		}
		if updated != pkast.Node(stmt) {
			t.Fatalf("nil table walk lost the outer stmt: got %T", updated)
		}
		if len(visited) != 1 {
			t.Fatalf("visited %d nodes, want only the stmt before the nil table", len(visited))
		}
	})

	t.Run("typed nil table core stops the walk", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		var table *ast.TableName // typed nil: the interface is non-nil, the core is nil
		stmt.Table = table
		_, ok := stmt.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) { return n, false },
			leave: func(n pkast.Node) (pkast.Node, bool) { return n, true },
		})
		if ok {
			t.Fatal("a typed-nil table must stop the walk through the core guard")
		}
	})

	t.Run("typed nil key core stops the walk", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		// A typed-nil IndexKeyPart: the interface is non-nil, IndexKeyPartCore() returns nil.
		var nilKey *ast.IndexKeyPart
		stmt.Keys[0] = nilKey
		_, ok := stmt.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) { return n, false },
			leave: func(n pkast.Node) (pkast.Node, bool) { return n, true },
		})
		if ok {
			t.Fatal("a typed-nil key must stop the walk through the core guard")
		}
	})

	t.Run("skip children", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		visited := []pkast.Node{}
		updated, ok := stmt.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) {
				visited = append(visited, n)
				// The skip applies only to the outer stmt: a skip at any
				// child level would be recorded by the parent's dispatch.
				return n, n == pkast.Node(stmt)
			},
			leave: func(n pkast.Node) (pkast.Node, bool) { return n, true },
		})
		if !ok || updated != pkast.Node(stmt) {
			t.Fatalf("skip path lost the outer stmt: ok=%v", ok)
		}
		// The skip only stops the children walk at the outer level; the
		// outer leave still runs exactly once.
		if len(visited) != 1 {
			t.Fatalf("visited %d nodes, want only the outer stmt", len(visited))
		}
		if stmt.Table.TableRefCore() == nil || len(stmt.Keys) != 2 {
			t.Fatal("skip path changed the stmt's children")
		}
	})

	t.Run("stop at the stmt", func(t *testing.T) {
		t.Parallel()

		stmt := newStmt()
		updated, ok := stmt.Accept(joinVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) { return n, false },
			leave: func(n pkast.Node) (pkast.Node, bool) {
				if _, isStmt := n.(*ast.CreateIndexStmt); isStmt {
					return n, false
				}
				return n, true
			},
		})
		if ok || updated != pkast.Node(stmt) {
			t.Fatalf("stop at the stmt must end the walk: ok=%v got=%T", ok, updated)
		}
	})
}

// TestSharedTableNameRefusedInCreateIndex pins that a child Leave returning a
// TableName for the key slot of CreateIndexStmt is refused: the key slot is
// an IndexKeyPartNode role, and the refused node does not pollute the slot.
func TestSharedTableNameRefusedInCreateIndex(t *testing.T) {
	t.Parallel()

	stmt := &ast.CreateIndexStmt{
		Table: &ast.TableName{BaseTableRef: pkast.BaseTableRef{Name: qualName("t")}},
		Keys:  []pkast.IndexKeyPartNode{&ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{Column: qualName("a")}}},
	}
	original := stmt.Keys[0]
	wrong := &ast.TableName{BaseTableRef: pkast.BaseTableRef{Name: qualName("x")}}
	_, ok := stmt.Accept(joinVisitor{
		enter: func(n pkast.Node) (pkast.Node, bool) { return n, false },
		leave: func(n pkast.Node) (pkast.Node, bool) {
			if n == pkast.Node(original) {
				return wrong, true
			}
			return n, true
		},
	})
	if ok {
		t.Fatal("a TableName in the key slot must stop the walk")
	}
	if stmt.Keys[0] != original {
		t.Fatal("the key slot was polluted by the refused table node")
	}
}

// sharedCreateIndexFailKey is a test-only key node whose Restore fails on
// purpose, so the stmt can be observed to chain the child error.
type sharedCreateIndexFailKey struct {
	pkast.BaseIndexKeyPart
}

func (n *sharedCreateIndexFailKey) IndexKeyPartCore() *pkast.BaseIndexKeyPart {
	if n == nil {
		return nil
	}
	return &n.BaseIndexKeyPart
}

func (n *sharedCreateIndexFailKey) Accept(v pkast.Visitor) (pkast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *sharedCreateIndexFailKey) Restore(ctx *format.RestoreCtx) error {
	return errors.New("create index key restore failed")
}

var _ pkast.IndexKeyPartNode = (*sharedCreateIndexFailKey)(nil)

// indexKeyFailWriter is reused from index_key_test.go; assert it is an
// io.Writer so the reuse is intentional and not an accidental shadow.
var _ io.Writer = (*indexKeyFailWriter)(nil)

package ast_test

import (
	"errors"
	"strings"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
)

// indexKeyRestoreErr is a marker error so tests can distinguish a failing
// expression restore from a real writer error.
type indexKeyRestoreErr struct{}

func (e indexKeyRestoreErr) Error() string { return "index key expression restore failure" }

// indexKeyExpr is a minimal shared leaf ExprNode whose Restore can be made
// to fail on purpose.
type indexKeyExpr struct {
	pkast.BaseExpr
	visited    bool
	restoreErr bool
}

// Accept implements the shared Node contract.
func (n *indexKeyExpr) Accept(v pkast.Visitor) (pkast.Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*indexKeyExpr)
	n.visited = true
	return v.Leave(n)
}

// Restore implements the shared Node contract.
func (n *indexKeyExpr) Restore(ctx *format.RestoreCtx) error {
	if n.restoreErr {
		return indexKeyRestoreErr{}
	}
	ctx.WritePlain("KEY-EXPRESSION")
	return ctx.Err()
}

func TestIndexKeyPartNilRestore(t *testing.T) {
	t.Parallel()

	t.Run("nil receiver errors", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("nil receiver Restore() panicked instead of returning an error: %v", r)
			}
		}()
		var n *ast.IndexKeyPart
		err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{}))
		if err == nil {
			t.Error("nil receiver Restore() must return an error")
		}
	})
}

// TestIndexKeyPartRestore checks the Restore branches: the column spelling
// with optional prefix and DESC, the expression branch which drops the
// column but keeps DESC, and the error chains.
func TestIndexKeyPartRestore(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		node ast.IndexKeyPart
		want string
	}{
		{
			name: "qualified column with prefix and desc",
			node: ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{
				Column:       pkast.QualifiedName{Parts: []lex.Ident{{Name: "db", Raw: "db"}, {Name: "col", Raw: "col"}}},
				PrefixLength: 8,
				Desc:         true,
			}},
			want: "`db`.`col`(8) DESC",
		},
		{
			// Manual node: with Expr set, Desc must survive the expression branch.
			name: "expression branch keeps desc",
			node: ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{
				Expr: &indexKeyExpr{},
				Desc: true,
			}},
			want: "(KEY-EXPRESSION) DESC",
		},
		{
			name: "plain column without modifiers",
			node: ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{
				Column: pkast.QualifiedName{Parts: []lex.Ident{{Name: "a", Raw: "a"}}},
			}},
			want: "`a`",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var sb strings.Builder
			err := (&tc.node).Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
			if err != nil {
				t.Fatalf("Restore() = %v, want no error", err)
			}
			if got := sb.String(); got != tc.want {
				t.Fatalf("Restore() = %q, want %q", got, tc.want)
			}
		})
	}
	t.Run("conflicting syntax is rejected before writing", func(t *testing.T) {
		for _, core := range []pkast.BaseIndexKeyPart{
			{Expr: &indexKeyExpr{}, Column: pkast.QualifiedName{Parts: []lex.Ident{{Name: "a"}}}},
			{Expr: &indexKeyExpr{}, PrefixLength: 3},
		} {
			var sb strings.Builder
			part := &ast.IndexKeyPart{BaseIndexKeyPart: core}
			if err := part.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err == nil || sb.Len() != 0 {
				t.Fatalf("conflicting part emitted %q, error = %v", sb.String(), err)
			}
		}
	})

	t.Run("expression restore error is wrapped", func(t *testing.T) {
		t.Parallel()

		var sb strings.Builder
		node := &ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{Expr: &indexKeyExpr{restoreErr: true}}}
		err := node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
		var marker indexKeyRestoreErr
		if !errors.As(err, &marker) {
			t.Fatalf("Restore() = %v, want the wrapped expression restore error", err)
		}
		if !strings.Contains(err.Error(), "IndexKeyPart expression") {
			t.Fatalf("Restore() = %v, want the IndexKeyPart annotation", err)
		}
		if got := sb.String(); got != "(" {
			t.Fatalf("partial output = %q, want the opened parenthesis only", got)
		}
	})

	for _, branch := range []struct {
		name string
		node ast.IndexKeyPart
	}{
		{name: "writer error propagates through the column branch", node: ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{
			Column: pkast.QualifiedName{Parts: []lex.Ident{{Name: "a", Raw: "a"}}},
		}}},
		{name: "writer error propagates through the expression branch", node: ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{
			Expr: &indexKeyExpr{},
			Desc: true,
		}}},
	} {
		t.Run(branch.name, func(t *testing.T) {
			t.Parallel()

			writeErr := errors.New("write failed")
			ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &indexKeyFailWriter{err: writeErr})
			err := (&branch.node).Restore(ctx)
			if !errors.Is(err, writeErr) {
				t.Fatalf("Restore() = %v, want the writer error", err)
			}
			if !errors.Is(ctx.Err(), writeErr) {
				t.Fatalf("context error = %v, want the writer error", ctx.Err())
			}
		})
	}
}

// indexKeyFailWriter fails every write so the restore context carries a
// pending error.
type indexKeyFailWriter struct{ err error }

func (w *indexKeyFailWriter) Write(p []byte) (int, error) { return 0, w.err }

// TestIndexKeyPartAccept checks the visitor contract on IndexKeyPart: an
// Enter replacement keeps the node type and its children are visited,
// skip and stop are honored, and a non-expression Leave replacement of
// the expression is refused without panicking.
func TestIndexKeyPartAccept(t *testing.T) {
	t.Parallel()

	root := &ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{Expr: &indexKeyExpr{}}}
	replacement := &indexKeyExpr{}

	// One node besides the outer part: the expression leaf.
	entered, left := []pkast.Node{}, []pkast.Node{}
	visited := 0
	v := indexKeyVisitor{
		enter: func(n pkast.Node) (pkast.Node, bool) {
			entered = append(entered, n)
			if _, ok := n.(*indexKeyExpr); ok {
				visited++
			}
			return n, false
		},
		leave: func(n pkast.Node) (pkast.Node, bool) {
			left = append(left, n)
			if _, ok := n.(*indexKeyExpr); ok {
				visited++
				return replacement, true
			}
			return n, true
		},
	}

	got, ok := root.Accept(v)
	if !ok {
		t.Fatal("traversal stopped unexpectedly")
	}
	part, isPart := got.(*ast.IndexKeyPart)
	if !isPart {
		t.Fatalf("Accept() = %T, want *ast.IndexKeyPart", got)
	}
	if part.Expr != replacement {
		t.Fatalf("replaced node's Expr = %T, want the Leave replacement", part.Expr)
	}
	// Two nodes: the outer part and the expression leaf.
	if len(entered) != 2 || len(left) != 2 {
		t.Fatalf("visited entered=%d left=%d nodes, want 2 and 2", len(entered), len(left))
	}
	if visited != 2 {
		t.Fatalf("expression child seen %d times, want 2 (enter + leave)", visited)
	}

	t.Run("enter replacement re-enters its children", func(t *testing.T) {
		t.Parallel()

		replaced := &ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{Expr: &indexKeyExpr{}}}
		visited := 0
		v := indexKeyVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) {
				if _, ok := n.(*ast.IndexKeyPart); ok {
					// The replacement must be re-entered and its children visited.
					return replaced, false
				}
				visited++
				return n, false
			},
			leave: func(n pkast.Node) (pkast.Node, bool) {
				if _, ok := n.(*indexKeyExpr); ok {
					visited++
					return n, true
				}
				return n, true
			},
		}
		if got, ok := root.Accept(v); !ok || got != replaced {
			t.Fatalf("Enter replacement lost: got %T ok=%v", got, ok)
		}
		if visited != 2 {
			t.Fatalf("expression child visited %d times, want 2 (enter + leave)", visited)
		}
	})

	t.Run("skip children", func(t *testing.T) {
		t.Parallel()

		skipped := &ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{Expr: &indexKeyExpr{}}}
		enteredCnt, leftCnt := 0, 0
		v := indexKeyVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) {
				enteredCnt++
				return n, true
			},
			leave: func(n pkast.Node) (pkast.Node, bool) {
				leftCnt++
				return n, true
			},
		}
		if got, ok := skipped.Accept(v); !ok || got != skipped {
			t.Fatalf("skip path lost the outer node: got %T ok=%v", got, ok)
		}
		if enteredCnt != 1 || leftCnt != 1 {
			t.Fatalf("visited entered=%d left=%d nodes, want 1 and 1", enteredCnt, leftCnt)
		}
		if skipped.Expr.(*indexKeyExpr).visited {
			t.Fatal("skipped children were visited despite skip")
		}
	})

	t.Run("stop after the expression", func(t *testing.T) {
		t.Parallel()

		stopped := &ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{Expr: &indexKeyExpr{}}}
		v := indexKeyVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) { return n, false },
			leave: func(n pkast.Node) (pkast.Node, bool) {
				// Stopping at the outer node must end the walk.
				if _, ok := n.(*ast.IndexKeyPart); ok {
					return n, false
				}
				return n, true
			},
		}
		if got, ok := stopped.Accept(v); ok || got != stopped {
			t.Fatalf("stop was ignored: got %T ok=%v", got, ok)
		}
	})

	t.Run("leave replacement of the expression is refused when not an expression", func(t *testing.T) {
		t.Parallel()

		refused := &ast.IndexKeyPart{BaseIndexKeyPart: pkast.BaseIndexKeyPart{Expr: &indexKeyExpr{}}}
		wrong := &ast.IndexKeyPart{}
		v := indexKeyVisitor{
			enter: func(n pkast.Node) (pkast.Node, bool) { return n, false },
			leave: func(n pkast.Node) (pkast.Node, bool) {
				if _, ok := n.(*indexKeyExpr); ok {
					return wrong, true
				}
				return n, true
			},
		}
		original := refused.Expr
		if got, ok := refused.Accept(v); ok {
			t.Fatalf("a non-expression Leave replacement must stop the traversal: got %T ok=%v", got, ok)
		}
		if refused.Expr != original {
			t.Fatal("refused replacement wrote into the expression slot")
		}
	})
}

type indexKeyVisitor struct {
	enter func(pkast.Node) (pkast.Node, bool)
	leave func(pkast.Node) (pkast.Node, bool)
}

func (v indexKeyVisitor) Enter(n pkast.Node) (pkast.Node, bool) { return v.enter(n) }
func (v indexKeyVisitor) Leave(n pkast.Node) (pkast.Node, bool) { return v.leave(n) }

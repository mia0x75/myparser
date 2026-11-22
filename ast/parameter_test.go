package ast_test

import (
	"strings"
	"testing"

	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
)

// TestParamMarkerExprIndexAndSpanAreIndependent proves that the parameter
// index and the source range travel independently: an unknown index never
// invents a range, a known index never claims one, and the legacy order setter
// never revives an unknown range.
func TestParamMarkerExprIndexAndSpanAreIndependent(t *testing.T) {
	t.Parallel()

	unknown := ast.NewParamMarkerExpr(-1)
	if unknown.Index != -1 {
		t.Fatalf("unknown index = %d, want -1", unknown.Index)
	}
	if got := unknown.Span(); got != loc.InvalidSpan() {
		t.Fatalf("unknown span = %v, want {-1 -1}", got)
	}

	zero := ast.NewParamMarkerExpr(0)
	if zero.Index != 0 || zero.Span() != loc.InvalidSpan() {
		t.Fatalf("zero marker = index %d span %v", zero.Index, zero.Span())
	}

	// An empty range is legal and must not collapse into the unknown range.
	empty := ast.NewParamMarkerExprAt(loc.Span{Start: 0, End: 0}, 1)
	if empty.Index != 1 || empty.Span() != (loc.Span{Start: 0, End: 0}) {
		t.Fatalf("empty marker = index %d span %v", empty.Index, empty.Span())
	}

	// The order setter must not revive the unknown range.
	unknown.SetOrder(2)
	if unknown.Index != 2 || unknown.Span() != loc.InvalidSpan() {
		t.Fatalf("ordered unknown marker = index %d span %v", unknown.Index, unknown.Span())
	}
}

// TestParamMarkerExprRestoresQuestionMark proves that the marker renders as a
// single question mark and that a failing writer is reported.
func TestParamMarkerExprRestoresQuestionMark(t *testing.T) {
	t.Parallel()

	for _, marker := range []*ast.ParamMarkerExpr{
		ast.NewParamMarkerExpr(-1),
		ast.NewParamMarkerExpr(0),
		ast.NewParamMarkerExprAt(loc.Span{Start: 0, End: 1}, 5),
	} {
		var sb strings.Builder
		if err := marker.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
			t.Fatalf("restore failed: %v", err)
		}
		if got := sb.String(); got != "?" {
			t.Fatalf("restore = %q, want %q", got, "?")
		}

		sb.Reset()
		marker.Format(&sb)
		if got := sb.String(); got != "?" {
			t.Fatalf("format = %q, want %q", got, "?")
		}
	}

	marker := ast.NewParamMarkerExpr(0)
	if err := marker.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, failingWriter{})); err == nil {
		t.Fatal("restore swallowed the write failure")
	}
}

// TestParamMarkerExprAcceptKeepsVisitorControl proves that the marker is a
// leaf node whose traversal stays under the visitor control: the outer marker
// is reported, children are skipped on demand, and Leave may replace the node
// or stop the traversal.
func TestParamMarkerExprAcceptKeepsVisitorControl(t *testing.T) {
	t.Parallel()

	marker := ast.NewParamMarkerExprAt(loc.Span{Start: 0, End: 1}, 0)

	t.Run("outer leaf", func(t *testing.T) {
		var entered, left []ast.Node
		v := operatorVisitor{
			enter: func(n ast.Node) (ast.Node, bool) { entered = append(entered, n); return n, false },
			leave: func(n ast.Node) (ast.Node, bool) { left = append(left, n); return n, true },
		}
		got, ok := marker.Accept(v)
		if !ok || got != marker || len(entered) != 1 || entered[0] != marker || len(left) != 1 || left[0] != marker {
			t.Fatalf("accept = %T ok=%v entered=%v left=%v", got, ok, entered, left)
		}
	})

	t.Run("skip children", func(t *testing.T) {
		var entered, left []ast.Node
		v := operatorVisitor{
			enter: func(n ast.Node) (ast.Node, bool) { entered = append(entered, n); return n, true },
			leave: func(n ast.Node) (ast.Node, bool) { left = append(left, n); return n, true },
		}
		got, ok := marker.Accept(v)
		if !ok || got != marker || len(entered) != 1 || len(left) != 1 || left[0] != marker {
			t.Fatalf("skip = %T ok=%v entered=%v left=%v", got, ok, entered, left)
		}
	})

	t.Run("leave replacement", func(t *testing.T) {
		replacement := ast.NewParamMarkerExpr(42)
		var left []ast.Node
		v := operatorVisitor{
			enter: func(n ast.Node) (ast.Node, bool) { return n, false },
			leave: func(n ast.Node) (ast.Node, bool) { left = append(left, n); return replacement, true },
		}
		got, ok := marker.Accept(v)
		if !ok || got != replacement {
			t.Fatalf("accept = %T ok=%v, want the replacement", got, ok)
		}
		if len(left) != 1 || left[0] != marker {
			t.Fatalf("left = %v, want the marker", left)
		}
	})

	t.Run("leave stops traversal", func(t *testing.T) {
		v := operatorVisitor{
			enter: func(n ast.Node) (ast.Node, bool) { return n, false },
			leave: func(n ast.Node) (ast.Node, bool) { return n, false },
		}
		got, ok := marker.Accept(v)
		if ok || got != marker {
			t.Fatalf("accept = %T ok=%v, want the marker and ok=false", got, ok)
		}
	})

	t.Run("invalid outer replacement", func(t *testing.T) {
		for _, wrong := range []ast.Node{nil, pureIntLiteral("1"), (*ast.ParamMarkerExpr)(nil)} {
			for _, skip := range []bool{false, true} {
				left := false
				v := operatorVisitor{
					enter: func(ast.Node) (ast.Node, bool) { return wrong, skip },
					leave: func(n ast.Node) (ast.Node, bool) { left = true; return n, true },
				}
				if _, ok := marker.Accept(v); ok || left {
					t.Fatal("invalid outer replacement must stop before Leave")
				}
			}
		}
	})
}

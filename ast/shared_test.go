package ast_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/driver"
	dformat "github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/myparser/ast"
)

// sharedPureExpr is an expression that only embeds the shared BaseExpr: it
// carries no legacy type, flag or format metadata, so it can only travel
// through the shared Node contract.
type sharedPureExpr struct {
	pkast.BaseExpr
	restoreErr bool
}

// Accept implements the shared Node contract.
func (n *sharedPureExpr) Accept(v pkast.Visitor) (pkast.Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*sharedPureExpr)
	return v.Leave(n)
}

// Restore implements the shared Node contract. The text is written before the
// failure on purpose: a failing restore must not leak partial output.
func (n *sharedPureExpr) Restore(ctx *dformat.RestoreCtx) error {
	ctx.WritePlain("PURE")
	if n.restoreErr {
		return &pureRestoreError{}
	}
	return ctx.Err()
}

// pureRestoreError is a marker error so tests can distinguish a failing
// restore from a real context error without importing the dialect errors.
type pureRestoreError struct{}

func (e *pureRestoreError) Error() string { return "shared pure expression restore failure" }

// sharedVisitor records the visited nodes and optionally replaces the child
// expression leaves (pure leaves) with a pure node.
type sharedVisitor struct {
	entered     []pkast.Node
	replacement pkast.Node
	skipRoot    bool
	stop        bool
}

func (v *sharedVisitor) Enter(n pkast.Node) (pkast.Node, bool) {
	v.entered = append(v.entered, n)
	if v.skipRoot && len(v.entered) == 1 {
		return n, true
	}
	return n, false
}

func (v *sharedVisitor) Leave(n pkast.Node) (pkast.Node, bool) {
	if v.replacement != nil {
		if _, ok := n.(*sharedPureExpr); ok {
			return v.replacement, true
		}
	}
	return n, !v.stop
}

// TestSharedNodeIdentity proves the dialect aliases are the shared types.
func TestSharedNodeIdentity(t *testing.T) {
	t.Parallel()

	if reflect.TypeFor[ast.Node]() != reflect.TypeFor[pkast.Node]() {
		t.Error("dialect Node is not the shared Node type")
	}
	if reflect.TypeFor[ast.ExprNode]() != reflect.TypeFor[pkast.ExprNode]() {
		t.Error("dialect ExprNode is not the shared ExprNode type")
	}
	if reflect.TypeFor[ast.StmtNode]() != reflect.TypeFor[pkast.StmtNode]() {
		t.Error("dialect StmtNode is not the shared StmtNode type")
	}
	if reflect.TypeFor[ast.DMLNode]() != reflect.TypeFor[pkast.DMLNode]() {
		t.Error("dialect DMLNode is not the shared DMLNode type")
	}
	if reflect.TypeFor[ast.DDLNode]() != reflect.TypeFor[pkast.DDLNode]() {
		t.Error("dialect DDLNode is not the shared DDLNode type")
	}
	if reflect.TypeFor[ast.Visitor]() != reflect.TypeFor[pkast.Visitor]() {
		t.Error("dialect Visitor is not the shared Visitor type")
	}
	// The pure expression satisfies the dialect expression contract.
	var _ ast.ExprNode = (*sharedPureExpr)(nil)
}

// TestSharedVisitorPureChild checks that the shared visitor sees the outer
// dialect node and the pure child, and that Leave can replace a pure child
// in place.
func TestSharedVisitorPureChild(t *testing.T) {
	t.Parallel()

	pure := &sharedPureExpr{}
	pure.SetText("pure")
	between := &ast.BetweenExpr{
		Expr:  pure,
		Left:  sharedIntLiteral("1"),
		Right: sharedIntLiteral("2"),
	}

	replacement := &sharedPureExpr{}
	replacement.SetText("replacement")
	v := &sharedVisitor{replacement: replacement}
	node, ok := between.Accept(v)
	if !ok {
		t.Fatal("traversal stopped unexpectedly")
	}
	if node != between {
		t.Fatalf("outer dialect node identity lost: got %T", node)
	}
	if len(v.entered) != 4 {
		t.Fatalf("visited %d nodes, want 4", len(v.entered))
	}
	if _, ok := v.entered[0].(*ast.BetweenExpr); !ok {
		t.Fatalf("first visited node is %T, want *ast.BetweenExpr", v.entered[0])
	}
	if _, ok := v.entered[1].(*sharedPureExpr); !ok {
		t.Fatalf("second visited node is %T, want *sharedPureExpr", v.entered[1])
	}
	if between.Expr != replacement {
		t.Fatalf("Expr child was not replaced: got %T, want *sharedPureExpr", between.Expr)
	}
	if _, ok := between.Left.(*ast.LiteralExpr); !ok {
		t.Fatalf("Left child type changed: got %T, want *ast.LiteralExpr", between.Left)
	}
	if _, ok := between.Right.(*ast.LiteralExpr); !ok {
		t.Fatalf("Right child type changed: got %T, want *ast.LiteralExpr", between.Right)
	}

	t.Run("skip children", func(t *testing.T) {
		skipped := &sharedVisitor{skipRoot: true}
		tree := &ast.BetweenExpr{
			Expr:  &sharedPureExpr{},
			Left:  sharedIntLiteral("1"),
			Right: sharedIntLiteral("2"),
		}
		if _, ok := tree.Accept(skipped); !ok {
			t.Fatal("traversal stopped unexpectedly")
		}
		if len(skipped.entered) != 1 {
			t.Fatalf("visited %d nodes, want 1", len(skipped.entered))
		}
		if _, ok := skipped.entered[0].(*ast.BetweenExpr); !ok {
			t.Fatalf("first visited node is %T, want *ast.BetweenExpr", skipped.entered[0])
		}
	})

	t.Run("stop on leave", func(t *testing.T) {
		stopper := &sharedVisitor{stop: true}
		tree := &ast.BetweenExpr{
			Expr:  &sharedPureExpr{},
			Left:  sharedIntLiteral("1"),
			Right: sharedIntLiteral("2"),
		}
		if _, ok := tree.Accept(stopper); ok {
			t.Fatal("visitor stop was ignored")
		}
		if len(stopper.entered) != 2 {
			t.Fatalf("visited %d nodes, want 2", len(stopper.entered))
		}
	})
}

// sharedIntLiteral builds a syntax-only integer literal for shared tests.
func sharedIntLiteral(value string) ast.ExprNode {
	return &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: value}}
}

// TestSharedParamMarkerAccept checks the visitor receives the outer
// parameter marker.
func TestSharedParamMarkerAccept(t *testing.T) {
	t.Parallel()

	marker := ast.NewParamMarkerExprAt(loc.Span{Start: 0, End: 1}, 0)
	v := &sharedVisitor{}
	node, ok := marker.Accept(v)
	if !ok {
		t.Fatal("param marker traversal failed")
	}
	if node != marker {
		t.Fatalf("param marker identity lost: got %T", node)
	}
	if len(v.entered) != 1 {
		t.Fatalf("visited %d nodes, want 1", len(v.entered))
	}
	if _, ok := v.entered[0].(*ast.ParamMarkerExpr); !ok {
		t.Fatalf("visited node is %T, want *ast.ParamMarkerExpr", v.entered[0])
	}
}

// TestSharedSpanAndText checks the shared source range and text contract.
func TestSharedSpanAndText(t *testing.T) {
	t.Parallel()

	pure := &sharedPureExpr{}
	unknown := pure.Span()
	if unknown != loc.InvalidSpan() || unknown.Start != -1 || unknown.End != -1 {
		t.Fatalf("unknown span is %v, want {-1 -1}", unknown)
	}
	pure.SetSpan(loc.Span{Start: 0, End: 0})
	if got := pure.Span(); got != (loc.Span{Start: 0, End: 0}) {
		t.Fatalf("empty span is %v, want [0 0)", got)
	}
	pure.SetSpan(loc.Span{Start: -1, End: -1})
	if got := pure.Span(); got != (loc.Span{Start: -1, End: -1}) {
		t.Fatalf("invalid span is %v, want {-1 -1}", got)
	}
	pure.SetText("hello")
	if got := pure.Text(); got != "hello" {
		t.Fatalf("text is %q, want %q", got, "hello")
	}
}

// TestSharedFormatExprAndFlag checks the syntax rendering and flag helpers.
func TestSharedFormatExprAndFlag(t *testing.T) {
	t.Parallel()

	var sb strings.Builder
	ast.FormatExpr(sharedIntLiteral("1"), &sb)
	if got := sb.String(); got != "1" {
		t.Fatalf("syntax literal format is %q, want %q", got, "1")
	}
	sb.Reset()
	// A node without a Format method falls back to Restore.
	ast.FormatExpr(&sharedPureExpr{}, &sb)
	if got := sb.String(); got != "PURE" {
		t.Fatalf("foreign format is %q, want %q", got, "PURE")
	}

	// A failing Restore is reported as a placeholder, partial output is dropped.
	sb.Reset()
	ast.FormatExpr(&sharedPureExpr{restoreErr: true}, &sb)
	if got := sb.String(); got != "<unsupported>" {
		t.Fatalf("failed foreign format is %q, want %q", got, "<unsupported>")
	}

	if got := ast.ExprFlag(&sharedPureExpr{}); got != 0 {
		t.Fatalf("pure flag is %d, want 0", got)
	}
}

// TestSharedConstructors checks that the constructors work without any
// registration side effect and that the legacy driver value expression is an
// explicit execution-layer bridge, not an AST node.
func TestSharedConstructors(t *testing.T) {
	t.Parallel()

	value := driver.NewValueExpr("abc")
	if got := driver.NewValueExpr(value); got != value {
		t.Fatal("driver.NewValueExpr did not keep the existing value expression")
	}
	// The driver value expression is an explicit execution-layer bridge: it is
	// not part of the shared ExprNode contract, so it must not be placed in an
	// AST slot.
	_ = value
	marker := ast.NewParamMarkerExprAt(loc.Span{Start: 0, End: 1}, 0)
	if marker.Index != 0 || marker.Span() != (loc.Span{Start: 0, End: 1}) {
		t.Fatalf("parameter syntax data was not preserved: index=%d span=%v", marker.Index, marker.Span())
	}
	marker.SetOrder(2)
	if marker.Index != 2 || marker.Span() != (loc.Span{Start: 0, End: 1}) {
		t.Fatalf("legacy order setter changed the wrong syntax data: index=%d span=%v", marker.Index, marker.Span())
	}
}

// TestSharedConvertRestore checks that a missing charset fails the
// CONVERT restore instead of panicking, and that a syntax string argument
// restores to the historical output.
func TestSharedConvertRestore(t *testing.T) {
	t.Parallel()

	convert := &ast.FuncCallExpr{
		BaseCallExpr: pkast.BaseCallExpr{
			Name: lex.Ident{Name: "CONVERT"},
			Args: []ast.ExprNode{sharedIntLiteral("1")},
		},
	}
	var sb strings.Builder
	err := convert.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &sb))
	if err == nil {
		t.Fatal("CONVERT without a charset must fail")
	}
	if !strings.Contains(err.Error(), "CONVERT") {
		t.Fatalf("unexpected error %v", err)
	}

	// The syntax charset restores without an execution-layer value expression.
	syntax := &ast.FuncCallExpr{
		BaseCallExpr: pkast.BaseCallExpr{
			Name: lex.Ident{Name: "CONVERT"},
			Args: []ast.ExprNode{&ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralString, Value: "abc"}}},
		},
		Charset: lex.Ident{Name: "utf8mb4"},
	}
	sb.Reset()
	if err := syntax.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &sb)); err != nil {
		t.Fatalf("syntax CONVERT restore failed: %v", err)
	}
	if got, want := sb.String(), "CONVERT('abc' USING UTF8MB4)"; got != want {
		t.Fatalf("syntax CONVERT restore is %q, want %q", got, want)
	}
}

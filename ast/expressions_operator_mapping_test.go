package ast_test

import (
	"bytes"
	"errors"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"

	. "github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/opcode"
)

// TestOperatorMappingMatchesOpcode asserts that every operator constant used
// by this dialect keeps its public numbering in lockstep with opcode.Op.
// The int conversion at the node boundary must never silently remap.
func TestOperatorMappingMatchesOpcode(t *testing.T) {
	pairs := []struct {
		op opcode.Op
		pk pkast.Operator
	}{
		{opcode.LogicAnd, pkast.OperatorLogicAnd},
		{opcode.LeftShift, pkast.OperatorLeftShift},
		{opcode.RightShift, pkast.OperatorRightShift},
		{opcode.LogicOr, pkast.OperatorLogicOr},
		{opcode.GE, pkast.OperatorGE},
		{opcode.LE, pkast.OperatorLE},
		{opcode.EQ, pkast.OperatorEQ},
		{opcode.NE, pkast.OperatorNE},
		{opcode.LT, pkast.OperatorLT},
		{opcode.GT, pkast.OperatorGT},
		{opcode.Plus, pkast.OperatorPlus},
		{opcode.Minus, pkast.OperatorMinus},
		{opcode.And, pkast.OperatorAnd},
		{opcode.Or, pkast.OperatorOr},
		{opcode.Mod, pkast.OperatorMod},
		{opcode.Xor, pkast.OperatorXor},
		{opcode.Div, pkast.OperatorDiv},
		{opcode.Mul, pkast.OperatorMul},
		{opcode.Not, pkast.OperatorNot},
		{opcode.BitNeg, pkast.OperatorBitNeg},
		{opcode.IntDiv, pkast.OperatorIntDiv},
		{opcode.LogicXor, pkast.OperatorLogicXor},
		{opcode.NullEQ, pkast.OperatorNullEQ},
		{opcode.In, pkast.OperatorIn},
		{opcode.Like, pkast.OperatorLike},
		{opcode.Case, pkast.OperatorCase},
		{opcode.Regexp, pkast.OperatorRegexp},
		{opcode.IsNull, pkast.OperatorIsNull},
		{opcode.IsTruth, pkast.OperatorIsTruth},
		{opcode.IsFalsity, pkast.OperatorIsFalsity},
	}
	for _, p := range pairs {
		if int(p.op) != int(p.pk) {
			t.Errorf("opcode.Op(%d) must map to pkast.Operator(%d), got %d", p.op, p.op, p.pk)
		}
	}
	// The 2 nodes under migration must round-trip through the conversion.
	bin := &BinaryOperationExpr{BaseBinaryExpr: pkast.BaseBinaryExpr{Op: pkast.Operator(opcode.Plus)}}
	if opcode.Op(bin.Op) != opcode.Plus {
		t.Errorf("binary Op round-trip failed: %v", bin.Op)
	}
	un := &UnaryOperationExpr{BaseUnaryExpr: pkast.BaseUnaryExpr{Op: pkast.Operator(opcode.Minus)}}
	if opcode.Op(un.Op) != opcode.Minus {
		t.Errorf("unary Op round-trip failed: %v", un.Op)
	}
}

// failingWriter makes Restore report an output error to exercise the error
// path without weakening the valid-SQL baseline.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("restore write failure")
}

func restoreExpr(t *testing.T, n Node, flags format.RestoreFlags) string {
	t.Helper()
	var out bytes.Buffer
	ctx := format.NewRestoreCtx(flags, &out)
	if err := n.Restore(ctx); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	return out.String()
}

// TestBinaryRestoreErrorPaths covers missing operands, an invalid Op, and
// propagating output errors.
func TestBinaryRestoreErrorPaths(t *testing.T) {
	validLeft := &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "1"}}
	validRight := &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "2"}}

	// Valid output baseline: 1 + 2.
	valid := &BinaryOperationExpr{BaseBinaryExpr: pkast.BaseBinaryExpr{Op: pkast.Operator(opcode.Plus), Left: validLeft, Right: validRight}}
	if got := restoreExpr(t, valid, format.DefaultRestoreFlags); got != "1+2" {
		t.Fatalf("valid binary restore changed: %q", got)
	}

	// Missing left operand must be an error, not nil.
	noLeft := &BinaryOperationExpr{BaseBinaryExpr: pkast.BaseBinaryExpr{Op: pkast.Operator(opcode.Plus), Right: validRight}}
	var buf bytes.Buffer
	if err := noLeft.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &buf)); err == nil {
		t.Fatal("nil Left must return an error")
	}

	// Missing right operand must be an error, not nil.
	noRight := &BinaryOperationExpr{BaseBinaryExpr: pkast.BaseBinaryExpr{Op: pkast.Operator(opcode.Plus), Left: validLeft}}
	buf.Reset()
	if err := noRight.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &buf)); err == nil {
		t.Fatal("nil Right must return an error")
	}

	// Unknown operator must be an error, not nil.
	badOp := &BinaryOperationExpr{BaseBinaryExpr: pkast.BaseBinaryExpr{Op: pkast.OperatorUnknown, Left: validLeft, Right: validRight}}
	buf.Reset()
	if err := badOp.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &buf)); err == nil {
		t.Fatal("unknown Op must return an error")
	}

	// Output errors must still propagate.
	if err := valid.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, failingWriter{})); err == nil {
		t.Fatal("Restore must surface the writer error")
	}
}

// TestUnaryRestoreErrorPaths mirrors the binary error-path coverage.
func TestUnaryRestoreErrorPaths(t *testing.T) {
	valid := &UnaryOperationExpr{BaseUnaryExpr: pkast.BaseUnaryExpr{Op: pkast.Operator(opcode.Minus), Expr: &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "7"}}}}
	if got := restoreExpr(t, valid, format.DefaultRestoreFlags); got != "-7" {
		t.Fatalf("valid unary restore changed: %q", got)
	}

	noExpr := &UnaryOperationExpr{BaseUnaryExpr: pkast.BaseUnaryExpr{Op: pkast.Operator(opcode.Minus)}}
	var buf bytes.Buffer
	if err := noExpr.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &buf)); err == nil {
		t.Fatal("nil Expr must return an error")
	}

	badOp := &UnaryOperationExpr{BaseUnaryExpr: pkast.BaseUnaryExpr{Op: pkast.OperatorUnknown, Expr: &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "7"}}}}
	buf.Reset()
	if err := badOp.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &buf)); err == nil {
		t.Fatal("unknown Op must return an error")
	}

	if err := valid.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, failingWriter{})); err == nil {
		t.Fatal("Restore must surface the writer error")
	}
}

// stmtChildReplacer returns a non-expression from a child Leave to prove the
// traversal refuses the slot instead of panicking on a bare assert.
type stmtChildReplacer struct {
	stmt Node
}

func (v *stmtChildReplacer) Enter(in Node) (Node, bool) {
	return in, false
}

func (v *stmtChildReplacer) Leave(in Node) (Node, bool) {
	if _, ok := in.(*ColumnRef); ok {
		return v.stmt, true
	}
	return in, true
}

// fakeStmt is a Node that is not an ExprNode, for slot-refusal tests.
type fakeStmt struct {
	pkast.BaseStmt
}

func (n *fakeStmt) Accept(v Visitor) (Node, bool) { return v.Leave(n) }

func (n *fakeStmt) Restore(ctx *format.RestoreCtx) error { return ctx.Err() }

func TestBinaryVisitorSkipsNilSlots(t *testing.T) {
	root := &BinaryOperationExpr{BaseBinaryExpr: pkast.BaseBinaryExpr{Op: pkast.Operator(opcode.Plus)}}

	v := &sharedVisitor{}
	if _, ok := root.Accept(v); !ok {
		t.Fatal("nil slots must not stop the traversal")
	}
	if len(v.entered) != 1 || v.entered[0] != root {
		t.Fatal("nil slots must not be visited")
	}
	if root.Left != nil || root.Right != nil {
		t.Fatal("a nil slot must stay cleared")
	}

	// A non-nil slot with a bad replacement must stop the walk, not panic.
	root2 := &BinaryOperationExpr{BaseBinaryExpr: pkast.BaseBinaryExpr{Op: pkast.Operator(opcode.Plus), Left: &ColumnRef{BaseColumnRef: pkast.BaseColumnRef{}}, Right: &ColumnRef{BaseColumnRef: pkast.BaseColumnRef{}}}}
	if _, ok := root2.Accept(&stmtChildReplacer{stmt: &fakeStmt{}}); ok {
		t.Fatal("a StmtNode replacement of a non-nil slot must stop the walk")
	}
	if root2.Left == nil || root2.Right == nil {
		t.Fatal("the failed replacement must not clear a non-nil slot")
	}
}

func TestOperatorsRefuseNilChildReplacement(t *testing.T) {
	for _, root := range []ExprNode{
		&BinaryOperationExpr{BaseBinaryExpr: pkast.BaseBinaryExpr{Left: &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "1"}}, Right: &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "2"}}}},
		&UnaryOperationExpr{BaseUnaryExpr: pkast.BaseUnaryExpr{Expr: &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "1"}}}},
	} {
		v := &nilChildReplacer{slots: 1}
		if _, ok := root.Accept(v); ok {
			t.Fatal("nil child replacement must be refused")
		}
		if v.slots != 0 {
			t.Fatal("test did not replace a child")
		}
	}
}

// nilChildReplacer is a visitor whose Leave reports a nil child for the two
// expression slots of the node under test.
type nilChildReplacer struct {
	slots int
}

func (v *nilChildReplacer) Enter(in Node) (Node, bool) {
	return in, false
}

func (v *nilChildReplacer) Leave(in Node) (Node, bool) {
	if _, ok := in.(ExprNode); ok {
		if v.slots > 0 {
			v.slots--
			return nil, true
		}
	}
	return in, true
}

// TestBinaryVisitorRefusesStmtChild proves a StmtNode child stops the walk.
func TestBinaryVisitorRefusesStmtChild(t *testing.T) {
	left := &ColumnRef{BaseColumnRef: pkast.BaseColumnRef{}}
	right := &ColumnRef{BaseColumnRef: pkast.BaseColumnRef{}}
	root := &BinaryOperationExpr{BaseBinaryExpr: pkast.BaseBinaryExpr{Op: pkast.Operator(opcode.Plus), Left: left, Right: right}}

	v := &stmtChildReplacer{stmt: &fakeStmt{}}
	if _, ok := root.Accept(v); ok {
		t.Fatal("a StmtNode child must refuse the expr slot and stop traversal")
	}
}

// TestUnaryVisitorRefusesStmtChild mirrors the binary slot-refusal coverage.
func TestUnaryVisitorRefusesStmtChild(t *testing.T) {
	child := &ColumnRef{BaseColumnRef: pkast.BaseColumnRef{}}
	root := &UnaryOperationExpr{BaseUnaryExpr: pkast.BaseUnaryExpr{Op: pkast.Operator(opcode.Minus), Expr: child}}

	v := &stmtChildReplacer{stmt: &fakeStmt{}}
	if _, ok := root.Accept(v); ok {
		t.Fatal("a StmtNode child must refuse the expr slot and stop traversal")
	}
}

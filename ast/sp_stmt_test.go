package ast

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

// spP60Visitor is a minimal visitor for p60 tests.
type spP60Visitor struct{}

func (spP60Visitor) Enter(in Node) (Node, bool) { return in, false }
func (spP60Visitor) Leave(in Node) (Node, bool) { return in, true }

func spP60Restore(t *testing.T, n Node) string {
	t.Helper()
	var sb strings.Builder
	if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
		t.Fatalf("restore %T: %v", n, err)
	}
	return sb.String()
}

func TestSpP60LabelRestore(t *testing.T) {
	// BeginEnd with label
	stmt := &BeginEndStmt{
		Label: lex.Ident{Name: "loop1", Raw: "loop1"},
		List:  []StmtNode{},
	}
	got := spP60Restore(t, stmt)
	if !strings.Contains(got, "loop1") {
		t.Errorf("BeginEndStmt.Restore() = %q, want label %q", got, "loop1")
	}
	// While with label
	w := &WhileStmt{
		Label:     lex.Ident{Name: "w1", Raw: "w1"},
		Condition: &PositionExpr{N: 1},
	}
	got = spP60Restore(t, w)
	if !strings.Contains(got, "w1") {
		t.Errorf("WhileStmt.Restore() = %q, want label %q", got, "w1")
	}
	// Repeat with label
	r := &RepeatStmt{
		Label:     lex.Ident{Name: "r1", Raw: "r1"},
		Condition: &PositionExpr{N: 1},
	}
	got = spP60Restore(t, r)
	if !strings.Contains(got, "r1") {
		t.Errorf("RepeatStmt.Restore() = %q, want label %q", got, "r1")
	}
	// Loop with label
	l := &LoopStmt{
		Label: lex.Ident{Name: "l1", Raw: "l1"},
	}
	got = spP60Restore(t, l)
	if !strings.Contains(got, "l1") {
		t.Errorf("LoopStmt.Restore() = %q, want label %q", got, "l1")
	}
	// For with label
	f := &ForStmt{
		Label:      lex.Ident{Name: "f1", Raw: "f1"},
		Variable:   lex.Ident{Name: "i", Raw: "i"},
		LowerBound: &PositionExpr{N: 1},
		UpperBound: &PositionExpr{N: 3},
	}
	got = spP60Restore(t, f)
	if !strings.Contains(got, "f1") {
		t.Errorf("ForStmt.Restore() = %q, want label %q", got, "f1")
	}
	// Leave with label
	le := &LeaveStmt{Label: lex.Ident{Name: "loop1", Raw: "loop1"}}
	got = spP60Restore(t, le)
	if got != "LEAVE `loop1`" {
		t.Errorf("LeaveStmt.Restore() = %q, want %q", got, "LEAVE `loop1`")
	}
	// Iterate with label
	it := &IterateStmt{Label: lex.Ident{Name: "loop1", Raw: "loop1"}}
	got = spP60Restore(t, it)
	if got != "ITERATE `loop1`" {
		t.Errorf("IterateStmt.Restore() = %q, want %q", got, "ITERATE `loop1`")
	}
}

func TestSpP60LabelAcceptNilSafe(t *testing.T) {
	// Accept on zero-value stmts should not panic
	for _, stmt := range []Node{
		&BeginEndStmt{},
		&WhileStmt{Condition: &PositionExpr{N: 1}},
		&RepeatStmt{Condition: &PositionExpr{N: 1}},
		&LoopStmt{},
		&ForStmt{Variable: lex.Ident{Name: "i"}, LowerBound: &PositionExpr{N: 1}, UpperBound: &PositionExpr{N: 2}},
		&LeaveStmt{},
		&IterateStmt{},
	} {
		stmt.Accept(spP60Visitor{})
	}
}

func TestSpP60DeclareNames(t *testing.T) {
	// DeclareStmt with Names as lex.Ident
	stmt := &DeclareStmt{
		Names: []lex.Ident{{Name: "a", Raw: "a"}, {Name: "b", Raw: "b"}},
		Type:  NewTypeName("INT"),
	}
	got := spP60Restore(t, stmt)
	if !strings.Contains(got, "DECLARE") || !strings.Contains(got, "a") || !strings.Contains(got, "b") {
		t.Errorf("DeclareStmt.Restore() = %q", got)
	}
	stmt.Accept(spP60Visitor{})
}

func TestSpP60DeclareCondition(t *testing.T) {
	// DeclareConditionStmt with Name as lex.Ident
	cond := NewErrnoHandlerCondition(1146)
	stmt := &DeclareConditionStmt{Name: lex.Ident{Name: "mycond", Raw: "mycond"}, Condition: cond}
	got := spP60Restore(t, stmt)
	if !strings.Contains(got, "DECLARE") || !strings.Contains(got, "mycond") || !strings.Contains(got, "CONDITION FOR") {
		t.Errorf("DeclareConditionStmt.Restore() = %q", got)
	}
	stmt.Accept(spP60Visitor{})
}

func TestSpP60DeclareCursor(t *testing.T) {
	// DeclareCursorStmt with Select as QueryNode
	// TableStmt is a valid QueryNode
	q := &SelectStmt{}
	stmt := &DeclareCursorStmt{
		Name:   lex.Ident{Name: "c1", Raw: "c1"},
		Select: q,
	}
	got := spP60Restore(t, stmt)
	if !strings.Contains(got, "DECLARE") || !strings.Contains(got, "CURSOR FOR") {
		t.Errorf("DeclareCursorStmt.Restore() = %q", got)
	}
	stmt.Accept(spP60Visitor{})
}

func TestSpP60ForQueryNode(t *testing.T) {
	// ForStmt with Query as QueryNode
	q := &SelectStmt{}
	stmt := &ForStmt{
		Variable: lex.Ident{Name: "i", Raw: "i"},
		Query:    q,
	}
	got := spP60Restore(t, stmt)
	if !strings.Contains(got, "FOR") || !strings.Contains(got, "IN") {
		t.Errorf("ForStmt.Restore() = %q", got)
	}
	stmt.Accept(spP60Visitor{})
}

func TestSpP60OpenClose(t *testing.T) {
	// OpenStmt and CloseStmt with Name as lex.Ident
	open := &OpenStmt{Name: lex.Ident{Name: "c1", Raw: "c1"}}
	got := spP60Restore(t, open)
	if got != "OPEN `c1`" {
		t.Errorf("OpenStmt.Restore() = %q, want %q", got, "OPEN `c1`")
	}
	open.Accept(spP60Visitor{})

	closeStmt := &CloseStmt{Name: lex.Ident{Name: "c1", Raw: "c1"}}
	got = spP60Restore(t, closeStmt)
	if got != "CLOSE `c1`" {
		t.Errorf("CloseStmt.Restore() = %q, want %q", got, "CLOSE `c1`")
	}
	closeStmt.Accept(spP60Visitor{})
}

func TestSpP60Fetch(t *testing.T) {
	// FetchStmt with Targets and presence fields
	targets := []*VariableTarget{
		{Kind: VariableTargetLocal, Name: QualifiedName{Parts: []lex.Ident{{Name: "a", Raw: "a"}}}},
		{Kind: VariableTargetUser, Name: QualifiedName{Parts: []lex.Ident{{Name: "b", Raw: "b"}}}, Prefix: VariablePrefixAt},
	}
	stmt := &FetchStmt{
		Name:    lex.Ident{Name: "c1", Raw: "c1"},
		Targets: targets,
	}
	got := spP60Restore(t, stmt)
	if !strings.Contains(got, "FETCH") || !strings.Contains(got, "c1") || !strings.Contains(got, "INTO") {
		t.Errorf("FetchStmt.Restore() = %q", got)
	}
	stmt.Accept(spP60Visitor{})

	// With HasNext/HasFrom
	stmt2 := &FetchStmt{
		Name:    lex.Ident{Name: "c1", Raw: "c1"},
		Targets: targets,
		HasNext: true,
		HasFrom: true,
	}
	got = spP60Restore(t, stmt2)
	if !strings.Contains(got, "NEXT") || !strings.Contains(got, "FROM") {
		t.Errorf("FetchStmt with NEXT/FROM Restore() = %q", got)
	}
	stmt2.Accept(spP60Visitor{})

	// No targets should return error
	noTargets := &FetchStmt{Name: lex.Ident{Name: "c1"}}
	var sb strings.Builder
	if err := noTargets.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err == nil {
		t.Error("FetchStmt.Restore() with no targets: expected error, got nil")
	}
}

func TestSpP60Signal(t *testing.T) {
	// SignalStmt with SignalInfoItem.Name as lex.Ident
	cond := &HandlerCondition{Type: HandlerConditionSQLWarning}
	items := []*SignalInfoItem{
		{Name: lex.Ident{Name: "message_text", Raw: "message_text"}, Value: &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralString, Value: "msg"}}},
	}
	stmt := &SignalStmt{Condition: cond, InfoItems: items}
	got := spP60Restore(t, stmt)
	if !strings.Contains(got, "SIGNAL") || !strings.Contains(got, "message_text") {
		t.Errorf("SignalStmt.Restore() = %q", got)
	}
	stmt.Accept(spP60Visitor{})

	// ResignalStmt
	rstmt := &ResignalStmt{Condition: cond, InfoItems: items}
	got = spP60Restore(t, rstmt)
	if !strings.Contains(got, "RESIGNAL") {
		t.Errorf("ResignalStmt.Restore() = %q", got)
	}
	rstmt.Accept(spP60Visitor{})
}

func TestSpP60NilSafe(t *testing.T) {
	// FetchStmt.Accept should not panic on zero-value
	zero := &FetchStmt{}
	zero.Accept(spP60Visitor{})

	// DeclareCursorStmt.Accept with nil Select should not panic
	curs := &DeclareCursorStmt{Name: lex.Ident{Name: "c"}}
	curs.Accept(spP60Visitor{})

	// ForStmt.Accept with nil Query should not panic
	f := &ForStmt{Variable: lex.Ident{Name: "i"}}
	f.Accept(spP60Visitor{})

	visitor := &forListVisitor{}
	f.List = []StmtNode{&BeginEndStmt{}}
	if _, ok := f.Accept(visitor); ok || visitor.visited {
		t.Fatal("ForStmt.Accept must fail and stop before its body when the range bounds are nil")
	}
}

type forListVisitor struct{ visited bool }

func (v *forListVisitor) Enter(n Node) (Node, bool) {
	if _, ok := n.(*BeginEndStmt); ok {
		v.visited = true
	}
	return n, false
}

func (*forListVisitor) Leave(n Node) (Node, bool) { return n, true }

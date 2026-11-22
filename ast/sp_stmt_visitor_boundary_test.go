package ast

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
)

type spStmtRoot struct {
	name     string
	node     Node
	typedNil Node
	wrong    Node
}

func spStmtRoots() []spStmtRoot {
	cond := NewParamMarkerExpr(0)
	stmt := &BeginEndStmt{}
	query := &SelectStmt{}
	target := &VariableTarget{Kind: VariableTargetLocal, Name: QualifiedName{Parts: []lex.Ident{{Name: "v"}}}}
	roots := []spStmtRoot{
		{name: "begin-end", node: &BeginEndStmt{}, typedNil: (*BeginEndStmt)(nil), wrong: &CaseStmt{}},
		{name: "declare", node: &DeclareStmt{Names: []lex.Ident{{Name: "a"}}, Type: NewTypeName("INT"), Default: cond},
			typedNil: (*DeclareStmt)(nil), wrong: &CaseStmt{}},
		{name: "declare-condition", node: &DeclareConditionStmt{Condition: NewErrnoHandlerCondition(1)},
			typedNil: (*DeclareConditionStmt)(nil), wrong: &CaseStmt{}},
		{name: "declare-cursor", node: &DeclareCursorStmt{Select: query}, typedNil: (*DeclareCursorStmt)(nil), wrong: &CaseStmt{}},
		{name: "declare-handler", node: &DeclareHandlerStmt{Conditions: []*HandlerCondition{NewErrnoHandlerCondition(1)}, Stmt: stmt},
			typedNil: (*DeclareHandlerStmt)(nil), wrong: &CaseStmt{}},
		{name: "while", node: &WhileStmt{Condition: cond}, typedNil: (*WhileStmt)(nil), wrong: &CaseStmt{}},
		{name: "repeat", node: &RepeatStmt{Condition: cond}, typedNil: (*RepeatStmt)(nil), wrong: &CaseStmt{}},
		{name: "loop", node: &LoopStmt{}, typedNil: (*LoopStmt)(nil), wrong: &CaseStmt{}},
		{name: "for-range", node: &ForStmt{LowerBound: cond, UpperBound: cond}, typedNil: (*ForStmt)(nil), wrong: &CaseStmt{}},
		{name: "for-query", node: &ForStmt{Query: query}, typedNil: (*ForStmt)(nil), wrong: &CaseStmt{}},
		{name: "leave", node: &LeaveStmt{}, typedNil: (*LeaveStmt)(nil), wrong: &CaseStmt{}},
		{name: "iterate", node: &IterateStmt{}, typedNil: (*IterateStmt)(nil), wrong: &CaseStmt{}},
		{name: "open", node: &OpenStmt{}, typedNil: (*OpenStmt)(nil), wrong: &CaseStmt{}},
		{name: "close", node: &CloseStmt{}, typedNil: (*CloseStmt)(nil), wrong: &CaseStmt{}},
		{name: "fetch", node: &FetchStmt{Targets: []*VariableTarget{target}}, typedNil: (*FetchStmt)(nil), wrong: &CaseStmt{}},
		{name: "signal", node: &SignalStmt{Condition: NewErrnoHandlerCondition(1), InfoItems: []*SignalInfoItem{{Name: lex.Ident{Name: "m"}, Value: cond}}},
			typedNil: (*SignalStmt)(nil), wrong: &CaseStmt{}},
		{name: "resignal", node: &ResignalStmt{InfoItems: []*SignalInfoItem{{Name: lex.Ident{Name: "m"}, Value: cond}}},
			typedNil: (*ResignalStmt)(nil), wrong: &CaseStmt{}},
	}
	return roots
}

func TestSpStmtVisitorRejectsInvalidEnter(t *testing.T) {
	for _, root := range spStmtRoots() {
		for _, replacement := range []Node{nil, root.typedNil, root.wrong} {
			for _, skip := range []bool{false, true} {
				t.Run(root.name, func(t *testing.T) {
					left := false
					visitor := spBoundaryVisitor{
						enter: func(Node) (Node, bool) { return replacement, skip },
						leave: func(n Node) (Node, bool) { left = true; return n, true },
					}
					if _, ok := root.node.Accept(visitor); ok || left {
						t.Fatalf("%T accepted invalid Enter replacement %T (skip=%v) or called Leave",
							root.node, replacement, skip)
					}
				})
			}
		}

		t.Run(root.name+"/valid", func(t *testing.T) {
			got, ok := root.node.Accept(spBoundaryVisitor{})
			if !ok || got != root.node {
				t.Fatalf("%T valid visit failed: ok=%v", root.node, ok)
			}
			switch n := root.node.(type) {
			case *WhileStmt:
				if n.Condition == nil {
					t.Fatal("WhileStmt.Condition was dropped by the valid visit")
				}
			case *ForStmt:
				if n.Query != nil {
					if _, isSel := n.Query.(*SelectStmt); !isSel {
						t.Fatalf("ForStmt.Query role changed: %T", n.Query)
					}
				}
			case *BeginEndStmt:
				if n.List != nil {
					for _, s := range n.List {
						if !nodePresent(s) {
							t.Fatal("BeginEndStmt.List has a nil entry")
						}
					}
				}
			}
		})
	}
}

func TestSpStmtVisitorRejectsInvalidChildReplacements(t *testing.T) {
	for _, root := range spStmtRoots() {
		noChild := map[string]bool{
			"begin-end":         true,
			"loop":              true,
			"leave":             true,
			"iterate":           true,
			"open":              true,
			"close":             true,
			"declare-condition": true,
		}
		if noChild[root.name] {
			continue
		}
		for _, bad := range []Node{
			&ColumnPosition{},
			nil,
			(*LiteralExpr)(nil),
		} {
			t.Run(root.name, func(t *testing.T) {
				rootLeft := false
				visitor := spBoundaryVisitor{
					enter: func(n Node) (Node, bool) {
						return n, false
					},
					leave: func(n Node) (Node, bool) {
						if n == root.node {
							rootLeft = true
							return n, true
						}
						return bad, true
					},
				}
				got, ok := root.node.Accept(visitor)
				if ok {
					t.Fatalf("%T accepted invalid child replacement %T: ok=%v got=%T",
						root.node, bad, ok, got)
				}
				if rootLeft {
					t.Fatalf("%T: root Leave ran even though the child was rejected", root.node)
				}
			})
		}
	}
}

func TestSpStmtRestoreMissingRequiredChildren(t *testing.T) {
	cases := []struct {
		name string
		node Node
	}{
		{name: "begin-end nil list item", node: &BeginEndStmt{List: []StmtNode{nil}}},
		{name: "while missing condition", node: &WhileStmt{}},
		{name: "repeat missing condition", node: &RepeatStmt{}},
		{name: "for-range missing bounds", node: &ForStmt{Variable: lex.Ident{Name: "i"}}},
		{name: "fetch empty targets", node: &FetchStmt{Name: lex.Ident{Name: "c"}}},
		{name: "fetch nil target", node: &FetchStmt{Name: lex.Ident{Name: "c"}, Targets: []*VariableTarget{nil}}},
		{name: "signal nil item", node: &SignalStmt{Condition: NewErrnoHandlerCondition(1), InfoItems: []*SignalInfoItem{nil}}},
		{name: "resignal typed nil value", node: &ResignalStmt{InfoItems: []*SignalInfoItem{{Value: (*LiteralExpr)(nil)}}}},
		{name: "signal missing condition", node: &SignalStmt{}},
		{name: "declare-handler empty conditions", node: &DeclareHandlerStmt{Stmt: &BeginEndStmt{}}},
		{name: "declare-handler nil stmt", node: &DeclareHandlerStmt{Conditions: []*HandlerCondition{NewErrnoHandlerCondition(1)}}},
		{name: "declare-condition nil condition", node: &DeclareConditionStmt{}},
		{name: "declare-cursor nil query", node: &DeclareCursorStmt{}},
		{name: "declare-cursor typed nil query", node: &DeclareCursorStmt{Select: (*SelectStmt)(nil)}},
		{name: "for typed nil query", node: &ForStmt{Query: (*SelectStmt)(nil)}},
		{name: "declare typed nil default", node: &DeclareStmt{Names: []lex.Ident{{Name: "v"}}, Type: NewTypeName("INT"), Default: (*LiteralExpr)(nil)}},
		{name: "declare nil row field", node: &DeclareStmt{Names: []lex.Ident{{Name: "v"}}, RowType: []*ColumnDef{nil}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if err := recover(); err != nil {
					t.Fatalf("restore %T panicked: %v", tc.node, err)
				}
			}()
			sb := new(strings.Builder)
			ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, sb)
			if err := tc.node.Restore(ctx); err == nil {
				t.Fatalf("restore %T succeeded with missing required child: %q", tc.node, sb.String())
			}
		})
	}
}

func TestSpStmtRestoreRejectsNilRoots(t *testing.T) {
	for _, root := range spStmtRoots() {
		t.Run(root.name, func(t *testing.T) {
			var out strings.Builder
			if err := root.typedNil.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil {
				t.Fatal("Restore accepted nil root")
			}
		})
	}
}

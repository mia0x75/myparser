package ast

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/auth"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
)

type routineRoot struct {
	name     string
	node     Node
	typedNil Node
}

func routineRoots() []routineRoot {
	definer := &auth.UserIdentity{Username: "u", Hostname: "localhost"}
	cond := NewParamMarkerExpr(0)
	body := &BeginEndStmt{}
	name := &TableName{Name: QualifiedName{Parts: []lex.Ident{{Name: "db"}, {Name: "t"}}}}
	roots := []routineRoot{
		{name: "create-function", node: &CreateFunctionStmt{
			Definer: definer, Name: name, ReturnType: NewTypeName("INT"), Body: body,
		}, typedNil: (*CreateFunctionStmt)(nil)},
		{name: "create-procedure", node: &CreateProcedureStmt{
			Definer: definer, Name: name, Body: body,
		}, typedNil: (*CreateProcedureStmt)(nil)},
		{name: "alter-function", node: &AlterFunctionStmt{
			Name: name, Characteristics: &RoutineCharacteristics{},
		}, typedNil: (*AlterFunctionStmt)(nil)},
		{name: "alter-procedure", node: &AlterProcedureStmt{
			Name: name, Characteristics: &RoutineCharacteristics{},
		}, typedNil: (*AlterProcedureStmt)(nil)},
		{name: "create-trigger", node: &CreateTriggerStmt{
			Definer: definer, Name: name, Table: name, Body: body,
		}, typedNil: (*CreateTriggerStmt)(nil)},
		{name: "create-event", node: &CreateEventStmt{
			Definer: definer, Name: name,
			Schedule: &EventSchedule{At: cond}, Body: body,
		}, typedNil: (*CreateEventStmt)(nil)},
		{name: "alter-event", node: &AlterEventStmt{
			Definer: definer, Name: name,
			Schedule: &EventSchedule{Every: cond}, Body: body,
		}, typedNil: (*AlterEventStmt)(nil)},
		{name: "return", node: &ReturnStmt{Expr: cond}, typedNil: (*ReturnStmt)(nil)},
	}
	return roots
}

func TestRoutineVisitorRejectsInvalidEnter(t *testing.T) {
	for _, root := range routineRoots() {
		for _, replacement := range []Node{nil, root.typedNil, &CaseStmt{}} {
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
			visitor := spBoundaryVisitor{
				enter: func(n Node) (Node, bool) { return n, false },
				leave: func(n Node) (Node, bool) { return n, true },
			}
			got, ok := root.node.Accept(visitor)
			if !ok || got != root.node {
				t.Fatalf("%T valid visit = %T, %v; want identity", root.node, got, ok)
			}
		})
	}
}

func TestRoutineVisitorRejectsInvalidChildReplacements(t *testing.T) {
	for _, root := range routineRoots() {
		for _, bad := range []Node{
			&ColumnPosition{},
			nil,
			(*LiteralExpr)(nil),
		} {
			t.Run(root.name, func(t *testing.T) {
				rootLeft := false
				visitor := spBoundaryVisitor{
					enter: func(n Node) (Node, bool) { return n, false },
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

func TestRoutineRestoreMissingRequiredChildren(t *testing.T) {
	name := &TableName{Name: QualifiedName{Parts: []lex.Ident{{Name: "t"}}}}
	cases := []struct {
		name string
		node interface {
			Restore(*format.RestoreCtx) error
		}
	}{
		{name: "create-function nil name", node: &CreateFunctionStmt{Definer: &auth.UserIdentity{Username: "u"}, ReturnType: NewTypeName("INT")}},
		{name: "create-function nil return type", node: &CreateFunctionStmt{Definer: &auth.UserIdentity{Username: "u"}, Name: name}},
		{name: "create-function nil body", node: &CreateFunctionStmt{Definer: &auth.UserIdentity{Username: "u"}, Name: name, ReturnType: NewTypeName("INT")}},
		{name: "create-procedure nil body", node: &CreateProcedureStmt{Definer: &auth.UserIdentity{Username: "u"}, Name: name}},
		{name: "create-event nil body", node: &CreateEventStmt{Definer: &auth.UserIdentity{Username: "u"}, Name: name}},
		{name: "create-event nil name", node: &CreateEventStmt{Definer: &auth.UserIdentity{Username: "u"}, Body: &BeginEndStmt{}}},
		{name: "alter-event nil name", node: &AlterEventStmt{Definer: &auth.UserIdentity{Username: "u"}}},
		{name: "return nil expr", node: &ReturnStmt{}},
		{name: "event-schedule nil every", node: &EventSchedule{}},
		{name: "event-schedule typed nil at", node: &EventSchedule{At: (*LiteralExpr)(nil)}},
		{name: "event-schedule typed nil starts", node: &EventSchedule{Every: NewParamMarkerExpr(0), Starts: (*LiteralExpr)(nil)}},
		{name: "event-schedule typed nil ends", node: &EventSchedule{Every: NewParamMarkerExpr(0), Ends: (*LiteralExpr)(nil)}},
		{name: "alter-event typed nil body", node: &AlterEventStmt{Definer: &auth.UserIdentity{CurrentUser: true}, Name: name, Body: (*BeginEndStmt)(nil)}},
		{name: "UDF missing scalar return type", node: &CreateFunctionStmt{Name: name, Soname: "MOCK_LIBRARY", ReturnsRowType: name}},
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

func TestRoutineRestoreRejectsNilRoots(t *testing.T) {
	for _, root := range routineRoots() {
		t.Run(root.name, func(t *testing.T) {
			var out strings.Builder
			if err := root.typedNil.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil {
				t.Fatal("Restore accepted nil root")
			}
		})
	}
}

func TestRoutineVisitorChecksLaterSlotsWithoutChangingRejectedValue(t *testing.T) {
	for _, tc := range []struct {
		name string
		new  func(ExprNode) (Node, func() ExprNode)
	}{
		{"schedule", func(expr ExprNode) (Node, func() ExprNode) {
			n := &CreateEventStmt{Name: &TableName{}, Schedule: &EventSchedule{At: expr}, Body: &BeginEndStmt{}}
			return n, func() ExprNode { return n.Schedule.At }
		}},
		{"RETURN", func(expr ExprNode) (Node, func() ExprNode) {
			n := &ReturnStmt{Expr: expr}
			return n, func() ExprNode { return n.Expr }
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, replacement := range []Node{nil, (*LiteralExpr)(nil), &ColumnPosition{}, &DefaultExpr{}} {
				old := NewParamMarkerExpr(0)
				n, slot := tc.new(old)
				_, ok := n.Accept(spBoundaryVisitor{
					leave: func(node Node) (Node, bool) {
						if node == old {
							return replacement, true
						}
						return node, true
					},
				})
				if expr, valid := replacement.(*DefaultExpr); valid {
					if !ok || slot() != expr {
						t.Fatal("Accept rejected legal expression replacement")
					}
				} else if ok || slot() != old {
					t.Fatal("Accept stored a rejected expression replacement")
				}
			}
		})
	}
}

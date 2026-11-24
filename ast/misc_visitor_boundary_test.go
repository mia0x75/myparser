package ast

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"

	"github.com/mia0x75/myparser/format"
)

func TestManagementVisitorsValidateEnteredRootBeforeSkipping(t *testing.T) {
	for _, root := range []Node{
		&ExplainForStmt{}, &PrepareStmt{}, &DeallocateStmt{}, &ExecuteStmt{},
		&BeginStmt{}, &BinlogStmt{}, &BackupStmt{}, &CommitStmt{}, &RollbackStmt{},
		&SavepointStmt{}, &ReleaseSavepointStmt{}, &XAStmt{}, &VariableAssignment{},
		&FlushStmt{}, &KillStmt{}, &SetStmt{}, &SetPwdStmt{},
		&SetRoleStmt{}, &SetDefaultRoleStmt{}, &CreateUserStmt{}, &AlterUserStmt{},
		&DropUserStmt{}, &DoStmt{}, &PrivElem{}, &RevokeRoleStmt{}, &GrantRoleStmt{},
		&LockTableStmt{}, &UnlockTableStmt{}, &PurgeStmt{}, &ChangeMasterStmt{},
		&ReplicaControlStmt{}, &ReplicationOptionValue{}, &ResetStmt{}, &LoadXMLStmt{},
		&RenameUserStmt{}, &ComponentSetValue{}, &InstallComponentStmt{},
		&DiagnosticsItem{}, &GetDiagnosticsStmt{}, &LoadDataStmt{}, &SetStatementStmt{},
		&ShutdownStmt{}, &RenameUserPair{}, &ImportTableStmt{}, &UninstallComponentStmt{}, &CloneStmt{},
		&VariableTarget{},
		&ValuesStmt{}, &TableStmt{}, &WindowSpec{}, &FrameClause{}, &FrameBound{},
	} {
		t.Run(fmt.Sprintf("%T", root), func(t *testing.T) {
			typedNil, ok := reflect.Zero(reflect.TypeOf(root)).Interface().(Node)
			if !ok {
				t.Fatal("typed nil does not implement Node")
			}
			for _, replacement := range []Node{nil, typedNil, NewParamMarkerExpr(0)} {
				for _, skip := range []bool{false, true} {
					left := false
					visitor := spBoundaryVisitor{
						enter: func(Node) (Node, bool) { return replacement, skip },
						leave: func(n Node) (Node, bool) { left = true; return n, true },
					}
					if _, ok := root.Accept(visitor); ok || left {
						t.Fatalf("accepted replacement %T with skip=%v or called Leave", replacement, skip)
					}
				}
			}
			left := false
			visitor := spBoundaryVisitor{
				enter: func(n Node) (Node, bool) { return n, true },
				leave: func(n Node) (Node, bool) { left = n == root; return n, true },
			}
			if got, ok := root.Accept(visitor); !ok || got != root || !left {
				t.Fatal("valid skipped root lost its identity")
			}
		})
	}
}

func TestManagementVisitorsRejectInvalidChildRoles(t *testing.T) {
	for _, tc := range []struct {
		name string
		root Node
	}{
		{"prepare", &PrepareStmt{SQLVar: &VariableExpr{}}},
		{"execute", &ExecuteStmt{UsingVars: []ExprNode{NewParamMarkerExpr(0)}}},
		{"backup", &BackupStmt{Table: &TableName{}}},
		{"variable assignment", &VariableAssignment{Value: NewParamMarkerExpr(0)}},
		{"flush", &FlushStmt{Tables: []*TableName{{}}}},
		{"kill", &KillStmt{Expr: NewParamMarkerExpr(0)}},
		{"set", &SetStmt{Variables: []*VariableAssignment{{}}}},
		{"do", &DoStmt{Exprs: []ExprNode{NewParamMarkerExpr(0)}}},
		{"change master", &ChangeMasterStmt{Options: []*ReplicationOption{{}}}},
		{"replica control", &ReplicaControlStmt{ConnectionOptions: []*ReplicationOption{{}}}},
		{"replication values", &ReplicationOptionValue{Values: []ExprNode{NewParamMarkerExpr(0)}}},
		{"reset", &ResetStmt{Args: []ExprNode{NewParamMarkerExpr(0)}}},
		{"load xml", &LoadXMLStmt{LoadDataStmt: LoadDataStmt{Table: &TableName{}}}},
		{"rename user", &RenameUserStmt{Pairs: []*RenameUserPair{{}}}},
		{"component value", &ComponentSetValue{Value: NewParamMarkerExpr(0)}},
		{"install component", &InstallComponentStmt{SetValues: []*ComponentSetValue{{}}}},
		{"diagnostics item", &DiagnosticsItem{Target: &VariableTarget{}}},
		{"get diagnostics", &GetDiagnosticsStmt{Condition: NewParamMarkerExpr(0)}},
		{"load data", &LoadDataStmt{Table: &TableName{}}},
		{"set statement", &SetStatementStmt{Stmt: &BeginStmt{}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, replacement := range []Node{&ColumnPosition{}, nil, (*LiteralExpr)(nil)} {
				var child, rootLeft bool
				visitor := spBoundaryVisitor{
					enter: func(n Node) (Node, bool) { return n, n != tc.root },
					leave: func(n Node) (Node, bool) {
						if n == tc.root {
							rootLeft = true
							return n, true
						}
						child = true
						return replacement, true
					},
				}
				if _, ok := tc.root.Accept(visitor); ok || !child || rootLeft {
					t.Fatalf("ok=%v child=%v root Leave=%v", ok, child, rootLeft)
				}
			}
		})
	}
}

func TestLoadAndSetRestoreValidateMissingChildren(t *testing.T) {
	table := &TableName{Name: QualifiedName{Parts: []lex.Ident{{Name: "t"}}}}
	for _, tc := range []struct {
		name string
		node Node
	}{
		{"nil load data", (*LoadDataStmt)(nil)},
		{"missing load data table", &LoadDataStmt{}},
		{"nil load xml", (*LoadXMLStmt)(nil)},
		{"missing load xml table", &LoadXMLStmt{}},
		{"nil load column", &LoadDataStmt{Table: table, ColumnsAndUserVars: []*ColumnNameOrUserVar{nil}}},
		{"empty xml column", &LoadXMLStmt{LoadDataStmt: LoadDataStmt{Table: table, ColumnsAndUserVars: []*ColumnNameOrUserVar{{}}}}},
		{"nil load assignment", &LoadDataStmt{Table: table, ColumnAssignments: []*Assignment{nil}}},
		{"nil xml assignment", &LoadXMLStmt{LoadDataStmt: LoadDataStmt{Table: table, ColumnAssignments: []*Assignment{nil}}}},
		{"nil set statement", (*SetStatementStmt)(nil)},
		{"missing set statement", &SetStatementStmt{}},
		{"typed nil set statement", &SetStatementStmt{Stmt: (*BeginStmt)(nil)}},
		{"nil set variable", &SetStatementStmt{Stmt: &BeginStmt{}, Variables: []*VariableAssignment{nil}}},
		{"typed nil assignment value", &VariableAssignment{Kind: VariableAssignmentNames, Value: (*LiteralExpr)(nil)}},
		{"nil component value", &ComponentSetValue{}},
		{"typed nil component value", &ComponentSetValue{Value: (*LiteralExpr)(nil)}},
		{"nil install component item", &InstallComponentStmt{SetValues: []*ComponentSetValue{nil}}},
		{"nil diagnostics item", &GetDiagnosticsStmt{Items: []*DiagnosticsItem{nil}}},
		{"typed nil diagnostics condition", &GetDiagnosticsStmt{Condition: (*LiteralExpr)(nil)}},
		{"nil reset option", &ResetStmt{Options: []*ResetStmt{nil}}},
		{"nil reset argument", &ResetStmt{Args: []ExprNode{nil}}},
		{"typed nil replication value", &ReplicationOptionValue{Values: []ExprNode{(*LiteralExpr)(nil)}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := tc.node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil {
				t.Fatal("Restore succeeded with missing child")
			}
		})
	}
}

func TestManagementVisitorPreservesCrossExpressionReplacement(t *testing.T) {
	old := NewParamMarkerExpr(0)
	n := &ExecuteStmt{UsingVars: []ExprNode{old}}
	replacement := &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "9"}}
	visitor := spBoundaryVisitor{leave: func(node Node) (Node, bool) {
		if node == old {
			return replacement, true
		}
		return node, true
	}}
	if got, ok := n.Accept(visitor); !ok || got != n || n.UsingVars[0] != replacement {
		t.Fatal("valid expression replacement was lost")
	}
}

func TestExecuteVisitorStopsBeforeChangingRejectedSlot(t *testing.T) {
	for _, replacement := range []Node{nil, (*LiteralExpr)(nil), &ColumnPosition{}} {
		first, second := NewParamMarkerExpr(0), NewParamMarkerExpr(1)
		n := &ExecuteStmt{UsingVars: []ExprNode{first, second}}
		visitedSecond := false
		visitor := spBoundaryVisitor{
			enter: func(node Node) (Node, bool) {
				visitedSecond = visitedSecond || node == second
				return node, false
			},
			leave: func(node Node) (Node, bool) {
				if node == first {
					return replacement, true
				}
				return node, true
			},
		}
		if _, ok := n.Accept(visitor); ok || visitedSecond || n.UsingVars[0] != first || n.UsingVars[1] != second {
			t.Fatalf("replacement %T changed a rejected slot or continued traversal", replacement)
		}
	}
}

// A typed-nil replacement must not be written back into an expression slot:
// the parent's Accept has to fail and keep the original child.
func TestVisitorsRejectTypedNilChildReplacements(t *testing.T) {
	probe := NewParamMarkerExpr(0)
	joinOn := &JoinExpr{BaseJoinExpr: pkast.BaseJoinExpr{Left: &TableSource{}, Right: &TableName{}, On: probe}}
	updateWhere := &UpdateStmt{BaseUpdateStmt: pkast.BaseUpdateStmt{Table: &TableSource{}, Where: probe}}
	deleteWhere := &DeleteStmt{BaseDeleteStmt: pkast.BaseDeleteStmt{Table: &TableSource{}, Where: probe}}
	typeModifier := NewTypeName("VARCHAR")
	typeModifier.Modifiers = []ExprNode{probe}
	cases := []struct {
		name string
		root Node
		slot func() ExprNode
	}{
		{"group item", &GroupItem{BaseGroupItem: pkast.BaseGroupItem{Expr: probe}}, func() ExprNode { return probe }},
		{"partition item", &PartitionItem{Expr: probe}, func() ExprNode { return probe }},
		{"update where", updateWhere, func() ExprNode { return updateWhere.Where }},
		{"delete where", deleteWhere, func() ExprNode { return deleteWhere.Where }},
		{"join on", joinOn, func() ExprNode { return joinOn.On }},
		{"type modifier", typeModifier, func() ExprNode { return probe }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			visited := false
			visitor := spBoundaryVisitor{leave: func(n Node) (Node, bool) {
				if n == ExprNode(probe) {
					visited = true
					return (*LiteralExpr)(nil), true
				}
				return n, true
			}}
			if _, ok := tc.root.Accept(visitor); ok {
				t.Fatal("Accept accepted a typed-nil child replacement")
			}
			if !visited {
				t.Fatal("the child was never visited, so the rejection was not exercised")
			}
			if tc.slot() != probe {
				t.Fatal("typed-nil replacement was written back into the child slot")
			}
		})
	}
}

// An incomplete AST must not pass Accept: its Restore will fail, and the
// visitor contract treats ok as "the tree stayed structurally valid".
func TestVisitorsRejectIncompleteStructures(t *testing.T) {
	visitor := spBoundaryVisitor{leave: func(n Node) (Node, bool) { return n, true }}
	functionName := &TableName{Name: QualifiedName{Parts: []lex.Ident{{Name: "f"}}}}
	emptyTypeName := NewTypeName("ENUM")
	emptyTypeName.Modifiers = []ExprNode{nil}
	cases := []struct {
		name string
		root Node
	}{
		{"group item", &GroupItem{BaseGroupItem: pkast.BaseGroupItem{}}},
		{"partition item", &PartitionItem{}},
		{"row item", &RowExpr{BaseRowExpr: pkast.BaseRowExpr{Items: []ExprNode{nil}}}},
		{"type modifier", emptyTypeName},
		{"nil routine param", &CreateFunctionStmt{Name: functionName, Params: []*FuncParam{nil}}},
		{"for bounds", &ForStmt{Query: nil}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := tc.root.Accept(visitor); ok {
				t.Fatal("Accept accepted a structurally incomplete node")
			}
		})
	}
}

func TestLoadRestoreOmitsAbsentFieldAndLineOptions(t *testing.T) {
	n := &LoadDataStmt{Table: &TableName{Name: QualifiedName{Parts: []lex.Ident{{Name: "t"}}}}}
	var out strings.Builder
	if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "FIELDS") || strings.Contains(out.String(), "LINES") {
		t.Fatalf("Restore invented optional clauses: %q", out.String())
	}
	for _, target := range []*ColumnNameOrUserVar{nil, {}, {ColumnName: &QualifiedName{}, UserVar: &VariableExpr{}}} {
		n.ColumnsAndUserVars = []*ColumnNameOrUserVar{target}
		if _, ok := n.Accept(spBoundaryVisitor{}); ok {
			t.Fatal("Accept accepted a missing or ambiguous LOAD target")
		}
		if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil {
			t.Fatal("Restore accepted a missing or ambiguous LOAD target")
		}
	}
}

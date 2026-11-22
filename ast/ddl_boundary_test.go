package ast

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/model"
)

func TestAlterTableVisitorRejectsInvalidReplacements(t *testing.T) {
	wrong := NewParamMarkerExpr(0)
	for _, tc := range []struct {
		name       string
		node       Node
		target     string
		occurrence int
	}{
		{
			name:   "table child",
			node:   &AlterTableStmt{Table: &TableName{}, Specs: []*AlterTableSpec{{Tp: AlterTableDropPrimaryKey}}},
			target: "table",
		},
		{
			name:   "spec child",
			node:   &AlterTableStmt{Table: &TableName{}, Specs: []*AlterTableSpec{{Tp: AlterTableDropPrimaryKey}}},
			target: "spec",
		},
		{
			name:   "new table child",
			node:   &AlterTableSpec{Tp: AlterTableRenameTable, NewTable: &TableName{}},
			target: "table",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var visited int
			_, ok := tc.node.Accept(spBoundaryVisitor{leave: func(node Node) (Node, bool) {
				match := false
				switch tc.target {
				case "table":
					_, match = node.(*TableName)
				case "spec":
					_, match = node.(*AlterTableSpec)
				}
				if match && visited == tc.occurrence {
					visited++
					return wrong, true
				}
				return node, true
			}})
			if ok || visited != 1 {
				t.Fatalf("Accept ok=%v, replaced children=%d; want false, 1", ok, visited)
			}
		})
	}
}

func TestDDLVisitorRejectsInvalidRootReplacement(t *testing.T) {
	wrong := NewParamMarkerExpr(0)
	for _, node := range []Node{
		&CreateDatabaseStmt{},
		&AlterDatabaseStmt{},
		&DropDatabaseStmt{},
		&IndexOption{},
		&PeriodDefinition{},
		&ColumnPosition{},
		&DropTableStmt{},
		&DropFunctionStmt{},
		&DropProcedureStmt{},
		&DropTriggerStmt{},
		&DropEventStmt{},
		&CreateViewStmt{},
		&AlterViewStmt{},
	} {
		if _, ok := node.Accept(spBoundaryVisitor{enter: func(Node) (Node, bool) { return wrong, true }}); ok {
			t.Errorf("%T.Accept accepted an incompatible root replacement", node)
		}
	}
}

func TestDDLVisitorRejectsInvalidTableChildren(t *testing.T) {
	wrong := NewParamMarkerExpr(0)
	for _, tc := range []struct {
		name string
		node Node
	}{
		{name: "drop table child", node: &DropTableStmt{Tables: []*TableName{{}}}},
		{name: "drop function child", node: &DropFunctionStmt{Name: &TableName{}}},
		{name: "drop procedure child", node: &DropProcedureStmt{Name: &TableName{}}},
		{name: "drop trigger child", node: &DropTriggerStmt{Name: &TableName{}}},
		{name: "drop event child", node: &DropEventStmt{Name: &TableName{}}},
		{name: "create view child", node: &CreateViewStmt{ViewName: &TableName{}, Select: &SelectStmt{}}},
		{name: "alter view child", node: &AlterViewStmt{ViewName: &TableName{}, Select: &SelectStmt{}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var visited int
			_, ok := tc.node.Accept(spBoundaryVisitor{leave: func(node Node) (Node, bool) {
				if _, isTable := node.(*TableName); isTable && visited == 0 {
					visited++
					return wrong, true
				}
				return node, true
			}})
			if ok || visited != 1 {
				t.Fatalf("Accept ok=%v, replaced children=%d; want false, 1", ok, visited)
			}
		})
	}
}

func TestDDLRestoreRejectsInvalidBoundaryValues(t *testing.T) {
	for _, tc := range []struct {
		name string
		node interface {
			Restore(*format.RestoreCtx) error
		}
	}{
		{name: "nil table option", node: &AlterTableStmt{Table: &TableName{}, Specs: []*AlterTableSpec{{Tp: AlterTableOption, Options: []*TableOption{nil}}}}},
		{name: "empty alter table options", node: &AlterTableStmt{Table: &TableName{}, Specs: []*AlterTableSpec{{Tp: AlterTableOption}}}},
		{name: "nil partition definition", node: &PartitionOptions{Tp: model.PartitionTypeRange, Expr: NewParamMarkerExpr(0), Definitions: []*PartitionDefinition{nil}}},
		{name: "nil partition option", node: &PartitionDefinition{Options: []*TableOption{nil}}},
		{name: "missing AFTER target", node: &ColumnPosition{Tp: ColumnPositionAfter}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := tc.node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil {
				t.Fatal("Restore succeeded, want validation error")
			}
		})
	}
}

func TestCallAndValuesVisitorsRejectInvalidReplacements(t *testing.T) {
	for _, tc := range []struct {
		name        string
		root        Node
		target      string
		replacement Node
	}{
		{name: "call root", root: &CallStmt{Name: &TableName{}}, target: "root", replacement: NewParamMarkerExpr(0)},
		{name: "call root typed nil", root: &CallStmt{Name: &TableName{}}, target: "root", replacement: (*CallStmt)(nil)},
		{name: "call name", root: &CallStmt{Name: &TableName{}}, target: "table", replacement: NewParamMarkerExpr(0)},
		{name: "call name typed nil", root: &CallStmt{Name: &TableName{}}, target: "table", replacement: (*TableName)(nil)},
		{name: "values root", root: &ValuesExpr{Column: &ColumnRef{}}, target: "root", replacement: NewParamMarkerExpr(0)},
		{name: "values root typed nil", root: &ValuesExpr{Column: &ColumnRef{}}, target: "root", replacement: (*ValuesExpr)(nil)},
		{name: "values column", root: &ValuesExpr{Column: &ColumnRef{}}, target: "column", replacement: NewParamMarkerExpr(0)},
		{name: "values column typed nil", root: &ValuesExpr{Column: &ColumnRef{}}, target: "column", replacement: (*ColumnRef)(nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var replaced bool
			visitor := spBoundaryVisitor{
				enter: func(node Node) (Node, bool) {
					if tc.target == "root" && node == tc.root {
						replaced = true
						return tc.replacement, false
					}
					return node, false
				},
				leave: func(node Node) (Node, bool) {
					match := false
					switch tc.target {
					case "table":
						_, match = node.(*TableName)
					case "column":
						_, match = node.(*ColumnRef)
					}
					if match {
						replaced = true
						return tc.replacement, true
					}
					return node, true
				},
			}
			if _, ok := tc.root.Accept(visitor); ok || !replaced {
				t.Fatalf("Accept ok=%v, replacement reached=%v; want false, true", ok, replaced)
			}
		})
	}
}

func TestCallAndValuesRestoreRejectNilChildren(t *testing.T) {
	for _, tc := range []struct {
		name string
		node interface {
			Restore(*format.RestoreCtx) error
		}
	}{
		{name: "nil call", node: (*CallStmt)(nil)},
		{name: "call without name", node: &CallStmt{}},
		{name: "call with typed nil name", node: &CallStmt{Name: (*TableName)(nil)}},
		{name: "call with nil argument", node: &CallStmt{Name: &TableName{}, Args: []ExprNode{nil}}},
		{name: "call with typed nil argument", node: &CallStmt{Name: &TableName{}, Args: []ExprNode{(*LiteralExpr)(nil)}}},
		{name: "nil values expression", node: (*ValuesExpr)(nil)},
		{name: "values without column", node: &ValuesExpr{}},
		{name: "values with typed nil column", node: &ValuesExpr{Column: (*ColumnRef)(nil)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := tc.node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil {
				t.Fatal("Restore succeeded with a missing child")
			}
		})
	}
}

func TestTableOptionVisitorWriteBack(t *testing.T) {
	replacement := &TableOption{Tp: TableOptionEngine, StrValue: "InnoDB"}
	for _, tc := range []struct {
		name   string
		node   Node
		option func(Node) *TableOption
	}{
		{
			name:   "create table",
			node:   &CreateTableStmt{Table: &TableName{}, Options: []*TableOption{{Tp: TableOptionComment, StrValue: "x"}}},
			option: func(node Node) *TableOption { return node.(*CreateTableStmt).Options[0] },
		},
		{
			name: "alter table spec",
			node: &AlterTableStmt{
				Table: &TableName{},
				Specs: []*AlterTableSpec{{Tp: AlterTableOption, Options: []*TableOption{{Tp: TableOptionComment, StrValue: "x"}}}},
			},
			option: func(node Node) *TableOption { return node.(*AlterTableStmt).Specs[0].Options[0] },
		},
		{
			name:   "partition definition",
			node:   &PartitionDefinition{Options: []*TableOption{{Tp: TableOptionComment, StrValue: "x"}}},
			option: func(node Node) *TableOption { return node.(*PartitionDefinition).Options[0] },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := tc.node.Accept(spBoundaryVisitor{leave: func(node Node) (Node, bool) {
				if _, isOption := node.(*TableOption); isOption {
					return replacement, true
				}
				return node, true
			}}); !ok {
				t.Fatal("Accept failed")
			}
			if got := tc.option(tc.node); got != replacement {
				t.Fatalf("table option not written back: got %+v", got)
			}
			if _, ok := tc.node.Accept(spBoundaryVisitor{leave: func(node Node) (Node, bool) {
				if _, isOption := node.(*TableOption); isOption {
					return NewParamMarkerExpr(0), true
				}
				return node, true
			}}); ok {
				t.Fatal("Accept accepted an incompatible table option replacement")
			}
			if got := tc.option(tc.node); got != replacement {
				t.Fatalf("rejected replacement still written back: got %+v", got)
			}
		})
	}
}

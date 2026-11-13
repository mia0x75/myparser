package ast_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/driver"

	"github.com/mia0x75/myparser"
	. "github.com/mia0x75/myparser/ast"
	. "github.com/mia0x75/myparser/format"
)

func TestCacheable(t *testing.T) {
	// test non-SelectStmt
	var stmt Node = &DeleteStmt{}
	if IsReadOnly(stmt) {
		t.Fatal("DeleteStmt is expected to be not read only")
	}

	stmt = &InsertStmt{}
	if IsReadOnly(stmt) {
		t.Fatal("InsertStmt is expected to be not read only")
	}

	stmt = &UpdateStmt{}
	if IsReadOnly(stmt) {
		t.Fatal("UpdateStmt is expected to be not read only")
	}

	stmt = &ExplainStmt{}
	if !IsReadOnly(stmt) {
		t.Fatal("ExplainStmt is expected to be read only")
	}

	stmt = &ExplainStmt{}
	if !IsReadOnly(stmt) {
		t.Fatal("ExplainStmt is expected to be read only")
	}

	stmt = &DoStmt{}
	if !IsReadOnly(stmt) {
		t.Fatal("DoStmt is expected to be read only")
	}
}

// CleanNodeText set the text of node and all child node empty.
// For test only.
func CleanNodeText(node Node) {
	var cleaner nodeTextCleaner
	node.Accept(&cleaner)
}

// nodeTextCleaner clean the text of a node and it's child node.
// For test only.
type nodeTextCleaner struct {
}

// Enter implements Visitor interface.
func (checker *nodeTextCleaner) Enter(in Node) (out Node, skipChildren bool) {
	in.SetText("")
	switch node := in.(type) {
	case *Constraint:
		if node.Option != nil {
			if node.Option.KeyBlockSize == 0x0 && node.Option.Tp == 0 && node.Option.Comment == "" {
				node.Option = nil
			}
		}
	case *FuncCallExpr:
		node.FnName.O = strings.ToLower(node.FnName.O)
		switch node.FnName.L {
		case "convert":
			node.Args[1].(*driver.ValueExpr).SetBytes(nil)
		}
	case *AggregateFuncExpr:
		node.F = strings.ToLower(node.F)
	case *WindowFuncExpr:
		node.F = strings.ToLower(node.F)
	case *FieldList:
		for _, f := range node.Fields {
			f.Offset = 0
		}
	case *TableName:
		node.NameOffset = 0
	case *PeriodDefinition:
		node.Offset = 0
	case *PartitionDefinition:
		node.NameOffset = 0
	case *IndexColName:
		node.ExprOffset = 0
	case *ValuesStmt:
		node.ValuesKeywordOffset = 0
	case *FrameClause:
		node.TypeOffset, node.ExclusionOffset = 0, 0
	case *FrameBound:
		node.ExprOffset = 0
	case *InsertStmt:
		if node.RowAlias != nil {
			node.RowAlias.Offset = 0
		}
	case *AlterTableSpec:
		for _, opt := range node.Options {
			opt.StrValue = strings.ToLower(opt.StrValue)
		}
	}
	return in, false
}

// Leave implements Visitor interface.
func (checker *nodeTextCleaner) Leave(in Node) (out Node, ok bool) {
	return in, true
}

type NodeRestoreTestCase struct {
	sourceSQL string
	expectSQL string
}

func RunNodeRestoreTest(t *testing.T, nodeTestCases []NodeRestoreTestCase, template string, extractNodeFunc func(node Node) Node) {
	RunNodeRestoreTestWithFlags(t, nodeTestCases, template, extractNodeFunc, DefaultRestoreFlags)
}

func RunNodeRestoreTestWithFlags(t *testing.T, nodeTestCases []NodeRestoreTestCase, template string, extractNodeFunc func(node Node) Node, flags RestoreFlags) {
	p := myparser.New()
	p.EnableWindowFunc(true)
	for _, testCase := range nodeTestCases {
		sourceSQL := fmt.Sprintf(template, testCase.sourceSQL)
		expectSQL := fmt.Sprintf(template, testCase.expectSQL)
		stmt, err := p.ParseOneStmt(sourceSQL, "", "")
		comment := fmt.Sprintf("source %#v", testCase)
		if err != nil {
			t.Fatalf("%s: %v", comment, err)
		}
		var sb strings.Builder
		err = extractNodeFunc(stmt).Restore(NewRestoreCtx(flags, &sb))
		if err != nil {
			t.Fatalf("%s: %v", comment, err)
		}
		restoreSql := fmt.Sprintf(template, sb.String())
		comment = fmt.Sprintf("source %#v; restore %v", testCase, restoreSql)
		if restoreSql != expectSQL {
			t.Fatalf("%s: got %#v, want %#v", comment, restoreSql, expectSQL)
		}
		stmt2, err := p.ParseOneStmt(restoreSql, "", "")
		if err != nil {
			t.Fatalf("%s: %v", comment, err)
		}
		CleanNodeText(stmt)
		CleanNodeText(stmt2)
		if !reflect.DeepEqual(stmt2, stmt) {
			t.Fatalf("%s: got %#v, want %#v", comment, stmt2, stmt)
		}
	}
}

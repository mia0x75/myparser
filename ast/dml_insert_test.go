package ast_test

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
)

type insertTestTable struct {
	pkast.BaseTableRef
	err error
}

func (n *insertTestTable) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *insertTestTable) Restore(ctx *format.RestoreCtx) error {
	if n.err != nil {
		return n.err
	}
	ctx.WriteName("t")
	return ctx.Err()
}

type insertTestQuery struct {
	pkast.BaseQuery
	err error
}

func (n *insertTestQuery) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *insertTestQuery) Restore(ctx *format.RestoreCtx) error {
	if n.err != nil {
		return n.err
	}
	ctx.WriteKeyword("SELECT 1")
	return ctx.Err()
}

var _ pkast.TableRefNode = (*insertTestTable)(nil)
var _ ast.QueryNode = (*insertTestQuery)(nil)
var _ ast.DMLNode = (*ast.InsertStmt)(nil)

// A deliberately mixed manual tree exercises every slot without altering the
// parser's mutually exclusive input forms.
func newInsertTestTree() (*ast.InsertStmt, []ast.Node) {
	query, table := &insertTestQuery{}, &insertTestTable{}
	column, value := &sharedPureExpr{}, &sharedPureExpr{}
	setTarget, setValue := &sharedPureExpr{}, &sharedPureExpr{}
	set := &ast.Assignment{Target: setTarget, Expr: setValue}
	conflictTarget, conflictValue := &sharedPureExpr{}, &sharedPureExpr{}
	conflict := &ast.Assignment{Target: conflictTarget, Expr: conflictValue}
	returnValue := &sharedPureExpr{}
	item := &ast.SelectItem{Expr: returnValue}
	returning := &ast.FieldList{Items: []*ast.SelectItem{item}}
	stmt := &ast.InsertStmt{
		BaseInsertStmt: pkast.BaseInsertStmt{
			Table: table, Query: query, Columns: []ast.ExprNode{column},
			Values: [][]ast.ExprNode{{value}}, Returning: returning,
		},
		Assignments: []*ast.Assignment{set}, OnDuplicate: []*ast.Assignment{conflict},
	}
	return stmt, []ast.Node{query, table, column, value, set, setTarget, setValue,
		conflict, conflictTarget, conflictValue, returning, item, returnValue}
}

func TestInsertVisitorOrderAndOuterReplacement(t *testing.T) {
	original, originalOrder := newInsertTestTree()
	replacement, order := newInsertTestTree()
	var entered []ast.Node
	result, ok := original.Accept(operatorVisitor{
		enter: func(node ast.Node) (ast.Node, bool) {
			entered = append(entered, node)
			if node == original {
				return replacement, false
			}
			return node, false
		},
		leave: func(node ast.Node) (ast.Node, bool) { return node, true },
	})
	want := append([]ast.Node{original}, order...)
	if !ok || result != replacement || !reflect.DeepEqual(entered, want) {
		t.Fatalf("outer identity or traversal order lost: ok=%v, entered=%v, want=%v", ok, entered, want)
	}
	if original.Query != originalOrder[0] || original.Table != originalOrder[1] || original.Columns[0] != originalOrder[2] {
		t.Fatal("Enter replacement must not mutate or visit the original tree")
	}
	visited, left := 0, false
	result, ok = original.Accept(operatorVisitor{
		enter: func(node ast.Node) (ast.Node, bool) {
			visited++
			return replacement, true
		},
		leave: func(node ast.Node) (ast.Node, bool) { left = node == replacement; return node, true },
	})
	if !ok || result != replacement || visited != 1 || !left {
		t.Fatal("skip must leave the replacement outer without traversing children")
	}
}

func TestInsertVisitorChildReplacement(t *testing.T) {
	n, old := newInsertTestTree()
	query, table := &insertTestQuery{}, &insertTestTable{}
	column, value := &sharedPureExpr{}, &sharedPureExpr{}
	set, conflict := &ast.Assignment{}, &ast.Assignment{}
	returning := &ast.FieldList{}
	replacements := map[ast.Node]ast.Node{
		old[0]: query, old[1]: table, old[2]: column, old[3]: value,
		old[4]: set, old[7]: conflict, old[10]: returning,
	}
	result, ok := n.Accept(operatorVisitor{
		enter: func(node ast.Node) (ast.Node, bool) { return node, false },
		leave: func(node ast.Node) (ast.Node, bool) {
			if replacement, found := replacements[node]; found {
				return replacement, true
			}
			return node, true
		},
	})
	if !ok || result != n || n.Query != query || n.Table != table || n.Columns[0] != column ||
		n.Values[0][0] != value || n.Assignments[0] != set || n.OnDuplicate[0] != conflict || n.Returning != returning {
		t.Fatal("a compatible replacement was not written back to its own slot")
	}
}

func TestInsertVisitorStopAtEveryChild(t *testing.T) {
	_, order := newInsertTestTree()
	for stopIndex := range order {
		t.Run(fmt.Sprint(stopIndex), func(t *testing.T) {
			n, order := newInsertTestTree()
			var entered []ast.Node
			parentLeft := false
			_, ok := n.Accept(operatorVisitor{
				enter: func(node ast.Node) (ast.Node, bool) {
					if node != n {
						entered = append(entered, node)
					}
					return node, false
				},
				leave: func(node ast.Node) (ast.Node, bool) {
					parentLeft = parentLeft || node == n
					return node, node != order[stopIndex]
				},
			})
			// A parent's Leave follows its children, so stop prefixes end at the
			// last descendant already entered, not necessarily the parent itself.
			last := stopIndex
			switch stopIndex {
			case 4, 7:
				last += 2
			case 10:
				last += 2
			case 11:
				last++
			}
			if ok || parentLeft || !reflect.DeepEqual(entered, order[:last+1]) {
				t.Fatalf("stop did not propagate: ok=%v parentLeft=%v entered=%v", ok, parentLeft, entered)
			}
		})
	}
}

func TestInsertVisitorRejectsInvalidReplacements(t *testing.T) {
	for _, replacement := range []ast.Node{nil, (*ast.InsertStmt)(nil), &ast.Assignment{}} {
		for _, skip := range []bool{false, true} {
			n, _ := newInsertTestTree()
			left := false
			_, ok := n.Accept(operatorVisitor{
				enter: func(ast.Node) (ast.Node, bool) { return replacement, skip },
				leave: func(node ast.Node) (ast.Node, bool) { left = true; return node, true },
			})
			if ok || left {
				t.Fatal("invalid Enter replacement must stop before traversal or Leave")
			}
		}
	}
	for _, slot := range []struct {
		name  string
		index int
		get   func(*ast.InsertStmt) ast.Node
		wrong ast.Node
	}{
		{"query", 0, func(n *ast.InsertStmt) ast.Node { return n.Query }, &insertTestTable{}},
		{"table", 1, func(n *ast.InsertStmt) ast.Node { return n.Table }, &insertTestQuery{}},
		{"column", 2, func(n *ast.InsertStmt) ast.Node { return n.Columns[0] }, &ast.Assignment{}},
		{"value", 3, func(n *ast.InsertStmt) ast.Node { return n.Values[0][0] }, &ast.Assignment{}},
		{"set", 4, func(n *ast.InsertStmt) ast.Node { return n.Assignments[0] }, &ast.FieldList{}},
		{"conflict", 7, func(n *ast.InsertStmt) ast.Node { return n.OnDuplicate[0] }, &ast.FieldList{}},
		{"returning", 10, func(n *ast.InsertStmt) ast.Node { return n.Returning }, &ast.Assignment{}},
	} {
		t.Run(slot.name, func(t *testing.T) {
			invalids := []ast.Node{nil, slot.wrong}
			switch slot.name {
			case "set", "conflict":
				invalids = append(invalids, (*ast.Assignment)(nil))
			case "returning":
				invalids = append(invalids, (*ast.FieldList)(nil))
			}
			for _, invalid := range invalids {
				n, order := newInsertTestTree()
				target := order[slot.index]
				_, ok := n.Accept(operatorVisitor{
					enter: func(node ast.Node) (ast.Node, bool) { return node, false },
					leave: func(node ast.Node) (ast.Node, bool) {
						if node == target {
							return invalid, true
						}
						return node, true
					},
				})
				if ok || slot.get(n) != target {
					t.Fatal("incompatible child replacement must stop without changing the slot")
				}
			}
		})
	}
}

func TestInsertRestorePresenceAndErrorChains(t *testing.T) {
	for _, n := range []*ast.InsertStmt{nil, {}} {
		var out strings.Builder
		if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil || out.Len() != 0 {
			t.Fatalf("missing target must fail before output: %q, %v", out.String(), err)
		}
	}
	for _, tc := range []struct {
		name string
		n    *ast.InsertStmt
		want string
	}{
		{"empty set", &ast.InsertStmt{BaseInsertStmt: pkast.BaseInsertStmt{Table: &insertTestTable{}}, Assignments: []*ast.Assignment{}}, "INSERT INTO `t` SET "},
		{"empty columns", &ast.InsertStmt{BaseInsertStmt: pkast.BaseInsertStmt{Table: &insertTestTable{}, Columns: []ast.ExprNode{}, Values: [][]ast.ExprNode{{&sharedPureExpr{}}}}}, "INSERT INTO `t` () VALUES (PURE)"},
		{"query category", &ast.InsertStmt{BaseInsertStmt: pkast.BaseInsertStmt{Table: &insertTestTable{}, Query: &insertTestQuery{}}}, "INSERT INTO `t` SELECT 1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := tc.n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil || out.String() != tc.want {
				t.Fatalf("output=%q want=%q err=%v", out.String(), tc.want, err)
			}
		})
	}
	for _, tc := range []struct {
		path      string
		breakNode func(*ast.InsertStmt)
	}{
		{"Table", func(n *ast.InsertStmt) { n.Table = &insertTestTable{err: &pureRestoreError{}} }},
		{"Columns[0]", func(n *ast.InsertStmt) { n.Columns[0] = &sharedPureExpr{restoreErr: true} }},
		{"Values[0][0]", func(n *ast.InsertStmt) { n.Values[0][0] = &sharedPureExpr{restoreErr: true} }},
		{"Query", func(n *ast.InsertStmt) { n.Query = &insertTestQuery{err: &pureRestoreError{}} }},
		{"Assignments[0]", func(n *ast.InsertStmt) { n.Assignments[0].Expr = &sharedPureExpr{restoreErr: true} }},
		{"OnDuplicate[0]", func(n *ast.InsertStmt) { n.OnDuplicate[0].Expr = &sharedPureExpr{restoreErr: true} }},
		{"Returning", func(n *ast.InsertStmt) { n.Returning.Items[0].Expr = &sharedPureExpr{restoreErr: true} }},
	} {
		t.Run(tc.path, func(t *testing.T) {
			n, _ := newInsertTestTree()
			tc.breakNode(n)
			err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, io.Discard))
			var cause *pureRestoreError
			if !errors.As(err, &cause) || !strings.Contains(err.Error(), "InsertStmt."+tc.path) {
				t.Fatalf("child error chain or path lost: %v", err)
			}
		})
	}
	n, _ := newInsertTestTree()
	if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, failingWriter{})); err == nil {
		t.Fatal("writer failure must propagate")
	}
}

func TestInsertRejectsMissingExpressions(t *testing.T) {
	for _, tc := range []struct {
		name  string
		clear func(*ast.InsertStmt)
	}{
		{"column", func(n *ast.InsertStmt) { n.Columns[0] = nil }},
		{"value", func(n *ast.InsertStmt) { n.Values[0][0] = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, _ := newInsertTestTree()
			tc.clear(n)
			if _, ok := n.Accept(operatorVisitor{
				enter: func(node ast.Node) (ast.Node, bool) { return node, false },
				leave: func(node ast.Node) (ast.Node, bool) { return node, true },
			}); ok {
				t.Fatal("an absent required expression must stop traversal")
			}
			if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, io.Discard)); err == nil {
				t.Fatal("an absent required expression must fail restoration")
			}
		})
	}
}

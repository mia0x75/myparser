package ast_test

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser"
	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
)

type dmlTestOrder struct {
	pkast.BaseOrderByClause
	err error
}

func (n *dmlTestOrder) OrderByClauseCore() *pkast.BaseOrderByClause {
	if n == nil {
		return nil
	}
	return &n.BaseOrderByClause
}

func (n *dmlTestOrder) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *dmlTestOrder) Restore(ctx *format.RestoreCtx) error {
	if n.err != nil {
		return n.err
	}
	ctx.WriteKeyword("ORDER BY 7")
	return ctx.Err()
}

type dmlTestLimit struct {
	pkast.BaseLimit
	err error
}

func (n *dmlTestLimit) LimitCore() *pkast.BaseLimit {
	if n == nil {
		return nil
	}
	return &n.BaseLimit
}

func (n *dmlTestLimit) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *dmlTestLimit) Restore(ctx *format.RestoreCtx) error {
	if n.err != nil {
		return n.err
	}
	ctx.WriteKeyword("LIMIT 8")
	return ctx.Err()
}

func dmlParse(t *testing.T, sql string) ast.StmtNode {
	t.Helper()
	n, err := myparser.New().ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestUpdateDeleteStrongClauseDispatch(t *testing.T) {
	for _, sql := range []string{
		"UPDATE t SET a=1 ORDER BY a LIMIT 1",
		"DELETE FROM t ORDER BY a LIMIT 1",
	} {
		t.Run(sql, func(t *testing.T) {
			n := dmlParse(t, sql)
			order, limit := &dmlTestOrder{}, &dmlTestLimit{}
			_, ok := n.Accept(operatorVisitor{
				enter: func(node ast.Node) (ast.Node, bool) { return node, false },
				leave: func(node ast.Node) (ast.Node, bool) {
					switch node.(type) {
					case *ast.OrderByClause:
						return order, true
					case *ast.Limit:
						return limit, true
					}
					return node, true
				},
			})
			if !ok || order.OrderByClauseCore() != &order.BaseOrderByClause || limit.LimitCore() != &limit.BaseLimit {
				t.Fatal("compatible clause replacement or Core identity lost")
			}
			var out strings.Builder
			if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil || !strings.HasSuffix(out.String(), "ORDER BY 7 LIMIT 8") {
				t.Fatalf("outer restoration bypassed: %s, %v", out.String(), err)
			}
			seen := map[ast.Node]int{}
			_, ok = n.Accept(operatorVisitor{
				enter: func(node ast.Node) (ast.Node, bool) { seen[node]++; return node, false },
				leave: func(node ast.Node) (ast.Node, bool) { return node, true },
			})
			if !ok || seen[order] != 1 || seen[limit] != 1 {
				t.Fatal("outer clause traversal bypassed or duplicated")
			}
		})
	}
}

func TestUpdateDeleteVisitorRejectsOuterAndChildReplacements(t *testing.T) {
	for _, sql := range []string{
		"WITH c AS (SELECT 1) UPDATE t SET a=1 WHERE b=2 ORDER BY a LIMIT 1 RETURNING a",
		"WITH c AS (SELECT 1) DELETE FROM t WHERE b=2 ORDER BY a LIMIT 1 RETURNING a",
		"WITH c AS (SELECT 1) DELETE FROM t,s USING t JOIN s USING(a) WHERE b=2 ORDER BY a LIMIT 1",
	} {
		t.Run(sql, func(t *testing.T) {
			for _, skip := range []bool{false, true} {
				for _, invalid := range []ast.Node{nil, (*ast.UpdateStmt)(nil), (*ast.DeleteStmt)(nil), &ast.FieldList{}} {
					n := dmlParse(t, sql)
					left := false
					_, ok := n.Accept(operatorVisitor{
						enter: func(ast.Node) (ast.Node, bool) { return invalid, skip },
						leave: func(node ast.Node) (ast.Node, bool) { left = true; return node, true },
					})
					if ok || left {
						t.Fatal("invalid outer reached children or Leave")
					}
				}
			}
			getSlots := func(n ast.StmtNode) []ast.Node {
				switch n := n.(type) {
				case *ast.UpdateStmt:
					return []ast.Node{n.With, n.Table, n.Assignments[0], n.Where, n.OrderBy, n.Limit, n.Returning}
				case *ast.DeleteStmt:
					if n.IsMultiTable {
						return []ast.Node{n.With, n.Using, n.Targets[0], n.Targets[1], n.Where, n.OrderBy, n.Limit}
					}
					return []ast.Node{n.With, n.Table, n.Where, n.OrderBy, n.Limit, n.Returning}
				default:
					t.Fatal("bad fixture")
					return nil
				}
			}
			for i := range getSlots(dmlParse(t, sql)) {
				t.Run(fmt.Sprint(i), func(t *testing.T) {
					for _, invalid := range []ast.Node{nil, &ast.WindowSpec{}, (*ast.WithClause)(nil), (*ast.Assignment)(nil), (*ast.TableName)(nil), (*ast.TableSource)(nil), (*ast.JoinExpr)(nil), (*ast.FieldList)(nil), (*dmlTestOrder)(nil), (*dmlTestLimit)(nil)} {
						n := dmlParse(t, sql)
						before := getSlots(n)
						_, ok := n.Accept(operatorVisitor{
							enter: func(node ast.Node) (ast.Node, bool) { return node, node != n },
							leave: func(node ast.Node) (ast.Node, bool) {
								if node == before[i] {
									return invalid, true
								}
								return node, true
							},
						})
						if ok || getSlots(n)[i] != before[i] {
							t.Fatalf("slot %d accepted %T or changed before rejecting", i, invalid)
						}
					}
					for _, stop := range []bool{false, true} {
						n := dmlParse(t, sql)
						children := getSlots(n)
						var entered []ast.Node
						parentLeft := false
						_, ok := n.Accept(operatorVisitor{
							enter: func(node ast.Node) (ast.Node, bool) {
								if node != n {
									entered = append(entered, node)
								}
								return node, node != n
							},
							leave: func(node ast.Node) (ast.Node, bool) {
								parentLeft = parentLeft || node == n
								return node, !stop || node != children[i]
							},
						})
						want := children
						if stop {
							want = children[:i+1]
						}
						if ok == stop || parentLeft == stop || !reflect.DeepEqual(entered, want) {
							t.Fatal("direct child order, skip, or stop propagation changed")
						}
					}
				})
			}
		})
	}
}

type dmlLevelRelation struct {
	pkast.BaseResultSet
	err   error
	level int
}

func (n *dmlLevelRelation) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *dmlLevelRelation) Restore(ctx *format.RestoreCtx) error {
	n.level = ctx.JoinLevel
	ctx.JoinLevel = 99
	if n.err != nil {
		return n.err
	}
	ctx.WriteName("t")
	return ctx.Err()
}

func TestUpdateDeleteRestoreJoinLevelAndErrorChains(t *testing.T) {
	cause := errors.New("DML child failure")
	for _, sql := range []string{"UPDATE t SET a=1", "DELETE FROM t", "DELETE t FROM t", "DELETE FROM t USING t"} {
		t.Run(sql, func(t *testing.T) {
			for _, fail := range []bool{false, true} {
				n := dmlParse(t, sql)
				relation := &dmlLevelRelation{}
				if fail {
					relation.err = cause
				}
				switch n := n.(type) {
				case *ast.UpdateStmt:
					n.Table = relation
				case *ast.DeleteStmt:
					if n.IsMultiTable {
						n.Using = relation
					} else {
						n.Table = relation
					}
				}
				ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, io.Discard)
				ctx.JoinLevel = 42
				err := n.Restore(ctx)
				if relation.level != 0 || ctx.JoinLevel != 42 || (fail && !errors.Is(err, cause)) || (!fail && err != nil) {
					t.Fatalf("join-level/error restoration: seen=%d final=%d err=%v", relation.level, ctx.JoinLevel, err)
				}
			}
			n := dmlParse(t, sql)
			for _, writer := range []io.Writer{failingWriter{}, dmlShortWriter{}} {
				if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, writer)); err == nil {
					t.Fatal("writer failure or short write lost")
				}
			}
		})
	}
	for _, sql := range []string{"UPDATE t SET a=1 ORDER BY a LIMIT 1", "DELETE FROM t ORDER BY a LIMIT 1"} {
		for _, path := range []string{"order", "limit"} {
			t.Run(sql+path, func(t *testing.T) {
				n := dmlParse(t, sql)
				_, ok := n.Accept(operatorVisitor{
					enter: func(node ast.Node) (ast.Node, bool) { return node, false },
					leave: func(node ast.Node) (ast.Node, bool) {
						if _, ok := node.(*ast.OrderByClause); ok && path == "order" {
							return &dmlTestOrder{err: cause}, true
						}
						if _, ok := node.(*ast.Limit); ok && path == "limit" {
							return &dmlTestLimit{err: cause}, true
						}
						return node, true
					},
				})
				if !ok || !errors.Is(n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, io.Discard)), cause) {
					t.Fatal("extended clause error chain lost")
				}
			})
		}
	}
}

type dmlShortWriter struct{}

func (dmlShortWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return len(p) - 1, nil
}

func TestUpdateDeleteRejectInconsistentManualTrees(t *testing.T) {
	for _, tc := range []struct {
		name string
		n    ast.StmtNode
	}{
		{"nil_update", (*ast.UpdateStmt)(nil)},
		{"empty_update", &ast.UpdateStmt{}},
		{"nil_delete", (*ast.DeleteStmt)(nil)},
		{"empty_delete", &ast.DeleteStmt{}},
		{"missing_assignments", &ast.UpdateStmt{BaseUpdateStmt: pkast.BaseUpdateStmt{Table: &insertTestTable{}}}},
		{"nil_assignment", &ast.UpdateStmt{BaseUpdateStmt: pkast.BaseUpdateStmt{Table: &insertTestTable{}, Assignments: []*ast.Assignment{nil}}}},
		{"single_using", &ast.DeleteStmt{BaseDeleteStmt: pkast.BaseDeleteStmt{Table: &insertTestTable{}, Using: &insertTestTable{}}}},
		{"single_targets", &ast.DeleteStmt{BaseDeleteStmt: pkast.BaseDeleteStmt{Table: &insertTestTable{}}, Targets: []*ast.TableName{{}}}},
		{"single_before", &ast.DeleteStmt{BaseDeleteStmt: pkast.BaseDeleteStmt{Table: &insertTestTable{}}, BeforeFrom: true}},
		{"multi_table", &ast.DeleteStmt{BaseDeleteStmt: pkast.BaseDeleteStmt{Table: &insertTestTable{}, Using: &insertTestTable{}}, IsMultiTable: true, Targets: []*ast.TableName{{}}}},
		{"multi_no_source", &ast.DeleteStmt{IsMultiTable: true, Targets: []*ast.TableName{{}}}},
		{"multi_no_targets", &ast.DeleteStmt{BaseDeleteStmt: pkast.BaseDeleteStmt{Using: &insertTestTable{}}, IsMultiTable: true}},
		{"multi_nil_target", &ast.DeleteStmt{BaseDeleteStmt: pkast.BaseDeleteStmt{Using: &insertTestTable{}}, IsMultiTable: true, Targets: []*ast.TableName{nil}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := tc.n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil || out.Len() != 0 {
				t.Fatalf("inconsistent tree not rejected before output: %q, %v", out.String(), err)
			}
		})
	}
	for _, sql := range []string{"UPDATE t SET a=1", "DELETE FROM t"} {
		for _, path := range []string{"hint", "order", "limit"} {
			t.Run(sql+path, func(t *testing.T) {
				n := dmlParse(t, sql)
				switch n := n.(type) {
				case *ast.UpdateStmt:
					switch path {
					case "hint":
						n.TableHints = []*ast.OptimizerHint{nil}
					case "order":
						n.OrderBy = (*ast.OrderByClause)(nil)
					case "limit":
						n.Limit = (*ast.Limit)(nil)
					}
				case *ast.DeleteStmt:
					switch path {
					case "hint":
						n.TableHints = []*ast.OptimizerHint{nil}
					case "order":
						n.OrderBy = (*ast.OrderByClause)(nil)
					case "limit":
						n.Limit = (*ast.Limit)(nil)
					}
				}
				if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, io.Discard)); err == nil {
					t.Fatal("nil hint or typed-nil clause accepted")
				}
			})
		}
	}
}

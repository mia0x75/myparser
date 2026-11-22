package ast_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/auth"
	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
)

type databaseRefVisitor struct {
	enter func(ast.Node) (ast.Node, bool)
	leave func(ast.Node) (ast.Node, bool)
}

func (v databaseRefVisitor) Enter(n ast.Node) (ast.Node, bool) {
	if v.enter != nil {
		return v.enter(n)
	}
	return n, false
}

func (v databaseRefVisitor) Leave(n ast.Node) (ast.Node, bool) {
	if v.leave != nil {
		return v.leave(n)
	}
	return n, true
}

type databaseRefStmt struct {
	pkast.BaseStmt
	err error
}

func (n *databaseRefStmt) Accept(v ast.Visitor) (ast.Node, bool) {
	next, _ := v.Enter(n)
	return v.Leave(next)
}

func (n *databaseRefStmt) Restore(*format.RestoreCtx) error { return n.err }

func TestDatabaseReferenceVisitorEnterBoundary(t *testing.T) {
	for _, root := range []ast.Node{&ast.UseStmt{}, &ast.ShowStmt{}, &ast.ExplainStmt{}} {
		for _, skip := range []bool{false, true} {
			for _, replacement := range []ast.Node{nil, (*ast.UseStmt)(nil), (*ast.ShowStmt)(nil), (*ast.ExplainStmt)(nil), &ast.Limit{}} {
				v := databaseRefVisitor{enter: func(ast.Node) (ast.Node, bool) { return replacement, skip }}
				if _, ok := root.Accept(v); ok {
					t.Fatalf("%T accepted %T, skip=%v", root, replacement, skip)
				}
			}
		}
	}
}

func TestShowVisitorVisitsEveryWhere(t *testing.T) {
	for _, tp := range []ast.ShowStmtType{
		ast.ShowTriggers, ast.ShowProcedureStatus, ast.ShowProcessList, ast.ShowEvents,
	} {
		where := pureLiteral("1")
		show := &ast.ShowStmt{Tp: tp, Where: where}
		visited := false
		v := databaseRefVisitor{enter: func(n ast.Node) (ast.Node, bool) {
			visited = visited || n == where
			return n, false
		}}
		if _, ok := show.Accept(v); !ok || !visited {
			t.Fatalf("SHOW %v skipped WHERE (ok=%v)", tp, ok)
		}
	}
}

func TestShowVisitorReplacementAndOrder(t *testing.T) {
	original := &ast.ShowStmt{Where: pureLiteral("old")}
	where := pureLiteral("new")
	next := &ast.ShowStmt{
		Table: &ast.TableName{}, Pattern: &ast.PatternLikeExpr{}, Where: where,
		Limit: &ast.Limit{}, ShowProfileLimit: &ast.Limit{},
	}
	var visited []ast.Node
	v := databaseRefVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		if n == original {
			return next, false
		}
		visited = append(visited, n)
		return n, true
	}}
	got, ok := original.Accept(v)
	want := []ast.Node{next.Table, next.Pattern, where, next.Limit, next.ShowProfileLimit}
	if !ok || got != next || !reflect.DeepEqual(visited, want) {
		t.Fatalf("replacement=%T ok=%v visited=%v, want %v", got, ok, visited, want)
	}
}

func TestShowVisitorRejectsInvalidChildren(t *testing.T) {
	for _, replacement := range []ast.Node{nil, (*ast.Limit)(nil), &ast.UseStmt{}} {
		for _, slot := range []string{"table", "pattern", "where", "limit", "profile limit"} {
			show := &ast.ShowStmt{}
			var child ast.Node
			switch slot {
			case "table":
				show.Table = &ast.TableName{}
				child = show.Table
			case "pattern":
				show.Pattern = &ast.PatternLikeExpr{}
				child = show.Pattern
			case "where":
				show.Where = pureLiteral("1")
				child = show.Where
			case "limit":
				show.Limit = &ast.Limit{}
				child = show.Limit
			case "profile limit":
				show.ShowProfileLimit = &ast.Limit{}
				child = show.ShowProfileLimit
			}
			v := databaseRefVisitor{
				enter: func(n ast.Node) (ast.Node, bool) { return n, n == child },
				leave: func(n ast.Node) (ast.Node, bool) {
					if n == child {
						return replacement, true
					}
					return n, true
				},
			}
			if _, ok := show.Accept(v); ok {
				t.Fatalf("%s accepted %T", slot, replacement)
			}
		}
	}
	show := &ast.ShowStmt{Where: (*ast.LiteralExpr)(nil)}
	if _, ok := show.Accept(databaseRefVisitor{}); ok {
		t.Fatal("accepted typed-nil WHERE")
	}
}

func TestExplainVisitorStatementRoleAndOrder(t *testing.T) {
	stmt := &databaseRefStmt{}
	into := &ast.SelectIntoStatement{}
	explain := &ast.ExplainStmt{Stmt: stmt, Into: into}
	replacement := &databaseRefStmt{}
	var visited []ast.Node
	v := databaseRefVisitor{
		enter: func(n ast.Node) (ast.Node, bool) {
			if n != explain {
				visited = append(visited, n)
				return n, true
			}
			return n, false
		},
		leave: func(n ast.Node) (ast.Node, bool) {
			if n == stmt {
				return replacement, true
			}
			return n, true
		},
	}
	if _, ok := explain.Accept(v); !ok || explain.Stmt != replacement {
		t.Fatal("EXPLAIN rejected a non-DML statement replacement")
	}
	if !reflect.DeepEqual(visited, []ast.Node{into, stmt}) {
		t.Fatalf("visit order=%v", visited)
	}
	for _, bad := range []ast.Node{nil, (*databaseRefStmt)(nil), pureLiteral("1")} {
		explain := &ast.ExplainStmt{Stmt: stmt}
		v := databaseRefVisitor{
			leave: func(n ast.Node) (ast.Node, bool) {
				if n == stmt {
					return bad, true
				}
				return n, true
			},
		}
		if _, ok := explain.Accept(v); ok {
			t.Fatalf("EXPLAIN accepted %T", bad)
		}
	}
	for _, stmt := range []ast.StmtNode{nil, (*databaseRefStmt)(nil)} {
		if _, ok := (&ast.ExplainStmt{Stmt: stmt}).Accept(databaseRefVisitor{}); ok {
			t.Fatal("EXPLAIN accepted missing statement")
		}
	}
}

func TestDatabaseReferenceRestoreChildErrors(t *testing.T) {
	sentinel := errors.New("database reference child")
	for _, stmt := range []ast.StmtNode{
		&ast.ExplainStmt{},
		&ast.ExplainStmt{Stmt: (*databaseRefStmt)(nil)},
		&ast.ExplainStmt{Stmt: &ast.ShowStmt{}},
		&ast.ShowStmt{Tp: ast.ShowCreateTable},
		&ast.ShowStmt{Tp: ast.ShowCreateUser},
		&ast.ShowStmt{Tp: ast.ShowGrants, Roles: []*auth.RoleIdentity{nil}},
		&ast.ShowStmt{Tp: ast.ShowTables, Where: (*ast.LiteralExpr)(nil)},
		&ast.ShowStmt{Tp: ast.ShowTables, Pattern: &ast.PatternLikeExpr{Pattern: (*ast.LiteralExpr)(nil)}},
	} {
		var out strings.Builder
		if err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil {
			t.Fatalf("%T restored invalid child", stmt)
		}
	}
	var out strings.Builder
	err := (&ast.ExplainStmt{Stmt: &databaseRefStmt{err: sentinel}}).Restore(
		format.NewRestoreCtx(format.DefaultRestoreFlags, &out),
	)
	if !errors.Is(err, sentinel) {
		t.Fatalf("EXPLAIN lost child error: %v", err)
	}
}

func TestExplainVisitorIntoBoundaryAndStop(t *testing.T) {
	for _, bad := range []ast.Node{nil, (*ast.SelectIntoStatement)(nil), &ast.UseStmt{}} {
		into := &ast.SelectIntoStatement{}
		explain := &ast.ExplainStmt{Into: into, Stmt: &databaseRefStmt{}}
		v := databaseRefVisitor{
			enter: func(n ast.Node) (ast.Node, bool) { return n, n == into },
			leave: func(n ast.Node) (ast.Node, bool) {
				if n == into {
					return bad, true
				}
				return n, true
			},
		}
		if _, ok := explain.Accept(v); ok || explain.Into != into {
			t.Fatalf("EXPLAIN accepted %T or changed Into on failure", bad)
		}
	}
	into := &ast.SelectIntoStatement{}
	explain := &ast.ExplainStmt{Into: into, Stmt: &databaseRefStmt{}}
	stmtVisited, rootLeft := false, false
	v := databaseRefVisitor{
		enter: func(n ast.Node) (ast.Node, bool) {
			stmtVisited = stmtVisited || n == explain.Stmt
			return n, n == into
		},
		leave: func(n ast.Node) (ast.Node, bool) {
			rootLeft = rootLeft || n == explain
			return n, n != into
		},
	}
	if _, ok := explain.Accept(v); ok || stmtVisited || rootLeft {
		t.Fatal("EXPLAIN continued after child stop")
	}
}

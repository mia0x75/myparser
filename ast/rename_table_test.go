package ast_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
)

func TestRenameTableVisitorBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name        string
		replacement ast.Node
	}{
		{name: "wrong role", replacement: &ast.SelectStmt{}},
		{name: "nil", replacement: nil},
		{name: "typed nil", replacement: (*ast.TableName)(nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, leave := range []bool{false, true} {
				n := &ast.RenameTableStmt{Renames: []*ast.TableRename{{Table: &ast.TableName{}, NewTable: &ast.TableName{}}}}
				enter := func(n ast.Node) (ast.Node, bool) {
					if _, ok := n.(*ast.TableName); ok && !leave {
						return tc.replacement, true
					}
					return n, false
				}
				leaveNode := func(n ast.Node) (ast.Node, bool) {
					if _, ok := n.(*ast.TableName); ok && leave {
						return tc.replacement, true
					}
					return n, true
				}
				if _, ok := n.Accept(joinVisitor{enter: enter, leave: leaveNode}); ok {
					t.Fatalf("accepted invalid table replacement, leave=%v", leave)
				}
			}
		})
	}
	for _, replacement := range []ast.Node{&ast.SelectStmt{}, nil, (*ast.TableRename)(nil)} {
		n := &ast.RenameTableStmt{Renames: []*ast.TableRename{{Table: &ast.TableName{}, NewTable: &ast.TableName{}}}}
		v := joinVisitor{
			enter: func(n ast.Node) (ast.Node, bool) { return n, false },
			leave: func(n ast.Node) (ast.Node, bool) {
				if _, ok := n.(*ast.TableRename); ok {
					return replacement, true
				}
				return n, true
			},
		}
		if _, ok := n.Accept(v); ok {
			t.Fatalf("accepted rename replacement %T", replacement)
		}
	}
}

func TestRenameTableRejectsMissingChildren(t *testing.T) {
	for _, n := range []ast.Node{
		(*ast.RenameTableStmt)(nil), &ast.RenameTableStmt{},
		&ast.RenameTableStmt{Renames: []*ast.TableRename{nil}},
		(*ast.TableRename)(nil), &ast.TableRename{},
		&ast.TableRename{Table: &ast.TableName{}},
		&ast.TableRename{NewTable: &ast.TableName{}},
	} {
		var out strings.Builder
		if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil {
			t.Errorf("%T accepted missing children", n)
		}
		v := joinVisitor{enter: func(n ast.Node) (ast.Node, bool) { return n, false }, leave: func(n ast.Node) (ast.Node, bool) { return n, true }}
		if _, ok := n.Accept(v); ok {
			t.Errorf("%T visited missing children", n)
		}
	}
}

func TestRenameTableRestorePreservesFirstError(t *testing.T) {
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, byShortWriter{})
	ctx.WritePlain("x")
	want := ctx.Err()
	if !errors.Is(want, io.ErrShortWrite) {
		t.Fatalf("writer error = %v", want)
	}
	if err := (&ast.RenameTableStmt{}).Restore(ctx); err != want {
		t.Fatalf("first error = %v", err)
	}
}

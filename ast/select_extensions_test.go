package ast_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	dformat "github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

// --- helpers ---

type selExtVisitor struct {
	enter func(ast.Node) (ast.Node, bool)
	leave func(ast.Node) (ast.Node, bool)
}

func (v selExtVisitor) Enter(n ast.Node) (ast.Node, bool) {
	if v.enter != nil {
		return v.enter(n)
	}
	return n, false
}

func (v selExtVisitor) Leave(n ast.Node) (ast.Node, bool) {
	if v.leave != nil {
		return v.leave(n)
	}
	return n, true
}

func qualifiedNameOf(name string) ast.QualifiedName {
	return ast.QualifiedName{Parts: []lex.Ident{{Name: name}}}
}

// selExtShortWriter makes every Restore fail with io.ErrShortWrite.
type selExtShortWriter struct{}

func (selExtShortWriter) Write(p []byte) (int, error) { return 0, nil }

// selExtFailExpr wraps a PositionExpr and makes Restore fail with cause.
type selExtFailExpr struct {
	*ast.PositionExpr
	cause error
}

func (n *selExtFailExpr) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	_, ok := v.Leave(entered)
	return n, ok
}

func (n *selExtFailExpr) Restore(ctx *dformat.RestoreCtx) error { return n.cause }

// --- SelectStmt.Accept: LockedTables ---

func TestSelectStmtAcceptWalksLockedTables(t *testing.T) {
	original := &ast.SelectStmt{}
	original.LockTp = ast.SelectLockForUpdate
	original.LockedTables = []*ast.TableName{
		{BaseTableRef: pkast.BaseTableRef{Name: qualifiedNameOf("t1")}},
		{BaseTableRef: pkast.BaseTableRef{Name: qualifiedNameOf("t2")}},
	}

	// A visitor that counts TableName visits must see both locked tables.
	var visited int
	_, ok := original.Accept(selExtVisitor{
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, isTable := n.(*ast.TableName); isTable {
				visited++
			}
			return n, true
		},
	})
	if !ok {
		t.Fatal("unchanged visitor rejected")
	}
	if visited != 2 {
		t.Fatalf("visited %d locked tables, want 2", visited)
	}

	// A visitor that replaces each locked table with a distinct valid
	// replacement must be applied in place.
	repl1 := &ast.TableName{BaseTableRef: pkast.BaseTableRef{Name: qualifiedNameOf("t1x")}}
	repl2 := &ast.TableName{BaseTableRef: pkast.BaseTableRef{Name: qualifiedNameOf("t2x")}}
	_, ok = original.Accept(selExtVisitor{
		leave: func(n ast.Node) (ast.Node, bool) {
			if table, isTable := n.(*ast.TableName); isTable {
				if table.Name.Parts[0].Name == "t1" {
					return repl1, true
				}
				return repl2, true
			}
			return n, true
		},
	})
	if !ok {
		t.Fatal("valid replacement rejected")
	}
	if original.LockedTables[0] != repl1 || original.LockedTables[1] != repl2 {
		t.Fatalf("LockedTables not updated: got [%T %T]", original.LockedTables[0], original.LockedTables[1])
	}
}

func TestSelectExpressionsRejectTypedNil(t *testing.T) {
	for _, slot := range []struct {
		name string
		root func(ast.ExprNode) ast.Node
	}{
		{name: "where", root: func(expr ast.ExprNode) ast.Node {
			return &ast.SelectStmt{BaseSelectStmt: pkast.BaseSelectStmt{Where: expr}}
		}},
		{name: "having", root: func(expr ast.ExprNode) ast.Node {
			return &ast.SelectStmt{BaseSelectStmt: pkast.BaseSelectStmt{Having: expr}}
		}},
		{name: "analyse arg", root: func(expr ast.ExprNode) ast.Node { return &ast.ProcedureAnalyseClause{Args: []ast.ExprNode{expr}} }},
	} {
		t.Run(slot.name, func(t *testing.T) {
			var visited bool
			root := slot.root((*selExtFailExpr)(nil))
			_, ok := root.Accept(selExtVisitor{enter: func(n ast.Node) (ast.Node, bool) {
				if _, expr := n.(ast.ExprNode); expr {
					visited = true
				}
				return n, false
			}})
			if ok || visited {
				t.Fatalf("typed-nil input accepted=%v, visited=%v", ok, visited)
			}
			root = slot.root(&ast.PositionExpr{N: 1})
			_, ok = root.Accept(selExtVisitor{leave: func(n ast.Node) (ast.Node, bool) {
				if _, expr := n.(ast.ExprNode); expr {
					return (*ast.LiteralExpr)(nil), true
				}
				return n, true
			}})
			if ok {
				t.Fatal("typed-nil expression replacement accepted")
			}
		})
	}
}

func TestSelectExtensionEnterRejectsInvalidRoot(t *testing.T) {
	for _, root := range []ast.Node{&ast.ProcedureAnalyseClause{}, &ast.SelectIntoStatement{}} {
		for _, skip := range []bool{false, true} {
			_, ok := root.Accept(selExtVisitor{enter: func(ast.Node) (ast.Node, bool) { return &ast.TableName{}, skip }})
			if ok {
				t.Fatalf("%T accepted wrong-role Enter, skip=%v", root, skip)
			}
		}
	}
}

func TestSelectStmtAcceptRejectsLockedTableWrongRole(t *testing.T) {
	original := &ast.SelectStmt{}
	original.LockTp = ast.SelectLockForUpdate
	original.LockedTables = []*ast.TableName{
		{BaseTableRef: pkast.BaseTableRef{Name: qualifiedNameOf("t1")}},
	}

	// Replace with a node that is not a *TableName. The walk must reject.
	_, ok := original.Accept(selExtVisitor{
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, isTable := n.(*ast.TableName); isTable {
				return &ast.Limit{}, true
			}
			return n, true
		},
	})
	if ok {
		t.Fatal("wrong-role locked-table replacement accepted")
	}
}

func TestSelectStmtAcceptRejectsLockedTableTypedNil(t *testing.T) {
	original := &ast.SelectStmt{}
	original.LockTp = ast.SelectLockForUpdate
	original.LockedTables = []*ast.TableName{
		{BaseTableRef: pkast.BaseTableRef{Name: qualifiedNameOf("t1")}},
	}

	_, ok := original.Accept(selExtVisitor{
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, isTable := n.(*ast.TableName); isTable {
				return (*ast.TableName)(nil), true
			}
			return n, true
		},
	})
	if ok {
		t.Fatal("typed-nil locked-table replacement accepted")
	}
}

func TestSelectStmtAcceptNilLockedTableEntry(t *testing.T) {
	original := &ast.SelectStmt{}
	original.LockTp = ast.SelectLockForUpdate
	original.LockedTables = []*ast.TableName{nil}

	_, ok := original.Accept(selExtVisitor{})
	if ok {
		t.Fatal("nil locked-table entry accepted")
	}
}

// --- SelectStmt.Accept: Into ---

func TestSelectStmtAcceptRejectsIntoWrongRole(t *testing.T) {
	original := &ast.SelectStmt{}
	original.Into = &ast.SelectIntoStatement{
		Tp:      ast.SelectIntoVarList,
		Targets: []*ast.VariableTarget{{Name: qualifiedNameOf("v")}},
	}

	_, ok := original.Accept(selExtVisitor{
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, isInto := n.(*ast.SelectIntoStatement); isInto {
				return &ast.Limit{}, true
			}
			return n, true
		},
	})
	if ok {
		t.Fatal("wrong-role INTO replacement accepted")
	}
}

func TestSelectStmtAcceptRejectsIntoTypedNil(t *testing.T) {
	original := &ast.SelectStmt{}
	original.Into = &ast.SelectIntoStatement{
		Tp:      ast.SelectIntoVarList,
		Targets: []*ast.VariableTarget{{Name: qualifiedNameOf("v")}},
	}

	_, ok := original.Accept(selExtVisitor{
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, isInto := n.(*ast.SelectIntoStatement); isInto {
				return (*ast.SelectIntoStatement)(nil), true
			}
			return n, true
		},
	})
	if ok {
		t.Fatal("typed-nil INTO replacement accepted")
	}
}

func TestSelectStmtAcceptAppliesIntoReplacement(t *testing.T) {
	original := &ast.SelectStmt{}
	original.Into = &ast.SelectIntoStatement{
		Tp:      ast.SelectIntoVarList,
		Targets: []*ast.VariableTarget{{Name: qualifiedNameOf("v")}},
	}
	replacement := &ast.SelectIntoStatement{
		Tp:      ast.SelectIntoVarList,
		Targets: []*ast.VariableTarget{{Name: qualifiedNameOf("v2")}},
	}

	_, ok := original.Accept(selExtVisitor{
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, isInto := n.(*ast.SelectIntoStatement); isInto {
				return replacement, true
			}
			return n, true
		},
	})
	if !ok {
		t.Fatal("valid INTO replacement rejected")
	}
	if original.Into != replacement {
		t.Fatal("INTO slot was not updated to the replacement")
	}
}

// --- SelectStmt.Accept: ProcedureAnalyse ---

func TestSelectStmtAcceptRejectsProcedureAnalyseWrongRole(t *testing.T) {
	original := &ast.SelectStmt{}
	original.ProcedureAnalyse = &ast.ProcedureAnalyseClause{}

	_, ok := original.Accept(selExtVisitor{
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, isClause := n.(*ast.ProcedureAnalyseClause); isClause {
				return &ast.Limit{}, true
			}
			return n, true
		},
	})
	if ok {
		t.Fatal("wrong-role PROCEDURE ANALYSE replacement accepted")
	}
}

func TestSelectStmtAcceptRejectsProcedureAnalyseTypedNil(t *testing.T) {
	original := &ast.SelectStmt{}
	original.ProcedureAnalyse = &ast.ProcedureAnalyseClause{}

	_, ok := original.Accept(selExtVisitor{
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, isClause := n.(*ast.ProcedureAnalyseClause); isClause {
				return (*ast.ProcedureAnalyseClause)(nil), true
			}
			return n, true
		},
	})
	if ok {
		t.Fatal("typed-nil PROCEDURE ANALYSE replacement accepted")
	}
}

func TestSelectStmtAcceptAppliesProcedureAnalyseReplacement(t *testing.T) {
	original := &ast.SelectStmt{}
	original.ProcedureAnalyse = &ast.ProcedureAnalyseClause{}
	replacement := &ast.ProcedureAnalyseClause{}

	_, ok := original.Accept(selExtVisitor{
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, isClause := n.(*ast.ProcedureAnalyseClause); isClause {
				return replacement, true
			}
			return n, true
		},
	})
	if !ok {
		t.Fatal("valid PROCEDURE ANALYSE replacement rejected")
	}
	if original.ProcedureAnalyse != replacement {
		t.Fatal("PROCEDURE ANALYSE slot was not updated to the replacement")
	}
}

// --- ProcedureAnalyseClause.Accept: Args ---

func TestProcedureAnalyseAcceptRejectsArgWrongRole(t *testing.T) {
	original := &ast.ProcedureAnalyseClause{
		Args: []ast.ExprNode{&ast.PositionExpr{N: 1}},
	}

	_, ok := original.Accept(selExtVisitor{
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, isExpr := n.(ast.ExprNode); isExpr {
				return &ast.TableName{}, true
			}
			return n, true
		},
	})
	if ok {
		t.Fatal("wrong-role arg replacement accepted")
	}
}

func TestProcedureAnalyseAcceptAppliesArgReplacement(t *testing.T) {
	original := &ast.ProcedureAnalyseClause{
		Args: []ast.ExprNode{&ast.PositionExpr{N: 1}},
	}
	replacement := &ast.PositionExpr{N: 2}

	_, ok := original.Accept(selExtVisitor{
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, isExpr := n.(ast.ExprNode); isExpr {
				return replacement, true
			}
			return n, true
		},
	})
	if !ok {
		t.Fatal("valid arg replacement rejected")
	}
	if original.Args[0] != replacement {
		t.Fatal("arg slot was not updated to the replacement")
	}
}

// --- SelectIntoStatement.Accept: Targets ---

func TestSelectIntoAcceptRejectsTargetWrongRole(t *testing.T) {
	original := &ast.SelectIntoStatement{
		Tp:      ast.SelectIntoVarList,
		Targets: []*ast.VariableTarget{{Name: qualifiedNameOf("v")}},
	}

	_, ok := original.Accept(selExtVisitor{
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, isTarget := n.(*ast.VariableTarget); isTarget {
				return &ast.TableName{}, true
			}
			return n, true
		},
	})
	if ok {
		t.Fatal("wrong-role target replacement accepted")
	}
}

func TestSelectIntoAcceptAppliesTargetReplacement(t *testing.T) {
	original := &ast.SelectIntoStatement{
		Tp:      ast.SelectIntoVarList,
		Targets: []*ast.VariableTarget{{Name: qualifiedNameOf("v")}},
	}
	replacement := &ast.VariableTarget{Name: qualifiedNameOf("v2")}

	_, ok := original.Accept(selExtVisitor{
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, isTarget := n.(*ast.VariableTarget); isTarget {
				return replacement, true
			}
			return n, true
		},
	})
	if !ok {
		t.Fatal("valid target replacement rejected")
	}
	if original.Targets[0] != replacement {
		t.Fatal("target slot was not updated to the replacement")
	}
}

// --- Restore: first-error and short-write ---

func TestProcedureAnalyseRestoreFirstError(t *testing.T) {
	cause := errors.New("procedure analyse arg restore failed")
	original := &ast.ProcedureAnalyseClause{
		Args: []ast.ExprNode{&selExtFailExpr{
			PositionExpr: &ast.PositionExpr{N: 1},
			cause:        cause,
		}},
	}
	var out strings.Builder
	err := original.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out))
	if !errors.Is(err, cause) {
		t.Fatalf("Restore lost the arg error: got %v, want %v", err, cause)
	}
}

func TestProcedureAnalyseRestoreShortWrite(t *testing.T) {
	original := &ast.ProcedureAnalyseClause{
		Args: []ast.ExprNode{&ast.PositionExpr{N: 1}},
	}
	err := original.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, selExtShortWriter{}))
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("Restore: expected io.ErrShortWrite, got %v", err)
	}
}

func TestSelectStmtRestoreShortWrite(t *testing.T) {
	original := &ast.SelectStmt{}
	original.Fields = &ast.FieldList{Items: []*pkast.SelectItem{{Expr: &ast.WildcardExpr{}}}}
	original.LockTp = ast.SelectLockForUpdate
	original.LockedTables = []*ast.TableName{
		{BaseTableRef: pkast.BaseTableRef{Name: qualifiedNameOf("t1")}},
	}
	err := original.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, selExtShortWriter{}))
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("Restore: expected io.ErrShortWrite, got %v", err)
	}
}

// --- SelectStmt.Accept: skip-children must not walk LockedTables ---

func TestSelectStmtAcceptSkipDoesNotWalkLockedTables(t *testing.T) {
	original := &ast.SelectStmt{}
	original.LockTp = ast.SelectLockForUpdate
	original.LockedTables = []*ast.TableName{
		{BaseTableRef: pkast.BaseTableRef{Name: qualifiedNameOf("t1")}},
	}

	var visited int
	_, ok := original.Accept(selExtVisitor{
		enter: func(n ast.Node) (ast.Node, bool) {
			if _, isSelect := n.(*ast.SelectStmt); isSelect {
				return n, true // skip children
			}
			return n, false
		},
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, isTable := n.(*ast.TableName); isTable {
				visited++
			}
			return n, true
		},
	})
	if !ok {
		t.Fatal("skip-children visitor rejected")
	}
	if visited != 0 {
		t.Fatalf("skip-children visited %d locked tables, want 0", visited)
	}
}

package ast_test

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

// handlerSyntaxVisitor records entries and optional stop/replacement hooks.
type handlerSyntaxVisitor struct {
	entered []ast.Node
	enter   func(n ast.Node) (ast.Node, bool)
	leave   func(n ast.Node) (ast.Node, bool)
}

func (v *handlerSyntaxVisitor) Enter(n ast.Node) (ast.Node, bool) {
	v.entered = append(v.entered, n)
	if v.enter != nil {
		return v.enter(n)
	}
	return n, false
}

func (v *handlerSyntaxVisitor) Leave(n ast.Node) (ast.Node, bool) {
	if v.leave != nil {
		return v.leave(n)
	}
	return n, true
}

func mustRestoreHandlerSyntax(t *testing.T, n ast.Node) string {
	t.Helper()
	var sb strings.Builder
	if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
		t.Fatalf("restore %T: %v", n, err)
	}
	return sb.String()
}

// handlerRestoreErr expects Restore to fail.
func handlerRestoreErr(t *testing.T, n ast.Node) {
	t.Helper()
	var sb strings.Builder
	if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err == nil {
		t.Fatalf("restore %T: expected error, got %q", n, sb.String())
	}
}

func TestHandlerSyntaxOpenNoAlias(t *testing.T) {
	t.Parallel()
	stmt := &ast.HandlerStmt{
		Tp:    ast.HandlerOpen,
		Table: &ast.TableName{},
	}
	if got := mustRestoreHandlerSyntax(t, stmt); !strings.Contains(got, "OPEN") || strings.Contains(got, "AS") {
		t.Fatalf("restore = %q, want OPEN without AS", got)
	}
}

func TestHandlerSyntaxBarePrimaryStillRejected(t *testing.T) {
	// The baseline probe already shows bare PRIMARY is rejected by the parser
	// while quoted `PRIMARY` is accepted. This test documents the AST side:
	// a HandlerStmt with an index named PRIMARY must restore it verbatim
	// through WriteIdent (quoted, no special-casing of the keyword value).
	t.Parallel()
	stmt := &ast.HandlerStmt{
		Tp:        ast.HandlerRead,
		Table:     &ast.TableName{},
		Index:     lex.Ident{Name: "PRIMARY", Raw: "`PRIMARY`", Quote: lex.QuoteBacktick},
		Op:        pkast.OperatorUnknown,
		Direction: ast.HandlerReadFirst,
	}
	if got := mustRestoreHandlerSyntax(t, stmt); !strings.Contains(got, "`PRIMARY`") {
		t.Fatalf("restore = %q, want quoted PRIMARY index", got)
	}
}

func TestHandlerSyntaxQuotedPrimaryIndex(t *testing.T) {
	t.Parallel()
	stmt := &ast.HandlerStmt{
		Tp:     ast.HandlerRead,
		Table:  &ast.TableName{},
		Index:  lex.Ident{Name: "PRIMARY", Raw: "`PRIMARY`", Quote: lex.QuoteBacktick},
		Values: []ast.ExprNode{ast.NewParamMarkerExpr(0)},
		Op:     pkast.OperatorEQ,
	}
	if got := mustRestoreHandlerSyntax(t, stmt); !strings.Contains(got, "`PRIMARY` = (?)") {
		t.Fatalf("restore = %q, want quoted PRIMARY comparison", got)
	}
}

func TestHandlerSyntaxAllComparisonOps(t *testing.T) {
	t.Parallel()
	val := ast.NewParamMarkerExpr(0)
	for op, want := range map[pkast.Operator]string{
		pkast.OperatorEQ: " = (",
		pkast.OperatorGE: " >= (",
		pkast.OperatorLE: " <= (",
		pkast.OperatorGT: " > (",
		pkast.OperatorLT: " < (",
	} {
		stmt := &ast.HandlerStmt{
			Tp:     ast.HandlerRead,
			Table:  &ast.TableName{},
			Index:  lex.Ident{Name: "ix"},
			Op:     op,
			Values: []ast.ExprNode{val},
		}
		if got := mustRestoreHandlerSyntax(t, stmt); !strings.Contains(got, want) {
			t.Fatalf("%v: restore = %q, want %q", op, got, want)
		}
	}
	// An out-of-range operator must be rejected, not written as SQL.
	bad := &ast.HandlerStmt{
		Tp:     ast.HandlerRead,
		Table:  &ast.TableName{},
		Index:  lex.Ident{Name: "ix"},
		Op:     pkast.OperatorNE, // not part of the five HANDLER operators
		Values: []ast.ExprNode{val},
	}
	handlerRestoreErr(t, bad)
}

func TestHandlerSyntaxDirectionWithoutIndex(t *testing.T) {
	t.Parallel()
	for _, direction := range []ast.HandlerReadDirection{
		ast.HandlerReadFirst,
		ast.HandlerReadNext,
		ast.HandlerReadPrev,
		ast.HandlerReadLast,
	} {
		stmt := &ast.HandlerStmt{
			Tp:        ast.HandlerRead,
			Table:     &ast.TableName{},
			Direction: direction,
		}
		// Without an index, only FIRST/NEXT are valid on the primary key;
		// PREV and LAST must be rejected.
		if direction == ast.HandlerReadPrev || direction == ast.HandlerReadLast {
			handlerRestoreErr(t, stmt)
			continue
		}
		_ = mustRestoreHandlerSyntax(t, stmt)
	}
}

func TestHandlerSyntaxVisitorOrder(t *testing.T) {
	t.Parallel()
	stmt := &ast.HandlerStmt{
		Tp:     ast.HandlerRead,
		Table:  &ast.TableName{},
		Index:  lex.Ident{Name: "ix"},
		Op:     pkast.OperatorEQ,
		Values: []ast.ExprNode{ast.NewParamMarkerExpr(0)},
		Where:  ast.NewParamMarkerExpr(1),
		Limit:  &ast.Limit{BaseLimit: pkast.BaseLimit{Count: ast.NewParamMarkerExpr(2)}},
	}
	v := &handlerSyntaxVisitor{}
	if _, ok := stmt.Accept(v); !ok {
		t.Fatal("Accept returned false for a complete statement")
	}
	// Table is visited before the expression children.
	if len(v.entered) < 2 || v.entered[0] != stmt || v.entered[1] != stmt.Table {
		t.Fatalf("visit order = %T sequence, want stmt then Table", v.entered)
	}
}

func TestHandlerSyntaxVisitorWrongRoleReplacement(t *testing.T) {
	t.Parallel()
	stmt := &ast.HandlerStmt{
		Tp:     ast.HandlerRead,
		Table:  &ast.TableName{},
		Values: []ast.ExprNode{ast.NewParamMarkerExpr(0)},
		Where:  ast.NewParamMarkerExpr(1),
	}
	// A concrete Limit slot must reject a different node role.
	stmt.Limit = &ast.Limit{BaseLimit: pkast.BaseLimit{Count: ast.NewParamMarkerExpr(0)}}
	_, ok := stmt.Accept(&handlerSyntaxVisitor{
		enter: func(n ast.Node) (ast.Node, bool) {
			if _, isLimit := n.(*ast.Limit); isLimit {
				return ast.NewParamMarkerExpr(1), false
			}
			return n, false
		},
	})
	if ok {
		t.Fatal("Accept returned ok=true after wrong-role Limit replacement, want false")
	}
}

func TestHandlerSyntaxVisitorSkipLeavesStmtUnchanged(t *testing.T) {
	t.Parallel()
	stmt := &ast.HandlerStmt{
		Tp:    ast.HandlerOpen,
		Table: &ast.TableName{},
	}
	v := &handlerSyntaxVisitor{
		enter: func(n ast.Node) (ast.Node, bool) {
			if n != stmt {
				t.Fatal("visited a child despite skip")
			}
			return stmt, true
		},
	}
	got, ok := stmt.Accept(v)
	if !ok || got != stmt {
		t.Fatalf("skip Accept = %v, %v; want identity, true", got, ok)
	}
}

func TestHandlerSyntaxSpanCoversStatement(t *testing.T) {
	t.Parallel()
	stmt := &ast.HandlerStmt{
		Tp:    ast.HandlerOpen,
		Table: &ast.TableName{},
	}
	span := loc.Span{Start: 10, End: 30}
	stmt.SetSpan(span)
	if stmt.Span() != span {
		t.Fatalf("stmt span = %v, want %v", stmt.Span(), span)
	}
	// The index ident span must not leak into the statement span.
	indexSpan := loc.Span{Start: 1, End: 5}
	stmt.Index = lex.Ident{Name: "ix", Span: indexSpan}
	if stmt.Span() != span {
		t.Fatalf("setting index span changed stmt span: %v", stmt.Span())
	}
}

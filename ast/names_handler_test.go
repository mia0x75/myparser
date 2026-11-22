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

// namesHandlerVisitor records entries and optional replacements/stops.
type namesHandlerVisitor struct {
	enter func(n ast.Node) (ast.Node, bool)
	leave func(n ast.Node) (ast.Node, bool)
}

func (v *namesHandlerVisitor) Enter(n ast.Node) (ast.Node, bool) {
	if v.enter != nil {
		return v.enter(n)
	}
	return n, false
}

func (v *namesHandlerVisitor) Leave(n ast.Node) (ast.Node, bool) {
	if v.leave != nil {
		return v.leave(n)
	}
	return n, true
}

func mustRestoreNamesHandler(t *testing.T, n ast.Node) string {
	t.Helper()
	var sb strings.Builder
	if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
		t.Fatalf("restore %T: %v", n, err)
	}
	return sb.String()
}

// namesHandlerRestoreErr restores n, expecting an error.
func namesHandlerRestoreErr(t *testing.T, n ast.Node) error {
	t.Helper()
	var sb strings.Builder
	err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
	if err == nil {
		t.Fatalf("restore %T: expected error, got %q", n, sb.String())
	}
	return err
}

func TestNamesHandlerOpenAlias(t *testing.T) {
	t.Parallel()
	// OPEN with alias restores cleanly.
	alias := &ast.HandlerStmt{
		Tp:        ast.HandlerOpen,
		Table:     &ast.TableName{},
		Alias:     lex.Ident{Name: "al", Quote: lex.QuoteUnquoted},
		Op:        pkast.OperatorUnknown,
		Direction: ast.HandlerReadUnknown,
	}
	if got := mustRestoreNamesHandler(t, alias); !strings.Contains(got, "OPEN AS `al`") {
		t.Fatalf("restore = %q, want OPEN AS `al`", got)
	}
	// OPEN with a present index must be rejected.
	bad := &ast.HandlerStmt{
		Tp:    ast.HandlerOpen,
		Table: &ast.TableName{},
		Index: lex.Ident{Name: "ix"},
	}
	_ = namesHandlerRestoreErr(t, bad)
}

func TestNamesHandlerClose(t *testing.T) {
	t.Parallel()
	// CLOSE with no alias is valid.
	valid := &ast.HandlerStmt{
		Tp:        ast.HandlerClose,
		Table:     &ast.TableName{},
		Op:        pkast.OperatorUnknown,
		Direction: ast.HandlerReadUnknown,
	}
	if got := mustRestoreNamesHandler(t, valid); !strings.Contains(got, "CLOSE") {
		t.Fatalf("restore = %q, want CLOSE", got)
	}
	// CLOSE with an alias must be rejected.
	bad := &ast.HandlerStmt{
		Tp:    ast.HandlerClose,
		Table: &ast.TableName{},
		Alias: lex.Ident{Name: "x"},
	}
	_ = namesHandlerRestoreErr(t, bad)
}

func TestNamesHandlerReadPrimaryKeyDirections(t *testing.T) {
	t.Parallel()
	// FIRST and NEXT are valid on the primary key (no index).
	for direction, want := range map[ast.HandlerReadDirection]string{
		ast.HandlerReadFirst: "FIRST",
		ast.HandlerReadNext:  "NEXT",
	} {
		stmt := &ast.HandlerStmt{
			Tp:        ast.HandlerRead,
			Table:     &ast.TableName{},
			Op:        pkast.OperatorUnknown,
			Direction: direction,
		}
		if got := mustRestoreNamesHandler(t, stmt); !strings.Contains(got, want) {
			t.Fatalf("%d: restore = %q, want %q", direction, got, want)
		}
	}
	// PREV and LAST are not valid on the primary key.
	for _, direction := range []ast.HandlerReadDirection{ast.HandlerReadPrev, ast.HandlerReadLast} {
		stmt := &ast.HandlerStmt{
			Tp:        ast.HandlerRead,
			Table:     &ast.TableName{},
			Op:        pkast.OperatorUnknown,
			Direction: direction,
		}
		_ = namesHandlerRestoreErr(t, stmt)
	}
}

func TestNamesHandlerReadIndexAllDirections(t *testing.T) {
	t.Parallel()
	for direction, want := range map[ast.HandlerReadDirection]string{
		ast.HandlerReadFirst: "FIRST",
		ast.HandlerReadNext:  "NEXT",
		ast.HandlerReadPrev:  "PREV",
		ast.HandlerReadLast:  "LAST",
	} {
		stmt := &ast.HandlerStmt{
			Tp:        ast.HandlerRead,
			Table:     &ast.TableName{},
			Index:     lex.Ident{Name: "ix"},
			Op:        pkast.OperatorUnknown,
			Direction: direction,
		}
		if got := mustRestoreNamesHandler(t, stmt); !strings.Contains(got, want) {
			t.Fatalf("index %d: restore = %q, want %q", direction, got, want)
		}
	}
}

func TestNamesHandlerReadComparisonOps(t *testing.T) {
	t.Parallel()
	val := ast.NewParamMarkerExpr(0)
	for op, want := range map[pkast.Operator]string{
		pkast.OperatorEQ: "=",
		pkast.OperatorGE: ">=",
		pkast.OperatorLE: "<=",
		pkast.OperatorGT: ">",
		pkast.OperatorLT: "<",
	} {
		stmt := &ast.HandlerStmt{
			Tp:        ast.HandlerRead,
			Table:     &ast.TableName{},
			Index:     lex.Ident{Name: "ix"},
			Op:        op,
			Values:    []ast.ExprNode{val},
			Direction: ast.HandlerReadUnknown,
		}
		if got := mustRestoreNamesHandler(t, stmt); !strings.Contains(got, want) {
			t.Fatalf("%v: restore = %q, want %q", op, got, want)
		}
	}
	// Comparison without Values must be rejected.
	badNoValues := &ast.HandlerStmt{
		Tp:        ast.HandlerRead,
		Table:     &ast.TableName{},
		Index:     lex.Ident{Name: "ix"},
		Op:        pkast.OperatorEQ,
		Direction: ast.HandlerReadUnknown,
	}
	_ = namesHandlerRestoreErr(t, badNoValues)
	// Comparison without an index must be rejected.
	badNoIndex := &ast.HandlerStmt{
		Tp:     ast.HandlerRead,
		Table:  &ast.TableName{},
		Op:     pkast.OperatorEQ,
		Values: []ast.ExprNode{val},
	}
	_ = namesHandlerRestoreErr(t, badNoIndex)
	// Comparison with a direction must be rejected.
	badDir := &ast.HandlerStmt{
		Tp:        ast.HandlerRead,
		Table:     &ast.TableName{},
		Index:     lex.Ident{Name: "ix"},
		Op:        pkast.OperatorEQ,
		Values:    []ast.ExprNode{val},
		Direction: ast.HandlerReadFirst,
	}
	_ = namesHandlerRestoreErr(t, badDir)
}

func TestNamesHandlerVisitorNilTable(t *testing.T) {
	t.Parallel()
	stmt := &ast.HandlerStmt{
		Tp:    ast.HandlerOpen,
		Table: nil,
	}
	_, ok := stmt.Accept(&namesHandlerVisitor{})
	if ok {
		t.Fatal("Accept returned ok=true for nil Table, want false")
	}
}

func TestNamesHandlerVisitorWrongTypeReplacement(t *testing.T) {
	t.Parallel()
	// A concrete Limit slot must reject a different node role.
	stmt := &ast.HandlerStmt{
		Tp:        ast.HandlerRead,
		Table:     &ast.TableName{},
		Index:     lex.Ident{Name: "ix"},
		Op:        pkast.OperatorUnknown,
		Direction: ast.HandlerReadFirst,
		Limit:     &ast.Limit{BaseLimit: pkast.BaseLimit{Count: ast.NewParamMarkerExpr(0)}},
	}
	_, ok := stmt.Accept(&namesHandlerVisitor{
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

func TestNamesHandlerVisitorSkip(t *testing.T) {
	t.Parallel()
	stmt := &ast.HandlerStmt{
		Tp:    ast.HandlerOpen,
		Table: &ast.TableName{},
	}
	var entered int
	v := &namesHandlerVisitor{
		enter: func(n ast.Node) (ast.Node, bool) {
			entered++
			return n, true // skip children
		},
	}
	got, ok := stmt.Accept(v)
	if !ok || got != stmt {
		t.Fatalf("skip: Accept = %v, %v; want identity, true", got, ok)
	}
	if entered != 1 {
		t.Fatalf("skip: entered %d nodes, want 1", entered)
	}
}

func TestNamesHandlerVisitorStop(t *testing.T) {
	t.Parallel()
	stmt := &ast.HandlerStmt{
		Tp:    ast.HandlerOpen,
		Table: &ast.TableName{},
	}
	v := &namesHandlerVisitor{
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, isTable := n.(*ast.TableName); isTable {
				return n, false
			}
			return n, true
		},
	}
	_, ok := stmt.Accept(v)
	if ok {
		t.Fatal("Accept returned ok=true after stop at Table, want false")
	}
}

func TestNamesHandlerVisitorNilValuesEntry(t *testing.T) {
	t.Parallel()
	stmt := &ast.HandlerStmt{
		Tp:     ast.HandlerRead,
		Table:  &ast.TableName{},
		Index:  lex.Ident{Name: "ix"},
		Op:     pkast.OperatorEQ,
		Values: []ast.ExprNode{ast.NewParamMarkerExpr(0), nil},
	}
	_, ok := stmt.Accept(&namesHandlerVisitor{})
	if ok {
		t.Fatal("Accept returned ok=true for nil Values entry, want false")
	}
}

func TestNamesHandlerQuotedEmptyAliasPresence(t *testing.T) {
	t.Parallel()
	stmt := &ast.HandlerStmt{
		Tp:    ast.HandlerOpen,
		Table: &ast.TableName{},
		Alias: lex.Ident{Raw: "``", Quote: lex.QuoteBacktick},
	}
	if !stmt.Alias.Present() {
		t.Fatal("quoted-empty alias: Present() = false, want true")
	}
}

func TestNamesHandlerIndexSpanIndependent(t *testing.T) {
	t.Parallel()
	indexSpan := loc.Span{Start: 0, End: 5}
	stmt := &ast.HandlerStmt{
		Tp:        ast.HandlerRead,
		Table:     &ast.TableName{},
		Index:     lex.Ident{Name: "ix", Span: indexSpan},
		Op:        pkast.OperatorUnknown,
		Direction: ast.HandlerReadFirst,
	}
	stmt.SetSpan(loc.Span{Start: 100, End: 200})
	if stmt.Index.Span != indexSpan {
		t.Fatalf("setting stmt span changed index span: got %v, want %v", stmt.Index.Span, indexSpan)
	}
}

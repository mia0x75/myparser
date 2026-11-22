package ast_test

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

func mustRestoreNamesPlugin(t *testing.T, n ast.Node) string {
	t.Helper()
	var sb strings.Builder
	if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
		t.Fatalf("restore %T: %v", n, err)
	}
	return sb.String()
}

func namesPluginRestoreErr(t *testing.T, n ast.Node) {
	t.Helper()
	var sb strings.Builder
	if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err == nil {
		t.Fatalf("restore %T: expected error, got %q", n, sb.String())
	}
}

func TestNamesPluginInstallName(t *testing.T) {
	t.Parallel()
	// The plugin name is a lex.Ident, restored through WriteIdent so quoted
	// names keep their quoting.
	stmt := &ast.InstallPluginStmt{
		Name:   lex.Ident{Name: "my_plugin", Raw: "`my_plugin`", Quote: lex.QuoteBacktick},
		Soname: "libmy_plugin.so",
	}
	if got := mustRestoreNamesPlugin(t, stmt); !strings.Contains(got, "`my_plugin`") {
		t.Fatalf("restore = %q, want quoted plugin name", got)
	}
	// SONAME is a string payload, not a name: it is written verbatim.
	if got := mustRestoreNamesPlugin(t, stmt); !strings.Contains(got, "libmy_plugin.so") {
		t.Fatalf("restore = %q, want SONAME string", got)
	}
	// An absent plugin name must be rejected.
	bad := &ast.InstallPluginStmt{Name: lex.Ident{Span: loc.InvalidSpan()}, Soname: "lib.so"}
	namesPluginRestoreErr(t, bad)
}

func TestNamesPluginUninstallName(t *testing.T) {
	t.Parallel()
	stmt := &ast.UninstallPluginStmt{
		Name: lex.Ident{Name: "plg", Raw: "`plg`", Quote: lex.QuoteBacktick},
	}
	if got := mustRestoreNamesPlugin(t, stmt); !strings.Contains(got, "`plg`") {
		t.Fatalf("restore = %q, want quoted plugin name", got)
	}
	// An absent name must be rejected.
	bad := &ast.UninstallPluginStmt{Name: lex.Ident{Span: loc.InvalidSpan()}}
	namesPluginRestoreErr(t, bad)
}

func TestNamesPluginNameSpanIndependent(t *testing.T) {
	t.Parallel()
	// The statement span must not leak into the name ident span.
	nameSpan := loc.Span{Start: 1, End: 9}
	stmt := &ast.InstallPluginStmt{Name: lex.Ident{Name: "p", Span: nameSpan}}
	stmt.SetSpan(loc.Span{Start: 0, End: 40})
	if stmt.Name.Span != nameSpan {
		t.Fatalf("stmt span overwrote name span: %v", stmt.Name.Span)
	}
}

func TestNamesPluginAcceptRoles(t *testing.T) {
	t.Parallel()
	// Plugin names are not nodes: Accept must not emit visitor events for the
	// name value, only for the statement itself.
	install := &ast.InstallPluginStmt{Name: lex.Ident{Name: "p"}}
	v := &namesPluginCountingVisitor{}
	if _, ok := install.Accept(v); !ok {
		t.Fatal("install Accept returned false for a complete statement")
	}
	if len(v.entered) != 1 || v.entered[0] != install {
		t.Fatalf("install visitor entered %d nodes, want 1 (the stmt)", len(v.entered))
	}
	uninstall := &ast.UninstallPluginStmt{Name: lex.Ident{Name: "p"}}
	v = &namesPluginCountingVisitor{}
	if _, ok := uninstall.Accept(v); !ok {
		t.Fatal("uninstall Accept returned false for a complete statement")
	}
	if len(v.entered) != 1 || v.entered[0] != uninstall {
		t.Fatalf("uninstall visitor entered %d nodes, want 1 (the stmt)", len(v.entered))
	}
}

type namesPluginCountingVisitor struct {
	entered []ast.Node
}

func (v *namesPluginCountingVisitor) Enter(n ast.Node) (ast.Node, bool) {
	v.entered = append(v.entered, n)
	return n, false
}

func (v *namesPluginCountingVisitor) Leave(n ast.Node) (ast.Node, bool) {
	return n, true
}

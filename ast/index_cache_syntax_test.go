package ast_test

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

// cacheIndexSyntaxVisitor records entries and optional stops/replacements.
type cacheIndexSyntaxVisitor struct {
	entered []ast.Node
	enter   func(n ast.Node) (ast.Node, bool)
	leave   func(n ast.Node) (ast.Node, bool)
}

func (v *cacheIndexSyntaxVisitor) Enter(n ast.Node) (ast.Node, bool) {
	v.entered = append(v.entered, n)
	if v.enter != nil {
		return v.enter(n)
	}
	return n, false
}

func (v *cacheIndexSyntaxVisitor) Leave(n ast.Node) (ast.Node, bool) {
	if v.leave != nil {
		return v.leave(n)
	}
	return n, true
}

func mustRestoreCacheIndex(t *testing.T, n ast.Node) string {
	t.Helper()
	var sb strings.Builder
	if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
		t.Fatalf("restore %T: %v", n, err)
	}
	return sb.String()
}

// cacheIndexRestoreErr expects Restore to fail.
func cacheIndexRestoreErr(t *testing.T, n ast.Node) {
	t.Helper()
	var sb strings.Builder
	if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err == nil {
		t.Fatalf("restore %T: expected error, got %q", n, sb.String())
	}
}

func cacheIndexTable(names ...string) *ast.CacheIndexTable {
	idents := make([]lex.Ident, len(names))
	for i, name := range names {
		idents[i] = lex.Ident{Name: name}
	}
	return &ast.CacheIndexTable{Table: &ast.TableName{}, IndexNames: idents}
}

func TestIndexCacheSyntaxNilVsEmptyLists(t *testing.T) {
	t.Parallel()
	// nil IndexNames means INDEX/KEY omitted entirely.
	omitted := cacheIndexTable()
	omitted.IndexNames = nil
	if got := mustRestoreCacheIndex(t, omitted); strings.Contains(got, "INDEX") {
		t.Fatalf("nil list restored INDEX clause: %q", got)
	}
	// A non-nil empty list preserves an explicit INDEX ().
	empty := &ast.CacheIndexTable{Table: &ast.TableName{}, IndexNames: []lex.Ident{}}
	if got := mustRestoreCacheIndex(t, empty); !strings.Contains(got, "INDEX ()") {
		t.Fatalf("empty list restored %q, want INDEX ()", got)
	}
	// A named list restores the names inside INDEX (...).
	named := cacheIndexTable("a", "b")
	if got := mustRestoreCacheIndex(t, named); !strings.Contains(got, "INDEX (`a`, `b`)") {
		t.Fatalf("named list restored %q, want INDEX (`a`, `b`)", got)
	}
}

func TestIndexCacheSyntaxPrimaryAndQuotedNames(t *testing.T) {
	t.Parallel()
	// PRIMARY is a normal index name here (no keyword special-casing in the
	// name lists), and quoted names keep their quoting through WriteIdent.
	primary := &ast.CacheIndexTable{
		Table:      &ast.TableName{},
		IndexNames: []lex.Ident{{Name: "PRIMARY", Raw: "`PRIMARY`", Quote: lex.QuoteBacktick}},
	}
	if got := mustRestoreCacheIndex(t, primary); !strings.Contains(got, "`PRIMARY`") {
		t.Fatalf("primary index restored %q, want quoted PRIMARY", got)
	}
	quotedDot := &ast.CacheIndexTable{
		Table:      &ast.TableName{},
		IndexNames: []lex.Ident{{Name: "a.b", Raw: "`a.b`", Quote: lex.QuoteBacktick}},
	}
	if got := mustRestoreCacheIndex(t, quotedDot); !strings.Contains(got, "`a.b`") {
		t.Fatalf("quoted-dot index restored %q, want `a.b`", got)
	}
}

func TestIndexCacheSyntaxDefaultTarget(t *testing.T) {
	t.Parallel()
	stmt := &ast.CacheIndexStmt{
		Tables: []*ast.CacheIndexTable{cacheIndexTable()},
		Target: &ast.CacheIndexTarget{
			Kind: ast.CacheIndexDefaultTarget,
		},
	}
	if got := mustRestoreCacheIndex(t, stmt); !strings.Contains(got, "IN DEFAULT") {
		t.Fatalf("restore = %q, want IN DEFAULT", got)
	}
	// A DEFAULT target must not carry a name.
	bad := &ast.CacheIndexTarget{
		Kind: ast.CacheIndexDefaultTarget,
		Name: lex.Ident{Name: "c"},
	}
	cacheIndexRestoreErr(t, bad)
	// A quoted `default` is an Identifier target, not a DEFAULT target.
	quotedDefault := &ast.CacheIndexTarget{
		Kind: ast.CacheIndexIdentifierTarget,
		Name: lex.Ident{Name: "default", Raw: "`default`", Quote: lex.QuoteBacktick},
	}
	if got := mustRestoreCacheIndex(t, quotedDefault); !strings.Contains(got, "`default`") {
		t.Fatalf("quoted default target restored %q, want `default`", got)
	}
	// An identifier target without a present name must be rejected.
	missingName := &ast.CacheIndexTarget{Kind: ast.CacheIndexIdentifierTarget}
	cacheIndexRestoreErr(t, missingName)
	// An unknown kind must be rejected, not silently emitted as DEFAULT.
	unknown := &ast.CacheIndexTarget{Kind: ast.CacheIndexUnknownTarget}
	cacheIndexRestoreErr(t, unknown)
}

func TestIndexCacheSyntaxThreeTablesMixedLists(t *testing.T) {
	t.Parallel()
	// Three tables with omitted / empty / named index lists coexist in one
	// statement, each per-table.
	stmt := &ast.CacheIndexStmt{
		Tables: []*ast.CacheIndexTable{
			cacheIndexTable(), // omitted
			&ast.CacheIndexTable{Table: &ast.TableName{}, IndexNames: []lex.Ident{}}, // empty
			cacheIndexTable("a"), // named
		},
		Target: &ast.CacheIndexTarget{Kind: ast.CacheIndexIdentifierTarget, Name: lex.Ident{Name: "c"}},
	}
	if got := mustRestoreCacheIndex(t, stmt); !strings.Contains(got, "IN `c`") {
		t.Fatalf("restore = %q, want IN `c`", got)
	}
}

func TestIndexCacheSyntaxPerTablePartitionAll(t *testing.T) {
	t.Parallel()
	// PartitionAll is stored per table entry.
	partitionAll := &ast.CacheIndexTable{
		Table:        &ast.TableName{},
		IndexNames:   nil,
		PartitionAll: true,
	}
	if got := mustRestoreCacheIndex(t, partitionAll); !strings.Contains(got, "PARTITION (ALL)") {
		t.Fatalf("restore = %q, want PARTITION (ALL)", got)
	}
}

func TestIndexCacheSyntaxMissingTablesAndTarget(t *testing.T) {
	t.Parallel()
	// CACHE without any table entry is invalid.
	empty := &ast.CacheIndexStmt{Target: &ast.CacheIndexTarget{Kind: ast.CacheIndexDefaultTarget}}
	cacheIndexRestoreErr(t, empty)
	// A missing target is invalid.
	noTarget := &ast.CacheIndexStmt{Tables: []*ast.CacheIndexTable{cacheIndexTable()}}
	cacheIndexRestoreErr(t, noTarget)
	// A nil table entry inside the list is invalid.
	nilEntry := &ast.CacheIndexStmt{
		Tables: []*ast.CacheIndexTable{nil, cacheIndexTable()},
		Target: &ast.CacheIndexTarget{Kind: ast.CacheIndexDefaultTarget},
	}
	cacheIndexRestoreErr(t, nilEntry)
}

func TestIndexCacheSyntaxVisitorOrder(t *testing.T) {
	t.Parallel()
	table := cacheIndexTable("a")
	stmt := &ast.CacheIndexStmt{
		Tables: []*ast.CacheIndexTable{table},
		Target: &ast.CacheIndexTarget{Kind: ast.CacheIndexIdentifierTarget, Name: lex.Ident{Name: "c"}},
	}
	v := &cacheIndexSyntaxVisitor{}
	if _, ok := stmt.Accept(v); !ok {
		t.Fatal("Accept returned false for a complete statement")
	}
	// Order: stmt, table entry, entry table, then the target leaf.
	if len(v.entered) < 3 || v.entered[0] != stmt || v.entered[1] != table {
		t.Fatalf("visit order = %T sequence, want stmt then entry", v.entered)
	}
}

func TestIndexCacheSyntaxVisitorWrongRole(t *testing.T) {
	t.Parallel()
	table := cacheIndexTable("a")
	stmt := &ast.CacheIndexStmt{
		Tables: []*ast.CacheIndexTable{table},
		Target: &ast.CacheIndexTarget{Kind: ast.CacheIndexDefaultTarget},
	}
	// Replacing a table entry with a non-CacheIndexTable must be rejected by
	// the entry's own Accept, not by a parent-level type assertion.
	_, ok := stmt.Accept(&cacheIndexSyntaxVisitor{
		enter: func(n ast.Node) (ast.Node, bool) {
			if _, isEntry := n.(*ast.CacheIndexTable); isEntry {
				return ast.NewParamMarkerExpr(1), false
			}
			return n, false
		},
	})
	if ok {
		t.Fatal("Accept returned ok=true after wrong-role entry replacement, want false")
	}
	// A nil table entry must be rejected before any dispatch.
	badStmt := &ast.CacheIndexStmt{
		Tables: []*ast.CacheIndexTable{nil},
		Target: &ast.CacheIndexTarget{Kind: ast.CacheIndexDefaultTarget},
	}
	_, ok = badStmt.Accept(&cacheIndexSyntaxVisitor{})
	if ok {
		t.Fatal("Accept returned ok=true for nil entry, want false")
	}
	// A nil target must be rejected.
	noTarget := &ast.CacheIndexStmt{Tables: []*ast.CacheIndexTable{cacheIndexTable()}}
	_, ok = noTarget.Accept(&cacheIndexSyntaxVisitor{})
	if ok {
		t.Fatal("Accept returned ok=true for nil target, want false")
	}
}

func TestIndexCacheSyntaxVisitorStopAndSkip(t *testing.T) {
	t.Parallel()
	table := cacheIndexTable("a")
	stmt := &ast.CacheIndexStmt{
		Tables: []*ast.CacheIndexTable{table},
		Target: &ast.CacheIndexTarget{Kind: ast.CacheIndexDefaultTarget},
	}
	// Stop at the entry via Leave(ok=false): the visit must fail.
	_, ok := stmt.Accept(&cacheIndexSyntaxVisitor{
		leave: func(n ast.Node) (ast.Node, bool) {
			if _, isEntry := n.(*ast.CacheIndexTable); isEntry {
				return n, false
			}
			return n, true
		},
	})
	if ok {
		t.Fatal("Accept returned ok=true after stop at entry, want false")
	}
	// Skip at the statement level: no children are visited.
	var childVisited bool
	_, ok = stmt.Accept(&cacheIndexSyntaxVisitor{
		enter: func(n ast.Node) (ast.Node, bool) {
			if n != stmt {
				childVisited = true
			}
			return stmt, true
		},
	})
	if !ok || childVisited {
		t.Fatalf("skip Accept = %v, childVisited=%v; want identity, false", ok, childVisited)
	}
}

func TestIndexCacheSyntaxLoadIgnoreLeaves(t *testing.T) {
	t.Parallel()
	entry := &ast.LoadIndexTable{
		Table:        &ast.TableName{},
		IndexNames:   []lex.Ident{},
		IgnoreLeaves: true,
	}
	stmt := &ast.LoadIndexStmt{Tables: []*ast.LoadIndexTable{entry}}
	if got := mustRestoreCacheIndex(t, stmt); !strings.Contains(got, "INDEX () IGNORE LEAVES") {
		t.Fatalf("restore = %q, want INDEX () IGNORE LEAVES", got)
	}
	// LOAD with a nil table entry is rejected.
	bad := &ast.LoadIndexStmt{Tables: []*ast.LoadIndexTable{nil}}
	cacheIndexRestoreErr(t, bad)
	// LOAD without any entries is rejected.
	cacheIndexRestoreErr(t, &ast.LoadIndexStmt{})
}

func TestIndexCacheSyntaxEntrySpanIndependent(t *testing.T) {
	t.Parallel()
	entry := &ast.CacheIndexTable{
		Table:      &ast.TableName{},
		IndexNames: []lex.Ident{{Name: "a", Span: loc.Span{Start: 1, End: 2}}},
	}
	span := loc.Span{Start: 10, End: 40}
	entry.SetSpan(span)
	if entry.Span() != span {
		t.Fatalf("entry span = %v, want %v", entry.Span(), span)
	}
	// The ident span must not be affected by the entry span.
	if entry.IndexNames[0].Span != (loc.Span{Start: 1, End: 2}) {
		t.Fatalf("entry span leaked into index span: %v", entry.IndexNames[0].Span)
	}
	// The target span is independent of the statement span.
	target := &ast.CacheIndexTarget{
		Kind: ast.CacheIndexIdentifierTarget,
		Name: lex.Ident{Name: "c", Span: loc.Span{Start: 50, End: 51}},
	}
	target.SetSpan(loc.Span{Start: 50, End: 51})
	if target.Span() != target.Name.Span {
		t.Fatalf("target span = %v, want the name span", target.Span())
	}
}

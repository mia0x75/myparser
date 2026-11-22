package ast_test

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

func mustRestoreNamesCache(t *testing.T, n ast.Node) string {
	t.Helper()
	var sb strings.Builder
	if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
		t.Fatalf("restore %T: %v", n, err)
	}
	return sb.String()
}

func TestNamesIndexCacheIndexNameSyntax(t *testing.T) {
	t.Parallel()
	// Backtick-quoted names must round-trip through WriteIdent.
	for _, name := range []struct {
		name  string
		raw   string
		quote lex.QuoteStyle
	}{
		{name: "索`名", raw: "`索``名`", quote: lex.QuoteBacktick},
		{name: "a.b", raw: "`a.b`", quote: lex.QuoteBacktick},
		{name: "PRIMARY", raw: "`PRIMARY`", quote: lex.QuoteBacktick},
	} {
		entry := &ast.CacheIndexTable{
			Table:      &ast.TableName{},
			IndexNames: []lex.Ident{{Name: name.name, Raw: name.raw, Quote: name.quote}},
		}
		if got := mustRestoreNamesCache(t, entry); !strings.Contains(got, name.raw) {
			t.Fatalf("index name %q restored %q, want %q", name.name, got, name.raw)
		}
	}
}

func TestNamesIndexCacheNilVsEmptyPresence(t *testing.T) {
	t.Parallel()
	// nil means the INDEX/KEY clause was omitted entirely.
	omitted := &ast.LoadIndexTable{Table: &ast.TableName{}}
	if got := mustRestoreNamesCache(t, omitted); strings.Contains(got, "INDEX") {
		t.Fatalf("nil IndexNames restored INDEX: %q", got)
	}
	// A non-nil empty slice preserves an explicit INDEX ().
	empty := &ast.LoadIndexTable{Table: &ast.TableName{}, IndexNames: []lex.Ident{}}
	if got := mustRestoreNamesCache(t, empty); !strings.Contains(got, "INDEX ()") {
		t.Fatalf("empty IndexNames restored %q, want INDEX ()", got)
	}
}

func TestNamesIndexCacheTargetNameSyntax(t *testing.T) {
	t.Parallel()
	// Identifier target: a quoted name preserves its quoting.
	quoted := &ast.CacheIndexTarget{
		Kind: ast.CacheIndexIdentifierTarget,
		Name: lex.Ident{Name: "my_cache", Raw: "`my_cache`", Quote: lex.QuoteBacktick},
	}
	if got := mustRestoreNamesCache(t, quoted); !strings.Contains(got, "`my_cache`") {
		t.Fatalf("target restored %q, want `my_cache`", got)
	}
	// Default target emits the DEFAULT keyword, not a name.
	defaultTarget := &ast.CacheIndexTarget{Kind: ast.CacheIndexDefaultTarget}
	if got := mustRestoreNamesCache(t, defaultTarget); !strings.Contains(got, "DEFAULT") {
		t.Fatalf("default target restored %q, want DEFAULT", got)
	}
	// A quoted `default` is an Identifier target, not a DEFAULT target.
	quotedDefault := &ast.CacheIndexTarget{
		Kind: ast.CacheIndexIdentifierTarget,
		Name: lex.Ident{Name: "default", Raw: "`default`", Quote: lex.QuoteBacktick},
	}
	if got := mustRestoreNamesCache(t, quotedDefault); !strings.Contains(got, "`default`") {
		t.Fatalf("quoted default target restored %q, want `default`", got)
	}
}

func TestNamesIndexCachePartitionAllFlag(t *testing.T) {
	t.Parallel()
	// PartitionAll is per-table, stored on CacheIndexTable.
	stmt := &ast.CacheIndexStmt{
		Tables: []*ast.CacheIndexTable{
			{Table: &ast.TableName{}, PartitionAll: true},
			{Table: &ast.TableName{}, PartitionAll: false},
		},
		Target: &ast.CacheIndexTarget{Kind: ast.CacheIndexDefaultTarget},
	}
	got := mustRestoreNamesCache(t, stmt)
	if !strings.Contains(got, "PARTITION (ALL)") {
		t.Fatalf("restore = %q, want PARTITION (ALL)", got)
	}
}

func TestNamesIndexCacheIdentSpansAreTokenBound(t *testing.T) {
	t.Parallel()
	// Each index ident carries its own token span; the entry span covers the
	// whole entry. Neither should overwrite the other.
	indexSpan := loc.Span{Start: 3, End: 8}
	entrySpan := loc.Span{Start: 0, End: 40}
	entry := &ast.CacheIndexTable{
		Table:      &ast.TableName{},
		IndexNames: []lex.Ident{{Name: "ix", Span: indexSpan}},
	}
	entry.SetSpan(entrySpan)
	if entry.IndexNames[0].Span != indexSpan {
		t.Fatalf("entry span overwrote index span: %v", entry.IndexNames[0].Span)
	}
	if entry.Span() != entrySpan {
		t.Fatalf("entry span = %v, want %v", entry.Span(), entrySpan)
	}
}

package parser

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	fmtrestore "github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

// TestDatabaseIdentMigration guards the migration of the CREATE/ALTER/DROP
// database statement names from plain strings to lex.Ident: decoded case,
// original Raw spelling, quote style, token span, unnamed semantics, and the
// full statement span.
func TestDatabaseIdentMigration(t *testing.T) {
	const utf8SQL = "create database `分``区` default collate utf8mb4_unicode_ci"
	const unquotedSQL = "create database db1"
	const altSQL = "alter database db1 default character set utf8"
	const altDefaultSQL = "alter database default collate utf8mb4_general_ci"
	const dropSQL = "drop database db2"

	span := func(sql string, start, length int) loc.Span {
		return loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + length)}
	}

	checkIdent := func(t *testing.T, got, want lex.Ident, wantSpan loc.Span, what string) {
		t.Helper()
		if got.Name != want.Name || got.Raw != want.Raw || got.Quote != want.Quote || got.Span != wantSpan {
			t.Fatalf("%s = %+v, want name=%q raw=%q quote=%d span=%v", what, got, want.Name, want.Raw, want.Quote, wantSpan)
		}
	}

	// Quoted UTF-8 name: backtick token bytes [16, 26).
	c := mustParseOne(t, utf8SQL).(*ast.CreateDatabaseStmt)
	checkIdent(t, c.Name, lex.Ident{Name: "分`区", Raw: "`分``区`", Quote: lex.QuoteBacktick},
		span(utf8SQL, 16, 10), "CreateDatabaseStmt.Name")
	if c.Span() != span(utf8SQL, 0, len(utf8SQL)) {
		t.Fatalf("CreateDatabaseStmt span = %v, want %v", c.Span(), span(utf8SQL, 0, len(utf8SQL)))
	}

	// Unquoted name: token bytes [16, 19).
	cu := mustParseOne(t, unquotedSQL).(*ast.CreateDatabaseStmt)
	checkIdent(t, cu.Name, lex.Ident{Name: "db1", Raw: "db1", Quote: lex.QuoteUnquoted},
		span(unquotedSQL, 16, 3), "CreateDatabaseStmt.Name (unquoted)")

	// ALTER DATABASE with name: token bytes [15, 18).
	a := mustParseOne(t, altSQL).(*ast.AlterDatabaseStmt)
	if a.AlterDefaultDatabase {
		t.Fatalf("named ALTER DATABASE must not flag AlterDefaultDatabase")
	}
	checkIdent(t, a.Name, lex.Ident{Name: "db1", Raw: "db1", Quote: lex.QuoteUnquoted},
		span(altSQL, 15, 3), "AlterDatabaseStmt.Name")
	if a.Span() != span(altSQL, 0, len(altSQL)) {
		t.Fatalf("AlterDatabaseStmt span = %v, want %v", a.Span(), span(altSQL, 0, len(altSQL)))
	}

	// ALTER DATABASE without a name: the absent identifier keeps an invalid
	// span and no Raw/Quote metadata.
	ad := mustParseOne(t, altDefaultSQL).(*ast.AlterDatabaseStmt)
	if !ad.AlterDefaultDatabase {
		t.Fatalf("unnamed ALTER DATABASE must flag AlterDefaultDatabase")
	}
	if ad.Name != (lex.Ident{Span: loc.InvalidSpan()}) {
		t.Fatalf("unnamed ALTER DATABASE Name = %+v, want %+v", ad.Name, lex.Ident{Span: loc.InvalidSpan()})
	}
	if ad.Span() != span(altDefaultSQL, 0, len(altDefaultSQL)) {
		t.Fatalf("unnamed AlterDatabaseStmt span = %v, want %v", ad.Span(), span(altDefaultSQL, 0, len(altDefaultSQL)))
	}

	// DROP DATABASE with name: token bytes [14, 17).
	d := mustParseOne(t, dropSQL).(*ast.DropDatabaseStmt)
	checkIdent(t, d.Name, lex.Ident{Name: "db2", Raw: "db2", Quote: lex.QuoteUnquoted},
		span(dropSQL, 14, 3), "DropDatabaseStmt.Name")
	if d.Span() != span(dropSQL, 0, len(dropSQL)) {
		t.Fatalf("DropDatabaseStmt span = %v, want %v", d.Span(), span(dropSQL, 0, len(dropSQL)))
	}
}

// TestDatabaseIdentRestore pins that the lex.Ident-based names restore to the
// same SQL under the default (backtick) and unquoted flags, and that parsing
// the restored SQL again yields an equal statement after normalizing lexical
// metadata.
func TestDatabaseIdentRestore(t *testing.T) {
	cases := []struct {
		flags fmtrestore.RestoreFlags
		sql   string
		want  string
	}{
		{fmtrestore.DefaultRestoreFlags, "create database `MyDb`", "CREATE DATABASE `MyDb`"},
		{fmtrestore.DefaultRestoreFlags, "create database db", "CREATE DATABASE `db`"},
		{fmtrestore.DefaultRestoreFlags, "create database if not exists `My``Db`", "CREATE DATABASE IF NOT EXISTS `My``Db`"},
		{fmtrestore.RestoreNameBackQuotes, "alter database db1 default character set utf8", "ALTER DATABASE `db1` CHARACTER SET = utf8"},
		{fmtrestore.RestoreNameBackQuotes, "alter database default collate utf8mb4_general_ci", "ALTER DATABASE COLLATE = utf8mb4_general_ci"},
		{fmtrestore.DefaultRestoreFlags, "drop database db2", "DROP DATABASE `db2`"},
		{fmtrestore.DefaultRestoreFlags, "drop database if exists `Gone`", "DROP DATABASE IF EXISTS `Gone`"},
	}
	for _, tc := range cases {
		t.Run(strings.TrimSpace(tc.sql), func(t *testing.T) {
			p := New()
			stmt, err := p.ParseOneStmt(tc.sql)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			var sb strings.Builder
			if err := stmt.Restore(fmtrestore.NewRestoreCtx(tc.flags, &sb)); err != nil {
				t.Fatalf("restore: %v", err)
			}
			restored := sb.String()
			if restored != tc.want {
				t.Fatalf("restore = %q, want %q", restored, tc.want)
			}
			stmt2, err := p.ParseOneStmt(restored)
			if err != nil {
				t.Fatalf("re-parse %q: %v", restored, err)
			}
			CleanNodeText(stmt)
			CleanNodeText(stmt2)
			normalizeTestSyntaxMetadata(stmt)
			normalizeTestSyntaxMetadata(stmt2)
			if !reflect.DeepEqual(stmt, stmt2) {
				t.Fatalf("round trip mismatch: got %#v, want %#v", stmt, stmt2)
			}
		})
	}
}

// TestDatabaseIdentParserReuse checks that one Parser instance reused for
// further parses keeps each parsed AST independent: names and spans must not
// leak between calls.
func TestDatabaseIdentParserReuse(t *testing.T) {
	const first = "create database `First`"
	const second = "drop database `Second`"
	p := New()
	stmt1, err := p.ParseOneStmt(first)
	if err != nil {
		t.Fatalf("first parse: %v", err)
	}
	stmt2, err := p.ParseOneStmt(second)
	if err != nil {
		t.Fatalf("second parse: %v", err)
	}
	// Re-parse the first input; the retained AST must still hold its own span.
	want, got := stmt1.(*ast.CreateDatabaseStmt), mustParseOne(t, first).(*ast.CreateDatabaseStmt)
	if got.Name.Span != (loc.Span{Start: loc.ByteOffset(16), End: loc.ByteOffset(23)}) {
		t.Fatalf("re-parsed name span = %v, want [16,23)", got.Name.Span)
	}
	if want.Name.Span != got.Name.Span {
		t.Fatalf("reused parser first result span corrupted: %+v vs %+v", want.Name.Span, got.Name.Span)
	}
	// Raw must be an independent slice of the original input for each parse.
	if want.Name.Raw != "`First`" || got.Name.Raw != "`First`" {
		t.Fatalf("reused parser raws = %q, %q", want.Name.Raw, got.Name.Raw)
	}
	d2 := stmt2.(*ast.DropDatabaseStmt)
	// The second parse's name spans its own input, not the first.
	// "drop database `Second`" → backtick token bytes [14, 22).
	if d2.Name != (lex.Ident{Name: "Second", Raw: "`Second`", Span: loc.Span{Start: loc.ByteOffset(14), End: loc.ByteOffset(22)}, Quote: lex.QuoteBacktick}) {
		t.Fatalf("reused parser second result name = %+v", d2.Name)
	}
	if d2.Span() != (loc.Span{Start: loc.ByteOffset(0), End: loc.ByteOffset(len(second))}) {
		t.Fatalf("reused parser second result span = %v", d2.Span())
	}
}

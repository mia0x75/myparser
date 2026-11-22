package myparser

import (
	"errors"
	stdio "io"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/mia0x75/myparser/ast"
	fmtrestore "github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

type p58DbRefCase struct {
	name         string
	sql          string
	decoded      string
	raw          string
	quote        lex.QuoteStyle
	mode         mysql.SQLMode
	wantRestored string
}

// tokenSpanP58 locates the raw token in the input. Repeated tokens use the
// last occurrence; the raw is an input substring for every spelling.
func tokenSpanP58(sql, raw string) (loc.Span, int) {
	i := strings.LastIndex(sql, raw)
	return loc.Span{Start: loc.ByteOffset(i), End: loc.ByteOffset(i + len(raw))}, i
}

// checkDbRefP58 compares the ident and its token span against the
// input-derived expectation.
func checkDbRefP58(t *testing.T, sql string, got, want lex.Ident, what string) {
	t.Helper()
	want.Span, _ = tokenSpanP58(sql, want.Raw)
	if got != want {
		t.Fatalf("%s = %+v span %v, want %+v span %v", what, got, got.Span, want, want.Span)
	}
}

// dbRefOfP58 extracts the database-reference ident from a parsed statement.
func dbRefOfP58(t *testing.T, stmt ast.StmtNode) lex.Ident {
	t.Helper()
	switch s := stmt.(type) {
	case *ast.UseStmt:
		return s.Database
	case *ast.ShowStmt:
		return s.Database
	case *ast.ExplainStmt:
		return s.Schema
	default:
		t.Fatalf("unexpected statement %T", stmt)
	}
	return lex.Ident{}
}

func TestUseStmtDatabaseReference(t *testing.T) {
	cases := []p58DbRefCase{
		{"unquoted", "use db1", "db1", "db1", lex.QuoteUnquoted, 0, "USE `db1`"},
		{"escaped backtick", "use `My``Db`", "My`Db", "`My``Db`", lex.QuoteBacktick, 0, "USE `My``Db`"},
		{"quoted empty", "use ``", "", "``", lex.QuoteBacktick, 0, "USE ``"},
		{"quoted dot", "use `a.b`", "a.b", "`a.b`", lex.QuoteBacktick, 0, "USE `a.b`"},
		{"UTF-8", "use `分``区`", "分`区", "`分``区`", lex.QuoteBacktick, 0, "USE `分``区`"},
		{"ANSI double", `use "a""b"`, `a"b`, `"a""b"`, lex.QuoteDouble, mysql.ModeANSIQuotes, "USE `a\"b`"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			p.SetSQLMode(tc.mode)
			stmt, err := p.ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			checkDbRefP58(t, tc.sql, dbRefOfP58(t, stmt), lex.Ident{Name: tc.decoded, Raw: tc.raw, Quote: tc.quote}, "UseStmt.Database")
			// Final span covers the whole statement, including its token bytes.
			if got, err := stmt.Span().Slice(tc.sql); err != nil || got != tc.sql {
				t.Fatalf("UseStmt span selects %q (%v), want %q", got, err, tc.sql)
			}
			if got := restoreStmt(t, stmt); got != tc.wantRestored {
				t.Fatalf("restore = %q, want %q", got, tc.wantRestored)
			}
		})
	}
	// The frozen grammar requires a database name.
	if _, err := New().ParseOneStmt("use", "", ""); err == nil {
		t.Fatal("bare USE: expected parse error")
	}
}

func TestShowCreateDatabaseReference(t *testing.T) {
	// Required name: absent is a parse error, quoted-empty stays present.
	if _, err := New().ParseOneStmt("show create database", "", ""); err == nil {
		t.Fatal("SHOW CREATE DATABASE without name: expected parse error")
	}
	empty := dbRefOfP58(t, mustParseOne(t, "show create database ``"))
	checkDbRefP58(t, "show create database ``", empty, lex.Ident{Name: "", Raw: "``", Quote: lex.QuoteBacktick}, "SHOW CREATE DATABASE ``")

	stmt := mustParseOne(t, "show create database if not exists `分``区`")
	show := stmt.(*ast.ShowStmt)
	checkDbRefP58(t, "show create database if not exists `分``区`", show.Database,
		lex.Ident{Name: "分`区", Raw: "`分``区`", Quote: lex.QuoteBacktick}, "SHOW CREATE DATABASE")
	if got := restoreStmt(t, stmt); got != "SHOW CREATE DATABASE IF NOT EXISTS `分``区`" {
		t.Fatalf("restore = %q", got)
	}
}

// TestShowOptionalDatabaseReference covers the seven optional-DB SHOW
// families plus SHOW DATABASES (Database must stay absent there).
func TestShowOptionalDatabaseReference(t *testing.T) {
	p := New()
	for _, tc := range []struct {
		name, sql    string
		decoded, raw string
		quote        lex.QuoteStyle
	}{
		{"tables", "show tables in db1", "db1", "db1", lex.QuoteUnquoted},
		{"full tables", "show full tables from `t1`", "t1", "`t1`", lex.QuoteBacktick},
		{"open tables", "show open tables in db2", "db2", "db2", lex.QuoteUnquoted},
		{"table status", "show table status from `t``s`", "t`s", "`t``s`", lex.QuoteBacktick},
		{"columns", "show columns from t1 in db3", "db3", "db3", lex.QuoteUnquoted},
		{"fields", "show fields from t1 in db4", "db4", "db4", lex.QuoteUnquoted},
		{"triggers", "show triggers in db5", "db5", "db5", lex.QuoteUnquoted},
		{"events", "show events from `ev``db`", "ev`db", "`ev``db`", lex.QuoteBacktick},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmt, err := p.ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			checkDbRefP58(t, tc.sql, dbRefOfP58(t, stmt), lex.Ident{Name: tc.decoded, Raw: tc.raw, Quote: tc.quote}, "ShowStmt.Database")
			// Final span covers the whole statement, token bytes included.
			if got, err := stmt.Span().Slice(tc.sql); err != nil || got != tc.sql {
				t.Fatalf("span selects %q (%v), want %q", got, err, tc.sql)
			}
		})
	}

	// Absent variants.
	for _, tc := range []struct {
		name string
		sql  string
	}{
		{"tables", "show tables"},
		{"open tables", "show open tables"},
		{"table status", "show table status"},
		{"columns", "show columns from t1"},
		{"fields", "show fields from t1"},
		{"triggers", "show triggers"},
		{"events", "show events"},
		{"databases", "show databases"},
	} {
		t.Run("absent "+tc.name, func(t *testing.T) {
			ident := dbRefOfP58(t, mustParseOne(t, tc.sql))
			if ident.Present() {
				t.Fatalf("%s: Database = %+v, want absent", tc.name, ident)
			}
		})
	}

	// Edge spellings.
	t.Run("quoted empty", func(t *testing.T) {
		ident := dbRefOfP58(t, mustParseOne(t, "show tables in ``"))
		if !ident.Present() || ident.Raw != "``" {
			t.Fatalf("Database = %+v, want present ``", ident)
		}
	})
	t.Run("quoted dot", func(t *testing.T) {
		ident := dbRefOfP58(t, mustParseOne(t, "show tables in `a.b`"))
		if ident.Name != "a.b" || ident.Raw != "`a.b`" {
			t.Fatalf("Database = %+v, want a.b / `a.b`", ident)
		}
	})
	t.Run("UTF-8", func(t *testing.T) {
		const sql = "show tables in `预``库`"
		ident := dbRefOfP58(t, mustParseOne(t, sql))
		checkDbRefP58(t, sql, ident, lex.Ident{Name: "预`库", Raw: "`预``库`", Quote: lex.QuoteBacktick}, "SHOW TABLES UTF-8")
		if got, err := ident.Span.Slice(sql); err != nil || got != ident.Raw {
			t.Fatalf("UTF-8 token span selects %q (%v)", got, err)
		}
	})
	t.Run("ANSI double", func(t *testing.T) {
		p.SetSQLMode(mysql.ModeANSIQuotes)
		const sql = `show tables in "a""b"`
		stmt, err := p.ParseOneStmt(sql, "", "")
		if err != nil {
			t.Fatal(err)
		}
		ident := dbRefOfP58(t, stmt)
		checkDbRefP58(t, sql, ident, lex.Ident{Name: `a"b`, Raw: `"a""b"`, Quote: lex.QuoteDouble}, "SHOW TABLES ANSI")
	})
}

func TestExplainSchemaReference(t *testing.T) {
	cases := []struct {
		name, sql    string
		decoded, raw string
		quote        lex.QuoteStyle
		wantRestored string
	}{
		{"normal", "explain for schema db9 select 1", "db9", "db9", lex.QuoteUnquoted,
			"EXPLAIN FORMAT = 'row' FOR SCHEMA `db9` SELECT 1"},
		{"format json into", "explain format=json into @v for schema `sc``9` select 1", "sc`9", "`sc``9`", lex.QuoteBacktick,
			"EXPLAIN FORMAT = 'json' INTO @`v` FOR SCHEMA `sc``9` SELECT 1"},
		{"analyze", "explain analyze for schema dby select 1", "dby", "dby", lex.QuoteUnquoted,
			"EXPLAIN ANALYZE FOR SCHEMA `dby` SELECT 1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stmt := mustParseOne(t, tc.sql)
			ex := stmt.(*ast.ExplainStmt)
			checkDbRefP58(t, tc.sql, ex.Schema, lex.Ident{Name: tc.decoded, Raw: tc.raw, Quote: tc.quote}, "ExplainStmt.Schema")
			// Complete node span: the whole statement.
			if got, err := ex.Span().Slice(tc.sql); err != nil || got != tc.sql {
				t.Fatalf("span selects %q (%v), want %q", got, err, tc.sql)
			}
			if got := restoreStmt(t, stmt); got != tc.wantRestored {
				t.Fatalf("restore = %q, want %q", got, tc.wantRestored)
			}
		})
	}

	t.Run("absent", func(t *testing.T) {
		// No FOR SCHEMA is legal; the field stays absent.
		plain := mustParseOne(t, "explain select 1").(*ast.ExplainStmt)
		if plain.Schema.Present() {
			t.Fatalf("plain EXPLAIN Schema = %+v, want absent", plain.Schema)
		}
	})
	t.Run("quoted empty", func(t *testing.T) {
		ident := dbRefOfP58(t, mustParseOne(t, "explain for schema `` select 1"))
		if !ident.Present() || ident.Raw != "``" {
			t.Fatalf("Schema = %+v, want present ``", ident)
		}
	})
	t.Run("quoted dot", func(t *testing.T) {
		ident := dbRefOfP58(t, mustParseOne(t, "explain for schema `a.b` select 1"))
		if ident.Name != "a.b" || ident.Raw != "`a.b`" {
			t.Fatalf("Schema = %+v, want a.b / `a.b`", ident)
		}
	})
}

func TestDatabaseReferenceLeadingCommentAndMultiStmt(t *testing.T) {
	// The final statement span covers only the statement tokens, not the
	// leading comment.
	const sql = "-- a comment\nuse dba"
	stmts, _, err := New().Parse(sql, "", "")
	if err != nil || len(stmts) != 1 {
		t.Fatalf("parse: %v (%d stmts)", err, len(stmts))
	}
	use := stmts[0].(*ast.UseStmt)
	checkDbRefP58(t, sql, use.Database, lex.Ident{Name: "dba", Raw: "dba", Quote: lex.QuoteUnquoted}, "leading comment USE")
	if got, err := use.Span().Slice(sql); err != nil || got != "use dba" {
		t.Fatalf("span selects %q (%v), want %q", got, err, "use dba")
	}

	// Multi-statement: each span covers its own statement.
	const multi = "use dba; show tables in dbb; use db1"
	stmts, _, err = New().Parse(multi, "", "")
	if err != nil || len(stmts) != 3 {
		t.Fatalf("parse multi: %v", err)
	}
	if got, err := stmts[0].Span().Slice(multi); err != nil || got != "use dba" {
		t.Fatalf("first span selects %q (%v)", got, err)
	}
	if got, err := stmts[1].Span().Slice(multi); err != nil || got != "show tables in dbb" {
		t.Fatalf("second span selects %q (%v)", got, err)
	}
	if got, err := stmts[2].Span().Slice(multi); err != nil || got != "use db1" {
		t.Fatalf("third span selects %q (%v)", got, err)
	}
}

// TestDatabaseReferenceReuse pins zero-copy aliasing against a later parse on
// the same parser: undecoded names alias the input substring, decoded names
// do not, and retained ASTs survive parser reuse.
func TestDatabaseReferenceReuse(t *testing.T) {
	cases := []struct {
		name  string
		sql   string
		alias bool // true: undecoded name must alias the input
	}{
		{"use unquoted", "use db1", true},
		{"use unquoted utf8", "use 数据库", true},
		{"use quoted utf8", "use `数据库`", true},
		{"use escaped utf8", "use `分``区`", false},
		{"explain unquoted", "explain for schema db9 select 1", true},
		{"show escaped", "show tables in `My``Db`", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			stmt, err := p.ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			ident := dbRefOfP58(t, stmt)
			checkDbRefP58(t, tc.sql, ident, lex.Ident{Name: ident.Name, Raw: ident.Raw, Quote: ident.Quote}, "reuse")
			span, _ := tokenSpanP58(tc.sql, ident.Raw)
			if unsafe.StringData(ident.Raw) != unsafe.StringData(tc.sql[span.Start:span.End]) {
				t.Fatal("raw spelling does not share the input substring")
			}
			start := span.Start
			if ident.Quote != lex.QuoteUnquoted {
				start++
			}
			if tc.alias {
				if unsafe.StringData(ident.Name) != unsafe.StringData(tc.sql[start:]) {
					t.Fatalf("undecoded name does not alias the input")
				}
			} else {
				if unsafe.StringData(ident.Name) == unsafe.StringData(tc.sql[start:]) {
					t.Fatalf("decoded name aliases the input")
				}
			}
			// A later parse on the same parser must not mutate the retained AST.
			before, restoredBefore := dbRefOfP58(t, stmt), restoreStmt(t, stmt)
			if _, err := p.ParseOneStmt("select `other``name`", "", ""); err != nil {
				t.Fatal(err)
			}
			after := dbRefOfP58(t, stmt)
			if after != before || restoreStmt(t, stmt) != restoredBefore {
				t.Fatal("parser reuse changed the retained statement")
			}
		})
	}
}

type p58FailingRestoreWriter struct {
	err error
}

// Write implements io.Writer. Short-write returns 0, nil: the RestoreCtx
// promotes it to io.ErrShortWrite, which Restore must report.
func (w p58FailingRestoreWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestDatabaseReferenceRestoreWriterErrors(t *testing.T) {
	// Short write on the first call surfaces io.ErrShortWrite.
	stmt := mustParseOne(t, "use db1")
	w := p58FailingRestoreWriter{}
	if err := stmt.Restore(fmtrestore.NewRestoreCtx(fmtrestore.DefaultRestoreFlags, w)); !errors.Is(err, stdio.ErrShortWrite) {
		t.Fatalf("short write err = %v, want io.ErrShortWrite", err)
	}

	// A plain writer error propagates.
	stmt2 := mustParseOne(t, "show tables in db1")
	sentinel := errors.New("database reference writer")
	w2 := p58FailingRestoreWriter{err: sentinel}
	if err := stmt2.Restore(fmtrestore.NewRestoreCtx(fmtrestore.DefaultRestoreFlags, w2)); !errors.Is(err, sentinel) {
		t.Fatalf("writer err = %v, want writer sentinel", err)
	}

	// Hand-crafted missing names error without panicking.
	manual := []struct {
		name string
		stmt ast.StmtNode
	}{
		{"use missing", &ast.UseStmt{}},
		{"show create database missing", &ast.ShowStmt{Tp: ast.ShowCreateDatabase}},
	}
	for _, m := range manual {
		t.Run(m.name, func(t *testing.T) {
			err := m.stmt.Restore(fmtrestore.NewRestoreCtx(fmtrestore.DefaultRestoreFlags, &strings.Builder{}))
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestDatabaseReferenceRoundTrip(t *testing.T) {
	cases := []string{
		"use db1",
		"use ``",
		"show create database ``",
		"show create database if not exists `分``区`",
		"show tables in db1",
		"show tables",
		"show tables in ``",
		"show full tables from `t1`",
		"show open tables in db2",
		"show table status from `t``s`",
		"show columns from t1 in db3",
		"show fields from t1 in db4",
		"show triggers in db5",
		"show events from `ev``db`",
		"explain for schema db9 select 1",
		"explain select 1",
		"explain for schema `` select 1",
		"explain format=json into @v for schema `sc``9` select 1",
		"explain analyze for schema dby select 1",
	}
	for _, sql := range cases {
		t.Run(sql, func(t *testing.T) {
			stmt := mustParseOne(t, sql)
			stmt2 := mustParseOne(t, restoreStmt(t, stmt))
			CleanNodeText(stmt)
			CleanNodeText(stmt2)
			normalizeTestSyntaxMetadata(stmt)
			normalizeTestSyntaxMetadata(stmt2)
			if !reflect.DeepEqual(stmt, stmt2) {
				t.Fatalf("round trip mismatch:\n got %#v\n want %#v", stmt, stmt2)
			}
		})
	}
}

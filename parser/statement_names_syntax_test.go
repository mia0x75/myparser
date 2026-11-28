package parser

import (
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/mia0x75/myparser/ast"
	myformat "github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

// statementNameCase is one spelling variant of a statement name field.
type statementNameCase struct {
	name    string
	sql     string
	decoded string
	raw     string
	quote   lex.QuoteStyle
	// flags: the optional second Restore context applied on the same AST.
	flags myformat.RestoreFlags
	want  string
}

func TestStatementNamesDecodedIdentity(t *testing.T) {
	cases := []statementNameCase{
		{"prepare unquoted", "prepare st from 'select 1'", "st", "st", lex.QuoteUnquoted, 0, "PREPARE `st` FROM 'select 1'"},
		{"prepare escaped backtick", "prepare `s``1` from 'select 1'", "s`1", "`s``1`", lex.QuoteBacktick, 0, "PREPARE `s``1` FROM 'select 1'"},
		{"savepoint unquoted", "savepoint sp", "sp", "sp", lex.QuoteUnquoted, 0, "SAVEPOINT `sp`"},
		{"savepoint escaped backtick", "savepoint `a``b`", "a`b", "`a``b`", lex.QuoteBacktick, 0, "SAVEPOINT `a``b`"},
		{"savepoint empty backtick", "savepoint ``", "", "``", lex.QuoteBacktick, 0, "SAVEPOINT ``"},
		{"release savepoint unquoted", "release savepoint sp", "sp", "sp", lex.QuoteUnquoted, 0, "RELEASE SAVEPOINT `sp`"},
		{"rollback savepoint unquoted", "rollback to savepoint sp", "sp", "sp", lex.QuoteUnquoted, 0, "ROLLBACK TO SAVEPOINT `sp`"},
		{"rollback savepoint empty backtick", "rollback to savepoint ``", "", "``", lex.QuoteBacktick, 0, "ROLLBACK TO SAVEPOINT ``"},
		{"execute unquoted", "execute st using @v", "st", "st", lex.QuoteUnquoted, 0, "EXECUTE `st` USING @`v`"},
		{"deallocate unquoted", "deallocate prepare st", "st", "st", lex.QuoteUnquoted, 0, "DEALLOCATE PREPARE `st`"},
		{"deallocate escaped backtick", "deallocate prepare `d``s`", "d`s", "`d``s`", lex.QuoteBacktick, 0, "DEALLOCATE PREPARE `d``s`"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			stmt, err := p.ParseOneStmt(tc.sql)
			if err != nil {
				t.Fatal(err)
			}
			ident := statementNodeName(t, stmt)
			present := ident.Name != "" || ident.Raw != "" || ident.Quote != lex.QuoteUnquoted
			if present != ident.Present() {
				t.Fatalf("presence inconsistent with identity: %+v", ident)
			}
			// The shared string must survive a later parse of different SQL.
			if _, err := p.ParseOneStmt("select 1"); err != nil {
				t.Fatal(err)
			}
			if ident.Name != tc.decoded || ident.Raw != tc.raw || ident.Quote != tc.quote {
				t.Fatalf("ident = %+v, want decoded=%q raw=%q quote=%v", ident, tc.decoded, tc.raw, tc.quote)
			}
			if statementNodeName(t, stmt) != ident || restoreStmt(t, stmt) != tc.want {
				t.Fatal("parser reuse changed the retained statement")
			}
		})
	}
}

func statementNodeName(t *testing.T, stmt ast.StmtNode) lex.Ident {
	t.Helper()
	var ident lex.Ident
	switch stmt := stmt.(type) {
	case *ast.PrepareStmt:
		ident = stmt.Name
	case *ast.SavepointStmt:
		ident = stmt.Name
	case *ast.ReleaseSavepointStmt:
		ident = stmt.Name
	case *ast.RollbackStmt:
		ident = stmt.SavepointName
	case *ast.ExecuteStmt:
		ident = stmt.Name
	case *ast.DeallocateStmt:
		ident = stmt.Name
	default:
		t.Fatalf("unexpected statement %T", stmt)
	}
	return ident
}

func TestStatementNamesSpans(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want loc.Span
	}{
		{"prepare", "prepare st from 'select 1'", loc.Span{Start: 8, End: 10}},
		{"savepoint", "savepoint sp", loc.Span{Start: 10, End: 12}},
		{"release savepoint", "release savepoint sp", loc.Span{Start: 18, End: 20}},
		{"rollback savepoint", "rollback to savepoint sp", loc.Span{Start: 22, End: 24}},
		{"execute", "execute st", loc.Span{Start: 8, End: 10}},
		{"deallocate", "deallocate prepare st", loc.Span{Start: 19, End: 21}},
		{"prepare node span", "prepare st from 'select 1'", loc.Span{Start: 0, End: 26}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stmt := mustParseOne(t, tc.sql)
			var ident lex.Ident
			switch stmt := stmt.(type) {
			case *ast.PrepareStmt:
				ident = stmt.Name
			case *ast.SavepointStmt:
				ident = stmt.Name
			case *ast.ReleaseSavepointStmt:
				ident = stmt.Name
			case *ast.RollbackStmt:
				ident = stmt.SavepointName
			case *ast.ExecuteStmt:
				ident = stmt.Name
			case *ast.DeallocateStmt:
				ident = stmt.Name
			default:
				t.Fatalf("unexpected statement %T", stmt)
			}
			if tc.name == "prepare node span" {
				if !stmt.Span().Within(len(tc.sql)) || stmt.Span() != tc.want {
					t.Fatalf("node span = %v, want %v", stmt.Span(), tc.want)
				}
				return
			}
			if !ident.Span.Within(len(tc.sql)) || ident.Span != tc.want {
				t.Fatalf("ident span = %v, want %v", ident.Span, tc.want)
			}
			if got, err := stmt.Span().Slice(tc.sql); err != nil || got != tc.sql {
				t.Fatalf("node span selects %q (%v), want %q", got, err, tc.sql)
			}
		})
	}
}

func TestStatementNamesRestoreSpelling(t *testing.T) {
	// Restore spelling follows the shared fold + writeQuoted pipeline: the
	// backtick in a decoded name is escaped only when backtick is the active
	// quote style, and user variable names fold to the active case.
	cases := []statementNameCase{
		// String quoting follows the string flags; single quotes only come
		// from the default RestoreStringSingleQuotes.
		{"prepare double quotes", "prepare `s``1` from 'select 1'", "s`1", "`s``1`", lex.QuoteBacktick, myformat.RestoreNameDoubleQuotes, "PREPARE \"s`1\" FROM select 1"},
		{"savepoint double quotes", "savepoint `a``b`", "a`b", "`a``b`", lex.QuoteBacktick, myformat.RestoreNameDoubleQuotes, "SAVEPOINT \"a`b\""},
		{"execute uppercase", "execute st using @v", "st", "st", lex.QuoteUnquoted, myformat.RestoreNameUppercase, "EXECUTE ST USING @V"},
		// Keywords stay uppercase: only the name slot honors RestoreNameLowercase.
		{"deallocate lowercase", "DEALLOCATE PREPARE ST", "ST", "ST", lex.QuoteUnquoted, myformat.RestoreNameLowercase, "DEALLOCATE PREPARE st"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stmt := mustParseOne(t, tc.sql)
			var sb strings.Builder
			err := stmt.Restore(myformat.NewRestoreCtx(tc.flags, &sb))
			if err != nil {
				t.Fatalf("restore: %v", err)
			}
			if got := sb.String(); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStatementNamesZeroCopyReuse(t *testing.T) {
	for _, name := range []string{"retained_name", "`保``留`"} {
		t.Run(name, func(t *testing.T) {
			p := New()
			sql := "prepare " + name + " from 'select 1'"
			stmt, err := p.ParseOneStmt(sql)
			if err != nil {
				t.Fatal(err)
			}
			ident := statementNodeName(t, stmt)
			if unsafe.StringData(ident.Raw) != unsafe.StringData(sql[ident.Span.Start:ident.Span.End]) {
				t.Fatal("raw name does not share the input substring")
			}
			if ident.Quote == lex.QuoteUnquoted && unsafe.StringData(ident.Name) != unsafe.StringData(ident.Raw) {
				t.Fatal("undecoded name unnecessarily copied the input substring")
			}
			canonical := restoreStmt(t, stmt)
			if _, err := p.ParseOneStmt("select `other``name`"); err != nil {
				t.Fatal(err)
			}
			if statementNodeName(t, stmt) != ident || restoreStmt(t, stmt) != canonical {
				t.Fatal("parser reuse changed the retained statement")
			}
		})
	}
}

func TestStatementNamesRoundTrip(t *testing.T) {
	cases := []string{
		"prepare st from 'select 1'",
		"prepare st from @a",
		"deallocate prepare st",
		"execute st using @v",
		"savepoint sp",
		"release savepoint sp",
		"rollback to savepoint sp",
		"rollback",
		"rollback to savepoint ``",
	}
	for _, sql := range cases {
		t.Run(sql, func(t *testing.T) {
			stmt := mustParseOne(t, sql)
			var sb strings.Builder
			if err := stmt.Restore(myformat.NewRestoreCtx(myformat.DefaultRestoreFlags, &sb)); err != nil {
				t.Fatalf("restore: %v", err)
			}
			restored := sb.String()
			stmt2 := mustParseOne(t, restored)
			CleanNodeText(stmt)
			CleanNodeText(stmt2)
			normalizeTestNameMetadata(stmt)
			normalizeTestNameMetadata(stmt2)
			if !reflect.DeepEqual(stmt, stmt2) {
				t.Fatalf("round trip: got %#v, want %#v", stmt2, stmt)
			}
		})
	}
}

func TestStatementNamesAbsentSavepoint(t *testing.T) {
	full := mustParseOne(t, "rollback").(*ast.RollbackStmt)
	if full.SavepointName.Present() {
		t.Fatalf("full rollback SavepointName = %+v, want absent", full.SavepointName)
	}
	if got := restoreStmt(t, full); got != "ROLLBACK" {
		t.Fatalf("restore = %q, want %q", got, "ROLLBACK")
	}
	empty := mustParseOne(t, "rollback to savepoint ``").(*ast.RollbackStmt)
	if !empty.SavepointName.Present() || empty.SavepointName.Raw != "``" {
		t.Fatalf("empty quoted rollback SavepointName = %+v, want present ``", empty.SavepointName)
	}
	if got := restoreStmt(t, empty); got != "ROLLBACK TO SAVEPOINT ``" {
		t.Fatalf("restore = %q, want %q", got, "ROLLBACK TO SAVEPOINT ``")
	}
}

func TestExecuteImmediateNameAbsent(t *testing.T) {
	cases := []string{
		"execute immediate 'select 1'",
		"execute immediate 'select 1' using @v",
	}
	for _, sql := range cases {
		t.Run(sql, func(t *testing.T) {
			stmt := mustParseOne(t, sql).(*ast.ExecuteStmt)
			if stmt.Name.Present() || stmt.SQLExpr == nil {
				t.Fatalf("EXECUTE IMMEDIATE = %+v, want absent name and SQL expression", stmt)
			}
			if got, err := stmt.Span().Slice(sql); err != nil || got != sql {
				t.Fatalf("node span selects %q (%v), want %q", got, err, sql)
			}
		})
	}
}

func TestStatementNamesUTF8Spans(t *testing.T) {
	for _, sql := range []string{
		"prepare `预``名` from 'select 1'", "execute `预``名` using @v", "deallocate prepare `预``名`",
		"savepoint `預``名`", "release savepoint `預``名`", "rollback work to savepoint `預``名`",
		"rollback work and no chain no release",
	} {
		t.Run(sql, func(t *testing.T) {
			stmt := mustParseOne(t, sql)
			if got, err := stmt.Span().Slice(sql); err != nil || got != sql {
				t.Fatalf("node span selects %q (%v), want %q", got, err, sql)
			}
			id := statementNodeName(t, stmt)
			if id.Present() {
				if got, err := id.Span.Slice(sql); err != nil || got != id.Raw {
					t.Fatalf("name span selects %q (%v), raw=%q", got, err, id.Raw)
				}
			}
		})
	}
}

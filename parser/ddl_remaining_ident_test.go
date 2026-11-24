package myparser

import (
	"strings"
	"testing"

	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/myparser/ast"
	fmtrestore "github.com/mia0x75/myparser/format"
)

// TestDdlRemainingPeriodIdent pins the PeriodDefinition Name/Span contract:
// Name is a lex.Ident with the real token span (not the node span, which
// covers the whole PERIOD FOR ... clause), and the MySQL dialect issue still
// highlights the PERIOD keyword at the node Span.Start.
func TestDdlRemainingPeriodIdent(t *testing.T) {
	const sql = "CREATE TABLE t(s DATE, e DATE, PERIOD FOR p(s,e))"
	stmt := mustParseOne(t, sql).(*ast.CreateTableStmt)
	if len(stmt.Periods) != 1 {
		t.Fatalf("period declaration lost: %#v", stmt.Periods)
	}
	period := stmt.Periods[0]

	// Name.Span is the real PERIOD-for-name token span, not the node span.
	if !period.Name.Span.Valid() || period.Name.Span.Start == period.Span().Start {
		t.Fatalf("period Name.Span = %#v, want the identifier token span distinct from the node span %#v", period.Name.Span, period.Span())
	}
	if got := sql[int(period.Name.Span.Start):int(period.Name.Span.End)]; got != "p" {
		t.Fatalf("period name slice = %q, want %q", got, "p")
	}

	// The node span covers the whole PERIOD FOR clause; its start is the
	// PERIOD keyword.
	if got := sql[int(period.Span().Start) : int(period.Span().Start)+len("PERIOD")]; got != "PERIOD" {
		t.Fatalf("node span start points at %q, want the PERIOD keyword", got)
	}

	// The MySQL dialect issue still highlights the PERIOD keyword.
	issues := CheckDialectCompatibility(stmt, sql, DialectMySQL)
	if len(issues) != 1 || issues[0].Code != IssueMySQLPeriod {
		t.Fatalf("issues = %#v, want exactly one IssueMySQLPeriod", issues)
	}
	if issues[0].Highlight != "PERIOD" {
		t.Fatalf("period highlight = %q, want the PERIOD keyword", issues[0].Highlight)
	}
}

// TestDdlRemainingPeriodSpanRestore pins that the period node span and name
// survive a restore/reparse round trip and a parser instance reused for a
// second parse.
func TestDdlRemainingPeriodSpanRestore(t *testing.T) {
	const sql = "CREATE TABLE t(s DATE, e DATE, PERIOD FOR p(s,e))"
	parser := New()
	stmts, _, err := parser.Parse(sql, "", "")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	restored := restoreStmt(t, stmts[0])
	if !strings.Contains(restored, "PERIOD FOR `p` (`s`,`e`)") {
		t.Fatalf("restore = %q, want the period spelling", restored)
	}

	// Reuse the same parser instance: spans must not be polluted by the
	// previous parse.
	reparse, err := parser.ParseOneStmt(restored, "", "")
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	rePeriod := reparse.(*ast.CreateTableStmt).Periods[0]
	if !rePeriod.Name.Span.Valid() || rePeriod.Name.Name != "p" {
		t.Fatalf("reparsed period = %#v, want name p with a valid span", rePeriod.Name)
	}
	if !rePeriod.Span().Valid() {
		t.Fatal("reparsed period node lost its span")
	}

	// A hand-built node keeps its explicitly set node span; the name span is
	// independent of the node span.
	manual := &ast.PeriodDefinition{Name: lex.Ident{Name: "q", Span: loc.Span{Start: 3, End: 4}}}
	manual.SetSpan(loc.Span{Start: 0, End: 40})
	manual.Start = qualPeriodName("s")
	manual.End = qualPeriodName("e")
	var sb strings.Builder
	if err := manual.Restore(fmtrestore.NewRestoreCtx(fmtrestore.DefaultRestoreFlags, &sb)); err != nil {
		t.Fatalf("manual restore: %v", err)
	}
	if got := sb.String(); got != "PERIOD FOR `q` (`s`,`e`)" {
		t.Fatalf("manual restore = %q, want the period spelling", got)
	}
}

func qualPeriodName(part string) ast.QualifiedName {
	return ast.QualifiedName{Parts: []lex.Ident{{Name: part}}}
}

func TestDDLRemainingAlterNames(t *testing.T) {
	for _, tc := range []struct {
		name, clause string
		rename       bool
	}{
		{"drop index", "DROP INDEX `索``名`", false},
		{"drop foreign", "DROP FOREIGN KEY IF EXISTS `索``名`", false},
		{"drop partition", "DROP PARTITION `索``名`", false},
		{"truncate partition", "TRUNCATE PARTITION `索``名`", false},
		{"exchange partition", "EXCHANGE PARTITION `索``名` WITH TABLE p WITHOUT VALIDATION", false},
		{"rename index", "RENAME INDEX `旧``名` TO `新``名`", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			sql := "ALTER TABLE t " + tc.clause
			stmt, err := p.ParseOneStmt(sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			spec := stmt.(*ast.AlterTableStmt).Specs[0]
			if got, err := spec.Span().Slice(sql); err != nil || got != tc.clause {
				t.Fatalf("spec span selects %q (%v), want %q", got, err, tc.clause)
			}
			ids, names := []lex.Ident{spec.Name}, []string{"索`名"}
			if tc.rename {
				ids, names = []lex.Ident{spec.FromKey, spec.ToKey}, []string{"旧`名", "新`名"}
			}
			for i, id := range ids {
				if id.Name != names[i] || !id.Present() || id.Quote != lex.QuoteBacktick {
					t.Fatalf("name %d = %+v, want %q", i, id, names[i])
				}
				if got, err := id.Span.Slice(sql); err != nil || got != id.Raw {
					t.Fatalf("name span selects %q (%v), raw=%q", got, err, id.Raw)
				}
			}
			canonical := restoreStmt(t, stmt)
			reparsed, err := p.ParseOneStmt(canonical, "", "")
			if err != nil {
				t.Fatalf("reparse %q: %v", canonical, err)
			}
			next := reparsed.(*ast.AlterTableStmt).Specs[0]
			if spec.Name.Name != next.Name.Name || spec.FromKey.Name != next.FromKey.Name || spec.ToKey.Name != next.ToKey.Name {
				t.Fatal("round trip changed name roles")
			}
			if spec.Name != ids[0] && !tc.rename || tc.rename && (spec.FromKey != ids[0] || spec.ToKey != ids[1]) || restoreStmt(t, stmt) != canonical {
				t.Fatal("parser reuse changed retained names")
			}
		})
	}
}

func TestDDLRemainingPeriodQuotedNames(t *testing.T) {
	for _, tc := range []struct {
		raw, name string
		quote     lex.QuoteStyle
	}{
		{"p", "p", lex.QuoteUnquoted},
		{"`期``间`", "期`间", lex.QuoteBacktick},
		{"``", "", lex.QuoteBacktick},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			clause := "PERIOD FOR " + tc.raw + "(s,e)"
			sql := "CREATE TABLE t(s DATE,e DATE," + clause + ")"
			p := New()
			stmt, err := p.ParseOneStmt(sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			period := stmt.(*ast.CreateTableStmt).Periods[0]
			id := period.Name
			if !id.Present() || id.Name != tc.name || id.Raw != tc.raw || id.Quote != tc.quote {
				t.Fatalf("period name = %+v", id)
			}
			if got, err := id.Span.Slice(sql); err != nil || got != tc.raw {
				t.Fatalf("name span selects %q (%v), want %q", got, err, tc.raw)
			}
			if got, err := period.Span().Slice(sql); err != nil || got != clause {
				t.Fatalf("period span selects %q (%v), want %q", got, err, clause)
			}
			issues := CheckDialectCompatibility(stmt, sql, DialectMySQL)
			if len(issues) != 1 || issues[0].Code != IssueMySQLPeriod || issues[0].Highlight != "PERIOD" {
				t.Fatalf("period issue changed: %+v", issues)
			}
			canonical := restoreStmt(t, stmt)
			reparsed, err := p.ParseOneStmt(canonical, "", "")
			if err != nil {
				t.Fatalf("reparse %q: %v", canonical, err)
			}
			next := reparsed.(*ast.CreateTableStmt).Periods[0].Name
			if next.Name != id.Name || !next.Present() || period.Name != id || restoreStmt(t, stmt) != canonical {
				t.Fatal("round trip or parser reuse changed period name")
			}
		})
	}
}

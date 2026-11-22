package myparser

import (
	"strings"
	"testing"
	"unsafe"

	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/myparser/mysql"
)

func TestSpanLexicalSpelling(t *testing.T) {
	for _, tc := range []struct {
		name, sql, lit, raw string
		tok, skip           int
		mode                mysql.SQLMode
		target              Dialect
	}{
		{name: "identifier", sql: "abc", tok: identifier, lit: "abc", raw: "abc"},
		{name: "keyword", sql: "SELECT", tok: selectKwd, lit: "SELECT", raw: "SELECT"},
		{name: "backtick", sql: "`a``b`", tok: identifier, lit: "a`b", raw: "`a``b`"},
		{name: "empty backtick", sql: "``", tok: identifier, raw: "``"},
		{name: "string", sql: "'abc'", tok: stringLit, lit: "abc", raw: "'abc'"},
		{name: "double quote", sql: `"abc"`, tok: stringLit, lit: "abc", raw: `"abc"`},
		{name: "empty string", sql: "''", tok: stringLit, raw: "''"},
		{name: "doubled quote", sql: "'a''b'", tok: stringLit, lit: "a'b", raw: "'a''b'"},
		{name: "doubled double quote", sql: `"a""b"`, tok: stringLit, lit: `a"b`, raw: `"a""b"`},
		{name: "escape", sql: `'a\nb'`, tok: stringLit, lit: "a\nb", raw: `'a\nb'`},
		{name: "no escape mode", sql: `'a\nb'`, tok: stringLit, lit: `a\nb`, raw: `'a\nb'`, mode: mysql.ModeNoBackslashEscapes},
		{name: "ANSI quote", sql: `"a""b"`, tok: identifier, lit: `a"b`, raw: `"a""b"`, mode: mysql.ModeANSIQuotes},
		{name: "empty ANSI quote", sql: `""`, tok: identifier, raw: `""`, mode: mysql.ModeANSIQuotes},
		{name: "UTF8 identifier", sql: " 名字", tok: identifier, lit: "名字", raw: "名字"},
		{name: "UTF8 string", sql: "'名字'", tok: stringLit, lit: "名字", raw: "'名字'"},
		{name: "hex", sql: "X'12'", tok: hexLit, lit: "X'12'", raw: "X'12'"},
		{name: "bit", sql: "B'01'", tok: bitLit, lit: "B'01'", raw: "B'01'"},
		{name: "decimal", sql: "1.20", tok: decLit, lit: "1.20", raw: "1.20"},
		{name: "parameter", sql: "?", tok: paramMarker, raw: "?", lit: "?"},
		{name: "line comment", sql: "-- comment\nSELECT", tok: selectKwd, lit: "SELECT", raw: "SELECT"},
		{name: "hash comment", sql: "# comment\nSELECT", tok: selectKwd, lit: "SELECT", raw: "SELECT"},
		{name: "block comment", sql: "/* skipped */ SELECT", tok: selectKwd, lit: "SELECT", raw: "SELECT"},
		{name: "MySQL comment", sql: "/*!50000 `a``b` */", tok: identifier, lit: "a`b", raw: "`a``b`", target: DialectMySQL},
		{name: "nested ordinary comment", sql: "/*! /* skipped */ `a``b` */", tok: identifier, lit: "a`b", raw: "`a``b`"},
		{name: "MariaDB comment", sql: "/*M! `a``b` */", tok: identifier, lit: "a`b", raw: "`a``b`", target: DialectMariaDB},
		{name: "skipped MariaDB comment", sql: "/*M! `a``b` */ x", tok: identifier, lit: "x", raw: "x", target: DialectMySQL},
		{name: "respect nulls", sql: "LAST_VALUE(x) RESPECT NULLS", skip: 4, tok: respectNulls, lit: "RESPECT", raw: "RESPECT NULLS"},
		{name: "ignore nulls", sql: "LAST_VALUE(x) IGNORE\t\nNULLS", skip: 4, tok: ignoreNulls, lit: "IGNORE", raw: "IGNORE\t\nNULLS"},
		{name: "nulls without follow", sql: "RESPECT NULLS", tok: respect, lit: "RESPECT", raw: "RESPECT"},
		{name: "nulls Unicode boundary", sql: "LAST_VALUE(x) IGNORE NULLS名", skip: 4, tok: ignore, lit: "IGNORE", raw: "IGNORE"},
		// The old boundary check merges an ASCII suffix; this batch must not fix it.
		{name: "nulls ASCII suffix", sql: "LAST_VALUE(x) IGNORE NULLSxyz", skip: 5, tok: identifier, lit: "xyz", raw: "xyz"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewScanner(tc.sql)
			s.sqlMode, s.target, s.supportWindowFunc = tc.mode, tc.target, true
			var v yySymType
			for range tc.skip {
				s.Lex(&v)
			}
			tok := s.Lex(&v)
			start := strings.Index(tc.sql, tc.raw)
			want := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(tc.raw))}
			if tok != tc.tok || v.ident != tc.lit || v.offset != start || v.span != want {
				t.Fatalf("got token=%d lit=%q offset=%d span=%v, want %d %q %d %v", tok, v.ident, v.offset, v.span, tc.tok, tc.lit, start, want)
			}
			raw, err := v.span.Slice(tc.sql)
			if err != nil || raw != tc.raw {
				t.Fatalf("raw = %q, %v; want %q", raw, err, tc.raw)
			}
		})
	}
}

func TestSpanHintMarkersAndBody(t *testing.T) {
	for _, sql := range []string{"/*+ a(1) */ x", "/*+ \x00 a(1) */ x"} {
		t.Run(sql, func(t *testing.T) {
			s := NewScanner(sql)
			var v yySymType
			type hintToken struct {
				tok      int
				lit, raw string
			}
			want := []hintToken{{hintBegin, "", "/*+"}}
			if strings.ContainsRune(sql, 0) {
				want = append(want, hintToken{invalid, "", "\x00"})
			}
			want = append(want, []hintToken{
				{identifier, "a", "a"}, {'(', "(", "("}, {intLit, "1", "1"},
				{')', ")", ")"}, {hintEnd, "", "*/"}, {identifier, "x", "x"},
			}...)
			for _, w := range want {
				tok := s.Lex(&v)
				start := strings.Index(sql, w.raw)
				span := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(w.raw))}
				if tok != w.tok || v.ident != w.lit || v.offset != start || v.span != span {
					t.Fatalf("got token=%d lit=%q offset=%d span=%v, want %d %q %d %v", tok, v.ident, v.offset, v.span, w.tok, w.lit, start, span)
				}
			}
			if tok := s.Lex(&v); tok != 0 || v.span != loc.InvalidSpan() {
				t.Fatalf("EOF = %d, %v", tok, v.span)
			}
		})
	}
}

func TestSpanDelimiterKeepsLegacyOffset(t *testing.T) {
	sql := "DELIMITER $$\n1$$\nDELIMITER ;\n2;"
	var s Scanner
	s.reset(sql)
	var v yySymType
	for _, w := range []struct {
		tok        int
		lit, raw   string
		start, end int
		offset     int
	}{
		{intLit, "1", "1", 13, 14, 13}, {';', "", "$$", 14, 16, 16},
		{intLit, "2", "2", 29, 30, 29}, {';', ";", ";", 30, 31, 30},
	} {
		tok := s.Lex(&v)
		span := loc.Span{Start: loc.ByteOffset(w.start), End: loc.ByteOffset(w.end)}
		if tok != w.tok || v.ident != w.lit || v.offset != w.offset || v.span != span {
			t.Fatalf("got token=%d lit=%q offset=%d span=%v, want %d %q %d %v", tok, v.ident, v.offset, v.span, w.tok, w.lit, w.offset, span)
		}
		raw, err := v.span.Slice(sql)
		if err != nil || raw != w.raw {
			t.Fatalf("delimiter raw = %q, %v", raw, err)
		}
	}
}

func TestSpanErrorsAndEOF(t *testing.T) {
	for _, tc := range []struct {
		name, sql string
		tok, errs int
		span      loc.Span
	}{
		{"empty", "", 0, 0, loc.InvalidSpan()},
		{"whitespace", " \n\t", 0, 0, loc.InvalidSpan()},
		{"unterminated backtick", "`abc", 0, 0, loc.Span{Start: 0, End: 4}},
		{"unterminated string", "'abc", 0, 1, loc.Span{Start: 0, End: 4}},
		{"unterminated comment", "/* abc", 0, 1, loc.Span{Start: 0, End: 6}},
		{"unconsumed invalid", "\x00", invalid, 0, loc.Span{Start: 0, End: 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewScanner(tc.sql)
			v := yySymType{span: loc.Span{Start: 50, End: 60}}
			tok := s.Lex(&v)
			if tok != tc.tok || v.span != tc.span || len(s.errs) != tc.errs {
				t.Fatalf("got token=%d span=%v errors=%v, want %d %v %d errors", tok, v.span, s.errs, tc.tok, tc.span, tc.errs)
			}
		})
	}
}

func TestSpanScannerReuseAndSubstringLifetime(t *testing.T) {
	for _, tc := range []struct{ sql, want string }{
		{"abc", "abc"}, {"'abc'", "abc"}, {"`a``b`", "a`b"}, {`'a\nb'`, "a\nb"},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			sql := tc.sql
			s := NewScanner(sql)
			var v yySymType
			s.Lex(&v)
			saved := v
			if sql == "abc" || sql == "'abc'" {
				start := 0
				if sql[0] == '\'' {
					start = 1
				}
				if unsafe.StringData(v.ident) != unsafe.StringData(sql[start:]) {
					t.Fatal("undecoded token no longer reuses the input substring")
				}
			}
			if tok := s.Lex(&v); tok != 0 || v.span != loc.InvalidSpan() {
				t.Fatalf("reused lval EOF = %d, %v", tok, v.span)
			}
			s.reset("'other\\ntext'")
			if s.lastScanSpan != loc.InvalidSpan() {
				t.Fatal("reset retained a previous token range")
			}
			s.Lex(&v)
			if saved.ident == "" || saved.span != (loc.Span{Start: 0, End: loc.ByteOffset(len(sql))}) {
				t.Fatal("saved semantic value lost its spelling or range")
			}
			if saved.ident != tc.want {
				t.Fatalf("saved spelling changed after reset: %q, want %q", saved.ident, tc.want)
			}
		})
	}
}

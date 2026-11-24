package myparser

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	dformat "github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/mysql"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

func TestSelectItemAliasSyntax(t *testing.T) {
	for _, tc := range []struct {
		name, sql, decoded, raw string
		quote                   lex.QuoteStyle
		mode                    mysql.SQLMode
	}{
		{name: "absent", sql: "SELECT 1"},
		{name: "unquoted", sql: "SELECT 1 AS Ab", decoded: "Ab", raw: "Ab"},
		{name: "implicit string", sql: "SELECT 1 'a.b'", decoded: "a.b", raw: "'a.b'", quote: lex.QuoteSingle},
		{name: "single empty", sql: "SELECT 1 AS ''", raw: "''", quote: lex.QuoteSingle},
		{name: "double empty", sql: `SELECT 1 AS ""`, raw: `""`, quote: lex.QuoteDouble},
		{name: "backtick empty", sql: "SELECT 1 AS ``", raw: "``", quote: lex.QuoteBacktick},
		{name: "escaped backtick", sql: "SELECT 1 AS `a``b.c`", decoded: "a`b.c", raw: "`a``b.c`", quote: lex.QuoteBacktick},
		{name: "unicode single", sql: "SELECT 1 AS '列''别'", decoded: "列'别", raw: "'列''别'", quote: lex.QuoteSingle},
		{name: "escaped string", sql: `SELECT 1 AS 'a\nb'`, decoded: "a\nb", raw: `'a\nb'`, quote: lex.QuoteSingle},
		{
			name: "no backslash escapes", sql: `SELECT 1 AS 'a\nb'`, decoded: `a\nb`, raw: `'a\nb'`,
			quote: lex.QuoteSingle, mode: mysql.ModeNoBackslashEscapes,
		},
		{name: "double string", sql: `SELECT 1 AS "a""b.c"`, decoded: `a"b.c`, raw: `"a""b.c"`, quote: lex.QuoteDouble},
		{
			name: "ansi identifier", sql: `SELECT 1 AS "a""b.c"`, decoded: `a"b.c`, raw: `"a""b.c"`,
			quote: lex.QuoteDouble, mode: mysql.ModeANSIQuotes,
		},
		{name: "window keyword", sql: "SELECT 1 AS RaNk", decoded: "RaNk", raw: "RaNk"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := New()
			p.SetSQLMode(tc.mode)
			p.EnableWindowFunc(true)
			stmt, err := p.ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			fields := stmt.(*ast.SelectStmt).Fields
			if len(fields.Items) != 1 {
				t.Fatalf("items = %d, want 1", len(fields.Items))
			}
			item := fields.Items[0]
			alias := item.Alias
			if alias.Name != tc.decoded || alias.Raw != tc.raw || alias.Quote != tc.quote {
				t.Fatalf("alias = %+v, want name %q, raw %q, quote %v", alias, tc.decoded, tc.raw, tc.quote)
			}
			if alias.Present() != (tc.raw != "") {
				t.Fatalf("alias presence = %v", alias.Present())
			}
			if tc.raw != "" {
				start := strings.Index(tc.sql, tc.raw)
				want := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(tc.raw))}
				if alias.Span != want {
					t.Fatalf("alias span = %v, want %v", alias.Span, want)
				}
			} else if alias.Span != loc.InvalidSpan() {
				t.Fatalf("absent alias span = %v, want unknown", alias.Span)
			}
			wantSpan := loc.Span{Start: 7, End: loc.ByteOffset(len(tc.sql))}
			if item.Span() != wantSpan || fields.Span() != wantSpan {
				t.Fatalf("item/list spans = %v/%v, want %v", item.Span(), fields.Span(), wantSpan)
			}
			var restored strings.Builder
			if err := item.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &restored)); err != nil {
				t.Fatal(err)
			}
			wantSQL := "1"
			if alias.Present() {
				wantSQL += " AS `" + strings.ReplaceAll(tc.decoded, "`", "``") + "`"
			}
			if restored.String() != wantSQL {
				t.Fatalf("restore = %q, want %q", restored.String(), wantSQL)
			}
		})
	}
}

func TestWildcardProjectionSyntax(t *testing.T) {
	for _, tc := range []struct {
		name, sql string
		parts     []lex.Ident
		mode      mysql.SQLMode
	}{
		{name: "unqualified", sql: "SELECT *"},
		{name: "table", sql: "SELECT t.*", parts: []lex.Ident{{Name: "t", Raw: "t"}}},
		{
			name: "quoted empty qualifier", sql: "SELECT ``.*",
			parts: []lex.Ident{{Raw: "``", Quote: lex.QuoteBacktick}},
		},
		{
			name: "quoted dots", sql: "SELECT `db.dot`.`t``.dot`.*",
			parts: []lex.Ident{
				{Name: "db.dot", Raw: "`db.dot`", Quote: lex.QuoteBacktick},
				{Name: "t`.dot", Raw: "`t``.dot`", Quote: lex.QuoteBacktick},
			},
		},
		{
			name: "ansi unicode", sql: `SELECT "库.名"."表""名".*`, mode: mysql.ModeANSIQuotes,
			parts: []lex.Ident{
				{Name: "库.名", Raw: `"库.名"`, Quote: lex.QuoteDouble},
				{Name: `表"名`, Raw: `"表""名"`, Quote: lex.QuoteDouble},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := New()
			p.SetSQLMode(tc.mode)
			stmt, err := p.ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			item := stmt.(*ast.SelectStmt).Fields.Items[0]
			wildcard, ok := item.Expr.(*ast.WildcardExpr)
			if !ok {
				t.Fatalf("projection expression = %T, want wildcard", item.Expr)
			}
			if item.Alias.Present() || len(wildcard.Qualifier.Parts) != len(tc.parts) {
				t.Fatalf("wildcard alias/qualifier = %+v/%+v", item.Alias, wildcard.Qualifier)
			}
			if item.Alias.Span != loc.InvalidSpan() {
				t.Fatalf("absent wildcard alias span = %v, want unknown", item.Alias.Span)
			}
			for i, want := range tc.parts {
				part := wildcard.Qualifier.Parts[i]
				start := strings.Index(tc.sql, want.Raw)
				want.Span = loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(want.Raw))}
				if part != want {
					t.Fatalf("part %d = %+v, want %+v", i, part, want)
				}
			}
			span := loc.Span{Start: 7, End: loc.ByteOffset(len(tc.sql))}
			if item.Span() != span || wildcard.Span() != span {
				t.Fatalf("item/wildcard spans = %v/%v, want %v", item.Span(), wildcard.Span(), span)
			}
			if len(tc.parts) > 0 {
				nameSpan := loc.Span{Start: 7, End: wildcard.Qualifier.Parts[len(tc.parts)-1].Span.End}
				if wildcard.Qualifier.Span != nameSpan {
					t.Fatalf("qualifier span = %v, want %v", wildcard.Qualifier.Span, nameSpan)
				}
			}
		})
	}
}

func TestProjectionDataOutlivesParserReuse(t *testing.T) {
	p := New()
	sql := "SELECT `库.名`.`表``名`.*, (1 + 2) AS '列''别'"
	stmt, err := p.ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}
	fields := stmt.(*ast.SelectStmt).Fields
	wildcard := fields.Items[0].Expr.(*ast.WildcardExpr)
	// Clone expected spelling so a mutable-buffer alias would fail this test.
	alias := fields.Items[1].Alias
	alias.Name, alias.Raw = strings.Clone(alias.Name), strings.Clone(alias.Raw)
	parts := append([]lex.Ident(nil), wildcard.Qualifier.Parts...)
	for i := range parts {
		parts[i].Name, parts[i].Raw = strings.Clone(parts[i].Name), strings.Clone(parts[i].Raw)
	}
	spans := []loc.Span{fields.Span(), fields.Items[0].Span(), fields.Items[1].Span(), wildcard.Span()}
	for i := range 20 {
		next := "SELECT 0 AS '" + strings.Repeat("更改", i+1) + "'"
		if _, err := p.ParseOneStmt(next, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	if fields.Items[1].Alias != alias || !reflect.DeepEqual(wildcard.Qualifier.Parts, parts) {
		t.Fatal("parser reuse changed lexical data in an earlier AST")
	}
	got := []loc.Span{fields.Span(), fields.Items[0].Span(), fields.Items[1].Span(), wildcard.Span()}
	if !reflect.DeepEqual(got, spans) {
		t.Fatalf("parser reuse changed spans: %v, want %v", got, spans)
	}
	sameItem := reflect.TypeFor[ast.SelectItem]() == reflect.TypeFor[pkast.SelectItem]()
	sameList := reflect.TypeFor[ast.FieldList]() == reflect.TypeFor[pkast.FieldList]()
	sameWildcard := reflect.TypeFor[ast.WildcardExpr]() == reflect.TypeFor[pkast.WildcardExpr]()
	if !sameItem || !sameList || !sameWildcard {
		t.Fatal("projection nodes must share the public concrete type identity")
	}
}

func TestProjectionSpansAcrossQueryAndReturningPaths(t *testing.T) {
	for _, tc := range []struct {
		name, sql string
		items     []string
		get       func(ast.StmtNode) *ast.FieldList
	}{
		{
			name: "select nested", sql: "SELECT (SELECT 1) AS sub, 2 FROM DUAL",
			items: []string{"(SELECT 1) AS sub", "2"},
			get:   func(stmt ast.StmtNode) *ast.FieldList { return stmt.(*ast.SelectStmt).Fields },
		},
		{
			name: "select comments", sql: "SELECT a /* gap */, t.* FROM t",
			items: []string{"a", "t.*"},
			get:   func(stmt ast.StmtNode) *ast.FieldList { return stmt.(*ast.SelectStmt).Fields },
		},
		{
			name: "insert returning", sql: "INSERT INTO t VALUES (1) RETURNING *, 2 AS ''",
			items: []string{"*", "2 AS ''"},
			get:   func(stmt ast.StmtNode) *ast.FieldList { return stmt.(*ast.InsertStmt).Returning },
		},
		{
			name: "delete returning", sql: "DELETE FROM t RETURNING `t.dot`.*, 1 AS 'x.y'",
			items: []string{"`t.dot`.*", "1 AS 'x.y'"},
			get:   func(stmt ast.StmtNode) *ast.FieldList { return stmt.(*ast.DeleteStmt).Returning },
		},
		{
			name: "update returning", sql: "UPDATE t SET a = 1 RETURNING t.*, (2 + 3) AS x",
			items: []string{"t.*", "(2 + 3) AS x"},
			get:   func(stmt ast.StmtNode) *ast.FieldList { return stmt.(*ast.UpdateStmt).Returning },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			stmt, err := New().ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			fields := tc.get(stmt)
			if fields == nil || len(fields.Items) != len(tc.items) {
				t.Fatalf("projection list = %+v, want %d items", fields, len(tc.items))
			}
			wantListSpan := loc.InvalidSpan()
			for i, spelling := range tc.items {
				start := strings.Index(tc.sql, spelling)
				want := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(spelling))}
				if got := fields.Items[i].Span(); got != want {
					t.Fatalf("item %d span = %v, want %v for %q", i, got, want, spelling)
				}
				wantListSpan = loc.Cover(wantListSpan, want)
			}
			if fields.Span() != wantListSpan {
				t.Fatalf("list span = %v, want %v", fields.Span(), wantListSpan)
			}
		})
	}
}

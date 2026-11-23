package myparser

// Table-source, window, CTE and INSERT-alias name/syntax contracts on the
// shared marker types. Source tokens (Raw, Quote, Span) stay independent of
// semantic spans and of Restore output spelling.

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

// locateTableSource walks the statement and returns the MySQL TableSource
// whose span starts at the given source start offset. The offset must be the
// byte offset of the first byte of the FROM-clause source expression.
func locateTableSource(t *testing.T, stmt ast.StmtNode, start loc.ByteOffset) *ast.TableSource {
	t.Helper()
	sink := &tableSourceSink{start: start}
	if _, ok := stmt.Accept(sink); !ok {
		t.Fatalf("locate table source: traversal aborted")
	}
	if sink.found == nil {
		t.Fatalf("no table source starts at offset %d", start)
	}
	return sink.found
}

// findFromStart locates the first source in these single-source fixtures.
func findFromStart(t *testing.T, sql string) int {
	t.Helper()
	fromStart := strings.Index(sql, "FROM ")
	if fromStart < 0 {
		t.Fatalf("no FROM clause in %q", sql)
	}
	return fromStart + len("FROM ")
}

type tableSourceSink struct {
	start loc.ByteOffset
	found *ast.TableSource
}

func (v *tableSourceSink) Enter(n ast.Node) (ast.Node, bool) {
	if v.found == nil {
		if ts, ok := n.(*ast.TableSource); ok {
			span := ts.Span()
			if span.Valid() && span.Start == v.start {
				v.found = ts
			}
		}
	}
	return n, false
}

func (v *tableSourceSink) Leave(n ast.Node) (ast.Node, bool) {
	return n, true
}

var _ ast.Visitor = (*tableSourceSink)(nil)

// TestTableSourceAliasAndSourceKinds covers alias Name/Raw/Quote/Span and
// Source type for plain tables, derived SELECT, set operators, VALUES,
// LATERAL and JSON_TABLE sources.
func TestTableSourceAliasAndSourceKinds(t *testing.T) {
	for _, tc := range []struct {
		name   string
		sql    string
		from   int
		alias  lex.Ident
		source func(ast.ResultSetNode) bool
	}{
		{
			name:  "plain table",
			sql:   "SELECT * FROM `a`",
			from:  findFromStart(t, "SELECT * FROM `a`"),
			alias: lex.Ident{Span: loc.InvalidSpan()},
			source: func(rs ast.ResultSetNode) bool {
				_, ok := rs.(*ast.TableName)
				return ok
			},
		},
		{
			name:  "derived select",
			sql:   "SELECT * FROM (SELECT 1 AS x) AS s",
			from:  findFromStart(t, "SELECT * FROM (SELECT 1 AS x) AS s"),
			alias: lex.Ident{Name: "s", Raw: "s"},
			source: func(rs ast.ResultSetNode) bool {
				_, ok := rs.(*ast.SelectStmt)
				return ok
			},
		},
		{
			name:  "derived set operator",
			sql:   "SELECT * FROM (SELECT 1 UNION SELECT 2) AS u",
			from:  findFromStart(t, "SELECT * FROM (SELECT 1 UNION SELECT 2) AS u"),
			alias: lex.Ident{Name: "u", Raw: "u"},
			source: func(rs ast.ResultSetNode) bool {
				_, ok := rs.(*ast.SetOprStmt)
				return ok
			},
		},
		{
			name:  "derived values",
			sql:   "SELECT * FROM (VALUES ROW(1)) AS v",
			from:  findFromStart(t, "SELECT * FROM (VALUES ROW(1)) AS v"),
			alias: lex.Ident{Name: "v", Raw: "v"},
			source: func(rs ast.ResultSetNode) bool {
				_, ok := rs.(*ast.ValuesStmt)
				return ok
			},
		},
		{
			name:  "lateral select",
			sql:   "SELECT * FROM LATERAL (SELECT 1 AS a) d",
			from:  findFromStart(t, "SELECT * FROM LATERAL (SELECT 1 AS a) d"),
			alias: lex.Ident{Name: "d", Raw: "d"},
			source: func(rs ast.ResultSetNode) bool {
				_, ok := rs.(*ast.SelectStmt)
				return ok
			},
		},
		{
			name:  "json table",
			sql:   "SELECT * FROM JSON_TABLE('[1]', '$' COLUMNS (n INT PATH '$')) AS jt",
			from:  findFromStart(t, "SELECT * FROM JSON_TABLE('[1]', '$' COLUMNS (n INT PATH '$')) AS jt"),
			alias: lex.Ident{Name: "jt", Raw: "jt"},
			source: func(rs ast.ResultSetNode) bool {
				_, ok := rs.(*ast.JSONTable)
				return ok
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := New()
			stmt, err := p.ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			ts := locateTableSource(t, stmt, loc.ByteOffset(tc.from))
			if tc.source != nil && !tc.source(ts.Source) {
				t.Fatalf("source %T does not match expected shape", ts.Source)
			}
			wantAlias := tc.alias
			if wantAlias.Present() {
				start := strings.LastIndex(tc.sql, wantAlias.Raw)
				wantAlias.Span = loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(wantAlias.Raw))}
			}
			if ts.Alias != wantAlias {
				t.Fatalf("alias = %+v, want %+v", ts.Alias, wantAlias)
			}
			if ts.Lateral {
				if tc.name != "lateral select" {
					t.Fatalf("Lateral flag set on non-lateral case %q", tc.name)
				}
			} else if tc.name == "lateral select" {
				t.Fatal("Lateral flag lost")
			}
		})
	}
}

// TestTableSourceSpanCoversSourceToAlias verifies the shared span contract:
// the TableSource span covers its source through its alias or column list.
func TestTableSourceSpanCoversSourceToAlias(t *testing.T) {
	for _, tc := range []struct {
		name string
		sql  string
	}{
		{name: "derived with alias", sql: "SELECT * FROM (SELECT 1) AS s"},
		{name: "plain table", sql: "SELECT * FROM t"},
		{name: "plain with alias", sql: "SELECT * FROM t AS x"},
		{name: "derived with columns", sql: "SELECT * FROM (SELECT 1) t (a, b)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := New()
			stmt, err := p.ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			start := loc.ByteOffset(findFromStart(t, tc.sql))
			ts := locateTableSource(t, stmt, start)
			want := loc.Span{Start: start, End: loc.ByteOffset(len(tc.sql))}
			if got := ts.Span(); got != want {
				t.Fatalf("span = %v, want %v", got, want)
			}
		})
	}
}

// TestTableSourceAliasQuotedForms covers quoted aliases, ANSI_QUOTES mode,
// missing aliases, and the presence contract of a quoted empty alias.
func TestTableSourceAliasQuotedForms(t *testing.T) {
	for _, tc := range []struct {
		name   string
		sql    string
		mode   mysql.SQLMode
		alias  lex.Ident
		reject bool
	}{
		{name: "quoted dotted", sql: "SELECT * FROM `库.表` AS `别.名`",
			alias: lex.Ident{Name: "别.名", Raw: "`别.名`", Quote: lex.QuoteBacktick}},
		{name: "doubled quote", sql: `SELECT * FROM t AS "a""b.c"`, mode: mysql.ModeANSIQuotes,
			alias: lex.Ident{Name: `a"b.c`, Raw: `"a""b.c"`, Quote: lex.QuoteDouble}},
		{name: "double quote needs ansi", sql: `SELECT * FROM t AS "a""b.c"`, reject: true},
		{name: "ansi unicode", sql: `SELECT * FROM t AS "表""名"`, mode: mysql.ModeANSIQuotes,
			alias: lex.Ident{Name: `表"名`, Raw: `"表""名"`, Quote: lex.QuoteDouble}},
		{name: "unicode single", sql: "SELECT * FROM t AS '列''别'", reject: true},
		{name: "absent", sql: "SELECT * FROM t",
			alias: lex.Ident{Span: loc.InvalidSpan()}},
		{name: "quoted empty present", sql: "SELECT * FROM t AS ``",
			alias: lex.Ident{Name: "", Raw: "``", Quote: lex.QuoteBacktick}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := New()
			p.SetSQLMode(tc.mode)
			stmt, err := p.ParseOneStmt(tc.sql, "", "")
			if tc.reject {
				if err == nil {
					t.Fatal("string literal accepted as a table alias")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			fromStart := findFromStart(t, tc.sql)
			ts := locateTableSource(t, stmt, loc.ByteOffset(fromStart))
			alias := ts.Alias
			wantAlias := tc.alias
			if wantAlias.Present() {
				start := strings.LastIndex(tc.sql, wantAlias.Raw)
				wantAlias.Span = loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(wantAlias.Raw))}
			}
			if alias != wantAlias {
				t.Fatalf("alias = %+v, want %+v", alias, wantAlias)
			}
			if alias.Present() != tc.alias.Present() {
				t.Fatalf("alias presence = %v, want %v", alias.Present(), tc.alias.Present())
			}
			if tc.alias.Raw == "" {
				if alias.Span != loc.InvalidSpan() {
					t.Fatalf("absent alias span = %v, want unknown", alias.Span)
				}
				return
			}
			start := strings.Index(tc.sql, tc.alias.Raw)
			want := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(tc.alias.Raw))}
			if alias.Span != want {
				t.Fatalf("alias span = %v, want %v", alias.Span, want)
			}
		})
	}
}

// TestTableSourceRestoreRoundtrip verifies that stmt.Restore produces a
// legal full SELECT and that its output spelling follows the quote policy
// rather than Raw.
func TestTableSourceRestoreRoundtrip(t *testing.T) {
	for _, tc := range []struct {
		name string
		sql  string
		want string
	}{
		{name: "plain with alias", sql: "SELECT * FROM t AS x",
			want: "SELECT * FROM `t` AS `x`"},
		{name: "derived with alias", sql: "SELECT * FROM (SELECT 1) s",
			want: "SELECT * FROM (SELECT 1) AS `s`"},
		{name: "derived with columns", sql: "SELECT * FROM (SELECT 1 AS a, 2 AS b) t (c, d)",
			want: "SELECT * FROM (SELECT 1 AS `a`,2 AS `b`) AS `t` (`c`, `d`)"},
		{name: "json table", sql: "SELECT * FROM JSON_TABLE('[1]', '$' COLUMNS (n INT PATH '$')) jt",
			want: "SELECT * FROM JSON_TABLE('[1]', '$' COLUMNS (`n` INT PATH '$')) AS `jt`"},
		{name: "lateral", sql: "SELECT * FROM LATERAL (SELECT 1 AS a) d",
			want: "SELECT * FROM LATERAL (SELECT 1 AS `a`) AS `d`"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := New()
			stmt, err := p.ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			var out strings.Builder
			if err := stmt.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); err != nil {
				t.Fatalf("restore: %v", err)
			}
			if got := out.String(); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
			// Re-parse to confirm the restored output is syntactically valid.
			if _, err := p.ParseOneStmt(out.String(), "", ""); err != nil {
				t.Fatalf("restored output is not parseable: %v", err)
			}
		})
	}
}

// TestCTEAndWindowNameSpans covers CTE Name/Columns/CycleColumns,
// WindowSpec Name/Ref token spans, and shared type identity.
func TestCTEAndWindowNameSpans(t *testing.T) {
	t.Parallel()
	p := New()
	p.EnableWindowFunc(true)

	// CTE: name span, column list spans, query is a shared result set.
	sql := "WITH c (x, y) AS (SELECT 1) SELECT * FROM c"
	stmt, err := p.ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}
	with := stmt.(*ast.SelectStmt).With.(*ast.WithClause)
	if with == nil {
		t.Fatalf("With clause = nil")
	}
	core := with.CTEs[0].CTECore()
	if core.Name.Raw != "c" {
		t.Fatalf("CTE name raw = %q, want c", core.Name.Raw)
	}
	if core.Name.Span != (loc.Span{Start: 5, End: 6}) {
		t.Fatalf("CTE name span = %v, want [5:6]", core.Name.Span)
	}
	if len(core.Columns) != 2 {
		t.Fatalf("CTE columns = %+v, want two names", core.Columns)
	}
	for i, col := range core.Columns {
		start := strings.Index(sql, col.Raw)
		wantCol := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(col.Raw))}
		if col.Span != wantCol {
			t.Fatalf("CTE column %d span = %v, want %v", i, col.Span, wantCol)
		}
	}
	if query, ok := core.Query.(*ast.SelectStmt); !ok || query == nil {
		t.Fatalf("CTE query = %T, want a non-nil SELECT", core.Query)
	}

	// A recursive CTE may omit CYCLE or retain its quoted column tokens.
	cycleSQL := "WITH RECURSIVE c (x) AS (SELECT 1 UNION ALL SELECT 2) SELECT * FROM c"
	rstmt, err := p.ParseOneStmt(cycleSQL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	rwith := rstmt.(*ast.SelectStmt).With.(*ast.WithClause)
	if !rwith.IsRecursive {
		t.Fatalf("recursive flag lost: %+v", rwith)
	}
	// Non-cycle recursive CTE has empty cycle columns.
	if len(rwith.CycleColumns) != 0 {
		t.Fatalf("non-cycle recursive CTE has cycle columns: %+v", rwith.CycleColumns)
	}
	cycleSQL = "WITH RECURSIVE `公``共` (`列.一`, y) AS (SELECT 1, 2) CYCLE `列.一`, y RESTRICT SELECT * FROM `公``共`"
	rstmt, err = p.ParseOneStmt(cycleSQL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	rwith = rstmt.(*ast.SelectStmt).With.(*ast.WithClause)
	assertSourceIdent(t, cycleSQL, rwith.CTEs[0].CTECore().Name, "公`共", "`公``共`", lex.QuoteBacktick, false)
	for listIndex, names := range [][]lex.Ident{rwith.CTEs[0].CTECore().Columns, rwith.CycleColumns} {
		if len(names) != 2 {
			t.Fatalf("CTE/CYCLE columns = %+v, want two", names)
		}
		for i, raw := range []string{"`列.一`", "y"} {
			name, quote := "列.一", lex.QuoteBacktick
			if i == 1 {
				name, quote = "y", lex.QuoteUnquoted
			}
			assertSourceIdent(t, cycleSQL, names[i], name, raw, quote, listIndex == 1)
		}
	}

	// Window spec: name token span and ref token span.
	windowSQL := "SELECT ROW_NUMBER() OVER w FROM t WINDOW w AS (ORDER BY a)"
	wstmt, err := p.ParseOneStmt(windowSQL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	selectStmt := wstmt.(*ast.SelectStmt)
	if len(selectStmt.WindowSpecs) != 1 {
		t.Fatalf("window specs = %+v, want one", selectStmt.WindowSpecs)
	}
	spec := selectStmt.WindowSpecs[0].(*ast.WindowSpec)
	// Use LastIndex to find the WINDOW clause (not the OVER w reference).
	winStart := strings.LastIndex(windowSQL, "WINDOW ")
	winEnd := winStart + len("WINDOW ")
	if spec.Name.Raw != "w" || spec.Name.Span != (loc.Span{Start: loc.ByteOffset(winEnd), End: loc.ByteOffset(winEnd + 1)}) {
		t.Fatalf("window name = %+v, want raw w at [%d:%d]", spec.Name, winEnd, winEnd+1)
	}
	if spec.Ref.Present() {
		t.Fatalf("window ref = %+v, want absent for a definition", spec.Ref)
	}
	// Sources share a core, not their concrete dialect identity.
	if reflect.TypeFor[ast.TableSource]() == reflect.TypeFor[pkast.TableSource]() {
		t.Fatal("MySQL TableSource must retain its own outer identity")
	}
	source := &ast.TableSource{}
	var role pkast.TableSourceNode = source
	if role.TableSourceCore() != &source.BaseTableSource {
		t.Fatal("TableSource role did not return the original shared core")
	}
	if reflect.TypeFor[ast.SelectItem]() != reflect.TypeFor[pkast.SelectItem]() {
		t.Fatalf("SelectItem identity differs: %T vs %T", ast.SelectItem{}, pkast.SelectItem{})
	}
}

// TestWindowSpecRefOnlyAlias covers the OVER w form where Ref is present
// and OnlyAlias is true, and the OVER(w) form where Ref is a separate ident.
func TestWindowSpecRefOnlyAlias(t *testing.T) {
	t.Parallel()
	p := New()
	p.EnableWindowFunc(true)

	// OVER w: the ref is the window name itself, OnlyAlias=true.
	sql := "SELECT ROW_NUMBER() OVER w FROM t WINDOW w AS (ORDER BY a)"
	stmt, err := p.ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}
	// The OVER spec is carried by the WindowFuncExpr, not by WindowSpecs.
	wfe, ok := stmt.(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.WindowFuncExpr)
	if !ok {
		t.Fatalf("Fields.Items[0].Expr is %T, want *ast.WindowFuncExpr", stmt.(*ast.SelectStmt).Fields.Items[0].Expr)
	}
	if !wfe.Spec.OnlyAlias {
		t.Fatalf("OnlyAlias = false, want true for OVER w form")
	}
	assertSourceIdent(t, sql, wfe.Spec.Name, "w", "w", lex.QuoteUnquoted, false)
	if wfe.Spec.Ref.Present() || wfe.Spec.Ref.Span != loc.InvalidSpan() {
		t.Fatalf("bare OVER name has a separate ref: %+v", wfe.Spec.Ref)
	}

	// OVER(w): the ref is a separate ident in parentheses, OnlyAlias=false.
	parenSQL := "SELECT ROW_NUMBER() OVER (w) FROM t WINDOW w AS (ORDER BY a)"
	pstmt, err := p.ParseOneStmt(parenSQL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	pWfe, ok := pstmt.(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.WindowFuncExpr)
	if !ok {
		t.Fatalf("Fields.Items[0].Expr is %T, want *ast.WindowFuncExpr", pstmt.(*ast.SelectStmt).Fields.Items[0].Expr)
	}
	if pWfe.Spec.OnlyAlias {
		t.Fatalf("OnlyAlias = true, want false for OVER(w) form")
	}
	assertSourceIdent(t, parenSQL, pWfe.Spec.Ref, "w", "w", lex.QuoteUnquoted, false)
	if pWfe.Spec.Name.Present() || pWfe.Spec.Name.Span != loc.InvalidSpan() {
		t.Fatalf("parenthesized OVER reference has a definition name: %+v", pWfe.Spec.Name)
	}
}

func assertSourceIdent(t *testing.T, sql string, got lex.Ident, name, raw string, quote lex.QuoteStyle, last bool) {
	t.Helper()
	start := strings.Index(sql, raw)
	if last {
		start = strings.LastIndex(sql, raw)
	}
	if start < 0 {
		t.Fatalf("fixture has no token %q", raw)
	}
	want := lex.Ident{Name: name, Raw: raw, Quote: quote, Span: loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(raw))}}
	if got != want {
		t.Fatalf("identifier = %+v, want %+v", got, want)
	}
}

// TestInsertRowAliasNameAndSpan covers the INSERT row alias Name, AS span,
// and the row-alias column list qualifier preservation.
func TestInsertRowAliasNameAndSpan(t *testing.T) {
	t.Parallel()
	p := New()
	sql := "INSERT INTO t1 VALUES (1) AS new ON DUPLICATE KEY UPDATE b = new.b"
	stmt, err := p.ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}
	ins := stmt.(*ast.InsertStmt)
	if ins.RowAlias == nil {
		t.Fatalf("RowAlias = nil, want the INSERT row alias")
	}
	alias := ins.RowAlias
	if alias.Name.Raw != "new" {
		t.Fatalf("alias name raw = %q, want new", alias.Name.Raw)
	}
	wantNameSpan := loc.Span{Start: loc.ByteOffset(strings.Index(sql, "new")), End: loc.ByteOffset(strings.Index(sql, "new") + 3)}
	if alias.Name.Span != wantNameSpan {
		t.Fatalf("alias name span = %v, want %v", alias.Name.Span, wantNameSpan)
	}
	// The row alias span covers AS through the name.
	wantASSpan := loc.Span{Start: loc.ByteOffset(strings.Index(sql, "AS")), End: loc.ByteOffset(strings.Index(sql, "AS") + 6)}
	if alias.Span != wantASSpan {
		t.Fatalf("alias span = %v, want %v (AS through name)", alias.Span, wantASSpan)
	}

	// Row-alias column list: qualified names keep their parts and spans.
	qualifiedSQL := "INSERT INTO t1 VALUES (1) AS new (a, db.b, `c`.`d`)"
	qstmt, err := p.ParseOneStmt(qualifiedSQL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	qins := qstmt.(*ast.InsertStmt)
	if qins.RowAlias == nil {
		t.Fatalf("qualified RowAlias = nil")
	}
	if len(qins.RowAlias.Columns) != 3 {
		t.Fatalf("row alias columns = %+v, want three", qins.RowAlias.Columns)
	}
	for i, rawParts := range [][]string{{"a"}, {"db", "b"}, {"`c`", "`d`"}} {
		col := qins.RowAlias.Columns[i]
		if len(col.Parts) != len(rawParts) {
			t.Fatalf("row alias column %d = %+v, want %d parts", i, col, len(rawParts))
		}
		for j, raw := range rawParts {
			name, quote := raw, lex.QuoteUnquoted
			if strings.HasPrefix(raw, "`") {
				name, quote = strings.Trim(raw, "`"), lex.QuoteBacktick
			}
			assertSourceIdent(t, qualifiedSQL, col.Parts[j], name, raw, quote, true)
		}
		start := strings.Index(qualifiedSQL, strings.Join(rawParts, "."))
		if want := (loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(strings.Join(rawParts, ".")))}); col.Span != want {
			t.Fatalf("column %d span = %v, want %v", i, col.Span, want)
		}
	}
	if want := (loc.Span{Start: loc.ByteOffset(strings.Index(qualifiedSQL, "AS")), End: loc.ByteOffset(len(qualifiedSQL))}); qins.RowAlias.Span != want {
		t.Fatalf("row alias clause span = %v, want %v", qins.RowAlias.Span, want)
	}

	// MariaDB strict diagnostic must point at AS, not at the alias name.
	issues := p.CheckDialectCompatibility(stmt, sql, DialectMariaDB)
	if len(issues) != 1 || issues[0].Code != IssueMariaDBInsertRowAlias {
		t.Fatalf("MariaDB issues = %#v, want one row-alias issue", issues)
	}
	if issues[0].Highlight != "AS" {
		t.Fatalf("issue highlight = %q, want AS", issues[0].Highlight)
	}
	// MySQL must not flag the same statement.
	if mysqlIssues := p.CheckDialectCompatibility(stmt, sql, DialectMySQL); len(mysqlIssues) != 0 {
		t.Fatalf("MySQL issues = %#v, want none", mysqlIssues)
	}
}

// TestSharedColumnTypeCoversViewAndAnalyze verifies the shared column name
// list type is used consistently by View, AlterView and Analyze statements.
func TestSharedColumnTypeCoversViewAndAnalyze(t *testing.T) {
	t.Parallel()
	p := New()
	viewSQL := "CREATE OR REPLACE VIEW v (a, b) AS SELECT 1 AS a, 2 AS b"
	vstmt, err := p.ParseOneStmt(viewSQL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	createView := vstmt.(*ast.CreateViewStmt)
	if len(createView.Columns) != 2 {
		t.Fatalf("view columns = %+v, want two", createView.Columns)
	}
	alterSQL := "ALTER VIEW v (a, b) AS SELECT 1 AS a, 2 AS b"
	a2stmt, err := p.ParseOneStmt(alterSQL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	alterView := a2stmt.(*ast.AlterViewStmt)
	if len(alterView.Columns) != 2 {
		t.Fatalf("alter view columns = %+v, want two", alterView.Columns)
	}
	analyzeSQL := "ANALYZE TABLE t, u PERSISTENT FOR COLUMNS (a, b) INDEXES ALL"
	astmt, err := p.ParseOneStmt(analyzeSQL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	analyze := astmt.(*ast.AnalyzeTableStmt)
	if len(analyze.Columns) != 2 {
		t.Fatalf("analyze columns = %+v, want two", analyze.Columns)
	}
	// Shared type identity: all three are []lex.Ident.
	for _, pair := range []struct {
		name string
		got  reflect.Type
		want reflect.Type
	}{
		{"CreateViewStmt.Columns", reflect.TypeOf(createView.Columns), reflect.TypeFor[[]lex.Ident]()},
		{"AlterViewStmt.Columns", reflect.TypeOf(alterView.Columns), reflect.TypeFor[[]lex.Ident]()},
		{"AnalyzeTableStmt.Columns", reflect.TypeOf(analyze.Columns), reflect.TypeFor[[]lex.Ident]()},
	} {
		if pair.got != pair.want {
			t.Fatalf("%s = %v, want %v", pair.name, pair.got, pair.want)
		}
	}
}

// TestParserReuseKeepsASTDataStable verifies that consecutive Parse calls
// on the same Parser instance do not mutate earlier ASTs: names, raws and
// spans survive reuse, including strings that required quote decoding.
func TestParserReuseKeepsASTDataStable(t *testing.T) {
	p := New()
	p.EnableWindowFunc(true)
	firstSQL := "SELECT * FROM `库`.`表``名` AS `别`"
	stmt, err := p.ParseOneStmt(firstSQL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	fromStart := findFromStart(t, firstSQL)
	ts := locateTableSource(t, stmt, loc.ByteOffset(fromStart))
	originalAlias := ts.Alias
	originalAlias.Name, originalAlias.Raw = strings.Clone(originalAlias.Name), strings.Clone(originalAlias.Raw)
	originalColumns := ts.Columns
	originalColumns = append([]lex.Ident(nil), originalColumns...)
	for i := range originalColumns {
		originalColumns[i].Name, originalColumns[i].Raw = strings.Clone(originalColumns[i].Name), strings.Clone(originalColumns[i].Raw)
	}
	originalSpan := ts.Span()
	originalSQL := restoreStmt(t, stmt)
	columnSQL := "SELECT * FROM (SELECT 1,2) AS `别``名` (`列.一`,`列``二`)"
	columnStmt, err := p.ParseOneStmt(columnSQL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	columnSource := locateTableSource(t, columnStmt, loc.ByteOffset(findFromStart(t, columnSQL)))
	wantColumns := []lex.Ident{
		{Name: "列.一", Raw: "`列.一`", Quote: lex.QuoteBacktick},
		{Name: "列`二", Raw: "`列``二`", Quote: lex.QuoteBacktick},
	}
	for i := range wantColumns {
		start := strings.Index(columnSQL, wantColumns[i].Raw)
		wantColumns[i].Span = loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(wantColumns[i].Raw))}
	}
	columnRestored := restoreStmt(t, columnStmt)
	// Multi-statement parse to exercise the statement-list path.
	multiSQL := "SELECT * FROM t; SELECT * FROM u"
	stmts, _, err := p.Parse(multiSQL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 2 {
		t.Fatalf("multi parse returned %d statements, want 2", len(stmts))
	}
	// Re-run with a different source to force buffer reuse.
	otherSQL := "SELECT * FROM `不 同`.`变 量` AS x"
	if _, err := p.ParseOneStmt(otherSQL, "", ""); err != nil {
		t.Fatal(err)
	}
	// The original AST must be unchanged.
	if ts.Alias != originalAlias {
		t.Fatalf("alias changed after parser reuse: got %+v, want %+v", ts.Alias, originalAlias)
	}
	if !reflect.DeepEqual(ts.Columns, originalColumns) {
		t.Fatalf("columns changed after parser reuse: got %+v, want %+v", ts.Columns, originalColumns)
	}
	if ts.Span() != originalSpan {
		t.Fatalf("span changed after parser reuse: got %v, want %v", ts.Span(), originalSpan)
	}
	if !reflect.DeepEqual(columnSource.Columns, wantColumns) {
		t.Fatalf("column aliases changed after reuse: %+v, want %+v", columnSource.Columns, wantColumns)
	}
	assertSourceIdent(t, columnSQL, columnSource.Alias, "别`名", "`别``名`", lex.QuoteBacktick, false)
	if got := restoreStmt(t, stmt); got != originalSQL {
		t.Fatalf("table names changed after reuse: %q -> %q", originalSQL, got)
	}
	if got := restoreStmt(t, columnStmt); got != columnRestored {
		t.Fatalf("derived source changed after reuse: %q -> %q", columnRestored, got)
	}
}

// TestParameterIndexStableAcrossParses verifies that parameter markers keep
// their zero-based Index across consecutive Parse calls.
func TestParameterIndexStableAcrossParses(t *testing.T) {
	p := New()
	sql := "SELECT ?, SUM(a) OVER (ORDER BY a ROWS BETWEEN ? PRECEDING AND ? FOLLOWING) LIMIT ?"
	p.EnableWindowFunc(true)
	stmt, err := p.ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}
	var params []*ast.ParamMarkerExpr
	stmt.Accept(parameterCollector{params: &params})
	if len(params) != 4 {
		t.Fatalf("got %d parameters, want 4", len(params))
	}
	for i, param := range params {
		if param.Index != i {
			t.Fatalf("parameter %d index = %d, want %d", i, param.Index, i)
		}
	}
	// Second parse on the same parser must not shift the first parse's indices.
	if _, err := p.ParseOneStmt("SELECT ?", "", ""); err != nil {
		t.Fatal(err)
	}
	for i, param := range params {
		if param.Index != i {
			t.Fatalf("after reuse, parameter %d index = %d, want %d", i, param.Index, i)
		}
	}
}

// TestTableSourceSourceRejectionByVisitor verifies that a TableSource with a
// source that is not a ResultSetNode is rejected by the shared Visitor.
func TestTableSourceSourceRejectionByVisitor(t *testing.T) {
	t.Parallel()
	p := New()
	sql := "SELECT * FROM t"
	fromStart := findFromStart(t, sql)
	stmt, err := p.ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}
	ts := locateTableSource(t, stmt, loc.ByteOffset(fromStart))
	originalSource := ts.Source
	// Replace the source with a non-ResultSetNode expression via Leave.
	literal := &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "1"}}
	rejector := &tableSourceSourceRejector{replacement: literal}
	_, ok := ts.Accept(rejector)
	// The shared TableSource.Accept rejects a non-ResultSet replacement.
	if ok {
		t.Fatalf("visitor accepted a non-ResultSet source replacement")
	}
	// The original source (TableName) is still a ResultSetNode; the
	// replacement was not applied.
	if ts.Source != originalSource {
		t.Fatalf("rejected replacement changed source: %T -> %T", originalSource, ts.Source)
	}
}

// tableSourceSourceRejector replaces the TableSource source with a
// non-ResultSetNode expression in Leave, which the shared Visitor must reject.
type tableSourceSourceRejector struct {
	replacement ast.ExprNode
}

func (v *tableSourceSourceRejector) Enter(n ast.Node) (ast.Node, bool) {
	return n, false
}

func (v *tableSourceSourceRejector) Leave(n ast.Node) (ast.Node, bool) {
	// Replace the source with a plain expression (not a ResultSetNode).
	// The shared TableSource.Accept must reject this and keep the old source.
	if _, isSource := n.(*ast.TableName); isSource {
		return v.replacement, true
	}
	return n, true
}

var _ ast.Visitor = (*tableSourceSourceRejector)(nil)

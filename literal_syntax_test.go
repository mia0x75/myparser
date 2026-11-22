package myparser

import (
	"strings"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/myparser/ast"
)

// literalSyntax is the pure-syntax shape of a literal expression.
type literalSyntax struct {
	kind  pkast.LiteralKind
	value string
	raw   string
	span  loc.Span
}

// nullBooleanSyntax describes the literals of "SELECT nUlL,TrUe,FALSE".
func nullBooleanSyntax() []literalSyntax {
	return []literalSyntax{
		{kind: pkast.LiteralNull, value: "null", raw: "nUlL", span: loc.Span{Start: 7, End: 11}},
		{kind: pkast.LiteralBoolean, value: "true", raw: "TrUe", span: loc.Span{Start: 12, End: 16}},
		{kind: pkast.LiteralBoolean, value: "false", raw: "FALSE", span: loc.Span{Start: 17, End: 22}},
	}
}

// literalFields returns the literal expression of every select field.
func literalFields(t *testing.T, stmt ast.StmtNode) []*ast.LiteralExpr {
	t.Helper()
	sel, ok := stmt.(*ast.SelectStmt)
	if !ok {
		t.Fatalf("statement is %T, want *ast.SelectStmt", stmt)
	}
	literals := make([]*ast.LiteralExpr, 0, len(sel.Fields.Items))
	for _, field := range sel.Fields.Items {
		literal, ok := field.Expr.(*ast.LiteralExpr)
		if !ok {
			t.Fatalf("field expression is %T, want *ast.LiteralExpr", field.Expr)
		}
		literals = append(literals, literal)
	}
	return literals
}

// checkLiteralSyntax compares the literals with their expected shape.
func checkLiteralSyntax(t *testing.T, literals []*ast.LiteralExpr, want []literalSyntax) {
	t.Helper()
	if len(literals) != len(want) {
		t.Fatalf("got %d literals, want %d", len(literals), len(want))
	}
	for i, literal := range literals {
		got := literalSyntax{kind: literal.Kind, value: literal.Value, raw: literal.Raw, span: literal.Span()}
		if got != want[i] {
			t.Errorf("literal %d = %+v, want %+v", i, got, want[i])
		}
	}
}

type literalStringSyntax struct {
	literalSyntax
	introducer string
	quote      lex.QuoteStyle
}

// literalShapes converts literal expressions to plain shapes for comparison.
func literalShapes(literals []*ast.LiteralExpr) []literalSyntax {
	shape := make([]literalSyntax, 0, len(literals))
	for _, literal := range literals {
		shape = append(shape, literalSyntax{kind: literal.Kind, value: literal.Value, raw: literal.Raw, span: literal.Span()})
	}
	return shape
}

// A NULL or boolean literal keeps its kind, its canonical value, the original
// spelling, and the byte range it was written at.
func TestLiteralSyntaxNullBoolean(t *testing.T) {
	const sql = "SELECT nUlL,TrUe,FALSE"
	checkLiteralSyntax(t, literalFields(t, mustParseOne(t, sql)), nullBooleanSyntax())
}

// A later Parse on the same Parser must not disturb the literals of an
// earlier statement.
func TestLiteralSyntaxParserReuse(t *testing.T) {
	const sql = "SELECT nUlL,TrUe,FALSE"
	parser := New()
	stmt, err := parser.ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}
	literals := literalFields(t, stmt)
	if _, err := parser.ParseOneStmt("SELECT 1 FROM t", "", ""); err != nil {
		t.Fatal(err)
	}
	checkLiteralSyntax(t, literals, nullBooleanSyntax())
}

// The span of a literal counts the bytes of the text before it, not the runes.
func TestLiteralSyntaxSpanAfterMultiByteText(t *testing.T) {
	const sql = "SELECT '中文', TrUe"
	stmt := mustParseOne(t, sql).(*ast.SelectStmt)
	literal, ok := stmt.Fields.Items[1].Expr.(*ast.LiteralExpr)
	if !ok {
		t.Fatalf("field expression is %T, want *ast.LiteralExpr", stmt.Fields.Items[1].Expr)
	}
	if literal.Kind != pkast.LiteralBoolean || literal.Value != "true" || literal.Raw != "TrUe" {
		t.Fatalf("literal = %v %q %q, want LiteralBoolean true TrUe", literal.Kind, literal.Value, literal.Raw)
	}
	// '中文' takes six bytes for two runes, so TrUe starts at byte 17.
	want := loc.Span{Start: 17, End: 21}
	if literal.Span() != want {
		t.Fatalf("span = %v, want %v", literal.Span(), want)
	}
	if got := sql[literal.Span().Start:literal.Span().End]; got != "TrUe" {
		t.Fatalf("span slice = %q, want %q", got, "TrUe")
	}
}

// ORDER BY keeps the legacy position meaning of a boolean: TRUE is 1 and
// FALSE is 0.
func TestLiteralSyntaxOrderByBooleanPosition(t *testing.T) {
	for _, tt := range []struct {
		sql  string
		want int
	}{
		{"SELECT a FROM t ORDER BY TRUE", 1},
		{"SELECT a FROM t ORDER BY FALSE", 0},
		{"SELECT a FROM t ORDER BY true", 1},
		{"SELECT a FROM t ORDER BY false", 0},
	} {
		t.Run(tt.sql, func(t *testing.T) {
			stmt := mustParseOne(t, tt.sql).(*ast.SelectStmt)
			orderItem := stmt.OrderBy.OrderByClauseCore().Items[0]
			position, ok := orderItem.OrderItemCore().Expr.(*ast.PositionExpr)
			if !ok {
				t.Fatalf("order by expression is %T, want *ast.PositionExpr", orderItem.OrderItemCore().Expr)
			}
			if position.N != tt.want {
				t.Fatalf("position = %d, want %d", position.N, tt.want)
			}
		})
	}
}

// NOW keeps accepting a boolean argument: TRUE and FALSE used to reach the
// precision check as the integers 1 and 0, so the boolean keyword must not
// become an error now that it is no longer a value expression.
func TestLiteralSyntaxNowBooleanArgument(t *testing.T) {
	for _, tt := range []struct {
		sql   string
		value string
	}{
		{"SELECT NOW(TRUE)", "true"},
		{"SELECT NOW(FALSE)", "false"},
		{"SELECT NOW(true)", "true"},
	} {
		t.Run(tt.sql, func(t *testing.T) {
			stmt := mustParseOne(t, tt.sql).(*ast.SelectStmt)
			call, ok := stmt.Fields.Items[0].Expr.(*ast.FuncCallExpr)
			if !ok {
				t.Fatalf("field expression is %T, want *ast.FuncCallExpr", stmt.Fields.Items[0].Expr)
			}
			literal, ok := call.Args[0].(*ast.LiteralExpr)
			if !ok {
				t.Fatalf("argument is %T, want *ast.LiteralExpr", call.Args[0])
			}
			if literal.Kind != pkast.LiteralBoolean || literal.Value != tt.value {
				t.Fatalf("argument = %v %q, want LiteralBoolean %q", literal.Kind, literal.Value, tt.value)
			}
		})
	}
	// A NULL argument is still not a precision.
	_, err := New().ParseOneStmt("SELECT NOW(NULL)", "", "")
	if err == nil {
		t.Fatal("accepted NULL as the NOW precision")
	}
	if !strings.Contains(err.Error(), "NOW precision must be a non-negative integer") {
		t.Fatalf("error = %v, want the NOW precision message", err)
	}
}

// KILL keeps the connection id of a boolean argument.
func TestLiteralSyntaxKillBooleanConnectionID(t *testing.T) {
	for _, tt := range []struct {
		sql  string
		want uint64
	}{
		{"KILL TRUE", 1},
		{"KILL FALSE", 0},
	} {
		t.Run(tt.sql, func(t *testing.T) {
			stmt := mustParseOne(t, tt.sql).(*ast.KillStmt)
			if stmt.ConnectionID != tt.want {
				t.Fatalf("connection id = %d, want %d", stmt.ConnectionID, tt.want)
			}
			if stmt.Expr != nil {
				t.Fatalf("expression = %T, want nil", stmt.Expr)
			}
		})
	}
}

// Every shape that carries a NULL or boolean literal restores, parses again,
// and restores to the same SQL.
func TestLiteralSyntaxRestoreRoundTrip(t *testing.T) {
	for _, tt := range []struct {
		sql     string
		literal string
	}{
		{"CREATE TABLE t (a INT DEFAULT TRUE)", "DEFAULT TRUE"},
		{"CREATE TABLE t (a INT DEFAULT FALSE)", "DEFAULT FALSE"},
		{"CREATE TABLE t (a INT DEFAULT NULL)", "DEFAULT NULL"},
		{"SELECT * FROM JSON_TABLE('[]', '$[*]' COLUMNS (id INT PATH '$.id' DEFAULT 1 ON EMPTY)) AS jt", "DEFAULT 1"},
		{"SELECT * FROM JSON_TABLE('[]', '$[*]' COLUMNS (id INT PATH '$.id' DEFAULT NULL ON EMPTY)) AS jt", "DEFAULT NULL"},
		{"SELECT * FROM JSON_TABLE('[]', '$[*]' COLUMNS (id INT PATH '$.id' DEFAULT TRUE ON EMPTY)) AS jt", "DEFAULT TRUE"},
		{"SELECT CHAR(65)", "CHAR"},
		{"SELECT TRIM(' x ')", "TRIM"},
		{"SELECT TRIM('x' FROM ' xx ')", "TRIM"},
		{"SELECT TRIM(BOTH FROM ' xx ')", "TRIM"},
		{"SELECT TRIM(LEADING FROM ' xx ')", "TRIM"},
		{"SELECT TRIM(TRAILING FROM ' xx ')", "TRIM"},
		{"SELECT TRIM(BOTH 'x' FROM ' xx ')", "TRIM"},
		{"SELECT TRIM(LEADING 'x' FROM ' xx ')", "TRIM"},
		{"SELECT TRIM(TRAILING 'x' FROM ' xx ')", "TRIM"},
		{"SELECT TRIM(NULL)", "NULL"},
		{"SELECT TRIM('x' FROM NULL)", "NULL"},
		{"SELECT TRIM(BOTH NULL FROM ' xx ')", "TRIM(BOTH FROM"},
	} {
		t.Run(tt.sql, func(t *testing.T) {
			restored := restoreStmt(t, mustParseOne(t, tt.sql))
			if !strings.Contains(restored, tt.literal) {
				t.Fatalf("restored %q, want it to contain %q", restored, tt.literal)
			}
			restoredAgain := restoreStmt(t, mustParseOne(t, restored))
			if restored != restoredAgain {
				t.Fatalf("re-restored %q, want %q", restoredAgain, restored)
			}
		})
	}
}

// The replication options keep their boolean and NULL spelling.
func TestLiteralSyntaxReplicationValues(t *testing.T) {
	for _, tt := range []struct {
		sql   string
		kind  pkast.LiteralKind
		value string
	}{
		{"CHANGE MASTER TO MASTER_SSL=TRUE", pkast.LiteralBoolean, "true"},
		{"CHANGE MASTER TO MASTER_SSL_VERIFY_SERVER_CERT=FALSE", pkast.LiteralBoolean, "false"},
		{"CHANGE MASTER TO MASTER_TLS_CIPHERSUITES=NULL", pkast.LiteralNull, "null"},
	} {
		t.Run(tt.sql, func(t *testing.T) {
			stmt := mustParseOne(t, tt.sql).(*ast.ChangeMasterStmt)
			literal, ok := stmt.Options[0].Expr.(*ast.LiteralExpr)
			if !ok {
				t.Fatalf("option expression is %T, want *ast.LiteralExpr", stmt.Options[0].Expr)
			}
			if literal.Kind != tt.kind || literal.Value != tt.value {
				t.Fatalf("option = %v %q, want %v %q", literal.Kind, literal.Value, tt.kind, tt.value)
			}
			if restored := restoreStmt(t, stmt); restored != tt.sql {
				t.Fatalf("restored %q, want %q", restored, tt.sql)
			}
		})
	}
	// A numeric replication option still rejects a boolean.
	if _, err := New().ParseOneStmt("change master to master_port=true", "", ""); err == nil {
		t.Fatal("accepted a boolean for a numeric replication option")
	}
}

func TestLiteralSyntaxStringEntryPoints(t *testing.T) {
	for _, tt := range []struct {
		name    string
		sql     string
		extract func(ast.StmtNode) []ast.ExprNode
		want    literalStringSyntax
	}{
		{
			name: "comment",
			sql:  "CREATE TABLE t (a INT COMMENT 'hello')",
			extract: func(s ast.StmtNode) []ast.ExprNode {
				ct := s.(*ast.CreateTableStmt)
				for _, opt := range ct.Columns[0].(*ast.ColumnDef).Options {
					if opt.Tp == ast.ColumnOptionComment {
						return []ast.ExprNode{opt.Expr}
					}
				}
				return nil
			},
			want: literalStringSyntax{
				literalSyntax: literalSyntax{kind: pkast.LiteralString, value: "hello", raw: "'hello'", span: loc.Span{Start: 30, End: 37}},
				quote:         lex.QuoteSingle,
			},
		},
		{
			name: "json arrow",
			sql:  "SELECT j->'$.a' FROM t",
			extract: func(s ast.StmtNode) []ast.ExprNode {
				call := s.(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.FuncCallExpr)
				return []ast.ExprNode{call.Args[1]}
			},
			want: literalStringSyntax{
				literalSyntax: literalSyntax{kind: pkast.LiteralString, value: "$.a", raw: "'$.a'", span: loc.Span{Start: 10, End: 15}},
				quote:         lex.QuoteSingle,
			},
		},
		{
			name: "date literal",
			sql:  "SELECT DATE '2026-01-01'",
			extract: func(s ast.StmtNode) []ast.ExprNode {
				call := s.(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.FuncCallExpr)
				return []ast.ExprNode{call.Args[0]}
			},
			want: literalStringSyntax{
				literalSyntax: literalSyntax{kind: pkast.LiteralString, value: "2026-01-01", raw: "'2026-01-01'", span: loc.Span{Start: 12, End: 24}},
				quote:         lex.QuoteSingle,
			},
		},
		{
			name: "group concat separator",
			sql:  "SELECT GROUP_CONCAT(a SEPARATOR ',') FROM t",
			extract: func(s ast.StmtNode) []ast.ExprNode {
				agg := s.(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.AggregateFuncExpr)
				return []ast.ExprNode{agg.Separator}
			},
			want: literalStringSyntax{
				literalSyntax: literalSyntax{kind: pkast.LiteralString, value: ",", raw: "','", span: loc.Span{Start: 32, End: 35}},
				quote:         lex.QuoteSingle,
			},
		},
		{
			name: "cast at time zone",
			sql:  "SELECT CAST(TIMESTAMP'2019-01-01' AT TIME ZONE 'UTC' AS DATETIME(6))",
			extract: func(s ast.StmtNode) []ast.ExprNode {
				cast := s.(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.FuncCastExpr)
				if cast.TimeZone == nil || cast.TimeZone.Expr == nil {
					return nil
				}
				return []ast.ExprNode{cast.TimeZone.Expr}
			},
			want: literalStringSyntax{
				literalSyntax: literalSyntax{kind: pkast.LiteralString, value: "UTC", raw: "'UTC'", span: loc.Span{Start: 37, End: 42}},
				quote:         lex.QuoteSingle,
			},
		},
		{
			name: "source host",
			sql:  "CHANGE MASTER TO SOURCE_HOST='host'",
			extract: func(s ast.StmtNode) []ast.ExprNode {
				return []ast.ExprNode{
					s.(*ast.ChangeMasterStmt).Options[0].Expr.(*ast.LiteralExpr),
				}
			},
			want: literalStringSyntax{
				literalSyntax: literalSyntax{kind: pkast.LiteralString, value: "host", raw: "'host'", span: loc.Span{Start: 29, End: 35}},
				quote:         lex.QuoteSingle,
			},
		},
		{
			name: "char using backtick charset",
			sql:  "SELECT CHAR(65 USING `utf8`)",
			extract: func(s ast.StmtNode) []ast.ExprNode {
				call := s.(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.FuncCallExpr)
				if !strings.EqualFold(call.Name.Name, ast.CharFunc) {
					return nil
				}
				return []ast.ExprNode{call.Args[len(call.Args)-1]}
			},
			want: literalStringSyntax{
				literalSyntax: literalSyntax{kind: pkast.LiteralString, value: "utf8", raw: "`utf8`", span: loc.Span{Start: 21, End: 27}},
				quote:         lex.QuoteBacktick,
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stmt := mustParseOne(t, tt.sql)
			checkStringSlice(t, stmt, tt.sql)
			extracted := tt.extract(stmt)
			if len(extracted) != 1 {
				t.Fatalf("extract returned nil for %q", tt.sql)
			}
			lit, ok := extracted[0].(*ast.LiteralExpr)
			if !ok {
				t.Fatalf("extracted %T, want *ast.LiteralExpr", extracted[0])
			}
			got := literalStringSyntax{
				literalSyntax: literalShapes([]*ast.LiteralExpr{lit})[0],
				introducer:    lit.Introducer.Name,
				quote:         lit.Quote,
			}
			start := strings.Index(tt.sql, tt.want.raw)
			if start < 0 {
				t.Fatal("test literal is absent from SQL")
			}
			tt.want.span = loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(tt.want.raw))}
			if got != tt.want {
				t.Errorf("literal = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// stringSliceVisitor walks a statement tree and asserts that every string or
// bytes literal carries a non-empty Raw that is a source slice.
type stringSliceVisitor struct {
	t   *testing.T
	src string
}

func (v *stringSliceVisitor) Enter(n ast.Node) (ast.Node, bool) {
	if lit, ok := n.(*ast.LiteralExpr); ok && (lit.Kind == pkast.LiteralString || lit.Kind == pkast.LiteralBytes) {
		if lit.Raw == "" {
			v.t.Errorf("literal %q has empty raw", lit.Value)
			return n, false
		}
		span := lit.Span()
		if !span.Within(len(v.src)) || v.src[span.Start:span.End] != lit.Raw {
			v.t.Errorf("literal raw %q does not match source slice", lit.Raw)
		}
	}
	return n, false
}

func (v *stringSliceVisitor) Leave(n ast.Node) (ast.Node, bool) {
	return n, true
}

// checkStringSlice visits the statement and asserts the Raw source slice.
func checkStringSlice(t *testing.T, stmt ast.StmtNode, src string) {
	t.Helper()
	if _, ok := stmt.Accept(&stringSliceVisitor{t: t, src: src}); !ok {
		t.Fatal("literal visitor aborted")
	}
}

// SELECT string literals keep their decoded value, the dialect quote style,
// and the UTF-8 byte span of the whole token.
func TestLiteralSyntaxSelectString(t *testing.T) {
	for _, tt := range []struct {
		sql        string
		introducer string
		kind       pkast.LiteralKind
		value      string
		quote      lex.QuoteStyle
		span       loc.Span
	}{
		{"SELECT 'a'", "", pkast.LiteralString, "a", lex.QuoteSingle, loc.Span{Start: 7, End: 10}},
		{`SELECT "a"`, "", pkast.LiteralString, "a", lex.QuoteDouble, loc.Span{Start: 7, End: 10}},
		{"SELECT _latin1'abc'", "latin1", pkast.LiteralString, "abc", lex.QuoteSingle, loc.Span{Start: 7, End: 19}},
		{"SELECT _binary'abc'", "binary", pkast.LiteralBytes, "abc", lex.QuoteSingle, loc.Span{Start: 7, End: 19}},
		{"SELECT _utf8'中文'", "utf8", pkast.LiteralString, "中文", lex.QuoteSingle, loc.Span{Start: 7, End: 21}},
		{"SELECT _utf8mb4'中文'", "utf8mb4", pkast.LiteralString, "中文", lex.QuoteSingle, loc.Span{Start: 7, End: 25}},
	} {
		t.Run(tt.sql, func(t *testing.T) {
			stmt := mustParseOne(t, tt.sql).(*ast.SelectStmt)
			literal, ok := stmt.Fields.Items[0].Expr.(*ast.LiteralExpr)
			if !ok {
				t.Fatalf("field expression is %T, want *ast.LiteralExpr", stmt.Fields.Items[0].Expr)
			}
			got := literalStringSyntax{
				literalSyntax: literalShapes([]*ast.LiteralExpr{literal})[0],
				introducer:    literal.Introducer.Name,
				quote:         literal.Quote,
			}
			want := literalStringSyntax{
				literalSyntax: literalSyntax{kind: tt.kind, value: tt.value, raw: tt.sql[7:], span: loc.Span{Start: 7, End: loc.ByteOffset(len(tt.sql))}},
				introducer:    tt.introducer,
				quote:         tt.quote,
			}
			if got != want {
				t.Errorf("literal = %+v, want %+v", got, want)
			}
		})
	}
}

// Doubled quote escapes decode without changing the source span.
func TestLiteralSyntaxDoubledQuotes(t *testing.T) {
	for _, tt := range []struct {
		sql   string
		value string
		span  loc.Span
	}{
		{"SELECT ''''", "'", loc.Span{Start: 7, End: 11}},
		{`SELECT "a""b"`, "a\"b", loc.Span{Start: 7, End: 13}},
		{`SELECT "a'"`, "a'", loc.Span{Start: 7, End: 11}},
		{`SELECT 'a"b'`, `a"b`, loc.Span{Start: 7, End: 12}},
		{`SELECT 'a\nb'`, "a\nb", loc.Span{Start: 7, End: 13}},
	} {
		t.Run(tt.sql, func(t *testing.T) {
			stmt := mustParseOne(t, tt.sql).(*ast.SelectStmt)
			literal, ok := stmt.Fields.Items[0].Expr.(*ast.LiteralExpr)
			if !ok {
				t.Fatalf("field expression is %T, want *ast.LiteralExpr", stmt.Fields.Items[0].Expr)
			}
			if literal.Value != tt.value {
				t.Fatalf("value = %q, want %q", literal.Value, tt.value)
			}
			if literal.Span() != tt.span {
				t.Fatalf("span = %v, want %v", literal.Span(), tt.span)
			}
			if literal.Raw != tt.sql[tt.span.Start:tt.span.End] {
				t.Fatalf("raw = %q, want %q", literal.Raw, tt.sql[tt.span.Start:tt.span.End])
			}
		})
	}
}

// Adjacent string tokens merge into one literal; the source span covers both
// tokens and the value is the concatenation.
func TestLiteralSyntaxAdjacentString(t *testing.T) {
	for _, tt := range []struct {
		sql   string
		value string
		span  loc.Span
	}{
		{`SELECT 'a' 'b'`, "ab", loc.Span{Start: 7, End: 14}},
		{`SELECT "a" "b"`, "ab", loc.Span{Start: 7, End: 14}},
		{`SELECT 'a' "b"`, "ab", loc.Span{Start: 7, End: 14}},
		{`SELECT 'a' 'b' 'c'`, "abc", loc.Span{Start: 7, End: 18}},
		{`SELECT '中文' /* gap */ "b" ''`, "中文b", loc.Span{Start: 7, End: 32}},
	} {
		t.Run(tt.sql, func(t *testing.T) {
			tt.span = loc.Span{Start: 7, End: loc.ByteOffset(len(tt.sql))}
			stmt := mustParseOne(t, tt.sql).(*ast.SelectStmt)
			literal, ok := stmt.Fields.Items[0].Expr.(*ast.LiteralExpr)
			if !ok {
				t.Fatalf("field expression is %T, want *ast.LiteralExpr", stmt.Fields.Items[0].Expr)
			}
			if literal.Value != tt.value {
				t.Fatalf("value = %q, want %q", literal.Value, tt.value)
			}
			if literal.Span() != tt.span {
				t.Fatalf("span = %v, want %v", literal.Span(), tt.span)
			}
			if literal.Raw != tt.sql[tt.span.Start:tt.span.End] {
				t.Fatalf("raw = %q, want %q", literal.Raw, tt.sql[tt.span.Start:tt.span.End])
			}
		})
	}
}

// The N'...' introducer decodes as a string literal without changing the
// value; the introducer is captured in the AST.
func TestLiteralSyntaxNStringIntroducer(t *testing.T) {
	for _, tt := range []struct {
		sql   string
		value string
		span  loc.Span
		name  string
	}{
		{"SELECT N'abc'", "abc", loc.Span{Start: 7, End: 13}, "utf8"},
		{"SELECT n'abc'", "abc", loc.Span{Start: 7, End: 13}, "utf8"},
	} {
		t.Run(tt.sql, func(t *testing.T) {
			stmt := mustParseOne(t, tt.sql).(*ast.SelectStmt)
			literal, ok := stmt.Fields.Items[0].Expr.(*ast.LiteralExpr)
			if !ok {
				t.Fatalf("field expression is %T, want *ast.LiteralExpr", stmt.Fields.Items[0].Expr)
			}
			if literal.Kind != pkast.LiteralString {
				t.Fatalf("kind = %v, want %v", literal.Kind, pkast.LiteralString)
			}
			if literal.Value != tt.value {
				t.Fatalf("value = %q, want %q", literal.Value, tt.value)
			}
			if literal.Span() != tt.span {
				t.Fatalf("span = %v, want %v", literal.Span(), tt.span)
			}
			if literal.Introducer.Name != tt.name {
				t.Fatalf("introducer = %q, want %q", literal.Introducer.Name, tt.name)
			}
			if literal.Introducer.Raw != tt.sql[7:8] || literal.Introducer.Span != (loc.Span{Start: 7, End: 8}) {
				t.Fatalf("introducer origin = %+v", literal.Introducer)
			}
		})
	}
}

// TestLiteralSyntaxStringRestoreRoundTrip checks that string literals with
// introducers restore to the canonical form and re-parse cleanly.
func TestLiteralSyntaxStringRestoreRoundTrip(t *testing.T) {
	for _, tt := range []struct {
		sql     string
		literal string
	}{
		{"SELECT _latin1'abc'", "SELECT _LATIN1'abc'"},
		{"SELECT _binary'abc'", "SELECT _BINARY'abc'"},
		{"SELECT _utf8'中文'", "SELECT _UTF8'中文'"},
		{"SELECT _utf8mb4'中文'", "SELECT '中文'"},
		{"SELECT N'abc'", "SELECT _UTF8'abc'"},
	} {
		t.Run(tt.sql, func(t *testing.T) {
			restored := restoreStmt(t, mustParseOne(t, tt.sql))
			if restored != tt.literal {
				t.Fatalf("restored %q, want %q", restored, tt.literal)
			}
			restoredAgain := restoreStmt(t, mustParseOne(t, restored))
			if restored != restoredAgain {
				t.Fatalf("re-restored %q, want %q", restoredAgain, restored)
			}
		})
	}
}

func TestLiteralSyntaxBinary(t *testing.T) {
	for _, tc := range []struct {
		raw, value, restored string
		kind                 pkast.LiteralKind
		quote                lex.QuoteStyle
		introducer           string
	}{
		{"0xF", "F", "x'0f'", pkast.LiteralHex, lex.QuoteUnquoted, ""},
		{"X'00AF'", "00AF", "x'00af'", pkast.LiteralHex, lex.QuoteSingle, ""},
		{"x''", "", "x''", pkast.LiteralHex, lex.QuoteSingle, ""},
		{"0b0001", "0001", "b'1'", pkast.LiteralBit, lex.QuoteUnquoted, ""},
		{"B'0000'", "0000", "b'0'", pkast.LiteralBit, lex.QuoteSingle, ""},
		{"b''", "", "b''", pkast.LiteralBit, lex.QuoteSingle, ""},
		{"_binary X'00AF'", "00AF", "_BINARY x'00af'", pkast.LiteralHex, lex.QuoteSingle, "binary"},
		{"_utf8mb4 0x41", "41", "_UTF8MB4 x'41'", pkast.LiteralHex, lex.QuoteUnquoted, "utf8mb4"},
		{"_latin1 B'0001'", "0001", "_LATIN1 b'1'", pkast.LiteralBit, lex.QuoteSingle, "latin1"},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			sql := "SELECT " + tc.raw
			stmt := mustParseOne(t, sql)
			literal := literalFields(t, stmt)[0]
			checkLiteralSyntax(t, []*ast.LiteralExpr{literal}, []literalSyntax{{
				kind: tc.kind, value: tc.value, raw: tc.raw, span: loc.Span{Start: 7, End: loc.ByteOffset(len(sql))},
			}})
			if literal.Quote != tc.quote || literal.Introducer.Name != tc.introducer {
				t.Fatalf("quote/introducer = %v/%+v", literal.Quote, literal.Introducer)
			}
			if tc.introducer != "" {
				want := lex.Ident{Name: tc.introducer, Raw: "_" + tc.introducer, Span: loc.Span{Start: 7, End: loc.ByteOffset(8 + len(tc.introducer))}}
				if literal.Introducer != want {
					t.Fatalf("introducer = %+v, want %+v", literal.Introducer, want)
				}
			}
			// The syntax literal no longer exposes any execution-layer metadata.
			if _, ok := interface{}(literal).(interface{ GetValue() interface{} }); ok {
				t.Fatal("literal exposes a runtime value accessor")
			}
			want := "SELECT " + tc.restored
			if got := restoreStmt(t, stmt); got != want {
				t.Fatalf("Restore = %q, want %q", got, want)
			}
			if got := restoreStmt(t, mustParseOne(t, want)); got != want {
				t.Fatalf("second Restore = %q, want %q", got, want)
			}
		})
	}
}

func TestLiteralSyntaxStringOriginAndReuse(t *testing.T) {
	const sql = `SELECT 'plain', 'a\nb', 'a''b', '中文' /* gap */ "b", _LATIN1'c', _binary'd', N'e', 0xF, b'001'`
	parser := New()
	stmt, err := parser.ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}
	literals := literalFields(t, stmt)
	want := literalShapes(literals)
	for _, literal := range literals {
		// The syntax literal no longer exposes any execution-layer metadata.
		if _, ok := interface{}(literal).(interface{ GetValue() interface{} }); ok {
			t.Fatal("literal exposes a runtime value accessor")
		}
	}
	latin1 := literals[4].Introducer
	start := loc.ByteOffset(strings.Index(sql, "_LATIN1"))
	if latin1 != (lex.Ident{Name: "latin1", Raw: "_LATIN1", Span: loc.Span{Start: start, End: start + 7}}) {
		t.Fatalf("introducer = %+v", latin1)
	}
	for _, tc := range []struct {
		sql   string
		valid bool
	}{
		{`SELECT 'longer\ntext' 'replacement', x'FFFF'`, true},
		{`SELECT 'invalid`, false},
	} {
		if _, err := parser.ParseOneStmt(tc.sql, "latin1", "latin1_bin"); (err == nil) != tc.valid {
			t.Fatalf("reuse Parse(%q): %v", tc.sql, err)
		}
		checkLiteralSyntax(t, literals, want)
		if literals[4].Introducer != latin1 {
			t.Fatal("Parser reuse changed introducer")
		}
	}

	original := literals[0]
	replacement := replaceLiteral(original, "changed")
	node, ok := original.Accept(extensionVisitor{enter: func(ast.Node) (ast.Node, bool) { return replacement, true }})
	if !ok || node != replacement || replacement.Raw != original.Raw || replacement.Span() != original.Span() || original.Value != "plain" {
		t.Fatal("visitor replacement lost outer identity or source origin")
	}
}

func TestLiteralStringValue(t *testing.T) {
	for _, tc := range []struct {
		name string
		expr ast.ExprNode
		want string
		ok   bool
	}{
		{"string", &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralString, Value: "text"}}, "text", true},
		{"bytes", &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralBytes, Value: "\x00"}}, "\x00", true},
		{"empty string", &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralString}}, "", true},
		{"integer", &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "1"}}, "", false},
		{"hex", &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralHex, Value: "41"}}, "", false},
		{"null", &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralNull}}, "", false},
		{"nil", nil, "", false},
		{"typed nil", (*ast.LiteralExpr)(nil), "", false},
		{"non-literal node", &ast.ColumnRef{BaseColumnRef: pkast.BaseColumnRef{Name: ast.QualifiedName{}}}, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := literalStringValue(tc.expr); got != tc.want || ok != tc.ok {
				t.Fatalf("literalStringValue = %q, %v, want %q, %v", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestLiteralSyntaxNestedStringSpans(t *testing.T) {
	for _, sql := range []string{
		`SELECT j->>'$.a' FROM t`,
		`SELECT TIME '12:00:00', TIMESTAMP '2026-01-01 12:00:00'`,
		`SELECT CAST(TIMESTAMP'2026-01-01' AT TIME ZONE INTERVAL '+00:00' AS DATETIME)`,
		`START REPLICA USER='MOCK_USER' PASSWORD='MOCK_TOKEN' DEFAULT_AUTH='MOCK_AUTH' PLUGIN_DIR='MOCK_DIR'`,
		`START REPLICA PASSWORD='MOCK_TOKEN' DEFAULT_AUTH='MOCK_AUTH' PLUGIN_DIR='MOCK_DIR'`,
		`START REPLICA DEFAULT_AUTH='MOCK_AUTH' PLUGIN_DIR='MOCK_DIR'`,
		`START REPLICA PLUGIN_DIR='MOCK_DIR'`,
		`START REPLICA UNTIL SQL_BEFORE_GTIDS='MOCK_GTID'`,
		`START REPLICA UNTIL SQL_AFTER_GTIDS='MOCK_GTID'`,
		`START SLAVE UNTIL MASTER_GTID_POS='0-1-2'`,
		`START REPLICA UNTIL SOURCE_LOG_FILE='MOCK_FILE', SOURCE_LOG_POS=4`,
		`CHANGE REPLICATION SOURCE TO SOURCE_HOST='MOCK_HOST', SOURCE_TLS_CIPHERSUITES='MOCK_TLS', ASSIGN_GTIDS_TO_ANONYMOUS_TRANSACTIONS='aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee'`,
	} {
		t.Run(sql, func(t *testing.T) {
			stmt := mustParseOne(t, sql)
			checkStringSlice(t, stmt, sql)
			count := 0
			if _, ok := stmt.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
				if literal, ok := n.(*ast.LiteralExpr); ok && literal.Kind == pkast.LiteralString {
					count++
					want := "'" + literal.Value + "'"
					start := strings.Index(sql, want)
					if start < 0 || literal.Raw != want || literal.Span() != (loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(want))}) || literal.Quote != lex.QuoteSingle {
						t.Errorf("literal %q has wrong token origin: raw=%q span=%v quote=%v", literal.Value, literal.Raw, literal.Span(), literal.Quote)
					}
				}
				return n, false
			}}); !ok {
				t.Fatal("visitor aborted")
			}
			if count != strings.Count(sql, "'")/2 {
				t.Fatalf("visited %d strings, want %d", count, strings.Count(sql, "'")/2)
			}
		})
	}
}

package myparser

import (
	"slices"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/parserkit/loc"
)

func TestSetOperationPaginationSpans(t *testing.T) {
	const sql = "SELECT 'é' UNION (SELECT 2 LIMIT 1) ORDER BY 1 LIMIT 3"
	stmt := mustParseOne(t, sql).(*ast.SetOprStmt)
	group := stmt.Right.(*ast.QueryTailStmt)
	inner := group.Query.(*ast.SelectStmt)
	for _, tc := range []struct {
		name string
		span loc.Span
		want string
	}{
		{name: "set", span: stmt.Span(), want: sql},
		{name: "group", span: group.Span(), want: "(SELECT 2 LIMIT 1)"},
		{name: "inner limit", span: inner.Limit.Span(), want: "LIMIT 1"},
		{name: "outer pagination", span: stmt.Tail.Span(), want: "ORDER BY 1 LIMIT 3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.span.Slice(sql)
			if err != nil || got != tc.want {
				t.Fatalf("span %v selects %q, want %q (err: %v)", tc.span, got, tc.want, err)
			}
		})
	}
	var entered []ast.Node
	if _, ok := stmt.Accept(queryPaginationVisitor{enter: func(n ast.Node) {
		switch n.(type) {
		case *ast.OrderByClause, *ast.Limit:
			entered = append(entered, n)
		}
	}}); !ok || !slices.Equal(entered, []ast.Node{inner.Limit, stmt.Tail.OrderBy, stmt.Tail.Limit}) {
		t.Fatalf("pagination visitation = %v, ok = %v", entered, ok)
	}
	if group.Tail.OrderBy != nil || group.Tail.Limit != nil || group.Tail.Span().Valid() {
		t.Fatal("group without an outer tail must not have pagination")
	}
	if got := restoreStmt(t, stmt); got != sql {
		t.Fatalf("restored SQL = %q, want %q", got, sql)
	}
}

type queryPaginationVisitor struct{ enter func(ast.Node) }

func (v queryPaginationVisitor) Enter(n ast.Node) (ast.Node, bool) {
	v.enter(n)
	return n, false
}

func (queryPaginationVisitor) Leave(n ast.Node) (ast.Node, bool) { return n, true }

func TestAbsentQueryPaginationUsesNilInterfaces(t *testing.T) {
	for _, sql := range []string{
		"SELECT 1 UNION SELECT 2",
		"VALUES ROW(1) UNION SELECT 2",
		"(SELECT 1) FOR UPDATE",
		"SELECT 1 UNION (SELECT 2) FOR UPDATE",
	} {
		t.Run(sql, func(t *testing.T) {
			var orderBy, limit ast.Node
			switch n := mustParseOne(t, sql).(type) {
			case *ast.SetOprStmt:
				orderBy, limit = n.Tail.OrderBy, n.Tail.Limit
			case *ast.QueryTailStmt:
				orderBy, limit = n.Tail.OrderBy, n.Tail.Limit
			default:
				t.Fatalf("unexpected query type %T", n)
			}
			if orderBy != nil || limit != nil {
				t.Fatalf("absent ORDER BY and LIMIT must be nil interfaces, got %T and %T", orderBy, limit)
			}
		})
	}
}

func TestValuesAndTableQueryLeafSyntax(t *testing.T) {
	valuesSQL := "values row('kept'), row(2)"
	parser := New()
	valuesNode, err := parser.ParseOneStmt(valuesSQL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	values := valuesNode.(*ast.ValuesStmt)
	if got, want := values.Span(), (loc.Span{Start: 0, End: loc.ByteOffset(len(valuesSQL))}); got != want {
		t.Fatalf("VALUES span = %v, want %v", got, want)
	}
	if len(values.Rows) != 2 {
		t.Fatalf("VALUES rows = %d, want 2", len(values.Rows))
	}
	for i, row := range values.Rows {
		start := len("values ")
		if i == 1 {
			start = len("values row('kept'), ")
		}
		want := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len("row('kept')"))}
		if i == 1 {
			want.End = loc.ByteOffset(start + len("row(2)"))
		}
		if got := row.Span(); got != want {
			t.Fatalf("VALUES row %d span = %v, want %v", i, got, want)
		}
	}
	valuesText := restoreStmt(t, values)
	if valuesText != "VALUES ROW('kept'), ROW(2)" {
		t.Fatalf("VALUES restore = %q", valuesText)
	}

	tableSQL := "table db.t"
	tableNode, err := parser.ParseOneStmt(tableSQL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	table := tableNode.(*ast.TableStmt)
	if got, want := table.Span(), (loc.Span{Start: 0, End: loc.ByteOffset(len(tableSQL))}); got != want {
		t.Fatalf("TABLE span = %v, want %v", got, want)
	}
	if table.Table.TableRefCore().Name.Parts[0].Name != "db" || table.Table.TableRefCore().Name.Parts[1].Name != "t" {
		t.Fatalf("TABLE name = %#v", table.Table.TableRefCore().Name)
	}
	tableText := restoreStmt(t, table)
	if tableText != "TABLE `db`.`t`" {
		t.Fatalf("TABLE restore = %q", tableText)
	}

	valuesVisits, rowVisits, literalVisits := 0, 0, 0
	if _, ok := values.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch n.(type) {
		case *ast.ValuesStmt:
			valuesVisits++
		case *ast.RowExpr:
			rowVisits++
		case *ast.LiteralExpr:
			literalVisits++
		}
		return n, false
	}}); !ok || valuesVisits != 1 || rowVisits != 2 || literalVisits != 2 {
		t.Fatalf("VALUES visitor: ok=%v, values=%d rows=%d literals=%d", ok, valuesVisits, rowVisits, literalVisits)
	}

	tableVisits, tableNameVisits := 0, 0
	if _, ok := table.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch n.(type) {
		case *ast.TableStmt:
			tableVisits++
		case *ast.TableName:
			tableNameVisits++
		}
		return n, false
	}}); !ok || tableVisits != 1 || tableNameVisits != 1 {
		t.Fatalf("TABLE visitor: ok=%v, statements=%d names=%d", ok, tableVisits, tableNameVisits)
	}

	if _, ok := values.Accept(extensionVisitor{leave: func(n ast.Node) (ast.Node, bool) {
		if _, isRow := n.(*ast.RowExpr); isRow {
			return mustParseOne(t, "select 1").(*ast.SelectStmt).Fields.Items[0].Expr, true
		}
		return n, true
	}}); ok {
		t.Fatal("VALUES accepted a replacement that is not a row syntax node")
	}
	if _, ok := table.Accept(extensionVisitor{leave: func(n ast.Node) (ast.Node, bool) {
		if _, isTable := n.(*ast.TableName); isTable {
			return mustParseOne(t, "select 1").(*ast.SelectStmt).Fields.Items[0].Expr, true
		}
		return n, true
	}}); ok {
		t.Fatal("TABLE accepted a replacement that is not a table reference")
	}

	if _, err := parser.ParseOneStmt(valuesText, "", ""); err != nil {
		t.Fatalf("reparse restored VALUES: %v", err)
	}
	if _, err := parser.ParseOneStmt(tableText, "", ""); err != nil {
		t.Fatalf("reparse restored TABLE: %v", err)
	}
	if _, err := parser.ParseOneStmt("values row('replacement')", "", ""); err != nil {
		t.Fatal(err)
	}
	if got := restoreStmt(t, values); got != valuesText || values.Span().End != loc.ByteOffset(len(valuesSQL)) {
		t.Fatalf("parser reuse changed VALUES: %q, span %v", got, values.Span())
	}
	if got := restoreStmt(t, table); got != tableText || table.Span().End != loc.ByteOffset(len(tableSQL)) {
		t.Fatalf("parser reuse changed TABLE: %q, span %v", got, table.Span())
	}
}

func TestCreateTableQueryCategories(t *testing.T) {
	for _, tc := range []struct {
		name      string
		sql       string
		bracketed bool
	}{
		{"select", "create table t as select 1", false},
		{"set operation", "create table t as select 1 union select 2", false},
		{"with select", "create table t as with c as (select 1) select * from c", false},
		{"with set operation", "create table t as with c as (select 1) select * from c union select 2", false},
		{"bracketed select", "create table t as (select 1)", true},
		{"bracketed set operation", "create table t as (select 1 union select 2)", true},
		{"bracketed with", "create table t as (with c as (select 1) select * from c)", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			stmt := mustParseOne(t, tc.sql).(*ast.CreateTableStmt)
			if stmt.Query == nil {
				t.Fatal("CTAS query was lost")
			}
			_, expression := stmt.Query.(ast.ExprNode)
			group, wrapped := stmt.Query.(*ast.QueryTailStmt)
			if expression || (wrapped && group.Parenthesized) != tc.bracketed {
				t.Fatalf("source %T: expression=%v, want bracketed=%v", stmt.Query, expression, tc.bracketed)
			}
			span := stmt.Query.Span()
			want := strings.TrimPrefix(tc.sql, "create table t as ")
			if !span.Within(len(tc.sql)) || tc.sql[span.Start:span.End] != want {
				t.Fatalf("query span %v, want %q", span, want)
			}
			original := restoreStmt(t, stmt)
			reparsed := mustParseOne(t, original).(*ast.CreateTableStmt)
			if restored := restoreStmt(t, reparsed); restored != original {
				t.Fatalf("CTAS restore drifted: %q -> %q", original, restored)
			}
			visits := 0
			if _, ok := stmt.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
				if n == stmt.Query {
					visits++
				}
				return n, false
			}}); !ok || visits != 1 {
				t.Fatalf("CTAS source traversal: ok=%v, visits=%d", ok, visits)
			}
		})
	}
}

func TestQuerySlotsRejectExpressionReplacement(t *testing.T) {
	for _, tc := range []struct {
		name string
		sql  string
	}{
		{"CTAS", "create table t as select 1"},
		{"CTAS subquery", "create table t as (select 1)"},
		{"scalar subquery", "select (select 1)"},
		{"CTE", "with c as (select 1) select * from c"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			stmt := mustParseOne(t, tc.sql)
			before := restoreStmt(t, stmt)
			replacement := mustParseOne(t, "select 1").(*ast.SelectStmt).Fields.Items[0].Expr
			_, ok := stmt.Accept(extensionVisitor{leave: func(n ast.Node) (ast.Node, bool) {
				if _, query := n.(ast.QueryNode); query {
					return replacement, true
				}
				return n, true
			}})
			if ok {
				t.Fatal("query slot accepted an ordinary expression")
			}
			if after := restoreStmt(t, stmt); after != before {
				t.Fatalf("rejected replacement changed the source: %q -> %q", before, after)
			}
		})
	}
}

func TestSubqueryVisitorOuterReplacement(t *testing.T) {
	makeSubquery := func() *ast.SubqueryExpr {
		return mustParseOne(t, "select (select 1)").(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.SubqueryExpr)
	}
	t.Run("replacement children", func(t *testing.T) {
		original, replacement := makeSubquery(), makeSubquery()
		replacementQuery := replacement.Query
		query := mustParseOne(t, "select 2").(ast.ResultSetNode)
		var visited []ast.Node
		updated, ok := original.Accept(extensionVisitor{
			enter: func(n ast.Node) (ast.Node, bool) {
				visited = append(visited, n)
				if n == original {
					return replacement, false
				}
				return n, false
			},
			leave: func(n ast.Node) (ast.Node, bool) {
				if n == replacement.Query {
					return query, true
				}
				return n, true
			},
		})
		if !ok || updated != replacement || replacement.Query != query {
			t.Fatalf("replacement traversal: node=%T, ok=%v, query=%T", updated, ok, replacement.Query)
		}
		if len(visited) < 2 || visited[1] != replacementQuery {
			t.Fatal("Enter must visit the replacement's original query before applying Leave")
		}
		if got := restoreStmt(t, original.Query.(ast.StmtNode)); got != "SELECT 1" {
			t.Fatalf("original query mutated: %s", got)
		}
	})
	for _, tc := range []struct {
		name string
		node ast.Node
	}{
		{"wrong type", mustParseOne(t, "select 1")},
		{"typed nil", (*ast.SubqueryExpr)(nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := makeSubquery()
			if _, ok := original.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
				return tc.node, false
			}}); ok {
				t.Fatal("invalid outer replacement was accepted")
			}
		})
	}
}

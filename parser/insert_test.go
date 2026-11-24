package parser

// Insert regression tests for the p41 batch: BaseInsertStmt slots, direct
// TableName target, expression column targets, source slots and spans.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/mysql"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

// parseInsert parses sql, runs one Restore→Parse→Restore round trip and
// returns the original statement.
func parseInsert(t *testing.T, sql string) *ast.InsertStmt {
	t.Helper()
	stmt := mustParseOne(t, sql).(*ast.InsertStmt)
	restored := restoreStmt(t, stmt)
	again := mustParseOne(t, restored).(*ast.InsertStmt)
	if second := restoreStmt(t, again); second != restored {
		t.Fatalf("restore unstable: %q -> %q", restored, second)
	}
	return stmt
}

func checkInsertSourceSlots(t *testing.T, stmt *ast.InsertStmt, wantValues, wantQuery, wantSet, wantOnDup bool) {
	t.Helper()
	if stmt.Table == nil {
		t.Fatal("missing write target")
	}
	if _, ok := stmt.Table.(*ast.TableName); !ok {
		t.Fatalf("Table = %T, want the direct *ast.TableName", stmt.Table)
	}
	got := []struct {
		slot string
		set  bool
	}{
		{"Values", stmt.Values != nil},
		{"Query", stmt.Query != nil},
		{"Assignments", stmt.Assignments != nil},
		{"OnDuplicate", stmt.OnDuplicate != nil},
	}
	want := []struct {
		slot string
		set  bool
	}{
		{"Values", wantValues},
		{"Query", wantQuery},
		{"Assignments", wantSet},
		{"OnDuplicate", wantOnDup},
	}
	for i := range got {
		if got[i].set != want[i].set {
			t.Fatalf("source slots = %v, want Values=%v Query=%v Assignments=%v OnDuplicate=%v",
				got, wantValues, wantQuery, wantSet, wantOnDup)
		}
	}
}

func TestInsertValuesForms(t *testing.T) {
	t.Run("single row with column list", func(t *testing.T) {
		stmt := parseInsert(t, "insert into t (a, b) values (1, 'x')")
		checkInsertSourceSlots(t, stmt, true, false, false, false)
		if len(stmt.Values) != 1 || len(stmt.Values[0]) != 2 {
			t.Fatalf("Values = %#v, want one row of two expressions", stmt.Values)
		}
	})
	t.Run("VALUE keyword multi row without column list", func(t *testing.T) {
		stmt := parseInsert(t, "insert into t value (1), (2, 'y')")
		checkInsertSourceSlots(t, stmt, true, false, false, false)
		if len(stmt.Values) != 2 || len(stmt.Values[0]) != 1 || len(stmt.Values[1]) != 2 {
			t.Fatalf("Values = %#v, want two rows", stmt.Values)
		}
		if stmt.Columns != nil {
			t.Fatalf("Columns = %#v, want nil without a list", stmt.Columns)
		}
	})
	t.Run("DEFAULT in row", func(t *testing.T) {
		stmt := parseInsert(t, "insert into t values (1, default)")
		if _, ok := stmt.Values[0][1].(*ast.DefaultExpr); !ok {
			t.Fatalf("Values[0][1] = %#v, want *ast.DefaultExpr", stmt.Values[0][1])
		}
	})
}

func TestInsertColumnTargetMetadata(t *testing.T) {
	t.Run("qualified unicode name", func(t *testing.T) {
		sql := "insert into t (`db`.`列`) values (1)"
		stmt := parseInsert(t, sql)
		ref, ok := stmt.Columns[0].(*ast.ColumnRef)
		if !ok {
			t.Fatalf("Columns[0] = %T, want *ast.ColumnRef", stmt.Columns[0])
		}
		want := ast.QualifiedName{
			Parts: []lex.Ident{
				{Name: "db", Raw: "`db`", Quote: lex.QuoteBacktick, Span: loc.Span{Start: 15, End: 19}},
				{Name: "列", Raw: "`列`", Quote: lex.QuoteBacktick, Span: loc.Span{Start: 20, End: 25}},
			},
			Span: loc.Span{Start: 15, End: 25},
		}
		if !reflect.DeepEqual(ref.Name, want) {
			t.Fatalf("qualified name = %#v, want %#v", ref.Name, want)
		}
		if ref.Span() != want.Span {
			t.Fatalf("ColumnRef span = %#v, want %#v", ref.Span(), want.Span)
		}
	})
	t.Run("escaped backtick inside name", func(t *testing.T) {
		stmt := parseInsert(t, "insert into t (`a``b`) values (1)")
		ref, ok := stmt.Columns[0].(*ast.ColumnRef)
		if !ok {
			t.Fatalf("Columns[0] = %T, want *ast.ColumnRef", stmt.Columns[0])
		}
		if ref.Name.Parts[0] != (lex.Ident{Name: "a`b", Raw: "`a``b`", Quote: lex.QuoteBacktick, Span: loc.Span{Start: 15, End: 21}}) {
			t.Fatalf("escaped name = %#v, want decoded a`b with backtick quotes", ref.Name.Parts[0])
		}
	})
	t.Run("quoted dots are not name separators", func(t *testing.T) {
		sql := "insert into `db.x`.`表``t` (`a.b`) values (1)"
		stmt := parseInsert(t, sql)
		table := stmt.Table.(*ast.TableName)
		if len(table.Name.Parts) != 2 || table.Name.Parts[0].Name != "db.x" || table.Name.Parts[1].Name != "表`t" {
			t.Fatalf("quoted target name lost its segments: %#v", table.Name)
		}
		ref := stmt.Columns[0].(*ast.ColumnRef)
		if len(ref.Name.Parts) != 1 || ref.Name.Parts[0].Name != "a.b" || ref.Name.Parts[0].Raw != "`a.b`" {
			t.Fatalf("quoted dot was flattened or split: %#v", ref.Name)
		}
		for _, parts := range [][]lex.Ident{table.Name.Parts, ref.Name.Parts} {
			for _, ident := range parts {
				if ident.Quote != lex.QuoteBacktick || sql[ident.Span.Start:ident.Span.End] != ident.Raw {
					t.Fatalf("quoted name lost Raw/Quote/byte span: %#v", ident)
				}
			}
		}
	})
	t.Run("absent list stays nil", func(t *testing.T) {
		stmt := parseInsert(t, "insert into t select * from u")
		if stmt.Columns != nil {
			t.Fatalf("Columns = %#v, want nil", stmt.Columns)
		}
	})
	t.Run("explicit empty list stays present", func(t *testing.T) {
		stmt := parseInsert(t, "insert into t () values (1)")
		if stmt.Columns == nil || len(stmt.Columns) != 0 {
			t.Fatalf("Columns = %#v, want non-nil empty list", stmt.Columns)
		}
	})
}

func TestInsertQuerySources(t *testing.T) {
	t.Run("plain select", func(t *testing.T) {
		stmt := parseInsert(t, "insert into t select a, b from u")
		checkInsertSourceSlots(t, stmt, false, true, false, false)
		if _, ok := stmt.Query.(*ast.SelectStmt); !ok {
			t.Fatalf("Query = %T, want *ast.SelectStmt", stmt.Query)
		}
	})
	t.Run("union set operator", func(t *testing.T) {
		stmt := parseInsert(t, "insert into t select a from u union all select a from v")
		checkInsertSourceSlots(t, stmt, false, true, false, false)
		if _, ok := stmt.Query.(*ast.SetOprStmt); !ok {
			t.Fatalf("Query = %T, want *ast.SetOprStmt", stmt.Query)
		}
	})
	t.Run("with cte", func(t *testing.T) {
		stmt := parseInsert(t, "insert into t with c as (select 1 as a) select * from c")
		checkInsertSourceSlots(t, stmt, false, true, false, false)
		sel, ok := stmt.Query.(*ast.SelectStmt)
		if !ok || sel.With == nil {
			t.Fatalf("Query = %#v, want *ast.SelectStmt with With", stmt.Query)
		}
	})
	t.Run("outer parentheses keep query category", func(t *testing.T) {
		stmt := parseInsert(t, "insert into t (select 1)")
		checkInsertSourceSlots(t, stmt, false, true, false, false)
		if _, ok := stmt.Query.(*ast.SelectStmt); !ok {
			t.Fatalf("parenthesized Query = %T, want *ast.SelectStmt", stmt.Query)
		}
	})
	t.Run("parenthesized union without outer columns", func(t *testing.T) {
		// Frozen grammar only parenthesizes a plain SelectStmt; a parenthesized
		// SetOprStmt is a syntax error. Documented here, not a regression.
		_, err := New().ParseOneStmt("insert into t (select 1 union select 2)", "", "")
		if err == nil {
			t.Fatal("frozen grammar accepted a parenthesized set operator")
		}
	})
}

func TestInsertQuerySourceForms(t *testing.T) {
	for _, source := range []struct {
		name, sql string
		setOp     bool
	}{
		{"select", "select 1", false},
		{"parenthesized select", "(select 1)", false},
		{"union", "select 1 union all select 2", true},
		{"with select", "with c as (select 1) select * from c", false},
		{"parenthesized with select", "(with c as (select 1) select * from c)", false},
		{"with union", "with c as (select 1) select * from c union all select 2", true},
		{"parenthesized with union", "(with c as (select 1) select * from c union all select 2)", true},
	} {
		t.Run(source.name, func(t *testing.T) {
			for _, columns := range []struct{ name, sql string }{
				{"absent columns", ""},
				{"column target", "(a) "},
				{"empty columns", "() "},
			} {
				t.Run(columns.name, func(t *testing.T) {
					for _, verb := range []string{"insert", "replace"} {
						t.Run(verb, func(t *testing.T) {
							sql := verb + " into t " + columns.sql + source.sql
							stmt := parseInsert(t, sql)
							checkInsertSourceSlots(t, stmt, false, true, false, false)
							if stmt.IsReplace != (verb == "replace") || (stmt.Columns != nil) != (columns.sql != "") {
								t.Fatal("statement kind or column-list presence lost")
							}
							wantColumns := 0
							if columns.sql == "(a) " {
								wantColumns = 1
							}
							if len(stmt.Columns) != wantColumns {
								t.Fatalf("got %d column targets, want %d", len(stmt.Columns), wantColumns)
							}
							_, setOp := stmt.Query.(*ast.SetOprStmt)
							if setOp != source.setOp {
								t.Fatalf("Query = %T, want set operation=%v", stmt.Query, source.setOp)
							}
							if stmt.Span() != (loc.Span{Start: 0, End: loc.ByteOffset(len(sql))}) {
								t.Fatalf("statement span = %#v", stmt.Span())
							}
						})
					}
				})
			}
		})
	}
}

func TestInsertSetForms(t *testing.T) {
	t.Run("SET source stays independent of OnDuplicate", func(t *testing.T) {
		stmt := parseInsert(t, "insert into t set a = 1, b = 2 on duplicate key update a = 1")
		checkInsertSourceSlots(t, stmt, false, false, true, true)
		if len(stmt.Assignments) != 2 {
			t.Fatalf("Assignments = %#v, want two entries", stmt.Assignments)
		}
		if len(stmt.OnDuplicate) != 1 {
			t.Fatalf("OnDuplicate = %#v, want one entry", stmt.OnDuplicate)
		}
		if stmt.Assignments[0] == stmt.OnDuplicate[0] {
			t.Fatal("SET and ON DUPLICATE share one assignment")
		}
	})
	t.Run("non-nil empty SET list", func(t *testing.T) {
		stmt := parseInsert(t, "insert into t set ")
		if stmt.Assignments == nil || len(stmt.Assignments) != 0 {
			t.Fatalf("Assignments = %#v, want non-nil empty list", stmt.Assignments)
		}
		checkInsertSourceSlots(t, stmt, false, false, true, false)
	})
}

func TestInsertTargetSpans(t *testing.T) {
	t.Run("partition span covers keyword while name span does not", func(t *testing.T) {
		sql := "insert into t partition (p0, p1) values (1)"
		stmt := parseInsert(t, sql)
		table, ok := stmt.Table.(*ast.TableName)
		if !ok {
			t.Fatalf("Table = %T, want *ast.TableName", stmt.Table)
		}
		if got := table.PartitionNames; len(got) != 2 || got[0].Name != "p0" || got[1].Name != "p1" {
			t.Fatalf("PartitionNames = %#v, want [p0 p1]", table.PartitionNames)
		}
		full, name := stmt.Span(), table.Name.Span
		if full.Start != 0 || full.End != loc.ByteOffset(len(sql)) {
			t.Fatalf("statement span = %#v, want [0, %d)", full, len(sql))
		}
		if name.Start != 12 || name.End != 13 {
			t.Fatalf("name span = %#v, want [12, 13)", name)
		}
		wantTarget := loc.Span{Start: 12, End: loc.ByteOffset(strings.Index(sql, " values"))}
		if got := table.Span(); got != wantTarget {
			t.Fatalf("table span = %#v, want name plus PARTITION %#v", got, wantTarget)
		}
	})
	t.Run("plain target span equals name span", func(t *testing.T) {
		sql := "insert into t values (1)"
		stmt := parseInsert(t, sql)
		if got, want := stmt.Table.Span(), stmt.Table.(*ast.TableName).Name.Span; got != want {
			t.Fatalf("table span = %#v, want name span %#v", got, want)
		}
		if stmt.Span() != (loc.Span{Start: 0, End: loc.ByteOffset(len(sql))}) {
			t.Fatalf("statement span = %#v", stmt.Span())
		}
	})
}

func TestReplaceForms(t *testing.T) {
	t.Run("replace without alias", func(t *testing.T) {
		stmt := parseInsert(t, "replace into t (a) values (1)")
		if !stmt.IsReplace || stmt.Priority != mysql.NoPriority {
			t.Fatalf("IsReplace=%v Priority=%v, want replace with no priority", stmt.IsReplace, stmt.Priority)
		}
		checkInsertSourceSlots(t, stmt, true, false, false, false)
	})
	t.Run("replace with priority", func(t *testing.T) {
		stmt := parseInsert(t, "replace low_priority into t values (1)")
		if !stmt.IsReplace || stmt.Priority != mysql.LowPriority {
			t.Fatalf("Priority = %v, want low priority", stmt.Priority)
		}
	})
}

func TestInsertRowAliasAndReturning(t *testing.T) {
	t.Run("row alias with column list", func(t *testing.T) {
		stmt := parseInsert(t, "insert into t (a, b) values (1, 2) as al (x, y)")
		if stmt.RowAlias == nil || stmt.RowAlias.Name.Name != "al" || len(stmt.RowAlias.Columns) != 2 {
			t.Fatalf("RowAlias = %#v, want name al with two columns", stmt.RowAlias)
		}
		if stmt.RowAlias.Columns[0].Parts[0].Name != "x" {
			t.Fatalf("row alias column = %#v, want x", stmt.RowAlias.Columns[0])
		}
	})
	t.Run("row alias with SET source", func(t *testing.T) {
		stmt := parseInsert(t, "insert into t set a = 1 as al")
		if stmt.RowAlias == nil || stmt.Assignments == nil {
			t.Fatalf("RowAlias=%#v Assignments=%#v, want both set", stmt.RowAlias, stmt.Assignments)
		}
	})
	t.Run("mariadb returning projection", func(t *testing.T) {
		sql := "insert into t (a, b) values (1, 2) returning a, b"
		stmt := parseInsert(t, sql)
		if stmt.Returning == nil || len(stmt.Returning.Items) != 2 {
			t.Fatalf("Returning = %#v, want two items", stmt.Returning)
		}
		_, err := New().ParseOneStmtForDialect(sql, "", "", DialectMariaDB)
		if err != nil {
			t.Fatalf("MariaDB rejected returning insert: %v", err)
		}
	})
}

func TestInsertStatementSpan(t *testing.T) {
	sql := "insert into t (a) values (1) on duplicate key update a = 1"
	stmt := parseInsert(t, sql)
	want := loc.Span{Start: 0, End: loc.ByteOffset(len(sql))}
	if got := stmt.Span(); got != want {
		t.Fatalf("statement span = %#v, want %#v", got, want)
	}
}

// TestInsertReuseAcrossParses checks that a second Parse on the same Parser
// does not mutate the first result, including decoded names and raw spans.
func TestInsertReuseAcrossParses(t *testing.T) {
	parser := New()
	firstSQL := "insert into `表``t` (`db`.`a``b`) values (1, '列')"
	first, _, err := parser.Parse(firstSQL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	stmt := first[0].(*ast.InsertStmt)
	ref, ok := stmt.Columns[0].(*ast.ColumnRef)
	if !ok {
		t.Fatalf("first result lost column refs: %#v", first[0].(*ast.InsertStmt).Columns)
	}
	wantName := ref.Name
	wantName.Parts = append([]lex.Ident(nil), ref.Name.Parts...)
	wantSpan, wantTableSpan := stmt.Span(), stmt.Table.Span()
	wantColumnSpan := ref.Span()
	wantSQL := restoreStmt(t, stmt)
	second, _, err := parser.Parse("insert into `different``table` (`longer``column`) values (2, 'changed')", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if second[0] == first[0] || !reflect.DeepEqual(ref.Name, wantName) {
		t.Fatalf("first result lost independent decoded/raw column parts: %#v", ref.Name)
	}
	if stmt.Span() != wantSpan || stmt.Table.Span() != wantTableSpan || ref.Span() != wantColumnSpan {
		t.Fatal("first result's statement/table/column byte spans changed after Parser reuse")
	}
	if got := restoreStmt(t, stmt); got != wantSQL {
		t.Fatalf("first AST changed after Parser reuse: %q, want %q", got, wantSQL)
	}
}

func TestInsertVisitorOrder(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"values", "insert into t (a, b) values (1, 2) on duplicate key update a = 3 returning a"},
		{"set", "insert into t set a = 1, b = 2 on duplicate key update a = 3 returning a"},
		{"query", "insert into t (a) select b from u returning a"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmt := parseInsert(t, tc.sql)
			var want []ast.Node
			if stmt.Query != nil {
				want = append(want, stmt.Query)
			}
			want = append(want, stmt.Table)
			for _, column := range stmt.Columns {
				want = append(want, column)
			}
			for _, row := range stmt.Values {
				for _, value := range row {
					want = append(want, value)
				}
			}
			for _, assignments := range [][]*ast.Assignment{stmt.Assignments, stmt.OnDuplicate} {
				for _, assignment := range assignments {
					want = append(want, assignment, assignment.Target, assignment.Expr)
				}
			}
			if stmt.Returning != nil {
				want = append(want, stmt.Returning)
				for _, item := range stmt.Returning.Items {
					want = append(want, item, item.Expr)
				}
			}
			selected := make(map[ast.Node]bool, len(want))
			for _, node := range want {
				selected[node] = true
			}
			var visited []ast.Node
			result, ok := stmt.Accept(insertOrderVisitor{record: &visited, selected: selected})
			if !ok || result != stmt || !reflect.DeepEqual(visited, want) {
				t.Fatalf("slot traversal incomplete or out of order: ok=%v, visited=%v, want=%v", ok, visited, want)
			}
		})
	}
}

type insertOrderVisitor struct {
	record   *[]ast.Node
	selected map[ast.Node]bool
}

func (v insertOrderVisitor) Enter(node ast.Node) (ast.Node, bool) {
	if v.selected[node] {
		*v.record = append(*v.record, node)
	}
	return node, false
}

func (v insertOrderVisitor) Leave(node ast.Node) (ast.Node, bool) { return node, true }

// TestInsertSharedTypes pins the dialect-agnostic slot types the design froze.
func TestInsertSharedTypes(t *testing.T) {
	var base pkast.BaseInsertStmt
	base.Table = &ast.TableName{}
	base.Columns = make([]pkast.ExprNode, 0)
	base.Values = [][]pkast.ExprNode{{}}
	base.Query = &ast.SelectStmt{}
	base.Returning = &ast.FieldList{}
	_ = &ast.InsertStmt{BaseInsertStmt: base}
}

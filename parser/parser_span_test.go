package parser

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/parserkit/loc"
)

func TestSpanGeneratedReductionAndReuse(t *testing.T) {
	var parser yyParserImpl
	for _, tc := range []struct {
		name, sql string
		span      loc.Span
	}{
		{"select", "SELECT 1", loc.Span{Start: 0, End: 8}},
		{"trailing punctuation", " SELECT 1; ", loc.Span{Start: 1, End: 10}},
		{"epsilon after select", "", loc.InvalidSpan()},
		{"skipped comment", "/* comment */ SELECT 1", loc.Span{Start: 14, End: 22}},
		{"epsilon comment", "/* comment */", loc.InvalidSpan()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := parseState{src: tc.sql}
			st.lexer.reset(tc.sql)
			st.lexer.parser = &st
			if result := parser.Parse(&st.lexer); result != 0 || len(st.lexer.errs) != 0 {
				t.Fatalf("parse = %d, errors = %v", result, st.lexer.errs)
			}
			// These short inputs stay in the fixed stack, whose first slot holds Start.
			if got := parser.stack[1].span; got != tc.span {
				t.Fatalf("Start reduction = %v, want %v", got, tc.span)
			}
		})
	}
}

func TestWindowFrameNodeSpans(t *testing.T) {
	sql := "SELECT SUM(a) OVER (ROWS BETWEEN 2 PRECEDING AND CURRENT ROW) FROM t"
	p := New()
	p.EnableWindowFunc(true)
	stmt, err := p.ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}
	window := stmt.(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.WindowFuncExpr)
	frame := window.Spec.Frame.(*ast.FrameClause)
	for _, tc := range []struct {
		name string
		node ast.Node
		text string
	}{
		{"frame", frame, "ROWS BETWEEN 2 PRECEDING AND CURRENT ROW"},
		{"start bound", frame.Extent.Start, "2 PRECEDING"},
		{"end bound", frame.Extent.End, "CURRENT ROW"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			span := tc.node.Span()
			if !span.Within(len(sql)) {
				t.Fatalf("span %v is outside the source", span)
			}
			if got := sql[span.Start:span.End]; got != tc.text {
				t.Fatalf("span %v selects %q, want %q", span, got, tc.text)
			}
		})
	}
}

func TestSelectCoreNodeSpans(t *testing.T) {
	sql := "WITH c AS (SELECT 1) SELECT x FROM t GROUP BY x HAVING x > 0"
	stmt, err := New().ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}
	selectStmt := stmt.(*ast.SelectStmt)
	for _, tc := range []struct {
		name string
		node ast.Node
		text string
	}{
		{"select", selectStmt, sql},
		{"from", selectStmt.From, "t"},
		{"having", selectStmt.Having, "x > 0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			span := tc.node.Span()
			if !span.Within(len(sql)) || sql[span.Start:span.End] != tc.text {
				t.Fatalf("span %v selects %q, want %q", span, sql[span.Start:span.End], tc.text)
			}
		})
	}
}

func TestAlterColumnDefinitionSpans(t *testing.T) {
	for _, clause := range []string{"ADD `名` INT DEFAULT 1", "MODIFY `名` INT DEFAULT 1"} {
		t.Run(clause, func(t *testing.T) {
			sql := "ALTER TABLE t " + clause
			stmt, err := New().ParseOneStmt(sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			col := stmt.(*ast.AlterTableStmt).Specs[0].NewColumns[0]
			span := col.Span()
			want := clause[strings.IndexByte(clause, '`'):]
			if !span.Within(len(sql)) || sql[span.Start:span.End] != want {
				t.Fatalf("column span %v, want %q", span, want)
			}
			if col.ColumnDefCore() != &col.BaseColumnDef || col.Name.Parts[0].Raw != "`名`" {
				t.Fatalf("shared column syntax lost: %+v", col.ColumnDefCore())
			}
		})
	}
}

func TestTableAndPartitionOptionSpans(t *testing.T) {
	sql := "CREATE TABLE t (a INT) ENGINE=InnoDB COMMENT='table' PARTITION BY RANGE (a) (PARTITION p VALUES LESS THAN MAXVALUE COMMENT='partition')"
	stmt, err := New().ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}
	create := stmt.(*ast.CreateTableStmt)
	checkSpan := func(name string, node ast.Node, want string) {
		t.Helper()
		span := node.Span()
		if !span.Within(len(sql)) || sql[span.Start:span.End] != want {
			t.Errorf("%s span %v selects %q, want %q", name, span, sql[span.Start:span.End], want)
		}
	}
	checkSpan("engine option", create.Options[0], "ENGINE=InnoDB")
	checkSpan("table comment option", create.Options[1], "COMMENT='table'")
	checkSpan("partition options", create.Partition, "PARTITION BY RANGE (a) (PARTITION p VALUES LESS THAN MAXVALUE COMMENT='partition')")
	definition := create.Partition.Definitions[0]
	checkSpan("partition definition", definition, "PARTITION p VALUES LESS THAN MAXVALUE COMMENT='partition'")
	checkSpan("partition boundary", definition.LessThan[0], "MAXVALUE")
	checkSpan("partition comment option", definition.Options[0], "COMMENT='partition'")
}

func TestAlterTableCharsetOptionSpans(t *testing.T) {
	sql := "ALTER TABLE t CONVERT TO CHARACTER SET utf8 COLLATE utf8_bin"
	stmt, err := New().ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}
	options := stmt.(*ast.AlterTableStmt).Specs[0].Options
	for i, want := range []string{"utf8", "COLLATE utf8_bin"} {
		span := options[i].Span()
		if !span.Within(len(sql)) || sql[span.Start:span.End] != want {
			t.Errorf("option %d span %v selects %q, want %q", i, span, sql[span.Start:span.End], want)
		}
	}
}

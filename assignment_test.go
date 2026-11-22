package myparser

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

func TestAssignmentSharedSyntax(t *testing.T) {
	if reflect.TypeFor[ast.Assignment]() != reflect.TypeFor[pkast.Assignment]() {
		t.Fatal("MySQL assignment must retain the shared concrete identity")
	}
	for _, tc := range []struct {
		name, sql, clause, target string
	}{
		{"update", "UPDATE t SET `列.名`=DEFAULT", "`列.名`=DEFAULT", "`列.名`"},
		{"qualified update", "UPDATE t SET `库`.`表`.`列``名`=1", "`库`.`表`.`列``名`=1", "`库`.`表`.`列``名`"},
		{"insert set", "INSERT INTO t SET `列.名`=DEFAULT", "`列.名`=DEFAULT", "`列.名`"},
		{"duplicate", "INSERT INTO t VALUES (1) ON DUPLICATE KEY UPDATE `列.名`=2", "`列.名`=2", "`列.名`"},
		{"load data", "LOAD DATA INFILE 'x' INTO TABLE t SET `列.名`=DEFAULT", "`列.名`=DEFAULT", "`列.名`"},
		{"load xml", "LOAD XML INFILE 'x' INTO TABLE t SET `列.名`=DEFAULT", "`列.名`=DEFAULT", "`列.名`"},
		{"subquery", "UPDATE t SET `列.名`=(SELECT 1)", "`列.名`=(SELECT 1)", "`列.名`"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := New()
			stmt, err := p.ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			var assignments []*ast.Assignment
			if _, ok := stmt.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
				if assignment, ok := n.(*ast.Assignment); ok {
					assignments = append(assignments, assignment)
				}
				return n, false
			}}); !ok || len(assignments) != 1 {
				t.Fatalf("assignment traversal: ok=%v, found=%d", ok, len(assignments))
			}
			a := assignments[0]
			target, ok := a.Target.(*ast.ColumnRef)
			if !ok {
				t.Fatalf("assignment target = %T, want ColumnRef", a.Target)
			}
			start := strings.Index(tc.sql, tc.clause)
			want := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(tc.clause))}
			if a.Span() != want {
				t.Fatalf("assignment span = %v, want %v", a.Span(), want)
			}
			if got, err := target.Span().Slice(tc.sql); err != nil || got != tc.target || target.Name.Span != target.Span() {
				t.Fatalf("target source = %q, err=%v, name=%+v", got, err, target.Name)
			}
			for _, part := range target.Name.Parts {
				if raw, err := part.Span.Slice(tc.sql); err != nil || raw != part.Raw || part.Quote != lex.QuoteBacktick {
					t.Fatalf("target token = %+v, raw=%q, err=%v", part, raw, err)
				}
			}
			if tc.name == "qualified update" {
				if len(target.Name.Parts) != 3 || target.Name.Parts[2].Name != "列`名" {
					t.Fatalf("qualified target lost decoded parts: %+v", target.Name)
				}
			} else if len(target.Name.Parts) != 1 || target.Name.Parts[0].Name != "列.名" {
				t.Fatalf("quoted dot was split: %+v", target.Name)
			}
			restored := restoreStmt(t, stmt)
			reparsed, err := p.ParseOneStmt(restored, "", "")
			if err != nil || restoreStmt(t, reparsed) != restored {
				t.Fatalf("assignment round trip: %s, err=%v", restored, err)
			}
			if _, err := p.ParseOneStmt("UPDATE q SET z=9", "", ""); err != nil {
				t.Fatal(err)
			}
			if a.Span() != want || restoreStmt(t, stmt) != restored {
				t.Fatal("parser reuse changed the previous assignment")
			}
		})
	}
}

func TestAssignmentTargetVisitorReplacement(t *testing.T) {
	stmt := mustParseOne(t, "UPDATE t SET a=1").(*ast.UpdateStmt)
	a := stmt.Assignments[0]
	originalTarget := a.Target
	replacement := columnRef(testSyntaxName("b"), loc.InvalidSpan())
	var entered []ast.Node
	if _, ok := a.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		entered = append(entered, n)
		if n == originalTarget {
			return replacement, false
		}
		return n, false
	}}); !ok || a.Target != replacement || len(entered) != 3 || entered[0] != a || entered[1] != originalTarget || entered[2] != a.Expr {
		t.Fatalf("target/value traversal or replacement lost: ok=%v, entered=%v", ok, entered)
	}
	if got := restoreStmt(t, stmt); !strings.Contains(got, "`b`=1") {
		t.Fatalf("target replacement not restored: %q", got)
	}
}

func TestReplicationOptionSyntax(t *testing.T) {
	for _, tc := range []struct {
		name, sql string
		clauses   []string
	}{
		{"change", "CHANGE MASTER TO master_host='数据库', master_port=3306, ignore_server_ids=(1, 2)",
			[]string{"master_host='数据库'", "master_port=3306", "ignore_server_ids=(1, 2)"}},
		{"until before", "START REPLICA UNTIL sql_before_gtids='x' USER='u' PASSWORD='p' DEFAULT_AUTH='a' PLUGIN_DIR='d' FOR CHANNEL 'c'",
			[]string{"sql_before_gtids='x'", "USER='u'", "PASSWORD='p'", "DEFAULT_AUTH='a'", "PLUGIN_DIR='d'"}},
		{"until after", "START REPLICA UNTIL sql_after_gtids='x' PASSWORD='p' DEFAULT_AUTH='a' PLUGIN_DIR='d'",
			[]string{"sql_after_gtids='x'", "PASSWORD='p'", "DEFAULT_AUTH='a'", "PLUGIN_DIR='d'"}},
		{"until maria", "START SLAVE UNTIL master_gtid_pos='0-1-4'", []string{"master_gtid_pos='0-1-4'"}},
		{"connection default", "START REPLICA DEFAULT_AUTH='a' PLUGIN_DIR='d'", []string{"DEFAULT_AUTH='a'", "PLUGIN_DIR='d'"}},
		{"connection plugin", "START REPLICA PLUGIN_DIR='d'", []string{"PLUGIN_DIR='d'"}},
		{"file coordinates", "START REPLICA UNTIL source_log_file='bin.1', source_log_pos=4", []string{"source_log_file='bin.1'", "source_log_pos=4"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmt := mustParseOne(t, tc.sql)
			var options []*ast.ReplicationOption
			_, ok := stmt.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
				switch n := n.(type) {
				case *ast.Assignment:
					t.Fatal("replication keyword classified as a column assignment")
				case *ast.ReplicationOption:
					options = append(options, n)
				}
				return n, false
			}})
			if !ok || len(options) != len(tc.clauses) {
				t.Fatalf("option traversal: ok=%v, count=%d, want=%d", ok, len(options), len(tc.clauses))
			}
			for i, clause := range tc.clauses {
				option := options[i]
				start := strings.Index(tc.sql, clause)
				if option.Span() != (loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(clause))}) {
					t.Fatalf("option %d span = %v, want only %q", i, option.Span(), clause)
				}
				name := strings.SplitN(clause, "=", 2)[0]
				if raw, err := option.Name.Span.Slice(tc.sql); err != nil || raw != name || raw != option.Name.Raw || option.Name.Quote != lex.QuoteUnquoted {
					t.Fatalf("option name span/raw lost: %+v, err=%v", option.Name, err)
				}
			}
		})
	}
}

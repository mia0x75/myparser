package parser

import (
	"testing"

	"github.com/mia0x75/myparser/ast"
)

func TestProgramReductionSpans(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want map[string][]string
	}{
		{name: "declare and set", body: "BEGIN DECLARE a INT DEFAULT 1; SET a=2; END", want: map[string][]string{
			"*ast.DeclareStmt": {"DECLARE a INT DEFAULT 1"}, "*ast.SetStmt": {"SET a=2"}, "*ast.VariableAssignment": {"a=2"},
		}},
		{name: "if", body: "BEGIN IF a THEN SET a=1; ELSEIF b THEN SET a=2; END IF; END", want: map[string][]string{
			"*ast.IfStmt": {"IF a THEN SET a=1; ELSEIF b THEN SET a=2; END IF"}, "*ast.IfClause": {"IF a THEN SET a=1;", "ELSEIF b THEN SET a=2;"},
		}},
		{name: "empty if", body: "BEGIN IF a THEN ELSEIF b THEN END IF; END", want: map[string][]string{
			"*ast.IfClause": {"IF a THEN", "ELSEIF b THEN"},
		}},
		{name: "case", body: "BEGIN CASE a WHEN 1 THEN SET a=1; END CASE; END", want: map[string][]string{
			"*ast.CaseStmt": {"CASE a WHEN 1 THEN SET a=1; END CASE"}, "*ast.CaseWhenClause": {"WHEN 1 THEN SET a=1;"},
		}},
		{name: "while label", body: "BEGIN lbl: WHILE a DO SET a=1; END WHILE lbl; END", want: map[string][]string{
			"*ast.WhileStmt": {"lbl: WHILE a DO SET a=1; END WHILE lbl"},
		}},
		{name: "repeat", body: "BEGIN REPEAT SET a=1; UNTIL a END REPEAT; END", want: map[string][]string{
			"*ast.RepeatStmt": {"REPEAT SET a=1; UNTIL a END REPEAT"},
		}},
		{name: "loop", body: "BEGIN lbl: LOOP ITERATE lbl; LEAVE lbl; END LOOP lbl; END", want: map[string][]string{
			"*ast.LoopStmt": {"lbl: LOOP ITERATE lbl; LEAVE lbl; END LOOP lbl"}, "*ast.IterateStmt": {"ITERATE lbl"}, "*ast.LeaveStmt": {"LEAVE lbl"},
		}},
		{name: "for", body: "BEGIN FOR i IN 1 .. 2 DO SET a=1; END FOR; END", want: map[string][]string{
			"*ast.ForStmt": {"FOR i IN 1 .. 2 DO SET a=1; END FOR"},
		}},
		{name: "cursor", body: "BEGIN DECLARE cur CURSOR FOR SELECT 1; OPEN cur; FETCH NEXT FROM cur INTO a; CLOSE cur; END", want: map[string][]string{
			"*ast.DeclareCursorStmt": {"DECLARE cur CURSOR FOR SELECT 1"}, "*ast.OpenStmt": {"OPEN cur"}, "*ast.FetchStmt": {"FETCH NEXT FROM cur INTO a"}, "*ast.CloseStmt": {"CLOSE cur"},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sql := "/* 源码 */ CREATE PROCEDURE p() " + tc.body
			stmt, err := New().ParseOneStmt(sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			if got, err := stmt.Span().Slice(sql); err != nil || got != "CREATE PROCEDURE p() "+tc.body {
				t.Fatalf("procedure span = %q, error=%v", got, err)
			}
			var nodes []ast.Node
			if _, ok := stmt.Accept(expressionSpanCollector{nodes: &nodes}); !ok {
				t.Fatal("procedure visitor failed")
			}
			checkNodeSpanTexts(t, sql, nodes, tc.want)
			checkNodeSpanTexts(t, sql, nodes, map[string][]string{"*ast.BeginEndStmt": {tc.body}})
		})
	}
}

func TestSetVariableReductionSpans(t *testing.T) {
	for _, tc := range []struct{ assignment, target, value string }{
		{assignment: "a=1", target: "a", value: "1"},
		{assignment: "r.a=1", target: "r.a", value: "1"},
		{assignment: "GLOBAL a=1", target: "GLOBAL a", value: "1"},
		{assignment: "SESSION a=1", target: "SESSION a", value: "1"},
		{assignment: "LOCAL a=1", target: "LOCAL a", value: "1"},
		{assignment: "@`名`=1", target: "@`名`", value: "1"},
		{assignment: "@@session.a=1", target: "@@session.a", value: "1"},
		{assignment: "NAMES utf8mb4", value: "utf8mb4"},
		{assignment: "NAMES utf8mb4 COLLATE utf8mb4_bin", value: "utf8mb4"},
		{assignment: "NAMES DEFAULT", value: "DEFAULT"},
	} {
		t.Run(tc.assignment, func(t *testing.T) {
			sql := "SET " + tc.assignment
			stmt, err := New().ParseOneStmt(sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			assignment := stmt.(*ast.SetStmt).Variables[0]
			if got, err := assignment.Span().Slice(sql); err != nil || got != tc.assignment {
				t.Fatalf("assignment span = %q, error=%v", got, err)
			}
			if tc.target != "" {
				if got, err := assignment.Target.Span().Slice(sql); err != nil || got != tc.target {
					t.Fatalf("target span = %q, error=%v", got, err)
				}
			}
			if got, err := assignment.Value.Span().Slice(sql); err != nil || got != tc.value {
				t.Fatalf("value span = %q, error=%v", got, err)
			}
		})
	}
}

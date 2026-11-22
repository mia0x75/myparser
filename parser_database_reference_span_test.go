package myparser

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
)

func TestDatabaseReferenceStatementSpans(t *testing.T) {
	for _, sql := range []string{
		"use db",
		"explain t", "describe t c", "explain select 1", "explain extended select 1",
		"explain partitions select 1", "explain for connection 1",
		"explain format=json for connection 1", "explain format=json select 1",
		"explain analyze select 1", "explain analyze format=json select 1",
		"analyze select 1", "analyze format=json select 1",
		"show tables in db where 1", "show create view t", "show create database db",
		"show create table t", "show create procedure p", "show create function f",
		"show create trigger tr", "show create event ev", "show create user 'test'@'host'",
		"show grants", "show grants for 'test'@'host'", "show master status",
		"show replica status", "show replicas", "show binary log status", "show binary logs",
		"show full processlist", "show profiles", "show profile cpu for query 1 limit 2",
		"show privileges", "show warnings limit 2", "show errors limit 2",
	} {
		t.Run(sql, func(t *testing.T) {
			input := "/* prefix */ " + sql + " ;"
			stmt := mustParseOne(t, input)
			got, err := stmt.Span().Slice(input)
			if err != nil || got != sql {
				t.Fatalf("span=%v selects %q, err=%v, want %q", stmt.Span(), got, err, sql)
			}
		})
	}
}

func TestShowIndexKeepsOneDatabaseSource(t *testing.T) {
	const sql = "show index from t from `a.b`"
	show := mustParseOne(t, sql).(*ast.ShowStmt)
	if show.Database.Present() || len(show.Table.Name.Parts) != 2 {
		t.Fatalf("SHOW INDEX duplicated or lost database source: %+v", show)
	}
	ident := show.Table.Name.Parts[0]
	if ident.Name != "a.b" || ident.Raw != "`a.b`" {
		t.Fatalf("SHOW INDEX database=%+v", ident)
	}
	if got, err := ident.Span.Slice(sql); err != nil || got != "`a.b`" {
		t.Fatalf("database span selects %q, err=%v", got, err)
	}
}

func TestDatabaseReferenceMultiStatementSpans(t *testing.T) {
	const input = "/* start */ use dba; show events in dbb where 1; explain for schema dbc select 1;"
	stmts, _, err := New().Parse(input, "", "")
	if err != nil || len(stmts) != 3 {
		t.Fatalf("statements=%d err=%v", len(stmts), err)
	}
	for i, want := range []string{"use dba", "show events in dbb where 1", "explain for schema dbc select 1"} {
		got, err := stmts[i].Span().Slice(input)
		if err != nil || got != want || int(stmts[i].Span().Start) != strings.Index(input, want) {
			t.Fatalf("statement %d selects %q, err=%v, want %q", i, got, err, want)
		}
	}
}

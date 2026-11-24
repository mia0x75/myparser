package parser

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	pkformat "github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

func TestPartitionIdentMetadata(t *testing.T) {
	const raw = "`分``区`"
	p := New()
	sql := "CREATE TABLE t (a INT) PARTITION BY RANGE (a) (PARTITION " + raw + " VALUES LESS THAN (10))"
	stmt, err := p.ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}
	def := stmt.(*ast.CreateTableStmt).Partition.Definitions[0]
	dml := "INSERT INTO t PARTITION (" + raw + ") VALUES (1)"
	stmt, err = p.ParseOneStmt(dml, "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		sql string
		id  lex.Ident
	}{
		{sql, def.Name},
		{dml, stmt.(*ast.InsertStmt).Table.(*ast.TableName).PartitionNames[0]},
	} {
		if got, err := tc.id.Span.Slice(tc.sql); err != nil || got != raw || tc.id.Raw != raw || tc.id.Name != "分`区" || tc.id.Quote != lex.QuoteBacktick {
			t.Fatalf("name = %+v, source = %q, error = %v", tc.id, got, err)
		}
	}
}

func TestIndexKeyPartSyntax(t *testing.T) {
	const sql = "CREATE INDEX i ON t (`名`(3) DESC, (a + 1))"
	stmt := mustParseOne(t, sql).(*ast.CreateIndexStmt)
	if got, err := stmt.Span().Slice(sql); err != nil || got != sql {
		t.Fatalf("CREATE INDEX span selects %q (%v), want full statement", got, err)
	}
	col, expr := stmt.Keys[0].IndexKeyPartCore(), stmt.Keys[1].IndexKeyPartCore()
	if col.PrefixLength != 3 || !col.Desc || col.Column.Parts[0].Name != "名" || expr.Expr == nil {
		t.Fatalf("key parts lost their syntax: %+v, %+v", col, expr)
	}
	for _, tc := range []struct {
		node ast.Node
		want string
	}{{stmt.Keys[0], "`名`(3) DESC"}, {stmt.Keys[1], "(a + 1)"}, {expr.Expr, "a + 1"}} {
		if got, err := tc.node.Span().Slice(sql); err != nil || got != tc.want {
			t.Fatalf("%T range = %q, %v; want %q", tc.node, got, err, tc.want)
		}
	}
	restored := restoreStmt(t, stmt)
	if !strings.Contains(restored, "`名`(3) DESC") || restoreStmt(t, mustParseOne(t, restored)) != restored {
		t.Fatalf("index round trip lost a key part: %s", restored)
	}
}

// TestDDLIndexConstraintIdentSpans guards the Phase 4 migration of DDL index
// and constraint names from plain strings to lex.Ident: decoded case, original
// Raw spelling, quote style, token span, unnamed semantics, and unchanged
// Restore output for unnamed names.
func TestDDLIndexConstraintIdentSpans(t *testing.T) {
	joinSpan := func(start, length int) loc.Span {
		return loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + length)}
	}

	cases := []struct {
		name  string
		sql   string
		check func(t *testing.T, stmt ast.StmtNode, sql string)
	}{
		{"create index ident", "CREATE INDEX `MyIdx` ON t (a)", func(t *testing.T, stmt ast.StmtNode, sql string) {
			n := stmt.(*ast.CreateIndexStmt)
			if n.IndexName.Name != "MyIdx" || n.IndexName.Raw != "`MyIdx`" || n.IndexName.Quote != lex.QuoteBacktick {
				t.Fatalf("IndexName = %+v, want decoded MyIdx with backtick Raw", n.IndexName)
			}
			// The backtick-quoted token occupies bytes [13, 20).
			want := joinSpan(13, 7)
			if n.IndexName.Span != want || !n.IndexName.Span.Valid() || !n.IndexName.Span.Within(len(sql)) {
				t.Fatalf("IndexName span = %v, want %v within source", n.IndexName.Span, want)
			}
		}},
		{"drop index ident", "DROP INDEX `MyIdx` ON t", func(t *testing.T, stmt ast.StmtNode, sql string) {
			n := stmt.(*ast.DropIndexStmt)
			if n.IndexName.Name != "MyIdx" || n.IndexName.Raw != "`MyIdx`" || n.IndexName.Quote != lex.QuoteBacktick {
				t.Fatalf("IndexName = %+v, want decoded MyIdx with backtick Raw", n.IndexName)
			}
			// The backtick-quoted token occupies bytes [11, 18).
			want := joinSpan(11, 7)
			if n.IndexName.Span != want || !n.IndexName.Span.Within(len(sql)) {
				t.Fatalf("IndexName span = %v, want %v within source", n.IndexName.Span, want)
			}
		}},
		{"table index constraint named", "CREATE TABLE t (a INT, KEY `k1` (a))", func(t *testing.T, stmt ast.StmtNode, sql string) {
			n := stmt.(*ast.CreateTableStmt)
			c := n.Constraints[0].(*ast.Constraint)
			if c.Tp != ast.ConstraintIndex || c.IndexName.Name != "k1" || c.IndexName.Raw != "`k1`" || c.IndexName.Quote != lex.QuoteBacktick {
				t.Fatalf("constraint index name = %+v, want backtick k1", c.IndexName)
			}
			// The backtick-quoted token occupies bytes [27, 31).
			want := joinSpan(27, 4)
			if c.IndexName.Span != want || !c.IndexName.Span.Within(len(sql)) {
				t.Fatalf("constraint index name span = %v, want %v within source", c.IndexName.Span, want)
			}
		}},
		{"constraint keyword symbol", "CREATE TABLE t (a INT, CONSTRAINT `c9` FOREIGN KEY (a) REFERENCES p(a))", func(t *testing.T, stmt ast.StmtNode, sql string) {
			n := stmt.(*ast.CreateTableStmt)
			c := n.Constraints[0].(*ast.Constraint)
			if c.Tp != ast.ConstraintForeignKey || c.Name.Name != "c9" || c.Name.Raw != "`c9`" || c.Name.Quote != lex.QuoteBacktick {
				t.Fatalf("constraint name = %+v, want backtick c9", c.Name)
			}
			// The backtick-quoted Symbol token occupies bytes [34, 38).
			want := joinSpan(34, 4)
			if c.Name.Span != want || !c.Name.Span.Within(len(sql)) {
				t.Fatalf("constraint name span = %v, want %v within source", c.Name.Span, want)
			}
		}},
		{"unnamed constraint unchanged", "CREATE TABLE t (a INT, PRIMARY KEY (a))", func(t *testing.T, stmt ast.StmtNode, sql string) {
			n := stmt.(*ast.CreateTableStmt)
			c := n.Constraints[0].(*ast.Constraint)
			if c.Tp != ast.ConstraintPrimaryKey || c.Name.Present() {
				t.Fatalf("primary key constraint = %+v, want absent Name", c.Name)
			}
			if c.Name != (lex.Ident{}) {
				t.Fatalf("unnamed constraint Name = %+v, want zero lex.Ident", c.Name)
			}
		}},
		{"unnamed index name has invalid span", "CREATE TABLE t (a INT, KEY (a))", func(t *testing.T, stmt ast.StmtNode, sql string) {
			name := stmt.(*ast.CreateTableStmt).Constraints[0].(*ast.Constraint).IndexName
			if name.Present() || name.Span != loc.InvalidSpan() {
				t.Fatalf("unnamed index name = %+v, want absent identifier with invalid span", name)
			}
		}},
		{"quoted empty index name remains present", "CREATE TABLE t (a INT, KEY `` (a))", func(t *testing.T, stmt ast.StmtNode, sql string) {
			name := stmt.(*ast.CreateTableStmt).Constraints[0].(*ast.Constraint).IndexName
			if !name.Present() || name.Name != "" || name.Raw != "``" || name.Quote != lex.QuoteBacktick {
				t.Fatalf("empty quoted index name = %+v, want present backtick identifier", name)
			}
			start := strings.Index(sql, "``")
			if want := joinSpan(start, 2); name.Span != want {
				t.Fatalf("empty quoted index name span = %v, want %v", name.Span, want)
			}
		}},
		{"check constraint named", "CREATE TABLE t (a INT, CONSTRAINT c1 CHECK (a > 0))", func(t *testing.T, stmt ast.StmtNode, sql string) {
			n := stmt.(*ast.CreateTableStmt)
			c := n.Constraints[0].(*ast.Constraint)
			if c.Tp != ast.ConstraintCheck || c.Name.Name != "c1" || c.Name.Present() == false || c.Name.Quote != lex.QuoteUnquoted {
				t.Fatalf("check constraint name = %+v, want unquoted present c1", c.Name)
			}
			// The unquoted symbol token occupies bytes [34, 36).
			want := joinSpan(34, 2)
			if c.Name.Span != want || !c.Name.Span.Within(len(sql)) {
				t.Fatalf("check constraint name span = %v, want %v within source", c.Name.Span, want)
			}
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			stmt, err := p.ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			tc.check(t, stmt, tc.sql)
		})
	}
}

// TestDDLIndexConstraintUnchangedRestore pins that moving names to lex.Ident
// does not change the restored SQL for unnamed or plain names under the
// default (backtick) and unquoted flags.
func TestDDLIndexConstraintUnchangedRestore(t *testing.T) {
	cases := []struct {
		flags pkformat.RestoreFlags
		sql   string
		want  string
	}{
		{pkformat.DefaultRestoreFlags, "create index idx on t (a)", "CREATE INDEX `idx` ON `t` (`a`)"},
		{pkformat.DefaultRestoreFlags, "create unique index `Idx` on t (a)", "CREATE UNIQUE INDEX `Idx` ON `t` (`a`)"},
		{pkformat.DefaultRestoreFlags, "drop index idx on t", "DROP INDEX `idx` ON `t`"},
		{pkformat.DefaultRestoreFlags, "create table t (a int, primary key (a))", "CREATE TABLE `t` (`a` INT,PRIMARY KEY(`a`))"},
		{pkformat.DefaultRestoreFlags, "create table t (a int, constraint c1 check (a > 0))", "CREATE TABLE `t` (`a` INT,CONSTRAINT `c1` CHECK (`a`>0))"},
		{pkformat.DefaultRestoreFlags, "create table t (a int, key `` (a))", "CREATE TABLE `t` (`a` INT,INDEX ``(`a`))"},
		{pkformat.DefaultRestoreFlags, "create table t (a int, key (a))", "CREATE TABLE `t` (`a` INT,INDEX(`a`))"},
	}

	for i, tc := range cases {
		t.Run(t.Name()+"/"+tc.sql, func(t *testing.T) {
			p := New()
			stmt, err := p.ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			var sb strings.Builder
			if err := stmt.Restore(pkformat.NewRestoreCtx(tc.flags, &sb)); err != nil {
				t.Fatalf("restore: %v", err)
			}
			if got := sb.String(); got != tc.want {
				t.Fatalf("case %d restore = %q, want %q", i, got, tc.want)
			}
		})
	}
}

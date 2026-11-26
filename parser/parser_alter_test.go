package parser

import (
	"github.com/mia0x75/myparser/ast"
	fmtctx "github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
	"strings"
	"testing"
)

func TestAlterColumnDefaultRestore(t *testing.T) {
	for _, tc := range []struct{ name, sql, want string }{
		{name: "set", sql: "ALTER TABLE t ALTER COLUMN a SET DEFAULT 1", want: "ALTER TABLE `t` ALTER COLUMN `a` SET DEFAULT 1"},
		{name: "drop", sql: "ALTER TABLE t ALTER COLUMN a DROP DEFAULT", want: "ALTER TABLE `t` ALTER COLUMN `a` DROP DEFAULT"},
		{name: "null", sql: "ALTER TABLE t ALTER COLUMN a SET DEFAULT NULL", want: "ALTER TABLE `t` ALTER COLUMN `a` SET DEFAULT NULL"},
		{name: "enable keys", sql: "ALTER TABLE t ENABLE KEYS", want: "ALTER TABLE `t` ENABLE KEYS"},
		{name: "disable keys", sql: "ALTER TABLE t DISABLE KEYS", want: "ALTER TABLE `t` DISABLE KEYS"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmt, err := New().ParseOneStmt(tc.sql)
			if err != nil {
				t.Fatal(err)
			}
			var out strings.Builder
			if err := stmt.Restore(fmtctx.NewRestoreCtx(fmtctx.DefaultRestoreFlags, &out)); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != tc.want {
				t.Fatalf("restore = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAlterReductionSpans(t *testing.T) {
	for _, spec := range []string{
		"ADD COLUMN a INT DEFAULT 1", "ADD (a INT, b INT)", "ADD PRIMARY KEY(a)",
		"DROP COLUMN a", "DROP PRIMARY KEY", "MODIFY COLUMN a INT FIRST", "CHANGE COLUMN a b INT AFTER c",
		"ALTER COLUMN a SET DEFAULT 1", "ALTER COLUMN a DROP DEFAULT", "RENAME TO u", "RENAME COLUMN a TO b",
		"LOCK=NONE", "ALGORITHM=INPLACE", "FORCE", "ENABLE KEYS", "DISABLE KEYS",
		"DROP PARTITION p", "TRUNCATE PARTITION p", "COALESCE PARTITION 2", "REMOVE PARTITIONING",
	} {
		t.Run(spec, func(t *testing.T) {
			sql := "/* 列 */ ALTER TABLE t " + spec
			stmt, err := New().ParseOneStmt(sql)
			if err != nil {
				t.Fatal(err)
			}
			if got, err := stmt.Span().Slice(sql); err != nil || got != "ALTER TABLE t "+spec {
				t.Fatalf("statement span = %q, error=%v", got, err)
			}
			alter := stmt.(*ast.AlterTableStmt)
			if got, err := alter.Specs[0].Span().Slice(sql); err != nil || got != spec {
				t.Fatalf("spec span = %q, error=%v", got, err)
			}
			for _, col := range alter.Specs[0].NewColumns {
				for _, option := range col.Options {
					if got, err := option.Span().Slice(sql); err != nil || got != "DEFAULT 1" {
						t.Fatalf("option span = %q, error=%v", got, err)
					}
				}
			}
			if position := alter.Specs[0].Position; position != nil {
				switch position.Tp {
				case ast.ColumnPositionNone:
					if position.Span() != loc.InvalidSpan() {
						t.Fatalf("missing position has span %v", position.Span())
					}
				case ast.ColumnPositionFirst, ast.ColumnPositionAfter:
					got, err := position.Span().Slice(sql)
					if err != nil || (got != "FIRST" && got != "AFTER c") {
						t.Fatalf("position span = %q, error=%v", got, err)
					}
				}
			}
		})
	}
}

func TestAlterDefaultSyntaxPayload(t *testing.T) {
	for _, tc := range []struct{ clause, value string }{
		{clause: "SET DEFAULT -2", value: "-2"},
		{clause: "SET DEFAULT NULL", value: "NULL"},
		{clause: "DROP DEFAULT"},
	} {
		t.Run(tc.clause, func(t *testing.T) {
			sql := "ALTER TABLE t ALTER `名` " + tc.clause
			stmt, err := New().ParseOneStmt(sql)
			if err != nil {
				t.Fatal(err)
			}
			spec := stmt.(*ast.AlterTableStmt).Specs[0]
			if len(spec.NewColumns) != 0 || spec.OldColumnName == nil {
				t.Fatal("default operation must not synthesize a column definition")
			}
			if got, err := spec.OldColumnName.Span.Slice(sql); err != nil || got != "`名`" {
				t.Fatalf("target span = %q, error=%v", got, err)
			}
			if tc.value == "" {
				if spec.Default != nil {
					t.Fatal("DROP DEFAULT retained a value")
				}
			} else {
				if spec.Default == nil {
					t.Fatal("SET DEFAULT lost its value")
				}
				if got, err := spec.Default.Span().Slice(sql); err != nil || got != tc.value {
					t.Fatalf("default span = %q, error=%v", got, err)
				}
			}
		})
	}
}

func TestPartitionIdentMetadata(t *testing.T) {
	const raw = "`分``区`"
	p := New()
	sql := "CREATE TABLE t (a INT) PARTITION BY RANGE (a) (PARTITION " + raw + " VALUES LESS THAN (10))"
	stmt, err := p.ParseOneStmt(sql)
	if err != nil {
		t.Fatal(err)
	}
	def := stmt.(*ast.CreateTableStmt).Partition.Definitions[0]
	dml := "INSERT INTO t PARTITION (" + raw + ") VALUES (1)"
	stmt, err = p.ParseOneStmt(dml)
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
			stmt, err := p.ParseOneStmt(tc.sql)
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
		flags fmtctx.RestoreFlags
		sql   string
		want  string
	}{
		{fmtctx.DefaultRestoreFlags, "create index idx on t (a)", "CREATE INDEX `idx` ON `t` (`a`)"},
		{fmtctx.DefaultRestoreFlags, "create unique index `Idx` on t (a)", "CREATE UNIQUE INDEX `Idx` ON `t` (`a`)"},
		{fmtctx.DefaultRestoreFlags, "drop index idx on t", "DROP INDEX `idx` ON `t`"},
		{fmtctx.DefaultRestoreFlags, "create table t (a int, primary key (a))", "CREATE TABLE `t` (`a` INT,PRIMARY KEY(`a`))"},
		{fmtctx.DefaultRestoreFlags, "create table t (a int, constraint c1 check (a > 0))", "CREATE TABLE `t` (`a` INT,CONSTRAINT `c1` CHECK (`a`>0))"},
		{fmtctx.DefaultRestoreFlags, "create table t (a int, key `` (a))", "CREATE TABLE `t` (`a` INT,INDEX ``(`a`))"},
		{fmtctx.DefaultRestoreFlags, "create table t (a int, key (a))", "CREATE TABLE `t` (`a` INT,INDEX(`a`))"},
	}

	for i, tc := range cases {
		t.Run(t.Name()+"/"+tc.sql, func(t *testing.T) {
			p := New()
			stmt, err := p.ParseOneStmt(tc.sql)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			var sb strings.Builder
			if err := stmt.Restore(fmtctx.NewRestoreCtx(tc.flags, &sb)); err != nil {
				t.Fatalf("restore: %v", err)
			}
			if got := sb.String(); got != tc.want {
				t.Fatalf("case %d restore = %q, want %q", i, got, tc.want)
			}
		})
	}
}

// TestDdlRemainingPeriodIdent pins the PeriodDefinition Name/Span contract:
// Name is a lex.Ident with the real token span (not the node span, which
// covers the whole PERIOD FOR ... clause), and the MySQL dialect issue still
// highlights the PERIOD keyword at the node Span.Start.
func TestDdlRemainingPeriodIdent(t *testing.T) {
	const sql = "CREATE TABLE t(s DATE, e DATE, PERIOD FOR p(s,e))"
	stmt := mustParseOne(t, sql).(*ast.CreateTableStmt)
	if len(stmt.Periods) != 1 {
		t.Fatalf("period declaration lost: %#v", stmt.Periods)
	}
	period := stmt.Periods[0]

	// Name.Span is the real PERIOD-for-name token span, not the node span.
	if !period.Name.Span.Valid() || period.Name.Span.Start == period.Span().Start {
		t.Fatalf("period Name.Span = %#v, want the identifier token span distinct from the node span %#v", period.Name.Span, period.Span())
	}
	if got := sql[int(period.Name.Span.Start):int(period.Name.Span.End)]; got != "p" {
		t.Fatalf("period name slice = %q, want %q", got, "p")
	}

	// The node span covers the whole PERIOD FOR clause; its start is the
	// PERIOD keyword.
	if got := sql[int(period.Span().Start) : int(period.Span().Start)+len("PERIOD")]; got != "PERIOD" {
		t.Fatalf("node span start points at %q, want the PERIOD keyword", got)
	}

	// The MySQL dialect issue still highlights the PERIOD keyword.
	issues := CheckDialectCompatibility(stmt, sql, DialectMySQL)
	if len(issues) != 1 || issues[0].Code != IssueMySQLPeriod {
		t.Fatalf("issues = %#v, want exactly one IssueMySQLPeriod", issues)
	}
	if issues[0].Highlight != "PERIOD" {
		t.Fatalf("period highlight = %q, want the PERIOD keyword", issues[0].Highlight)
	}
}

// TestDdlRemainingPeriodSpanRestore pins that the period node span and name
// survive a restore/reparse round trip and a parser instance reused for a
// second parse.
func TestDdlRemainingPeriodSpanRestore(t *testing.T) {
	const sql = "CREATE TABLE t(s DATE, e DATE, PERIOD FOR p(s,e))"
	parser := New()
	stmts, _, err := parser.ParseWithWarnings(sql)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	restored := restoreStmt(t, stmts[0])
	if !strings.Contains(restored, "PERIOD FOR `p` (`s`,`e`)") {
		t.Fatalf("restore = %q, want the period spelling", restored)
	}

	// Reuse the same parser instance: spans must not be polluted by the
	// previous parse.
	reparse, err := parser.ParseOneStmt(restored)
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	rePeriod := reparse.(*ast.CreateTableStmt).Periods[0]
	if !rePeriod.Name.Span.Valid() || rePeriod.Name.Name != "p" {
		t.Fatalf("reparsed period = %#v, want name p with a valid span", rePeriod.Name)
	}
	if !rePeriod.Span().Valid() {
		t.Fatal("reparsed period node lost its span")
	}

	// A hand-built node keeps its explicitly set node span; the name span is
	// independent of the node span.
	manual := &ast.PeriodDefinition{Name: lex.Ident{Name: "q", Span: loc.Span{Start: 3, End: 4}}}
	manual.SetSpan(loc.Span{Start: 0, End: 40})
	manual.Start = qualPeriodName("s")
	manual.End = qualPeriodName("e")
	var sb strings.Builder
	if err := manual.Restore(fmtctx.NewRestoreCtx(fmtctx.DefaultRestoreFlags, &sb)); err != nil {
		t.Fatalf("manual restore: %v", err)
	}
	if got := sb.String(); got != "PERIOD FOR `q` (`s`,`e`)" {
		t.Fatalf("manual restore = %q, want the period spelling", got)
	}
}

func qualPeriodName(part string) ast.QualifiedName {
	return ast.QualifiedName{Parts: []lex.Ident{{Name: part}}}
}

func TestDDLRemainingAlterNames(t *testing.T) {
	for _, tc := range []struct {
		name, clause string
		rename       bool
	}{
		{"drop index", "DROP INDEX `索``名`", false},
		{"drop foreign", "DROP FOREIGN KEY IF EXISTS `索``名`", false},
		{"drop partition", "DROP PARTITION `索``名`", false},
		{"truncate partition", "TRUNCATE PARTITION `索``名`", false},
		{"exchange partition", "EXCHANGE PARTITION `索``名` WITH TABLE p WITHOUT VALIDATION", false},
		{"rename index", "RENAME INDEX `旧``名` TO `新``名`", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			sql := "ALTER TABLE t " + tc.clause
			stmt, err := p.ParseOneStmt(sql)
			if err != nil {
				t.Fatal(err)
			}
			spec := stmt.(*ast.AlterTableStmt).Specs[0]
			if got, err := spec.Span().Slice(sql); err != nil || got != tc.clause {
				t.Fatalf("spec span selects %q (%v), want %q", got, err, tc.clause)
			}
			ids, names := []lex.Ident{spec.Name}, []string{"索`名"}
			if tc.rename {
				ids, names = []lex.Ident{spec.FromKey, spec.ToKey}, []string{"旧`名", "新`名"}
			}
			for i, id := range ids {
				if id.Name != names[i] || !id.Present() || id.Quote != lex.QuoteBacktick {
					t.Fatalf("name %d = %+v, want %q", i, id, names[i])
				}
				if got, err := id.Span.Slice(sql); err != nil || got != id.Raw {
					t.Fatalf("name span selects %q (%v), raw=%q", got, err, id.Raw)
				}
			}
			canonical := restoreStmt(t, stmt)
			reparsed, err := p.ParseOneStmt(canonical)
			if err != nil {
				t.Fatalf("reparse %q: %v", canonical, err)
			}
			next := reparsed.(*ast.AlterTableStmt).Specs[0]
			if spec.Name.Name != next.Name.Name || spec.FromKey.Name != next.FromKey.Name || spec.ToKey.Name != next.ToKey.Name {
				t.Fatal("round trip changed name roles")
			}
			if spec.Name != ids[0] && !tc.rename || tc.rename && (spec.FromKey != ids[0] || spec.ToKey != ids[1]) || restoreStmt(t, stmt) != canonical {
				t.Fatal("parser reuse changed retained names")
			}
		})
	}
}

func TestDDLRemainingPeriodQuotedNames(t *testing.T) {
	for _, tc := range []struct {
		raw, name string
		quote     lex.QuoteStyle
	}{
		{"p", "p", lex.QuoteUnquoted},
		{"`期``间`", "期`间", lex.QuoteBacktick},
		{"``", "", lex.QuoteBacktick},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			clause := "PERIOD FOR " + tc.raw + "(s,e)"
			sql := "CREATE TABLE t(s DATE,e DATE," + clause + ")"
			p := New()
			stmt, err := p.ParseOneStmt(sql)
			if err != nil {
				t.Fatal(err)
			}
			period := stmt.(*ast.CreateTableStmt).Periods[0]
			id := period.Name
			if !id.Present() || id.Name != tc.name || id.Raw != tc.raw || id.Quote != tc.quote {
				t.Fatalf("period name = %+v", id)
			}
			if got, err := id.Span.Slice(sql); err != nil || got != tc.raw {
				t.Fatalf("name span selects %q (%v), want %q", got, err, tc.raw)
			}
			if got, err := period.Span().Slice(sql); err != nil || got != clause {
				t.Fatalf("period span selects %q (%v), want %q", got, err, clause)
			}
			issues := CheckDialectCompatibility(stmt, sql, DialectMySQL)
			if len(issues) != 1 || issues[0].Code != IssueMySQLPeriod || issues[0].Highlight != "PERIOD" {
				t.Fatalf("period issue changed: %+v", issues)
			}
			canonical := restoreStmt(t, stmt)
			reparsed, err := p.ParseOneStmt(canonical)
			if err != nil {
				t.Fatalf("reparse %q: %v", canonical, err)
			}
			next := reparsed.(*ast.CreateTableStmt).Periods[0].Name
			if next.Name != id.Name || !next.Present() || period.Name != id || restoreStmt(t, stmt) != canonical {
				t.Fatal("round trip or parser reuse changed period name")
			}
		})
	}
}

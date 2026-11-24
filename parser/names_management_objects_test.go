package parser

// p59 59.5/59.6/59.2/59.7 parser-side regression: parse each management
// statement, snapshot the AST, parse the restored SQL and normalize both
// trees with the package's syntax normalizer, then DeepEqual.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	fmtrestore "github.com/mia0x75/myparser/format"
)

// mgmtParse parses one statement; fatal on syntax errors.
func mgmtParse(t *testing.T, sql string) ast.StmtNode {
	t.Helper()
	stmt, err := New().ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatalf("parse %q: %v", sql, err)
	}
	return stmt
}

// mgmtRestore writes stmt with the default restore flags.
func mgmtRestoreStmt(t *testing.T, stmt ast.StmtNode) string {
	t.Helper()
	var out strings.Builder
	if err := stmt.Restore(fmtrestore.NewRestoreCtx(fmtrestore.DefaultRestoreFlags, &out)); err != nil {
		t.Fatalf("restore: %v", err)
	}
	return out.String()
}

// mgmtCleanText clears statement and child text fields.
func mgmtCleanText(stmt ast.StmtNode) {
	CleanNodeText(stmt)
}

// mgmtNormalize clears positions and raw metadata, keeping decoded name
// parts and presence information intact.
func mgmtNormalize(stmt ast.StmtNode) {
	normalizeTestSyntaxMetadata(stmt)
}

// mgmtRoundTrip parses sql, restores the AST, parses the restored SQL again
// and checks both trees normalize to the same shape.
func mgmtRoundTrip(t *testing.T, sql string) {
	t.Helper()
	before := mgmtParse(t, sql)
	restored := mgmtRestoreStmt(t, before)
	after := mgmtParse(t, restored)
	mgmtCleanText(before)
	mgmtCleanText(after)
	mgmtNormalize(before)
	mgmtNormalize(after)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("round trip drifted for %q:\n got restored %q\n got %#v\n want %#v", sql, restored, after, before)
	}
}

// TestMgmtTablespaceParserReuse covers 59.5 parse/normalize/reuse, including
// the quoted-empty rename where the old and new name spans differ.
func TestMgmtTablespaceParserReuse(t *testing.T) {
	t.Parallel()
	for _, sql := range []string{
		"CREATE TABLESPACE ts ADD DATAFILE '/u/f.ibd' ENGINE = INNODB",
		"CREATE UNDO TABLESPACE uts COMMENT = 'c'",
		"ALTER TABLESPACE ts RENAME TO ``",
		"ALTER UNDO TABLESPACE uts WAIT",
		"DROP TABLESPACE ts NO WAIT",
		"DROP UNDO TABLESPACE uts",
	} {
		mgmtRoundTrip(t, sql)
	}
}

// TestMgmtLogfileGroupParserReuse covers 59.5 logfile group operations.
func TestMgmtLogfileGroupParserReuse(t *testing.T) {
	t.Parallel()
	for _, sql := range []string{
		"CREATE LOGFILE GROUP lg ADD UNDOFILE 'u1' ENGINE = InnoDB INITIAL_SIZE = 1",
		"ALTER LOGFILE GROUP lg ADD UNDOFILE 'u2' WAIT",
		"DROP LOGFILE GROUP lg COMMENT = 'c'",
	} {
		mgmtRoundTrip(t, sql)
	}
}

// TestMgmtResourceGroupParserReuse covers 59.6a: an unknown type label keeps
// its decoded spelling; USER stays canonical; names keep their case.
func TestMgmtResourceGroupParserReuse(t *testing.T) {
	t.Parallel()
	for _, sql := range []string{
		"CREATE RESOURCE GROUP `MyRG` TYPE = custom VCPU = 0, 3-7",
		"CREATE RESOURCE GROUP rg TYPE = user",
		"ALTER RESOURCE GROUP `MyRG` VCPU = 1 DISABLE",
		"DROP RESOURCE GROUP rg FORCE",
	} {
		mgmtRoundTrip(t, sql)
	}
}

// TestMgmtServerParserReuse covers 59.6a: bare DATABASE is a keyword, the
// quoted “DATABASE“ label keeps name syntax, wrapper stays a string.
func TestMgmtServerParserReuse(t *testing.T) {
	t.Parallel()
	for _, sql := range []string{
		"CREATE SERVER s1 FOREIGN DATA WRAPPER 'w' OPTIONS(DATABASE 'd', HOST 'h')",
		"CREATE SERVER s2 FOREIGN DATA WRAPPER 'w' OPTIONS(`DATABASE` 'd')",
		"ALTER SERVER s1 OPTIONS(HOST 'h2')",
		"DROP SERVER IF EXISTS s1",
	} {
		mgmtRoundTrip(t, sql)
	}
}

// TestMgmtPluginParserReuse covers 59.6b plugin names through the parser.
func TestMgmtPluginParserReuse(t *testing.T) {
	t.Parallel()
	for _, sql := range []string{
		"INSTALL PLUGIN p SONAME 'libp.so'",
		"UNINSTALL PLUGIN p",
	} {
		mgmtRoundTrip(t, sql)
	}
}

// TestMgmtGrantObjectParserReuse covers 59.2: the five legal shapes, the
// quoted literal-star name, and the star spans.
func TestMgmtGrantObjectParserReuse(t *testing.T) {
	t.Parallel()
	for _, sql := range []string{
		"GRANT PROCESS ON * TO `MOCK_USER`@`MOCK_HOST`",
		"GRANT PROCESS ON *.* TO `MOCK_USER`@`MOCK_HOST`",
		"GRANT PROCESS ON db.* TO `MOCK_USER`@`MOCK_HOST`",
		"GRANT PROCESS ON db.tbl TO `MOCK_USER`@`MOCK_HOST`",
		"GRANT ALL ON db.tbl TO `MOCK_USER`@`MOCK_HOST`",
		"GRANT PROCESS ON `*`.* TO `MOCK_USER`@`MOCK_HOST`",
		"REVOKE ALL ON db.tbl FROM `MOCK_USER`@`MOCK_HOST`",
	} {
		mgmtRoundTrip(t, sql)
	}
}

// TestMgmtOptimizerHintParserReuse covers 59.7: zero-value hints, multiple
// hints, and the warn-and-discard path for unsupported hint text.
func TestMgmtOptimizerHintParserReuse(t *testing.T) {
	t.Parallel()
	for _, sql := range []string{
		"SELECT /*+ MAX_EXECUTION_TIME(0) */ 1",
		"SELECT /*+ MAX_EXECUTION_TIME(1) MAX_EXECUTION_TIME(2) */ 1",
		"UPDATE /*+ MAX_EXECUTION_TIME(3) */ t SET a = 1",
		"DELETE /*+ MAX_EXECUTION_TIME(4) */ FROM t",
	} {
		mgmtRoundTrip(t, sql)
	}
	// Unsupported hint content parses with a warning and no hints.
	stmts, warns, err := New().Parse("SELECT /*+ bka(t) */ 1", "", "")
	if err != nil {
		t.Fatalf("parse unsupported hint: %v", err)
	}
	if len(warns) != 1 {
		t.Fatalf("warnings = %d, want 1 for discarded hint", len(warns))
	}
	if got, want := mgmtRestoreStmt(t, stmts[0]), "SELECT 1"; got != want {
		t.Fatalf("restored = %q, want %q", got, want)
	}
}

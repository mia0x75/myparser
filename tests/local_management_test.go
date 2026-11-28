package parser_test

import (
	"testing"

	"github.com/mia0x75/myparser/ast"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

// TestLocalNamesAndBackupVisitors covers DECLARE keyword names, loop labels,
// BACKUP LOCK/UNLOCK table shapes, and SET CHARACTER SET assignments.
func TestLocalNamesAndBackupVisitors(t *testing.T) {
	for _, name := range []string{"query", "names", "role", "password", "count"} {
		t.Run("declare "+name, func(t *testing.T) {
			src := "create procedure p() begin declare " + name + " int default 1; select " + name + "; end"
			proc := mustParseOne(t, src).(*ast.CreateProcedureStmt)
			decl := proc.Body.(*ast.BeginEndStmt).List[0].(*ast.DeclareStmt)
			if len(decl.Names) != 1 || decl.Names[0].Name != name || decl.Default == nil {
				t.Fatalf("DECLARE name was lost: %#v", decl)
			}
			localRunRestoreTest(t, src, restoreStmt(t, proc), true)
			assertVisitorControls(t, decl, isValueOrLiteral)
		})
	}
	for _, label := range []string{
		"ACCOUNT", "ALWAYS", "BACKUP", "CLOSE", "FORMAT", "GROUP_REPLICATION", "HOST", "INVISIBLE", "OPEN", "OPTIONS",
		"OWNER", "PARSER", "PORT", "REMOVE", "RESTORE", "ROLE", "SECONDARY", "SECONDARY_ENGINE", "SECONDARY_LOAD",
		"SECONDARY_UNLOAD", "SECURITY", "SERVER", "SOCKET", "SONAME", "UPGRADE", "VISIBLE", "WRAPPER", "names", "password",
	} {
		t.Run("label "+label, func(t *testing.T) {
			src := "create procedure p() " + label + ": loop leave " + label + "; iterate " + label + "; end loop " + label
			stmt := mustParseOne(t, src)
			localRunRestoreTest(t, src, restoreStmt(t, stmt), true)
		})
	}
	backup := mustParseOne(t, "backup lock db.t").(*ast.BackupStmt)
	if backup.Unlock || len(backup.Table.Name.Parts) != 2 ||
		backup.Table.Name.Parts[0].Name != "db" ||
		backup.Table.Name.Parts[len(backup.Table.Name.Parts)-1].Name != "t" {
		t.Fatalf("backup lock table was lost: %#v", backup)
	}
	_, ok := backup.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		if _, table := n.(*ast.TableName); table {
			// Mirror the production syntaxTableName for a bare replacement.
			name := ast.QualifiedName{Parts: []lex.Ident{{Name: "u", Span: loc.InvalidSpan()}}}
			name.Span = loc.InvalidSpan()
			return &ast.TableName{BaseTableRef: pkast.BaseTableRef{Name: name}}, false
		}
		return n, false
	}})
	if !ok || restoreStmt(t, backup) != "BACKUP LOCK `u`" {
		t.Fatal("BACKUP table visitor replacement was lost")
	}
	assertVisitorControls(t, backup, func(n ast.Node) bool { _, ok := n.(*ast.TableName); return ok })
	if unlock := mustParseOne(t, "backup unlock").(*ast.BackupStmt); !unlock.Unlock || unlock.Table != nil {
		t.Fatalf("BACKUP UNLOCK shape changed: %#v", unlock)
	}
	charset := mustParseOne(t, "set character set cp1250_latin2").(*ast.SetStmt).Variables[0]
	value, ok := charset.Value.(*ast.LiteralExpr)
	if charset.Kind != ast.VariableAssignmentCharset || !ok || value.Value != "cp1250_latin2" {
		t.Fatalf("SET CHARACTER SET was normalized to SET NAMES: %#v", charset)
	}
	assertVisitorControls(t, charset, func(n ast.Node) bool { _, ok := n.(*ast.LiteralExpr); return ok })
}

// TestLocalDDLAndCharsetRoundTrip covers DDL (periods, partitions, indexes,
// backup, charset) and character-set statement restore round trips.
func TestLocalDDLAndCharsetRoundTrip(t *testing.T) {
	for _, tc := range []struct{ src, restored string }{
		{"create table t(s date,e date,period for p(s,e))", "CREATE TABLE `t` (`s` DATE,`e` DATE,PERIOD FOR `p` (`s`,`e`))"},
		{"create table t(a int) partition by key(a) partitions 3", "CREATE TABLE `t` (`a` INT) PARTITION BY KEY (`a`) PARTITIONS 3"},
		{"alter table t partition by linear hash(a) partitions 2", "ALTER TABLE `t` PARTITION BY LINEAR HASH (`a`) PARTITIONS 2"},
		{"alter table t partition by key(a) partitions 2", "ALTER TABLE `t` PARTITION BY KEY (`a`) PARTITIONS 2"},
		{"create table t(a int) partition by range(a) partitions 2 subpartition by hash(a+1) subpartitions 3 (partition p values less than (10) engine=InnoDB tablespace ts comment 'x',partition q values less than maxvalue)",
			"CREATE TABLE `t` (`a` INT) PARTITION BY RANGE (`a`) PARTITIONS 2 SUBPARTITION BY HASH (`a`+1) SUBPARTITIONS 3 (PARTITION `p` VALUES LESS THAN (10) ENGINE = InnoDB TABLESPACE = `ts` COMMENT = 'x',PARTITION `q` VALUES LESS THAN MAXVALUE)"},
		{"create table t(a int) partition by range columns(a) subpartition by key(a) subpartitions 2 (p values less than (10),q values less than maxvalue)",
			"CREATE TABLE `t` (`a` INT) PARTITION BY RANGE COLUMNS(`a`) SUBPARTITION BY KEY (`a`) SUBPARTITIONS 2 (`p` VALUES LESS THAN (10),`q` VALUES LESS THAN MAXVALUE)"},
		{"alter table t reorganize partition p into (partition q values less than (1) engine=InnoDB comment '' tablespace ts)",
			"ALTER TABLE `t` REORGANIZE PARTITION `p` INTO (PARTITION `q` VALUES LESS THAN (1) ENGINE = InnoDB COMMENT = '' TABLESPACE = `ts`)"},
		{"create function f() returns row type of db.t return 1", "CREATE DEFINER = CURRENT_USER FUNCTION `f`() RETURNS ROW TYPE OF `db`.`t` RETURN 1"},
		{"alter table t drop foreign key if exists fk", "ALTER TABLE `t` DROP FOREIGN KEY IF EXISTS `fk`"},
		{"set character set cp1250_latin2", "SET CHARACTER SET 'cp1250_latin2'"},
		{"set charset utf8mb4", "SET CHARACTER SET 'utf8mb4'"},
		{"set character set default", "SET CHARACTER SET DEFAULT"},
		{"set names utf8mb4", "SET NAMES 'utf8mb4'"},
		{"create table t(a int,key k((a+1),a))", "CREATE TABLE `t` (`a` INT,INDEX `k`((`a`+1), `a`))"},
		{"create unique index k on t((a+1),a)", "CREATE UNIQUE INDEX `k` ON `t` ((`a`+1), `a`)"},
		{"alter table t add unique key k((a+1))", "ALTER TABLE `t` ADD UNIQUE `k`((`a`+1))"},
		{"backup lock db.t", "BACKUP LOCK `db`.`t`"},
		{"backup unlock", "BACKUP UNLOCK"},
		{"set statement max_statement_time=180 for backup lock db.t", "SET STATEMENT `max_statement_time`=180 FOR BACKUP LOCK `db`.`t`"},
		{"set statement max_statement_time=180 for backup unlock", "SET STATEMENT `max_statement_time`=180 FOR BACKUP UNLOCK"},
		{"analyze local tables t,u", "ANALYZE NO_WRITE_TO_BINLOG TABLE `t`,`u`"},
		{"delete from t using t where a=1 order by a limit 2", "DELETE FROM `t` USING `t` WHERE `a`=1 ORDER BY `a` LIMIT 2"},
	} {
		t.Run(tc.src, func(t *testing.T) {
			localRunRestoreTest(t, tc.src, tc.restored, true)
		})
	}
}

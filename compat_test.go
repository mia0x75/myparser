package myparser

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	fmtrestore "github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/model"
)

func restoreStmt(t *testing.T, stmt ast.StmtNode) string {
	t.Helper()
	var sb strings.Builder
	if err := stmt.Restore(fmtrestore.NewRestoreCtx(fmtrestore.DefaultRestoreFlags, &sb)); err != nil {
		t.Fatalf("restore: %v", err)
	}
	return sb.String()
}

// TestCompatRoundTrip makes sure the newly supported MySQL/MariaDB syntax can be
// parsed, restored, and parsed again.
func TestCompatRoundTrip(t *testing.T) {
	cases := []string{
		// SELECT lock clauses.
		"select * from t for update",
		"select * from t for share",
		"select * from t lock in share mode",
		"select * from t for update of t",
		"select * from t for share of t, t2 nowait",
		"select * from t for update skip locked",

		// ALTER TABLE RENAME COLUMN.
		"alter table t rename column a to b",

		// CREATE TEMPORARY TABLE.
		"create temporary table t (a int)",
		"create temporary table t like t2",

		// GROUP BY WITH ROLLUP / CUBE.
		"select a, sum(b) from t group by a with rollup",
		"select a, sum(b) from t group by a with cube",
		"select cube(a, b), rollup(c) from t",
		"select a from t group by a with cascaded check option",
		"select a from t group by a with local check option",
		"create view v as select a from t group by a with cascaded check option",
		"create view v as select a from t group by a with local check option",
		"alter view v as select a from t group by a with local check option",

		// SAVEPOINT family.
		"savepoint sp1",
		"rollback to savepoint sp1",
		"rollback to sp1",
		"rollback work to sp1",
		"release savepoint sp1",
		"rollback",

		// CHECK constraints retained in the AST.
		"create table t (a int check (a > 0), constraint c1 check (a < 10))",

		// Partition selection.
		"insert into t partition (p0) values (1)",
		"delete from t partition (p0) where a = 1",
		"update t partition (p0) set a = 1 where a = 2",
		"select a from t partition (p0)",

		// VALUES / TABLE statements.
		"values row(1, 2), row(3, 4)",
		"table t",

		// Sequences.
		"create sequence s",
		"create or replace temporary sequence if not exists s increment by 2 minvalue 1 maxvalue 100 start with 5 cache 10 cycle",
		"create sequence s no minvalue no maxvalue nocycle",
		"drop sequence if exists s, t",

		// User defined functions.
		"create function udf_a returns string soname 'liba.so'",
		"create aggregate function udf_b returns integer soname 'libb.so'",
		"drop function udf_a",

		// RETURNING.
		"insert into t (a) values (1) returning a, b",
		"delete from t where a = 1 returning a",
		"update t set a = 1 where b = 2 returning a",

		// XA transactions.
		"xa start 'x1'",
		"xa begin 'x1' join",
		"xa end 'x1' suspend for migrate",
		"xa prepare 'x1'",
		"xa commit 'x1' one phase",
		"xa rollback 'x1'",
		"xa recover",

		// SELECT ... INTO user variables.
		"select a, b into @x, @y from t",
		"select a into x from t",

		// Common table expressions.
		"with cte as (select 1) select * from cte",
		"with recursive cte (n) as (select 1 union all select n + 1 from cte where n < 5) select * from cte",
		"with a as (select 1), b as (select 2) select * from a, b",

		// Set operations.
		"select 1 union select 2",
		"select 1 union all select 2",
		"select 1 union distinct select 2",
		"select 1 intersect select 2",
		"select 1 intersect all select 2",
		"select 1 except select 2",
		"select 1 except all select 2",
		"select 1 union select 2 intersect select 3",
		"select 1 intersect select 2 union select 3",
		"select 1 except select 2 union select 3",
		"(select 1) union (select 2)",
		"select 1 union (select 2 order by 1)",
		"select 1 union select 2 order by 1 limit 1",
		"select 1 union (select 2) order by 1 limit 1",
		"(select 1 order by 1) union select 2",
		"with cte as (select 1) select * from cte union select 2",
		"insert into t select 1 union select 2",
		"create table t as select 1 intersect select 2",

		// Tablespaces.
		"create tablespace ts add datafile 'ts.ibd' initial_size 1048576 engine innodb",
		"create undo tablespace ts add datafile 'ts.ibd'",
		"create tablespace ts add datafile 'ts.ibd' file_block_size 16384 extent_size 1048576 autoextend_size 4194304 max_size 268435456 nodegroup 0 wait comment 'hello'",
		"alter tablespace ts add datafile 'ts2.ibd' no wait",
		"alter tablespace ts rename to ts2",
		"drop tablespace ts",
		"drop undo tablespace ts engine innodb",

		// Foreign servers.
		"create server srv foreign data wrapper 'mysql' options(host 'h', database 'd', user 'u', password 'p', port '3306')",
		"create server srv foreign data wrapper 'mysql'",
		"alter server srv options(user 'u2')",
		"drop server if exists srv",

		// Resource groups.
		"create resource group rg type = system vcpu = 0-3, 5 thread_priority 10 enable",
		"create resource group rg type = user vcpu = 1",
		"alter resource group rg vcpu = 2 thread_priority 5 disable",
		"drop resource group rg",
		"drop resource group rg force",

		// Logfile groups.
		"create logfile group lg add undofile 'undo.dat' initial_size 1048576 undo_buffer_size 8388608 redo_buffer_size 4194304 nodegroup 0 wait comment 'x' engine innodb",
		"alter logfile group lg add undofile 'undo2.dat' initial_size 2097152",
		"drop logfile group lg engine innodb",

		// ALTER TABLE partition / tablespace operations.
		"alter table t exchange partition p0 with table t2",
		"alter table t exchange partition p0 with table t2 without validation",
		"alter table t reorganize partition p0, p1 into (partition p0 values less than (10), partition p1 values less than (20))",
		"alter table t reorganize partition all into (partition p0 values less than (10))",
		"alter table t remove partitioning",
		"alter table t discard tablespace",
		"alter table t import tablespace",

		// SELECT ... INTO OUTFILE / DUMPFILE and trailing INTO.
		"select * from t into outfile 'file.txt'",
		"select * from t into outfile 'file.txt' fields terminated by ',' enclosed by '\"' lines terminated by '\n'",
		"select * from t into outfile 'file.txt' character set utf8mb4 fields terminated by ','",
		"select * from t into dumpfile 'file.bin'",
		"select a from t into @x, @y",
		"select a, b from t into x",
		"select a into @x from t",

		// CHECK / REPAIR / OPTIMIZE / CHECKSUM TABLE.
		"check table t",
		"check table t1, t2 quick",
		"check table t for upgrade",
		"check table t fast medium extended changed",
		"repair table t",
		"repair table t quick extended use_frm",
		"repair table t local",
		"optimize table t",
		"optimize local table t",
		"checksum table t",
		"checksum table t1, t2 quick",

		// Column VISIBLE / INVISIBLE.
		"create table t (a int invisible)",
		"create table t (a int visible, b varchar(10) not null invisible)",
		"alter table t modify column a int invisible",
		"alter table t change column a b int visible",

		// EXPLAIN FORMAT=JSON|TRADITIONAL|TREE.
		"explain format=json select * from t",
		"explain format=traditional select 1",
		"explain format=tree select 1",
		"explain format='json' select 1",
		"explain format=tree for connection 3",
	}
	for _, sql := range cases {
		stmt, err := New().ParseOneStmt(sql, "", "")
		if err != nil {
			t.Fatalf("parse %q: %v", sql, err)
		}
		got := restoreStmt(t, stmt)
		if _, err := New().ParseOneStmt(got, "", ""); err != nil {
			t.Fatalf("reparse %q (from %q): %v", got, sql, err)
		}
	}
}

func TestSelectLockAST(t *testing.T) {
	stmt := mustParseOne(t, "select * from t for update of t skip locked")
	sel := stmt.(*ast.SelectStmt)
	if sel.LockTp != ast.SelectLockForUpdate {
		t.Fatalf("LockTp = %v, want SelectLockForUpdate", sel.LockTp)
	}
	if sel.LockWaitPolicy != ast.SelectLockWaitSkipLocked {
		t.Fatalf("LockWaitPolicy = %v, want SelectLockWaitSkipLocked", sel.LockWaitPolicy)
	}
	if len(sel.LockedTables) != 1 || sel.LockedTables[0].Name.O != "t" {
		t.Fatalf("LockedTables = %+v, want [t]", sel.LockedTables)
	}
}

func TestSavepointAST(t *testing.T) {
	sp := mustParseOne(t, "savepoint sp1").(*ast.SavepointStmt)
	if sp.Name != "sp1" {
		t.Fatalf("SavepointStmt.Name = %q, want sp1", sp.Name)
	}
	rb := mustParseOne(t, "rollback to savepoint sp1").(*ast.RollbackStmt)
	if rb.SavepointName != "sp1" {
		t.Fatalf("RollbackStmt.SavepointName = %q, want sp1", rb.SavepointName)
	}
	rs := mustParseOne(t, "release savepoint sp1").(*ast.ReleaseSavepointStmt)
	if rs.Name != "sp1" {
		t.Fatalf("ReleaseSavepointStmt.Name = %q, want sp1", rs.Name)
	}
}

func TestCheckConstraintRetained(t *testing.T) {
	stmt := mustParseOne(t, "create table t (a int check (a > 0), constraint c1 check (a < 10))").(*ast.CreateTableStmt)
	if len(stmt.Cols) != 1 || len(stmt.Cols[0].Options) != 1 || stmt.Cols[0].Options[0].Tp != ast.ColumnOptionCheck {
		t.Fatalf("column CHECK option not retained: %+v", stmt.Cols)
	}
	if len(stmt.Constraints) != 1 || stmt.Constraints[0].Tp != ast.ConstraintCheck || stmt.Constraints[0].Name != "c1" {
		t.Fatalf("table CHECK constraint not retained: %+v", stmt.Constraints)
	}
}

func TestReturningAST(t *testing.T) {
	ins := mustParseOne(t, "insert into t (a) values (1) returning a, b").(*ast.InsertStmt)
	if ins.Returning == nil || len(ins.Returning.Fields) != 2 {
		t.Fatalf("InsertStmt.Returning = %+v, want 2 fields", ins.Returning)
	}
	del := mustParseOne(t, "delete from t where a = 1 returning a").(*ast.DeleteStmt)
	if del.Returning == nil || len(del.Returning.Fields) != 1 {
		t.Fatalf("DeleteStmt.Returning = %+v, want 1 field", del.Returning)
	}
	upd := mustParseOne(t, "update t set a = 1 where b = 2 returning a").(*ast.UpdateStmt)
	if upd.Returning == nil || len(upd.Returning.Fields) != 1 {
		t.Fatalf("UpdateStmt.Returning = %+v, want 1 field", upd.Returning)
	}
}

func TestSequenceAST(t *testing.T) {
	stmt := mustParseOne(t, "create or replace sequence if not exists s increment by 2 minvalue 1").(*ast.CreateSequenceStmt)
	if !stmt.OrReplace || !stmt.IfNotExists || stmt.Name.Name.O != "s" {
		t.Fatalf("unexpected CreateSequenceStmt: %+v", stmt)
	}
	if len(stmt.Options) != 2 {
		t.Fatalf("CreateSequenceStmt.Options = %+v, want 2", stmt.Options)
	}
	drop := mustParseOne(t, "drop sequence if exists s, t").(*ast.DropSequenceStmt)
	if !drop.IfExists || len(drop.Names) != 2 {
		t.Fatalf("unexpected DropSequenceStmt: %+v", drop)
	}
}

func TestUDFAST(t *testing.T) {
	stmt := mustParseOne(t, "create aggregate function udf_b returns integer soname 'libb.so'").(*ast.CreateFunctionStmt)
	if !stmt.Aggregate || stmt.Soname != "libb.so" || stmt.Body != nil {
		t.Fatalf("unexpected UDF CreateFunctionStmt: %+v", stmt)
	}
}

func TestCTEAST(t *testing.T) {
	stmt := mustParseOne(t, "with recursive cte (n) as (select 1 union all select n + 1 from cte where n < 5) select * from cte").(*ast.SelectStmt)
	if stmt.With == nil || !stmt.With.IsRecursive || len(stmt.With.CTEs) != 1 {
		t.Fatalf("unexpected WithClause: %+v", stmt.With)
	}
	if stmt.With.CTEs[0].Name.O != "cte" || len(stmt.With.CTEs[0].ColNames) != 1 {
		t.Fatalf("unexpected CTE: %+v", stmt.With.CTEs[0])
	}
}

func TestXAStmtAST(t *testing.T) {
	stmt := mustParseOne(t, "xa commit 'x1' one phase").(*ast.XAStmt)
	if stmt.Command != "COMMIT" || stmt.XID != "x1" || stmt.Extra != "ONE PHASE" {
		t.Fatalf("unexpected XAStmt: %+v", stmt)
	}
}

func TestTablespaceAST(t *testing.T) {
	stmt := mustParseOne(t, "create undo tablespace ts add datafile 'ts.ibd' engine innodb").(*ast.CreateTablespaceStmt)
	if !stmt.Undo || stmt.Name != "ts" || len(stmt.Options) != 2 {
		t.Fatalf("unexpected CreateTablespaceStmt: %+v", stmt)
	}
	if stmt.Options[0].Tp != ast.TablespaceOptionDataFile || stmt.Options[0].Str != "ts.ibd" {
		t.Fatalf("unexpected datafile option: %+v", stmt.Options[0])
	}
	if stmt.Options[1].Tp != ast.TablespaceOptionEngine || stmt.Options[1].Str != "innodb" {
		t.Fatalf("unexpected engine option: %+v", stmt.Options[1])
	}
	alt := mustParseOne(t, "alter tablespace ts rename to ts2").(*ast.AlterTablespaceStmt)
	if alt.Name != "ts" || alt.RenameTo != "ts2" {
		t.Fatalf("unexpected AlterTablespaceStmt: %+v", alt)
	}
	drop := mustParseOne(t, "drop tablespace ts engine innodb").(*ast.DropTablespaceStmt)
	if drop.Name != "ts" || drop.Option == nil || drop.Option.Str != "innodb" {
		t.Fatalf("unexpected DropTablespaceStmt: %+v", drop)
	}
}

func TestServerAST(t *testing.T) {
	stmt := mustParseOne(t, "create server srv foreign data wrapper 'mysql' options(host 'h', user 'u')").(*ast.CreateServerStmt)
	if stmt.Name != "srv" || stmt.Wrapper != "mysql" || len(stmt.Options) != 2 {
		t.Fatalf("unexpected CreateServerStmt: %+v", stmt)
	}
	if stmt.Options[0].Name != "host" || stmt.Options[0].Value != "h" {
		t.Fatalf("unexpected server option: %+v", stmt.Options[0])
	}
	drop := mustParseOne(t, "drop server if exists srv").(*ast.DropServerStmt)
	if !drop.IfExists || drop.Name != "srv" {
		t.Fatalf("unexpected DropServerStmt: %+v", drop)
	}
}

func TestResourceGroupAST(t *testing.T) {
	stmt := mustParseOne(t, "create resource group rg type = system vcpu = 0-3, 5 thread_priority 10 enable").(*ast.CreateResourceGroupStmt)
	if stmt.Name != "rg" || !stmt.IsSystem || stmt.State != ast.ResourceGroupStateEnable {
		t.Fatalf("unexpected CreateResourceGroupStmt: %+v", stmt)
	}
	if len(stmt.VCPU) != 2 || stmt.VCPU[0].From != 0 || stmt.VCPU[0].To != 3 || stmt.VCPU[1].From != 5 || stmt.VCPU[1].To != 5 {
		t.Fatalf("unexpected VCPU: %+v", stmt.VCPU)
	}
	if stmt.ThreadPriority == nil || *stmt.ThreadPriority != 10 {
		t.Fatalf("unexpected ThreadPriority: %v", stmt.ThreadPriority)
	}
	drop := mustParseOne(t, "drop resource group rg force").(*ast.DropResourceGroupStmt)
	if drop.Name != "rg" || !drop.Force {
		t.Fatalf("unexpected DropResourceGroupStmt: %+v", drop)
	}
}

func TestLogfileGroupAST(t *testing.T) {
	stmt := mustParseOne(t, "create logfile group lg add undofile 'undo.dat' initial_size 1048576 undo_buffer_size 8388608").(*ast.CreateLogfileGroupStmt)
	if stmt.Name != "lg" || stmt.UndoFile != "undo.dat" || len(stmt.Options) != 2 {
		t.Fatalf("unexpected CreateLogfileGroupStmt: %+v", stmt)
	}
	if stmt.Options[0].Tp != ast.LogfileGroupOptionInitialSize || stmt.Options[0].Val != 1048576 {
		t.Fatalf("unexpected option: %+v", stmt.Options[0])
	}
	if stmt.Options[1].Tp != ast.LogfileGroupOptionUndoBufferSize || stmt.Options[1].Val != 8388608 {
		t.Fatalf("unexpected option: %+v", stmt.Options[1])
	}
}

func TestAlterTablePartitionAST(t *testing.T) {
	stmt := mustParseOne(t, "alter table t exchange partition p0 with table t2 without validation").(*ast.AlterTableStmt)
	spec := stmt.Specs[0]
	if spec.Tp != ast.AlterTableExchangePartition || spec.Name != "p0" {
		t.Fatalf("unexpected exchange spec: %+v", spec)
	}
	if spec.NewTable == nil || spec.NewTable.Name.O != "t2" || !spec.WithoutValidation {
		t.Fatalf("unexpected exchange spec: %+v", spec)
	}
	reorg := mustParseOne(t, "alter table t reorganize partition p0, p1 into (partition p0 values less than (10))").(*ast.AlterTableStmt).Specs[0]
	if reorg.Tp != ast.AlterTableReorganizePartition || len(reorg.PartitionNames) != 2 || len(reorg.PartDefinitions) != 1 {
		t.Fatalf("unexpected reorganize spec: %+v", reorg)
	}
	rm := mustParseOne(t, "alter table t remove partitioning").(*ast.AlterTableStmt).Specs[0]
	if rm.Tp != ast.AlterTableRemovePartitioning {
		t.Fatalf("unexpected remove spec: %+v", rm)
	}
	disc := mustParseOne(t, "alter table t discard tablespace").(*ast.AlterTableStmt).Specs[0]
	if disc.Tp != ast.AlterTableDiscardTablespace {
		t.Fatalf("unexpected discard spec: %+v", disc)
	}
	impt := mustParseOne(t, "alter table t import tablespace").(*ast.AlterTableStmt).Specs[0]
	if impt.Tp != ast.AlterTableImportTablespace {
		t.Fatalf("unexpected import spec: %+v", impt)
	}
}

func TestSelectIntoAST(t *testing.T) {
	sel := mustParseOne(t, "select a from t into outfile 'f.txt' character set utf8mb4 fields terminated by ','").(*ast.SelectStmt)
	if sel.Into == nil || sel.Into.Tp != ast.SelectIntoOutfile || sel.Into.FileName != "f.txt" || sel.Into.Charset != "utf8mb4" {
		t.Fatalf("unexpected Into: %+v", sel.Into)
	}
	if sel.Into.Fields == nil || sel.Into.Fields.Terminated != "," {
		t.Fatalf("unexpected Into.Fields: %+v", sel.Into.Fields)
	}
	dump := mustParseOne(t, "select a from t into dumpfile 'f.bin'").(*ast.SelectStmt)
	if dump.Into == nil || dump.Into.Tp != ast.SelectIntoDumpfile || dump.Into.FileName != "f.bin" {
		t.Fatalf("unexpected dump Into: %+v", dump.Into)
	}
	trailing := mustParseOne(t, "select a from t into @x, y").(*ast.SelectStmt)
	if len(trailing.IntoVars) != 2 || trailing.IntoVars[0] != "@x" || trailing.IntoVars[1] != "y" {
		t.Fatalf("unexpected trailing IntoVars: %+v", trailing.IntoVars)
	}
}

func TestTableMaintenanceAST(t *testing.T) {
	chk := mustParseOne(t, "check table t1, t2 quick").(*ast.TableMaintenanceStmt)
	if chk.Tp != ast.TableMaintenanceCheck || len(chk.Tables) != 2 || chk.Tables[0].Name.L != "t1" || len(chk.Options) != 1 || chk.Options[0] != "QUICK" {
		t.Fatalf("unexpected check stmt: %+v", chk)
	}
	opt := mustParseOne(t, "optimize local table t").(*ast.TableMaintenanceStmt)
	if opt.Tp != ast.TableMaintenanceOptimize || len(opt.Tables) != 1 || len(opt.Options) != 0 {
		t.Fatalf("unexpected optimize stmt: %+v", opt)
	}
	sum := mustParseOne(t, "checksum table t").(*ast.TableMaintenanceStmt)
	if sum.Tp != ast.TableMaintenanceChecksum || len(sum.Tables) != 1 {
		t.Fatalf("unexpected checksum stmt: %+v", sum)
	}
	rep := mustParseOne(t, "repair table t for upgrade").(*ast.TableMaintenanceStmt)
	if rep.Tp != ast.TableMaintenanceRepair || len(rep.Options) != 1 || rep.Options[0] != "FOR UPGRADE" {
		t.Fatalf("unexpected repair stmt: %+v", rep)
	}
}

func TestColumnVisibilityAST(t *testing.T) {
	ct := mustParseOne(t, "create table t (a int invisible)").(*ast.CreateTableStmt)
	opts := ct.Cols[0].Options
	if len(opts) != 1 || opts[0].Tp != ast.ColumnOptionInvisible {
		t.Fatalf("unexpected options: %+v", opts)
	}
	ctv := mustParseOne(t, "create table t (a int visible)").(*ast.CreateTableStmt)
	if got := ctv.Cols[0].Options[len(ctv.Cols[0].Options)-1].Tp; got != ast.ColumnOptionVisible {
		t.Fatalf("unexpected visible option: %v", got)
	}
}

func TestExplainFormatAST(t *testing.T) {
	tree := mustParseOne(t, "explain format=tree select 1").(*ast.ExplainStmt)
	if tree.Format != "tree" {
		t.Fatalf("unexpected format: %q", tree.Format)
	}
	tj := mustParseOne(t, "explain format=JSON select 1").(*ast.ExplainStmt)
	if tj.Format != "json" {
		t.Fatalf("unexpected format: %q", tj.Format)
	}
	forConn := mustParseOne(t, "explain format=traditional for connection 7").(*ast.ExplainForStmt)
	if forConn.Format != "traditional" || forConn.ConnectionID != 7 {
		t.Fatalf("unexpected explain for: %+v", forConn)
	}
}

func TestGroupByCheckOptionAST(t *testing.T) {
	local := mustParseOne(t, "select a from t group by a with local check option").(*ast.SelectStmt)
	if local.GroupBy == nil || !local.GroupBy.HasCheckOption || local.GroupBy.CheckOption != model.CheckOptionLocal {
		t.Fatalf("unexpected group by: %+v", local.GroupBy)
	}
	cascaded := mustParseOne(t, "select a from t group by a with cascaded check option").(*ast.SelectStmt)
	if cascaded.GroupBy == nil || !cascaded.GroupBy.HasCheckOption || cascaded.GroupBy.CheckOption != model.CheckOptionCascaded {
		t.Fatalf("unexpected group by: %+v", cascaded.GroupBy)
	}
	rollup := mustParseOne(t, "select a from t group by a with rollup").(*ast.SelectStmt)
	if rollup.GroupBy == nil || !rollup.GroupBy.WithRollup || rollup.GroupBy.HasCheckOption {
		t.Fatalf("unexpected group by: %+v", rollup.GroupBy)
	}
}

func TestSetOprAST(t *testing.T) {
	stmt := mustParseOne(t, "select 1 union all select 2 except select 3 intersect select 4").(*ast.SetOprStmt)
	if len(stmt.SelectList.Selects) != 3 {
		t.Fatalf("Selects len = %d, want 3", len(stmt.SelectList.Selects))
	}
	if op := stmt.SelectList.Selects[1].(*ast.SelectStmt).AfterSetOperator; op == nil || *op != ast.UnionAll {
		t.Fatalf("Selects[1].AfterSetOperator = %v, want UnionAll", op)
	}
	intersect, ok := stmt.SelectList.Selects[2].(*ast.SetOprSelectList)
	if !ok {
		t.Fatalf("Selects[2] = %T, want *ast.SetOprSelectList", stmt.SelectList.Selects[2])
	}
	if intersect.AfterSetOperator == nil || *intersect.AfterSetOperator != ast.Except {
		t.Fatalf("intersect.AfterSetOperator = %v, want Except", intersect.AfterSetOperator)
	}
	if len(intersect.Selects) != 2 {
		t.Fatalf("intersect.Selects len = %d, want 2", len(intersect.Selects))
	}
	if op := intersect.Selects[1].(*ast.SelectStmt).AfterSetOperator; op == nil || *op != ast.Intersect {
		t.Fatalf("intersect.Selects[1].AfterSetOperator = %v, want Intersect", op)
	}
	if stmt.SelectList.Selects[0].(*ast.SelectStmt).AfterSetOperator != nil {
		t.Fatalf("first select should not have a set operator")
	}

	distinct := mustParseOne(t, "select 1 except select 2").(*ast.SetOprStmt)
	if op := distinct.SelectList.Selects[1].(*ast.SelectStmt).AfterSetOperator; op == nil || *op != ast.Except {
		t.Fatalf("EXCEPT = %v, want Except", op)
	}
	all := mustParseOne(t, "select 1 intersect all select 2").(*ast.SetOprStmt)
	if op := all.SelectList.Selects[1].(*ast.SelectStmt).AfterSetOperator; op == nil || *op != ast.IntersectAll {
		t.Fatalf("INTERSECT ALL = %v, want IntersectAll", op)
	}

	grouped := mustParseOne(t, "select 1 union (select 2 intersect select 3)").(*ast.SetOprStmt)
	group, ok := grouped.SelectList.Selects[1].(*ast.SetOprSelectList)
	if !ok {
		t.Fatalf("Selects[1] = %T, want *ast.SetOprSelectList", grouped.SelectList.Selects[1])
	}
	if group.AfterSetOperator == nil || *group.AfterSetOperator != ast.Union {
		t.Fatalf("group.AfterSetOperator = %v, want Union", group.AfterSetOperator)
	}

	unionIntersect := mustParseOne(t, "select 1 union select 2 intersect select 3").(*ast.SetOprStmt)
	if len(unionIntersect.SelectList.Selects) != 2 {
		t.Fatalf("UNION/INTERSECT Selects len = %d, want 2", len(unionIntersect.SelectList.Selects))
	}
	unionGroup, ok := unionIntersect.SelectList.Selects[1].(*ast.SetOprSelectList)
	if !ok || unionGroup.AfterSetOperator == nil || *unionGroup.AfterSetOperator != ast.Union {
		t.Fatalf("UNION/INTERSECT group = %#v, want Union group", unionIntersect.SelectList.Selects[1])
	}
	if len(unionGroup.Selects) != 2 {
		t.Fatalf("UNION/INTERSECT group len = %d, want 2", len(unionGroup.Selects))
	}
	if unionGroup.Selects[0].(*ast.SelectStmt).AfterSetOperator != nil {
		t.Fatalf("UNION/INTERSECT group first select should not have a set operator")
	}
	if op := unionGroup.Selects[1].(*ast.SelectStmt).AfterSetOperator; op == nil || *op != ast.Intersect {
		t.Fatalf("UNION/INTERSECT group child op = %v, want Intersect", op)
	}

	intersectUnion := mustParseOne(t, "select 1 intersect select 2 union select 3").(*ast.SetOprStmt)
	intersectGroup, ok := intersectUnion.SelectList.Selects[0].(*ast.SetOprSelectList)
	if !ok || intersectGroup.AfterSetOperator != nil {
		t.Fatalf("INTERSECT/UNION first group = %#v, want unqualified group", intersectUnion.SelectList.Selects[0])
	}
	if op := intersectUnion.SelectList.Selects[1].(*ast.SelectStmt).AfterSetOperator; op == nil || *op != ast.Union {
		t.Fatalf("INTERSECT/UNION child op = %v, want Union", op)
	}

	intersectChain := mustParseOne(t, "select 1 intersect select 2 intersect select 3").(*ast.SetOprStmt)
	if len(intersectChain.SelectList.Selects) != 3 {
		t.Fatalf("INTERSECT chain len = %d, want 3", len(intersectChain.SelectList.Selects))
	}

	restoreCases := []struct {
		src     string
		restore string
	}{
		{"select 1 union select 2 intersect select 3", "SELECT 1 UNION (SELECT 2 INTERSECT SELECT 3)"},
		{"select 1 intersect select 2 union select 3", "(SELECT 1 INTERSECT SELECT 2) UNION SELECT 3"},
		{"select 1 intersect select 2 intersect select 3", "SELECT 1 INTERSECT SELECT 2 INTERSECT SELECT 3"},
		{"select 1 union all select 2 except select 3 intersect select 4", "SELECT 1 UNION ALL SELECT 2 EXCEPT (SELECT 3 INTERSECT SELECT 4)"},
	}
	for _, c := range restoreCases {
		RunRestoreTest(t, c.src, c.restore, false)
	}
}

func TestLateralTableSource(t *testing.T) {
	cases := []struct {
		src     string
		restore string
	}{
		{"select * from t1, lateral (select 1) dt", "SELECT * FROM (`t1`) JOIN LATERAL (SELECT 1) AS `dt`"},
		{"select * from t1 cross join lateral (select a from t2 where t2.b=t1.b) d", "SELECT * FROM `t1` JOIN LATERAL (SELECT `a` FROM `t2` WHERE `t2`.`b`=`t1`.`b`) AS `d`"},
	}
	for _, c := range cases {
		RunRestoreTest(t, c.src, c.restore, false)
	}
}

func TestJSONTableTableSource(t *testing.T) {
	cases := []struct {
		src     string
		restore string
	}{
		{"select * from json_table('[{\"a\":1}]', '$[*]' columns (a int path '$.a')) jt", "SELECT * FROM JSON_TABLE('[{\"a\":1}]', '$[*]' COLUMNS (`a` INT PATH '$.a')) AS `jt`"},
		{"select * from json_table('[{\"a\":1}]', '$[*]' columns (id for ordinality, a int path '$.a')) jt", "SELECT * FROM JSON_TABLE('[{\"a\":1}]', '$[*]' COLUMNS (`id` FOR ORDINALITY, `a` INT PATH '$.a')) AS `jt`"},
		{"select * from json_table('[{\"a\":1}]', '$[*]' columns (nested path '$.b' columns (c int path '$.c'))) jt", "SELECT * FROM JSON_TABLE('[{\"a\":1}]', '$[*]' COLUMNS (NESTED PATH '$.b' COLUMNS (`c` INT PATH '$.c'))) AS `jt`"},
	}
	for _, c := range cases {
		RunRestoreTest(t, c.src, c.restore, false)
	}
}

// TestManagementStatements covers replication and administration statements.
// The syntax of each statement follows the MySQL 8.0 / MariaDB documentation
// linked below.
//
// https://dev.mysql.com/doc/refman/8.0/en/start-replica.html
// https://dev.mysql.com/doc/refman/8.0/en/stop-replica.html
// https://dev.mysql.com/doc/refman/8.0/en/change-master-to.html
// https://dev.mysql.com/doc/refman/8.0/en/change-replication-source-to.html
// https://dev.mysql.com/doc/refman/8.0/en/reset-master.html
// https://dev.mysql.com/doc/refman/8.0/en/reset-replica.html
// https://dev.mysql.com/doc/refman/8.0/en/reset-binary-logs-and-gtids.html
// https://dev.mysql.com/doc/refman/8.0/en/reset-persist.html
// https://dev.mysql.com/doc/refman/8.0/en/shutdown.html
// https://dev.mysql.com/doc/refman/8.0/en/handler.html
// https://dev.mysql.com/doc/refman/8.0/en/cache-index.html
// https://dev.mysql.com/doc/refman/8.0/en/load-index.html
// https://dev.mysql.com/doc/refman/8.0/en/load-xml.html
// https://dev.mysql.com/doc/refman/8.0/en/rename-user.html
func TestManagementStatements(t *testing.T) {
	cases := []string{
		// START/STOP REPLICA (SLAVE is the deprecated alias).
		"start slave",
		"start slave io_thread",
		"start replica sql_thread",
		"start slave until master_log_file='bin.000001', master_log_pos=4",
		"start replica until source_log_file='bin.000001', source_log_pos=4",
		"start replica for channel 'c1'",
		"stop slave",
		"stop replica io_thread for channel 'x'",

		// CHANGE MASTER TO / CHANGE REPLICATION SOURCE TO.
		"change master to master_host='h', master_port=3306, master_user='u', master_password='p' for channel 'c'",
		"change replication source to source_host='h', source_port=3306 for channel 'c'",

		// RESET variants.
		"reset master",
		"reset master to 42",
		"reset slave",
		"reset slave all",
		"reset replica all for channel 'c'",
		"reset binary logs and gtids",
		"reset binary logs and gtids to 1234",
		"reset query cache",
		"reset persist",
		"reset persist if exists max_connections",
		"reset persist if exists default.max_connections",

		// SHUTDOWN.
		"shutdown",
		"shutdown wait for all slaves",
		"shutdown wait for all replicas",

		// HANDLER.
		"handler t open",
		"handler t open as a",
		"handler t close",
		"handler t read a > (1, 2)",
		"handler t read idx > (1)",
		"handler t read first",
		"handler t read next limit 10",
		"handler t read idx last limit ?",
		"handler t read idx first where a = 1 limit 1, 2",
		"handler t read next where a > 0",

		// CACHE INDEX.
		"cache index t1, t2 in default",
		"cache index db.t in my_cache",
		"cache index t in cache_name",

		// LOAD INDEX INTO CACHE.
		"load index into cache t",
		"load index into cache t ignore leaves",
		"load index into cache t index (a, b) ignore leaves",
		"load index into cache db.t1 index (a), db.t2 ignore leaves",

		// LOAD XML.
		"load xml infile 'f' into table t",
		"load xml local infile 'f' replace into table t character set utf8mb4",
		"load xml infile 'f' into table t rows identified by '<row>'",
		"load xml infile 'f' into table t (@1, @2)",

		// RENAME USER.
		"rename user 'a'@'%' to 'b'@'%'",
		"rename user 'a'@'%' to 'b'@'%', current_user to 'c'@'localhost'",
	}
	for _, src := range cases {
		stmt := mustParseOne(t, src)
		restored := restoreStmt(t, stmt)
		if _, err := New().ParseOneStmt(restored, "", ""); err != nil {
			t.Fatalf("source %q: restore %q: reparse: %v", src, restored, err)
		}
	}

	// These forms are not part of the MySQL/MariaDB grammars.
	for _, src := range []string{
		"start replica until source_log_file='bin.000001', source_log_pos=4, relay_log_pos=0",
		"handler t read last",
		"cache index t in 'cache_name'",
		"install component 'file://component1' set var1 = 1",
		"install component 'file://component1' set persist_only var1 = 1",
	} {
		if _, err := New().ParseOneStmt(src, "", ""); err == nil {
			t.Fatalf("%q: expected parse error", src)
		}
	}

	restoreCases := []struct {
		src     string
		restore string
	}{
		{"handler t open as a", "HANDLER `t` OPEN AS `a`"},
		{"handler t read idx first where a = 1 limit 1", "HANDLER `t` READ `idx` FIRST WHERE `a`=1 LIMIT 1"},
		{"reset master to 42", "RESET MASTER TO 42"},
		{"reset binary logs and gtids to 1234", "RESET BINARY LOGS AND GTIDS TO 1234"},
		{"reset persist if exists default.max_connections", "RESET PERSIST IF EXISTS default.max_connections"},
		{"shutdown wait for all slaves", "SHUTDOWN WAIT FOR ALL SLAVES"},
		{"shutdown wait for all replicas", "SHUTDOWN WAIT FOR ALL SLAVES"},
		{"load index into cache db.t1 index (a), db.t2 ignore leaves", "LOAD INDEX INTO CACHE `db`.`t1` INDEX (`a`), `db`.`t2` IGNORE LEAVES"},
		{"load xml infile 'f' into table t rows identified by '<row>'", "LOAD XML INFILE 'f' INTO TABLE `t` ROWS IDENTIFIED BY '<row>'"},
		{"rename user 'a'@'%' to 'b'@'%', current_user to 'c'@'localhost'", "RENAME USER 'a'@'%' TO 'b'@'%', CURRENT_USER TO 'c'@'localhost'"},
	}
	for _, c := range restoreCases {
		RunRestoreTest(t, c.src, c.restore, false)
	}
}

// TestPersistVariables covers SET PERSIST / SET PERSIST_ONLY.
// https://dev.mysql.com/doc/refman/8.0/en/set-variable.html
func TestPersistVariables(t *testing.T) {
	cases := []struct {
		src     string
		restore string
	}{
		{"set persist max_connections = 1000", "SET @@PERSIST.`max_connections`=1000"},
		{"set persist_only max_connections = 1000", "SET @@PERSIST_ONLY.`max_connections`=1000"},
		{"set @@persist.max_connections = 1000", "SET @@PERSIST.`max_connections`=1000"},
		{"set @@persist_only.max_connections = 1000", "SET @@PERSIST_ONLY.`max_connections`=1000"},
	}
	for _, c := range cases {
		RunRestoreTest(t, c.src, c.restore, false)
	}

	setStmt, ok := mustParseOne(t, "set persist max_connections = 1000").(*ast.SetStmt)
	if !ok {
		t.Fatal("expected *ast.SetStmt")
	}
	if len(setStmt.Variables) != 1 || !setStmt.Variables[0].IsPersist || setStmt.Variables[0].IsPersistOnly {
		t.Fatalf("unexpected persist flag: %#v", setStmt.Variables)
	}

	setStmt, ok = mustParseOne(t, "set persist_only max_connections = 1000").(*ast.SetStmt)
	if !ok {
		t.Fatal("expected *ast.SetStmt")
	}
	if len(setStmt.Variables) != 1 || setStmt.Variables[0].IsPersist || !setStmt.Variables[0].IsPersistOnly {
		t.Fatalf("unexpected persist_only flag: %#v", setStmt.Variables)
	}
}

// TestImportInstallClone covers IMPORT TABLE, INSTALL/UNINSTALL PLUGIN and
// COMPONENT, and CLONE.
// https://dev.mysql.com/doc/refman/8.0/en/import-table.html
// https://dev.mysql.com/doc/refman/8.0/en/install-plugin.html
// https://dev.mysql.com/doc/refman/8.0/en/uninstall-plugin.html
// https://dev.mysql.com/doc/refman/8.0/en/install-component.html
// https://dev.mysql.com/doc/refman/8.0/en/uninstall-component.html
// https://dev.mysql.com/doc/refman/8.0/en/clone.html
func TestImportInstallClone(t *testing.T) {
	cases := []struct {
		src     string
		restore string
	}{
		{"import table from 'a.sdi'", "IMPORT TABLE FROM 'a.sdi'"},
		{"import table from 'a.sdi', 'b.sdi'", "IMPORT TABLE FROM 'a.sdi', 'b.sdi'"},
		{"install plugin myplug soname 'myplug.so'", "INSTALL PLUGIN `myplug` SONAME 'myplug.so'"},
		{"uninstall plugin myplug", "UNINSTALL PLUGIN `myplug`"},
		{"install component 'file://component1'", "INSTALL COMPONENT 'file://component1'"},
		{"install component 'file://component1', 'file://component2'",
			"INSTALL COMPONENT 'file://component1', 'file://component2'"},
		{"install component 'file://component1' set global var1 = 1", "INSTALL COMPONENT 'file://component1' SET GLOBAL `var1`=1"},
		{"install component 'file://component1' set global var1 = 1, persist var2 = on",
			"INSTALL COMPONENT 'file://component1' SET GLOBAL `var1`=1, PERSIST `var2`='ON'"},
		{"uninstall component 'file://component1', 'file://component2'",
			"UNINSTALL COMPONENT 'file://component1', 'file://component2'"},
		{"clone local data directory = '/tmp/clone'", "CLONE LOCAL DATA DIRECTORY = '/tmp/clone'"},
		{"clone instance from 'u'@'h':3306 identified by 'p'",
			"CLONE INSTANCE FROM 'u'@'h':3306 IDENTIFIED BY 'p'"},
		{"clone instance from 'u'@'h':3306 identified by 'p' require ssl",
			"CLONE INSTANCE FROM 'u'@'h':3306 IDENTIFIED BY 'p' REQUIRE SSL"},
		{"clone instance from 'u'@'h':3306 identified by 'p' data directory = '/tmp/c' require no ssl",
			"CLONE INSTANCE FROM 'u'@'h':3306 IDENTIFIED BY 'p' DATA DIRECTORY = '/tmp/c' REQUIRE NO SSL"},
	}
	for _, c := range cases {
		RunRestoreTest(t, c.src, c.restore, false)
	}

	clone, ok := mustParseOne(t, "clone instance from 'u'@'h':3306 identified by 'p'").(*ast.CloneStmt)
	if !ok {
		t.Fatal("expected *ast.CloneStmt")
	}
	if clone.Local || clone.Port != 3306 || clone.User.Username != "u" || clone.User.Hostname != "h" {
		t.Fatalf("unexpected clone statement: %#v", clone)
	}
}

// TestGetDiagnostics covers GET [CURRENT|STACKED] DIAGNOSTICS.
// https://dev.mysql.com/doc/refman/8.0/en/get-diagnostics.html
func TestGetDiagnostics(t *testing.T) {
	cases := []struct {
		src     string
		restore string
	}{
		{"get diagnostics @a = number", "GET CURRENT DIAGNOSTICS @`a` = NUMBER"},
		{"get current diagnostics @a = number, @b = row_count",
			"GET CURRENT DIAGNOSTICS @`a` = NUMBER, @`b` = ROW_COUNT"},
		{"get stacked diagnostics condition 1 @c = mysql_errno, @d = message_text",
			"GET STACKED DIAGNOSTICS CONDITION 1 @`c` = MYSQL_ERRNO, @`d` = MESSAGE_TEXT"},
		{"get diagnostics condition 1 @x = class_origin, @y = subclass_origin, @z = returned_sqlstate",
			"GET CURRENT DIAGNOSTICS CONDITION 1 @`x` = CLASS_ORIGIN, @`y` = SUBCLASS_ORIGIN, @`z` = RETURNED_SQLSTATE"},
		{"get diagnostics condition 1 @x = constraint_catalog, @y = constraint_schema, @z = constraint_name",
			"GET CURRENT DIAGNOSTICS CONDITION 1 @`x` = CONSTRAINT_CATALOG, @`y` = CONSTRAINT_SCHEMA, @`z` = CONSTRAINT_NAME"},
		{"get diagnostics condition 1 @x = catalog_name, @y = schema_name, @z = table_name, @w = column_name, @v = cursor_name",
			"GET CURRENT DIAGNOSTICS CONDITION 1 @`x` = CATALOG_NAME, @`y` = SCHEMA_NAME, @`z` = TABLE_NAME, @`w` = COLUMN_NAME, @`v` = CURSOR_NAME"},
		{"create procedure p() begin get stacked diagnostics condition 1 @a = message_text; end",
			"CREATE DEFINER = CURRENT_USER PROCEDURE `p`() BEGIN\n  GET STACKED DIAGNOSTICS CONDITION 1 @`a` = MESSAGE_TEXT;\nEND"},
	}
	for _, c := range cases {
		RunRestoreTest(t, c.src, c.restore, false)
	}

	stmt, ok := mustParseOne(t, "get stacked diagnostics condition 1 @a = message_text").(*ast.GetDiagnosticsStmt)
	if !ok {
		t.Fatal("expected *ast.GetDiagnosticsStmt")
	}
	if !stmt.Stacked || stmt.Condition == nil || len(stmt.Items) != 1 || stmt.Items[0].Name != "MESSAGE_TEXT" {
		t.Fatalf("unexpected GET DIAGNOSTICS statement: %#v", stmt)
	}

	// Statement items and condition items are not interchangeable, and unknown
	// item names are rejected.
	invalid := []string{
		"get diagnostics @a = bogus_name",
		"get diagnostics @a = message_text",
		"get diagnostics condition 1 @a = row_count",
		"get diagnostics condition 1 @a = 1",
	}
	for _, src := range invalid {
		if _, err := New().ParseOneStmt(src, "", ""); err == nil {
			t.Fatalf("%q: expected parse error", src)
		}
	}
}

// TestJSONPredicates covers the MEMBER OF operator and the IS [NOT] JSON
// predicate.
// https://dev.mysql.com/doc/refman/8.0/en/json-search-functions.html#operator_member-of
// https://mariadb.com/docs/server/reference/sql-structure/operators/comparison-operators/is-json
func TestJSONPredicates(t *testing.T) {
	cases := []struct {
		src     string
		restore string
	}{
		{"select 1 member of('[1,2]')", "SELECT 1 MEMBER OF ('[1,2]')"},
		{"select * from t where a member of(j)", "SELECT * FROM `t` WHERE `a` MEMBER OF (`j`)"},
		{"select '{}' is json", "SELECT '{}' IS JSON"},
		{"select 'x' is not json", "SELECT 'x' IS NOT JSON"},
		{"select * from t where j is json", "SELECT * FROM `t` WHERE `j` IS JSON"},
	}
	for _, c := range cases {
		RunRestoreTest(t, c.src, c.restore, false)
	}

	// MEMBER is a non-reserved keyword in MySQL 8.0.19+, so it is still usable
	// as an identifier.
	RunRestoreTest(t, "select member from t", "SELECT `member` FROM `t`", false)
	RunRestoreTest(t, "select t.member from t", "SELECT `t`.`member` FROM `t`", false)
	RunRestoreTest(t, "select a as member from t", "SELECT `a` AS `member` FROM `t`", false)
}

// TestFullTextAndSoundsLike covers MATCH ... AGAINST and SOUNDS LIKE.
// https://dev.mysql.com/doc/refman/8.0/en/fulltext-search.html
// https://dev.mysql.com/doc/refman/8.0/en/string-functions.html#operator_sounds-like
func TestFullTextAndSoundsLike(t *testing.T) {
	cases := []struct {
		src     string
		restore string
	}{
		{"select match (a, b) against ('x') from t",
			"SELECT MATCH (`a`, `b`) AGAINST ('x') FROM `t`"},
		{"select match(a) against('x' in boolean mode) from t",
			"SELECT MATCH (`a`) AGAINST ('x' IN BOOLEAN MODE) FROM `t`"},
		{"select match(a) against('x' with query expansion) from t",
			"SELECT MATCH (`a`) AGAINST ('x' WITH QUERY EXPANSION) FROM `t`"},
		{"select match(a) against('x' in natural language mode with query expansion) from t",
			"SELECT MATCH (`a`) AGAINST ('x' WITH QUERY EXPANSION) FROM `t`"},
		{"select match(a) against('x' in natural language mode) from t",
			"SELECT MATCH (`a`) AGAINST ('x') FROM `t`"},
		{"select * from t where match(a) against('x') > 0.5",
			"SELECT * FROM `t` WHERE MATCH (`a`) AGAINST ('x')>0.5"},
		{"select 'a' sounds like 'b'", "SELECT 'a' SOUNDS LIKE 'b'"},
		{"select * from t where a sounds like b and c = 1",
			"SELECT * FROM `t` WHERE `a` SOUNDS LIKE `b` AND `c`=1"},
	}
	for _, c := range cases {
		RunRestoreTest(t, c.src, c.restore, false)
	}
}

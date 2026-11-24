package myparser

import "testing"

func TestTopLevelStatementSpans(t *testing.T) {
	sqls := []string{
		"ALTER EVENT ev ENABLE",
		"alter function f deterministic",
		"alter procedure p contains sql",
		"ALTER USER IF EXISTS `root`@`localhost` IDENTIFIED BY 'new-password'",
		"alter view v as select * from t",
		"ANALYZE TABLE t",
		"begin",
		"CALL sysdate()",
		"commit",
		"create event ev on schedule at '2020-01-01 00:00:00' do select 1",
		"create function f() returns int return 1",
		"create trigger trg before insert on t for each row set @a=1",
		"CREATE USER `ttt`@`%` REQUIRE X509",
		"CREATE ROLE `test-role`",
		"create view v as select * from t",
		"DO 1",
		"drop event ev",
		"drop function f",
		"drop index a on t",
		"DROP PROCEDURE p",
		"drop table xxx",
		"drop trigger trg",
		"DROP ROLE `role`",
		"DROP USER `root`@`localhost`",
		"flush no_write_to_binlog tables tbl1 with read lock",
		"GRANT 'role1' TO 'user1'@'localhost'",
		"kill 23123",
		"PURGE BINARY LOGS",
		"return 1",
		"REVOKE 'role1' FROM 'user1'@'localhost'",
		"SET DEFAULT ROLE administrator TO `joe`@`10.0.0.1`",
		"SET PASSWORD = 'password'",
		"set role `role1`",
		"SET SESSION TRANSACTION ISOLATION LEVEL REPEATABLE READ",
		"TRUNCATE TABLE t1",
		"BACKUP LOCK t",
		"SHUTDOWN",
	}
	for _, sql := range sqls {
		stmt, err := New().ParseOneStmt(sql, "", "")
		if err != nil {
			t.Errorf("parse %q: %v", sql, err)
			continue
		}
		if !stmt.Span().Valid() {
			t.Errorf("%q -> %T has invalid span %v", sql, stmt, stmt.Span())
		}
	}
}

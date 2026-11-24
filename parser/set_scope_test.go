package myparser

import (
	"strings"
	"testing"
)

// TestSetBareScopeRejected covers the parser bug where "SET GLOBAL = DEFAULT" and
// the other four bare scope forms were accepted; both MySQL and MariaDB reject a
// SET with a scope keyword but no variable name.
func TestSetBareScopeRejected(t *testing.T) {
	rejected := []string{
		"SET GLOBAL = DEFAULT",
		"SET LOCAL = DEFAULT",
		"SET PERSIST = DEFAULT",
		"SET PERSIST_ONLY = DEFAULT",
		"SET SESSION = DEFAULT",
		"set global = default",
		"SET GLOBAL = 1",
	}
	for _, sql := range rejected {
		_, err := New().ParseOneStmt(sql, "", "")
		if err == nil {
			t.Errorf("%q: expected rejection, parsed", sql)
		} else if !strings.Contains(err.Error(), "requires a variable name") {
			t.Errorf("%q: unexpected error %v", sql, err)
		}
	}
	accepted := []string{
		"SET GLOBAL max_connections = 100",
		"SET LOCAL autocommit = 1",
		"SET PERSIST max_connections = 100",
		"SET PERSIST_ONLY max_connections = 100",
		"SET SESSION sql_mode = ''",
		"SET NAMES DEFAULT",
		"SET @x = 1",
		"SET @@global.max_connections = 100",
		"SET GLOBAL TRANSACTION READ ONLY",
		"SET TRANSACTION ISOLATION LEVEL REPEATABLE READ",
	}
	for _, sql := range accepted {
		if _, err := New().ParseOneStmt(sql, "", ""); err != nil {
			t.Errorf("%q: expected accept, got %v", sql, err)
		}
	}
}

package myparser

import "testing"

// Per-operand ORDER BY/LIMIT is only valid inside parentheses; MySQL rejects a
// bare operand that carries them before another UNION/EXCEPT/INTERSECT.
func TestSetOperationPaginationPlacement(t *testing.T) {
	for _, tc := range []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{
			name:    "mysql parser.test 1355",
			sql:     "SELECT a FROM t1 UNION SELECT a FROM t1 ORDER BY a UNION SELECT a FROM t1",
			wantErr: true,
		},
		{
			name:    "order by before union",
			sql:     "SELECT a FROM t1 ORDER BY a UNION SELECT a FROM t1",
			wantErr: true,
		},
		{
			name:    "limit before union",
			sql:     "SELECT a FROM t1 UNION SELECT a FROM t1 LIMIT 1 UNION SELECT a FROM t1",
			wantErr: true,
		},
		{
			name:    "except after ordered operand",
			sql:     "SELECT a FROM t1 ORDER BY a EXCEPT SELECT a FROM t1",
			wantErr: true,
		},
		{
			name: "trailing order by",
			sql:  "SELECT a FROM t1 UNION SELECT a FROM t1 ORDER BY a",
		},
		{
			name: "trailing limit",
			sql:  "SELECT a FROM t1 UNION SELECT a FROM t1 LIMIT 1",
		},
		{
			name: "parenthesized first operand",
			sql:  "(SELECT a FROM t1 ORDER BY a) UNION SELECT a FROM t1",
		},
		{
			name: "parenthesized last operand",
			sql:  "SELECT a FROM t1 UNION (SELECT a FROM t1 ORDER BY a)",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New().ParseOneStmt(tc.sql, "", "")
			if tc.wantErr {
				if err == nil {
					t.Fatal("parse succeeded, want syntax error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
		})
	}
}

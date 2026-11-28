// Local regression tests prepared for the parser package migration; they
// exercise the public parse/AST/restore/visitor APIs only. Shared helpers
// (mustParseOne, restoreStmt, assertParseErrors, assertVisitorControls,
// syntaxVisitor, isValueOrLiteral, replaceLiteral) are provided by the
// roundtrip-helpers test file and are intentionally not duplicated here.
package parser_test

import (
	"testing"

	"github.com/mia0x75/myparser/parser"
)

func localRunRestoreTest(t *testing.T, sourceSQLs, expectSQLs string, enableWindowFunc bool) {
	t.Helper()
	p := parser.New()
	p.EnableWindowFunc(enableWindowFunc)
	stmts, _, err := p.ParseWithWarnings(sourceSQLs)
	if err != nil {
		t.Fatalf("source %s: got err %v, want nil", sourceSQLs, err)
	}
	var restored string
	for _, stmt := range stmts {
		sql := restoreStmt(t, stmt)
		if sql == "" {
			continue
		}
		if _, err := p.ParseOneStmt(sql); err != nil {
			t.Fatalf("source %s; re-parse restored: %v", sourceSQLs, err)
		}
		restored = sql
	}
	if restored != expectSQLs {
		t.Fatalf("source %s; restored = %q, want %q", sourceSQLs, restored, expectSQLs)
	}
}

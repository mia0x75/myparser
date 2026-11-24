package myparser_test

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser"
	_ "github.com/mia0x75/myparser/driver"
)

func TestRejectOverflowingIntegerFields(t *testing.T) {
	for _, sql := range []string{
		"create table t (a varchar(18446744073709551615))",
		"create resource group rg type = system thread_priority 18446744073709551615",
	} {
		if _, err := myparser.New().ParseOneStmt(sql, "", ""); err == nil {
			t.Fatalf("overflow accepted: %s", sql)
		}
	}
}

// TestOutOfRangeIntegerLiteralFallsBackToDecimal pins the MySQL-compatible
// behaviour: an integer literal that exceeds the decimal precision (81 digits)
// is truncated to the maximum decimal value and a warning is emitted.
func TestOutOfRangeIntegerLiteralFallsBackToDecimal(t *testing.T) {
	// 82 digits overflows the MyDecimal wordBufLen (9 words × 9 digits = 81).
	// MySQL truncates to the max decimal value and emits a warning.
	overflow := "1" + strings.Repeat("9", 81)
	_, warns, err := myparser.New().Parse("select "+overflow, "", "")
	if err != nil {
		t.Fatalf("parse returned error, want nil: %v", err)
	}
	if len(warns) == 0 {
		t.Fatal("expected a truncation warning, got none")
	}
}

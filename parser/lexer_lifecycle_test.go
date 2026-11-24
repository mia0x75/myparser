package parser

import (
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
)

func TestScannerEmptyStatementText(t *testing.T) {
	var scanner Scanner
	scanner.reset("")
	if got := scanner.stmtText(); got != "" {
		t.Fatalf("empty scanner text: %q", got)
	}
}

func TestScannerResetReleasesDiagnostics(t *testing.T) {
	scanner := NewScanner("SELECT 1")
	scanner.errs = append(scanner.errs, errors.New("MOCK_ERROR"))
	scanner.warns = append(scanner.warns, errors.New("MOCK_WARNING"))
	errorsBacking, warningsBacking := scanner.errs, scanner.warns
	scanner.reset("SELECT 2")
	if len(scanner.errs) != 0 || len(scanner.warns) != 0 {
		t.Fatal("reset kept diagnostics")
	}
	if errorsBacking[0] != nil || warningsBacking[0] != nil {
		t.Fatal("reset retained diagnostic references in backing arrays")
	}
}

func TestScannerResetDropsLargeBuffer(t *testing.T) {
	scanner := NewScanner("'" + strings.Repeat("a\\n", 40000) + "'")
	var v yySymType
	scanner.Lex(&v)
	if scanner.buf.Cap() <= maxRetainedScanBuffer {
		t.Fatal("test input did not grow the decode buffer")
	}
	scanner.reset("")
	if scanner.buf.Cap() > maxRetainedScanBuffer {
		t.Fatalf("reset kept a %d byte decode buffer", scanner.buf.Cap())
	}
}

func TestNormalizeClearsTokenReferences(t *testing.T) {
	d := &sqlDigester{lexer: NewScanner(""), hasher: sha256.New()}
	d.doNormalize("/*" + strings.Repeat("x", 1<<16) + "*/ SELECT 1")
	for i, token := range d.tokens[:cap(d.tokens)] {
		if token.lit != "" || token.tok != 0 {
			t.Fatalf("slot %d kept token %#v", i, token)
		}
	}
}

func TestUnterminatedTokensRejected(t *testing.T) {
	for _, sql := range []string{
		"SELECT 1 `unterminated",
		"SELECT 1 X'AB",
		"SELECT 1 B'01",
	} {
		t.Run(sql, func(t *testing.T) {
			if _, err := New().ParseOneStmt(sql, "", ""); err == nil {
				t.Fatalf("accepted truncated input %q", sql)
			}
		})
	}
}

func TestLastErrorAsWarnReleasesErrorSlot(t *testing.T) {
	st := &parseState{}
	err := errors.New("MOCK_ERROR")
	st.lexer.errs = append(st.lexer.errs, err)
	backing := st.lexer.errs
	st.lastErrorAsWarn()
	if len(st.lexer.errs) != 0 || len(st.lexer.warns) != 1 || st.lexer.warns[0] != err {
		t.Fatal("diagnostic promotion lost the warning")
	}
	if backing[0] != nil {
		t.Fatal("promotion retained a duplicate error reference")
	}
}

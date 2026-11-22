package myparser

import (
	"errors"
	"testing"
)

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

package mysql

import (
	"testing"
)

func TestSQLError(t *testing.T) {
	e := NewErrf(ErrNoDB, "no db error")
	if len(e.Error()) == 0 {
		t.Fatalf("got %d, want > 0", len(e.Error()))
	}

	e = NewErrf(0, "customized error")
	if len(e.Error()) == 0 {
		t.Fatalf("got %d, want > 0", len(e.Error()))
	}

	e = NewErr(ErrNoDB)
	if len(e.Error()) == 0 {
		t.Fatalf("got %d, want > 0", len(e.Error()))
	}

	e = NewErr(0, "customized error")
	if len(e.Error()) == 0 {
		t.Fatalf("got %d, want > 0", len(e.Error()))
	}
}

func TestJSONVacuousPathErrorMapping(t *testing.T) {
	if got := NewErr(ErrJSONVacuousPath).Code; got != 3153 {
		t.Fatalf("JSON vacuous path error code = %d, want 3153", got)
	}
	if got := NewErr(ErrJSONVacuousPath).State; got != "42000" {
		t.Fatalf("JSON vacuous path SQLSTATE = %q, want 42000", got)
	}
}

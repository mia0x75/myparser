package opcode

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/format"
)

func TestT(t *testing.T) {
	op := Plus
	if op.String() != "plus" {
		t.Fatalf("invalid op code")
	}

	if len(Ops) != len(opsLiteral) {
		t.Error("inconsistent count ops and opsliteral")
	}
	var buf bytes.Buffer
	for op := range Ops {
		op.Format(&buf)
		if buf.String() != opsLiteral[op] {
			t.Error("format op fail", op)
		}
		buf.Reset()
	}

	// Test invalid opcode
	defer func() {
		_ = recover()
	}()

	op = 0
	s := op.String()
	if len(s) > 0 {
		t.Fail()
	}
}

func TestOpRestore(t *testing.T) {
	var sb strings.Builder
	if err := Plus.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
		t.Fatal(err)
	}
	if got := sb.String(); got != opsLiteral[Plus] {
		t.Fatalf("Restore: got %#v, want %#v", got, opsLiteral[Plus])
	}

	// Invalid opcode must return an error, not panic.
	sb.Reset()
	if err := Op(0).Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err == nil {
		t.Fatal("expected error for invalid opcode")
	}
}

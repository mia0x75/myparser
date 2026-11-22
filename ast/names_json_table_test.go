package ast

import (
	"testing"

	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

// p59-59.1: JSON_TABLE name values use lex.Ident and presence checks.
func TestJSONTableNamePresence(t *testing.T) {
	ordinality := &JSONTableColumn{Name: lex.Ident{Name: "ord"}, Kind: JSONTableColumnOrdinality}
	if !ordinality.Name.Present() {
		t.Fatal("ordinality column should keep its name present")
	}
	if got := restoreNode(t, ordinality); got != "`ord` FOR ORDINALITY" {
		t.Fatalf("ordinality Restore() = %q, want %q", got, "`ord` FOR ORDINALITY")
	}

	quotedEmpty := &JSONTableColumn{
		Name: lex.Ident{Name: "", Raw: "``", Quote: lex.QuoteBacktick, Span: loc.InvalidSpan()},
		Kind: JSONTableColumnPath,
		Type: NewTypeName("INT"),
		Path: "$",
	}
	if !quotedEmpty.Name.Present() {
		t.Fatal("quoted-empty name should be present")
	}
	if got := restoreNode(t, quotedEmpty); got != "`` INT PATH '$'" {
		t.Fatalf("quoted-empty Restore() = %q, want %q", got, "`` INT PATH '$'")
	}

	absent := &JSONTableColumn{Kind: JSONTableColumnPath, Type: NewTypeName("INT"), Path: "$"}
	if absent.Name.Present() {
		t.Fatal("zero Ident should not be present")
	}
	if err := absent.Restore(newJSONRestoreCtx()); err == nil {
		t.Fatal("path column without a name should fail restore")
	}
}

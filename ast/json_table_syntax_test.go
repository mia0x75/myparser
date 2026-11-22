package ast

import (
	"io"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

// jsonLitExpr builds a minimal integer literal for behavior values.
func jsonLitExpr(_ *testing.T, value string) ExprNode {
	return &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: value}}
}

// jsonNopVisitor passes every node through unchanged.
type jsonNopVisitor struct{}

func (jsonNopVisitor) Enter(in Node) (Node, bool) { return in, false }
func (jsonNopVisitor) Leave(in Node) (Node, bool) { return in, true }

func TestJSONTableColumnKindsRestore(t *testing.T) {
	expr := jsonLitExpr(t, "7")
	nested := &JSONTableColumn{
		Kind:    JSONTableColumnNested,
		Path:    "$.n",
		Columns: []*JSONTableColumn{{Name: lex.Ident{Name: "x"}, Kind: JSONTableColumnPath, Type: NewTypeName("INT"), Path: "$.x"}},
	}
	table := &JSONTable{
		Expr: expr,
		Path: "$.a",
		Columns: []*JSONTableColumn{
			{Name: lex.Ident{Name: "ord"}, Kind: JSONTableColumnOrdinality},
			{Name: lex.Ident{Name: "v"}, Kind: JSONTableColumnPath, Type: NewTypeName("INT"), Path: "$.v"},
			{Name: lex.Ident{Name: "e"}, Kind: JSONTableColumnExists, Type: NewTypeName("INT"), Path: "$.e"},
			nested,
		},
	}
	want := "JSON_TABLE(7, '$.a' COLUMNS (`ord` FOR ORDINALITY, `v` INT PATH '$.v', `e` INT EXISTS PATH '$.e', NESTED '$.n' COLUMNS (`x` INT PATH '$.x')))"
	if got := restoreNode(t, table); got != want {
		t.Fatalf("JSONTable.Restore() = %q, want %q", got, want)
	}

	keywordNested := *nested
	keywordNested.PathKeyword = true
	if got, want := restoreNode(t, &keywordNested), "NESTED PATH '$.n' COLUMNS (`x` INT PATH '$.x')"; got != want {
		t.Fatalf("nested PATH spell Restore() = %q, want %q", got, want)
	}
	if err := (&JSONTableColumn{Kind: JSONTableColumnNested}).Restore(newJSONRestoreCtx()); err == nil {
		t.Fatal("nested column with no children should fail restore")
	}

	// Ordinality, path, and exists columns reject type, path, nested columns, and behaviors.
	cases := []*JSONTableColumn{
		{Kind: JSONTableColumnOrdinality, Type: NewTypeName("INT")},
		{Kind: JSONTableColumnOrdinality, Path: "$"},
		{Kind: JSONTableColumnOrdinality, Behaviors: []*JSONTableBehavior{{Kind: JSONTableBehaviorNull, On: JSONTableBehaviorOnEmpty}}},
		{Kind: JSONTableColumnPath, PathKeyword: true},
		{Kind: JSONTableColumnNested, Name: lex.Ident{Name: "n"}},
		{Kind: JSONTableColumnNested, Type: NewTypeName("INT")},
		{Kind: JSONTableColumnNested, Path: "$.n"},
	}
	for i, column := range cases {
		if _, ok := column.Accept(jsonNopVisitor{}); ok {
			t.Fatalf("case %d Accept accepted an invalid column shape: %+v", i, column)
		}
	}
}

func TestJSONTableBehaviorSequenceRestore(t *testing.T) {
	literal := jsonLitExpr(t, "1")
	column := &JSONTableColumn{
		Name: lex.Ident{Name: "c"},
		Kind: JSONTableColumnPath,
		Type: NewTypeName("INT"),
		Path: "$.c",
		Behaviors: []*JSONTableBehavior{
			{Kind: JSONTableBehaviorError, On: JSONTableBehaviorOnEmpty},
			{Kind: JSONTableBehaviorDefault, On: JSONTableBehaviorOnError, Value: literal},
		},
	}
	want := "`c` INT PATH '$.c' ERROR ON EMPTY DEFAULT 1 ON ERROR"
	if got := restoreNode(t, column); got != want {
		t.Fatalf("JSONTableColumn.Restore() = %q, want %q", got, want)
	}
	if _, ok := column.Accept(jsonNopVisitor{}); !ok {
		t.Fatal("column with source-ordered behaviors should accept a nop visitor")
	}
}

func TestJSONTableBehaviorAcceptAndRestoreRejects(t *testing.T) {
	behavior := &JSONTableBehavior{Kind: JSONTableBehaviorDefault, On: JSONTableBehaviorOnError, Value: jsonLitExpr(t, "2")}
	if _, ok := behavior.Accept(jsonNopVisitor{}); !ok {
		t.Fatal("valid default behavior should accept a nop visitor")
	}
	if got, want := restoreNode(t, behavior), " DEFAULT 2 ON ERROR"; got != want {
		t.Fatalf("JSONTableBehavior.Restore() = %q, want %q", got, want)
	}

	invalid := []*JSONTableBehavior{
		{Kind: JSONTableBehaviorNull, On: JSONTableBehaviorOnError, Value: jsonLitExpr(t, "3")},
		{Kind: JSONTableBehaviorDefault, On: JSONTableBehaviorOnEmpty},
		{Kind: JSONTableBehaviorUnknown, On: JSONTableBehaviorOnEmpty},
		{Kind: JSONTableBehaviorNull, On: JSONTableBehaviorOnUnknown},
		{Kind: JSONTableBehaviorUnknown, On: JSONTableBehaviorOnUnknown},
	}
	for i, b := range invalid {
		if _, ok := b.Accept(jsonNopVisitor{}); ok {
			t.Fatalf("case %d Accept accepted an invalid behavior: %+v", i, b)
		}
		if err := b.Restore(newJSONRestoreCtx()); err == nil {
			t.Fatalf("case %d Restore accepted an invalid behavior: %+v", i, b)
		}
	}

	replaceVisitor := replaceJSONBehaviorVisitor{behavior: behavior}
	column := &JSONTableColumn{Name: lex.Ident{Name: "c"}, Kind: JSONTableColumnPath, Type: NewTypeName("INT"), Behaviors: []*JSONTableBehavior{behavior}}
	if _, ok := column.Accept(&replaceVisitor); !ok {
		t.Fatal("behavior replacement should be accepted")
	}
	if got := restoreNode(t, column); got != "`c` INT PATH '' DEFAULT 9 ON ERROR" {
		t.Fatalf("behavior replacement was not retained: %q", got)
	}
}

// replaceJSONBehaviorVisitor replaces the default behavior value.
type replaceJSONBehaviorVisitor struct{ behavior *JSONTableBehavior }

func (v *replaceJSONBehaviorVisitor) Enter(n Node) (Node, bool) {
	behavior, ok := n.(*JSONTableBehavior)
	if !ok || behavior != v.behavior {
		return n, false
	}
	replacement := *behavior
	replacement.Value = jsonLitExpr(nil, "9")
	return &replacement, false
}

func (v *replaceJSONBehaviorVisitor) Leave(n Node) (Node, bool) { return n, true }

func TestJSONTableBehaviorsWriterError(t *testing.T) {
	column := &JSONTableColumn{
		Name: lex.Ident{Name: "c"},
		Kind: JSONTableColumnPath,
		Path: "$.c",
		Type: NewTypeName("INT"),
		Behaviors: []*JSONTableBehavior{
			{Kind: JSONTableBehaviorNull, On: JSONTableBehaviorOnEmpty},
		},
	}
	if err := column.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, failWriter{})); err == nil {
		t.Fatal("restore through a failing writer should report an error")
	}
	if err := restoreJSONTableColumns(format.NewRestoreCtx(format.DefaultRestoreFlags, failWriter{}), []*JSONTableColumn{column}); err == nil {
		t.Fatal("restoreJSONTableColumns through a failing writer should report an error")
	}
}

// failWriter rejects every write.
type failWriter struct{}

func (failWriter) Write(p []byte) (int, error) { return 0, io.ErrShortWrite }

// newJSONRestoreCtx builds a default restore context that discards output.
func newJSONRestoreCtx() *format.RestoreCtx {
	return format.NewRestoreCtx(format.DefaultRestoreFlags, &strings.Builder{})
}

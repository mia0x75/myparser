package ast_test

import (
	"strings"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"

	"github.com/mia0x75/myparser/ast"
	dformat "github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/parser"
)

// literalRestore restores a literal node with the default MySQL flags.
func literalRestore(t *testing.T, node *ast.LiteralExpr) (string, error) {
	t.Helper()
	var out strings.Builder
	err := node.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out))
	return out.String(), err
}

// TestLiteralRestore checks the canonical SQL spelling of every restored
// literal kind.
func TestLiteralRestore(t *testing.T) {
	for _, tc := range []struct {
		name  string
		kind  pkast.LiteralKind
		value string
		want  string
	}{
		{"null", pkast.LiteralNull, "null", "NULL"},
		{"null without value", pkast.LiteralNull, "", "NULL"},
		{"boolean true", pkast.LiteralBoolean, "true", "TRUE"},
		{"boolean false", pkast.LiteralBoolean, "false", "FALSE"},
		{"boolean mixed case true", pkast.LiteralBoolean, "TrUe", "TRUE"},
		{"boolean mixed case false", pkast.LiteralBoolean, "FaLsE", "FALSE"},
		{"boolean upper case true", pkast.LiteralBoolean, "TRUE", "TRUE"},
		{"boolean upper case false", pkast.LiteralBoolean, "FALSE", "FALSE"},
		{"integer", pkast.LiteralInteger, "42", "42"},
		{"decimal", pkast.LiteralDecimal, "1.50", "1.50"},
		{"float", pkast.LiteralFloat, "1e-12", "1e-12"},
		{"float normalizes exponent", pkast.LiteralFloat, "1e+05", "1e+05"},
		{"string", pkast.LiteralString, "abc", "'abc'"},
		{"bytes", pkast.LiteralBytes, "a\x00b", "'a\x00b'"},
		{"hex odd digits", pkast.LiteralHex, "F", "x'0f'"},
		{"hex byte width", pkast.LiteralHex, "00AF", "x'00af'"},
		{"hex empty", pkast.LiteralHex, "", "x''"},
		{"bit leading zeros", pkast.LiteralBit, "000101", "b'101'"},
		{"bit all zeros", pkast.LiteralBit, "0000", "b'0'"},
		{"bit empty", pkast.LiteralBit, "", "b''"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			node := &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: tc.kind, Value: tc.value}}
			got, err := literalRestore(t, node)
			if err != nil {
				t.Fatalf("Restore failed: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Restore = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestLiteralRestoreInvalidBoolean checks that an unknown boolean spelling
// fails instead of being written out as SQL.
func TestLiteralRestoreInvalidBoolean(t *testing.T) {
	for _, value := range []string{"", "1", "yes", "unknown", "truex"} {
		t.Run(value, func(t *testing.T) {
			node := &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralBoolean, Value: value}}
			got, err := literalRestore(t, node)
			if err == nil {
				t.Fatalf("Restore of %q succeeded, want an error", value)
			}
			if !strings.Contains(err.Error(), "boolean") {
				t.Fatalf("error %q does not name the boolean literal", err)
			}
			if got != "" {
				t.Fatalf("Restore of %q wrote %q, want no output", value, got)
			}
		})
	}
}

// TestLiteralRestoreKeywordFlags checks that the keyword case flags drive the
// restored spelling of NULL and the boolean keywords.
func TestLiteralRestoreKeywordFlags(t *testing.T) {
	for _, tc := range []struct {
		name  string
		kind  pkast.LiteralKind
		value string
		want  string
	}{
		{"null", pkast.LiteralNull, "null", "null"},
		{"boolean true", pkast.LiteralBoolean, "true", "true"},
		{"boolean false", pkast.LiteralBoolean, "false", "false"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			node := &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: tc.kind, Value: tc.value}}
			var out strings.Builder
			ctx := dformat.NewRestoreCtx(dformat.RestoreKeyWordLowercase, &out)
			if err := node.Restore(ctx); err != nil {
				t.Fatalf("Restore failed: %v", err)
			}
			if out.String() != tc.want {
				t.Fatalf("Restore = %q, want %q", out.String(), tc.want)
			}
		})
	}
}

// TestLiteralFormatKeepsCanonicalKeywords checks that the legacy Format path
// renders the same canonical keywords as Restore.
func TestLiteralFormatKeepsCanonicalKeywords(t *testing.T) {
	for _, tc := range []struct {
		name  string
		kind  pkast.LiteralKind
		value string
		want  string
	}{
		{"null", pkast.LiteralNull, "null", "NULL"},
		{"boolean true", pkast.LiteralBoolean, "true", "TRUE"},
		{"boolean false", pkast.LiteralBoolean, "false", "FALSE"},
		{"integer", pkast.LiteralInteger, "42", "42"},
		{"string", pkast.LiteralString, "a\n\"b", `"a\n\"b"`},
		{"bytes", pkast.LiteralBytes, "a\x00b", `"a\x00b"`},
		{"hex", pkast.LiteralHex, "00AF", "x'00af'"},
		{"bit", pkast.LiteralBit, "0001", "b'1'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			node := &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: tc.kind, Value: tc.value}}
			var out strings.Builder
			node.Format(&out)
			if out.String() != tc.want {
				t.Fatalf("Format = %q, want %q", out.String(), tc.want)
			}
		})
	}
}

func TestLiteralRestoreInvalidBinary(t *testing.T) {
	for _, tc := range []struct {
		name  string
		kind  pkast.LiteralKind
		value string
	}{
		{"hex quote", pkast.LiteralHex, "00'"},
		{"hex outside range", pkast.LiteralHex, "fg"},
		{"hex Unicode", pkast.LiteralHex, "Ｆ"},
		{"bit nonbinary digit", pkast.LiteralBit, "012"},
		{"bit whitespace", pkast.LiteralBit, "01 "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			node := &ast.LiteralExpr{
				BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: tc.kind, Value: tc.value},
				Introducer:      lex.Ident{Name: "binary"},
			}
			if got, err := literalRestore(t, node); err == nil || got != "" {
				t.Fatalf("invalid literal wrote %q, error = %v", got, err)
			}
		})
	}
}

func TestLiteralRestoreRejectsInvalidIntroducer(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind pkast.LiteralKind
	}{
		{"string", pkast.LiteralString},
		{"hex", pkast.LiteralHex},
	} {
		t.Run(tc.name, func(t *testing.T) {
			node := &ast.LiteralExpr{
				BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: tc.kind, Value: "1"},
				Introducer:      lex.Ident{Name: "utf8; DROP TABLE users; --"},
			}
			if got, err := literalRestore(t, node); err == nil || got != "" {
				t.Fatalf("Restore wrote %q, error = %v; want no output and an error", got, err)
			}
		})
	}
}

func TestLiteralRestoreRejectsInvalidNumericText(t *testing.T) {
	for _, tc := range []struct {
		name  string
		kind  pkast.LiteralKind
		value string
	}{
		{"integer statement injection", pkast.LiteralInteger, "1); DROP TABLE users; --"},
		{"decimal suffix", pkast.LiteralDecimal, "1.2 + 3"},
		{"incomplete exponent", pkast.LiteralFloat, "1e+"},
		{"sign only", pkast.LiteralInteger, "-"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			node := &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: tc.kind, Value: tc.value}}
			if got, err := literalRestore(t, node); err == nil || got != "" {
				t.Fatalf("Restore wrote %q, error = %v; want no output and an error", got, err)
			}
		})
	}
}

func TestStatementRestoreRejectsModifiedNumericLiteral(t *testing.T) {
	stmt, err := parser.New().ParseOneStmt("SELECT 1")
	if err != nil {
		t.Fatal(err)
	}
	literal := stmt.(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.LiteralExpr)
	literal.Value = "1); DROP TABLE users; --"

	var out strings.Builder
	err = stmt.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out))
	if err == nil {
		t.Fatal("Restore accepted modified numeric syntax")
	}
	if strings.Contains(out.String(), "DROP TABLE") {
		t.Fatalf("Restore emitted injected SQL: %q", out.String())
	}
}

func TestLiteralRestoreAcceptsNumericGrammar(t *testing.T) {
	for _, tc := range []struct {
		name  string
		kind  pkast.LiteralKind
		value string
	}{
		{"integer", pkast.LiteralInteger, "007"},
		{"signed integer", pkast.LiteralInteger, "-7"},
		{"leading decimal point", pkast.LiteralDecimal, ".5"},
		{"trailing decimal point", pkast.LiteralDecimal, "1."},
		{"exponent", pkast.LiteralFloat, "1e-12"},
		{"signed exponent", pkast.LiteralFloat, "+1.25E+6"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			node := &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: tc.kind, Value: tc.value}}
			got, err := literalRestore(t, node)
			if err != nil || got != tc.value {
				t.Fatalf("Restore = %q, %v; want %q", got, err, tc.value)
			}
		})
	}
}

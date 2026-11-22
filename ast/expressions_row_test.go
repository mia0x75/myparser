package ast_test

import (
	"strings"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
)

// rowTestLiteral builds a syntax-only integer literal for row tests.
func rowTestLiteral(value string) ast.ExprNode {
	return &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: value}}
}

func TestRowSharedCoreIdentity(t *testing.T) {
	row := &ast.RowExpr{}
	var node pkast.RowNode = row
	if node.RowCore() != &row.BaseRowExpr {
		t.Fatal("row core must return the original embedded base")
	}
	child := &sharedPureExpr{}
	node.RowCore().Items = []ast.ExprNode{child}
	row.SetSpan(loc.Span{Start: 0, End: 0})
	if len(row.Items) != 1 || row.Items[0] != child || node.RowCore().Span() != (loc.Span{}) {
		t.Fatal("row core lost item or span identity")
	}
}

func TestRowRestoreKeepsDialectSyntax(t *testing.T) {
	for _, tc := range []struct {
		name       string
		items      []ast.ExprNode
		omit       bool
		want       string
		formatWant string
	}{
		{"row", []ast.ExprNode{rowTestLiteral("1"), rowTestLiteral("2")}, false, "ROW(1,2)", "row(1,2)"},
		{"omit keyword", []ast.ExprNode{rowTestLiteral("1"), rowTestLiteral("2")}, true, "(1,2)", "(1,2)"},
		{"empty row", nil, false, "ROW()", "row()"},
		{"empty omitted row", nil, true, "()", "()"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := &ast.RowExpr{BaseRowExpr: pkast.BaseRowExpr{Items: tc.items}, OmitRowKeyword: tc.omit}
			var out strings.Builder
			if err := row.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
				t.Fatal(err)
			}
			if out.String() != tc.want {
				t.Fatalf("restore = %q, want %q", out.String(), tc.want)
			}
			out.Reset()
			row.Format(&out)
			if out.String() != tc.formatWant {
				t.Fatalf("format = %q, want %q", out.String(), tc.formatWant)
			}
		})
	}
}

func TestRowRestoreRejectsInvalidItems(t *testing.T) {
	for _, tc := range []struct {
		name string
		item ast.ExprNode
	}{
		{"nil item", nil},
		{"child error", &sharedPureExpr{restoreErr: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := &ast.RowExpr{BaseRowExpr: pkast.BaseRowExpr{Items: []ast.ExprNode{rowTestLiteral("1"), tc.item}}}
			var out strings.Builder
			err := row.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out))
			if err == nil || !strings.Contains(err.Error(), "RowExpr.Items[1]") {
				t.Fatalf("missing row item error context: %v", err)
			}
		})
	}
}

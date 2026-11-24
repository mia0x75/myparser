package ast_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	dformat "github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

func typeTestLiteral(kind pkast.LiteralKind, value string) *ast.LiteralExpr {
	return &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: kind, Value: value}}
}

func TestTypeNameRestore(t *testing.T) {
	for _, tc := range []struct {
		name, keyword, normal, cast string
		modifiers                   []string
	}{
		{"decimal default", "DECIMAL", "DECIMAL", "DECIMAL(11)", nil},
		{"decimal width", "DECIMAL", "DECIMAL", "DECIMAL(12)", []string{"12"}},
		{"decimal scale", "DECIMAL", "DECIMAL(12,3)", "DECIMAL(12, 3)", []string{"12", "3"}},
		{"decimal zero scale", "DECIMAL", "DECIMAL(12,0)", "DECIMAL(12, 0)", []string{"12", "0"}},
		{"explicit zero fsp", "DATETIME", "DATETIME", "DATETIME(0)", []string{"0"}},
		{"overflow fsp", "TIME", "TIME", "TIME(18446744073709551615)", []string{"18446744073709551615"}},
		{"float normalization", "FLOAT", "DOUBLE(25,3)", "DOUBLE", []string{"25", "3"}},
		{"integer cast", "INT", "INT", "INT", nil},
		{"unsized binary cast", "BINARY", "BINARY", "BINARY", nil},
		{"sized binary cast", "BINARY", "BINARY(8)", "BINARY(8)", []string{"8"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			node := ast.NewTypeName(tc.keyword)
			for _, value := range tc.modifiers {
				node.Modifiers = append(node.Modifiers, typeTestLiteral(pkast.LiteralInteger, value))
			}
			var normal, cast strings.Builder
			if err := node.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &normal)); err != nil {
				t.Fatal(err)
			}
			if err := node.RestoreAsCastType(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &cast)); err != nil {
				t.Fatal(err)
			}
			if normal.String() != tc.normal || cast.String() != tc.cast {
				t.Fatalf("Restore = %q/%q, want %q/%q", normal.String(), cast.String(), tc.normal, tc.cast)
			}
		})
	}
}

func TestTypeNameQualifiersAndElements(t *testing.T) {
	for _, tc := range []struct {
		name string
		node *ast.TypeName
		want string
	}{
		{"binary charset", &ast.TypeName{BaseTypeName: pkast.BaseTypeName{Keywords: []string{"CHAR"}},
			Charset: lex.Ident{Name: "BiNaRy"}}, "BINARY"},
		{"explicit qualifiers", &ast.TypeName{BaseTypeName: pkast.BaseTypeName{Keywords: []string{"VARCHAR"}},
			Charset: lex.Ident{Name: "UtF8"}, Collation: lex.Ident{Name: "UtF8_BiN"}, Binary: true},
			"VARCHAR BINARY CHARACTER SET utf8 COLLATE utf8_bin"},
		{"unsigned zerofill", &ast.TypeName{BaseTypeName: pkast.BaseTypeName{Keywords: []string{"BIGINT"}},
			Unsigned: true, Zerofill: true}, "BIGINT UNSIGNED ZEROFILL"},
		{"enum quotes", &ast.TypeName{BaseTypeName: pkast.BaseTypeName{Keywords: []string{"ENUM"},
			Modifiers: []ast.ExprNode{typeTestLiteral(pkast.LiteralString, "a'b"), typeTestLiteral(pkast.LiteralString, "")}}},
			"ENUM('a''b','')"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := tc.node.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); err != nil {
				t.Fatal(err)
			}
			if out.String() != tc.want {
				t.Fatalf("Restore = %q, want %q", out.String(), tc.want)
			}
		})
	}
}

func TestTypeNameRestoreRejectsConflictingNameAndKeywords(t *testing.T) {
	for _, tc := range []struct {
		name string
		node *ast.TypeName
	}{
		{"unknown keyword", &ast.TypeName{BaseTypeName: pkast.BaseTypeName{Keywords: []string{"INT); DROP TABLE users; --"}}}},
		{"name and keyword", &ast.TypeName{BaseTypeName: pkast.BaseTypeName{
			Name:     pkast.QualifiedName{Parts: []lex.Ident{{Name: "custom_type"}}},
			Keywords: []string{"INT); DROP TABLE users; --"},
		}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, restore := range []struct {
				name string
				fn   func(*dformat.RestoreCtx) error
			}{
				{"type", tc.node.Restore},
				{"cast type", tc.node.RestoreAsCastType},
			} {
				t.Run(restore.name, func(t *testing.T) {
					var out strings.Builder
					err := restore.fn(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out))
					if err == nil || out.Len() != 0 {
						t.Fatalf("Restore wrote %q, error = %v; want no output and an error", out.String(), err)
					}
				})
			}
		})
	}
}

func TestTypeNameRestoreRejectsInvalidQualifiers(t *testing.T) {
	for _, tc := range []struct {
		name string
		node *ast.TypeName
	}{
		{"charset", &ast.TypeName{BaseTypeName: pkast.BaseTypeName{Keywords: []string{"CHAR"}},
			Charset: lex.Ident{Name: "utf8; DROP TABLE users; --"}}},
		{"charset with leading digit", &ast.TypeName{BaseTypeName: pkast.BaseTypeName{Keywords: []string{"CHAR"}},
			Charset: lex.Ident{Name: "8utf8"}}},
		{"collation", &ast.TypeName{BaseTypeName: pkast.BaseTypeName{Keywords: []string{"VARCHAR"}},
			Collation: lex.Ident{Name: "utf8_bin; DROP TABLE users; --"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, restore := range []struct {
				name string
				fn   func(*dformat.RestoreCtx) error
			}{
				{"type", tc.node.Restore},
				{"cast type", tc.node.RestoreAsCastType},
			} {
				t.Run(restore.name, func(t *testing.T) {
					var out strings.Builder
					err := restore.fn(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out))
					if err == nil || out.Len() != 0 {
						t.Fatalf("Restore wrote %q, error = %v; want no output and an error", out.String(), err)
					}
				})
			}
		})
	}
}

func TestTypeSyntaxHasNoExecutionMetadata(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeFor[ast.TypeName](), reflect.TypeFor[ast.LiteralExpr]()} {
		t.Run(typ.Name(), func(t *testing.T) {
			for _, name := range []string{"Tp", "Flag", "Flen", "Decimal", "Datum", "Refer", "FieldType"} {
				if _, ok := typ.FieldByName(name); ok {
					t.Errorf("syntax node contains %s", name)
				}
			}
			for _, name := range []string{"GetType", "SetType", "GetFlag", "SetFlag", "GetValue", "SetValue"} {
				if _, ok := reflect.PointerTo(typ).MethodByName(name); ok {
					t.Errorf("syntax node exposes %s", name)
				}
			}
		})
	}
}

type typeNodeVisitor struct {
	entered []ast.Node
	enter   func(ast.Node) (ast.Node, bool)
	leave   func(ast.Node) (ast.Node, bool)
}

func (v *typeNodeVisitor) Enter(n ast.Node) (ast.Node, bool) {
	v.entered = append(v.entered, n)
	if v.enter != nil {
		return v.enter(n)
	}
	return n, false
}

func (v *typeNodeVisitor) Leave(n ast.Node) (ast.Node, bool) {
	if v.leave != nil {
		return v.leave(n)
	}
	return n, true
}

func TestTypeNameVisitorTraversal(t *testing.T) {
	t.Parallel()

	mkType := func() *ast.TypeName {
		node := ast.NewTypeName("ENUM")
		node.Modifiers = append(node.Modifiers,
			typeTestLiteral(pkast.LiteralString, "a"),
			typeTestLiteral(pkast.LiteralString, "b"),
		)
		return node
	}

	t.Run("identity", func(t *testing.T) {
		node := mkType()
		v := &typeNodeVisitor{}
		if out, ok := node.Accept(v); !ok || out != node {
			t.Fatalf("Accept = (%T, %v), want (*ast.TypeName, true)", out, ok)
		}
		if len(v.entered) != 3 {
			t.Fatalf("visited %d nodes, want 3 (type, a, b)", len(v.entered))
		}
		if v.entered[0] != node {
			t.Fatal("outer type identity was lost")
		}
		if node.Modifiers[0] != v.entered[1] || node.Modifiers[1] != v.entered[2] {
			t.Fatal("modifier pointers were not visited in place")
		}
	})

	t.Run("root replacement", func(t *testing.T) {
		node := mkType()
		replacement := mkType()
		v := &typeNodeVisitor{enter: func(n ast.Node) (ast.Node, bool) {
			if n == node {
				return replacement, false
			}
			return n, false
		}}
		out, ok := node.Accept(v)
		if !ok || out != replacement {
			t.Fatalf("Accept = (%T, %v), want (*ast.TypeName, true)", out, ok)
		}
		if len(v.entered) != 3 || v.entered[1] != replacement.Modifiers[0] || v.entered[2] != replacement.Modifiers[1] {
			t.Fatalf("replacement children were not traversed: %v", v.entered)
		}
	})

	t.Run("modifier replacement", func(t *testing.T) {
		node := mkType()
		second := typeTestLiteral(pkast.LiteralInteger, "2")
		third := typeTestLiteral(pkast.LiteralInteger, "3")
		v := &typeNodeVisitor{enter: func(n ast.Node) (ast.Node, bool) {
			if _, ok := n.(*ast.LiteralExpr); ok {
				return second, false
			}
			return n, false
		}}
		out, ok := node.Accept(v)
		if !ok || out != node {
			t.Fatalf("Accept = (%T, %v), want (*ast.TypeName, true)", out, ok)
		}
		if node.Modifiers[0] != second || node.Modifiers[1] != second {
			t.Fatalf("replacements not written back: %T %T", node.Modifiers[0], node.Modifiers[1])
		}
		if len(v.entered) != 3 {
			t.Fatalf("visited %d nodes, want 3", len(v.entered))
		}
		v2 := &typeNodeVisitor{leave: func(n ast.Node) (ast.Node, bool) {
			if _, ok := n.(*ast.LiteralExpr); ok {
				return third, true
			}
			return n, true
		}}
		if _, ok := node.Accept(v2); !ok {
			t.Fatal("second traversal stopped unexpectedly")
		}
		if node.Modifiers[0] != third {
			t.Fatal("replacement is not re-applied on re-traversal")
		}
		if len(v2.entered) != 3 {
			t.Fatalf("second traversal visited %d nodes, want 3", len(v2.entered))
		}
	})

	t.Run("skip children", func(t *testing.T) {
		node := mkType()
		v := &typeNodeVisitor{enter: func(n ast.Node) (ast.Node, bool) { return n, true }}
		if out, ok := node.Accept(v); !ok || out != node {
			t.Fatalf("Accept = (%T, %v), want (*ast.TypeName, true)", out, ok)
		}
		if len(v.entered) != 1 {
			t.Fatalf("visited %d nodes, want 1", len(v.entered))
		}
	})

	t.Run("stop on leave", func(t *testing.T) {
		node := mkType()
		v := &typeNodeVisitor{leave: func(n ast.Node) (ast.Node, bool) { return n, false }}
		if _, ok := node.Accept(v); ok {
			t.Fatal("visitor stop was ignored")
		}
		if len(v.entered) != 2 {
			t.Fatalf("visited %d nodes, want 2 before abort", len(v.entered))
		}
	})

	t.Run("invalid replacement aborts without panic", func(t *testing.T) {
		node := mkType()
		foreign := &sharedPureExpr{}
		v := &typeNodeVisitor{leave: func(n ast.Node) (ast.Node, bool) {
			if _, ok := n.(*ast.LiteralExpr); ok {
				return foreign, true
			}
			return n, true
		}}
		if _, ok := node.Accept(v); !ok {
			t.Fatal("legal ExprNode replacement was rejected")
		}
		if len(v.entered) != 3 {
			t.Fatalf("visited %d nodes, want 3", len(v.entered))
		}
		if node.Modifiers[1] != foreign {
			t.Fatal("ExprNode replacement was not written back")
		}

		node2 := mkType()
		invalid := &typeNodeVisitor{leave: func(n ast.Node) (ast.Node, bool) {
			if _, ok := n.(*ast.LiteralExpr); ok {
				return ast.NewTypeName("INT"), true
			}
			return n, true
		}}
		if _, ok := node2.Accept(invalid); ok {
			t.Fatal("non-ExprNode replacement was accepted")
		}
		if len(invalid.entered) != 2 {
			t.Fatalf("visited %d nodes, want 2 before rejection", len(invalid.entered))
		}
	})
	t.Run("invalid Enter type", func(t *testing.T) {
		node := mkType()
		v := &typeNodeVisitor{enter: func(n ast.Node) (ast.Node, bool) { return &sharedPureExpr{}, false }}
		if _, ok := node.Accept(v); ok || len(v.entered) != 1 {
			t.Fatal("invalid Enter replacement was not rejected immediately")
		}
	})
}

func TestTypeNameModifierRestoreErrorPropagation(t *testing.T) {
	t.Parallel()

	failing := &sharedPureExpr{restoreErr: true}
	node := ast.NewTypeName("ENUM")
	node.Modifiers = append(node.Modifiers, failing)

	var out strings.Builder
	if err := node.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); err == nil ||
		!strings.Contains(err.Error(), "shared pure expression restore failure") {
		t.Fatalf("modifier error lost: %v", err)
	}

	node = ast.NewTypeName("VARCHAR")
	node.Modifiers = []ast.ExprNode{failing}
	out.Reset()
	if err := node.RestoreAsCastType(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); err == nil ||
		!strings.Contains(err.Error(), "shared pure expression restore failure") {
		t.Fatalf("CAST modifier error lost: %v", err)
	}
	cast := &ast.FuncCastExpr{Expr: &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "1"}}, Type: node, FunctionType: ast.CastFunction}
	out.Reset()
	if err := cast.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); err == nil ||
		!strings.Contains(err.Error(), "FuncCastExpr.Type") || !strings.Contains(err.Error(), "shared pure expression restore failure") {
		t.Fatalf("outer CAST type error lost: %v", err)
	}
}

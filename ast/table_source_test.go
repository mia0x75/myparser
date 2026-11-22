package ast_test

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

func mysqlSourceTable(name string) *ast.TableName {
	return &ast.TableName{BaseTableRef: pkast.BaseTableRef{Name: ast.QualifiedName{Parts: []lex.Ident{{Name: name}}}}}
}

func mysqlHintSource() *ast.TableSource {
	return &ast.TableSource{
		BaseTableSource: pkast.BaseTableSource{Source: mysqlSourceTable("t"), Alias: lex.Ident{Name: "x"}},
		IndexHints:      []*ast.IndexHint{{HintType: ast.HintUse, HintScope: ast.HintForScan, IndexNames: []lex.Ident{{Name: "i"}}}},
	}
}

type mysqlSourceVisitor struct {
	enter func(ast.Node) (ast.Node, bool)
	leave func(ast.Node) (ast.Node, bool)
}

func (v mysqlSourceVisitor) Enter(n ast.Node) (ast.Node, bool) {
	if v.enter != nil {
		return v.enter(n)
	}
	return n, false
}

func (v mysqlSourceVisitor) Leave(n ast.Node) (ast.Node, bool) {
	if v.leave != nil {
		return v.leave(n)
	}
	return n, true
}

func TestTableSourceDialectCoreAndVisitor(t *testing.T) {
	t.Parallel()
	t.Run("actual outer and one source", func(t *testing.T) {
		n := mysqlHintSource()
		var role pkast.TableSourceNode = n
		if role.TableSourceCore() != &n.BaseTableSource {
			t.Fatal("role did not return the original Base")
		}
		var entered, left []ast.Node
		got, ok := n.Accept(mysqlSourceVisitor{
			enter: func(node ast.Node) (ast.Node, bool) { entered = append(entered, node); return node, false },
			leave: func(node ast.Node) (ast.Node, bool) { left = append(left, node); return node, true },
		})
		if !ok || got != n || !reflect.DeepEqual(entered, []ast.Node{n, n.Source}) || !reflect.DeepEqual(left, []ast.Node{n.Source, n}) {
			t.Fatalf("wrong outer, source order or extra hint events: %T %v %v %v", got, ok, entered, left)
		}
		if reflect.TypeFor[*ast.IndexHint]().Implements(reflect.TypeFor[ast.Node]()) {
			t.Fatal("index hints must remain non-Node syntax values")
		}
	})
	t.Run("nil receiver", func(t *testing.T) {
		var n *ast.TableSource
		var role pkast.TableSourceNode = n
		called := false
		got, ok := n.Accept(mysqlSourceVisitor{enter: func(node ast.Node) (ast.Node, bool) { called = true; return node, false }})
		if ok || got != nil || called || role.TableSourceCore() != nil {
			t.Fatal("typed nil source was dispatched")
		}
	})
	t.Run("skip", func(t *testing.T) {
		n := mysqlHintSource()
		leaves := 0
		got, ok := n.Accept(mysqlSourceVisitor{
			enter: func(node ast.Node) (ast.Node, bool) {
				if node != n {
					t.Fatal("skip visited child")
				}
				return node, true
			},
			leave: func(node ast.Node) (ast.Node, bool) { leaves++; return node, true },
		})
		if !ok || got != n || leaves != 1 {
			t.Fatal("skip did not leave the actual outer once")
		}
	})
	t.Run("nil source leaf", func(t *testing.T) {
		n := &ast.TableSource{}
		entries, leaves := 0, 0
		got, ok := n.Accept(mysqlSourceVisitor{
			enter: func(node ast.Node) (ast.Node, bool) { entries++; return node, false },
			leave: func(node ast.Node) (ast.Node, bool) { leaves++; return node, true },
		})
		if !ok || got != n || entries != 1 || leaves != 1 {
			t.Fatal("nil source did not remain a leaf")
		}
	})
	t.Run("outer replacement", func(t *testing.T) {
		n, replacement := mysqlHintSource(), mysqlHintSource()
		replacement.Source = mysqlSourceTable("replacement")
		var entered []ast.Node
		got, ok := n.Accept(mysqlSourceVisitor{enter: func(node ast.Node) (ast.Node, bool) {
			entered = append(entered, node)
			if node == n {
				return replacement, false
			}
			return node, false
		}})
		if !ok || got != replacement || !reflect.DeepEqual(entered, []ast.Node{n, replacement.Source}) {
			t.Fatal("outer replacement did not traverse its own source")
		}
	})
	t.Run("source replacement", func(t *testing.T) {
		n := mysqlHintSource()
		original, replacement := n.Source, mysqlSourceTable("replacement")
		got, ok := n.Accept(mysqlSourceVisitor{leave: func(node ast.Node) (ast.Node, bool) {
			if node == original {
				return replacement, true
			}
			return node, true
		}})
		if !ok || got != n || n.Source != replacement {
			t.Fatal("source replacement was lost")
		}
	})
	for _, tc := range []struct {
		name        string
		replacement ast.Node
	}{
		{"nil", nil}, {"standard outer", &pkast.TableSource{}}, {"typed nil", (*ast.TableSource)(nil)},
	} {
		for _, skip := range []bool{false, true} {
			name := tc.name + "/walk"
			if skip {
				name = tc.name + "/skip"
			}
			t.Run(name, func(t *testing.T) {
				n := mysqlHintSource()
				left := false
				_, ok := n.Accept(mysqlSourceVisitor{
					enter: func(ast.Node) (ast.Node, bool) { return tc.replacement, skip },
					leave: func(node ast.Node) (ast.Node, bool) { left = true; return node, true },
				})
				if ok || left {
					t.Fatal("invalid outer replacement was traversed or left")
				}
			})
		}
	}
	for _, tc := range []struct {
		name        string
		replacement ast.Node
		proceed     bool
	}{
		{"wrong source role", &ast.LiteralExpr{}, true}, {"nil source replacement", nil, true}, {"source stop", mysqlSourceTable("s"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := mysqlHintSource()
			original := n.Source
			leftOuter := false
			_, ok := n.Accept(mysqlSourceVisitor{leave: func(node ast.Node) (ast.Node, bool) {
				if node == original {
					return tc.replacement, tc.proceed
				}
				leftOuter = true
				return node, true
			}})
			if ok || leftOuter || n.Source != original {
				t.Fatal("invalid child replacement changed or left outer")
			}
		})
	}
}

func TestTableSourceDialectFieldContracts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                 string
		typ                  reflect.Type
		baseName, suffixName string
		baseType, suffixType reflect.Type
	}{
		{"source", reflect.TypeFor[ast.TableSource](), "BaseTableSource", "IndexHints", reflect.TypeFor[pkast.BaseTableSource](), reflect.TypeFor[[]*ast.IndexHint]()},
		{"table", reflect.TypeFor[ast.TableName](), "BaseTableRef", "PartitionNames", reflect.TypeFor[pkast.BaseTableRef](), reflect.TypeFor[[]lex.Ident]()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.typ.NumField() != 2 {
				t.Fatalf("%v has %d fields, want 2", tc.typ, tc.typ.NumField())
			}
			base, suffix := tc.typ.Field(0), tc.typ.Field(1)
			if base.Name != tc.baseName || base.Type != tc.baseType || !base.Anonymous || suffix.Name != tc.suffixName || suffix.Type != tc.suffixType || suffix.Anonymous {
				t.Fatalf("wrong syntax-only field layout: %+v, %+v", base, suffix)
			}
		})
	}
}

func TestTableSourceDialectRestoreErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		node *ast.TableSource
		want string
	}{
		{"nil", nil, "missing source"},
		{"missing source", &ast.TableSource{}, "missing source"},
		{"columns without alias", &ast.TableSource{BaseTableSource: pkast.BaseTableSource{Source: mysqlSourceTable("t"), Columns: []lex.Ident{{Name: "a"}}}}, "column aliases require an alias"},
		{"nil hint", &ast.TableSource{BaseTableSource: pkast.BaseTableSource{Source: mysqlSourceTable("t")}, IndexHints: []*ast.IndexHint{nil}}, "nil hint"},
		{"invalid hint type", &ast.TableSource{BaseTableSource: pkast.BaseTableSource{Source: mysqlSourceTable("t")}, IndexHints: []*ast.IndexHint{{HintType: 99, HintScope: ast.HintForScan}}}, "IndexHintType"},
		{"invalid hint scope", &ast.TableSource{BaseTableSource: pkast.BaseTableSource{Source: mysqlSourceTable("t")}, IndexHints: []*ast.IndexHint{{HintType: ast.HintUse, HintScope: 99}}}, "IndexHintScope"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			err := tc.node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("restore error = %v, want %q", err, tc.want)
			}
		})
	}
	for _, name := range []string{"non-table source", "typed nil table", "lateral", "column aliases"} {
		t.Run(name, func(t *testing.T) {
			n := mysqlHintSource()
			switch name {
			case "non-table source":
				n.Source = &pkast.TableSource{BaseTableSource: n.BaseTableSource}
			case "typed nil table":
				n.Source = (*ast.TableName)(nil)
			case "lateral":
				n.Lateral = true
			case "column aliases":
				n.Columns = []lex.Ident{{Name: "a"}}
			}
			var out strings.Builder
			if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err == nil || !strings.Contains(err.Error(), "index hints require") {
				t.Fatalf("invalid hinted source restored: %v", err)
			}
		})
	}
}

type mysqlSourceFailWriter struct {
	remaining int
	err       error
}

func (w *mysqlSourceFailWriter) Write(p []byte) (int, error) {
	if len(p) > w.remaining {
		n := w.remaining
		w.remaining = 0
		return n, w.err
	}
	w.remaining -= len(p)
	return len(p), nil
}

func TestTableSourceDialectRestoreWriterAndQuotes(t *testing.T) {
	t.Parallel()
	n := mysqlHintSource()
	failure := errors.New("source writer failed")
	for _, budget := range []int{0, len("`t`"), len("`t` AS `x`"), len("`t` AS `x` USE INDEX (") + 1} {
		ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &mysqlSourceFailWriter{remaining: budget, err: failure})
		if err := n.Restore(ctx); !errors.Is(err, failure) {
			t.Fatalf("writer budget %d lost first error: %v", budget, err)
		}
	}
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &mysqlSourceFailWriter{err: failure})
	ctx.WritePlain("preexisting failure")
	if err := n.Restore(ctx); !errors.Is(err, failure) {
		t.Fatalf("preexisting writer error lost: %v", err)
	}
	short := format.NewRestoreCtx(format.DefaultRestoreFlags, &mysqlSourceFailWriter{})
	if err := n.Restore(short); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short writer error lost: %v", err)
	}
	var out strings.Builder
	if err := n.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase|format.RestoreNameDoubleQuotes, &out)); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), `"t" AS "x" USE INDEX ("i")`; got != want {
		t.Fatalf("quote policy = %q, want %q", got, want)
	}
}

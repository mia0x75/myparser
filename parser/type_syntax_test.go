package parser

import (
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/mia0x75/myparser/ast"
	myformat "github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/mysql"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

func testTypeModifier(value string) ast.ExprNode {
	return &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: value}}
}

// Only documented Restore aliases/defaults are canonicalized; arbitrary modifiers survive.
func normalizeTestTypeMetadata(node ast.Node) {
	switch node := node.(type) {
	case *ast.FuncCastExpr:
		if node.Type != nil && node.Type.Keyword() == "DECIMAL" && len(node.Type.Modifiers) == 0 {
			node.Type.Modifiers = []ast.ExprNode{testTypeModifier("11")}
		}
	case *ast.TypeName:
		normalizeTestName(&node.Name)
		switch node.Keyword() {
		case "BOOL", "BOOLEAN":
			node.Keywords = []string{"TINYINT"}
			node.Modifiers = []ast.ExprNode{testTypeModifier("1")}
		case "UUID":
			node.Keywords = []string{"CHAR"}
			node.Modifiers = []ast.ExprNode{testTypeModifier("36")}
			node.Charset.Name = mysql.DefaultCharset
		case "INET4":
			node.Keywords = []string{"BIGINT"}
			node.Unsigned = true
		case "INET6":
			node.Keywords = []string{"VARBINARY"}
			node.Modifiers = []ast.ExprNode{testTypeModifier("16")}
		}
		for _, ident := range []*lex.Ident{&node.Charset, &node.Collation} {
			ident.Name = strings.ToUpper(ident.Name)
			ident.Raw = ""
			ident.Quote = lex.QuoteUnquoted
			ident.Span = loc.InvalidSpan()
		}
	case *ast.LiteralExpr:
		node.Raw = ""
		node.Quote = lex.QuoteUnquoted
		node.Introducer.Raw = ""
		node.Introducer.Quote = lex.QuoteUnquoted
		node.Introducer.Span = loc.InvalidSpan()
		switch node.Kind {
		case pkast.LiteralString:
			if node.Introducer.Name == mysql.DefaultCharset {
				node.Introducer.Name = ""
			}
		case pkast.LiteralHex:
			node.Value = strings.ToLower(node.Value)
			if len(node.Value)%2 != 0 {
				node.Value = "0" + node.Value
			}
		case pkast.LiteralBit:
			value := strings.TrimLeft(node.Value, "0")
			if value == "" && node.Value != "" {
				value = "0"
			}
			node.Value = value
		}
	}
}

func TestLiteralNormalizationPreservesMeaning(t *testing.T) {
	for _, tc := range []struct {
		name        string
		kind        pkast.LiteralKind
		left, right string
	}{
		{"string contents", pkast.LiteralString, "a", "b"},
		{"hex byte width", pkast.LiteralHex, "00", "0000"},
		{"empty bit and zero", pkast.LiteralBit, "", "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			left := &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: tc.kind, Value: tc.left}}
			right := &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: tc.kind, Value: tc.right}}
			CleanNodeText(left)
			CleanNodeText(right)
			if reflect.DeepEqual(left, right) {
				t.Fatal("normalization erased literal meaning")
			}
		})
	}
	left := &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralString, Value: "a"}, Introducer: lex.Ident{Name: "latin1"}}
	right := &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralString, Value: "a"}, Introducer: lex.Ident{Name: "utf8"}}
	CleanNodeText(left)
	CleanNodeText(right)
	if reflect.DeepEqual(left, right) {
		t.Fatal("normalization erased the explicit charset")
	}
}

func TestTypeSyntaxNormalizationPreservesMeaning(t *testing.T) {
	for _, tc := range []struct {
		name        string
		left, right *ast.TypeName
	}{
		{"keyword", ast.NewTypeName("INT"), ast.NewTypeName("BIGINT")},
		{"width", ast.NewTypeName("VARCHAR"), ast.NewTypeName("VARCHAR")},
		{"scale", ast.NewTypeName("DECIMAL"), ast.NewTypeName("DECIMAL")},
		{"unsigned", ast.NewTypeName("INT"), ast.NewTypeName("INT")},
		{"enum value", ast.NewTypeName("ENUM"), ast.NewTypeName("ENUM")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			switch tc.name {
			case "width":
				tc.left.Modifiers = []ast.ExprNode{testTypeModifier("12")}
				tc.right.Modifiers = []ast.ExprNode{testTypeModifier("13")}
			case "enum value":
				tc.left.Modifiers = []ast.ExprNode{&ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralString, Value: "a"}}}
				tc.right.Modifiers = []ast.ExprNode{&ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralString, Value: "b"}}}
			case "scale":
				tc.left.Modifiers = []ast.ExprNode{testTypeModifier("12"), testTypeModifier("0")}
				tc.right.Modifiers = []ast.ExprNode{testTypeModifier("12"), testTypeModifier("1")}
			case "unsigned":
				tc.right.Unsigned = true
			}
			CleanNodeText(tc.left)
			CleanNodeText(tc.right)
			if reflect.DeepEqual(tc.left, tc.right) {
				t.Fatal("normalization erased type meaning")
			}
		})
	}
}

func TestTypeSyntaxMetadataAndReuse(t *testing.T) {
	parser := New()
	sql := "CREATE TABLE t (a VARCHAR(12) CHARACTER SET `UtF8`, b ENUM('plain',\"a'b\",'c''d'))"
	parsed, err := parser.ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}
	stmt := parsed.(*ast.CreateTableStmt)
	first := stmt.Columns[0].ColumnDefCore().Type.(*ast.TypeName)
	second := stmt.Columns[1].ColumnDefCore().Type.(*ast.TypeName)
	if first.Keyword() != "VARCHAR" || second.Keyword() != "ENUM" {
		t.Fatalf("type keywords = %q, %q", first.Keyword(), second.Keyword())
	}
	for _, node := range []*ast.TypeName{first, second} {
		span := node.Span()
		if !span.Within(len(sql)) || node.Text() != sql[span.Start:span.End] ||
			unsafe.StringData(node.Text()) != unsafe.StringData(sql[span.Start:]) {
			t.Fatalf("type range/text does not share its input: %+v %q", span, node.Text())
		}
	}
	start := strings.Index(sql, "`UtF8`")
	wantCharset := lex.Ident{Name: "UtF8", Raw: "`UtF8`", Quote: lex.QuoteBacktick,
		Span: loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + 6)}}
	if first.Charset != wantCharset || unsafe.StringData(first.Charset.Raw) != unsafe.StringData(sql[start:]) {
		t.Fatalf("charset spelling lost: %+v, want %+v", first.Charset, wantCharset)
	}
	for _, tc := range []struct {
		typ        *ast.TypeName
		index      int
		raw, value string
		kind       pkast.LiteralKind
		quote      lex.QuoteStyle
	}{
		{first, 0, "12", "12", pkast.LiteralInteger, lex.QuoteUnquoted},
		{second, 0, "'plain'", "plain", pkast.LiteralString, lex.QuoteSingle},
		{second, 1, `"a'b"`, "a'b", pkast.LiteralString, lex.QuoteDouble},
		{second, 2, "'c''d'", "c'd", pkast.LiteralString, lex.QuoteSingle},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			literal := tc.typ.Modifiers[tc.index].(*ast.LiteralExpr)
			start := strings.Index(sql, tc.raw)
			span := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(tc.raw))}
			if literal.Raw != tc.raw || literal.Value != tc.value || literal.Kind != tc.kind || literal.Quote != tc.quote || literal.Span() != span {
				t.Fatalf("literal syntax lost: %+v span=%+v", literal, literal.Span())
			}
			if unsafe.StringData(literal.Raw) != unsafe.StringData(sql[start:]) {
				t.Fatal("literal Raw copied the input")
			}
			if tc.raw == tc.value && unsafe.StringData(literal.Value) != unsafe.StringData(literal.Raw) {
				t.Fatal("undecoded literal Value copied the input")
			}
			if tc.raw == "'"+tc.value+"'" && unsafe.StringData(literal.Value) != unsafe.StringData(sql[start+1:]) {
				t.Fatal("undecoded string Value copied the input")
			}
		})
	}
	before := restoreStmt(t, stmt)
	modifiers := append([]ast.ExprNode(nil), second.Modifiers...)
	if _, err := parser.ParseOneStmt("CREATE TABLE other (x ENUM('different'))", "", ""); err != nil {
		t.Fatal(err)
	}
	if after := restoreStmt(t, stmt); after != before || first.Charset != wantCharset || !reflect.DeepEqual(second.Modifiers, modifiers) {
		t.Fatal("Parser reuse changed the retained type AST")
	}
}

type typeCarrierVisitor struct {
	mode        string
	literals    int
	types       int
	replacement *ast.TypeName
}

func (v *typeCarrierVisitor) Enter(n ast.Node) (ast.Node, bool) {
	if _, ok := n.(*ast.TypeName); ok {
		v.types++
	}
	if _, ok := n.(*ast.LiteralExpr); ok {
		v.literals++
	}
	return n, v.mode == "skip"
}

func (v *typeCarrierVisitor) Leave(n ast.Node) (ast.Node, bool) {
	if _, ok := n.(*ast.TypeName); ok {
		switch v.mode {
		case "abort":
			return n, false
		case "replace":
			return v.replacement, true
		case "wrong type":
			return &ast.LiteralExpr{}, true
		}
	}
	if literal, ok := n.(*ast.LiteralExpr); ok && v.mode == "literal" {
		copy := *literal
		copy.Value = "9"
		return &copy, true
	}
	return n, true
}

func TestTypeSyntaxCarrierVisitors(t *testing.T) {
	for _, tc := range []struct {
		name string
		make func(*ast.TypeName) (ast.Node, func() *ast.TypeName)
	}{
		{"column", func(typ *ast.TypeName) (ast.Node, func() *ast.TypeName) {
			n := &ast.ColumnDef{BaseColumnDef: pkast.BaseColumnDef{Type: typ}}
			return n, func() *ast.TypeName { return n.Type.(*ast.TypeName) }
		}},
		{"cast", func(typ *ast.TypeName) (ast.Node, func() *ast.TypeName) {
			n := &ast.FuncCastExpr{Expr: &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "1"}}, Type: typ}
			return n, func() *ast.TypeName { return n.Type }
		}},
		{"json table", func(typ *ast.TypeName) (ast.Node, func() *ast.TypeName) {
			n := &ast.JSONTableColumn{Kind: ast.JSONTableColumnPath, Name: lex.Ident{Name: "c"}, Type: typ}
			return n, func() *ast.TypeName { return n.Type.(*ast.TypeName) }
		}},
		{"declare", func(typ *ast.TypeName) (ast.Node, func() *ast.TypeName) {
			n := &ast.DeclareStmt{Type: typ}
			return n, func() *ast.TypeName { return n.Type }
		}},
		{"sequence", func(typ *ast.TypeName) (ast.Node, func() *ast.TypeName) {
			n := &ast.SequenceOption{Type: typ}
			return n, func() *ast.TypeName { return n.Type }
		}},
		{"function return", func(typ *ast.TypeName) (ast.Node, func() *ast.TypeName) {
			n := &ast.CreateFunctionStmt{Name: &ast.TableName{}, ReturnType: typ}
			return n, func() *ast.TypeName { return n.ReturnType }
		}},
		{"function parameter", func(typ *ast.TypeName) (ast.Node, func() *ast.TypeName) {
			n := &ast.CreateFunctionStmt{Name: &ast.TableName{}, Params: []*ast.FuncParam{{Type: typ}}}
			return n, func() *ast.TypeName { return n.Params[0].Type }
		}},
		{"procedure parameter", func(typ *ast.TypeName) (ast.Node, func() *ast.TypeName) {
			n := &ast.CreateProcedureStmt{Name: &ast.TableName{}, Params: []*ast.FuncParam{{Type: typ}}, Body: &ast.BeginEndStmt{}}
			return n, func() *ast.TypeName { return n.Params[0].Type }
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, mode := range []string{"traverse", "replace", "literal", "skip", "abort", "wrong type"} {
				t.Run(mode, func(t *testing.T) {
					typ := ast.NewTypeName("VARCHAR")
					typ.Modifiers = []ast.ExprNode{&ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "12"}}}
					node, current := tc.make(typ)
					v := &typeCarrierVisitor{mode: mode, replacement: ast.NewTypeName("TEXT")}
					got, ok := node.Accept(v)
					if ok != (mode != "abort" && mode != "wrong type") || got != node {
						t.Fatalf("Accept = %T, %v in %s", got, ok, mode)
					}
					if mode == "skip" {
						if v.types != 0 || v.literals != 0 {
							t.Fatal("skip visited children")
						}
						return
					}
					// The cast carrier also visits its converted expression literal; every other
					// carrier holds exactly one type and one modifier literal.
					wantLiterals := 2
					if tc.name != "cast" {
						wantLiterals = 1
					}
					if v.types != 1 || v.literals != wantLiterals {
						t.Fatalf("visits = %d types, %d literals, want 1 types, %d literals", v.types, v.literals, wantLiterals)
					}
					if mode == "replace" && current() != v.replacement {
						t.Fatal("type replacement not written back")
					}
					if mode == "literal" && current().Modifiers[0].(*ast.LiteralExpr).Value != "9" {
						t.Fatal("literal replacement not written back")
					}
					if mode == "traverse" {
						var out strings.Builder
						if err := typ.Restore(myformat.NewRestoreCtx(myformat.DefaultRestoreFlags, &out)); err != nil || out.String() != "VARCHAR(12)" {
							t.Fatalf("visited type Restore = %q, %v", out.String(), err)
						}
					}
				})
			}
		})
	}
}

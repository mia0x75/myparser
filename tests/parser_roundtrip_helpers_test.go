package parser_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/myparser/parser"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

func mustParseOne(t *testing.T, sql string) ast.StmtNode {
	t.Helper()
	stmt, err := parser.New().ParseOneStmt(sql)
	if err != nil {
		t.Fatalf("parse %q: %v", sql, err)
	}
	return stmt
}

func restoreStmt(t *testing.T, stmt ast.StmtNode) string {
	t.Helper()
	var sb strings.Builder
	if err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
		t.Fatalf("restore: %v", err)
	}
	return sb.String()
}

func runRestoreTest(t *testing.T, sourceSQLs, expectSQLs string, enableWindowFunc bool) {
	t.Helper()
	parser := parser.New()
	parser.EnableWindowFunc(enableWindowFunc)
	stmts, _, err := parser.ParseWithWarnings(sourceSQLs)
	if err != nil {
		t.Fatalf("source %s: %v", sourceSQLs, err)
	}
	restored := make([]string, 0, len(stmts))
	for _, stmt := range stmts {
		sql := restoreStmt(t, stmt)
		if sql == "" {
			continue
		}
		again, err := parser.ParseOneStmt(sql)
		if err != nil {
			t.Fatalf("source %s; restore %s: %v", sourceSQLs, sql, err)
		}
		cleanNodeText(stmt)
		cleanNodeText(again)
		normalizeSyntaxMetadata(stmt)
		normalizeSyntaxMetadata(again)
		if !reflect.DeepEqual(again, stmt) {
			t.Fatalf("source %s; restore %s: got %#v, want %#v", sourceSQLs, sql, again, stmt)
		}
		restored = append(restored, sql)
	}
	if sql := strings.Join(restored, "; "); sql != expectSQLs {
		t.Fatalf("restore %s; expect %s", sql, expectSQLs)
	}
}

func assertParseErrors(t *testing.T, sqls []string) {
	t.Helper()
	for _, src := range sqls {
		t.Run(src, func(t *testing.T) {
			if _, err := parser.New().ParseOneStmt(src); err == nil {
				t.Fatalf("expected parse error for %q", src)
			}
		})
	}
}

type syntaxVisitor struct {
	enter func(ast.Node) (ast.Node, bool)
	leave func(ast.Node) (ast.Node, bool)
}

func (v syntaxVisitor) Enter(n ast.Node) (ast.Node, bool) {
	if v.enter != nil {
		return v.enter(n)
	}
	return n, false
}

func (v syntaxVisitor) Leave(n ast.Node) (ast.Node, bool) {
	if v.leave != nil {
		return v.leave(n)
	}
	return n, true
}

func isValueOrLiteral(n ast.Node) bool {
	_, ok := n.(*ast.LiteralExpr)
	return ok
}

func replaceLiteral(n *ast.LiteralExpr, text string) *ast.LiteralExpr {
	literal := *n
	literal.Value = text
	return &literal
}

func assertVisitorControls(t *testing.T, node ast.Node, isChild func(ast.Node) bool) {
	t.Helper()
	_, ok := node.Accept(syntaxVisitor{leave: func(n ast.Node) (ast.Node, bool) {
		return n, !isChild(n)
	}})
	if ok {
		t.Fatal("visitor abort was not propagated")
	}
	entered := 0
	_, ok = node.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		entered++
		return n, true
	}})
	if !ok || entered != 1 {
		t.Fatalf("skipChildren: ok=%v entered=%d", ok, entered)
	}
}

// Restore changes spelling and source ranges, not decoded name parts or literal meaning.
func normalizeName(name *ast.QualifiedName) {
	if name == nil {
		return
	}
	name.Span = loc.InvalidSpan()
	for i := range name.Parts {
		name.Parts[i].Raw = ""
		name.Parts[i].Quote = lex.QuoteUnquoted
		name.Parts[i].Span = loc.InvalidSpan()
	}
}

func normalizeIdent(ident *lex.Ident) {
	if !ident.Present() {
		*ident = lex.Ident{Span: loc.InvalidSpan()}
		return
	}
	if ident.Name == "" {
		ident.Quote = lex.QuoteBacktick
		ident.Raw = "``"
	} else {
		ident.Raw = ""
		ident.Quote = lex.QuoteUnquoted
	}
	ident.Span = loc.InvalidSpan()
}

func normalizeIdents(idents []lex.Ident) {
	for i := range idents {
		normalizeIdent(&idents[i])
	}
}

func cleanNodeText(node ast.Node) {
	node.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		n.SetText("")
		n.SetSpan(loc.InvalidSpan())
		switch n := n.(type) {
		case *ast.SetOprStmt:
			n.Tail.SetSpan(loc.InvalidSpan())
		case *ast.QueryTailStmt:
			n.Tail.SetSpan(loc.InvalidSpan())
		case *ast.TableName:
			normalizeName(&n.Name)
		case *ast.ColumnRef:
			normalizeName(&n.Name)
		case *ast.DefaultExpr:
			normalizeName(&n.Name)
		case *ast.ColumnDef:
			normalizeName(&n.Name)
		case *ast.IndexKeyPart:
			normalizeName(&n.Column)
		case *ast.TableSource:
			normalizeIdent(&n.Alias)
			normalizeIdents(n.Columns)
		case *ast.CommonTableExpression:
			normalizeIdent(&n.Name)
			normalizeIdents(n.Columns)
		case *ast.WithClause:
			normalizeIdents(n.CycleColumns)
		case *ast.WindowSpec:
			normalizeIdent(&n.Name)
			normalizeIdent(&n.Ref)
		case *ast.SelectItem:
			normalizeIdent(&n.Alias)
			if wildcard, ok := n.Expr.(*ast.WildcardExpr); ok {
				normalizeName(&wildcard.Qualifier)
			}
		case *ast.CreateViewStmt:
			normalizeIdents(n.Columns)
		case *ast.AlterViewStmt:
			normalizeIdents(n.Columns)
		case *ast.ShowStmt:
			normalizeName(n.Column)
			normalizeIdent(&n.Database)
		case *ast.ExplainStmt:
			normalizeIdent(&n.Schema)
		case *ast.VariableTarget:
			normalizeName(&n.Name)
		case *ast.VariableExpr:
			if n.Target != nil {
				normalizeName(&n.Target.Name)
			}
		case *ast.VariableAssignment:
			if n.Target != nil {
				normalizeName(&n.Target.Name)
			}
		case *ast.FuncCallExpr:
			for i := range n.Name.Parts {
				n.Name.Parts[i].Name = strings.ToLower(n.Name.Parts[i].Name)
			}
			normalizeName(&n.Name)
		case *ast.AggregateFuncExpr:
			for i := range n.Name.Parts {
				n.Name.Parts[i].Name = strings.ToLower(n.Name.Parts[i].Name)
			}
			normalizeName(&n.Name)
		case *ast.WindowFuncExpr:
			for i := range n.Name.Parts {
				n.Name.Parts[i].Name = strings.ToLower(n.Name.Parts[i].Name)
			}
			normalizeName(&n.Name)
		case *ast.CollateExpr:
			n.Collation = strings.ToLower(n.Collation)
		case *ast.TypeName:
			normalizeName(&n.Name)
			for _, ident := range []*lex.Ident{&n.Charset, &n.Collation} {
				ident.Name = strings.ToUpper(ident.Name)
				ident.Raw = ""
				ident.Quote = lex.QuoteUnquoted
				ident.Span = loc.InvalidSpan()
			}
		case *ast.LiteralExpr:
			n.Raw = ""
			n.Quote = lex.QuoteUnquoted
			n.Introducer.Raw = ""
			n.Introducer.Quote = lex.QuoteUnquoted
			n.Introducer.Span = loc.InvalidSpan()
			switch n.Kind {
			case pkast.LiteralString:
				if n.Introducer.Name == mysql.DefaultCharset {
					n.Introducer.Name = ""
				}
			case pkast.LiteralHex:
				n.Value = strings.ToLower(n.Value)
				if len(n.Value)%2 != 0 {
					n.Value = "0" + n.Value
				}
			case pkast.LiteralBit:
				value := strings.TrimLeft(n.Value, "0")
				if value == "" && n.Value != "" {
					value = "0"
				}
				n.Value = value
			}
		case *ast.CreateTableStmt:
			for _, opt := range n.Options {
				if opt.Tp == ast.TableOptionCharset || opt.Tp == ast.TableOptionCollate {
					opt.StrValue = strings.ToUpper(opt.StrValue)
				}
			}
		case *ast.Constraint:
			if opt := n.Option; opt != nil && opt.KeyBlockSize == 0 && opt.Tp == 0 && opt.Comment == "" {
				n.Option = nil
			}
		case *ast.AnalyzeTableStmt:
			normalizeIdents(n.Columns)
		case *ast.BeginEndStmt:
			normalizeIdent(&n.Label)
		case *ast.ForStmt:
			normalizeIdent(&n.Label)
			normalizeIdent(&n.Variable)
		case *ast.DeclareStmt:
			normalizeName(n.TypeRef)
			normalizeIdents(n.Names)
		case *ast.JoinExpr:
			for i := range n.Using {
				normalizeName(&n.Using[i])
			}
		case *ast.ValuesStmt:
			n.ValuesKeywordOffset = 0
		case *ast.FrameClause:
			n.TypeOffset, n.ExclusionOffset = 0, 0
		case *ast.FrameBound:
			n.ExprOffset = 0
		}
		return n, false
	}})
}

func normalizeSyntaxMetadata(node ast.Node) {
	node.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		clearIdent := func(ident *lex.Ident) {
			ident.Name = strings.ToUpper(ident.Name)
			ident.Raw = ""
			ident.Quote = lex.QuoteUnquoted
			ident.Span = loc.InvalidSpan()
		}
		switch n := n.(type) {
		case *ast.VariableTarget:
			if n.Kind == ast.VariableTargetSystem {
				n.Prefix = ast.VariablePrefixDoubleAt
				if n.Scope == ast.VariableScopeImplicit || n.Scope == ast.VariableScopeLocal {
					n.Scope = ast.VariableScopeSession
				}
				n.ScopeExplicit = true
			}
		case *ast.FuncCallExpr:
			clearIdent(&n.Unit)
			clearIdent(&n.FormatSelector)
		case *ast.AggregateFuncExpr:
			for i := range n.Name.Parts {
				clearIdent(&n.Name.Parts[i])
			}
		case *ast.WindowFuncExpr:
			for i := range n.Name.Parts {
				clearIdent(&n.Name.Parts[i])
			}
		case *ast.Constraint:
			clearIdent(&n.Name)
			clearIdent(&n.IndexName)
		case *ast.CreateIndexStmt:
			clearIdent(&n.IndexName)
		case *ast.FrameBound:
			clearIdent(&n.Unit)
		case *ast.AnalyzeTableStmt:
			normalizeIdents(n.PartitionNames)
			normalizeIdents(n.IndexNames)
		case *ast.ExecuteStmt:
			clearIdent(&n.Name)
		case *ast.RollbackStmt:
			clearIdent(&n.SavepointName)
		}
		return n, false
	}})
}

func TestRoundTripNormalizationPreservesSemantics(t *testing.T) {
	for _, tc := range []struct {
		name, left, right string
		equal             bool
	}{
		{"spelling", "select `a` from `t`", "SELECT a FROM t", true},
		{"identifier case", "select A", "select a", false},
		{"name parts", "select `a.b`", "select a.b", false},
		{"literal value", "select 'a'", "select 'b'", false},
		{"hex byte width", "select x'00'", "select x'0000'", false},
		{"explicit charset", "select _latin1'a'", "select _koi8r'a'", false},
		{"limit", "select 1 limit 2", "select 1 limit 3", false},
		{"lock", "select 1 for update", "select 1", false},
		{"into", "select 1 into @a", "select 1 into @b", false},
		{"type width", "create table t(a varchar(10))", "create table t(a varchar(11))", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			left, right := mustParseOne(t, tc.left), mustParseOne(t, tc.right)
			cleanNodeText(left)
			cleanNodeText(right)
			normalizeSyntaxMetadata(left)
			normalizeSyntaxMetadata(right)
			if got := reflect.DeepEqual(left, right); got != tc.equal {
				t.Fatalf("normalized equality = %v, want %v", got, tc.equal)
			}
		})
	}
}

package parser

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

func TestDialectQualifiedFunctionsAreNotBuiltins(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		expr ast.ExprNode
		sql  string
	}{
		{"log", &ast.FuncCallExpr{BaseCallExpr: pkast.BaseCallExpr{
			Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "mock_schema"}, {Name: "log"}}},
		}}, "SELECT LOG()"},
		{"group concat", &ast.AggregateFuncExpr{BaseCallExpr: pkast.BaseCallExpr{
			Name: pkast.QualifiedName{Parts: []lex.Ident{{Name: "mock_schema"}, {Name: "group_concat"}}},
		}, Limit: &ast.Limit{}}, "SELECT GROUP_CONCAT(x LIMIT 1)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := NewDialectCompatibilityVisitor(DialectMySQL, tc.sql)
			v.Enter(tc.expr)
			if issues := v.Issues(); len(issues) != 0 {
				t.Fatalf("qualified function treated as builtin: %v", issues)
			}
		})
	}
}

func TestDialectQuotedFunctionCursor(t *testing.T) {
	t.Parallel()
	const sql = "SELECT `log`(), LOG(1)"
	p := New()
	stmt, err := p.ParseOneStmt(sql)
	if err != nil {
		t.Fatal(err)
	}
	// Parsing rejects invalid builtin arity before compatibility diagnostics run.
	stmt.(*ast.SelectStmt).Fields.Items[1].Expr.(*ast.FuncCallExpr).Args = nil
	issues := p.CheckDialectCompatibility(stmt, sql, DialectMySQL)
	if len(issues) != 1 || issues[0].Code != IssueMySQLLogArgumentCount || issues[0].Offset != strings.LastIndex(sql, "LOG") {
		t.Fatalf("quoted call shifted builtin diagnostics: %v", issues)
	}
	if _, err := p.ParseOneStmt("SELECT mock_schema.mock_func(1)"); err == nil {
		t.Fatal("shared name support must not expand the MySQL grammar")
	}
}

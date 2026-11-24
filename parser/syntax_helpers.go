package parser

import (
	"github.com/mia0x75/myparser/ast"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

// Only the final DML boundary sheds an empty root, never an internal join.
func dmlRelation(root ast.ResultSetNode, span loc.Span) ast.ResultSetNode {
	if join, ok := root.(*ast.JoinExpr); ok && join != nil && join.Left != nil &&
		join.Right == nil && join.Type == pkast.JoinTypeUnknown && join.On == nil && len(join.Using) == 0 &&
		!join.Natural && !join.StraightJoin {
		root = join.Left
	}
	if root != nil {
		root.SetSpan(span)
	}
	return root
}

// Only INSERT needs expression targets; other ColumnNameList consumers keep
// their qualified-name values and the grammar's nil/empty distinction.
func syntaxInsertColumns(names []ast.QualifiedName) []ast.ExprNode {
	if names == nil {
		return nil
	}
	columns := make([]ast.ExprNode, len(names))
	for i, name := range names {
		columns[i] = columnRef(name, name.Span)
	}
	return columns
}

// Table options that accept either DEFAULT or an integer keep the spelling
// distinction so Restore does not silently replace an explicit value.
func syntaxTableOptionValue(tp ast.TableOptionType, value interface{}) *ast.TableOption {
	option := &ast.TableOption{Tp: tp}
	if text, ok := value.(string); ok {
		option.StrValue = text
	} else {
		option.UintValue = value.(uint64)
	}
	return option
}

func grantName(name lex.Ident) ast.GrantObjectPart {
	return ast.GrantObjectPart{Name: name, WildcardSpan: loc.InvalidSpan()}
}

func grantStar(span loc.Span) ast.GrantObjectPart {
	return ast.GrantObjectPart{Name: lex.Ident{Span: loc.InvalidSpan()}, Wildcard: true, WildcardSpan: span}
}

func grantObject(span loc.Span, parts ...ast.GrantObjectPart) *ast.GrantObject {
	object := &ast.GrantObject{Parts: parts}
	object.SetSpan(span)
	return object
}

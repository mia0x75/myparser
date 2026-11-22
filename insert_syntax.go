package myparser

import "github.com/mia0x75/myparser/ast"

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

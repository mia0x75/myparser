package parser

import (
	"github.com/mia0x75/myparser/ast"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

func syntaxIdent(l yyLexer, name string, span loc.Span) lex.Ident {
	ident := lex.Ident{Name: name, Span: loc.InvalidSpan()}
	src := yyGetParser(l).src
	if !span.Within(len(src)) || span.Empty() {
		return ident
	}
	ident.Span = span
	ident.Raw = src[span.Start:span.End]
	switch ident.Raw[0] {
	case '`':
		ident.Quote = lex.QuoteBacktick
	case '"':
		ident.Quote = lex.QuoteDouble
	case '\'':
		ident.Quote = lex.QuoteSingle
	}
	if ident.Quote != lex.QuoteUnquoted && len(ident.Raw) >= 2 {
		body := ident.Raw[1 : len(ident.Raw)-1]
		if body == name && ident.Raw[len(ident.Raw)-1] == ident.Raw[0] {
			// The scanner may allocate even when no unescaping was needed.
			ident.Name = body
		}
	}
	return ident
}

func functionName(l yyLexer, name string, span loc.Span) ast.QualifiedName {
	return syntaxName(syntaxIdent(l, name, span))
}

func syntaxName(parts ...lex.Ident) ast.QualifiedName {
	name := ast.QualifiedName{Parts: parts, Span: loc.InvalidSpan()}
	for _, part := range parts {
		name.Span = loc.Cover(name.Span, part.Span)
	}
	return name
}

func syntaxTableName(name ast.QualifiedName) *ast.TableName {
	table := &ast.TableName{BaseTableRef: pkast.BaseTableRef{Name: name}}
	table.SetSpan(name.Span)
	return table
}

func syntaxTableSource(source ast.ResultSetNode, span loc.Span) *ast.TableSource {
	table := &ast.TableSource{BaseTableSource: pkast.BaseTableSource{
		Source: source, Alias: lex.Ident{Span: loc.InvalidSpan()},
	}}
	table.SetSpan(span)
	return table
}

func columnRef(name ast.QualifiedName, span loc.Span) *ast.ColumnRef {
	ref := &ast.ColumnRef{BaseColumnRef: pkast.BaseColumnRef{Name: name}}
	ref.SetSpan(span)
	return ref
}

func namePointer(name ast.QualifiedName) *ast.QualifiedName { return &name }

func syntaxColumnDef(name ast.QualifiedName, typ pkast.TypeNode, options []*ast.ColumnOption, span loc.Span) *ast.ColumnDef {
	column := &ast.ColumnDef{BaseColumnDef: pkast.BaseColumnDef{Name: name, Type: typ}, Options: options}
	column.SetSpan(span)
	return column
}

func syntaxSelectItem(expr ast.ExprNode, alias lex.Ident, span loc.Span) *ast.SelectItem {
	item := &ast.SelectItem{Expr: expr, Alias: alias}
	item.SetSpan(span)
	return item
}

func syntaxWildcard(qualifier ast.QualifiedName, span loc.Span) *ast.WildcardExpr {
	wildcard := &ast.WildcardExpr{Qualifier: qualifier}
	wildcard.SetSpan(span)
	return wildcard
}

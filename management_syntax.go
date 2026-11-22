package myparser

import (
	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

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

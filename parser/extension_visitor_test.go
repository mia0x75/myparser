package parser

import "github.com/mia0x75/myparser/ast"

type extensionVisitor struct {
	enter func(ast.Node) (ast.Node, bool)
	leave func(ast.Node) (ast.Node, bool)
}

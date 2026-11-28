package parser

import (
	"sync"

	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/myparser/ast"
)

// parseState holds the mutable state of one parse run. Parse takes a state
// from a pool so that the scanner buffers are reused between calls.
type parseState struct {
	result     []ast.StmtNode
	src        string
	lexer      Scanner
	joinGroups map[*ast.JoinExpr]bool
	paramIndex int
}

var parseStatePool = sync.Pool{
	New: func() interface{} {
		return &parseState{}
	},
}

func setParseResult(lex yyLexer, statements []ast.StmtNode) {
	yyGetParser(lex).result = statements
}

func appendStatement(lex yyLexer, statements []ast.StmtNode, node ast.Node, _ loc.Span) []ast.StmtNode {
	stmt, ok := node.(ast.StmtNode)
	if !ok {
		return statements
	}
	// Statement actions already carry precise dialect spans; keep them intact.
	stmt.SetText(lex.(*Scanner).stmtText())
	yyGetParser(lex).paramIndex = 0
	return append(statements, stmt)
}

func (st *parseState) newParamMarkerExpr(span loc.Span) *ast.ParamMarkerExpr {
	param := ast.NewParamMarkerExprAt(span, st.paramIndex)
	st.paramIndex++
	return param
}

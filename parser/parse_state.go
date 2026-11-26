package parser

import (
	"sync"

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

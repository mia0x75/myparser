package parser_test

import (
	"fmt"

	_ "github.com/mia0x75/myparser/driver"
	"github.com/mia0x75/myparser/parser"
)

// This example shows how to parse SQL into an AST.
func Example() {

	// 0. Ensure the driver is imported.

	// 1. Create a parser. The parser is NOT goroutine safe and should
	// not be shared among multiple goroutines. However, parser is also
	// heavy, so each goroutine should reuse its own local instance if
	// possible.
	p := parser.New()

	// 2. Parse a text SQL into AST([]ast.StmtNode).
	stmtNodes, _, err := p.ParseWithWarnings("select * from tbl where id = 1")

	// 3. Use AST to do cool things.
	fmt.Println(stmtNodes[0], err)
}

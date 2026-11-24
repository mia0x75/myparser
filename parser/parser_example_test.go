package myparser_test

import (
	"fmt"

	"github.com/mia0x75/myparser"
	_ "github.com/mia0x75/myparser/driver"
)

// This example show how to parse a text sql into ast.
func Example_parseSQL() {

	// 0. Ensure the driver is imported.

	// 1. Create a parser. The parser is NOT goroutine safe and should
	// not be shared among multiple goroutines. However, parser is also
	// heavy, so each goroutine should reuse its own local instance if
	// possible.
	p := myparser.New()

	// 2. Parse a text SQL into AST([]ast.StmtNode).
	stmtNodes, _, err := p.Parse("select * from tbl where id = 1", "", "")

	// 3. Use AST to do cool things.
	fmt.Println(stmtNodes[0], err)
}

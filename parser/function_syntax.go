package parser

import (
	"fmt"
	"strconv"
	"strings"

	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/myparser/ast"
)

type lastValueWindow struct {
	spec       ast.WindowSpec
	ignoreNull bool
}

// validateFunctionCall applies argument rules that both pinned servers
// enforce. Other function calls remain generic.
func validateSourceFunctionCall(l yyLexer, offset int, name string, args []ast.ExprNode) error {
	source := yyGetParser(l).src
	if offset >= 0 && offset < len(source) && (source[offset] == '`' || source[offset] == '"') {
		return nil
	}
	return validateFunctionCall(name, args)
}

func validateFunctionCall(name string, args []ast.ExprNode) error {
	switch strings.ToUpper(name) {
	case "LOG":
		if len(args) == 0 || len(args) > 2 {
			return fmt.Errorf("LOG requires one or two arguments")
		}
	case "NOW":
		if len(args) != 1 {
			return nil
		}
		if literal, ok := args[0].(*ast.LiteralExpr); ok && literal.Kind == pkast.LiteralBoolean {
			return nil
		}
		if literal, ok := args[0].(*ast.LiteralExpr); ok && literal.Kind == pkast.LiteralInteger {
			if _, err := strconv.ParseUint(literal.Value, 10, 64); err == nil {
				return nil
			}
		}
		return fmt.Errorf("NOW precision must be a non-negative integer")
	case "JSON_ARRAYAGG":
		if len(args) != 1 {
			return fmt.Errorf("JSON_ARRAYAGG requires one argument")
		}
	case "JSON_OBJECTAGG":
		if len(args) != 2 {
			return fmt.Errorf("JSON_OBJECTAGG requires two arguments")
		}
	}
	return nil
}

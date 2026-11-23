package ast

import (
	"strings"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"

	"github.com/mia0x75/myparser/errors"
)

func builtinFunctionName(name QualifiedName) string {
	if len(name.Parts) != 1 || name.Parts[0].Quote != lex.QuoteUnquoted {
		return ""
	}
	return strings.ToLower(name.Parts[0].Name)
}

func validateFunctionName(name QualifiedName) error {
	if len(name.Parts) == 0 {
		return errors.New("function name is missing")
	}
	for _, part := range name.Parts {
		if !part.Present() || part.Quote == lex.QuoteUnquoted && part.Name == "" {
			return errors.New("function name is missing")
		}
		if part.Quote != lex.QuoteUnquoted {
			continue
		}
		// Bare parts must not terminate the token in hand-built ASTs.
		for i := 0; i < len(part.Name); i++ {
			c := part.Name[i]
			valid := c >= 0x80 || c == '_' || c == '$' ||
				c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
			if !valid {
				return errors.New("invalid unquoted function name")
			}
		}
	}
	return nil
}

func validTimeUnit(unit string) bool {
	switch unit {
	case "MICROSECOND", "SECOND", "MINUTE", "HOUR", "DAY", "WEEK", "MONTH", "QUARTER", "YEAR",
		"SECOND_MICROSECOND", "MINUTE_MICROSECOND", "MINUTE_SECOND", "HOUR_MICROSECOND", "HOUR_SECOND",
		"HOUR_MINUTE", "DAY_MICROSECOND", "DAY_SECOND", "DAY_MINUTE", "DAY_HOUR", "YEAR_MONTH":
		return true
	}
	return false
}

func validTimestampUnit(unit string) bool {
	switch unit {
	case "MICROSECOND", "SECOND", "MINUTE", "HOUR", "DAY", "WEEK", "MONTH", "QUARTER", "YEAR",
		"SQL_TSI_SECOND", "SQL_TSI_MINUTE", "SQL_TSI_HOUR", "SQL_TSI_DAY", "SQL_TSI_WEEK",
		"SQL_TSI_MONTH", "SQL_TSI_QUARTER", "SQL_TSI_YEAR":
		return true
	}
	return false
}

func isNullSyntaxArgument(expr ExprNode) bool {
	switch expr := expr.(type) {
	case *LiteralExpr:
		return expr != nil && expr.Kind == pkast.LiteralNull
	}
	return false
}

// Check modifiers before writing so malformed hand-built nodes cannot emit SQL.
func (n *FuncCallExpr) validateSyntaxArgs() error {
	if n == nil {
		return errors.New("nil function call")
	}
	if err := validateFunctionName(n.Name); err != nil {
		return err
	}
	count := -1
	name := builtinFunctionName(n.Name)
	switch name {
	case DateAdd, DateSub, AddDate, SubDate:
		count = 2
		if !validTimeUnit(n.Unit.Name) {
			return errors.New("invalid function time unit")
		}
	case TimestampAdd, TimestampDiff:
		count = 2
		if !validTimestampUnit(n.Unit.Name) {
			return errors.New("invalid timestamp function unit")
		}
	case Extract:
		count = 1
		if !validTimeUnit(n.Unit.Name) {
			return errors.New("invalid extract time unit")
		}
	case GetFormat:
		count = 1
		switch n.FormatSelector.Name {
		case "DATE", "DATETIME", "TIME", "TIMESTAMP":
		default:
			return errors.New("invalid get_format selector")
		}
	case "convert":
		if !n.Charset.Present() {
			return errors.New("CONVERT charset is missing")
		}
		if err := validateBareIdentifier("charset", n.Charset); err != nil {
			return err
		}
	case Trim:
		if len(n.Args) < 1 || len(n.Args) > 2 || n.TrimDirection < TrimBothDefault || n.TrimDirection > TrimTrailing ||
			n.TrimDirection != TrimBothDefault && len(n.Args) != 2 {
			return errors.New("invalid trim syntax")
		}
	}
	if name == Position && len(n.Args) != 2 {
		return errors.New("invalid position argument count")
	}
	if name == "convert" && len(n.Args) != 1 {
		return errors.New("invalid convert argument count")
	}
	for _, arg := range n.Args {
		if !nodePresent(arg) {
			return errors.New("nil function argument")
		}
	}
	if n.IntervalFirst && len(n.Args) == 2 && n.Args[1] == nil {
		return errors.New("leading interval requires an expression")
	}
	if n.IntervalFirst && (name != DateAdd || len(n.Args) != 2 || !validTimeUnit(n.Unit.Name)) {
		return errors.New("invalid leading interval syntax")
	}
	if count >= 0 && len(n.Args) != count {
		return errors.Errorf("invalid %s argument count", name)
	}
	return nil
}

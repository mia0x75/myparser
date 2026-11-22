package ast

// HasAggFlag checks if the expr contains FlagHasAggregateFunc.
func HasAggFlag(expr ExprNode) bool {
	return ExprFlag(expr)&FlagHasAggregateFunc > 0
}

// HasWindowFlag reports whether the expression contains a window function.
func HasWindowFlag(expr ExprNode) bool {
	return ExprFlag(expr)&FlagHasWindowFunc > 0
}

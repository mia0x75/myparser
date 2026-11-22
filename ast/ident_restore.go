package ast

import (
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
)

// restoreStoredIdent keeps parser spelling for unquoted stored-program names;
// hand-built identifiers still use the normal quoted AST rendering.
func restoreStoredIdent(ctx *format.RestoreCtx, ident lex.Ident) {
	if ident.Quote == lex.QuoteUnquoted && ident.Span.Valid() && (ident.Span.Start != 0 || ident.Span.End != 0) {
		ctx.WritePlain(ident.Name)
		return
	}
	ctx.WriteIdent(ident)
}

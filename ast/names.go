package ast

import (
	"io"
	"strings"

	pkast "github.com/mia0x75/parserkit/ast"

	"github.com/mia0x75/myparser/format"
)

// QualifiedName keeps the shared syntax-only name value.
type QualifiedName = pkast.QualifiedName

func restoreQualifiedName(ctx *format.RestoreCtx, name QualifiedName) error {
	for i, part := range name.Parts {
		if i != 0 {
			ctx.WritePlain(".")
		}
		ctx.WriteIdent(part)
	}
	return ctx.Err()
}

func formatQualifiedName(w io.Writer, name QualifiedName) {
	// Legacy Format lowercases even quoted names; Restore preserves their case.
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags|format.RestoreNameLowercase, w)
	if len(name.Parts) == 0 {
		ctx.WriteName("")
		return
	}
	for i, part := range name.Parts {
		if i != 0 {
			ctx.WritePlain(".")
		}
		ctx.WriteIdent(part)
	}
}

func originalQualifiedName(name QualifiedName) string {
	parts := make([]string, 0, len(name.Parts))
	for _, part := range name.Parts {
		parts = append(parts, part.Name)
	}
	return strings.Join(parts, ".")
}

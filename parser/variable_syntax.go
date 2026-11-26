package parser

import (
	"strings"

	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/myparser/ast"
)

// isBareSetScopeKeyword reports whether name is a SET scope keyword (GLOBAL, LOCAL,
// SESSION, PERSIST, PERSIST_ONLY) usable only as a scope prefix in SET, not as a
// variable name: "SET GLOBAL = DEFAULT" has no variable and is rejected by both
// MySQL and MariaDB.
func isBareSetScopeKeyword(name string) bool {
	switch strings.ToUpper(name) {
	case "GLOBAL", "LOCAL", "SESSION", "PERSIST", "PERSIST_ONLY":
		return true
	}
	return false
}

func syntaxVariableTarget(l yyLexer, name string, span loc.Span, kind ast.VariableTargetKind, prefix ast.VariablePrefix) *ast.VariableTarget {
	target := &ast.VariableTarget{
		Name:   syntaxName(syntaxIdent(l, name, span)),
		Kind:   kind,
		Prefix: prefix,
	}
	target.SetSpan(span)
	return target
}

func syntaxQualifiedVariableTarget(l yyLexer, left, right string, span, rightSpan loc.Span, kind ast.VariableTargetKind) *ast.VariableTarget {
	target := &ast.VariableTarget{
		Name: syntaxName(
			syntaxIdent(l, left, loc.Span{Start: span.Start, End: rightSpan.Start - 1}),
			syntaxIdent(l, right, rightSpan),
		),
		Kind: kind,
	}
	target.SetSpan(span)
	return target
}

func syntaxAtVariableTarget(l yyLexer, raw string, span loc.Span) *ast.VariableTarget {
	name := strings.TrimPrefix(raw, "@")
	nameSpan := span
	if span.Valid() {
		nameSpan.Start++
		source := yyGetParser(l).src[span.Start:span.End]
		if len(source) >= 3 && (source[1] == '`' || source[1] == '\'' || source[1] == '"') && source[len(source)-1] == source[1] {
			quote := source[1]
			name = strings.ReplaceAll(source[2:len(source)-1], string([]byte{quote, quote}), string(quote))
		}
	}
	target := syntaxVariableTarget(l, name, nameSpan, ast.VariableTargetUser, ast.VariablePrefixAt)
	target.SetSpan(span)
	return target
}

func syntaxSystemVariableTarget(l yyLexer, raw string, span loc.Span) *ast.VariableTarget {
	lower := strings.ToLower(raw)
	name := strings.TrimPrefix(raw, "@@")
	target := &ast.VariableTarget{Kind: ast.VariableTargetSystem, Prefix: ast.VariablePrefixDoubleAt}
	target.ScopeExplicit = true
	for _, scope := range []struct {
		prefix string
		scope  ast.VariableScope
	}{
		{prefix: "persist_only.", scope: ast.VariableScopePersistOnly},
		{prefix: "persist.", scope: ast.VariableScopePersist},
		{prefix: "global.", scope: ast.VariableScopeGlobal},
		{prefix: "session.", scope: ast.VariableScopeSession},
		{prefix: "local.", scope: ast.VariableScopeLocal},
	} {
		if strings.HasPrefix(lower, "@@"+scope.prefix) {
			name = raw[len("@@")+len(scope.prefix):]
			target.Scope = scope.scope
			break
		}
	}
	if target.Scope == ast.VariableScopeImplicit {
		target.ScopeExplicit = false
	}
	name = strings.ToLower(name)
	nameStart := span.Start + 2
	if span.Valid() && span.Within(len(yyGetParser(l).src)) {
		// Recover the exact prefix from the source so quoted identifiers retain Raw.
		source := yyGetParser(l).src[span.Start:span.End]
		for _, prefix := range []string{"persist_only.", "persist.", "global.", "session.", "local."} {
			if len(source) >= 2+len(prefix) && strings.EqualFold(source[2:2+len(prefix)], prefix) {
				nameStart = span.Start + loc.ByteOffset(2+len(prefix))
				break
			}
		}
	}
	nameSpan := loc.Span{Start: nameStart, End: span.End}
	target.Name = syntaxName(syntaxIdent(l, name, nameSpan))
	target.SetSpan(span)
	return target
}

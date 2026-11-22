package ast_test

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	dformat "github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
)

// TestFuncParamRestoreIdent checks the parameter name renders through the
// ident model and an empty ident fails instead of writing an empty name.
func TestFuncParamRestoreIdent(t *testing.T) {
	t.Parallel()

	quoted := &ast.FuncParam{Name: lex.Ident{Name: "p", Quote: lex.QuoteBacktick, Raw: "`p`"}}
	var sb strings.Builder
	if err := quoted.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &sb)); err != nil {
		t.Fatalf("quoted restore failed: %v", err)
	}
	if got, want := sb.String(), "`p` "; got != want {
		t.Fatalf("quoted restore is %q, want %q", got, want)
	}

	empty := &ast.FuncParam{Name: lex.Ident{}}
	sb.Reset()
	if err := empty.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &sb)); err == nil {
		t.Fatal("an empty parameter name must fail restore")
	}
}

// TestCreateTriggerRestoreIdent checks FOLLOWS only renders when the other
// trigger name is present and the name keeps its quoting.
func TestCreateTriggerRestoreIdent(t *testing.T) {
	t.Parallel()

	trigger := &ast.CreateTriggerStmt{
		Name:             &ast.TableName{},
		Table:            &ast.TableName{},
		Order:            "FOLLOWS",
		OtherTriggerName: lex.Ident{Name: "other"},
		Body:             &ast.BeginEndStmt{},
	}
	var sb strings.Builder
	if err := trigger.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &sb)); err != nil {
		t.Fatalf("trigger restore failed: %v", err)
	}
	if !strings.Contains(sb.String(), "FOLLOWS `other`") {
		t.Fatalf("trigger restore is %q, want FOLLOWS `other`", sb.String())
	}

	trigger.OtherTriggerName = lex.Ident{}
	sb.Reset()
	if err := trigger.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &sb)); err != nil {
		t.Fatalf("trigger restore failed: %v", err)
	}
	if strings.Contains(sb.String(), "FOLLOWS") {
		t.Fatalf("absent other trigger name must not render the clause: %q", sb.String())
	}
}

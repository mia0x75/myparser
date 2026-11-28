package ast_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"

	. "github.com/mia0x75/myparser/ast"
	. "github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/parser"
)

func TestCacheable(t *testing.T) {
	// test non-SelectStmt
	var stmt Node = &DeleteStmt{}
	if IsReadOnly(stmt) {
		t.Fatal("DeleteStmt is expected to be not read only")
	}

	stmt = &InsertStmt{}
	if IsReadOnly(stmt) {
		t.Fatal("InsertStmt is expected to be not read only")
	}

	stmt = &UpdateStmt{}
	if IsReadOnly(stmt) {
		t.Fatal("UpdateStmt is expected to be not read only")
	}

	stmt = &ExplainStmt{}
	if !IsReadOnly(stmt) {
		t.Fatal("ExplainStmt is expected to be read only")
	}

	stmt = &ExplainStmt{}
	if !IsReadOnly(stmt) {
		t.Fatal("ExplainStmt is expected to be read only")
	}

	stmt = &DoStmt{}
	if !IsReadOnly(stmt) {
		t.Fatal("DoStmt is expected to be read only")
	}
}

// A locking read nested in a subquery or derived table makes the outer
// select non-cacheable even when its own tail has no lock.
func TestIsReadOnlySubqueryLocks(t *testing.T) {
	parser := parser.New()
	for _, tc := range []struct {
		sql  string
		want bool
	}{
		{"SELECT * FROM t WHERE x = (SELECT y FROM u)", true},
		{"SELECT * FROM t WHERE x = (SELECT y FROM u FOR UPDATE)", false},
		{"SELECT * FROM t WHERE EXISTS (SELECT y FROM u)", true},
		{"SELECT * FROM t WHERE EXISTS (SELECT y FROM u FOR UPDATE)", false},
		{"SELECT * FROM (SELECT y FROM u FOR UPDATE) d", false},
		{"SELECT * FROM (SELECT y FROM u) d", true},
	} {
		stmt, err := parser.ParseOneStmt(tc.sql)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tc.sql, err)
		}
		if got := IsReadOnly(stmt); got != tc.want {
			t.Errorf("IsReadOnly(%q) = %t, want %t", tc.sql, got, tc.want)
		}
	}
}

// CleanNodeText set the text of node and all child node empty.
// For test only.
func CleanNodeText(node Node) {
	var cleaner nodeTextCleaner
	node.Accept(&cleaner)
}

// nodeTextCleaner clean the text of a node and it's child node.
// For test only.
type nodeTextCleaner struct {
}

// Enter implements Visitor interface.
func (checker *nodeTextCleaner) Enter(in Node) (out Node, skipChildren bool) {
	in.SetText("")
	switch node := in.(type) {
	case *SetOprStmt:
		node.Tail.SetSpan(loc.InvalidSpan())
	case *QueryTailStmt:
		node.Tail.SetSpan(loc.InvalidSpan())
	case *Constraint:
		if node.Option != nil {
			if node.Option.KeyBlockSize == 0x0 && node.Option.Tp == 0 && node.Option.Comment == "" {
				node.Option = nil
			}
		}
	case *FuncCallExpr:
		normalizeCallName(&node.Name)
		for _, ident := range []*lex.Ident{&node.Unit, &node.FormatSelector} {
			ident.Name = strings.ToUpper(ident.Name)
			ident.Raw = ""
			ident.Quote = lex.QuoteUnquoted
			ident.Span = loc.InvalidSpan()
		}
		if len(node.Name.Parts) == 1 && strings.EqualFold(node.Name.Parts[0].Name, "convert") {
			node.Charset.Name = strings.ToLower(node.Charset.Name)
			node.Charset.Raw = ""
			node.Charset.Quote = lex.QuoteUnquoted
			node.Charset.Span = loc.InvalidSpan()
		}
	case *VariableTarget:
		normalizeName(&node.Name)
		if node.Scope == VariableScopeLocal {
			node.Scope = VariableScopeSession
		}
	case *CreateFunctionStmt:
		for _, param := range node.Params {
			normalizeIdent(&param.Name)
		}
	case *CreateProcedureStmt:
		for _, param := range node.Params {
			normalizeIdent(&param.Name)
		}
	case *CreateTriggerStmt:
		normalizeIdent(&node.OtherTriggerName)
	case *AggregateFuncExpr:
		normalizeCallName(&node.Name)
	case *WindowFuncExpr:
		normalizeCallName(&node.Name)
	case *FieldList:
		for _, item := range node.Items {
			normalizeSelectItem(item)
		}
	case *TableName:
		normalizeName(&node.Name)
	case *PeriodDefinition:
		normalizeIdent(&node.Name)
	case *PartitionDefinition:
		normalizeAlias(&node.Name)
	case *ValuesStmt:
		node.ValuesKeywordOffset = 0
	case *FrameClause:
		node.TypeOffset, node.ExclusionOffset = 0, 0
	case *FrameBound:
		node.ExprOffset = 0
		node.Unit.Name = strings.ToUpper(node.Unit.Name)
		node.Unit.Raw = ""
		node.Unit.Quote = lex.QuoteUnquoted
		node.Unit.Span = loc.InvalidSpan()
	case *InsertStmt:
		if node.RowAlias != nil {
			node.RowAlias.Span = loc.InvalidSpan()
		}
	case *AlterTableSpec:
		for _, opt := range node.Options {
			opt.StrValue = strings.ToLower(opt.StrValue)
		}
	}
	return in, false
}

// Leave implements Visitor interface.
func (checker *nodeTextCleaner) Leave(in Node) (out Node, ok bool) {
	return in, true
}

type NodeRestoreTestCase struct {
	sourceSQL string
	expectSQL string
}

type normalizeSyntaxVisitor struct{}

// Restore changes spelling and positions, but must preserve decoded name parts.
func normalizeName(qn *QualifiedName) {
	if qn == nil {
		return
	}
	qn.Span = loc.InvalidSpan()
	for i := range qn.Parts {
		qn.Parts[i].Span = loc.InvalidSpan()
		qn.Parts[i].Raw = ""
		qn.Parts[i].Quote = 0
	}
}

func normalizeNames(names []QualifiedName) {
	for i := range names {
		normalizeName(&names[i])
	}
}

func normalizeIdent(ident *lex.Ident) {
	if ident == nil {
		return
	}
	ident.Raw = ""
	ident.Quote = lex.QuoteUnquoted
	ident.Span = loc.InvalidSpan()
}

// normalizeSelectItem is the shared test helper: it clears projection
// metadata without losing the decoded name parts or alias presence.
func normalizeSelectItem(item *SelectItem) {
	if item == nil {
		return
	}
	item.SetSpan(loc.InvalidSpan())
	if wildcard, ok := item.Expr.(*WildcardExpr); ok {
		normalizeName(&wildcard.Qualifier)
	}
	normalizeAlias(&item.Alias)
}

// normalizeAlias keeps a quoted empty alias present; an absent alias becomes
// the canonical invalid-span value.
func normalizeAlias(alias *lex.Ident) {
	if !alias.Present() {
		*alias = lex.Ident{Span: loc.InvalidSpan()}
		return
	}
	if alias.Name == "" {
		alias.Quote = lex.QuoteBacktick
		alias.Raw = "``"
	} else {
		alias.Raw = ""
		alias.Quote = lex.QuoteUnquoted
	}
	alias.Span = loc.InvalidSpan()
}

func normalizeIdents(idents []lex.Ident) {
	for i := range idents {
		normalizeAlias(&idents[i])
	}
}

func normalizeTypeIdent(ident *lex.Ident) {
	if ident == nil {
		return
	}
	ident.Name = strings.ToLower(ident.Name)
	normalizeIdent(ident)
}

func normalizeCallName(name *QualifiedName) {
	if name == nil {
		return
	}
	name.Span = loc.InvalidSpan()
	for i := range name.Parts {
		name.Parts[i].Name = strings.ToLower(name.Parts[i].Name)
		normalizeAlias(&name.Parts[i])
	}
}

func TestNormalizeCallNameKeepsQuotedEmptyPart(t *testing.T) {
	name := QualifiedName{Parts: []lex.Ident{{Quote: lex.QuoteBacktick}}}
	normalizeCallName(&name)
	if len(name.Parts) != 1 || !name.Parts[0].Present() || name.Parts[0].Quote != lex.QuoteBacktick {
		t.Fatal("normalization must not erase a quoted empty function name")
	}
}

func TestNormalizeNamePreservesSemantics(t *testing.T) {
	for _, tc := range []struct {
		name        string
		left, right QualifiedName
		equal       bool
	}{
		{"spelling only", QualifiedName{Parts: []lex.Ident{{Name: "A", Raw: "A", Span: loc.Span{Start: 1, End: 2}}}},
			QualifiedName{Parts: []lex.Ident{{Name: "A", Raw: "`A`", Quote: lex.QuoteBacktick, Span: loc.Span{Start: 7, End: 10}}}}, true},
		{"case differs", qualName("A"), qualName("a"), false},
		{"parts differ", qualName("a.b"), qualName("a", "b"), false},
		{"absent and empty quoted name", QualifiedName{}, qualName(""), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			normalizeName(&tc.left)
			normalizeName(&tc.right)
			if equal := reflect.DeepEqual(tc.left, tc.right); equal != tc.equal {
				t.Fatalf("normalized equality = %v, want %v: left=%+v right=%+v", equal, tc.equal, tc.left, tc.right)
			}
		})
	}
}

func (v normalizeSyntaxVisitor) Enter(n Node) (Node, bool) {
	switch e := n.(type) {
	case *TableSource:
		normalizeAlias(&e.Alias)
		normalizeIdents(e.Columns)
		for _, hint := range e.IndexHints {
			normalizeIdents(hint.IndexNames)
			hint.Span = loc.InvalidSpan()
		}
	case *CommonTableExpression:
		normalizeAlias(&e.Name)
		normalizeIdents(e.Columns)
	case *WithClause:
		normalizeIdents(e.CycleColumns)
	case *WindowSpec:
		normalizeAlias(&e.Name)
		normalizeAlias(&e.Ref)
	case *CreateViewStmt:
		normalizeIdents(e.Columns)
	case *AlterViewStmt:
		normalizeIdents(e.Columns)
	case *CreateDatabaseStmt:
		normalizeIdent(&e.Name)
	case *AlterDatabaseStmt:
		normalizeIdent(&e.Name)
	case *DropDatabaseStmt:
		normalizeIdent(&e.Name)
	case *AnalyzeTableStmt:
		normalizeIdents(e.Columns)
		normalizeIdents(e.PartitionNames)
		normalizeIdents(e.IndexNames)
	case *TableName:
		normalizeIdents(e.PartitionNames)
	case *ColumnRef:
		normalizeName(&e.Name)
	case *ReplicationOption:
		normalizeTypeIdent(&e.Name)
	case *DefaultExpr:
		normalizeName(&e.Name)
	case *IndexKeyPart:
		normalizeName(&e.Column)
	case *Constraint:
		normalizeIdent(&e.Name)
		normalizeIdent(&e.IndexName)
	case *CreateIndexStmt:
		normalizeIdent(&e.IndexName)
	case *DropIndexStmt:
		normalizeIdent(&e.IndexName)
	case *ColumnDef:
		normalizeName(&e.Name)
	case *PeriodDefinition:
		normalizeName(&e.Start)
		normalizeName(&e.End)
		normalizeIdent(&e.Name)
	case *AlterTableSpec:
		normalizeIdents(e.PartitionNames)
		normalizeName(e.OldColumnName)
		normalizeName(e.NewColumnName)
		normalizeIdent(&e.Name)
		normalizeIdent(&e.FromKey)
		normalizeIdent(&e.ToKey)
	case *ColumnPosition:
		normalizeName(e.RelativeColumn)
	case *ShowStmt:
		normalizeName(e.Column)
		normalizeIdent(&e.Database)
	case *DeclareStmt:
		normalizeName(e.TypeRef)
	case *CreateTriggerStmt:
		normalizeNames(e.UpdateColumns)
	case *JoinExpr:
		normalizeNames(e.Using)
	case *MatchAgainst:
		normalizeNames(e.Columns)
	case *PartitionOptions:
		normalizeNames(e.ColumnNames)
	case *InsertStmt:
		if e.RowAlias != nil {
			e.RowAlias.Span = loc.InvalidSpan()
			normalizeAlias(&e.RowAlias.Name)
			normalizeNames(e.RowAlias.Columns)
		}
	case *PrivElem:
		normalizeNames(e.Cols)
	case *LoadDataStmt:
		normalizeNames(e.Columns)
		for _, column := range e.ColumnsAndUserVars {
			normalizeName(column.ColumnName)
		}
	case *LoadXMLStmt:
		for _, column := range e.ColumnsAndUserVars {
			normalizeName(column.ColumnName)
		}
	case *TypeName:
		normalizeTypeIdent(&e.Charset)
		normalizeTypeIdent(&e.Collation)
	case *LiteralExpr:
		e.Raw = ""
		e.Quote = lex.QuoteUnquoted
	case *FuncCallExpr:
		for _, ident := range []*lex.Ident{&e.Unit, &e.FormatSelector, &e.Charset} {
			normalizeTypeIdent(ident)
		}
	case *FrameBound:
		normalizeTypeIdent(&e.Unit)
	case *AggregateFuncExpr:
		normalizeCallName(&e.Name)
	case *WindowFuncExpr:
		normalizeCallName(&e.Name)
	case *UseStmt:
		normalizeIdent(&e.Database)
	case *ExplainStmt:
		normalizeIdent(&e.Schema)
	case *PrepareStmt:
		normalizeAlias(&e.Name)
	case *ExecuteStmt:
		normalizeAlias(&e.Name)
	case *DeallocateStmt:
		normalizeAlias(&e.Name)
	case *RollbackStmt:
		normalizeAlias(&e.SavepointName)
	case *SavepointStmt:
		normalizeAlias(&e.Name)
	case *ReleaseSavepointStmt:
		normalizeAlias(&e.Name)
	case *JSONTableColumn:
		normalizeAlias(&e.Name)
	case *GrantObject:
		for i := range e.Parts {
			normalizeAlias(&e.Parts[i].Name)
			e.Parts[i].WildcardSpan = loc.InvalidSpan()
		}
	case *HandlerStmt:
		normalizeAlias(&e.Alias)
		normalizeAlias(&e.Index)
	case *CacheIndexTable:
		normalizeIdents(e.IndexNames)
	case *LoadIndexTable:
		normalizeIdents(e.IndexNames)
	case *CacheIndexTarget:
		normalizeAlias(&e.Name)
	case *CreateTablespaceStmt:
		normalizeAlias(&e.Name)
	case *AlterTablespaceStmt:
		normalizeAlias(&e.Name)
		normalizeAlias(&e.RenameTo)
	case *DropTablespaceStmt:
		normalizeAlias(&e.Name)
	case *TablespaceOption:
		normalizeAlias(&e.Engine)
	case *CreateLogfileGroupStmt:
		normalizeAlias(&e.Name)
	case *AlterLogfileGroupStmt:
		normalizeAlias(&e.Name)
	case *DropLogfileGroupStmt:
		normalizeAlias(&e.Name)
	case *LogfileGroupOption:
		normalizeAlias(&e.Engine)
	case *CreateResourceGroupStmt:
		normalizeAlias(&e.Name)
		if e.Type.Quote == lex.QuoteUnquoted && (strings.EqualFold(e.Type.Name, "USER") || strings.EqualFold(e.Type.Name, "SYSTEM")) {
			e.Type.Name = strings.ToUpper(e.Type.Name)
		}
		normalizeAlias(&e.Type)
	case *AlterResourceGroupStmt:
		normalizeAlias(&e.Name)
	case *DropResourceGroupStmt:
		normalizeAlias(&e.Name)
	case *CreateServerStmt:
		normalizeAlias(&e.Name)
	case *AlterServerStmt:
		normalizeAlias(&e.Name)
	case *DropServerStmt:
		normalizeAlias(&e.Name)
	case *ServerOption:
		normalizeAlias(&e.Name)
	case *InstallPluginStmt:
		normalizeAlias(&e.Name)
	case *UninstallPluginStmt:
		normalizeAlias(&e.Name)
	}
	n.SetSpan(loc.InvalidSpan())
	return n, false
}

func (v normalizeSyntaxVisitor) Leave(n Node) (Node, bool) { return n, true }

func normalizeSyntaxMetadata(root Node) {
	root.Accept(normalizeSyntaxVisitor{})
}

func RunNodeRestoreTest(t *testing.T, nodeTestCases []NodeRestoreTestCase, template string, extractNodeFunc func(node Node) Node) {
	RunNodeRestoreTestWithFlags(t, nodeTestCases, template, extractNodeFunc, DefaultRestoreFlags)
}

func RunNodeRestoreTestWithFlags(t *testing.T, nodeTestCases []NodeRestoreTestCase, template string, extractNodeFunc func(node Node) Node, flags RestoreFlags) {
	p := parser.New()
	p.EnableWindowFunc(true)
	for _, testCase := range nodeTestCases {
		sourceSQL := fmt.Sprintf(template, testCase.sourceSQL)
		expectSQL := fmt.Sprintf(template, testCase.expectSQL)
		stmt, err := p.ParseOneStmt(sourceSQL)
		comment := fmt.Sprintf("source %#v", testCase)
		if err != nil {
			t.Fatalf("%s: %v", comment, err)
		}
		var sb strings.Builder
		err = extractNodeFunc(stmt).Restore(NewRestoreCtx(flags, &sb))
		if err != nil {
			t.Fatalf("%s: %v", comment, err)
		}
		restoreSql := fmt.Sprintf(template, sb.String())
		comment = fmt.Sprintf("source %#v; restore %v", testCase, restoreSql)
		if restoreSql != expectSQL {
			t.Fatalf("%s: got %#v, want %#v", comment, restoreSql, expectSQL)
		}
		stmt2, err := p.ParseOneStmt(restoreSql)
		if err != nil {
			t.Fatalf("%s: %v", comment, err)
		}
		// Compare syntax independently of Restore's spelling and byte positions.
		CleanNodeText(stmt)
		CleanNodeText(stmt2)
		normalizeSyntaxMetadata(stmt)
		normalizeSyntaxMetadata(stmt2)
		if !reflect.DeepEqual(stmt2, stmt) {
			gotSelect, gotOK := stmt2.(*SelectStmt)
			wantSelect, wantOK := stmt.(*SelectStmt)
			if gotOK && wantOK && len(gotSelect.Fields.Items) > 0 && len(wantSelect.Fields.Items) > 0 {
				t.Fatalf("%s: got expr %#v, want %#v", comment, gotSelect.Fields.Items[0].Expr, wantSelect.Fields.Items[0].Expr)
			}
			t.Fatalf("%s: got %#v, want %#v", comment, stmt2, stmt)
		}
	}
}

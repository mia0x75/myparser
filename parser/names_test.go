package parser

import (
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/mia0x75/myparser/ast"
	myformat "github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/mysql"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

// testSyntaxLiteral builds a syntax-only string literal for name tests.
func testSyntaxLiteral(value string) ast.ExprNode {
	return &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralString, Value: value}}
}

func testSyntaxName(parts ...string) ast.QualifiedName {
	idents := make([]lex.Ident, len(parts))
	for i, part := range parts {
		idents[i] = lex.Ident{Name: part, Span: loc.InvalidSpan()}
	}
	return syntaxName(idents...)
}

// Restore rewrites spelling and ranges, not decoded identifier parts.
func normalizeTestName(name *ast.QualifiedName) {
	if name == nil {
		return
	}
	name.Span = loc.InvalidSpan()
	for i := range name.Parts {
		name.Parts[i].Raw = ""
		name.Parts[i].Quote = lex.QuoteUnquoted
		name.Parts[i].Span = loc.InvalidSpan()
	}
}

func normalizeTestNames(names []ast.QualifiedName) {
	for i := range names {
		normalizeTestName(&names[i])
	}
}

func normalizeTestIdent(ident *lex.Ident) {
	if !ident.Present() {
		*ident = lex.Ident{Span: loc.InvalidSpan()}
		return
	}
	if ident.Name == "" {
		ident.Quote = lex.QuoteBacktick
		ident.Raw = "``"
	} else {
		ident.Raw = ""
		ident.Quote = lex.QuoteUnquoted
	}
	ident.Span = loc.InvalidSpan()
}

func normalizeTestIdents(idents []lex.Ident) {
	for i := range idents {
		normalizeTestIdent(&idents[i])
	}
}

func normalizeTestReplicationNames(options []*ast.ReplicationOption) {
	for _, option := range options {
		option.Name.Name = strings.ToUpper(option.Name.Name)
	}
}

func normalizeTestNameMetadata(node ast.Node) {
	node.SetSpan(loc.InvalidSpan())
	normalizeTestManagementNames(node)
	switch node := node.(type) {
	case *ast.TableSource:
		normalizeTestIdent(&node.Alias)
		normalizeTestIdents(node.Columns)
	case *ast.CommonTableExpression:
		normalizeTestIdent(&node.Name)
		normalizeTestIdents(node.Columns)
	case *ast.WithClause:
		normalizeTestIdents(node.CycleColumns)
	case *ast.WindowSpec:
		normalizeTestIdent(&node.Name)
		normalizeTestIdent(&node.Ref)
	case *ast.CreateViewStmt:
		normalizeTestIdents(node.Columns)
	case *ast.AlterViewStmt:
		normalizeTestIdents(node.Columns)
	case *ast.AnalyzeTableStmt:
		normalizeTestIdents(node.Columns)
	case *ast.TableName:
		normalizeTestName(&node.Name)
	case *ast.ColumnRef:
		normalizeTestName(&node.Name)
	case *ast.ReplicationOption:
		normalizeTestIdent(&node.Name)
	case *ast.DefaultExpr:
		normalizeTestName(&node.Name)
	case *ast.FuncCallExpr:
		normalizeTestName(&node.Name)
	case *ast.AggregateFuncExpr:
		normalizeTestName(&node.Name)
	case *ast.WindowFuncExpr:
		normalizeTestName(&node.Name)
	case *ast.ColumnDef:
		normalizeTestName(&node.Name)
	case *ast.IndexKeyPart:
		normalizeTestName(&node.Column)
	case *ast.PeriodDefinition:
		normalizeTestIdent(&node.Name)
		normalizeTestName(&node.Start)
		normalizeTestName(&node.End)
	case *ast.AlterTableSpec:
		normalizeTestName(node.OldColumnName)
		normalizeTestName(node.NewColumnName)
		normalizeTestIdent(&node.Name)
		normalizeTestIdent(&node.FromKey)
		normalizeTestIdent(&node.ToKey)
	case *ast.ColumnPosition:
		normalizeTestName(node.RelativeColumn)
	case *ast.UseStmt:
		normalizeTestIdent(&node.Database)
	case *ast.ShowStmt:
		normalizeTestName(node.Column)
		normalizeTestIdent(&node.Database)
	case *ast.ExplainStmt:
		normalizeTestIdent(&node.Schema)
	case *ast.DeclareStmt:
		normalizeTestName(node.TypeRef)
		normalizeTestIdents(node.Names)
	case *ast.VariableTarget:
		normalizeTestName(&node.Name)
	case *ast.VariableExpr:
		if node.Target != nil {
			normalizeTestName(&node.Target.Name)
		}
	case *ast.VariableAssignment:
		if node.Target != nil {
			normalizeTestName(&node.Target.Name)
		}
	case *ast.DiagnosticsItem:
		if node.Target != nil {
			normalizeTestName(&node.Target.Name)
		}
	case *ast.ResetStmt:
		normalizeTestName(&node.PersistName)
	case *ast.CreateFunctionStmt:
		for _, param := range node.Params {
			normalizeTestIdent(&param.Name)
		}
	case *ast.CreateProcedureStmt:
		for _, param := range node.Params {
			normalizeTestIdent(&param.Name)
		}
	case *ast.CreateTriggerStmt:
		normalizeTestIdent(&node.OtherTriggerName)
		normalizeTestNames(node.UpdateColumns)
	case *ast.BeginEndStmt:
		normalizeTestIdent(&node.Label)
	case *ast.WhileStmt:
		normalizeTestIdent(&node.Label)
	case *ast.RepeatStmt:
		normalizeTestIdent(&node.Label)
	case *ast.LoopStmt:
		normalizeTestIdent(&node.Label)
	case *ast.ForStmt:
		normalizeTestIdent(&node.Label)
		normalizeTestIdent(&node.Variable)
	case *ast.LeaveStmt:
		normalizeTestIdent(&node.Label)
	case *ast.IterateStmt:
		normalizeTestIdent(&node.Label)
	case *ast.DeclareConditionStmt:
		normalizeTestIdent(&node.Name)
	case *ast.DeclareCursorStmt:
		normalizeTestIdent(&node.Name)
	case *ast.OpenStmt:
		normalizeTestIdent(&node.Name)
	case *ast.CloseStmt:
		normalizeTestIdent(&node.Name)
	case *ast.FetchStmt:
		normalizeTestIdent(&node.Name)
	case *ast.SignalStmt:
		for _, item := range node.InfoItems {
			normalizeTestIdent(&item.Name)
		}
	case *ast.ResignalStmt:
		for _, item := range node.InfoItems {
			normalizeTestIdent(&item.Name)
		}
	case *ast.JoinExpr:
		normalizeTestNames(node.Using)
	case *ast.MatchAgainst:
		normalizeTestNames(node.Columns)
	case *ast.PartitionOptions:
		normalizeTestNames(node.ColumnNames)
	case *ast.InsertStmt:
		if node.RowAlias != nil {
			node.RowAlias.Span = loc.InvalidSpan()
			normalizeTestIdent(&node.RowAlias.Name)
			normalizeTestNames(node.RowAlias.Columns)
		}
	case *ast.PrivElem:
		normalizeTestNames(node.Cols)
	case *ast.LoadDataStmt:
		normalizeTestNames(node.Columns)
		for _, column := range node.ColumnsAndUserVars {
			normalizeTestName(column.ColumnName)
		}
	case *ast.LoadXMLStmt:
		for _, column := range node.ColumnsAndUserVars {
			normalizeTestName(column.ColumnName)
		}
	case *ast.ChangeMasterStmt:
		normalizeTestReplicationNames(node.Options)
	case *ast.PrepareStmt:
		normalizeTestIdent(&node.Name)
	case *ast.DeallocateStmt:
		normalizeTestIdent(&node.Name)
	case *ast.ExecuteStmt:
		normalizeTestIdent(&node.Name)
	case *ast.RollbackStmt:
		normalizeTestIdent(&node.SavepointName)
	case *ast.SavepointStmt:
		normalizeTestIdent(&node.Name)
	case *ast.ReleaseSavepointStmt:
		normalizeTestIdent(&node.Name)
	case *ast.ReplicaControlStmt:
		normalizeTestReplicationNames(node.UntilOptions)
		normalizeTestReplicationNames(node.ConnectionOptions)
	}
}

func normalizeTestManagementNames(node ast.Node) {
	switch n := node.(type) {
	case *ast.JSONTableColumn:
		normalizeTestIdent(&n.Name)
	case *ast.GrantObject:
		for i := range n.Parts {
			normalizeTestIdent(&n.Parts[i].Name)
			n.Parts[i].WildcardSpan = loc.InvalidSpan()
		}
	case *ast.HandlerStmt:
		normalizeTestIdent(&n.Alias)
		normalizeTestIdent(&n.Index)
	case *ast.CacheIndexTable:
		normalizeTestIdents(n.IndexNames)
	case *ast.LoadIndexTable:
		normalizeTestIdents(n.IndexNames)
	case *ast.CacheIndexTarget:
		normalizeTestIdent(&n.Name)
	case *ast.CreateTablespaceStmt:
		normalizeTestIdent(&n.Name)
	case *ast.AlterTablespaceStmt:
		normalizeTestIdent(&n.Name)
		normalizeTestIdent(&n.RenameTo)
	case *ast.DropTablespaceStmt:
		normalizeTestIdent(&n.Name)
	case *ast.TablespaceOption:
		normalizeTestIdent(&n.Engine)
	case *ast.CreateLogfileGroupStmt:
		normalizeTestIdent(&n.Name)
	case *ast.AlterLogfileGroupStmt:
		normalizeTestIdent(&n.Name)
	case *ast.DropLogfileGroupStmt:
		normalizeTestIdent(&n.Name)
	case *ast.LogfileGroupOption:
		normalizeTestIdent(&n.Engine)
	case *ast.CreateResourceGroupStmt:
		normalizeTestIdent(&n.Name)
		if n.Type.Quote == lex.QuoteUnquoted && (strings.EqualFold(n.Type.Name, "USER") || strings.EqualFold(n.Type.Name, "SYSTEM")) {
			n.Type.Name = strings.ToUpper(n.Type.Name)
		}
		normalizeTestIdent(&n.Type)
	case *ast.AlterResourceGroupStmt:
		normalizeTestIdent(&n.Name)
	case *ast.DropResourceGroupStmt:
		normalizeTestIdent(&n.Name)
	case *ast.CreateServerStmt:
		normalizeTestIdent(&n.Name)
	case *ast.AlterServerStmt:
		normalizeTestIdent(&n.Name)
	case *ast.DropServerStmt:
		normalizeTestIdent(&n.Name)
	case *ast.ServerOption:
		normalizeTestIdent(&n.Name)
	case *ast.InstallPluginStmt:
		normalizeTestIdent(&n.Name)
	case *ast.UninstallPluginStmt:
		normalizeTestIdent(&n.Name)
	}
}

func TestNameNormalizationPreservesSemantics(t *testing.T) {
	for _, tc := range []struct {
		name        string
		left, right ast.Node
		equal       bool
	}{
		{"identifier case", columnRef(testSyntaxName("A"), loc.InvalidSpan()), columnRef(testSyntaxName("a"), loc.InvalidSpan()), false},
		{"name parts", columnRef(testSyntaxName("a.b"), loc.InvalidSpan()), columnRef(testSyntaxName("a", "b"), loc.InvalidSpan()), false},
		{"optional name", &ast.DefaultExpr{}, &ast.DefaultExpr{Name: testSyntaxName("")}, false},
		{"ordinary assignment", &ast.Assignment{Target: columnRef(testSyntaxName("A"), loc.InvalidSpan()), Expr: testSyntaxLiteral("1")},
			&ast.Assignment{Target: columnRef(testSyntaxName("a"), loc.InvalidSpan()), Expr: testSyntaxLiteral("1")}, false},
		{"replication keyword", &ast.ChangeMasterStmt{Options: []*ast.ReplicationOption{{Name: lex.Ident{Name: "master_host"}, Expr: testSyntaxLiteral("db")}}},
			&ast.ChangeMasterStmt{Options: []*ast.ReplicationOption{{Name: lex.Ident{Name: "MASTER_HOST"}, Expr: testSyntaxLiteral("db")}}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			CleanNodeText(tc.left)
			CleanNodeText(tc.right)
			if equal := reflect.DeepEqual(tc.left, tc.right); equal != tc.equal {
				t.Fatalf("normalized equality = %v, want %v: left=%#v right=%#v", equal, tc.equal, tc.left, tc.right)
			}
		})
	}
}

func TestInsertNormalizationPreservesSyntax(t *testing.T) {
	for _, tc := range []struct {
		name, left, right string
		equal             bool
	}{
		{"spelling only", "insert into `t` (`a`) values (1)", "INSERT INTO t (a) VALUES (1)", true},
		{"target table", "insert into t (a) values (1)", "insert into u (a) values (1)", false},
		{"column target", "insert into t (a) values (1)", "insert into t (b) values (1)", false},
		{"source expression", "insert into t (a) values (1)", "insert into t (a) values (2)", false},
		{"column-list presence", "insert into t values (1)", "insert into t () values (1)", false},
		{"source kind", "insert into t set a=1", "insert into t (a) values (1)", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			left, right := mustParseOne(t, tc.left), mustParseOne(t, tc.right)
			CleanNodeText(left)
			CleanNodeText(right)
			if equal := reflect.DeepEqual(left, right); equal != tc.equal {
				t.Fatalf("normalized INSERT equality = %v, want %v", equal, tc.equal)
			}
		})
	}
}

func TestSyntaxNameSpellingAndReuse(t *testing.T) {
	parser := New()
	for _, tc := range []struct {
		name, expr string
		names, raw []string
		quotes     []lex.QuoteStyle
		mode       mysql.SQLMode
	}{
		{"unquoted", "Db.Tab.Col", []string{"Db", "Tab", "Col"}, []string{"Db", "Tab", "Col"}, []lex.QuoteStyle{0, 0, 0}, 0},
		{"quoted dot", "`a.b`", []string{"a.b"}, []string{"`a.b`"}, []lex.QuoteStyle{lex.QuoteBacktick}, 0},
		{"escaped backtick", "`a``b`", []string{"a`b"}, []string{"`a``b`"}, []lex.QuoteStyle{lex.QuoteBacktick}, 0},
		{"ANSI quotes", `"a""b"."c.d"`, []string{`a"b`, "c.d"}, []string{`"a""b"`, `"c.d"`}, []lex.QuoteStyle{lex.QuoteDouble, lex.QuoteDouble}, mysql.ModeANSIQuotes},
		{"implicit schema", ".Tab.Col", []string{"Tab", "Col"}, []string{"Tab", "Col"}, []lex.QuoteStyle{0, 0}, 0},
		{"window keyword", "window", []string{"window"}, []string{"window"}, []lex.QuoteStyle{0}, 0},
		{"rank keyword", "rank", []string{"rank"}, []string{"rank"}, []lex.QuoteStyle{0}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parser.SetSQLMode(tc.mode)
			sql := "SELECT " + tc.expr
			stmt, err := parser.ParseOneStmt(sql)
			if err != nil {
				t.Fatal(err)
			}
			ref := stmt.(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.ColumnRef)
			if len(ref.Name.Parts) != len(tc.names) {
				t.Fatalf("parts = %v, want %v", ref.Name.Parts, tc.names)
			}
			wantSpan := loc.InvalidSpan()
			for i, part := range ref.Name.Parts {
				start := strings.Index(sql, tc.raw[i])
				span := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(tc.raw[i]))}
				if part.Name != tc.names[i] || part.Raw != tc.raw[i] || part.Quote != tc.quotes[i] || part.Span != span {
					t.Fatalf("part[%d] = %+v, want name=%q raw=%q quote=%d span=%v", i, part, tc.names[i], tc.raw[i], tc.quotes[i], span)
				}
				wantSpan = loc.Cover(wantSpan, span)
				if unsafe.StringData(part.Raw) != unsafe.StringData(sql[start:]) {
					t.Fatalf("part[%d] raw no longer shares the input", i)
				}
				if part.Quote == lex.QuoteUnquoted && unsafe.StringData(part.Name) != unsafe.StringData(part.Raw) {
					t.Fatalf("part[%d] unquoted name no longer shares the input", i)
				}
			}
			if ref.Name.Span != wantSpan {
				t.Fatalf("name span = %v, want %v", ref.Name.Span, wantSpan)
			}
			span := ref.Span()
			if span != (loc.Span{Start: 7, End: loc.ByteOffset(len(sql))}) {
				t.Fatalf("expression span = %v", span)
			}
			parts := append([]lex.Ident(nil), ref.Name.Parts...)
			var before, after strings.Builder
			if err := ref.Restore(myformat.NewRestoreCtx(myformat.DefaultRestoreFlags, &before)); err != nil {
				t.Fatal(err)
			}
			if _, err := parser.ParseOneStmt("SELECT `other``name`"); err != nil {
				t.Fatal(err)
			}
			if err := ref.Restore(myformat.NewRestoreCtx(myformat.DefaultRestoreFlags, &after)); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(ref.Name.Parts, parts) || before.String() != after.String() {
				t.Fatal("a later parse changed the retained name")
			}
		})
	}
}

func TestTableNameKeepsQualifiedPartsAndSpans(t *testing.T) {
	const sql = "CREATE TABLE `a.b`.`T``x` (id INT)"
	stmt, err := New().ParseOneStmt(sql)
	if err != nil {
		t.Fatal(err)
	}
	table := stmt.(*ast.CreateTableStmt).Table.(*ast.TableName)
	if len(table.Name.Parts) != 2 {
		t.Fatalf("table name parts = %d, want 2", len(table.Name.Parts))
	}
	want := []struct {
		name, raw string
		start     int
	}{
		{name: "a.b", raw: "`a.b`", start: strings.Index(sql, "`a.b`")},
		{name: "T`x", raw: "`T``x`", start: strings.Index(sql, "`T``x`")},
	}
	for i, part := range table.Name.Parts {
		if part.Name != want[i].name || part.Raw != want[i].raw || part.Quote != lex.QuoteBacktick {
			t.Fatalf("part[%d] = %+v, want name=%q raw=%q", i, part, want[i].name, want[i].raw)
		}
		span := loc.Span{Start: loc.ByteOffset(want[i].start), End: loc.ByteOffset(want[i].start + len(want[i].raw))}
		if part.Span != span {
			t.Fatalf("part[%d] span = %v, want %v", i, part.Span, span)
		}
	}
	if table.Name.Span != (loc.Span{Start: loc.ByteOffset(want[0].start), End: loc.ByteOffset(want[1].start + len(want[1].raw))}) {
		t.Fatalf("qualified table span = %v", table.Name.Span)
	}
	if table.Span() != table.Name.Span {
		t.Fatalf("table span = %v, want %v", table.Span(), table.Name.Span)
	}
}

func TestTableNameSpanOnDirectConstructionPaths(t *testing.T) {
	for _, tc := range []struct {
		name, sql string
		get       func(ast.StmtNode) *ast.TableName
	}{
		{
			name: "show index",
			sql:  "SHOW INDEX FROM `tbl` FROM `db`",
			get:  func(stmt ast.StmtNode) *ast.TableName { return stmt.(*ast.ShowStmt).Table },
		},
		{
			name: "handler close",
			sql:  "HANDLER `tbl` CLOSE",
			get:  func(stmt ast.StmtNode) *ast.TableName { return stmt.(*ast.HandlerStmt).Table },
		},
		{
			name: "handler read direction",
			sql:  "HANDLER `tbl` READ FIRST",
			get:  func(stmt ast.StmtNode) *ast.TableName { return stmt.(*ast.HandlerStmt).Table },
		},
		{
			name: "handler read index direction",
			sql:  "HANDLER `tbl` READ `idx` FIRST",
			get:  func(stmt ast.StmtNode) *ast.TableName { return stmt.(*ast.HandlerStmt).Table },
		},
		{
			name: "handler read index key",
			sql:  "HANDLER `tbl` READ `idx` = (1)",
			get:  func(stmt ast.StmtNode) *ast.TableName { return stmt.(*ast.HandlerStmt).Table },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmt, err := New().ParseOneStmt(tc.sql)
			if err != nil {
				t.Fatal(err)
			}
			table := tc.get(stmt)
			for _, part := range table.Name.Parts {
				start := strings.Index(tc.sql, part.Raw)
				wantSpan := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(part.Raw))}
				if start < 0 || part.Span != wantSpan {
					t.Fatalf("table part %+v has span %v, want %v", part, part.Span, wantSpan)
				}
			}
		})
	}
}

type syntaxNameVisitor struct {
	entered, left []ast.Node
	replacement   ast.Node
}

func (v *syntaxNameVisitor) Enter(n ast.Node) (ast.Node, bool) {
	v.entered = append(v.entered, n)
	return v.replacement, false
}

func (v *syntaxNameVisitor) Leave(n ast.Node) (ast.Node, bool) {
	v.left = append(v.left, n)
	return n, true
}

func TestColumnRefVisitorIdentity(t *testing.T) {
	ref := columnRef(testSyntaxName("original"), loc.InvalidSpan())
	replacement := columnRef(testSyntaxName("replacement"), loc.InvalidSpan())
	visitor := &syntaxNameVisitor{replacement: replacement}
	got, ok := ref.Accept(visitor)
	if !ok || got != replacement || len(visitor.entered) != 1 || visitor.entered[0] != ref || len(visitor.left) != 1 || visitor.left[0] != replacement {
		t.Fatalf("Accept = %v, %v; enter=%v leave=%v", got, ok, visitor.entered, visitor.left)
	}
}

func TestSyntaxNameRestrictions(t *testing.T) {
	for _, tc := range []struct {
		name, sql, errorText string
	}{
		{"bare now", "CREATE TABLE t (x INT DEFAULT now)", "DEFAULT NOW requires parentheses"},
		{"parenthesized now", "CREATE TABLE t (x INT DEFAULT NOW())", ""},
		{"qualified period", "CREATE TABLE t (x INT, PERIOD FOR p (db.start_col, end_col))", "PERIOD FOR requires two unqualified column names"},
		{"quoted dot period", "CREATE TABLE t (x INT, PERIOD FOR p (`start.col`, end_col))", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New().ParseOneStmt(tc.sql)
			if tc.errorText == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.errorText) {
				t.Fatalf("error = %v, want %q", err, tc.errorText)
			}
		})
	}
}

func TestReplicationOptionSyntaxSpelling(t *testing.T) {
	for _, tc := range []struct {
		name, sql string
		keywords  []string
	}{
		{"change", "change master to master_host='db'", []string{"master_host"}},
		{"list", "CHANGE MASTER TO ignore_domain_ids=(1,2)", []string{"ignore_domain_ids"}},
		{"until", "start replica until sql_before_gtids='x'", []string{"sql_before_gtids"}},
		{"connection", "start replica user='u' password='p' default_auth='a' plugin_dir='d'", []string{"user", "password", "default_auth", "plugin_dir"}},
		{"connection without user", "start replica password='p' default_auth='a' plugin_dir='d'", []string{"password", "default_auth", "plugin_dir"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stmt, err := New().ParseOneStmt(tc.sql)
			if err != nil {
				t.Fatal(err)
			}
			var options []*ast.ReplicationOption
			switch stmt := stmt.(type) {
			case *ast.ChangeMasterStmt:
				options = stmt.Options
			case *ast.ReplicaControlStmt:
				options = append(options, stmt.UntilOptions...)
				options = append(options, stmt.ConnectionOptions...)
			default:
				t.Fatalf("unexpected statement %T", stmt)
			}
			if len(options) != len(tc.keywords) {
				t.Fatalf("options = %d, want %d", len(options), len(tc.keywords))
			}
			for i, option := range options {
				keyword := tc.keywords[i]
				start := strings.Index(tc.sql, keyword)
				span := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(keyword))}
				part := option.Name
				if part.Name != keyword || part.Raw != keyword || part.Quote != lex.QuoteUnquoted || part.Span != span {
					t.Fatalf("option part = %+v, want %q at %v", part, keyword, span)
				}
				if replicationOptionName(option) != strings.ToUpper(keyword) {
					t.Fatalf("keyword comparison changed for %q", keyword)
				}
			}
		})
	}
}

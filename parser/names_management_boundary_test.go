package parser

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

type p59BoundaryVisitor func(ast.Node)

func (v p59BoundaryVisitor) Enter(n ast.Node) (ast.Node, bool) { v(n); return n, false }
func (p59BoundaryVisitor) Leave(n ast.Node) (ast.Node, bool)   { return n, true }

func p59Span(t *testing.T, sql string, span loc.Span, want string) {
	t.Helper()
	if !span.Within(len(sql)) || sql[span.Start:span.End] != want {
		t.Fatalf("span %v in %q, want %q", span, sql, want)
	}
}

func p59Ident(t *testing.T, sql string, name lex.Ident, decoded, raw string, quote lex.QuoteStyle) {
	t.Helper()
	if name.Name != decoded || name.Raw != raw || name.Quote != quote || !name.Present() {
		t.Fatalf("ident %+v, want %q/%q/%v", name, decoded, raw, quote)
	}
	p59Span(t, sql, name.Span, raw)
}

func TestManagementNamesBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, sql string
		check     func(*testing.T, string, ast.StmtNode)
	}{
		{"JSONTable", "SELECT * FROM JSON_TABLE(j, '$' COLUMNS (`名``x` FOR ORDINALITY, `` INT PATH '$' ERROR ON ERROR DEFAULT NULL ON EMPTY, e INT EXISTS PATH '$', NESTED '$.a' COLUMNS (c INT PATH '$'))) jt", func(t *testing.T, sql string, stmt ast.StmtNode) {
			var table *ast.JSONTable
			stmt.Accept(p59BoundaryVisitor(func(n ast.Node) {
				if n, ok := n.(*ast.JSONTable); ok {
					table = n
				}
			}))
			if table == nil || len(table.Columns) != 4 {
				t.Fatal("missing JSON_TABLE columns")
			}
			p59Span(t, sql, table.Span(), sql[strings.Index(sql, "JSON_TABLE"):len(sql)-3])
			p59Ident(t, sql, table.Columns[0].Name, "名`x", "`名``x`", lex.QuoteBacktick)
			p59Span(t, sql, table.Columns[0].Span(), "`名``x` FOR ORDINALITY")
			c := table.Columns[1]
			p59Ident(t, sql, c.Name, "", "``", lex.QuoteBacktick)
			p59Span(t, sql, c.Span(), "`` INT PATH '$' ERROR ON ERROR DEFAULT NULL ON EMPTY")
			if len(c.Behaviors) != 2 || c.Behaviors[0].On != ast.JSONTableBehaviorOnError || c.Behaviors[1].On != ast.JSONTableBehaviorOnEmpty {
				t.Fatal("behavior order lost")
			}
			p59Span(t, sql, c.Behaviors[0].Span(), "ERROR ON ERROR")
			p59Span(t, sql, c.Behaviors[1].Span(), "DEFAULT NULL ON EMPTY")
			if table.Columns[2].Kind != ast.JSONTableColumnExists || table.Columns[3].PathKeyword || table.Columns[3].Name.Present() || table.Columns[3].Name.Span.Valid() {
				t.Fatal("JSON_TABLE roles lost")
			}
			p59Span(t, sql, table.Columns[3].Span(), "NESTED '$.a' COLUMNS (c INT PATH '$')")
		}},
		{"Handler", "HANDLER `名` OPEN AS ``", func(t *testing.T, sql string, stmt ast.StmtNode) {
			n := stmt.(*ast.HandlerStmt)
			p59Ident(t, sql, n.Alias, "", "``", lex.QuoteBacktick)
			if n.Index.Present() || n.Index.Span.Valid() {
				t.Fatal("absent index has metadata")
			}
		}},
		{"HandlerIndex", "HANDLER t READ `a.b` >= (1) WHERE a>0 LIMIT 2", func(t *testing.T, sql string, stmt ast.StmtNode) {
			n := stmt.(*ast.HandlerStmt)
			p59Ident(t, sql, n.Index, "a.b", "`a.b`", lex.QuoteBacktick)
			if n.Where == nil || n.Limit == nil || len(n.Values) != 1 {
				t.Fatal("read clauses lost")
			}
		}},
		{"CacheIndex", "CACHE INDEX t, t2 INDEX (), t3 KEY (PRIMARY, `a.b`) IN DEFAULT", func(t *testing.T, sql string, stmt ast.StmtNode) {
			n := stmt.(*ast.CacheIndexStmt)
			if len(n.Tables) != 3 || n.Tables[0].IndexNames != nil || n.Tables[1].IndexNames == nil || len(n.Tables[1].IndexNames) != 0 || len(n.Tables[2].IndexNames) != 2 {
				t.Fatal("index list presence lost")
			}
			p59Ident(t, sql, n.Tables[2].IndexNames[0], "PRIMARY", "PRIMARY", lex.QuoteUnquoted)
			p59Ident(t, sql, n.Tables[2].IndexNames[1], "a.b", "`a.b`", lex.QuoteBacktick)
			p59Span(t, sql, n.Tables[2].Span(), "t3 KEY (PRIMARY, `a.b`)")
			p59Span(t, sql, n.Target.Span(), "DEFAULT")
			if n.Target.Kind != ast.CacheIndexDefaultTarget || n.Target.Name.Present() || n.Target.Name.Span.Valid() {
				t.Fatal("DEFAULT target lost")
			}
		}},
		{"CachePartition", "CACHE INDEX t PARTITION (`p.1`, ``) INDEX () IN `default`", func(t *testing.T, sql string, stmt ast.StmtNode) {
			n := stmt.(*ast.CacheIndexStmt)
			entry := n.Tables[0]
			p59Span(t, sql, entry.Table.Span(), "t PARTITION (`p.1`, ``)")
			p59Span(t, sql, entry.Span(), "t PARTITION (`p.1`, ``) INDEX ()")
			p59Ident(t, sql, entry.Table.PartitionNames[1], "", "``", lex.QuoteBacktick)
			p59Ident(t, sql, n.Target.Name, "default", "`default`", lex.QuoteBacktick)
			p59Span(t, sql, n.Target.Span(), "`default`")
			if entry.PartitionAll || n.Target.Kind != ast.CacheIndexIdentifierTarget {
				t.Fatal("target/partition kind lost")
			}
		}},
		{"LoadPartition", "LOAD INDEX INTO CACHE t PARTITION (ALL) INDEX (PRIMARY) IGNORE LEAVES", func(t *testing.T, sql string, stmt ast.StmtNode) {
			e := stmt.(*ast.LoadIndexStmt).Tables[0]
			p59Span(t, sql, e.Table.Span(), "t PARTITION (ALL)")
			p59Span(t, sql, e.Span(), "t PARTITION (ALL) INDEX (PRIMARY) IGNORE LEAVES")
			if !e.PartitionAll || !e.IgnoreLeaves {
				t.Fatal("load modifiers lost")
			}
		}},
		{"Tablespace", "ALTER TABLESPACE `旧` RENAME TO ``", func(t *testing.T, sql string, stmt ast.StmtNode) {
			n := stmt.(*ast.AlterTablespaceStmt)
			p59Ident(t, sql, n.Name, "旧", "`旧`", lex.QuoteBacktick)
			p59Ident(t, sql, n.RenameTo, "", "``", lex.QuoteBacktick)
		}},
		{"TablespaceEngine", "CREATE TABLESPACE ts ADD DATAFILE 'path' ENGINE = `x``y` COMMENT = 'text'", func(t *testing.T, sql string, stmt ast.StmtNode) {
			n := stmt.(*ast.CreateTablespaceStmt)
			p59Ident(t, sql, n.Name, "ts", "ts", lex.QuoteUnquoted)
			p59Ident(t, sql, n.Options[1].Engine, "x`y", "`x``y`", lex.QuoteBacktick)
			p59Span(t, sql, n.Options[1].Span(), "ENGINE = `x``y`")
			if n.Options[0].Str != "path" || n.Options[2].Str != "text" || n.Options[1].Str != "" || n.Options[0].Engine.Present() || n.Options[0].Engine.Span.Valid() {
				t.Fatal("option roles lost")
			}
		}},
		{"Logfile", "CREATE LOGFILE GROUP `l.g` ADD UNDOFILE 'path' ENGINE = `InnoDB`", func(t *testing.T, sql string, stmt ast.StmtNode) {
			n := stmt.(*ast.CreateLogfileGroupStmt)
			p59Ident(t, sql, n.Name, "l.g", "`l.g`", lex.QuoteBacktick)
			p59Ident(t, sql, n.Options[0].Engine, "InnoDB", "`InnoDB`", lex.QuoteBacktick)
			p59Span(t, sql, n.Options[0].Span(), "ENGINE = `InnoDB`")
			if n.UndoFile != "path" {
				t.Fatal("undo file lost")
			}
		}},
		{"Server", "CREATE SERVER `` FOREIGN DATA WRAPPER 'mysql' OPTIONS(DATABASE 'd', `DATABASE` 'q')", func(t *testing.T, sql string, stmt ast.StmtNode) {
			n := stmt.(*ast.CreateServerStmt)
			p59Ident(t, sql, n.Name, "", "``", lex.QuoteBacktick)
			p59Ident(t, sql, n.Options[0].Name, "DATABASE", "DATABASE", lex.QuoteUnquoted)
			p59Ident(t, sql, n.Options[1].Name, "DATABASE", "`DATABASE`", lex.QuoteBacktick)
			p59Span(t, sql, n.Options[0].Span(), "DATABASE 'd'")
			if n.Wrapper != "mysql" || n.Options[1].Value != "q" {
				t.Fatal("server strings lost")
			}
		}},
		{"Resource", "CREATE RESOURCE GROUP `名` TYPE = custom VCPU = 3-7 THREAD_PRIORITY = 2 ENABLE", func(t *testing.T, sql string, stmt ast.StmtNode) {
			n := stmt.(*ast.CreateResourceGroupStmt)
			p59Ident(t, sql, n.Name, "名", "`名`", lex.QuoteBacktick)
			p59Ident(t, sql, n.Type, "custom", "custom", lex.QuoteUnquoted)
			p59Span(t, sql, n.VCPU[0].Span(), "3-7")
			if n.ThreadPriority == nil || *n.ThreadPriority != 2 || n.State != ast.ResourceGroupStateEnable {
				t.Fatal("resource options lost")
			}
		}},
		{"Plugin", "INSTALL PLUGIN `p.名` SONAME 'lib.so'", func(t *testing.T, sql string, stmt ast.StmtNode) {
			n := stmt.(*ast.InstallPluginStmt)
			p59Ident(t, sql, n.Name, "p.名", "`p.名`", lex.QuoteBacktick)
			if n.Soname != "lib.so" {
				t.Fatal("SONAME lost")
			}
		}},
		{"ResourceQuotedType", "CREATE RESOURCE GROUP rg TYPE = `USER`", func(t *testing.T, sql string, stmt ast.StmtNode) {
			p59Ident(t, sql, stmt.(*ast.CreateResourceGroupStmt).Type, "USER", "`USER`", lex.QuoteBacktick)
		}},
		{"HandlerPlainAlias", "HANDLER t OPEN h", nil},
		{"HandlerClose", "HANDLER h CLOSE", nil},
		{"CacheAll", "CACHE INDEX t PARTITION (ALL) IN DEFAULT", nil},
		{"LoadNamedPartitions", "LOAD INDEX INTO CACHE t PARTITION (`p.1`) INDEX ()", nil},
		{"AlterTablespaceOptions", "ALTER UNDO TABLESPACE ts WAIT", nil},
		{"DropTablespace", "DROP UNDO TABLESPACE ts ENGINE `innodb`", nil},
		{"AlterLogfile", "ALTER LOGFILE GROUP lg ADD UNDOFILE '' WAIT", nil},
		{"DropLogfile", "DROP LOGFILE GROUP lg ENGINE innodb", nil},
		{"AlterResource", "ALTER RESOURCE GROUP rg VCPU = 0 DISABLE", nil},
		{"DropResource", "DROP RESOURCE GROUP rg FORCE", nil},
		{"AlterServer", "ALTER SERVER s OPTIONS(HOST 'h')", nil},
		{"DropServer", "DROP SERVER IF EXISTS s", nil},
		{"UninstallPlugin", "UNINSTALL PLUGIN `名`", nil},
		{"SelectHint", "SELECT /*+ MAX_EXECUTION_TIME(0) MAX_EXECUTION_TIME(18446744073709551615) */ 1", nil},
		{"UpdateHint", "UPDATE /*+ MAX_EXECUTION_TIME(3) */ t SET a=1", nil},
		{"DeleteHint", "DELETE /*+ MAX_EXECUTION_TIME(4) */ FROM t", nil},
		{"NestedPath", "SELECT * FROM JSON_TABLE(j, '$' COLUMNS (NESTED PATH '$.x' COLUMNS (c INT PATH '$'))) jt", func(t *testing.T, sql string, stmt ast.StmtNode) {
			stmt.Accept(p59BoundaryVisitor(func(n ast.Node) {
				if c, ok := n.(*ast.JSONTableColumn); ok && c.Kind == ast.JSONTableColumnNested {
					if !c.PathKeyword {
						t.Fatal("PATH keyword lost")
					}
					p59Span(t, sql, c.Span(), "NESTED PATH '$.x' COLUMNS (c INT PATH '$')")
				}
			}))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			stmt, err := p.ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			p59Span(t, tc.sql, stmt.Span(), tc.sql)
			if tc.check != nil {
				tc.check(t, tc.sql, stmt)
			}
			if strings.HasSuffix(tc.name, "Hint") {
				count := 0
				stmt.Accept(p59BoundaryVisitor(func(n ast.Node) {
					if h, ok := n.(*ast.OptimizerHint); ok {
						count++
						want := "MAX_EXECUTION_TIME(" + strconv.FormatUint(h.MaxExecutionTime, 10) + ")"
						p59Span(t, tc.sql, h.Span(), want)
					}
				}))
				want := strings.Count(tc.sql, "MAX_EXECUTION_TIME(")
				if count != want {
					t.Fatalf("hint count %d, want %d", count, want)
				}
			}
			before, err := json.Marshal(stmt)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := p.ParseOneStmt("SELECT other FROM elsewhere", "", ""); err != nil {
				t.Fatal(err)
			}
			after, err := json.Marshal(stmt)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("later parse mutated the old AST")
			}
		})
	}
}

func TestManagementGrantNameBoundary(t *testing.T) {
	for _, verb := range []string{"GRANT", "REVOKE"} {
		for _, object := range []string{"*", "*.*", "db.*", "`d.b`.``", "``", "`*`.*"} {
			t.Run(verb+"/"+object, func(t *testing.T) {
				tail := " TO 'MOCK_USER'@'MOCK_HOST'"
				if verb == "REVOKE" {
					tail = " FROM 'MOCK_USER'@'MOCK_HOST'"
				}
				sql := verb + " SELECT ON " + object + tail
				p := New()
				stmt, err := p.ParseOneStmt(sql, "", "")
				if err != nil {
					t.Fatal(err)
				}
				var obj *ast.GrantObject
				switch n := stmt.(type) {
				case *ast.GrantStmt:
					obj = n.Object
				case *ast.RevokeStmt:
					obj = n.Object
				}
				p59Span(t, sql, stmt.Span(), sql)
				p59Span(t, sql, obj.Span(), object)
				wantParts := 2
				if object == "*" || object == "``" {
					wantParts = 1
				}
				if len(obj.Parts) != wantParts {
					t.Fatalf("parts %d, want %d", len(obj.Parts), wantParts)
				}
				switch object {
				case "`d.b`.``":
					p59Ident(t, sql, obj.Parts[0].Name, "d.b", "`d.b`", lex.QuoteBacktick)
					p59Ident(t, sql, obj.Parts[1].Name, "", "``", lex.QuoteBacktick)
				case "``":
					p59Ident(t, sql, obj.Parts[0].Name, "", "``", lex.QuoteBacktick)
				case "`*`.*":
					p59Ident(t, sql, obj.Parts[0].Name, "*", "`*`", lex.QuoteBacktick)
				}
				for _, part := range obj.Parts {
					if part.Wildcard {
						p59Span(t, sql, part.WildcardSpan, "*")
						if part.Name.Present() || part.Name.Span.Valid() {
							t.Fatal("wildcard has a name")
						}
					} else {
						p59Span(t, sql, part.Name.Span, part.Name.Raw)
						if !part.Name.Present() || part.WildcardSpan.Valid() {
							t.Fatal("named part has wildcard metadata")
						}
					}
				}
				if object == "*.*" {
					if obj.Parts[0].WildcardSpan.Start >= obj.Parts[1].WildcardSpan.Start {
						t.Fatal("star token spans overlap")
					}
				}
				before, err := json.Marshal(obj)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := p.ParseOneStmt("SELECT 1", "", ""); err != nil {
					t.Fatal(err)
				}
				after, err := json.Marshal(obj)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(before, after) {
					t.Fatal("parser reuse mutated the grant object")
				}
			})
		}
	}
}

func TestManagementANSINameBoundary(t *testing.T) {
	p := New()
	p.SetSQLMode(mysql.ModeANSIQuotes)
	for _, sql := range []string{`ALTER TABLESPACE "旧" RENAME TO ""`, `INSTALL PLUGIN "p.名" SONAME 'lib.so'`, `SELECT * FROM JSON_TABLE(j, '$' COLUMNS ("" INT PATH '$')) jt`} {
		stmt, err := p.ParseOneStmt(sql, "", "")
		if err != nil {
			t.Fatal(err)
		}
		var name lex.Ident
		switch n := stmt.(type) {
		case *ast.AlterTablespaceStmt:
			name = n.RenameTo
		case *ast.InstallPluginStmt:
			name = n.Name
		default:
			stmt.Accept(p59BoundaryVisitor(func(n ast.Node) {
				if c, ok := n.(*ast.JSONTableColumn); ok {
					name = c.Name
				}
			}))
		}
		if name.Quote != lex.QuoteDouble || !name.Present() {
			t.Fatalf("ANSI name %+v", name)
		}
		p59Span(t, sql, name.Span, name.Raw)
	}
}

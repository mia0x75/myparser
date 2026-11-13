package myparser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/model"
)

// https://mariadb.com/docs/server/reference/sql-structure/operators/comparison-operators/is-json
func TestISJSONModifiers(t *testing.T) {
	types := []string{"", "VALUE", "ARRAY", "OBJECT", "SCALAR"}
	keys := []struct{ src, restored string }{
		{"", ""},
		{"WITH UNIQUE", "WITH UNIQUE KEYS"},
		{"WITH UNIQUE KEYS", "WITH UNIQUE KEYS"},
		{"WITHOUT UNIQUE", "WITHOUT UNIQUE KEYS"},
		{"WITHOUT UNIQUE KEYS", "WITHOUT UNIQUE KEYS"},
	}
	for _, not := range []string{"IS JSON", "IS NOT JSON"} {
		for _, tp := range types {
			for _, key := range keys {
				src := strings.TrimSpace(strings.Join([]string{"select j", not, tp, key.src}, " "))
				expected := strings.TrimSpace(strings.Join([]string{"SELECT `j`", not, tp, key.restored}, " "))
				src = strings.Join(strings.Fields(src), " ")
				expected = strings.Join(strings.Fields(expected), " ")
				t.Run(src, func(t *testing.T) {
					RunRestoreTest(t, src, expected, false)
					expr := mustParseOne(t, src).(*ast.SelectStmt).Fields.Fields[0].Expr.(*ast.IsJSONExpr)
					if expr.Type != tp || expr.UniqueKeys != key.restored || expr.Not != (not == "IS NOT JSON") {
						t.Fatalf("unexpected IS JSON constraints: %#v", expr)
					}
					var out strings.Builder
					expr.Format(&out)
					if out.String() != strings.TrimPrefix(expected, "SELECT ") {
						t.Fatalf("unexpected IS JSON Format: %q", out.String())
					}
				})
			}
		}
	}
	RunRestoreTest(t, "select array, object, scalar from t", "SELECT `array`,`object`,`scalar` FROM `t`", false)
	RunRestoreTest(t, "select j is json object as object from t where j is not json array without unique and a=1",
		"SELECT `j` IS JSON OBJECT AS `object` FROM `t` WHERE `j` IS NOT JSON ARRAY WITHOUT UNIQUE KEYS AND `a`=1", false)
	RunRestoreTest(t, "select j is json with unique keys object", "SELECT `j` IS JSON WITH UNIQUE KEYS AS `object`", false)
	for _, src := range []string{
		"select j is json with keys",
		"select j is json without keys",
		"select j is json with unique keys without unique keys",
		"select j is json with unique keys object array",
	} {
		t.Run(src, func(t *testing.T) {
			if _, err := New().ParseOneStmt(src, "", ""); err == nil {
				t.Fatalf("expected parse error for %q", src)
			}
		})
	}
}

// https://dev.mysql.com/doc/refman/8.0/en/handler.html
func TestHandlerKeywordIndexes(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"handler db.t open a", "HANDLER `db`.`t` OPEN AS `a`"},
		{"handler t read first", "HANDLER `t` READ FIRST"},
		{"handler t read first next", "HANDLER `t` READ `first` NEXT"},
		{"handler t read next first where a=1 limit 2", "HANDLER `t` READ `next` FIRST WHERE `a`=1 LIMIT 2"},
		{"handler t read last prev", "HANDLER `t` READ `last` PREV"},
		{"handler t read cache >= (1, 2) where a>0 limit 2 offset 1", "HANDLER `t` READ `cache` >= (1, 2) WHERE `a`>0 LIMIT 1,2"},
		{"handler t read `PRIMARY` = (1)", "HANDLER `t` READ `PRIMARY` = (1)"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			RunRestoreTest(t, tc.src, tc.restored, false)
		})
	}
	for _, src := range []string{"handler t read prev", "handler db.t read first", "handler db.t close", "handler t read primary first"} {
		t.Run(src, func(t *testing.T) {
			if _, err := New().ParseOneStmt(src, "", ""); err == nil {
				t.Fatalf("expected parse error for %q", src)
			}
		})
	}
}

// https://dev.mysql.com/doc/refman/8.0/en/get-diagnostics.html
// https://mariadb.com/docs/server/reference/sql-statements/programmatic-compound-statements/programmatic-compound-statements-diagnostics/get-diagnostics
func TestDiagnosticsConditionNumbers(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"get diagnostics condition @n @r = row_number", "GET CURRENT DIAGNOSTICS CONDITION @`n` @`r` = ROW_NUMBER"},
		{"get diagnostics condition n r = mysql_errno", "GET CURRENT DIAGNOSTICS CONDITION `n` `r` = MYSQL_ERRNO"},
		{"get diagnostics condition @@session.max_error_count @r = message_text", "GET CURRENT DIAGNOSTICS CONDITION @@SESSION.`max_error_count` @`r` = MESSAGE_TEXT"},
		{"get diagnostics condition '1' @r = returned_sqlstate", "GET CURRENT DIAGNOSTICS CONDITION '1' @`r` = RETURNED_SQLSTATE"},
	}
	for _, window := range []bool{false, true} {
		for _, tc := range cases {
			t.Run(tc.src+map[bool]string{false: "/plain", true: "/window"}[window], func(t *testing.T) {
				RunRestoreTest(t, tc.src, tc.restored, window)
			})
		}
	}
	for _, src := range []string{
		"get diagnostics condition 1+1 @r = mysql_errno",
		"get diagnostics condition (1) @r = mysql_errno",
		"get diagnostics condition @n := 1 @r = mysql_errno",
		"get diagnostics @r = row_number",
	} {
		t.Run(src, func(t *testing.T) {
			if _, err := New().ParseOneStmt(src, "", ""); err == nil {
				t.Fatalf("expected parse error for %q", src)
			}
		})
	}
}

// https://dev.mysql.com/doc/refman/8.0/en/cache-index.html
// https://dev.mysql.com/doc/refman/8.0/en/load-index.html
func TestIndexCacheModifiers(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"cache index db.t1 key (primary, Cache), t2 index (), t3 in c", "CACHE INDEX `db`.`t1` INDEX (`PRIMARY`, `Cache`), `t2` INDEX (), `t3` IN `c`"},
		{"cache index t partition (p0, p1) key (a, b) in default", "CACHE INDEX `t` PARTITION(`p0`, `p1`) INDEX (`a`, `b`) IN DEFAULT"},
		{"cache index db.t partition (all) in c", "CACHE INDEX `db`.`t` PARTITION (ALL) IN `c`"},
		{"cache index t partition (all) index () in default", "CACHE INDEX `t` PARTITION (ALL) INDEX () IN DEFAULT"},
		{"load index into cache t key () ignore leaves", "LOAD INDEX INTO CACHE `t` INDEX () IGNORE LEAVES"},
		{"load index into cache t key (primary, a), t2 index () ignore leaves, t3", "LOAD INDEX INTO CACHE `t` INDEX (`PRIMARY`, `a`), `t2` INDEX () IGNORE LEAVES, `t3`"},
		{"load index into cache db.t partition (p0, p1) key (a) ignore leaves", "LOAD INDEX INTO CACHE `db`.`t` PARTITION(`p0`, `p1`) INDEX (`a`) IGNORE LEAVES"},
		{"load index into cache t partition (all) index () ignore leaves", "LOAD INDEX INTO CACHE `t` PARTITION (ALL) INDEX () IGNORE LEAVES"},
		{"load index into cache t partition (all)", "LOAD INDEX INTO CACHE `t` PARTITION (ALL)"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			RunRestoreTest(t, tc.src, tc.restored, false)
		})
	}
	cache := mustParseOne(t, "cache index t, t2 index (), t3 key (a) in c").(*ast.CacheIndexStmt)
	if len(cache.Tables) != 3 || len(cache.TableIndexes) != 3 || cache.TableIndexes[0] != nil || cache.TableIndexes[1] == nil || len(cache.TableIndexes[1]) != 0 || len(cache.TableIndexes[2]) != 1 {
		t.Fatalf("omitted, empty, and named index lists were not preserved: %#v", cache)
	}
	load := mustParseOne(t, "load index into cache t, t2 index (), t3 key (a)").(*ast.LoadIndexStmt)
	if len(load.Tables) != 3 || load.Tables[0].IndexNames != nil || load.Tables[1].IndexNames == nil || len(load.Tables[1].IndexNames) != 0 || len(load.Tables[2].IndexNames) != 1 {
		t.Fatalf("omitted, empty, and named preload lists were not preserved: %#v", load)
	}
	assertExtensionParseErrors(t, []string{
		"cache index t partition () in c",
		"cache index t partition (all, p0) in c",
		"cache index t partition (p0), t2 in c",
		"cache index t, t2 partition (all) in c",
		"cache index t index (a) partition (p0) in c",
		"cache index t key (a,) in c",
		"cache index t ignore leaves in c",
		"load index into cache t partition ()",
		"load index into cache t partition (p0), t2",
		"load index into cache t, t2 partition (p0)",
		"load index into cache t ignore leaves key (a)",
		"load index into cache t key (a,) ignore leaves",
	})
}

// https://dev.mysql.com/doc/refman/8.0/en/reset.html
// https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/reset
func TestResetOptionLists(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"reset master, slave", "RESET MASTER, SLAVE"},
		{"reset slave all for channel 'c', master to 42", "RESET SLAVE ALL FOR CHANNEL 'c', MASTER TO 42"},
		{"reset binary logs and gtids to 123, replica all for channel 'c'", "RESET BINARY LOGS AND GTIDS TO 123, REPLICA ALL FOR CHANNEL 'c'"},
		{"reset query cache, master to 7, slave all", "RESET QUERY CACHE, MASTER TO 7, SLAVE ALL"},
		{"reset slave for channel ''", "RESET SLAVE FOR CHANNEL ''"},
		{"reset replica all for channel '', binary logs and gtids", "RESET REPLICA ALL FOR CHANNEL '', BINARY LOGS AND GTIDS"},
		{"reset persist max_connections", "RESET PERSIST max_connections"},
		{"reset persist default.max_connections", "RESET PERSIST default.max_connections"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			RunRestoreTest(t, tc.src, tc.restored, false)
		})
	}
	single := mustParseOne(t, "reset master to 42").(*ast.ResetStmt)
	if single.Tp != ast.ResetMaster || single.Options != nil || len(single.Args) != 1 {
		t.Fatalf("single RESET option changed shape: %#v", single)
	}
	multiple := mustParseOne(t, "reset query cache, master to 42, slave all").(*ast.ResetStmt)
	if len(multiple.Options) != 3 || multiple.Options[0].Tp != ast.ResetQueryCache || multiple.Options[1].Tp != ast.ResetMaster || !multiple.Options[2].All {
		t.Fatalf("unexpected multi-option RESET: %#v", multiple)
	}
	single = mustParseOne(t, "reset replica for channel ''").(*ast.ResetStmt)
	if !single.HasChannel || single.Channel != "" || single.Options != nil {
		t.Fatalf("empty RESET channel was lost: %#v", single)
	}
	assertExtensionParseErrors(t, []string{
		"reset", "reset master,", "reset master slave", "reset , master",
		"reset persist, master", "reset master, persist", "reset persist if exists x, slave",
		"reset persist x, slave", "reset persist if exists", "reset persist @x",
		"reset master to '42'", "reset replica to 42", "reset master all",
	})
}

// https://dev.mysql.com/doc/refman/8.0/en/load-xml.html
func TestLoadXMLModifiers(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"load xml low_priority local infile 'f' ignore into table db.t partition (p0, p1) character set utf8mb4 rows identified by '<r>' ignore 0 rows (a, @b) set c=@b, d=default", "LOAD XML LOW_PRIORITY LOCAL INFILE 'f' IGNORE INTO TABLE `db`.`t` PARTITION(`p0`, `p1`) CHARACTER SET `utf8mb4` ROWS IDENTIFIED BY '<r>' IGNORE 0 ROWS (`a`, @`b`) SET `c`=@`b`, `d`=DEFAULT"},
		{"load xml concurrent infile 'f' replace into table t ignore 2 lines", "LOAD XML CONCURRENT INFILE 'f' REPLACE INTO TABLE `t` IGNORE 2 LINES"},
		{"load xml local infile 'f' into table t ignore 0 lines", "LOAD XML LOCAL INFILE 'f' INTO TABLE `t` IGNORE 0 LINES"},
		{"load xml infile 'f' into table t rows identified by '' ignore 18446744073709551615 rows", "LOAD XML INFILE 'f' INTO TABLE `t` ROWS IDENTIFIED BY '' IGNORE 18446744073709551615 ROWS"},
		{"load xml infile 'f' into table t () set a=1", "LOAD XML INFILE 'f' INTO TABLE `t` SET `a`=1"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			RunRestoreTest(t, tc.src, tc.restored, false)
		})
	}
	for _, priority := range []string{"", "LOW_PRIORITY ", "CONCURRENT "} {
		for _, local := range []string{"", "LOCAL "} {
			for _, ignore := range []string{"", " IGNORE 0 LINES", " IGNORE 2 ROWS"} {
				src := "LOAD XML " + priority + local + "INFILE 'f' INTO TABLE t" + ignore
				t.Run(src, func(t *testing.T) {
					RunRestoreTest(t, src, strings.Replace(src, "TABLE t", "TABLE `t`", 1), false)
				})
			}
		}
	}
	xml := mustParseOne(t, cases[0].src).(*ast.LoadXMLStmt)
	if xml.Priority != "LOW_PRIORITY" || !xml.IsLocal || !xml.HasIgnoreLines || xml.IgnoreLines != 0 || !xml.IgnoreRows || len(xml.Table.PartitionNames) != 2 || len(xml.ColumnsAndUserVars) != 2 || len(xml.ColumnAssignments) != 2 {
		t.Fatalf("unexpected LOAD XML modifiers: %#v", xml)
	}
	RunRestoreTest(t, "select concurrent from t", "SELECT `concurrent` FROM `t`", false)
	assertExtensionParseErrors(t, []string{
		"load xml high_priority infile 'f' into table t",
		"load xml low_priority concurrent infile 'f' into table t",
		"load xml local low_priority infile 'f' into table t",
		"load xml infile 'f' into table t ignore -1 rows",
		"load xml infile 'f' into table t ignore 1.5 lines",
		"load xml infile 'f' into table t ignore @n rows",
		"load xml infile 'f' into table t ignore 1 rows rows identified by '<r>'",
		"load xml infile 'f' into table t partition (all)",
	})
}

// https://dev.mysql.com/doc/refman/8.0/en/json-table-functions.html
// https://mariadb.com/docs/server/reference/sql-functions/special-functions/json-functions/json_table
func TestJSONTableBehaviorClauses(t *testing.T) {
	behaviors := []string{"", "NULL", "ERROR", "DEFAULT '0'", "DEFAULT -1", "DEFAULT +2.5", "DEFAULT TRUE", "DEFAULT NULL", "DEFAULT x'41'"}
	for _, exists := range []string{"", "EXISTS "} {
		for _, empty := range behaviors {
			for _, errorBehavior := range behaviors {
				clauses := ""
				if empty != "" {
					clauses += " " + empty + " ON EMPTY"
				}
				if errorBehavior != "" {
					clauses += " " + errorBehavior + " ON ERROR"
				}
				src := "select * from json_table(j, '$' columns (a int " + exists + "path '$.a'" + clauses + ")) jt"
				expected := "SELECT * FROM JSON_TABLE(`j`, '$' COLUMNS (`a` INT " + exists + "PATH '$.a'" + clauses + ")) AS `jt`"
				t.Run(src, func(t *testing.T) {
					RunRestoreTest(t, src, expected, false)
				})
			}
		}
	}
	cases := []struct{ src, restored string }{
		{"select * from json_table(j, '$' columns (a int path '$.a' error on error default -1 on empty)) jt", "SELECT * FROM JSON_TABLE(`j`, '$' COLUMNS (`a` INT PATH '$.a' DEFAULT -1 ON EMPTY ERROR ON ERROR)) AS `jt`"},
		{"select * from json_table(j, '$' columns (id for ordinality, nested '$.a[*]' columns (v int path '$' default 0 on error null on empty))) jt", "SELECT * FROM JSON_TABLE(`j`, '$' COLUMNS (`id` FOR ORDINALITY, NESTED PATH '$.a[*]' COLUMNS (`v` INT PATH '$' NULL ON EMPTY DEFAULT 0 ON ERROR))) AS `jt`"},
		{"select * from json_table(j, '$' columns (a varchar(10) path '' default '\"missing\"' on empty)) jt", "SELECT * FROM JSON_TABLE(`j`, '$' COLUMNS (`a` VARCHAR(10) PATH '' DEFAULT '\"missing\"' ON EMPTY)) AS `jt`"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			RunRestoreTest(t, tc.src, tc.restored, false)
		})
	}
	for _, col := range []string{
		"for ordinality int", "id int for ordinality", "a int", "a int '$.a'",
		"a int path '$' on empty null", "a int path '$' default @v on empty",
		"a int path '$' default (1) on empty", "a int path '$' default 1+1 on empty",
		"a int path '$' null on empty error on empty", "a int path '$' null on error error on error",
		"id for ordinality error on empty", "nested '$' columns ()", "",
	} {
		assertExtensionParseErrors(t, []string{"select * from json_table(j, '$' columns (" + col + ")) jt"})
	}
	assertExtensionParseErrors(t, []string{
		"select * from json_table(j, '$' columns (a int path '$'))",
		"select * from lateral json_table(j, '$' columns (a int path '$')) jt",
	})
}

type extensionVisitor struct {
	enter func(ast.Node) (ast.Node, bool)
	leave func(ast.Node) (ast.Node, bool)
}

func TestFullTextAndSoundsLikeFormatting(t *testing.T) {
	cases := []struct{ src, formatted string }{
		{"select match(db.t.a, t.b) against('Search')", `MATCH (db.t.a, t.b) AGAINST ("Search")`},
		{"select match(a) against('Search' in boolean mode)", `MATCH (a) AGAINST ("Search" IN BOOLEAN MODE)`},
		{"select match(a) against('Search' in natural language mode with query expansion)", `MATCH (a) AGAINST ("Search" WITH QUERY EXPANSION)`},
		{"select 'a' sounds like 'b'", `"a" SOUNDS LIKE "b"`},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			expr := mustParseOne(t, tc.src).(*ast.SelectStmt).Fields.Fields[0].Expr
			var out strings.Builder
			expr.Format(&out)
			if out.String() != tc.formatted {
				t.Fatalf("Format: got %q, want %q", out.String(), tc.formatted)
			}
		})
	}
}

func (v extensionVisitor) Enter(n ast.Node) (ast.Node, bool) {
	if v.enter != nil {
		return v.enter(n)
	}
	return n, false
}

func (v extensionVisitor) Leave(n ast.Node) (ast.Node, bool) {
	if v.leave != nil {
		return v.leave(n)
	}
	return n, true
}

func TestCompatibilityExtensionVisitors(t *testing.T) {
	cases := []struct {
		src  string
		want map[string]int
	}{
		{"reset master to 1, slave all", map[string]int{"*ast.ResetStmt": 3, "*driver.ValueExpr": 1}},
		{"cache index t key (a), t2 index () in c", map[string]int{"*ast.CacheIndexStmt": 1, "*ast.TableName": 2}},
		{"load index into cache t key (a), t2 index ()", map[string]int{"*ast.LoadIndexStmt": 1, "*ast.LoadIndexTable": 2, "*ast.TableName": 2}},
		{"load xml infile 'f' into table t (a, @b) set c=@b", map[string]int{"*ast.LoadXMLStmt": 1, "*ast.LoadDataStmt": 0, "*ast.TableName": 1, "*ast.ColumnName": 2, "*ast.VariableExpr": 2, "*ast.Assignment": 1}},
		{"select * from json_table(j, '$' columns (id for ordinality, nested '$.a' columns (a int path '$' default 1 on empty default 2 on error))) jt", map[string]int{"*ast.JSONTable": 1, "*ast.JSONTableColumn": 3, "*driver.ValueExpr": 2, "*ast.ColumnNameExpr": 1}},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			stmt := mustParseOne(t, tc.src)
			counts := map[string]int{}
			_, ok := stmt.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
				counts[fmt.Sprintf("%T", n)]++
				return n, false
			}})
			if !ok {
				t.Fatal("visitor unexpectedly aborted")
			}
			for tp, want := range tc.want {
				if counts[tp] != want {
					t.Errorf("visited %s %d times, want %d", tp, counts[tp], want)
				}
			}
			entered, left := 0, 0
			_, ok = stmt.Accept(extensionVisitor{
				enter: func(n ast.Node) (ast.Node, bool) { entered++; return n, true },
				leave: func(n ast.Node) (ast.Node, bool) { left++; return n, true },
			})
			if !ok || entered != 1 || left != 1 {
				t.Fatalf("skip-children visitor: ok=%v, enter=%d, leave=%d", ok, entered, left)
			}
		})
	}
	stmt := mustParseOne(t, cases[4].src)
	_, ok := stmt.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch n := n.(type) {
		case *ast.JSONTableColumn:
			if n.Name.L == "a" {
				copy := *n
				copy.Name = model.NewCIStr("b")
				return &copy, false
			}
		case ast.ValueExpr:
			return ast.NewValueExpr(int64(9)), false
		}
		return n, false
	}})
	if !ok {
		t.Fatal("replacement visitor aborted")
	}
	if got := restoreStmt(t, stmt); got != "SELECT * FROM JSON_TABLE(`j`, '$' COLUMNS (`id` FOR ORDINALITY, NESTED PATH '$.a' COLUMNS (`b` INT PATH '$' DEFAULT 9 ON EMPTY DEFAULT 9 ON ERROR))) AS `jt`" {
		t.Fatalf("replacement visitor did not reach nested defaults: %s", got)
	}
	_, ok = stmt.Accept(extensionVisitor{leave: func(n ast.Node) (ast.Node, bool) {
		_, value := n.(ast.ValueExpr)
		return n, !value
	}})
	if ok {
		t.Fatal("visitor abort was not propagated")
	}
}

func assertExtensionParseErrors(t *testing.T, sqls []string) {
	t.Helper()
	for _, src := range sqls {
		t.Run(src, func(t *testing.T) {
			if _, err := New().ParseOneStmt(src, "", ""); err == nil {
				t.Fatalf("expected parse error for %q", src)
			}
		})
	}
}

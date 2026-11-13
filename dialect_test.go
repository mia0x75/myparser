package myparser

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/mysql"
)

func TestDialectCompatibilityLastValue(t *testing.T) {
	tests := []struct {
		name          string
		windowFunc    bool
		sql           string
		mysqlIssues   []DialectIssueCode
		mariadbIssues []DialectIssueCode
	}{
		{"scalar", false, "SELECT LAST_VALUE(NULL,e)", []DialectIssueCode{IssueMySQLScalarLastValue}, nil},
		{"scalar with window keywords", true, "SELECT LAST_VALUE(NULL,e)", []DialectIssueCode{IssueMySQLScalarLastValue}, nil},
		{"scalar without arguments", false, "SELECT LAST_VALUE()", []DialectIssueCode{IssueMySQLScalarLastValue}, []DialectIssueCode{IssueMariaDBLastValueArgumentCount}},
		{"window", true, "SELECT LAST_VALUE(1) OVER ()", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := New()
			parser.EnableWindowFunc(tt.windowFunc)
			stmt, err := parser.ParseOneStmt(tt.sql, "", "")
			if err != nil {
				t.Fatalf("union parse rejected: %v", err)
			}
			for target, want := range map[Dialect][]DialectIssueCode{
				DialectMySQL:   tt.mysqlIssues,
				DialectMariaDB: tt.mariadbIssues,
			} {
				issues := parser.CheckDialectCompatibility(stmt, tt.sql, target)
				if len(issues) != len(want) {
					t.Fatalf("%s issues = %#v, want %#v", target, issues, want)
				}
				for i, issue := range issues {
					if issue.Code != want[i] || issue.Target != target {
						t.Fatalf("%s issue = %#v, want %#v", target, issue, want[i])
					}
					if tt.sql[issue.Offset:issue.Offset+issue.Length] != issue.Highlight {
						t.Fatalf("%s issue range = %#v", target, issue)
					}
				}
			}
			if issues := parser.CheckDialectCompatibility(stmt, tt.sql, DialectUnion); len(issues) != 0 {
				t.Fatalf("union dialect issues = %#v", issues)
			}
		})
	}
}

func TestDialectExecutableComments(t *testing.T) {
	for _, tt := range []struct {
		name    string
		sql     string
		mysql   int
		mariadb int
	}{
		{name: "shared old version", sql: `SELECT 1 /*!40101 , 2 */`, mysql: 2, mariadb: 2},
		{name: "MySQL 5.7 version", sql: `SELECT 1 /*!50700 , 2 */`, mysql: 2, mariadb: 1},
		{name: "MySQL 8.4 version", sql: `SELECT 1 /*!80400 , 2 */`, mysql: 2, mariadb: 1},
		{name: "MariaDB six digit version", sql: `SELECT 1 /*!100000 , 2 */`, mysql: 1, mariadb: 2},
		{name: "MariaDB marker", sql: `SELECT 1 /*M!100000 , 2 */`, mysql: 1, mariadb: 2},
		{name: "future version", sql: `SELECT 1 /*!99998 , 2 */`, mysql: 1, mariadb: 1},
		{name: "future MariaDB version", sql: `SELECT 1 /*M!999999 , 2 */`, mysql: 1, mariadb: 1},
		{name: "skipped nested code", sql: `SELECT 1 /*!99998 , /*!01000 2, */ 3 */`, mysql: 1, mariadb: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			parser := New()
			for target, want := range map[Dialect]int{DialectMySQL: tt.mysql, DialectMariaDB: tt.mariadb} {
				stmt, err := parser.ParseOneStmtForDialect(tt.sql, "", "", target)
				if err != nil {
					t.Fatalf("%s: %v", target, err)
				}
				if got := len(stmt.(*ast.SelectStmt).Fields.Fields); got != want {
					t.Fatalf("%s fields = %d, want %d", target, got, want)
				}
			}
			stmt, err := parser.ParseOneStmt(tt.sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			want := tt.mysql
			if tt.mariadb > want {
				want = tt.mariadb
			}
			if got := len(stmt.(*ast.SelectStmt).Fields.Fields); got != want {
				t.Fatalf("union fields = %d, want %d (target state leaked)", got, want)
			}
		})
	}
	for _, tt := range []struct {
		name   string
		sql    string
		target Dialect
	}{
		{name: "skip MySQL-only invalid body", sql: `SELECT 1 /*!50700 + */`, target: DialectMariaDB},
		{name: "skip MariaDB-only invalid body", sql: `SELECT 1 /*M!100000 + */`, target: DialectMySQL},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := New().ParseOneStmtForDialect(tt.sql, "", "", tt.target); err != nil {
				t.Fatal(err)
			}
			if _, err := New().ParseOneStmt(tt.sql, "", ""); err == nil {
				t.Fatal("union silently skipped an executed invalid body")
			}
		})
	}
}

func TestDialectCompatibilitySelectAggregates(t *testing.T) {
	for _, tt := range []struct {
		name      string
		sql       string
		mysqlCode DialectIssueCode
		mariaCode DialectIssueCode
	}{
		{name: "group concat", sql: "SELECT GROUP_CONCAT(a,b ORDER BY a DESC SEPARATOR '|') FROM t"},
		{name: "group concat limit", sql: "SELECT GROUP_CONCAT(a ORDER BY b LIMIT 2 OFFSET 1) FROM t", mysqlCode: IssueMySQLGroupConcatLimit},
		{name: "json array aggregate", sql: "SELECT JSON_ARRAYAGG(a) FROM t"},
		{name: "json distinct", sql: "SELECT JSON_ARRAYAGG(DISTINCT a) FROM t", mysqlCode: IssueMySQLJSONArrayAggClauses},
		{name: "json order and limit", sql: "SELECT JSON_ARRAYAGG(a ORDER BY b DESC LIMIT 2) FROM t", mysqlCode: IssueMySQLJSONArrayAggClauses},
		{name: "json all", sql: "SELECT JSON_ARRAYAGG(ALL 1)", mariaCode: IssueMariaDBJSONArrayAggAll},
		{name: "json window", sql: "SELECT JSON_ARRAYAGG(a) OVER () FROM t", mariaCode: IssueMariaDBJSONArrayAggWindow},
	} {
		t.Run(tt.name, func(t *testing.T) {
			parser := New()
			parser.EnableWindowFunc(true)
			stmt, err := parser.ParseOneStmt(tt.sql, "", "")
			if err != nil {
				t.Fatalf("union parse: %v", err)
			}
			original := restoreStmt(t, stmt)
			for target, code := range map[Dialect]DialectIssueCode{DialectMySQL: tt.mysqlCode, DialectMariaDB: tt.mariaCode} {
				issues := parser.CheckDialectCompatibility(stmt, tt.sql, target)
				if code == "" && len(issues) != 0 {
					t.Fatalf("%s control issues = %#v", target, issues)
				}
				if code != "" && (len(issues) != 1 || issues[0].Code != code) {
					t.Fatalf("%s issues = %#v, want %s", target, issues, code)
				}
				if len(issues) != 0 && tt.sql[issues[0].Offset:issues[0].Offset+issues[0].Length] != issues[0].Highlight {
					t.Fatalf("invalid source range: %#v", issues[0])
				}
				_, err := parser.ParseOneStmtForDialect(tt.sql, "", "", target)
				if (err != nil) != (code != "") {
					t.Fatalf("%s strict parse = %v", target, err)
				}
			}
			if got := restoreStmt(t, stmt); got != original {
				t.Fatalf("visitor mutated AST: %q -> %q", original, got)
			}
		})
	}
	for _, sql := range []string{
		"SELECT JSON_ARRAYAGG()", "SELECT JSON_ARRAYAGG(a,b)", "SELECT JSON_ARRAYAGG(a SEPARATOR '|')",
		"SELECT JSON_ARRAYAGG(DISTINCT a) OVER ()", "SELECT JSON_ARRAYAGG(a ORDER BY a) OVER ()",
		"SELECT JSON_ARRAYAGG(a LIMIT 1) OVER ()",
	} {
		t.Run(sql, func(t *testing.T) {
			parser := New()
			parser.EnableWindowFunc(true)
			if _, err := parser.ParseOneStmt(sql, "", ""); err == nil {
				t.Fatal("accepted invalid aggregate call")
			}
		})
	}
}

func TestDialectCompatibilityVerifiedBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		target    Dialect
		code      DialectIssueCode
		highlight string
		sql       string
	}{
		{"numeric function", DialectMariaDB, IssueMariaDBNumericIdentifier, "1.e", "SELECT 1.e(1)"},
		{"numeric alias", DialectMariaDB, IssueMariaDBNumericIdentifier, "1.e", "SELECT 1 1.e, 1"},
		{"CAST AS JSON", DialectMariaDB, IssueMariaDBCastJSON, "json", `select cast('{"a":1}' as json)`},
		{"CAST AS VARCHAR", DialectMySQL, IssueMySQLCastVarchar, "varchar", "select cast('x' as varchar(10))"},
		{"CONVERT to VARCHAR", DialectMySQL, IssueMySQLCastVarchar, "varchar", "select convert('x', varchar(10))"},
		{"VALUES without ROW", DialectMySQL, IssueMySQLValuesRowSyntax, "(", "values (1) union select 2"},
		{"VALUES with ROW", DialectMariaDB, IssueMariaDBValuesRowSyntax, "row", "values row(1) union select 2"},
		{"SIGNED ZEROFILL", DialectMariaDB, IssueMariaDBTypeModifier, "ZEROFILL", "CREATE TABLE t1 (a INT SIGNED ZEROFILL)"},
		{"SIGNED UNSIGNED", DialectMariaDB, IssueMariaDBTypeModifier, "UNSIGNED", "CREATE TABLE t1 (a INT SIGNED UNSIGNED)"},
		{"duplicate ZEROFILL", DialectMariaDB, IssueMariaDBTypeModifier, "ZEROFILL", "CREATE TABLE t1 (a INT ZEROFILL UNSIGNED ZEROFILL)"},
		{"leading dot SET value", DialectMySQL, IssueMySQLLeadingDotSetValue, ".", "SET default_storage_engine = .a.MyISAM"},
		// The five bare-set-scope forms ("SET GLOBAL = DEFAULT" et al.) are now
		// rejected by the shared grammar itself; see TestSetBareScopeRejected.
		{"quoted ODBC marker", DialectMySQL, IssueMySQLODBCEscapeMarker, "`OJ`", "SELECT * FROM { `OJ` t1 LEFT JOIN t2 ON TRUE }"},
		{"arbitrary ODBC marker", DialectMySQL, IssueMySQLODBCEscapeMarker, "random_identifier", "SELECT * FROM { random_identifier t1 LEFT JOIN t2 ON TRUE }"},
		{"duplicate same COLLATE", DialectMySQL, IssueMySQLDuplicateCollate, "COLLATE", "CREATE TABLE t1 (\n  x VARCHAR(10)\n  COLLATE ascii_bin\n  COLLATE ascii_bin\n)"},
		{"duplicate conflicting COLLATE", DialectMySQL, IssueMySQLDuplicateCollate, "COLLATE", "CREATE TABLE t2 (\n  x VARCHAR(10)\n  COLLATE ascii_bin\n  COLLATE ascii_general_ci\n)"},
		{"separated duplicate COLLATE", DialectMySQL, IssueMySQLDuplicateCollate, "COLLATE", "CREATE TABLE t3 (\n  x VARCHAR(10)\n  COLLATE ascii_bin\n  NOT NULL\n  COLLATE ascii_bin\n)"},
		{"INSERT row alias", DialectMariaDB, IssueMariaDBInsertRowAlias, "AS", "INSERT INTO t1 VALUES (1) AS new ON DUPLICATE KEY UPDATE b = new.b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := New()
			stmt, err := parser.ParseOneStmt(tt.sql, "", "")
			if err != nil {
				t.Fatalf("union parse rejected verified boundary: %v", err)
			}
			issues := parser.CheckDialectCompatibility(stmt, tt.sql, tt.target)
			if len(issues) != 1 {
				t.Fatalf("issues = %#v, want one", issues)
			}
			issue := issues[0]
			if issue.Code != tt.code || issue.Target != tt.target || issue.Highlight != tt.highlight {
				t.Fatalf("issue = %#v", issue)
			}
			if issue.Offset < 0 || issue.Length != len(tt.highlight) || tt.sql[issue.Offset:issue.Offset+issue.Length] != tt.highlight {
				t.Fatalf("invalid source range: %#v", issue)
			}
			line, column := dialectTestLineColumn(tt.sql, issue.Offset)
			if issue.Line != line || issue.Column != column {
				t.Fatalf("position = %d:%d, want %d:%d", issue.Line, issue.Column, line, column)
			}

			other := DialectMariaDB
			if tt.target == DialectMariaDB {
				other = DialectMySQL
			}
			if got := parser.CheckDialectCompatibility(stmt, tt.sql, other); len(got) != 0 {
				t.Fatalf("counterpart dialect issues = %#v", got)
			}
			if got := parser.CheckDialectCompatibility(stmt, tt.sql, DialectUnion); len(got) != 0 {
				t.Fatalf("union dialect issues = %#v", got)
			}
		})
	}
}

func TestDialectCompatibilityStrictAPI(t *testing.T) {
	sql := "SELECT CAST(1 AS JSON)"
	parser := New()
	if _, err := parser.ParseOneStmt(sql, "", ""); err != nil {
		t.Fatalf("default union parse: %v", err)
	}
	if _, err := parser.ParseOneStmtForDialect(sql, "", "", DialectMySQL); err != nil {
		t.Fatalf("MySQL strict parse: %v", err)
	}
	if _, err := parser.ParseOneStmtForDialect(sql, "", "", DialectMariaDB); err == nil {
		t.Fatal("MariaDB strict parse accepted CAST AS JSON")
	} else {
		var compatibilityErr *DialectCompatibilityError
		if !errors.As(err, &compatibilityErr) || len(compatibilityErr.Issues) != 1 {
			t.Fatalf("strict error = %#v", err)
		}
		if compatibilityErr.Issues[0].Highlight != "JSON" {
			t.Fatalf("strict diagnostic = %#v", compatibilityErr.Issues[0])
		}
	}
}

func TestDialectCompatibilityHighFrequencyDDLAndShow(t *testing.T) {
	for _, tc := range []struct {
		sql       string
		target    Dialect
		code      DialectIssueCode
		highlight string
	}{
		{"/* 汉字 */\nSHOW REPLICAS", DialectMariaDB, IssueMariaDBShowReplicas, "REPLICAS"},
		{"SHOW MASTER STATUS", DialectMySQL, IssueMySQLMasterStatus, "MASTER"},
		{"CREATE TABLE t(a SERIAL DEFAULT 5)", DialectMariaDB, IssueMariaDBSerialOption, "DEFAULT"},
		{"ALTER TABLE t MODIFY a SERIAL DEFAULT NULL", DialectMariaDB, IssueMariaDBSerialOption, "DEFAULT"},
	} {
		t.Run(tc.sql, func(t *testing.T) {
			parser := New()
			stmt := mustParseOne(t, tc.sql)
			before := restoreStmt(t, stmt)
			issues := parser.CheckDialectCompatibility(stmt, tc.sql, tc.target)
			if len(issues) != 1 || issues[0].Code != tc.code || issues[0].Highlight != tc.highlight {
				t.Fatalf("issues = %#v", issues)
			}
			issue := issues[0]
			line, column := dialectTestLineColumn(tc.sql, issue.Offset)
			if tc.sql[issue.Offset:issue.Offset+issue.Length] != tc.highlight || issue.Line != line || issue.Column != column {
				t.Fatalf("invalid source location: %#v", issue)
			}
			if after := restoreStmt(t, stmt); after != before {
				t.Fatalf("compatibility check mutated AST: %q -> %q", before, after)
			}
			if _, err := parser.ParseOneStmtForDialect(tc.sql, "", "", tc.target); err == nil {
				t.Fatal("strict API accepted incompatible syntax")
			}
			other := DialectMySQL
			if tc.target == DialectMySQL {
				other = DialectMariaDB
			}
			if _, err := parser.ParseOneStmtForDialect(tc.sql, "", "", other); err != nil {
				t.Fatalf("counterpart dialect rejected: %v", err)
			}
		})
	}
	for _, sql := range []string{
		"SHOW REPLICA STATUS", "SHOW BINARY LOG STATUS", "SHOW BINARY LOGS",
		"CREATE TABLE t(a SERIAL COMMENT 'NULL DEFAULT AUTO_INCREMENT' NOT NULL PRIMARY KEY)",
		"CREATE TABLE t(a SERIAL CHECK(a IS NOT NULL))",
		"CREATE TABLE serial(a INT)", "SELECT serial FROM t", "SELECT db.show replicas",
		"CREATE TABLE t(a INT DEFAULT (serial IS NULL))",
	} {
		t.Run(sql, func(t *testing.T) {
			for _, target := range []Dialect{DialectMySQL, DialectMariaDB} {
				if _, err := New().ParseOneStmtForDialect(sql, "", "", target); err != nil {
					t.Fatalf("control rejected for %s: %v", target, err)
				}
			}
		})
	}
}

func TestDialectCompatibilityWindowFramesAndViewWith(t *testing.T) {
	for _, tc := range []struct {
		name, sql, highlight string
		mysql, mariadb       DialectIssueCode
	}{
		{"string start", "/* 汉字 */\nSELECT SUM(a) OVER (RANGE 'abcd' PRECEDING) FROM t", "'abcd'", IssueMySQLWindowStringOffset, ""},
		{"string end", "SELECT SUM(a) OVER (ROWS BETWEEN CURRENT ROW AND '3' FOLLOWING) FROM t", "'3'", IssueMySQLWindowStringOffset, ""},
		{"named window", "SELECT SUM(a) OVER w FROM t WINDOW w AS (ROWS '2' PRECEDING)", "'2'", IssueMySQLWindowStringOffset, ""},
		{"groups", "SELECT SUM(a) OVER (GROUPS CURRENT ROW) FROM t", "GROUPS", IssueMySQLWindowGroups, IssueMariaDBWindowGroups},
		{"exclude current row", "SELECT SUM(a) OVER (ROWS CURRENT ROW EXCLUDE CURRENT ROW) FROM t", "EXCLUDE", IssueMySQLWindowExclusion, IssueMariaDBWindowExclusion},
		{"exclude group", "SELECT SUM(a) OVER (ROWS CURRENT ROW EXCLUDE GROUP) FROM t", "EXCLUDE", IssueMySQLWindowExclusion, IssueMariaDBWindowExclusion},
		{"exclude ties", "SELECT SUM(a) OVER (ROWS CURRENT ROW EXCLUDE TIES) FROM t", "EXCLUDE", IssueMySQLWindowExclusion, IssueMariaDBWindowExclusion},
		{"exclude no others", "SELECT SUM(a) OVER (ROWS CURRENT ROW EXCLUDE NO OTHERS) FROM t", "EXCLUDE", IssueMySQLWindowExclusion, ""},
		{"create view", "/* 汉字 */\nCREATE VIEW v AS (WITH c AS (SELECT 1 AS a) SELECT * FROM c)", "WITH", "", IssueMariaDBViewParenthesizedWith},
		{"alter view", "ALTER VIEW v AS ((WITH c AS (SELECT 1 AS a) SELECT * FROM c))", "WITH", "", IssueMariaDBViewParenthesizedWith},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parser := New()
			parser.EnableWindowFunc(true)
			stmt, err := parser.ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			before := restoreStmt(t, stmt)
			for _, target := range []Dialect{DialectMySQL, DialectMariaDB, DialectUnion} {
				t.Run(target.String(), func(t *testing.T) {
					want := tc.mysql
					switch target {
					case DialectMariaDB:
						want = tc.mariadb
					case DialectUnion:
						want = ""
					}
					issues := parser.CheckDialectCompatibility(stmt, tc.sql, target)
					_, err := parser.ParseOneStmtForDialect(tc.sql, "", "", target)
					if want == "" {
						if len(issues) != 0 || err != nil {
							t.Fatalf("control rejected: issues=%#v err=%v", issues, err)
						}
						return
					}
					if len(issues) != 1 || issues[0].Code != want || issues[0].Highlight != tc.highlight {
						t.Fatalf("issues = %#v", issues)
					}
					issue := issues[0]
					line, column := dialectTestLineColumn(tc.sql, issue.Offset)
					if tc.sql[issue.Offset:issue.Offset+issue.Length] != tc.highlight || issue.Line != line || issue.Column != column {
						t.Fatalf("invalid source range: %#v", issue)
					}
					if _, ok := errors.AsType[*DialectCompatibilityError](err); !ok {
						t.Fatalf("strict error = %v, want compatibility error", err)
					}
				})
			}
			if after := restoreStmt(t, stmt); after != before {
				t.Fatalf("compatibility check mutated AST: %q -> %q", before, after)
			}
		})
	}
	for _, sql := range []string{
		"SELECT exclude,ties,others FROM t",
		"SELECT SUM(a) OVER (ROWS 2 PRECEDING), 'GROUPS EXCLUDE WITH' FROM t",
		"CREATE VIEW v AS WITH c AS (SELECT 1 AS a) SELECT * FROM c",
		"CREATE VIEW v AS (SELECT 1 AS a)",
		"CREATE VIEW v AS SELECT * FROM (WITH c AS (SELECT 1 AS a) SELECT * FROM c) d",
		"WITH c AS (SELECT 1 AS a) SELECT * FROM c",
	} {
		t.Run(sql, func(t *testing.T) {
			for _, target := range []Dialect{DialectMySQL, DialectMariaDB} {
				parser := New()
				parser.EnableWindowFunc(true)
				if _, err := parser.ParseOneStmtForDialect(sql, "", "", target); err != nil {
					t.Fatalf("control rejected for %s: %v", target, err)
				}
			}
		})
	}
}

func TestDialectCompatibilityBuiltinTableNames(t *testing.T) {
	for _, tc := range []struct {
		name, sql, highlight string
		target               Dialect
		code                 DialectIssueCode
		ignoreSpaceOnly      bool
	}{
		{name: "qualified substr", sql: "CREATE TABLE db.SUBSTR(a INT)", highlight: "SUBSTR", target: DialectMariaDB, code: IssueMariaDBBuiltinTableName},
		{name: "qualified substring", sql: "CREATE TABLE db.SUBSTRING(a INT)", highlight: "SUBSTRING", target: DialectMariaDB, code: IssueMariaDBBuiltinTableName},
		{name: "qualified trim", sql: "CREATE TABLE db.TRIM(a INT)", highlight: "TRIM", target: DialectMariaDB, code: IssueMariaDBBuiltinTableName},
		{name: "inline reference", sql: "CREATE TABLE t(a INT REFERENCES db.SUBSTR(a))", highlight: "SUBSTR", target: DialectMariaDB, code: IssueMariaDBBuiltinTableName},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, mode := range []mysql.SQLMode{0, mysql.ModeIgnoreSpace} {
				parser := New()
				parser.SetSQLMode(mode)
				stmt, err := parser.ParseOneStmt(tc.sql, "", "")
				if err != nil {
					t.Fatal(err)
				}
				before := restoreStmt(t, stmt)
				for _, target := range []Dialect{DialectMySQL, DialectMariaDB, DialectUnion} {
					wantIssue := target == tc.target && (!tc.ignoreSpaceOnly || mode.HasIgnoreSpaceMode())
					issues := parser.CheckDialectCompatibility(stmt, tc.sql, target)
					_, err := parser.ParseOneStmtForDialect(tc.sql, "", "", target)
					if !wantIssue {
						if len(issues) != 0 || err != nil {
							t.Fatalf("%s mode %d: control rejected: issues=%#v err=%v", target, mode, issues, err)
						}
						continue
					}
					if len(issues) != 1 || issues[0].Code != tc.code || issues[0].Highlight != tc.highlight {
						t.Fatalf("%s mode %d: issues = %#v", target, mode, issues)
					}
					issue := issues[0]
					line, column := dialectTestLineColumn(tc.sql, issue.Offset)
					if issue.Offset != strings.Index(tc.sql, tc.highlight) || issue.Line != line || issue.Column != column ||
						tc.sql[issue.Offset:issue.Offset+issue.Length] != tc.highlight {
						t.Fatalf("invalid source location: %#v", issue)
					}
					if _, ok := errors.AsType[*DialectCompatibilityError](err); !ok {
						t.Fatalf("strict error = %v, want compatibility error", err)
					}
				}
				if after := restoreStmt(t, stmt); after != before {
					t.Fatalf("compatibility check mutated AST: %q -> %q", before, after)
				}
			}
		})
	}
	for _, sql := range []string{
		"CREATE TABLE `SYSDATE`(a INT)", "CREATE TABLE db.SYSDATE(a INT)",
		"CREATE TABLE SYSDATE AS SELECT 1 AS a", "CREATE TABLE SYSDATE/**/(a INT)",
		"CREATE TABLE db.`SUBSTRING`(a INT)", "CREATE TABLE db.SUBSTR (a INT)",
		"CREATE TABLE db.SUBSTRING\n(a INT)", "CREATE TABLE db.TRIM/**/(a INT)",
		"CREATE TABLE t(a DATETIME DEFAULT (SYSDATE()))",
		"SELECT SYSDATE(),SUBSTRING('abc',1),TRIM('abc')",
		"SELECT 'SYSDATE(a INT)' FROM sysdate",
	} {
		t.Run(sql, func(t *testing.T) {
			for _, mode := range []mysql.SQLMode{0, mysql.ModeIgnoreSpace} {
				for _, target := range []Dialect{DialectMySQL, DialectMariaDB} {
					parser := New()
					parser.SetSQLMode(mode)
					if _, err := parser.ParseOneStmtForDialect(sql, "", "", target); err != nil {
						t.Fatalf("%s mode %d: control rejected: %v", target, mode, err)
					}
				}
			}
		})
	}
}

func TestDialectCompatibilityVisitorAndValidation(t *testing.T) {
	sql := "select cast(cast(1 as json) as json), log(2), log(10)"
	stmt := mustParseOne(t, sql)
	original := restoreStmt(t, stmt)

	visitor := NewDialectCompatibilityVisitor(DialectMariaDB, sql)
	if _, ok := stmt.Accept(visitor); !ok {
		t.Fatal("visitor aborted")
	}
	issues := visitor.Issues()
	if len(issues) != 2 || issues[0].Code != IssueMariaDBCastJSON || issues[1].Code != IssueMariaDBCastJSON {
		t.Fatalf("visitor issues = %#v", issues)
	}
	if got := restoreStmt(t, stmt); got != original {
		t.Fatalf("visitor mutated AST: %q -> %q", original, got)
	}
	if err := ValidateDialectCompatibility(stmt, sql, DialectMariaDB); err == nil {
		t.Fatal("strict validation accepted CAST AS JSON")
	}
	if err := ValidateDialectCompatibility(stmt, sql, DialectUnion); err != nil {
		t.Fatalf("union validation: %v", err)
	}

	mysqlIssues := CheckDialectCompatibility(stmt, sql, DialectMySQL)
	if len(mysqlIssues) != 0 {
		t.Fatalf("MySQL issues = %#v", mysqlIssues)
	}
}

func TestDialectCompatibilityRemainingSyntax(t *testing.T) {
	type finding struct {
		code            DialectIssueCode
		highlight, near string
	}
	for _, tc := range []struct {
		name, sql      string
		mysql, mariadb []finding
	}{
		{name: "period", sql: "/* 汉字 */\nCREATE TABLE t(s DATE,e DATE,PERIOD FOR p(s,e))",
			mysql: []finding{{IssueMySQLPeriod, "PERIOD", "PERIOD FOR"}}},
		{name: "expression index", sql: "/* 索引 */\nCREATE INDEX k ON t ((a+1),a)",
			mariadb: []finding{{IssueMariaDBExpressionIndex, "(", "(a+1)"}}},
		{name: "inline expression index", sql: "CREATE TABLE t(a INT,UNIQUE KEY k((a+1)))",
			mariadb: []finding{{IssueMariaDBExpressionIndex, "(", "(a+1)"}}},
		{name: "alter expression index", sql: "ALTER TABLE t ADD KEY k((a+1))",
			mariadb: []finding{{IssueMariaDBExpressionIndex, "(", "(a+1)"}}},
		{name: "omitted partition keyword", sql: "CREATE TABLE t(a INT) PARTITION BY RANGE(a) (p VALUES LESS THAN(2),PARTITION q VALUES LESS THAN MAXVALUE)",
			mysql: []finding{{IssueMySQLPartitionKeyword, "p", "p VALUES"}}},
		{name: "alter omitted partition keyword", sql: "ALTER TABLE t ADD PARTITION (p VALUES LESS THAN(2))",
			mysql: []finding{{IssueMySQLPartitionKeyword, "p", "p VALUES"}}},
		{name: "row return", sql: "CREATE FUNCTION f() RETURNS ROW TYPE OF db.t RETURN 1",
			mysql: []finding{{IssueMySQLReturnsRowType, "ROW", "ROW TYPE"}}},
		{name: "backup lock", sql: "BACKUP LOCK db.t",
			mysql: []finding{{IssueMySQLBackup, "BACKUP", "BACKUP LOCK"}}},
		{name: "wrapped backup unlock", sql: "SET STATEMENT max_statement_time=180 FOR BACKUP UNLOCK",
			mysql: []finding{{IssueMySQLBackup, "BACKUP", "BACKUP UNLOCK"}}},
		{name: "conditional foreign key drop", sql: "ALTER TABLE t DROP FOREIGN KEY IF EXISTS fk",
			mysql: []finding{{IssueMySQLDropForeignKeyIfExists, "IF", "IF EXISTS"}}},
		{name: "analyze plural", sql: "ANALYZE TABLES t,u",
			mysql: []finding{{IssueMySQLAnalyzeTables, "TABLES", "TABLES t"}}},
		{name: "analyze local plural", sql: "ANALYZE LOCAL TABLES t",
			mysql: []finding{{IssueMySQLAnalyzeTables, "TABLES", "TABLES t"}}},
		{name: "analyze no binlog plural", sql: "ANALYZE NO_WRITE_TO_BINLOG TABLES t",
			mysql: []finding{{IssueMySQLAnalyzeTables, "TABLES", "TABLES t"}}},
		{name: "delete using tail", sql: "DELETE FROM t USING t WHERE a=1 ORDER BY a LIMIT 2",
			mysql: []finding{{IssueMySQLMultiDeleteTail, "ORDER", "ORDER BY"}, {IssueMySQLMultiDeleteTail, "LIMIT", "LIMIT 2"}}},
		{name: "nested delete tail", sql: "WITH c AS (SELECT 1 ORDER BY 1 LIMIT 1) DELETE FROM t USING t,c WHERE a IN (SELECT a FROM u ORDER BY a LIMIT 1) LIMIT 2",
			mysql: []finding{{IssueMySQLMultiDeleteTail, "LIMIT", "LIMIT 2"}}},
		{name: "routine delete boundary", sql: "CREATE PROCEDURE p() BEGIN DELETE FROM t USING t ORDER BY a; DELETE FROM u WHERE a IN (SELECT a FROM v ORDER BY a LIMIT 1) LIMIT 2; END",
			mysql: []finding{{IssueMySQLMultiDeleteTail, "ORDER", "ORDER BY a;"}}},
		{name: "bare window names", sql: "SELECT window,rank FROM t",
			mysql: []finding{{IssueMySQLWindowIdentifier, "window", "window,"}, {IssueMySQLWindowIdentifier, "rank", "rank FROM"}}},
		{name: "window column alias", sql: "SELECT window AS w FROM t",
			mysql: []finding{{IssueMySQLWindowIdentifier, "window", "window AS"}}},
		{name: "rank alias", sql: "SELECT 1 AS rank",
			mysql: []finding{{IssueMySQLWindowIdentifier, "rank", "rank"}}},
		{name: "window function alias", sql: "SELECT RANK() OVER w AS rank FROM t WINDOW w AS (ORDER BY a)",
			mysql: []finding{{IssueMySQLWindowIdentifier, "rank", "rank FROM"}}},
		{name: "window prefix index", sql: "CREATE TABLE t(window VARCHAR(10), KEY(window(3)))",
			mysql: []finding{{IssueMySQLWindowIdentifier, "window", "window VARCHAR"}, {IssueMySQLWindowIdentifier, "window", "window(3)"}}},
		{name: "rank prefix index", sql: "CREATE TABLE t(rank VARCHAR(10), KEY(rank(3)))",
			mysql: []finding{{IssueMySQLWindowIdentifier, "rank", "rank VARCHAR"}, {IssueMySQLWindowIdentifier, "rank", "rank(3)"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parser := New()
			parser.EnableWindowFunc(true)
			stmt, err := parser.ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatalf("union parse: %v", err)
			}
			before := restoreStmt(t, stmt)
			original, err := parser.ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			for _, target := range []Dialect{DialectMySQL, DialectMariaDB, DialectUnion} {
				want := tc.mysql
				switch target {
				case DialectMariaDB:
					want = tc.mariadb
				case DialectUnion:
					want = nil
				}
				issues := parser.CheckDialectCompatibility(stmt, tc.sql, target)
				if len(issues) != len(want) {
					t.Fatalf("%s issues = %#v, want %#v", target, issues, want)
				}
				for i, issue := range issues {
					offset := strings.Index(tc.sql, want[i].near)
					line, column := dialectTestLineColumn(tc.sql, offset)
					if issue.Code != want[i].code || issue.Target != target || issue.Highlight != want[i].highlight ||
						issue.Offset != offset || issue.Length != len(want[i].highlight) || issue.Line != line || issue.Column != column ||
						tc.sql[issue.Offset:issue.Offset+issue.Length] != issue.Highlight {
						t.Fatalf("%s diagnostic = %#v, want %#v at %d", target, issue, want[i], offset)
					}
				}
				_, err := parser.ParseOneStmtForDialect(tc.sql, "", "", target)
				if len(want) == 0 {
					if err != nil {
						t.Fatalf("%s control rejected: %v", target, err)
					}
				} else {
					var compatibilityErr *DialectCompatibilityError
					if !errors.As(err, &compatibilityErr) || !reflect.DeepEqual(compatibilityErr.Issues, issues) {
						t.Fatalf("%s strict diagnostic = %v, want %#v", target, err, issues)
					}
				}
			}
			if after := restoreStmt(t, stmt); after != before || !reflect.DeepEqual(stmt, original) {
				t.Fatal("compatibility check mutated AST")
			}
		})
	}
}

func TestDialectCompatibilityRemainingControls(t *testing.T) {
	for _, sql := range []string{
		"CREATE TABLE t(s DATE,e DATE,`period` INT)",
		"CREATE TABLE t(a INT) PARTITION BY KEY() PARTITIONS 2",
		"CREATE TABLE t(a INT) PARTITION BY LINEAR HASH(a) PARTITIONS 2",
		"CREATE TABLE t(a INT) PARTITION BY RANGE(a) (PARTITION p VALUES LESS THAN(2))",
		"ALTER TABLE t DROP FOREIGN KEY fk",
		"ANALYZE TABLE t", "ANALYZE NO_WRITE_TO_BINLOG TABLE t",
		"CREATE FUNCTION f() RETURNS INT RETURN 1",
		"DELETE FROM t WHERE a IN (SELECT a FROM u ORDER BY a LIMIT 1) ORDER BY a LIMIT 2",
		"DELETE FROM t USING t WHERE EXISTS(SELECT 1 FROM u ORDER BY a LIMIT 1)",
		"CREATE PROCEDURE p() BEGIN DELETE FROM t USING t; SELECT 1 ORDER BY 1 LIMIT 2; END",
		"SELECT `rank`,`window`,t.rank,db.t.window FROM t",
		"SELECT rank() OVER (ORDER BY a) FROM t",
		"SELECT row_number() OVER w FROM t WINDOW w AS (ORDER BY a)",
		"SELECT SUM(a) OVER w FROM t WINDOW w AS (ORDER BY a), z AS ()",
		"SELECT 'BACKUP LOCK t RETURNS ROW TYPE OF t ANALYZE TABLES t DROP FOREIGN KEY IF EXISTS fk window rank' /* ANALYZE TABLES t */",
		"SELECT t.analyze tables FROM t",
		"CREATE TABLE t(`window` VARCHAR(10), KEY(`window`(3)))",
		"SET CHARACTER SET cp1250_latin2", "SET NAMES utf8mb4",
		"SELECT SUM(a) OVER `window` FROM t WINDOW `window` AS ()",
	} {
		t.Run(sql, func(t *testing.T) {
			parser := New()
			parser.EnableWindowFunc(true)
			stmt, err := parser.ParseOneStmt(sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			before := restoreStmt(t, stmt)
			for _, target := range []Dialect{DialectMySQL, DialectMariaDB, DialectUnion} {
				if issues := parser.CheckDialectCompatibility(stmt, sql, target); len(issues) != 0 {
					t.Fatalf("%s control issues = %#v", target, issues)
				}
				if _, err := parser.ParseOneStmtForDialect(sql, "", "", target); err != nil {
					t.Fatalf("%s control rejected: %v", target, err)
				}
			}
			if after := restoreStmt(t, stmt); after != before {
				t.Fatal("compatibility check mutated AST")
			}
		})
	}
}

func TestDialectCompatibilityControlsAndUTF8Position(t *testing.T) {
	controls := []string{
		"SELECT 1 INTO @v FROM t",
		"SELECT 1 UNION SELECT 2 INTO @v",
		"SELECT 1.e9",
		"SELECT CAST(1 AS CHAR)",
		"INSERT INTO t VALUES (1) ON DUPLICATE KEY UPDATE a=VALUES(a)",
		"CREATE TABLE t (a INT UNSIGNED ZEROFILL)",
		"SELECT LOG(1), LOG(2, 8)",
		"SELECT `log`()",
		"SET GLOBAL default_storage_engine = DEFAULT",
		"CREATE ROLE `FILE`, 'PROXY'",
		"SELECT * FROM { OJ t1 LEFT JOIN t2 ON TRUE }",
		"SELECT CONCAT(a1,a2) FROM t1",
		"CREATE TABLE t (a VARCHAR(10) COLLATE ascii_bin)",
	}
	for _, sql := range controls {
		stmt := mustParseOne(t, sql)
		if got := CheckDialectCompatibility(stmt, sql, DialectMySQL); len(got) != 0 {
			t.Errorf("MySQL control %q: %#v", sql, got)
		}
		if got := CheckDialectCompatibility(stmt, sql, DialectMariaDB); len(got) != 0 {
			t.Errorf("MariaDB control %q: %#v", sql, got)
		}
	}

	sql := "SELECT '汉字', 1.e(1)"
	stmt := mustParseOne(t, sql)
	issues := CheckDialectCompatibility(stmt, sql, DialectMariaDB)
	if len(issues) != 1 {
		t.Fatalf("UTF-8 issues = %#v", issues)
	}
	wantOffset := strings.Index(sql, "1.e")
	if issues[0].Offset != wantOffset || issues[0].Column != utf8.RuneCountInString(sql[:wantOffset]) {
		t.Fatalf("UTF-8 diagnostic = %#v", issues[0])
	}
}

func dialectTestLineColumn(source string, offset int) (line, column int) {
	line = 1
	for _, r := range source[:offset] {
		if r == '\n' {
			line++
			column = 0
		} else {
			column++
		}
	}
	return line, column
}

var _ ast.Visitor = (*DialectCompatibilityVisitor)(nil)

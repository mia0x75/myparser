[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/mia0x75/myparser "Go version")](https://go.dev/dl/)

# myparser

A Go SQL syntax parser for MySQL 5.7 / 8.0 / 8.4 and MariaDB. It builds an
abstract syntax tree (AST), supports visitor-based inspection and rewriting, and
restores AST nodes to SQL.

The default API accepts a combined MySQL/MariaDB grammar. Strict dialect APIs
reject known incompatibilities with the selected database family. This is a
syntax library, not a SQL execution engine or a complete server validator.

## Getting started

Requires **Go 1.27.3 or later**.

```sh
go get github.com/mia0x75/myparser
```

Import the driver to register literal and value-expression implementations:

```go
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/mia0x75/myparser"
	_ "github.com/mia0x75/myparser/driver"
	"github.com/mia0x75/myparser/format"
)

func main() {
	p := myparser.New()
	p.EnableWindowFunc(true)
	stmts, warnings, err := p.Parse("SELECT * FROM tbl WHERE id = 1", "", "")
	if err != nil {
		log.Fatal(err)
	}
	for _, warning := range warnings {
		log.Print(warning)
	}
	for _, stmt := range stmts {
		var sql strings.Builder
		if err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sql)); err != nil {
			log.Fatal(err)
		}
		fmt.Println(sql.String())
	}
}
```

A parser instance can be shared by multiple goroutines: each parse uses its own
state. `SetSQLMode` and `EnableWindowFunc` can also be changed concurrently;
in-flight parses keep the settings captured when they started. ASTs are mutable;
coordinate access when sharing a returned tree.

### Dialect checks

Use the strict entry point when the target database family is known:

```go
sql := "SELECT CAST(doc AS JSON) FROM t"
stmt, err := p.ParseOneStmtForDialect(sql, "", "", myparser.DialectMariaDB)
if err != nil {
	// Handle syntax errors or *myparser.DialectCompatibilityError.
	return
}
_ = stmt
```

For an existing AST, call
`myparser.CheckDialectCompatibility(stmt, sql, myparser.DialectMariaDB)`.
Each `DialectIssue` includes a stable rule code, target dialect, UTF-8 byte range,
line number, rune-based column number, and highlighted source text.

Executable version comments are filtered against fixed targets: **MySQL 8.4.13**
and **MariaDB 12.3.3**. The combined API executes comments accepted by either
target. When version comments matter, use a strict parse entry point directly:
checking a combined AST afterward cannot undo comment filtering.

## Features

- DDL and DML, including table and index definitions, queries, joins, CTEs, set
  operations, and insert/update/delete statements.
- Stored procedures, functions, triggers, events, compound blocks, handlers,
  cursors, loops, and diagnostic statements.
- Management statements for users, replication, plugins, components, tablespaces,
  logfile groups, resource groups, and table maintenance.
- Expressions, aggregate and window functions, SQL modes, and executable comments.
  Window-function parsing must be enabled with `EnableWindowFunc(true)`.
- AST traversal through `Accept` and SQL output through `Restore`, with configurable
  quoting and keyword formatting.
- Dialect-specific compatibility diagnostics for supported MySQL/MariaDB differences.

Support is defined by `parser.y` and the tests. These categories do not imply
complete coverage of every server version or every combination of clauses.

### AST details and normalization

- `GROUP_CONCAT` stores its separator, ordering, and row limit in
  `AggregateFuncExpr.Separator`, `OrderBy`, and `Limit`; `Args` contains only actual
  arguments. MariaDB's `GROUP_CONCAT ... LIMIT` and `JSON_ARRAYAGG` extensions are
  accepted by the combined grammar and rejected by the strict MySQL API.
- Leading `INTERVAL ... + ...` syntax is preserved through
  `FuncCallExpr.IntervalFirst`.
- Parenthesized CTAS supports SELECT, set operations, and CTEs. Quantified
  comparisons (`ALL`/`ANY`/`SOME`) support `VALUES ROW(...)` queries.
- `PERIOD FOR p(s,e)` is stored in `CreateTableStmt.Periods` and accepts exactly
  two unqualified column names. Partition definitions preserve KEY/LINEAR,
  counts, HASH/KEY subpartitions, and ENGINE/TABLESPACE/COMMENT options.
  `PartitionDefinition.Options` retains all options; `Comment` exposes the first
  COMMENT for compatibility.
- Expression indexes are supported for ordinary and unique secondary indexes,
  including CREATE INDEX. PRIMARY, FOREIGN, REFERENCES, FULLTEXT, and SPATIAL
  keys still accept columns only.
- Foreign keys accept ON UPDATE alone and either ordering of UPDATE/DELETE
  actions. Restore emits DELETE before UPDATE.
- The combined grammar also accepts `DROP FOREIGN KEY IF EXISTS`,
  `RETURNS ROW TYPE OF`, `BACKUP LOCK/UNLOCK`, `ANALYZE TABLES`, and DELETE USING
  with ORDER BY/LIMIT. Strict dialect APIs check target-specific restrictions.
- LEAD/LAG/NTH_VALUE/NTILE arguments can be expressions. `GROUP_CONCAT` with OVER
  becomes `WindowFuncExpr`; otherwise it remains `AggregateFuncExpr`.
  `rank`/`window` are allowed in selected column, expression, and alias positions;
  labels and DECLARE names follow their own rules.
- SET CHARACTER SET / SET CHARSET use `VariableAssignment.Name == ast.SetCharset`
  and restore as CHARACTER SET, distinct from SET NAMES. `cp1250_latin2` is
  accepted only in this syntax, not registered as a general charset.
- UUID, INET4, and INET6 map to CHAR(36), BIGINT UNSIGNED, and VARBINARY(16).
  Bare LONG maps to BIGINT. LONG BYTE uses `TypeLongBlob` and currently restores
  as LONGTEXT. Overflowing YEAR widths normalize to an unspecified width.
  These mappings do not guarantee native type semantics or original spelling.

## Limitations

- Parsing does not execute SQL, resolve schema objects, enforce privileges, or
  provide full type checking or stored-program variable-scope validation.
- Restore normalizes SQL. It does not promise byte-for-byte preservation of
  whitespace, comments, spelling, or all dialect-specific type semantics.
- Custom `DELIMITER` separators require whitespace after the statement, such as
  `END $$`.
- `utf8mb3` is not registered. NCHAR/NATIONAL/NVARCHAR types are unsupported.
- ODBC expression escapes are unsupported; `{ OJ ... }` table references are supported.
- `UNION=(t,t2)` table options and `FLUSH DES_KEY_FILE` are not implemented.
- Hexadecimal ENUM/SET elements and `ENUM('a') BINARY` are not implemented.
  An explicit CHARACTER SET / COLLATE clause changes charset or comparison
  semantics and must be evaluated against the data and target server.
- Table aliases written as `= alias` are unsupported; use `AS alias` or an
  implicit alias.

The parser targets SQL used in everyday database administration. It does not aim
to reproduce the exact validation stage, error code, or error text a server uses
for malformed input. In particular:

| Unsupported form | Example | Alternative |
| --- | --- | --- |
| Aliases inside function arguments | `ABS(3 AS three)` or `ABS(3 three)` | `ABS(3) AS three` |
| Derived tables without aliases, including nested/set-operation forms | `SELECT * FROM (SELECT 1 UNION SELECT 2)` | Add `AS t` |
| Invalid LOG or JSON_ARRAYAGG arity | `LOG()`, `LOG(1,2,3)`, `JSON_ARRAYAGG(a,b)` | Use one or two arguments for LOG, one for JSON_ARRAYAGG |
| Standalone, incomplete stored-program control flow with subqueries and undeclared variables | `CASE (SELECT 2) OR foo WHEN 1 THEN SET @x=10;` | Supply a complete control structure and valid declarations |

The function-argument alias restriction also applies to INSTR, CONV, ATAN,
CRC32, and CRC32C. The stored-program restriction covers these incomplete IF,
CASE, WHILE, and REPEAT forms; it does not reject every undeclared variable use.

## Development

```sh
make build           # Build all packages
make test            # Run unit tests
make race            # Race detection with randomized test order
make vet             # Go static analysis
make lint            # Pinned golangci-lint
make verify          # Build, vet, race, and lint
make parser          # Regenerate parser.go and reject grammar conflicts
make generate-check  # Regenerate and check for an uncommitted parser diff
make fmt             # Format Go source
```

`parser.go` is generated: edit `parser.y` and run `make parser` rather than
editing it manually. Generation downloads
`golang.org/x/tools/cmd/goyacc@v0.38.0`; lint uses golangci-lint v2.15.0.
Both need access to the configured Go module proxy on their first run.

GitHub Actions runs the Makefile checks, CodeQL analysis, and Conventional Commit
PR-title validation. Dependabot checks Go modules and GitHub Actions daily.
Dependabot PRs, including major updates, are automatically approved and queued
for squash merging. Enable **Allow auto-merge**, **Allow squash merging**, and
**Allow GitHub Actions to create and approve pull requests** in repository
settings. Protect `main` with required status checks so merging waits for CI.
Inactive issues become stale after 90 days and close after another 5; PRs become
stale after 45 days and close after another 10. Items with milestones are exempt.

### Layout

| Path | Purpose |
| --- | --- |
| `parser.y` / `parser.go` | Grammar and generated LALR parser |
| `ast/` | AST nodes, visitors, and SQL restoration |
| `dialect.go` | Dialect compatibility checks and strict parsing APIs |
| `types/` | Data types and literals |
| `sqlerr/` | SQL error classes and codes |
| `mysql/` | Server constants and error codes |
| `format/` | SQL restoration output and formatting |
| `driver/` | Value-expression implementation registered with the AST |

## License

Dual-licensed under the [MIT License](LICENSE-MIT) and
[Anti 996 License 1.0](LICENSE-ANTI-996). You may choose either license.

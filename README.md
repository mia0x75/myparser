# myparser

[![Go version](https://img.shields.io/github/go-mod/go-version/mia0x75/myparser)](https://go.dev/dl/)

A Go SQL syntax parser for MySQL 5.7 / 8.0 / 8.4 and MariaDB. It builds an AST,
supports visitor-based inspection and rewriting, and restores AST nodes to SQL.
It does not execute SQL or provide full server validation.

## Getting started

Requires **Go 1.27.3 or later**. The current `go.mod` uses a local `parserkit`
dependency. Clone both repositories side by side; a standalone checkout or
`go get` is not supported yet:

```sh
git clone https://github.com/mia0x75/parserkit.git
git clone https://github.com/mia0x75/myparser.git
cd myparser
go mod download
```

No explicit driver import is needed:

```go
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/mia0x75/myparser"
	"github.com/mia0x75/myparser/format"
)

func main() {
	p := myparser.New()
	p.EnableWindowFunc(true)
	stmts, warnings, err := p.Parse("SELECT id FROM tbl WHERE id = 1", "", "")
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

A parser can be shared by multiple goroutines. Each parse captures its own
settings, so `SetSQLMode` and `EnableWindowFunc` can also be changed concurrently.
Returned ASTs are mutable; synchronize access to shared trees.

## Dialects

The default API accepts a combined MySQL/MariaDB grammar. Strict entry points
reject known incompatibilities with the selected database family:

```go
sql := "SELECT CAST(doc AS JSON) FROM t"
stmt, err := p.ParseOneStmtForDialect(sql, "", "", myparser.DialectMariaDB)
if err != nil {
	// Handle syntax errors or *myparser.DialectCompatibilityError.
	return
}
_ = stmt
```

For an existing AST, use
`myparser.CheckDialectCompatibility(stmt, sql, myparser.DialectMariaDB)`.
Issues include a rule code, target dialect, UTF-8 byte range, line number,
rune-based column, and highlighted source text. These checks are not a complete
validator for every server version.

Executable version comments use fixed targets: **MySQL 8.4.13** and
**MariaDB 12.3.3**. The combined API accepts comments enabled by either target.
Use strict parsing when this matters: checking an AST afterward cannot undo
comment filtering.

## Features

- DDL and DML: tables, indexes, queries, joins, CTEs, set operations, and
  insert/update/delete statements.
- Stored procedures, functions, triggers, events, blocks, handlers, cursors,
  loops, and diagnostics.
- User, replication, plugin, component, resource, and table management statements.
- Expressions, aggregates, SQL modes, and executable comments. Enable window
  functions with `EnableWindowFunc(true)`.
- AST traversal through `Accept` and SQL output through `Restore`, with
  configurable quoting and keyword formatting.

Support is defined by `parser.y` and the tests, not by these broad categories.

### AST API

The AST API has breaking changes from older versions. Use the current types in
`ast/` and the shared interfaces in `parserkit/ast`:

- `Span()` returns a half-open UTF-8 byte range. Names use `QualifiedName.Parts`;
  identifiers retain spelling, quoting, and source ranges.
- SELECT fields use `FieldList.Items`, with `SelectItem.Expr` and `Alias`.
  Wildcards are `WildcardExpr` nodes. Use `Alias.Present()` to detect an alias,
  including an empty quoted alias.
- Literals use `LiteralExpr.Kind`, `Value`, and `Raw`, not runtime `Datum` or
  `FieldType` values. Parameter markers have a zero-based `Index` per statement.
  Legacy helpers remain in `driver/`; `driver.ValueExpr` is not an AST node.
- INSERT uses `Values`, `Query`, and `Assignments`; UPDATE uses `Table`,
  `Assignments`, and `OrderBy`; DELETE uses `Table`, `Using`, `Targets`, and
  `OrderBy`. Assignment targets are expressions, normally `ColumnRef` nodes.
- Shared fields use `*Core()` methods or embedded `Base*` structs. Composite
  literals must initialize embedded structs explicitly. ORDER BY, GROUP BY,
  and window partitions use distinct item types.
- `TableSource` holds aliases and index hints. `QueryNode` represents a query;
  a parenthesized `SubqueryExpr` is not a bare query node.
- Aggregate and window calls store `COUNT(*)` in `Star`, and ordering,
  separators, and row limits outside `Args`. `FuncCallExpr.IntervalFirst`
  preserves leading INTERVAL syntax.
- `Accept` reports failed traversal with `ok == false`; always check it.
  `Restore` rejects missing required children and invalid window-frame enums.
- Connection charset and collation arguments are kept for API compatibility;
  they do not add type metadata or implicit charset introducers to the AST.

## Limitations

- No SQL execution, schema resolution, privilege checks, full type checking, or
  stored-program variable-scope validation.
- Restore normalizes SQL; it does not preserve all whitespace, comments,
  spelling, or dialect-specific type semantics.
- UUID, INET4, and INET6 normalize to CHAR(36), BIGINT UNSIGNED, and VARBINARY(16).
  LONG normalizes to BIGINT; LONG BYTE restores as LONGTEXT. These mappings do
  not preserve native type semantics. Overflowing YEAR widths become unspecified.
- Expression indexes are limited to ordinary and unique secondary indexes.
  PRIMARY, FOREIGN, REFERENCES, FULLTEXT, and SPATIAL keys accept columns only.
- Custom `DELIMITER` separators need preceding whitespace, such as `END $$`.
- `utf8mb3` is not registered. NCHAR/NATIONAL/NVARCHAR types are unsupported.
- ODBC escapes, `UNION=(t,t2)`, and `FLUSH DES_KEY_FILE` are unsupported.
- Hexadecimal ENUM/SET elements and `ENUM('a') BINARY` are unsupported. An
  explicit CHARACTER SET / COLLATE clause has different semantics; evaluate it
  against the data and target server.
- Table aliases written as `= alias` are unsupported; use `AS alias` or an
  implicit alias.

Malformed SQL may fail at a different stage, or with different diagnostics, than
on a server. For example, function-argument aliases are rejected; derived tables
need aliases; LOG takes one or two arguments and JSON_ARRAYAGG takes one.
Incomplete IF/CASE/WHILE/REPEAT fragments are not substitutes for complete stored
programs, but parsing does not reject every undeclared variable use.

## Development

```sh
make build           # Build all packages
make test            # Run unit tests
make race            # Race detection with randomized test order
make vet             # Go static analysis
make lint            # Pinned golangci-lint
make verify          # Build, vet, race, and lint
make parser          # Regenerate parser.go and reject grammar conflicts
make generate-check  # Check committed generated parser consistency
make fmt             # Format Go source
```

Edit `parser.y`, not the generated `parser.go`, then run `make parser`.
Generation uses goyacc v0.52.0 and `parserkit/cmd/yaccspan`; lint uses
golangci-lint v2.15.0. Initial runs need access to the Go module proxy.

### MTR tests

`compat_mtr*_test.go` contains offline regressions derived from MTR cases:
parsing, AST inspection, visitor behavior, and SQL round trips. These tests do
not connect to a database. `cmd/mtrcheck/` checks the corpus in `testdata/mtr/`;
live-check scripts compare results with actual servers and distinguish syntax
errors from execution errors.

Some MTR files and tools are local-only via `.git/info/exclude`, including
`compat_mtr_remaining_test.go`, `cmd/mtrcheck/`, `testdata/mtr/`, `scripts/`, and
the `mtrcheck` binary. Go tests outside `testdata/` still run locally, but excluded
files are absent from fresh clones and CI. Local-only tests are not release gates.

### CI

GitHub Actions runs Makefile checks, CodeQL, and Conventional Commit PR-title
validation. The local `parserkit` replacement requires a sibling checkout on CI;
the workflow does not provide one yet.

Dependabot checks modules and actions daily, with automatic approval and squash
auto-merge, including major updates. Enable **Allow auto-merge**, **Allow squash
merging**, and **Allow GitHub Actions to create and approve pull requests**.
Protect `main` with required checks. The stale workflow closes inactive
issues and PRs; items with milestones are exempt.

### Layout

| Path | Purpose |
| --- | --- |
| `parser.y` / `parser.go` | Grammar and generated LALR parser |
| `ast/` | AST nodes, visitors, and SQL restoration |
| `dialect.go` | Dialect checks and strict parsing APIs |
| `types/` | Legacy data types and value conversion |
| `errors/` | Generic error wrapping and stack traces |
| `sqlerr/` | SQL error classes and codes |
| `mysql/` | Server constants and error codes |
| `format/` | SQL output and formatting |
| `driver/` | Legacy value helpers and literal conversion |

## License

Dual-licensed under the [MIT License](LICENSE-MIT) and
[Anti 996 License 1.0](LICENSE-ANTI-996). You may choose either license.

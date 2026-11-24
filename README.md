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

The parser calls the driver package directly for legacy literal conversion;
callers do not need to import it explicitly:

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

- 本批跨方言 API 收口：`SelectStmt.From.FromClauseCore().Table` 使用公共 `ResultSetNode`，解析器仍
  生成既有 Join 根。`WithClause.CTEs` 使用 `[]CTENode`，共享字段通过
  `CTECore()` 访问，主体 `Query StmtNode` 可供其它方言表示可写 CTE；MySQL
  文法允许的 CTE 主体保持原范围，MariaDB CYCLE 字段保留。
- 普通、聚合、窗口调用的 `Name` 统一为 `QualifiedName`，使用 `Name.Parts`
  读取标识符，整体范围使用 `Name.Span`。名称引号及分段保留，只有单段未加引号
  名称参与特殊内建语法识别；限定名称不会因末段同名被当作内建函数。
  MySQL 限定函数调用文法未扩张。分段表示每个函数名增加一次分配；五类解析
  样本的 B/op 增加 1.1%～3.4%，耗时测量噪声较大，不作速度结论。
- 窗口单位与边界方向增加 Unknown 零值，合法值必须显式设置；`UnBounded`
  重命名为 `Unbounded`。Restore 拒绝未知单位、方向与 EXCLUDE 枚举；合法
  ROWS/RANGE/GROUPS 和边界输出保持。公共具体节点的 nil 接收者 Accept
  统一返回 `(nil, false)`，skip 前校验外层类型，typed-nil 子槽与替换结果会停止。
- 本地开发仍通过 `replace github.com/mia0x75/parserkit => ../parserkit` 解析依赖。
  parserkit 尚未发布可获取版本，因此当前依赖配置不支持单独 fresh clone 或远程 CI。
- `ast.UpdateStmt/DeleteStmt` 分别值内嵌公共 `BaseUpdateStmt/BaseDeleteStmt`。
  旧 UPDATE `TableRefs/List/Order` 改为 `Table/Assignments/OrderBy`；
  DELETE 的 `TableRefs/Tables/Order` 改为单目标 `Table`、来源 `Using`、
  完整多目标 `Targets []*TableName` 和 `OrderBy`，移除 `DeleteTableList` 节点。
  这是源不兼容的 AST API 迁移，复合字面量需显式初始化对应 Base。
  `Table/Using` 保存完整真实关系树，只去掉最终 DML 边界的一层空合成根；
  单 DELETE 即使无索引提示也始终保留原 MySQL `TableSource`，不剥别名／分区／hint。
  `OrderByClauseNode/LimitNode` 用 nil 安全 Core 暴露公共数据，仍分发到实际外层。
  `With`、优先级及真实修饰保留；多 DELETE 两式与 `MultipleTable` 的原含义不变。
  Visitor 只减少 DML 的 TableRefsClause、空合成根 Join 和 DeleteTableList 事件，
  来源和表达式的相对访问次序不变。五个裸入口及两个 WITH 包装保存最终字节 Span。
  单 DELETE 现在保留文法已接收但旧动作丢弃的优化器提示，其 canonical 因此有意变化；
  不改变文法、优先级或 LR 表，不引入服务端绑定／可写性验证。
- 旧 `ast.ByItem` 按真实语法角色拆为 `OrderItem`、`GroupItem` 和窗口
  `PartitionItem`。排序项别名共享公共具体类型；分组项值内嵌公共
  `BaseGroupItem`，MySQL 的 `Desc` 仍保留在外层，窗口分区也不丢弃原方向。
  `OrderByClause/GroupByClause` 分别值内嵌公共 `BaseOrderByClause/BaseGroupByClause`；
  `Items` 为对应 `OrderItemNode/GroupItemNode` 切片，公共字段经
  `item.OrderItemCore().Expr/Desc` 或 `item.GroupItemCore().Expr` 读取。
  分组的 ROLLUP/CUBE/视图 CHECK OPTION 字段与排序的旧 `ForUnion` 字段保留。
  窗口分区使用 `Items []*PartitionItem`；不能将所有旧 ByItem 机械改为 OrderItem。
  Visitor 保持外层身份与条目源顺序，角色不匹配的替换会停止；条目和子句
  保存字节 Span。缺省方向与显式 ASC 仍同为 `Desc=false`，不新增词法信息。
- `ast.InsertStmt` 值内嵌公共 `pkast.BaseInsertStmt`，公共字段在复合字面量中
  显式写入 `BaseInsertStmt: pkast.BaseInsertStmt{...}`，读取使用提升字段。
  `Lists/Select/Setlist` 分别改为 `Values/Query/Assignments`；INSERT SET 的
  `Assignments` 与冲突动作 `OnDuplicate` 分开，非 nil 空 SET 列表仍保留。
  `Table` 是公共 `TableRefNode`，解析结果直接保存原 `*ast.TableName`，分区
  数据仍属于该节点，不再经过 TableRefsClause/Join/TableSource 包装。
  名称可用 `stmt.Table.TableRefCore().Name` 读取，目标 Span 包含显式分区。
  `Columns []ExprNode` 在 MySQL 中保存真实 `*ast.ColumnRef`，其 `Name` 仍为
  完整 QualifiedName；该公共类别也能承载方言字段/下标目标，不执行绑定检查。
  `Query` 必须是公共 `QueryNode`，不能用普通表来源或子查询表达式冒充。
  Visitor 顺序为 Query、Table、Columns、Values、Assignments、OnDuplicate、
  Returning；原有子节点相对顺序不变，但目标三层包装的事件移除、列引用事件新增。
  空列清单与缺少清单不合并，RowAlias.Columns 仍是名称值而不是表达式。
- `ast.Assignment` 是公共具体节点的别名，旧 `Column QualifiedName` 改为
  `Target ExprNode`，MySQL 文法在该槽内构造 `*ast.ColumnRef`。读取名称使用
  `assignment.Target.(*ast.ColumnRef).Name`；通用代码可经公共 `ColumnRefNode`
  访问名称。Visitor 现在先遍历目标，再遍历值；整个赋值和目标分别保存 Span。
  复制管理选项不再复用 Assignment，改为方言 `ast.ReplicationOption`，
  使用 `Name lex.Ident` 与 `Expr`，只遍历值并继续按关键字还原名称。
  不改变文法接受范围；UPDATE 使用 `Assignments`，加载操作的 `Setlist` 等仍按各自
  语法角色在后续批次处理，不做全局同名字段替换。
- `ast.Limit` 值内嵌公共 `ast.BaseLimit`，构造时显式设置
  `BaseLimit: pkast.BaseLimit{Count: ..., Offset: ...}`；读取仍使用提升的
  `Count/Offset`。公共数据不绑定 LIMIT/OFFSET/FETCH 的方言输出，
  MySQL 外层继续输出既有 `LIMIT offset,count`。Visitor 的 Enter 替换
  现在遍历替换节点的表达式；不兼容替换会停止。缺少 Count 的手工节点
  Restore 返回错误，不再因空接口崩溃。PG 的接入仍在后续批次。
- `ast.TableSource` 是值内嵌公共 `pkast.BaseTableSource` 的真实 MySQL 外层，
  不再是公共具体类型别名。构造时显式写入 `BaseTableSource: pkast.BaseTableSource{...}`；
  `Source` 是公共 `ResultSetNode`，`Alias` 是 `lex.Ident`，`Columns` 保存有序列别名。
  通用读取使用 `TableSourceNode.TableSourceCore()`，Visitor 保留实际 MySQL 外层身份。
  原 `TableName.IndexHints` 移至 `TableSource.IndexHints`，在来源、分区、别名之后
  按序还原；`TableName.PartitionNames` 仍供 INSERT、管理/装载等直接目标使用。
  不保留两份提示字段，不新增 INSERT 索引提示语法；IndexHint 仍是非 Node 值。
  SELECT、集合运算、VALUES 和 TABLE 语句通过公共 `QueryNode` 区分裸查询
  与表来源；普通表达式不能再冒充结果集节点。表来源的 `Span()` 包含完整
  来源、别名和索引提示；TableName 节点范围在表因子及单 DELETE 中含分区，
  `TableName.Name.Span` 仍只含限定名称，各 Ident.Span 只覆盖各自 token。
  `SubqueryExpr` 内嵌公共 `BaseQueryExpr`，同时是表达式和结果集，保留自身
  括号，但不是裸查询 `QueryNode`；CTAS 可保留这个包装而不丢失括号。
  CTE 的 `Name/Columns`、窗口 `Name/Ref`、WITH 的 `CycleColumns`、
  视图及 ANALYZE 的 `Columns` 同样保留词法标识符，而不是 `CIStr` 缓存。
  INSERT 的行别名使用 `Name lex.Ident` 与完整 `Span loc.Span`，不再用
  `Offset`；它的 `Columns []QualifiedName` 保留冻结文法已接受的限定分段，
  默认 Restore 继续输出最后一段列别名，不改变既有宽松输入行为。
  CTE、窗口及 DML/DDL 的剩余结构仍按后续批次收敛，并非全量迁移已完成。
- SELECT 和 RETURNING 投影统一使用公共具体节点：`ast.FieldList.Items`
  保存 `[]*ast.SelectItem`，每项只有 `Expr` 与单个 `lex.Ident` 类型的 `Alias`。
  `*`、`t.*` 和 `db.t.*` 使用 `*ast.WildcardExpr`，`Qualifier.Parts`
  保留限定名分段、原始拼写、引号与字节范围。Visitor 会遍历这些通配符。
  旧 `SelectField`、`WildCardField`、`WildCard/Expr` 双槽、`AsName`、
  投影项 `Offset` 和执行器 `Auxiliary` 已移除；位置读取 `Span()`。
  用 `Alias.Present()` 判断是否有别名，不能用 `Alias.Name == ""`，因为
  空引号别名仍然存在。默认 Restore 继续按上下文输出引号，`Raw/Quote`
  保留词法来源而不强制原样输出。没有别名的投影 `Text()` 保持旧用途，
  需要精确语法范围时使用左闭右开的 `Span()`。
- **已移除（D 批次）**：`ast.ValueExpr` interface、`ast.NewValueExpr`、
  `ast.NewDecimal`、`ast.NewHexLiteral`、`ast.NewBitLiteral`、
  `ast.TexprNode`、`ast.exprNode`（legacy 类型/flag 缓存基类）、
  `ast.SetFlag`（per-stmt 空遍历）全部移除。语法 AST 不再携带
  `types.FieldType`、`types.Datum` 或执行层 `flag/projectionOffset` 字段。
  旧的执行层字面量桥接和解码函数保留在 `driver.ValueExpr`、
  `driver.NewValueExpr`、`driver.NewDecimal`、`driver.NewHexLiteral`、
  `driver.NewBitLiteral`，仅供显式 legacy 调用方使用；`driver.ValueExpr`
  不再满足 `ast.ExprNode`。
- `NULL`、`TRUE` 和 `FALSE` 解析为纯语法 `*ast.LiteralExpr`，通过
  `Kind` 区分，`Value` 保存规范小写文本，`Raw` 保留输入拼写；
  下游应按字面量种类读取语法信息，不再使用 `ValueExpr`/`GetValue()`。
- 整数、定点数和浮点数字面量同样使用 `*ast.LiteralExpr`，`Kind` 分别为
  `LiteralInteger`、`LiteralDecimal` 和 `LiteralFloat`；`Value` 保存兼容既有
  Restore 行为的规范数值文本，`Raw` 和 `Span()` 保留源码拼写与字节范围。
  下游不能再将这些节点断言为旧的执行层值节点或通过 `GetValue()` 读取运行时值。
  复制选项仍按原规则读取整数前缀并在溢出时饱和，heartbeat 保留方言精度边界。
- 字符串、二进制字符串、hex 和 bit 使用 `*ast.LiteralExpr` 的
  `LiteralString`、`LiteralBytes`、`LiteralHex`、`LiteralBit`；字符串的 `Value`
  是解码文本，hex/bit 的 `Value` 是不含前缀或引号的源码数字串。
  `Quote` 保留原始引号，显式字符集 introducer（含 `N`）保存在
  `Introducer` 的名称、原文与字节范围中。相邻字符串的 `Value` 合并，
  `Raw/Span()` 保留完整拼接源码；不再保存逻辑计划使用的 `projectionOffset`。
  `CHAR ... USING`、`SET ... ON` 和事务特征沿用既有的字符串规范化表示，
  但同样不再构造带执行期类型的旧值节点。
  `Parse` 的连接字符集和排序规则参数仅为 API 兼容保留，不注入语法 AST，
  也不再由这些参数为普通字符串补 Restore introducer。显式 introducer
  的既有规范化 Restore 保持不变，下游应独立处理连接设置和类型推断。
- 参数占位符解析为 `*ast.ParamMarkerExpr`：`Index` 在每条顶层语句内按词法顺序
  从零编号，`Span()` 独立保存左闭右开字节范围。手工构造时可用 `-1` 表示
  未知编号；`NewParamMarkerExpr(index)` 的位置未知，指定位置使用
  `NewParamMarkerExprAt(span, index)`，`SetOrder` 仅更新编号。
- `GROUP_CONCAT` stores its separator, ordering, and row limit in
  `AggregateFuncExpr.Separator`, `OrderBy`, and `Limit`; `Args` contains only actual
  arguments. MariaDB's `GROUP_CONCAT ... LIMIT` and `JSON_ARRAYAGG` extensions are
  accepted by the combined grammar and rejected by the strict MySQL API.
- `COUNT(*)` sets `Star` on `AggregateFuncExpr` / `WindowFuncExpr` instead of
  inserting a numeric `1` into `Args`; `Restore` emits `COUNT(*)` verbatim.
- Windowed `GROUP_CONCAT` with `ORDER BY`, `SEPARATOR`, `LIMIT` stores those
  tails on `WindowFuncExpr.OrderBy`, `.Separator`,
  `.Limit`; `Restore` emits them inside the call's argument list before `OVER`.
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
- ODBC escapes, including `{ OJ ... }` table references, are unsupported.
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
`golang.org/x/tools/cmd/goyacc@v0.52.0`; lint uses golangci-lint v2.15.0.
Both need access to the configured Go module proxy on their first run.

Some MTR compatibility files and helper tools are local-only and excluded via
`.git/info/exclude`: `compat_mtr_remaining_test.go`, `cmd/mtrcheck/`,
`testdata/mtr/`, `scripts/`, and the `mtrcheck` binary. They exist in this
working tree and are discovered by `go test ./...`, but a fresh clone and CI do
not receive them. A local all-green run therefore covers more tests than CI;
do not treat local-only MTR tests as release gates.

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
| `driver/` | 迁移期旧值表达式与词法字面量转换的兼容实现 |

## License

Dual-licensed under the [MIT License](LICENSE-MIT) and
[Anti 996 License 1.0](LICENSE-ANTI-996). You may choose either license.

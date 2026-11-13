package myparser

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/mysql"
)

// Dialect identifies the SQL grammar against which a parsed AST is checked.
// DialectUnion is the parser's default, permissive MySQL/MariaDB surface.
type Dialect uint8

const (
	DialectUnion Dialect = iota
	DialectMySQL
	DialectMariaDB
)

// String implements fmt.Stringer.
func (d Dialect) String() string {
	switch d {
	case DialectUnion:
		return "union"
	case DialectMySQL:
		return "mysql"
	case DialectMariaDB:
		return "mariadb"
	default:
		return fmt.Sprintf("dialect(%d)", d)
	}
}

// DialectIssueCode identifies a stable compatibility rule.
type DialectIssueCode string

const (
	IssueMariaDBMultipleInto           DialectIssueCode = "mariadb.multiple-into"
	IssueMariaDBNumericIdentifier      DialectIssueCode = "mariadb.numeric-identifier"
	IssueMariaDBCastJSON               DialectIssueCode = "mariadb.cast-json"
	IssueMariaDBValuesRowSyntax        DialectIssueCode = "mariadb.values-row-syntax"
	IssueMariaDBTypeModifier           DialectIssueCode = "mariadb.type-modifier"
	IssueMariaDBLastValueArgumentCount DialectIssueCode = "mariadb.last-value-argument-count"
	IssueMariaDBInsertRowAlias         DialectIssueCode = "mariadb.insert-row-alias"
	IssueMySQLCastVarchar              DialectIssueCode = "mysql.cast-varchar"
	IssueMySQLValuesRowSyntax          DialectIssueCode = "mysql.values-row-syntax"
	IssueMySQLLogArgumentCount         DialectIssueCode = "mysql.log-argument-count"
	IssueMySQLLeadingDotSetValue       DialectIssueCode = "mysql.leading-dot-set-value"
	IssueMySQLBareSetScope             DialectIssueCode = "mysql.bare-set-scope"
	IssueMySQLODBCEscapeMarker         DialectIssueCode = "mysql.odbc-escape-marker"
	IssueMySQLDuplicateCollate         DialectIssueCode = "mysql.duplicate-collate"
	IssueMySQLScalarLastValue          DialectIssueCode = "mysql.scalar-last-value"
	IssueMySQLGroupConcatLimit         DialectIssueCode = "mysql.group-concat-limit"
	IssueMySQLJSONArrayAggClauses      DialectIssueCode = "mysql.json-arrayagg-clauses"
	IssueMariaDBJSONArrayAggAll        DialectIssueCode = "mariadb.json-arrayagg-all"
	IssueMariaDBJSONArrayAggWindow     DialectIssueCode = "mariadb.json-arrayagg-window"
	IssueMySQLCastTarget               DialectIssueCode = "mysql.cast-target"
	IssueMySQLValueFunction            DialectIssueCode = "mysql.value-function"
	IssueMySQLFilenameCharset          DialectIssueCode = "mysql.filename-charset"
	IssueMySQLMasterStatus             DialectIssueCode = "mysql.master-status"
	IssueMariaDBShowReplicas           DialectIssueCode = "mariadb.show-replicas"
	IssueMariaDBSerialOption           DialectIssueCode = "mariadb.serial-option"
	IssueMySQLWindowStringOffset       DialectIssueCode = "mysql.window-string-offset"
	IssueMySQLWindowGroups             DialectIssueCode = "mysql.window-groups"
	IssueMySQLWindowExclusion          DialectIssueCode = "mysql.window-exclusion"
	IssueMariaDBWindowGroups           DialectIssueCode = "mariadb.window-groups"
	IssueMariaDBWindowExclusion        DialectIssueCode = "mariadb.window-exclusion"
	IssueMariaDBViewParenthesizedWith  DialectIssueCode = "mariadb.view-parenthesized-with"
	IssueMariaDBBuiltinTableName       DialectIssueCode = "mariadb.builtin-table-name"
	IssueMariaDBExpressionIndex        DialectIssueCode = "mariadb.expression-index"
	IssueMySQLPeriod                   DialectIssueCode = "mysql.period"
	IssueMySQLPartitionKeyword         DialectIssueCode = "mysql.partition-keyword"
	IssueMySQLReturnsRowType           DialectIssueCode = "mysql.returns-row-type"
	IssueMySQLBackup                   DialectIssueCode = "mysql.backup"
	IssueMySQLDropForeignKeyIfExists   DialectIssueCode = "mysql.drop-foreign-key-if-exists"
	IssueMySQLAnalyzeTables            DialectIssueCode = "mysql.analyze-tables"
	IssueMySQLMultiDeleteTail          DialectIssueCode = "mysql.multi-delete-tail"
	IssueMySQLWindowIdentifier         DialectIssueCode = "mysql.window-identifier"
)

// DialectIssue is one source-located incompatibility. Offset and Length are
// UTF-8 byte offsets. Line is one-based and Column is the zero-based rune
// column. Highlight is the exact source slice at that range.
type DialectIssue struct {
	Code      DialectIssueCode
	Target    Dialect
	Message   string
	Offset    int
	Length    int
	Line      int
	Column    int
	Highlight string
}

// Error returns a human-readable form of the issue.
func (i DialectIssue) Error() string {
	return fmt.Sprintf("%s compatibility issue at line %d column %d near %q: %s", i.Target, i.Line, i.Column, i.Highlight, i.Message)
}

// DialectCompatibilityError is returned by strict validation when one or more
// target-dialect issues were found.
type DialectCompatibilityError struct {
	Issues []DialectIssue
}

// Error implements error.
func (e *DialectCompatibilityError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "dialect compatibility error"
	}
	if len(e.Issues) == 1 {
		return e.Issues[0].Error()
	}
	return fmt.Sprintf("%s (and %d more issues)", e.Issues[0].Error(), len(e.Issues)-1)
}

type dialectSourceToken struct {
	kind int
	text string
	raw  string
	pos  Pos
	end  int
}

type dialectFunctionCandidate struct {
	token   dialectSourceToken
	builtin bool
	index   int
}

func (t dialectSourceToken) isWord(word string) bool {
	return t.kind == identifier && strings.EqualFold(t.text, word)
}

func (t dialectSourceToken) isRaw(raw string) bool {
	return t.raw == raw
}

// DialectCompatibilityVisitor applies target-dialect policy after parsing. It
// never changes the AST. Source tokens are used only where the AST deliberately
// normalizes or omits spelling details.
type DialectCompatibilityVisitor struct {
	Target Dialect
	Source string

	sqlMode  mysql.SQLMode
	analyzed bool
	tokens   []dialectSourceToken
	issues   []DialectIssue
	seen     map[string]struct{}

	castJSONCandidates     []dialectSourceToken
	castJSONCursor         int
	castVarcharCandidates  []dialectSourceToken
	castVarcharCursor      int
	logCandidates          []dialectFunctionCandidate
	logCursor              int
	lastValueCandidates    []dialectFunctionCandidate
	lastValueCursor        int
	duplicateCollateTokens []dialectSourceToken
	duplicateCollateCursor int
	functionCandidates     map[string][]dialectFunctionCandidate
	functionCursors        map[string]int
}

// NewDialectCompatibilityVisitor creates a non-mutating compatibility visitor.
func NewDialectCompatibilityVisitor(target Dialect, source string) *DialectCompatibilityVisitor {
	return &DialectCompatibilityVisitor{Target: target, Source: source}
}

// SetSQLMode configures source tokenization. It must be called before visiting.
func (v *DialectCompatibilityVisitor) SetSQLMode(mode mysql.SQLMode) {
	if v.analyzed {
		return
	}
	v.sqlMode = mode
}

// Issues returns a source-ordered copy of the collected diagnostics.
func (v *DialectCompatibilityVisitor) Issues() []DialectIssue {
	issues := append([]DialectIssue(nil), v.issues...)
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Offset != issues[j].Offset {
			return issues[i].Offset < issues[j].Offset
		}
		return issues[i].Code < issues[j].Code
	})
	return issues
}

// Enter implements ast.Visitor.
func (v *DialectCompatibilityVisitor) Enter(n ast.Node) (ast.Node, bool) {
	v.analyzeSource()

	switch x := n.(type) {
	case *ast.TableName:
		v.checkBuiltinTableName(x)
	case *ast.PeriodDefinition:
		if v.Target == DialectMySQL {
			if token, ok := v.tokenAtOffset(x.Offset); ok {
				v.addIssue(token, IssueMySQLPeriod, "PERIOD FOR declarations are MariaDB-only")
			}
		}
	case *ast.PartitionDefinition:
		if v.Target == DialectMySQL && x.OmitPartitionKeyword {
			if token, ok := v.tokenAtOffset(x.NameOffset); ok {
				v.addIssue(token, IssueMySQLPartitionKeyword, "MySQL partition definitions require PARTITION before the name")
			}
		}
	case *ast.IndexColName:
		if v.Target == DialectMariaDB && x.Expr != nil {
			if token, ok := v.tokenAtOffset(x.ExprOffset); ok {
				v.addIssue(token, IssueMariaDBExpressionIndex, "expression index key parts are MySQL-only")
			}
		}
	case *ast.DeleteStmt:
		if v.Target == DialectMySQL && x.IsMultiTable && (x.Order != nil || x.Limit != nil) {
			v.checkMultiDeleteTail(x)
		}
	case *ast.CreateViewStmt, *ast.AlterViewStmt:
		v.checkParenthesizedViewWith()
	case *ast.FrameClause:
		if x.Type == ast.Groups {
			if token, ok := v.tokenAtOffset(x.TypeOffset); ok && token.isWord("groups") {
				switch v.Target {
				case DialectMySQL:
					v.addIssue(token, IssueMySQLWindowGroups, "MySQL recognizes GROUPS but does not support it")
				case DialectMariaDB:
					v.addIssue(token, IssueMariaDBWindowGroups, "MariaDB does not recognize GROUPS window frames")
				}
			}
		}
		if token, ok := v.tokenAtOffset(x.ExclusionOffset); ok && x.Exclusion != "" && token.isWord("exclude") {
			switch v.Target {
			case DialectMySQL:
				v.addIssue(token, IssueMySQLWindowExclusion, "MySQL recognizes EXCLUDE but does not support it")
			case DialectMariaDB:
				if x.Exclusion != "NO OTHERS" {
					v.addIssue(token, IssueMariaDBWindowExclusion, "MariaDB supports only EXCLUDE NO OTHERS")
				}
			}
		}
	case *ast.FrameBound:
		if v.Target == DialectMySQL && x.Unit == nil {
			if token, ok := v.tokenAtOffset(x.ExprOffset); ok && token.kind == stringLit {
				v.addIssue(token, IssueMySQLWindowStringOffset, "string window frame offsets are MariaDB-only")
			}
		}
	case *ast.FuncCastExpr:
		if v.Target == DialectMariaDB && x.FunctionType == ast.CastFunction && x.Tp != nil && x.Tp.Tp == mysql.TypeJSON {
			if token, ok := v.nextCastJSONCandidate(); ok {
				v.addIssue(token, IssueMariaDBCastJSON, "CAST(... AS JSON) is MySQL-only")
			}
		}
		if v.Target == DialectMySQL && x.FunctionType != ast.CastBinaryOperator && x.Tp != nil && x.Tp.Tp == mysql.TypeVarchar {
			if token, ok := v.nextCastVarcharCandidate(); ok {
				v.addIssue(token, IssueMySQLCastVarchar, "VARCHAR(n) is a MariaDB-only CAST/CONVERT target")
			}
		}
	case *ast.ValuesStmt:
		if len(x.Rows) == 0 {
			break
		}
		if v.Target == DialectMySQL && x.Rows[0].OmitRowKeyword {
			if token, ok := v.valuesRowSyntaxToken(x.ValuesKeywordOffset, "("); ok {
				v.addIssue(token, IssueMySQLValuesRowSyntax, "MySQL table value constructors require ROW before each row")
			}
		}
		if v.Target == DialectMariaDB && !x.Rows[0].OmitRowKeyword {
			if token, ok := v.valuesRowSyntaxToken(x.ValuesKeywordOffset, "row"); ok {
				v.addIssue(token, IssueMariaDBValuesRowSyntax, "MariaDB table value constructors do not accept ROW")
			}
		}
	case *ast.InsertStmt:
		if v.Target == DialectMariaDB && x.RowAlias != nil {
			if token, ok := v.tokenAtOffset(x.RowAlias.Offset); ok {
				v.addIssue(token, IssueMariaDBInsertRowAlias, "the INSERT row alias is MySQL-only")
			}
		}
	case *ast.FuncCallExpr:
		if v.Target == DialectMySQL && x.FnName.L == "log" {
			candidate, ok := v.nextLogCandidate()
			if ok && candidate.builtin && len(x.Args) != 1 && len(x.Args) != 2 {
				v.addIssue(candidate.token, IssueMySQLLogArgumentCount, "MySQL LOG accepts one or two arguments")
			}
		}
		if x.FnName.L == "last_value" {
			candidate, ok := v.nextLastValueCandidate()
			if ok && candidate.builtin {
				switch v.Target {
				case DialectMySQL:
					v.addIssue(candidate.token, IssueMySQLScalarLastValue, "LAST_VALUE without OVER is a MariaDB-only scalar function")
				case DialectMariaDB:
					if len(x.Args) == 0 {
						v.addIssue(candidate.token, IssueMariaDBLastValueArgumentCount, "MariaDB LAST_VALUE requires at least one argument")
					}
				}
			}
		}
	case *ast.AggregateFuncExpr:
		switch strings.ToLower(x.F) {
		case "group_concat":
			candidate, ok := v.nextFunctionCandidate(x.F)
			if ok && v.Target == DialectMySQL && x.Limit != nil {
				v.addIssue(candidate.token, IssueMySQLGroupConcatLimit, "GROUP_CONCAT LIMIT is MariaDB-only")
			}
		case "json_arrayagg":
			candidate, ok := v.nextFunctionCandidate(x.F)
			if !ok {
				break
			}
			if v.Target == DialectMySQL && (x.Distinct || x.OrderBy != nil || x.Limit != nil) {
				v.addIssue(candidate.token, IssueMySQLJSONArrayAggClauses, "JSON_ARRAYAGG DISTINCT, ORDER BY, and LIMIT are MariaDB-only")
			}
			v.checkJSONArrayAggAll(candidate)
		}
	case *ast.WindowFuncExpr:
		if strings.EqualFold(x.F, "last_value") {
			v.nextLastValueCandidate()
		}
		if strings.EqualFold(x.F, "json_arrayagg") {
			if candidate, ok := v.nextFunctionCandidate(x.F); ok && candidate.builtin {
				if v.Target == DialectMariaDB {
					v.addIssue(candidate.token, IssueMariaDBJSONArrayAggWindow, "MariaDB JSON_ARRAYAGG cannot be used as a window function")
				}
				v.checkJSONArrayAggAll(candidate)
			}
		}
	case *ast.ColumnDef:
		if v.Target == DialectMySQL {
			collates, generated := 0, false
			for _, option := range x.Options {
				switch option.Tp {
				case ast.ColumnOptionCollate:
					collates++
				case ast.ColumnOptionGenerated:
					generated = true
				}
			}
			if !generated {
				for i := 1; i < collates && v.duplicateCollateCursor < len(v.duplicateCollateTokens); i++ {
					token := v.duplicateCollateTokens[v.duplicateCollateCursor]
					v.duplicateCollateCursor++
					v.addIssue(token, IssueMySQLDuplicateCollate, "MySQL permits only one COLLATE clause per column")
				}
			}
		}
	}

	return n, false
}

func (v *DialectCompatibilityVisitor) checkBuiltinTableName(table *ast.TableName) {
	// The parser rejects an unqualified SYSDATE before '(' for both dialects,
	// so only MariaDB's qualified table names still need a compatibility issue.
	if v.Target != DialectMariaDB || table.Schema.L == "" {
		return
	}
	code := IssueMariaDBBuiltinTableName
	message := "MariaDB requires quoting or a gap before '(' for this qualified table name"
	switch table.Name.L {
	case "substr", "substring", "trim":
	default:
		return
	}
	for i, token := range v.tokens {
		if token.pos.Offset != table.NameOffset {
			continue
		}
		if !token.isWord(table.Name.O) || i+1 >= len(v.tokens) {
			return
		}
		next := v.tokens[i+1]
		if !next.isRaw("(") || next.pos.Offset < token.end {
			return
		}
		nativeSyntax := token.end == next.pos.Offset
		if v.Target == DialectMySQL && v.sqlMode.HasIgnoreSpaceMode() {
			nativeSyntax = strings.TrimSpace(v.Source[token.end:next.pos.Offset]) == ""
		}
		if nativeSyntax {
			v.addIssue(token, code, message)
		}
		return
	}
}

func (v *DialectCompatibilityVisitor) checkParenthesizedViewWith() {
	if v.Target != DialectMariaDB {
		return
	}
	depth := 0
	for i, token := range v.tokens {
		switch token.raw {
		case "(":
			depth++
		case ")":
			depth--
		}
		if depth != 0 || !token.isWord("as") {
			continue
		}
		j := i + 1
		for j < len(v.tokens) && v.tokens[j].isRaw("(") {
			j++
		}
		// The AST normalizes parentheses, so this boundary needs the original tokens.
		if j > i+1 && j < len(v.tokens) && v.tokens[j].isWord("with") {
			v.addIssue(v.tokens[j], IssueMariaDBViewParenthesizedWith, "MariaDB view bodies cannot start with a parenthesized WITH clause")
		}
		return
	}
}

// Leave implements ast.Visitor.
func (v *DialectCompatibilityVisitor) Leave(n ast.Node) (ast.Node, bool) {
	return n, true
}

func (v *DialectCompatibilityVisitor) nextCastJSONCandidate() (dialectSourceToken, bool) {
	if v.castJSONCursor >= len(v.castJSONCandidates) {
		return dialectSourceToken{}, false
	}
	token := v.castJSONCandidates[v.castJSONCursor]
	v.castJSONCursor++
	return token, true
}

func (v *DialectCompatibilityVisitor) nextCastVarcharCandidate() (dialectSourceToken, bool) {
	if v.castVarcharCursor >= len(v.castVarcharCandidates) {
		return dialectSourceToken{}, false
	}
	token := v.castVarcharCandidates[v.castVarcharCursor]
	v.castVarcharCursor++
	return token, true
}

func (v *DialectCompatibilityVisitor) tokenAtOffset(offset int) (dialectSourceToken, bool) {
	for _, token := range v.tokens {
		if token.pos.Offset == offset {
			return token, true
		}
	}
	return dialectSourceToken{}, false
}

func (v *DialectCompatibilityVisitor) checkMultiDeleteTail(stmt *ast.DeleteStmt) {
	if stmt.Tables == nil || len(stmt.Tables.Tables) == 0 {
		return
	}
	start := stmt.Tables.Tables[0].NameOffset
	depth := 0
	for i, token := range v.tokens {
		if token.pos.Offset < start {
			continue
		}
		if token.isRaw("(") {
			depth++
			continue
		}
		if token.isRaw(")") {
			if depth == 0 {
				break
			}
			depth--
			continue
		}
		if token.isRaw(";") {
			break
		}
		if depth == 0 && ((stmt.Limit != nil && token.isWord("limit")) ||
			(stmt.Order != nil && token.isWord("order") && i+1 < len(v.tokens) && v.tokens[i+1].isWord("by"))) {
			v.addIssue(token, IssueMySQLMultiDeleteTail, "ORDER BY and LIMIT on a multi-table DELETE are MariaDB-only")
		}
	}
}

func (v *DialectCompatibilityVisitor) valuesRowSyntaxToken(offset int, spelling string) (dialectSourceToken, bool) {
	for i, token := range v.tokens {
		if token.pos.Offset != offset || !token.isWord("values") {
			continue
		}
		if i+1 >= len(v.tokens) {
			return dialectSourceToken{}, false
		}
		next := v.tokens[i+1]
		if (spelling == "(" && next.isRaw("(")) || next.isWord(spelling) {
			return next, true
		}
		return dialectSourceToken{}, false
	}
	return dialectSourceToken{}, false
}

func (v *DialectCompatibilityVisitor) nextLastValueCandidate() (dialectFunctionCandidate, bool) {
	if v.lastValueCursor >= len(v.lastValueCandidates) {
		return dialectFunctionCandidate{}, false
	}
	token := v.lastValueCandidates[v.lastValueCursor]
	v.lastValueCursor++
	return token, true
}

func (v *DialectCompatibilityVisitor) nextLogCandidate() (dialectFunctionCandidate, bool) {
	if v.logCursor >= len(v.logCandidates) {
		return dialectFunctionCandidate{}, false
	}
	token := v.logCandidates[v.logCursor]
	v.logCursor++
	return token, true
}

func (v *DialectCompatibilityVisitor) nextFunctionCandidate(name string) (dialectFunctionCandidate, bool) {
	name = strings.ToLower(name)
	if v.functionCandidates == nil {
		v.functionCandidates = make(map[string][]dialectFunctionCandidate)
		v.functionCursors = make(map[string]int)
	}
	candidates, found := v.functionCandidates[name]
	if !found {
		candidates = findFunctionCandidates(v.tokens, name)
		v.functionCandidates[name] = candidates
	}
	index := v.functionCursors[name]
	if index >= len(candidates) {
		return dialectFunctionCandidate{}, false
	}
	v.functionCursors[name]++
	return candidates[index], true
}

func (v *DialectCompatibilityVisitor) checkJSONArrayAggAll(candidate dialectFunctionCandidate) {
	index := candidate.index + 2
	if v.Target == DialectMariaDB && index < len(v.tokens) && v.tokens[index].isWord("all") {
		v.addIssue(v.tokens[index], IssueMariaDBJSONArrayAggAll, "MariaDB JSON_ARRAYAGG does not accept ALL")
	}
}

func (v *DialectCompatibilityVisitor) addIssue(token dialectSourceToken, code DialectIssueCode, message string) {
	key := fmt.Sprintf("%s:%d", code, token.pos.Offset)
	if _, exists := v.seen[key]; exists {
		return
	}
	v.seen[key] = struct{}{}
	v.issues = append(v.issues, DialectIssue{
		Code:      code,
		Target:    v.Target,
		Message:   message,
		Offset:    token.pos.Offset,
		Length:    token.end - token.pos.Offset,
		Line:      token.pos.Line,
		Column:    token.pos.Col,
		Highlight: token.raw,
	})
}

func (v *DialectCompatibilityVisitor) analyzeSource() {
	if v.analyzed {
		return
	}
	v.analyzed = true
	v.seen = make(map[string]struct{})
	if v.Target == DialectUnion || v.Source == "" {
		return
	}
	v.tokens = tokenizeDialectSourceForTarget(v.Source, v.sqlMode, v.Target)
	v.castJSONCandidates = findCastJSONCandidates(v.tokens)
	v.castVarcharCandidates = findCastVarcharCandidates(v.tokens)
	v.logCandidates = findFunctionCandidates(v.tokens, "log")
	v.lastValueCandidates = findFunctionCandidates(v.tokens, "last_value")
	v.duplicateCollateTokens = findDuplicateColumnCollates(v.tokens)
	for i, token := range v.tokens {
		if !token.isWord("show") || i+1 >= len(v.tokens) || (i > 0 && v.tokens[i-1].isRaw(".")) {
			continue
		}
		if v.Target == DialectMariaDB && v.tokens[i+1].isWord("replicas") {
			v.addIssue(v.tokens[i+1], IssueMariaDBShowReplicas, "SHOW REPLICAS is MySQL-only")
		}
		if v.Target == DialectMySQL && i+2 < len(v.tokens) &&
			v.tokens[i+1].isWord("master") && v.tokens[i+2].isWord("status") {
			v.addIssue(v.tokens[i+1], IssueMySQLMasterStatus, "MySQL 8.4 uses SHOW BINARY LOG STATUS instead of SHOW MASTER STATUS")
		}
	}

	switch v.Target {
	case DialectMariaDB:
		for _, token := range findDuplicateIntoTokens(v.tokens) {
			v.addIssue(token, IssueMariaDBMultipleInto, "MariaDB permits only one INTO clause per query block")
		}
		for _, token := range v.tokens {
			if isMySQLNumericIdentifier(token) {
				v.addIssue(token, IssueMariaDBNumericIdentifier, "this numeric identifier boundary is not accepted by MariaDB")
			}
		}
		for _, token := range findMariaDBTypeModifierIssues(v.tokens) {
			v.addIssue(token, IssueMariaDBTypeModifier, "this numeric type modifier combination is not accepted by MariaDB")
		}
		for _, token := range findMariaDBSerialOptionIssues(v.tokens) {
			v.addIssue(token, IssueMariaDBSerialOption, "MariaDB SERIAL does not accept NULL, DEFAULT, or another AUTO_INCREMENT")
		}
	case DialectMySQL:
		for _, target := range []string{"int", "integer", "interval"} {
			for _, token := range findCastVarcharCandidatesForTarget(v.tokens, target) {
				v.addIssue(token, IssueMySQLCastTarget, "this CAST/CONVERT target is MariaDB-only")
			}
		}
		for _, candidate := range findFunctionCandidates(v.tokens, "value") {
			if candidate.builtin {
				v.addIssue(candidate.token, IssueMySQLValueFunction, "VALUE(column) is MariaDB-only")
			}
		}
		for i, token := range v.tokens {
			if i > 0 && v.tokens[i-1].isRaw(".") {
				continue
			}
			var next dialectSourceToken
			if i+1 < len(v.tokens) {
				next = v.tokens[i+1]
			}
			if token.isWord("backup") && (next.isWord("lock") || next.isWord("unlock")) {
				v.addIssue(token, IssueMySQLBackup, "BACKUP LOCK and BACKUP UNLOCK are MariaDB-only")
			}
			if token.isWord("returns") && i+3 < len(v.tokens) && next.isWord("row") &&
				v.tokens[i+2].isWord("type") && v.tokens[i+3].isWord("of") {
				v.addIssue(next, IssueMySQLReturnsRowType, "RETURNS ROW TYPE OF is MariaDB-only")
			}
			if token.isWord("drop") && i+4 < len(v.tokens) && next.isWord("foreign") &&
				v.tokens[i+2].isWord("key") && v.tokens[i+3].isWord("if") && v.tokens[i+4].isWord("exists") {
				v.addIssue(v.tokens[i+3], IssueMySQLDropForeignKeyIfExists, "DROP FOREIGN KEY IF EXISTS is MariaDB-only")
			}
			if token.isWord("analyze") {
				index := i + 1
				if next.isWord("local") || next.isWord("no_write_to_binlog") {
					index++
				}
				if index < len(v.tokens) && v.tokens[index].isWord("tables") {
					v.addIssue(v.tokens[index], IssueMySQLAnalyzeTables, "ANALYZE TABLES is a MariaDB-only spelling")
				}
			}
			if token.isWord("rank") || token.isWord("window") {
				windowClause := token.isWord("window") && i+2 < len(v.tokens) && v.tokens[i+2].isWord("as")
				rankFunction := token.isWord("rank") && i+3 < len(v.tokens) && next.isRaw("(") &&
					v.tokens[i+2].isRaw(")") && v.tokens[i+3].isWord("over")
				if !rankFunction && !windowClause {
					v.addIssue(token, IssueMySQLWindowIdentifier, "unqualified RANK and WINDOW identifiers must be quoted in MySQL")
				}
			}
			if strings.EqualFold(token.raw, "_filename") && i+1 < len(v.tokens) {
				switch v.tokens[i+1].kind {
				case stringLit, hexLit, bitLit:
					v.addIssue(token, IssueMySQLFilenameCharset, "the filename character set introducer is MariaDB-only")
				}
			}
		}
		for _, finding := range findMySQLSetIssues(v.tokens) {
			v.addIssue(finding.token, finding.code, finding.message)
		}
		for _, token := range findInvalidMySQLODBCMarkers(v.tokens) {
			v.addIssue(token, IssueMySQLODBCEscapeMarker, "MySQL ODBC join escapes require the unquoted OJ marker")
		}
	}
}

//nolint:unused

func tokenizeDialectSourceForTarget(source string, mode mysql.SQLMode, target Dialect) []dialectSourceToken {
	var scanner Scanner
	scanner.SetSQLMode(mode)
	scanner.target = target
	scanner.reset(source)
	lineStarts := dialectLineStarts(source)
	tokens := make([]dialectSourceToken, 0, 32)
	for {
		kind, pos, text := scanner.scan()
		if kind == 0 {
			break
		}
		if kind == int(';') {
			scanner.stmtStart = true
		} else {
			scanner.stmtStart = false
		}
		end := dialectScannerOffset(&scanner)
		if end < pos.Offset || end > len(source) {
			end = pos.Offset + len(text)
		}
		if end < pos.Offset || end > len(source) {
			end = pos.Offset
		}
		pos.Line, pos.Col = dialectLineColumn(source, lineStarts, pos.Offset)
		tokens = append(tokens, dialectSourceToken{
			kind: kind,
			text: text,
			raw:  source[pos.Offset:end],
			pos:  pos,
			end:  end,
		})
	}
	return tokens
}

func dialectLineStarts(source string) []int {
	starts := []int{0}
	for offset, r := range source {
		if r == '\n' {
			starts = append(starts, offset+1)
		}
	}
	return starts
}

func dialectLineColumn(source string, starts []int, offset int) (line, column int) {
	lineIndex := sort.Search(len(starts), func(i int) bool { return starts[i] > offset }) - 1
	if lineIndex < 0 {
		lineIndex = 0
	}
	return lineIndex + 1, utf8.RuneCountInString(source[starts[lineIndex]:offset])
}

func dialectScannerOffset(scanner *Scanner) int {
	switch special := scanner.specialComment.(type) {
	case *mysqlSpecificCodeScanner:
		return special.Offset + dialectScannerOffset(special.Scanner)
	case *optimizerHintScanner:
		return special.Offset + dialectScannerOffset(special.Scanner)
	default:
		return scanner.r.p.Offset
	}
}

func findDuplicateIntoTokens(tokens []dialectSourceToken) []dialectSourceToken {
	type queryBlock struct{ intoCount int }
	blocks := make(map[int]*queryBlock)
	depth := 0
	var duplicates []dialectSourceToken
	for _, token := range tokens {
		switch token.raw {
		case "(":
			depth++
			continue
		case ")":
			delete(blocks, depth)
			if depth > 0 {
				depth--
			}
			continue
		case ";":
			clear(blocks)
			depth = 0
			continue
		}
		if token.isWord("select") {
			blocks[depth] = &queryBlock{}
			continue
		}
		if token.isWord("into") {
			if block := blocks[depth]; block != nil {
				block.intoCount++
				if block.intoCount > 1 {
					duplicates = append(duplicates, token)
				}
			}
		}
	}
	return duplicates
}

func isMySQLNumericIdentifier(token dialectSourceToken) bool {
	if token.kind != identifier || token.raw == "" || !isASCIIDigit(token.raw[0]) {
		return false
	}
	i := 0
	for i < len(token.raw) && isASCIIDigit(token.raw[i]) {
		i++
	}
	return i+1 < len(token.raw) && token.raw[i] == '.' && (token.raw[i+1] == 'e' || token.raw[i+1] == 'E')
}

func findCastJSONCandidates(tokens []dialectSourceToken) []dialectSourceToken {
	return findCastTargetCandidates(tokens, "json")
}

func findCastVarcharCandidates(tokens []dialectSourceToken) []dialectSourceToken {
	return findCastVarcharCandidatesForTarget(tokens, "varchar")
}

func findCastVarcharCandidatesForTarget(tokens []dialectSourceToken, target string) []dialectSourceToken {
	var candidates []dialectSourceToken
	for i := 0; i+2 < len(tokens); i++ {
		if tokens[i].isWord("cast") {
			candidates = append(candidates, findCastTargetCandidate(tokens, i, target)...)
			continue
		}
		if !tokens[i].isWord("convert") || !tokens[i+1].isRaw("(") {
			continue
		}
		close := matchingParen(tokens, i+1)
		if close < 0 {
			continue
		}
		depth := 0
		for j := i + 2; j < close; j++ {
			switch tokens[j].raw {
			case "(":
				depth++
			case ")":
				if depth > 0 {
					depth--
				}
			case ",":
				if depth == 0 && j+1 < close && tokens[j+1].isWord(target) {
					candidates = append(candidates, tokens[j+1])
					j = close
				}
			}
		}
	}
	return candidates
}

func findCastTargetCandidates(tokens []dialectSourceToken, target string) []dialectSourceToken {
	var candidates []dialectSourceToken
	for i := 0; i+2 < len(tokens); i++ {
		candidates = append(candidates, findCastTargetCandidate(tokens, i, target)...)
	}
	return candidates
}

func findCastTargetCandidate(tokens []dialectSourceToken, start int, target string) []dialectSourceToken {
	if start+2 >= len(tokens) || !tokens[start].isWord("cast") || !tokens[start+1].isRaw("(") {
		return nil
	}
	close := matchingParen(tokens, start+1)
	if close < 0 {
		return nil
	}
	depth := 0
	for i := start + 2; i < close; i++ {
		switch tokens[i].raw {
		case "(":
			depth++
		case ")":
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 && tokens[i].isWord("as") && i+1 < close && tokens[i+1].isWord(target) {
				return []dialectSourceToken{tokens[i+1]}
			}
		}
	}
	return nil
}

func findFunctionCandidates(tokens []dialectSourceToken, name string) []dialectFunctionCandidate {
	var candidates []dialectFunctionCandidate
	for i := 0; i+1 < len(tokens); i++ {
		if !strings.EqualFold(tokens[i].text, name) || !tokens[i+1].isRaw("(") {
			continue
		}
		builtin := tokens[i].kind == identifier && (i == 0 || !tokens[i-1].isRaw("."))
		candidates = append(candidates, dialectFunctionCandidate{token: tokens[i], builtin: builtin, index: i})
	}
	return candidates
}

func findMariaDBTypeModifierIssues(tokens []dialectSourceToken) []dialectSourceToken {
	var issues []dialectSourceToken
	for i := range tokens {
		if !isNumericType(tokens[i]) {
			continue
		}
		j := i + 1
		if j < len(tokens) && tokens[j].isWord("precision") && tokens[i].isWord("double") {
			j++
		}
		if j < len(tokens) && tokens[j].isRaw("(") {
			if close := matchingParen(tokens, j); close >= 0 {
				j = close + 1
			}
		}
		seen := make(map[string]bool)
		signedSeen := false
		modifierCount := 0
		for ; j < len(tokens); j++ {
			modifier := strings.ToLower(tokens[j].text)
			if tokens[j].kind != identifier || (modifier != "signed" && modifier != "unsigned" && modifier != "zerofill") {
				break
			}
			modifierCount++
			invalid := seen[modifier]
			if modifier == "signed" {
				invalid = invalid || modifierCount > 1
				signedSeen = true
			} else if signedSeen {
				invalid = true
			}
			seen[modifier] = true
			if invalid {
				issues = append(issues, tokens[j])
				break
			}
		}
	}
	return issues
}

func isNumericType(token dialectSourceToken) bool {
	if token.kind != identifier {
		return false
	}
	switch strings.ToLower(token.text) {
	case "tinyint", "smallint", "mediumint", "int", "integer", "bigint", "decimal", "dec", "numeric", "fixed", "float", "double", "real", "year":
		return true
	default:
		return false
	}
}

func findMariaDBSerialOptionIssues(tokens []dialectSourceToken) []dialectSourceToken {
	var issues []dialectSourceToken
	tableDDL := false
	for i, token := range tokens {
		if token.isRaw(";") {
			tableDDL = false
		}
		if token.isWord("create") || token.isWord("alter") {
			j := i + 1
			if j < len(tokens) && tokens[j].isWord("temporary") {
				j++
			}
			tableDDL = j < len(tokens) && tokens[j].isWord("table")
		}
		if !tableDDL || !token.isWord("serial") || i < 2 {
			continue
		}
		prefix := tokens[i-2]
		columnStart := prefix.isRaw("(") || prefix.isRaw(",") || prefix.isWord("add") ||
			prefix.isWord("modify") || prefix.isWord("column")
		changedColumn := i >= 3 && (tokens[i-3].isWord("change") || tokens[i-3].isWord("column"))
		if !columnStart && !changedColumn {
			continue
		}
		for j := i + 1; j < len(tokens); j++ {
			if tokens[j].isRaw(",") || tokens[j].isRaw(")") || tokens[j].isRaw(";") {
				break
			}
			if tokens[j].isRaw("(") {
				if close := matchingParen(tokens, j); close >= 0 {
					j = close
				}
				continue
			}
			nullable := tokens[j].isWord("null") && !tokens[j-1].isWord("not")
			if nullable || tokens[j].isWord("default") || tokens[j].isWord("auto_increment") {
				issues = append(issues, tokens[j])
				break
			}
		}
	}
	return issues
}

type dialectSourceFinding struct {
	token   dialectSourceToken
	code    DialectIssueCode
	message string
}

func findMySQLSetIssues(tokens []dialectSourceToken) []dialectSourceFinding {
	var findings []dialectSourceFinding
	for i := 0; i < len(tokens); i++ {
		if !tokens[i].isWord("set") {
			continue
		}
		end := i + 1
		for end < len(tokens) && !tokens[end].isRaw(";") {
			end++
		}
		if i+2 < end && tokens[i+1].kind == identifier && (tokens[i+2].isRaw("=") || tokens[i+2].isRaw(":=")) {
			switch strings.ToLower(tokens[i+1].text) {
			case "global", "local", "session", "persist", "persist_only":
				findings = append(findings, dialectSourceFinding{
					token: tokens[i+1], code: IssueMySQLBareSetScope,
					message: "MySQL requires a variable name after a SET scope",
				})
			}
		}
		for j := i + 1; j+3 < end; j++ {
			if tokens[j].isRaw("=") && tokens[j+1].isRaw(".") && tokens[j+2].kind == identifier && tokens[j+3].isRaw(".") {
				findings = append(findings, dialectSourceFinding{
					token: tokens[j+1], code: IssueMySQLLeadingDotSetValue,
					message: "MySQL does not accept a SET value beginning with a qualified-name dot",
				})
			}
		}
		i = end
	}
	return findings
}

func findInvalidMySQLODBCMarkers(tokens []dialectSourceToken) []dialectSourceToken {
	var markers []dialectSourceToken
	for i := 0; i+1 < len(tokens); i++ {
		if !tokens[i].isRaw("{") {
			continue
		}
		close := matchingBrace(tokens, i)
		if close < 0 || !containsTopLevelJoin(tokens, i+2, close) {
			continue
		}
		marker := tokens[i+1]
		if marker.kind != identifier || !strings.EqualFold(marker.text, "oj") {
			markers = append(markers, marker)
		}
		i = close
	}
	return markers
}

func matchingBrace(tokens []dialectSourceToken, open int) int {
	depth := 0
	for i := open; i < len(tokens); i++ {
		switch tokens[i].raw {
		case "{":
			depth++
		case "}":
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func containsTopLevelJoin(tokens []dialectSourceToken, start, end int) bool {
	parentheses := 0
	for i := start; i < end; i++ {
		switch tokens[i].raw {
		case "(":
			parentheses++
		case ")":
			if parentheses > 0 {
				parentheses--
			}
		default:
			if parentheses == 0 && tokens[i].isWord("join") {
				return true
			}
		}
	}
	return false
}

func findDuplicateColumnCollates(tokens []dialectSourceToken) []dialectSourceToken {
	var duplicates []dialectSourceToken
	for i := 0; i < len(tokens); i++ {
		if !tokens[i].isWord("create") {
			continue
		}
		j := i + 1
		if j < len(tokens) && tokens[j].isWord("temporary") {
			j++
		}
		if j >= len(tokens) || !tokens[j].isWord("table") {
			continue
		}
		for j++; j < len(tokens) && !tokens[j].isRaw("(") && !tokens[j].isRaw(";"); j++ {
			if tokens[j].isWord("like") || tokens[j].isWord("as") {
				break
			}
		}
		if j >= len(tokens) || !tokens[j].isRaw("(") {
			continue
		}
		close := matchingParen(tokens, j)
		if close < 0 {
			continue
		}
		segmentStart := j + 1
		depth := 0
		for k := j + 1; k <= close; k++ {
			if k < close {
				switch tokens[k].raw {
				case "(":
					depth++
				case ")":
					if depth > 0 {
						depth--
					}
				}
			}
			if k == close || (depth == 0 && tokens[k].isRaw(",")) {
				collates := 0
				generated := false
				localDepth := 0
				for p := segmentStart; p < k; p++ {
					switch tokens[p].raw {
					case "(":
						localDepth++
					case ")":
						if localDepth > 0 {
							localDepth--
						}
					default:
						if localDepth == 0 && tokens[p].isWord("generated") {
							generated = true
						}
						if localDepth == 0 && tokens[p].isWord("collate") {
							collates++
							if collates > 1 && !generated {
								duplicates = append(duplicates, tokens[p])
							}
						}
					}
				}
				segmentStart = k + 1
			}
		}
		i = close
	}
	return duplicates
}

func matchingParen(tokens []dialectSourceToken, open int) int {
	if open < 0 || open >= len(tokens) || !tokens[open].isRaw("(") {
		return -1
	}
	depth := 0
	for i := open; i < len(tokens); i++ {
		switch tokens[i].raw {
		case "(":
			depth++
		case ")":
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func isASCIIDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

// CheckDialectCompatibility walks node and returns all target-dialect issues.
func CheckDialectCompatibility(node ast.Node, source string, target Dialect) []DialectIssue {
	return checkDialectCompatibility(node, source, target, 0)
}

// CheckDialectCompatibility uses the parser's SQL mode when tokenizing source.
func (parser *Parser) CheckDialectCompatibility(node ast.Node, source string, target Dialect) []DialectIssue {
	return checkDialectCompatibility(node, source, target, parser.sqlModeValue())
}

func checkDialectCompatibility(node ast.Node, source string, target Dialect, mode mysql.SQLMode) []DialectIssue {
	if node == nil || target == DialectUnion {
		return nil
	}
	visitor := NewDialectCompatibilityVisitor(target, source)
	visitor.SetSQLMode(mode)
	_, _ = node.Accept(visitor)
	return visitor.Issues()
}

// ValidateDialectCompatibility performs strict target-dialect validation.
func ValidateDialectCompatibility(node ast.Node, source string, target Dialect) error {
	issues := CheckDialectCompatibility(node, source, target)
	if len(issues) == 0 {
		return nil
	}
	return &DialectCompatibilityError{Issues: issues}
}

// ParseOneStmtForDialect parses the shared grammar, then rejects constructs
// incompatible with target. ParseOneStmt remains the permissive default.
func (parser *Parser) ParseOneStmtForDialect(sql, charsetName, collation string, target Dialect) (ast.StmtNode, error) {
	stmt, err := parser.parseOneStmt(sql, charsetName, collation, target)
	if err != nil {
		return nil, err
	}
	issues := parser.CheckDialectCompatibility(stmt, sql, target)
	if len(issues) != 0 {
		return nil, &DialectCompatibilityError{Issues: issues}
	}
	return stmt, nil
}

var _ ast.Visitor = (*DialectCompatibilityVisitor)(nil)
var _ error = (*DialectCompatibilityError)(nil)
var _ error = DialectIssue{}

package parser

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	goatomic "sync/atomic"
	"unicode"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/myparser/errors"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/myparser/opcode"
)

const (
	codeErrParse                   = mysql.ErrCode(mysql.ErrParse)
	codeErrSyntax                  = mysql.ErrCode(mysql.ErrSyntax)
	codeErrUnknownCharacterSet     = mysql.ErrCode(mysql.ErrUnknownCharacterSet)
	codeErrInvalidYearColumnLength = mysql.ErrCode(mysql.ErrInvalidYearColumnLength)
	codeWrongArgument              = mysql.ErrCode(mysql.ErrWrongArguments)
	codeWrongFieldTerminators      = mysql.ErrCode(mysql.ErrWrongFieldTerminators)
	codeTooBigDisplayWidth         = mysql.ErrCode(mysql.ErrTooBigDisplaywidth)
)

var (
	// ErrSyntax returns for sql syntax error.
	ErrSyntax = mysql.ClassParser.New(codeErrSyntax, mysql.MySQLErrName[mysql.ErrSyntax])
	// ErrParse returns for sql parse error.
	ErrParse = mysql.ClassParser.New(codeErrParse, mysql.MySQLErrName[mysql.ErrParse])
	// ErrUnknownCharacterSet returns for no character set found error.
	ErrUnknownCharacterSet = mysql.ClassParser.New(codeErrUnknownCharacterSet, mysql.MySQLErrName[mysql.ErrUnknownCharacterSet])
	// ErrInvalidYearColumnLength returns for illegal column length for year type.
	ErrInvalidYearColumnLength = mysql.ClassParser.New(codeErrInvalidYearColumnLength, mysql.MySQLErrName[mysql.ErrInvalidYearColumnLength])
	// ErrWrongArguments returns for illegal argument.
	ErrWrongArguments = mysql.ClassParser.New(codeWrongArgument, mysql.MySQLErrName[mysql.ErrWrongArguments])
	// ErrWrongFieldTerminators returns for illegal field terminators.
	ErrWrongFieldTerminators = mysql.ClassParser.New(codeWrongFieldTerminators, mysql.MySQLErrName[mysql.ErrWrongFieldTerminators])
	// ErrTooBigDisplayWidth returns for data display width exceed limit .
	ErrTooBigDisplayWidth = mysql.ClassParser.New(codeTooBigDisplayWidth, mysql.MySQLErrName[mysql.ErrTooBigDisplaywidth])
	// SpecFieldPattern special result field pattern
	SpecFieldPattern = regexp.MustCompile(`(\/\*(?:!|M!)([0-9]{5,6})?|\*\/)`)
	specCodeStart    = regexp.MustCompile(`^\/\*(?:!|M!)([0-9]{5,6})?[ \t]*`)
	specCodeEnd      = regexp.MustCompile(`[ \t]*\*\/$`)
)

func init() {
	parserMySQLErrCodes := map[mysql.ErrCode]uint16{
		codeErrSyntax:                  mysql.ErrSyntax,
		codeErrParse:                   mysql.ErrParse,
		codeErrUnknownCharacterSet:     mysql.ErrUnknownCharacterSet,
		codeErrInvalidYearColumnLength: mysql.ErrInvalidYearColumnLength,
		codeWrongArgument:              mysql.ErrWrongArguments,
		codeWrongFieldTerminators:      mysql.ErrWrongFieldTerminators,
		codeTooBigDisplayWidth:         mysql.ErrTooBigDisplaywidth,
	}
	mysql.ErrClassToMySQLCodes[mysql.ClassParser] = parserMySQLErrCodes
}

// TrimComment trim comment for special comment code of MySQL.
func TrimComment(txt string) string {
	txt = specCodeStart.ReplaceAllString(txt, "")
	return specCodeEnd.ReplaceAllString(txt, "")
}

// Parser holds the settings used when parsing SQL.
//
// A Parser is safe for concurrent use: every Parse call runs on its own parse
// state, so one Parser can be shared by many goroutines. The settings are read
// atomically, so they may also be changed while parsing runs.
type Parser struct {
	// sqlMode must stay the first field: atomic 64-bit access needs the
	// 64-bit alignment the runtime only guarantees for the first word.
	sqlMode           int64
	supportWindowFunc uint32
}

// sqlModeValue returns the SQL mode of the parser.
func (parser *Parser) sqlModeValue() mysql.SQLMode {
	return mysql.SQLMode(goatomic.LoadInt64(&parser.sqlMode))
}

// supportWindowFuncValue reports whether the parser recognizes the keywords
// of window function.
func (parser *Parser) supportWindowFuncValue() bool {
	return goatomic.LoadUint32(&parser.supportWindowFunc) != 0
}

type stmtTexter interface {
	stmtText() string
}

type selectOptionState struct {
	options     *ast.SelectStmtOpts
	distinct    bool
	distinctSet bool
	prioritySet bool
}

func newSelectOptionState() *selectOptionState {
	return &selectOptionState{options: &ast.SelectStmtOpts{SQLCache: true}}
}

func applySelectOption(state *selectOptionState, option string) {
	switch option {
	case "ALL":
		state.distinct = false
		state.distinctSet = true
	case "DISTINCT":
		state.distinct = true
		state.distinctSet = true
	case "LOW_PRIORITY":
		state.options.Priority = mysql.LowPriority
		state.prioritySet = true
	case "HIGH_PRIORITY":
		state.options.Priority = mysql.HighPriority
		state.prioritySet = true
	case "DELAYED":
		state.options.Priority = mysql.DelayedPriority
		state.prioritySet = true
	case "SQL_SMALL_RESULT":
		state.options.SQLSmallResult = true
	case "SQL_BIG_RESULT":
		state.options.SQLBigResult = true
	case "SQL_BUFFER_RESULT":
		state.options.SQLBufferResult = true
	case "SQL_CALC_FOUND_ROWS":
		state.options.CalcFoundRows = true
	case "STRAIGHT_JOIN":
		state.options.StraightJoin = true
	}
}

func mergeSelectOptionStates(left, right *selectOptionState) *selectOptionState {
	if right.distinctSet {
		left.distinct = right.distinct
		left.distinctSet = true
	}
	if right.prioritySet {
		left.options.Priority = right.options.Priority
		left.prioritySet = true
	}
	left.options.SQLSmallResult = left.options.SQLSmallResult || right.options.SQLSmallResult
	left.options.SQLBigResult = left.options.SQLBigResult || right.options.SQLBigResult
	left.options.SQLBufferResult = left.options.SQLBufferResult || right.options.SQLBufferResult
	left.options.CalcFoundRows = left.options.CalcFoundRows || right.options.CalcFoundRows
	left.options.StraightJoin = left.options.StraightJoin || right.options.StraightJoin
	return left
}

// New returns a Parser object.
func New() *Parser {
	return &Parser{}
}

// Parse parses SQL and discards warnings. Use ParseWithWarnings to retrieve them.
func (parser *Parser) Parse(sql string) ([]ast.StmtNode, error) {
	stmt, _, err := parser.ParseWithWarnings(sql)
	return stmt, err
}

// ParseWithWarnings parses SQL and returns its warnings separately from errors.
func (parser *Parser) ParseWithWarnings(sql string) (stmt []ast.StmtNode, warns []error, err error) {
	return parser.parse(sql, DialectUnion)
}

func (parser *Parser) parse(sql string, target Dialect) (stmt []ast.StmtNode, warns []error, err error) {
	st := parseStatePool.Get().(*parseState)
	defer func() {
		// Drop references to the input SQL and parsed statements so the pool
		// does not keep them alive between calls.
		st.result = nil
		st.src = ""
		st.lexer.reset("")
		parseStatePool.Put(st)
	}()

	st.src = sql
	st.result = nil
	st.joinGroups = nil
	st.paramIndex = 0
	st.lexer.sqlMode = parser.sqlModeValue()
	st.lexer.target = target
	st.lexer.supportWindowFunc = parser.supportWindowFuncValue()
	st.lexer.reset(sql)
	st.lexer.parser = st
	yyParse(&st.lexer)
	st.joinGroups = nil

	warns, errs := st.lexer.Errors()
	if len(warns) > 0 {
		warns = append([]error(nil), warns...)
	} else {
		warns = nil
	}
	if len(errs) != 0 {
		return nil, warns, errors.Trace(errs[0])
	}
	for _, stmt := range st.result {
		if err := validateQueryBoundaries(stmt); err != nil {
			return nil, warns, errors.Trace(err)
		}
	}
	return st.result, warns, nil
}

func (st *parseState) lastErrorAsWarn() {
	if len(st.lexer.errs) == 0 {
		return
	}
	i := len(st.lexer.errs) - 1
	st.lexer.warns = append(st.lexer.warns, st.lexer.errs[i])
	st.lexer.errs[i] = nil
	st.lexer.errs = st.lexer.errs[:i]
}

// ParseOneStmt parses a query and returns an ast.StmtNode.
// The query must have one statement, otherwise ErrSyntax is returned.
func (parser *Parser) ParseOneStmt(sql string) (ast.StmtNode, error) {
	return parser.parseOneStmt(sql, DialectUnion)
}

func (parser *Parser) parseOneStmt(sql string, target Dialect) (ast.StmtNode, error) {
	stmts, _, err := parser.parse(sql, target)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if len(stmts) != 1 {
		return nil, ErrSyntax
	}
	return stmts[0], nil
}

// SetSQLMode sets the SQL mode for parser.
func (parser *Parser) SetSQLMode(mode mysql.SQLMode) {
	goatomic.StoreInt64(&parser.sqlMode, int64(mode))
}

// EnableWindowFunc controls whether the parser to parse syntax related with window function.
func (parser *Parser) EnableWindowFunc(val bool) {
	var v uint32
	if val {
		v = 1
	}
	goatomic.StoreUint32(&parser.supportWindowFunc, v)
}

// ParseErrorWith returns "You have a syntax error near..." error message compatible with mysql.
func ParseErrorWith(errstr string, lineno int) error {
	if len(errstr) > mysql.ErrTextLength {
		errstr = errstr[:mysql.ErrTextLength]
	}
	return fmt.Errorf("near '%-.80s' at line %d", errstr, lineno)
}

// The select statement is not at the end of the whole statement, if the last
// field text was set from its offset to the end of the src string, update
// the last field text.
func (st *parseState) setLastSelectFieldText(stmt *ast.SelectStmt, lastEnd int) {
	lastField := stmt.Fields.Items[len(stmt.Fields.Items)-1]
	start := int(lastField.Span().Start)
	if start+len(lastField.Text()) >= len(st.src)-1 {
		lastField.SetText(st.src[start:lastEnd])
	}
}

func (st *parseState) startOffset(v *yySymType) int {
	return v.offset
}

func (st *parseState) endOffset(v *yySymType) int {
	offset := v.offset
	for offset > 0 && unicode.IsSpace(rune(st.src[offset-1])) {
		offset--
	}
	return offset
}

// yyGetParser returns the parse state bound to the lexer currently driving
// yyParse. The generated parser (golang.org/x/tools/cmd/goyacc) hands actions
// only the yyLexer, so grammar actions resolve the owning state through it.
func yyGetParser(l yyLexer) *parseState {
	return l.(*Scanner).parser
}

func (st *parseState) newParamMarkerExpr(span loc.Span) *ast.ParamMarkerExpr {
	param := ast.NewParamMarkerExprAt(span, st.paramIndex)
	st.paramIndex++
	return param
}

// yyErrorf builds a scanner-located error, mirroring the yyLexer.Errorf that the
// previous generator exposed on the lexer interface.
func yyErrorf(l yyLexer, format string, a ...interface{}) error {
	return l.(*Scanner).Errorf(format, a...)
}

// yyAppendErr records an error on the scanner, mirroring the yyLexer.AppendError
// that the previous generator exposed on the lexer interface.
func yyAppendErr(l yyLexer, err error) {
	l.(*Scanner).AppendError(err)
}

// isBareSetScopeKeyword reports whether name is a SET scope keyword (GLOBAL, LOCAL,
// SESSION, PERSIST, PERSIST_ONLY) usable only as a scope prefix in SET, not as a
// variable name: "SET GLOBAL = DEFAULT" has no variable and is rejected by both
// MySQL and MariaDB.
func isBareSetScopeKeyword(name string) bool {
	switch strings.ToUpper(name) {
	case "GLOBAL", "LOCAL", "SESSION", "PERSIST", "PERSIST_ONLY":
		return true
	}
	return false
}

func sequenceSignedInt(l yyLexer, value interface{}, negative bool, span loc.Span) (ast.ExprNode, error) {
	var magnitude uint64
	switch v := value.(type) {
	case int64:
		if v < 0 {
			return nil, errors.New("expected an unsigned integer literal")
		}
		magnitude = uint64(v)
	case uint64:
		magnitude = v
	default:
		return nil, errors.New("expected an integer sequence value")
	}
	limit := uint64(math.MaxInt64)
	if negative {
		limit++
	}
	if magnitude > limit {
		return nil, errors.New("signed sequence integer out of range")
	}
	if negative {
		if magnitude == uint64(math.MaxInt64)+1 {
			return syntaxLiteral(l, pkast.LiteralInteger, strconv.FormatInt(math.MinInt64, 10), span), nil
		}
		return syntaxLiteral(l, pkast.LiteralInteger, strconv.FormatInt(-int64(magnitude), 10), span), nil
	}
	return syntaxLiteral(l, pkast.LiteralInteger, strconv.FormatUint(magnitude, 10), span), nil
}

func makeSequenceOption(tp ast.SequenceOptionType, value ast.ExprNode) *ast.SequenceOption {
	option := &ast.SequenceOption{Tp: tp, Value: value}
	negative := false
	if unary, ok := value.(*ast.UnaryOperationExpr); ok && (unary.Op == pkast.Operator(opcode.Plus) || unary.Op == pkast.Operator(opcode.Minus)) {
		negative = unary.Op == pkast.Operator(opcode.Minus)
		value = unary.Expr
	}
	if literal, ok := value.(*ast.LiteralExpr); ok && literal.Kind == pkast.LiteralInteger {
		if v, err := strconv.ParseInt(literal.Value, 10, 64); err == nil {
			option.Val = v
		} else if negative && literal.Value == "9223372036854775808" {
			option.Val = math.MinInt64
			return option
		}
	}
	if negative {
		option.Val = -option.Val
	}
	return option
}

// explainIntoStmt builds the INTO clause of MySQL's EXPLAIN ... INTO @var.
func explainIntoStmt(target *ast.VariableTarget, span loc.Span) *ast.SelectIntoStatement {
	stmt := &ast.SelectIntoStatement{
		Tp:      ast.SelectIntoVarList,
		Targets: []*ast.VariableTarget{target},
	}
	stmt.SetSpan(span)
	return stmt
}

func syntaxVariableTarget(l yyLexer, name string, span loc.Span, kind ast.VariableTargetKind, prefix ast.VariablePrefix) *ast.VariableTarget {
	target := &ast.VariableTarget{
		Name:   syntaxName(syntaxIdent(l, name, span)),
		Kind:   kind,
		Prefix: prefix,
	}
	target.SetSpan(span)
	return target
}

func syntaxQualifiedVariableTarget(l yyLexer, left, right string, span, rightSpan loc.Span, kind ast.VariableTargetKind) *ast.VariableTarget {
	target := &ast.VariableTarget{
		Name: syntaxName(
			syntaxIdent(l, left, loc.Span{Start: span.Start, End: rightSpan.Start - 1}),
			syntaxIdent(l, right, rightSpan),
		),
		Kind: kind,
	}
	target.SetSpan(span)
	return target
}

func syntaxAtVariableTarget(l yyLexer, raw string, span loc.Span) *ast.VariableTarget {
	name := strings.TrimPrefix(raw, "@")
	nameSpan := span
	if span.Valid() {
		nameSpan.Start++
		source := yyGetParser(l).src[span.Start:span.End]
		if len(source) >= 3 && (source[1] == '`' || source[1] == '\'' || source[1] == '"') && source[len(source)-1] == source[1] {
			quote := source[1]
			name = strings.ReplaceAll(source[2:len(source)-1], string([]byte{quote, quote}), string(quote))
		}
	}
	target := syntaxVariableTarget(l, name, nameSpan, ast.VariableTargetUser, ast.VariablePrefixAt)
	target.SetSpan(span)
	return target
}

func syntaxSystemVariableTarget(l yyLexer, raw string, span loc.Span) *ast.VariableTarget {
	lower := strings.ToLower(raw)
	name := strings.TrimPrefix(raw, "@@")
	target := &ast.VariableTarget{Kind: ast.VariableTargetSystem, Prefix: ast.VariablePrefixDoubleAt}
	target.ScopeExplicit = true
	for _, scope := range []struct {
		prefix string
		scope  ast.VariableScope
	}{
		{prefix: "persist_only.", scope: ast.VariableScopePersistOnly},
		{prefix: "persist.", scope: ast.VariableScopePersist},
		{prefix: "global.", scope: ast.VariableScopeGlobal},
		{prefix: "session.", scope: ast.VariableScopeSession},
		{prefix: "local.", scope: ast.VariableScopeLocal},
	} {
		if strings.HasPrefix(lower, "@@"+scope.prefix) {
			name = raw[len("@@")+len(scope.prefix):]
			target.Scope = scope.scope
			break
		}
	}
	if target.Scope == ast.VariableScopeImplicit {
		target.ScopeExplicit = false
	}
	name = strings.ToLower(name)
	nameStart := span.Start + 2
	if span.Valid() && span.Within(len(yyGetParser(l).src)) {
		// Recover the exact prefix from the source so quoted identifiers retain Raw.
		source := yyGetParser(l).src[span.Start:span.End]
		for _, prefix := range []string{"persist_only.", "persist.", "global.", "session.", "local."} {
			if len(source) >= 2+len(prefix) && strings.EqualFold(source[2:2+len(prefix)], prefix) {
				nameStart = span.Start + loc.ByteOffset(2+len(prefix))
				break
			}
		}
	}
	nameSpan := loc.Span{Start: nameStart, End: span.End}
	target.Name = syntaxName(syntaxIdent(l, name, nameSpan))
	target.SetSpan(span)
	return target
}

// validateFunctionCall applies argument rules that both pinned servers
// enforce. Other function calls remain generic.
func validateSourceFunctionCall(l yyLexer, offset int, name string, args []ast.ExprNode) error {
	source := yyGetParser(l).src
	if offset >= 0 && offset < len(source) && (source[offset] == '`' || source[offset] == '"') {
		return nil
	}
	return validateFunctionCall(name, args)
}

func temporalCastType(tp byte, precision *pkast.BaseLiteralExpr) *ast.TypeName {
	x := newSyntaxType(tp)
	x.Modifiers = typeModifiers(precision)
	return x
}

// reservedRoleNames are the MySQL keywords and labels that must be quoted when
// they are used as role names.
var reservedRoleNames = map[string]struct{}{
	"EVENT":       {},
	"EXECUTE":     {},
	"FILE":        {},
	"NONE":        {},
	"PROCESS":     {},
	"PROXY":       {},
	"RELOAD":      {},
	"REPLICATION": {},
	"RESOURCE":    {},
	"RESTART":     {},
	"SHUTDOWN":    {},
	"SUPER":       {},
}

// isReservedRoleName reports whether an unquoted identifier spells a role name
// that MySQL requires to be quoted.
func isReservedRoleName(src string, offset int, name string) bool {
	if _, ok := reservedRoleNames[strings.ToUpper(name)]; !ok {
		return false
	}
	return offset < 0 || offset >= len(src) || (src[offset] != '`' && src[offset] != '"')
}

func validateFunctionCall(name string, args []ast.ExprNode) error {
	switch strings.ToUpper(name) {
	case "LOG":
		if len(args) == 0 || len(args) > 2 {
			return fmt.Errorf("LOG requires one or two arguments")
		}
	case "NOW":
		if len(args) != 1 {
			return nil
		}
		if literal, ok := args[0].(*ast.LiteralExpr); ok && literal.Kind == pkast.LiteralBoolean {
			return nil
		}
		if literal, ok := args[0].(*ast.LiteralExpr); ok && literal.Kind == pkast.LiteralInteger {
			if _, err := strconv.ParseUint(literal.Value, 10, 64); err == nil {
				return nil
			}
		}
		return fmt.Errorf("NOW precision must be a non-negative integer")
	case "JSON_ARRAYAGG":
		if len(args) != 1 {
			return fmt.Errorf("JSON_ARRAYAGG requires one argument")
		}
	case "JSON_OBJECTAGG":
		if len(args) != 2 {
			return fmt.Errorf("JSON_OBJECTAGG requires two arguments")
		}
	}
	return nil
}

// setSelectInto stores the structured INTO clause and keeps the legacy
// string-only variable list available for existing AST consumers.
func setSelectInto(st *ast.SelectStmt, in interface{}) error {
	if in == nil {
		return nil
	}
	if st.Into != nil {
		return fmt.Errorf("multiple INTO clauses in one query block")
	}
	x := in.(*ast.SelectIntoStatement)
	st.Into = x
	return nil
}

// validateQueryBoundaries enforces the INTO and set-operation boundaries that
// both MySQL and MariaDB reject at parse time. It runs after the grammar has
// produced the AST so that the checks can see the complete query shape.
func validateQueryBoundaries(stmt ast.StmtNode) error {
	v := &queryBoundaryValidator{}
	stmt.Accept(v)
	return v.err
}

func newKillStmt(query bool, expr ast.ExprNode) *ast.KillStmt {
	stmt := &ast.KillStmt{Query: query, Expr: expr}
	if literal, ok := expr.(*ast.LiteralExpr); ok && literal.Kind == pkast.LiteralBoolean {
		if literal.Value == "true" {
			stmt.ConnectionID = 1
		}
		stmt.Expr = nil
	}
	if literal, ok := expr.(*ast.LiteralExpr); ok && literal.Kind == pkast.LiteralInteger {
		if id, err := strconv.ParseUint(literal.Value, 10, 64); err == nil {
			stmt.ConnectionID, stmt.Expr = id, nil
		}
	}
	return stmt
}

type queryBoundaryValidator struct {
	err error
}

func (v *queryBoundaryValidator) Enter(n ast.Node) (ast.Node, bool) {
	if v.err != nil {
		return n, true
	}
	switch x := n.(type) {
	case *ast.FuncCastExpr:
		if x.Type != nil && x.Type.Keyword() == "JSON" {
			expr := x.Expr
			for {
				paren, ok := expr.(*ast.ParenthesesExpr)
				if !ok {
					break
				}
				expr = paren.Expr
			}
			// Empty JSON text is left to runtime validation.
			if text, ok := literalStringValue(expr); ok && text != "" && !json.Valid([]byte(text)) {
				v.err = fmt.Errorf("invalid JSON text in argument 1 to function cast_as_json")
			}
		}
	case *ast.SetOprStmt:
		v.err = validateSetOprBoundaries(x)
		if v.err != nil {
			return n, true
		}
	case *ast.QueryTailStmt:
		if inner, ok := x.Query.(*ast.QueryTailStmt); ok && inner.Into != nil &&
			(x.Into != nil || x.Tail.OrderBy != nil || x.Tail.Limit != nil ||
				x.LockTp != ast.SelectLockNone || x.ProcedureAnalyse != nil) {
			v.err = fmt.Errorf("a query tail cannot follow a parenthesized INTO clause")
			return n, true
		}
	case *ast.SubqueryExpr:
		if queryHasTrailingInto(x.Query) {
			v.err = fmt.Errorf("INTO is not allowed inside a subquery")
			return n, true
		}
	case ast.TableSourceNode:
		if core := x.TableSourceCore(); core != nil && queryHasTrailingInto(core.Source) {
			v.err = fmt.Errorf("INTO is not allowed in a derived table")
			return n, true
		}
	case *ast.CommonTableExpression:
		if queryHasTrailingInto(x.Query) {
			v.err = fmt.Errorf("INTO is not allowed in a common table expression")
			return n, true
		}
	}
	return n, false
}

func (v *queryBoundaryValidator) Leave(n ast.Node) (ast.Node, bool) {
	return n, true
}

// validateSetOprBoundaries checks a set operation's operands. A bare SELECT
// followed by another set operator must not carry ORDER BY, LIMIT, a locking
// clause, or INTO; those clauses belong after the last operand. A
// parenthesized operand may carry its own ORDER BY and LIMIT. MySQL also
// accepts SELECT ... INTO ... FROM in the final parenthesized operand, while
// both servers reject a query-tail INTO there; BeforeFrom distinguishes them.
func validateSetOprBoundaries(stmt *ast.SetOprStmt) error {
	if stmt == nil {
		return nil
	}
	if stmt.Left == nil || stmt.Right == nil {
		return fmt.Errorf("set operation contains a nil operand")
	}
	if err := validateSetOperand(stmt.Left, false); err != nil {
		return err
	}
	if err := validateSetOperand(stmt.Right, true); err != nil {
		return err
	}
	return nil
}

func validateSetOperand(operand ast.QueryNode, last bool) error {
	switch x := operand.(type) {
	case *ast.SelectStmt:
		if !last && (x.OrderBy != nil || x.Limit != nil || x.LockTp != ast.SelectLockNone || queryHasTrailingInto(x)) {
			return fmt.Errorf("ORDER BY, LIMIT, locking clauses, and INTO must follow the last set operation operand")
		}
	case *ast.QueryTailStmt:
		if queryHasTrailingInto(x) {
			return fmt.Errorf("INTO is not allowed in a parenthesized set operation operand")
		}
	case *ast.SetOprStmt:
		// Nested operations are validated when the visitor reaches them.
		if x.Into != nil {
			return fmt.Errorf("INTO cannot be followed by another query tail or set operation")
		}
	}
	return nil
}

func queryHasTrailingInto(node ast.Node) bool {
	switch x := node.(type) {
	case *ast.SelectStmt:
		return x.Into != nil && !x.Into.BeforeFrom
	case *ast.SetOprStmt:
		return x.Into != nil || queryHasTrailingInto(x.Left) || queryHasTrailingInto(x.Right)
	case *ast.QueryTailStmt:
		return x.Into != nil || queryHasTrailingInto(x.Query)
	}
	return false
}

// attachProcedureAnalyse attaches MariaDB's PROCEDURE ANALYSE query tail only
// after a complete top-level query. Keeping this out of SelectStmt grammar is
// important: the clause is not valid in subqueries or CREATE TABLE ... SELECT.
func attachProcedureAnalyse(stmt ast.StmtNode, clause *ast.ProcedureAnalyseClause, lock interface{}) error {
	applyLock := func(lockTp *ast.SelectLockType, tables *[]*ast.TableName, waitPolicy *ast.SelectLockWaitPolicy) {
		if lock == nil {
			return
		}
		info := lock.(*ast.SelectLockInfo)
		*lockTp = info.Type
		*tables = info.Tables
		*waitPolicy = info.WaitPolicy
	}

	switch x := stmt.(type) {
	case *ast.SelectStmt:
		if x.ProcedureAnalyse != nil {
			return fmt.Errorf("duplicate PROCEDURE ANALYSE clause")
		}
		if x.Into != nil {
			return fmt.Errorf("PROCEDURE ANALYSE cannot be combined with INTO")
		}
		if x.LockTp != ast.SelectLockNone {
			return fmt.Errorf("a locking clause must follow PROCEDURE ANALYSE")
		}
		x.ProcedureAnalyse = clause
		applyLock(&x.LockTp, &x.LockedTables, &x.LockWaitPolicy)
		return nil
	case *ast.SetOprStmt:
		if x.ProcedureAnalyse != nil {
			return fmt.Errorf("duplicate PROCEDURE ANALYSE clause")
		}
		if x.Into != nil {
			return fmt.Errorf("PROCEDURE ANALYSE cannot be combined with INTO")
		}
		if x.LockTp != ast.SelectLockNone {
			return fmt.Errorf("a locking clause must follow PROCEDURE ANALYSE")
		}
		if trailing := trailingSetOprSelect(x); trailing != nil {
			if trailing.Into != nil {
				return fmt.Errorf("PROCEDURE ANALYSE cannot be combined with INTO")
			}
			if trailing.LockTp != ast.SelectLockNone {
				return fmt.Errorf("a locking clause must follow PROCEDURE ANALYSE")
			}
		}
		x.ProcedureAnalyse = clause
		applyLock(&x.LockTp, &x.LockedTables, &x.LockWaitPolicy)
		return nil
	default:
		return fmt.Errorf("PROCEDURE ANALYSE requires a SELECT query")
	}
}

func trailingSetOprSelect(stmt *ast.SetOprStmt) *ast.SelectStmt {
	if stmt == nil {
		return nil
	}
	return trailingQuerySelect(stmt.Right)
}

func trailingQuerySelect(node ast.QueryNode) *ast.SelectStmt {
	switch x := node.(type) {
	case *ast.SelectStmt:
		return x
	case *ast.SetOprStmt:
		return trailingQuerySelect(x.Right)
	case *ast.QueryTailStmt:
		return trailingQuerySelect(x.Query)
	default:
		return nil
	}
}

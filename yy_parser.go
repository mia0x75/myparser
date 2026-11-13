package myparser

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	goatomic "sync/atomic"
	"unicode"

	"github.com/mia0x75/myparser/errors"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/charset"
	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/myparser/opcode"
	"github.com/mia0x75/myparser/sqlerr"
	"github.com/mia0x75/myparser/types"
)

const (
	codeErrParse                   = sqlerr.ErrCode(mysql.ErrParse)
	codeErrSyntax                  = sqlerr.ErrCode(mysql.ErrSyntax)
	codeErrUnknownCharacterSet     = sqlerr.ErrCode(mysql.ErrUnknownCharacterSet)
	codeErrInvalidYearColumnLength = sqlerr.ErrCode(mysql.ErrInvalidYearColumnLength)
	codeWrongArgument              = sqlerr.ErrCode(mysql.ErrWrongArguments)
	codeWrongFieldTerminators      = sqlerr.ErrCode(mysql.ErrWrongFieldTerminators)
	codeTooBigDisplayWidth         = sqlerr.ErrCode(mysql.ErrTooBigDisplaywidth)
)

var (
	// ErrSyntax returns for sql syntax error.
	ErrSyntax = sqlerr.ClassParser.New(codeErrSyntax, mysql.MySQLErrName[mysql.ErrSyntax])
	// ErrParse returns for sql parse error.
	ErrParse = sqlerr.ClassParser.New(codeErrParse, mysql.MySQLErrName[mysql.ErrParse])
	// ErrUnknownCharacterSet returns for no character set found error.
	ErrUnknownCharacterSet = sqlerr.ClassParser.New(codeErrUnknownCharacterSet, mysql.MySQLErrName[mysql.ErrUnknownCharacterSet])
	// ErrInvalidYearColumnLength returns for illegal column length for year type.
	ErrInvalidYearColumnLength = sqlerr.ClassParser.New(codeErrInvalidYearColumnLength, mysql.MySQLErrName[mysql.ErrInvalidYearColumnLength])
	// ErrWrongArguments returns for illegal argument.
	ErrWrongArguments = sqlerr.ClassParser.New(codeWrongArgument, mysql.MySQLErrName[mysql.ErrWrongArguments])
	// ErrWrongFieldTerminators returns for illegal field terminators.
	ErrWrongFieldTerminators = sqlerr.ClassParser.New(codeWrongFieldTerminators, mysql.MySQLErrName[mysql.ErrWrongFieldTerminators])
	// ErrTooBigDisplayWidth returns for data display width exceed limit .
	ErrTooBigDisplayWidth = sqlerr.ClassParser.New(codeTooBigDisplayWidth, mysql.MySQLErrName[mysql.ErrTooBigDisplaywidth])
	// SpecFieldPattern special result field pattern
	SpecFieldPattern = regexp.MustCompile(`(\/\*(?:!|M!)([0-9]{5,6})?|\*\/)`)
	specCodeStart    = regexp.MustCompile(`^\/\*(?:!|M!)([0-9]{5,6})?[ \t]*`)
	specCodeEnd      = regexp.MustCompile(`[ \t]*\*\/$`)
)

func init() {
	parserMySQLErrCodes := map[sqlerr.ErrCode]uint16{
		codeErrSyntax:                  mysql.ErrSyntax,
		codeErrParse:                   mysql.ErrParse,
		codeErrUnknownCharacterSet:     mysql.ErrUnknownCharacterSet,
		codeErrInvalidYearColumnLength: mysql.ErrInvalidYearColumnLength,
		codeWrongArgument:              mysql.ErrWrongArguments,
		codeWrongFieldTerminators:      mysql.ErrWrongFieldTerminators,
		codeTooBigDisplayWidth:         mysql.ErrTooBigDisplaywidth,
	}
	sqlerr.ErrClassToMySQLCodes[sqlerr.ClassParser] = parserMySQLErrCodes
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

// parseState holds the mutable state of one parse run. Parse takes a state
// from a pool so that the scanner buffers are reused between calls.
type parseState struct {
	result     []ast.StmtNode
	src        string
	lexer      Scanner
	joinGroups map[*ast.Join]bool
	// charset and collation are the connection settings applied to plain
	// string literals, resolved once per Parse call.
	charset   string
	collation string
}

var parseStatePool = sync.Pool{
	New: func() interface{} {
		return &parseState{}
	},
}

type stmtTexter interface {
	stmtText() string
}

type selectOptionState struct {
	options     *ast.SelectStmtOpts
	distinctSet bool
	prioritySet bool
}

func newSelectOptionState() *selectOptionState {
	return &selectOptionState{options: &ast.SelectStmtOpts{SQLCache: true}}
}

func applySelectOption(state *selectOptionState, option string) {
	switch option {
	case "ALL":
		state.options.Distinct = false
		state.distinctSet = true
	case "DISTINCT":
		state.options.Distinct = true
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
		left.options.Distinct = right.options.Distinct
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
	if ast.NewValueExpr == nil ||
		ast.NewParamMarkerExpr == nil ||
		ast.NewHexLiteral == nil ||
		ast.NewBitLiteral == nil {
		panic("no parser driver")
	}

	return &Parser{}
}

// Parse parses a query string to raw ast.StmtNode.
// The charset and collation arguments are the connection settings applied to
// string literals. An empty argument selects the MySQL default.
func (parser *Parser) Parse(sql, charset, collation string) (stmt []ast.StmtNode, warns []error, err error) {
	return parser.parse(sql, charset, collation, DialectUnion)
}

// resolveLiteralCharset fills in the connection charset and collation that are
// applied to plain string literals.
func resolveLiteralCharset(charsetName, collation string) (string, string) {
	if charsetName == "" {
		charsetName = mysql.DefaultCharset
	}
	if collation == "" {
		// An unknown charset keeps the default collation instead of failing
		// the parse: callers may pass metadata such as "default".
		if co, err := charset.GetDefaultCollation(charsetName); err == nil {
			collation = co
		} else {
			collation = mysql.DefaultCollationName
		}
	}
	return charsetName, collation
}

// parse parses the query for the given target dialect.
func (parser *Parser) parse(sql, charset, collation string, target Dialect) (stmt []ast.StmtNode, warns []error, err error) {
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
	st.charset, st.collation = resolveLiteralCharset(charset, collation)
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
	for _, stmt := range st.result {
		ast.SetFlag(stmt)
	}
	return st.result, warns, nil
}

func (st *parseState) lastErrorAsWarn() {
	if len(st.lexer.errs) == 0 {
		return
	}
	st.lexer.warns = append(st.lexer.warns, st.lexer.errs[len(st.lexer.errs)-1])
	st.lexer.errs = st.lexer.errs[:len(st.lexer.errs)-1]
}

// ParseOneStmt parses a query and returns an ast.StmtNode.
// The query must have one statement, otherwise ErrSyntax is returned.
func (parser *Parser) ParseOneStmt(sql, charset, collation string) (ast.StmtNode, error) {
	return parser.parseOneStmt(sql, charset, collation, DialectUnion)
}

// parseOneStmt parses the query for the given target dialect and returns the
// single statement it produced.
func (parser *Parser) parseOneStmt(sql, charset, collation string, target Dialect) (ast.StmtNode, error) {
	stmts, _, err := parser.parse(sql, charset, collation, target)
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
	lastField := stmt.Fields.Fields[len(stmt.Fields.Fields)-1]
	if lastField.Offset+len(lastField.Text()) >= len(st.src)-1 {
		lastField.SetText(st.src[lastField.Offset:lastEnd])
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

func toInt(l *Scanner, lval *yySymType, str string) int {
	n, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		e := err.(*strconv.NumError)
		if e.Err == strconv.ErrRange {
			// Fall through to decimal: MySQL truncates out-of-range integer
			// literals to the max decimal value and emits a warning.
			return toDecimal(l, lval, str)
		}
		l.AppendError(l.Errorf("integer literal: %v", err))
		return int(unicode.ReplacementChar)
	}

	switch {
	case n <= math.MaxInt64:
		lval.item = int64(n)
	default:
		lval.item = n
	}
	return intLit
}

func toDecimal(l *Scanner, lval *yySymType, str string) int {
	dec, err := ast.NewDecimal(str)
	if err != nil {
		if sqlErr, ok := err.(*mysql.SQLError); ok && sqlErr.Code == mysql.ErrTruncatedWrongValue {
			l.warns = append(l.warns, err)
		} else {
			l.AppendError(l.Errorf("decimal literal: %v", err))
			return int(unicode.ReplacementChar)
		}
	}
	lval.item = dec
	return decLit
}

func toFloat(l *Scanner, lval *yySymType, str string) int {
	n, err := strconv.ParseFloat(str, 64)
	if err != nil {
		l.AppendError(mysql.NewErr(mysql.ErrIllegalValueForType, "double", str))
		return int(unicode.ReplacementChar)
	}

	lval.item = n
	return floatLit
}

// See https://dev.mysql.com/doc/refman/5.7/en/hexadecimal-literals.html
func toHex(l *Scanner, lval *yySymType, str string) int {
	h, err := ast.NewHexLiteral(str)
	if err != nil {
		l.AppendError(l.Errorf("hex literal: %v", err))
		return int(unicode.ReplacementChar)
	}
	lval.item = h
	return hexLit
}

// See https://dev.mysql.com/doc/refman/5.7/en/bit-type.html
func toBit(l *Scanner, lval *yySymType, str string) int {
	b, err := ast.NewBitLiteral(str)
	if err != nil {
		l.AppendError(l.Errorf("bit literal: %v", err))
		return int(unicode.ReplacementChar)
	}
	lval.item = b
	return bitLit
}

func getUint64FromNUM(num interface{}) uint64 {
	switch v := num.(type) {
	case int64:
		return uint64(v)
	case uint64:
		return v
	}
	return 0
}

func hexInteger(text string, allowQuoted bool) (uint64, error) {
	var digits string
	switch {
	case len(text) >= 3 && text[0] == '0' && text[1] == 'x':
		digits = text[2:]
	case allowQuoted && len(text) >= 3 && (text[0] == 'x' || text[0] == 'X') && text[1] == '\'' && text[len(text)-1] == '\'':
		digits = text[2 : len(text)-1]
	default:
		return 0, errors.New("expected a numeric 0x hexadecimal literal")
	}
	if digits == "" {
		return 0, nil
	}
	value, err := strconv.ParseUint(digits, 16, 64)
	if err != nil {
		return 0, errors.Annotate(err, "hexadecimal integer out of range")
	}
	return value, nil
}

func sequenceSignedInt(value interface{}, negative bool) (ast.ExprNode, error) {
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
			return ast.NewValueExpr(int64(math.MinInt64)), nil
		}
		return ast.NewValueExpr(-int64(magnitude)), nil
	}
	return ast.NewValueExpr(int64(magnitude)), nil
}

func makeSequenceOption(tp ast.SequenceOptionType, value ast.ExprNode) *ast.SequenceOption {
	option := &ast.SequenceOption{Tp: tp, Value: value}
	negative := false
	if unary, ok := value.(*ast.UnaryOperationExpr); ok && (unary.Op == opcode.Plus || unary.Op == opcode.Minus) {
		negative = unary.Op == opcode.Minus
		value = unary.V
	}
	if literal, ok := value.(ast.ValueExpr); ok {
		switch v := literal.GetValue().(type) {
		case int64:
			option.Val = v
		case uint64:
			if v <= math.MaxInt64 {
				option.Val = int64(v)
			} else if negative && v == uint64(math.MaxInt64)+1 {
				option.Val = math.MinInt64
				return option
			}
		}
	}
	if negative {
		option.Val = -option.Val
	}
	return option
}

func variableTargetNames(targets []*ast.VariableTarget) []string {
	names := make([]string, len(targets))
	for i, target := range targets {
		names[i] = target.LegacyName()
	}
	return names
}

// explainIntoStmt builds the INTO clause of MySQL's EXPLAIN ... INTO @var.
func explainIntoStmt(target *ast.VariableTarget) *ast.SelectIntoStatement {
	return &ast.SelectIntoStatement{
		Tp:      ast.SelectIntoVarList,
		VarList: variableTargetNames([]*ast.VariableTarget{target}),
		Targets: []*ast.VariableTarget{target},
	}
}

func normalizeSetOprPrecedence(stmt *ast.SetOprStmt) {
	if stmt == nil || stmt.SelectList == nil {
		return
	}
	stmt.SelectList.Selects = normalizeSetOprSelects(stmt.SelectList.Selects)
}

func normalizeSetOprSelects(selects []ast.Node) []ast.Node {
	if len(selects) < 2 {
		return selects
	}
	type setOprRun struct {
		nodes []ast.Node
		op    *ast.SetOprType
	}
	var runs []setOprRun
	current := setOprRun{nodes: []ast.Node{selects[0]}}
	for _, node := range selects[1:] {
		op := nodeSetOpr(node)
		if op != nil && (*op == ast.Intersect || *op == ast.IntersectAll) {
			current.nodes = append(current.nodes, node)
			continue
		}
		runs = append(runs, current)
		current = setOprRun{nodes: []ast.Node{node}, op: op}
	}
	runs = append(runs, current)
	if len(runs) == 1 {
		return selects
	}
	out := make([]ast.Node, 0, len(runs))
	for _, run := range runs {
		if len(run.nodes) == 1 {
			setNodeSetOpr(run.nodes[0], run.op)
			out = append(out, run.nodes[0])
			continue
		}
		setNodeSetOpr(run.nodes[0], nil)
		out = append(out, &ast.SetOprSelectList{
			AfterSetOperator: run.op,
			Selects:          run.nodes,
		})
	}
	return out
}

func nodeSetOpr(n ast.Node) *ast.SetOprType {
	switch x := n.(type) {
	case *ast.SelectStmt:
		return x.AfterSetOperator
	case *ast.ValuesStmt:
		return x.AfterSetOperator
	case *ast.SetOprSelectList:
		return x.AfterSetOperator
	}
	return nil
}

func setNodeSetOpr(n ast.Node, op *ast.SetOprType) {
	switch x := n.(type) {
	case *ast.SelectStmt:
		x.AfterSetOperator = op
	case *ast.ValuesStmt:
		x.AfterSetOperator = op
	case *ast.SetOprSelectList:
		x.AfterSetOperator = op
	}
}

type parenQueryTailShape struct {
	OrderBy *ast.OrderByClause
	Limit   *ast.Limit
	Lock    *ast.SelectLockInfo
	Into    *ast.SelectIntoStatement
}

// Each tail keeps its query level so an outer LIMIT never replaces an inner one.
func parenQueryTail(query ast.ResultSetNode, tail parenQueryTailShape) ast.ResultSetNode {
	out := &ast.SetOprStmt{
		SelectList: &ast.SetOprSelectList{Selects: []ast.Node{
			setOprSelectListFromSubSelect(&ast.SubqueryExpr{Query: query}),
		}},
		OrderBy: tail.OrderBy,
		Limit:   tail.Limit,
		Into:    tail.Into,
	}
	if tail.Lock != nil {
		out.LockTp = tail.Lock.Type
		out.LockedTables = tail.Lock.Tables
		out.LockWaitPolicy = tail.Lock.WaitPolicy
	}
	return out
}

// addCrossJoin inserts a conditionless join at the left-most leaf of the
// right join tree, without crossing explicit parentheses.
func (st *parseState) addCrossJoin(left, right ast.ResultSetNode) ast.ResultSetNode {
	if j, ok := right.(*ast.Join); ok && !st.joinGroups[j] {
		j.Left = st.addCrossJoin(left, j.Left)
		return j
	}
	return &ast.Join{Left: left, Right: right, Tp: ast.CrossJoin}
}

// withClauseSelect attaches the common table expression clause to the head
// query of a WITH statement without duplicating the clause on a nested level.
func withClauseSelect(clause *ast.WithClause, query ast.ResultSetNode) ast.ResultSetNode {
	switch target := query.(type) {
	case *ast.SelectStmt:
		if target.With == nil {
			target.With = clause
			return target
		}
	case *ast.SetOprStmt:
		if target.With == nil {
			target.With = clause
			return target
		}
	}
	outer := parenQueryTail(query, parenQueryTailShape{}).(*ast.SetOprStmt)
	outer.With = clause
	return outer
}

// setOprSelectListFromSubSelect wraps a parenthesized subquery that appears as
// an operand of a set operation (UNION/EXCEPT/INTERSECT) into a SetOprSelectList,
// preserving its own limit/order by/with clause.
func setOprSelectListFromSubSelect(sub *ast.SubqueryExpr) *ast.SetOprSelectList {
	switch x := sub.Query.(type) {
	case *ast.SelectStmt:
		return &ast.SetOprSelectList{Selects: []ast.Node{x}}
	case *ast.SetOprStmt:
		return setOprSelectListFromSetOprStmt(x)
	}
	return &ast.SetOprSelectList{}
}

func setOprSelectListFromSetOprStmt(stmt *ast.SetOprStmt) *ast.SetOprSelectList {
	// Flatten only tails the list can represent; keep other tails at their query level.
	if stmt.Into != nil || stmt.LockTp != ast.SelectLockNone || stmt.ProcedureAnalyse != nil {
		return &ast.SetOprSelectList{Selects: []ast.Node{stmt}}
	}
	return &ast.SetOprSelectList{
		Selects: stmt.SelectList.Selects,
		With:    stmt.With,
		Limit:   stmt.Limit,
		OrderBy: stmt.OrderBy,
	}
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

func temporalCastType(tp byte, precision string) *types.FieldType {
	x := types.NewFieldType(tp)
	x.Flen, x.Decimal = mysql.GetDefaultFieldLengthAndDecimalForCast(tp)
	x.CastPrecision = precision
	if precision != "" {
		// CastPrecision is authoritative when the value cannot fit the legacy int fields.
		if value, err := strconv.Atoi(precision); err == nil && value >= 0 && value <= math.MaxInt-20 {
			x.Decimal = value
			if value > 0 {
				x.Flen += 1 + value
			}
		}
	}
	x.Charset, x.Collate = charset.CharsetBin, charset.CollationBin
	x.Flag |= mysql.BinaryFlag
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
		value, ok := args[0].(ast.ValueExpr)
		if !ok {
			return fmt.Errorf("NOW precision must be a non-negative integer")
		}
		switch v := value.GetValue().(type) {
		case int64:
			if v < 0 {
				return fmt.Errorf("NOW precision must be a non-negative integer")
			}
		case uint64:
		default:
			return fmt.Errorf("NOW precision must be a non-negative integer")
		}
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
	if x.Tp == ast.SelectIntoVarList {
		st.IntoVars = x.VarList
	}
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
	if value, ok := expr.(ast.ValueExpr); ok {
		switch id := value.GetValue().(type) {
		case int64:
			stmt.ConnectionID, stmt.Expr = uint64(id), nil
		case uint64:
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
		if x.Tp != nil && x.Tp.Tp == mysql.TypeJSON {
			expr := x.Expr
			for {
				paren, ok := expr.(*ast.ParenthesesExpr)
				if !ok {
					break
				}
				expr = paren.Expr
			}
			if value, ok := expr.(ast.ValueExpr); ok {
				// Empty JSON text is left to runtime validation.
				if text, ok := value.GetValue().(string); ok && text != "" && !json.Valid([]byte(text)) {
					v.err = fmt.Errorf("invalid JSON text in argument 1 to function cast_as_json")
				}
			}
		}
	case *ast.SetOprStmt:
		v.err = validateSetOprBoundaries(x.SelectList)
		if v.err != nil {
			return n, true
		}
	case *ast.SubqueryExpr:
		if queryHasTrailingInto(x.Query) {
			v.err = fmt.Errorf("INTO is not allowed inside a subquery")
			return n, true
		}
	case *ast.TableSource:
		if queryHasTrailingInto(x.Source) {
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
func validateSetOprBoundaries(list *ast.SetOprSelectList) error {
	if list == nil {
		return nil
	}
	for i, operand := range list.Selects {
		last := i == len(list.Selects)-1
		switch x := operand.(type) {
		case *ast.SelectStmt:
			if !last && (x.OrderBy != nil || x.Limit != nil || x.LockTp != ast.SelectLockNone || queryHasTrailingInto(x)) {
				return fmt.Errorf("ORDER BY, LIMIT, locking clauses, and INTO must follow the last set operation operand")
			}
		case *ast.SetOprSelectList:
			if queryHasTrailingInto(x) {
				return fmt.Errorf("INTO is not allowed in a parenthesized set operation operand")
			}
			if err := validateSetOprBoundaries(x); err != nil {
				return err
			}
		case *ast.SetOprStmt:
			if x.Into != nil {
				return fmt.Errorf("INTO cannot be followed by another query tail or set operation")
			}
		}
	}
	return nil
}

func queryHasTrailingInto(node ast.Node) bool {
	switch x := node.(type) {
	case *ast.SelectStmt:
		return x.Into != nil && !x.Into.BeforeFrom
	case *ast.SetOprStmt:
		return x.Into != nil || queryHasTrailingInto(x.SelectList)
	case *ast.SetOprSelectList:
		if x == nil || len(x.Selects) == 0 {
			return false
		}
		return queryHasTrailingInto(x.Selects[len(x.Selects)-1])
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
		if x.Into != nil || len(x.IntoVars) > 0 {
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
		if trailing := trailingSetOprSelect(x.SelectList); trailing != nil {
			if trailing.Into != nil || len(trailing.IntoVars) > 0 {
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

func trailingSetOprSelect(list *ast.SetOprSelectList) *ast.SelectStmt {
	if list == nil || len(list.Selects) == 0 {
		return nil
	}
	switch x := list.Selects[len(list.Selects)-1].(type) {
	case *ast.SelectStmt:
		return x
	case *ast.SetOprSelectList:
		return trailingSetOprSelect(x)
	case *ast.SetOprStmt:
		return trailingSetOprSelect(x.SelectList)
	default:
		return nil
	}
}

package parser

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/opcode"
	"github.com/mia0x75/myparser/types"
)

func newReplicationOption(name lex.Ident, value ast.ExprNode) *ast.ReplicationOption {
	return &ast.ReplicationOption{
		Name: name,
		Expr: value,
	}
}

func newReplicaControlStmt(start, replica, all bool, suffix *ast.ReplicaSuffix) *ast.ReplicaControlStmt {
	return &ast.ReplicaControlStmt{
		Start:              start,
		Replica:            replica,
		All:                all,
		Threads:            suffix.Threads,
		UntilOptions:       suffix.UntilOptions,
		UntilAfterGaps:     suffix.UntilAfterGaps,
		ConnectionOptions:  suffix.ConnectionOptions,
		ConnectionName:     suffix.ConnectionName,
		HasConnectionName:  suffix.HasConnectionName,
		Channel:            suffix.Channel,
		HasChannel:         suffix.HasChannel,
		ChannelBeforeUntil: suffix.ChannelBeforeUntil,
	}
}

type replicationDialect uint8

const (
	replicationMySQL replicationDialect = 1 << iota
	replicationMariaDB
	replicationBoth = replicationMySQL | replicationMariaDB
)

// MySQL 8.0 accepts MASTER_* and SOURCE_* aliases with either CHANGE spelling.
// MariaDB-only options and DEFAULT/boolean values must still form a MariaDB statement.
// https://dev.mysql.com/doc/refman/8.0/en/change-master-to.html
// https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to
func replicationOptionName(option *ast.ReplicationOption) string {
	if !option.Name.Present() {
		return ""
	}
	return strings.ToUpper(option.Name.Name)
}

func replicationOptionDialects(option *ast.ReplicationOption) replicationDialect {
	name := replicationOptionName(option)
	dialects := replicationBoth
	if strings.HasPrefix(name, "SOURCE_") {
		dialects = replicationMySQL
	} else {
		switch name {
		case "MASTER_USE_GTID", "MASTER_DEMOTE_TO_SLAVE", "MASTER_DEMOTE_TO_REPLICA",
			"DO_DOMAIN_IDS", "IGNORE_DOMAIN_IDS", "MASTER_GTID_POS":
			dialects = replicationMariaDB
		case "MASTER_BIND", "MASTER_TLS_VERSION", "MASTER_TLS_CIPHERSUITES",
			"MASTER_PUBLIC_KEY_PATH", "GET_MASTER_PUBLIC_KEY", "GET_SOURCE_PUBLIC_KEY",
			"MASTER_AUTO_POSITION", "MASTER_COMPRESSION_ALGORITHMS", "MASTER_ZSTD_COMPRESSION_LEVEL",
			"NETWORK_NAMESPACE", "PRIVILEGE_CHECKS_USER", "REQUIRE_ROW_FORMAT",
			"REQUIRE_TABLE_PRIMARY_KEY_CHECK", "ASSIGN_GTIDS_TO_ANONYMOUS_TRANSACTIONS", "GTID_ONLY":
			dialects = replicationMySQL
		}
	}
	if replicationMariaDBOnlyValue(option.Expr) {
		dialects &= replicationMariaDB
	}
	if name == "MASTER_RETRY_COUNT" && strings.HasPrefix(strings.ToLower(option.Expr.Text()), "0x") {
		// MariaDB uses ulonglong_num here, which does not accept HEX_NUM.
		dialects &= replicationMySQL
	}
	if str, ok := literalStringValue(option.Expr); ok {
		if strings.Contains(str, "\n") && name != "MASTER_COMPRESSION_ALGORITHMS" && name != "SOURCE_COMPRESSION_ALGORITHMS" && name != "SQL_BEFORE_GTIDS" && name != "SQL_AFTER_GTIDS" {
			// MySQL's TEXT_STRING_sys_nonewline excludes linefeed characters.
			dialects &= replicationMariaDB
		}
		if (name == "MASTER_PASSWORD" || name == "SOURCE_PASSWORD") && len(str) > 32 {
			dialects &= replicationMariaDB
		}
	}
	if name == "MASTER_HEARTBEAT_PERIOD" || name == "SOURCE_HEARTBEAT_PERIOD" {
		dialects &= replicationHeartbeatDialects(option.Expr)
	}
	return dialects
}

func replicationMariaDBOnlyValue(value ast.ExprNode) bool {
	switch value := value.(type) {
	case *ast.DefaultExpr:
		return true
	case *ast.UnaryOperationExpr:
		return value.Op == pkast.Operator(opcode.Plus)
	case *ast.LiteralExpr:
		return value.Kind == pkast.LiteralBoolean
	case *ast.ReplicationOptionValue:
		if slices.ContainsFunc(value.Values, replicationMariaDBOnlyValue) {
			return true
		}
	}
	return false
}

func validateChangeMaster(stmt *ast.ChangeMasterStmt) error {
	if stmt.HasConnectionName && stmt.HasChannel {
		return fmt.Errorf("connection name and FOR CHANNEL cannot be specified together")
	}
	dialects := replicationBoth
	if stmt.Replica {
		dialects &= replicationMySQL
	}
	if stmt.HasConnectionName {
		dialects &= replicationMariaDB
	}
	if strings.Contains(stmt.Channel, "\n") {
		dialects &= replicationMariaDB
	}
	useGTID := false
	for _, option := range stmt.Options {
		dialects &= replicationOptionDialects(option)
		if replicationOptionName(option) == "MASTER_DELAY" || replicationOptionName(option) == "SOURCE_DELAY" {
			value := option.Expr
			if unary, ok := value.(*ast.UnaryOperationExpr); ok {
				value = unary.Expr
			}
			if replicationUint(value) > 1<<31-1 {
				return fmt.Errorf("%s exceeds the maximum delay of 2147483647", replicationOptionName(option))
			}
		}
		if replicationOptionName(option) == "MASTER_USE_GTID" {
			if useGTID {
				return fmt.Errorf("duplicate MASTER_USE_GTID option")
			}
			useGTID = true
		}
	}
	if dialects == 0 {
		return fmt.Errorf("replication options do not form a MySQL or MariaDB CHANGE statement")
	}
	return nil
}

func replicaUntilDialects(suffix *ast.ReplicaSuffix) replicationDialect {
	if suffix.UntilAfterGaps {
		return replicationMySQL
	}
	dialects := replicationBoth
	for _, option := range suffix.UntilOptions {
		dialects &= replicationOptionDialects(option)
	}
	return dialects
}

// UNTIL selects exactly one stopping condition; file and position are required together.
// https://dev.mysql.com/doc/refman/8.4/en/start-replica.html
// https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/start-replica
func validateReplicaUntilFiles(options []*ast.ReplicationOption) error {
	var coordinates uint8
	for _, option := range options {
		switch replicationOptionName(option) {
		case "MASTER_LOG_FILE", "SOURCE_LOG_FILE":
			coordinates |= 1
		case "MASTER_LOG_POS", "SOURCE_LOG_POS":
			coordinates |= 2
		case "RELAY_LOG_FILE":
			coordinates |= 4
		case "RELAY_LOG_POS":
			coordinates |= 8
		}
	}
	if coordinates != 3 && coordinates != 12 {
		return fmt.Errorf("UNTIL requires one complete source or relay log file/position pair")
	}
	return nil
}

func validateReplicaStart(suffix *ast.ReplicaSuffix) error {
	if suffix.HasConnectionName && suffix.HasChannel {
		return fmt.Errorf("connection name and FOR CHANNEL cannot be specified together")
	}
	if len(suffix.ConnectionOptions) != 0 {
		var ioThread, sqlThread bool
		for _, thread := range suffix.Threads {
			ioThread = ioThread || thread == "IO_THREAD"
			sqlThread = sqlThread || thread == "SQL_THREAD"
		}
		if sqlThread && !ioThread {
			return fmt.Errorf("START SQL_THREAD alone does not allow connection options")
		}
	}
	dialects := replicaUntilDialects(suffix)
	if suffix.HasConnectionName || suffix.ChannelBeforeUntil {
		dialects &= replicationMariaDB
	}
	if len(suffix.ConnectionOptions) != 0 {
		dialects &= replicationMySQL
	}
	if suffix.HasChannel && !suffix.ChannelBeforeUntil && (len(suffix.UntilOptions) != 0 || suffix.UntilAfterGaps) {
		dialects &= replicationMySQL
	}
	if strings.Contains(suffix.Channel, "\n") {
		dialects &= replicationMariaDB
	}
	if dialects == 0 {
		return fmt.Errorf("replication clauses do not form a MySQL or MariaDB START statement")
	}
	return nil
}

func replicationBinaryValue(value ast.ExprNode) bool {
	if unary, ok := value.(*ast.UnaryOperationExpr); ok {
		return unary.Op == pkast.Operator(opcode.Plus) && replicationBinaryValue(unary.Expr)
	}
	literal, ok := value.(*ast.LiteralExpr)
	if !ok {
		return false
	}
	switch literal.Kind {
	case pkast.LiteralInteger:
		n, err := strconv.ParseUint(literal.Value, 10, 64)
		return err == nil && n <= 1
	case pkast.LiteralFloat:
		n, err := strconv.ParseFloat(literal.Value, 64)
		return err == nil && (n == 0 || n == 1)
	case pkast.LiteralDecimal:
		var n types.MyDecimal
		if err := n.FromString([]byte(literal.Value)); err != nil {
			return false
		}
		return n.Compare(types.NewDecFromInt(0)) == 0 || n.Compare(types.NewDecFromInt(1)) == 0
	}
	return false
}

func replicationUint(value ast.ExprNode) uint64 {
	literal, ok := value.(*ast.LiteralExpr)
	if !ok || literal.Kind != pkast.LiteralInteger {
		return 0
	}
	n, _ := strconv.ParseUint(literal.Value, 10, 64)
	return n
}

// The official ulong_num/ulonglong_num actions read the integer prefix and saturate
// at uint64 on overflow, rather than evaluating the fraction or exponent.
// https://github.com/mysql/mysql-server/blob/8.4/sql/sql_yacc.yy
// https://github.com/MariaDB/server/blob/12.3/sql/sql_yacc.yy
func replicationUnsignedNumber(text string, span loc.Span) ast.ExprNode {
	number, _ := replicationUnsignedPrefix(text) // The saturated value is intentional.
	value := replicationIntegerValue(number)
	value.SetText(text)
	value.Raw = text
	value.SetSpan(span)
	return value
}

func replicationUnsignedToken(value *yySymType, token int, text string) int {
	number, overflow := replicationUnsignedPrefix(text)
	if number <= 1<<63-1 {
		value.item = int64(number)
	} else {
		value.item = number
	}
	if token == intLit && overflow {
		// Oversized integer tokens are DECIMAL_NUM, not real_ulong_num integers.
		return decLit
	}
	return token
}

func replicationUnsignedPrefix(text string) (number uint64, overflow bool) {
	end := 0
	for end < len(text) && isDigit(rune(text[end])) {
		end++
	}
	if end != 0 {
		value, err := strconv.ParseUint(text[:end], 10, 64)
		if err != nil {
			// The lexer already validated the numeric token; only overflow remains.
			number = 1<<64 - 1
			overflow = true
		} else {
			number = value
		}
	}
	return number, overflow
}

func replicationIntegerValue(number uint64) *ast.LiteralExpr {
	return &ast.LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: strconv.FormatUint(number, 10)}}
}

func replicationHeartbeatDialects(value ast.ExprNode) replicationDialect {
	if _, ok := value.(*ast.DefaultExpr); ok {
		return replicationMariaDB
	}
	if unary, ok := value.(*ast.UnaryOperationExpr); ok {
		value = unary.Expr
	}
	literal, ok := value.(*ast.LiteralExpr)
	if !ok {
		return 0
	}
	dialects := replicationBoth
	var number float64
	var native interface{}
	switch literal.Kind {
	case pkast.LiteralInteger, pkast.LiteralFloat:
		converted, err := strconv.ParseFloat(literal.Value, 64)
		if err != nil {
			return 0
		}
		native = converted
	case pkast.LiteralDecimal:
		decimal := new(types.MyDecimal)
		if err := decimal.FromString([]byte(literal.Value)); err != nil {
			return 0
		}
		native = decimal
	default:
		return 0
	}
	switch value := native.(type) {
	case float64:
		number = value
	case *types.MyDecimal:
		converted, err := value.ToFloat64()
		if err != nil {
			return 0
		}
		number = converted
		// MariaDB compares the original decimal, not its rounded double value.
		// https://github.com/MariaDB/server/blob/12.3/sql/rpl_master_info_file.h
		maximum := types.NewDecFromUint(1<<32 - 1)
		if err := maximum.Shift(-3); err != nil {
			return 0
		}
		if value.Compare(maximum) > 0 {
			dialects &^= replicationMariaDB
		}
	}
	if number < 0 || float32(number) > 4294967 {
		dialects &^= replicationMySQL
	}
	if number < 0 || number > 4294967.295 {
		dialects &^= replicationMariaDB
	}
	return dialects
}

func replicationHexNumber(value interface{}, text string, span loc.Span) (ast.ExprNode, error) {
	if !strings.HasPrefix(strings.ToLower(text), "0x") {
		return nil, fmt.Errorf("replication numbers require 0x hexadecimal notation, not X'...' string literals")
	}
	var bytes []byte
	switch value := value.(type) {
	case types.HexLiteral:
		bytes = []byte(value)
	case types.BinaryLiteral:
		bytes = []byte(value)
	default:
		return nil, fmt.Errorf("invalid hexadecimal replication number")
	}
	var number uint64
	for _, b := range bytes {
		if number > (1<<64-1)>>8 {
			return nil, fmt.Errorf("hexadecimal replication number exceeds uint64")
		}
		number = number<<8 | uint64(b)
	}
	numeric := replicationIntegerValue(number)
	numeric.SetText(text)
	numeric.Raw = text
	numeric.SetSpan(span)
	return numeric, nil
}

func validReplicationUUID(uuid string) bool {
	if len(uuid) != 36 {
		return false
	}
	for i := range len(uuid) {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if uuid[i] != '-' {
				return false
			}
			continue
		}
		b := uuid[i]
		if isDigit(rune(b)) || b >= 'a' && b <= 'f' || b >= 'A' && b <= 'F' {
			continue
		}
		return false
	}
	return true
}

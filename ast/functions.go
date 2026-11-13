package ast

import (
	"fmt"
	"io"

	"github.com/mia0x75/myparser/errors"

	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/model"
	"github.com/mia0x75/myparser/types"
)

var (
	_ FuncNode = &AggregateFuncExpr{}
	_ FuncNode = &FuncCallExpr{}
	_ FuncNode = &FuncCastExpr{}
	_ FuncNode = &WeightStringExpr{}
	_ FuncNode = &WindowFuncExpr{}
)

// List scalar function names.
const (
	LogicAnd   = "and"
	Cast       = "cast"
	LeftShift  = "leftshift"
	RightShift = "rightshift"
	LogicOr    = "or"
	GE         = "ge"
	LE         = "le"
	EQ         = "eq"
	NE         = "ne"
	LT         = "lt"
	GT         = "gt"
	Plus       = "plus"
	Minus      = "minus"
	And        = "bitand"
	Or         = "bitor"
	Mod        = "mod"
	Xor        = "bitxor"
	Div        = "div"
	Mul        = "mul"
	UnaryNot   = "not"
	BitNeg     = "bitneg"
	IntDiv     = "intdiv"
	LogicXor   = "xor"
	NullEQ     = "nulleq"
	UnaryPlus  = "unaryplus"
	UnaryMinus = "unaryminus"
	In         = "in"
	Like       = "like"
	Case       = "case"
	Regexp     = "regexp"
	IsNull     = "isnull"
	IsTruth    = "istrue"
	IsFalsity  = "isfalse"
	RowFunc    = "row"
	SetVar     = "setvar"
	GetVar     = "getvar"
	Values     = "values"
	BitCount   = "bit_count"
	GetParam   = "getparam"

	// common functions
	Coalesce = "coalesce"
	Greatest = "greatest"
	Least    = "least"
	Interval = "interval"

	// math functions
	Abs      = "abs"
	Acos     = "acos"
	Asin     = "asin"
	Atan     = "atan"
	Atan2    = "atan2"
	Ceil     = "ceil"
	Ceiling  = "ceiling"
	Conv     = "conv"
	Cos      = "cos"
	Cot      = "cot"
	CRC32    = "crc32"
	Degrees  = "degrees"
	Exp      = "exp"
	Floor    = "floor"
	Ln       = "ln"
	Log      = "log"
	Log2     = "log2"
	Log10    = "log10"
	PI       = "pi"
	Pow      = "pow"
	Power    = "power"
	Radians  = "radians"
	Rand     = "rand"
	Round    = "round"
	Sign     = "sign"
	Sin      = "sin"
	Sqrt     = "sqrt"
	Tan      = "tan"
	Truncate = "truncate"

	// time functions
	AddDate          = "adddate"
	AddTime          = "addtime"
	ConvertTz        = "convert_tz"
	Curdate          = "curdate"
	CurrentDate      = "current_date"
	CurrentTime      = "current_time"
	CurrentTimestamp = "current_timestamp"
	Curtime          = "curtime"
	Date             = "date"
	DateLiteral      = "dateliteral"
	DateAdd          = "date_add"
	DateFormat       = "date_format"
	DateSub          = "date_sub"
	DateDiff         = "datediff"
	Day              = "day"
	DayName          = "dayname"
	DayOfMonth       = "dayofmonth"
	DayOfWeek        = "dayofweek"
	DayOfYear        = "dayofyear"
	Extract          = "extract"
	FromDays         = "from_days"
	FromUnixTime     = "from_unixtime"
	GetFormat        = "get_format"
	Hour             = "hour"
	LocalTime        = "localtime"
	LocalTimestamp   = "localtimestamp"
	MakeDate         = "makedate"
	MakeTime         = "maketime"
	MicroSecond      = "microsecond"
	Minute           = "minute"
	Month            = "month"
	MonthName        = "monthname"
	Now              = "now"
	PeriodAdd        = "period_add"
	PeriodDiff       = "period_diff"
	Quarter          = "quarter"
	SecToTime        = "sec_to_time"
	Second           = "second"
	StrToDate        = "str_to_date"
	SubDate          = "subdate"
	SubTime          = "subtime"
	Sysdate          = "sysdate"
	Time             = "time"
	TimeLiteral      = "timeliteral"
	TimeFormat       = "time_format"
	TimeToSec        = "time_to_sec"
	TimeDiff         = "timediff"
	Timestamp        = "timestamp"
	TimestampLiteral = "timestampliteral"
	TimestampAdd     = "timestampadd"
	TimestampDiff    = "timestampdiff"
	ToDays           = "to_days"
	ToSeconds        = "to_seconds"
	UnixTimestamp    = "unix_timestamp"
	UTCDate          = "utc_date"
	UTCTime          = "utc_time"
	UTCTimestamp     = "utc_timestamp"
	Week             = "week"
	Weekday          = "weekday"
	WeekOfYear       = "weekofyear"
	Year             = "year"
	YearWeek         = "yearweek"
	LastDay          = "last_day"

	// string functions
	ASCII           = "ascii"
	Bin             = "bin"
	Concat          = "concat"
	ConcatWS        = "concat_ws"
	Convert         = "convert"
	Elt             = "elt"
	ExportSet       = "export_set"
	Field           = "field"
	Format          = "format"
	FromBase64      = "from_base64"
	InsertFunc      = "insert_func"
	Instr           = "instr"
	Lcase           = "lcase"
	Left            = "left"
	Length          = "length"
	LoadFile        = "load_file"
	Locate          = "locate"
	Lower           = "lower"
	Lpad            = "lpad"
	LTrim           = "ltrim"
	MakeSet         = "make_set"
	Mid             = "mid"
	Oct             = "oct"
	Ord             = "ord"
	Position        = "position"
	Quote           = "quote"
	Repeat          = "repeat"
	Replace         = "replace"
	Reverse         = "reverse"
	Right           = "right"
	RTrim           = "rtrim"
	Space           = "space"
	Strcmp          = "strcmp"
	Substring       = "substring"
	Substr          = "substr"
	SubstringIndex  = "substring_index"
	ToBase64        = "to_base64"
	Trim            = "trim"
	Upper           = "upper"
	Ucase           = "ucase"
	Hex             = "hex"
	Unhex           = "unhex"
	Rpad            = "rpad"
	BitLength       = "bit_length"
	CharFunc        = "char_func"
	CharLength      = "char_length"
	CharacterLength = "character_length"
	FindInSet       = "find_in_set"

	// information functions
	Benchmark    = "benchmark"
	Charset      = "charset"
	Coercibility = "coercibility"
	Collation    = "collation"
	ConnectionID = "connection_id"
	CurrentUser  = "current_user"
	CurrentRole  = "current_role"
	Database     = "database"
	FoundRows    = "found_rows"
	LastInsertId = "last_insert_id"
	RowCount     = "row_count"
	Schema       = "schema"
	SessionUser  = "session_user"
	SystemUser   = "system_user"
	User         = "user"
	Version      = "version"

	// control functions
	If     = "if"
	Ifnull = "ifnull"
	Nullif = "nullif"

	// miscellaneous functions
	AnyValue        = "any_value"
	DefaultFunc     = "default_func"
	InetAton        = "inet_aton"
	InetNtoa        = "inet_ntoa"
	Inet6Aton       = "inet6_aton"
	Inet6Ntoa       = "inet6_ntoa"
	IsFreeLock      = "is_free_lock"
	IsIPv4          = "is_ipv4"
	IsIPv4Compat    = "is_ipv4_compat"
	IsIPv4Mapped    = "is_ipv4_mapped"
	IsIPv6          = "is_ipv6"
	IsUsedLock      = "is_used_lock"
	MasterPosWait   = "master_pos_wait"
	NameConst       = "name_const"
	ReleaseAllLocks = "release_all_locks"
	Sleep           = "sleep"
	UUID            = "uuid"
	UUIDShort       = "uuid_short"
	// get_lock() and release_lock() is parsed but do nothing.
	// It is used for preventing error in Ruby's activerecord migrations.
	GetLock     = "get_lock"
	ReleaseLock = "release_lock"

	// encryption and compression functions
	AesDecrypt               = "aes_decrypt"
	AesEncrypt               = "aes_encrypt"
	Compress                 = "compress"
	Decode                   = "decode"
	DesDecrypt               = "des_decrypt"
	DesEncrypt               = "des_encrypt"
	Encode                   = "encode"
	Encrypt                  = "encrypt"
	MD5                      = "md5"
	OldPassword              = "old_password"
	PasswordFunc             = "password_func"
	RandomBytes              = "random_bytes"
	SHA1                     = "sha1"
	SHA                      = "sha"
	SHA2                     = "sha2"
	Uncompress               = "uncompress"
	UncompressedLength       = "uncompressed_length"
	ValidatePasswordStrength = "validate_password_strength"

	// json functions
	JSONType          = "json_type"
	JSONExtract       = "json_extract"
	JSONUnquote       = "json_unquote"
	JSONArray         = "json_array"
	JSONObject        = "json_object"
	JSONMerge         = "json_merge"
	JSONSet           = "json_set"
	JSONInsert        = "json_insert"
	JSONReplace       = "json_replace"
	JSONRemove        = "json_remove"
	JSONContains      = "json_contains"
	JSONContainsPath  = "json_contains_path"
	JSONValid         = "json_valid"
	JSONArrayAppend   = "json_array_append"
	JSONArrayInsert   = "json_array_insert"
	JSONMergePatch    = "json_merge_patch"
	JSONMergePreserve = "json_merge_preserve"
	JSONPretty        = "json_pretty"
	JSONQuote         = "json_quote"
	JSONSearch        = "json_search"
	JSONStorageSize   = "json_storage_size"
	JSONDepth         = "json_depth"
	JSONKeys          = "json_keys"
	JSONLength        = "json_length"
)

// FuncCallExpr is for function expression.
type FuncCallExpr struct {
	funcNode
	// FnName is the function name.
	FnName model.CIStr
	// Args is the function args.
	Args []ExprNode
	// IntervalFirst preserves leading INTERVAL, whose unary-minus semantics differ by dialect.
	IntervalFirst bool
}

// Restore implements Node interface.
func (n *FuncCallExpr) Restore(ctx *format.RestoreCtx) error {
	if n.IntervalFirst {
		ctx.WriteKeyWord("INTERVAL ")
		if err := n.Args[1].Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore FuncCallExpr interval")
		}
		ctx.WritePlain(" ")
		ctx.WriteKeyWord(n.Args[2].(ValueExpr).GetString())
		ctx.WritePlain(" + ")
		return n.Args[0].Restore(ctx)
	}
	if isBuiltinFunc(n.FnName.L) {
		ctx.WriteKeyWord(n.FnName.O)
	} else {
		ctx.WritePlain(n.FnName.O)
	}
	ctx.WritePlain("(")
	switch n.FnName.L {
	case "convert":
		if err := n.Args[0].Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore FuncCastExpr.Expr")
		}
		ctx.WriteKeyWord(" USING ")
		ctx.WriteKeyWord(n.Args[1].GetType().Charset)
	case "adddate", "subdate", "date_add", "date_sub":
		if err := n.Args[0].Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore FuncCallExpr")
		}
		ctx.WritePlain(", ")
		ctx.WriteKeyWord("INTERVAL ")
		if err := n.Args[1].Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore FuncCallExpr")
		}
		ctx.WritePlain(" ")
		ctx.WriteKeyWord(n.Args[2].(ValueExpr).GetString())
	case "extract":
		ctx.WriteKeyWord(n.Args[0].(ValueExpr).GetString())
		ctx.WriteKeyWord(" FROM ")
		if err := n.Args[1].Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore FuncCallExpr")
		}
	case "get_format":
		ctx.WriteKeyWord(n.Args[0].(ValueExpr).GetString())
		ctx.WritePlain(", ")
		if err := n.Args[1].Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore FuncCallExpr")
		}
	case "position":
		if err := n.Args[0].Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore FuncCallExpr")
		}
		ctx.WriteKeyWord(" IN ")
		if err := n.Args[1].Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore FuncCallExpr")
		}
	case "trim":
		switch len(n.Args) {
		case 1:
			if err := n.Args[0].Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore FuncCallExpr")
			}
		case 2:
			if err := n.Args[1].Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore FuncCallExpr")
			}
			ctx.WriteKeyWord(" FROM ")
			if err := n.Args[0].Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore FuncCallExpr")
			}
		case 3:
			switch fmt.Sprint(n.Args[2].(ValueExpr).GetValue()) {
			case "3":
				ctx.WriteKeyWord("TRAILING ")
			case "2":
				ctx.WriteKeyWord("LEADING ")
			case "0", "1":
				ctx.WriteKeyWord("BOTH ")
			}
			if n.Args[1].(ValueExpr).GetValue() != nil {
				if err := n.Args[1].Restore(ctx); err != nil {
					return errors.Annotatef(err, "An error occurred while restore FuncCallExpr")
				}
				ctx.WritePlain(" ")
			}
			ctx.WriteKeyWord("FROM ")
			if err := n.Args[0].Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore FuncCallExpr")
			}
		}
	case "timestampdiff", "timestampadd":
		ctx.WriteKeyWord(n.Args[0].(ValueExpr).GetString())
		for i := 1; i < len(n.Args); {
			ctx.WritePlain(", ")
			if err := n.Args[i].Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore FuncCallExpr")
			}
			i++
		}
	default:
		for i, argv := range n.Args {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if err := argv.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore FuncCallExpr.Args %d", i)
			}
		}
	}
	ctx.WritePlain(")")
	return nil
}

// Format the ExprNode into a Writer.
func (n *FuncCallExpr) Format(w io.Writer) {
	if n.IntervalFirst {
		_, _ = fmt.Fprint(w, "INTERVAL ")
		n.Args[1].Format(w)
		_, _ = fmt.Fprintf(w, " %s + ", n.Args[2].(ValueExpr).GetDatumString())
		n.Args[0].Format(w)
		return
	}
	_, _ = fmt.Fprintf(w, "%s(", n.FnName.L)
	if !n.specialFormatArgs(w) {
		for i, arg := range n.Args {
			arg.Format(w)
			if i != len(n.Args)-1 {
				_, _ = fmt.Fprint(w, ", ")
			}
		}
	}
	_, _ = fmt.Fprint(w, ")")
}

// specialFormatArgs formats argument list for some special functions.
func (n *FuncCallExpr) specialFormatArgs(w io.Writer) bool {
	switch n.FnName.L {
	case DateAdd, DateSub, AddDate, SubDate:
		n.Args[0].Format(w)
		_, _ = fmt.Fprint(w, ", INTERVAL ")
		n.Args[1].Format(w)
		_, _ = fmt.Fprintf(w, " %s", n.Args[2].(ValueExpr).GetDatumString())
		return true
	case TimestampAdd, TimestampDiff:
		_, _ = fmt.Fprintf(w, "%s, ", n.Args[0].(ValueExpr).GetDatumString())
		n.Args[1].Format(w)
		_, _ = fmt.Fprint(w, ", ")
		n.Args[2].Format(w)
		return true
	}
	return false
}

// Accept implements Node interface.
func (n *FuncCallExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*FuncCallExpr)
	for i, val := range n.Args {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Args[i] = node.(ExprNode)
	}
	return v.Leave(n)
}

// WeightStringCastType is the optional result type in WEIGHT_STRING.
type WeightStringCastType int

// WEIGHT_STRING result types.
const (
	WeightStringCastNone WeightStringCastType = iota
	WeightStringCastChar
	WeightStringCastBinary
)

// WeightStringLevelOrder is the optional ordering flag in WEIGHT_STRING LEVEL.
type WeightStringLevelOrder int

// WEIGHT_STRING LEVEL ordering flags.
const (
	WeightStringLevelOrderNone WeightStringLevelOrder = iota
	WeightStringLevelOrderAsc
	WeightStringLevelOrderDesc
)

// WeightStringLevel is one item in a MariaDB WEIGHT_STRING LEVEL list.
type WeightStringLevel struct {
	Number  uint64
	Order   WeightStringLevelOrder
	Reverse bool
}

// WeightStringLevelRange preserves a LEVEL start-end range, including reversed
// ranges and values that the server clamps to its supported collation levels.
type WeightStringLevelRange struct {
	Start uint64
	End   uint64
}

// WeightStringExpr is a WEIGHT_STRING expression, including its nonstandard
// AS type and MariaDB LEVEL options.
type WeightStringExpr struct {
	funcNode

	Expr     ExprNode
	CastType WeightStringCastType
	Length   uint64
	// Params contains the three numeric parameters of the four-argument form.
	Params []ExprNode
	// Levels and LevelRange are mutually exclusive. They preserve LEVEL 0,
	// per-level flags, and ranges without applying the server's clamping rules.
	Levels     []WeightStringLevel
	LevelRange *WeightStringLevelRange
	// The single-level fields remain available for existing AST consumers.
	// Restore prefers Levels or LevelRange when either is populated.
	HasLevel   bool
	Level      uint64
	LevelOrder WeightStringLevelOrder
	Reverse    bool
}

// Restore implements Node interface.
func (n *WeightStringExpr) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("WEIGHT_STRING")
	ctx.WritePlain("(")
	if err := n.Expr.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore WeightStringExpr.Expr")
	}
	for i, param := range n.Params {
		ctx.WritePlain(",")
		if err := param.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore WeightStringExpr.Params[%d]", i)
		}
	}
	switch n.CastType {
	case WeightStringCastChar:
		ctx.WriteKeyWord(" AS CHAR")
		ctx.WritePlainf("(%d)", n.Length)
	case WeightStringCastBinary:
		ctx.WriteKeyWord(" AS BINARY")
		ctx.WritePlainf("(%d)", n.Length)
	}
	if n.LevelRange != nil {
		ctx.WriteKeyWord(" LEVEL ")
		ctx.WritePlainf("%d-%d", n.LevelRange.Start, n.LevelRange.End)
	} else {
		levels := n.Levels
		if len(levels) == 0 && n.HasLevel {
			levels = []WeightStringLevel{{Number: n.Level, Order: n.LevelOrder, Reverse: n.Reverse}}
		}
		if len(levels) > 0 {
			ctx.WriteKeyWord(" LEVEL ")
		}
		for i, level := range levels {
			if i > 0 {
				ctx.WritePlain(",")
			}
			ctx.WritePlainf("%d", level.Number)
			switch level.Order {
			case WeightStringLevelOrderAsc:
				ctx.WriteKeyWord(" ASC")
			case WeightStringLevelOrderDesc:
				ctx.WriteKeyWord(" DESC")
			}
			if level.Reverse {
				ctx.WriteKeyWord(" REVERSE")
			}
		}
	}
	ctx.WritePlain(")")
	return nil
}

// Format formats the expression into a writer.
func (n *WeightStringExpr) Format(w io.Writer) {
	_, _ = fmt.Fprint(w, "WEIGHT_STRING(")
	n.Expr.Format(w)
	for _, param := range n.Params {
		_, _ = fmt.Fprint(w, ",")
		param.Format(w)
	}
	switch n.CastType {
	case WeightStringCastChar:
		_, _ = fmt.Fprintf(w, " AS CHAR(%d)", n.Length)
	case WeightStringCastBinary:
		_, _ = fmt.Fprintf(w, " AS BINARY(%d)", n.Length)
	}
	if n.LevelRange != nil {
		_, _ = fmt.Fprintf(w, " LEVEL %d-%d", n.LevelRange.Start, n.LevelRange.End)
	} else {
		levels := n.Levels
		if len(levels) == 0 && n.HasLevel {
			levels = []WeightStringLevel{{Number: n.Level, Order: n.LevelOrder, Reverse: n.Reverse}}
		}
		if len(levels) > 0 {
			_, _ = fmt.Fprint(w, " LEVEL ")
		}
		for i, level := range levels {
			if i > 0 {
				_, _ = fmt.Fprint(w, ",")
			}
			_, _ = fmt.Fprint(w, level.Number)
			switch level.Order {
			case WeightStringLevelOrderAsc:
				_, _ = fmt.Fprint(w, " ASC")
			case WeightStringLevelOrderDesc:
				_, _ = fmt.Fprint(w, " DESC")
			}
			if level.Reverse {
				_, _ = fmt.Fprint(w, " REVERSE")
			}
		}
	}
	_, _ = fmt.Fprint(w, ")")
}

// Accept implements Node interface.
func (n *WeightStringExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*WeightStringExpr)
	node, ok := n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	n.Expr = node.(ExprNode)
	for i, param := range n.Params {
		node, ok := param.Accept(v)
		if !ok {
			return n, false
		}
		n.Params[i] = node.(ExprNode)
	}
	return v.Leave(n)
}

// CastFunctionType is the type for cast function.
type CastFunctionType int

// CastFunction types
const (
	CastFunction CastFunctionType = iota + 1
	CastConvertFunction
	CastBinaryOperator
)

// CastTimeZone is the optional MySQL `AT TIME ZONE` or `AT LOCAL` part of a
// CAST expression. Interval distinguishes `AT TIME ZONE INTERVAL 'zone'`.
type CastTimeZone struct {
	Expr     ExprNode
	Interval bool
	Local    bool
}

// FuncCastExpr is the cast function converting value to another type, e.g, cast(expr AS signed).
// See https://dev.mysql.com/doc/refman/5.7/en/cast-functions.html
type FuncCastExpr struct {
	funcNode
	// Expr is the expression to be converted.
	Expr ExprNode
	// Tp is the conversion type.
	Tp *types.FieldType
	// FunctionType is either Cast, Convert or Binary.
	FunctionType CastFunctionType
	// TimeZone is only valid for CAST(... AS DATETIME[(fsp)]).
	TimeZone *CastTimeZone
}

// Restore implements Node interface.
func (n *FuncCastExpr) Restore(ctx *format.RestoreCtx) error {
	switch n.FunctionType {
	case CastFunction:
		ctx.WriteKeyWord("CAST")
		ctx.WritePlain("(")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore FuncCastExpr.Expr")
		}
		if n.TimeZone != nil {
			if n.TimeZone.Local {
				ctx.WriteKeyWord(" AT LOCAL")
			} else {
				ctx.WriteKeyWord(" AT TIME ZONE ")
				if n.TimeZone.Interval {
					ctx.WriteKeyWord("INTERVAL ")
				}
				if err := n.TimeZone.Expr.Restore(ctx); err != nil {
					return errors.Annotate(err, "An error occurred while restore FuncCastExpr.TimeZone.Expr")
				}
			}
		}
		ctx.WriteKeyWord(" AS ")
		n.Tp.RestoreAsCastType(ctx)
		ctx.WritePlain(")")
	case CastConvertFunction:
		ctx.WriteKeyWord("CONVERT")
		ctx.WritePlain("(")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore FuncCastExpr.Expr")
		}
		ctx.WritePlain(", ")
		n.Tp.RestoreAsCastType(ctx)
		ctx.WritePlain(")")
	case CastBinaryOperator:
		ctx.WriteKeyWord("BINARY ")
		if err := n.Expr.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore FuncCastExpr.Expr")
		}
	}
	return nil
}

// Format the ExprNode into a Writer.
func (n *FuncCastExpr) Format(w io.Writer) {
	switch n.FunctionType {
	case CastFunction:
		_, _ = fmt.Fprint(w, "CAST(")
		n.Expr.Format(w)
		if n.TimeZone != nil {
			if n.TimeZone.Local {
				_, _ = fmt.Fprint(w, " AT LOCAL")
			} else {
				_, _ = fmt.Fprint(w, " AT TIME ZONE ")
				if n.TimeZone.Interval {
					_, _ = fmt.Fprint(w, "INTERVAL ")
				}
				n.TimeZone.Expr.Format(w)
			}
		}
		_, _ = fmt.Fprint(w, " AS ")
		n.Tp.FormatAsCastType(w)
		_, _ = fmt.Fprint(w, ")")
	case CastConvertFunction:
		_, _ = fmt.Fprint(w, "CONVERT(")
		n.Expr.Format(w)
		_, _ = fmt.Fprint(w, ", ")
		n.Tp.FormatAsCastType(w)
		_, _ = fmt.Fprint(w, ")")
	case CastBinaryOperator:
		_, _ = fmt.Fprint(w, "BINARY ")
		n.Expr.Format(w)
	}
}

// Accept implements Node Accept interface.
func (n *FuncCastExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*FuncCastExpr)
	node, ok := n.Expr.Accept(v)
	if !ok {
		return n, false
	}
	n.Expr = node.(ExprNode)
	if n.TimeZone != nil && n.TimeZone.Expr != nil {
		node, ok = n.TimeZone.Expr.Accept(v)
		if !ok {
			return n, false
		}
		n.TimeZone.Expr = node.(ExprNode)
	}
	return v.Leave(n)
}

// TrimDirectionType is the type for trim direction.
type TrimDirectionType int

const (
	// TrimBothDefault trims from both direction by default.
	TrimBothDefault TrimDirectionType = iota
	// TrimBoth trims from both direction with explicit notation.
	TrimBoth
	// TrimLeading trims from left.
	TrimLeading
	// TrimTrailing trims from right.
	TrimTrailing
)

// DateArithType is type for DateArith type.
type DateArithType byte

const (
	// DateArithAdd is to run adddate or date_add function option.
	// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_adddate
	// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_date-add
	DateArithAdd DateArithType = iota + 1
	// DateArithSub is to run subdate or date_sub function option.
	// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_subdate
	// See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_date-sub
	DateArithSub
)

const (
	// AggFuncCount is the name of Count function.
	AggFuncCount = "count"
	// AggFuncSum is the name of Sum function.
	AggFuncSum = "sum"
	// AggFuncAvg is the name of Avg function.
	AggFuncAvg = "avg"
	// AggFuncFirstRow is the name of FirstRowColumn function.
	AggFuncFirstRow = "firstrow"
	// AggFuncMax is the name of max function.
	AggFuncMax = "max"
	// AggFuncMin is the name of min function.
	AggFuncMin = "min"
	// AggFuncGroupConcat is the name of group_concat function.
	AggFuncGroupConcat = "group_concat"
	// AggFuncBitOr is the name of bit_or function.
	AggFuncBitOr = "bit_or"
	// AggFuncBitXor is the name of bit_xor function.
	AggFuncBitXor = "bit_xor"
	// AggFuncBitAnd is the name of bit_and function.
	AggFuncBitAnd = "bit_and"
	// AggFuncVarPop is the name of var_pop function
	AggFuncVarPop = "var_pop"
	// AggFuncVarSamp is the name of var_samp function
	AggFuncVarSamp = "var_samp"
	// AggFuncStddevPop is the name of stddev_pop function
	AggFuncStddevPop = "stddev_pop"
	// AggFuncStddevSamp is the name of stddev_samp function
	AggFuncStddevSamp = "stddev_samp"
)

// AggregateFuncExpr represents aggregate function expression.
type AggregateFuncExpr struct {
	funcNode
	// F is the function name.
	F string
	// Args is the function args.
	Args []ExprNode
	// Distinct is true, function hence only aggregate distinct values.
	// For example, column c1 values are "1", "2", "2",  "sum(c1)" is "5",
	// but "sum(distinct c1)" is "3".
	Distinct bool
	// Separator is the optional SEPARATOR expression of GROUP_CONCAT. It is
	// nil when the clause was not written, in which case the server default
	// separator applies.
	Separator ExprNode
	// OrderBy is the optional ORDER BY clause of an ordered-set aggregate.
	OrderBy *OrderByClause
	// Limit is the optional LIMIT clause of an aggregate function.
	Limit *Limit
}

// Restore implements Node interface.
func (n *AggregateFuncExpr) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord(n.F)
	ctx.WritePlain("(")
	if n.Distinct {
		ctx.WriteKeyWord("DISTINCT ")
	}
	for i, argv := range n.Args {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := argv.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore AggregateFuncExpr.Args[%d]", i)
		}
	}
	if n.OrderBy != nil {
		ctx.WritePlain(" ")
		if err := n.OrderBy.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AggregateFuncExpr.OrderBy")
		}
	}
	if n.Separator != nil {
		ctx.WriteKeyWord(" SEPARATOR ")
		if err := n.Separator.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AggregateFuncExpr.Separator")
		}
	}
	if n.Limit != nil {
		ctx.WritePlain(" ")
		if err := n.Limit.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore AggregateFuncExpr.Limit")
		}
	}
	ctx.WritePlain(")")
	return nil
}

// Format the ExprNode into a Writer.
func (n *AggregateFuncExpr) Format(w io.Writer) {
	formatViaRestore(n, w)
}

// Accept implements Node Accept interface.
func (n *AggregateFuncExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*AggregateFuncExpr)
	for i, val := range n.Args {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Args[i] = node.(ExprNode)
	}
	if n.Separator != nil {
		node, ok := n.Separator.Accept(v)
		if !ok {
			return n, false
		}
		n.Separator = node.(ExprNode)
	}
	if n.OrderBy != nil {
		node, ok := n.OrderBy.Accept(v)
		if !ok {
			return n, false
		}
		n.OrderBy = node.(*OrderByClause)
	}
	if n.Limit != nil {
		node, ok := n.Limit.Accept(v)
		if !ok {
			return n, false
		}
		n.Limit = node.(*Limit)
	}
	return v.Leave(n)
}

const (
	// WindowFuncRowNumber is the name of row_number function.
	WindowFuncRowNumber = "row_number"
	// WindowFuncRank is the name of rank function.
	WindowFuncRank = "rank"
	// WindowFuncDenseRank is the name of dense_rank function.
	WindowFuncDenseRank = "dense_rank"
	// WindowFuncCumeDist is the name of cume_dist function.
	WindowFuncCumeDist = "cume_dist"
	// WindowFuncPercentRank is the name of percent_rank function.
	WindowFuncPercentRank = "percent_rank"
	// WindowFuncNtile is the name of ntile function.
	WindowFuncNtile = "ntile"
	// WindowFuncLead is the name of lead function.
	WindowFuncLead = "lead"
	// WindowFuncLag is the name of lag function.
	WindowFuncLag = "lag"
	// WindowFuncFirstValue is the name of first_value function.
	WindowFuncFirstValue = "first_value"
	// WindowFuncLastValue is the name of last_value function.
	WindowFuncLastValue = "last_value"
	// WindowFuncNthValue is the name of nth_value function.
	WindowFuncNthValue = "nth_value"
)

// WindowFuncExpr represents window function expression.
type WindowFuncExpr struct {
	funcNode

	// F is the function name.
	F string
	// Args is the function args.
	Args []ExprNode
	// Distinct cannot be true for most window functions, except `max` and `min`.
	// We need to raise error if it is not allowed to be true.
	Distinct bool
	// IgnoreNull indicates how to handle null value.
	// MySQL only supports `RESPECT NULLS`, so we need to raise error if it is true.
	IgnoreNull bool
	// FromLast indicates the calculation direction of this window function.
	// MySQL only supports calculation from first, so we need to raise error if it is true.
	FromLast bool
	// Spec is the specification of this window.
	Spec WindowSpec
	// WithinGroup is the ordering clause of an ordered-set aggregate.
	WithinGroup *OrderByClause
}

// Restore implements Node interface.
func (n *WindowFuncExpr) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord(n.F)
	ctx.WritePlain("(")
	for i, v := range n.Args {
		if i != 0 {
			ctx.WritePlain(", ")
		} else if n.Distinct {
			ctx.WriteKeyWord("DISTINCT ")
		}
		if err := v.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore WindowFuncExpr.Args[%d]", i)
		}
	}
	ctx.WritePlain(")")
	if n.WithinGroup != nil {
		ctx.WriteKeyWord(" WITHIN GROUP (")
		if err := n.WithinGroup.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore WindowFuncExpr.WithinGroup")
		}
		ctx.WritePlain(")")
	}
	if n.FromLast {
		ctx.WriteKeyWord(" FROM LAST")
	}
	if n.IgnoreNull {
		ctx.WriteKeyWord(" IGNORE NULLS")
	}
	ctx.WriteKeyWord(" OVER ")
	if err := n.Spec.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore WindowFuncExpr.Spec")
	}

	return nil
}

// Format formats the window function expression into a Writer.
func (n *WindowFuncExpr) Format(w io.Writer) {
	formatViaRestore(n, w)
}

// Accept implements Node Accept interface.
func (n *WindowFuncExpr) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*WindowFuncExpr)
	for i, val := range n.Args {
		node, ok := val.Accept(v)
		if !ok {
			return n, false
		}
		n.Args[i] = node.(ExprNode)
	}
	if n.WithinGroup != nil {
		node, ok := n.WithinGroup.Accept(v)
		if !ok {
			return n, false
		}
		n.WithinGroup = node.(*OrderByClause)
	}
	node, ok := n.Spec.Accept(v)
	if !ok {
		return n, false
	}
	n.Spec = *node.(*WindowSpec)
	return v.Leave(n)
}

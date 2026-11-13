package myparser

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mia0x75/myparser/charset"
)

func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isIdentChar(ch rune) bool {
	return isLetter(ch) || isDigit(ch) || ch == '_' || ch == '$' || isIdentExtend(ch)
}

func isIdentExtend(ch rune) bool {
	// Non-ASCII identifier characters, including non-BMP runes such as 𩸽.
	return ch >= 0x80
}

func isUserVarChar(ch rune) bool {
	return isLetter(ch) || isDigit(ch) || ch == '_' || ch == '$' || ch == '.' || isIdentExtend(ch)
}

type trieNode struct {
	childs [256]*trieNode
	token  int
	fn     func(s *Scanner) (int, Pos, string)
}

var ruleTable trieNode

func initTokenByte(c byte, tok int) {
	if ruleTable.childs[c] == nil {
		ruleTable.childs[c] = &trieNode{}
	}
	ruleTable.childs[c].token = tok
}

func initTokenString(str string, tok int) {
	node := &ruleTable
	for _, c := range str {
		if node.childs[c] == nil {
			node.childs[c] = &trieNode{}
		}
		node = node.childs[c]
	}
	node.token = tok
}

func initTokenFunc(str string, fn func(s *Scanner) (int, Pos, string)) {
	for i := 0; i < len(str); i++ {
		c := str[i]
		if ruleTable.childs[c] == nil {
			ruleTable.childs[c] = &trieNode{}
		}
		ruleTable.childs[c].fn = fn
	}
}

func init() {
	// invalid is a special token defined in parser.y, when parser meet
	// this token, it will throw an error.
	// set root trie node's token to invalid, so when input match nothing
	// in the trie, invalid will be the default return token.
	ruleTable.token = invalid
	initTokenByte('*', int('*'))
	initTokenByte('/', int('/'))
	initTokenByte('+', int('+'))
	initTokenByte('>', int('>'))
	initTokenByte('<', int('<'))
	initTokenByte('(', int('('))
	initTokenByte(')', int(')'))
	initTokenByte(';', int(';'))
	initTokenByte(',', int(','))
	initTokenByte('&', int('&'))
	initTokenByte('%', int('%'))
	initTokenByte(':', int(':'))
	initTokenByte('|', int('|'))
	initTokenByte('!', int('!'))
	initTokenByte('^', int('^'))
	initTokenByte('~', int('~'))
	initTokenByte('\\', int('\\'))
	initTokenByte('?', paramMarker)
	initTokenByte('=', eq)
	initTokenByte('{', int('{'))
	initTokenByte('}', int('}'))

	initTokenString("||", pipes)
	initTokenString("&&", andand)
	initTokenString("&^", andnot)
	initTokenString(":=", assignmentEq)
	initTokenString("<=>", nulleq)
	initTokenString(">=", ge)
	initTokenString("<=", le)
	initTokenString("!=", neq)
	initTokenString("<>", neqSynonym)
	initTokenString("<<", lsh)
	initTokenString(">>", rsh)
	initTokenString("\\N", null)

	initTokenFunc("@", startWithAt)
	initTokenFunc("/", startWithSlash)
	initTokenFunc("-", startWithDash)
	initTokenFunc("#", startWithSharp)
	initTokenFunc("Xx", startWithXx)
	initTokenFunc("Nn", startWithNn)
	initTokenFunc("Bb", startWithBb)
	initTokenFunc(".", startWithDot)
	initTokenFunc("_$ACDEFGHIJKLMOPQRSTUVWYZacdefghijklmopqrstuvwyz", scanIdentifier)
	initTokenFunc("`", scanQuotedIdent)
	initTokenFunc("0123456789", startWithNumber)
	initTokenFunc("'\"", startString)
}

var tokenMap = map[string]int{
	"ACCOUNT":                  account,
	"ACTION":                   action,
	"ADD":                      add,
	"ADDDATE":                  addDate,
	"AFTER":                    after,
	"ALL":                      all,
	"ALGORITHM":                algorithm,
	"ALTER":                    alter,
	"ALWAYS":                   always,
	"ANALYZE":                  analyze,
	"ANALYSE":                  analyse,
	"AND":                      and,
	"ANY":                      any,
	"AS":                       as,
	"ASC":                      asc,
	"ASCII":                    ascii,
	"AUTO_INCREMENT":           autoIncrement,
	"AVG":                      avg,
	"AVG_ROW_LENGTH":           avgRowLength,
	"BEFORE":                   before,
	"BACKUP":                   backup,
	"BEGIN":                    begin,
	"BETWEEN":                  between,
	"BIGINT":                   bigIntType,
	"BINARY":                   binaryType,
	"BUCKETS":                  buckets,
	"BINLOG":                   binlog,
	"BIT":                      bitType,
	"BIT_AND":                  bitAnd,
	"BIT_OR":                   bitOr,
	"BIT_XOR":                  bitXor,
	"BLOB":                     blobType,
	"BLOCK":                    block,
	"BOOL":                     boolType,
	"BOOLEAN":                  booleanType,
	"BOTH":                     both,
	"BTREE":                    btree,
	"BY":                       by,
	"CALL":                     call,
	"BYTE":                     byteType,
	"CASCADE":                  cascade,
	"CASCADED":                 cascaded,
	"CASE":                     caseKwd,
	"CAST":                     cast,
	"CHANGE":                   change,
	"CHAR":                     charType,
	"CHARACTER":                character,
	"CHARSET":                  charsetKwd,
	"CHECK":                    check,
	"CHECKSUM":                 checksum,
	"CHAIN":                    chain,
	"CIPHER":                   cipher,
	"CLEANUP":                  cleanup,
	"CLIENT":                   client,
	"COALESCE":                 coalesce,
	"COLLATE":                  collate,
	"COLLATION":                collation,
	"COLUMN":                   column,
	"COLUMNS":                  columns,
	"COMMENT":                  comment,
	"COMMIT":                   commit,
	"COMMITTED":                committed,
	"COMPACT":                  compact,
	"COMPRESSED":               compressed,
	"COMPRESSION":              compression,
	"CONNECTION":               connection,
	"CONSISTENT":               consistent,
	"CONSTRAINT":               constraint,
	"CONTEXT":                  context,
	"CONVERT":                  convert,
	"COPY":                     copyKwd,
	"COUNT":                    count,
	"CPU":                      cpu,
	"CREATE":                   create,
	"CROSS":                    cross,
	"CURRENT":                  current,
	"CURRENT_DATE":             currentDate,
	"CURRENT_TIME":             currentTime,
	"CURRENT_TIMESTAMP":        currentTs,
	"CURRENT_USER":             currentUser,
	"CURRENT_ROLE":             currentRole,
	"CURTIME":                  curTime,
	"DATA":                     data,
	"DATABASE":                 database,
	"DATABASES":                databases,
	"DATE":                     dateType,
	"DATE_ADD":                 dateAdd,
	"DATE_SUB":                 dateSub,
	"DATETIME":                 datetimeType,
	"DAY":                      day,
	"DAY_HOUR":                 dayHour,
	"DAY_MICROSECOND":          dayMicrosecond,
	"DAY_MINUTE":               dayMinute,
	"DAY_SECOND":               daySecond,
	"DEALLOCATE":               deallocate,
	"DEC":                      decimalType,
	"DECIMAL":                  decimalType,
	"DEFAULT":                  defaultKwd,
	"DEFINER":                  definer,
	"DELAY_KEY_WRITE":          delayKeyWrite,
	"DELAYED":                  delayed,
	"DELETE":                   deleteKwd,
	"DESC":                     desc,
	"DESCRIBE":                 describe,
	"DISABLE":                  disable,
	"DISTINCT":                 distinct,
	"DISTINCTROW":              distinct,
	"DIV":                      div,
	"DO":                       do,
	"DOUBLE":                   doubleType,
	"DROP":                     drop,
	"DUAL":                     dual,
	"DUPLICATE":                duplicate,
	"DYNAMIC":                  dynamic,
	"ELSE":                     elseKwd,
	"ENABLE":                   enable,
	"ENCLOSED":                 enclosed,
	"END":                      end,
	"ENGINE":                   engine,
	"ENGINES":                  engines,
	"ENUM":                     enum,
	"ESCAPE":                   escape,
	"ESCAPED":                  escaped,
	"EVENT":                    event,
	"EVENTS":                   events,
	"EXCLUSIVE":                exclusive,
	"EXCLUDE":                  exclude,
	"EXCEPT":                   except,
	"EXECUTE":                  execute,
	"EXISTS":                   exists,
	"EXPIRE":                   expire,
	"EXPLAIN":                  explain,
	"EXTRACT":                  extract,
	"FALSE":                    falseKwd,
	"FAULTS":                   faultsSym,
	"FIELDS":                   fields,
	"FIRST":                    first,
	"FIXED":                    fixed,
	"FLOAT":                    floatType,
	"FLUSH":                    flush,
	"FOLLOWING":                following,
	"FOR":                      forKwd,
	"FORCE":                    force,
	"FOREIGN":                  foreign,
	"FORMAT":                   format,
	"FROM":                     from,
	"FULL":                     full,
	"FULLTEXT":                 fulltext,
	"FUNCTION":                 function,
	"GENERATED":                generated,
	"GET_FORMAT":               getFormat,
	"GEOMETRY":                 geometryType,
	"GEOMETRYCOLLECTION":       geometryCollectionType,
	"GLOBAL":                   global,
	"GRANT":                    grant,
	"GRANTS":                   grants,
	"GROUP":                    group,
	"GROUP_CONCAT":             groupConcat,
	"HASH":                     hash,
	"HAVING":                   having,
	"HIGH_PRIORITY":            highPriority,
	"HOUR":                     hour,
	"HOUR_MICROSECOND":         hourMicrosecond,
	"HOUR_MINUTE":              hourMinute,
	"HOUR_SECOND":              hourSecond,
	"IDENTIFIED":               identified,
	"IF":                       ifKwd,
	"IGNORE":                   ignore,
	"IMMEDIATE":                immediate,
	"IN":                       in,
	"INDEX":                    index,
	"INDEXES":                  indexes,
	"INFILE":                   infile,
	"INNER":                    inner,
	"INET4":                    inet4Type,
	"INET6":                    inet6Type,
	"INPLACE":                  inplace,
	"INSTANT":                  instant,
	"INSERT":                   insert,
	"INT":                      intType,
	"INT1":                     int1Type,
	"INT2":                     int2Type,
	"INT3":                     int3Type,
	"INT4":                     int4Type,
	"INT8":                     int8Type,
	"IO":                       io,
	"IPC":                      ipc,
	"INTEGER":                  integerType,
	"INTERVAL":                 interval,
	"INTERNAL":                 internal,
	"INTERSECT":                intersect,
	"INTO":                     into,
	"INVOKER":                  invoker,
	"IS":                       is,
	"ISSUER":                   issuer,
	"ISOLATION":                isolation,
	"JOIN":                     join,
	"JSON":                     jsonType,
	"KEY":                      key,
	"KEY_BLOCK_SIZE":           keyBlockSize,
	"KEYS":                     keys,
	"KILL":                     kill,
	"LAST":                     last,
	"LEADING":                  leading,
	"LEFT":                     left,
	"LESS":                     less,
	"LEVEL":                    level,
	"LIKE":                     like,
	"LIMIT":                    limit,
	"LINES":                    lines,
	"LINESTRING":               lineStringType,
	"LINEAR":                   linear,
	"LOAD":                     load,
	"LOCAL":                    local,
	"LOCALTIME":                localTime,
	"LOCALTIMESTAMP":           localTs,
	"LOCK":                     lock,
	"LOGS":                     logs,
	"LOG":                      log,
	"LONG":                     long,
	"LONGBLOB":                 longblobType,
	"LONGTEXT":                 longtextType,
	"LOW_PRIORITY":             lowPriority,
	"MASTER":                   master,
	"MAX":                      max,
	"MAX_CONNECTIONS_PER_HOUR": maxConnectionsPerHour,
	"MAX_EXECUTION_TIME":       maxExecutionTime,
	"MAX_QUERIES_PER_HOUR":     maxQueriesPerHour,
	"MAX_ROWS":                 maxRows,
	"MAX_UPDATES_PER_HOUR":     maxUpdatesPerHour,
	"MAX_USER_CONNECTIONS":     maxUserConnections,
	"MAXVALUE":                 maxValue,
	"MEDIUMBLOB":               mediumblobType,
	"MEDIUMINT":                mediumIntType,
	"MEDIUMTEXT":               mediumtextType,
	"MEMORY":                   memory,
	"MERGE":                    merge,
	"MICROSECOND":              microsecond,
	"MIN":                      min,
	"MIN_ROWS":                 minRows,
	"MINUTE":                   minute,
	"MINUTE_MICROSECOND":       minuteMicrosecond,
	"MINUTE_SECOND":            minuteSecond,
	"MOD":                      mod,
	"MODE":                     mode,
	"MODIFY":                   modify,
	"MONTH":                    month,
	"MULTILINESTRING":          multiLineStringType,
	"MULTIPOINT":               multiPointType,
	"MULTIPOLYGON":             multiPolygonType,
	"NAMES":                    names,
	"NATIONAL":                 national,
	"NATURAL":                  natural,
	"NEVER":                    never,
	"NEXT_ROW_ID":              next_row_id,
	"NO":                       no,
	"NO_WRITE_TO_BINLOG":       noWriteToBinLog,
	"NONE":                     none,
	"NOT":                      not,
	"NOW":                      now,
	"NULL":                     null,
	"NULLS":                    nulls,
	"NUMERIC":                  numericType,
	"NVARCHAR":                 nvarcharType,
	"OFFSET":                   offset,
	"ON":                       on,
	"ONLY":                     only,
	"OTHERS":                   others,
	"OPTION":                   option,
	"OPTIONALLY":               optionally,
	"OR":                       or,
	"ORDER":                    order,
	"OUTER":                    outer,
	"PACK_KEYS":                packKeys,
	"PAGE":                     pageSym,
	"PARTITION":                partition,
	"PARTITIONS":               partitions,
	"PASSWORD":                 password,
	"PERIOD":                   period,
	"PLUGINS":                  plugins,
	"POINT":                    pointType,
	"POLYGON":                  polygonType,
	"POSITION":                 position,
	"PRECEDING":                preceding,
	"PRECISION":                precisionType,
	"PREPARE":                  prepare,
	"PRIMARY":                  primary,
	"PRIVILEGES":               privileges,
	"PROCEDURE":                procedure,
	"PROCESS":                  process,
	"PROCESSLIST":              processlist,
	"PROFILE":                  profile,
	"PROFILES":                 profiles,
	"PURGE":                    purge,
	"QUARTER":                  quarter,
	"QUERY":                    query,
	"QUERIES":                  queries,
	"QUICK":                    quick,
	"SHARD_ROW_ID_BITS":        shardRowIDBits,
	"RANGE":                    rangeKwd,
	"RECOVER":                  recover,
	"READ":                     read,
	"REAL":                     realType,
	"RECENT":                   recent,
	"REDUNDANT":                redundant,
	"REFERENCES":               references,
	"REGEXP":                   regexpKwd,
	"RELOAD":                   reload,
	"RENAME":                   rename,
	"REPEAT":                   repeat,
	"REPEATABLE":               repeatable,
	"REPLACE":                  replace,
	"RESPECT":                  respect,
	"REPLICATION":              replication,
	"REQUIRE":                  require,
	"RESTRICT":                 restrict,
	"REVERSE":                  reverse,
	"REVOKE":                   revoke,
	"RECURSIVE":                recursive,
	"RIGHT":                    right,
	"RLIKE":                    rlike,
	"ROLE":                     role,
	"ROLLBACK":                 rollback,
	"ROUTINE":                  routine,
	"ROW":                      row,
	"ROW_COUNT":                rowCount,
	"ROW_FORMAT":               rowFormat,
	"RTREE":                    rtree,
	"SCHEMA":                   database,
	"SCHEMAS":                  databases,
	"SECOND":                   second,
	"SECOND_MICROSECOND":       secondMicrosecond,
	"SECURITY":                 security,
	"SELECT":                   selectKwd,
	"SERIAL":                   serialType,
	"SERIALIZABLE":             serializable,
	"SESSION":                  session,
	"SET":                      set,
	"SEPARATOR":                separator,
	"SHARE":                    share,
	"SHARED":                   shared,
	"SHOW":                     show,
	"SIGNED":                   signed,
	"SLAVE":                    slave,
	"SLOW":                     slow,
	"SMALLINT":                 smallIntType,
	"SNAPSHOT":                 snapshot,
	"SOME":                     some,
	"SQL":                      sql,
	"SQL_BIG_RESULT":           sqlBigResult,
	"SQL_BUFFER_RESULT":        sqlBufferResult,
	"SQL_CACHE":                sqlCache,
	"SQL_CALC_FOUND_ROWS":      sqlCalcFoundRows,
	"SQL_NO_CACHE":             sqlNoCache,
	"SQL_SMALL_RESULT":         sqlSmallResult,
	"SRID":                     srid,
	"SPATIAL":                  spatial,
	"SOURCE":                   source,
	"SSL":                      ssl,
	"START":                    start,
	"STARTING":                 starting,
	"STATEMENT":                statement,
	"STATS_PERSISTENT":         statsPersistent,
	"STATUS":                   status,
	"SWAPS":                    swaps,
	"SWITCHES":                 switchesSym,
	"OPEN":                     open,
	"STD":                      stddevPop,
	"STDDEV":                   stddevPop,
	"STDDEV_POP":               stddevPop,
	"STDDEV_SAMP":              stddevSamp,
	"STORED":                   stored,
	"STRAIGHT_JOIN":            straightJoin,
	"SUBDATE":                  subDate,
	"SUBJECT":                  subject,
	"SUBPARTITION":             subpartition,
	"SUBPARTITIONS":            subpartitions,
	"SUBSTR":                   substring,
	"SUBSTRING":                substring,
	"SUM":                      sum,
	"SUPER":                    super,
	"TABLE":                    tableKwd,
	"TABLES":                   tables,
	"TABLESPACE":               tablespace,
	"TEMPORARY":                temporary,
	"TEMPTABLE":                temptable,
	"TERMINATED":               terminated,
	"TEXT":                     textType,
	"THAN":                     than,
	"THEN":                     then,
	"TIME":                     timeType,
	"TIES":                     ties,
	"TIMESTAMP":                timestampType,
	"TIMESTAMPADD":             timestampAdd,
	"TIMESTAMPDIFF":            timestampDiff,
	"TINYBLOB":                 tinyblobType,
	"TINYINT":                  tinyIntType,
	"TINYTEXT":                 tinytextType,
	"TO":                       to,
	"TOKUDB_DEFAULT":           tokudbDefault,
	"TOKUDB_FAST":              tokudbFast,
	"TOKUDB_LZMA":              tokudbLzma,
	"TOKUDB_QUICKLZ":           tokudbQuickLZ,
	"TOKUDB_SNAPPY":            tokudbSnappy,
	"TOKUDB_SMALL":             tokudbSmall,
	"TOKUDB_UNCOMPRESSED":      tokudbUncompressed,
	"TOKUDB_ZLIB":              tokudbZlib,
	"TOP":                      top,
	"TRAILING":                 trailing,
	"TRANSACTION":              transaction,
	"TRIGGER":                  trigger,
	"TRIGGERS":                 triggers,
	"TRIM":                     trim,
	"TRUE":                     trueKwd,
	"TRUNCATE":                 truncate,
	"UNBOUNDED":                unbounded,
	"UNCOMMITTED":              uncommitted,
	"UNDEFINED":                undefined,
	"UNION":                    union,
	"UNIQUE":                   unique,
	"UNKNOWN":                  unknown,
	"UNLOCK":                   unlock,
	"UNSIGNED":                 unsigned,
	"UPDATE":                   update,
	"USAGE":                    usage,
	"USE":                      use,
	"USER":                     user,
	"USING":                    using,
	"UUID":                     uuidType,
	"UTC_DATE":                 utcDate,
	"UTC_TIME":                 utcTime,
	"UTC_TIMESTAMP":            utcTimestamp,
	"VALUE":                    value,
	"VALUES":                   values,
	"VARBINARY":                varbinaryType,
	"VARCHAR":                  varcharType,
	"VARIABLES":                variables,
	"VARIANCE":                 varPop,
	"VAR_POP":                  varPop,
	"VAR_SAMP":                 varSamp,
	"VIEW":                     view,
	"VIRTUAL":                  virtual,
	"WARNINGS":                 warnings,
	"WEIGHT_STRING":            weightString,
	"ERRORS":                   identSQLErrors,
	"ERROR":                    errorKwd,
	"EMPTY":                    emptyKwd,
	"LATERAL":                  lateral,
	"PATH":                     pathKwd,
	"NESTED":                   nested,
	"ORDINALITY":               ordinality,
	"JSON_TABLE":               jsonTable,
	"WEEK":                     week,
	"WHEN":                     when,
	"WHERE":                    where,
	"WITHIN":                   within,
	"WITH":                     with,
	"WRITE":                    write,
	"XOR":                      xor,
	"X509":                     x509,
	"YEAR":                     yearType,
	"YEAR_MONTH":               yearMonth,
	"ZEROFILL":                 zerofill,
	"ZONE":                     zone,
	"AT":                       at,
	"COMPLETION":               completion,
	"CONTAINS":                 contains,
	"DETERMINISTIC":            deterministic,
	"EACH":                     each,
	"ENDS":                     ends,
	"EVERY":                    every,
	"FOLLOWS":                  follows,
	"INOUT":                    inout,
	"LANGUAGE":                 language,
	"MODIFIES":                 modifies,
	"OF":                       of,
	"OUT":                      out,
	"PRECEDES":                 precedes,
	"PRESERVE":                 preserve,
	"READS":                    reads,
	"RETURNS":                  returns,
	"RETURN":                   returnKwd,
	"SCHEDULE":                 schedule,
	"STARTS":                   starts,
	"ATOMIC":                   atomic,
	"CLOSE":                    close,
	"CONDITION":                condition,
	"CONTINUE":                 continueKwd,
	"CURSOR":                   cursor,
	"DECLARE":                  declare,
	"ELSEIF":                   elseif,
	"EXIT":                     exit,
	"FETCH":                    fetch,
	"FOUND":                    found,
	"HANDLER":                  handler,
	"ITERATE":                  iterate,
	"LEAVE":                    leave,
	"LOOP":                     loop,
	"RESIGNAL":                 resignal,
	"SIGNAL":                   signal,
	"SQLEXCEPTION":             sqlException,
	"SQLSTATE":                 sqlState,
	"SQLWARNING":               sqlWarning,
	"UNDO":                     undo,
	"UNTIL":                    until,
	"WHILE":                    while,
	"NOWAIT":                   nowait,
	"SKIP":                     skip,
	"LOCKED":                   locked,
	"ROLLUP":                   rollup,
	"WORK":                     work,
	"RELEASE":                  release,
	"SAVEPOINT":                savepoint,
	"SEQUENCE":                 sequence,
	"INCREMENT":                increment,
	"MINVALUE":                 minvalue,
	"CACHE":                    cache,
	"CYCLE":                    cycle,
	"NOMAXVALUE":               nomaxvalue,
	"NOMINVALUE":               nominvalue,
	"NOCYCLE":                  nocycle,
	"NOCACHE":                  nocache,
	"AGGREGATE":                aggregate,
	"SONAME":                   soname,
	"STRING":                   stringType,
	"RETURNING":                returning,
	"XA":                       xa,
	"RESUME":                   resume,
	"MIGRATE":                  migrate,
	"SUSPEND":                  suspend,
	"ONE":                      one,
	"PHASE":                    phase,
	"CUBE":                     cube,
	"SERVER":                   server,
	"DATAFILE":                 datafile,
	"WRAPPER":                  wrapper,
	"OPTIONS":                  options,
	"FILE_BLOCK_SIZE":          fileBlockSize,
	"EXTENT_SIZE":              extentSize,
	"INITIAL_SIZE":             initialSize,
	"AUTOEXTEND_SIZE":          autoextendSize,
	"MAX_SIZE":                 maxSize,
	"NODEGROUP":                nodegroup,
	"WAIT":                     wait,
	"RESOURCE":                 resource,
	"LOGFILE":                  logfile,
	"VCPU":                     vcpu,
	"UNDO_BUFFER_SIZE":         undoBufferSize,
	"REDO_BUFFER_SIZE":         redoBufferSize,
	"THREAD_PRIORITY":          threadPriority,
	"UNDOFILE":                 undofile,
	"TYPE":                     typeSym,
	"SYSTEM":                   systemSym,
	"REORGANIZE":               reorganize,
	"EXCHANGE":                 exchange,
	"DISCARD":                  discard,
	"IMPORT":                   import_,
	"VALIDATION":               validation,
	"WITHOUT":                  without,
	"REMOVE":                   remove,
	"PARTITIONING":             partitioning,
	"OUTFILE":                  outfile,
	"DUMPFILE":                 dumpfile,
	"REPAIR":                   repair,
	"OPTIMIZE":                 optimize,
	"UPGRADE":                  upgrade,
	"VISIBLE":                  visible,
	"INVISIBLE":                invisible,
	"TRADITIONAL":              traditional,
	"TREE":                     tree,
	"AGAINST":                  against,
	"CHANNEL":                  channel,
	"CLONE":                    clone,
	"COMPONENT":                component,
	"DIAGNOSTICS":              diagnostics,
	"DIRECTORY":                directory,
	"EXPANSION":                expansion,
	"EXTENDED":                 extended,
	"GET":                      get,
	"GTIDS":                    gtids,
	"INSTALL":                  install,
	"INSTANCE":                 instance,
	"IO_THREAD":                ioThread,
	"LEAVES":                   leaves,
	"MATCH":                    match,
	"MEMBER":                   memberKwd,
	"NEXT":                     next,
	"PERSIST":                  persist,
	"PERSIST_ONLY":             persistOnly,
	"PERSISTENT":               persistent,
	"PLUGIN":                   plugin,
	"PREV":                     prev,
	"REPLICA":                  replica,
	"REPLICAS":                 replicas,
	"RESET":                    reset,
	"ROWS":                     rows,
	"SHUTDOWN":                 shutdown,
	"SLAVES":                   slaves,
	"SOUNDS":                   sounds,
	"SQL_THREAD":               sqlThread,
	"STACKED":                  stacked,
	"STOP":                     stop,
	"UNINSTALL":                uninstall,
	"XML":                      xml,
	"ARRAY":                    arrayKwd,
	"OBJECT":                   objectKwd,
	"SCALAR":                   scalarKwd,
	"CONCURRENT":               concurrent,

	"MASTER_HOST":                            masterHost,
	"SOURCE_HOST":                            sourceHost,
	"MASTER_BIND":                            masterBind,
	"SOURCE_BIND":                            sourceBind,
	"MASTER_USER":                            masterUser,
	"SOURCE_USER":                            sourceUser,
	"MASTER_PASSWORD":                        masterPassword,
	"SOURCE_PASSWORD":                        sourcePassword,
	"MASTER_PORT":                            masterPort,
	"SOURCE_PORT":                            sourcePort,
	"MASTER_CONNECT_RETRY":                   masterConnectRetry,
	"SOURCE_CONNECT_RETRY":                   sourceConnectRetry,
	"MASTER_RETRY_COUNT":                     masterRetryCount,
	"SOURCE_RETRY_COUNT":                     sourceRetryCount,
	"MASTER_DELAY":                           masterDelay,
	"SOURCE_DELAY":                           sourceDelay,
	"MASTER_SSL":                             masterSSL,
	"SOURCE_SSL":                             sourceSSL,
	"MASTER_SSL_CA":                          masterSSLCA,
	"SOURCE_SSL_CA":                          sourceSSLCA,
	"MASTER_SSL_CAPATH":                      masterSSLCAPath,
	"SOURCE_SSL_CAPATH":                      sourceSSLCAPath,
	"MASTER_SSL_CERT":                        masterSSLCert,
	"SOURCE_SSL_CERT":                        sourceSSLCert,
	"MASTER_SSL_CIPHER":                      masterSSLCipher,
	"SOURCE_SSL_CIPHER":                      sourceSSLCipher,
	"MASTER_SSL_KEY":                         masterSSLKey,
	"SOURCE_SSL_KEY":                         sourceSSLKey,
	"MASTER_SSL_VERIFY_SERVER_CERT":          masterSSLVerifyServerCert,
	"SOURCE_SSL_VERIFY_SERVER_CERT":          sourceSSLVerifyServerCert,
	"MASTER_SSL_CRL":                         masterSSLCRL,
	"SOURCE_SSL_CRL":                         sourceSSLCRL,
	"MASTER_SSL_CRLPATH":                     masterSSLCRLPath,
	"SOURCE_SSL_CRLPATH":                     sourceSSLCRLPath,
	"MASTER_TLS_VERSION":                     masterTLSVersion,
	"SOURCE_TLS_VERSION":                     sourceTLSVersion,
	"MASTER_TLS_CIPHERSUITES":                masterTLSCipherSuites,
	"SOURCE_TLS_CIPHERSUITES":                sourceTLSCipherSuites,
	"MASTER_PUBLIC_KEY_PATH":                 masterPublicKeyPath,
	"SOURCE_PUBLIC_KEY_PATH":                 sourcePublicKeyPath,
	"GET_MASTER_PUBLIC_KEY":                  getMasterPublicKey,
	"GET_SOURCE_PUBLIC_KEY":                  getSourcePublicKey,
	"MASTER_HEARTBEAT_PERIOD":                masterHeartbeatPeriod,
	"SOURCE_HEARTBEAT_PERIOD":                sourceHeartbeatPeriod,
	"MASTER_COMPRESSION_ALGORITHMS":          masterCompressionAlgorithms,
	"SOURCE_COMPRESSION_ALGORITHMS":          sourceCompressionAlgorithms,
	"MASTER_ZSTD_COMPRESSION_LEVEL":          masterZstdCompressionLevel,
	"SOURCE_ZSTD_COMPRESSION_LEVEL":          sourceZstdCompressionLevel,
	"MASTER_AUTO_POSITION":                   masterAutoPosition,
	"SOURCE_AUTO_POSITION":                   sourceAutoPosition,
	"SOURCE_CONNECTION_AUTO_FAILOVER":        sourceConnectionAutoFailover,
	"REQUIRE_ROW_FORMAT":                     requireRowFormat,
	"REQUIRE_TABLE_PRIMARY_KEY_CHECK":        requireTablePrimaryKeyCheck,
	"ASSIGN_GTIDS_TO_ANONYMOUS_TRANSACTIONS": assignGTIDsToAnonymousTransactions,
	"PRIVILEGE_CHECKS_USER":                  privilegeChecksUser,
	"GTID_ONLY":                              gtidOnly,
	"NETWORK_NAMESPACE":                      networkNamespace,
	"IGNORE_SERVER_IDS":                      ignoreServerIDs,
	"DO_DOMAIN_IDS":                          doDomainIDs,
	"IGNORE_DOMAIN_IDS":                      ignoreDomainIDs,
	"MASTER_USE_GTID":                        masterUseGTID,
	"MASTER_DEMOTE_TO_SLAVE":                 masterDemoteToSlave,
	"MASTER_DEMOTE_TO_REPLICA":               masterDemoteToReplica,
	"MASTER_LOG_FILE":                        masterLogFile,
	"SOURCE_LOG_FILE":                        sourceLogFile,
	"MASTER_LOG_POS":                         masterLogPos,
	"SOURCE_LOG_POS":                         sourceLogPos,
	"RELAY_LOG_FILE":                         relayLogFile,
	"RELAY_LOG_POS":                          relayLogPos,
	"MASTER_GTID_POS":                        masterGTIDPos,
	"SQL_BEFORE_GTIDS":                       sqlBeforeGTIDs,
	"SQL_AFTER_GTIDS":                        sqlAfterGTIDs,
	"SQL_AFTER_MTS_GAPS":                     sqlAfterMTSGaps,
	"DEFAULT_AUTH":                           defaultAuth,
	"PLUGIN_DIR":                             pluginDir,
	"CURRENT_POS":                            currentPos,
	"SLAVE_POS":                              slavePos,
	"REPLICA_POS":                            replicaPos,
	"STREAM":                                 streamKwd,
	"GENERATE":                               generateKwd,
	"OFF":                                    off,
}

// diagnosticsStatementItemName validates a GET DIAGNOSTICS statement item.
func diagnosticsStatementItemName(name string) (string, bool) {
	switch strings.ToUpper(name) {
	case "NUMBER", "ROW_COUNT":
		return strings.ToUpper(name), true
	}
	return "", false
}

// diagnosticsConditionItemName validates a MySQL or MariaDB condition item.
func diagnosticsConditionItemName(name string) (string, bool) {
	switch strings.ToUpper(name) {
	case "CLASS_ORIGIN", "SUBCLASS_ORIGIN", "RETURNED_SQLSTATE",
		"MESSAGE_TEXT", "MYSQL_ERRNO", "CONSTRAINT_CATALOG",
		"CONSTRAINT_SCHEMA", "CONSTRAINT_NAME", "CATALOG_NAME",
		"SCHEMA_NAME", "TABLE_NAME", "COLUMN_NAME", "CURSOR_NAME", "ROW_NUMBER":
		return strings.ToUpper(name), true
	}
	return "", false
}

// See https://dev.mysql.com/doc/refman/5.7/en/function-resolution.html for details
var btFuncTokenMap = map[string]int{
	"ADDDATE":       builtinAddDate,
	"BIT_AND":       builtinBitAnd,
	"BIT_OR":        builtinBitOr,
	"BIT_XOR":       builtinBitXor,
	"CAST":          builtinCast,
	"COUNT":         builtinCount,
	"CURDATE":       builtinCurDate,
	"CURTIME":       builtinCurTime,
	"DATE_ADD":      builtinDateAdd,
	"DATE_SUB":      builtinDateSub,
	"EXTRACT":       builtinExtract,
	"GROUP_CONCAT":  builtinGroupConcat,
	"JSON_ARRAYAGG": builtinJSONArrayAgg,
	"MAX":           builtinMax,
	"MID":           builtinSubstring,
	"MIN":           builtinMin,
	"NOW":           builtinNow,
	"POSITION":      builtinPosition,
	"SESSION_USER":  builtinUser,
	"STD":           builtinStddevPop,
	"STDDEV":        builtinStddevPop,
	"STDDEV_POP":    builtinStddevPop,
	"STDDEV_SAMP":   builtinStddevSamp,
	"SUBDATE":       builtinSubDate,
	"SUBSTR":        builtinSubstring,
	"SUBSTRING":     builtinSubstring,
	"SUM":           builtinSum,
	"SYSDATE":       builtinSysDate,
	"SYSTEM_USER":   builtinUser,
	"TRIM":          builtinTrim,
	"VARIANCE":      builtinVarPop,
	"VAR_POP":       builtinVarPop,
	"VAR_SAMP":      builtinVarSamp,
}

var windowFuncTokenMap = map[string]int{
	"CUME_DIST":    cumeDist,
	"DENSE_RANK":   denseRank,
	"FIRST_VALUE":  firstValue,
	"GROUPS":       groups,
	"LAG":          lag,
	"LAST_VALUE":   lastValue,
	"LEAD":         lead,
	"NTH_VALUE":    nthValue,
	"NTILE":        ntile,
	"OVER":         over,
	"PERCENT_RANK": percentRank,
	"RANK":         rank,
	"ROW_NUMBER":   rowNumber,
	"WINDOW":       window,
}

// aliases are strings directly map to another string and use the same token.
var aliases = map[string]string{
	"SCHEMA":  "DATABASE",
	"SCHEMAS": "DATABASES",
	"DEC":     "DECIMAL",
	"SUBSTR":  "SUBSTRING",
}

func (s *Scanner) isTokenIdentifier(lit string, offset int) int {
	// An identifier before or after '.' means it is part of a qualified identifier.
	// We do not parse it as keyword.
	next := offset + len(lit)
	if next < len(s.r.s) && s.r.s[next] == '.' {
		return 0
	}
	if offset > 0 && s.r.s[offset-1] == '.' {
		return 0
	}
	buf := &s.buf
	buf.Reset()
	buf.Grow(len(lit))
	data := buf.Bytes()[:len(lit)]
	for i := 0; i < len(lit); i++ {
		if lit[i] >= 'a' && lit[i] <= 'z' {
			data[i] = lit[i] + 'A' - 'a'
		} else {
			data[i] = lit[i]
		}
	}

	checkBtFuncToken := false
	if s.sqlMode.HasIgnoreSpaceMode() {
		for next < len(s.r.s) && unicode.IsSpace(rune(s.r.s[next])) {
			next++
		}
	}
	if next < len(s.r.s) && s.r.s[next] == '(' {
		checkBtFuncToken = true
	}
	if checkBtFuncToken {
		if tok := btFuncTokenMap[string(data)]; tok != 0 {
			return tok
		}
	}
	tok, ok := tokenMap[string(data)]
	if !ok && s.supportWindowFunc {
		tok = windowFuncTokenMap[string(data)]
	}
	if tok == respect || tok == ignore {
		if s.nullTreatmentFollows() && s.consumeNullsWord() {
			if tok == respect {
				return respectNulls
			}
			return ignoreNulls
		}
	}
	return tok
}

// nullTreatmentFollows reports whether a NULL treatment keyword can follow the
// previous token. The treatment follows the closing parenthesis of a function
// call or the FROM FIRST/FROM LAST clause of NTH_VALUE.
func (s *Scanner) nullTreatmentFollows() bool {
	switch s.lastTok {
	case ')', first, last:
		return true
	}
	return false
}

// consumeNullsWord consumes a NULLS keyword that follows the current word.
// RESPECT NULLS and IGNORE NULLS are the only two-word keywords of the
// grammar and they always follow the closing parenthesis of a function call.
// Merging them keeps the grammar free of the shift/reduce conflict between
// the scalar LAST_VALUE call and the window function null treatment.
func (s *Scanner) consumeNullsWord() bool {
	rest := s.r.s[s.r.p.Offset:]
	i := 0
	for i < len(rest) && (rest[i] == ' ' || rest[i] == '\t' || rest[i] == '\r' || rest[i] == '\n') {
		i++
	}
	if len(rest)-i < len("nulls") || !strings.EqualFold(rest[i:i+len("nulls")], "nulls") {
		return false
	}
	if j := i + len("nulls"); j < len(rest) {
		if r, _ := utf8.DecodeRuneInString(rest[j:]); isIdentExtend(r) {
			return false
		}
	}
	for k := 0; k < i+len("nulls"); k++ {
		s.r.peek()
		s.r.inc()
	}
	return true
}

func handleIdent(lval *yySymType) int {
	s := lval.ident
	if strings.EqualFold(s, "_filename") {
		lval.ident = "filename"
		return underscoreCS
	}
	// A character string literal may have an optional character set introducer and COLLATE clause:
	// [_charset_name]'string' [COLLATE collation_name]
	// See https://dev.mysql.com/doc/refman/5.7/en/charset-literal.html
	if !strings.HasPrefix(s, "_") {
		return identifier
	}
	cs, _, err := charset.GetCharsetInfo(s[1:])
	if err != nil {
		return identifier
	}
	lval.ident = cs
	return underscoreCS
}

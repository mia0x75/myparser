%{
package myparser

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/model"
	"github.com/mia0x75/myparser/opcode"
	"github.com/mia0x75/myparser/auth"
	"github.com/mia0x75/myparser/charset"
	"github.com/mia0x75/myparser/types"
)

// lastValueWindow carries the window part of a LAST_VALUE call.
type lastValueWindow struct {
	spec       ast.WindowSpec
	ignoreNull bool
}

%}


%union {
	offset int // offset
	item interface{}
	ident string
	expr ast.ExprNode
	statement ast.StmtNode
}


%token	<ident>
	/*yy:token "%c"     */	identifier      /* "identifier" */
	/*yy:token "_%c"    */  underscoreCS    /* "UNDERSCORE_CHARSET" */
	/*yy:token "\"%c\"" */	stringLit       /* "string literal" */
	singleAtIdentifier                      /* "identifier with single leading at" */
	doubleAtIdentifier                      /* "identifier with double leading at" */
	invalid                                 /* "a special token never used by parser, used by lexer to indicate error" */
	hintBegin                               /* "hintBegin is a virtual token for optimizer hint grammar" */
	hintEnd                                 /* "hintEnd is a virtual token for optimizer hint grammar" */
	andand                                  /* "&&" */
	pipes                                   /* "||" */


	/* The following tokens belong to ReservedKeyword. */
	add               /* "ADD" */
	all               /* "ALL" */
	alter             /* "ALTER" */
	analyze           /* "ANALYZE" */
	and               /* "AND" */
	as                /* "AS" */
	asc               /* "ASC" */
	between           /* "BETWEEN" */
	bigIntType        /* "BIGINT" */
	binaryType        /* "BINARY" */
	blobType          /* "BLOB" */
	both              /* "BOTH" */
	by                /* "BY" */
	call              /* "CALL" */
	cascade           /* "CASCADE" */
	caseKwd           /* "CASE" */
	change            /* "CHANGE" */
	character         /* "CHARACTER" */
	charType          /* "CHAR" */
	check             /* "CHECK" */
	collate           /* "COLLATE" */
	column            /* "COLUMN" */
	constraint        /* "CONSTRAINT" */
	convert           /* "CONVERT" */
	create            /* "CREATE" */
	cross             /* "CROSS" */
	cumeDist          /* "CUME_DIST" */
	currentDate       /* "CURRENT_DATE" */
	currentTime       /* "CURRENT_TIME" */
	currentTs         /* "CURRENT_TIMESTAMP" */
	currentUser       /* "CURRENT_USER" */
	currentRole       /* "CURRENT_ROLE" */
	database          /* "DATABASE" */
	databases         /* "DATABASES" */
	dayHour           /* "DAY_HOUR" */
	dayMicrosecond    /* "DAY_MICROSECOND" */
	dayMinute         /* "DAY_MINUTE" */
	daySecond         /* "DAY_SECOND" */
	decimalType       /* "DECIMAL" */
	defaultKwd        /* "DEFAULT" */
	delayed           /* "DELAYED" */
	deleteKwd         /* "DELETE" */
	denseRank         /* "DENSE_RANK" */
	desc              /* "DESC" */
	describe          /* "DESCRIBE" */
	distinct          /* "DISTINCT" */
	distinctRow       /* "DISTINCTROW" */
	div               /* "DIV" */
	doubleType        /* "DOUBLE" */
	drop              /* "DROP" */
	dual              /* "DUAL" */
	elseKwd           /* "ELSE" */
	enclosed          /* "ENCLOSED" */
	escaped           /* "ESCAPED" */
	exists            /* "EXISTS" */
	explain           /* "EXPLAIN" */
	except            /* "EXCEPT" */
	intersect         /* "INTERSECT" */
	falseKwd          /* "FALSE" */
	firstValue        /* "FIRST_VALUE" */
	floatType         /* "FLOAT" */
	forKwd            /* "FOR" */
	force             /* "FORCE" */
	foreign           /* "FOREIGN" */
	from              /* "FROM" */
	fulltext          /* "FULLTEXT" */
	generated         /* "GENERATED" */
	grant             /* "GRANT" */
	group             /* "GROUP" */
	groups            /* "GROUPS" */
	having            /* "HAVING" */
	highPriority      /* "HIGH_PRIORITY" */
	hourMicrosecond   /* "HOUR_MICROSECOND" */
	hourMinute        /* "HOUR_MINUTE" */
	hourSecond        /* "HOUR_SECOND" */
	ifKwd             /* "IF" */
	ignore            /* "IGNORE" */
	in                /* "IN" */
	index             /* "INDEX" */
	infile            /* "INFILE" */
	inner             /* "INNER" */
	integerType       /* "INTEGER" */
	interval          /* "INTERVAL" */
	into              /* "INTO" */
	is                /* "IS" */
	insert            /* "INSERT" */
	intType           /* "INT" */
	int1Type          /* "INT1" */
	int2Type          /* "INT2" */
	int3Type          /* "INT3" */
	int4Type          /* "INT4" */
	int8Type          /* "INT8" */
	join              /* "JOIN" */
	key               /* "KEY" */
	keys              /* "KEYS" */
	kill              /* "KILL" */
	lag               /* "LAG" */
	lastValue         /* "LAST_VALUE" */
	lead              /* "LEAD" */
	leading           /* "LEADING" */
	left              /* "LEFT" */
	like              /* "LIKE" */
	limit             /* "LIMIT" */
	lines             /* "LINES" */
	linear            /* "LINEAR" */
	load              /* "LOAD" */
	localTime         /* "LOCALTIME" */
	localTs           /* "LOCALTIMESTAMP" */
	lock              /* "LOCK" */
	longblobType      /* "LONGBLOB" */
	longtextType      /* "LONGTEXT" */
	lowPriority       /* "LOW_PRIORITY" */
	maxValue          /* "MAXVALUE" */
	mediumblobType    /* "MEDIUMBLOB" */
	mediumIntType     /* "MEDIUMINT" */
	mediumtextType    /* "MEDIUMTEXT" */
	minuteMicrosecond /* "MINUTE_MICROSECOND" */
	minuteSecond      /* "MINUTE_SECOND" */
	mod               /* "MOD" */
	not               /* "NOT" */
	noWriteToBinLog   /* "NO_WRITE_TO_BINLOG" */
	nthValue          /* "NTH_VALUE" */
	ntile             /* "NTILE" */
	null              /* "NULL" */
	numericType       /* "NUMERIC" */
	nvarcharType      /* "NVARCHAR" */
	on                /* "ON" */
	option            /* "OPTION" */
	optionally        /* "OPTIONALLY" */
	or                /* "OR" */
	order             /* "ORDER" */
	outer             /* "OUTER" */
	inout             /* "INOUT" */
	out               /* "OUT" */
	over              /* "OVER" */
	packKeys          /* "PACK_KEYS" */
	partition         /* "PARTITION" */
	percentRank       /* "PERCENT_RANK" */
	precisionType     /* "PRECISION" */
	primary           /* "PRIMARY" */
	procedure         /* "PROCEDURE" */
	shardRowIDBits    /* "SHARD_ROW_ID_BITS" */
	rangeKwd          /* "RANGE" */
	rank              /* "RANK" */
	read              /* "READ" */
	realType          /* "REAL" */
	references        /* "REFERENCES" */
	regexpKwd         /* "REGEXP" */
	rename            /* "RENAME" */
	repeat            /* "REPEAT" */
	replace           /* "REPLACE" */
	require           /* "REQUIRE" */
	restrict          /* "RESTRICT" */
	recursive         /* "RECURSIVE" */
	revoke            /* "REVOKE" */
	returning         /* "RETURNING" */
	right             /* "RIGHT" */
	rlike             /* "RLIKE" */
	row               /* "ROW" */
	rows              /* "ROWS" */
	rowNumber         /* "ROW_NUMBER" */
	secondMicrosecond /* "SECOND_MICROSECOND" */
	selectKwd         /* "SELECT" */
	set               /* "SET" */
	show              /* "SHOW" */
	smallIntType      /* "SMALLINT" */
	sql               /* "SQL" */
	sqlBigResult      /* "SQL_BIG_RESULT" */
	sqlCalcFoundRows  /* "SQL_CALC_FOUND_ROWS" */
	sqlSmallResult    /* "SQL_SMALL_RESULT" */
	ssl               /* "SSL" */
	starting          /* "STARTING" */
	straightJoin      /* "STRAIGHT_JOIN" */
	tableKwd          /* "TABLE" */
	stored            /* "STORED" */
	terminated        /* "TERMINATED" */
	then              /* "THEN" */
	tinyblobType      /* "TINYBLOB" */
	tinyIntType       /* "TINYINT" */
	tinytextType      /* "TINYTEXT" */
	to                /* "TO" */
	trailing          /* "TRAILING" */
	trigger           /* "TRIGGER" */
	trueKwd           /* "TRUE" */
	unique            /* "UNIQUE" */
	union             /* "UNION" */
	unlock            /* "UNLOCK" */
	unsigned          /* "UNSIGNED" */
	update            /* "UPDATE" */
	usage             /* "USAGE" */
	use               /* "USE" */
	using             /* "USING" */
	utcDate           /* "UTC_DATE" */
	utcTimestamp      /* "UTC_TIMESTAMP" */
	utcTime           /* "UTC_TIME" */
	values            /* "VALUES" */
	long              /* "LONG" */
	varcharType       /* "VARCHAR" */
	varbinaryType     /* "VARBINARY" */
	virtual           /* "VIRTUAL" */
	when              /* "WHEN" */
	where             /* "WHERE" */
	write             /* "WRITE" */
	window            /* "WINDOW" */
	with              /* "WITH" */
	xor               /* "XOR" */
	yearMonth         /* "YEAR_MONTH" */
	zerofill          /* "ZEROFILL" */
	lateral           /* "LATERAL" */
	jsonTable         /* "JSON_TABLE" */
	errorKwd          /* "ERROR" */
	natural           /* "NATURAL" */
	against           /* "AGAINST" */
	clone             /* "CLONE" */
	component         /* "COMPONENT" */
	diagnostics       /* "DIAGNOSTICS" */
	expansion         /* "EXPANSION" */
	get               /* "GET" */
	gtids             /* "GTIDS" */
	install           /* "INSTALL" */
	instance          /* "INSTANCE" */
	match             /* "MATCH" */
	replica           /* "REPLICA" */
	reset             /* "RESET" */
	shutdown          /* "SHUTDOWN" */
	sounds            /* "SOUNDS" */
	stacked           /* "STACKED" */
	stop              /* "STOP" */
	uninstall         /* "UNINSTALL" */
	spatial           /* "SPATIAL" */

	/* The following tokens belong to UnReservedKeyword. */
	account               /* "ACCOUNT" */
	action                /* "ACTION" */
	after                 /* "AFTER" */
	always                /* "ALWAYS" */
	algorithm             /* "ALGORITHM" */
	any                   /* "ANY" */
	ascii                 /* "ASCII" */
	autoIncrement         /* "AUTO_INCREMENT" */
	avgRowLength          /* "AVG_ROW_LENGTH" */
	avg                   /* "AVG" */
	backup                /* "BACKUP" */
	begin                 /* "BEGIN" */
	binlog                /* "BINLOG" */
	bitType               /* "BIT" */
	block                 /* "BLOCK" */
	booleanType           /* "BOOLEAN" */
	boolType              /* "BOOL" */
	btree                 /* "BTREE" */
	rtree                 /* "RTREE" */
	byteType              /* "BYTE" */
	cascaded              /* "CASCADED" */
	chain                 /* "CHAIN" */
	charsetKwd            /* "CHARSET" */
	checksum              /* "CHECKSUM" */
	cipher                /* "CIPHER" */
	cleanup               /* "CLEANUP" */
	client                /* "CLIENT" */
	coalesce              /* "COALESCE" */
	collation             /* "COLLATION" */
	columns               /* "COLUMNS" */
	comment               /* "COMMENT" */
	commit                /* "COMMIT" */
	committed             /* "COMMITTED" */
	compact               /* "COMPACT" */
	compressed            /* "COMPRESSED" */
	compression           /* "COMPRESSION" */
	connection            /* "CONNECTION" */
	consistent            /* "CONSISTENT" */
	context               /* "CONTEXT" */
	cpu                   /* "CPU" */
	current               /* "CURRENT" */
	day                   /* "DAY" */
	data                  /* "DATA" */
	dateType              /* "DATE" */
	datetimeType          /* "DATETIME" */
	deallocate            /* "DEALLOCATE" */
	definer               /* "DEFINER" */
	delayKeyWrite         /* "DELAY_KEY_WRITE" */
	disable               /* "DISABLE" */
	do                    /* "DO" */
	duplicate             /* "DUPLICATE" */
	dynamic               /* "DYNAMIC" */
	enable                /* "ENABLE" */
	end                   /* "END" */
	engine                /* "ENGINE" */
	engines               /* "ENGINES" */
	enum                  /* "ENUM" */
	event                 /* "EVENT" */
	events                /* "EVENTS" */
	escape                /* "ESCAPE" */
	exclude               /* "EXCLUDE" */
	exclusive             /* "EXCLUSIVE" */
	execute               /* "EXECUTE" */
	expire                /* "EXPIRE" */
	extended              /* "EXTENDED" */
	immediate             /* "IMMEDIATE" */
	persist               /* "PERSIST" */
	persistOnly           /* "PERSIST_ONLY" */
	persistent            /* "PERSISTENT" */
	statement             /* "STATEMENT" */
	faultsSym             /* "FAULTS" */
	fields                /* "FIELDS" */
	first                 /* "FIRST" */
	fixed                 /* "FIXED" */
	flush                 /* "FLUSH" */
	following             /* "FOLLOWING" */
	format                /* "FORMAT" */
	full                  /* "FULL" */
	function              /* "FUNCTION" */
	grants                /* "GRANTS" */
	hash                  /* "HASH" */
	hour                  /* "HOUR" */
	identified            /* "IDENTIFIED" */
	isolation             /* "ISOLATION" */
	issuer                /* "ISSUER" */
	indexes               /* "INDEXES" */
	invoker               /* "INVOKER" */
	io                    /* "IO" */
	ipc                   /* "IPC" */
	jsonType              /* "JSON" */
	serialType            /* "SERIAL" */
	keyBlockSize          /* "KEY_BLOCK_SIZE" */
	local                 /* "LOCAL" */
	last                  /* "LAST" */
	less                  /* "LESS" */
	level                 /* "LEVEL" */
	master                /* "MASTER" */
	microsecond           /* "MICROSECOND" */
	minute                /* "MINUTE" */
	mode                  /* "MODE" */
	modify                /* "MODIFY" */
	month                 /* "MONTH" */
	maxRows               /* "MAX_ROWS" */
	maxConnectionsPerHour /* "MAX_CONNECTIONS_PER_HOUR" */
	maxQueriesPerHour     /* "MAX_QUERIES_PER_HOUR" */
	maxUpdatesPerHour     /* "MAX_UPDATES_PER_HOUR" */
	maxUserConnections    /* "MAX_USER_CONNECTIONS" */
	memory                /* "MEMORY" */
	merge                 /* "MERGE" */
	minRows               /* "MIN_ROWS" */
	names                 /* "NAMES" */
	national              /* "NATIONAL" */
	never                 /* "NEVER" */
	no                    /* "NO" */
	none                  /* "NONE" */
	nulls                 /* "NULLS" */
	offset                /* "OFFSET" */
	only                  /* "ONLY" */
	others                /* "OTHERS" */
	pageSym               /* "PAGE" */
	password              /* "PASSWORD" */
	partitions            /* "PARTITIONS" */
	period                /* "PERIOD" */
	pipesAsOr
	plugins               /* "PLUGINS" */
	preceding             /* "PRECEDING" */
	prepare               /* "PREPARE" */
	privileges            /* "PRIVILEGES" */
	process               /* "PROCESS" */
	processlist           /* "PROCESSLIST" */
	profile               /* "PROFILE" */
	profiles              /* "PROFILES" */
	quarter               /* "QUARTER" */
	query                 /* "QUERY" */
	queries               /* "QUERIES" */
	quick                 /* "QUICK" */
	recover               /* "RECOVER" */
	redundant             /* "REDUNDANT" */
	reload                /* "RELOAD" */
	repeatable            /* "REPEATABLE" */
	respect               /* "RESPECT" */
	replication           /* "REPLICATION" */
	reverse               /* "REVERSE" */
	role                  /* "ROLE" */
	rollback              /* "ROLLBACK" */
	routine               /* "ROUTINE" */
	rowCount              /* "ROW_COUNT" */
	rowFormat             /* "ROW_FORMAT" */
	second                /* "SECOND" */
	security              /* "SECURITY" */
	separator             /* "SEPARATOR" */
	serializable          /* "SERIALIZABLE" */
	session               /* "SESSION" */
	share                 /* "SHARE" */
	shared                /* "SHARED" */
	signed                /* "SIGNED" */
	slave                 /* "SLAVE" */
	slow                  /* "SLOW" */
	snapshot              /* "SNAPSHOT" */
	sqlBufferResult       /* "SQL_BUFFER_RESULT" */
	sqlCache              /* "SQL_CACHE" */
	sqlNoCache            /* "SQL_NO_CACHE" */
	start                 /* "START" */
	statsPersistent       /* "STATS_PERSISTENT" */
	status                /* "STATUS" */
	swaps                 /* "SWAPS" */
	switchesSym           /* "SWITCHES" */
	open                  /* "OPEN" */
	source                /* "SOURCE" */
	subject               /* "SUBJECT" */
	subpartition          /* "SUBPARTITION" */
	subpartitions         /* "SUBPARTITIONS" */
	super                 /* "SUPER" */
	some                  /* "SOME" */
	global                /* "GLOBAL" */
	tables                /* "TABLES" */
	tablespace            /* "TABLESPACE" */
	temporary             /* "TEMPORARY" */
	temptable             /* "TEMPTABLE" */
	textType              /* "TEXT" */
	than                  /* "THAN" */
	timeType              /* "TIME" */
	ties                  /* "TIES" */
	timestampType         /* "TIMESTAMP" */
	transaction           /* "TRANSACTION" */
	triggers              /* "TRIGGERS" */
	truncate              /* "TRUNCATE" */
	unbounded             /* "UNBOUNDED" */
	uncommitted           /* "UNCOMMITTED" */
	unknown               /* "UNKNOWN" */
	user                  /* "USER" */
	undefined             /* "UNDEFINED" */
	value                 /* "VALUE" */
	variables             /* "VARIABLES" */
	view                  /* "VIEW" */
	warnings              /* "WARNINGS" */
	within                /* "WITHIN" */
	identSQLErrors        /* "ERRORS" */
	week                  /* "WEEK" */
	x509                  /* "X509" */
	yearType              /* "YEAR" */
	uuidType              /* "UUID" */
	inet4Type             /* "INET4" */
	inet6Type             /* "INET6" */
	purge                 /* "PURGE" */
	logs                  /* "LOGS" */
	log                   /* "LOG" */
	before                /* "BEFORE" */
	buckets               /* "BUCKETS" */
	at                    /* "AT" */
	completion            /* "COMPLETION" */
	contains              /* "CONTAINS" */
	deterministic         /* "DETERMINISTIC" */
	each                  /* "EACH" */
	ends                  /* "ENDS" */
	every                 /* "EVERY" */
	follows               /* "FOLLOWS" */
	language              /* "LANGUAGE" */
	modifies              /* "MODIFIES" */
	of                    /* "OF" */
	precedes              /* "PRECEDES" */
	preserve              /* "PRESERVE" */
	reads                 /* "READS" */
	returns               /* "RETURNS" */
	returnKwd             /* "RETURN" */
	schedule              /* "SCHEDULE" */
	starts                /* "STARTS" */
	atomic                /* "ATOMIC" */
	close                 /* "CLOSE" */
	condition             /* "CONDITION" */
	continueKwd           /* "CONTINUE" */
	cursor                /* "CURSOR" */
	declare               /* "DECLARE" */
	elseif                /* "ELSEIF" */
	exit                  /* "EXIT" */
	fetch                 /* "FETCH" */
	found                 /* "FOUND" */
	handler               /* "HANDLER" */
	iterate               /* "ITERATE" */
	leave                 /* "LEAVE" */
	loop                  /* "LOOP" */
	resignal              /* "RESIGNAL" */
	signal                /* "SIGNAL" */
	sqlException          /* "SQLEXCEPTION" */
	sqlState              /* "SQLSTATE" */
	sqlWarning            /* "SQLWARNING" */
	undo                  /* "UNDO" */
	until                 /* "UNTIL" */
	while                 /* "WHILE" */
	nowait                /* "NOWAIT" */
	skip                  /* "SKIP" */
	locked                /* "LOCKED" */
	rollup                /* "ROLLUP" */
	work                  /* "WORK" */
	release               /* "RELEASE" */
	savepoint             /* "SAVEPOINT" */
	sequence              /* "SEQUENCE" */
	increment             /* "INCREMENT" */
	minvalue              /* "MINVALUE" */
	cache                 /* "CACHE" */
	cycle                 /* "CYCLE" */
	nomaxvalue            /* "NOMAXVALUE" */
	nominvalue            /* "NOMINVALUE" */
	nocycle               /* "NOCYCLE" */
	nocache               /* "NOCACHE" */
	aggregate             /* "AGGREGATE" */
	soname                /* "SONAME" */
	stringType            /* "STRING" */
	xa                    /* "XA" */
	resume                /* "RESUME" */
	migrate               /* "MIGRATE" */
	suspend               /* "SUSPEND" */
	one                   /* "ONE" */
	phase                 /* "PHASE" */
	cube                  /* "CUBE" */
	server                /* "SERVER" */
	datafile              /* "DATAFILE" */
	wrapper               /* "WRAPPER" */
	options               /* "OPTIONS" */
	fileBlockSize         /* "FILE_BLOCK_SIZE" */
	extentSize            /* "EXTENT_SIZE" */
	initialSize           /* "INITIAL_SIZE" */
	autoextendSize        /* "AUTOEXTEND_SIZE" */
	maxSize               /* "MAX_SIZE" */
	nodegroup             /* "NODEGROUP" */
	wait                  /* "WAIT" */
	resource              /* "RESOURCE" */
	logfile               /* "LOGFILE" */
	vcpu                  /* "VCPU" */
	undoBufferSize        /* "UNDO_BUFFER_SIZE" */
	redoBufferSize        /* "REDO_BUFFER_SIZE" */
	threadPriority        /* "THREAD_PRIORITY" */
	undofile              /* "UNDOFILE" */
	typeSym               /* "TYPE" */
	systemSym             /* "SYSTEM" */
	reorganize            /* "REORGANIZE" */
	exchange              /* "EXCHANGE" */
	discard               /* "DISCARD" */
	import_               /* "IMPORT" */
	validation            /* "VALIDATION" */
	without               /* "WITHOUT" */
	remove                /* "REMOVE" */
	partitioning          /* "PARTITIONING" */
	outfile               /* "OUTFILE" */
	dumpfile              /* "DUMPFILE" */
	repair                /* "REPAIR" */
	optimize              /* "OPTIMIZE" */
	upgrade               /* "UPGRADE" */
	visible               /* "VISIBLE" */
	invisible             /* "INVISIBLE" */
	traditional           /* "TRADITIONAL" */
	tree                  /* "TREE" */
	pathKwd              /* "PATH" */
	nested                /* "NESTED" */
	ordinality            /* "ORDINALITY" */
	emptyKwd              /* "EMPTY" */
	channel               /* "CHANNEL" */
	directory             /* "DIRECTORY" */
	ioThread              /* "IO_THREAD" */
	leaves                /* "LEAVES" */
	memberKwd             /* "MEMBER" */
	next                  /* "NEXT" */
	plugin                /* "PLUGIN" */
	prev                  /* "PREV" */
	replicas              /* "REPLICAS" */
	slaves                /* "SLAVES" */
	sqlThread             /* "SQL_THREAD" */
	xml                   /* "XML" */
	arrayKwd              /* "ARRAY" */
	objectKwd             /* "OBJECT" */
	scalarKwd             /* "SCALAR" */
	concurrent            /* "CONCURRENT" */
	masterHost            /* "MASTER_HOST" */
	sourceHost            /* "SOURCE_HOST" */
	masterBind            /* "MASTER_BIND" */
	sourceBind            /* "SOURCE_BIND" */
	masterUser            /* "MASTER_USER" */
	sourceUser            /* "SOURCE_USER" */
	masterPassword        /* "MASTER_PASSWORD" */
	sourcePassword        /* "SOURCE_PASSWORD" */
	masterPort            /* "MASTER_PORT" */
	sourcePort            /* "SOURCE_PORT" */
	masterConnectRetry    /* "MASTER_CONNECT_RETRY" */
	sourceConnectRetry    /* "SOURCE_CONNECT_RETRY" */
	masterRetryCount      /* "MASTER_RETRY_COUNT" */
	sourceRetryCount      /* "SOURCE_RETRY_COUNT" */
	masterDelay           /* "MASTER_DELAY" */
	sourceDelay           /* "SOURCE_DELAY" */
	masterSSL             /* "MASTER_SSL" */
	sourceSSL             /* "SOURCE_SSL" */
	masterSSLCA           /* "MASTER_SSL_CA" */
	sourceSSLCA           /* "SOURCE_SSL_CA" */
	masterSSLCAPath       /* "MASTER_SSL_CAPATH" */
	sourceSSLCAPath       /* "SOURCE_SSL_CAPATH" */
	masterSSLCert         /* "MASTER_SSL_CERT" */
	sourceSSLCert         /* "SOURCE_SSL_CERT" */
	masterSSLCipher       /* "MASTER_SSL_CIPHER" */
	sourceSSLCipher       /* "SOURCE_SSL_CIPHER" */
	masterSSLKey          /* "MASTER_SSL_KEY" */
	sourceSSLKey          /* "SOURCE_SSL_KEY" */
	masterSSLVerifyServerCert /* "MASTER_SSL_VERIFY_SERVER_CERT" */
	sourceSSLVerifyServerCert /* "SOURCE_SSL_VERIFY_SERVER_CERT" */
	masterSSLCRL          /* "MASTER_SSL_CRL" */
	sourceSSLCRL          /* "SOURCE_SSL_CRL" */
	masterSSLCRLPath      /* "MASTER_SSL_CRLPATH" */
	sourceSSLCRLPath      /* "SOURCE_SSL_CRLPATH" */
	masterTLSVersion      /* "MASTER_TLS_VERSION" */
	sourceTLSVersion      /* "SOURCE_TLS_VERSION" */
	masterTLSCipherSuites /* "MASTER_TLS_CIPHERSUITES" */
	sourceTLSCipherSuites /* "SOURCE_TLS_CIPHERSUITES" */
	masterPublicKeyPath   /* "MASTER_PUBLIC_KEY_PATH" */
	sourcePublicKeyPath   /* "SOURCE_PUBLIC_KEY_PATH" */
	getMasterPublicKey    /* "GET_MASTER_PUBLIC_KEY" */
	getSourcePublicKey    /* "GET_SOURCE_PUBLIC_KEY" */
	masterHeartbeatPeriod /* "MASTER_HEARTBEAT_PERIOD" */
	sourceHeartbeatPeriod /* "SOURCE_HEARTBEAT_PERIOD" */
	masterCompressionAlgorithms /* "MASTER_COMPRESSION_ALGORITHMS" */
	sourceCompressionAlgorithms /* "SOURCE_COMPRESSION_ALGORITHMS" */
	masterZstdCompressionLevel /* "MASTER_ZSTD_COMPRESSION_LEVEL" */
	sourceZstdCompressionLevel /* "SOURCE_ZSTD_COMPRESSION_LEVEL" */
	masterAutoPosition    /* "MASTER_AUTO_POSITION" */
	sourceAutoPosition    /* "SOURCE_AUTO_POSITION" */
	sourceConnectionAutoFailover /* "SOURCE_CONNECTION_AUTO_FAILOVER" */
	requireRowFormat      /* "REQUIRE_ROW_FORMAT" */
	requireTablePrimaryKeyCheck /* "REQUIRE_TABLE_PRIMARY_KEY_CHECK" */
	assignGTIDsToAnonymousTransactions /* "ASSIGN_GTIDS_TO_ANONYMOUS_TRANSACTIONS" */
	privilegeChecksUser   /* "PRIVILEGE_CHECKS_USER" */
	gtidOnly              /* "GTID_ONLY" */
	networkNamespace      /* "NETWORK_NAMESPACE" */
	ignoreServerIDs       /* "IGNORE_SERVER_IDS" */
	doDomainIDs           /* "DO_DOMAIN_IDS" */
	ignoreDomainIDs       /* "IGNORE_DOMAIN_IDS" */
	masterUseGTID         /* "MASTER_USE_GTID" */
	masterDemoteToSlave   /* "MASTER_DEMOTE_TO_SLAVE" */
	masterDemoteToReplica /* "MASTER_DEMOTE_TO_REPLICA" */
	masterLogFile         /* "MASTER_LOG_FILE" */
	sourceLogFile         /* "SOURCE_LOG_FILE" */
	masterLogPos          /* "MASTER_LOG_POS" */
	sourceLogPos          /* "SOURCE_LOG_POS" */
	relayLogFile          /* "RELAY_LOG_FILE" */
	relayLogPos           /* "RELAY_LOG_POS" */
	masterGTIDPos         /* "MASTER_GTID_POS" */
	sqlBeforeGTIDs        /* "SQL_BEFORE_GTIDS" */
	sqlAfterGTIDs         /* "SQL_AFTER_GTIDS" */
	sqlAfterMTSGaps       /* "SQL_AFTER_MTS_GAPS" */
	defaultAuth           /* "DEFAULT_AUTH" */
	pluginDir             /* "PLUGIN_DIR" */
	currentPos            /* "CURRENT_POS" */
	slavePos              /* "SLAVE_POS" */
	replicaPos            /* "REPLICA_POS" */
	streamKwd             /* "STREAM" */
	generateKwd           /* "GENERATE" */
	off                   /* "OFF" */
	analyse               /* "ANALYSE" */
	geometryType          /* "GEOMETRY" */
	pointType             /* "POINT" */
	lineStringType        /* "LINESTRING" */
	polygonType           /* "POLYGON" */
	multiPointType        /* "MULTIPOINT" */
	multiLineStringType   /* "MULTILINESTRING" */
	multiPolygonType      /* "MULTIPOLYGON" */
	geometryCollectionType /* "GEOMETRYCOLLECTION" */
	srid                  /* "SRID" */
	zone                  /* "ZONE" */

	/* The following tokens belong to NotKeywordToken. */
	addDate            /* "ADDDATE" */
	bitAnd             /* "BIT_AND" */
	bitOr              /* "BIT_OR" */
	bitXor             /* "BIT_XOR" */
	cast               /* "CAST" */
	copyKwd            /* "COPY" */
	count              /* "COUNT" */
	curTime            /* "CURTIME" */
	dateAdd            /* "DATE_ADD" */
	dateSub            /* "DATE_SUB" */
	extract            /* "EXTRACT" */
	getFormat          /* "GET_FORMAT" */
	groupConcat        /* "GROUP_CONCAT" */
	next_row_id        /* "NEXT_ROW_ID" */
	inplace            /* "INPLACE" */
	instant            /* "INSTANT" */
	internal           /* "INTERNAL" */
	min                /* "MIN" */
	max                /* "MAX" */
	maxExecutionTime   /* "MAX_EXECUTION_TIME" */
	now                /* "NOW" */
	position           /* "POSITION" */
	recent             /* "RECENT" */
	std                /* "STD" */
	stddev             /* "STDDEV" */
	stddevPop          /* "STDDEV_POP" */
	stddevSamp         /* "STDDEV_SAMP" */
	subDate            /* "SUBDATE" */
	sum                /* "SUM" */
	substring          /* "SUBSTRING" */
	timestampAdd       /* "TIMESTAMPADD" */
	timestampDiff      /* "TIMESTAMPDIFF" */
	tokudbDefault      /* "TOKUDB_DEFAULT" */
	tokudbFast         /* "TOKUDB_FAST" */
	tokudbLzma         /* "TOKUDB_LZMA" */
	tokudbQuickLZ      /* "TOKUDB_QUICKLZ" */
	tokudbSnappy       /* "TOKUDB_SNAPPY" */
	tokudbSmall        /* "TOKUDB_SMALL" */
	tokudbUncompressed /* "TOKUDB_UNCOMPRESSED" */
	tokudbZlib         /* "TOKUDB_ZLIB" */
	top                /* "TOP" */
	trim               /* "TRIM" */
	variance           /* "VARIANCE" */
	varPop             /* "VAR_POP" */
	varSamp            /* "VAR_SAMP" */
	weightString       /* "WEIGHT_STRING" */


	builtinAddDate
	builtinBitAnd
	builtinBitOr
	builtinBitXor
	builtinCast
	builtinCount
	builtinCurDate
	builtinCurTime
	builtinDateAdd
	builtinDateSub
	builtinExtract
	builtinGroupConcat
	builtinJSONArrayAgg
	builtinMax
	builtinMin
	builtinNow
	builtinPosition
	builtinSubDate
	builtinSubstring
	builtinSum
	builtinSysDate
	builtinStddevPop
	builtinStddevSamp
	builtinTrim
	builtinUser
	builtinVarPop
	builtinVarSamp

%token	<item>

	/*yy:token "1.%d"   */	floatLit        /* "floating-point literal" */
	/*yy:token "1.%d"   */	decLit          /* "decimal literal" */
	/*yy:token "%d"     */	intLit          /* "integer literal" */
	/*yy:token "%x"     */	hexLit          /* "hexadecimal literal" */
	/*yy:token "%b"     */	bitLit          /* "bit literal" */

	andnot       /* "&^" */
	assignmentEq /* ":=" */
	eq           /* "=" */
	ge           /* ">=" */
	le           /* "<=" */
	jss          /* "->" */
	juss         /* "->>" */
	lsh          /* "<<" */
	neq          /* "!=" */
	neqSynonym   /* "<>" */
	nulleq       /* "<=>" */
	paramMarker  /* "?" */
	rsh          /* ">>" */
	rangeDots    /* ".." */


%token not2

/* RESPECT NULLS and IGNORE NULLS are the only two-word keywords. The scanner
   merges them, so they are not part of the single word keyword lists. */
%token respectNulls
%token ignoreNulls


%type	<expr>
	DiagnosticsConditionNumber      /* Literal or variable condition number */
	ReplicationStringOrDefaultValue /* string or MariaDB DEFAULT */
	ReplicationStringOrNullValue    /* TLS cipher suites string or NULL */
	ReplicationNumberOrDefaultValue /* unsigned number or MariaDB DEFAULT */
	ReplicationUnsignedOrDefaultValue /* unsigned/hexadecimal number or MariaDB DEFAULT */
	ReplicationBoolValue            /* numeric or MariaDB boolean */
	ReplicationBoolOrDefaultValue   /* boolean or MariaDB DEFAULT */
	ReplicationBinaryValue          /* numeric value 0 or 1 */
	ReplicationStrictBinaryValue    /* integer value 0 or 1 */
	ReplicationUnsignedNumber       /* unsigned numeric literal, optionally with MariaDB '+' */
	ReplicationNumber               /* numeric literal, optionally with MariaDB '+' */
	ReplicationUnsignedValue        /* unsigned numeric or hexadecimal literal */
	ReplicationHexNumber            /* hexadecimal integer normalized to decimal */
	ReplicationPrimaryKeyCheckValue /* STREAM|ON|OFF|GENERATE */
	ReplicationAssignGTIDValue      /* OFF|LOCAL|UUID string */
	ReplicationPrivilegeValue       /* account or NULL */
	ReplicationUseGTIDValue         /* CURRENT_POS|SLAVE_POS|REPLICA_POS|NO|DEFAULT */
	Expression                      /* expression */
	MaxValueOrExpression            /* maxvalue or expression */
	BoolPri                         /* boolean primary expression */
	ExprOrDefault                   /* expression or default */
	PredicateExpr                   /* Predicate expression factor */
	SetExpr                         /* Set variable statement value's expression */
	BitExpr                         /* bit expression */
	SimpleExpr                      /* simple expression */
	SimpleIdent                     /* Simple Identifier expression */
	SumExpr                         /* aggregate functions */
	FunctionCallGeneric             /* Function call with Identifier */
	FunctionCallKeyword             /* Function call with keyword as function name */
	FunctionCallNonKeyword          /* Function call with nonkeyword as function name */
	Literal                         /* literal value */
	Variable                        /* User or system variable */
	SystemVariable                  /* System defined variable name */
	UserVariable                    /* User defined variable name */
	SubSelect                       /* Sub Select */
	StringLiteral                   /* text literal */
	ExpressionOpt                   /* Optional expression */
	SignedLiteral                   /* Literal or NumLiteral with sign */
	DefaultValueExpr                /* DefaultValueExpr(Now or Signed Literal) */
	SequenceSignedInt               /* signed integer sequence option value */
	SequenceStartValue              /* signed or unsigned integer sequence value */
	SequenceBoundValue              /* sequence bound allowing truncated decimals */
	WeightStringParam               /* numeric parameter in four-argument WEIGHT_STRING */
	NowSymOptionFraction            /* NowSym with optional fraction part */


%type	<statement>
	AlterTableStmt                  /* Alter table statement */
	AlterDatabaseStmt               /* Alter database statement */
	AlterUserStmt                   /* Alter user statement */
	AlterFunctionStmt               /* ALTER FUNCTION statement */
	AlterProcedureStmt              /* ALTER PROCEDURE statement */
	AlterEventStmt                  /* ALTER EVENT statement */
	AlterViewStmt                   /* Alter view statement */
	AnalyzeTableStmt                /* Analyze table statement */
	BackupStmt                      /* MariaDB BACKUP LOCK / UNLOCK */
	BeginTransactionStmt            /* BEGIN TRANSACTION statement */
	BinlogStmt                      /* Binlog base64 statement */
	CallStmt                        /* CALL statement */
	CommitStmt                      /* COMMIT statement */
	CreateTableStmt                 /* CREATE TABLE statement */
	CreateFunctionStmt              /* CREATE FUNCTION statement */
	CreateProcedureStmt             /* CREATE PROCEDURE statement */
	CreateTriggerStmt               /* CREATE TRIGGER statement */
	CreateEventStmt                 /* CREATE EVENT statement */
	CreateViewStmt                  /* CREATE VIEW stetement */
	CreateUserStmt                  /* CREATE User statement */
	CreateRoleStmt                  /* CREATE Role statement */
	CreateDatabaseStmt              /* Create Database Statement */
	CreateIndexStmt                 /* CREATE INDEX statement */
	DoStmt                          /* Do statement */
	DropDatabaseStmt                /* DROP DATABASE statement */
	DropIndexStmt                   /* DROP INDEX statement */
	DropTableStmt                   /* DROP TABLE statement */
	DropFunctionStmt                /* DROP FUNCTION statement */
	DropProcedureStmt               /* DROP PROCEDURE statement */
	DropTriggerStmt                 /* DROP TRIGGER statement */
	DropEventStmt                   /* DROP EVENT statement */
	DropUserStmt                    /* DROP USER */
	DropRoleStmt                    /* DROP ROLE */
	DropViewStmt                    /* DROP VIEW statement */
	DeallocateStmt                  /* Deallocate prepared statement */
	DeleteFromStmt                  /* DELETE FROM statement */
	EmptyStmt                       /* empty statement */
	ExecuteStmt                     /* Execute statement */
	ExplainStmt                     /* EXPLAIN statement */
	ExplainableStmt                 /* explainable statement */
	ExplainTarget                   /* explained statement with INTO/FOR SCHEMA clauses */
	ExplainTargetNoInto             /* explained statement with FOR SCHEMA only */
	FlushStmt                       /* Flush statement */
	GrantStmt                       /* Grant statement */
	GrantRoleStmt                   /* Grant role statement */
	InsertIntoStmt                  /* INSERT INTO statement */
	KillStmt                        /* Kill statement */
	LoadDataStmt                    /* Load data statement */
	ChangeMasterStmt                /* CHANGE MASTER TO statement */
	ReplicaControlStmt              /* START/STOP REPLICA statement */
	ResetStmt                       /* RESET statement */
	ShutdownStmt                    /* SHUTDOWN statement */
	HandlerStmt                     /* HANDLER statement */
	CacheIndexStmt                  /* CACHE INDEX statement */
	LoadIndexStmt                   /* LOAD INDEX INTO CACHE statement */
	LoadXMLStmt                     /* LOAD XML statement */
	RenameUserStmt                  /* RENAME USER statement */
	ImportTableStmt                 /* IMPORT TABLE statement */
	InstallPluginStmt               /* INSTALL PLUGIN statement */
	UninstallPluginStmt             /* UNINSTALL PLUGIN statement */
	InstallComponentStmt            /* INSTALL COMPONENT statement */
	UninstallComponentStmt          /* UNINSTALL COMPONENT statement */
	CloneStmt                       /* CLONE statement */
	GetDiagnosticsStmt              /* GET DIAGNOSTICS statement */
	LockTablesStmt                  /* Lock tables statement */
	PreparedStmt                    /* PreparedStmt */
	SelectStmt                      /* SELECT statement */
	ValuesStmt                      /* VALUES statement */
	ValuesSetOprStmt                /* set operation containing VALUES */
	ValuesSetOprList                /* unfinalized set operation containing VALUES */
	TableStmt                       /* TABLE statement */
	WithSelectStmt                  /* WITH ... SELECT statement */
	WithUpdateStmt                  /* WITH ... UPDATE statement */
	WithDeleteStmt                  /* WITH ... DELETE statement */
	ViewSelect                      /* SELECT, set operation, or CTE for a view */
	RenameTableStmt                 /* rename table statement */
	ReplaceIntoStmt                 /* REPLACE INTO statement */
	RevokeStmt                      /* Revoke statement */
	RevokeRoleStmt                  /* Revoke role statement */
	RollbackStmt                    /* ROLLBACK statement */
	SavepointStmt                   /* SAVEPOINT statement */
	ReleaseSavepointStmt            /* RELEASE SAVEPOINT statement */
	CreateSequenceStmt              /* CREATE SEQUENCE statement */
	DropSequenceStmt                /* DROP SEQUENCE statement */
	CreateTablespaceStmt            /* CREATE TABLESPACE statement */
	AlterTablespaceStmt             /* ALTER TABLESPACE statement */
	DropTablespaceStmt              /* DROP TABLESPACE statement */
	CreateServerStmt                /* CREATE SERVER statement */
	AlterServerStmt                 /* ALTER SERVER statement */
	DropServerStmt                  /* DROP SERVER statement */
	CreateResourceGroupStmt         /* CREATE RESOURCE GROUP statement */
	AlterResourceGroupStmt          /* ALTER RESOURCE GROUP statement */
	DropResourceGroupStmt           /* DROP RESOURCE GROUP statement */
	CreateLogfileGroupStmt          /* CREATE LOGFILE GROUP statement */
	AlterLogfileGroupStmt           /* ALTER LOGFILE GROUP statement */
	DropLogfileGroupStmt            /* DROP LOGFILE GROUP statement */
	CheckTableStmt                  /* CHECK TABLE statement */
	RepairTableStmt                 /* REPAIR TABLE statement */
	OptimizeTableStmt               /* OPTIMIZE TABLE statement */
	ChecksumTableStmt               /* CHECKSUM TABLE statement */
	XAStmt                          /* XA transaction statement */
	SetStmt                         /* Set variable statement */
	SetStatementStmt                /* MariaDB SET STATEMENT ... FOR */
	SetRoleStmt                     /* Set active role statement */
	SetDefaultRoleStmt              /* Set default statement for some user */
	ShowStmt                        /* Show engines/databases/tables/user/columns/warnings/status statement */
	Statement                       /* statement */
	TruncateTableStmt               /* TRUNCATE TABLE statement */
	UnlockTablesStmt                /* Unlock tables statement */
	UpdateStmt                      /* UPDATE statement */
	SetOprStmt                      /* Union/Except/Intersect select statement */
	SetOprStmtWoutLimitOrderBy      /* Set operation select statement without limit/order by */
	SetOprStmtWithLimitOrderBy      /* Set operation select statement with limit/order by */
	ParenthesizedQueryStmt          /* parenthesized top-level query */
	ProcedureAnalyseStmt            /* query with MariaDB PROCEDURE ANALYSE tail */
	UseStmt                         /* USE statement */
	PurgeStmt                       /* PURGE statement */
	ReturnStmt                      /* RETURN statement */
	SpStmt                          /* stored program statement */
	SpStmtOpt                       /* optional stored program statement */
	SpBeginStmt                     /* BEGIN ... END compound statement */
	SpIfStmt                        /* IF ... END IF statement */
	SpCaseStmt                      /* CASE ... END CASE statement */
	SpWhileStmt                     /* WHILE ... END WHILE statement */
	SpRepeatStmt                    /* REPEAT ... END REPEAT statement */
	SpLoopStmt                      /* LOOP ... END LOOP statement */
	SpForStmt                       /* MariaDB FOR ... END FOR statement */
	SpForBody                       /* unlabeled MariaDB FOR body */
	TopLevelForStmt                 /* top-level unlabeled MariaDB FOR statement */
	SpLeaveStmt                     /* LEAVE statement */
	SpIterateStmt                   /* ITERATE statement */
	SpDeclareStmt                   /* DECLARE statement */
	SpOpenStmt                      /* OPEN cursor statement */
	SpCloseStmt                     /* CLOSE cursor statement */
	SpFetchStmt                     /* FETCH cursor statement */
	SpSignalStmt                    /* SIGNAL statement */
	SpResignalStmt                  /* RESIGNAL statement */
	SpStartTransactionStmt          /* START TRANSACTION statement */


%type	<item>
	BinaryOrMaster                  /* BINARY or MASTER keyword for PURGE */
	KillOption                      /* optional KILL CONNECTION/QUERY */
	CTEBody                         /* CTE query body */
	ParenQueryTail                  /* ORDER BY/LIMIT tail of a parenthesized query level */
	AlterAlgorithm                  /* Alter table algorithm */
	AlterTableOptionListOpt         /* alter table option list opt */
	AlterTableSpec                  /* Alter table specification */
	AlterTableSpecList              /* Alter table specification list */
	AnyOrAll                        /* Any or All for subquery */
	Assignment                      /* assignment */
	AssignmentList                  /* assignment list */
	BucketsOpt                      /* analyze partitions with buckets option */
	AnalyzePersistentOpt            /* MariaDB PERSISTENT statistics clause */
	AnalyzeColumnList               /* ALL or a parenthesized column list */
	AnalyzeIndexList                /* ALL or a parenthesized index list */
	AuthOption                      /* User auth option */
	AuthString                      /* Password string value */
	OptionalBraces                  /* optional braces */
	CastType                        /* Cast function target type */
	CastDatetimeType                /* DATETIME target for CAST ... AT TIME ZONE */
	CastTimeZone                    /* CAST AT TIME ZONE / AT LOCAL clause */
	CharsetName                     /* Character set name */
	SetCharsetName                  /* SET CHARACTER SET name or legacy conversion pair */
	ClearPasswordExpireOptions      /* Clear password expire options */
	CollationName                   /* Collation name */
	ColumnDef                       /* table column definition */
	ColumnDefList                   /* table column definition list */
	ColumnName                      /* column name */
	ColumnNameOrUserVariable        /* column name or user variable */
	ColumnNameList                  /* column name list */
	ColumnNameOrUserVariableList    /* column name or user variable list */
	ColumnList                      /* column list */
	ColumnNameListOpt               /* column name list opt */
	ColumnNameOrUserVarListOpt             /* column name or user vairiabe list opt */
	ColumnNameOrUserVarListOptWithBrackets /* column name or user variable list opt with brackets */
	ColumnSetValue                  /* insert statement set value by column name */
	ColumnSetValueList              /* insert statement set value by column name list */
	CompareOp                       /* Compare opcode */
	ColumnOption                    /* column definition option */
	ColumnOptionList                /* column definition option list */
	VirtualOrStored                 /* indicate generated column is stored or not */
	ColumnOptionListOpt             /* optional column definition option list */
	ConnectionOption                /* single connection options */
	ConnectionOptionList            /* connection options for CREATE USER statement */
	ConnectionOptions               /* optional connection options for CREATE USER statement */
	Constraint                      /* table constraint */
	ConstraintElem                  /* table constraint element */
	ConstraintKeywordOpt            /* Constraint Keyword or empty */
	CreateIndexStmtUnique           /* CREATE INDEX optional UNIQUE clause */
	CreateTableOptionList           /* nonempty create table option list */
	CreateTableSelectOpt            /* Select/Union statement in CREATE TABLE ... SELECT */
	CreateTableBody                 /* table definition and/or query */
	CreateTableTail                 /* table options, partitioning and optional query */
	TemporaryOpt                    /* optional TEMPORARY keyword */
	DatabaseOption                  /* CREATE Database specification */
	DatabaseOptionList              /* CREATE Database specification list */
	DatabaseOptionListOpt           /* CREATE Database specification list opt */
	DBName                          /* Database Name */
	DistinctOpt                     /* Explicit distinct option */
	DefaultFalseDistinctOpt         /* Distinct option which defaults to false */
	DefaultTrueDistinctOpt          /* Distinct option which defaults to true */
	BuggyDefaultFalseDistinctOpt    /* Distinct option which accepts DISTINCT ALL and defaults to false */
	RequireClause                   /* Encrypted connections options */
	EqOpt                           /* = or empty */
	EscapedTableRef                 /* escaped table reference */
	ExpressionList                  /* expression list */
	MaxValueOrExpressionList        /* maxvalue or expression list */
	ExpressionListOpt               /* expression list opt */
	ExecuteUsingOpt                 /* optional EXECUTE parameter expressions */
	ExecuteUsingList                /* EXECUTE expressions or DEFAULT */
	FuncDatetimePrecListOpt         /* Function datetime precision list opt */
	FuncDatetimePrecList            /* Function datetime precision list */
	Field                           /* field expression */
	Fields                          /* Fields clause */
	FieldAsName                     /* Field alias name */
	FieldAsNameOpt                  /* Field alias name opt */
	FieldList                       /* field expression list */
	FlushOption                     /* Flush option */
	TableRefsClause                 /* Table references clause */
	FieldItem                       /* Field item for load data clause */
	FieldItemList                   /* Field items for load data clause */
	FuncDatetimePrec                /* Function datetime precision */
	GlobalScope                     /* The scope of variable */
	GroupByClause                   /* GROUP BY clause */
	HashString                      /* Hashed string */
	HavingClause                    /* HAVING clause */
	IfExists                        /* If Exists */
	IfNotExists                     /* If Not Exists */
	IgnoreOptional                  /* IGNORE or empty */
	IndexColName                    /* Index column name */
	IndexColNameList                /* List of index column name */
	IndexKeyPart                    /* Column or expression in a secondary index */
	IndexKeyPartList                /* Secondary index key parts */
	IndexHint                       /* index hint */
	IndexHintList                   /* index hint list */
	IndexHintListOpt                /* index hint list opt */
	IndexHintScope                  /* index hint scope */
	IndexHintType                   /* index hint type */
	IndexName                       /* index name */
	IndexNameList                   /* index name list */
	IndexOption                     /* Index Option */
	IndexOptionList                 /* Index Option List or empty */
	CreateIndexTypeOpt              /* Optional CREATE INDEX type, including TYPE */
	IndexType                       /* index type */
	IndexTypeOpt                    /* Optional index type */
	InsertValues                    /* Rest part of INSERT/REPLACE INTO statement */
	InsertRowAliasOpt               /* optional MySQL row alias of INSERT */
	InsertRowAliasColumnsOpt        /* optional column alias list of the row alias */
	JoinTable                       /* join table */
	JoinType                        /* join type */
	JSONTableExpr                   /* JSON_TABLE expression */
	JSONTableColumnList             /* JSON_TABLE columns list */
	JSONTableColumn                 /* JSON_TABLE column */
	JSONTablePath                   /* JSON_TABLE path */
	JSONTableNestedPath             /* NESTED [PATH] path */
	JSONTableBehaviors              /* optional ON EMPTY and ON ERROR clauses */
	JSONTableBehavior               /* NULL, ERROR, or DEFAULT literal */
	JSONTableOnEmpty                /* behavior ON EMPTY */
	JSONTableOnError                /* behavior ON ERROR */
	LastValueSuffix                 /* Optional LAST_VALUE window suffix */
	LikeEscapeOpt                   /* like escape option */
	LikeTableWithOrWithoutParen     /* LIKE table_name or ( LIKE table_name ) */
	LimitClause                     /* LIMIT clause */
	LimitOption                     /* Limit option could be integer or parameter marker. */
	Lines                           /* Lines clause */
	LinesTerminated                 /* Lines terminated by */
	LoadDataSetSpecOpt              /* Optional load data specification */
	LoadDataSetList                 /* Load data specifications */
	LoadDataSetItem                 /* Single load data specification */
	ReplicationLiteralMode          /* integer-prefix replication numeric conversion */
	ReplicationOptionList           /* CHANGE MASTER TO option list */
	ReplicationOption               /* CHANGE MASTER TO option */
	ReplicationFileOption           /* source or relay log coordinate option */
	ReplicationIDListOpt            /* optional comma-separated server/domain IDs */
	ReplicationIDList               /* comma-separated server/domain IDs */
	ReplicaConnectionNameOpt        /* optional MariaDB connection name */
	ReplicaStartSuffix              /* START replication clauses */
	ReplicaStartTail                /* UNTIL, connection options, and channel clauses */
	ReplicaStopSuffix               /* STOP replication clauses */
	ReplicaThreadOpt                /* IO_THREAD/SQL_THREAD option list */
	ReplicaThreadList               /* comma-separated IO_THREAD/SQL_THREAD options */
	ReplicaUntil                    /* START replication UNTIL condition */
	ReplicaUntilOpt                 /* START REPLICA UNTIL option list */
	ReplicaUntilFileOptions         /* comma-separated UNTIL log coordinates */
	ReplicaConnectionOptionsOpt     /* optional ordered START connection options */
	ReplicaConnectionOptions        /* ordered USER/PASSWORD/DEFAULT_AUTH/PLUGIN_DIR options */
	ReplicaPasswordOpt              /* optional PASSWORD */
	ReplicaDefaultAuthOpt           /* optional DEFAULT_AUTH */
	ReplicaPluginDirOpt             /* optional PLUGIN_DIR */
	ReplicaChannel                  /* FOR CHANNEL clause, preserving empty names */
	ReplicaChannelOpt               /* optional FOR CHANNEL */
	ChannelOpt                      /* FOR CHANNEL option */
	ResetPersistName                /* RESET PERSIST variable name */
	ResetAllOpt                     /* optional RESET REPLICA ALL */
	ResetOption                     /* single RESET option */
	ResetOptionList                 /* comma-separated RESET options */
	HandlerLimitOpt                 /* optional HANDLER LIMIT clause */
	HandlerWhereOpt                 /* optional HANDLER WHERE clause */
	HandlerAliasOpt                 /* optional HANDLER OPEN alias */
	HandlerDirection                /* HANDLER scan direction */
	HandlerRKeyFunction             /* HANDLER index scan function */
	HandlerRKeyMode                 /* HANDLER index range comparison */
	CacheIndexName                  /* CACHE INDEX cache name */
	CacheIndexTable                 /* table and optional cache index names */
	CacheIndexTableList             /* tables with cache index specifications */
	CacheIndexListOpt               /* optional INDEX|KEY (...) specification */
	AdminPartition                 /* PARTITION (ALL|partition_list) */
	LoadIndexTable                  /* single LOAD INDEX INTO CACHE table */
	LoadIndexTableList              /* LOAD INDEX INTO CACHE table list */
	LoadIndexNameList               /* index name list for LOAD INDEX INTO CACHE */
	LoadIndexIgnoreLeavesOpt        /* optional IGNORE LEAVES */
	LoadXMLRowTagOpt                /* optional LOAD XML ROWS IDENTIFIED BY clause */
	LoadXMLIgnoreLinesOpt           /* optional LOAD XML IGNORE n LINES|ROWS */
	RenameUserPair                  /* RENAME USER old TO new pair */
	RenameUserPairList              /* RENAME USER pair list */
	ComponentSetValue               /* INSTALL COMPONENT SET value */
	ComponentSetValueList           /* INSTALL COMPONENT SET value list */
	CloneSSL                        /* CLONE REQUIRE [NO] SSL option */
	DiagnosticsArea                 /* GET DIAGNOSTICS CURRENT/STACKED area */
	DiagnosticsInfo                 /* GET DIAGNOSTICS information */
	DiagnosticsTarget               /* GET DIAGNOSTICS assignment target */
	DiagnosticsStatementItem        /* GET DIAGNOSTICS statement item */
	DiagnosticsConditionItem        /* GET DIAGNOSTICS condition item */
	DiagnosticsStatementItemList    /* GET DIAGNOSTICS statement item list */
	DiagnosticsConditionItemList    /* GET DIAGNOSTICS condition item list */
	MatchAgainstOpt                 /* MATCH ... AGAINST fulltext options */
	LocalOpt                        /* Local opt */
	LockClause                      /* Alter table lock clause */
	NumLiteral                      /* Num/Int/Float/Decimal Literal */
	NoWriteToBinLogAliasOpt         /* NO_WRITE_TO_BINLOG alias LOCAL or empty */
	ObjectType                      /* Grant statement object type */
	OnDuplicateKeyUpdate            /* ON DUPLICATE KEY UPDATE value list */
	DuplicateOpt                    /* [IGNORE|REPLACE] in CREATE TABLE ... SELECT statement or LOAD DATA statement */
	OptFull                         /* Full or empty */
	Order                           /* ORDER BY clause optional collation specification */
	OrderBy                         /* ORDER BY clause */
	WithinGroupOrderBy              /* ordered-set aggregate ORDER BY clause */
	OrReplace                       /* or replace */
	ByItem                          /* BY item */
	OrderByOptional                 /* Optional ORDER BY clause optional */
	ByList                          /* BY list */
	QuickOptional                   /* QUICK or empty */
	PartitionDefinition             /* Partition definition */
	PartitionDefinitionList         /* Partition definition list */
	PartitionDefinitionListOpt      /* Partition definition list option */
	PartitionOpt                    /* Partition option */
	PartitionByOpt                  /* Partition option with PARTITION BY */
	PartitionKeywordOpt             /* optional PARTITION keyword */
	PartitionNameList               /* Partition name list */
	PartitionNameListOpt            /* table partition names list optional */
	PartitionNumOpt                 /* PARTITION NUM option */
	PartDefValuesOpt                /* VALUES {LESS THAN {(expr | value_list) | MAXVALUE} | IN {value_list} */
	PartDefOptionsOpt               /* PartDefOptionList option */
	PartDefOptionList               /* PartDefOption list */
	PartDefOption                   /* COMMENT [=] xxx | TABLESPACE [=] tablespace_name | ENGINE [=] xxx */
	PasswordExpire                  /* Single password option for create user statement */
	PasswordOpt                     /* Password option */
	PasswordOrLockOption            /* Single password or lock option for create user statement */
	PasswordOrLockOptionList        /* Password or lock options for create user statement */
	PasswordOrLockOptions           /* Optional password or lock options for create user statement */
	ColumnPosition                  /* Column position [First|After ColumnName] */
	PrepareSQL                      /* Prepare statement sql string */
	PriorityOpt                     /* Statement priority option */
	PrivElem                        /* Privilege element */
	PrivElemList                    /* Privilege element list */
	PrivLevel                       /* Privilege scope */
	PrivType                        /* Privilege type */
	ReferDef                        /* Reference definition */
	ReferOptionsOpt                 /* optional ON DELETE / ON UPDATE clauses */
	OnDeleteOpt                     /* optional ON DELETE clause */
	OnUpdateOpt                     /* optional ON UPDATE clause */
	OptGConcatSeparator             /* optional GROUP_CONCAT SEPARATOR */
	ReferOpt                        /* reference option */
	RequireList                     /* require list */
	RequireListElement              /* require list element */
	Rolename                        /* Rolename */
	RolenameList                    /* RolenameList */
	RoleSpec                        /* Rolename and auth option */
	RoleSpecList                    /* Rolename and auth option list */
	RoleNameString                  /* role name string */
	RowFormat                       /* Row format option */
	RowValue                        /* Row value */
	GroupByWithOpt                  /* optional WITH ROLLUP/CUBE */
	SelectLockOpt                   /* FOR UPDATE or LOCK IN SHARE MODE, */
	SelectLockClause                /* non-empty locking clause */
	LockedTableListOpt              /* SELECT ... OF tbl_name list */
	SelectLockWaitOpt               /* SELECT ... NOWAIT / SKIP LOCKED */
	SelectOption                   /* one SELECT switch */
	SelectOptionList                /* SELECT options, at most one cache switch */
	SelectOptionListNoCache         /* reorderable/repeatable SELECT switches */
	SelectStmtFieldList             /* SELECT statement field list */
	SelectStmtLimit                 /* SELECT statement optional LIMIT clause */
	SelectStmtOpts                  /* Select statement options */
	SelectStmtBasic                 /* SELECT statement from constant value */
	SelectStmtWithoutFrom           /* optional clauses without a FROM table */
	SelectStmtFromDualTable         /* SELECT statement from dual table */
	SelectStmtFromTable             /* SELECT statement from table */
	SelectStmtGroup                 /* SELECT statement optional GROUP BY clause */
	ProcedureAnalyseClause          /* MariaDB PROCEDURE ANALYSE clause */
	SetRoleOpt                      /* Set role options */
	SetDefaultRoleOpt               /* Set default role options */
	ShowTargetFilterable            /* Show target that can be filtered by WHERE or LIKE */
	ShowDatabaseNameOpt             /* Show tables/columns statement database name option */
	ShowTableAliasOpt               /* Show table alias option */
	ShowLikeOrWhereOpt              /* Show like or where clause option */
	ShowProfileArgsOpt              /* Show profile args option */
	ShowProfileTypesOpt             /* Show profile types option */
	ShowProfileType                 /* Show profile type */
	ShowProfileTypes                /* Show profile types */
	Starting                        /* Starting by */
	StatementList                   /* statement list */
	StatsPersistentVal              /* stats_persistent value */
	StringName                      /* string literal or identifier */
	StringList                      /* string list */
	SubPartitionOpt                 /* SubPartition option */
	SubPartitionNumOpt              /* SubPartition NUM option */
	Symbol                          /* Constraint Symbol */
	TableAsName                     /* table alias name */
	TableAsNameOpt                  /* table alias name optional */
	TableElement                    /* table definition element */
	TableElementList                /* table definition element list */
	CreateTableDefinition          /* parenthesized table definition */
	TableFactor                     /* table factor */
	TableLock                       /* Table name and lock type */
	TableLockList                   /* Table lock list */
	TableName                       /* Table name */
	RoutineName                     /* Stored routine name */
	TableNameList                   /* Table name list */
	TableNameListOpt                /* Table name list opt */
	TableOption                     /* create table option */
	TableOptionList                 /* create table option list */
	TableRef                        /* table reference */
	TableRefs                       /* table references */
	TableToTable                    /* rename table to table */
	TableToTableList                /* rename table to table by list */
	TransactionChar                 /* Transaction characteristic */
	TransactionChars                /* Transaction characteristic list */
	TrimDirection                   /* Trim string direction */
	SetOpr                          /* Set operator: UNION/EXCEPT/INTERSECT */
	SetOprClause                    /* Set operation select clause */
	SetOprClauseList                /* Set operation select clause list */
	SetOprOpt                       /* Set operator option(empty/ALL/DISTINCT) */
	SetOprLimit                     /* LIMIT clause of a set operation (non-empty) */
	Username                        /* Username */
	UsernameList                    /* UsernameList */
	UserSpec                        /* Username and auth option */
	UserSpecList                    /* Username and auth option list */
	UsingRoles                      /*UsingRoles is role option for SHOW GRANT */
	Values                          /* values */
	ValuesList                      /* values list */
	ValuesOpt                       /* values optional */
	ValuesRow                       /* values table row constructor */
	ValuesRowList                   /* values table row constructor list */
	ValuesStmtKeyword               /* VALUES starting a table value constructor */
	ValuesSetOprOperand             /* VALUES-enabled set operation operand */
	ValuesSetOprGroup               /* parenthesized set operation containing VALUES */
	SequenceOption                  /* sequence option */
	SequenceOptionListOpt           /* sequence option list optional */
	TablespaceOption                /* tablespace option */
	TablespaceOptionList            /* tablespace option list */
	UndoOpt                         /* optional UNDO keyword */
	ServerOption                    /* foreign server option */
	ServerOptionList                /* foreign server option list */
	ServerOptionListOpt             /* optional foreign server option list */
	ResourceGroupType               /* resource group type SYSTEM or USER */
	VCPUListOpt                     /* optional VCPU list */
	VCPUList                        /* VCPU list */
	ResourceGroupVCPU               /* resource group VCPU spec */
	ThreadPriorityOpt               /* optional THREAD_PRIORITY */
	ResourceGroupStateOpt           /* optional ENABLE/DISABLE */
	ForceOpt                        /* optional FORCE keyword */
	ExchangePartitionValidationOpt  /* optional WITHOUT VALIDATION */
	LogfileGroupOption              /* logfile group option */
	LogfileGroupOptionList          /* logfile group option list */
	TableMaintenanceOption          /* table maintenance option */
	TableMaintenanceOptionList      /* table maintenance option list */
	UDFReturnType                   /* user defined function return type */
	ReturningOpt                    /* optional RETURNING field list */
	XAIdent                         /* XA transaction identifier */
	XAStartExtraOpt                 /* XA START extra keywords */
	XAEndExtraOpt                   /* XA END extra keywords */
	XACommitExtraOpt                /* XA COMMIT extra keywords */
	WithClause                      /* common table expression clause */
	WithList                        /* common table expression list */
	WithClauseItem                  /* single common table expression */
	WithColNameListOpt              /* optional common table expression column list */
	WithCycleOpt                    /* optional MariaDB CYCLE ... RESTRICT clause */
	VariableAssignment              /* set variable value */
	VariableAssignmentList          /* set variable value list */
	StatementAssignment             /* per-statement system variable assignment */
	StatementAssignmentList         /* nonempty per-statement assignment list */
	SetAssignmentEq                 /* = or := in SET assignments */
	ViewAlgorithm                   /* view algorithm */
	ViewCheckOption                 /* view check option */
	ViewDefiner                     /* view definer */
	ViewName                        /* view name */
	ViewFieldList                   /* create view statement field list */
	ViewSQLSecurity                 /* view sql security */
	WhereClause                     /* WHERE clause */
	WhereClauseOptional             /* Optional WHERE clause */
	WhenClause                      /* When clause */
	WhenClauseList                  /* When clause list */
	WithReadLockOpt                 /* With Read Lock opt */
	WithGrantOptionOpt              /* With Grant Option opt */
	ElseOpt                         /* Optional else clause */
	Type                            /* Types */
	TransactionChainOpt             /* Optional AND [NO] CHAIN */
	TransactionReleaseOpt           /* Optional [NO] RELEASE */
	TransactionChainReleaseOpt      /* Transaction completion options */

	OptExistingWindowName           /* Optional existing WINDOW name */
	OptFromFirstLast                /* Optional FROM FIRST/LAST */
	OptLLDefault                    /* Optional LEAD/LAG default */
	OptLeadLagInfo                  /* Optional LEAD/LAG info */
	OptNullTreatment                /* Optional NULL treatment */
	OptPartitionClause              /* Optional PARTITION clause */
	OptWindowOrderByClause          /* Optional ORDER BY clause in WINDOW */
	OptWindowFrameClause            /* Optional FRAME clause in WINDOW */
	OptWindowFrameExclusion         /* Optional EXCLUDE clause in WINDOW */
	OptWindowingClause              /* Optional OVER clause */
	WindowingClause                 /* OVER clause */
	WindowClauseOptional            /* Optional WINDOW clause */
	WindowDefinitionList            /* WINDOW definition list */
	WindowDefinition                /* WINDOW definition */
	WindowFrameUnits                /* WINDOW frame units */
	WindowFrameBetween              /* WINDOW frame between */
	WindowFrameBound                /* WINDOW frame bound */
	WindowFrameExtent               /* WINDOW frame extent */
	WindowFrameStart                /* WINDOW frame start */
	WindowFuncCall                  /* WINDOW function call */
	WindowName                      /* WINDOW name */
	WindowNameOrSpec                /* WINDOW name or spec */
	WindowSpec                      /* WINDOW spec */
	WindowSpecDetails               /* WINDOW spec details */

	BetweenOrNotOp                  /* Between predicate */
	IsOrNotOp                       /* Is predicate */
	InOrNotOp                       /* In predicate */
	LikeOrNotOp                     /* Like predicate */
	RegexpOrNotOp                   /* Regexp predicate */

	NumericType                     /* Numeric types */
	IntegerType                     /* Integer Types types */
	BooleanType                     /* Boolean Types types */
	FixedPointType                  /* Exact value types */
	FloatingPointType               /* Approximate value types */
	BitValueType                    /* bit value types */
	StringType                      /* String types */
	BlobType                        /* Blob types */
	TextType                        /* Text types */
	DateAndTimeType                 /* Date and Time types */
	SpatialType                     /* Spatial types */

	OptFieldLen                     /* Field length or empty */
	YearLenOpt                      /* YEAR field length, tolerating overflowed literals */
	CastPrecisionOpt                /* CAST temporal precision without integer narrowing */
	FieldLen                        /* Field length */
	FieldOpts                       /* Field type definition option list */
	FieldOpt                        /* Field type definition option */
	FloatOpt                        /* Floating-point type option */
	Precision                       /* Floating-point precision option */
	OptBinary                       /* Optional BINARY */
	OptBinMod                       /* Optional BINARY mode */
	OptCharset                      /* Optional Character setting */
	OptCollate                      /* Optional Collate setting */
	IgnoreLines                     /* Ignore num(int) lines */
	NUM                             /* A number */
	LengthNum                       /* Field length num(uint64) */
	TableOptimizerHintOpt           /* Table level optimizer hint */
	TableOptimizerHints             /* Table level optimizer hints */
	TableOptimizerHintList          /* Table level optimizer hint list */
	FuncParams                      /* routine parameter list with parentheses */
	FuncParamList                   /* routine parameter list */
	FuncParam                       /* single routine parameter */
	RoutineCharacteristicList       /* stored routine characteristic list */
	RoutineCharacteristic           /* stored routine characteristic */
	TriggerTime                     /* trigger action time */
	TriggerEvent                    /* trigger event */
	TriggerUpdateColumnsOpt         /* optional trigger UPDATE OF columns */
	TriggerOrderOpt                 /* optional trigger FOLLOWS/PRECEDES */
	EventSchedule                   /* event ON SCHEDULE clause */
	EventStartsOpt                  /* optional event STARTS clause */
	EventEndsOpt                    /* optional event ENDS clause */
	EventStatusOpt                  /* optional event status clause */
	EventOnCompletionOpt            /* optional event ON COMPLETION clause */
	EventCommentOpt                 /* optional event COMMENT clause */
	AlterEventClause                /* single ALTER EVENT clause */
	AlterEventClauses               /* ALTER EVENT clause list */
	EventDoOpt                      /* optional ALTER EVENT DO body */
	SpStmtListOpt                   /* optional stored program statement list */
	SpStmtList                      /* stored program statement list */
	SpLabelPrefix                   /* optional stored program label prefix */
	SpLabelSuffix                   /* optional stored program label suffix */
	SpLabelName                     /* stored program label name */
	SpAtomicOpt                     /* optional NOT ATOMIC */
	SpElseIfClauseList              /* ELSEIF clause list */
	SpElseOpt                       /* optional ELSE clause */
	SpWhenClauseList                /* CASE WHEN clause list */
	SpWhenClause                    /* single CASE WHEN clause */
	SpCaseElseOpt                   /* optional CASE ELSE clause */
	SpVarNameList                   /* DECLARE variable name list */
	SpVarName                       /* DECLARE variable name */
	SpRowType                       /* MariaDB ROW local-variable type */
	SpRowFieldList                  /* MariaDB ROW type fields */
	SpRowField                      /* MariaDB ROW type field */
	SpDeclareDefaultOpt             /* optional DECLARE DEFAULT clause */
	SpHandlerAction                 /* handler action keyword */
	SpConditionValue                /* handler/signal condition value */
	SpConditionValueList            /* handler condition value list */
	SpConditionValueOpt             /* optional handler/signal condition value */
	SpNameList                      /* SELECT INTO variable targets */
	SpName                          /* stored-program or user variable target */
	SpLocalName                     /* local variable or ROW field target */
	SpLocalNameList                 /* FETCH local variable targets */
	WeightStringAsOpt               /* optional WEIGHT_STRING AS clause */
	WeightStringLevelOpt            /* optional WEIGHT_STRING LEVEL clause */
	WeightStringLevelList           /* WEIGHT_STRING LEVEL list */
	WeightStringLevelItem           /* one WEIGHT_STRING LEVEL item */
	WeightStringLevelFlags          /* WEIGHT_STRING LEVEL flags */
	WeightStringLength              /* positive WEIGHT_STRING result length */
	UnsignedInteger                 /* decimal or numeric hexadecimal integer */
	HexInteger                      /* numeric 0x hexadecimal integer */
	SelectIntoOpt                    /* optional SELECT INTO clause */
	SelectIntoClause                 /* non-empty SELECT INTO clause */
	SelectIntoOutfileOpts            /* SELECT INTO OUTFILE options */
	CharsetOpt                       /* optional CHARACTER SET clause */
	SpSignalInfoItemListOpt         /* optional SIGNAL/RESIGNAL SET clause */
	SpSignalInfoItemList            /* SIGNAL/RESIGNAL SET item list */
	SpSignalInfoItem                /* single SIGNAL/RESIGNAL SET item */

%type	<ident>
	AsOpt                           /* AS or EmptyString */
	WindowFuncAliasName             /* window function keyword usable as a field alias */
	KeyOrIndex                      /* {KEY|INDEX} */
	ColumnKeywordOpt                /* Column keyword or empty */
	PrimaryOpt                      /* Optional primary keyword */
	NowSym                          /* CURRENT_TIMESTAMP/LOCALTIME/LOCALTIMESTAMP */
	NowSymFunc                      /* CURRENT_TIMESTAMP/LOCALTIME/LOCALTIMESTAMP/NOW */
	DefaultKwdOpt                   /* optional DEFAULT keyword */
	DatabaseSym                     /* DATABASE or SCHEMA */
	ExplainSym                      /* EXPLAIN or DESCRIBE or DESC */
	RegexpSym                       /* REGEXP or RLIKE */
	IntoOpt                         /* INTO or EmptyString */
	ValueSym                        /* Value or Values */
	Varchar                         /* VARCHAR */
	TimeUnit                        /* Time unit for 'DATE_ADD', 'DATE_SUB', 'ADDDATE', 'SUBDATE', 'EXTRACT' */
	TimestampUnit                   /* Time unit for 'TIMESTAMPADD' and 'TIMESTAMPDIFF' */
	DeallocateSym                   /* Deallocate or drop */
	OuterOpt		                /* optional OUTER clause */
	CrossOpt		                /* Cross join option */
	TablesTerminalSym               /* {TABLE|TABLES} */
	IsolationLevel                  /* Isolation level */
	ShowIndexKwd                    /* Show index/indexs/key keyword */
	DistinctKwd                     /* DISTINCT/DISTINCTROW keyword */
	FromOrIn                        /* From or In */
	OptTable                        /* Optional table keyword */
	OptInteger                      /* Optional Integer keyword */
	CharsetOrCharacterSet           /* charset or charater set */
	LockType                        /* Table locks type */
	logAnd                          /* logical and operator */
	logOr                           /* logical or operator */
	LinearOpt                       /* linear or empty */
	FieldsOrColumns                 /* Fields or columns */
	GetFormatSelector               /* {DATE|DATETIME|TIME|TIMESTAMP} */
	IsJsonTypeOpt                   /* optional IS JSON type constraint */
	IsJsonUniqueKeysOpt             /* optional IS JSON unique keys constraint */
	DiagnosticsConditionItemName    /* GET DIAGNOSTICS condition item name */
	LoadIndexName                   /* identifier or PRIMARY index name */
	LoadXMLPriorityOpt              /* LOW_PRIORITY|CONCURRENT or empty */
	ReplicaThread                   /* IO_THREAD|SQL_THREAD */
	ReplicationStringOptionName     /* string-valued replication options */
	ReplicationStringDefaultOptionName /* MariaDB path options supporting DEFAULT */
	ReplicationTLSCipherSuitesOptionName /* MASTER/SOURCE_TLS_CIPHERSUITES */
	ReplicationNumericOptionName    /* numeric-valued replication options */
	ReplicationNumericDefaultOptionName /* MariaDB retry options supporting DEFAULT */
	ReplicationBooleanOptionName    /* numeric boolean replication options */
	ReplicationMariaBooleanDefaultOptionName /* MariaDB TLS booleans supporting DEFAULT */
	ReplicationMariaBooleanOptionName /* MariaDB demotion boolean options */
	ReplicationStrictBinaryOptionName /* GTID_ONLY|SOURCE_CONNECTION_AUTO_FAILOVER */
	ReplicationIDListOptionName     /* IGNORE_SERVER_IDS|DO_DOMAIN_IDS|IGNORE_DOMAIN_IDS */
	ReplicationLogFileOptionName    /* MASTER/SOURCE/RELAY_LOG_FILE */


%type	<ident>
	ExplainFormat                   /* EXPLAIN FORMAT= option value */
	Identifier                      /* Identifier or unreserved keyword */
	NotKeywordToken                 /* Tokens not mysql keyword but treated specially */
	UnReservedKeyword               /* MySQL unreserved keywords */
	FunctionNameConflict            /* Built-in function call names which are conflict with keywords */
	FunctionNameOptionalBraces      /* Function with optional braces, all of them are reserved keywords. */
	FunctionNameDatetimePrecision   /* Function with optional datetime precision, all of them are reserved keywords. */
	FunctionNameDateArith           /* Date arith function call names (date_add or date_sub) */
	FunctionNameDateArithMultiForms /* Date arith function call names (adddate or subdate) */
	FunctionNameIdentifier          /* Built-in function token usable as a table name */
	SpatialFunctionName             /* spatial type keywords used as function names */
	ServerOptionName                /* foreign server option name */


%left empty
%left into
%left with
/* KILL consumes its scope before parsing an expression-valued ID. */
%left connection query
/* Prefer IS JSON constraints over implicit select aliases. */
%left value arrayKwd objectKwd scalarKwd without

%left sqlBufferResult
%left sqlBigResult
%left sqlSmallResult
%left sqlCache sqlNoCache
%left lowerThanStringLitToken
%left stringLit
%left lowerThanSetKeyword
%left set
%left lowerThanInsertValues
%left insertValues
%left lowerThanCreateTableSelect
%left createTableSelect
%left lowerThanCharsetKwd
%left charsetKwd
%left lowerThanKey
%left key


/* Lower than every join/condition token: keep shifting until the last ON is seen. */
%left       conditionlessJoin
%left       join straightJoin inner cross left right full natural
/* A dummy token to force the priority of TableRef production in a join. */
%left       tableRefPriority
%left lowerThanOn
%left on using
%right      assignmentEq
%left       pipes or pipesAsOr
%left       xor
%left       andand and
%left       between
%left       memberKwd sounds
%left lowerThanEq
%left       eq ge le neq neqSynonym nulleq '>' '<' is in regexpKwd rlike
%left       like
%left       '|'
%left       '&'
%left       rsh lsh
%left       '-' '+'
%left       '*' '/' '%' div mod
%left       '^'
%left       '~' neg
%right      not not2
%right      collate
%left       interval


%left '(' ')'
%left quick
%left escape
%left lowerThanComma
%left ','
%left higherThanComma
%left lowerThanWithin
%left within


%start	Start

%%

Start:
	StatementList


/**************************************AlterTableStmt***************************************
 * See https://dev.mysql.com/doc/refman/5.7/en/alter-table.html
 *******************************************************************************************/
AlterTableStmt:
	alter IgnoreOptional tableKwd TableName AlterTableSpecList
	{
		$$ = &ast.AlterTableStmt{
			Table: $4.(*ast.TableName),
			Specs: $5.([]*ast.AlterTableSpec),
		}
	}
|	alter IgnoreOptional tableKwd TableName analyze partition PartitionNameList BucketsOpt
	{
		$$ = &ast.AnalyzeTableStmt{TableNames: []*ast.TableName{$4.(*ast.TableName)}, PartitionNames: $7.([]model.CIStr), MaxNumBuckets: $8.(uint64) }
	}
|	alter IgnoreOptional tableKwd TableName analyze partition PartitionNameList index IndexNameList BucketsOpt
	{
		$$ = &ast.AnalyzeTableStmt{
			TableNames: []*ast.TableName{$4.(*ast.TableName)},
			PartitionNames: $7.([]model.CIStr),
			IndexNames: $9.([]model.CIStr),
			IndexFlag: true,
			MaxNumBuckets: $10.(uint64),
		}
	}


BucketsOpt:
	/* empty */   { $$ = uint64(0) }
|	with NUM buckets    { $$ = getUint64FromNUM($2) }


ExchangePartitionValidationOpt:
	/* EMPTY */                  { $$ = false }
|	without validation       { $$ = true }


AlterTableSpec:
	AlterTableOptionListOpt
	{
		$$ = &ast.AlterTableSpec{
			Tp:	ast.AlterTableOption,
			Options:$1.([]*ast.TableOption),
		}
	}
|	convert to CharsetOrCharacterSet defaultKwd OptCollate
	{
		op := &ast.AlterTableSpec{
			Tp: ast.AlterTableOption,
			CharsetConvert: true,
			Options:[]*ast.TableOption{{Tp: ast.TableOptionCharset, StrValue: "DEFAULT"}},
		}
		if $5 != "" {
			op.Options = append(op.Options, &ast.TableOption{Tp: ast.TableOptionCollate, StrValue: $5.(string)})
		}
		$$ = op
	}
|	convert to CharsetOrCharacterSet CharsetName OptCollate
	{
		op := &ast.AlterTableSpec{
			Tp: ast.AlterTableOption,
			CharsetConvert: true,
			Options:[]*ast.TableOption{{Tp: ast.TableOptionCharset, StrValue: $4.(string)}},
		}
		if $5 != "" {
			op.Options = append(op.Options, &ast.TableOption{Tp: ast.TableOptionCollate, StrValue: $5.(string)})
		}
		$$ = op
	}
|	add ColumnKeywordOpt ColumnDef ColumnPosition
	{
		$$ = &ast.AlterTableSpec{
			Tp: 		ast.AlterTableAddColumns,
			NewColumns:	[]*ast.ColumnDef{$3.(*ast.ColumnDef)},
			Position:	$4.(*ast.ColumnPosition),
		}
	}
|	add ColumnKeywordOpt '(' ColumnDefList ')'
	{
		$$ = &ast.AlterTableSpec{
			Tp: 		ast.AlterTableAddColumns,
			NewColumns:	$4.([]*ast.ColumnDef),
		}
	}
|	add Constraint
	{
		constraint := $2.(*ast.Constraint)
		$$ = &ast.AlterTableSpec{
			Tp: ast.AlterTableAddConstraint,
			Constraint: constraint,
		}
	}
|	add partition PartitionDefinitionListOpt
	{
		var defs []*ast.PartitionDefinition
		if $3 != nil {
			defs = $3.([]*ast.PartitionDefinition)
		}
		$$ = &ast.AlterTableSpec{
			Tp: ast.AlterTableAddPartitions,
			PartDefinitions: defs,
		}
	}
|	add partition partitions NUM
	{
		$$ = &ast.AlterTableSpec{
			Tp: ast.AlterTableAddPartitions,
			Num: getUint64FromNUM($4),
		}
	}
|	coalesce partition NUM
	{
		$$ = &ast.AlterTableSpec{
			Tp: ast.AlterTableCoalescePartitions,
			Num: getUint64FromNUM($3),
		}
	}
|	drop ColumnKeywordOpt ColumnName RestrictOrCascadeOpt
	{
		$$ = &ast.AlterTableSpec{
			Tp: ast.AlterTableDropColumn,
			OldColumnName: $3.(*ast.ColumnName),
		}
	}
|	drop primary key
	{
		$$ = &ast.AlterTableSpec{Tp: ast.AlterTableDropPrimaryKey}
	}
|	drop partition Identifier
	{
		$$ = &ast.AlterTableSpec{
			Tp: ast.AlterTableDropPartition,
			Name: $3,
		}
	}
|	truncate partition Identifier
	{
		$$ = &ast.AlterTableSpec{
			Tp: ast.AlterTableTruncatePartition,
			Name: $3,
		}
	}
|	exchange partition Identifier with tableKwd TableName ExchangePartitionValidationOpt
	{
		$$ = &ast.AlterTableSpec{
			Tp:                ast.AlterTableExchangePartition,
			Name:              $3,
			NewTable:          $6.(*ast.TableName),
			WithoutValidation: $7.(bool),
		}
	}
|	reorganize partition PartitionNameList into '(' PartitionDefinitionList ')'
	{
		$$ = &ast.AlterTableSpec{
			Tp:              ast.AlterTableReorganizePartition,
			PartitionNames:  $3.([]model.CIStr),
			PartDefinitions: $6.([]*ast.PartitionDefinition),
		}
	}
|	reorganize partition all into '(' PartitionDefinitionList ')'
	{
		$$ = &ast.AlterTableSpec{
			Tp:              ast.AlterTableReorganizePartition,
			PartDefinitions: $6.([]*ast.PartitionDefinition),
		}
	}
|	remove partitioning
	{
		$$ = &ast.AlterTableSpec{Tp: ast.AlterTableRemovePartitioning}
	}
|	PartitionByOpt
	{
		$$ = &ast.AlterTableSpec{Tp: ast.AlterTablePartition, Partition: $1.(*ast.PartitionOptions)}
	}
|	discard tablespace
	{
		$$ = &ast.AlterTableSpec{Tp: ast.AlterTableDiscardTablespace}
	}
|	import_ tablespace
	{
		$$ = &ast.AlterTableSpec{Tp: ast.AlterTableImportTablespace}
	}
|	drop KeyOrIndex Identifier
	{
		$$ = &ast.AlterTableSpec{
			Tp: ast.AlterTableDropIndex,
			Name: $3,
		}
	}
|	drop foreign key IfExists Symbol
	{
		$$ = &ast.AlterTableSpec{
			Tp: ast.AlterTableDropForeignKey,
			Name: $5.(string),
			IfExists: $4.(bool),
		}
	}
|	disable keys
	{
		$$ = &ast.AlterTableSpec{}
	}
|	enable keys
	{
		$$ = &ast.AlterTableSpec{}
	}
|	modify ColumnKeywordOpt ColumnDef ColumnPosition
	{
		$$ = &ast.AlterTableSpec{
			Tp:		ast.AlterTableModifyColumn,
			NewColumns:	[]*ast.ColumnDef{$3.(*ast.ColumnDef)},
			Position:	$4.(*ast.ColumnPosition),
		}
	}
|	change ColumnKeywordOpt ColumnName ColumnDef ColumnPosition
	{
		$$ = &ast.AlterTableSpec{
			Tp:    		ast.AlterTableChangeColumn,
			OldColumnName:	$3.(*ast.ColumnName),
			NewColumns:	[]*ast.ColumnDef{$4.(*ast.ColumnDef)},
			Position:	$5.(*ast.ColumnPosition),
		}
	}
|	alter ColumnKeywordOpt ColumnName set defaultKwd SignedLiteral
	{
		option := &ast.ColumnOption{Expr: $6}
		colDef := &ast.ColumnDef{
			Name: 	 $3.(*ast.ColumnName),
			Options: []*ast.ColumnOption{option},
		}
		$$ = &ast.AlterTableSpec{
			Tp:		ast.AlterTableAlterColumn,
			NewColumns:	[]*ast.ColumnDef{colDef},
		}
	}
|	alter ColumnKeywordOpt ColumnName drop defaultKwd
	{
		colDef := &ast.ColumnDef{
			Name: 	 $3.(*ast.ColumnName),
		}
		$$ = &ast.AlterTableSpec{
			Tp:		ast.AlterTableAlterColumn,
			NewColumns:	[]*ast.ColumnDef{colDef},
		}
	}
|	rename to TableName
	{
		$$ = &ast.AlterTableSpec{
			Tp:    		ast.AlterTableRenameTable,
			NewTable:      $3.(*ast.TableName),
		}
	}
|	rename TableName
	{
		$$ = &ast.AlterTableSpec{
			Tp:    		ast.AlterTableRenameTable,
			NewTable:      $2.(*ast.TableName),
		}
	}
|	rename as TableName
	{
		$$ = &ast.AlterTableSpec{
			Tp:    		ast.AlterTableRenameTable,
			NewTable:      $3.(*ast.TableName),
		}
	}
|	rename KeyOrIndex Identifier to Identifier
	{
		$$ = &ast.AlterTableSpec{
			Tp:    	    ast.AlterTableRenameIndex,
			FromKey:    model.NewCIStr($3),
			ToKey:      model.NewCIStr($5),
		}
	}
|	rename column ColumnName to ColumnName
	{
		$$ = &ast.AlterTableSpec{
			Tp:            ast.AlterTableRenameColumn,
			OldColumnName: $3.(*ast.ColumnName),
			NewColumnName: $5.(*ast.ColumnName),
		}
	}
|	LockClause
	{
		$$ = &ast.AlterTableSpec{
			Tp:    		ast.AlterTableLock,
			LockType:   $1.(ast.LockType),
		}
	}
| algorithm EqOpt AlterAlgorithm
	{
		// Parse it and ignore it. Just for compatibility.
		$$ = &ast.AlterTableSpec{
			Tp:    		ast.AlterTableAlgorithm,
			Algorithm:	$3.(ast.AlterAlgorithm),
		}
	}
| force
	{
		// Parse it and ignore it. Just for compatibility.
		$$ = &ast.AlterTableSpec{
			Tp:    		ast.AlterTableForce,
		}
	}


AlterAlgorithm:
	defaultKwd { $$ = ast.AlterAlgorithmDefault }
|	copyKwd    { $$ = ast.AlterAlgorithmCopy }
|	inplace { $$ = ast.AlterAlgorithmInplace }
|	instant { $$ = ast.AlterAlgorithmInstant }


LockClauseOpt:
	/* EMPTY */ {}
|	LockClause  {}


LockClause:
	lock eq none      { $$ = ast.LockTypeNone }
|	lock eq defaultKwd   { $$ = ast.LockTypeDefault }
|	lock eq shared    { $$ = ast.LockTypeShared }
|	lock eq exclusive { $$ = ast.LockTypeExclusive }


KeyOrIndex: key | index


KeyOrIndexOpt:
	/* EMPTY */ {}
|	KeyOrIndex


ColumnKeywordOpt:
	/* EMPTY */ {}
|	column


ColumnPosition:
	{
		$$ = &ast.ColumnPosition{Tp: ast.ColumnPositionNone}
	}
|	first
	{
		$$ = &ast.ColumnPosition{Tp: ast.ColumnPositionFirst}
	}
|	after ColumnName
	{
		$$ = &ast.ColumnPosition{
			Tp: ast.ColumnPositionAfter,
			RelativeColumn: $2.(*ast.ColumnName),
		}
	}


AlterTableSpecList:
	AlterTableSpec
	{
		$$ = []*ast.AlterTableSpec{$1.(*ast.AlterTableSpec)}
	}
|	AlterTableSpecList ',' AlterTableSpec
	{
		$$ = append($1.([]*ast.AlterTableSpec), $3.(*ast.AlterTableSpec))
	}


PartitionNameList:
	Identifier
	{
		$$ = []model.CIStr{model.NewCIStr($1)}
	}
|	PartitionNameList ',' Identifier
	{
		$$ = append($1.([]model.CIStr), model.NewCIStr($3))
	}


ConstraintKeywordOpt:
	/* EMPTY */         { $$ = nil }
|	constraint        { $$ = nil }
|	constraint Symbol { $$ = $2.(string) }


Symbol:
	Identifier { $$ = $1 }


/**************************************RenameTableStmt***************************************
 * See http://dev.mysql.com/doc/refman/5.7/en/rename-table.html
 *******************************************************************************************/
RenameTableStmt:
	rename tableKwd TableToTableList
	{
		$$ = &ast.RenameTableStmt{
			OldTable: $3.([]*ast.TableToTable)[0].OldTable,
			NewTable: $3.([]*ast.TableToTable)[0].NewTable,
			TableToTables: $3.([]*ast.TableToTable),
		}
	}


TableToTableList:
	TableToTable
	{
		$$ = []*ast.TableToTable{$1.(*ast.TableToTable)}
	}
|	TableToTableList ',' TableToTable
	{
		$$ = append($1.([]*ast.TableToTable), $3.(*ast.TableToTable))
	}


TableToTable:
	TableName to TableName
	{
		$$ = &ast.TableToTable{
			OldTable: $1.(*ast.TableName),
			NewTable: $3.(*ast.TableName),
		}
	}


/*******************************************************************************************/

AnalyzeTableStmt:
	analyze NoWriteToBinLogAliasOpt tableKwd TableNameList AnalyzePersistentOpt
	 {
		st := $5.(*ast.AnalyzeTableStmt)
		st.TableNames = $4.([]*ast.TableName)
		st.NoWriteToBinLog = $2.(bool)
		$$ = st
	 }
|	analyze NoWriteToBinLogAliasOpt tables TableNameList AnalyzePersistentOpt
	{
		st := $5.(*ast.AnalyzeTableStmt)
		st.TableNames = $4.([]*ast.TableName)
		st.NoWriteToBinLog = $2.(bool)
		$$ = st
	}
|	analyze NoWriteToBinLogAliasOpt tableKwd TableName index IndexNameList
	{
		$$ = &ast.AnalyzeTableStmt{ TableNames: []*ast.TableName{$4.(*ast.TableName)}, IndexNames: $6.([]model.CIStr), IndexFlag: true, NoWriteToBinLog: $2.(bool) }
	}
|	analyze NoWriteToBinLogAliasOpt tableKwd TableName partition PartitionNameList BucketsOpt
	{
		$$ = &ast.AnalyzeTableStmt{ TableNames: []*ast.TableName{$4.(*ast.TableName)}, PartitionNames: $6.([]model.CIStr), MaxNumBuckets: $7.(uint64), NoWriteToBinLog: $2.(bool) }
	}
|	analyze NoWriteToBinLogAliasOpt tableKwd TableName partition PartitionNameList index IndexNameList BucketsOpt
	{
		$$ = &ast.AnalyzeTableStmt{
			TableNames: []*ast.TableName{$4.(*ast.TableName)},
			PartitionNames: $6.([]model.CIStr),
			IndexNames: $8.([]model.CIStr),
			IndexFlag: true,
			MaxNumBuckets: $9.(uint64),
			NoWriteToBinLog: $2.(bool),
		}
	}


BackupStmt:
	backup lock TableName
	{
		$$ = &ast.BackupStmt{Table: $3.(*ast.TableName)}
	}
|	backup unlock
	{
		$$ = &ast.BackupStmt{Unlock: true}
	}


AnalyzePersistentOpt:
	/* EMPTY */ { $$ = &ast.AnalyzeTableStmt{} }
|	persistent forKwd all { $$ = &ast.AnalyzeTableStmt{Persistent: true, PersistentAll: true} }
|	persistent forKwd columns AnalyzeColumnList indexes AnalyzeIndexList
	{
		$$ = &ast.AnalyzeTableStmt{Persistent: true, ColumnNames: $4.([]model.CIStr), IndexNames: $6.([]model.CIStr)}
	}


AnalyzeColumnList:
	all { $$ = []model.CIStr(nil) }
|	'(' ')' { $$ = []model.CIStr{} }
|	'(' ColumnList ')' { $$ = $2 }


AnalyzeIndexList:
	all { $$ = []model.CIStr(nil) }
|	'(' IndexNameList ')'
	{
		names := $2.([]model.CIStr)
		if names == nil {
			names = []model.CIStr{}
		}
		$$ = names
	}


CheckTableStmt:
	check tableKwd TableNameList TableMaintenanceOptionList
	{
		$$ = &ast.TableMaintenanceStmt{
			Tp:      ast.TableMaintenanceCheck,
			Tables:  $3.([]*ast.TableName),
			Options: $4.([]string),
		}
	}


RepairTableStmt:
	repair LocalOpt tableKwd TableNameList TableMaintenanceOptionList
	{
		$$ = &ast.TableMaintenanceStmt{
			Tp:      ast.TableMaintenanceRepair,
			Tables:  $4.([]*ast.TableName),
			Options: $5.([]string),
		}
	}


OptimizeTableStmt:
	optimize LocalOpt tableKwd TableNameList
	{
		$$ = &ast.TableMaintenanceStmt{
			Tp:     ast.TableMaintenanceOptimize,
			Tables: $4.([]*ast.TableName),
		}
	}


ChecksumTableStmt:
	checksum tableKwd TableNameList TableMaintenanceOptionList
	{
		$$ = &ast.TableMaintenanceStmt{
			Tp:      ast.TableMaintenanceChecksum,
			Tables:  $3.([]*ast.TableName),
			Options: $4.([]string),
		}
	}


TableMaintenanceOptionList:
	/* EMPTY */
	{
		$$ = []string{}
	}
|	TableMaintenanceOptionList TableMaintenanceOption
	{
		$$ = append($1.([]string), $2.(string))
	}


TableMaintenanceOption:
	Identifier
	{
		$$ = strings.ToUpper($1)
	}
|	forKwd upgrade
	{
		$$ = "FOR UPGRADE"
	}


/*******************************************************************************************/
Assignment:
	ColumnName eq ExprOrDefault
	{
		$$ = &ast.Assignment{Column: $1.(*ast.ColumnName), Expr:$3}
	}


AssignmentList:
	Assignment
	{
		$$ = []*ast.Assignment{$1.(*ast.Assignment)}
	}
|	AssignmentList ',' Assignment
	{
		$$ = append($1.([]*ast.Assignment), $3.(*ast.Assignment))
	}


BeginTransactionStmt:
	/* EMPTY */ begin                            { $$ = &ast.BeginStmt{} }
|	begin not atomic SpStmtListOpt end
	{
		$$ = &ast.BeginEndStmt{NotAtomic: true, List: $4.([]ast.StmtNode)}
	}
|	start transaction                          { $$ = &ast.BeginStmt{} }
|	start transaction with consistent snapshot { $$ = &ast.BeginStmt{} }


BinlogStmt:
	binlog stringLit { $$ = &ast.BinlogStmt{Str: $2} }


ColumnDefList:
	ColumnDef
	{
		$$ = []*ast.ColumnDef{$1.(*ast.ColumnDef)}
	}
|	ColumnDefList ',' ColumnDef
	{
		$$ = append($1.([]*ast.ColumnDef), $3.(*ast.ColumnDef))
	}


ColumnDef:
	ColumnName Type ColumnOptionListOpt
	{
		colDef := &ast.ColumnDef{Name: $1.(*ast.ColumnName), Tp: $2.(*types.FieldType), Options: $3.([]*ast.ColumnOption)}
		if !colDef.Validate() {
			yyAppendErr(yylex, yyErrorf(yylex, "Invalid column definition"))
			return 1
		}
		$$ = colDef
	}
|	ColumnName serialType ColumnOptionListOpt
	{
		// SERIAL is an alias for BIGINT UNSIGNED NOT NULL AUTO_INCREMENT UNIQUE.
		// Reject options that conflict with SERIAL's implied flags.
		for _, opt := range $3.([]*ast.ColumnOption) {
			if opt.Tp == ast.ColumnOptionNull || opt.Tp == ast.ColumnOptionAutoIncrement {
				yyAppendErr(yylex, yyErrorf(yylex, "Invalid use of SERIAL with a NULL or AUTO_INCREMENT option"))
				return 1
			}
		}
		tp := types.NewFieldType(mysql.TypeLonglong)
		tp.Flag |= mysql.UnsignedFlag
		colDef := &ast.ColumnDef{
			Name: $1.(*ast.ColumnName),
			Tp:   tp,
			Options: []*ast.ColumnOption{
				{Tp: ast.ColumnOptionNotNull},
				{Tp: ast.ColumnOptionAutoIncrement},
				{Tp: ast.ColumnOptionUniqKey},
			},
		}
		colDef.Options = append(colDef.Options, $3.([]*ast.ColumnOption)...)
		if !colDef.Validate() {
			yyAppendErr(yylex, yyErrorf(yylex, "Invalid column definition"))
			return 1
		}
		$$ = colDef
	}


ColumnName:
	Identifier
	{
		$$ = &ast.ColumnName{Name: model.NewCIStr($1)}
	}
|	window
	{
		$$ = &ast.ColumnName{Name: model.NewCIStr($1)}
	}
|	rank
	{
		$$ = &ast.ColumnName{Name: model.NewCIStr($1)}
	}
|	Identifier '.' Identifier
	{
		$$ = &ast.ColumnName{Table: model.NewCIStr($1), Name: model.NewCIStr($3)}
	}
|	Identifier '.' Identifier '.' Identifier
	{
		$$ = &ast.ColumnName{Schema: model.NewCIStr($1), Table: model.NewCIStr($3), Name: model.NewCIStr($5)}
	}


ColumnNameList:
	ColumnName
	{
		$$ = []*ast.ColumnName{$1.(*ast.ColumnName)}
	}
|	ColumnNameList ',' ColumnName
	{
		$$ = append($1.([]*ast.ColumnName), $3.(*ast.ColumnName))
	}


ColumnNameListOpt:
	/* EMPTY */    { $$ = []*ast.ColumnName{} }
|	ColumnNameList { $$ = $1.([]*ast.ColumnName) }


ColumnNameOrUserVarListOpt:
	/* EMPTY */                  { $$ = []*ast.ColumnNameOrUserVar{} }
|	ColumnNameOrUserVariableList { $$ = $1.([]*ast.ColumnNameOrUserVar) }


ColumnNameOrUserVariableList:
	ColumnNameOrUserVariable
	{
		$$ = []*ast.ColumnNameOrUserVar{$1.(*ast.ColumnNameOrUserVar)}
	}
|	ColumnNameOrUserVariableList ',' ColumnNameOrUserVariable
	{
		$$ = append($1.([]*ast.ColumnNameOrUserVar), $3.(*ast.ColumnNameOrUserVar))
	}


ColumnNameOrUserVariable:
	ColumnName
	{
		$$ = &ast.ColumnNameOrUserVar{ColumnName: $1.(*ast.ColumnName)}
	}
|	UserVariable
	{
		$$ = &ast.ColumnNameOrUserVar{UserVar: $1.(*ast.VariableExpr)}
	}


ColumnNameOrUserVarListOptWithBrackets:
	/* EMPTY */                        { $$ = []*ast.ColumnNameOrUserVar{} }
|	'(' ColumnNameOrUserVarListOpt ')' { $$ = $2.([]*ast.ColumnNameOrUserVar) }


CommitStmt:
	commit TransactionWorkOpt TransactionChainReleaseOpt
	{
		$$ = &ast.CommitStmt{TransactionOptions: $3.(ast.TransactionOptions)}
	}


TransactionWorkOpt:
	/* EMPTY */
|	work


TransactionChainOpt:
	/* EMPTY */ { $$ = nil }
|	and chain { value := true; $$ = &value }
|	and no chain { value := false; $$ = &value }


TransactionReleaseOpt:
	/* EMPTY */ { $$ = nil }
|	release { value := true; $$ = &value }
|	no release { value := false; $$ = &value }


TransactionChainReleaseOpt:
	TransactionChainOpt TransactionReleaseOpt
	{
		options := ast.TransactionOptions{}
		if $1 != nil {
			options.Chain = $1.(*bool)
		}
		if $2 != nil {
			options.Release = $2.(*bool)
		}
		if options.Chain != nil && *options.Chain && options.Release != nil && *options.Release {
			yyAppendErr(yylex, yyErrorf(yylex, "CHAIN and RELEASE cannot both be enabled"))
			return 1
		}
		$$ = options
	}


PrimaryOpt:
	/* EMPTY */ {}
|	primary


ColumnOption:
	not null
	{
		$$ = &ast.ColumnOption{Tp: ast.ColumnOptionNotNull}
	}
|	null
	{
		$$ = &ast.ColumnOption{Tp: ast.ColumnOptionNull}
	}
|	autoIncrement
	{
		$$ = &ast.ColumnOption{Tp: ast.ColumnOptionAutoIncrement}
	}
|	PrimaryOpt key
	{
		// KEY is normally a synonym for INDEX. The key attribute PRIMARY KEY
		// can also be specified as just KEY when given in a column definition.
		// See http://dev.mysql.com/doc/refman/5.7/en/create-table.html
		$$ = &ast.ColumnOption{Tp: ast.ColumnOptionPrimaryKey}
	}
|	unique %prec lowerThanKey
	{
		$$ = &ast.ColumnOption{Tp: ast.ColumnOptionUniqKey}
	}
|	unique key
	{
		$$ = &ast.ColumnOption{Tp: ast.ColumnOptionUniqKey}
	}
|	defaultKwd DefaultValueExpr
	{
		$$ = &ast.ColumnOption{Tp: ast.ColumnOptionDefaultValue, Expr: $2}
	}
|	on update NowSymOptionFraction
	{
		$$ = &ast.ColumnOption{Tp: ast.ColumnOptionOnUpdate, Expr: $3}
	}
|	comment stringLit
	{
		$$ =  &ast.ColumnOption{Tp: ast.ColumnOptionComment, Expr: ast.NewValueExpr($2)}
	}
|	check '(' Expression ')'
	{
		$$ = &ast.ColumnOption{Tp: ast.ColumnOptionCheck, Expr: $3}
	}
|	GeneratedAlways as '(' Expression ')' VirtualOrStored
	{
		startOffset := yyGetParser(yylex).startOffset(&yyS[yypt-2])
		endOffset := yyGetParser(yylex).endOffset(&yyS[yypt-1])
		expr := $4
		expr.SetText(yyGetParser(yylex).src[startOffset:endOffset])

		$$ = &ast.ColumnOption{
			Tp: ast.ColumnOptionGenerated,
			Expr: expr,
			Stored: $6.(bool),
		}
	}
|	ReferDef
	{
		$$ = &ast.ColumnOption{
			Tp: ast.ColumnOptionReference,
			Refer: $1.(*ast.ReferenceDef),
		}
	}
|	collate CollationName
	{
		$$ = &ast.ColumnOption{Tp: ast.ColumnOptionCollate, StrValue: $2.(string)}
	}
|	visible
	{
		$$ = &ast.ColumnOption{Tp: ast.ColumnOptionVisible}
	}
|	invisible
	{
		$$ = &ast.ColumnOption{Tp: ast.ColumnOptionInvisible}
	}
|	srid LengthNum
	{
		$$ = &ast.ColumnOption{Tp: ast.ColumnOptionSRID, Expr: ast.NewValueExpr($2)}
	}

GeneratedAlways:
	/* EMPTY */
|	generated always


VirtualOrStored:
	/* EMPTY */ { $$ = false }
|	virtual   { $$ = false }
|	stored    { $$ = true }


ColumnOptionList:
	ColumnOption                  { $$ = []*ast.ColumnOption{$1.(*ast.ColumnOption)} }
|	ColumnOptionList ColumnOption { $$ = append($1.([]*ast.ColumnOption), $2.(*ast.ColumnOption)) }


ColumnOptionListOpt:
	/* EMPTY */      { $$ = []*ast.ColumnOption{} }
|	ColumnOptionList { $$ = $1.([]*ast.ColumnOption) }


ConstraintElem:
	primary key IndexName IndexTypeOpt '(' IndexColNameList ')' IndexOptionList
	{
		c := &ast.Constraint{
			Tp: ast.ConstraintPrimaryKey,
			Keys: $6.([]*ast.IndexColName),
		}
		if $8 != nil {
			c.Option = $8.(*ast.IndexOption)
		}
		if $4 != nil {
			if c.Option == nil {
				c.Option = &ast.IndexOption{}
			}
			c.Option.Tp = $4.(model.IndexType)
		}
		$$ = c
	}
|	fulltext KeyOrIndexOpt IndexName '(' IndexColNameList ')' IndexOptionList
	{
		c := &ast.Constraint{
			Tp:	ast.ConstraintFulltext,
			Keys:	$5.([]*ast.IndexColName),
			Name:	$3.(string),
		}
		if $7 != nil {
			c.Option = $7.(*ast.IndexOption)
		}
		$$ = c
	}
|	spatial KeyOrIndexOpt IndexName '(' IndexColNameList ')' IndexOptionList
	{
		c := &ast.Constraint{
			Tp:   ast.ConstraintSpatial,
			Keys: $5.([]*ast.IndexColName),
			Name: $3.(string),
		}
		if $7 != nil {
			c.Option = $7.(*ast.IndexOption)
		}
		$$ = c
	}
|	KeyOrIndex IndexName IndexTypeOpt '(' IndexKeyPartList ')' IndexOptionList
	{
		c := &ast.Constraint{
			Tp:	ast.ConstraintIndex,
			Keys:	$5.([]*ast.IndexColName),
			Name:	$2.(string),
		}
		if $7 != nil {
			c.Option = $7.(*ast.IndexOption)
		}
		if $3 != nil {
			if c.Option == nil {
				c.Option = &ast.IndexOption{}
			}
			c.Option.Tp = $3.(model.IndexType)
		}
		$$ = c
	}
|	unique KeyOrIndexOpt IndexName IndexTypeOpt '(' IndexKeyPartList ')' IndexOptionList
	{
		c := &ast.Constraint{
			Tp:	ast.ConstraintUniq,
			Keys:	$6.([]*ast.IndexColName),
			Name:	$3.(string),
		}
		if $8 != nil {
			c.Option = $8.(*ast.IndexOption)
		}
		if $4 != nil {
			if c.Option == nil {
				c.Option = &ast.IndexOption{}
			}
			c.Option.Tp = $4.(model.IndexType)
		}
		$$ = c
	}
|	foreign key IndexName '(' IndexColNameList ')' ReferDef
	{
		$$ = &ast.Constraint{
			Tp:	ast.ConstraintForeignKey,
			Keys:	$5.([]*ast.IndexColName),
			Name:	$3.(string),
			Refer:	$7.(*ast.ReferenceDef),
		}
	}
|	check '(' Expression ')'
	{
		$$ = &ast.Constraint{
			Tp:	ast.ConstraintCheck,
			Expr:	$3.(ast.ExprNode),
		}
	}


ReferDef:
	references TableName '(' IndexColNameList ')' ReferOptionsOpt
	{
		ref := $6.(*ast.ReferenceDef)
		ref.Table = $2.(*ast.TableName)
		ref.IndexColNames = $4.([]*ast.IndexColName)
		$$ = ref
	}


ReferOptionsOpt:
	%prec lowerThanOn
	{
		$$ = &ast.ReferenceDef{OnDelete: &ast.OnDeleteOpt{}, OnUpdate: &ast.OnUpdateOpt{}}
	}
|	on deleteKwd ReferOpt OnUpdateOpt
	{
		$$ = &ast.ReferenceDef{
			OnDelete: &ast.OnDeleteOpt{ReferOpt: $3.(ast.ReferOptionType)},
			OnUpdate: $4.(*ast.OnUpdateOpt),
		}
	}
|	on update ReferOpt OnDeleteOpt
	{
		$$ = &ast.ReferenceDef{
			OnDelete: $4.(*ast.OnDeleteOpt),
			OnUpdate: &ast.OnUpdateOpt{ReferOpt: $3.(ast.ReferOptionType)},
		}
	}


OnDeleteOpt:
	%prec lowerThanOn
	{
		$$ = &ast.OnDeleteOpt{}
	}
|	on deleteKwd ReferOpt
	{
		$$ = &ast.OnDeleteOpt{ReferOpt: $3.(ast.ReferOptionType)}
	}


OnUpdateOpt:
	%prec lowerThanOn
	{
		$$ = &ast.OnUpdateOpt{}
	}
|	on update ReferOpt
	{
		$$ = &ast.OnUpdateOpt{ReferOpt: $3.(ast.ReferOptionType)}
	}


ReferOpt:
	restrict    { $$ = ast.ReferOptionRestrict }
|	cascade     { $$ = ast.ReferOptionCascade }
|	set null  { $$ = ast.ReferOptionSetNull }
|	no action { $$ = ast.ReferOptionNoAction }

// MySQL expression defaults require parentheses; MariaDB also accepts bare
// functions, column references, and variables. Bare arithmetic is not allowed.
DefaultValueExpr:
	NowSymOptionFraction | SignedLiteral
|	FunctionCallGeneric
|	FunctionCallNonKeyword
|	SimpleIdent
	{
		x := $1.(*ast.ColumnNameExpr)
		if x.Name.Schema.O == "" && x.Name.Table.O == "" && strings.EqualFold(x.Name.Name.O, "now") {
			yyAppendErr(yylex, yyErrorf(yylex, "DEFAULT NOW requires parentheses"))
			return 1
		}
		$$ = x
	}
|	Variable
|	SumExpr
|	SpatialFunctionName '(' ExpressionListOpt ')'
	{
		$$ = &ast.FuncCallExpr{FnName: model.NewCIStr($1), Args: $3.([]ast.ExprNode)}
	}
|	'(' Expression ')'
	{
		$$ = &ast.ParenthesesExpr{Expr: $2}
	}
|	SubSelect
|	cast '(' Expression as CastType ')'
	{
		tp := $5.(*types.FieldType)
		flen, decimal := mysql.GetDefaultFieldLengthAndDecimalForCast(tp.Tp)
		if tp.Flen == types.UnspecifiedLength {
			tp.Flen = flen
		}
		if tp.Decimal == types.UnspecifiedLength {
			tp.Decimal = decimal
		}
		$$ = &ast.FuncCastExpr{Expr: $3, Tp: tp, FunctionType: ast.CastFunction}
	}
|	builtinCast '(' Expression as CastType ')'
	{
		tp := $5.(*types.FieldType)
		flen, decimal := mysql.GetDefaultFieldLengthAndDecimalForCast(tp.Tp)
		if tp.Flen == types.UnspecifiedLength {
			tp.Flen = flen
		}
		if tp.Decimal == types.UnspecifiedLength {
			tp.Decimal = decimal
		}
		$$ = &ast.FuncCastExpr{Expr: $3, Tp: tp, FunctionType: ast.CastFunction}
	}

NowSymOptionFraction:
	NowSym                 { $$ = &ast.FuncCallExpr{FnName: model.NewCIStr("CURRENT_TIMESTAMP")} }
|	NowSymFunc '(' ')'     { $$ = &ast.FuncCallExpr{FnName: model.NewCIStr("CURRENT_TIMESTAMP")} }
|	NowSymFunc '(' NUM ')' { $$ = &ast.FuncCallExpr{FnName: model.NewCIStr("CURRENT_TIMESTAMP"), Args: []ast.ExprNode{ast.NewValueExpr($3)}} }


/*
* See https://dev.mysql.com/doc/refman/5.7/en/date-and-time-functions.html#function_localtime
* TODO: Process other three keywords
*/
NowSymFunc:
	currentTs | localTime | localTs | builtinNow
NowSym:
	currentTs | localTime | localTs


SignedLiteral:
	Literal        { $$ = ast.NewValueExpr($1) }
|	'+' NumLiteral { $$ = &ast.UnaryOperationExpr{Op: opcode.Plus, V: ast.NewValueExpr($2)} }
|	'-' NumLiteral { $$ = &ast.UnaryOperationExpr{Op: opcode.Minus, V: ast.NewValueExpr($2)} }


NumLiteral:
	intLit
|	floatLit
|	decLit


CreateIndexStmt:
	create CreateIndexStmtUnique index Identifier CreateIndexTypeOpt on TableName '(' IndexKeyPartList ')' IndexOptionList LockClauseOpt
	{
		var indexOption *ast.IndexOption
		if $11 != nil {
			indexOption = $11.(*ast.IndexOption)
			if indexOption.Tp == model.IndexTypeInvalid {
				if $5 != nil {
					indexOption.Tp = $5.(model.IndexType)
				}
			}
		} else {
			indexOption = &ast.IndexOption{}
			if $5 != nil {
				indexOption.Tp = $5.(model.IndexType)
			}
		}
		$$ = &ast.CreateIndexStmt{
			Unique:        $2.(bool),
			IndexName:     $4,
			Table:         $7.(*ast.TableName),
			IndexColNames: $9.([]*ast.IndexColName),
			IndexOption:   indexOption,
		}
	}


CreateIndexStmtUnique:
	/* EMPTY */ { $$ = false }
|	unique    { $$ = true }


IndexColName:
	ColumnName OptFieldLen Order
	{
		//Order is parsed but just ignored as MySQL did
		$$ = &ast.IndexColName{Column: $1.(*ast.ColumnName), Length: $2.(int)}
	}


IndexKeyPart:
	IndexColName
|	'(' Expression ')'
	{
		$$ = &ast.IndexColName{Expr: $2.(ast.ExprNode), ExprOffset: yyS[yypt-2].offset}
	}


IndexKeyPartList:
	IndexKeyPart { $$ = []*ast.IndexColName{$1.(*ast.IndexColName)} }
|	IndexKeyPartList ',' IndexKeyPart { $$ = append($1.([]*ast.IndexColName), $3.(*ast.IndexColName)) }


IndexColNameList:
	IndexColName                      { $$ = []*ast.IndexColName{$1.(*ast.IndexColName)} }
|	IndexColNameList ',' IndexColName { $$ = append($1.([]*ast.IndexColName), $3.(*ast.IndexColName)) }


/**************************************AlterDatabaseStmt***************************************
 * See https://dev.mysql.com/doc/refman/5.7/en/alter-database.html
 * 'ALTER DATABASE ... UPGRADE DATA DIRECTORY NAME' is not supported yet.
 *
 *  ALTER {DATABASE | SCHEMA} [db_name]
 *   alter_specification ...
 *
 *  alter_specification:
 *   [DEFAULT] CHARACTER SET [=] charset_name
 * | [DEFAULT] COLLATE [=] collation_name
 *******************************************************************************************/
AlterDatabaseStmt:
	alter DatabaseSym DBName DatabaseOptionList
	{
		$$ = &ast.AlterDatabaseStmt{
			Name:                 $3.(string),
			AlterDefaultDatabase: false,
			Options:              $4.([]*ast.DatabaseOption),
		}
	}
|	alter DatabaseSym DatabaseOptionList
	{
		$$ = &ast.AlterDatabaseStmt{
			Name:                 "",
			AlterDefaultDatabase: true,
			Options:              $3.([]*ast.DatabaseOption),
		}
	}


/*******************************************************************
 *
 *  Create Database Statement
 *  CREATE {DATABASE | SCHEMA} [IF NOT EXISTS] db_name
 *      [create_specification] ...
 *
 *  create_specification:
 *      [DEFAULT] CHARACTER SET [=] charset_name
 *    | [DEFAULT] COLLATE [=] collation_name
 *******************************************************************/
CreateDatabaseStmt:
	create DatabaseSym IfNotExists DBName DatabaseOptionListOpt
	{
		$$ = &ast.CreateDatabaseStmt{
			IfNotExists:	$3.(bool),
			Name:		$4.(string),
			Options:	$5.([]*ast.DatabaseOption),
		}
	}


CreateSequenceStmt:
	create OrReplace TemporaryOpt sequence IfNotExists TableName SequenceOptionListOpt
	{
		$$ = &ast.CreateSequenceStmt{
			OrReplace:   $2.(bool),
			Temporary:   $3.(bool),
			IfNotExists: $5.(bool),
			Name:        $6.(*ast.TableName),
			Options:     $7.([]*ast.SequenceOption),
		}
	}


DropSequenceStmt:
	drop sequence IfExists TableNameList
	{
		$$ = &ast.DropSequenceStmt{
			IfExists: $3.(bool),
			Names:    $4.([]*ast.TableName),
		}
	}


SequenceOptionListOpt:
	/* EMPTY */
	{
		$$ = []*ast.SequenceOption{}
	}
|	SequenceOptionListOpt SequenceOption
	{
		$$ = append($1.([]*ast.SequenceOption), $2.(*ast.SequenceOption))
	}


SequenceOption:
	as IntegerType FieldOpts
	{
		tp := types.NewFieldType($2.(byte))
		for _, opt := range $3.([]*ast.TypeOpt) {
			if opt.IsUnsigned {
				tp.Flag |= mysql.UnsignedFlag
			}
			if opt.IsZerofill {
				tp.Flag |= mysql.ZerofillFlag
			}
		}
		$$ = &ast.SequenceOption{Tp: ast.SequenceOptionAs, Type: tp}
	}
|	increment by SequenceSignedInt
	{
		$$ = makeSequenceOption(ast.SequenceOptionIncrement, $3)
	}
|	increment EqOpt SequenceSignedInt
	{
		$$ = makeSequenceOption(ast.SequenceOptionIncrement, $3)
	}
|	minvalue EqOpt SequenceBoundValue
	{
		$$ = makeSequenceOption(ast.SequenceOptionMinValue, $3)
	}
|	no minvalue
	{
		$$ = &ast.SequenceOption{Tp: ast.SequenceOptionNoMinValue}
	}
|	nominvalue
	{
		$$ = &ast.SequenceOption{Tp: ast.SequenceOptionNoMinValue}
	}
|	maxValue EqOpt SequenceBoundValue
	{
		$$ = makeSequenceOption(ast.SequenceOptionMaxValue, $3)
	}
|	no maxValue
	{
		$$ = &ast.SequenceOption{Tp: ast.SequenceOptionNoMaxValue}
	}
|	nomaxvalue
	{
		$$ = &ast.SequenceOption{Tp: ast.SequenceOptionNoMaxValue}
	}
|	start with SequenceStartValue
	{
		$$ = makeSequenceOption(ast.SequenceOptionStart, $3)
	}
|	start EqOpt SequenceStartValue
	{
		$$ = makeSequenceOption(ast.SequenceOptionStart, $3)
	}
|	cache EqOpt SequenceSignedInt
	{
		$$ = makeSequenceOption(ast.SequenceOptionCache, $3)
	}
|	nocache
	{
		$$ = &ast.SequenceOption{Tp: ast.SequenceOptionNoCache}
	}
|	cycle
	{
		$$ = &ast.SequenceOption{Tp: ast.SequenceOptionCycle}
	}
|	no cycle
	{
		$$ = &ast.SequenceOption{Tp: ast.SequenceOptionNoCycle}
	}
|	nocycle
	{
		$$ = &ast.SequenceOption{Tp: ast.SequenceOptionNoCycle}
	}


SequenceSignedInt:
	intLit
	{
		value, err := sequenceSignedInt($1, false)
		if err != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "%v", err))
			return 1
		}
		$$ = value
	}
|	'+' intLit
	{
		value, err := sequenceSignedInt($2, false)
		if err != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "%v", err))
			return 1
		}
		$$ = value
	}
|	'-' intLit
	{
		value, err := sequenceSignedInt($2, true)
		if err != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "%v", err))
			return 1
		}
		$$ = value
	}


SequenceStartValue:
	intLit       { $$ = ast.NewValueExpr($1) }
|	'+' intLit   { $$ = ast.NewValueExpr($2) }
|	'-' intLit
	{
		value, err := sequenceSignedInt($2, true)
		if err != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "%v", err))
			return 1
		}
		$$ = value
	}


SequenceBoundValue:
	NumLiteral     { $$ = ast.NewValueExpr($1) }
|	'+' NumLiteral { $$ = &ast.UnaryOperationExpr{Op: opcode.Plus, V: ast.NewValueExpr($2)} }
|	'-' NumLiteral { $$ = &ast.UnaryOperationExpr{Op: opcode.Minus, V: ast.NewValueExpr($2)} }


UndoOpt:
	/* EMPTY */ { $$ = false }
|	undo      { $$ = true }


CreateTablespaceStmt:
	create UndoOpt tablespace Identifier TablespaceOptionList
	{
		$$ = &ast.CreateTablespaceStmt{
			Undo:    $2.(bool),
			Name:    $4,
			Options: $5.([]*ast.TablespaceOption),
		}
	}


AlterTablespaceStmt:
	alter UndoOpt tablespace Identifier TablespaceOptionList
	{
		$$ = &ast.AlterTablespaceStmt{
			Undo:    $2.(bool),
			Name:    $4,
			Options: $5.([]*ast.TablespaceOption),
		}
	}
|	alter UndoOpt tablespace Identifier rename to Identifier
	{
		$$ = &ast.AlterTablespaceStmt{
			Undo:     $2.(bool),
			Name:     $4,
			RenameTo: $7,
		}
	}


DropTablespaceStmt:
	drop UndoOpt tablespace Identifier
	{
		$$ = &ast.DropTablespaceStmt{
			Undo: $2.(bool),
			Name: $4,
		}
	}
|	drop UndoOpt tablespace Identifier TablespaceOption
	{
		$$ = &ast.DropTablespaceStmt{
			Undo:   $2.(bool),
			Name:   $4,
			Option: $5.(*ast.TablespaceOption),
		}
	}


TablespaceOptionList:
	/* EMPTY */
	{
		$$ = []*ast.TablespaceOption{}
	}
|	TablespaceOptionList TablespaceOption
	{
		$$ = append($1.([]*ast.TablespaceOption), $2.(*ast.TablespaceOption))
	}


TablespaceOption:
	add datafile stringLit
	{
		$$ = &ast.TablespaceOption{Tp: ast.TablespaceOptionDataFile, Str: $3}
	}
|	fileBlockSize EqOpt LengthNum
	{
		$$ = &ast.TablespaceOption{Tp: ast.TablespaceOptionFileBlockSize, Val: $3.(uint64)}
	}
|	extentSize EqOpt LengthNum
	{
		$$ = &ast.TablespaceOption{Tp: ast.TablespaceOptionExtentSize, Val: $3.(uint64)}
	}
|	initialSize EqOpt LengthNum
	{
		$$ = &ast.TablespaceOption{Tp: ast.TablespaceOptionInitialSize, Val: $3.(uint64)}
	}
|	autoextendSize EqOpt LengthNum
	{
		$$ = &ast.TablespaceOption{Tp: ast.TablespaceOptionAutoextendSize, Val: $3.(uint64)}
	}
|	maxSize EqOpt LengthNum
	{
		$$ = &ast.TablespaceOption{Tp: ast.TablespaceOptionMaxSize, Val: $3.(uint64)}
	}
|	nodegroup EqOpt LengthNum
	{
		$$ = &ast.TablespaceOption{Tp: ast.TablespaceOptionNodegroup, Val: $3.(uint64)}
	}
|	wait
	{
		$$ = &ast.TablespaceOption{Tp: ast.TablespaceOptionWait}
	}
|	no wait
	{
		$$ = &ast.TablespaceOption{Tp: ast.TablespaceOptionNoWait}
	}
|	comment EqOpt stringLit
	{
		$$ = &ast.TablespaceOption{Tp: ast.TablespaceOptionComment, Str: $3}
	}
|	engine EqOpt Identifier
	{
		$$ = &ast.TablespaceOption{Tp: ast.TablespaceOptionEngine, Str: $3}
	}


CreateServerStmt:
	create server Identifier foreign data wrapper stringLit ServerOptionListOpt
	{
		$$ = &ast.CreateServerStmt{
			Name:    $3,
			Wrapper: $7,
			Options: $8.([]*ast.ServerOption),
		}
	}


AlterServerStmt:
	alter server Identifier ServerOptionListOpt
	{
		$$ = &ast.AlterServerStmt{
			Name:    $3,
			Options: $4.([]*ast.ServerOption),
		}
	}


DropServerStmt:
	drop server IfExists Identifier
	{
		$$ = &ast.DropServerStmt{
			IfExists: $3.(bool),
			Name:     $4,
		}
	}


ServerOptionListOpt:
	/* EMPTY */
	{
		$$ = []*ast.ServerOption{}
	}
|	options '(' ServerOptionList ')'
	{
		$$ = $3.([]*ast.ServerOption)
	}


ServerOptionList:
	ServerOption
	{
		$$ = []*ast.ServerOption{$1.(*ast.ServerOption)}
	}
|	ServerOptionList ',' ServerOption
	{
		$$ = append($1.([]*ast.ServerOption), $3.(*ast.ServerOption))
	}


ServerOption:
	ServerOptionName stringLit
	{
		$$ = &ast.ServerOption{Name: $1, Value: $2}
	}


ServerOptionName:
	Identifier   { $$ = $1 }
|	database   { $$ = "DATABASE" }


ResourceGroupType:
	Identifier
	{
		$$ = strings.EqualFold($1, "SYSTEM")
	}


VCPUListOpt:
	/* EMPTY */
	{
		$$ = []*ast.ResourceGroupVCPU{}
	}
|	vcpu EqOpt VCPUList
	{
		$$ = $3.([]*ast.ResourceGroupVCPU)
	}


VCPUList:
	ResourceGroupVCPU
	{
		$$ = []*ast.ResourceGroupVCPU{$1.(*ast.ResourceGroupVCPU)}
	}
|	VCPUList ',' ResourceGroupVCPU
	{
		$$ = append($1.([]*ast.ResourceGroupVCPU), $3.(*ast.ResourceGroupVCPU))
	}


ResourceGroupVCPU:
	NUM
	{
		$$ = &ast.ResourceGroupVCPU{From: getUint64FromNUM($1), To: getUint64FromNUM($1)}
	}
|	NUM '-' NUM
	{
		$$ = &ast.ResourceGroupVCPU{From: getUint64FromNUM($1), To: getUint64FromNUM($3)}
	}


ThreadPriorityOpt:
	/* EMPTY */
	{
		$$ = (*int)(nil)
	}
|	threadPriority EqOpt intLit
	{
		v, err := strconv.Atoi(strconv.FormatUint(getUint64FromNUM($3), 10))
		if err != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "thread priority is out of range"))
			return 1
		}
		$$ = &v
	}


ResourceGroupStateOpt:
	/* EMPTY */ { $$ = ast.ResourceGroupStateNone }
|	enable    { $$ = ast.ResourceGroupStateEnable }
|	disable   { $$ = ast.ResourceGroupStateDisable }


CreateResourceGroupStmt:
	create resource group Identifier typeSym EqOpt ResourceGroupType VCPUListOpt ThreadPriorityOpt ResourceGroupStateOpt
	{
		var tp *int
		if $9 != nil {
			tp = $9.(*int)
		}
		$$ = &ast.CreateResourceGroupStmt{
			Name:           $4,
			IsSystem:       $7.(bool),
			VCPU:           $8.([]*ast.ResourceGroupVCPU),
			ThreadPriority: tp,
			State:          $10.(int),
		}
	}


AlterResourceGroupStmt:
	alter resource group Identifier VCPUListOpt ThreadPriorityOpt ResourceGroupStateOpt
	{
		var tp *int
		if $6 != nil {
			tp = $6.(*int)
		}
		$$ = &ast.AlterResourceGroupStmt{
			Name:           $4,
			VCPU:           $5.([]*ast.ResourceGroupVCPU),
			ThreadPriority: tp,
			State:          $7.(int),
		}
	}


ForceOpt:
	/* EMPTY */ { $$ = false }
|	force     { $$ = true }


DropResourceGroupStmt:
	drop resource group Identifier ForceOpt
	{
		$$ = &ast.DropResourceGroupStmt{Name: $4, Force: $5.(bool)}
	}


LogfileGroupOptionList:
	/* EMPTY */
	{
		$$ = []*ast.LogfileGroupOption{}
	}
|	LogfileGroupOptionList LogfileGroupOption
	{
		$$ = append($1.([]*ast.LogfileGroupOption), $2.(*ast.LogfileGroupOption))
	}


LogfileGroupOption:
	initialSize EqOpt LengthNum
	{
		$$ = &ast.LogfileGroupOption{Tp: ast.LogfileGroupOptionInitialSize, Val: $3.(uint64)}
	}
|	undoBufferSize EqOpt LengthNum
	{
		$$ = &ast.LogfileGroupOption{Tp: ast.LogfileGroupOptionUndoBufferSize, Val: $3.(uint64)}
	}
|	redoBufferSize EqOpt LengthNum
	{
		$$ = &ast.LogfileGroupOption{Tp: ast.LogfileGroupOptionRedoBufferSize, Val: $3.(uint64)}
	}
|	nodegroup EqOpt LengthNum
	{
		$$ = &ast.LogfileGroupOption{Tp: ast.LogfileGroupOptionNodegroup, Val: $3.(uint64)}
	}
|	wait
	{
		$$ = &ast.LogfileGroupOption{Tp: ast.LogfileGroupOptionWait}
	}
|	comment EqOpt stringLit
	{
		$$ = &ast.LogfileGroupOption{Tp: ast.LogfileGroupOptionComment, Str: $3}
	}
|	engine EqOpt Identifier
	{
		$$ = &ast.LogfileGroupOption{Tp: ast.LogfileGroupOptionEngine, Str: $3}
	}


CreateLogfileGroupStmt:
	create logfile group Identifier add undofile stringLit LogfileGroupOptionList
	{
		$$ = &ast.CreateLogfileGroupStmt{
			Name:     $4,
			UndoFile: $7,
			Options:  $8.([]*ast.LogfileGroupOption),
		}
	}


AlterLogfileGroupStmt:
	alter logfile group Identifier add undofile stringLit LogfileGroupOptionList
	{
		$$ = &ast.AlterLogfileGroupStmt{
			Name:     $4,
			UndoFile: $7,
			Options:  $8.([]*ast.LogfileGroupOption),
		}
	}


DropLogfileGroupStmt:
	drop logfile group Identifier LogfileGroupOptionList
	{
		$$ = &ast.DropLogfileGroupStmt{
			Name:    $4,
			Options: $5.([]*ast.LogfileGroupOption),
		}
	}


DBName:
	Identifier { $$ = $1 }


DatabaseOption:
	DefaultKwdOpt CharsetOrCharacterSet EqOpt CharsetName
	{
		$$ = &ast.DatabaseOption{Tp: ast.DatabaseOptionCharset, Value: $4.(string)}
	}
|	DefaultKwdOpt collate EqOpt CollationName
	{
		$$ = &ast.DatabaseOption{Tp: ast.DatabaseOptionCollate, Value: $4.(string)}
	}


DatabaseOptionListOpt:
	/* EMPTY */ { $$ = []*ast.DatabaseOption{} }
|	DatabaseOptionList


DatabaseOptionList:
	DatabaseOption                    { $$ = []*ast.DatabaseOption{$1.(*ast.DatabaseOption)} }
|	DatabaseOptionList DatabaseOption { $$ = append($1.([]*ast.DatabaseOption), $2.(*ast.DatabaseOption)) }


/*******************************************************************
 *
 *  Create Table Statement
 *
 *  Example:
 *      CREATE TABLE Persons
 *      (
 *          P_Id int NOT NULL,
 *          LastName varchar(255) NOT NULL,
 *          FirstName varchar(255),
 *          Address varchar(255),
 *          City varchar(255),
 *          PRIMARY KEY (P_Id)
 *      )
 *******************************************************************/

CreateTableStmt:
	create OrReplace TemporaryOpt tableKwd IfNotExists TableName CreateTableBody
	{
		stmt := $7.(*ast.CreateTableStmt)
		stmt.OrReplace = $2.(bool)
		stmt.Temporary = $3.(bool)
		stmt.Table = $6.(*ast.TableName)
		stmt.IfNotExists = $5.(bool)
		$$ = stmt
	}
|	create OrReplace TemporaryOpt tableKwd IfNotExists TableName LikeTableWithOrWithoutParen
	{
		$$ = &ast.CreateTableStmt{
			OrReplace:   $2.(bool),
			Temporary:   $3.(bool),
			Table:       $6.(*ast.TableName),
			ReferTable:  $7.(*ast.TableName),
			IfNotExists: $5.(bool),
		}
	}


// Shift '(' before choosing between a column definition and a CTAS query.
CreateTableBody:
	CreateTableDefinition CreateTableTail
	{
		stmt := $1.(*ast.CreateTableStmt)
		tail := $2.(*ast.CreateTableStmt)
		stmt.Options = tail.Options
		stmt.Partition = tail.Partition
		stmt.OnDuplicate = tail.OnDuplicate
		stmt.Select = tail.Select
		$$ = stmt
	}
|	CreateTableTail


CreateTableTail:
	CreateTableOptionList PartitionOpt DuplicateOpt AsOpt CreateTableSelectOpt
	{
		stmt := $5.(*ast.CreateTableStmt)
		stmt.Options = $1.([]*ast.TableOption)
		if $2 != nil {
			stmt.Partition = $2.(*ast.PartitionOptions)
		}
		stmt.OnDuplicate = $3.(ast.OnDuplicateKeyHandlingType)
		$$ = stmt
	}
|	PartitionByOpt DuplicateOpt AsOpt CreateTableSelectOpt
	{
		stmt := $4.(*ast.CreateTableStmt)
		if $1 != nil {
			stmt.Partition = $1.(*ast.PartitionOptions)
		}
		stmt.OnDuplicate = $2.(ast.OnDuplicateKeyHandlingType)
		$$ = stmt
	}
|	ignore AsOpt CreateTableSelectOpt
	{
		stmt := $3.(*ast.CreateTableStmt)
		stmt.OnDuplicate = ast.OnDuplicateKeyHandlingIgnore
		$$ = stmt
	}
|	replace AsOpt CreateTableSelectOpt
	{
		stmt := $3.(*ast.CreateTableStmt)
		stmt.OnDuplicate = ast.OnDuplicateKeyHandlingReplace
		$$ = stmt
	}
|	as CreateTableSelectOpt { $$ = $2 }
|	CreateTableSelectOpt


TemporaryOpt:
	/* EMPTY */ { $$ = false }
|	temporary { $$ = true }


DefaultKwdOpt:
	%prec lowerThanCharsetKwd {}
|	defaultKwd


PartitionOpt:
	/* EMPTY */ { $$ = nil }
|	PartitionByOpt { $$ = $1 }


PartitionByOpt:
	partition by LinearOpt key '(' ColumnNameListOpt ')' PartitionNumOpt PartitionDefinitionListOpt
	{
		part := &ast.PartitionOptions{
			Tp: model.PartitionTypeKey,
			ColumnNames: $6.([]*ast.ColumnName),
			Num: 1, HasNum: true, Linear: $3 != "",
		}
		if $8 != nil {
			part.Num = getUint64FromNUM($8)
		}
		if $9 != nil {
			part.Definitions = $9.([]*ast.PartitionDefinition)
			for _, def := range part.Definitions {
				if def.LessThan != nil {
					yyAppendErr(yylex, yyErrorf(yylex, "KEY partitions do not accept VALUES LESS THAN"))
					return 1
				}
			}
		}
		$$ = part
	}
|	partition by LinearOpt hash '(' Expression ')' PartitionNumOpt
	{
		tmp := &ast.PartitionOptions{
			Tp: model.PartitionTypeHash,
			Expr: $6.(ast.ExprNode),
			Num: 1, HasNum: true, Linear: $3 != "",
		}
		if $8 != nil {
			tmp.Num = getUint64FromNUM($8)
		}
		$$ = tmp
	}
|	partition by rangeKwd '(' Expression ')' PartitionNumOpt SubPartitionOpt PartitionDefinitionListOpt
	{
		var defs []*ast.PartitionDefinition
		if $9 != nil {
			defs = $9.([]*ast.PartitionDefinition)
		}
		part := &ast.PartitionOptions{
			Tp:		model.PartitionTypeRange,
			Expr:		$5.(ast.ExprNode),
			Definitions:	defs,
		}
		if $7 != nil {
			part.Num, part.HasNum = getUint64FromNUM($7), true
		}
		if $8 != nil {
			part.Sub = $8.(*ast.PartitionOptions)
		}
		$$ = part
	}
|	partition by rangeKwd columns '(' ColumnNameList ')' PartitionNumOpt SubPartitionOpt PartitionDefinitionListOpt
	{
		var defs []*ast.PartitionDefinition
		if $10 != nil {
			defs = $10.([]*ast.PartitionDefinition)
		}
		part := &ast.PartitionOptions{
			Tp:		model.PartitionTypeRange,
			ColumnNames:	$6.([]*ast.ColumnName),
			Definitions:	defs,
		}
		if $8 != nil {
			part.Num, part.HasNum = getUint64FromNUM($8), true
		}
		if $9 != nil {
			part.Sub = $9.(*ast.PartitionOptions)
		}
		$$ = part
	}


LinearOpt:
	/* EMPTY */	{ $$ = "" }
|	linear	{ $$ = $1 }


SubPartitionOpt:
	/* EMPTY */ { $$ = nil }
|	subpartition by LinearOpt hash '(' Expression ')' SubPartitionNumOpt
	{
		part := &ast.PartitionOptions{Tp: model.PartitionTypeHash, Expr: $6.(ast.ExprNode), Linear: $3 != "", IsSubPartition: true}
		if $8 != nil {
			part.Num, part.HasNum = getUint64FromNUM($8), true
		}
		$$ = part
	}
|	subpartition by LinearOpt key '(' ColumnNameListOpt ')' SubPartitionNumOpt
	{
		part := &ast.PartitionOptions{Tp: model.PartitionTypeKey, ColumnNames: $6.([]*ast.ColumnName), Linear: $3 != "", IsSubPartition: true}
		if $8 != nil {
			part.Num, part.HasNum = getUint64FromNUM($8), true
		}
		$$ = part
	}


SubPartitionNumOpt:
	/* EMPTY */ { $$ = nil }
|	subpartitions NUM { $$ = $2 }


PartitionNumOpt:
	/* EMPTY */      { $$ = nil }
|	partitions NUM { $$ = $2 }


PartitionDefinitionListOpt:
	/* empty */ %prec lowerThanCreateTableSelect
	{
		$$ = nil
	}
|	'(' PartitionDefinitionList ')'
	{
		$$ = $2.([]*ast.PartitionDefinition)
	}


PartitionDefinitionList:
	PartitionDefinition
	{
		$$ = []*ast.PartitionDefinition{$1.(*ast.PartitionDefinition)}
	}
|	PartitionDefinitionList ',' PartitionDefinition
	{
		$$ = append($1.([]*ast.PartitionDefinition), $3.(*ast.PartitionDefinition))
	}


PartitionDefinition:
	PartitionKeywordOpt Identifier PartDefValuesOpt PartDefOptionsOpt
	{
		partDef := &ast.PartitionDefinition{
			Name: model.NewCIStr($2),
			OmitPartitionKeyword: $1.(bool),
			NameOffset: yyS[yypt-2].offset,
		}
		switch $3.(type) {
		case []ast.ExprNode:
			partDef.LessThan = $3.([]ast.ExprNode)
		case ast.ExprNode:
			partDef.LessThan = make([]ast.ExprNode, 1)
			partDef.LessThan[0] = $3.(ast.ExprNode)
		}

		if $4 != nil {
			partDef.Options = $4.([]*ast.TableOption)
			for _, option := range partDef.Options {
				if option.Tp == ast.TableOptionComment {
					partDef.Comment = option.StrValue
					break
				}
			}
		}
		$$ = partDef
	}


PartitionKeywordOpt:
	/* EMPTY */ { $$ = true }
|	partition { $$ = false }


PartDefOptionsOpt:
	/* EMPTY */       { $$ = nil }
|	PartDefOptionList { $$ = $1 }


PartDefOptionList:
	PartDefOption { $$ = []*ast.TableOption{$1.(*ast.TableOption)} }
|	PartDefOptionList PartDefOption
	{
		$$ = append($1.([]*ast.TableOption), $2.(*ast.TableOption))
	}


PartDefOption:
	comment EqOpt stringLit { $$ = &ast.TableOption{Tp: ast.TableOptionComment, StrValue: $3} }
|	engine EqOpt Identifier { $$ = &ast.TableOption{Tp: ast.TableOptionEngine, StrValue: $3} }
|	tablespace EqOpt Identifier { $$ = &ast.TableOption{Tp: ast.TableOptionTablespace, StrValue: $3} }


PartDefValuesOpt:
	/* EMPTY */                                             { $$ = nil }
|	values less than maxValue                       { $$ = &ast.MaxValueExpr{} }
|	values less than '(' MaxValueOrExpressionList ')' { $$ = $5 }


DuplicateOpt:
	/* EMPTY */ { $$ = ast.OnDuplicateKeyHandlingError }
|	ignore    { $$ = ast.OnDuplicateKeyHandlingIgnore }
|	replace   { $$ = ast.OnDuplicateKeyHandlingReplace }


AsOpt:
	/* EMPTY */ {}
|	as {}


CreateTableSelectOpt:
	/* EMPTY */ { $$ = &ast.CreateTableStmt{} }
|	SelectStmt  { $$ = &ast.CreateTableStmt{Select: $1} }
|	SetOprStmt   { $$ = &ast.CreateTableStmt{Select: $1} }
|	WithSelectStmt { $$ = &ast.CreateTableStmt{Select: $1} }
|	SubSelect %prec createTableSelect
	{
		$$ = &ast.CreateTableStmt{Select: $1}
	}


LikeTableWithOrWithoutParen:
	like TableName         { $$ = $2 }
|	'(' like TableName ')' { $$ = $3 }


/*******************************************************************
 *
 *  Create View Statement
 *
 *  Example:
 *      CREATE VIEW OR REPLACE ALGORITHM = MERGE DEFINER="root@localhost" SQL SECURITY = definer view_name (col1,col2)
 *          as select Col1,Col2 from table WITH LOCAL CHECK OPTION
 *******************************************************************/
CreateViewStmt:
    create OrReplace ViewAlgorithm ViewDefiner ViewSQLSecurity view ViewName ViewFieldList as ViewSelect ViewCheckOption
    {
		startOffset := yyGetParser(yylex).startOffset(&yyS[yypt-1])
		selStmt := $10
		selStmt.SetText(strings.TrimSpace(yyGetParser(yylex).src[startOffset:]))
		x := &ast.CreateViewStmt {
			OrReplace:     $2.(bool),
			ViewName:      $7.(*ast.TableName),
			Select:        selStmt,
			Definer:       $4.(*auth.UserIdentity),
		}
		if alg := $3.(*model.ViewAlgorithm); alg != nil {
			x.Algorithm = *alg
		}
		if sec := $5.(*model.ViewSecurity); sec != nil {
			x.Security = *sec
		}
		if $8 != nil{
			x.Cols = $8.([]model.CIStr)
		}
		if $11 !=nil {
		    x.HasCheckOption = true
		    x.CheckOption = $11.(model.ViewCheckOption)
		    endOffset := yyGetParser(yylex).startOffset(&yyS[yypt])
		    selStmt.SetText(strings.TrimSpace(yyGetParser(yylex).src[startOffset:endOffset]))
		} else {
		    x.CheckOption = model.CheckOptionCascaded
		}
		$$ = x
	}

AlterViewStmt:
	alter ViewAlgorithm ViewDefiner ViewSQLSecurity view ViewName ViewFieldList as ViewSelect ViewCheckOption
	{
		startOffset := yyGetParser(yylex).startOffset(&yyS[yypt-1])
		selStmt := $9
		selStmt.SetText(strings.TrimSpace(yyGetParser(yylex).src[startOffset:]))
		x := &ast.AlterViewStmt{
			ViewName:  $6.(*ast.TableName),
			Select:    selStmt,
			Definer:   $3.(*auth.UserIdentity),
		}
		if alg := $2.(*model.ViewAlgorithm); alg != nil {
			x.Algorithm = *alg
		}
		if sec := $4.(*model.ViewSecurity); sec != nil {
			x.Security = *sec
		}
		if $7 != nil {
			x.Cols = $7.([]model.CIStr)
		}
		if $10 != nil {
			x.HasCheckOption = true
			x.CheckOption = $10.(model.ViewCheckOption)
			endOffset := yyGetParser(yylex).startOffset(&yyS[yypt])
			selStmt.SetText(strings.TrimSpace(yyGetParser(yylex).src[startOffset:endOffset]))
		} else {
			x.CheckOption = model.CheckOptionCascaded
		}
		$$ = x
	}


ViewSelect:
	SelectStmt
|	SetOprStmt
|	WithSelectStmt
|	ParenthesizedQueryStmt


/******************************************************************
 * Stored programs: CREATE/ALTER FUNCTION|PROCEDURE|TRIGGER|EVENT
 ******************************************************************/

CreateFunctionStmt:
	create OrReplace ViewAlgorithm ViewDefiner ViewSQLSecurity function IfNotExists RoutineName FuncParams returns row typeSym of TableName RoutineCharacteristicList SpStmtOpt
	{
		if $3.(*model.ViewAlgorithm) != nil || $5.(*model.ViewSecurity) != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "ALGORITHM and SQL SECURITY are not valid for a function"))
			return 1
		}
		body, _ := $16.(ast.StmtNode)
		if body == nil {
			yyAppendErr(yylex, yyErrorf(yylex, "a function body is required"))
			return 1
		}
		$$ = &ast.CreateFunctionStmt{
			OrReplace:       $2.(bool),
			Definer:         $4.(*auth.UserIdentity),
			IfNotExists:     $7.(bool),
			Name:            $8.(*ast.TableName),
			Params:          $9.([]*ast.FuncParam),
			ReturnsRowType:  $14.(*ast.TableName),
			Characteristics: $15.(*ast.RoutineCharacteristics),
			Body:            body,
		}
	}
|	create OrReplace ViewAlgorithm ViewDefiner ViewSQLSecurity function IfNotExists RoutineName FuncParams returns Type RoutineCharacteristicList SpStmtOpt
	{
		if $3.(*model.ViewAlgorithm) != nil || $5.(*model.ViewSecurity) != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "ALGORITHM and SQL SECURITY are not valid for a function"))
			return 1
		}
		body, _ := $13.(ast.StmtNode)
		if body == nil {
			yyAppendErr(yylex, yyErrorf(yylex, "a function body is required"))
			return 1
		}
		$$ = &ast.CreateFunctionStmt{
			OrReplace:       $2.(bool),
			Definer:         $4.(*auth.UserIdentity),
			IfNotExists:     $7.(bool),
			Name:            $8.(*ast.TableName),
			Params:          $9.([]*ast.FuncParam),
			Returns:         $11.(*types.FieldType),
			Characteristics: $12.(*ast.RoutineCharacteristics),
			Body:            body,
		}
	}
|	create OrReplace ViewAlgorithm ViewDefiner ViewSQLSecurity function IfNotExists TableName returns UDFReturnType soname stringLit
	{
		$$ = &ast.CreateFunctionStmt{
			OrReplace:   $2.(bool),
			IfNotExists: $7.(bool),
			Name:        $8.(*ast.TableName),
			Returns:     $10.(*types.FieldType),
			Soname:      $12,
		}
	}
|	create aggregate function IfNotExists TableName returns UDFReturnType soname stringLit
	{
		$$ = &ast.CreateFunctionStmt{
			Aggregate:   true,
			IfNotExists: $4.(bool),
			Name:        $5.(*ast.TableName),
			Returns:     $7.(*types.FieldType),
			Soname:      $9,
		}
	}


UDFReturnType:
	stringType
	{
		$$ = &types.FieldType{Tp: mysql.TypeString, Flen: types.UnspecifiedLength}
	}
|	Type
	{
		$$ = $1
	}


CreateProcedureStmt:
	create OrReplace ViewAlgorithm ViewDefiner ViewSQLSecurity procedure IfNotExists RoutineName FuncParams RoutineCharacteristicList SpStmtOpt
	{
		if $3.(*model.ViewAlgorithm) != nil || $5.(*model.ViewSecurity) != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "ALGORITHM and SQL SECURITY are not valid for a procedure"))
			return 1
		}
		body, _ := $11.(ast.StmtNode)
		if body == nil {
			yyAppendErr(yylex, yyErrorf(yylex, "a procedure body is required"))
			return 1
		}
		$$ = &ast.CreateProcedureStmt{
			OrReplace:       $2.(bool),
			Definer:         $4.(*auth.UserIdentity),
			IfNotExists:     $7.(bool),
			Name:            $8.(*ast.TableName),
			Params:          $9.([]*ast.FuncParam),
			Characteristics: $10.(*ast.RoutineCharacteristics),
			Body:            body,
		}
	}


AlterFunctionStmt:
	alter function TableName RoutineCharacteristicList
	{
		if !$4.(*ast.RoutineCharacteristics).HasAny() {
			yyAppendErr(yylex, yyErrorf(yylex, "at least one routine characteristic is required"))
			return 1
		}
		$$ = &ast.AlterFunctionStmt{
			Name:            $3.(*ast.TableName),
			Characteristics: $4.(*ast.RoutineCharacteristics),
		}
	}


AlterProcedureStmt:
	alter procedure TableName RoutineCharacteristicList
	{
		if !$4.(*ast.RoutineCharacteristics).HasAny() {
			yyAppendErr(yylex, yyErrorf(yylex, "at least one routine characteristic is required"))
			return 1
		}
		$$ = &ast.AlterProcedureStmt{
			Name:            $3.(*ast.TableName),
			Characteristics: $4.(*ast.RoutineCharacteristics),
		}
	}


CreateTriggerStmt:
	create OrReplace ViewAlgorithm ViewDefiner ViewSQLSecurity trigger IfNotExists TableName TriggerTime TriggerEvent TriggerUpdateColumnsOpt on TableName forKwd each row TriggerOrderOpt SpStmtOpt
	{
		if $3.(*model.ViewAlgorithm) != nil || $5.(*model.ViewSecurity) != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "ALGORITHM and SQL SECURITY are not valid for a trigger"))
			return 1
		}
		body, _ := $18.(ast.StmtNode)
		if body == nil {
			yyAppendErr(yylex, yyErrorf(yylex, "a trigger body is required"))
			return 1
		}
		x := &ast.CreateTriggerStmt{
			OrReplace:   $2.(bool),
			Definer:     $4.(*auth.UserIdentity),
			IfNotExists: $7.(bool),
			Name:        $8.(*ast.TableName),
			Time:        $9.(string),
			Event:       $10.(string),
			Table:       $13.(*ast.TableName),
			Body:        body,
		}
		if cols := $11.([]*ast.ColumnName); cols != nil {
			if x.Event != "UPDATE" {
				yyAppendErr(yylex, yyErrorf(yylex, "UPDATE OF clause is only valid for UPDATE triggers"))
				return 1
			}
			x.UpdateColumns = cols
		}
		if order := $17.([]string); order != nil {
			x.Order = order[0]
			x.OtherTriggerName = order[1]
		}
		$$ = x
	}


CreateEventStmt:
	create OrReplace ViewAlgorithm ViewDefiner ViewSQLSecurity event IfNotExists TableName EventSchedule EventOnCompletionOpt EventStatusOpt EventCommentOpt do SpStmtOpt
	{
		if $3.(*model.ViewAlgorithm) != nil || $5.(*model.ViewSecurity) != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "ALGORITHM and SQL SECURITY are not valid for an event"))
			return 1
		}
		body, _ := $14.(ast.StmtNode)
		if body == nil {
			yyAppendErr(yylex, yyErrorf(yylex, "an event body is required"))
			return 1
		}
		x := &ast.CreateEventStmt{
			Definer:     $4.(*auth.UserIdentity),
			IfNotExists: $7.(bool),
			Name:        $8.(*ast.TableName),
			Schedule:    $9.(*ast.EventSchedule),
			Status:      $11.(ast.EventStatus),
			Body:        body,
		}
		if preserve := $10.(*bool); preserve != nil {
			x.OnCompletion = true
			x.Preserve = *preserve
		}
		if comment := $12.(*string); comment != nil {
			x.Comment = *comment
			x.CommentSpecified = true
		}
		$$ = x
	}


AlterEventStmt:
	alter ViewAlgorithm ViewDefiner ViewSQLSecurity event TableName AlterEventClauses EventDoOpt
	{
		if $2.(*model.ViewAlgorithm) != nil || $4.(*model.ViewSecurity) != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "ALGORITHM and SQL SECURITY are not valid for an event"))
			return 1
		}
		x := $7.(*ast.AlterEventStmt)
		x.Definer = $3.(*auth.UserIdentity)
		x.Name = $6.(*ast.TableName)
		if body, ok := $8.(ast.StmtNode); ok && body != nil {
			x.Body = body
		}
		if x.Schedule == nil && !x.OnCompletion && x.Status == ast.EventStatusDefault &&
			!x.CommentSpecified && x.RenameName == nil && x.Body == nil {
			yyAppendErr(yylex, yyErrorf(yylex, "ALTER EVENT requires at least one clause"))
			return 1
		}
		$$ = x
	}


AlterEventClause:
	EventSchedule
	{
		$$ = &ast.AlterEventStmt{Schedule: $1.(*ast.EventSchedule)}
	}
|	on completion preserve
	{
		$$ = &ast.AlterEventStmt{OnCompletion: true, Preserve: true}
	}
|	on completion not preserve
	{
		$$ = &ast.AlterEventStmt{OnCompletion: true, Preserve: false}
	}
|	enable
	{
		$$ = &ast.AlterEventStmt{Status: ast.EventStatusEnable}
	}
|	disable %prec empty
	{
		$$ = &ast.AlterEventStmt{Status: ast.EventStatusDisable}
	}
|	disable on slave
	{
		$$ = &ast.AlterEventStmt{Status: ast.EventStatusDisableOnSlave}
	}
|	comment stringLit
	{
		$$ = &ast.AlterEventStmt{Comment: $2, CommentSpecified: true}
	}
|	rename to TableName
	{
		$$ = &ast.AlterEventStmt{RenameName: $3.(*ast.TableName)}
	}


AlterEventClauses:
	/* EMPTY */
	{
		$$ = &ast.AlterEventStmt{}
	}
|	AlterEventClauses AlterEventClause
	{
		acc := $1.(*ast.AlterEventStmt)
		c := $2.(*ast.AlterEventStmt)
		if c.Schedule != nil {
			acc.Schedule = c.Schedule
		}
		if c.OnCompletion {
			acc.OnCompletion = true
			acc.Preserve = c.Preserve
		}
		if c.Status != ast.EventStatusDefault {
			acc.Status = c.Status
		}
		if c.CommentSpecified {
			acc.Comment = c.Comment
			acc.CommentSpecified = true
		}
		if c.RenameName != nil {
			acc.RenameName = c.RenameName
		}
		$$ = acc
	}


EventDoOpt:
	/* EMPTY */
	{
		$$ = (ast.StmtNode)(nil)
	}
|	do SpStmtOpt
	{
		$$ = $2.(ast.StmtNode)
	}


FuncParams:
	'(' ')'
	{
		$$ = []*ast.FuncParam{}
	}
|	'(' FuncParamList ')'
	{
		$$ = $2.([]*ast.FuncParam)
	}


FuncParamList:
	FuncParam
	{
		$$ = []*ast.FuncParam{$1.(*ast.FuncParam)}
	}
|	FuncParamList ',' FuncParam
	{
		$$ = append($1.([]*ast.FuncParam), $3.(*ast.FuncParam))
	}


FuncParam:
	Identifier Type
	{
		$$ = &ast.FuncParam{Name: $1, Type: $2.(*types.FieldType), Dir: ast.FuncParamDirIn}
	}
|	in Identifier Type
	{
		$$ = &ast.FuncParam{Name: $2, Type: $3.(*types.FieldType), Dir: ast.FuncParamDirIn}
	}
|	out Identifier Type
	{
		$$ = &ast.FuncParam{Name: $2, Type: $3.(*types.FieldType), Dir: ast.FuncParamDirOut}
	}
|	inout Identifier Type
	{
		$$ = &ast.FuncParam{Name: $2, Type: $3.(*types.FieldType), Dir: ast.FuncParamDirInOut}
	}


RoutineCharacteristicList:
	/* EMPTY */
	{
		$$ = &ast.RoutineCharacteristics{}
	}
|	RoutineCharacteristicList RoutineCharacteristic
	{
		acc := $1.(*ast.RoutineCharacteristics)
		c := $2.(*ast.RoutineCharacteristics)
		if c.LanguageSpecified {
			acc.Language = c.Language
			acc.LanguageSpecified = true
		}
		if c.DeterministicSpecified {
			acc.Deterministic = c.Deterministic
			acc.DeterministicSpecified = true
		}
		if c.DataAccessSpecified {
			acc.DataAccess = c.DataAccess
			acc.DataAccessSpecified = true
		}
		if c.SecuritySpecified {
			acc.Security = c.Security
			acc.SecuritySpecified = true
		}
		if c.CommentSpecified {
			acc.Comment = c.Comment
			acc.CommentSpecified = true
		}
		$$ = acc
	}


RoutineCharacteristic:
	language sql
	{
		$$ = &ast.RoutineCharacteristics{Language: "SQL", LanguageSpecified: true}
	}
|	deterministic
	{
		$$ = &ast.RoutineCharacteristics{Deterministic: true, DeterministicSpecified: true}
	}
|	not deterministic
	{
		$$ = &ast.RoutineCharacteristics{Deterministic: false, DeterministicSpecified: true}
	}
|	contains sql
	{
		$$ = &ast.RoutineCharacteristics{DataAccess: ast.RoutineDataAccessContainsSQL, DataAccessSpecified: true}
	}
|	no sql
	{
		$$ = &ast.RoutineCharacteristics{DataAccess: ast.RoutineDataAccessNoSQL, DataAccessSpecified: true}
	}
|	reads sql data
	{
		$$ = &ast.RoutineCharacteristics{DataAccess: ast.RoutineDataAccessReadsSQLData, DataAccessSpecified: true}
	}
|	modifies sql data
	{
		$$ = &ast.RoutineCharacteristics{DataAccess: ast.RoutineDataAccessModifiesSQLData, DataAccessSpecified: true}
	}
|	sql security definer
	{
		$$ = &ast.RoutineCharacteristics{Security: model.SecurityDefiner, SecuritySpecified: true}
	}
|	sql security invoker
	{
		$$ = &ast.RoutineCharacteristics{Security: model.SecurityInvoker, SecuritySpecified: true}
	}
|	comment stringLit
	{
		$$ = &ast.RoutineCharacteristics{Comment: $2, CommentSpecified: true}
	}


TriggerTime:
	before
	{
		$$ = "BEFORE"
	}
|	after
	{
		$$ = "AFTER"
	}


TriggerEvent:
	insert
	{
		$$ = "INSERT"
	}
|	update
	{
		$$ = "UPDATE"
	}
|	deleteKwd
	{
		$$ = "DELETE"
	}


TriggerUpdateColumnsOpt:
	/* EMPTY */
	{
		$$ = ([]*ast.ColumnName)(nil)
	}
|	of ColumnNameList
	{
		$$ = $2.([]*ast.ColumnName)
	}


TriggerOrderOpt:
	/* EMPTY */
	{
		$$ = ([]string)(nil)
	}
|	follows Identifier
	{
		$$ = []string{"FOLLOWS", $2}
	}
|	precedes Identifier
	{
		$$ = []string{"PRECEDES", $2}
	}


EventSchedule:
	on schedule at Expression
	{
		$$ = &ast.EventSchedule{At: $4.(ast.ExprNode)}
	}
|	on schedule every Expression TimeUnit EventStartsOpt EventEndsOpt
	{
		starts, _ := $6.(ast.ExprNode)
		ends, _ := $7.(ast.ExprNode)
		$$ = &ast.EventSchedule{
			Every:    $4.(ast.ExprNode),
			Interval: $5,
			Starts:   starts,
			Ends:     ends,
		}
	}


EventStartsOpt:
	/* EMPTY */
	{
		$$ = (ast.ExprNode)(nil)
	}
|	starts Expression
	{
		$$ = $2.(ast.ExprNode)
	}


EventEndsOpt:
	/* EMPTY */
	{
		$$ = (ast.ExprNode)(nil)
	}
|	ends Expression
	{
		$$ = $2.(ast.ExprNode)
	}


EventStatusOpt:
	/* EMPTY */
	{
		$$ = ast.EventStatusDefault
	}
|	enable
	{
		$$ = ast.EventStatusEnable
	}
|	disable
	{
		$$ = ast.EventStatusDisable
	}
|	disable on slave
	{
		$$ = ast.EventStatusDisableOnSlave
	}


EventOnCompletionOpt:
	/* EMPTY */
	{
		$$ = (*bool)(nil)
	}
|	on completion preserve
	{
		b := true
		$$ = &b
	}
|	on completion not preserve
	{
		b := false
		$$ = &b
	}


EventCommentOpt:
	/* EMPTY */
	{
		$$ = (*string)(nil)
	}
|	comment stringLit
	{
		s := $2
		$$ = &s
	}


ReturnStmt:
	returnKwd Expression
	{
		$$ = &ast.ReturnStmt{Expr: $2.(ast.ExprNode)}
	}


OrReplace:
	/* EMPTY */    { $$ = false }
|	or replace { $$ = true }


ViewAlgorithm:
	/* EMPTY */                 { $$ = (*model.ViewAlgorithm)(nil) }
|	algorithm eq undefined { v := model.AlgorithmUndefined; $$ = &v }
|	algorithm eq merge     { v := model.AlgorithmMerge; $$ = &v }
|	algorithm eq temptable { v := model.AlgorithmTemptable; $$ = &v }


ViewDefiner:
	/* EMPTY */ { $$ = &auth.UserIdentity{CurrentUser: true} }
|	definer eq Username { $$ = $3 }


ViewSQLSecurity:
	/* EMPTY */                { $$ = (*model.ViewSecurity)(nil) }
|	sql security definer { v := model.SecurityDefiner; $$ = &v }
|	sql security invoker { v := model.SecurityInvoker; $$ = &v }


ViewName:
	TableName { $$ = $1.(*ast.TableName) }


ViewFieldList:
	/* Empty */        { $$ = nil }
|	'(' ColumnList ')' { $$ = $2.([]model.CIStr) }


ColumnList:
	Identifier { $$ = []model.CIStr{model.NewCIStr($1)} }
|	ColumnList ',' Identifier { $$ = append($1.([]model.CIStr), model.NewCIStr($3)) }


ViewCheckOption:
	/* EMPTY */ { $$ = nil }
|	with check option { $$ = model.CheckOptionCascaded }
|	with cascaded check option { $$ = model.CheckOptionCascaded }
|	with local check option { $$ = model.CheckOptionLocal }


/******************************************************************
 * Do statement
 * See https://dev.mysql.com/doc/refman/5.7/en/do.html
 ******************************************************************/
DoStmt:
	do ExpressionList
	{
		$$ = &ast.DoStmt {
			Exprs: $2.([]ast.ExprNode),
		}
	}


CallStmt:
	call RoutineName
	{
		$$ = &ast.CallStmt{Name: $2.(*ast.TableName), Args: []ast.ExprNode{}}
	}
|	call RoutineName '(' ExpressionListOpt ')'
	{
		$$ = &ast.CallStmt{Name: $2.(*ast.TableName), Args: $4.([]ast.ExprNode)}
	}


/*******************************************************************
 *
 *  Delete Statement
 *
 *******************************************************************/
DeleteFromStmt:
	deleteKwd TableOptimizerHints PriorityOpt QuickOptional IgnoreOptional from TableName PartitionNameListOpt IndexHintListOpt WhereClauseOptional OrderByOptional LimitClause ReturningOpt
	{
		// Single Table
		tn := $7.(*ast.TableName)
		tn.PartitionNames = $8.([]model.CIStr)
		tn.IndexHints = $9.([]*ast.IndexHint)
		join := &ast.Join{Left: &ast.TableSource{Source: tn}, Right: nil}
		x := &ast.DeleteStmt{
			TableRefs: &ast.TableRefsClause{TableRefs: join},
			Priority:  $3.(mysql.PriorityEnum),
			Quick:	   $4.(bool),
			IgnoreErr: $5.(bool),
		}
		if $10 != nil {
			x.Where = $10.(ast.ExprNode)
		}
		if $11 != nil {
			x.Order = $11.(*ast.OrderByClause)
		}
		if $12 != nil {
			x.Limit = $12.(*ast.Limit)
		}
		if $13 != nil {
			x.Returning = $13.(*ast.FieldList)
		}

		$$ = x
	}
|	deleteKwd TableOptimizerHints PriorityOpt QuickOptional IgnoreOptional TableNameList from TableRefs WhereClauseOptional
	{
		// Multiple Table
		x := &ast.DeleteStmt{
			Priority:	  $3.(mysql.PriorityEnum),
			Quick:		  $4.(bool),
			IgnoreErr:	  $5.(bool),
			IsMultiTable: 	  true,
			BeforeFrom:	  true,
			Tables:		  &ast.DeleteTableList{Tables: $6.([]*ast.TableName)},
			TableRefs:	  &ast.TableRefsClause{TableRefs: $8.(*ast.Join)},
		}
		if $2 != nil {
			x.TableHints = $2.([]*ast.TableOptimizerHint)
		}
		if $9 != nil {
			x.Where = $9.(ast.ExprNode)
		}
		$$ = x
	}

|	deleteKwd TableOptimizerHints PriorityOpt QuickOptional IgnoreOptional from TableNameList using TableRefs WhereClauseOptional OrderByOptional LimitClause
	{
		// Multiple Table
		x := &ast.DeleteStmt{
			Priority:	  $3.(mysql.PriorityEnum),
			Quick:		  $4.(bool),
			IgnoreErr:	  $5.(bool),
			IsMultiTable:	  true,
			Tables:		  &ast.DeleteTableList{Tables: $7.([]*ast.TableName)},
			TableRefs:	  &ast.TableRefsClause{TableRefs: $9.(*ast.Join)},
		}
		if $2 != nil {
			x.TableHints = $2.([]*ast.TableOptimizerHint)
		}
		if $10 != nil {
			x.Where = $10.(ast.ExprNode)
		}
		if $11 != nil {
			x.Order = $11.(*ast.OrderByClause)
		}
		if $12 != nil {
			x.Limit = $12.(*ast.Limit)
		}
		$$ = x
	}


DatabaseSym:
database


DropDatabaseStmt:
	drop DatabaseSym IfExists DBName
	{
		$$ = &ast.DropDatabaseStmt{IfExists: $3.(bool), Name: $4.(string)}
	}


DropIndexStmt:
	drop index IfExists Identifier on TableName
	{
		$$ = &ast.DropIndexStmt{IfExists: $3.(bool), IndexName: $4, Table: $6.(*ast.TableName)}
	}


DropTableStmt:
	drop TemporaryOpt TableOrTables TableNameList RestrictOrCascadeOpt
	{
		$$ = &ast.DropTableStmt{Tables: $4.([]*ast.TableName), Temporary: $2.(bool)}
	}
|	drop TemporaryOpt TableOrTables ifKwd exists TableNameList RestrictOrCascadeOpt
	{
		$$ = &ast.DropTableStmt{IfExists: true, Tables: $6.([]*ast.TableName), Temporary: $2.(bool)}
	}


DropViewStmt:
	drop view TableNameList RestrictOrCascadeOpt
	{
		$$ = &ast.DropTableStmt{Tables: $3.([]*ast.TableName), IsView: true}
	}
|
	drop view ifKwd exists TableNameList RestrictOrCascadeOpt
	{
		$$ = &ast.DropTableStmt{IfExists: true, Tables: $5.([]*ast.TableName), IsView: true}
	}


DropFunctionStmt:
	drop function IfExists TableName
	{
		$$ = &ast.DropFunctionStmt{IfExists: $3.(bool), Name: $4.(*ast.TableName)}
	}


DropProcedureStmt:
	drop procedure IfExists TableName
	{
		$$ = &ast.DropProcedureStmt{IfExists: $3.(bool), Name: $4.(*ast.TableName)}
	}


DropTriggerStmt:
	drop trigger IfExists TableName
	{
		$$ = &ast.DropTriggerStmt{IfExists: $3.(bool), Name: $4.(*ast.TableName)}
	}


DropEventStmt:
	drop event IfExists TableName
	{
		$$ = &ast.DropEventStmt{IfExists: $3.(bool), Name: $4.(*ast.TableName)}
	}


DropUserStmt:
	drop user UsernameList
	{
		$$ = &ast.DropUserStmt{IsDropRole: false, IfExists: false, UserList: $3.([]*auth.UserIdentity)}
	}
|	drop user ifKwd exists UsernameList
	{
		$$ = &ast.DropUserStmt{IsDropRole: false, IfExists: true, UserList: $5.([]*auth.UserIdentity)}
	}


DropRoleStmt:
	drop role RolenameList {
		tmp := make([]*auth.UserIdentity, 0, 10)
		roleList := $3.([]*auth.RoleIdentity)
		for _, r := range roleList {
			tmp = append(tmp, &auth.UserIdentity{Username:r.Username, Hostname: r.Hostname})
		}
		$$ = &ast.DropUserStmt{IsDropRole: true, IfExists: false, UserList: tmp}
	}
|	drop role ifKwd exists RolenameList {
		tmp := make([]*auth.UserIdentity, 0, 10)
		roleList := $5.([]*auth.RoleIdentity)
		for _, r := range roleList {
			tmp = append(tmp, &auth.UserIdentity{Username:r.Username, Hostname: r.Hostname})
		}
		$$ = &ast.DropUserStmt{IsDropRole: true, IfExists: true, UserList: tmp}
	}


RestrictOrCascadeOpt:
	/* EMPTY */ {}
|	restrict
|	cascade


TableOrTables:
	tableKwd
|	tables


EqOpt:
	/* EMPTY */ {}
|	eq


EmptyStmt:
	/* EMPTY */ { $$ = nil }


ExplainSym:
explain | describe | desc


ExplainStmt:
	ExplainSym TableName
	{
		$$ = &ast.ExplainStmt{
			Stmt: &ast.ShowStmt{
				Tp:	ast.ShowColumns,
				Table:	$2.(*ast.TableName),
			},
		}
	}
|	ExplainSym TableName ColumnName
	{
		$$ = &ast.ExplainStmt{
			Stmt: &ast.ShowStmt{
				Tp:	ast.ShowColumns,
				Table:	$2.(*ast.TableName),
				Column:	$3.(*ast.ColumnName),
			},
		}
	}
|	ExplainSym ExplainTarget
	{
		x := $2.(*ast.ExplainStmt)
		x.Format = "row"
		$$ = x
	}
|	ExplainSym extended ExplainableStmt
	{
		$$ = &ast.ExplainStmt{Stmt: $3, Format: "row", Extended: true}
	}
|	ExplainSym partitions ExplainableStmt
	{
		$$ = &ast.ExplainStmt{Stmt: $3, Format: "row", Partitions: true}
	}
|	ExplainSym forKwd connection NUM
	{
		$$ = &ast.ExplainForStmt{
			Format:       "row",
			ConnectionID: getUint64FromNUM($4),
		}
	}
|	ExplainSym format eq ExplainFormat forKwd connection NUM
	{
		$$ = &ast.ExplainForStmt{
			Format:       $4,
			ConnectionID: getUint64FromNUM($7),
		}
	}
|	ExplainSym format eq ExplainFormat ExplainTarget
	{
		x := $5.(*ast.ExplainStmt)
		x.Format = $4
		$$ = x
	}
|	ExplainSym analyze ExplainTargetNoInto
	{
		x := $3.(*ast.ExplainStmt)
		x.Format = "row"
		x.Analyze = true
		$$ = x
	}
|	ExplainSym analyze format eq ExplainFormat ExplainTargetNoInto
	{
		x := $6.(*ast.ExplainStmt)
		x.Format = $5
		x.Analyze = true
		$$ = x
	}
|	analyze ExplainableStmt
	{
		$$ = &ast.ExplainStmt{Stmt: $2, Format: "row", Analyze: true, AnalyzeOnly: true}
	}
|	analyze format eq ExplainFormat ExplainableStmt
	{
		$$ = &ast.ExplainStmt{Stmt: $5, Format: $4, Analyze: true, AnalyzeOnly: true}
	}


/* INTO targets and FOR SCHEMA follow FORMAT and precede the explained statement. */
ExplainTarget:
	ExplainableStmt
	{
		$$ = &ast.ExplainStmt{Stmt: $1}
	}
|	into SpName ExplainableStmt
	{
		$$ = &ast.ExplainStmt{Stmt: $3, Into: explainIntoStmt($2.(*ast.VariableTarget))}
	}
|	forKwd database Identifier ExplainableStmt
	{
		$$ = &ast.ExplainStmt{Stmt: $4, Schema: model.NewCIStr($3)}
	}
|	into SpName forKwd database Identifier ExplainableStmt
	{
		$$ = &ast.ExplainStmt{Stmt: $6, Into: explainIntoStmt($2.(*ast.VariableTarget)), Schema: model.NewCIStr($5)}
	}


/* EXPLAIN ANALYZE rejects INTO, so it keeps a separate target list. */
ExplainTargetNoInto:
	ExplainableStmt
	{
		$$ = &ast.ExplainStmt{Stmt: $1}
	}
|	forKwd database Identifier ExplainableStmt
	{
		$$ = &ast.ExplainStmt{Stmt: $4, Schema: model.NewCIStr($3)}
	}


ExplainFormat:
	stringLit
	{
		$$ = $1
	}
|	jsonType
	{
		$$ = strings.ToLower($1)
	}
|	traditional
	{
		$$ = strings.ToLower($1)
	}
|	tree
	{
		$$ = strings.ToLower($1)
	}


LengthNum:
	NUM { $$ = getUint64FromNUM($1) }


NUM:
	intLit


Expression:
	singleAtIdentifier assignmentEq Expression %prec assignmentEq
	{
		v := $1
		v = strings.TrimPrefix(v, "@")
		$$ = &ast.VariableExpr{
				Name: 	  v,
				IsGlobal: false,
				IsSystem: false,
				Value:	  $3,
		}
	}
|	Expression logOr Expression %prec pipes
	{
		$$ = &ast.BinaryOperationExpr{Op: opcode.LogicOr, L: $1, R: $3}
	}
|	Expression xor Expression %prec xor
	{
		$$ = &ast.BinaryOperationExpr{Op: opcode.LogicXor, L: $1, R: $3}
	}
|	Expression logAnd Expression %prec andand
	{
		$$ = &ast.BinaryOperationExpr{Op: opcode.LogicAnd, L: $1, R: $3}
	}
|	not Expression %prec not
	{
		expr, ok := $2.(*ast.ExistsSubqueryExpr)
		if ok {
			expr.Not = true
			$$ = $2
		} else {
			$$ = &ast.UnaryOperationExpr{Op: opcode.Not, V: $2}
		}
	}
|	BoolPri IsOrNotOp trueKwd %prec is
	{
		$$ = &ast.IsTruthExpr{Expr:$1, Not: !$2.(bool), True: int64(1)}
	}
|	BoolPri IsOrNotOp falseKwd %prec is
	{
		$$ = &ast.IsTruthExpr{Expr:$1, Not: !$2.(bool), True: int64(0)}
	}
|	BoolPri IsOrNotOp unknown %prec is
	{
		/* https://dev.mysql.com/doc/refman/5.7/en/comparison-operators.html#operator_is */
		$$ = &ast.IsNullExpr{Expr: $1, Not: !$2.(bool)}
	}
/* https://mariadb.com/docs/server/reference/sql-structure/operators/comparison-operators/is-json */
|	BoolPri IsOrNotOp jsonType IsJsonTypeOpt IsJsonUniqueKeysOpt %prec is
	{
		$$ = &ast.IsJSONExpr{Expr: $1, Not: !$2.(bool), Type: $4, UniqueKeys: $5}
	}
|	BoolPri %prec empty


/* MariaDB 12.3: https://mariadb.com/docs/server/reference/sql-structure/operators/comparison-operators/is-json */
IsJsonTypeOpt:
	/* EMPTY */ %prec empty { $$ = "" }
|	value                  { $$ = "VALUE" }
|	arrayKwd               { $$ = "ARRAY" }
|	objectKwd              { $$ = "OBJECT" }
|	scalarKwd              { $$ = "SCALAR" }


/* MariaDB 12.3: https://mariadb.com/docs/server/reference/sql-structure/operators/comparison-operators/is-json */
IsJsonUniqueKeysOpt:
	/* EMPTY */ %prec empty { $$ = "" }
|	with unique            { $$ = "WITH UNIQUE KEYS" }
|	with unique keys       { $$ = "WITH UNIQUE KEYS" }
|	without unique         { $$ = "WITHOUT UNIQUE KEYS" }
|	without unique keys    { $$ = "WITHOUT UNIQUE KEYS" }


MaxValueOrExpression:
	/* EMPTY */ maxValue { $$ = &ast.MaxValueExpr{} }
|	Expression             { $$ = $1 }


logOr:
	pipesAsOr
|	or


logAnd:
andand | and


ExpressionList:
	Expression %prec lowerThanComma { $$ = []ast.ExprNode{$1} }
|	ExpressionList ',' Expression { $$ = append($1.([]ast.ExprNode), $3) }


MaxValueOrExpressionList:
	MaxValueOrExpression                              { $$ = []ast.ExprNode{$1} }
|	MaxValueOrExpressionList ',' MaxValueOrExpression { $$ = append($1.([]ast.ExprNode), $3) }


ExpressionListOpt:
	/* EMPTY */ { $$ = []ast.ExprNode{} }
|	ExpressionList


FuncDatetimePrecListOpt:
	/* EMPTY */ { $$ = []ast.ExprNode{} }
|	FuncDatetimePrecList { $$ = $1 }


FuncDatetimePrecList:
	intLit
	{
		expr := ast.NewValueExpr($1)
		$$ = []ast.ExprNode{expr}
	}


BoolPri:
	BoolPri IsOrNotOp null %prec is
	{
		$$ = &ast.IsNullExpr{Expr: $1, Not: !$2.(bool)}
	}
|	BoolPri CompareOp PredicateExpr %prec eq
	{
		$$ = &ast.BinaryOperationExpr{Op: $2.(opcode.Op), L: $1, R: $3}
	}
|	BoolPri CompareOp AnyOrAll SubSelect %prec eq
	{
		sq := $4.(*ast.SubqueryExpr)
		sq.MultiRows = true
		$$ = &ast.CompareSubqueryExpr{Op: $2.(opcode.Op), L: $1, R: sq, All: $3.(bool)}
	}
|	BoolPri CompareOp AnyOrAll '(' ValuesStmt ')' %prec eq
	{
		sq := &ast.SubqueryExpr{Query: $5.(*ast.ValuesStmt), MultiRows: true}
		$$ = &ast.CompareSubqueryExpr{Op: $2.(opcode.Op), L: $1, R: sq, All: $3.(bool)}
	}
|	BoolPri CompareOp singleAtIdentifier assignmentEq PredicateExpr %prec assignmentEq
	{
		v := $3
		v = strings.TrimPrefix(v, "@")
		variable := &ast.VariableExpr{
				Name: 	  v,
				IsGlobal: false,
				IsSystem: false,
				Value:	  $5,
		}
		$$ = &ast.BinaryOperationExpr{Op: $2.(opcode.Op), L: $1, R: variable}
	}
|	PredicateExpr


CompareOp:
	ge  { $$ = opcode.GE }
|	'>'   { $$ = opcode.GT }
|	le  { $$ = opcode.LE }
|	'<'   { $$ = opcode.LT }
|	neq  { $$ = opcode.NE }
|	neqSynonym  { $$ = opcode.NE }
|	eq   { $$ = opcode.EQ }
|	nulleq { $$ = opcode.NullEQ }


BetweenOrNotOp:
	between       { $$ = true }
|	not between { $$ = false }


IsOrNotOp:
	is       { $$ = true }
|	is not { $$ = false }


InOrNotOp:
	in       { $$ = true }
|	not in { $$ = false }


LikeOrNotOp:
	like { $$ = true }
|	not like { $$ = false }


RegexpOrNotOp:
	RegexpSym       { $$ = true }
|	not RegexpSym { $$ = false }


AnyOrAll:
	any  { $$ = false }
|	some { $$ = false }
|	all  { $$ = true }


PredicateExpr:
	BitExpr InOrNotOp '(' ExpressionList ')'
	{
		$$ = &ast.PatternInExpr{Expr: $1, Not: !$2.(bool), List: $4.([]ast.ExprNode)}
	}
|	BitExpr InOrNotOp SubSelect
	{
		sq := $3.(*ast.SubqueryExpr)
		sq.MultiRows = true
		$$ = &ast.PatternInExpr{Expr: $1, Not: !$2.(bool), Sel: sq}
	}
|	BitExpr BetweenOrNotOp BitExpr and PredicateExpr
	{
		$$ = &ast.BetweenExpr{
			Expr:	$1,
			Left:	$3,
			Right:	$5,
			Not:	!$2.(bool),
		}
	}
|	BitExpr LikeOrNotOp Expression LikeEscapeOpt %prec like
	{
		escapeExpr, _ := $4.(ast.ExprNode)
		escape := byte('\\')
		if value, ok := escapeExpr.(ast.ValueExpr); ok {
			text := value.GetString()
			if _, isString := value.GetValue().(string); isString && len(text) <= 1 {
				escapeExpr = nil
			}
			if _, isString := value.GetValue().(string); isString && len(text) == 1 {
				escape = text[0]
				escapeExpr = nil
			}
		}
		$$ = &ast.PatternLikeExpr{
			Expr:		$1,
			Pattern:	$3,
			Not: 		!$2.(bool),
			Escape: 	escape,
			EscapeExpr: escapeExpr,
		}
	}
|	BitExpr RegexpOrNotOp Expression %prec regexpKwd
	{
		$$ = &ast.PatternRegexpExpr{Expr: $1, Pattern: $3, Not: !$2.(bool)}
	}
/* https://dev.mysql.com/doc/refman/8.0/en/json-search-functions.html#operator_member-of */
|	BitExpr memberKwd of '(' SimpleExpr ')' %prec memberKwd
	{
		$$ = &ast.MemberOfExpr{Value: $1, Array: $5}
	}
/* https://dev.mysql.com/doc/refman/8.0/en/string-functions.html#operator_sounds-like */
|	BitExpr sounds like BitExpr %prec sounds
	{
		$$ = &ast.SoundsLikeExpr{Expr: $1, Pattern: $4}
	}
|	BitExpr %prec empty


RegexpSym:
regexpKwd | rlike


LikeEscapeOpt:
	%prec empty
	{
		$$ = (ast.ExprNode)(nil)
	}
|	escape SimpleExpr
	{
		$$ = $2
	}


/* https://dev.mysql.com/doc/refman/8.0/en/fulltext-search.html */
MatchAgainstOpt:
	/* EMPTY */                         { $$ = 0 }
|	in natural language mode          { $$ = 0 }
|	in booleanType mode               { $$ = 1 }
|	with query expansion              { $$ = 2 }
|	in natural language mode with query expansion { $$ = 2 }


Field:
	'*'
	{
		$$ = &ast.SelectField{WildCard: &ast.WildCardField{}}
	}
|	Identifier '.' '*'
	{
		wildCard := &ast.WildCardField{Table: model.NewCIStr($1)}
		$$ = &ast.SelectField{WildCard: wildCard}
	}
|	Identifier '.' Identifier '.' '*'
	{
		wildCard := &ast.WildCardField{Schema: model.NewCIStr($1), Table: model.NewCIStr($3)}
		$$ = &ast.SelectField{WildCard: wildCard}
	}
|	Expression FieldAsNameOpt
	{
		expr := $1
		asName := $2.(string)
		$$ = &ast.SelectField{Expr: expr, AsName: model.NewCIStr(asName)}
	}


FieldAsNameOpt:
	/* EMPTY */ { $$ = "" }
|	FieldAsName { $$ = $1 }


FieldAsName:
	Identifier      { $$ = $1 }
|	as Identifier { $$ = $2 }
|	stringLit       { $$ = $1 }
|	as stringLit  { $$ = $2 }
|	as WindowFuncAliasName { $$ = $2 }


/* Window function keywords are non-reserved in MariaDB and may name aliases. */
WindowFuncAliasName:
	lead         { $$ = "lead" }
|	rank         { $$ = "rank" }
|	lag          { $$ = "lag" }
|	ntile        { $$ = "ntile" }
|	firstValue   { $$ = "first_value" }
|	lastValue    { $$ = "last_value" }
|	rowNumber    { $$ = "row_number" }
|	denseRank    { $$ = "dense_rank" }
|	cumeDist     { $$ = "cume_dist" }
|	percentRank  { $$ = "percent_rank" }
|	nthValue     { $$ = "nth_value" }


FieldList:
	Field
	{
		field := $1.(*ast.SelectField)
		field.Offset = yyGetParser(yylex).startOffset(&yyS[yypt])
		$$ = []*ast.SelectField{field}
	}
|	FieldList ',' Field
	{

		fl := $1.([]*ast.SelectField)
		last := fl[len(fl)-1]
		if last.Expr != nil && last.AsName.O == "" {
			lastEnd := yyGetParser(yylex).endOffset(&yyS[yypt-1])
			last.SetText(yyGetParser(yylex).src[last.Offset:lastEnd])
		}
		newField := $3.(*ast.SelectField)
		newField.Offset = yyGetParser(yylex).startOffset(&yyS[yypt])
		$$ = append(fl, newField)
	}


GroupByClause:
	group by ByList GroupByWithOpt
	{
		gc := &ast.GroupByClause{Items: $3.([]*ast.ByItem)}
		switch $4.(int) {
		case 1:
			gc.WithRollup = true
		case 2:
			gc.WithCube = true
		case 3:
			gc.CheckOption = model.CheckOptionCascaded
			gc.HasCheckOption = true
		case 4:
			gc.CheckOption = model.CheckOptionLocal
			gc.HasCheckOption = true
		}
		$$ = gc
	}


GroupByWithOpt:
	/* EMPTY */ %prec empty { $$ = 0 }
|	with rollup { $$ = 1 }
|	with cube { $$ = 2 }
|	with cascaded check option { $$ = 3 }
|	with local check option { $$ = 4 }


HavingClause:
	/* EMPTY */ { $$ = nil }
|	having Expression { $$ = &ast.HavingClause{Expr: $2} }


IfExists:
	/* EMPTY */   { $$ = false }
|	ifKwd exists { $$ = true }


IfNotExists:
	/* EMPTY */         { $$ = false }
|	ifKwd not exists { $$ = true }


IgnoreOptional:
	/* EMPTY */ { $$ = false }
|	ignore    { $$ = true }


IndexName:
	/* EMPTY */ { $$ = "" }
|	Identifier  { $$ = $1 } //"index name"


IndexOptionList:
	/* EMPTY */ { $$ = nil }
|	IndexOptionList IndexOption
	{
		// Merge the options
		if $1 == nil {
			$$ = $2
		} else {
			opt1 := $1.(*ast.IndexOption)
			opt2 := $2.(*ast.IndexOption)
			if len(opt2.Comment) > 0 {
				opt1.Comment = opt2.Comment
			} else if opt2.Tp != 0 {
				opt1.Tp = opt2.Tp
			} else if opt2.KeyBlockSize > 0 {
			    opt1.KeyBlockSize = opt2.KeyBlockSize
			}
			$$ = opt1
		}
	}


IndexOption:
	keyBlockSize EqOpt LengthNum
	{
		$$ = &ast.IndexOption{
			KeyBlockSize: $3.(uint64),
		}
	}
|	IndexType
	{
		$$ = &ast.IndexOption {
			Tp: $1.(model.IndexType),
		}
	}
|	typeSym btree { $$ = &ast.IndexOption{Tp: model.IndexTypeBtree} }
|	typeSym hash  { $$ = &ast.IndexOption{Tp: model.IndexTypeHash} }
|	typeSym rtree { $$ = &ast.IndexOption{Tp: model.IndexTypeRtree} }
|	comment stringLit
	{
		$$ = &ast.IndexOption {
			Comment: $2,
		}
	}


IndexType:
	using btree { $$ = model.IndexTypeBtree }
|	using hash  { $$ = model.IndexTypeHash }
|	using rtree { $$ = model.IndexTypeRtree }


IndexTypeOpt:
	/* EMPTY */ { $$ = nil }
|	IndexType   { $$ = $1 }


CreateIndexTypeOpt:
	/* EMPTY */ { $$ = nil }
|	IndexType   { $$ = $1 }
|	typeSym btree { $$ = model.IndexTypeBtree }
|	typeSym hash  { $$ = model.IndexTypeHash }
|	typeSym rtree { $$ = model.IndexTypeRtree }


/**********************************Identifier********************************************/
Identifier:
identifier | UnReservedKeyword | NotKeywordToken


UnReservedKeyword:
  action | ascii | autoIncrement | after | always | avg | backup | begin | bitType | boolType | booleanType | btree | rtree | byteType | cleanup | charsetKwd %prec charsetKwd
| columns | commit | compact | compressed | consistent | current | data | dateType %prec lowerThanStringLitToken| datetimeType | day | deallocate | do | duplicate
| dynamic| end | engine | engines | enum | identSQLErrors | escape | execute | fields | first | fixed | flush | following | format | full |global
| hash | hour | less | local | last | names | offset | password %prec lowerThanEq | prepare | quick | redundant
| role |rollback | session | signed | snapshot | start | status | open | subpartitions | subpartition | tables | tablespace | textType | than | timeType %prec lowerThanStringLitToken
| timestampType %prec lowerThanStringLitToken | transaction | truncate | unbounded | unknown | value | warnings | yearType | mode  | week  | any | some | user | identified
| uuidType | inet4Type | inet6Type
| collation | comment | avgRowLength | connection | checksum | chain | compression | keyBlockSize | master | maxRows
| minRows | national | rowFormat | quarter | grants | triggers | delayKeyWrite | isolation | jsonType | serialType
| repeatable | respect | committed | uncommitted | only | serializable | level | variables | sqlCache | indexes | processlist
| sqlNoCache | disable  | enable | reverse | privileges | no | binlog | function | view | modify | events | partitions
| none | nulls | super | exclusive | statsPersistent | rowCount | coalesce | month | process | profile | profiles
| microsecond | minute | plugins | preceding | query | queries | second | separator | share | shared | slow | maxConnectionsPerHour | maxQueriesPerHour | maxUpdatesPerHour
| maxUserConnections | replication | client | slave | reload | temporary | routine | event | algorithm | definer | invoker | merge | temptable | undefined | security | cascaded
| recover | cipher | subject | issuer | x509 | never | expire | account | cpu | memory | block | io | context | switchesSym | pageSym | faultsSym | ipc | swaps | source
| sqlBufferResult | before | logs | log | purge | buckets
| at | completion | contains | deterministic | each | ends | every | follows | language | modifies | of | precedes | preserve | reads | returns | returnKwd | schedule | starts
| atomic | close | condition | continueKwd | cursor | declare | elseif | exit | fetch | found | handler | iterate | leave | loop
| resignal | signal | sqlException | sqlState | sqlWarning | undo | until | while
| nowait | skip | locked | rollup | work | release | savepoint | sequence | increment | minvalue | cache | cycle | nomaxvalue | nominvalue | nocycle | nocache | aggregate | soname | stringType | xa | resume | migrate | suspend | one | phase | cube
| server | datafile | wrapper | options | fileBlockSize | extentSize | initialSize | autoextendSize | maxSize | nodegroup | wait | resource | logfile | vcpu | undoBufferSize | redoBufferSize | threadPriority | undofile | typeSym | systemSym | reorganize | exchange | discard | import_ | validation | without | remove | partitioning | outfile | dumpfile | repair | optimize | upgrade | visible | invisible | traditional | tree
| pathKwd | nested | ordinality | emptyKwd | extended | immediate | persistent | statement | within
| persist | persistOnly
| channel | directory | ioThread | leaves | memberKwd | next | plugin | prev | replicas | slaves | sqlThread | xml
| arrayKwd | objectKwd | scalarKwd | concurrent
| exclude | ties | others
| masterHost | sourceHost | masterBind | sourceBind | masterUser | sourceUser | masterPassword | sourcePassword | masterPort | sourcePort
| masterConnectRetry | sourceConnectRetry | masterRetryCount | sourceRetryCount | masterDelay | sourceDelay | masterSSL | sourceSSL
| masterSSLCA | sourceSSLCA | masterSSLCAPath | sourceSSLCAPath | masterSSLCert | sourceSSLCert | masterSSLCipher | sourceSSLCipher | masterSSLKey | sourceSSLKey
| masterSSLVerifyServerCert | sourceSSLVerifyServerCert | masterSSLCRL | sourceSSLCRL | masterSSLCRLPath | sourceSSLCRLPath
| masterTLSVersion | sourceTLSVersion | masterTLSCipherSuites | sourceTLSCipherSuites | masterPublicKeyPath | sourcePublicKeyPath | getMasterPublicKey | getSourcePublicKey
| masterHeartbeatPeriod | sourceHeartbeatPeriod | masterCompressionAlgorithms | sourceCompressionAlgorithms | masterZstdCompressionLevel | sourceZstdCompressionLevel
| masterAutoPosition | sourceAutoPosition | sourceConnectionAutoFailover | requireRowFormat | requireTablePrimaryKeyCheck | assignGTIDsToAnonymousTransactions
| privilegeChecksUser | gtidOnly | networkNamespace | ignoreServerIDs | doDomainIDs | ignoreDomainIDs | masterUseGTID | masterDemoteToSlave | masterDemoteToReplica
| masterLogFile | sourceLogFile | masterLogPos | sourceLogPos | relayLogFile | relayLogPos | masterGTIDPos | sqlBeforeGTIDs | sqlAfterGTIDs | sqlAfterMTSGaps
| defaultAuth | pluginDir | currentPos | slavePos | replicaPos | streamKwd | generateKwd | off
| analyse | geometryType | pointType | lineStringType | polygonType | multiPointType | multiLineStringType | multiPolygonType | geometryCollectionType | srid | zone | period


NotKeywordToken:
  addDate | bitAnd | bitOr | bitXor | cast | copyKwd | count | curTime | dateAdd | dateSub | extract | getFormat | groupConcat
| inplace | instant | internal |min | max | maxExecutionTime | now | recent | position | subDate | substring | sum
| std | stddev | stddevPop | stddevSamp | variance | varPop | varSamp
| timestampAdd | timestampDiff | tokudbDefault | tokudbFast | tokudbLzma | tokudbQuickLZ | tokudbSnappy | tokudbSmall | tokudbUncompressed | tokudbZlib  | top | trim | next_row_id | weightString

/************************************************************************************
 * https://dev.mysql.com/doc/refman/5.7/en/insert.html
 * INSERT [LOW_PRIORITY | DELAYED | HIGH_PRIORITY] [IGNORE]
 *     [INTO] tbl_name
 *     [PARTITION (partition_name [, partition_name] ...)]
 *     [(col_name [, col_name] ...)]
 *     {VALUES | VALUE} (value_list) [, (value_list)] ...
 *     [ON DUPLICATE KEY UPDATE assignment_list]
 *
 * INSERT [LOW_PRIORITY | DELAYED | HIGH_PRIORITY] [IGNORE]
 *     [INTO] tbl_name
 *     [PARTITION (partition_name [, partition_name] ...)]
 *     SET assignment_list
 *     [ON DUPLICATE KEY UPDATE assignment_list]
 *
 * INSERT [LOW_PRIORITY | HIGH_PRIORITY] [IGNORE]
 *     [INTO] tbl_name
 *     [PARTITION (partition_name [, partition_name] ...)]
 *     [(col_name [, col_name] ...)]
 *     SELECT ...
 *     [ON DUPLICATE KEY UPDATE assignment_list]
 *
 * value:
 *     {expr | DEFAULT}
 *
 * value_list:
 *     value [, value] ...
 *
 * assignment:
 *     col_name = value
 *
 * assignment_list:
 *     assignment [, assignment] ...
 *
 *  Insert statement with optional PARTITION clause.
 **********************************************************************************/
InsertIntoStmt:
	insert TableOptimizerHints PriorityOpt IgnoreOptional IntoOpt TableName PartitionNameListOpt InsertValues OnDuplicateKeyUpdate ReturningOpt
	{
		x := $8.(*ast.InsertStmt)
		x.Priority = $3.(mysql.PriorityEnum)
		x.IgnoreErr = $4.(bool)
		// Wraps many layers here so that it can be processed the same way as select statement.
		tn := $6.(*ast.TableName)
		tn.PartitionNames = $7.([]model.CIStr)
		ts := &ast.TableSource{Source: tn}
		x.Table = &ast.TableRefsClause{TableRefs: &ast.Join{Left: ts}}
		if $9 != nil {
			x.OnDuplicate = $9.([]*ast.Assignment)
		}
		if $10 != nil {
			x.Returning = $10.(*ast.FieldList)
		}
		$$ = x
	}


InsertRowAliasOpt:
	/* EMPTY */
	{
		$$ = (*ast.InsertRowAlias)(nil)
	}
|	as Identifier InsertRowAliasColumnsOpt
	{
		alias := &ast.InsertRowAlias{
			Name:   model.NewCIStr($2),
			Offset: yyGetParser(yylex).startOffset(&yyS[yypt-2]),
		}
		if $3 != nil {
			alias.Columns = $3.([]model.CIStr)
		}
		$$ = alias
	}


InsertRowAliasColumnsOpt:
	/* EMPTY */
	{
		$$ = ([]model.CIStr)(nil)
	}
|	'(' ColumnNameList ')'
	{
		columns := $2.([]*ast.ColumnName)
		names := make([]model.CIStr, 0, len(columns))
		for _, column := range columns {
			names = append(names, column.Name)
		}
		$$ = names
	}


IntoOpt:
	/* EMPTY */ {}
|	into


ReturningOpt:
	/* EMPTY */
	{
		$$ = (*ast.FieldList)(nil)
	}
|	returning FieldList
	{
		$$ = &ast.FieldList{Fields: $2.([]*ast.SelectField)}
	}


InsertValues:
	'(' ColumnNameListOpt ')' ValueSym ValuesList InsertRowAliasOpt
	{
		$$ = &ast.InsertStmt{
			Columns:   $2.([]*ast.ColumnName),
			Lists:      $5.([][]ast.ExprNode),
			RowAlias: $6.(*ast.InsertRowAlias),
		}
	}
|	'(' ColumnNameListOpt ')' SelectStmt
	{
		$$ = &ast.InsertStmt{Columns: $2.([]*ast.ColumnName), Select: $4.(*ast.SelectStmt)}
	}
|	'(' ColumnNameListOpt ')' '(' SelectStmt ')'
	{
		$$ = &ast.InsertStmt{Columns: $2.([]*ast.ColumnName), Select: $5.(*ast.SelectStmt)}
	}
|	'(' ColumnNameListOpt ')' SetOprStmt
	{
		$$ = &ast.InsertStmt{Columns: $2.([]*ast.ColumnName), Select: $4.(*ast.SetOprStmt)}
	}
|	'(' ColumnNameListOpt ')' WithSelectStmt
	{
		$$ = &ast.InsertStmt{Columns: $2.([]*ast.ColumnName), Select: $4.(ast.ResultSetNode)}
	}
|	'(' ColumnNameListOpt ')' '(' WithSelectStmt ')'
	{
		$$ = &ast.InsertStmt{Columns: $2.([]*ast.ColumnName), Select: $5.(ast.ResultSetNode)}
	}
|	ValueSym ValuesList InsertRowAliasOpt %prec insertValues
	{
		$$ = &ast.InsertStmt{Lists:  $2.([][]ast.ExprNode), RowAlias: $3.(*ast.InsertRowAlias)}
	}
|	'(' SelectStmt ')'
	{
		$$ = &ast.InsertStmt{Select: $2.(*ast.SelectStmt)}
	}
|	SelectStmt
	{
		$$ = &ast.InsertStmt{Select: $1.(*ast.SelectStmt)}
	}
|	SetOprStmt
	{
		$$ = &ast.InsertStmt{Select: $1.(*ast.SetOprStmt)}
	}
|	WithSelectStmt
	{
		$$ = &ast.InsertStmt{Select: $1.(ast.ResultSetNode)}
	}
|	'(' WithSelectStmt ')'
	{
		$$ = &ast.InsertStmt{Select: $2.(ast.ResultSetNode)}
	}
|	set ColumnSetValueList InsertRowAliasOpt
	{
		$$ = &ast.InsertStmt{Setlist: $2.([]*ast.Assignment), RowAlias: $3.(*ast.InsertRowAlias)}
	}


ValueSym:
value | values


ValuesList:
	RowValue
	{
		$$ = [][]ast.ExprNode{$1.([]ast.ExprNode)}
	}
|	ValuesList ',' RowValue
	{
		$$ = append($1.([][]ast.ExprNode), $3.([]ast.ExprNode))
	}


RowValue:
	'(' ValuesOpt ')'
	{
		$$ = $2
	}


ValuesOpt:
	// MySQL and MariaDB accept an empty VALUES() row, this parser rejects it
	// as non-human usage (see testdata/mtr/report/manual-review.md).
	Values


Values:
	Values ',' ExprOrDefault
	{
		$$ = append($1.([]ast.ExprNode), $3)
	}
|	ExprOrDefault
	{
		$$ = []ast.ExprNode{$1}
	}


ExprOrDefault:
	Expression
|	defaultKwd
	{
		$$ = &ast.DefaultExpr{}
	}


ColumnSetValue:
	ColumnName eq ExprOrDefault
	{
		$$ = &ast.Assignment{
			Column:	$1.(*ast.ColumnName),
			Expr:	$3,
		}
	}


ColumnSetValueList:
	/* EMPTY */ { $$ = []*ast.Assignment{} }
|	ColumnSetValue
	{
		$$ = []*ast.Assignment{$1.(*ast.Assignment)}
	}
|	ColumnSetValueList ',' ColumnSetValue
	{
		$$ = append($1.([]*ast.Assignment), $3.(*ast.Assignment))
	}

/*
 * ON DUPLICATE KEY UPDATE col_name=expr [, col_name=expr] ...
 * See https://dev.mysql.com/doc/refman/5.7/en/insert-on-duplicate.html
 */
OnDuplicateKeyUpdate:
	/* EMPTY */                                    { $$ = nil }
|	on duplicate key update AssignmentList { $$ = $5 }

/***********************************Insert Statements END************************************/

/************************************************************************************
 *  Replace Statements
 *  See https://dev.mysql.com/doc/refman/5.7/en/replace.html
 *
 *  TODO: support PARTITION
 **********************************************************************************/
ReplaceIntoStmt:
	replace TableOptimizerHints PriorityOpt IntoOpt TableName InsertValues
	{
		x := $6.(*ast.InsertStmt)
		if x.RowAlias != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "REPLACE does not accept a row alias"))
			return 1
		}
		x.IsReplace = true
		x.Priority = $3.(mysql.PriorityEnum)
		ts := &ast.TableSource{Source: $5.(*ast.TableName)}
		x.Table = &ast.TableRefsClause{TableRefs: &ast.Join{Left: ts}}
		$$ = x
	}

/***********************************Replace Statements END************************************/


Literal:
	falseKwd
	{
		$$ = ast.NewValueExpr(false)
	}
|	null
	{
		$$ = ast.NewValueExpr(nil)
	}
|	trueKwd
	{
		$$ = ast.NewValueExpr(true)
	}
|	floatLit
	{
		$$ = ast.NewValueExpr($1)
	}
|	decLit
	{
		$$ = ast.NewValueExpr($1)
	}
|	intLit
	{
		$$ = ast.NewValueExpr($1)
	}
|	StringLiteral %prec lowerThanStringLitToken
	{
		$$ = $1
	}
|	underscoreCS StringLiteral %prec lowerThanStringLitToken
	{
		// See https://dev.mysql.com/doc/refman/5.7/en/charset-literal.html
		co, err := charset.GetDefaultCollation($1)
		if err != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "Get collation error for charset: %s", $1))
			return 1
		}
		expr := $2.(ast.ValueExpr)
		tp := expr.GetType()
		tp.Charset = $1
		tp.Collate = co
		if tp.Collate == charset.CollationBin {
			tp.Flag |= mysql.BinaryFlag
		}
		$$ = expr
	}
|	underscoreCS hexLit
	{
		expr := ast.NewValueExpr($2)
		co, err := charset.GetDefaultCollation($1)
		if err != nil {
			yyAppendErr(yylex, err)
			return 1
		}
		tp := expr.GetType()
		tp.Charset, tp.Collate = $1, co
		tp.Flag |= mysql.HasCharsetIntroducerFlag
		tp.Flag &^= mysql.BinaryFlag
		if co == charset.CollationBin {
			tp.Flag |= mysql.BinaryFlag
		}
		$$ = expr
	}
|	underscoreCS bitLit
	{
		expr := ast.NewValueExpr($2)
		co, err := charset.GetDefaultCollation($1)
		if err != nil {
			yyAppendErr(yylex, err)
			return 1
		}
		tp := expr.GetType()
		tp.Charset, tp.Collate = $1, co
		tp.Flag |= mysql.HasCharsetIntroducerFlag
		tp.Flag &^= mysql.BinaryFlag
		if co == charset.CollationBin {
			tp.Flag |= mysql.BinaryFlag
		}
		$$ = expr
	}
|	hexLit
	{
		$$ = ast.NewValueExpr($1)
	}
|	bitLit
	{
		$$ = ast.NewValueExpr($1)
	}


StringLiteral:
	stringLit
	{
		expr := ast.NewValueExpr($1)
		tp := expr.GetType()
		tp.Charset = yyGetParser(yylex).charset
		tp.Collate = yyGetParser(yylex).collation
		$$ = expr
	}
|	StringLiteral stringLit
	{
		valExpr := $1.(ast.ValueExpr)
		strLit := valExpr.GetString()
		expr := ast.NewValueExpr(strLit+$2)
		tp := expr.GetType()
		tp.Charset = yyGetParser(yylex).charset
		tp.Collate = yyGetParser(yylex).collation
		// Fix #4239, use first string literal as projection name.
		if valExpr.GetProjectionOffset() >= 0 {
			expr.SetProjectionOffset(valExpr.GetProjectionOffset())
		} else {
			expr.SetProjectionOffset(len(strLit))
		}
		$$ = expr
	}


OrderBy:
	order by ByList
	{
		$$ = &ast.OrderByClause{Items: $3.([]*ast.ByItem)}
	}


WithinGroupOrderBy:
	order by ByItem
	{
		$$ = &ast.OrderByClause{Items: []*ast.ByItem{$3.(*ast.ByItem)}}
	}


ByList:
	ByItem
	{
		$$ = []*ast.ByItem{$1.(*ast.ByItem)}
	}
|	ByList ',' ByItem
	{
		$$ = append($1.([]*ast.ByItem), $3.(*ast.ByItem))
	}


ByItem:
	Expression Order
	{
		expr := $1
		valueExpr, ok := expr.(ast.ValueExpr)
		if ok {
			position, isPosition := valueExpr.GetValue().(int64)
			if isPosition {
				expr = &ast.PositionExpr{N: int(position)}
			}
		}
		$$ = &ast.ByItem{Expr: expr, Desc: $2.(bool)}
	}


Order:
	/* EMPTY */ { $$ = false } // ASC by default
|	asc       { $$ = false }
|	desc      { $$ = true }


OrderByOptional:
	/* EMPTY */ { $$ = nil }
|	OrderBy     { $$ = $1 }


BitExpr:
	BitExpr '|' BitExpr %prec '|'
	{
		$$ = &ast.BinaryOperationExpr{Op: opcode.Or, L: $1, R: $3}
	}
|	BitExpr '&' BitExpr %prec '&'
	{
		$$ = &ast.BinaryOperationExpr{Op: opcode.And, L: $1, R: $3}
	}
|	BitExpr lsh BitExpr %prec lsh
	{
		$$ = &ast.BinaryOperationExpr{Op: opcode.LeftShift, L: $1, R: $3}
	}
|	BitExpr rsh BitExpr %prec rsh
	{
		$$ = &ast.BinaryOperationExpr{Op: opcode.RightShift, L: $1, R: $3}
	}
|	BitExpr '+' BitExpr %prec '+'
	{
		$$ = &ast.BinaryOperationExpr{Op: opcode.Plus, L: $1, R: $3}
	}
|	BitExpr '-' BitExpr %prec '-'
	{
		$$ = &ast.BinaryOperationExpr{Op: opcode.Minus, L: $1, R: $3}
	}
|	BitExpr '+' interval Expression TimeUnit %prec '+'
	{
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr("DATE_ADD"),
			Args: []ast.ExprNode{
				$1,
				$4,
				ast.NewValueExpr($5),
			},
		}
	}
|	BitExpr '-' interval Expression TimeUnit %prec '+'
	{
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr("DATE_SUB"),
			Args: []ast.ExprNode{
				$1,
				$4,
				ast.NewValueExpr($5),
			},
		}
	}
|	BitExpr '*' BitExpr %prec '*'
	{
		$$ = &ast.BinaryOperationExpr{Op: opcode.Mul, L: $1, R: $3}
	}
|	BitExpr '/' BitExpr %prec '/'
	{
		$$ = &ast.BinaryOperationExpr{Op: opcode.Div, L: $1, R: $3}
	}
|	BitExpr '%' BitExpr %prec '%'
	{
		$$ = &ast.BinaryOperationExpr{Op: opcode.Mod, L: $1, R: $3}
	}
|	BitExpr div BitExpr %prec div
	{
		$$ = &ast.BinaryOperationExpr{Op: opcode.IntDiv, L: $1, R: $3}
	}
|	BitExpr mod BitExpr %prec mod
	{
		$$ = &ast.BinaryOperationExpr{Op: opcode.Mod, L: $1, R: $3}
	}
|	BitExpr '^' BitExpr
	{
		$$ = &ast.BinaryOperationExpr{Op: opcode.Xor, L: $1, R: $3}
	}
|	SimpleExpr %prec empty


SimpleIdent:
	Identifier
	{
		$$ = &ast.ColumnNameExpr{Name: &ast.ColumnName{
			Name: model.NewCIStr($1),
		}}
	}
|	window
	{
		$$ = &ast.ColumnNameExpr{Name: &ast.ColumnName{Name: model.NewCIStr($1)}}
	}
|	rank %prec empty
	{
		$$ = &ast.ColumnNameExpr{Name: &ast.ColumnName{Name: model.NewCIStr($1)}}
	}
|	Identifier '.' Identifier
	{
		$$ = &ast.ColumnNameExpr{Name: &ast.ColumnName{
			Table: model.NewCIStr($1),
			Name: model.NewCIStr($3),
		}}
	}
|	'.' Identifier '.' Identifier
	{
		$$ = &ast.ColumnNameExpr{Name: &ast.ColumnName{
			Table: model.NewCIStr($2),
			Name: model.NewCIStr($4),
		}}
	}
|	Identifier '.' Identifier '.' Identifier
	{
		$$ = &ast.ColumnNameExpr{Name: &ast.ColumnName{
			Schema: model.NewCIStr($1),
			Table: model.NewCIStr($3),
			Name: model.NewCIStr($5),
		}}
	}


SimpleExpr:
	SimpleIdent
|	FunctionCallKeyword
|	FunctionCallNonKeyword
|	FunctionCallGeneric
|	interval Expression TimeUnit '+' Expression %prec interval
	{
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr("DATE_ADD"),
			Args: []ast.ExprNode{$5, $2, ast.NewValueExpr($3)},
			IntervalFirst: true,
		}
	}
|	cube '(' ExpressionListOpt ')'
	{
		$$ = &ast.FuncCallExpr{FnName: model.NewCIStr("CUBE"), Args: $3.([]ast.ExprNode)}
	}
|	rollup '(' ExpressionListOpt ')'
	{
		$$ = &ast.FuncCallExpr{FnName: model.NewCIStr("ROLLUP"), Args: $3.([]ast.ExprNode)}
	}
|	SimpleExpr collate StringName %prec neg
	{
		$$ = &ast.CollateExpr{Expr: $1, Collation: $3.(string)}
	}
|	WindowFuncCall
	{
		$$ = $1.(ast.ExprNode)
	}
/* https://dev.mysql.com/doc/refman/8.0/en/fulltext-search.html */
|	match '(' ColumnNameList ')' against '(' BitExpr MatchAgainstOpt ')'
	{
		m := &ast.MatchAgainst{Columns: $3.([]*ast.ColumnName), Expr: $7}
		switch $8.(int) {
		case 1:
			m.Modifier = ast.MatchAgainstBoolean
		case 2:
			m.QueryExpansion = true
		}
		$$ = m
	}
|	Literal
|	paramMarker
	{
		$$ = ast.NewParamMarkerExpr(yyS[yypt].offset)
	}
|	Variable
|	SumExpr
|	'!' SimpleExpr %prec neg
	{
		$$ = &ast.UnaryOperationExpr{Op: opcode.Not, V: $2}
	}
|	'~'  SimpleExpr %prec neg
	{
		$$ = &ast.UnaryOperationExpr{Op: opcode.BitNeg, V: $2}
	}
|	'-' SimpleExpr %prec neg
	{
		$$ = &ast.UnaryOperationExpr{Op: opcode.Minus, V: $2}
	}
|	'+' SimpleExpr %prec neg
	{
		$$ = &ast.UnaryOperationExpr{Op: opcode.Plus, V: $2}
	}
|	SimpleExpr pipes SimpleExpr
	{
		$$ = &ast.FuncCallExpr{FnName: model.NewCIStr(ast.Concat), Args: []ast.ExprNode{$1, $3}}
	}
|	not2 SimpleExpr %prec neg
	{
		$$ = &ast.UnaryOperationExpr{Op: opcode.Not, V: $2}
	}
|	SubSelect %prec empty
/* Complete nested query parentheses before treating the subquery as a scalar. */
|	'(' Expression ')' {
		startOffset := yyGetParser(yylex).startOffset(&yyS[yypt-1])
		endOffset := yyGetParser(yylex).endOffset(&yyS[yypt])
		expr := $2
		expr.SetText(yyGetParser(yylex).src[startOffset:endOffset])
		$$ = &ast.ParenthesesExpr{Expr: expr}
	}
|	'(' ExpressionList ',' Expression ')'
	{
		values := append($2.([]ast.ExprNode), $4)
		$$ = &ast.RowExpr{Values: values}
	}
|	row '(' ExpressionList ',' Expression ')'
	{
		values := append($3.([]ast.ExprNode), $5)
		$$ = &ast.RowExpr{Values: values}
	}
|	exists SubSelect
	{
		sq := $2.(*ast.SubqueryExpr)
		sq.Exists = true
		$$ = &ast.ExistsSubqueryExpr{Sel: sq}
	}
|	binaryType SimpleExpr %prec neg
	{
		// See https://dev.mysql.com/doc/refman/5.7/en/cast-functions.html#operator_binary
		x := types.NewFieldType(mysql.TypeString)
		x.Charset = charset.CharsetBin
		x.Collate = charset.CharsetBin
		$$ = &ast.FuncCastExpr{
			Expr: $2,
			Tp: x,
			FunctionType: ast.CastBinaryOperator,
		}
	}
|	builtinCast '(' Expression as CastType ')'
	{
		/* See https://dev.mysql.com/doc/refman/5.7/en/cast-functions.html#function_cast */
		tp := $5.(*types.FieldType)
		defaultFlen, defaultDecimal := mysql.GetDefaultFieldLengthAndDecimalForCast(tp.Tp)
		if tp.Flen == types.UnspecifiedLength {
			tp.Flen = defaultFlen
		}
		if tp.Decimal == types.UnspecifiedLength {
			tp.Decimal = defaultDecimal
		}
		$$ = &ast.FuncCastExpr{
			Expr: $3,
			Tp: tp,
			FunctionType: ast.CastFunction,
		}
	}
|	builtinCast '(' Expression CastTimeZone as CastDatetimeType ')'
	{
		/* AT TIME ZONE / AT LOCAL may only target DATETIME[(fsp)]. */
		tp := $6.(*types.FieldType)
		defaultFlen, defaultDecimal := mysql.GetDefaultFieldLengthAndDecimalForCast(tp.Tp)
		if tp.Flen == types.UnspecifiedLength {
			tp.Flen = defaultFlen
		}
		if tp.Decimal == types.UnspecifiedLength {
			tp.Decimal = defaultDecimal
		}
		$$ = &ast.FuncCastExpr{
			Expr:         $3,
			Tp:           tp,
			FunctionType: ast.CastFunction,
			TimeZone:     $4.(*ast.CastTimeZone),
		}
	}
|	caseKwd ExpressionOpt WhenClauseList ElseOpt end
	{
		x := &ast.CaseExpr{WhenClauses: $3.([]*ast.WhenClause)}
		if $2 != nil {
			x.Value = $2
		}
		if $4 != nil {
			x.ElseClause = $4.(ast.ExprNode)
		}
		$$ = x
	}
|	convert '(' Expression ',' CastType ')'
	{
		// See https://dev.mysql.com/doc/refman/5.7/en/cast-functions.html#function_convert
		tp := $5.(*types.FieldType)
		defaultFlen, defaultDecimal := mysql.GetDefaultFieldLengthAndDecimalForCast(tp.Tp)
		if tp.Flen == types.UnspecifiedLength {
			tp.Flen = defaultFlen
		}
		if tp.Decimal == types.UnspecifiedLength {
			tp.Decimal = defaultDecimal
		}
		$$ = &ast.FuncCastExpr{
			Expr: $3,
			Tp: tp,
			FunctionType: ast.CastConvertFunction,
		}
	}
|	convert '(' Expression using CharsetName ')'
	{
		// See https://dev.mysql.com/doc/refman/5.7/en/cast-functions.html#function_convert
		charset1 := ast.NewValueExpr($5)
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr($1),
			Args: []ast.ExprNode{$3, charset1},
		}
	}
|	defaultKwd '(' SimpleIdent ')'
	{
		$$ = &ast.DefaultExpr{Name: $3.(*ast.ColumnNameExpr).Name}
	}
|	values '(' SimpleIdent ')' %prec lowerThanInsertValues
	{
		$$ = &ast.ValuesExpr{Column: $3.(*ast.ColumnNameExpr)}
	}
|	value '(' SimpleIdent ')'
	{
		$$ = &ast.ValuesExpr{Column: $3.(*ast.ColumnNameExpr), Singular: true}
	}
|	SimpleIdent jss stringLit
	{
	    expr := ast.NewValueExpr($3)
	    $$ = &ast.FuncCallExpr{FnName: model.NewCIStr(ast.JSONExtract), Args: []ast.ExprNode{$1, expr}}
	}
|	SimpleIdent juss stringLit
	{
	    expr := ast.NewValueExpr($3)
	    extract := &ast.FuncCallExpr{FnName: model.NewCIStr(ast.JSONExtract), Args: []ast.ExprNode{$1, expr}}
	    $$ = &ast.FuncCallExpr{FnName: model.NewCIStr(ast.JSONUnquote), Args: []ast.ExprNode{extract}}
	}


DistinctKwd:
	distinct
|	distinctRow


DistinctOpt:
	all       { $$ = false }
|	DistinctKwd { $$ = true }


DefaultFalseDistinctOpt:
	/* EMPTY */ { $$ = false }
|	DistinctOpt


DefaultTrueDistinctOpt:
	/* EMPTY */ { $$ = true }
|	DistinctOpt


BuggyDefaultFalseDistinctOpt:
	DefaultFalseDistinctOpt
|	DistinctKwd all { $$ = true }


FunctionNameConflict:
	ascii
|	charsetKwd
|	coalesce
|	collation
|	dateType
|	database
|	day
|	hour
|	ifKwd
|	format
|	left
|	log
|	microsecond
|	minute
|	month
|	builtinNow
|	quarter
|	repeat
|	replace
|	reverse
|	right
|	rowCount
|	second
|	timeType
|	timestampType
|	truncate
|	user
|	week
|	yearType
|	uuidType


OptionalBraces:
	{} | '(' ')' {}


FunctionNameOptionalBraces:
	currentUser
|	currentDate
|	currentRole
|	utcDate


FunctionNameDatetimePrecision:
	currentTime
|	currentTs
|	localTime
|	localTs
|	utcTime
|	utcTimestamp


FunctionCallKeyword:
	FunctionNameConflict '(' ExpressionListOpt ')'
	{
		args := $3.([]ast.ExprNode)
		if err := validateFunctionCall($1, args); err != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "%v", err))
			return 1
		}
		$$ = &ast.FuncCallExpr{FnName: model.NewCIStr($1), Args: args}
	}
|	interval '(' Expression ',' ExpressionList ')'
	{
		args := append([]ast.ExprNode{$3}, $5.([]ast.ExprNode)...)
		$$ = &ast.FuncCallExpr{FnName: model.NewCIStr($1), Args: args}
	}
|	SpatialFunctionName '(' ExpressionListOpt ')'
	{
		$$ = &ast.FuncCallExpr{FnName: model.NewCIStr($1), Args: $3.([]ast.ExprNode)}
	}
|	builtinUser '('	ExpressionListOpt ')'
	{
		$$ = &ast.FuncCallExpr{FnName: model.NewCIStr($1), Args: $3.([]ast.ExprNode)}
	}
|	FunctionNameOptionalBraces OptionalBraces
	{
		$$ = &ast.FuncCallExpr{FnName: model.NewCIStr($1)}
	}
|	builtinCurDate '(' ')'
    {
		$$ = &ast.FuncCallExpr{FnName: model.NewCIStr($1)}
    }
|	FunctionNameDatetimePrecision FuncDatetimePrec
	{
		args := []ast.ExprNode{}
		if $2 != nil {
			args = append(args, $2.(ast.ExprNode))
		}
		$$ = &ast.FuncCallExpr{FnName: model.NewCIStr($1), Args: args}
	}
|	charType '(' ExpressionList ')'
	{
		nilVal := ast.NewValueExpr(nil)
		args := $3.([]ast.ExprNode)
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr(ast.CharFunc),
			Args: append(args, nilVal),
		}
	}
|	charType '(' ExpressionList using CharsetName ')'
	{
		charset1 := ast.NewValueExpr($5)
		args := $3.([]ast.ExprNode)
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr(ast.CharFunc),
			Args: append(args, charset1),
		}
	}
|	dateType  stringLit
	{
		expr := ast.NewValueExpr($2)
		$$ = &ast.FuncCallExpr{FnName: model.NewCIStr(ast.DateLiteral), Args: []ast.ExprNode{expr}}
	}
|	timeType  stringLit
	{
		expr := ast.NewValueExpr($2)
		$$ = &ast.FuncCallExpr{FnName: model.NewCIStr(ast.TimeLiteral), Args: []ast.ExprNode{expr}}
	}
|	timestampType  stringLit
	{
		expr := ast.NewValueExpr($2)
		$$ = &ast.FuncCallExpr{FnName: model.NewCIStr(ast.TimestampLiteral), Args: []ast.ExprNode{expr}}
	}
|	insert '(' ExpressionListOpt ')'
	{
		$$ = &ast.FuncCallExpr{FnName:model.NewCIStr(ast.InsertFunc), Args: $3.([]ast.ExprNode)}
	}
|	mod '(' BitExpr ',' BitExpr ')'
	{
		$$ = &ast.BinaryOperationExpr{Op: opcode.Mod, L: $3, R: $5}
	}
|	password '(' ExpressionListOpt ')'
	{
		$$ = &ast.FuncCallExpr{FnName:model.NewCIStr(ast.PasswordFunc), Args: $3.([]ast.ExprNode)}
	}


FunctionCallNonKeyword:
	weightString '(' Expression WeightStringAsOpt WeightStringLevelOpt ')'
	{
		x := $4.(*ast.WeightStringExpr)
		level := $5.(*ast.WeightStringExpr)
		if x.CastType == ast.WeightStringCastBinary && level.HasLevel {
			yyAppendErr(yylex, yyErrorf(yylex, "WEIGHT_STRING AS BINARY does not allow LEVEL"))
			return 1
		}
		x.Expr = $3
		x.Levels = level.Levels
		x.LevelRange = level.LevelRange
		x.HasLevel = level.HasLevel
		x.Level = level.Level
		x.LevelOrder = level.LevelOrder
		x.Reverse = level.Reverse
		$$ = x
	}
|	weightString '(' Expression ',' WeightStringParam ',' WeightStringParam ',' WeightStringParam ')'
	{
		$$ = &ast.WeightStringExpr{Expr: $3, Params: []ast.ExprNode{$5, $7, $9}}
	}
|	builtinCurTime '(' FuncDatetimePrecListOpt ')'
	{
		$$ = &ast.FuncCallExpr{FnName: model.NewCIStr($1), Args: $3.([]ast.ExprNode)}
	}
|	builtinSysDate '(' FuncDatetimePrecListOpt ')'
	{
		$$ = &ast.FuncCallExpr{FnName: model.NewCIStr($1), Args: $3.([]ast.ExprNode)}
	}
|	FunctionNameDateArithMultiForms '(' Expression ',' Expression ')'
	{
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr($1),
			Args: []ast.ExprNode{
				$3,
				$5,
				ast.NewValueExpr("DAY"),
			},
		}
	}
|	FunctionNameDateArithMultiForms '(' Expression ',' interval Expression TimeUnit ')'
	{
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr($1),
			Args: []ast.ExprNode{
				$3,
				$6,
				ast.NewValueExpr($7),
			},
		}
	}
|	FunctionNameDateArith '(' Expression ',' interval Expression TimeUnit ')'
	{
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr($1),
			Args: []ast.ExprNode{
				$3,
				$6,
				ast.NewValueExpr($7),
			},
		}
	}
|	builtinExtract '(' TimeUnit from Expression ')'
	{
		timeUnit := ast.NewValueExpr($3)
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr($1),
			Args: []ast.ExprNode{timeUnit, $5},
		}
	}
|	getFormat '(' GetFormatSelector ','  Expression ')'
	{
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr($1),
			Args: []ast.ExprNode{ast.NewValueExpr($3), $5},
		}
	}
|	builtinPosition '(' BitExpr in Expression ')'
	{
		$$ = &ast.FuncCallExpr{FnName: model.NewCIStr($1), Args: []ast.ExprNode{$3, $5}}
	}
|	builtinSubstring '(' Expression ',' Expression ')'
	{
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr($1),
			Args: []ast.ExprNode{$3, $5},
		}
	}
|	builtinSubstring '(' Expression from Expression ')'
	{
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr($1),
			Args: []ast.ExprNode{$3, $5},
		}
	}
|	builtinSubstring '(' Expression ',' Expression ',' Expression ')'
	{
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr($1),
			Args: []ast.ExprNode{$3, $5, $7},
		}
	}
|	builtinSubstring '(' Expression from Expression forKwd Expression ')'
	{
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr($1),
			Args: []ast.ExprNode{$3, $5, $7},
		}
	}
|	timestampAdd '(' TimestampUnit ',' Expression ',' Expression ')'
	{
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr($1),
			Args: []ast.ExprNode{ast.NewValueExpr($3), $5, $7},
		}
	}
|	timestampDiff '(' TimestampUnit ',' Expression ',' Expression ')'
	{
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr($1),
			Args: []ast.ExprNode{ast.NewValueExpr($3), $5, $7},
		}
	}
|	builtinTrim '(' Expression ')'
	{
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr($1),
			Args: []ast.ExprNode{$3},
		}
	}
|	builtinTrim '(' Expression from Expression ')'
	{
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr($1),
			Args: []ast.ExprNode{$5, $3},
		}
	}
|	builtinTrim '(' TrimDirection from Expression ')'
	{
		nilVal := ast.NewValueExpr(nil)
		direction := ast.NewValueExpr(int($3.(ast.TrimDirectionType)))
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr($1),
			Args: []ast.ExprNode{$5, nilVal, direction},
		}
	}
|	builtinTrim '(' TrimDirection Expression from Expression ')'
	{
		direction := ast.NewValueExpr(int($3.(ast.TrimDirectionType)))
		$$ = &ast.FuncCallExpr{
			FnName: model.NewCIStr($1),
			Args: []ast.ExprNode{$6, $4, direction},
		}
	}


WeightStringAsOpt:
	/* EMPTY */
	{
		$$ = &ast.WeightStringExpr{}
	}
|	as charType '(' WeightStringLength ')'
	{
		$$ = &ast.WeightStringExpr{CastType: ast.WeightStringCastChar, Length: $4.(uint64)}
	}
|	as binaryType '(' WeightStringLength ')'
	{
		$$ = &ast.WeightStringExpr{CastType: ast.WeightStringCastBinary, Length: $4.(uint64)}
	}


WeightStringLevelOpt:
	/* EMPTY */
	{
		$$ = &ast.WeightStringExpr{}
	}
|	level WeightStringLevelList
	{
		levels := $2.([]ast.WeightStringLevel)
		first := levels[0]
		$$ = &ast.WeightStringExpr{
			HasLevel: true, Levels: levels,
			Level: first.Number, LevelOrder: first.Order, Reverse: first.Reverse,
		}
	}
|	level UnsignedInteger '-' UnsignedInteger
	{
		$$ = &ast.WeightStringExpr{
			HasLevel: true, Level: $2.(uint64),
			LevelRange: &ast.WeightStringLevelRange{Start: $2.(uint64), End: $4.(uint64)},
		}
	}


WeightStringLevelList:
	WeightStringLevelItem
	{
		$$ = []ast.WeightStringLevel{$1.(ast.WeightStringLevel)}
	}
|	WeightStringLevelList ',' WeightStringLevelItem
	{
		$$ = append($1.([]ast.WeightStringLevel), $3.(ast.WeightStringLevel))
	}


WeightStringLevelItem:
	UnsignedInteger WeightStringLevelFlags
	{
		flags := $2.(int)
		x := ast.WeightStringLevel{Number: $1.(uint64)}
		if flags&1 != 0 {
			x.Order = ast.WeightStringLevelOrderAsc
		} else if flags&2 != 0 {
			x.Order = ast.WeightStringLevelOrderDesc
		}
		x.Reverse = flags&4 != 0
		$$ = x
	}


WeightStringLevelFlags:
	/* EMPTY */ { $$ = 0 }
|	asc         { $$ = 1 }
|	desc        { $$ = 2 }
|	reverse     { $$ = 4 }
|	asc reverse { $$ = 1 | 4 }
|	desc reverse { $$ = 2 | 4 }


WeightStringLength:
	LengthNum
	{
		value := $1.(uint64)
		if value == 0 {
			yyAppendErr(yylex, yyErrorf(yylex, "length must be greater than zero"))
			return 1
		}
		$$ = value
	}
|	hexLit
	{
		value, err := hexInteger(yyDollar[1].ident, true)
		if err != nil || value == 0 {
			yyAppendErr(yylex, yyErrorf(yylex, "expected a positive integer length"))
			return 1
		}
		$$ = value
	}


UnsignedInteger:
	LengthNum
|	HexInteger


HexInteger:
	hexLit
	{
		value, err := hexInteger(yyDollar[1].ident, false)
		if err != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "%v", err))
			return 1
		}
		$$ = value
	}


WeightStringParam:
	NumLiteral     { $$ = ast.NewValueExpr($1) }
|	hexLit
	{
		value, err := hexInteger(yyDollar[1].ident, true)
		if err != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "%v", err))
			return 1
		}
		// Use the same numeric AST type as the decimal spelling emitted by Restore.
		if value <= math.MaxInt64 {
			$$ = ast.NewValueExpr(int64(value))
		} else {
			$$ = ast.NewValueExpr(value)
		}
	}
|	'+' NumLiteral { $$ = &ast.UnaryOperationExpr{Op: opcode.Plus, V: ast.NewValueExpr($2)} }


GetFormatSelector:
	dateType      { $$ = strings.ToUpper($1) }
| 	datetimeType  { $$ = strings.ToUpper($1) }
|	timeType      { $$ = strings.ToUpper($1) }
|	timestampType { $$ = strings.ToUpper($1) }


FunctionNameDateArith:
	builtinDateAdd
|	builtinDateSub


FunctionNameDateArithMultiForms:
	builtinAddDate
|	builtinSubDate


TrimDirection:
	both     { $$ = ast.TrimBoth }
|	leading  { $$ = ast.TrimLeading }
|	trailing { $$ = ast.TrimTrailing }


SumExpr:
	avg '(' BuggyDefaultFalseDistinctOpt Expression ')'  OptWindowingClause
	{
		if $6 != nil {
			$$ = &ast.WindowFuncExpr{F: $1, Args: []ast.ExprNode{$4}, Distinct: $3.(bool), Spec: *($6.(*ast.WindowSpec)),}
		} else {
			$$ = &ast.AggregateFuncExpr{F: $1, Args: []ast.ExprNode{$4}, Distinct: $3.(bool)}
		}
	}
|	builtinBitAnd '(' Expression ')'  OptWindowingClause
	{
		if $5 != nil {
			$$ = &ast.WindowFuncExpr{F: $1, Args: []ast.ExprNode{$3}, Spec: *($5.(*ast.WindowSpec)),}
		} else {
			$$ = &ast.AggregateFuncExpr{F: $1, Args: []ast.ExprNode{$3},}
		}
	}
|	builtinBitAnd '(' all Expression ')'  OptWindowingClause
	{
		if $6 != nil {
			$$ = &ast.WindowFuncExpr{F: $1, Args: []ast.ExprNode{$4}, Spec: *($6.(*ast.WindowSpec)),}
		} else {
			$$ = &ast.AggregateFuncExpr{F: $1, Args: []ast.ExprNode{$4},}
		}
	}
|	builtinBitOr '(' Expression ')'  OptWindowingClause
	{
		if $5 != nil {
			$$ = &ast.WindowFuncExpr{F: $1, Args: []ast.ExprNode{$3}, Spec: *($5.(*ast.WindowSpec)),}
		} else {
			$$ = &ast.AggregateFuncExpr{F: $1, Args: []ast.ExprNode{$3},}
		}
	}
|	builtinBitOr '(' all Expression ')'  OptWindowingClause
	{
		if $6 != nil {
			$$ = &ast.WindowFuncExpr{F: $1, Args: []ast.ExprNode{$4}, Spec: *($6.(*ast.WindowSpec)),}
		} else {
			$$ = &ast.AggregateFuncExpr{F: $1, Args: []ast.ExprNode{$4},}
		}
	}
|	builtinBitXor '(' Expression ')'  OptWindowingClause
	{
		if $5 != nil {
			$$ = &ast.WindowFuncExpr{F: $1, Args: []ast.ExprNode{$3}, Spec: *($5.(*ast.WindowSpec)),}
		} else {
			$$ = &ast.AggregateFuncExpr{F: $1, Args: []ast.ExprNode{$3},}
		}
	}
|	builtinBitXor '(' all Expression ')'  OptWindowingClause
	{
		if $6 != nil {
			$$ = &ast.WindowFuncExpr{F: $1, Args: []ast.ExprNode{$4}, Spec: *($6.(*ast.WindowSpec)),}
		} else {
			$$ = &ast.AggregateFuncExpr{F: $1, Args: []ast.ExprNode{$4},}
		}
	}
|	builtinCount '(' DistinctKwd ExpressionList ')' OptWindowingClause
	{
		if $6 != nil {
			$$ = &ast.WindowFuncExpr{F: $1, Args: $4.([]ast.ExprNode), Distinct: true, Spec: *($6.(*ast.WindowSpec))}
		} else {
			$$ = &ast.AggregateFuncExpr{F: $1, Args: $4.([]ast.ExprNode), Distinct: true}
		}
	}
|	builtinCount '(' all Expression ')'  OptWindowingClause
	{
		if $6 != nil {
			$$ = &ast.WindowFuncExpr{F: $1, Args: []ast.ExprNode{$4}, Spec: *($6.(*ast.WindowSpec)),}
		} else {
			$$ = &ast.AggregateFuncExpr{F: $1, Args: []ast.ExprNode{$4},}
		}
	}
|	builtinCount '(' Expression ')'  OptWindowingClause
	{
		if $5 != nil {
			$$ = &ast.WindowFuncExpr{F: $1, Args: []ast.ExprNode{$3}, Spec: *($5.(*ast.WindowSpec)),}
		} else {
			$$ = &ast.AggregateFuncExpr{F: $1, Args: []ast.ExprNode{$3},}
		}
	}
|	builtinCount '(' '*' ')'  OptWindowingClause
	{
		args := []ast.ExprNode{ast.NewValueExpr(1)}
		if $5 != nil {
			$$ = &ast.WindowFuncExpr{F: $1, Args: args, Spec: *($5.(*ast.WindowSpec)),}
		} else {
			$$ = &ast.AggregateFuncExpr{F: $1, Args: args,}
		}
	}
|	builtinGroupConcat '(' BuggyDefaultFalseDistinctOpt ExpressionList OrderByOptional OptGConcatSeparator SelectStmtLimit ')' OptWindowingClause
	{
		args := $4.([]ast.ExprNode)
		if $9 != nil {
			$$ = &ast.WindowFuncExpr{F: $1, Args: args, Distinct: $3.(bool), Spec: *($9.(*ast.WindowSpec))}
		} else {
			agg := &ast.AggregateFuncExpr{F: $1, Args: args, Distinct: $3.(bool)}
			if $5 != nil {
				agg.OrderBy = $5.(*ast.OrderByClause)
			}
			if $6 != nil {
				agg.Separator = $6.(ast.ExprNode)
			}
			if $7 != nil {
				agg.Limit = $7.(*ast.Limit)
			}
			$$ = agg
		}
	}
|	builtinJSONArrayAgg '(' BuggyDefaultFalseDistinctOpt Expression OrderByOptional SelectStmtLimit ')' OptWindowingClause
	{
		if $8 != nil {
			if $3.(bool) || $5 != nil || $6 != nil {
				yyAppendErr(yylex, yyErrorf(yylex, "JSON_ARRAYAGG window calls do not accept DISTINCT, ORDER BY, or LIMIT"))
				return 1
			}
			$$ = &ast.WindowFuncExpr{F: $1, Args: []ast.ExprNode{$4}, Spec: *($8.(*ast.WindowSpec))}
		} else {
			agg := &ast.AggregateFuncExpr{F: $1, Args: []ast.ExprNode{$4}, Distinct: $3.(bool)}
			if $5 != nil {
				agg.OrderBy = $5.(*ast.OrderByClause)
			}
			if $6 != nil {
				agg.Limit = $6.(*ast.Limit)
			}
			$$ = agg
		}
	}
|	builtinMax '(' BuggyDefaultFalseDistinctOpt Expression ')'  OptWindowingClause
	{
		if $6 != nil {
			$$ = &ast.WindowFuncExpr{F: $1, Args: []ast.ExprNode{$4}, Distinct: $3.(bool), Spec: *($6.(*ast.WindowSpec)),}
		} else {
			$$ = &ast.AggregateFuncExpr{F: $1, Args: []ast.ExprNode{$4}, Distinct: $3.(bool)}
		}
	}
|	builtinMin '(' BuggyDefaultFalseDistinctOpt Expression ')'  OptWindowingClause
	{
		if $6 != nil {
			$$ = &ast.WindowFuncExpr{F: $1, Args: []ast.ExprNode{$4}, Distinct: $3.(bool), Spec: *($6.(*ast.WindowSpec)),}
		} else {
			$$ = &ast.AggregateFuncExpr{F: $1, Args: []ast.ExprNode{$4}, Distinct: $3.(bool)}
		}
	}
|	builtinSum '(' BuggyDefaultFalseDistinctOpt Expression ')'  OptWindowingClause
	{
		if $6 != nil {
			$$ = &ast.WindowFuncExpr{F: $1, Args: []ast.ExprNode{$4}, Distinct: $3.(bool), Spec: *($6.(*ast.WindowSpec)),}
		} else {
			$$ = &ast.AggregateFuncExpr{F: $1, Args: []ast.ExprNode{$4}, Distinct: $3.(bool)}
		}
	}
|	builtinStddevPop '(' BuggyDefaultFalseDistinctOpt Expression ')'  OptWindowingClause
	{
		if $6 != nil {
			$$ = &ast.WindowFuncExpr{F: $1, Args: []ast.ExprNode{$4}, Distinct: $3.(bool), Spec: *($6.(*ast.WindowSpec)),}
		} else {
			$$ = &ast.AggregateFuncExpr{F: $1, Args: []ast.ExprNode{$4}, Distinct: $3.(bool)}
		}
	}
|	builtinStddevSamp '(' BuggyDefaultFalseDistinctOpt Expression ')'  OptWindowingClause
	{
		if $6 != nil {
			$$ = &ast.WindowFuncExpr{F: $1, Args: []ast.ExprNode{$4}, Distinct: $3.(bool), Spec: *($6.(*ast.WindowSpec)),}
		} else {
			$$ = &ast.AggregateFuncExpr{F: $1, Args: []ast.ExprNode{$4}, Distinct: $3.(bool)}
		}
	}
|	builtinVarPop '(' BuggyDefaultFalseDistinctOpt Expression ')'  OptWindowingClause
	{
		if $6 != nil {
			$$ = &ast.WindowFuncExpr{F: ast.AggFuncVarPop, Args: []ast.ExprNode{$4}, Distinct: $3.(bool), Spec: *($6.(*ast.WindowSpec)),}
		} else {
			$$ = &ast.AggregateFuncExpr{F: ast.AggFuncVarPop, Args: []ast.ExprNode{$4}, Distinct: $3.(bool)}
		}
	}
|	builtinVarSamp '(' BuggyDefaultFalseDistinctOpt Expression ')'  OptWindowingClause
	{
		if $6 != nil {
			$$ = &ast.WindowFuncExpr{F: strings.ToLower($1), Args: []ast.ExprNode{$4}, Distinct: $3.(bool), Spec: *($6.(*ast.WindowSpec)),}
		} else {
			$$ = &ast.AggregateFuncExpr{F: strings.ToLower($1), Args: []ast.ExprNode{$4}, Distinct: $3.(bool)}
		}
	}


OptGConcatSeparator:
	/* EMPTY */         { $$ = nil }
| separator stringLit { $$ = ast.NewValueExpr($2) }


FunctionCallGeneric:
	identifier '(' ExpressionListOpt ')' %prec lowerThanWithin
	{
		args := $3.([]ast.ExprNode)
		if err := validateSourceFunctionCall(yylex, yyDollar[1].offset, $1, args); err != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "%v", err))
			return 1
		}
		$$ = &ast.FuncCallExpr{FnName: model.NewCIStr($1), Args: args}
	}
|	identifier '(' ExpressionListOpt ')' WindowingClause
	{
		$$ = &ast.WindowFuncExpr{F: strings.ToLower($1), Args: $3.([]ast.ExprNode), Spec: $5.(ast.WindowSpec)}
	}
|	identifier '(' ExpressionListOpt ')' within group '(' WithinGroupOrderBy ')' WindowingClause
	{
		$$ = &ast.WindowFuncExpr{
			F:           strings.ToLower($1),
			Args:        $3.([]ast.ExprNode),
			WithinGroup: $8.(*ast.OrderByClause),
			Spec:        $10.(ast.WindowSpec),
		}
	}


FuncDatetimePrec:
	/* EMPTY */ { $$ = nil }
|	'(' ')'     { $$ = nil }
|	'(' intLit ')'
	{
		expr := ast.NewValueExpr($2)
		$$ = expr
	}


TimeUnit:
	microsecond        { $$ = strings.ToUpper($1) }
|	second             { $$ = strings.ToUpper($1) }
|	minute             { $$ = strings.ToUpper($1) }
|	hour               { $$ = strings.ToUpper($1) }
|	day                { $$ = strings.ToUpper($1) }
|	week               { $$ = strings.ToUpper($1) }
|	month              { $$ = strings.ToUpper($1) }
|	quarter            { $$ = strings.ToUpper($1) }
|	yearType               { $$ = strings.ToUpper($1) }
|	secondMicrosecond { $$ = strings.ToUpper($1) }
|	minuteMicrosecond { $$ = strings.ToUpper($1) }
|	minuteSecond      { $$ = strings.ToUpper($1) }
|	hourMicrosecond   { $$ = strings.ToUpper($1) }
|	hourSecond        { $$ = strings.ToUpper($1) }
|	hourMinute        { $$ = strings.ToUpper($1) }
|	dayMicrosecond    { $$ = strings.ToUpper($1) }
|	daySecond         { $$ = strings.ToUpper($1) }
|	dayMinute         { $$ = strings.ToUpper($1) }
|	dayHour           { $$ = strings.ToUpper($1) }
|	yearMonth         { $$ = strings.ToUpper($1) }


TimestampUnit:
	microsecond { $$ = strings.ToUpper($1) }
|	second      { $$ = strings.ToUpper($1) }
|	minute      { $$ = strings.ToUpper($1) }
|	hour        { $$ = strings.ToUpper($1) }
|	day         { $$ = strings.ToUpper($1) }
|	week        { $$ = strings.ToUpper($1) }
|	month       { $$ = strings.ToUpper($1) }
|	quarter     { $$ = strings.ToUpper($1) }
|	yearType        { $$ = strings.ToUpper($1) }
|	identifier
	{
		unit := strings.ToUpper($1)
		switch unit {
		case "SQL_TSI_SECOND", "SQL_TSI_MINUTE",
			"SQL_TSI_HOUR", "SQL_TSI_DAY", "SQL_TSI_WEEK", "SQL_TSI_MONTH", "SQL_TSI_QUARTER", "SQL_TSI_YEAR":
			$$ = unit
		default:
			yyAppendErr(yylex, yyErrorf(yylex, "invalid TIMESTAMPADD/TIMESTAMPDIFF unit %q", $1))
			return 1
		}
	}


ExpressionOpt:
	/* EMPTY */ { $$ = nil }
|	Expression  { $$ = $1 }


WhenClauseList:
	WhenClause
	{
		$$ = []*ast.WhenClause{$1.(*ast.WhenClause)}
	}
|	WhenClauseList WhenClause
	{
		$$ = append($1.([]*ast.WhenClause), $2.(*ast.WhenClause))
	}


WhenClause:
	when Expression then Expression
	{
		$$ = &ast.WhenClause{
			Expr: $2,
			Result: $4,
		}
	}


ElseOpt:
	/* EMPTY */ { $$ = nil }
|	elseKwd Expression { $$ = $2 }


CastType:
	binaryType OptFieldLen
	{
		x := types.NewFieldType(mysql.TypeVarString)
		x.Flen = $2.(int)
		if x.Flen != types.UnspecifiedLength {
			x.Tp = mysql.TypeString
		}
		x.Charset = charset.CharsetBin
		x.Collate = charset.CollationBin
		x.Flag |= mysql.BinaryFlag
		$$ = x
	}
|	charType OptFieldLen OptBinary
	{
		x := types.NewFieldType(mysql.TypeVarString)
		x.Flen = $2.(int)
		x.Charset = $3.(*ast.OptBinary).Charset
		if $3.(*ast.OptBinary).IsBinary{
			x.Flag |= mysql.BinaryFlag
		}
		if x.Charset == "" {
			x.Charset = mysql.DefaultCharset
			x.Collate = mysql.DefaultCollationName
		}
		$$ = x
	}
|	varcharType FieldLen OptBinary
	{
		x := types.NewFieldType(mysql.TypeVarchar)
		x.Flen = $2.(int)
		x.Charset = $3.(*ast.OptBinary).Charset
		if $3.(*ast.OptBinary).IsBinary {
			x.Flag |= mysql.BinaryFlag
		}
		if x.Charset == "" {
			x.Charset = mysql.DefaultCharset
			x.Collate = mysql.DefaultCollationName
		}
		$$ = x
	}
|	dateType
	{
		x := types.NewFieldType(mysql.TypeDate)
		x.Charset = charset.CharsetBin
		x.Collate = charset.CollationBin
		x.Flag |= mysql.BinaryFlag
		$$ = x
	}
|	CastDatetimeType
|	decimalType FloatOpt
	{
		fopt := $2.(*ast.FloatOpt)
		x := types.NewFieldType(mysql.TypeNewDecimal)
		x.Flen = fopt.Flen
		x.Decimal = fopt.Decimal
		x.Charset = charset.CharsetBin
		x.Collate = charset.CollationBin
		x.Flag |= mysql.BinaryFlag
		$$ = x
	}
|	FloatingPointType FloatOpt
	{
		fopt := $2.(*ast.FloatOpt)
		x := types.NewFieldType($1.(byte))
		x.Flen = fopt.Flen
		x.Decimal = fopt.Decimal
		if x.Tp == mysql.TypeFloat && x.Flen > 24 {
			x.Tp = mysql.TypeDouble
		}
		x.Charset = charset.CharsetBin
		x.Collate = charset.CollationBin
		x.Flag |= mysql.BinaryFlag
		$$ = x
	}
|	timeType CastPrecisionOpt
	{
		$$ = temporalCastType(mysql.TypeDuration, $2.(string))
	}
|	interval daySecond CastPrecisionOpt
	{
		x := temporalCastType(mysql.TypeDuration, $3.(string))
		x.CastTypeName = "INTERVAL DAY_SECOND"
		$$ = x
	}
|	intType
	{
		x := types.NewFieldType(mysql.TypeLong)
		x.CastTypeName = "INT"
		x.Charset, x.Collate = charset.CharsetBin, charset.CollationBin
		x.Flag |= mysql.BinaryFlag
		$$ = x
	}
|	integerType
	{
		x := types.NewFieldType(mysql.TypeLong)
		x.CastTypeName = "INTEGER"
		x.Charset, x.Collate = charset.CharsetBin, charset.CollationBin
		x.Flag |= mysql.BinaryFlag
		$$ = x
	}
|	signed OptInteger
	{
		x := types.NewFieldType(mysql.TypeLonglong)
		x.Charset = charset.CharsetBin
		x.Collate = charset.CollationBin
		x.Flag |= mysql.BinaryFlag
		$$ = x
	}
|	unsigned OptInteger
	{
		x := types.NewFieldType(mysql.TypeLonglong)
		x.Flag |= mysql.UnsignedFlag | mysql.BinaryFlag
		x.Charset = charset.CharsetBin
		x.Collate = charset.CollationBin
		$$ = x
	}
|	jsonType
	{
		x := types.NewFieldType(mysql.TypeJSON)
		x.Flag |= mysql.BinaryFlag | (mysql.ParseToJSONFlag)
		x.Charset = mysql.DefaultCharset
		x.Collate = mysql.DefaultCollationName
		$$ = x
	}
|	yearType
	{
		x := types.NewFieldType(mysql.TypeYear)
		x.Flag |= mysql.BinaryFlag
		x.Charset = charset.CharsetBin
		x.Collate = charset.CollationBin
		$$ = x
	}


CastDatetimeType:
	datetimeType CastPrecisionOpt
	{
		$$ = temporalCastType(mysql.TypeDatetime, $2.(string))
	}

CastPrecisionOpt:
	/* EMPTY */ %prec empty { $$ = "" }
|	'(' LengthNum ')'
	{
		$$ = fmt.Sprintf("%d", $2.(uint64))
	}


CastTimeZone:
	at local
	{
		$$ = &ast.CastTimeZone{Local: true}
	}
|	at timeType zone stringLit
	{
		$$ = &ast.CastTimeZone{Expr: ast.NewValueExpr($4)}
	}
|	at timeType zone interval stringLit
	{
		$$ = &ast.CastTimeZone{Expr: ast.NewValueExpr($5), Interval: true}
	}


PriorityOpt:
	/* EMPTY */     { $$ = mysql.NoPriority }
|	lowPriority  { $$ = mysql.LowPriority }
|	highPriority { $$ = mysql.HighPriority }
|	delayed       { $$ = mysql.DelayedPriority }


TableName:
	Identifier                { $$ = &ast.TableName{Name:model.NewCIStr($1), NameOffset:yyS[yypt].offset} }
|	FunctionNameIdentifier    { $$ = &ast.TableName{Name:model.NewCIStr($1), NameOffset:yyS[yypt].offset} }
|	Identifier '.' Identifier { $$ = &ast.TableName{Schema:model.NewCIStr($1), Name:model.NewCIStr($3), NameOffset:yyS[yypt].offset} }
|	Identifier '.' '*'        { $$ = &ast.TableName{Name:model.NewCIStr($1), NameOffset:yyS[yypt-2].offset} }


/* Only these function tokens remain table names before '('; SYSDATE is rejected. */
FunctionNameIdentifier:
	builtinAddDate
|	builtinSubDate
|	builtinUser


/* Stored routine names may shadow a built-in function token, unlike table names. */
RoutineName:
	TableName
|	builtinSysDate
	{
		$$ = &ast.TableName{Name: model.NewCIStr($1), NameOffset: yyS[yypt].offset}
	}


TableNameList:
	TableName
	{
		tbl := []*ast.TableName{$1.(*ast.TableName)}
		$$ = tbl
	}
|	TableNameList ',' TableName
	{
		$$ = append($1.([]*ast.TableName), $3.(*ast.TableName))
	}


ValuesStmt:
	ValuesStmtKeyword ValuesRowList
	{
		$$ = &ast.ValuesStmt{
			Rows:                $2.([]*ast.RowExpr),
			ValuesKeywordOffset: $1.(int),
		}
	}


ValuesStmtKeyword:
	values
	{
		$$ = yyGetParser(yylex).startOffset(&yyS[yypt])
	}


TableStmt:
	tableKwd TableName
	{
		$$ = &ast.TableStmt{Table: $2.(*ast.TableName)}
	}


ValuesRowList:
	ValuesRow
	{
		$$ = []*ast.RowExpr{$1.(*ast.RowExpr)}
	}
|	ValuesRowList ',' ValuesRow
	{
		rows := $1.([]*ast.RowExpr)
		next := $3.(*ast.RowExpr)
		if rows[0].OmitRowKeyword != next.OmitRowKeyword {
			yyAppendErr(yylex, yyErrorf(yylex, "VALUES cannot mix ROW and parenthesized row forms"))
			return 1
		}
		$$ = append(rows, next)
	}


ValuesRow:
	row '(' ExpressionList ')'
	{
		$$ = &ast.RowExpr{Values: $3.([]ast.ExprNode)}
	}
|	'(' ExpressionList ')'
	{
		$$ = &ast.RowExpr{Values: $2.([]ast.ExprNode), OmitRowKeyword: true}
	}


QuickOptional:
	%prec empty
	{
		$$ = false
	}
|	quick
	{
		$$ = true
	}


/***************************Prepared Statement Start******************************
 * See https://dev.mysql.com/doc/refman/5.7/en/prepare.html
 * Example:
 * PREPARE stmt_name FROM 'SELECT SQRT(POW(?,2) + POW(?,2)) AS hypotenuse';
 * OR
 * SET @s = 'SELECT SQRT(POW(?,2) + POW(?,2)) AS hypotenuse';
 * PREPARE stmt_name FROM @s;
 */


PreparedStmt:
	prepare Identifier from PrepareSQL
	{
		var sqlText string
		var sqlVar *ast.VariableExpr
		switch $4.(type) {
		case string:
			sqlText = $4.(string)
		case *ast.VariableExpr:
			sqlVar = $4.(*ast.VariableExpr)
		}
		$$ = &ast.PrepareStmt{
			Name:		$2,
			SQLText:	sqlText,
			SQLVar: 	sqlVar,
		}
	}


PrepareSQL:
	/* EMPTY */ stringLit { $$ = $1 }
|	UserVariable          { $$ = $1.(interface{}) }


/*
 * See https://dev.mysql.com/doc/refman/5.7/en/execute.html
 * Example:
 * EXECUTE stmt1 USING @a, @b;
 * OR
 * EXECUTE stmt1;
 */
ExecuteStmt:
	execute Identifier ExecuteUsingOpt
	{
		$$ = &ast.ExecuteStmt{Name: $2, UsingVars: $3.([]ast.ExprNode)}
	}
|	execute immediate Expression ExecuteUsingOpt
	{
		$$ = &ast.ExecuteStmt{SQLExpr: $3, UsingVars: $4.([]ast.ExprNode)}
	}


ExecuteUsingOpt:
	/* EMPTY */ { $$ = []ast.ExprNode(nil) }
|	using ExecuteUsingList { $$ = $2 }


ExecuteUsingList:
	ExprOrDefault { $$ = []ast.ExprNode{$1} }
|	ExecuteUsingList ',' ExprOrDefault { $$ = append($1.([]ast.ExprNode), $3) }


/*
 * See https://dev.mysql.com/doc/refman/5.0/en/deallocate-prepare.html
 */
DeallocateStmt:
	DeallocateSym prepare Identifier
	{
		$$ = &ast.DeallocateStmt{Name: $3}
	}


DeallocateSym:
deallocate | drop


/****************************Prepared Statement End*******************************/
RollbackStmt:
	rollback TransactionWorkOpt TransactionChainReleaseOpt
	{
		$$ = &ast.RollbackStmt{TransactionOptions: $3.(ast.TransactionOptions)}
	}
|	rollback TransactionWorkOpt to SavepointSym identifier
	{
		$$ = &ast.RollbackStmt{SavepointName: $5}
	}


SavepointSym:
	/* EMPTY */
|	savepoint


SavepointStmt:
	savepoint identifier
	{
		$$ = &ast.SavepointStmt{Name: $2}
	}


ReleaseSavepointStmt:
	release savepoint identifier
	{
		$$ = &ast.ReleaseSavepointStmt{Name: $3}
	}


XAStmt:
	xa start XAIdent XAStartExtraOpt
	{
		$$ = &ast.XAStmt{Command: "START", XID: $3.(string), Extra: $4.(string)}
	}
|	xa begin XAIdent XAStartExtraOpt
	{
		$$ = &ast.XAStmt{Command: "BEGIN", XID: $3.(string), Extra: $4.(string)}
	}
|	xa end XAIdent XAEndExtraOpt
	{
		$$ = &ast.XAStmt{Command: "END", XID: $3.(string), Extra: $4.(string)}
	}
|	xa prepare XAIdent
	{
		$$ = &ast.XAStmt{Command: "PREPARE", XID: $3.(string)}
	}
|	xa commit XAIdent XACommitExtraOpt
	{
		$$ = &ast.XAStmt{Command: "COMMIT", XID: $3.(string), Extra: $4.(string)}
	}
|	xa rollback XAIdent
	{
		$$ = &ast.XAStmt{Command: "ROLLBACK", XID: $3.(string)}
	}
|	xa recover
	{
		$$ = &ast.XAStmt{Command: "RECOVER"}
	}


XAIdent:
	stringLit
	{
		$$ = $1
	}


XAStartExtraOpt:
	/* EMPTY */
	{
		$$ = ""
	}
|	join
	{
		$$ = "JOIN"
	}
|	resume
	{
		$$ = "RESUME"
	}


XAEndExtraOpt:
	/* EMPTY */
	{
		$$ = ""
	}
|	suspend
	{
		$$ = "SUSPEND"
	}
|	suspend forKwd migrate
	{
		$$ = "SUSPEND FOR MIGRATE"
	}


XACommitExtraOpt:
	/* EMPTY */
	{
		$$ = ""
	}
|	one phase
	{
		$$ = "ONE PHASE"
	}


WithSelectStmt:
	WithClause SelectStmt
	{
		st := $2.(*ast.SelectStmt)
		st.With = $1.(*ast.WithClause)
		$$ = st
	}
|	WithClause SetOprStmt
	{
		ut := $2.(*ast.SetOprStmt)
		ut.With = $1.(*ast.WithClause)
		$$ = ut
	}
|	WithClause ValuesSetOprStmt
	{
		ut := $2.(*ast.SetOprStmt)
		ut.With = $1.(*ast.WithClause)
		$$ = ut
	}
|	WithClause ParenthesizedQueryStmt
	{
		$$ = withClauseSelect($1.(*ast.WithClause), $2.(ast.ResultSetNode)).(ast.StmtNode)
	}


WithUpdateStmt:
	WithClause UpdateStmt
	{
		st := $2.(*ast.UpdateStmt)
		st.With = $1.(*ast.WithClause)
		$$ = st
	}


WithDeleteStmt:
	WithClause DeleteFromStmt
	{
		st := $2.(*ast.DeleteStmt)
		st.With = $1.(*ast.WithClause)
		$$ = st
	}


WithClause:
	with WithList
	{
		$$ = &ast.WithClause{CTEs: $2.([]*ast.CommonTableExpression)}
	}
|	with recursive WithList WithCycleOpt
	{
		$$ = &ast.WithClause{
			IsRecursive: true,
			CTEs: $3.([]*ast.CommonTableExpression),
			CycleColumns: $4.([]model.CIStr),
		}
	}


WithCycleOpt:
	/* EMPTY */
	{
		$$ = []model.CIStr{}
	}
|	cycle ColumnList restrict
	{
		$$ = $2.([]model.CIStr)
	}


WithList:
	WithClauseItem
	{
		$$ = []*ast.CommonTableExpression{$1.(*ast.CommonTableExpression)}
	}
|	WithList ',' WithClauseItem
	{
		$$ = append($1.([]*ast.CommonTableExpression), $3.(*ast.CommonTableExpression))
	}


WithClauseItem:
	Identifier WithColNameListOpt as CTEBody
	{
		$$ = &ast.CommonTableExpression{
			Name:     model.NewCIStr($1),
			ColNames: $2.([]model.CIStr),
			Query:    $4.(ast.ResultSetNode),
		}
	}


CTEBody:
	'(' SelectStmt ')'
	{
		$$ = $2.(*ast.SelectStmt)
	}
|	'(' SetOprStmt ')'
	{
		$$ = $2.(*ast.SetOprStmt)
	}
|	'(' ValuesSetOprStmt ')'
	{
		$$ = $2.(*ast.SetOprStmt)
	}
|	'(' WithSelectStmt ')'
	{
		$$ = $2.(ast.ResultSetNode)
	}
|	'(' ParenthesizedQueryStmt ')'
	{
		$$ = $2.(ast.ResultSetNode)
	}


WithColNameListOpt:
	/* EMPTY */
	{
		$$ = []model.CIStr{}
	}
|	'(' ColumnList ')'
	{
		$$ = $2.([]model.CIStr)
	}


SelectStmtBasic:
	selectKwd SelectStmtOpts SelectStmtFieldList SelectIntoOpt	{
		st := &ast.SelectStmt {
			SelectStmtOpts: $2.(*ast.SelectStmtOpts),
			Distinct:      $2.(*ast.SelectStmtOpts).Distinct,
			Fields:        $3.(*ast.FieldList),
		}
		if into, ok := $4.(*ast.SelectIntoStatement); ok && into != nil {
			into.BeforeFrom = true
		}
		if err := setSelectInto(st, $4); err != nil {
			yyAppendErr(yylex, err)
			return 1
		}
		$$ = st
	}


SelectStmtFromDualTable:
	SelectStmtBasic FromDual WhereClauseOptional SelectStmtGroup HavingClause WindowClauseOptional
	{
		st := $1.(*ast.SelectStmt)
		lastField := st.Fields.Fields[len(st.Fields.Fields)-1]
		if lastField.Expr != nil && lastField.AsName.O == "" {
			lastEnd := yyS[yypt-4].offset-1
			lastField.SetText(yyGetParser(yylex).src[lastField.Offset:lastEnd])
		}
		if $3 != nil {
			st.Where = $3.(ast.ExprNode)
		}
		if $4 != nil {
			st.GroupBy = $4.(*ast.GroupByClause)
		}
		if $5 != nil {
			st.Having = $5.(*ast.HavingClause)
		}
		if $6 != nil {
			st.WindowSpecs = $6.([]ast.WindowSpec)
		}
		$$ = st
	}


SelectStmtFromTable:
	SelectStmtBasic from
	TableRefsClause WhereClauseOptional SelectStmtGroup HavingClause WindowClauseOptional
	{
		st := $1.(*ast.SelectStmt)
		st.From = $3.(*ast.TableRefsClause)
		if st.SelectStmtOpts.TableHints != nil {
			st.TableHints = st.SelectStmtOpts.TableHints
		}
		lastField := st.Fields.Fields[len(st.Fields.Fields)-1]
		if lastField.Expr != nil && lastField.AsName.O == "" {
			lastEnd := yyGetParser(yylex).endOffset(&yyS[yypt-5])
			lastField.SetText(yyGetParser(yylex).src[lastField.Offset:lastEnd])
		}
		if $4 != nil {
			st.Where = $4.(ast.ExprNode)
		}
		if $5 != nil {
			st.GroupBy = $5.(*ast.GroupByClause)
		}
		if $6 != nil {
			st.Having = $6.(*ast.HavingClause)
		}
		if $7 != nil {
		    st.WindowSpecs = ($7.([]ast.WindowSpec))
		}
		$$ = st
	}


SelectStmt:
	SelectStmtWithoutFrom OrderByOptional SelectStmtLimit SelectLockOpt SelectIntoOpt
	{
		st := $1.(*ast.SelectStmt)
		if $4 != nil {
			lock := $4.(*ast.SelectLockInfo)
			st.LockTp = lock.Type
			st.LockedTables = lock.Tables
			st.LockWaitPolicy = lock.WaitPolicy
		}
		lastField := st.Fields.Fields[len(st.Fields.Fields)-1]
		if lastField.Expr != nil && lastField.AsName.O == "" {
			src := yyGetParser(yylex).src
			var lastEnd int
			if $2 != nil {
				lastEnd = yyS[yypt-3].offset-1
			} else if $3 != nil {
				lastEnd = yyS[yypt-2].offset-1
			} else if $4 != nil {
				lastEnd = yyS[yypt-1].offset-1
			} else if $5 != nil {
				lastEnd = yyS[yypt].offset-1
			} else {
				lastEnd = len(src)
				if src[lastEnd-1] == ';' {
					lastEnd--
				}
			}
			lastField.SetText(src[lastField.Offset:lastEnd])
		}
		if $2 != nil {
			st.OrderBy = $2.(*ast.OrderByClause)
		}
		if $3 != nil {
			st.Limit = $3.(*ast.Limit)
		}
		if err := setSelectInto(st, $5); err != nil {
			yyAppendErr(yylex, err)
			return 1
		}
		$$ = st
	}
|	SelectStmtWithoutFrom OrderByOptional SelectStmtLimit SelectIntoClause SelectLockOpt
	{
		st := $1.(*ast.SelectStmt)
		if $4 != nil {
			if err := setSelectInto(st, $4); err != nil {
				yyAppendErr(yylex, err)
				return 1
			}
		}
		if $5 != nil {
			lock := $5.(*ast.SelectLockInfo)
			st.LockTp = lock.Type
			st.LockedTables = lock.Tables
			st.LockWaitPolicy = lock.WaitPolicy
		}
		lastField := st.Fields.Fields[len(st.Fields.Fields)-1]
		if lastField.Expr != nil && lastField.AsName.O == "" {
			src := yyGetParser(yylex).src
			var lastEnd int
			if $2 != nil {
				lastEnd = yyS[yypt-3].offset-1
			} else if $3 != nil {
				lastEnd = yyS[yypt-2].offset-1
			} else if $4 != nil {
				lastEnd = yyS[yypt-1].offset-1
			} else {
				lastEnd = yyS[yypt].offset-1
			}
			lastField.SetText(src[lastField.Offset:lastEnd])
		}
		if $2 != nil {
			st.OrderBy = $2.(*ast.OrderByClause)
		}
		if $3 != nil {
			st.Limit = $3.(*ast.Limit)
		}
		$$ = st
	}
|	SelectStmtFromDualTable OrderByOptional SelectStmtLimit SelectLockOpt SelectIntoOpt
	{
		st := $1.(*ast.SelectStmt)
		if $2 != nil {
			st.OrderBy = $2.(*ast.OrderByClause)
		}
		if $3 != nil {
			st.Limit = $3.(*ast.Limit)
		}
		if $4 != nil {
			lock := $4.(*ast.SelectLockInfo)
			st.LockTp = lock.Type
			st.LockedTables = lock.Tables
			st.LockWaitPolicy = lock.WaitPolicy
		}
		if err := setSelectInto(st, $5); err != nil {
			yyAppendErr(yylex, err)
			return 1
		}
		$$ = st
	}
|	SelectStmtFromDualTable OrderByOptional SelectStmtLimit SelectIntoClause SelectLockOpt
	{
		st := $1.(*ast.SelectStmt)
		if $2 != nil {
			st.OrderBy = $2.(*ast.OrderByClause)
		}
		if $3 != nil {
			st.Limit = $3.(*ast.Limit)
		}
		if err := setSelectInto(st, $4); err != nil {
			yyAppendErr(yylex, err)
			return 1
		}
		if $5 != nil {
			lock := $5.(*ast.SelectLockInfo)
			st.LockTp = lock.Type
			st.LockedTables = lock.Tables
			st.LockWaitPolicy = lock.WaitPolicy
		}
		$$ = st
	}
|	SelectStmtFromTable OrderByOptional SelectStmtLimit SelectLockOpt SelectIntoOpt
	{
		st := $1.(*ast.SelectStmt)
		if $4 != nil {
			lock := $4.(*ast.SelectLockInfo)
			st.LockTp = lock.Type
			st.LockedTables = lock.Tables
			st.LockWaitPolicy = lock.WaitPolicy
		}
		if $2 != nil {
			st.OrderBy = $2.(*ast.OrderByClause)
		}
		if $3 != nil {
			st.Limit = $3.(*ast.Limit)
		}
		if err := setSelectInto(st, $5); err != nil {
			yyAppendErr(yylex, err)
			return 1
		}
		$$ = st
	}
|	SelectStmtFromTable OrderByOptional SelectStmtLimit SelectIntoClause SelectLockOpt
	{
		st := $1.(*ast.SelectStmt)
		if $4 != nil {
			if err := setSelectInto(st, $4); err != nil {
				yyAppendErr(yylex, err)
				return 1
			}
		}
		if $5 != nil {
			lock := $5.(*ast.SelectLockInfo)
			st.LockTp = lock.Type
			st.LockedTables = lock.Tables
			st.LockWaitPolicy = lock.WaitPolicy
		}
		if $2 != nil {
			st.OrderBy = $2.(*ast.OrderByClause)
		}
		if $3 != nil {
			st.Limit = $3.(*ast.Limit)
		}
		$$ = st
	}


SelectStmtWithoutFrom:
	SelectStmtBasic WhereClauseOptional SelectStmtGroup HavingClause WindowClauseOptional
	{
		st := $1.(*ast.SelectStmt)
		if $2 != nil {
			st.Where = $2.(ast.ExprNode)
		}
		if $3 != nil {
			st.GroupBy = $3.(*ast.GroupByClause)
		}
		if $4 != nil {
			st.Having = $4.(*ast.HavingClause)
		}
		if $5 != nil {
			st.WindowSpecs = $5.([]ast.WindowSpec)
		}
		$$ = st
	}


ProcedureAnalyseClause:
	procedure analyse '(' ExpressionListOpt ')'
	{
		$$ = &ast.ProcedureAnalyseClause{Args: $4.([]ast.ExprNode)}
	}


FromDual:
	from dual


WindowClauseOptional:
	/* EMPTY */                   { $$ = nil }
|	window WindowDefinitionList { $$ = $2.([]ast.WindowSpec) }


WindowDefinitionList:
	WindowDefinition                          { $$ = []ast.WindowSpec{$1.(ast.WindowSpec)} }
|	WindowDefinitionList ',' WindowDefinition { $$ = append($1.([]ast.WindowSpec), $3.(ast.WindowSpec)) }


WindowDefinition:
	WindowName as WindowSpec
	{
		var spec = $3.(ast.WindowSpec)
		spec.Name = $1.(model.CIStr)
		$$ = spec
	}


WindowName:
	Identifier { $$ = model.NewCIStr($1) }


WindowSpec:
	'(' WindowSpecDetails ')' { $$ = $2.(ast.WindowSpec) }


WindowSpecDetails:
	OptExistingWindowName OptPartitionClause OptWindowOrderByClause OptWindowFrameClause
	{
		spec := ast.WindowSpec{Ref: $1.(model.CIStr),}
		if $2 != nil {
		    spec.PartitionBy = $2.(*ast.PartitionByClause)
		}
		if $3 != nil {
		    spec.OrderBy = $3.(*ast.OrderByClause)
		}
		if $4 != nil {
		    spec.Frame = $4.(*ast.FrameClause)
		}
		$$ = spec
	}


OptExistingWindowName:
	/* EMPTY */ { $$ = model.CIStr{} }
|	WindowName  { $$ = $1.(model.CIStr) }


OptPartitionClause:
	/* EMPTY */ {
		$$ = nil
	}
|	partition by ByList
	{
		$$ = &ast.PartitionByClause{Items: $3.([]*ast.ByItem)}
	}


OptWindowOrderByClause:
	/* EMPTY */ {
		$$ = nil
	}
|	order by ByList
	{
		$$ = &ast.OrderByClause{Items: $3.([]*ast.ByItem)}
	}


OptWindowFrameClause:
	/* EMPTY */ {
		$$ = nil
	}
|	WindowFrameUnits WindowFrameExtent OptWindowFrameExclusion
	{
		frame, _ := $3.(*ast.FrameClause)
		if frame == nil {
			frame = &ast.FrameClause{}
		}
		frame.Type, frame.Extent = $1.(ast.FrameType), $2.(ast.FrameExtent)
		frame.TypeOffset = yyS[yypt-2].offset
		$$ = frame
	}


OptWindowFrameExclusion:
	/* EMPTY */ { $$ = nil }
|	exclude current row
	{
		$$ = &ast.FrameClause{Exclusion: "CURRENT ROW", ExclusionOffset: yyS[yypt-2].offset}
	}
|	exclude group
	{
		$$ = &ast.FrameClause{Exclusion: "GROUP", ExclusionOffset: yyS[yypt-1].offset}
	}
|	exclude ties
	{
		$$ = &ast.FrameClause{Exclusion: "TIES", ExclusionOffset: yyS[yypt-1].offset}
	}
|	exclude no others
	{
		$$ = &ast.FrameClause{Exclusion: "NO OTHERS", ExclusionOffset: yyS[yypt-2].offset}
	}


WindowFrameUnits:
	rows   { $$ = ast.FrameType(ast.Rows) }
|	rangeKwd  { $$ = ast.FrameType(ast.Ranges) }
|	groups { $$ = ast.FrameType(ast.Groups) }


WindowFrameExtent:
	WindowFrameStart
	{
		$$ = ast.FrameExtent {
			Start: $1.(ast.FrameBound),
			End: ast.FrameBound{Type: ast.CurrentRow,},
		}
	}
|	WindowFrameBetween
	{
		$$ = $1.(ast.FrameExtent)
	}


WindowFrameStart:
	unbounded preceding
	{
		$$ = ast.FrameBound{Type: ast.Preceding, UnBounded: true,}
	}
|	NumLiteral preceding
	{
		$$ = ast.FrameBound{Type: ast.Preceding, Expr: ast.NewValueExpr($1),}
	}
|	StringLiteral preceding
	{
		$$ = ast.FrameBound{Type: ast.Preceding, Expr: $1, ExprOffset: yyS[yypt-1].offset}
	}
|	paramMarker preceding
	{
		$$ = ast.FrameBound{Type: ast.Preceding, Expr: ast.NewParamMarkerExpr(yyS[yypt].offset),}
	}
|	interval Expression TimeUnit preceding
	{
		$$ = ast.FrameBound{Type: ast.Preceding, Expr: $2, Unit: ast.NewValueExpr($3),}
	}
|	current row
	{
		$$ = ast.FrameBound{Type: ast.CurrentRow,}
	}


WindowFrameBetween:
	between WindowFrameBound and WindowFrameBound
	{
		$$ = ast.FrameExtent{Start: $2.(ast.FrameBound), End: $4.(ast.FrameBound),}
	}


WindowFrameBound:
	WindowFrameStart
	{
		$$ = $1.(ast.FrameBound)
	}
|	unbounded following
	{
		$$ = ast.FrameBound{Type: ast.Following, UnBounded: true,}
	}
|	NumLiteral following
	{
		$$ = ast.FrameBound{Type: ast.Following, Expr: ast.NewValueExpr($1),}
	}
|	StringLiteral following
	{
		$$ = ast.FrameBound{Type: ast.Following, Expr: $1, ExprOffset: yyS[yypt-1].offset}
	}
|	paramMarker following
	{
		$$ = ast.FrameBound{Type: ast.Following, Expr: ast.NewParamMarkerExpr(yyS[yypt].offset),}
	}
|	interval Expression TimeUnit following
	{
		$$ = ast.FrameBound{Type: ast.Following, Expr: $2, Unit: ast.NewValueExpr($3),}
	}


OptWindowingClause:
	/* EMPTY */ { $$ = nil }
|	WindowingClause
	{
		spec := $1.(ast.WindowSpec)
		$$ = &spec
	}


WindowingClause:
	over WindowNameOrSpec { $$ = $2.(ast.WindowSpec) }


WindowNameOrSpec:
	WindowName { $$ = ast.WindowSpec{Name: $1.(model.CIStr), OnlyAlias: true, } }
|	WindowSpec { $$ = $1.(ast.WindowSpec) }


WindowFuncCall:
	rowNumber '(' ')' WindowingClause
	{
		$$ = &ast.WindowFuncExpr{F: $1, Spec: $4.(ast.WindowSpec),}
	}
|	rank '(' ')' WindowingClause
	{
		$$ = &ast.WindowFuncExpr{F: $1, Spec: $4.(ast.WindowSpec),}
	}
|	denseRank '(' ')' WindowingClause
	{
		$$ = &ast.WindowFuncExpr{F: $1, Spec: $4.(ast.WindowSpec),}
	}
|	cumeDist '(' ')' WindowingClause
	{
		$$ = &ast.WindowFuncExpr{F: $1, Spec: $4.(ast.WindowSpec),}
	}
|	percentRank '(' ')' WindowingClause
	{
		$$ = &ast.WindowFuncExpr{F: $1, Spec: $4.(ast.WindowSpec),}
	}
|	ntile '(' Expression ')' WindowingClause
	{
		$$ = &ast.WindowFuncExpr{F: $1, Args: []ast.ExprNode{$3}, Spec: $5.(ast.WindowSpec),}
	}
|	lead '(' Expression OptLeadLagInfo ')' OptNullTreatment WindowingClause
	{
		args := []ast.ExprNode{$3}
		if $4 != nil {
			args = append(args, $4.([]ast.ExprNode)...)
		}
		$$ = &ast.WindowFuncExpr{F: $1, Args: args, IgnoreNull: $6.(bool), Spec: $7.(ast.WindowSpec),}
	}
|	lag '(' Expression OptLeadLagInfo ')' OptNullTreatment WindowingClause
	{
		args := []ast.ExprNode{$3}
		if $4 != nil {
			args = append(args, $4.([]ast.ExprNode)...)
		}
		$$ = &ast.WindowFuncExpr{F: $1, Args: args, IgnoreNull: $6.(bool), Spec: $7.(ast.WindowSpec),}
	}
|	firstValue '(' Expression ')' OptNullTreatment WindowingClause
	{
		$$ = &ast.WindowFuncExpr{F: $1, Args: []ast.ExprNode{$3}, IgnoreNull: $5.(bool), Spec: $6.(ast.WindowSpec),}
	}
|	lastValue '(' Expression ')' LastValueSuffix
	{
		if window, ok := $5.(*lastValueWindow); ok {
			$$ = &ast.WindowFuncExpr{F: $1, Args: []ast.ExprNode{$3}, IgnoreNull: window.ignoreNull, Spec: window.spec}
		} else {
			// MariaDB scalar LAST_VALUE without OVER: evaluates all arguments and returns the last.
			$$ = &ast.FuncCallExpr{FnName: model.NewCIStr($1), Args: []ast.ExprNode{$3}}
		}
	}
|	lastValue '(' Expression ',' ExpressionList ')'
	{
		// MariaDB scalar LAST_VALUE without OVER: evaluates all arguments and returns the last.
		args := append([]ast.ExprNode{$3}, $5.([]ast.ExprNode)...)
		$$ = &ast.FuncCallExpr{FnName: model.NewCIStr($1), Args: args}
	}
|	nthValue '(' Expression ',' Expression ')' OptFromFirstLast OptNullTreatment WindowingClause
	{
		$$ = &ast.WindowFuncExpr{F: $1, Args: []ast.ExprNode{$3, $5}, FromLast: $7.(bool), IgnoreNull: $8.(bool), Spec: $9.(ast.WindowSpec),}
	}

OptLeadLagInfo:
	/* EMPTY */ {
		$$ = nil
	}
|	',' Expression OptLLDefault
	{
		args := []ast.ExprNode{$2}
		if $3 != nil {
			args = append(args, $3.(ast.ExprNode))
		}
		$$ = args
	}


OptLLDefault:
	/* EMPTY */    { $$ = nil }
|	',' Expression { $$ = $2 }


OptNullTreatment:
	/* EMPTY */       { $$ = false }
|	respectNulls { $$ = false }
|	ignoreNulls  { $$ = true }


/* LAST_VALUE is a window function in MySQL and MariaDB and a scalar information
   function in MariaDB. The suffix keeps both spellings in one production so the
   parser stays free of shift/reduce conflicts. */
LastValueSuffix:
	/* EMPTY */                        { $$ = nil }
|	WindowingClause                   { $$ = &lastValueWindow{spec: $1.(ast.WindowSpec)} }
|	respectNulls WindowingClause      { $$ = &lastValueWindow{spec: $2.(ast.WindowSpec)} }
|	ignoreNulls WindowingClause       { $$ = &lastValueWindow{spec: $2.(ast.WindowSpec), ignoreNull: true} }


OptFromFirstLast:
	/* EMPTY */    { $$ = false }
|	from first { $$ = false }
|	from last  { $$ = true }


TableRefsClause:
	TableRefs
	{
		$$ = &ast.TableRefsClause{TableRefs: $1.(*ast.Join)}
	}

TableRefs:
	EscapedTableRef
	{
		if j, ok := $1.(*ast.Join); ok {
			// if $1 is Join, use it directly
			$$ = j
		} else {
			$$ = &ast.Join{Left: $1.(ast.ResultSetNode), Right: nil}
		}
	}
|	TableRefs ',' EscapedTableRef
	{
		/* from a, b is default cross join */
		$$ = &ast.Join{Left: $1.(ast.ResultSetNode), Right: $3.(ast.ResultSetNode), Tp: ast.CrossJoin}
	}

EscapedTableRef:
	TableRef %prec lowerThanSetKeyword
	{
		$$ = $1
	}
|	'{' Identifier TableRef '}'
	{
		/*
		* ODBC escape syntax for outer join is { OJ join_table }
		* Use an Identifier for OJ
		*/
		$$ = $3
	}


TableRef:
	TableFactor { $$ = $1 }
|	JoinTable   { $$ = $1 }


TableFactor:
	TableName PartitionNameListOpt TableAsNameOpt IndexHintListOpt
	{
		tn := $1.(*ast.TableName)
		tn.PartitionNames = $2.([]model.CIStr)
		tn.IndexHints = $4.([]*ast.IndexHint)
		ts := $3.(*ast.TableSource)
		ts.Source = tn
		$$ = ts
	}
|	SubSelect TableAsName WithColNameListOpt
	{
		ts := $2.(*ast.TableSource)
		ts.Source, ts.ColumnNames = $1.(*ast.SubqueryExpr).Query, $3.([]model.CIStr)
		$$ = ts
	}
|	'(' ValuesStmt ')' TableAsName WithColNameListOpt
	{
		ts := $4.(*ast.TableSource)
		ts.Source, ts.ColumnNames = $2.(*ast.ValuesStmt), $5.([]model.CIStr)
		$$ = ts
	}
|	'(' TableRefs ')'
	{
		parser := yyGetParser(yylex)
		if parser.joinGroups == nil {
			parser.joinGroups = make(map[*ast.Join]bool)
		}
		parser.joinGroups[$2.(*ast.Join)] = true
		$$ = $2
	}
/* https://dev.mysql.com/doc/refman/8.0/en/lateral-derived-tables.html */
|	lateral '(' SelectStmt ')' TableAsName
	{
		st := $3.(*ast.SelectStmt)
		endOffset := yyGetParser(yylex).endOffset(&yyS[yypt-1])
		yyGetParser(yylex).setLastSelectFieldText(st, endOffset)
		ts := $5.(*ast.TableSource)
		ts.Source, ts.Lateral = st, true
		$$ = ts
	}
/* https://dev.mysql.com/doc/refman/8.0/en/lateral-derived-tables.html */
|	lateral '(' SetOprStmt ')' TableAsName
	{
		ts := $5.(*ast.TableSource)
		ts.Source, ts.Lateral = $3.(*ast.SetOprStmt), true
		$$ = ts
	}
/* https://dev.mysql.com/doc/refman/8.0/en/json-table-functions.html */
|	JSONTableExpr TableAsName
	{
		ts := $2.(*ast.TableSource)
		ts.Source = $1.(*ast.JSONTable)
		$$ = ts
	}

/* https://dev.mysql.com/doc/refman/8.0/en/json-table-functions.html */
JSONTableExpr:
	jsonTable '(' Expression ',' stringLit columns '(' JSONTableColumnList ')' ')'
	{
		$$ = &ast.JSONTable{Expr: $3, Path: $5, Columns: $8.([]*ast.JSONTableColumn)}
	}

/* https://dev.mysql.com/doc/refman/8.0/en/json-table-functions.html */
JSONTableColumnList:
	JSONTableColumn
	{
		$$ = []*ast.JSONTableColumn{$1.(*ast.JSONTableColumn)}
	}
|	JSONTableColumnList ',' JSONTableColumn
	{
		$$ = append($1.([]*ast.JSONTableColumn), $3.(*ast.JSONTableColumn))
	}

/* https://dev.mysql.com/doc/refman/8.0/en/json-table-functions.html */
/* https://mariadb.com/docs/server/reference/sql-functions/special-functions/json-functions/json_table */
JSONTableColumn:
	Identifier forKwd ordinality
	{
		$$ = &ast.JSONTableColumn{Name: model.NewCIStr($1), Ordinality: true}
	}
|	Identifier Type JSONTablePath JSONTableBehaviors
	{
		column := &ast.JSONTableColumn{Name: model.NewCIStr($1), Tp: $2.(*types.FieldType), Path: $3.(string)}
		ast.AssignJSONTableBehaviors(column, $4.([]*ast.JSONTableBehavior))
		$$ = column
	}
|	Identifier Type exists JSONTablePath JSONTableBehaviors
	{
		column := &ast.JSONTableColumn{Name: model.NewCIStr($1), Tp: $2.(*types.FieldType), Path: $4.(string), Exists: true}
		ast.AssignJSONTableBehaviors(column, $5.([]*ast.JSONTableBehavior))
		$$ = column
	}
|	nested JSONTableNestedPath columns '(' JSONTableColumnList ')'
	{
		$$ = &ast.JSONTableColumn{Path: $2.(string), Nested: true, Columns: $5.([]*ast.JSONTableColumn)}
	}

/* https://dev.mysql.com/doc/refman/8.0/en/json-table-functions.html */
JSONTablePath:
	pathKwd stringLit
	{
		$$ = $2
	}

/* https://dev.mysql.com/doc/refman/8.0/en/json-table-functions.html */
JSONTableNestedPath:
	JSONTablePath
|	stringLit
	{
		$$ = $1
	}

/* https://dev.mysql.com/doc/refman/8.0/en/json-table-functions.html */
JSONTableBehaviors:
	/* EMPTY */ { $$ = []*ast.JSONTableBehavior(nil) }
|	JSONTableOnEmpty { $$ = []*ast.JSONTableBehavior{$1.(*ast.JSONTableBehavior)} }
|	JSONTableOnError { $$ = []*ast.JSONTableBehavior{$1.(*ast.JSONTableBehavior)} }
|	JSONTableOnEmpty JSONTableOnError { $$ = []*ast.JSONTableBehavior{$1.(*ast.JSONTableBehavior), $2.(*ast.JSONTableBehavior)} }
|	JSONTableOnError JSONTableOnEmpty { $$ = []*ast.JSONTableBehavior{$1.(*ast.JSONTableBehavior), $2.(*ast.JSONTableBehavior)} }


/* https://dev.mysql.com/doc/refman/8.0/en/json-table-functions.html */
JSONTableOnEmpty:
	JSONTableBehavior on emptyKwd
	{
		behavior := $1.(*ast.JSONTableBehavior)
		behavior.OnEmpty = true
		$$ = behavior
	}


/* https://dev.mysql.com/doc/refman/8.0/en/json-table-functions.html */
JSONTableOnError:
	JSONTableBehavior on errorKwd { $$ = $1 }


/* https://dev.mysql.com/doc/refman/8.0/en/json-table-functions.html */
/* https://mariadb.com/docs/server/reference/sql-functions/special-functions/json-functions/json_table */
JSONTableBehavior:
	null { $$ = &ast.JSONTableBehavior{Tp: ast.JSONTableBehaviorValue} }
|	errorKwd { $$ = &ast.JSONTableBehavior{Tp: ast.JSONTableBehaviorError} }
|	defaultKwd SignedLiteral { $$ = &ast.JSONTableBehavior{Tp: ast.JSONTableBehaviorDefault, Value: $2} }

PartitionNameListOpt:
    /* EMPTY */                           { $$ = []model.CIStr{} }
|	partition '(' PartitionNameList ')' { $$ = $3 }


TableAsNameOpt:
	/* EMPTY */ { $$ = &ast.TableSource{} }
|	TableAsName { $$ = $1 }


TableAsName:
	Identifier      { $$ = &ast.TableSource{AsName: model.NewCIStr($1)} }
|	as Identifier { $$ = &ast.TableSource{AsName: model.NewCIStr($2)} }


IndexHintType:
	use KeyOrIndex    { $$ = ast.HintUse }
|	ignore KeyOrIndex { $$ = ast.HintIgnore }
|	force KeyOrIndex  { $$ = ast.HintForce }


IndexHintScope:
	/* EMPTY */        { $$ = ast.HintForScan }
|	forKwd join       { $$ = ast.HintForJoin }
|	forKwd order by { $$ = ast.HintForOrderBy }
|	forKwd group by { $$ = ast.HintForGroupBy }


IndexHint:
	IndexHintType IndexHintScope '(' IndexNameList ')'
	{
		$$ = &ast.IndexHint{
			IndexNames:	$4.([]model.CIStr),
			HintType:	$1.(ast.IndexHintType),
			HintScope:	$2.(ast.IndexHintScope),
		}
	}

IndexNameList:
	/* EMPTY */ {
		var nameList []model.CIStr
		$$ = nameList
	}
|	Identifier
	{
		$$ = []model.CIStr{model.NewCIStr($1)}
	}
|	IndexNameList ',' Identifier
	{
		$$ = append($1.([]model.CIStr), model.NewCIStr($3))
	}
|	primary
	{
		$$ = []model.CIStr{model.NewCIStr($1)}
	}

IndexHintList:
	IndexHint
	{
		$$ = []*ast.IndexHint{$1.(*ast.IndexHint)}
	}
|	IndexHintList IndexHint
	{
		$$ = append($1.([]*ast.IndexHint), $2.(*ast.IndexHint))
	}

IndexHintListOpt:
	/* EMPTY */ {
		var hintList []*ast.IndexHint
		$$ = hintList
	}
|	IndexHintList
	{
		$$ = $1
	}

JoinTable:
	/* Use %prec to evaluate production TableRef before cross join */
	TableRef CrossOpt TableRef %prec conditionlessJoin
	{
		$$ = yyGetParser(yylex).addCrossJoin($1.(ast.ResultSetNode), $3.(ast.ResultSetNode))
	}
|	TableRef CrossOpt TableRef on Expression
	{
		on := &ast.OnCondition{Expr: $5}
		$$ = &ast.Join{Left: $1.(ast.ResultSetNode), Right: $3.(ast.ResultSetNode), Tp: ast.CrossJoin, On: on}
	}
|	TableRef CrossOpt TableRef using '(' ColumnNameList ')'
	{
		$$ = &ast.Join{Left: $1.(ast.ResultSetNode), Right: $3.(ast.ResultSetNode), Tp: ast.CrossJoin, Using: $6.([]*ast.ColumnName)}
	}
|	TableRef JoinType OuterOpt join TableRef on Expression
	{
		on := &ast.OnCondition{Expr: $7}
		$$ = &ast.Join{Left: $1.(ast.ResultSetNode), Right: $5.(ast.ResultSetNode), Tp: $2.(ast.JoinType), On: on}
	}
|	TableRef JoinType OuterOpt join TableRef using '(' ColumnNameList ')'
	{
		$$ = &ast.Join{Left: $1.(ast.ResultSetNode), Right: $5.(ast.ResultSetNode), Tp: $2.(ast.JoinType), Using: $8.([]*ast.ColumnName)}
	}
|	TableRef natural join TableRef
	{
		$$ = &ast.Join{Left: $1.(ast.ResultSetNode), Right: $4.(ast.ResultSetNode), NaturalJoin: true}
	}
|	TableRef natural inner join TableRef
	{
		$$ = &ast.Join{Left: $1.(ast.ResultSetNode), Right: $5.(ast.ResultSetNode), NaturalJoin: true}
	}
|	TableRef natural JoinType OuterOpt join TableRef
	{
		$$ = &ast.Join{Left: $1.(ast.ResultSetNode), Right: $6.(ast.ResultSetNode), Tp: $3.(ast.JoinType), NaturalJoin: true}
	}
|	TableRef straightJoin TableRef
	{
		$$ = &ast.Join{Left: $1.(ast.ResultSetNode), Right: $3.(ast.ResultSetNode), StraightJoin: true}
	}
|	TableRef straightJoin TableRef on Expression
	{
		on := &ast.OnCondition{Expr: $5}
		$$ = &ast.Join{Left: $1.(ast.ResultSetNode), Right: $3.(ast.ResultSetNode), StraightJoin: true, On: on}
	}
|	TableRef straightJoin TableRef using '(' ColumnNameList ')'
	{
		$$ = &ast.Join{Left: $1.(ast.ResultSetNode), Right: $3.(ast.ResultSetNode), StraightJoin: true, Using: $6.([]*ast.ColumnName)}
	}

JoinType:
	left
	{
		$$ = ast.LeftJoin
	}
|	right
	{
		$$ = ast.RightJoin
	}

OuterOpt:
	/* EMPTY */ {}
|	outer

CrossOpt:
	join
|	cross join
|	inner join


LimitClause:
	/* EMPTY */ {
		$$ = nil
	}
|	limit LimitOption
	{
		$$ = &ast.Limit{Count: $2.(ast.ValueExpr)}
	}

LimitOption:
	LengthNum
	{
		$$ = ast.NewValueExpr($1)
	}
|	paramMarker
	{
		$$ = ast.NewParamMarkerExpr(yyS[yypt].offset)
	}

SelectStmtLimit:
	/* EMPTY */ {
		$$ = nil
	}
|	limit LimitOption
	{
		$$ = &ast.Limit{Count: $2.(ast.ExprNode)}
	}
|	limit LimitOption ',' LimitOption
	{
		$$ = &ast.Limit{Offset: $2.(ast.ExprNode), Count: $4.(ast.ExprNode)}
	}
|	limit LimitOption offset LimitOption
	{
		$$ = &ast.Limit{Offset: $4.(ast.ExprNode), Count: $2.(ast.ExprNode)}
	}


SelectStmtOpts:
	TableOptimizerHints SelectOptionList %prec empty
	{
		opt := $2.(*selectOptionState).options
		if $1 != nil {
			opt.TableHints = $1.([]*ast.TableOptimizerHint)
		}
		$$ = opt
	}


SelectOptionList:
	SelectOptionListNoCache %prec empty
|	SelectOptionListNoCache sqlCache SelectOptionListNoCache
	{
		state := mergeSelectOptionStates($1.(*selectOptionState), $3.(*selectOptionState))
		state.options.SQLCache = true
		$$ = state
	}
|	SelectOptionListNoCache sqlNoCache SelectOptionListNoCache
	{
		state := mergeSelectOptionStates($1.(*selectOptionState), $3.(*selectOptionState))
		state.options.SQLCache = false
		$$ = state
	}


SelectOptionListNoCache:
	/* EMPTY */ { $$ = newSelectOptionState() }
|	SelectOptionListNoCache SelectOption
	{
		state := $1.(*selectOptionState)
		applySelectOption(state, $2.(string))
		$$ = state
	}


SelectOption:
	all { $$ = "ALL" }
|	DistinctKwd { $$ = "DISTINCT" }
|	lowPriority { $$ = "LOW_PRIORITY" }
|	highPriority { $$ = "HIGH_PRIORITY" }
|	delayed { $$ = "DELAYED" }
|	sqlSmallResult { $$ = "SQL_SMALL_RESULT" }
|	sqlBigResult { $$ = "SQL_BIG_RESULT" }
|	sqlBufferResult { $$ = "SQL_BUFFER_RESULT" }
|	sqlCalcFoundRows { $$ = "SQL_CALC_FOUND_ROWS" }
|	straightJoin { $$ = "STRAIGHT_JOIN" }


TableOptimizerHints:
	/* EMPTY */
	{
		$$ = nil
	}
|	hintBegin TableOptimizerHintList hintEnd
	{
		$$ = $2
	}
|	hintBegin error hintEnd
	{
		Errflag = 0
		yyGetParser(yylex).lastErrorAsWarn()
		$$ = nil
	}


TableOptimizerHintList:
	TableOptimizerHintOpt
	{
		$$ = []*ast.TableOptimizerHint{$1.(*ast.TableOptimizerHint)}
	}
|	TableOptimizerHintList TableOptimizerHintOpt
	{
		$$ = append($1.([]*ast.TableOptimizerHint), $2.(*ast.TableOptimizerHint))
	}


TableOptimizerHintOpt:
	maxExecutionTime '(' NUM ')'
	{
		$$ = &ast.TableOptimizerHint{HintName: model.NewCIStr($1), MaxExecutionTime: getUint64FromNUM($3)}
	}


SelectStmtFieldList:
	FieldList
	{
		$$ = &ast.FieldList{Fields: $1.([]*ast.SelectField)}
	}


SelectStmtGroup:
	/* EMPTY */
	{
		$$ = nil
	}
|	GroupByClause


// See https://dev.mysql.com/doc/refman/5.7/en/subqueries.html
SubSelect:
	'(' SelectStmt ')'
	{
		s := $2.(*ast.SelectStmt)
		endOffset := yyGetParser(yylex).endOffset(&yyS[yypt])
		yyGetParser(yylex).setLastSelectFieldText(s, endOffset)
		src := yyGetParser(yylex).src
		// See the implementation of yyParse function
		s.SetText(src[yyS[yypt-1].offset:yyS[yypt].offset])
		$$ = &ast.SubqueryExpr{Query: s}
	}
|	'(' SetOprStmt ')'
	{
		s := $2.(*ast.SetOprStmt)
		src := yyGetParser(yylex).src
		// See the implementation of yyParse function
		s.SetText(src[yyS[yypt-1].offset:yyS[yypt].offset])
		$$ = &ast.SubqueryExpr{Query: s}
	}
|	'(' WithSelectStmt ')'
	{
		s := $2.(ast.ResultSetNode)
		s.SetText(yyGetParser(yylex).src[yyS[yypt-1].offset:yyS[yypt].offset])
		$$ = &ast.SubqueryExpr{Query: s}
	}
|	'(' SubSelect ')'
	{
		$$ = $2
	}
|	'(' SubSelect ParenQueryTail ')'
	{
		s := parenQueryTail($2.(*ast.SubqueryExpr).Query, $3.(parenQueryTailShape))
		s.SetText(yyGetParser(yylex).src[yyS[yypt-2].offset:yyS[yypt].offset])
		$$ = &ast.SubqueryExpr{Query: s}
	}


// See https://dev.mysql.com/doc/refman/5.7/en/innodb-locking-reads.html
SelectLockOpt:
	/* EMPTY */ %prec empty    { $$ = nil }
|	SelectLockClause

SelectLockClause:
	forKwd update LockedTableListOpt SelectLockWaitOpt
	{
		$$ = &ast.SelectLockInfo{
			Type:       ast.SelectLockForUpdate,
			Tables:     $3.([]*ast.TableName),
			WaitPolicy: $4.(ast.SelectLockWaitPolicy),
		}
	}
|	forKwd share LockedTableListOpt SelectLockWaitOpt
	{
		$$ = &ast.SelectLockInfo{
			Type:       ast.SelectLockForShare,
			Tables:     $3.([]*ast.TableName),
			WaitPolicy: $4.(ast.SelectLockWaitPolicy),
		}
	}
|	lock in share mode { $$ = &ast.SelectLockInfo{Type: ast.SelectLockInShareMode} }


LockedTableListOpt:
	/* EMPTY */
	{
		$$ = ([]*ast.TableName)(nil)
	}
|	of TableNameList
	{
		$$ = $2.([]*ast.TableName)
	}


SelectLockWaitOpt:
	/* EMPTY */
	{
		$$ = ast.SelectLockWaitDefault
	}
|	nowait
	{
		$$ = ast.SelectLockWaitNoWait
	}
|	skip locked
	{
		$$ = ast.SelectLockWaitSkipLocked
	}


// See https://dev.mysql.com/doc/refman/8.0/en/union.html
// See https://mariadb.com/kb/en/intersect/
// See https://mariadb.com/kb/en/except/
ValuesSetOprStmt:
	ValuesSetOprList
	{
		setOpr := $1.(*ast.SetOprStmt)
		if last, ok := setOpr.SelectList.Selects[len(setOpr.SelectList.Selects)-1].(*ast.SelectStmt); ok {
			setOpr.OrderBy = last.OrderBy
			setOpr.Limit = last.Limit
			last.OrderBy = nil
			last.Limit = nil
		}
		normalizeSetOprPrecedence(setOpr)
		$$ = setOpr
	}


ValuesSetOprList:
	ValuesStmt SetOpr ValuesSetOprOperand
	{
		first := $1.(*ast.ValuesStmt)
		next := $3.(ast.Node)
		setNodeSetOpr(next, $2.(*ast.SetOprType))
		setOpr := &ast.SetOprStmt{SelectList: &ast.SetOprSelectList{Selects: []ast.Node{first, next}}}
		$$ = setOpr
	}
|	SetOprClauseList SetOpr ValuesStmt
	{
		setOprList := $1.([]ast.Node)
		if sel, isSelect := setOprList[len(setOprList)-1].(*ast.SelectStmt); isSelect && !sel.IsInBraces {
			endOffset := yyGetParser(yylex).endOffset(&yyS[yypt-1])
			yyGetParser(yylex).setLastSelectFieldText(sel, endOffset)
		}
		next := $3.(*ast.ValuesStmt)
		next.AfterSetOperator = $2.(*ast.SetOprType)
		setOpr := &ast.SetOprStmt{SelectList: &ast.SetOprSelectList{Selects: append(setOprList, next)}}
		$$ = setOpr
	}
|	SetOprClauseList SetOpr ValuesSetOprGroup
	{
		setOprList := $1.([]ast.Node)
		if sel, isSelect := setOprList[len(setOprList)-1].(*ast.SelectStmt); isSelect && !sel.IsInBraces {
			endOffset := yyGetParser(yylex).endOffset(&yyS[yypt-1])
			yyGetParser(yylex).setLastSelectFieldText(sel, endOffset)
		}
		next := $3.(*ast.SetOprSelectList)
		next.AfterSetOperator = $2.(*ast.SetOprType)
		setOpr := &ast.SetOprStmt{SelectList: &ast.SetOprSelectList{Selects: append(setOprList, next)}}
		$$ = setOpr
	}
|	ValuesSetOprGroup SetOpr ValuesSetOprOperand
	{
		first := $1.(*ast.SetOprSelectList)
		next := $3.(ast.Node)
		setNodeSetOpr(next, $2.(*ast.SetOprType))
		setOpr := &ast.SetOprStmt{SelectList: &ast.SetOprSelectList{Selects: []ast.Node{first, next}}}
		$$ = setOpr
	}
|	ValuesSetOprList SetOpr ValuesSetOprOperand
	{
		setOpr := $1.(*ast.SetOprStmt)
		if last, ok := setOpr.SelectList.Selects[len(setOpr.SelectList.Selects)-1].(*ast.SelectStmt); ok && !last.IsInBraces {
			endOffset := yyGetParser(yylex).endOffset(&yyS[yypt-1])
			yyGetParser(yylex).setLastSelectFieldText(last, endOffset)
		}
		next := $3.(ast.Node)
		setNodeSetOpr(next, $2.(*ast.SetOprType))
		setOpr.SelectList.Selects = append(setOpr.SelectList.Selects, next)
		$$ = setOpr
	}


ValuesSetOprOperand:
	SetOprClause
	{
		$$ = $1.([]ast.Node)[0]
	}
|	ValuesStmt
	{
		$$ = $1.(*ast.ValuesStmt)
	}
|	ValuesSetOprGroup
	{
		$$ = $1.(*ast.SetOprSelectList)
	}


ValuesSetOprGroup:
	// Do not add ValuesSetOprStmt to SubSelect: that would make VALUES '('
	// ambiguous with the VALUES(column) expression in general expression contexts.
	'(' ValuesSetOprStmt ')'
	{
		$$ = setOprSelectListFromSetOprStmt($2.(*ast.SetOprStmt))
	}


SetOprStmt:
	SetOprStmtWoutLimitOrderBy
|	SetOprStmtWithLimitOrderBy


SetOprStmtWoutLimitOrderBy:
	SetOprClauseList SetOpr SelectStmt
	{
		setOprList := $1.([]ast.Node)
		if sel, isSelect := setOprList[len(setOprList)-1].(*ast.SelectStmt); isSelect && !sel.IsInBraces {
			endOffset := yyGetParser(yylex).endOffset(&yyS[yypt-1])
			yyGetParser(yylex).setLastSelectFieldText(sel, endOffset)
		}
		setOpr := &ast.SetOprStmt{SelectList: &ast.SetOprSelectList{Selects: setOprList}}
		st := $3.(*ast.SelectStmt)
		st.AfterSetOperator = $2.(*ast.SetOprType)
		// The trailing ORDER BY / LIMIT of the last operand belongs to the
		// whole set operation, so move them up.
		setOpr.OrderBy = st.OrderBy
		setOpr.Limit = st.Limit
		st.OrderBy = nil
		st.Limit = nil
		setOpr.SelectList.Selects = append(setOpr.SelectList.Selects, st)
		normalizeSetOprPrecedence(setOpr)
		$$ = setOpr
	}
|	SetOprClauseList SetOpr SubSelect
	{
		setOprList := $1.([]ast.Node)
		if sel, isSelect := setOprList[len(setOprList)-1].(*ast.SelectStmt); isSelect && !sel.IsInBraces {
			endOffset := yyGetParser(yylex).endOffset(&yyS[yypt-1])
			yyGetParser(yylex).setLastSelectFieldText(sel, endOffset)
		}
		nextSetOprList := setOprSelectListFromSubSelect($3.(*ast.SubqueryExpr))
		nextSetOprList.AfterSetOperator = $2.(*ast.SetOprType)
		setOpr := &ast.SetOprStmt{SelectList: &ast.SetOprSelectList{Selects: append(setOprList, nextSetOprList)}}
		normalizeSetOprPrecedence(setOpr)
		$$ = setOpr
	}


SetOprStmtWithLimitOrderBy:
	SetOprClauseList SetOpr SubSelect ParenQueryTail
	{
		setOprList := $1.([]ast.Node)
		if sel, isSelect := setOprList[len(setOprList)-1].(*ast.SelectStmt); isSelect && !sel.IsInBraces {
			endOffset := yyGetParser(yylex).endOffset(&yyS[yypt-2])
			yyGetParser(yylex).setLastSelectFieldText(sel, endOffset)
		}
		nextSetOprList := setOprSelectListFromSubSelect($3.(*ast.SubqueryExpr))
		nextSetOprList.AfterSetOperator = $2.(*ast.SetOprType)
		setOpr := &ast.SetOprStmt{SelectList: &ast.SetOprSelectList{Selects: append(setOprList, nextSetOprList)}}
		tail := $4.(parenQueryTailShape)
		setOpr.OrderBy = tail.OrderBy
		setOpr.Limit = tail.Limit
		setOpr.Into = tail.Into
		if tail.Lock != nil {
			setOpr.LockTp = tail.Lock.Type
			setOpr.LockedTables = tail.Lock.Tables
			setOpr.LockWaitPolicy = tail.Lock.WaitPolicy
		}
		normalizeSetOprPrecedence(setOpr)
		$$ = setOpr
	}


SetOprClauseList:
	SetOprClause
|	SetOprClauseList SetOpr SetOprClause
	{
		setOprList1 := $1.([]ast.Node)
		setOprList2 := $3.([]ast.Node)
		if sel, isSelect := setOprList1[len(setOprList1)-1].(*ast.SelectStmt); isSelect && !sel.IsInBraces {
			endOffset := yyGetParser(yylex).endOffset(&yyS[yypt-1])
			yyGetParser(yylex).setLastSelectFieldText(sel, endOffset)
		}
		switch x := setOprList2[0].(type) {
		case *ast.SelectStmt:
			x.AfterSetOperator = $2.(*ast.SetOprType)
		case *ast.SetOprSelectList:
			x.AfterSetOperator = $2.(*ast.SetOprType)
		}
		$$ = append(setOprList1, setOprList2...)
	}


SetOprClause:
	SelectStmt
	{
		$$ = []ast.Node{$1.(*ast.SelectStmt)}
	}
|	SubSelect
	{
		$$ = []ast.Node{setOprSelectListFromSubSelect($1.(*ast.SubqueryExpr))}
	}


SetOpr:
	union SetOprOpt
	{
		var tp ast.SetOprType
		tp = ast.Union
		if !$2.(bool) {
			tp = ast.UnionAll
		}
		$$ = &tp
	}
|	except SetOprOpt
	{
		var tp ast.SetOprType
		tp = ast.Except
		if !$2.(bool) {
			tp = ast.ExceptAll
		}
		$$ = &tp
	}
|	intersect SetOprOpt
	{
		var tp ast.SetOprType
		tp = ast.Intersect
		if !$2.(bool) {
			tp = ast.IntersectAll
		}
		$$ = &tp
	}


SetOprOpt:
DefaultTrueDistinctOpt



// SetOprLimit is a non-empty LIMIT clause, used so that a set operation
// operand followed by an optional LIMIT does not introduce a reduce/reduce
// conflict against the operand-without-LIMIT production.
SetOprLimit:
	limit LimitOption
	{
		$$ = &ast.Limit{Count: $2.(ast.ExprNode)}
	}
|	limit LimitOption ',' LimitOption
	{
		$$ = &ast.Limit{Offset: $2.(ast.ExprNode), Count: $4.(ast.ExprNode)}
	}
|	limit LimitOption offset LimitOption
	{
		$$ = &ast.Limit{Offset: $4.(ast.ExprNode), Count: $2.(ast.ExprNode)}
	}


/********************Set Statement*******************************/
SetStmt:
	set VariableAssignmentList                   { $$ = &ast.SetStmt{Variables: $2.([]*ast.VariableAssignment)} }
/* https://dev.mysql.com/doc/refman/8.0/en/set-variable.html */
|	set persist VariableAssignmentList
	{
		vars := $3.([]*ast.VariableAssignment)
		for _, v := range vars {
			v.IsPersist = true
			v.IsSystem = true
		}
		$$ = &ast.SetStmt{Variables: vars}
	}
/* https://dev.mysql.com/doc/refman/8.0/en/set-variable.html */
|	set persistOnly VariableAssignmentList
	{
		vars := $3.([]*ast.VariableAssignment)
		for _, v := range vars {
			v.IsPersistOnly = true
			v.IsSystem = true
		}
		$$ = &ast.SetStmt{Variables: vars}
	}
|	set password eq PasswordOpt                { $$ = &ast.SetPwdStmt{Password: $4.(string)} }
|	set password forKwd Username eq PasswordOpt { $$ = &ast.SetPwdStmt{User: $4.(*auth.UserIdentity), Password: $6.(string)} }
|	set global transaction TransactionChars
	{
		vars := $4.([]*ast.VariableAssignment)
		for _, v := range vars {
			v.IsGlobal = true
		}
		$$ = &ast.SetStmt{Variables: vars}
	}
|	set session transaction TransactionChars
	{
		$$ = &ast.SetStmt{Variables: $4.([]*ast.VariableAssignment)}
	}
|	set transaction TransactionChars
	{
		assigns := $3.([]*ast.VariableAssignment)
		for i:=0; i<len(assigns); i++ {
			if assigns[i].Name == "tx_isolation" {
				// A special session variable that make setting tx_isolation take effect one time.
				assigns[i].Name = "tx_isolation_one_shot"
			}
		}
		$$ = &ast.SetStmt{Variables: assigns}
	}


SetStatementStmt:
	set statement StatementAssignmentList forKwd Statement
	{
		if $5 == nil {
			yyAppendErr(yylex, yyErrorf(yylex, "SET STATEMENT requires a statement after FOR"))
			return 1
		}
		$$ = &ast.SetStatementStmt{Variables: $3.([]*ast.VariableAssignment), Stmt: $5}
	}


StatementAssignment:
	Identifier eq SetExpr
	{
		$$ = &ast.VariableAssignment{Name: $1, Value: $3, IsSystem: true}
	}


StatementAssignmentList:
	StatementAssignment { $$ = []*ast.VariableAssignment{$1.(*ast.VariableAssignment)} }
|	StatementAssignmentList ',' StatementAssignment
	{
		$$ = append($1.([]*ast.VariableAssignment), $3.(*ast.VariableAssignment))
	}


SetRoleStmt:
	set role SetRoleOpt { $$ = $3.(*ast.SetRoleStmt) }


SetDefaultRoleStmt:
	set defaultKwd role SetDefaultRoleOpt to UsernameList {
		tmp := $4.(*ast.SetRoleStmt)
		$$ = &ast.SetDefaultRoleStmt{
			SetRoleOpt: tmp.SetRoleOpt,
			RoleList: tmp.RoleList,
			UserList: $6.([]*auth.UserIdentity),
		}
	}


SetDefaultRoleOpt:
	none       { $$ = &ast.SetRoleStmt{SetRoleOpt: ast.SetRoleNone, RoleList: nil} }
|	all        { $$ = &ast.SetRoleStmt{SetRoleOpt: ast.SetRoleAll, RoleList: nil} }
|	RolenameList { $$ = &ast.SetRoleStmt{SetRoleOpt: ast.SetRoleRegular, RoleList: $1.([]*auth.RoleIdentity)} }


SetRoleOpt:
	all except RolenameList { $$ = &ast.SetRoleStmt{SetRoleOpt: ast.SetRoleAllExcept, RoleList: $3.([]*auth.RoleIdentity)} }
|	SetDefaultRoleOpt           { $$ = $1 }
|	defaultKwd                   { $$ = &ast.SetRoleStmt{SetRoleOpt: ast.SetRoleDefault, RoleList: nil} }


TransactionChars:
	TransactionChar
	{
		if $1 != nil {
			$$ = $1
		} else {
			$$ = []*ast.VariableAssignment{}
		}
	}
|	TransactionChars ',' TransactionChar
	{
		if $3 != nil {
			varAssigns := $3.([]*ast.VariableAssignment)
			$$ = append($1.([]*ast.VariableAssignment), varAssigns...)
		} else {
			$$ = $1
		}
	}


TransactionChar:
	isolation level IsolationLevel
	{
		varAssigns := []*ast.VariableAssignment{}
		expr := ast.NewValueExpr($3)
		varAssigns = append(varAssigns, &ast.VariableAssignment{Name: "tx_isolation", Value: expr, IsSystem: true})
		$$ = varAssigns
	}
|	read write
	{
		varAssigns := []*ast.VariableAssignment{}
		expr := ast.NewValueExpr("0")
		varAssigns = append(varAssigns, &ast.VariableAssignment{Name: "tx_read_only", Value: expr, IsSystem: true})
		$$ = varAssigns
	}
|	read only
	{
		varAssigns := []*ast.VariableAssignment{}
		expr := ast.NewValueExpr("1")
		varAssigns = append(varAssigns, &ast.VariableAssignment{Name: "tx_read_only", Value: expr, IsSystem: true})
		$$ = varAssigns
	}


IsolationLevel:
	repeatable read   { $$ = ast.RepeatableRead }
|	read	committed   { $$ = ast.ReadCommitted }
|	read	uncommitted { $$ = ast.ReadUncommitted }
|	serializable        { $$ = ast.Serializable }


SetExpr:
	on { $$ = ast.NewValueExpr("ON") }
|	ExprOrDefault

SetAssignmentEq:
	eq
|	assignmentEq

VariableAssignment:
	Identifier SetAssignmentEq SetExpr
	{
		if isBareSetScopeKeyword($1) {
			yyAppendErr(yylex, yyErrorf(yylex, "SET %s requires a variable name before '='", $1))
			return 1
		}
		$$ = &ast.VariableAssignment{Name: $1, Value: $3, IsSystem: true}
	}
|	Identifier '.' Identifier SetAssignmentEq SetExpr
	{
		target := &ast.VariableTarget{Parts: []string{$1, $3}}
		$$ = &ast.VariableAssignment{Name: target.LegacyName(), Target: target, Value: $5}
	}
|	global Identifier SetAssignmentEq SetExpr
	{
		$$ = &ast.VariableAssignment{Name: $2, Value: $4, IsGlobal: true, IsSystem: true}
	}
|	session Identifier SetAssignmentEq SetExpr
	{
		$$ = &ast.VariableAssignment{Name: $2, Value: $4, IsSystem: true}
	}
|	local Identifier SetAssignmentEq SetExpr
	{
		$$ = &ast.VariableAssignment{Name: $2, Value: $4, IsSystem: true}
	}
/* https://dev.mysql.com/doc/refman/8.0/en/set-variable.html */
|	doubleAtIdentifier SetAssignmentEq SetExpr
	{
		v := strings.ToLower($1)
		var isGlobal, isPersist, isPersistOnly bool
		if strings.HasPrefix(v, "@@persist_only.") {
			isPersistOnly = true
			v = strings.TrimPrefix(v, "@@persist_only.")
		} else if strings.HasPrefix(v, "@@persist.") {
			isPersist = true
			v = strings.TrimPrefix(v, "@@persist.")
		} else if strings.HasPrefix(v, "@@global.") {
			isGlobal = true
			v = strings.TrimPrefix(v, "@@global.")
		} else if strings.HasPrefix(v, "@@session.") {
			v = strings.TrimPrefix(v, "@@session.")
		} else if strings.HasPrefix(v, "@@local.") {
			v = strings.TrimPrefix(v, "@@local.")
		} else if strings.HasPrefix(v, "@@") {
			v = strings.TrimPrefix(v, "@@")
		}
		$$ = &ast.VariableAssignment{Name: v, Value: $3, IsGlobal: isGlobal, IsPersist: isPersist, IsPersistOnly: isPersistOnly, IsSystem: true}
	}
|	singleAtIdentifier eq Expression
	{
		v := $1
		v = strings.TrimPrefix(v, "@")
		$$ = &ast.VariableAssignment{Name: v, Value: $3}
	}
|	singleAtIdentifier assignmentEq Expression
	{
		v := $1
		v = strings.TrimPrefix(v, "@")
		$$ = &ast.VariableAssignment{Name: v, Value: $3}
	}
|	names CharsetName
	{
		$$ = &ast.VariableAssignment{
			Name: ast.SetNames,
			Value: ast.NewValueExpr($2.(string)),
		}
	}
|	names defaultKwd
	{
		$$ = &ast.VariableAssignment{Name: ast.SetNames, Value: &ast.DefaultExpr{}}
	}
|	names CharsetName collate defaultKwd
	{
		$$ = &ast.VariableAssignment{
			Name: ast.SetNames,
			Value: ast.NewValueExpr($2.(string)),
		}
	}
|	names CharsetName collate StringName
	{
		$$ = &ast.VariableAssignment{
			Name: ast.SetNames,
			Value: ast.NewValueExpr($2.(string)),
			ExtendValue: ast.NewValueExpr($4.(string)),
		}
	}
|	CharsetOrCharacterSet SetCharsetName
	{
		$$ = &ast.VariableAssignment{
			Name: ast.SetCharset,
			Value: ast.NewValueExpr($2.(string)),
		}
	}
|	CharsetOrCharacterSet defaultKwd
	{
		$$ = &ast.VariableAssignment{Name: ast.SetCharset, Value: &ast.DefaultExpr{}}
	}


CharsetName:
	StringName {
		// Validate input charset name to keep the same behavior as parser of MySQL.
		name, _, err := charset.GetCharsetInfo($1.(string))
		if err != nil {
			yyAppendErr(yylex, ErrUnknownCharacterSet.GenWithStackByArgs($1))
			return 1
		}
		// Use charset name returned from charset.GetCharsetInfo(),
		// to keep lower case of input for generated column restore.
		$$ = name
	}
|	binaryType { $$ = charset.CharsetBin }


SetCharsetName:
	StringName
	{
		// This legacy conversion pair is not a charset for DDL or SET NAMES.
		if strings.EqualFold($1.(string), "cp1250_latin2") {
			$$ = "cp1250_latin2"
			break
		}
		name, _, err := charset.GetCharsetInfo($1.(string))
		if err != nil {
			yyAppendErr(yylex, ErrUnknownCharacterSet.GenWithStackByArgs($1))
			return 1
		}
		$$ = name
	}
|	binaryType { $$ = charset.CharsetBin }


CollationName:
	StringName
	{
		info, err := charset.GetCollationByName($1.(string))
		if err != nil {
			yyAppendErr(yylex, err)
			return 1
		}
		$$ = info.Name
	}


VariableAssignmentList:
	/* EMPTY */                                   { $$ = []*ast.VariableAssignment{} }
|	VariableAssignment                            { $$ = []*ast.VariableAssignment{$1.(*ast.VariableAssignment)} }
|	VariableAssignmentList ',' VariableAssignment { $$ = append($1.([]*ast.VariableAssignment), $3.(*ast.VariableAssignment)) }


Variable:
	SystemVariable | UserVariable


SystemVariable:
	doubleAtIdentifier
	{
		v := strings.ToLower($1)
		var isGlobal, isPersist, isPersistOnly bool
		explicitScope := true
		if strings.HasPrefix(v, "@@persist_only.") {
			isPersistOnly = true
			v = strings.TrimPrefix(v, "@@persist_only.")
		} else if strings.HasPrefix(v, "@@persist.") {
			isPersist = true
			v = strings.TrimPrefix(v, "@@persist.")
		} else if strings.HasPrefix(v, "@@global.") {
			isGlobal = true
			v = strings.TrimPrefix(v, "@@global.")
		} else if strings.HasPrefix(v, "@@session.") {
			v = strings.TrimPrefix(v, "@@session.")
		} else if strings.HasPrefix(v, "@@local.") {
			v = strings.TrimPrefix(v, "@@local.")
		} else if strings.HasPrefix(v, "@@") {
			v, explicitScope = strings.TrimPrefix(v, "@@"), false
		}
		$$ = &ast.VariableExpr{Name: v, IsGlobal: isGlobal, IsPersist: isPersist, IsPersistOnly: isPersistOnly, IsSystem: true, ExplicitScope: explicitScope}
	}


UserVariable:
	singleAtIdentifier
	{
		v := $1
		v = strings.TrimPrefix(v, "@")
		$$ = &ast.VariableExpr{Name: v, IsGlobal: false, IsSystem: false}
	}


Username:
	StringName                    { $$ = &auth.UserIdentity{Username: $1.(string), Hostname: "%"} }
|	StringName '@' StringName     { $$ = &auth.UserIdentity{Username: $1.(string), Hostname: $3.(string)} }
|	StringName singleAtIdentifier { $$ = &auth.UserIdentity{Username: $1.(string), Hostname: strings.TrimPrefix($2, "@")} }
|	currentUser OptionalBraces { $$ = &auth.UserIdentity{CurrentUser: true} }


UsernameList:
	Username                  { $$ = []*auth.UserIdentity{$1.(*auth.UserIdentity)} }
|	UsernameList ',' Username { $$ = append($1.([]*auth.UserIdentity), $3.(*auth.UserIdentity)) }


PasswordOpt:
	stringLit                     { $$ = $1 }
|	password '(' AuthString ')' { $$ = $3.(string) }


AuthString:
	stringLit { $$ = $1 }


RoleNameString:
	stringLit  { $$ = $1 }
|	identifier
	{
		if isReservedRoleName(yyGetParser(yylex).src, yyS[yypt].offset, $1) {
			yyAppendErr(yylex, yyErrorf(yylex, "Reserved word '%s' must be quoted to be used as a role name", $1))
			return 1
		}
		$$ = $1
	}


Rolename:
    RoleNameString                { $$ = &auth.RoleIdentity{Username: $1.(string), Hostname: "%"} }
|	StringName '@' StringName     { $$ = &auth.RoleIdentity{Username: $1.(string), Hostname: $3.(string)} }
|	StringName singleAtIdentifier { $$ = &auth.RoleIdentity{Username: $1.(string), Hostname: strings.TrimPrefix($2, "@")} }


RolenameList:
	Rolename                  { $$ = []*auth.RoleIdentity{$1.(*auth.RoleIdentity)} }
|	RolenameList ',' Rolename { $$ = append($1.([]*auth.RoleIdentity), $3.(*auth.RoleIdentity)) }


/****************************Show Statement*******************************/
ShowStmt:
	show ShowTargetFilterable ShowLikeOrWhereOpt
	{
		stmt := $2.(*ast.ShowStmt)
		if $3 != nil {
			if x, ok := $3.(*ast.PatternLikeExpr); ok && x.Expr == nil {
				stmt.Pattern = x
			} else {
				stmt.Where = $3.(ast.ExprNode)
			}
		}
		$$ = stmt
	}
|	show create view TableName
	{
		$$ = &ast.ShowStmt{
			Tp:	ast.ShowCreateView,
			Table:	$4.(*ast.TableName),
		}
	}
|	show create database IfNotExists DBName
	{
		$$ = &ast.ShowStmt{
			Tp:	ast.ShowCreateDatabase,
			IfNotExists: $4.(bool),
			DBName:	$5.(string),
		}
	}
|	show create tableKwd TableName { $$ = &ast.ShowStmt{Tp: ast.ShowCreateTable, Table: $4.(*ast.TableName)} }
|	show create procedure TableName { $$ = &ast.ShowStmt{Tp: ast.ShowCreateProcedure, Table: $4.(*ast.TableName)} }
|	show create function TableName { $$ = &ast.ShowStmt{Tp: ast.ShowCreateFunction, Table: $4.(*ast.TableName)} }
|	show create trigger TableName { $$ = &ast.ShowStmt{Tp: ast.ShowCreateTrigger, Table: $4.(*ast.TableName)} }
|	show create event TableName { $$ = &ast.ShowStmt{Tp: ast.ShowCreateEvent, Table: $4.(*ast.TableName)} }
|	show create user Username   { $$ = &ast.ShowStmt{Tp: ast.ShowCreateUser, User: $4.(*auth.UserIdentity)} }
|	show grants                   { $$ = &ast.ShowStmt{Tp: ast.ShowGrants} }
|	show grants forKwd Username UsingRoles
	{
		if $5 != nil {
			$$ = &ast.ShowStmt{
				Tp:	ast.ShowGrants,
				User:	$4.(*auth.UserIdentity),
				Roles:  $5.([]*auth.RoleIdentity),
			}
		} else {
			$$ = &ast.ShowStmt{
				Tp:	ast.ShowGrants,
				User:	$4.(*auth.UserIdentity),
				Roles:  nil,
			}
		}
	}
|	show master status          { $$ = &ast.ShowStmt{Tp: ast.ShowMasterStatus} }
|	show replica status         { $$ = &ast.ShowStmt{Tp: ast.ShowReplicaStatus} }
|	show replicas               { $$ = &ast.ShowStmt{Tp: ast.ShowReplicas} }
|	show binaryType log status  { $$ = &ast.ShowStmt{Tp: ast.ShowBinaryLogStatus} }
|	show binaryType logs        { $$ = &ast.ShowStmt{Tp: ast.ShowBinaryLogs} }
|	show OptFull processlist      { $$ = &ast.ShowStmt{Tp: ast.ShowProcessList, Full: $2.(bool)} }
|	show profiles                 { $$ = &ast.ShowStmt{Tp: ast.ShowProfiles} }
|	show profile ShowProfileTypesOpt ShowProfileArgsOpt SelectStmtLimit
	{
		v := &ast.ShowStmt{
			Tp: ast.ShowProfile,
		}
		if $3 != nil {
			v.ShowProfileTypes = $3.([]int)
		}
		if $4 != nil {
			v.ShowProfileArgs = $4.(*int64)
		}
		if $5 != nil {
			v.ShowProfileLimit = $5.(*ast.Limit)
		}
		$$ = v
	}
|	show privileges               { $$ = &ast.ShowStmt{Tp: ast.ShowPrivileges} }
|	show warnings SelectStmtLimit
	{
		stmt := &ast.ShowStmt{Tp: ast.ShowWarnings}
		if $3 != nil {
			stmt.Limit = $3.(*ast.Limit)
		}
		$$ = stmt
	}
|	show identSQLErrors SelectStmtLimit
	{
		stmt := &ast.ShowStmt{Tp: ast.ShowErrors}
		if $3 != nil {
			stmt.Limit = $3.(*ast.Limit)
		}
		$$ = stmt
	}


ShowProfileTypesOpt:
	/* EMPTY */      { $$ = nil }
|	ShowProfileTypes { $$ = $1 }


ShowProfileTypes:
	ShowProfileType { $$ = []int{$1.(int)} }
|	ShowProfileTypes ',' ShowProfileType
	{
		l := $1.([]int)
		l = append(l, $3.(int))
		$$ = l
	}


ShowProfileType:
	cpu                { $$ = ast.ProfileTypeCPU }
|	memory             { $$ = ast.ProfileTypeMemory }
|	block io         { $$ = ast.ProfileTypeBlockIo }
|	context switchesSym { $$ = ast.ProfileTypeContextSwitch }
|	pageSym faultsSym      { $$ = ast.ProfileTypePageFaults }
|	ipc                { $$ = ast.ProfileTypeIpc }
|	swaps              { $$ = ast.ProfileTypeSwaps }
|	source             { $$ = ast.ProfileTypeSource }
|	all                { $$ = ast.ProfileTypeAll }


ShowProfileArgsOpt:
	/* EMPTY */ { $$ = nil }
|	forKwd query NUM
	{
		v := $3.(int64)
		$$ = &v
	}


UsingRoles:
	/* EMPTY */          { $$ = nil }
|	using RolenameList { $$ = $2.([]*auth.RoleIdentity) }


ShowIndexKwd:
	index
|	indexes
|	keys


FromOrIn:
from | in


ShowTargetFilterable:
	engines            { $$ = &ast.ShowStmt{Tp: ast.ShowEngines} }
|	databases          { $$ = &ast.ShowStmt{Tp: ast.ShowDatabases} }
|	CharsetOrCharacterSet            { $$ = &ast.ShowStmt{Tp: ast.ShowCharset} }
|	collation          { $$ = &ast.ShowStmt{Tp: ast.ShowCollation} }
|	procedure status { $$ = &ast.ShowStmt{Tp: ast.ShowProcedureStatus} }
|	OptFull tables ShowDatabaseNameOpt
	{
		$$ = &ast.ShowStmt{
			Tp:	ast.ShowTables,
			DBName:	$3.(string),
			Full:	$1.(bool),
		}
	}
|	open tables ShowDatabaseNameOpt
	{
		$$ = &ast.ShowStmt{
			Tp:	ast.ShowOpenTables,
			DBName:	$3.(string),
		}
	}
|	tableKwd status ShowDatabaseNameOpt
	{
		$$ = &ast.ShowStmt{
			Tp:	ast.ShowTableStatus,
			DBName:	$3.(string),
		}
	}
|	ShowIndexKwd FromOrIn TableName
	{
        $$ = &ast.ShowStmt{
            Tp: ast.ShowIndex,
            Table: $3.(*ast.TableName),
		}
	}
|	ShowIndexKwd FromOrIn Identifier FromOrIn Identifier
	{
        show := &ast.ShowStmt{
            Tp: ast.ShowIndex,
            Table: &ast.TableName{Name:model.NewCIStr($3), Schema: model.NewCIStr($5)},
		}
        $$ = show
	}
|	OptFull columns ShowTableAliasOpt ShowDatabaseNameOpt
	{
		$$ = &ast.ShowStmt{
			Tp:     ast.ShowColumns,
			Table:	$3.(*ast.TableName),
			DBName:	$4.(string),
			Full:	$1.(bool),
		}
	}
|	OptFull fields ShowTableAliasOpt ShowDatabaseNameOpt
	{
		// SHOW FIELDS is a synonym for SHOW COLUMNS.
		$$ = &ast.ShowStmt{
			Tp:     ast.ShowColumns,
			Table:	$3.(*ast.TableName),
			DBName:	$4.(string),
			Full:	$1.(bool),
		}
	}
|	GlobalScope variables
	{
		$$ = &ast.ShowStmt{
			Tp: ast.ShowVariables,
			GlobalScope: $1.(bool),
		}
	}
|	GlobalScope status
	{
		$$ = &ast.ShowStmt{
			Tp: ast.ShowStatus,
			GlobalScope: $1.(bool),
		}
	}
|	triggers ShowDatabaseNameOpt
	{
		$$ = &ast.ShowStmt{
			Tp:	ast.ShowTriggers,
			DBName:	$2.(string),
		}
	}
|	function status
	{
		// This statement is similar to SHOW PROCEDURE STATUS but for stored functions.
		// See http://dev.mysql.com/doc/refman/5.7/en/show-function-status.html
		// We do not support neither stored functions nor stored procedures.
		// So we reuse show procedure status process logic.
		$$ = &ast.ShowStmt {
			Tp: ast.ShowProcedureStatus,
		}
	}
|	events ShowDatabaseNameOpt
	{
		$$ = &ast.ShowStmt{
			Tp:	ast.ShowEvents,
			DBName:	$2.(string),
		}
	}
|	plugins
	{
		$$ = &ast.ShowStmt{
			Tp: 	ast.ShowPlugins,
		}
	}


ShowLikeOrWhereOpt:
	/* EMPTY */        { $$ = nil }
|	like SimpleExpr  { $$ = &ast.PatternLikeExpr{Pattern: $2, Escape: '\\'} }
|	where Expression { $$ = $2 }


GlobalScope:
	/* EMPTY */ { $$ = false }
|	global    { $$ = true }
|	session   { $$ = false }
|	local     { $$ = false }


OptFull:
	/* EMPTY */ { $$ = false }
|	full      { $$ = true }


ShowDatabaseNameOpt:
	/* EMPTY */     { $$ = "" }
|	FromOrIn DBName { $$ = $2.(string) }


ShowTableAliasOpt:
	FromOrIn TableName { $$ = $2.(*ast.TableName) }


// https://dev.mysql.com/doc/refman/5.7/en/flush.html
FlushStmt:
	flush NoWriteToBinLogAliasOpt FlushOption
	{
		tmp := $3.(*ast.FlushStmt)
		tmp.NoWriteToBinLog = $2.(bool)
		$$ = tmp
	}

FlushOption:
	privileges
	{
		$$ = &ast.FlushStmt{
			Tp: ast.FlushPrivileges,
		}
	}
|	status
	{
		$$ = &ast.FlushStmt{
			Tp: ast.FlushStatus,
		}
	}
|	TableOrTables TableNameListOpt WithReadLockOpt
	{
		$$ = &ast.FlushStmt{
			Tp: ast.FlushTables,
			Tables: $2.([]*ast.TableName),
			ReadLock: $3.(bool),
		}
	}


NoWriteToBinLogAliasOpt:
	/* EMPTY */          { $$ = false }
|	noWriteToBinLog { $$ = true }
|	local              { $$ = true }


TableNameListOpt:
	%prec empty
	{
		$$ = []*ast.TableName{}
	}
|	TableNameList
	{
		$$ = $1
	}


WithReadLockOpt:
	/* EMPTY */          { $$ = false }
|	with read lock { $$ = true }


SpStmtOpt:
	/* EMPTY */
	{
		$$ = (ast.StmtNode)(nil)
	}
|	SpStmt
	{
		$$ = $1
	}


SpStmt:
	SpBeginStmt
|	SpIfStmt
|	SpCaseStmt
|	SpWhileStmt
|	SpRepeatStmt
|	SpLoopStmt
|	SpForStmt
|	SpLeaveStmt
|	SpIterateStmt
|	SpDeclareStmt
|	SpOpenStmt
|	SpCloseStmt
|	SpFetchStmt
|	SpSignalStmt
|	SpResignalStmt
|	SpStartTransactionStmt
|	AlterTableStmt
|	AlterDatabaseStmt
|	AlterUserStmt
|	AlterFunctionStmt
|	AlterProcedureStmt
|	AlterEventStmt
|	AlterViewStmt
|	AnalyzeTableStmt
|	BinlogStmt
|	CommitStmt
|	CallStmt
|	DeallocateStmt
|	DeleteFromStmt
|	ExecuteStmt
|	ExplainStmt
|	CreateDatabaseStmt
|	CreateIndexStmt
|	CreateTableStmt
|	CreateFunctionStmt
|	CreateProcedureStmt
|	CreateTriggerStmt
|	CreateEventStmt
|	CreateViewStmt
|	CreateUserStmt
|	CreateRoleStmt
|	DoStmt
|	DropDatabaseStmt
|	DropIndexStmt
|	DropTableStmt
|	DropViewStmt
|	DropFunctionStmt
|	DropProcedureStmt
|	DropTriggerStmt
|	DropEventStmt
|	DropUserStmt
|	DropRoleStmt
|	FlushStmt
|	GrantStmt
|	GrantRoleStmt
|	InsertIntoStmt
|	KillStmt
|	LoadDataStmt
|	ChangeMasterStmt
|	ReplicaControlStmt
|	ResetStmt
|	ShutdownStmt
|	HandlerStmt
|	CacheIndexStmt
|	LoadIndexStmt
|	LoadXMLStmt
|	RenameUserStmt
|	ImportTableStmt
|	InstallPluginStmt
|	UninstallPluginStmt
|	InstallComponentStmt
|	UninstallComponentStmt
|	CloneStmt
|	GetDiagnosticsStmt
|	PreparedStmt
|	RollbackStmt
|	SavepointStmt
|	ReleaseSavepointStmt
|	ValuesStmt
|	ValuesSetOprStmt
|	TableStmt
|	WithSelectStmt
|	WithUpdateStmt
|	WithDeleteStmt
|	CreateSequenceStmt
|	DropSequenceStmt
|	XAStmt
|	RenameTableStmt
|	ReplaceIntoStmt
|	RevokeStmt
|	RevokeRoleStmt
|	SelectStmt
|	SetOprStmt
|	SetStmt
|	SetStatementStmt
|	SetRoleStmt
|	SetDefaultRoleStmt
|	ShowStmt
|	TruncateTableStmt
|	UpdateStmt
|	UseStmt
|	UnlockTablesStmt
|	LockTablesStmt
|	ReturnStmt
|	PurgeStmt


SpStmtListOpt:
	/* EMPTY */
	{
		$$ = []ast.StmtNode{}
	}
|	SpStmtList
	{
		$$ = $1
	}


SpStmtList:
	SpStmt ';'
	{
		$$ = []ast.StmtNode{$1.(ast.StmtNode)}
	}
|	SpStmtList SpStmt ';'
	{
		$$ = append($1.([]ast.StmtNode), $2.(ast.StmtNode))
	}


SpLabelPrefix:
	/* EMPTY */
	{
		$$ = ""
	}
|	SpLabelName ':'
	{
		$$ = $1
	}


SpLabelName:
	identifier { $$ = $1 }
|	account { $$ = $1 }
|	always { $$ = $1 }
|	backup { $$ = $1 }
|	close { $$ = $1 }
|	format { $$ = $1 }
|	invisible { $$ = $1 }
|	open { $$ = $1 }
|	options { $$ = $1 }
|	remove { $$ = $1 }
|	role { $$ = $1 }
|	query { $$ = $1 }
|	names { $$ = $1 }
|	password { $$ = $1 }
|	security { $$ = $1 }
|	server { $$ = $1 }
|	soname { $$ = $1 }
|	upgrade { $$ = $1 }
|	visible { $$ = $1 }
|	wrapper { $$ = $1 }


SpLabelSuffix:
	/* EMPTY */
	{
		$$ = ""
	}
|	SpLabelName
	{
		$$ = $1
	}


SpAtomicOpt:
	/* EMPTY */
	{
		$$ = false
	}
|	not atomic
	{
		$$ = true
	}


SpBeginStmt:
	SpLabelPrefix begin SpAtomicOpt SpStmtListOpt end SpLabelSuffix
	{
		if $6 != "" && $1 != "" && $6 != $1 {
			yyAppendErr(yylex, yyErrorf(yylex, "the end label '%s' does not match the begin label '%s'", $6, $1))
			return 1
		}
		x := &ast.BeginEndStmt{NotAtomic: $3.(bool), List: $4.([]ast.StmtNode)}
		if $1 != "" {
			x.Label = $1.(string)
		}
		$$ = x
	}


SpIfStmt:
	ifKwd Expression then SpStmtListOpt SpElseIfClauseList SpElseOpt end ifKwd
	{
		conditions := make([]*ast.IfClause, 0, 1+len($5.([]*ast.IfClause)))
		conditions = append(conditions, &ast.IfClause{Condition: $2.(ast.ExprNode), List: $4.([]ast.StmtNode)})
		conditions = append(conditions, $5.([]*ast.IfClause)...)
		$$ = &ast.IfStmt{Conditions: conditions, ElseList: $6.([]ast.StmtNode)}
	}


SpElseIfClauseList:
	/* EMPTY */
	{
		$$ = []*ast.IfClause(nil)
	}
|	SpElseIfClauseList elseif Expression then SpStmtListOpt
	{
		$$ = append($1.([]*ast.IfClause), &ast.IfClause{Condition: $3.(ast.ExprNode), List: $5.([]ast.StmtNode)})
	}


SpElseOpt:
	/* EMPTY */
	{
		$$ = []ast.StmtNode(nil)
	}
|	elseKwd SpStmtListOpt
	{
		$$ = $2
	}


SpCaseStmt:
	caseKwd ExpressionOpt SpWhenClauseList SpCaseElseOpt end caseKwd
	{
		x := &ast.CaseStmt{
			Whens:    $3.([]*ast.CaseWhenStmt),
			ElseList: $4.([]ast.StmtNode),
		}
		if v, ok := $2.(ast.ExprNode); ok {
			x.Value = v
		}
		$$ = x
	}


SpWhenClauseList:
	SpWhenClause
	{
		$$ = []*ast.CaseWhenStmt{$1.(*ast.CaseWhenStmt)}
	}
|	SpWhenClauseList SpWhenClause
	{
		$$ = append($1.([]*ast.CaseWhenStmt), $2.(*ast.CaseWhenStmt))
	}


SpWhenClause:
	when Expression then SpStmtListOpt
	{
		$$ = &ast.CaseWhenStmt{Expr: $2.(ast.ExprNode), List: $4.([]ast.StmtNode)}
	}


SpCaseElseOpt:
	/* EMPTY */
	{
		$$ = []ast.StmtNode(nil)
	}
|	elseKwd SpStmtListOpt
	{
		$$ = $2
	}


SpWhileStmt:
	SpLabelPrefix while Expression do SpStmtListOpt end while SpLabelSuffix
	{
		if $8 != "" && $1 != "" && $8 != $1 {
			yyAppendErr(yylex, yyErrorf(yylex, "the end label '%s' does not match the begin label '%s'", $8, $1))
			return 1
		}
		$$ = &ast.WhileStmt{Label: $1.(string), Condition: $3.(ast.ExprNode), List: $5.([]ast.StmtNode)}
	}


SpRepeatStmt:
	SpLabelPrefix repeat SpStmtListOpt until Expression end repeat SpLabelSuffix
	{
		if $8 != "" && $1 != "" && $8 != $1 {
			yyAppendErr(yylex, yyErrorf(yylex, "the end label '%s' does not match the begin label '%s'", $8, $1))
			return 1
		}
		$$ = &ast.RepeatStmt{Label: $1.(string), Condition: $5.(ast.ExprNode), List: $3.([]ast.StmtNode)}
	}


SpLoopStmt:
	SpLabelPrefix loop SpStmtListOpt end loop SpLabelSuffix
	{
		if $6 != "" && $1 != "" && $6 != $1 {
			yyAppendErr(yylex, yyErrorf(yylex, "the end label '%s' does not match the begin label '%s'", $6, $1))
			return 1
		}
		$$ = &ast.LoopStmt{Label: $1.(string), List: $3.([]ast.StmtNode)}
	}


SpForStmt:
	SpLabelPrefix SpForBody SpLabelSuffix
	{
		if $3 != "" && $1 != "" && $3 != $1 {
			yyAppendErr(yylex, yyErrorf(yylex, "the end label '%s' does not match the begin label '%s'", $3, $1))
			return 1
		}
		stmt := $2.(*ast.ForStmt)
		stmt.Label = $1.(string)
		$$ = stmt
	}


TopLevelForStmt:
	SpForBody


SpForBody:
	forKwd identifier in Expression rangeDots Expression do SpStmtListOpt end forKwd
	{
		$$ = &ast.ForStmt{
			Variable:   $2,
			LowerBound: $4,
			UpperBound: $6,
			List:       $8.([]ast.StmtNode),
		}
	}
|	forKwd identifier in '(' SelectStmt ')' do SpStmtListOpt end forKwd
	{
		$$ = &ast.ForStmt{
			Variable: $2,
			Query:    $5.(ast.StmtNode),
			List:     $8.([]ast.StmtNode),
		}
	}


SpLeaveStmt:
	leave SpLabelName
	{
		$$ = &ast.LeaveStmt{Label: $2.(string)}
	}


SpIterateStmt:
	iterate SpLabelName
	{
		$$ = &ast.IterateStmt{Label: $2.(string)}
	}


SpOpenStmt:
	open identifier
	{
		$$ = &ast.OpenStmt{Name: $2}
	}


SpCloseStmt:
	close identifier
	{
		$$ = &ast.CloseStmt{Name: $2}
	}


SpFetchStmt:
	fetch Identifier into SpLocalNameList
	{
		targets := $4.([]*ast.VariableTarget)
		$$ = &ast.FetchStmt{Name: $2, Vars: variableTargetNames(targets), Targets: targets}
	}
|	fetch from Identifier into SpLocalNameList
	{
		targets := $5.([]*ast.VariableTarget)
		$$ = &ast.FetchStmt{Name: $3, Vars: variableTargetNames(targets), Targets: targets}
	}
|	fetch next from Identifier into SpLocalNameList
	{
		targets := $6.([]*ast.VariableTarget)
		$$ = &ast.FetchStmt{Name: $4, Vars: variableTargetNames(targets), Targets: targets}
	}


SpNameList:
	SpName
	{
		$$ = []*ast.VariableTarget{$1.(*ast.VariableTarget)}
	}
|	SpNameList ',' SpName
	{
		$$ = append($1.([]*ast.VariableTarget), $3.(*ast.VariableTarget))
	}


SpName:
	SpLocalName
|	singleAtIdentifier
	{
		$$ = &ast.VariableTarget{Parts: []string{$1}, IsUser: true}
	}


SpLocalName:
	Identifier
	{
		$$ = &ast.VariableTarget{Parts: []string{$1}}
	}
|	Identifier '.' Identifier
	{
		$$ = &ast.VariableTarget{Parts: []string{$1, $3}}
	}


SpLocalNameList:
	Identifier
	{
		$$ = []*ast.VariableTarget{{Parts: []string{$1}}}
	}
|	SpLocalNameList ',' Identifier
	{
		$$ = append($1.([]*ast.VariableTarget), &ast.VariableTarget{Parts: []string{$3}})
	}


SelectIntoOpt:
	/* EMPTY */ %prec empty
	{
		$$ = nil
	}
|	SelectIntoClause

SelectIntoClause:
	into SpNameList
	{
		targets := $2.([]*ast.VariableTarget)
		$$ = &ast.SelectIntoStatement{Tp: ast.SelectIntoVarList, VarList: variableTargetNames(targets), Targets: targets}
	}
|	into outfile stringLit SelectIntoOutfileOpts
	{
		st := $4.(*ast.SelectIntoStatement)
		st.FileName = $3
		$$ = st
	}
|	into dumpfile stringLit
	{
		$$ = &ast.SelectIntoStatement{Tp: ast.SelectIntoDumpfile, FileName: $3}
	}


SelectIntoOutfileOpts:
	CharsetOpt Fields Lines
	{
		$$ = &ast.SelectIntoStatement{
			Tp:      ast.SelectIntoOutfile,
			Charset: $1.(string),
			Fields:  $2.(*ast.FieldsClause),
			Lines:   $3.(*ast.LinesClause),
		}
	}


SpDeclareStmt:
	declare SpVarNameList Type SpDeclareDefaultOpt
	{
		st := &ast.DeclareStmt{Names: $2.([]string), Type: $3.(*types.FieldType)}
		if $4 != nil {
			st.Default = $4.(ast.ExprNode)
		}
		$$ = st
	}
|	declare SpVarNameList SpRowType SpDeclareDefaultOpt
	{
		st := &ast.DeclareStmt{Names: $2.([]string), RowType: $3.([]*ast.ColumnDef)}
		if $4 != nil {
			st.Default = $4.(ast.ExprNode)
		}
		$$ = st
	}
|	declare SpVarNameList typeSym of ColumnName SpDeclareDefaultOpt
	{
		st := &ast.DeclareStmt{Names: $2.([]string), TypeRef: $5.(*ast.ColumnName)}
		if $6 != nil {
			st.Default = $6.(ast.ExprNode)
		}
		$$ = st
	}
|	declare SpVarNameList row typeSym of ColumnName SpDeclareDefaultOpt
	{
		st := &ast.DeclareStmt{Names: $2.([]string), TypeRef: $6.(*ast.ColumnName), RowTypeRef: true}
		if $7 != nil {
			st.Default = $7.(ast.ExprNode)
		}
		$$ = st
	}
|	declare identifier condition forKwd SpConditionValue
	{
		$$ = &ast.DeclareConditionStmt{Name: $2, Condition: $5.(*ast.HandlerCondition)}
	}
|	declare identifier cursor forKwd SelectStmt
	{
		$$ = &ast.DeclareCursorStmt{Name: $2, Select: $5.(ast.StmtNode)}
	}
|	declare identifier cursor forKwd ProcedureAnalyseStmt
	{
		$$ = &ast.DeclareCursorStmt{Name: $2, Select: $5.(ast.StmtNode)}
	}
|	declare SpHandlerAction handler forKwd SpConditionValueList SpStmt
	{
		$$ = &ast.DeclareHandlerStmt{Action: $2.(ast.HandlerType), Conditions: $5.([]*ast.HandlerCondition), Stmt: $6.(ast.StmtNode)}
	}


SpVarNameList:
	SpVarName
	{
		$$ = []string{$1.(string)}
	}
|	SpVarNameList ',' SpVarName
	{
		$$ = append($1.([]string), $3.(string))
	}


SpVarName:
	identifier { $$ = $1 }
|	query { $$ = $1 }
|	names { $$ = $1 }
|	role { $$ = $1 }
|	password { $$ = $1 }
|	count { $$ = $1 }


SpRowType:
	row '(' SpRowFieldList ')'
	{
		$$ = $3
	}


SpRowFieldList:
	SpRowField
	{
		$$ = []*ast.ColumnDef{$1.(*ast.ColumnDef)}
	}
|	SpRowFieldList ',' SpRowField
	{
		$$ = append($1.([]*ast.ColumnDef), $3.(*ast.ColumnDef))
	}


SpRowField:
	Identifier Type
	{
		$$ = &ast.ColumnDef{
			Name: &ast.ColumnName{Name: model.NewCIStr($1)},
			Tp:   $2.(*types.FieldType),
		}
	}


SpDeclareDefaultOpt:
	/* EMPTY */
	{
		$$ = (ast.ExprNode)(nil)
	}
|	defaultKwd Expression
	{
		$$ = $2
	}


SpHandlerAction:
	continueKwd
	{
		$$ = ast.HandlerContinue
	}
|	exit
	{
		$$ = ast.HandlerExit
	}
|	undo
	{
		$$ = ast.HandlerUndo
	}


SpConditionValueList:
	SpConditionValue
	{
		$$ = []*ast.HandlerCondition{$1.(*ast.HandlerCondition)}
	}
|	SpConditionValueList ',' SpConditionValue
	{
		$$ = append($1.([]*ast.HandlerCondition), $3.(*ast.HandlerCondition))
	}


SpConditionValue:
	sqlState value stringLit
	{
		$$ = &ast.HandlerCondition{Type: ast.HandlerConditionSQLState, Value: $3}
	}
|	sqlState stringLit
	{
		$$ = &ast.HandlerCondition{Type: ast.HandlerConditionSQLState, Value: $2}
	}
|	intLit
	{
		$$ = ast.NewErrnoHandlerCondition($1)
	}
|	identifier
	{
		$$ = &ast.HandlerCondition{Type: ast.HandlerConditionName, Value: $1}
	}
|	sqlWarning
	{
		$$ = &ast.HandlerCondition{Type: ast.HandlerConditionSQLWarning}
	}
|	not found
	{
		$$ = &ast.HandlerCondition{Type: ast.HandlerConditionNotFound}
	}
|	sqlException
	{
		$$ = &ast.HandlerCondition{Type: ast.HandlerConditionSQLException}
	}


SpConditionValueOpt:
	/* EMPTY */
	{
		$$ = (*ast.HandlerCondition)(nil)
	}
|	SpConditionValue
	{
		$$ = $1
	}


SpSignalInfoItemListOpt:
	/* EMPTY */
	{
		$$ = []*ast.SignalInfoItem(nil)
	}
|	set SpSignalInfoItemList
	{
		$$ = $2
	}


SpSignalInfoItemList:
	SpSignalInfoItem
	{
		$$ = []*ast.SignalInfoItem{$1.(*ast.SignalInfoItem)}
	}
|	SpSignalInfoItemList ',' SpSignalInfoItem
	{
		$$ = append($1.([]*ast.SignalInfoItem), $3.(*ast.SignalInfoItem))
	}


SpSignalInfoItem:
	identifier eq Expression
	{
		$$ = &ast.SignalInfoItem{Name: $1, Value: $3.(ast.ExprNode)}
	}


SpSignalStmt:
	signal SpConditionValue SpSignalInfoItemListOpt
	{
		$$ = &ast.SignalStmt{Condition: $2.(*ast.HandlerCondition), InfoItems: $3.([]*ast.SignalInfoItem)}
	}


SpResignalStmt:
	resignal SpConditionValueOpt SpSignalInfoItemListOpt
	{
		$$ = &ast.ResignalStmt{Condition: $2.(*ast.HandlerCondition), InfoItems: $3.([]*ast.SignalInfoItem)}
	}


SpStartTransactionStmt:
	start transaction
	{
		$$ = &ast.BeginStmt{}
	}
|	start transaction with consistent snapshot
	{
		$$ = &ast.BeginStmt{}
	}


// A nonempty tail avoids ambiguity with ordinary parenthesized expressions.
ParenQueryTail:
	OrderBy SelectStmtLimit SelectLockOpt SelectIntoOpt
	{
		limit, _ := $2.(*ast.Limit)
		tail := parenQueryTailShape{OrderBy: $1.(*ast.OrderByClause), Limit: limit}
		if $3 != nil {
			tail.Lock = $3.(*ast.SelectLockInfo)
		}
		if $4 != nil {
			tail.Into = $4.(*ast.SelectIntoStatement)
		}
		$$ = tail
	}
|	OrderBy SelectStmtLimit SelectIntoClause SelectLockOpt
	{
		limit, _ := $2.(*ast.Limit)
		tail := parenQueryTailShape{OrderBy: $1.(*ast.OrderByClause), Limit: limit, Into: $3.(*ast.SelectIntoStatement)}
		if $4 != nil {
			tail.Lock = $4.(*ast.SelectLockInfo)
		}
		$$ = tail
	}
|	SetOprLimit SelectLockOpt SelectIntoOpt
	{
		tail := parenQueryTailShape{Limit: $1.(*ast.Limit)}
		if $2 != nil {
			tail.Lock = $2.(*ast.SelectLockInfo)
		}
		if $3 != nil {
			tail.Into = $3.(*ast.SelectIntoStatement)
		}
		$$ = tail
	}
|	SetOprLimit SelectIntoClause SelectLockOpt
	{
		tail := parenQueryTailShape{Limit: $1.(*ast.Limit), Into: $2.(*ast.SelectIntoStatement)}
		if $3 != nil {
			tail.Lock = $3.(*ast.SelectLockInfo)
		}
		$$ = tail
	}
|	SelectLockClause SelectIntoOpt
	{
		tail := parenQueryTailShape{Lock: $1.(*ast.SelectLockInfo)}
		if $2 != nil {
			tail.Into = $2.(*ast.SelectIntoStatement)
		}
		$$ = tail
	}
|	SelectIntoClause SelectLockOpt
	{
		tail := parenQueryTailShape{Into: $1.(*ast.SelectIntoStatement)}
		if $2 != nil {
			tail.Lock = $2.(*ast.SelectLockInfo)
		}
		$$ = tail
	}


ParenthesizedQueryStmt:
	SubSelect %prec '('
	{
		$$ = $1.(*ast.SubqueryExpr).Query.(ast.StmtNode)
	}
|	SubSelect ParenQueryTail %prec '('
	{
		$$ = parenQueryTail($1.(*ast.SubqueryExpr).Query, $2.(parenQueryTailShape)).(ast.StmtNode)
	}


ProcedureAnalyseStmt:
	SelectStmt ProcedureAnalyseClause SelectLockOpt
	{
		stmt := $1.(ast.StmtNode)
		if err := attachProcedureAnalyse(stmt, $2.(*ast.ProcedureAnalyseClause), $3); err != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "%v", err))
			return 1
		}
		$$ = stmt
	}
|	SetOprStmt ProcedureAnalyseClause SelectLockOpt
	{
		stmt := $1.(ast.StmtNode)
		if err := attachProcedureAnalyse(stmt, $2.(*ast.ProcedureAnalyseClause), $3); err != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "%v", err))
			return 1
		}
		$$ = stmt
	}
|	ParenthesizedQueryStmt ProcedureAnalyseClause SelectLockOpt
	{
		stmt := $1.(ast.StmtNode)
		if err := attachProcedureAnalyse(stmt, $2.(*ast.ProcedureAnalyseClause), $3); err != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "%v", err))
			return 1
		}
		$$ = stmt
	}


Statement:
	EmptyStmt
|	AlterTableStmt
|	AlterDatabaseStmt
|	AlterUserStmt
|	AlterFunctionStmt
|	AlterProcedureStmt
|	AlterEventStmt
|	AlterViewStmt
|	AnalyzeTableStmt
|	BackupStmt
|	BeginTransactionStmt
|	BinlogStmt
|	CommitStmt
|	CallStmt
|	DeallocateStmt
|	DeleteFromStmt
|	ExecuteStmt
|	ExplainStmt
|	CreateDatabaseStmt
|	CreateIndexStmt
|	CreateTableStmt
|	CreateFunctionStmt
|	CreateProcedureStmt
|	CreateTriggerStmt
|	CreateEventStmt
|	CreateViewStmt
|	CreateUserStmt
|	CreateRoleStmt
|	DoStmt
|	DropDatabaseStmt
|	DropIndexStmt
|	DropTableStmt
|	DropViewStmt
|	DropFunctionStmt
|	DropProcedureStmt
|	DropTriggerStmt
|	DropEventStmt
|	DropUserStmt
|	DropRoleStmt
|	FlushStmt
|	GrantStmt
|	GrantRoleStmt
|	InsertIntoStmt
|	KillStmt
|	LoadDataStmt
|	ChangeMasterStmt
|	ReplicaControlStmt
|	ResetStmt
|	ShutdownStmt
|	HandlerStmt
|	CacheIndexStmt
|	LoadIndexStmt
|	LoadXMLStmt
|	RenameUserStmt
|	ImportTableStmt
|	InstallPluginStmt
|	UninstallPluginStmt
|	InstallComponentStmt
|	UninstallComponentStmt
|	CloneStmt
|	GetDiagnosticsStmt
|	PreparedStmt
|	RollbackStmt
|	SavepointStmt
|	ReleaseSavepointStmt
|	ValuesStmt
|	ValuesSetOprStmt
|	TableStmt
|	WithSelectStmt
|	WithUpdateStmt
|	WithDeleteStmt
|	CreateSequenceStmt
|	DropSequenceStmt
|	XAStmt
|	RenameTableStmt
|	ReplaceIntoStmt
|	RevokeStmt
|	RevokeRoleStmt
|	SelectStmt
|	SetOprStmt
|	SetStmt
|	SetStatementStmt
|	SetRoleStmt
|	SetDefaultRoleStmt
|	ShowStmt
|	CreateTablespaceStmt
|	AlterTablespaceStmt
|	DropTablespaceStmt
|	CreateServerStmt
|	AlterServerStmt
|	DropServerStmt
|	CreateResourceGroupStmt
|	AlterResourceGroupStmt
|	DropResourceGroupStmt
|	CreateLogfileGroupStmt
|	AlterLogfileGroupStmt
|	DropLogfileGroupStmt
|	CheckTableStmt
|	RepairTableStmt
|	OptimizeTableStmt
|	ChecksumTableStmt
|	ParenthesizedQueryStmt
|	ProcedureAnalyseStmt
|	TopLevelForStmt
|	TruncateTableStmt
|	UpdateStmt
|	UseStmt
|	UnlockTablesStmt
|	LockTablesStmt
|	ReturnStmt
|	PurgeStmt


ExplainableStmt:
	SelectStmt
|	ParenthesizedQueryStmt
|	ProcedureAnalyseStmt
|	WithSelectStmt
|	WithUpdateStmt
|	WithDeleteStmt
|	DeleteFromStmt
|	UpdateStmt
|	InsertIntoStmt
|	ReplaceIntoStmt
|	SetOprStmt


StatementList:
	Statement
	{
		if $1 != nil {
			s := $1
			if lexer, ok := yylex.(stmtTexter); ok {
				s.SetText(lexer.stmtText())
			}
			yyGetParser(yylex).result = append(yyGetParser(yylex).result, s)
		}
	}
|	StatementList ';' Statement
	{
		if $3 != nil {
			s := $3
			if lexer, ok := yylex.(stmtTexter); ok {
				s.SetText(lexer.stmtText())
			}
			yyGetParser(yylex).result = append(yyGetParser(yylex).result, s)
		}
	}


Constraint:
	ConstraintKeywordOpt ConstraintElem
	{
		cst := $2.(*ast.Constraint)
		if $1 != nil {
			cst.Name = $1.(string)
		}
		$$ = cst
	}


TableElement:
	ColumnDef {
		$$ = $1.(*ast.ColumnDef)
	}
|	Constraint
	{
		$$ = $1.(*ast.Constraint)
	}
|	period forKwd Identifier '(' ColumnName ',' ColumnName ')'
	{
		start, end := $5.(*ast.ColumnName), $7.(*ast.ColumnName)
		if start.Schema.O != "" || start.Table.O != "" || end.Schema.O != "" || end.Table.O != "" {
			yyAppendErr(yylex, yyErrorf(yylex, "PERIOD FOR requires two unqualified column names"))
			return 1
		}
		$$ = &ast.PeriodDefinition{Name: model.NewCIStr($3), Start: start, End: end, Offset: yyS[yypt-7].offset}
	}


TableElementList:
	TableElement
	{
		if $1 != nil {
			$$ = []interface{}{$1.(interface{})}
		} else {
			$$ = []interface{}{}
		}
	}
|	TableElementList ',' TableElement
	{
		if $3 != nil {
			$$ = append($1.([]interface{}), $3)
		} else {
			$$ = $1
		}
	}


CreateTableDefinition:
	'(' TableElementList ')'
	{
		tes := $2.([]interface {})
		var columnDefs []*ast.ColumnDef
		var constraints []*ast.Constraint
		var periods []*ast.PeriodDefinition
		for _, te := range tes {
			switch te := te.(type) {
			case *ast.ColumnDef:
				columnDefs = append(columnDefs, te)
			case *ast.Constraint:
				constraints = append(constraints, te)
			case *ast.PeriodDefinition:
				periods = append(periods, te)
			}
		}
		$$ = &ast.CreateTableStmt{
			Cols:           columnDefs,
			Constraints:    constraints,
			Periods:        periods,
		}
	}


TableOption:
	engine StringName
	{
		$$ = &ast.TableOption{Tp: ast.TableOptionEngine, StrValue: $2.(string)}
	}
|	engine eq StringName
	{
		$$ = &ast.TableOption{Tp: ast.TableOptionEngine, StrValue: $3.(string)}
	}
|	DefaultKwdOpt CharsetOrCharacterSet EqOpt CharsetName
	{
		$$ = &ast.TableOption{Tp: ast.TableOptionCharset, StrValue: $4.(string)}
	}
|	DefaultKwdOpt collate EqOpt CollationName
	{
		$$ = &ast.TableOption{Tp: ast.TableOptionCollate, StrValue: $4.(string)}
	}
|	autoIncrement EqOpt LengthNum
	{
		$$ = &ast.TableOption{Tp: ast.TableOptionAutoIncrement, UintValue: $3.(uint64)}
	}
|	comment EqOpt stringLit
	{
		$$ = &ast.TableOption{Tp: ast.TableOptionComment, StrValue: $3}
	}
|	avgRowLength EqOpt LengthNum
	{
		$$ = &ast.TableOption{Tp: ast.TableOptionAvgRowLength, UintValue: $3.(uint64)}
	}
|	connection EqOpt stringLit
	{
		$$ = &ast.TableOption{Tp: ast.TableOptionConnection, StrValue: $3}
	}
|	checksum EqOpt LengthNum
	{
		$$ = &ast.TableOption{Tp: ast.TableOptionCheckSum, UintValue: $3.(uint64)}
	}
|	password EqOpt stringLit
	{
		$$ = &ast.TableOption{Tp: ast.TableOptionPassword, StrValue: $3}
	}
|	compression EqOpt stringLit
	{
		$$ = &ast.TableOption{Tp: ast.TableOptionCompression, StrValue: $3}
	}
|	keyBlockSize EqOpt LengthNum
	{
		$$ = &ast.TableOption{Tp: ast.TableOptionKeyBlockSize, UintValue: $3.(uint64)}
	}
|	maxRows EqOpt LengthNum
	{
		$$ = &ast.TableOption{Tp: ast.TableOptionMaxRows, UintValue: $3.(uint64)}
	}
|	minRows EqOpt LengthNum
	{
		$$ = &ast.TableOption{Tp: ast.TableOptionMinRows, UintValue: $3.(uint64)}
	}
|	delayKeyWrite EqOpt LengthNum
	{
		$$ = &ast.TableOption{Tp: ast.TableOptionDelayKeyWrite, UintValue: $3.(uint64)}
	}
|	RowFormat
	{
		$$ = &ast.TableOption{Tp: ast.TableOptionRowFormat, UintValue: $1.(uint64)}
	}
|	statsPersistent EqOpt StatsPersistentVal
	{
		$$ = &ast.TableOption{Tp: ast.TableOptionStatsPersistent}
	}
|	shardRowIDBits EqOpt LengthNum
	{
		$$ = &ast.TableOption{Tp: ast.TableOptionShardRowID, UintValue: $3.(uint64)}
	}
|	packKeys EqOpt StatsPersistentVal
	{
		// Parse it but will ignore it.
		$$ = &ast.TableOption{Tp: ast.TableOptionPackKeys}
	}


StatsPersistentVal:
	defaultKwd {}
|	LengthNum {}


AlterTableOptionListOpt:
	/* EMPTY */ { $$ = []*ast.TableOption{} }
|	TableOptionList %prec higherThanComma


CreateTableOptionList:
	TableOptionList %prec lowerThanComma


TableOptionList:
	TableOption
	{
		$$ = []*ast.TableOption{$1.(*ast.TableOption)}
	}
|	TableOptionList TableOption
	{
		$$ = append($1.([]*ast.TableOption), $2.(*ast.TableOption))
	}
|	TableOptionList ','  TableOption
	{
		$$ = append($1.([]*ast.TableOption), $3.(*ast.TableOption))
	}


OptTable:
	/* EMPTY */ {}
|	tableKwd


TruncateTableStmt:
	truncate OptTable TableName
	{
		$$ = &ast.TruncateTableStmt{Table: $3.(*ast.TableName)}
	}


RowFormat:
	rowFormat EqOpt defaultKwd             { $$ = ast.RowFormatDefault }
|	rowFormat EqOpt dynamic             { $$ = ast.RowFormatDynamic }
|	rowFormat EqOpt fixed               { $$ = ast.RowFormatFixed }
|	rowFormat EqOpt compressed          { $$ = ast.RowFormatCompressed }
|	rowFormat EqOpt redundant           { $$ = ast.RowFormatRedundant }
|	rowFormat EqOpt compact             { $$ = ast.RowFormatCompact }
|	rowFormat EqOpt tokudbDefault      { $$ = ast.TokuDBRowFormatDefault }
|	rowFormat EqOpt tokudbFast         { $$ = ast.TokuDBRowFormatFast }
|	rowFormat EqOpt tokudbSmall        { $$ = ast.TokuDBRowFormatSmall }
|	rowFormat EqOpt tokudbZlib         { $$ = ast.TokuDBRowFormatZlib }
|	rowFormat EqOpt tokudbQuickLZ      { $$ = ast.TokuDBRowFormatQuickLZ }
|	rowFormat EqOpt tokudbLzma         { $$ = ast.TokuDBRowFormatLzma }
|	rowFormat EqOpt tokudbSnappy       { $$ = ast.TokuDBRowFormatSnappy }
|	rowFormat EqOpt tokudbUncompressed { $$ = ast.TokuDBRowFormatUncompressed }


/*************************************Type Begin***************************************/
Type:
	NumericType
	{
		$$ = $1
	}
|	StringType
	{
		$$ = $1
	}
|	DateAndTimeType
	{
		$$ = $1
	}
|	SpatialType
	{
		$$ = $1
	}


SpatialType:
	geometryType
	{
		x := types.NewFieldType(mysql.TypeGeometry)
		x.GeometryType = "GEOMETRY"
		$$ = x
	}
|	pointType
	{
		x := types.NewFieldType(mysql.TypeGeometry)
		x.GeometryType = "POINT"
		$$ = x
	}
|	lineStringType
	{
		x := types.NewFieldType(mysql.TypeGeometry)
		x.GeometryType = "LINESTRING"
		$$ = x
	}
|	polygonType
	{
		x := types.NewFieldType(mysql.TypeGeometry)
		x.GeometryType = "POLYGON"
		$$ = x
	}
|	multiPointType
	{
		x := types.NewFieldType(mysql.TypeGeometry)
		x.GeometryType = "MULTIPOINT"
		$$ = x
	}
|	multiLineStringType
	{
		x := types.NewFieldType(mysql.TypeGeometry)
		x.GeometryType = "MULTILINESTRING"
		$$ = x
	}
|	multiPolygonType
	{
		x := types.NewFieldType(mysql.TypeGeometry)
		x.GeometryType = "MULTIPOLYGON"
		$$ = x
	}
|	geometryCollectionType
	{
		x := types.NewFieldType(mysql.TypeGeometry)
		x.GeometryType = "GEOMETRYCOLLECTION"
		$$ = x
	}


SpatialFunctionName:
	geometryType
|	pointType
|	lineStringType
|	polygonType
|	multiPointType
|	multiLineStringType
|	multiPolygonType
|	geometryCollectionType


NumericType:
	IntegerType OptFieldLen FieldOpts
	{
		x := types.NewFieldType($1.(byte))
		x.Flen = $2.(int)
		for _, o := range $3.([]*ast.TypeOpt) {
			if o.IsUnsigned {
				x.Flag |= mysql.UnsignedFlag
			}
			if o.IsZerofill {
				x.Flag |= mysql.ZerofillFlag
			}
		}
		$$ = x
	}
|	BooleanType FieldOpts
	{
		x := types.NewFieldType($1.(byte))
		x.Flen = 1
		for _, o := range $2.([]*ast.TypeOpt) {
			if o.IsUnsigned {
				x.Flag |= mysql.UnsignedFlag
			}
			if o.IsZerofill {
				x.Flag |= mysql.ZerofillFlag
			}
		}
		$$ = x
	}
|	FixedPointType FloatOpt FieldOpts
	{
		fopt := $2.(*ast.FloatOpt)
		x := types.NewFieldType($1.(byte))
		x.Flen = fopt.Flen
		x.Decimal = fopt.Decimal
		for _, o := range $3.([]*ast.TypeOpt) {
			if o.IsUnsigned {
				x.Flag |= mysql.UnsignedFlag
			}
			if o.IsZerofill {
				x.Flag |= mysql.ZerofillFlag
			}
		}
		$$ = x
	}
|	FloatingPointType FloatOpt FieldOpts
	{
		fopt := $2.(*ast.FloatOpt)
		x := types.NewFieldType($1.(byte))
		x.Flen = fopt.Flen
		if x.Tp == mysql.TypeFloat {
			if x.Flen > 24 {
				x.Tp = mysql.TypeDouble
			}
		}
		x.Decimal = fopt.Decimal
		for _, o := range $3.([]*ast.TypeOpt) {
			if o.IsUnsigned {
				x.Flag |= mysql.UnsignedFlag
			}
			if o.IsZerofill {
				x.Flag |= mysql.ZerofillFlag
			}
		}
		$$ = x
	}
|	BitValueType OptFieldLen
	{
		x := types.NewFieldType($1.(byte))
		x.Flen = $2.(int)
		if x.Flen == types.UnspecifiedLength || x.Flen == 0 {
			x.Flen = 1
		} else if x.Flen > 64 {
			yyAppendErr(yylex, ErrTooBigDisplayWidth.GenWithStackByArgs(x.Flen))
		}
		$$ = x
	}


IntegerType:
	tinyIntType
	{
		$$ = mysql.TypeTiny
	}
|	smallIntType
	{
		$$ = mysql.TypeShort
	}
|	mediumIntType
	{
		$$ = mysql.TypeInt24
	}
|	intType
	{
		$$ = mysql.TypeLong
	}
|	int1Type
	{
		$$ = mysql.TypeTiny
	}
| 	int2Type
	{
		$$ = mysql.TypeShort
	}
| 	int3Type
	{
		$$ = mysql.TypeInt24
	}
|	int4Type
	{
		$$ = mysql.TypeLong
	}
|	int8Type
	{
		$$ = mysql.TypeLonglong
	}
|	integerType
	{
		$$ = mysql.TypeLong
	}
|	bigIntType
	{
		$$ = mysql.TypeLonglong
	}
|	long
	{
		// MariaDB accepts bare LONG as a column type (BIGINT).
		$$ = mysql.TypeLonglong
	}


BooleanType:
	boolType
	{
		$$ = mysql.TypeTiny
	}
|	booleanType
	{
		$$ = mysql.TypeTiny
	}


OptInteger:
	{}
|	integerType
|	intType


FixedPointType:
	decimalType { $$ = mysql.TypeNewDecimal }
|	numericType { $$ = mysql.TypeNewDecimal }


FloatingPointType:
	floatType
	{
		$$ = mysql.TypeFloat
	}
|	realType
	{
	    if yyGetParser(yylex).lexer.GetSQLMode().HasRealAsFloatMode() {
		    $$ = mysql.TypeFloat
	    } else {
		    $$ = mysql.TypeDouble
	    }
	}
|	doubleType
	{
		$$ = mysql.TypeDouble
	}
|	doubleType precisionType
	{
		$$ = mysql.TypeDouble
	}


BitValueType:
	bitType
	{
		$$ = mysql.TypeBit
	}


StringType:
	charType FieldLen OptBinary
	{
		x := types.NewFieldType(mysql.TypeString)
		x.Flen = $2.(int)
		x.Charset = $3.(*ast.OptBinary).Charset
		if $3.(*ast.OptBinary).IsBinary {
			x.Flag |= mysql.BinaryFlag
		}
		$$ = x
	}
|	charType OptBinary
	{
		x := types.NewFieldType(mysql.TypeString)
		x.Charset = $2.(*ast.OptBinary).Charset
		if $2.(*ast.OptBinary).IsBinary {
			x.Flag |= mysql.BinaryFlag
		}
		$$ = x
	}
|	Varchar FieldLen OptBinary
	{
		x := types.NewFieldType(mysql.TypeVarchar)
		x.Flen = $2.(int)
		x.Charset = $3.(*ast.OptBinary).Charset
		if $3.(*ast.OptBinary).IsBinary {
			x.Flag |= mysql.BinaryFlag
		}
		$$ = x
	}
|	binaryType OptFieldLen
	{
		x := types.NewFieldType(mysql.TypeString)
		x.Flen = $2.(int)
		x.Charset = charset.CharsetBin
		x.Collate = charset.CharsetBin
		x.Flag |= mysql.BinaryFlag
		$$ = x
	}
|	varbinaryType FieldLen
	{
		x := types.NewFieldType(mysql.TypeVarchar)
		x.Flen = $2.(int)
		x.Charset = charset.CharsetBin
		x.Collate = charset.CharsetBin
		x.Flag |= mysql.BinaryFlag
		$$ = x
	}
|	BlobType
	{
		x := $1.(*types.FieldType)
		x.Charset = charset.CharsetBin
		x.Collate = charset.CharsetBin
		x.Flag |= mysql.BinaryFlag
		$$ = $1.(*types.FieldType)
	}
|	TextType OptBinary
	{
		x := $1.(*types.FieldType)
		x.Charset = $2.(*ast.OptBinary).Charset
		if $2.(*ast.OptBinary).IsBinary {
			x.Flag |= mysql.BinaryFlag
		}
		$$ = x
	}
|	enum '(' StringList ')' OptCharset
	{
		x := types.NewFieldType(mysql.TypeEnum)
		x.Elems = $3.([]string)
		x.Charset = $5.(string)
		$$ = x
	}
|	set '(' StringList ')' OptCharset
	{
		x := types.NewFieldType(mysql.TypeSet)
		x.Elems = $3.([]string)
		x.Charset = $5.(string)
		$$ = x
	}
|	jsonType
	{
		x := types.NewFieldType(mysql.TypeJSON)
		x.Decimal = 0
		x.Charset = charset.CharsetBin
		x.Collate = charset.CollationBin
		$$ = x
	}
|	uuidType
	{
		// MariaDB UUID column type: char(36).
		x := types.NewFieldType(mysql.TypeString)
		x.Flen = 36
		x.Charset = mysql.DefaultCharset
		$$ = x
	}
|	inet4Type
	{
		// MariaDB INET4 column type: unsigned bigint.
		x := types.NewFieldType(mysql.TypeLonglong)
		x.Flag |= mysql.UnsignedFlag
		$$ = x
	}
|	inet6Type
	{
		// MariaDB INET6 column type: varbinary(16).
		x := types.NewFieldType(mysql.TypeVarchar)
		x.Flen = 16
		x.Charset = charset.CharsetBin
		x.Collate = charset.CollationBin
		x.Flag |= mysql.BinaryFlag
		$$ = x
	}


Varchar:
varcharType


BlobType:
	tinyblobType
	{
		x := types.NewFieldType(mysql.TypeTinyBlob)
		$$ = x
	}
|	blobType OptFieldLen
	{
		x := types.NewFieldType(mysql.TypeBlob)
		x.Flen = $2.(int)
		$$ = x
	}
|	mediumblobType
	{
		x := types.NewFieldType(mysql.TypeMediumBlob)
		$$ = x
	}
|	longblobType
	{
		x := types.NewFieldType(mysql.TypeLongBlob)
		$$ = x
	}


TextType:
	tinytextType
	{
		x := types.NewFieldType(mysql.TypeTinyBlob)
		$$ = x

	}
|	textType OptFieldLen
	{
		x := types.NewFieldType(mysql.TypeBlob)
		x.Flen = $2.(int)
		$$ = x
	}
|	mediumtextType
	{
		x := types.NewFieldType(mysql.TypeMediumBlob)
		$$ = x
	}
|	longtextType
	{
		x := types.NewFieldType(mysql.TypeLongBlob)
		$$ = x
	}
|	long varcharType
	{
		x := types.NewFieldType(mysql.TypeMediumBlob)
		$$ = x
	}
|	long byteType
	{
		x := types.NewFieldType(mysql.TypeLongBlob)
		$$ = x
	}


DateAndTimeType:
	dateType
	{
		x := types.NewFieldType(mysql.TypeDate)
		$$ = x
	}
|	datetimeType OptFieldLen
	{
		x := types.NewFieldType(mysql.TypeDatetime)
		x.Flen = mysql.MaxDatetimeWidthNoFsp
		x.Decimal = $2.(int)
		if x.Decimal > 0 {
			x.Flen = x.Flen + 1 + x.Decimal
		}
		$$ = x
	}
|	timestampType OptFieldLen
	{
		x := types.NewFieldType(mysql.TypeTimestamp)
		x.Flen = mysql.MaxDatetimeWidthNoFsp
		x.Decimal = $2.(int)
		if x.Decimal > 0 {
			x.Flen = x.Flen + 1 + x.Decimal
		}
		$$ = x
	}
|	timeType OptFieldLen
	{
		x := types.NewFieldType(mysql.TypeDuration)
		x.Flen = mysql.MaxDurationWidthNoFsp
		x.Decimal = $2.(int)
		if x.Decimal > 0 {
			x.Flen = x.Flen + 1 + x.Decimal
		}
		$$ = x
	}
|	yearType YearLenOpt FieldOpts
	{
		x := types.NewFieldType(mysql.TypeYear)
		x.Flen = $2.(int)
		$$ = x
	}


FieldLen:
	'(' LengthNum ')'
	{
		value, err := strconv.Atoi(strconv.FormatUint($2.(uint64), 10))
		if err != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "field length is out of range"))
			return 1
		}
		$$ = value
	}


OptFieldLen:
	/* EMPTY */ %prec empty { $$ = types.UnspecifiedLength }
|	FieldLen    { $$ = $1.(int) }


/* YEAR display width tolerating overflowed numeric literals: the lexer falls
   back to a decimal literal for 20+ digit numbers, and MariaDB accepts such
   YEAR(n) forms at parse time. */
YearLenOpt:
	/* EMPTY */ %prec empty { $$ = types.UnspecifiedLength }
|	FieldLen    { $$ = $1.(int) }
|	'(' decLit ')' { $$ = types.UnspecifiedLength }


FieldOpt:
	unsigned { $$ = &ast.TypeOpt{IsUnsigned: true} }
|	signed   { $$ = &ast.TypeOpt{IsUnsigned: false} }
|	zerofill { $$ = &ast.TypeOpt{IsZerofill: true, IsUnsigned: true} }


FieldOpts:
	/* EMPTY */        { $$ = []*ast.TypeOpt{} }
|	FieldOpts FieldOpt { $$ = append($1.([]*ast.TypeOpt), $2.(*ast.TypeOpt)) }


FloatOpt:
	/* EMPTY */ %prec empty { $$ = &ast.FloatOpt{Flen: types.UnspecifiedLength, Decimal: types.UnspecifiedLength} }
|	FieldLen    { $$ = &ast.FloatOpt{Flen: $1.(int), Decimal: types.UnspecifiedLength} }
|	Precision   { $$ = $1.(*ast.FloatOpt) }


Precision:
	'(' LengthNum ',' LengthNum ')'
	{
		flen, decimal := $2.(uint64), $4.(uint64)
		// Both pinned servers reject floating-point display widths and
		// fractional digits above the signed 32-bit range during parsing.
		if flen > math.MaxInt32 || decimal > math.MaxInt32 {
			yyAppendErr(yylex, yyErrorf(yylex, "Too big precision"))
			return 1
		}
		$$ = &ast.FloatOpt{Flen: int(flen), Decimal: int(decimal)}
	}

OptBinMod:
	/* EMPTY */ { $$ = false }
|	binaryType    { $$ = true }


OptBinary:
	/* EMPTY */ %prec empty
	{
		$$ = &ast.OptBinary{
			IsBinary: false,
			Charset:  "",
		}
	}
|	binaryType OptCharset
	{
		$$ = &ast.OptBinary{
			IsBinary: true,
			Charset:  $2.(string),
		}
	}
|	CharsetOrCharacterSet CharsetName OptBinMod
	{
		$$ = &ast.OptBinary{
			IsBinary: $3.(bool),
			Charset:  $2.(string),
		}
	}


OptCharset:
	/* EMPTY */           { $$ = "" }
|	CharsetOrCharacterSet CharsetName { $$ = $2.(string) }


CharsetOrCharacterSet:
	character set
|	charsetKwd


OptCollate:
	/* EMPTY */             { $$ = "" }
|	collate CollationName { $$ = $2.(string) }


StringList:
	stringLit                { $$ = []string{$1} }
|	StringList ',' stringLit { $$ = append($1.([]string), $3) }


StringName:
	stringLit  { $$ = $1 }
|	Identifier { $$ = $1 }


/***********************************************************************************
 * Update Statement
 * See https://dev.mysql.com/doc/refman/5.7/en/update.html
 ***********************************************************************************/
UpdateStmt:
	update TableOptimizerHints PriorityOpt IgnoreOptional TableRef set AssignmentList WhereClauseOptional OrderByOptional LimitClause ReturningOpt
	{
		var refs *ast.Join
		if x, ok := $5.(*ast.Join); ok {
			refs = x
		} else {
			refs = &ast.Join{Left: $5.(ast.ResultSetNode)}
		}
		st := &ast.UpdateStmt{
			Priority:  $3.(mysql.PriorityEnum),
			TableRefs: &ast.TableRefsClause{TableRefs: refs},
			List:	   $7.([]*ast.Assignment),
			IgnoreErr: $4.(bool),
		}
		if $2 != nil {
			st.TableHints = $2.([]*ast.TableOptimizerHint)
		}
		if $8 != nil {
			st.Where = $8.(ast.ExprNode)
		}
		if $9 != nil {
			st.Order = $9.(*ast.OrderByClause)
		}
		if $10 != nil {
			st.Limit = $10.(*ast.Limit)
		}
		if $11 != nil {
			st.Returning = $11.(*ast.FieldList)
		}
		$$ = st
	}
|	update TableOptimizerHints PriorityOpt IgnoreOptional TableRefs set AssignmentList WhereClauseOptional ReturningOpt
	{
		st := &ast.UpdateStmt{
			Priority:  $3.(mysql.PriorityEnum),
			TableRefs: &ast.TableRefsClause{TableRefs: $5.(*ast.Join)},
			List:	   $7.([]*ast.Assignment),
			IgnoreErr: $4.(bool),
		}
		if $2 != nil {
			st.TableHints = $2.([]*ast.TableOptimizerHint)
		}
		if $8 != nil {
			st.Where = $8.(ast.ExprNode)
		}
		if $9 != nil {
			st.Returning = $9.(*ast.FieldList)
		}
		$$ = st
	}


UseStmt:
	use DBName { $$ = &ast.UseStmt{DBName: $2.(string)} }


PurgeStmt:
	purge BinaryOrMaster logs { $$ = &ast.PurgeStmt{Binary: $2.(bool)} }
;

BinaryOrMaster:
	binaryType { $$ = true }
|	master { $$ = false }
;



WhereClause:
	where Expression { $$ = $2 }


WhereClauseOptional:
	/* EMPTY */ { $$ = nil }
|	WhereClause { $$ = $1 }


/************************************************************************************
 *  Account Management Statements
 *  https://dev.mysql.com/doc/refman/5.7/en/account-management-sql.html
 ************************************************************************************/
CreateUserStmt:
	create user IfNotExists UserSpecList RequireClause ConnectionOptions PasswordOrLockOptions
	{
		// See https://dev.mysql.com/doc/refman/5.7/en/create-user.html
		$$ = &ast.CreateUserStmt{
			IsCreateRole: false,
			IfNotExists: $3.(bool),
			Specs: $4.([]*ast.UserSpec),
			TslOptions: $5.([]*ast.TslOption),
			ResourceOptions: $6.([]*ast.ResourceOption),
			PasswordOrLockOptions: $7.([]*ast.PasswordOrLockOption),		}
	}


CreateRoleStmt:
	create role IfNotExists RoleSpecList {
		// See https://dev.mysql.com/doc/refman/8.0/en/create-role.html
		$$ = &ast.CreateUserStmt{
			IsCreateRole: true,
			IfNotExists: $3.(bool),
			Specs: $4.([]*ast.UserSpec),
		}
	}


/* See http://dev.mysql.com/doc/refman/5.7/en/alter-user.html */
AlterUserStmt:
	alter user IfExists UserSpecList
	{
		$$ = &ast.AlterUserStmt{
			IfExists: $3.(bool),
			Specs: $4.([]*ast.UserSpec),
		}
	}
| 	alter user IfExists user '(' ')' identified by AuthString
	{
		auth := &ast.AuthOption {
			AuthString: $9.(string),
			ByAuthString: true,
		}
		$$ = &ast.AlterUserStmt{
			IfExists: $3.(bool),
			CurrentAuth: auth,
		}
	}


UserSpec:
	Username AuthOption
	{
		userSpec := &ast.UserSpec{
			User: $1.(*auth.UserIdentity),
		}
		if $2 != nil {
			userSpec.AuthOpt = $2.(*ast.AuthOption)
		}
		$$ = userSpec
	}


UserSpecList:
	UserSpec                  { $$ = []*ast.UserSpec{$1.(*ast.UserSpec)} }
|	UserSpecList ',' UserSpec { $$ = append($1.([]*ast.UserSpec), $3.(*ast.UserSpec)) }


ConnectionOptions:
	/* EMPTY */
	{
		l := []*ast.ResourceOption{}
		$$ = l
	}
|	with ConnectionOptionList { $$ = $2 }


ConnectionOptionList:
	ConnectionOption                      { $$ = []*ast.ResourceOption{$1.(*ast.ResourceOption)} }
|	ConnectionOptionList ConnectionOption
	{
		l := $1.([]*ast.ResourceOption)
		l = append(l, $2.(*ast.ResourceOption))
		$$ = l
	}


ConnectionOption:
	maxQueriesPerHour NUM
	{
		$$ = &ast.ResourceOption {
			Type: ast.MaxQueriesPerHour,
			Count: $2.(int64),
		}
	}
|	maxUpdatesPerHour NUM
	{
		$$ = &ast.ResourceOption {
			Type: ast.MaxUpdatesPerHour,
			Count: $2.(int64),
		}
	}
|	maxConnectionsPerHour NUM
	{
		$$ = &ast.ResourceOption {
			Type: ast.MaxConnectionsPerHour,
			Count: $2.(int64),
		}
	}
|	maxUserConnections NUM
	{
		$$ = &ast.ResourceOption {
			Type: ast.MaxUserConnections,
			Count: $2.(int64),
		}
	}


RequireClause:
	/* EMPTY */
	{
		l := []*ast.TslOption{}
		$$ = l
	}
|	require none
	{
		t := &ast.TslOption {
			Type: ast.TslNone,
		}
		$$ = []*ast.TslOption{t}
	}
|	require ssl
	{
		t := &ast.TslOption {
			Type: ast.Ssl,
		}
		$$ = []*ast.TslOption{t}
	}
|	require x509
	{
		t := &ast.TslOption {
			Type: ast.X509,
		}
		$$ = []*ast.TslOption{t}
	}
|	require RequireList { $$ = $2 }



RequireList:
	RequireListElement
	{
		$$ = []*ast.TslOption{$1.(*ast.TslOption)}
	}
|	RequireListElement and RequireList
	{
		l := $3.([]*ast.TslOption)
		l = append(l, $1.(*ast.TslOption))
		$$ = l
	}


RequireListElement:
	issuer stringLit
	{
		$$ = &ast.TslOption {
			Type: ast.Issuer,
			Value: $2,
		}
	}
|	subject stringLit
	{
		$$ = &ast.TslOption {
			Type: ast.Subject,
			Value: $2,
		}
	}
|	cipher stringLit
	{
		$$ = &ast.TslOption {
			Type: ast.Cipher,
			Value: $2,
		}
	}


PasswordOrLockOptions:
	/* EMPTY */
	{
		l := []*ast.PasswordOrLockOption{}
		$$ = l
	}
|	PasswordOrLockOptionList
	{
		$$ = $1
	}


PasswordOrLockOptionList:
	PasswordOrLockOption
	{
		$$ = []*ast.PasswordOrLockOption{$1.(*ast.PasswordOrLockOption)}
	}
|	PasswordOrLockOptionList PasswordOrLockOption
	{
		l := $1.([]*ast.PasswordOrLockOption)
		l = append(l, $2.(*ast.PasswordOrLockOption))
		$$ = l
	}


PasswordOrLockOption:
	account unlock
	{
		$$ = &ast.PasswordOrLockOption {
			Type: ast.Unlock,
		}
	}
|	account lock
	{
		$$ = &ast.PasswordOrLockOption {
			Type: ast.Lock,
		}
	}
|	PasswordExpire
	{
		$$ = &ast.PasswordOrLockOption {
			Type: ast.PasswordExpire,
		}
	}
|	PasswordExpire interval NUM day
	{
		$$ = &ast.PasswordOrLockOption {
			Type: ast.PasswordExpireInterval,
			Count: $3.(int64),
		}
	}
|	PasswordExpire never
	{
		$$ = &ast.PasswordOrLockOption {
			Type: ast.PasswordExpireNever,
		}
	}
|	PasswordExpire defaultKwd
	{
		$$ = &ast.PasswordOrLockOption {
			Type: ast.PasswordExpireDefault,
		}
	}


PasswordExpire:
	password expire ClearPasswordExpireOptions { $$ = nil }


ClearPasswordExpireOptions:
	/* EMPTY */ { $$ = nil }


AuthOption:
	/* EMPTY */                                    { $$ = nil }
|	identified by AuthString                   { $$ = &ast.AuthOption { AuthString: $3.(string), ByAuthString: true } }
|	identified with StringName                 { $$ = nil }
|	identified with StringName by AuthString { $$ = &ast.AuthOption { AuthString: $5.(string), ByAuthString: true } }
|	identified with StringName as HashString { $$ = &ast.AuthOption{ HashString: $5.(string) } }
|	identified by password HashString        { $$ = &ast.AuthOption{ HashString: $4.(string) } }


HashString:
	stringLit { $$ = $1 }


RoleSpec:
	Rolename
	{
		role := $1.(*auth.RoleIdentity)
		roleSpec := &ast.UserSpec{
			User: &auth.UserIdentity {
				Username: role.Username,
				Hostname: role.Hostname,
			},
			IsRole: true,
		}
		$$ = roleSpec
	}


RoleSpecList:
	RoleSpec                  { $$ = []*ast.UserSpec{$1.(*ast.UserSpec)} }
|	RoleSpecList ',' RoleSpec { $$ = append($1.([]*ast.UserSpec), $3.(*ast.UserSpec)) }


/*************************************************************************************
 * Grant statement
 * See https://dev.mysql.com/doc/refman/5.7/en/grant.html
 *************************************************************************************/
GrantStmt:
	 grant PrivElemList on ObjectType PrivLevel to UserSpecList WithGrantOptionOpt
	 {
		$$ = &ast.GrantStmt{
			Privs: $2.([]*ast.PrivElem),
			ObjectType: $4.(ast.ObjectTypeType),
			Level: $5.(*ast.GrantLevel),
			Users: $7.([]*ast.UserSpec),
			WithGrant: $8.(bool),
		}
	 }


GrantRoleStmt:
	 grant RolenameList to UsernameList
	 {
		$$ = &ast.GrantRoleStmt {
			Roles: $2.([]*auth.RoleIdentity),
			Users: $4.([]*auth.UserIdentity),
		}
	}


WithGrantOptionOpt:
	/* EMPTY */                           { $$ = false }
|	with grant option               { $$ = true }
|	with maxQueriesPerHour NUM     { $$ = false }
|	with maxUpdatesPerHour NUM     { $$ = false }
|	with maxConnectionsPerHour NUM { $$ = false }
|	with maxUserConnections NUM     { $$ = false }


PrivElem:
	PrivType
	{
		$$ = &ast.PrivElem{
			Priv: $1.(mysql.PrivilegeType),
		}
	}
|	PrivType '(' ColumnNameList ')'
	{
		$$ = &ast.PrivElem{
			Priv: $1.(mysql.PrivilegeType),
			Cols: $3.([]*ast.ColumnName),
		}
	}


PrivElemList:
	PrivElem                  { $$ = []*ast.PrivElem{$1.(*ast.PrivElem)} }
|	PrivElemList ',' PrivElem { $$ = append($1.([]*ast.PrivElem), $3.(*ast.PrivElem)) }


PrivType:
	all                         { $$ = mysql.AllPriv }
|	all privileges            { $$ = mysql.AllPriv }
|	alter                       { $$ = mysql.AlterPriv }
|	create                      { $$ = mysql.CreatePriv }
|	create user               { $$ = mysql.CreateUserPriv }
|	trigger                     { $$ = mysql.TriggerPriv }
|	deleteKwd                      { $$ = mysql.DeletePriv }
|	drop                        { $$ = mysql.DropPriv }
|	process                     { $$ = mysql.ProcessPriv }
|	execute                     { $$ = mysql.ExecutePriv }
|	index                       { $$ = mysql.IndexPriv }
|	insert                      { $$ = mysql.InsertPriv }
|	selectKwd                      { $$ = mysql.SelectPriv }
|	super                       { $$ = mysql.SuperPriv }
|	show databases            { $$ = mysql.ShowDBPriv }
|	update                      { $$ = mysql.UpdatePriv }
|	grant option              { $$ = mysql.GrantPriv }
|	references                  { $$ = mysql.ReferencesPriv }
|	replication slave         { $$ = mysql.PrivilegeType(0) }
|	replication client        { $$ = mysql.PrivilegeType(0) }
|	usage                       { $$ = mysql.PrivilegeType(0) }
|	reload                      { $$ = mysql.PrivilegeType(0) }
|	create temporary tables { $$ = mysql.PrivilegeType(0) }
|	lock tables               { $$ = mysql.PrivilegeType(0) }
|	create view               { $$ = mysql.CreateViewPriv }
|	show view                 { $$ = mysql.ShowViewPriv }
|	create role               { $$ = mysql.CreateRolePriv }
|	drop role                 { $$ = mysql.DropRolePriv }
|	create routine            { $$ = mysql.PrivilegeType(0) }
|	alter routine             { $$ = mysql.PrivilegeType(0) }
|	event                       { $$ = mysql.PrivilegeType(0) }


ObjectType:
	/* EMPTY */ { $$ = ast.ObjectTypeNone }
|	tableKwd     { $$ = ast.ObjectTypeTable }


PrivLevel:
	'*'
	{
		$$ = &ast.GrantLevel {
			Level: ast.GrantLevelDB,
		}
	}
|	'*' '.' '*'
	{
		$$ = &ast.GrantLevel {
			Level: ast.GrantLevelGlobal,
		}
	}
| 	Identifier '.' '*'
	{
		$$ = &ast.GrantLevel {
			Level: ast.GrantLevelDB,
			DBName: $1,
		}
	}
|	Identifier '.' Identifier
	{
		$$ = &ast.GrantLevel {
			Level: ast.GrantLevelTable,
			DBName: $1,
			TableName: $3,
		}
	}
|	Identifier
	{
		$$ = &ast.GrantLevel {
			Level: ast.GrantLevelTable,
			TableName: $1,
		}
	}


/**************************************RevokeStmt*******************************************
 * See https://dev.mysql.com/doc/refman/5.7/en/revoke.html
 *******************************************************************************************/
RevokeStmt:
	 revoke PrivElemList on ObjectType PrivLevel from UserSpecList
	 {
		$$ = &ast.RevokeStmt{
			Privs: $2.([]*ast.PrivElem),
			ObjectType: $4.(ast.ObjectTypeType),
			Level: $5.(*ast.GrantLevel),
			Users: $7.([]*ast.UserSpec),
		}
	 }


RevokeRoleStmt:
	revoke RolenameList from UsernameList {
		$$ = &ast.RevokeRoleStmt {
			Roles: $2.([]*auth.RoleIdentity),
			Users: $4.([]*auth.UserIdentity),
		}
	}


/**************************************LoadDataStmt*****************************************
 * See https://dev.mysql.com/doc/refman/5.7/en/load-data.html
 *******************************************************************************************/
LoadDataStmt:
	load data LocalOpt infile stringLit DuplicateOpt into tableKwd TableName CharsetOpt Fields Lines IgnoreLines ColumnNameOrUserVarListOptWithBrackets LoadDataSetSpecOpt
	{
		x := &ast.LoadDataStmt{
			Path:               $5,
			OnDuplicate:        $6.(ast.OnDuplicateKeyHandlingType),
			Table:              $9.(*ast.TableName),
			ColumnsAndUserVars: $14.([]*ast.ColumnNameOrUserVar),
			IgnoreLines:        $13.(uint64),
		}
		if $3 != nil {
			x.IsLocal = true
			// See https://dev.mysql.com/doc/refman/5.7/en/load-data.html#load-data-duplicate-key-handling
			// If you do not specify IGNORE or REPLACE modifier , then we set default behavior to IGNORE when LOCAL modifier is specified
			if x.OnDuplicate == ast.OnDuplicateKeyHandlingError {
				x.OnDuplicate = ast.OnDuplicateKeyHandlingIgnore
			}
		}
		if $11 != nil {
			x.FieldsInfo = $11.(*ast.FieldsClause)
		}
		if $12 != nil {
			x.LinesInfo = $12.(*ast.LinesClause)
		}
		if $15 != nil {
			x.ColumnAssignments = $15.([]*ast.Assignment)
		}
		columns := []*ast.ColumnName{}
		for _, v := range x.ColumnsAndUserVars {
			if v.ColumnName != nil {
				columns = append(columns, v.ColumnName)
			}
		}
		x.Columns = columns
		$$ = x
	}


IgnoreLines:
	/* EMPTY */          { $$ = uint64(0) }
|	ignore NUM lines { $$ = getUint64FromNUM($2) }


CharsetOpt:
	/* EMPTY */ { $$ = "" }
|	CharsetOrCharacterSet CharsetName { $$ = $2.(string) }


LocalOpt:
	/* EMPTY */ { $$ = nil }
|	local     { $$ = $1 }


Fields:
	/* EMPTY */ {
		escape := "\\"
		$$ = &ast.FieldsClause{
			Terminated: "\t",
			Escaped:    escape[0],
		}
	}
|	FieldsOrColumns FieldItemList
	{
		fieldsClause := &ast.FieldsClause{
			Terminated: "\t",
			Escaped:    []byte("\\")[0],
		}
		fieldItems := $2.([]*ast.FieldItem)
		for _, item := range fieldItems {
			switch item.Type {
			case ast.Terminated:
				fieldsClause.Terminated = item.Value
			case ast.Enclosed:
				var enclosed byte
				if len(item.Value) > 0 {
					enclosed = item.Value[0]
				}
				fieldsClause.Enclosed = enclosed
			case ast.Escaped:
				var escaped byte
				if len(item.Value) > 0 {
					escaped = item.Value[0]
				}
				fieldsClause.Escaped = escaped
			}
		}
		$$ = fieldsClause
	}


FieldsOrColumns:
	fields
|	columns


FieldItemList:
	FieldItemList FieldItem
	{
		fieldItems := $1.([]*ast.FieldItem)
		$$ = append(fieldItems, $2.(*ast.FieldItem))
	}
|	FieldItem
	{
		fieldItems := make([]*ast.FieldItem, 1, 1)
		fieldItems[0] = $1.(*ast.FieldItem)
		$$ = fieldItems
	}


FieldItem:
	terminated by stringLit
	{
		$$ = &ast.FieldItem{
			Type:    ast.Terminated,
			Value:   $3,
		}
	}
|	optionally enclosed by stringLit
	{
		str := $4
		if str != "\\" && len(str) > 1 {
			yyAppendErr(yylex, ErrWrongFieldTerminators.GenWithStackByArgs())
			return 1
		}
		$$ = &ast.FieldItem{
			Type:    ast.Enclosed,
			Value:   str,
		}
	}
|	enclosed by stringLit
	{
		str := $3
		if str != "\\" && len(str) > 1 {
			yyAppendErr(yylex, ErrWrongFieldTerminators.GenWithStackByArgs())
			return 1
		}
		$$ = &ast.FieldItem{
			Type:    ast.Enclosed,
			Value:   str,
		}
	}
|	escaped by stringLit
	{
		str := $3
		if str != "\\" && len(str) > 1 {
			yyAppendErr(yylex, ErrWrongFieldTerminators.GenWithStackByArgs())
			return 1
		}
		$$ = &ast.FieldItem{
			Type:    ast.Escaped,
			Value:   str,
		}
	}


Lines:
	/* EMPTY */                      { $$ = &ast.LinesClause{Terminated: "\n"} }
|	lines Starting LinesTerminated { $$ = &ast.LinesClause{ Starting: $2.(string), Terminated: $3.(string) } }


Starting:
	/* EMPTY */               { $$ = "" }
|	starting by stringLit { $$ = $3 }


LinesTerminated:
	/* EMPTY */                 { $$ = "\n" }
|	terminated by stringLit { $$ = $3 }


LoadDataSetSpecOpt:
	/* EMPTY */           { $$ = nil }
|	set LoadDataSetList { $$ = $2 }


LoadDataSetList:
	LoadDataSetList ',' LoadDataSetItem
	{
		l := $1.([]*ast.Assignment)
		$$ = append(l, $3.(*ast.Assignment))
	}
|	LoadDataSetItem
	{
		$$ = []*ast.Assignment{$1.(*ast.Assignment)}
	}


LoadDataSetItem:
	SimpleIdent eq ExprOrDefault
	{
		$$ = &ast.Assignment{
			Column:	$1.(*ast.ColumnNameExpr).Name,
			Expr:	$3,
		}
	}


/*********************************************************************
 * Lock/Unlock Tables
 * See http://dev.mysql.com/doc/refman/5.7/en/lock-tables.html
 * All the statement leaves empty. This is used to prevent mysqldump error.
 *********************************************************************/

UnlockTablesStmt:
	unlock TablesTerminalSym { $$ = &ast.UnlockTableStmt{} }


LockTablesStmt:
	lock TablesTerminalSym TableLockList { $$ = &ast.LockTableStmt{} }


TablesTerminalSym:
	tables
|	tableKwd


TableLock:
	TableName LockType


LockType:
	read
|	read local
|	write


TableLockList:
	TableLock
|	TableLockList ',' TableLock


/********************************************************************
 * Kill Statement
 * See https://dev.mysql.com/doc/refman/5.7/en/kill.html
 *******************************************************************/
KillStmt:
	kill KillOption Expression { $$ = newKillStmt($2.(bool), $3) }


KillOption:
	/* EMPTY */ %prec empty { $$ = false }
|	connection             { $$ = false }
|	query                  { $$ = true }


/********************************************************************
 * Replication and administration statements.
 * Each production follows the syntax documented by MySQL 8.0/8.4 and
 * MariaDB; the corresponding manual page is linked above the rule.
 *******************************************************************/

/* https://dev.mysql.com/doc/refman/8.0/en/start-slave.html */
ReplicaControlStmt:
	start slave ReplicaStartSuffix
	{
		$$ = newReplicaControlStmt(true, false, false, $3.(*ast.ReplicaSuffix))
	}
/* https://dev.mysql.com/doc/refman/8.4/en/start-replica.html */
|	start replica ReplicaStartSuffix
	{
		$$ = newReplicaControlStmt(true, true, false, $3.(*ast.ReplicaSuffix))
	}
/* https://dev.mysql.com/doc/refman/8.0/en/stop-slave.html */
|	stop slave ReplicaStopSuffix
	{
		$$ = newReplicaControlStmt(false, false, false, $3.(*ast.ReplicaSuffix))
	}
/* https://dev.mysql.com/doc/refman/8.4/en/stop-replica.html */
|	stop replica ReplicaStopSuffix
	{
		$$ = newReplicaControlStmt(false, true, false, $3.(*ast.ReplicaSuffix))
	}
/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/start-replica */
|	start all slaves ReplicaThreadOpt
	{
		$$ = newReplicaControlStmt(true, false, true, &ast.ReplicaSuffix{Threads: $4.([]string)})
	}
/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/start-replica */
|	start all replicas ReplicaThreadOpt
	{
		$$ = newReplicaControlStmt(true, true, true, &ast.ReplicaSuffix{Threads: $4.([]string)})
	}
/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/stop-replica */
|	stop all slaves ReplicaThreadOpt
	{
		$$ = newReplicaControlStmt(false, false, true, &ast.ReplicaSuffix{Threads: $4.([]string)})
	}
/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/stop-replica */
|	stop all replicas ReplicaThreadOpt
	{
		$$ = newReplicaControlStmt(false, true, true, &ast.ReplicaSuffix{Threads: $4.([]string)})
	}


/* https://dev.mysql.com/doc/refman/8.4/en/start-replica.html */
/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/start-replica */
ReplicaStartSuffix:
	ReplicationLiteralMode ReplicaConnectionNameOpt ReplicaThreadOpt ReplicaStartTail
	{
		name := $2.(*ast.ReplicaSuffix)
		suffix := $4.(*ast.ReplicaSuffix)
		suffix.Threads = $3.([]string)
		suffix.ConnectionName = name.ConnectionName
		suffix.HasConnectionName = name.HasConnectionName
		if err := validateReplicaStart(suffix); err != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "%v", err))
			return 1
		}
		$$ = suffix
	}


/* https://dev.mysql.com/doc/refman/8.4/en/start-replica.html */
ReplicaStartTail:
	/* EMPTY */ { $$ = &ast.ReplicaSuffix{} }
|	ReplicaUntil ReplicaConnectionOptionsOpt ReplicaChannelOpt
	{
		suffix := $1.(*ast.ReplicaSuffix)
		channel := $3.(*ast.ReplicaSuffix)
		suffix.ConnectionOptions = $2.([]*ast.Assignment)
		suffix.Channel = channel.Channel
		suffix.HasChannel = channel.HasChannel
		$$ = suffix
	}
|	ReplicaConnectionOptions ReplicaChannelOpt
	{
		suffix := $2.(*ast.ReplicaSuffix)
		suffix.ConnectionOptions = $1.([]*ast.Assignment)
		$$ = suffix
	}
/* MariaDB places FOR CHANNEL before UNTIL.
   https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/start-replica */
|	ReplicaChannel ReplicaUntilOpt
	{
		channel := $1.(*ast.ReplicaSuffix)
		suffix := $2.(*ast.ReplicaSuffix)
		suffix.Channel = channel.Channel
		suffix.HasChannel = true
		suffix.ChannelBeforeUntil = len(suffix.UntilOptions) != 0 || suffix.UntilAfterGaps
		$$ = suffix
	}


/* https://dev.mysql.com/doc/refman/8.4/en/stop-replica.html */
/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/stop-replica */
ReplicaStopSuffix:
	ReplicaConnectionNameOpt ReplicaThreadOpt ReplicaChannelOpt
	{
		name := $1.(*ast.ReplicaSuffix)
		suffix := $3.(*ast.ReplicaSuffix)
		suffix.Threads = $2.([]string)
		suffix.ConnectionName = name.ConnectionName
		suffix.HasConnectionName = name.HasConnectionName
		if suffix.HasConnectionName && suffix.HasChannel {
			yyAppendErr(yylex, yyErrorf(yylex, "Connection name and FOR CHANNEL cannot be specified together"))
			return 1
		}
		$$ = suffix
	}


/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to */
ReplicaConnectionNameOpt:
	/* EMPTY */ { $$ = &ast.ReplicaSuffix{} }
|	stringLit { $$ = &ast.ReplicaSuffix{ConnectionName: $1, HasConnectionName: true} }


/* https://dev.mysql.com/doc/refman/8.4/en/start-replica.html */
ReplicaThreadOpt:
	/* EMPTY */ { $$ = []string{} }
|	ReplicaThreadList


/* https://dev.mysql.com/doc/refman/8.4/en/start-replica.html */
ReplicaThreadList:
	ReplicaThread { $$ = []string{$1} }
|	ReplicaThreadList ',' ReplicaThread { $$ = append($1.([]string), $3) }


/* https://dev.mysql.com/doc/refman/8.4/en/start-replica.html */
ReplicaThread:
	ioThread { $$ = "IO_THREAD" }
|	sqlThread { $$ = "SQL_THREAD" }


/* https://dev.mysql.com/doc/refman/8.4/en/start-replica.html */
ReplicaUntilOpt:
	/* EMPTY */ { $$ = &ast.ReplicaSuffix{} }
|	ReplicaUntil


/* https://dev.mysql.com/doc/refman/8.4/en/start-replica.html */
/* https://dev.mysql.com/doc/refman/8.0/en/start-replica.html */
ReplicaUntil:
	until ReplicaUntilFileOptions
	{
		options := $2.([]*ast.Assignment)
		if err := validateReplicaUntilFiles(options); err != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "%v", err))
			return 1
		}
		$$ = &ast.ReplicaSuffix{UntilOptions: options}
	}
|	until sqlBeforeGTIDs eq stringLit
	{
		$$ = &ast.ReplicaSuffix{UntilOptions: []*ast.Assignment{newReplicationOption($2, ast.NewValueExpr($4))}}
	}
|	until sqlAfterGTIDs eq stringLit
	{
		$$ = &ast.ReplicaSuffix{UntilOptions: []*ast.Assignment{newReplicationOption($2, ast.NewValueExpr($4))}}
	}
|	until sqlAfterMTSGaps
	{
		$$ = &ast.ReplicaSuffix{UntilAfterGaps: true}
	}
/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/start-replica */
|	until masterGTIDPos eq stringLit
	{
		$$ = &ast.ReplicaSuffix{UntilOptions: []*ast.Assignment{newReplicationOption($2, ast.NewValueExpr($4))}}
	}


/* https://dev.mysql.com/doc/refman/8.4/en/start-replica.html */
ReplicaUntilFileOptions:
	ReplicationFileOption { $$ = []*ast.Assignment{$1.(*ast.Assignment)} }
|	ReplicaUntilFileOptions ',' ReplicationFileOption { $$ = append($1.([]*ast.Assignment), $3.(*ast.Assignment)) }


/* https://dev.mysql.com/doc/refman/8.4/en/start-replica.html */
ReplicaConnectionOptionsOpt:
	/* EMPTY */ { $$ = []*ast.Assignment(nil) }
|	ReplicaConnectionOptions


/* https://dev.mysql.com/doc/refman/8.4/en/start-replica.html */
ReplicaConnectionOptions:
	user eq stringLit ReplicaPasswordOpt ReplicaDefaultAuthOpt ReplicaPluginDirOpt
	{
		options := []*ast.Assignment{newReplicationOption("USER", ast.NewValueExpr($3))}
		options = append(options, $4.([]*ast.Assignment)...)
		options = append(options, $5.([]*ast.Assignment)...)
		$$ = append(options, $6.([]*ast.Assignment)...)
	}
|	password eq stringLit ReplicaDefaultAuthOpt ReplicaPluginDirOpt
	{
		options := []*ast.Assignment{newReplicationOption("PASSWORD", ast.NewValueExpr($3))}
		options = append(options, $4.([]*ast.Assignment)...)
		$$ = append(options, $5.([]*ast.Assignment)...)
	}
|	defaultAuth eq stringLit ReplicaPluginDirOpt
	{
		options := []*ast.Assignment{newReplicationOption("DEFAULT_AUTH", ast.NewValueExpr($3))}
		$$ = append(options, $4.([]*ast.Assignment)...)
	}
|	pluginDir eq stringLit
	{
		$$ = []*ast.Assignment{newReplicationOption("PLUGIN_DIR", ast.NewValueExpr($3))}
	}


/* https://dev.mysql.com/doc/refman/8.4/en/start-replica.html */
ReplicaPasswordOpt:
	/* EMPTY */ { $$ = []*ast.Assignment(nil) }
|	password eq stringLit { $$ = []*ast.Assignment{newReplicationOption("PASSWORD", ast.NewValueExpr($3))} }


/* https://dev.mysql.com/doc/refman/8.4/en/start-replica.html */
ReplicaDefaultAuthOpt:
	/* EMPTY */ { $$ = []*ast.Assignment(nil) }
|	defaultAuth eq stringLit { $$ = []*ast.Assignment{newReplicationOption("DEFAULT_AUTH", ast.NewValueExpr($3))} }


/* https://dev.mysql.com/doc/refman/8.4/en/start-replica.html */
ReplicaPluginDirOpt:
	/* EMPTY */ { $$ = []*ast.Assignment(nil) }
|	pluginDir eq stringLit { $$ = []*ast.Assignment{newReplicationOption("PLUGIN_DIR", ast.NewValueExpr($3))} }


/* https://dev.mysql.com/doc/refman/8.4/en/start-replica.html */
ReplicaChannelOpt:
	/* EMPTY */ { $$ = &ast.ReplicaSuffix{} }
|	ReplicaChannel


/* https://dev.mysql.com/doc/refman/8.4/en/start-replica.html */
ReplicaChannel:
	forKwd channel stringLit { $$ = &ast.ReplicaSuffix{Channel: $3, HasChannel: true} }


/* https://dev.mysql.com/doc/refman/8.4/en/reset-replica.html */
ChannelOpt:
	/* EMPTY */ { $$ = &ast.ResetSuffix{} }
|	forKwd channel stringLit { $$ = &ast.ResetSuffix{Channel: $3, HasChannel: true} }


/* https://dev.mysql.com/doc/refman/8.0/en/change-master-to.html */
/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to */
ChangeMasterStmt:
	change master ReplicationLiteralMode ReplicaConnectionNameOpt to ReplicationOptionList ReplicaChannelOpt
	{
		name := $4.(*ast.ReplicaSuffix)
		channel := $7.(*ast.ReplicaSuffix)
		stmt := &ast.ChangeMasterStmt{
			Options: $6.([]*ast.Assignment),
			ConnectionName: name.ConnectionName,
			HasConnectionName: name.HasConnectionName,
			Channel: channel.Channel,
			HasChannel: channel.HasChannel,
		}
		if err := validateChangeMaster(stmt); err != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "%v", err))
			return 1
		}
		$$ = stmt
	}
/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
/* MySQL 8.0 also accepts MASTER_* aliases with this spelling.
   https://dev.mysql.com/doc/refman/8.0/en/change-replication-source-to.html */
|	change replication source ReplicationLiteralMode to ReplicationOptionList ReplicaChannelOpt
	{
		channel := $7.(*ast.ReplicaSuffix)
		stmt := &ast.ChangeMasterStmt{Replica: true, Options: $6.([]*ast.Assignment), Channel: channel.Channel, HasChannel: channel.HasChannel}
		if err := validateChangeMaster(stmt); err != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "%v", err))
			return 1
		}
		$$ = stmt
	}


/* Integer replication options convert token text using my_strtoll10 rather than
   NUM_literal's decimal/float evaluation; heartbeat periods are the exception.
   https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html
   https://github.com/mysql/mysql-server/blob/8.4/sql/sql_yacc.yy
   https://github.com/MariaDB/server/blob/12.3/sql/sql_yacc.yy */
ReplicationLiteralMode:
	/* EMPTY */
	{
		yylex.(*Scanner).replicationNumbers = true
		$$ = nil
	}


/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
/* https://dev.mysql.com/doc/refman/8.0/en/change-master-to.html */
/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to */
ReplicationOptionList:
	ReplicationOption
	{
		$$ = []*ast.Assignment{$1.(*ast.Assignment)}
	}
|	ReplicationOptionList ',' ReplicationOption
	{
		$$ = append($1.([]*ast.Assignment), $3.(*ast.Assignment))
	}


/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
/* https://dev.mysql.com/doc/refman/8.0/en/change-master-to.html */
ReplicationOption:
	ReplicationStringOptionName eq stringLit
	{
		$$ = newReplicationOption($1, ast.NewValueExpr($3))
	}
|	ReplicationTLSCipherSuitesOptionName eq ReplicationStringOrNullValue
	{
		$$ = newReplicationOption($1, $3)
	}
|	ReplicationNumericOptionName eq ReplicationUnsignedValue
	{
		$$ = newReplicationOption($1, $3)
	}
|	sourceHeartbeatPeriod eq ReplicationNumber
	{
		$$ = newReplicationOption($1, $3)
	}
|	ReplicationBooleanOptionName eq ReplicationUnsignedValue
	{
		$$ = newReplicationOption($1, $3)
	}
|	requireRowFormat eq ReplicationBinaryValue
	{
		$$ = newReplicationOption($1, $3)
	}
|	ReplicationStrictBinaryOptionName eq ReplicationStrictBinaryValue
	{
		$$ = newReplicationOption($1, $3)
	}
|	ReplicationIDListOptionName eq '(' ReplicationIDListOpt ')'
	{
		$$ = newReplicationOption($1, &ast.ReplicationOptionValue{Values: $4.([]ast.ExprNode)})
	}
|	requireTablePrimaryKeyCheck eq ReplicationPrimaryKeyCheckValue
	{
		$$ = newReplicationOption($1, $3)
	}
|	assignGTIDsToAnonymousTransactions eq ReplicationAssignGTIDValue
	{
		$$ = newReplicationOption($1, $3)
	}
|	privilegeChecksUser eq ReplicationPrivilegeValue
	{
		$$ = newReplicationOption($1, $3)
	}
|	ReplicationFileOption
/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to */
|	ReplicationStringDefaultOptionName eq ReplicationStringOrDefaultValue
	{
		$$ = newReplicationOption($1, $3)
	}
|	ReplicationNumericDefaultOptionName eq ReplicationUnsignedOrDefaultValue
	{
		$$ = newReplicationOption($1, $3)
	}
|	masterHeartbeatPeriod eq ReplicationNumberOrDefaultValue
	{
		$$ = newReplicationOption($1, $3)
	}
|	ReplicationMariaBooleanDefaultOptionName eq ReplicationBoolOrDefaultValue
	{
		$$ = newReplicationOption($1, $3)
	}
|	ReplicationMariaBooleanOptionName eq ReplicationBoolValue
	{
		$$ = newReplicationOption($1, $3)
	}
|	masterUseGTID eq ReplicationUseGTIDValue
	{
		$$ = newReplicationOption($1, $3)
	}


/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
/* https://dev.mysql.com/doc/refman/8.0/en/change-master-to.html */
ReplicationStringOptionName:
	masterHost | sourceHost | masterBind | sourceBind | masterUser | sourceUser | masterPassword | sourcePassword
|	sourceSSLCA | sourceSSLCAPath | sourceSSLCert | sourceSSLCipher | sourceSSLKey | sourceSSLCRL | sourceSSLCRLPath
|	masterTLSVersion | sourceTLSVersion | masterPublicKeyPath | sourcePublicKeyPath | networkNamespace
|	masterCompressionAlgorithms | sourceCompressionAlgorithms


/* https://dev.mysql.com/doc/refman/8.0/en/change-master-to.html */
/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to */
ReplicationStringDefaultOptionName:
	masterSSLCA | masterSSLCAPath | masterSSLCert | masterSSLCipher | masterSSLKey | masterSSLCRL | masterSSLCRLPath


/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
/* https://dev.mysql.com/doc/refman/8.0/en/change-master-to.html */
ReplicationTLSCipherSuitesOptionName:
	masterTLSCipherSuites | sourceTLSCipherSuites


/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
/* https://dev.mysql.com/doc/refman/8.0/en/change-master-to.html */
ReplicationNumericOptionName:
	masterPort | sourcePort | sourceConnectRetry | sourceRetryCount | masterDelay | sourceDelay
|	masterZstdCompressionLevel | sourceZstdCompressionLevel


/* https://dev.mysql.com/doc/refman/8.0/en/change-master-to.html */
/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to */
ReplicationNumericDefaultOptionName:
	masterConnectRetry | masterRetryCount


/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
/* https://dev.mysql.com/doc/refman/8.0/en/change-master-to.html */
ReplicationBooleanOptionName:
	sourceSSL | sourceSSLVerifyServerCert | masterAutoPosition | sourceAutoPosition
|	getMasterPublicKey | getSourcePublicKey


/* https://dev.mysql.com/doc/refman/8.0/en/change-master-to.html */
/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to */
ReplicationMariaBooleanDefaultOptionName:
	masterSSL | masterSSLVerifyServerCert


/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to */
/* https://github.com/MariaDB/server/blob/12.3/sql/lex.h */
ReplicationMariaBooleanOptionName:
	masterDemoteToSlave | masterDemoteToReplica


/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
ReplicationStrictBinaryOptionName:
	gtidOnly | sourceConnectionAutoFailover


/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to */
ReplicationIDListOptionName:
	ignoreServerIDs | doDomainIDs | ignoreDomainIDs


/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
ReplicationStringOrNullValue:
	stringLit { $$ = ast.NewValueExpr($1) }
|	null { $$ = ast.NewValueExpr(nil) }


/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to */
ReplicationStringOrDefaultValue:
	stringLit { $$ = ast.NewValueExpr($1) }
|	defaultKwd { $$ = &ast.DefaultExpr{} }


/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to */
ReplicationNumberOrDefaultValue:
	ReplicationNumber
|	defaultKwd { $$ = &ast.DefaultExpr{} }


/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to */
ReplicationUnsignedOrDefaultValue:
	ReplicationUnsignedValue
|	defaultKwd { $$ = &ast.DefaultExpr{} }


/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to */
ReplicationBoolOrDefaultValue:
	ReplicationBoolValue
|	defaultKwd { $$ = &ast.DefaultExpr{} }


/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to */
ReplicationBoolValue:
	ReplicationUnsignedValue
|	trueKwd { $$ = ast.NewValueExpr(true) }
|	falseKwd { $$ = ast.NewValueExpr(false) }


/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
ReplicationBinaryValue:
	ReplicationUnsignedValue
	{
		if !replicationBinaryValue($1) {
			yyAppendErr(yylex, yyErrorf(yylex, "Replication boolean option only accepts 0 or 1"))
			return 1
		}
		$$ = $1
	}


/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
ReplicationStrictBinaryValue:
	NUM
	{
		if getUint64FromNUM($1) > 1 {
			yyAppendErr(yylex, yyErrorf(yylex, "Replication boolean option only accepts integer 0 or 1"))
			return 1
		}
		$$ = ast.NewValueExpr($1)
	}
|	ReplicationHexNumber
	{
		if !replicationBinaryValue($1) {
			yyAppendErr(yylex, yyErrorf(yylex, "Replication boolean option only accepts integer 0 or 1"))
			return 1
		}
		$$ = $1
	}


/* ulong_num/ulonglong_num convert unsigned numeric tokens using their integer prefix.
   https://github.com/mysql/mysql-server/blob/8.4/sql/sql_yacc.yy
   https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
ReplicationUnsignedNumber:
	NumLiteral { $$ = replicationUnsignedNumber(yyDollar[1].ident) }
/* MariaDB additionally accepts a leading '+'.
   https://github.com/MariaDB/server/blob/12.3/sql/sql_yacc.yy
   https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to */
|	'+' NumLiteral { $$ = &ast.UnaryOperationExpr{Op: opcode.Plus, V: replicationUnsignedNumber(yyDollar[2].ident)} }


/* Heartbeat periods use NUM_literal rather than ulong_num/ulonglong_num.
   https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
ReplicationNumber:
	NumLiteral { $$ = ast.NewValueExpr($1) }
/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to */
|	'+' NumLiteral { $$ = &ast.UnaryOperationExpr{Op: opcode.Plus, V: ast.NewValueExpr($2)} }


/* https://github.com/mysql/mysql-server/blob/8.4/sql/sql_yacc.yy */
/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
ReplicationUnsignedValue:
	ReplicationUnsignedNumber
|	ReplicationHexNumber


/* https://github.com/mysql/mysql-server/blob/8.4/sql/sql_yacc.yy */
/* https://dev.mysql.com/doc/refman/8.4/en/hexadecimal-literals.html */
ReplicationHexNumber:
	hexLit
	{
		value, err := replicationHexNumber($1, yyDollar[1].ident)
		if err != nil {
			yyAppendErr(yylex, yyErrorf(yylex, "%v", err))
			return 1
		}
		$$ = value
	}


/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
ReplicationIDListOpt:
	/* EMPTY */ { $$ = []ast.ExprNode{} }
|	ReplicationIDList


/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
ReplicationIDList:
	ReplicationUnsignedValue { $$ = []ast.ExprNode{$1} }
|	ReplicationIDList ',' ReplicationUnsignedValue { $$ = append($1.([]ast.ExprNode), $3) }


/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
ReplicationPrimaryKeyCheckValue:
	streamKwd { $$ = &ast.ReplicationOptionValue{Keyword: "STREAM"} }
|	on { $$ = &ast.ReplicationOptionValue{Keyword: "ON"} }
|	off { $$ = &ast.ReplicationOptionValue{Keyword: "OFF"} }
|	generateKwd { $$ = &ast.ReplicationOptionValue{Keyword: "GENERATE"} }


/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
ReplicationAssignGTIDValue:
	off { $$ = &ast.ReplicationOptionValue{Keyword: "OFF"} }
|	local { $$ = &ast.ReplicationOptionValue{Keyword: "LOCAL"} }
|	stringLit
	{
		if !validReplicationUUID($1) {
			yyAppendErr(yylex, yyErrorf(yylex, "Invalid UUID for ASSIGN_GTIDS_TO_ANONYMOUS_TRANSACTIONS"))
			return 1
		}
		$$ = ast.NewValueExpr($1)
	}


/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
ReplicationPrivilegeValue:
	Username
	{
		user := $1.(*auth.UserIdentity)
		if user.CurrentUser || user.Username == "" {
			yyAppendErr(yylex, yyErrorf(yylex, "PRIVILEGE_CHECKS_USER requires a named account, not CURRENT_USER"))
			return 1
		}
		$$ = &ast.ReplicationOptionValue{User: user}
	}
|	null { $$ = ast.NewValueExpr(nil) }


/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to */
ReplicationUseGTIDValue:
	no { $$ = &ast.ReplicationOptionValue{Keyword: "NO"} }
|	currentPos { $$ = &ast.ReplicationOptionValue{Keyword: "CURRENT_POS"} }
|	slavePos { $$ = &ast.ReplicationOptionValue{Keyword: "SLAVE_POS"} }
/* https://github.com/MariaDB/server/blob/12.3/sql/lex.h */
/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to */
|	replicaPos { $$ = &ast.ReplicationOptionValue{Keyword: "REPLICA_POS"} }
|	defaultKwd { $$ = &ast.DefaultExpr{} }


/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
/* https://dev.mysql.com/doc/refman/8.0/en/change-master-to.html */
ReplicationFileOption:
	ReplicationLogFileOptionName eq stringLit
	{
		$$ = newReplicationOption($1, ast.NewValueExpr($3))
	}
|	masterLogPos eq ReplicationUnsignedNumber
	{
		$$ = newReplicationOption($1, $3)
	}
|	sourceLogPos eq ReplicationUnsignedNumber
	{
		$$ = newReplicationOption($1, $3)
	}
|	relayLogPos eq ReplicationUnsignedValue
	{
		$$ = newReplicationOption($1, $3)
	}


/* https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html */
/* https://dev.mysql.com/doc/refman/8.0/en/change-master-to.html */
ReplicationLogFileOptionName:
	masterLogFile | sourceLogFile | relayLogFile


/* https://dev.mysql.com/doc/refman/8.0/en/reset.html */
ResetStmt:
	reset ResetOptionList
	{
		options := $2.([]*ast.ResetStmt)
		if len(options) == 1 {
			$$ = options[0]
		} else {
			$$ = &ast.ResetStmt{Options: options}
		}
	}
/* https://dev.mysql.com/doc/refman/8.0/en/reset-persist.html */
|	reset persist
	{
		$$ = &ast.ResetStmt{Tp: ast.ResetPersist}
	}
|	reset persist ResetPersistName
	{
		$$ = &ast.ResetStmt{Tp: ast.ResetPersist, PersistName: $3.(string)}
	}
|	reset persist ifKwd exists ResetPersistName
	{
		$$ = &ast.ResetStmt{Tp: ast.ResetPersist, IfExists: true, PersistName: $5.(string)}
	}


/* https://dev.mysql.com/doc/refman/8.0/en/reset.html */
ResetOptionList:
	ResetOption { $$ = []*ast.ResetStmt{$1.(*ast.ResetStmt)} }
|	ResetOptionList ',' ResetOption { $$ = append($1.([]*ast.ResetStmt), $3.(*ast.ResetStmt)) }


/* https://dev.mysql.com/doc/refman/8.0/en/reset-master.html */
ResetOption:
	master
	{
		$$ = &ast.ResetStmt{Tp: ast.ResetMaster}
	}
|	master to NUM
	{
		$$ = &ast.ResetStmt{Tp: ast.ResetMaster, Args: []ast.ExprNode{ast.NewValueExpr($3)}}
	}
/* https://dev.mysql.com/doc/refman/8.0/en/reset-slave.html */
|	slave ResetAllOpt ChannelOpt
	{
		channel := $3.(*ast.ResetSuffix)
		$$ = &ast.ResetStmt{Tp: ast.ResetSlave, All: $2.(bool), Channel: channel.Channel, HasChannel: channel.HasChannel}
	}
/* https://dev.mysql.com/doc/refman/8.0/en/reset-replica.html */
|	replica ResetAllOpt ChannelOpt
	{
		channel := $3.(*ast.ResetSuffix)
		$$ = &ast.ResetStmt{Tp: ast.ResetReplica, All: $2.(bool), Channel: channel.Channel, HasChannel: channel.HasChannel}
	}
/* RESET BINARY LOGS AND GTIDS was added in MySQL 8.4.
   https://dev.mysql.com/doc/refman/8.4/en/reset-binary-logs-and-gtids.html */
|	binaryType logs and gtids
	{
		$$ = &ast.ResetStmt{Tp: ast.ResetBinaryLogsAndGTIDs}
	}
|	binaryType logs and gtids to NUM
	{
		$$ = &ast.ResetStmt{Tp: ast.ResetBinaryLogsAndGTIDs, Args: []ast.ExprNode{ast.NewValueExpr($6)}}
	}
/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/reset */
|	query cache
	{
		$$ = &ast.ResetStmt{Tp: ast.ResetQueryCache}
	}


/* https://dev.mysql.com/doc/refman/8.4/en/reset-replica.html */
ResetAllOpt:
	/* EMPTY */ { $$ = false }
|	all         { $$ = true }


/* https://dev.mysql.com/doc/refman/8.0/en/reset-persist.html */
ResetPersistName:
	Identifier
	{
		$$ = $1
	}
|	Identifier '.' Identifier
	{
		$$ = $1 + "." + $3
	}
|	defaultKwd '.' Identifier
	{
		$$ = "default." + $3
	}


/* https://dev.mysql.com/doc/refman/8.0/en/shutdown.html */
ShutdownStmt:
	shutdown
	{
		$$ = &ast.ShutdownStmt{}
	}
/* https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/shutdown */
|	shutdown wait forKwd all slaves
	{
		$$ = &ast.ShutdownStmt{WaitForAllSlaves: true}
	}
|	shutdown wait forKwd all replicas
	{
		$$ = &ast.ShutdownStmt{WaitForAllSlaves: true}
	}


/* https://dev.mysql.com/doc/refman/8.0/en/handler.html */
HandlerStmt:
	handler TableName open HandlerAliasOpt
	{
		$$ = &ast.HandlerStmt{Tp: ast.HandlerOpen, Table: $2.(*ast.TableName), Alias: $4.(string)}
	}
|	handler Identifier close
	{
		$$ = &ast.HandlerStmt{Tp: ast.HandlerClose, Table: &ast.TableName{Name: model.NewCIStr($2)}}
	}
|	handler Identifier read HandlerDirection HandlerWhereOpt HandlerLimitOpt
	{
		x := &ast.HandlerStmt{
			Tp:        ast.HandlerRead,
			Table:     &ast.TableName{Name: model.NewCIStr($2)},
			Direction: $4.(string),
			Limit:     $6.(*ast.Limit),
		}
		if $5 != nil {
			x.Where = $5.(ast.ExprNode)
		}
		$$ = x
	}
|	handler Identifier read Identifier HandlerRKeyFunction HandlerWhereOpt HandlerLimitOpt
	{
		x := &ast.HandlerStmt{
			Tp:        ast.HandlerRead,
			Table:     &ast.TableName{Name: model.NewCIStr($2)},
			HasIndex:  true,
			Index:     $4,
			Direction: $5.(string),
			Limit:     $7.(*ast.Limit),
		}
		if $6 != nil {
			x.Where = $6.(ast.ExprNode)
		}
		$$ = x
	}
|	handler Identifier read Identifier HandlerRKeyMode '(' ExpressionList ')' HandlerWhereOpt HandlerLimitOpt
	{
		x := &ast.HandlerStmt{
			Tp:       ast.HandlerRead,
			Table:    &ast.TableName{Name: model.NewCIStr($2)},
			HasIndex: true,
			Index:    $4,
			Op:       $5.(string),
			Values:   $7.([]ast.ExprNode),
			Limit:    $10.(*ast.Limit),
		}
		if $9 != nil {
			x.Where = $9.(ast.ExprNode)
		}
		$$ = x
	}


/* https://dev.mysql.com/doc/refman/8.0/en/handler.html */
HandlerAliasOpt:
	/* EMPTY */ { $$ = "" }
|	Identifier { $$ = $1 }
|	as Identifier { $$ = $2 }


/* https://dev.mysql.com/doc/refman/8.0/en/handler.html */
HandlerDirection:
	first { $$ = "FIRST" }
|	next  { $$ = "NEXT" }


/* https://dev.mysql.com/doc/refman/8.0/en/handler.html */
HandlerRKeyFunction:
	first { $$ = "FIRST" }
|	next  { $$ = "NEXT" }
|	prev  { $$ = "PREV" }
|	last  { $$ = "LAST" }


/* https://dev.mysql.com/doc/refman/8.0/en/handler.html */
HandlerRKeyMode:
	eq    { $$ = "=" }
|	ge    { $$ = ">=" }
|	le    { $$ = "<=" }
|	'>'   { $$ = ">" }
|	'<'   { $$ = "<" }


/* https://dev.mysql.com/doc/refman/8.0/en/handler.html */
HandlerLimitOpt:
	/* EMPTY */ { $$ = (*ast.Limit)(nil) }
|	limit LimitOption
	{
		$$ = &ast.Limit{Count: $2.(ast.ExprNode)}
	}
|	limit LimitOption ',' LimitOption
	{
		$$ = &ast.Limit{Offset: $2.(ast.ExprNode), Count: $4.(ast.ExprNode)}
	}
|	limit LimitOption offset LimitOption
	{
		$$ = &ast.Limit{Offset: $4.(ast.ExprNode), Count: $2.(ast.ExprNode)}
	}


/* https://dev.mysql.com/doc/refman/8.0/en/handler.html */
HandlerWhereOpt:
	/* EMPTY */ { $$ = nil }
|	where Expression
	{
		$$ = $2
	}


/* https://dev.mysql.com/doc/refman/8.0/en/cache-index.html */
CacheIndexStmt:
	cache index CacheIndexTableList in CacheIndexName
	{
		x := $5.(*ast.CacheIndexStmt)
		for _, table := range $3.([]*ast.LoadIndexTable) {
			x.Tables = append(x.Tables, table.Table)
			x.TableIndexes = append(x.TableIndexes, table.IndexNames)
		}
		$$ = x
	}
/* https://dev.mysql.com/doc/refman/8.0/en/cache-index.html */
|	cache index TableName AdminPartition CacheIndexListOpt in CacheIndexName
	{
		x := $7.(*ast.CacheIndexStmt)
		table := $3.(*ast.TableName)
		partitions := $4.([]model.CIStr)
		table.PartitionNames = partitions
		x.PartitionAll = len(partitions) == 0
		x.Tables = []*ast.TableName{table}
		x.TableIndexes = [][]string{$5.([]string)}
		$$ = x
	}


/* https://dev.mysql.com/doc/refman/8.0/en/cache-index.html */
CacheIndexTableList:
	CacheIndexTable { $$ = []*ast.LoadIndexTable{$1.(*ast.LoadIndexTable)} }
|	CacheIndexTableList ',' CacheIndexTable { $$ = append($1.([]*ast.LoadIndexTable), $3.(*ast.LoadIndexTable)) }


/* https://dev.mysql.com/doc/refman/8.0/en/cache-index.html */
CacheIndexTable:
	TableName CacheIndexListOpt { $$ = &ast.LoadIndexTable{Table: $1.(*ast.TableName), IndexNames: $2.([]string)} }


/* https://dev.mysql.com/doc/refman/8.0/en/cache-index.html */
/* https://dev.mysql.com/doc/refman/8.0/en/load-index.html */
CacheIndexListOpt:
	/* EMPTY */ { $$ = []string(nil) }
|	KeyOrIndex '(' ')' { $$ = []string{} }
|	KeyOrIndex '(' LoadIndexNameList ')' { $$ = $3.([]string) }


/* https://dev.mysql.com/doc/refman/8.0/en/cache-index.html */
/* https://dev.mysql.com/doc/refman/8.0/en/load-index.html */
AdminPartition:
	partition '(' PartitionNameList ')' { $$ = $3.([]model.CIStr) }
|	partition '(' all ')' { $$ = []model.CIStr{} }


/* https://dev.mysql.com/doc/refman/8.0/en/cache-index.html */
CacheIndexName:
	defaultKwd   { $$ = &ast.CacheIndexStmt{Target: ast.CacheIndexDefaultTarget} }
|	Identifier   { $$ = &ast.CacheIndexStmt{Target: ast.CacheIndexIdentifierTarget, Name: $1} }


/* https://dev.mysql.com/doc/refman/8.0/en/load-index.html */
LoadIndexStmt:
	load index into cache LoadIndexTableList
	{
		$$ = &ast.LoadIndexStmt{Tables: $5.([]*ast.LoadIndexTable)}
	}
/* https://dev.mysql.com/doc/refman/8.0/en/load-index.html */
|	load index into cache TableName AdminPartition CacheIndexListOpt LoadIndexIgnoreLeavesOpt
	{
		table := $5.(*ast.TableName)
		partitions := $6.([]model.CIStr)
		table.PartitionNames = partitions
		$$ = &ast.LoadIndexStmt{Tables: []*ast.LoadIndexTable{{
			Table: table,
			PartitionAll: len(partitions) == 0,
			IndexNames: $7.([]string),
			IgnoreLeaves: $8.(bool),
		}}}
	}


/* https://dev.mysql.com/doc/refman/8.0/en/load-index.html */
LoadIndexTableList:
	LoadIndexTable
	{
		$$ = []*ast.LoadIndexTable{$1.(*ast.LoadIndexTable)}
	}
|	LoadIndexTableList ',' LoadIndexTable
	{
		$$ = append($1.([]*ast.LoadIndexTable), $3.(*ast.LoadIndexTable))
	}


/* https://dev.mysql.com/doc/refman/8.0/en/load-index.html */
LoadIndexTable:
	TableName CacheIndexListOpt LoadIndexIgnoreLeavesOpt
	{
		$$ = &ast.LoadIndexTable{Table: $1.(*ast.TableName), IndexNames: $2.([]string), IgnoreLeaves: $3.(bool)}
	}


/* https://dev.mysql.com/doc/refman/8.0/en/load-index.html */
LoadIndexIgnoreLeavesOpt:
	/* EMPTY */ { $$ = false }
|	ignore leaves { $$ = true }


/* https://dev.mysql.com/doc/refman/8.0/en/load-index.html */
LoadIndexNameList:
	LoadIndexName
	{
		$$ = []string{$1}
	}
|	LoadIndexNameList ',' LoadIndexName
	{
		$$ = append($1.([]string), $3)
	}


/* https://dev.mysql.com/doc/refman/8.0/en/load-index.html */
LoadIndexName:
	Identifier
|	primary { $$ = "PRIMARY" }


/* https://dev.mysql.com/doc/refman/8.0/en/load-xml.html */
LoadXMLStmt:
	load xml LoadXMLPriorityOpt LocalOpt infile stringLit DuplicateOpt into tableKwd TableName PartitionNameListOpt CharsetOpt LoadXMLRowTagOpt LoadXMLIgnoreLinesOpt ColumnNameOrUserVarListOptWithBrackets LoadDataSetSpecOpt
	{
		x := $14.(*ast.LoadXMLStmt)
		x.Priority = $3
		x.Path = $6
		x.OnDuplicate = $7.(ast.OnDuplicateKeyHandlingType)
		x.Table = $10.(*ast.TableName)
		x.Table.PartitionNames = $11.([]model.CIStr)
		x.Charset = $12.(string)
		x.ColumnsAndUserVars = $15.([]*ast.ColumnNameOrUserVar)
		if $16 != nil {
			x.ColumnAssignments = $16.([]*ast.Assignment)
		}
		if $4 != nil {
			x.IsLocal = true
		}
		if tag := $13.(*ast.LoadXMLTagInfo); tag != nil {
			x.RowTag = tag.RowTag
			x.HasRowTag = tag.HasRowTag
		}
		$$ = x
	}


/* https://dev.mysql.com/doc/refman/8.0/en/load-xml.html */
LoadXMLPriorityOpt:
	/* EMPTY */ { $$ = "" }
|	lowPriority { $$ = "LOW_PRIORITY" }
|	concurrent { $$ = "CONCURRENT" }


/* https://dev.mysql.com/doc/refman/8.0/en/load-xml.html */
LoadXMLIgnoreLinesOpt:
	/* EMPTY */ { $$ = &ast.LoadXMLStmt{} }
|	ignore NUM lines
	{
		$$ = &ast.LoadXMLStmt{LoadDataStmt: ast.LoadDataStmt{IgnoreLines: getUint64FromNUM($2)}, HasIgnoreLines: true}
	}
|	ignore NUM rows
	{
		$$ = &ast.LoadXMLStmt{LoadDataStmt: ast.LoadDataStmt{IgnoreLines: getUint64FromNUM($2)}, HasIgnoreLines: true, IgnoreRows: true}
	}


/* https://dev.mysql.com/doc/refman/8.0/en/load-xml.html */
LoadXMLRowTagOpt:
	/* EMPTY */ { $$ = (*ast.LoadXMLTagInfo)(nil) }
|	rows identified by stringLit
	{
		$$ = &ast.LoadXMLTagInfo{RowTag: $4, HasRowTag: true}
	}


/* https://dev.mysql.com/doc/refman/8.0/en/rename-user.html */
RenameUserStmt:
	rename user RenameUserPairList
	{
		$$ = &ast.RenameUserStmt{Pairs: $3.([]*ast.RenameUserPair)}
	}


/* https://dev.mysql.com/doc/refman/8.0/en/rename-user.html */
RenameUserPairList:
	RenameUserPair
	{
		$$ = []*ast.RenameUserPair{$1.(*ast.RenameUserPair)}
	}
|	RenameUserPairList ',' RenameUserPair
	{
		$$ = append($1.([]*ast.RenameUserPair), $3.(*ast.RenameUserPair))
	}


/* https://dev.mysql.com/doc/refman/8.0/en/rename-user.html */
RenameUserPair:
	Username to Username
	{
		$$ = &ast.RenameUserPair{From: $1.(*auth.UserIdentity), To: $3.(*auth.UserIdentity)}
	}


/* https://dev.mysql.com/doc/refman/8.0/en/import-table.html */
ImportTableStmt:
	import_ tableKwd from StringList
	{
		$$ = &ast.ImportTableStmt{Paths: $4.([]string)}
	}


/* https://dev.mysql.com/doc/refman/8.0/en/install-plugin.html */
InstallPluginStmt:
	install plugin Identifier soname stringLit
	{
		$$ = &ast.InstallPluginStmt{Name: $3, Soname: $5}
	}


/* https://dev.mysql.com/doc/refman/8.0/en/uninstall-plugin.html */
UninstallPluginStmt:
	uninstall plugin Identifier
	{
		$$ = &ast.UninstallPluginStmt{Name: $3}
	}


/* https://dev.mysql.com/doc/refman/8.0/en/install-component.html */
InstallComponentStmt:
	install component StringList
	{
		$$ = &ast.InstallComponentStmt{URIs: $3.([]string)}
	}
|	install component StringList set ComponentSetValueList
	{
		$$ = &ast.InstallComponentStmt{URIs: $3.([]string), SetValues: $5.([]*ast.ComponentSetValue)}
	}


/* https://dev.mysql.com/doc/refman/8.0/en/uninstall-component.html */
UninstallComponentStmt:
	uninstall component StringList
	{
		$$ = &ast.UninstallComponentStmt{URIs: $3.([]string)}
	}


/* https://dev.mysql.com/doc/refman/8.0/en/install-component.html */
ComponentSetValueList:
	ComponentSetValue
	{
		$$ = []*ast.ComponentSetValue{$1.(*ast.ComponentSetValue)}
	}
|	ComponentSetValueList ',' ComponentSetValue
	{
		$$ = append($1.([]*ast.ComponentSetValue), $3.(*ast.ComponentSetValue))
	}


/* https://dev.mysql.com/doc/refman/8.0/en/install-component.html */
ComponentSetValue:
	global Identifier eq SetExpr
	{
		$$ = &ast.ComponentSetValue{Scope: "GLOBAL", Name: $2, Value: $4}
	}
|	persist Identifier eq SetExpr
	{
		$$ = &ast.ComponentSetValue{Scope: "PERSIST", Name: $2, Value: $4}
	}


/* https://dev.mysql.com/doc/refman/8.0/en/clone.html */
CloneStmt:
	clone local data directory EqOpt stringLit
	{
		$$ = &ast.CloneStmt{Local: true, DataDir: $6}
	}
|	clone instance from Username ':' NUM identified by stringLit CloneSSL
	{
		$$ = &ast.CloneStmt{
			User:     $4.(*auth.UserIdentity),
			Port:     getUint64FromNUM($6),
			Password: $9,
			SSL:      $10.(string),
		}
	}
|	clone instance from Username ':' NUM identified by stringLit data directory EqOpt stringLit CloneSSL
	{
		$$ = &ast.CloneStmt{
			User:     $4.(*auth.UserIdentity),
			Port:     getUint64FromNUM($6),
			Password: $9,
			DataDir:  $13,
			SSL:      $14.(string),
		}
	}


/* https://dev.mysql.com/doc/refman/8.0/en/clone.html */
CloneSSL:
	/* EMPTY */ { $$ = "" }
|	require ssl { $$ = "REQUIRE SSL" }
|	require no ssl { $$ = "REQUIRE NO SSL" }


/* https://dev.mysql.com/doc/refman/8.0/en/get-diagnostics.html */
GetDiagnosticsStmt:
	get DiagnosticsArea diagnostics DiagnosticsInfo
	{
		x := $4.(*ast.GetDiagnosticsStmt)
		x.Stacked = $2.(bool)
		$$ = x
	}


/* https://dev.mysql.com/doc/refman/8.0/en/get-diagnostics.html */
DiagnosticsArea:
	/* EMPTY */ { $$ = false }
|	current     { $$ = false }
|	stacked     { $$ = true }


/* https://dev.mysql.com/doc/refman/8.0/en/get-diagnostics.html */
DiagnosticsInfo:
	DiagnosticsStatementItemList
	{
		$$ = &ast.GetDiagnosticsStmt{Items: $1.([]*ast.DiagnosticsItem)}
	}
|	condition DiagnosticsConditionNumber DiagnosticsConditionItemList
	{
		$$ = &ast.GetDiagnosticsStmt{
			Condition: $2,
			Items:     $3.([]*ast.DiagnosticsItem),
		}
	}


/* https://dev.mysql.com/doc/refman/8.0/en/get-diagnostics.html */
/* https://mariadb.com/docs/server/reference/sql-statements/programmatic-compound-statements/programmatic-compound-statements-diagnostics/get-diagnostics */
DiagnosticsConditionNumber:
	Literal
|	SimpleIdent
|	Variable


/* https://dev.mysql.com/doc/refman/8.0/en/get-diagnostics.html */
DiagnosticsStatementItemList:
	DiagnosticsStatementItem
	{
		$$ = []*ast.DiagnosticsItem{$1.(*ast.DiagnosticsItem)}
	}
|	DiagnosticsStatementItemList ',' DiagnosticsStatementItem
	{
		$$ = append($1.([]*ast.DiagnosticsItem), $3.(*ast.DiagnosticsItem))
	}


/* https://dev.mysql.com/doc/refman/8.0/en/get-diagnostics.html */
DiagnosticsConditionItemList:
	DiagnosticsConditionItem
	{
		$$ = []*ast.DiagnosticsItem{$1.(*ast.DiagnosticsItem)}
	}
|	DiagnosticsConditionItemList ',' DiagnosticsConditionItem
	{
		$$ = append($1.([]*ast.DiagnosticsItem), $3.(*ast.DiagnosticsItem))
	}


/* https://dev.mysql.com/doc/refman/8.0/en/get-diagnostics.html */
DiagnosticsStatementItem:
	DiagnosticsTarget eq Identifier
	{
		item := $1.(*ast.DiagnosticsItem)
		name, ok := diagnosticsStatementItemName($3)
		if !ok {
			yyAppendErr(yylex, yyErrorf(yylex, "unknown diagnostics statement item '%s'", $3))
			return 1
		}
		item.Name = name
		$$ = item
	}


/* https://dev.mysql.com/doc/refman/8.0/en/get-diagnostics.html */
DiagnosticsConditionItem:
	DiagnosticsTarget eq DiagnosticsConditionItemName
	{
		item := $1.(*ast.DiagnosticsItem)
		name, ok := diagnosticsConditionItemName($3)
		if !ok {
			yyAppendErr(yylex, yyErrorf(yylex, "unknown diagnostics condition item '%s'", $3))
			return 1
		}
		item.Name = name
		$$ = item
	}


/* https://mariadb.com/docs/server/reference/sql-statements/programmatic-compound-statements/programmatic-compound-statements-diagnostics/get-diagnostics */
DiagnosticsConditionItemName:
	Identifier
|	rowNumber { $$ = "ROW_NUMBER" }


/* https://dev.mysql.com/doc/refman/8.0/en/get-diagnostics.html */
DiagnosticsTarget:
	Identifier
	{
		$$ = &ast.DiagnosticsItem{Target: $1}
	}
|	singleAtIdentifier
	{
		$$ = &ast.DiagnosticsItem{Target: strings.TrimPrefix($1, "@"), IsUserVar: true}
	}

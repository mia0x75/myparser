package myparser

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/auth"
	restoreformat "github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/model"
)

// https://dev.mysql.com/doc/refman/8.4/en/change-replication-source-to.html
// https://dev.mysql.com/doc/refman/8.0/en/change-master-to.html
// https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to
func TestReplicationChangeOptionTypes(t *testing.T) {
	groups := []struct {
		names  string
		values []struct{ src, restored string }
	}{
		{
			"MASTER_HOST SOURCE_HOST MASTER_BIND SOURCE_BIND MASTER_USER SOURCE_USER MASTER_PASSWORD SOURCE_PASSWORD " +
				"MASTER_SSL_CA SOURCE_SSL_CA MASTER_SSL_CAPATH SOURCE_SSL_CAPATH MASTER_SSL_CERT SOURCE_SSL_CERT " +
				"MASTER_SSL_CIPHER SOURCE_SSL_CIPHER MASTER_SSL_KEY SOURCE_SSL_KEY MASTER_SSL_CRL SOURCE_SSL_CRL " +
				"MASTER_SSL_CRLPATH SOURCE_SSL_CRLPATH MASTER_TLS_VERSION SOURCE_TLS_VERSION MASTER_PUBLIC_KEY_PATH SOURCE_PUBLIC_KEY_PATH " +
				"MASTER_COMPRESSION_ALGORITHMS SOURCE_COMPRESSION_ALGORITHMS NETWORK_NAMESPACE MASTER_LOG_FILE SOURCE_LOG_FILE RELAY_LOG_FILE",
			[]struct{ src, restored string }{{"'Mixed Case'", "'Mixed Case'"}},
		},
		{
			"MASTER_TLS_CIPHERSUITES SOURCE_TLS_CIPHERSUITES",
			[]struct{ src, restored string }{{"'TLS_AES_128_GCM_SHA256'", "'TLS_AES_128_GCM_SHA256'"}, {"null", "NULL"}},
		},
		{
			"MASTER_PORT SOURCE_PORT MASTER_CONNECT_RETRY SOURCE_CONNECT_RETRY MASTER_RETRY_COUNT SOURCE_RETRY_COUNT " +
				"MASTER_ZSTD_COMPRESSION_LEVEL SOURCE_ZSTD_COMPRESSION_LEVEL RELAY_LOG_POS",
			[]struct{ src, restored string }{
				{"0", "0"}, {"1.5", "1"}, {"12e2", "12"}, {".5", "0"}, {"0x10", "16"},
				{"18446744073709551615", "18446744073709551615"}, {"18446744073709551616", "18446744073709551615"},
			},
		},
		{
			"MASTER_LOG_POS SOURCE_LOG_POS",
			[]struct{ src, restored string }{{"0", "0"}, {"4.5", "4"}, {"12e2", "12"}, {"18446744073709551615", "18446744073709551615"}},
		},
		{
			"MASTER_DELAY SOURCE_DELAY",
			[]struct{ src, restored string }{{"0", "0"}, {"2147483647", "2147483647"}, {"0x7fffffff", "2147483647"}},
		},
		{
			"MASTER_SSL SOURCE_SSL MASTER_SSL_VERIFY_SERVER_CERT SOURCE_SSL_VERIFY_SERVER_CERT MASTER_AUTO_POSITION SOURCE_AUTO_POSITION " +
				"GET_MASTER_PUBLIC_KEY GET_SOURCE_PUBLIC_KEY",
			[]struct{ src, restored string }{{"0", "0"}, {"1", "1"}, {"2", "2"}, {"1.5", "1"}, {"0x1", "1"}},
		},
		{
			"REQUIRE_ROW_FORMAT",
			[]struct{ src, restored string }{{"0", "0"}, {"1", "1"}, {"1.0", "1"}, {"1.5", "1"}, {"0e2", "0"}, {"0x1", "1"}},
		},
		{
			"GTID_ONLY SOURCE_CONNECTION_AUTO_FAILOVER",
			[]struct{ src, restored string }{{"0", "0"}, {"1", "1"}, {"0x1", "1"}},
		},
		{
			"MASTER_HEARTBEAT_PERIOD SOURCE_HEARTBEAT_PERIOD",
			[]struct{ src, restored string }{{"0", "0"}, {"0.001", "0.001"}, {"1e-3", "1e-03"}, {"4294967", "4294967"}},
		},
		{
			"IGNORE_SERVER_IDS",
			[]struct{ src, restored string }{{"()", "()"}, {"(1)", "(1)"}, {"(1, 2.5, 0x03)", "(1, 2, 3)"}},
		},
		{
			"REQUIRE_TABLE_PRIMARY_KEY_CHECK",
			[]struct{ src, restored string }{{"stream", "STREAM"}, {"on", "ON"}, {"off", "OFF"}, {"generate", "GENERATE"}},
		},
		{
			"ASSIGN_GTIDS_TO_ANONYMOUS_TRANSACTIONS",
			[]struct{ src, restored string }{{"off", "OFF"}, {"local", "LOCAL"}, {"'AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE'", "'AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE'"}},
		},
		{
			"PRIVILEGE_CHECKS_USER",
			[]struct{ src, restored string }{{"null", "NULL"}, {"repl", "'repl'@'%'"}, {"`Repl`@`LOCALHOST`", "'Repl'@'LOCALHOST'"}, {"'rep''l'@''", "'rep''l'@''"}},
		},
	}
	for _, group := range groups {
		for name := range strings.FieldsSeq(group.names) {
			for _, value := range group.values {
				for _, prefix := range []string{"CHANGE MASTER TO ", "CHANGE REPLICATION SOURCE TO "} {
					src := strings.ToLower(prefix+name) + "=" + value.src
					t.Run(src, func(t *testing.T) {
						RunRestoreTest(t, src, prefix+name+"="+value.restored, false)
					})
				}
			}
		}
	}
	assertExtensionParseErrors(t, []string{
		"change master to", "change master to unknown_option=1", "change master to master_host=1",
		"change master to master_port='3306'", "change master to master_port=@p", "change master to master_port=1+1",
		"change master to master_port=(1)", "change master to master_port=-1", "change master to master_port=true",
		"change master to master_host=default", "change master to master_host=null", "change master to `MASTER_HOST`='h'",
		"change master to source_compression_algorithm='zstd'", "change master to master_compression_algorithm='zstd'",
		"change master to ignore_server_ids=1", "change master to ignore_server_ids=(1,)", "change master to ignore_server_ids=(,1)",
		"change master to ignore_server_ids=('1')", "change master to ignore_server_ids=(-1)",
		"change master to source_log_pos=0x4", "change master to master_log_pos=0x4", "change master to master_port=x'01'",
		"change master to master_port=0x10000000000000000", "change master to source_heartbeat_period=0x1",
		"change master to require_row_format=2", "change master to require_row_format=0x2", "change master to require_row_format=true",
		"change master to require_table_primary_key_check='ON'", "change master to require_table_primary_key_check=1",
		"change master to assign_gtids_to_anonymous_transactions='local'", "change master to assign_gtids_to_anonymous_transactions='not-a-uuid'",
		"change master to privilege_checks_user=current_user", "change master to privilege_checks_user=current_user()",
		"change master to privilege_checks_user=''@'%'", "change master to privilege_checks_user=1",
		"change master to master_delay=2147483648", "change master to source_delay=2147483648",
		"change master to source_heartbeat_period=4294968", "change master to master_heartbeat_period=4294968",
		"change master to source_host='host\nname'", "change replication source to master_host='host\nname'",
		"change replication source to source_host='h' for channel 'c\nname'",
		"change replication source to source_password='" + strings.Repeat("p", 33) + "'",
	})
	for _, name := range []string{"GTID_ONLY", "SOURCE_CONNECTION_AUTO_FAILOVER"} {
		for _, value := range []string{"2", "1.0", "1e0", "+1", "true", "on", "'1'", "default", "x'01'", "0x2", "18446744073709551616"} {
			assertExtensionParseErrors(t, []string{"change master to " + name + "=" + value})
		}
	}
}

// https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/change-master-to
func TestReplicationMariaDBOptions(t *testing.T) {
	defaults := strings.FieldsSeq("MASTER_CONNECT_RETRY MASTER_RETRY_COUNT MASTER_HEARTBEAT_PERIOD MASTER_SSL MASTER_SSL_CA " +
		"MASTER_SSL_CAPATH MASTER_SSL_CERT MASTER_SSL_CIPHER MASTER_SSL_KEY MASTER_SSL_VERIFY_SERVER_CERT MASTER_SSL_CRL MASTER_SSL_CRLPATH MASTER_USE_GTID")
	for name := range defaults {
		src := "change master 'c' to " + strings.ToLower(name) + "=default"
		t.Run(src, func(t *testing.T) {
			RunRestoreTest(t, src, "CHANGE MASTER 'c' TO "+name+"=DEFAULT", false)
		})
	}
	cases := []struct{ src, restored string }{
		{"change master to master_use_gtid=no", "CHANGE MASTER TO MASTER_USE_GTID=NO"},
		{"change master to master_use_gtid=current_pos", "CHANGE MASTER TO MASTER_USE_GTID=CURRENT_POS"},
		{"change master to master_use_gtid=slave_pos", "CHANGE MASTER TO MASTER_USE_GTID=SLAVE_POS"},
		{"change master to master_use_gtid=replica_pos", "CHANGE MASTER TO MASTER_USE_GTID=REPLICA_POS"},
		{"change master '' to master_ssl=true, master_ssl_verify_server_cert=false", "CHANGE MASTER '' TO MASTER_SSL=TRUE, MASTER_SSL_VERIFY_SERVER_CERT=FALSE"},
		{"change master 'c' to master_demote_to_slave=2, master_demote_to_replica=true", "CHANGE MASTER 'c' TO MASTER_DEMOTE_TO_SLAVE=2, MASTER_DEMOTE_TO_REPLICA=TRUE"},
		{"change master 'c' to master_port=+3306, master_log_pos=+4.5, master_retry_count=18446744073709551615", "CHANGE MASTER 'c' TO MASTER_PORT=+3306, MASTER_LOG_POS=+4, MASTER_RETRY_COUNT=18446744073709551615"},
		{"change master to master_use_gtid=slave_pos, do_domain_ids=(0, +2, 0x3), ignore_domain_ids=() for channel ''", "CHANGE MASTER TO MASTER_USE_GTID=SLAVE_POS, DO_DOMAIN_IDS=(0, +2, 3), IGNORE_DOMAIN_IDS=() FOR CHANNEL ''"},
		{"change master to do_domain_ids=(), ignore_domain_ids=(1, 2)", "CHANGE MASTER TO DO_DOMAIN_IDS=(), IGNORE_DOMAIN_IDS=(1, 2)"},
		{"change master 'c' to master_heartbeat_period=+0.001", "CHANGE MASTER 'c' TO MASTER_HEARTBEAT_PERIOD=+0.001"},
		{"change master 'c' to master_heartbeat_period=4294967.295", "CHANGE MASTER 'c' TO MASTER_HEARTBEAT_PERIOD=4294967.295"},
		{"change master 'c' to master_heartbeat_period=4294967.2950000000000000", "CHANGE MASTER 'c' TO MASTER_HEARTBEAT_PERIOD=4294967.2950000000000000"},
		{"change master to master_host='host\nname'", "CHANGE MASTER TO MASTER_HOST='host\nname'"},
		{"change master to master_password='" + strings.Repeat("p", 33) + "'", "CHANGE MASTER TO MASTER_PASSWORD='" + strings.Repeat("p", 33) + "'"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) { RunRestoreTest(t, tc.src, tc.restored, false) })
	}
	assertExtensionParseErrors(t, []string{
		"change replication source to master_use_gtid=slave_pos", "change replication source to do_domain_ids=(1)",
		"change replication source to master_ssl=default", "change replication source to master_ssl=true",
		"change master 'c' to source_host='h'", "change master 'c' to master_bind='eth0'", "change master 'c' to gtid_only=1",
		"change master 'c' to master_retry_count=0x10", "change master '' to master_host='h' for channel ''",
		"change master to master_use_gtid=slave_pos, source_host='h'", "change master to master_ssl=true, source_host='h'",
		"change master to master_ssl=default, master_auto_position=1", "change master to source_ssl=true",
		"change master to source_port=+3306", "change master to source_connect_retry=default",
		"change master to master_use_gtid='slave_pos'", "change master to master_use_gtid=1",
		"change master to master_use_gtid=no, master_use_gtid=slave_pos", "change master to master_demote_to_slave=default",
		"change master to master_use_gtid=no, master_retry_count=0x10", "change master 'c' to master_heartbeat_period=4294967.296",
		"change master 'c' to master_heartbeat_period=4294967.2950000000000001",
		"change master to master_heartbeat_period=4294967.2950000000000001",
		"change master to master_port=+0x1", "change master to ignore_server_ids=(+0x1)",
	})
}

// MySQL tests the heartbeat after conversion to float; MariaDB compares exact decimals.
// https://github.com/mysql/mysql-server/blob/8.4/sql/sql_yacc.yy
// https://github.com/MariaDB/server/blob/12.3/sql/rpl_master_info_file.h
func TestReplicationHeartbeatBoundaries(t *testing.T) {
	for _, prefix := range []string{"CHANGE MASTER TO ", "CHANGE REPLICATION SOURCE TO "} {
		src := prefix + "SOURCE_HEARTBEAT_PERIOD=4294967.25"
		t.Run(src, func(t *testing.T) { RunRestoreTest(t, src, src, false) })
		assertExtensionParseErrors(t, []string{
			prefix + "SOURCE_HEARTBEAT_PERIOD=4294967.250001",
			prefix + "SOURCE_HEARTBEAT_PERIOD=4294967.295",
			prefix + "SOURCE_HEARTBEAT_PERIOD=-1",
			prefix + "SOURCE_HEARTBEAT_PERIOD=+1",
		})
	}
}

// Integer-prefix options must not evaluate an exponent or overflow a decimal first.
// https://github.com/mysql/mysql-server/blob/8.4/sql/sql_yacc.yy
// https://github.com/MariaDB/server/blob/12.3/sql/sql_yacc.yy
func TestReplicationNumericLexerBoundaries(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"change master to source_port=1e309", "CHANGE MASTER TO SOURCE_PORT=1"},
		{"change replication source to source_port=1e9999", "CHANGE REPLICATION SOURCE TO SOURCE_PORT=1"},
		{"change master 'c' to master_retry_count=+1e9999", "CHANGE MASTER 'c' TO MASTER_RETRY_COUNT=+1"},
		{"change master to source_port=" + strings.Repeat("9", 100), "CHANGE MASTER TO SOURCE_PORT=18446744073709551615"},
		{"change master to source_port=" + strings.Repeat("9", 100) + ".5", "CHANGE MASTER TO SOURCE_PORT=18446744073709551615"},
		{"change master to ignore_server_ids=(1e309, .5e309)", "CHANGE MASTER TO IGNORE_SERVER_IDS=(1, 0)"},
		{"change master to source_heartbeat_period=0.5, source_port=1e309", "CHANGE MASTER TO SOURCE_HEARTBEAT_PERIOD=0.5, SOURCE_PORT=1"},
		{"change master to source_port=1e309, source_heartbeat_period=0.5", "CHANGE MASTER TO SOURCE_PORT=1, SOURCE_HEARTBEAT_PERIOD=0.5"},
		{"start replica until source_log_file='bin.1', source_log_pos=1e309", "START REPLICA UNTIL SOURCE_LOG_FILE='bin.1', SOURCE_LOG_POS=1"},
		{"change master to source_port=1e309; select 12e2", "CHANGE MASTER TO SOURCE_PORT=1; SELECT 1.2e+03"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) { RunRestoreTest(t, tc.src, tc.restored, false) })
	}
	p := New()
	for _, tc := range []struct {
		src   string
		valid bool
	}{
		{"change master to source_port=1e309", true},
		{"start replica until sql_after_mts_gaps", true},
		{"change master to source_host=1e309", false},
		{"start replica until source_log_pos=1e309", false},
	} {
		// Reuse after both successful parsing and a syntax error must reset the mode.
		_, err := p.ParseOneStmt(tc.src, "", "")
		if (err == nil) != tc.valid {
			t.Fatalf("parse %s: err=%v, valid=%v", tc.src, err, tc.valid)
		}
		if _, err := p.ParseOneStmt("select 1e309", "", ""); err == nil {
			t.Fatal("replication conversion leaked into ordinary expressions")
		}
	}
	assertExtensionParseErrors(t, []string{
		"change master to gtid_only=0e309", "change master to source_connection_auto_failover=1e309",
		"change master to gtid_only=" + strings.Repeat("9", 100),
		"change master to source_heartbeat_period=1e309", "change master to master_heartbeat_period=1e309",
		"change master to source_port=1e309; select 1e309",
		"start replica until sql_after_mts_gaps; select 1e309",
	})
}

// https://dev.mysql.com/doc/refman/8.4/en/start-replica.html
// https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/start-replica
// https://mariadb.com/docs/server/reference/sql-statements/administrative-sql-statements/replication-statements/stop-replica
func TestReplicationControlClauses(t *testing.T) {
	cases := []struct{ src, restored string }{
		{"start slave io_thread, sql_thread", "START SLAVE IO_THREAD, SQL_THREAD"},
		{"stop replica sql_thread, io_thread, sql_thread", "STOP REPLICA SQL_THREAD, IO_THREAD, SQL_THREAD"},
		{"start replica until sql_after_mts_gaps", "START REPLICA UNTIL SQL_AFTER_MTS_GAPS"},
		{"start replica until source_log_pos=4, source_log_file='bin.1' for channel ''", "START REPLICA UNTIL SOURCE_LOG_POS=4, SOURCE_LOG_FILE='bin.1' FOR CHANNEL ''"},
		{"start slave until relay_log_file='relay.1', relay_log_pos=0x4", "START SLAVE UNTIL RELAY_LOG_FILE='relay.1', RELAY_LOG_POS=4"},
		{"start replica until master_log_file='bin.1', source_log_pos=4", "START REPLICA UNTIL MASTER_LOG_FILE='bin.1', SOURCE_LOG_POS=4"},
		{"start replica until source_log_file='bin.1', master_log_pos=4", "START REPLICA UNTIL SOURCE_LOG_FILE='bin.1', MASTER_LOG_POS=4"},
		{"start slave until master_log_file='bin.1', master_log_file='bin.2', master_log_pos=4", "START SLAVE UNTIL MASTER_LOG_FILE='bin.1', MASTER_LOG_FILE='bin.2', MASTER_LOG_POS=4"},
		{"start replica until sql_before_gtids='aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee:1-2' for channel 'c'", "START REPLICA UNTIL SQL_BEFORE_GTIDS='aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee:1-2' FOR CHANNEL 'c'"},
		{"start replica until sql_after_gtids=''", "START REPLICA UNTIL SQL_AFTER_GTIDS=''"},
		{"start replica '' sql_thread until master_gtid_pos='0-1-4'", "START REPLICA '' SQL_THREAD UNTIL MASTER_GTID_POS='0-1-4'"},
		{"start slave 'c' until master_log_file='bin.1', master_log_pos=+4", "START SLAVE 'c' UNTIL MASTER_LOG_FILE='bin.1', MASTER_LOG_POS=+4"},
		{"start replica sql_thread for channel 'c' until master_log_file='bin.1', master_log_pos=4", "START REPLICA SQL_THREAD FOR CHANNEL 'c' UNTIL MASTER_LOG_FILE='bin.1', MASTER_LOG_POS=4"},
		{"start slave for channel '' until sql_after_gtids='0-1-4'", "START SLAVE FOR CHANNEL '' UNTIL SQL_AFTER_GTIDS='0-1-4'"},
		{"start slave 'c' io_thread", "START SLAVE 'c' IO_THREAD"},
		{"stop replica '' io_thread, sql_thread", "STOP REPLICA '' IO_THREAD, SQL_THREAD"},
		{"stop slave for channel ''", "STOP SLAVE FOR CHANNEL ''"},
		{"start all slaves", "START ALL SLAVES"},
		{"start all replicas io_thread, sql_thread", "START ALL REPLICAS IO_THREAD, SQL_THREAD"},
		{"stop all slaves io_thread", "STOP ALL SLAVES IO_THREAD"},
		{"stop all replicas sql_thread", "STOP ALL REPLICAS SQL_THREAD"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) { RunRestoreTest(t, tc.src, tc.restored, false) })
	}
	options := []string{"USER='Repl'", "PASSWORD='Secret'", "DEFAULT_AUTH='caching_sha2_password'", "PLUGIN_DIR='/Plugins'"}
	for mask := range 16 {
		for _, thread := range []string{"", " IO_THREAD", " SQL_THREAD, IO_THREAD"} {
			for _, until := range []string{"", " UNTIL SQL_AFTER_MTS_GAPS"} {
				src := "START REPLICA" + thread + until
				for i, option := range options {
					if mask&(1<<i) != 0 {
						src += " " + option
					}
				}
				src += " FOR CHANNEL ''"
				t.Run(src, func(t *testing.T) { RunRestoreTest(t, src, src, false) })
			}
		}
	}
	for _, option := range options {
		assertExtensionParseErrors(t, []string{"START REPLICA SQL_THREAD " + option})
	}
	assertExtensionParseErrors(t, []string{
		"start replica io_thread sql_thread", "start replica io_thread,", "start replica ,io_thread",
		"start replica until source_log_file='bin.1'", "start replica until source_log_pos=4",
		"start replica until relay_log_file='relay.1'", "start replica until relay_log_pos=4",
		"start replica until source_log_file='bin.1', relay_log_pos=4",
		"start replica until source_log_file='bin.1', source_log_pos=4, relay_log_pos=0",
		"start replica until source_log_file='bin.1', source_log_pos=4, relay_log_file='relay.1', relay_log_pos=4",
		"start replica until unknown_option=1", "start replica until source_host='h'",
		"start replica until sql_after_mts_gaps=1", "start replica until sql_after_mts_gaps, source_log_file='bin.1', source_log_pos=4",
		"start replica until sql_before_gtids='g', sql_after_gtids='g'",
		"start replica until sql_after_gtids='g', source_log_file='bin.1', source_log_pos=4",
		"start replica until sql_before_gtids=@g", "start replica until master_gtid_pos=1",
		"start replica user='u', password='p'", "start replica password='p' user='u'",
		"start replica default_auth='p' password='p'", "start replica plugin_dir='p' default_auth='p'",
		"start replica user=1", "start replica user=@u", "start replica default auth='p'",
		"start replica user='u' until sql_after_mts_gaps", "start replica for channel 'c' user='u'",
		"start replica 'c' user='u'", "start slave '' for channel ''", "stop slave '' for channel ''",
		"start replica 'c' until source_log_file='bin.1', source_log_pos=4", "start replica 'c' until sql_after_mts_gaps",
		"start replica for channel 'c' until source_log_file='bin.1', source_log_pos=4",
		"start replica for channel 'c' until sql_after_mts_gaps", "start replica until master_gtid_pos='0-1-4' for channel 'c'",
		"start all replicas until sql_after_mts_gaps", "start all slaves 'c'", "start all slaves for channel 'c'",
		"stop all replicas for channel 'c'", "stop replica until sql_after_mts_gaps", "stop replica user='u'",
	})
}

func TestReplicationASTPresence(t *testing.T) {
	change := mustParseOne(t, "change master '' to master_ssl=default").(*ast.ChangeMasterStmt)
	if !change.HasConnectionName || change.ConnectionName != "" || change.HasChannel {
		t.Fatalf("empty MariaDB connection name was lost: %#v", change)
	}
	change = mustParseOne(t, "change replication source to ignore_server_ids=() for channel ''").(*ast.ChangeMasterStmt)
	list := change.Options[0].Expr.(*ast.ReplicationOptionValue)
	if !change.Replica || !change.HasChannel || change.Channel != "" || list.Values == nil || len(list.Values) != 0 {
		t.Fatalf("empty channel or explicit empty ID list was lost: %#v", change)
	}
	control := mustParseOne(t, "start replica sql_thread for channel '' until master_gtid_pos='0-1-4'").(*ast.ReplicaControlStmt)
	if !control.Start || !control.Replica || !control.HasChannel || !control.ChannelBeforeUntil || len(control.UntilOptions) != 1 || control.UntilOptions[0].Column.Name.O != "MASTER_GTID_POS" {
		t.Fatalf("MariaDB channel/UNTIL ordering was lost: %#v", control)
	}
	control = mustParseOne(t, "start replica until sql_after_mts_gaps user='u'").(*ast.ReplicaControlStmt)
	if !control.UntilAfterGaps || control.UntilOptions != nil || len(control.ConnectionOptions) != 1 || control.ConnectionOptions[0].Column.Name.O != "USER" {
		t.Fatalf("bare stopping condition or connection options were lost: %#v", control)
	}
	control = mustParseOne(t, "stop all replicas").(*ast.ReplicaControlStmt)
	if control.Start || !control.All || !control.Replica {
		t.Fatalf("unexpected STOP ALL flags: %#v", control)
	}
}

func TestReplicationVisitors(t *testing.T) {
	cases := []struct {
		src  string
		want map[string]int
	}{
		{"change replication source to ignore_server_ids=(1,2), require_table_primary_key_check=on, privilege_checks_user='u'@'h'",
			map[string]int{"*ast.ChangeMasterStmt": 1, "*ast.Assignment": 3, "*ast.ColumnName": 3, "*ast.ReplicationOptionValue": 3, "*driver.ValueExpr": 2}},
		{"start replica until source_log_file='bin.1', source_log_pos=4 user='u' password='p' default_auth='a' plugin_dir='d'",
			map[string]int{"*ast.ReplicaControlStmt": 1, "*ast.Assignment": 6, "*ast.ColumnName": 6, "*driver.ValueExpr": 6}},
		{"change master to master_use_gtid=default, master_ssl_ca=default",
			map[string]int{"*ast.ChangeMasterStmt": 1, "*ast.Assignment": 2, "*ast.ColumnName": 2, "*ast.DefaultExpr": 2}},
		{"change master 'c' to master_port=+3306, do_domain_ids=(+1)",
			map[string]int{"*ast.ChangeMasterStmt": 1, "*ast.Assignment": 2, "*ast.ColumnName": 2, "*ast.UnaryOperationExpr": 2, "*ast.ReplicationOptionValue": 1, "*driver.ValueExpr": 2}},
		{"start replica until sql_after_mts_gaps", map[string]int{"*ast.ReplicaControlStmt": 1}},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			stmt := mustParseOne(t, tc.src)
			entered, left := map[string]int{}, map[string]int{}
			_, ok := stmt.Accept(extensionVisitor{
				enter: func(n ast.Node) (ast.Node, bool) { entered[fmt.Sprintf("%T", n)]++; return n, false },
				leave: func(n ast.Node) (ast.Node, bool) { left[fmt.Sprintf("%T", n)]++; return n, true },
			})
			if !ok || !reflect.DeepEqual(entered, tc.want) || !reflect.DeepEqual(left, tc.want) {
				t.Fatalf("visitor counts: ok=%v, enter=%v, leave=%v; want %v", ok, entered, left, tc.want)
			}
			visits, leaves := 0, 0
			_, ok = stmt.Accept(extensionVisitor{
				enter: func(n ast.Node) (ast.Node, bool) { visits++; return n, true },
				leave: func(n ast.Node) (ast.Node, bool) { leaves++; return n, true },
			})
			if !ok || visits != 1 || leaves != 1 {
				t.Fatalf("skip-children visitor: ok=%v, enter=%d, leave=%d", ok, visits, leaves)
			}
			for target := range tc.want {
				t.Run("abort/"+target, func(t *testing.T) {
					_, ok := mustParseOne(t, tc.src).Accept(extensionVisitor{leave: func(n ast.Node) (ast.Node, bool) {
						return n, fmt.Sprintf("%T", n) != target
					}})
					if ok {
						t.Fatalf("abort at %s was not propagated", target)
					}
				})
			}
		})
	}
	stmt := mustParseOne(t, cases[0].src)
	replaced, ok := stmt.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch n := n.(type) {
		case *ast.ChangeMasterStmt:
			copy := *n
			copy.HasChannel, copy.Channel = true, "replacement"
			return &copy, false
		case *ast.ReplicationOptionValue:
			copy := *n
			if n.Keyword != "" {
				copy.Keyword = "OFF"
			}
			if n.User != nil {
				copy.User = &auth.UserIdentity{Username: "v", Hostname: "%"}
			}
			return &copy, false
		case ast.ValueExpr:
			return ast.NewValueExpr(int64(9)), false
		}
		return n, false
	}})
	if !ok {
		t.Fatal("replacement visitor aborted")
	}
	// Enter can replace the root; callers must use the returned node.
	stmt = replaced.(ast.StmtNode)
	if restoreStmt(t, stmt) != "CHANGE REPLICATION SOURCE TO IGNORE_SERVER_IDS=(9, 9), REQUIRE_TABLE_PRIMARY_KEY_CHECK=OFF, PRIVILEGE_CHECKS_USER='v'@'%' FOR CHANNEL 'replacement'" {
		t.Fatalf("replacement visitor missed replication values: %s", restoreStmt(t, stmt))
	}
	start := mustParseOne(t, "start replica until source_log_file='bin.1', source_log_pos=4 user='u'")
	_, ok = start.Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		if n, match := n.(*ast.Assignment); match && n.Column.Name.O == "USER" {
			copy := *n
			copy.Column = &ast.ColumnName{Name: model.NewCIStr("PASSWORD")}
			copy.Expr = ast.NewValueExpr("replacement")
			return &copy, false
		}
		return n, false
	}})
	if !ok || restoreStmt(t, start) != "START REPLICA UNTIL SOURCE_LOG_FILE='bin.1', SOURCE_LOG_POS=4 PASSWORD='replacement'" {
		t.Fatalf("replacement visitor missed START connection options: %s", restoreStmt(t, start))
	}
	values := 0
	_, ok = mustParseOne(t, cases[0].src).Accept(extensionVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		if _, match := n.(ast.ValueExpr); match {
			values++
		}
		_, skip := n.(*ast.ReplicationOptionValue)
		return n, skip
	}})
	if !ok || values != 0 {
		t.Fatalf("skipping replication value children visited %d literals", values)
	}
}

func TestReplicationValueFormatting(t *testing.T) {
	cases := []struct {
		value *ast.ReplicationOptionValue
		want  string
	}{
		{&ast.ReplicationOptionValue{Keyword: "ON"}, "ON"},
		{&ast.ReplicationOptionValue{Values: []ast.ExprNode{}}, "()"},
		{&ast.ReplicationOptionValue{Values: []ast.ExprNode{ast.NewValueExpr(int64(1)), ast.NewValueExpr(uint64(18446744073709551615))}}, "(1, 18446744073709551615)"},
		{&ast.ReplicationOptionValue{User: &auth.UserIdentity{Username: "u'v", Hostname: ""}}, "'u''v'@''"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			var out strings.Builder
			tc.value.Format(&out)
			if out.String() != tc.want {
				t.Fatalf("Format: got %q, want %q", out.String(), tc.want)
			}
			out.Reset()
			if err := tc.value.Restore(restoreformat.NewRestoreCtx(restoreformat.DefaultRestoreFlags, &out)); err != nil || out.String() != tc.want {
				t.Fatalf("Restore: got %q, err=%v; want %q", out.String(), err, tc.want)
			}
		})
	}
	var out strings.Builder
	if err := (&ast.ReplicationOptionValue{}).Restore(restoreformat.NewRestoreCtx(restoreformat.DefaultRestoreFlags, &out)); err == nil {
		t.Fatal("empty replication value should fail Restore")
	}
	for _, stmt := range []ast.StmtNode{
		&ast.ChangeMasterStmt{Options: []*ast.Assignment{nil}},
		&ast.ChangeMasterStmt{Options: []*ast.Assignment{{Expr: ast.NewValueExpr(int64(1))}}},
		&ast.ReplicaControlStmt{Start: true, UntilOptions: []*ast.Assignment{{Column: &ast.ColumnName{Name: model.NewCIStr("SOURCE_LOG_POS")}}}},
		&ast.ReplicaControlStmt{Start: true, ConnectionOptions: []*ast.Assignment{nil}},
	} {
		t.Run(fmt.Sprintf("%#v", stmt), func(t *testing.T) {
			out.Reset()
			if err := stmt.Restore(restoreformat.NewRestoreCtx(restoreformat.DefaultRestoreFlags, &out)); err == nil {
				t.Fatal("incomplete replication assignment should fail Restore")
			}
		})
	}
}

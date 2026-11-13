package ast

import "strings"

var builtinFuncs = map[string]struct{}{
	"cast": {}, "values": {}, "bit_count": {}, "coalesce": {}, "greatest": {}, "least": {},
	"interval": {}, "abs": {}, "acos": {}, "asin": {}, "atan": {}, "atan2": {},
	"ceil": {}, "ceiling": {}, "conv": {}, "cos": {}, "cot": {}, "crc32": {},
	"degrees": {}, "exp": {}, "floor": {}, "ln": {}, "log": {}, "log2": {},
	"log10": {}, "pi": {}, "pow": {}, "power": {}, "radians": {}, "rand": {},
	"round": {}, "sign": {}, "sin": {}, "sqrt": {}, "tan": {}, "truncate": {},
	"adddate": {}, "addtime": {}, "convert_tz": {}, "curdate": {}, "current_date": {}, "current_time": {},
	"current_timestamp": {}, "curtime": {}, "date": {}, "date_add": {}, "date_format": {}, "date_sub": {},
	"datediff": {}, "day": {}, "dayname": {}, "dayofmonth": {}, "dayofweek": {}, "dayofyear": {},
	"extract": {}, "from_days": {}, "from_unixtime": {}, "get_format": {}, "hour": {}, "localtime": {},
	"localtimestamp": {}, "makedate": {}, "maketime": {}, "microsecond": {}, "minute": {}, "month": {},
	"monthname": {}, "now": {}, "period_add": {}, "period_diff": {}, "quarter": {}, "sec_to_time": {},
	"second": {}, "str_to_date": {}, "subdate": {}, "subtime": {}, "sysdate": {}, "time": {},
	"time_format": {}, "time_to_sec": {}, "timediff": {}, "timestamp": {}, "timestampadd": {}, "timestampdiff": {},
	"to_days": {}, "to_seconds": {}, "unix_timestamp": {}, "utc_date": {}, "utc_time": {}, "utc_timestamp": {},
	"week": {}, "weekday": {}, "weekofyear": {}, "year": {}, "yearweek": {}, "last_day": {},
	"ascii": {}, "bin": {}, "concat": {}, "concat_ws": {}, "convert": {}, "elt": {},
	"export_set": {}, "field": {}, "format": {}, "from_base64": {}, "insert_func": {}, "instr": {},
	"lcase": {}, "left": {}, "length": {}, "load_file": {}, "locate": {}, "lower": {},
	"lpad": {}, "ltrim": {}, "make_set": {}, "mid": {}, "oct": {}, "ord": {},
	"position": {}, "quote": {}, "repeat": {}, "replace": {}, "reverse": {}, "right": {},
	"rtrim": {}, "space": {}, "strcmp": {}, "substring": {}, "substr": {}, "substring_index": {},
	"to_base64": {}, "trim": {}, "upper": {}, "ucase": {}, "hex": {}, "unhex": {},
	"rpad": {}, "bit_length": {}, "char_func": {}, "char_length": {}, "character_length": {}, "find_in_set": {},
	"benchmark": {}, "charset": {}, "coercibility": {}, "collation": {}, "connection_id": {}, "current_user": {},
	"current_role": {}, "database": {}, "found_rows": {}, "last_insert_id": {}, "row_count": {}, "schema": {},
	"session_user": {}, "system_user": {}, "user": {}, "version": {}, "if": {}, "ifnull": {},
	"nullif": {}, "any_value": {}, "default_func": {}, "inet_aton": {}, "inet_ntoa": {}, "inet6_aton": {},
	"inet6_ntoa": {}, "is_free_lock": {}, "is_ipv4": {}, "is_ipv4_compat": {}, "is_ipv4_mapped": {}, "is_ipv6": {},
	"is_used_lock": {}, "master_pos_wait": {}, "name_const": {}, "release_all_locks": {}, "sleep": {}, "uuid": {},
	"uuid_short": {}, "get_lock": {}, "release_lock": {}, "aes_decrypt": {}, "aes_encrypt": {}, "compress": {},
	"decode": {}, "des_decrypt": {}, "des_encrypt": {}, "encode": {}, "encrypt": {}, "md5": {},
	"old_password": {}, "password_func": {}, "random_bytes": {}, "sha1": {}, "sha": {}, "sha2": {},
	"uncompress": {}, "uncompressed_length": {}, "validate_password_strength": {}, "json_type": {}, "json_extract": {}, "json_unquote": {},
	"json_array": {}, "json_object": {}, "json_merge": {}, "json_set": {}, "json_insert": {}, "json_replace": {},
	"json_remove": {}, "json_contains": {}, "json_contains_path": {}, "json_valid": {}, "json_array_append": {}, "json_array_insert": {},
	"json_merge_patch": {}, "json_merge_preserve": {}, "json_pretty": {}, "json_quote": {}, "json_search": {}, "json_storage_size": {},
	"json_depth": {}, "json_keys": {}, "json_length": {}, "count": {}, "sum": {}, "avg": {},
	"firstrow": {}, "max": {}, "min": {}, "group_concat": {}, "bit_or": {}, "bit_xor": {},
	"bit_and": {}, "var_pop": {}, "var_samp": {}, "stddev_pop": {}, "stddev_samp": {}, "std": {},
	"stddev": {}, "variance": {}, "row_number": {},
	"rank": {}, "dense_rank": {}, "cume_dist": {}, "percent_rank": {}, "ntile": {}, "lead": {},
	"lag": {}, "first_value": {}, "last_value": {}, "nth_value": {}, "cube": {}, "rollup": {},
	"grouping": {}, "json_arrayagg": {}, "json_objectagg": {},
	"dateliteral": {}, "timeliteral": {}, "timestampliteral": {},
	"tidb_version": {}, "tidb_is_ddl_owner": {}, "tidb_parse_tso": {},
}

func isBuiltinFunc(name string) bool {
	_, ok := builtinFuncs[strings.ToLower(name)]
	return ok
}

func normalizeFuncName(name string) string {
	if isBuiltinFunc(name) {
		return strings.ToUpper(name)
	}
	return name
}

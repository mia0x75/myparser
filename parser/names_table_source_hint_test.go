package parser

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

func TestTableSourceHintRoundtripMatrix(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, sql, canonical string }{
		{"select_plain", "SELECT * FROM t", "SELECT * FROM `t`"},
		{"select_alias", "SELECT * FROM t AS x", "SELECT * FROM `t` AS `x`"},
		{"select_use", "SELECT * FROM t USE INDEX(i)", "SELECT * FROM `t` USE INDEX (`i`)"},
		{"select_alias_use", "SELECT * FROM t AS x USE INDEX(i)", "SELECT * FROM `t` AS `x` USE INDEX (`i`)"},
		{"select_alias_ignore", "SELECT * FROM t x IGNORE KEY FOR JOIN(i)", "SELECT * FROM `t` AS `x` IGNORE INDEX FOR JOIN (`i`)"},
		{"select_partition_alias_force", "SELECT * FROM t PARTITION(p0,p1) AS x FORCE INDEX FOR ORDER BY(i)", "SELECT * FROM `t` PARTITION(`p0`, `p1`) AS `x` FORCE INDEX FOR ORDER BY (`i`)"},
		{"select_quoted_alias", "SELECT * FROM `db.dot`.`表` AS `x.y` USE INDEX(`i.dot`)", "SELECT * FROM `db.dot`.`表` AS `x.y` USE INDEX (`i.dot`)"},
		{"select_empty_alias", "SELECT * FROM t AS `` FORCE INDEX FOR GROUP BY(i)", "SELECT * FROM `t` AS `` FORCE INDEX FOR GROUP BY (`i`)"},
		{"select_multi_hint", "SELECT * FROM t USE INDEX(i) IGNORE INDEX FOR ORDER BY(j)", "SELECT * FROM `t` USE INDEX (`i`) IGNORE INDEX FOR ORDER BY (`j`)"},
		{"select_alias_empty_hint", "SELECT * FROM t AS x USE INDEX()", "SELECT * FROM `t` AS `x` USE INDEX ()"},
		{"select_join_left", "SELECT * FROM t AS x USE INDEX(i) JOIN s USING(id)", "SELECT * FROM `t` AS `x` USE INDEX (`i`) JOIN `s` USING (`id`)"},
		{"select_join_right", "SELECT * FROM t JOIN s AS y FORCE INDEX(j) ON t.id=y.id", "SELECT * FROM `t` JOIN `s` AS `y` FORCE INDEX (`j`) ON `t`.`id`=`y`.`id`"},
		{"select_comma", "SELECT * FROM t AS x USE INDEX(i),s", "SELECT * FROM (`t` AS `x` USE INDEX (`i`)) JOIN `s`"},
		{"select_subquery", "SELECT * FROM (SELECT a FROM t AS x USE INDEX(i)) AS q", "SELECT * FROM (SELECT `a` FROM `t` AS `x` USE INDEX (`i`)) AS `q`"},
		{"select_union_derived", "SELECT * FROM (SELECT a FROM t UNION SELECT a FROM s) AS q", "SELECT * FROM (SELECT `a` FROM `t` UNION SELECT `a` FROM `s`) AS `q`"},
		{"select_values", "SELECT * FROM (VALUES ROW(1)) AS v(a)", "SELECT * FROM (VALUES ROW(1)) AS `v` (`a`)"},
		{"select_lateral", "SELECT * FROM t,LATERAL (SELECT 1) AS q", "SELECT * FROM (`t`) JOIN LATERAL (SELECT 1) AS `q`"},
		{"select_json", "SELECT * FROM JSON_TABLE('[1]', '$[*]' COLUMNS(a INT PATH '$')) AS jt", "SELECT * FROM JSON_TABLE('[1]', '$[*]' COLUMNS (`a` INT PATH '$')) AS `jt`"},
		{"select_with", "WITH c AS (SELECT a FROM t AS x USE INDEX(i)) SELECT a FROM c", "WITH `c` AS (SELECT `a` FROM `t` AS `x` USE INDEX (`i`)) SELECT `a` FROM `c`"},
		{"update_original", "UPDATE `db.dot`.`表` PARTITION(p0) AS x USE INDEX(idx) SET x.a=?, x.b=2 WHERE x.c=? ORDER BY x.a LIMIT 1 RETURNING x.*", "UPDATE `db.dot`.`表` PARTITION(`p0`) AS `x` USE INDEX (`idx`) SET `x`.`a`=?, `x`.`b`=2 WHERE `x`.`c`=? ORDER BY `x`.`a` LIMIT 1 RETURNING `x`.*"},
		{"update_no_alias", "UPDATE t PARTITION(p0) USE INDEX(i) SET a=1", "UPDATE `t` PARTITION(`p0`) USE INDEX (`i`) SET `a`=1"},
		{"delete_single", "DELETE FROM t PARTITION(p0) USE INDEX(i) WHERE a=1", "DELETE FROM `t` PARTITION(`p0`) USE INDEX (`i`) WHERE `a`=1"},
		{"delete_before", "DELETE x FROM t AS x USE INDEX(i)", "DELETE `x` FROM `t` AS `x` USE INDEX (`i`)"},
		{"delete_using", "DELETE FROM x USING t AS x USE INDEX(i)", "DELETE FROM `x` USING `t` AS `x` USE INDEX (`i`)"},
		{"insert_partition", "INSERT INTO t PARTITION(p0) (a) VALUES(1)", "INSERT INTO `t` PARTITION(`p0`) (`a`) VALUES (1)"},
		{"replace_partition", "REPLACE INTO t PARTITION(p0) (a) VALUES(1)", ""},
		{"load_xml_partition", "LOAD XML INFILE 'f.xml' INTO TABLE t PARTITION(p0)", "LOAD XML INFILE 'f.xml' INTO TABLE `t` PARTITION(`p0`)"},
		{"cache_index_partition", "CACHE INDEX t PARTITION(p0) KEY(i) IN DEFAULT", "CACHE INDEX `t` PARTITION(`p0`) INDEX (`i`) IN DEFAULT"},
		{"load_index_partition", "LOAD INDEX INTO CACHE t PARTITION(p0) KEY(i) IGNORE LEAVES", "LOAD INDEX INTO CACHE `t` PARTITION(`p0`) INDEX (`i`) IGNORE LEAVES"},
		{"reject_hint_before_alias", "SELECT * FROM t USE INDEX(i) AS x", ""},
		{"reject_derived_hint", "SELECT * FROM (SELECT 1) AS q USE INDEX(i)", ""},
		{"reject_base_columns", "SELECT * FROM t AS x(a)", ""},
		{"reject_lateral_hint", "SELECT * FROM LATERAL (SELECT 1) AS q USE INDEX(i)", ""},
		{"reject_lateral_set_hint", "SELECT * FROM LATERAL (SELECT 1 UNION SELECT 2) AS q USE INDEX(i)", ""},
		{"reject_json_hint", "SELECT * FROM JSON_TABLE('[1]', '$' COLUMNS(a INT PATH '$')) AS q USE INDEX(i)", ""},
		{"reject_insert_hint", "INSERT INTO t USE INDEX(i) VALUES(1)", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.canonical == "" {
				if _, err := New().ParseOneStmt(tc.sql, "", ""); err == nil {
					t.Fatal("frozen grammar accepted an unsupported source")
				}
				return
			}
			RunRestoreTest(t, tc.sql, tc.canonical, true)
		})
	}
}

func TestTableSourceHintOwnedSpans(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, sql, source, table string }{
		{"plain", "SELECT * FROM t WHERE a=1", "t", "t"},
		{"alias", "SELECT * FROM t AS x WHERE a=1", "t AS x", "t"},
		{"hint", "SELECT * FROM t USE INDEX() WHERE a=1", "t USE INDEX()", "t"},
		{"partition", "SELECT * FROM t PARTITION(p0) WHERE a=1", "t PARTITION(p0)", "t PARTITION(p0)"},
		{"all", "SELECT * FROM `库`.`表` PARTITION(p0,p1) AS `别` FORCE INDEX(i) WHERE a=?", "`库`.`表` PARTITION(p0,p1) AS `别` FORCE INDEX(i)", "`库`.`表` PARTITION(p0,p1)"},
		{"comments", "SELECT * FROM t /*p*/ PARTITION(p0) /*a*/ AS x /*h*/ USE INDEX(i) /*w*/ WHERE a=1", "t /*p*/ PARTITION(p0) /*a*/ AS x /*h*/ USE INDEX(i)", "t /*p*/ PARTITION(p0)"},
		{"delete_plain", "DELETE FROM t WHERE a=1", "t", "t"},
		{"delete_partition", "DELETE FROM t PARTITION(p0) WHERE a=1", "t PARTITION(p0)", "t PARTITION(p0)"},
		{"delete_hint", "DELETE FROM t USE INDEX() WHERE a=1", "t USE INDEX()", "t"},
		{"delete_both", "DELETE FROM `库`.`表` PARTITION(p0) USE INDEX(i) IGNORE INDEX FOR JOIN(j) WHERE a=? ORDER BY a LIMIT 1", "`库`.`表` PARTITION(p0) USE INDEX(i) IGNORE INDEX FOR JOIN(j)", "`库`.`表` PARTITION(p0)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			n, err := New().ParseOneStmt(tc.sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			start := strings.Index(tc.sql, tc.source)
			if start < 0 {
				t.Fatal("bad source fixture")
			}
			source := locateTableSource(t, n, loc.ByteOffset(start))
			table, ok := source.Source.(*ast.TableName)
			if !ok {
				t.Fatalf("source = %T, want TableName", source.Source)
			}
			wantSource := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(tc.source))}
			wantTable := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(tc.table))}
			if source.Span() != wantSource || table.Span() != wantTable {
				t.Fatalf("source/table spans = %v/%v, want %v/%v", source.Span(), table.Span(), wantSource, wantTable)
			}
			parts := table.Name.Parts
			wantName := loc.Span{Start: parts[0].Span.Start, End: parts[len(parts)-1].Span.End}
			if table.Name.Span != wantName || wantName.End > table.Span().End {
				t.Fatalf("name span = %v, want %v within table", table.Name.Span, wantName)
			}
		})
	}
}

func TestTableSourceHintModifierCombinations(t *testing.T) {
	t.Parallel()
	for _, partition := range []struct{ name, sql, canonical string }{{"no_partition", "", ""}, {"partition", " PARTITION(p0)", " PARTITION(`p0`)"}} {
		for _, alias := range []struct{ name, sql, canonical string }{{"no_alias", "", ""}, {"alias", " AS x", " AS `x`"}} {
			for _, hints := range []struct{ name, sql, canonical string }{
				{"no_hint", "", ""},
				{"one_hint", " USE INDEX(i)", " USE INDEX (`i`)"},
				{"two_hints", " USE INDEX(i) IGNORE INDEX FOR GROUP BY(j)", " USE INDEX (`i`) IGNORE INDEX FOR GROUP BY (`j`)"},
			} {
				t.Run(partition.name+"/"+alias.name+"/"+hints.name, func(t *testing.T) {
					t.Parallel()
					RunRestoreTest(t, "SELECT * FROM t"+partition.sql+alias.sql+hints.sql,
						"SELECT * FROM `t`"+partition.canonical+alias.canonical+hints.canonical, true)
				})
			}
		}
	}
}

func TestTableSourceHintReuseKeepsOwnedData(t *testing.T) {
	t.Parallel()
	p := New()
	sql := "SELECT ? FROM `库.dot`.`表``名` PARTITION(`p.0`) AS `别``名` USE INDEX(`i.0`) IGNORE INDEX FOR JOIN(j) WHERE a=?"
	n, err := p.ParseOneStmt(sql, "", "")
	if err != nil {
		t.Fatal(err)
	}
	source := locateTableSource(t, n, loc.ByteOffset(findFromStart(t, sql)))
	table := source.Source.(*ast.TableName)
	cloneIdent := func(id lex.Ident) lex.Ident {
		id.Name, id.Raw = strings.Clone(id.Name), strings.Clone(id.Raw)
		return id
	}
	alias := cloneIdent(source.Alias)
	parts := append([]lex.Ident(nil), table.Name.Parts...)
	for i := range parts {
		parts[i] = cloneIdent(parts[i])
	}
	cloneNames := func(names []lex.Ident) []lex.Ident {
		result := append([]lex.Ident(nil), names...)
		for i := range result {
			result[i] = cloneIdent(result[i])
		}
		return result
	}
	partitions := append([]lex.Ident(nil), table.PartitionNames...)
	for i := range partitions {
		partitions[i] = cloneIdent(partitions[i])
	}
	hints := make([]ast.IndexHint, len(source.IndexHints))
	for i, hint := range source.IndexHints {
		hints[i] = *hint
		hints[i].IndexNames = cloneNames(hint.IndexNames)
	}
	if len(hints) != 2 || len(partitions) != 1 {
		t.Fatal("source modifiers missing before reuse")
	}
	sourceSpan, tableSpan, nameSpan := source.Span(), table.Span(), table.Name.Span
	var params []*ast.ParamMarkerExpr
	if _, ok := n.Accept(parameterCollector{params: &params}); !ok || len(params) != 2 {
		t.Fatal("parameter traversal lost before parser reuse")
	}
	sort.Slice(params, func(i, j int) bool { return params[i].Span().Start < params[j].Span().Start })
	paramStates := make([]struct {
		index int
		span  loc.Span
	}, len(params))
	searchFrom := 0
	for i, param := range params {
		relative := strings.Index(sql[searchFrom:], "?")
		if relative < 0 {
			t.Fatalf("parameter %d has no source placeholder", i)
		}
		start := searchFrom + relative
		wantSpan := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + 1)}
		if param.Index != i || param.Span() != wantSpan {
			t.Fatalf("parameter %d index/span = %d/%v, want %d/%v", i, param.Index, param.Span(), i, wantSpan)
		}
		paramStates[i].index, paramStates[i].span = param.Index, param.Span()
		searchFrom = start + 1
	}
	canonical := restoreStmt(t, n)
	for range 8 {
		if _, _, err := p.Parse("SELECT * FROM `另一库`.`另一表` AS y FORCE INDEX(k); DELETE FROM s PARTITION(p1) USE INDEX()", "", ""); err != nil {
			t.Fatal(err)
		}
		if _, err := p.ParseOneStmt("SELECT * FROM t USE INDEX(i) AS x", "", ""); err == nil {
			t.Fatal("negative reuse input unexpectedly parsed")
		}
	}
	if source.Alias != alias || !reflect.DeepEqual(table.Name.Parts, parts) || !reflect.DeepEqual(table.PartitionNames, partitions) {
		t.Fatal("source names changed after parser reuse")
	}
	if len(source.IndexHints) != len(hints) {
		t.Fatal("source hint list changed after parser reuse")
	}
	for i, hint := range source.IndexHints {
		if !reflect.DeepEqual(*hint, hints[i]) {
			t.Fatalf("hint %d changed after parser reuse", i)
		}
	}
	if source.Span() != sourceSpan || table.Span() != tableSpan || table.Name.Span != nameSpan || restoreStmt(t, n) != canonical {
		t.Fatal("source spans or restoration changed after parser reuse")
	}
	for i, param := range params {
		if param.Index != paramStates[i].index || param.Span() != paramStates[i].span {
			t.Fatalf("parameter %d index/span changed or confused with byte offsets", i)
		}
	}
	assertSourceIdent(t, sql, source.Alias, "别`名", "`别``名`", lex.QuoteBacktick, false)
	assertSourceIdent(t, sql, table.Name.Parts[0], "库.dot", "`库.dot`", lex.QuoteBacktick, false)
	assertSourceIdent(t, sql, table.Name.Parts[1], "表`名", "`表``名`", lex.QuoteBacktick, false)
}

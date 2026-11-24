package parser

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/loc"
)

func TestDMLRelationUnwrapsOnlyEmptyRoot(t *testing.T) {
	t.Parallel()
	span := loc.Span{Start: 7, End: 19}
	for _, tc := range []struct {
		name   string
		change func(*ast.JoinExpr)
		unwrap bool
	}{
		{"empty_root", func(*ast.JoinExpr) {}, true},
		{"right", func(j *ast.JoinExpr) { j.Right = &ast.TableName{} }, false},
		{"type", func(j *ast.JoinExpr) { j.Type = pkast.JoinTypeInner }, false},
		{"on", func(j *ast.JoinExpr) { j.On = &ast.ColumnRef{} }, false},
		{"using", func(j *ast.JoinExpr) { j.Using = []ast.QualifiedName{{}} }, false},
		{"natural", func(j *ast.JoinExpr) { j.Natural = true }, false},
		{"straight", func(j *ast.JoinExpr) { j.StraightJoin = true }, false},
		{"missing_left", func(j *ast.JoinExpr) { j.Left = nil }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			leaf := &ast.TableSource{}
			inner := &ast.JoinExpr{BaseJoinExpr: pkast.BaseJoinExpr{Left: leaf}}
			root := &ast.JoinExpr{BaseJoinExpr: pkast.BaseJoinExpr{Left: inner}}
			tc.change(root)
			got := dmlRelation(root, span)
			var want ast.ResultSetNode = root
			if tc.unwrap {
				want = inner
			}
			if got != want || got.Span() != span || inner.Left != leaf || leaf.Span() != loc.InvalidSpan() {
				t.Fatalf("relation=%T span=%v: recursive unwrap or lost root metadata", got, got.Span())
			}
		})
	}
	source := &ast.TableSource{}
	if dmlRelation(source, span) != source || source.Span() != span || dmlRelation(nil, span) != nil {
		t.Fatal("source identity or nil relation changed")
	}
}

func TestDMLOriginalSixteenCanonical(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, sql, canonical string }{
		{"select_control", "SELECT /*+ MAX_EXECUTION_TIME(1) */ a FROM t ORDER BY a", "SELECT /*+ MAX_EXECUTION_TIME(1)*/ `a` FROM `t` ORDER BY `a`"},
		{"update_single", "UPDATE `db.dot`.`表` PARTITION(p0) AS x USE INDEX(idx) SET x.a=?, x.b=2 WHERE x.c=? ORDER BY x.a LIMIT 1 RETURNING x.*", "UPDATE `db.dot`.`表` PARTITION(`p0`) AS `x` USE INDEX (`idx`) SET `x`.`a`=?, `x`.`b`=2 WHERE `x`.`c`=? ORDER BY `x`.`a` LIMIT 1 RETURNING `x`.*"},
		{"update_join", "UPDATE t1 LEFT JOIN t2 USING(id) SET t1.a=t2.a WHERE t1.id=1", "UPDATE `t1` LEFT JOIN `t2` USING (`id`) SET `t1`.`a`=`t2`.`a` WHERE `t1`.`id`=1"},
		{"update_comma", "UPDATE t1,t2 SET t1.a=t2.a WHERE t1.id=t2.id", "UPDATE (`t1`) JOIN `t2` SET `t1`.`a`=`t2`.`a` WHERE `t1`.`id`=`t2`.`id`"},
		{"update_derived", "UPDATE (SELECT a FROM t) AS s SET a=1", "UPDATE (SELECT `a` FROM `t`) AS `s` SET `a`=1"},
		{"update_values", "UPDATE (VALUES ROW(1)) AS v(a) SET a=2", "UPDATE (VALUES ROW(1)) AS `v` (`a`) SET `a`=2"},
		{"update_json", "UPDATE JSON_TABLE('[1]', '$[*]' COLUMNS(a INT PATH '$')) AS jt SET a=2", "UPDATE JSON_TABLE('[1]', '$[*]' COLUMNS (`a` INT PATH '$')) AS `jt` SET `a`=2"},
		{"update_with", "WITH c AS (SELECT 1 AS a) UPDATE t JOIN c USING(a) SET t.a=1 RETURNING t.*", "WITH `c` AS (SELECT 1 AS `a`) UPDATE `t` JOIN `c` USING (`a`) SET `t`.`a`=1 RETURNING `t`.*"},
		{"delete_single", "DELETE FROM `db.dot`.`表` PARTITION(p0) USE INDEX(idx) WHERE a=? ORDER BY a LIMIT 1 RETURNING `db.dot`.*", "DELETE FROM `db.dot`.`表` PARTITION(`p0`) USE INDEX (`idx`) WHERE `a`=? ORDER BY `a` LIMIT 1 RETURNING `db.dot`.*"},
		{"delete_before", "DELETE t1,t2 FROM t1 JOIN t2 USING(id) WHERE t1.a=1", "DELETE `t1`,`t2` FROM `t1` JOIN `t2` USING (`id`) WHERE `t1`.`a`=1"},
		{"delete_using", "WITH c AS (SELECT 1 AS a) DELETE FROM t1,t2 USING t1 JOIN t2 USING(id),c WHERE t1.a=c.a ORDER BY t1.a LIMIT 1", "WITH `c` AS (SELECT 1 AS `a`) DELETE FROM `t1`,`t2` USING (`t1` JOIN `t2` USING (`id`)) JOIN `c` WHERE `t1`.`a`=`c`.`a` ORDER BY `t1`.`a` LIMIT 1"},
		{"delete_star", "DELETE t.* FROM t JOIN s USING(a)", "DELETE `t` FROM `t` JOIN `s` USING (`a`)"},
		{"hint_update", "UPDATE /*+ MAX_EXECUTION_TIME(1) */ t SET a=1", "UPDATE /*+ MAX_EXECUTION_TIME(1)*/ `t` SET `a`=1"},
		{"hint_delete", "DELETE /*+ MAX_EXECUTION_TIME(1) */ FROM t WHERE a=1", "DELETE /*+ MAX_EXECUTION_TIME(1)*/ FROM `t` WHERE `a`=1"},
		{"hint_multi", "DELETE /*+ MAX_EXECUTION_TIME(1) */ t FROM t WHERE a=1", "DELETE /*+ MAX_EXECUTION_TIME(1)*/ `t` FROM `t` WHERE `a`=1"},
		{"delete_with", "WITH c AS (SELECT 1 AS a) DELETE FROM t WHERE a IN (SELECT a FROM c) RETURNING a", "WITH `c` AS (SELECT 1 AS `a`) DELETE FROM `t` WHERE `a` IN (SELECT `a` FROM `c`) RETURNING `a`"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			RunRestoreTest(t, tc.sql, tc.canonical, true)
		})
	}
}

func TestDMLStatementAndRelationByteSpans(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, sql, relation string }{
		{"update_ref", "UPDATE `库`.`表` /*a*/ AS x USE INDEX(i) SET a=? WHERE b=? ORDER BY a LIMIT 1 RETURNING x.*", "`库`.`表` /*a*/ AS x USE INDEX(i)"},
		{"update_refs", "UPDATE t1, /*j*/ t2 SET t1.a=t2.a WHERE t1.id=? RETURNING t1.*", "t1, /*j*/ t2"},
		{"delete_single", "DELETE FROM `库`.`表` /*p*/ PARTITION(p0) USE INDEX(i) WHERE a=? ORDER BY a LIMIT 1 RETURNING a", "`库`.`表` /*p*/ PARTITION(p0) USE INDEX(i)"},
		{"delete_before", "DELETE t1,t2 FROM t1 /*j*/ JOIN t2 USING(id) WHERE a=?", "t1 /*j*/ JOIN t2 USING(id)"},
		{"delete_using", "DELETE FROM t1,t2 USING t1 /*j*/ JOIN t2 USING(id) WHERE a=? ORDER BY a LIMIT 1", "t1 /*j*/ JOIN t2 USING(id)"},
		{"with_update", "WITH `公` AS (SELECT 1 AS a) UPDATE t JOIN `公` USING(a) SET t.a=? RETURNING t.*", "t JOIN `公` USING(a)"},
		{"with_delete", "WITH `公` AS (SELECT 1 AS a) DELETE FROM t USING t JOIN `公` USING(a) WHERE t.a=? ORDER BY a LIMIT 1", "t JOIN `公` USING(a)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			prefix := "  /* 外部 */ \n"
			sql := prefix + tc.sql + ";  "
			n := mustParseOne(t, sql)
			want := loc.Span{Start: loc.ByteOffset(len(prefix)), End: loc.ByteOffset(len(prefix) + len(tc.sql))}
			if n.Span() != want {
				t.Fatalf("statement span=%v want %v", n.Span(), want)
			}
			var relation ast.ResultSetNode
			switch n := n.(type) {
			case *ast.UpdateStmt:
				relation = n.Table
				if n.MultipleTable || len(n.Assignments) == 0 {
					t.Fatal("assignment or legacy MultipleTable semantics changed")
				}
			case *ast.DeleteStmt:
				relation = n.Table
				if n.IsMultiTable {
					relation = n.Using
					if n.Table != nil || len(n.Targets) == 0 {
						t.Fatal("multi-delete mixed target and source roles")
					}
				} else if _, ok := n.Table.(*ast.TableSource); !ok || n.Using != nil || len(n.Targets) != 0 {
					t.Fatal("single-delete lost its actual source outer")
				}
			}
			start := strings.Index(sql, tc.relation)
			wantRelation := loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(tc.relation))}
			if start < 0 || relation == nil || relation.Span() != wantRelation {
				t.Fatalf("relation=%T span=%v want %v", relation, relation.Span(), wantRelation)
			}
		})
	}
	parts := []string{"UPDATE `一` SET a=?", "DELETE FROM `二` WHERE b=?"}
	sql := " /*头*/ " + parts[0] + ";\n /*中*/ " + parts[1] + "; "
	stmts, _, err := New().Parse(sql, "", "")
	if err != nil || len(stmts) != 2 {
		t.Fatalf("multi-statement parse: %v", err)
	}
	for i, n := range stmts {
		start := strings.Index(sql, parts[i])
		if n.Span() != (loc.Span{Start: loc.ByteOffset(start), End: loc.ByteOffset(start + len(parts[i]))}) {
			t.Fatalf("statement %d span=%v", i, n.Span())
		}
	}
}

func TestDMLSourceAndTargetRoles(t *testing.T) {
	t.Parallel()
	for _, sql := range []string{"DELETE FROM t", "DELETE FROM t PARTITION(p0)", "DELETE FROM t USE INDEX(i)"} {
		t.Run(sql, func(t *testing.T) {
			n := mustParseOne(t, sql).(*ast.DeleteStmt)
			source, ok := n.Table.(*ast.TableSource)
			if !ok || source == nil || n.Using != nil || n.IsMultiTable || n.BeforeFrom || len(n.Targets) != 0 {
				t.Fatalf("single table role changed: %#v", n)
			}
			if _, ok := source.Source.(*ast.TableName); !ok {
				t.Fatalf("single-delete source=%T", source.Source)
			}
		})
	}
	for _, tc := range []struct {
		name, sql string
		before    bool
	}{
		{"before", "DELETE `d.dot`.`t`,s.* FROM `d.dot`.`t` JOIN s USING(a)", true},
		{"using", "DELETE FROM `d.dot`.`t`,s.* USING `d.dot`.`t` JOIN s USING(a)", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := mustParseOne(t, tc.sql).(*ast.DeleteStmt)
			join, ok := n.Using.(*ast.JoinExpr)
			if n.Table != nil || !n.IsMultiTable || n.BeforeFrom != tc.before || len(n.Targets) != 2 || !ok || len(join.Using) != 1 {
				t.Fatalf("multi-delete role/target loss: %#v", n)
			}
			if len(n.Targets[0].Name.Parts) != 2 || n.Targets[0].Name.Parts[0].Name != "d.dot" || n.Targets[1].Name.Parts[0].Name != "s" {
				t.Fatal("qualified target segments or original order lost")
			}
			var events []ast.Node
			_, ok = n.Accept(extensionVisitor{enter: func(node ast.Node) (ast.Node, bool) {
				events = append(events, node)
				return node, false
			}})
			if !ok {
				t.Fatal("visitor stopped")
			}
			index := func(want ast.Node) int {
				for i, node := range events {
					if node == want {
						return i
					}
				}
				return -1
			}
			if index(n.Using) < 1 || index(n.Targets[0]) <= index(n.Using) || index(n.Targets[1]) <= index(n.Targets[0]) {
				t.Fatal("source-before-target visitor order changed")
			}
		})
	}
}

func TestDMLNestedRelationCombinations(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, sql         string
		joins, sources    int
		natural, straight bool
	}{
		{"natural_left", "UPDATE t NATURAL LEFT JOIN s SET t.a=1", 1, 2, true, false},
		{"straight", "UPDATE t STRAIGHT_JOIN s ON t.a=s.a SET t.a=1", 1, 2, false, true},
		{"right", "UPDATE t RIGHT JOIN s USING(a) SET t.a=1", 1, 2, false, false},
		{"right_group", "UPDATE t LEFT JOIN (s JOIN q ON s.a=q.a) ON t.a=s.a SET t.a=1", 2, 3, false, false},
		{"left_group", "UPDATE (t LEFT JOIN s ON t.a=s.a) JOIN q ON s.a=q.a SET t.a=1", 2, 3, false, false},
		{"comma_chain", "UPDATE t,s,q SET t.a=s.a WHERE s.a=q.a", 3, 3, false, false},
		{"lateral_select", "UPDATE t,LATERAL (SELECT 1 AS a) AS q SET t.a=q.a", 2, 2, false, false},
		{"lateral_set", "UPDATE t,LATERAL (SELECT 1 AS a UNION SELECT 2) AS q SET t.a=q.a", 2, 2, false, false},
		{"derived_set", "UPDATE (SELECT 1 AS a UNION SELECT 2) AS q SET a=1", 0, 1, false, false},
		{"delete_group", "DELETE x,y FROM t AS x LEFT JOIN (s AS y JOIN q ON y.a=q.a) ON x.a=y.a", 2, 3, false, false},
		{"delete_lateral", "DELETE FROM x USING t AS x,LATERAL (SELECT 1) AS q", 2, 2, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			n := mustParseOne(t, tc.sql)
			var joins, sources int
			var natural, straight bool
			_, ok := n.Accept(extensionVisitor{enter: func(node ast.Node) (ast.Node, bool) {
				switch node := node.(type) {
				case *ast.JoinExpr:
					joins++
					natural = natural || node.Natural
					straight = straight || node.StraightJoin
				case *ast.TableSource:
					sources++
				}
				return node, false
			}})
			if !ok || joins != tc.joins || sources != tc.sources || natural != tc.natural || straight != tc.straight {
				t.Fatalf("relation shape: joins=%d sources=%d natural=%v straight=%v", joins, sources, natural, straight)
			}
			var root ast.ResultSetNode
			switch node := n.(type) {
			case *ast.UpdateStmt:
				root = node.Table
			case *ast.DeleteStmt:
				root = node.Using
			}
			if tc.name == "right_group" || tc.name == "delete_group" {
				join := root.(*ast.JoinExpr)
				_, left := join.Left.(*ast.TableSource)
				_, right := join.Right.(*ast.JoinExpr)
				if !left || !right {
					t.Fatal("right-associated join group was flattened or reordered")
				}
			}
			if tc.name == "left_group" {
				join := root.(*ast.JoinExpr)
				_, left := join.Left.(*ast.JoinExpr)
				_, right := join.Right.(*ast.TableSource)
				if !left || !right {
					t.Fatal("left-associated join group was flattened or reordered")
				}
			}
			canonical := restoreStmt(t, n)
			again := mustParseOne(t, canonical)
			if restoreStmt(t, again) != canonical {
				t.Fatal("relation parentheses or association is not canonical-stable")
			}
		})
	}
}

func TestDMLKeepsParametersAndASTAcrossReuse(t *testing.T) {
	t.Parallel()
	for _, sql := range []string{
		"UPDATE `库.dot`.`表``名` PARTITION(p0) AS `别` USE INDEX(i) SET a=?,b=? WHERE c=? ORDER BY a LIMIT ? RETURNING ?",
		"WITH c AS (SELECT ?) DELETE FROM `库.dot`.`表``名` PARTITION(p0) USE INDEX(i) WHERE a=? ORDER BY b LIMIT ? RETURNING ?",
	} {
		t.Run(sql, func(t *testing.T) {
			p := New()
			n, err := p.ParseOneStmt(sql, "", "")
			if err != nil {
				t.Fatal(err)
			}
			canonical := restoreStmt(t, n)
			var params []*ast.ParamMarkerExpr
			var nodes []ast.Node
			_, ok := n.Accept(extensionVisitor{enter: func(node ast.Node) (ast.Node, bool) {
				nodes = append(nodes, node)
				if param, ok := node.(*ast.ParamMarkerExpr); ok {
					params = append(params, param)
				}
				return node, false
			}})
			if !ok || len(params) != strings.Count(sql, "?") {
				t.Fatal("parameters not all visited exactly once")
			}
			sort.Slice(params, func(i, j int) bool { return params[i].Span().Start < params[j].Span().Start })
			spans := make([]loc.Span, len(nodes))
			texts := make([]string, len(nodes))
			for i, node := range nodes {
				spans[i] = node.Span()
				texts[i] = node.Text()
			}
			syntax, err := json.Marshal(n)
			if err != nil {
				t.Fatal(err)
			}
			for i, param := range params {
				span := param.Span()
				if param.Index != i || !span.Valid() || sql[span.Start:span.End] != "?" {
					t.Fatalf("parameter %d index=%d span=%v", i, param.Index, span)
				}
			}
			for _, next := range []string{"UPDATE other SET a=9", "DELETE FROM", "SELECT ? FROM q", "DELETE FROM other WHERE a=2"} {
				_, _ = p.ParseOneStmt(next, "", "")
				if got := restoreStmt(t, n); got != canonical {
					t.Fatalf("reuse changed old AST: %s", got)
				}
				for i, node := range nodes {
					if node.Span() != spans[i] || node.Text() != texts[i] {
						t.Fatal("reuse changed original node span or text")
					}
				}
				again, err := json.Marshal(n)
				if err != nil || string(again) != string(syntax) {
					t.Fatal("reuse changed syntax data, including Raw/Quote or name segments")
				}
				for i, param := range params {
					if param.Index != i || sql[param.Span().Start:param.Span().End] != "?" {
						t.Fatal("reuse changed parameter index or source range")
					}
				}
			}
		})
	}
}

func TestDMLOuterFieldContracts(t *testing.T) {
	t.Parallel()
	update := reflect.TypeFor[ast.UpdateStmt]()
	deleteStmt := reflect.TypeFor[ast.DeleteStmt]()
	for _, tc := range []struct {
		name   string
		typ    reflect.Type
		base   reflect.Type
		fields []string
	}{
		{"update", update, reflect.TypeFor[pkast.BaseUpdateStmt](), []string{"BaseUpdateStmt", "With", "Priority", "IgnoreErr", "MultipleTable", "TableHints"}},
		{"delete", deleteStmt, reflect.TypeFor[pkast.BaseDeleteStmt](), []string{"BaseDeleteStmt", "With", "Targets", "Priority", "IgnoreErr", "Quick", "IsMultiTable", "BeforeFrom", "TableHints"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.typ.NumField() != len(tc.fields) || tc.typ.Field(0).Type != tc.base || !tc.typ.Field(0).Anonymous {
				t.Fatal("outer shape or value embedding changed")
			}
			for i, name := range tc.fields {
				if tc.typ.Field(i).Name != name {
					t.Fatalf("field %d=%s want %s", i, tc.typ.Field(i).Name, name)
				}
			}
		})
	}
	if field, _ := deleteStmt.FieldByName("Targets"); field.Type != reflect.TypeFor[[]*ast.TableName]() {
		t.Fatal("targets lost their concrete type")
	}
	var order *ast.OrderByClause
	var limit *ast.Limit
	if order.OrderByClauseCore() != nil || limit.LimitCore() != nil {
		t.Fatal("typed-nil clause core is not safe")
	}
}

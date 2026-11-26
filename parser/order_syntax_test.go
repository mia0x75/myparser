package parser

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	dformat "github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

type byRoleVisitor struct{ enter func(ast.Node) }

func (v byRoleVisitor) Enter(n ast.Node) (ast.Node, bool) { v.enter(n); return n, false }
func (byRoleVisitor) Leave(n ast.Node) (ast.Node, bool)   { return n, true }

func restoreBySQL(t *testing.T, node ast.Node) string {
	t.Helper()
	var out strings.Builder
	if err := node.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func requireBySpan(t *testing.T, node ast.Node, src, want string) {
	t.Helper()
	span := node.Span()
	if !span.Within(len(src)) || src[span.Start:span.End] != want {
		t.Fatalf("%T span = %v, want source slice %q in %q", node, span, want, src)
	}
}

func TestSyntaxByRoleConstruction(t *testing.T) {
	span := loc.Span{Start: 2, End: 9}
	expr := &ast.PositionExpr{N: 1}
	items := []*syntaxByItem{{expr: expr, desc: true, span: span}}
	order := syntaxOrderBy(items, span)
	group := syntaxGroupBy(items, span)
	partition := syntaxPartitionBy(items, span)
	partitionItem := partition.Items[0].(*ast.PartitionItem)
	if order.Items[0].OrderItemCore().Expr != expr || !order.Items[0].OrderItemCore().Desc ||
		group.Items[0].GroupItemCore().Expr != expr || !group.Items[0].(*ast.GroupItem).Desc ||
		partitionItem.Expr != expr || !partitionItem.Desc {
		t.Fatal("role conversion lost the expression or direction")
	}
	for _, node := range []ast.Node{order, group, partition, order.Items[0], group.Items[0], partition.Items[0]} {
		if node.Span() != span {
			t.Fatalf("%T lost source span: %v", node, node.Span())
		}
	}
	if _, wrongRole := group.Items[0].(ast.OrderItemNode); wrongRole {
		t.Fatal("a grouping item must not masquerade as an ordering item")
	}
	if _, wrongRole := ast.Node(partition.Items[0]).(ast.GroupItemNode); wrongRole {
		t.Fatal("a window partition must not masquerade as grouping")
	}
	*items[0] = syntaxByItem{expr: &ast.PositionExpr{N: 2}}
	if order.Items[0].OrderItemCore().Expr != expr || group.Items[0].GroupItemCore().Expr != expr ||
		partition.Items[0].WindowPartitionItemCore().Expr != expr || !order.Items[0].OrderItemCore().Desc {
		t.Fatal("final AST must not retain mutable reduction carriers")
	}

	if syntaxOrderBy(nil, span).Items != nil || syntaxGroupBy(nil, span).Items != nil || syntaxPartitionBy(nil, span).Items != nil {
		t.Fatal("nil item lists must remain nil")
	}
	empty := []*syntaxByItem{}
	if syntaxOrderBy(empty, span).Items == nil || syntaxGroupBy(empty, span).Items == nil || syntaxPartitionBy(empty, span).Items == nil {
		t.Fatal("explicit empty item lists must remain present")
	}
}

func TestParsedByRoleContexts(t *testing.T) {
	for _, tc := range []struct {
		name, sql               string
		order, group, partition int
	}{
		{"select order", "SELECT a FROM t ORDER BY a ASC,b DESC", 2, 0, 0},
		{"group direction", "SELECT a FROM t GROUP BY a,b DESC", 0, 2, 0},
		{"rollup", "SELECT a FROM t GROUP BY a WITH ROLLUP", 0, 1, 0},
		{"cube", "SELECT a FROM t GROUP BY a WITH CUBE", 0, 1, 0},
		{"view local", "CREATE VIEW v AS SELECT a FROM t GROUP BY a WITH LOCAL CHECK OPTION", 0, 1, 0},
		{"view cascaded", "CREATE VIEW v AS SELECT a FROM t GROUP BY a WITH CASCADED CHECK OPTION", 0, 1, 0},
		{"union tail", "SELECT a FROM t UNION SELECT b FROM t ORDER BY 1 DESC LIMIT 2", 1, 0, 0},
		{"values union tail", "VALUES (1) UNION SELECT 2 ORDER BY 1 LIMIT 1", 1, 0, 0},
		{"parenthesized tails", "(SELECT a FROM t ORDER BY a DESC LIMIT 1) UNION SELECT b FROM t ORDER BY 1 LIMIT 2", 2, 0, 0},
		{"update tail", "UPDATE t SET a=1 ORDER BY b DESC LIMIT 1", 1, 0, 0},
		{"delete tail", "DELETE FROM t ORDER BY b DESC LIMIT 1", 1, 0, 0},
		{"group concat", "SELECT GROUP_CONCAT(a ORDER BY b DESC LIMIT 2) FROM t", 1, 0, 0},
		{"json aggregate", "SELECT JSON_ARRAYAGG(a ORDER BY b DESC) FROM t", 1, 0, 0},
		{"within group", "SELECT PERCENTILE_CONT(0.5) WITHIN GROUP(ORDER BY score DESC) OVER(PARTITION BY grp DESC) FROM t", 1, 0, 1},
		{"window", "SELECT SUM(a) OVER(PARTITION BY b DESC ORDER BY c DESC) FROM t", 1, 0, 1},
		{"named window", "SELECT SUM(a) OVER w FROM t WINDOW w AS(PARTITION BY b DESC ORDER BY c DESC)", 1, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := New()
			p.EnableWindowFunc(true)
			stmt, err := p.ParseOneStmt(tc.sql)
			if err != nil {
				t.Fatal(err)
			}
			var order, group, partition int
			if _, ok := stmt.Accept(byRoleVisitor{enter: func(node ast.Node) {
				switch node.(type) {
				case *ast.OrderItem:
					order++
				case *ast.GroupItem:
					group++
				case *ast.PartitionItem:
					partition++
				default:
					return
				}
				if !node.Span().Within(len(tc.sql)) {
					t.Errorf("%T has invalid byte span %v", node, node.Span())
				}
			}}); !ok {
				t.Fatal("visitor aborted")
			}
			if order != tc.order || group != tc.group || partition != tc.partition {
				t.Fatalf("role counts = (%d,%d,%d), want (%d,%d,%d)", order, group, partition, tc.order, tc.group, tc.partition)
			}
			canonical := restoreBySQL(t, stmt)
			reparsed, err := p.ParseOneStmt(canonical)
			if err != nil {
				t.Fatalf("reparse %q: %v", canonical, err)
			}
			if got := restoreBySQL(t, reparsed); got != canonical {
				t.Fatalf("unstable canonical SQL: %q != %q", got, canonical)
			}
		})
	}
}

func TestParsedByPositionsAndSpans(t *testing.T) {
	for _, role := range []string{"order", "group", "partition"} {
		for _, tc := range []struct {
			input    string
			position bool
			value    int
		}{
			{"TRUE", true, 1}, {"FALSE", true, 0}, {"1", true, 1}, {"0", true, 0},
			{"1.50", false, 0}, {"18446744073709551615", false, 0},
		} {
			t.Run(role+"/"+tc.input, func(t *testing.T) {
				p := New()
				p.EnableWindowFunc(true)
				var sql string
				switch role {
				case "order":
					sql = "SELECT x FROM t ORDER BY " + tc.input + " DESC"
				case "group":
					sql = "SELECT x FROM t GROUP BY " + tc.input + " DESC"
				case "partition":
					sql = "SELECT SUM(x) OVER(PARTITION BY " + tc.input + " DESC) FROM t"
				}
				stmt, err := p.ParseOneStmt(sql)
				if err != nil {
					t.Fatal(err)
				}
				selectStmt := stmt.(*ast.SelectStmt)
				var item ast.Node
				var expr ast.ExprNode
				switch role {
				case "order":
					item = selectStmt.OrderBy.OrderByClauseCore().Items[0]
					expr = selectStmt.OrderBy.OrderByClauseCore().Items[0].OrderItemCore().Expr
				case "group":
					item = selectStmt.GroupBy.GroupByClauseCore().Items[0]
					expr = selectStmt.GroupBy.GroupByClauseCore().Items[0].GroupItemCore().Expr
				case "partition":
					partition := selectStmt.Fields.Items[0].Expr.(*ast.WindowFuncExpr).Spec.PartitionBy.(*ast.PartitionByClause)
					item, expr = partition.Items[0], partition.Items[0].WindowPartitionItemCore().Expr
				}
				requireBySpan(t, item, sql, tc.input+" DESC")
				requireBySpan(t, expr, sql, tc.input)
				position, ok := expr.(*ast.PositionExpr)
				if ok != tc.position || ok && position.N != tc.value {
					t.Fatalf("position = %#v (%T), want position=%v value=%d", expr, expr, tc.position, tc.value)
				}
				if !tc.position {
					literal, ok := expr.(*ast.LiteralExpr)
					if !ok || literal.Raw != tc.input {
						t.Fatalf("non-position literal lost: %#v", expr)
					}
				}
			})
		}
	}
}

func TestByNameParametersAndParserReuse(t *testing.T) {
	const name = "`a.b`.`列`"
	const sql = "SELECT GROUP_CONCAT(" + name + " ORDER BY ? DESC),SUM(x) OVER(PARTITION BY " + name + " DESC ORDER BY ? DESC) FROM t GROUP BY " + name + " DESC WITH ROLLUP ORDER BY ? ASC"
	p := New()
	p.EnableWindowFunc(true)
	stmt, err := p.ParseOneStmt(sql)
	if err != nil {
		t.Fatal(err)
	}
	selectStmt := stmt.(*ast.SelectStmt)
	group := selectStmt.GroupBy.GroupByClauseCore().Items[0].(*ast.GroupItem)
	column := group.Expr.(*ast.ColumnRef)
	if len(column.Name.Parts) != 2 || column.Name.Parts[0].Name != "a.b" || column.Name.Parts[1].Name != "列" {
		t.Fatalf("qualified name flattened: %#v", column.Name)
	}
	for i, raw := range []string{"`a.b`", "`列`"} {
		part := column.Name.Parts[i]
		if part.Raw != raw || part.Quote != lex.QuoteBacktick || !part.Span.Within(len(sql)) || sql[part.Span.Start:part.Span.End] != raw {
			t.Fatalf("name part %d lost lexical metadata: %#v", i, part)
		}
	}
	requireBySpan(t, group, sql, name+" DESC")
	requireBySpan(t, selectStmt.OrderBy, sql, "ORDER BY ? ASC")
	if selectStmt.OrderBy.OrderByClauseCore().Items[0].OrderItemCore().Desc {
		t.Fatal("explicit ASC must keep Desc=false")
	}
	var roles []string
	var indices []int
	if _, ok := stmt.Accept(byRoleVisitor{enter: func(node ast.Node) {
		switch node := node.(type) {
		case *ast.OrderItem:
			roles = append(roles, "order")
		case *ast.GroupItem:
			roles = append(roles, "group")
		case *ast.PartitionItem:
			roles = append(roles, "partition")
		case *ast.ParamMarkerExpr:
			indices = append(indices, node.Index)
		}
	}}); !ok {
		t.Fatal("visitor aborted")
	}
	if !reflect.DeepEqual(roles, []string{"order", "partition", "order", "group", "order"}) || !reflect.DeepEqual(indices, []int{0, 1, 2}) {
		t.Fatalf("visit roles=%v parameter indices=%v", roles, indices)
	}
	canonical := restoreBySQL(t, stmt)
	beforeName := column.Name
	beforeName.Parts = append([]lex.Ident(nil), column.Name.Parts...)
	for range 8 {
		if _, err := p.ParseOneStmt("SELECT SUM(a) OVER(PARTITION BY b ORDER BY c) FROM replacement GROUP BY d ORDER BY e"); err != nil {
			t.Fatal(err)
		}
	}
	if got := restoreBySQL(t, stmt); got != canonical || !reflect.DeepEqual(column.Name, beforeName) {
		t.Fatal("subsequent Parse overwrote the earlier AST")
	}
	if _, ok := ast.Node(group).(pkast.OrderItemNode); ok {
		t.Fatal("MySQL grouping direction must not change its semantic role")
	}
}

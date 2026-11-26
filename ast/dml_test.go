package ast_test

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	. "github.com/mia0x75/myparser/ast"
	dformat "github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/parser"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

func TestDMLVisitorCover(t *testing.T) {
	ce := &checkExpr{}

	joinWithOn := &JoinExpr{BaseJoinExpr: pkast.BaseJoinExpr{
		Left:  &TableSource{BaseTableSource: pkast.BaseTableSource{Source: &TableName{}}},
		Right: &TableName{}, Type: pkast.JoinTypeInner, On: ce,
	}}
	tableRefsClause := &TableRefsClause{BaseFromClause: pkast.BaseFromClause{Table: joinWithOn}}

	stmts := []struct {
		node             Node
		expectedEnterCnt int
		expectedLeaveCnt int
	}{
		{&DeleteStmt{BaseDeleteStmt: pkast.BaseDeleteStmt{Table: tableRefsClause.Table, Where: ce,
			OrderBy: &OrderByClause{}, Limit: &Limit{BaseLimit: pkast.BaseLimit{Count: ce, Offset: ce}}}}, 4, 4},
		{&ShowStmt{Table: &TableName{}, Column: &QualifiedName{}, Pattern: &PatternLikeExpr{Expr: ce, Pattern: ce}, Where: ce}, 3, 3},
		{&LoadDataStmt{Table: &TableName{}, Columns: []QualifiedName{}, FieldsInfo: &FieldsClause{}, LinesInfo: &LinesClause{}}, 0, 0},
		{&Assignment{Target: ce, Expr: ce}, 2, 2},
		{&OrderItem{BaseOrderItem: pkast.BaseOrderItem{Expr: ce}}, 1, 1},
		{&GroupItem{BaseGroupItem: pkast.BaseGroupItem{Expr: ce}}, 1, 1},
		{&PartitionItem{Expr: ce}, 1, 1},
		{&GroupByClause{Items: []GroupItemNode{&GroupItem{BaseGroupItem: pkast.BaseGroupItem{Expr: ce}}, &GroupItem{BaseGroupItem: pkast.BaseGroupItem{Expr: ce}}}}, 2, 2},
		{&SelectStmt{BaseSelectStmt: pkast.BaseSelectStmt{Having: ce}}, 1, 1},
		{&JoinExpr{BaseJoinExpr: pkast.BaseJoinExpr{Left: &TableSource{BaseTableSource: pkast.BaseTableSource{Source: &TableName{}}}}}, 0, 0},
		{&Limit{BaseLimit: pkast.BaseLimit{Count: ce, Offset: ce}}, 2, 2},
		{joinWithOn, 1, 1},
		{&OrderByClause{BaseOrderByClause: pkast.BaseOrderByClause{Items: []OrderItemNode{&OrderItem{BaseOrderItem: pkast.BaseOrderItem{Expr: ce}}, &OrderItem{BaseOrderItem: pkast.BaseOrderItem{Expr: ce}}}}}, 2, 2},
		{&SelectItem{Expr: ce}, 1, 1},
		{&WildcardExpr{}, 0, 0},
		{&TableName{}, 0, 0},
		{tableRefsClause, 1, 1},
		{&TableSource{BaseTableSource: pkast.BaseTableSource{Source: &TableName{}}}, 0, 0},

		{&InsertStmt{BaseInsertStmt: pkast.BaseInsertStmt{
			Table: &TableName{}, Columns: []ExprNode{ce}, Values: [][]ExprNode{{ce}},
		}}, 2, 2},
		{&SetOprStmt{}, 0, 0},
		{&UpdateStmt{BaseUpdateStmt: pkast.BaseUpdateStmt{Table: tableRefsClause.Table}}, 1, 1},
		{&SelectStmt{}, 0, 0},
		{&FieldList{}, 0, 0},
		{&WindowSpec{}, 0, 0},
		{&PartitionByClause{}, 0, 0},
		{&FrameClause{}, 0, 0},
		{&FrameBound{}, 0, 0},
	}

	for _, v := range stmts {
		ce.reset()
		v.node.Accept(checkVisitor{})
		if ce.enterCnt != v.expectedEnterCnt {
			t.Errorf("enterCnt got %d, want %d", ce.enterCnt, v.expectedEnterCnt)
		}
		if ce.leaveCnt != v.expectedLeaveCnt {
			t.Errorf("leaveCnt got %d, want %d", ce.leaveCnt, v.expectedLeaveCnt)
		}
		v.node.Accept(visitor1{})
	}
}

func TestSelectStmtVisitsSharedClauses(t *testing.T) {
	expr := &checkExpr{}
	stmt := &SelectStmt{BaseSelectStmt: pkast.BaseSelectStmt{
		Fields: &FieldList{Items: []*SelectItem{{Expr: expr}}},
		From: &TableRefsClause{BaseFromClause: pkast.BaseFromClause{Table: &JoinExpr{
			BaseJoinExpr: pkast.BaseJoinExpr{Left: &TableName{}, On: expr},
		}}},
		Where: expr,
		GroupBy: &GroupByClause{BaseGroupByClause: pkast.BaseGroupByClause{
			Items: []pkast.GroupItemNode{&GroupItem{BaseGroupItem: pkast.BaseGroupItem{Expr: expr}}},
		}},
		Having:      expr,
		WindowSpecs: []pkast.WindowSpecNode{&WindowSpec{}},
		OrderBy: &OrderByClause{BaseOrderByClause: pkast.BaseOrderByClause{
			Items: []OrderItemNode{&OrderItem{BaseOrderItem: pkast.BaseOrderItem{Expr: expr}}},
		}},
		Limit: &Limit{BaseLimit: pkast.BaseLimit{Count: expr}},
		With:  &WithClause{},
	}}

	got, ok := stmt.Accept(checkVisitor{})
	if !ok || got != stmt {
		t.Fatalf("Accept() = (%T, %v), want the same SELECT node", got, ok)
	}
	if expr.enterCnt != 7 || expr.leaveCnt != 7 {
		t.Fatalf("expression visits = (%d, %d), want (7, 7)", expr.enterCnt, expr.leaveCnt)
	}
}

func TestQueryLeafInvalidChildren(t *testing.T) {
	values := &ValuesStmt{BaseValuesStmt: pkast.BaseValuesStmt{Rows: []pkast.RowNode{nil}}}
	var valuesOut bytes.Buffer
	if err := values.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &valuesOut)); err == nil {
		t.Fatal("VALUES restored a nil row")
	}
	if _, ok := values.Accept(checkVisitor{}); ok {
		t.Fatal("VALUES visited a nil row")
	}
	values.Rows = []pkast.RowNode{(*RowExpr)(nil)}
	if err := values.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &valuesOut)); err == nil {
		t.Fatal("VALUES restored a typed nil row")
	}
	if _, ok := values.Accept(checkVisitor{}); ok {
		t.Fatal("VALUES visited a typed nil row")
	}

	table := &TableStmt{}
	var tableOut bytes.Buffer
	if err := table.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &tableOut)); err == nil {
		t.Fatal("TABLE restored a nil reference")
	}
	if _, ok := table.Accept(checkVisitor{}); ok {
		t.Fatal("TABLE visited a nil reference")
	}
	table.Table = (*TableName)(nil)
	if err := table.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &tableOut)); err == nil {
		t.Fatal("TABLE restored a typed nil reference")
	}
	if _, ok := table.Accept(checkVisitor{}); ok {
		t.Fatal("TABLE visited a typed nil reference")
	}
}

func TestTableNameRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"dbb.`tbb1`", "`dbb`.`tbb1`"},
		{"`tbb2`", "`tbb2`"},
		{"tbb3", "`tbb3`"},
		{"dbb.`hello-world`", "`dbb`.`hello-world`"},
		{"`dbb`.`hello-world`", "`dbb`.`hello-world`"},
		{"`dbb.HelloWorld`", "`dbb.HelloWorld`"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*CreateTableStmt).Table
	}
	RunNodeRestoreTest(t, testCases, "CREATE TABLE %s (id VARCHAR(128) NOT NULL);", extractNodeFunc)
}

func TestTableSourceIndexHintsRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"t use index (hello)", "`t` USE INDEX (`hello`)"},
		{"t use index (hello, world)", "`t` USE INDEX (`hello`, `world`)"},
		{"t use index ()", "`t` USE INDEX ()"},
		{"t use key ()", "`t` USE INDEX ()"},
		{"t ignore key ()", "`t` IGNORE INDEX ()"},
		{"t force key ()", "`t` FORCE INDEX ()"},
		{"t use index for order by (idx1)", "`t` USE INDEX FOR ORDER BY (`idx1`)"},

		{"t use index (hello, world, yes) force key (good)", "`t` USE INDEX (`hello`, `world`, `yes`) FORCE INDEX (`good`)"},
		{"t use index (hello, world, yes) use index for order by (good)", "`t` USE INDEX (`hello`, `world`, `yes`) USE INDEX FOR ORDER BY (`good`)"},
		{"t ignore key (hello, world, yes) force key (good)", "`t` IGNORE INDEX (`hello`, `world`, `yes`) FORCE INDEX (`good`)"},

		{"t use index for group by (idx1) use index for order by (idx2)", "`t` USE INDEX FOR GROUP BY (`idx1`) USE INDEX FOR ORDER BY (`idx2`)"},
		{"t use index for group by (idx1) ignore key for order by (idx2)", "`t` USE INDEX FOR GROUP BY (`idx1`) IGNORE INDEX FOR ORDER BY (`idx2`)"},
		{"t use index for group by (idx1) ignore key for group by (idx2)", "`t` USE INDEX FOR GROUP BY (`idx1`) IGNORE INDEX FOR GROUP BY (`idx2`)"},
		{"t use index for order by (idx1) ignore key for group by (idx2)", "`t` USE INDEX FOR ORDER BY (`idx1`) IGNORE INDEX FOR GROUP BY (`idx2`)"},

		{"t use index for order by (idx1) ignore key for group by (idx2) use index (idx3)", "`t` USE INDEX FOR ORDER BY (`idx1`) IGNORE INDEX FOR GROUP BY (`idx2`) USE INDEX (`idx3`)"},
		{"t use index for order by (idx1) ignore key for group by (idx2) use index (idx3)", "`t` USE INDEX FOR ORDER BY (`idx1`) IGNORE INDEX FOR GROUP BY (`idx2`) USE INDEX (`idx3`)"},

		{"t use index (`foo``bar`) force index (`baz``1`, `xyz`)", "`t` USE INDEX (`foo``bar`) FORCE INDEX (`baz``1`, `xyz`)"},
		{"t force index (`foo``bar`) ignore index (`baz``1`, xyz)", "`t` FORCE INDEX (`foo``bar`) IGNORE INDEX (`baz``1`, `xyz`)"},
		{"t ignore index (`foo``bar`) force key (`baz``1`, xyz)", "`t` IGNORE INDEX (`foo``bar`) FORCE INDEX (`baz``1`, `xyz`)"},
		{"t ignore index (`foo``bar`) ignore key for group by (`baz``1`, xyz)", "`t` IGNORE INDEX (`foo``bar`) IGNORE INDEX FOR GROUP BY (`baz``1`, `xyz`)"},
		{"t ignore index (`foo``bar`) ignore key for order by (`baz``1`, xyz)", "`t` IGNORE INDEX (`foo``bar`) IGNORE INDEX FOR ORDER BY (`baz``1`, `xyz`)"},

		{"t use index for group by (`foo``bar`) use index for order by (`baz``1`, `xyz`)", "`t` USE INDEX FOR GROUP BY (`foo``bar`) USE INDEX FOR ORDER BY (`baz``1`, `xyz`)"},
		{"t use index for group by (`foo``bar`) ignore key for order by (`baz``1`, `xyz`)", "`t` USE INDEX FOR GROUP BY (`foo``bar`) IGNORE INDEX FOR ORDER BY (`baz``1`, `xyz`)"},
		{"t use index for group by (`foo``bar`) ignore key for group by (`baz``1`, `xyz`)", "`t` USE INDEX FOR GROUP BY (`foo``bar`) IGNORE INDEX FOR GROUP BY (`baz``1`, `xyz`)"},
		{"t use index for order by (`foo``bar`) ignore key for group by (`baz``1`, `xyz`)", "`t` USE INDEX FOR ORDER BY (`foo``bar`) IGNORE INDEX FOR GROUP BY (`baz``1`, `xyz`)"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).From.FromClauseCore().Table.(*JoinExpr).Left.(*TableSource)
	}
	RunNodeRestoreTest(t, testCases, "SELECT * FROM %s", extractNodeFunc)
}

func TestLimitRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"limit 10", "LIMIT 10"},
		{"limit 10,20", "LIMIT 10,20"},
		{"limit 20 offset 10", "LIMIT 10,20"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Limit
	}
	RunNodeRestoreTest(t, testCases, "SELECT 1 %s", extractNodeFunc)
}

func TestLimitEnterReplacement(t *testing.T) {
	original := &Limit{BaseLimit: pkast.BaseLimit{Count: sharedIntLiteral("1"), Offset: sharedIntLiteral("2")}}
	originalCount, originalOffset := original.Count, original.Offset
	count, offset := sharedIntLiteral("3"), sharedIntLiteral("4")
	replacement := &Limit{BaseLimit: pkast.BaseLimit{Count: count, Offset: offset}}
	updatedCount := sharedIntLiteral("5")
	var entered, left []Node
	got, ok := original.Accept(operatorVisitor{
		enter: func(n Node) (Node, bool) {
			entered = append(entered, n)
			if n == original {
				return replacement, false
			}
			return n, false
		},
		leave: func(n Node) (Node, bool) {
			left = append(left, n)
			if n == count {
				return updatedCount, true
			}
			return n, true
		},
	})
	if !ok || got != replacement || replacement.Count != updatedCount || replacement.Offset != offset {
		t.Fatal("Enter replacement must receive its visited children")
	}
	if !reflect.DeepEqual(entered, []Node{original, count, offset}) || !reflect.DeepEqual(left, []Node{count, offset, replacement}) {
		t.Fatalf("wrong traversal: entered=%v, left=%v", entered, left)
	}
	if original.Count != originalCount || original.Offset != originalOffset {
		t.Fatal("replacement traversal changed the old LIMIT")
	}
	t.Run("skip replacement", func(t *testing.T) {
		var visited []Node
		got, ok := original.Accept(operatorVisitor{
			enter: func(n Node) (Node, bool) {
				visited = append(visited, n)
				return replacement, true
			},
			leave: func(n Node) (Node, bool) { return n, true },
		})
		if !ok || got != replacement || !reflect.DeepEqual(visited, []Node{original}) {
			t.Fatal("skipping must leave the replacement without visiting its expressions")
		}
	})
	t.Run("stop replacement child", func(t *testing.T) {
		var visited, left []Node
		_, ok := original.Accept(operatorVisitor{
			enter: func(n Node) (Node, bool) {
				visited = append(visited, n)
				if n == original {
					return replacement, false
				}
				return n, false
			},
			leave: func(n Node) (Node, bool) {
				left = append(left, n)
				return n, false
			},
		})
		if ok || !reflect.DeepEqual(visited, []Node{original, updatedCount}) || !reflect.DeepEqual(left, []Node{updatedCount}) {
			t.Fatal("stopping the count must skip the offset and the outer Leave")
		}
	})
}

func TestLimitRejectsInvalidReplacement(t *testing.T) {
	for _, skip := range []bool{false, true} {
		for _, wrong := range []struct {
			name string
			node Node
		}{{"wrong type", &TableName{}}, {"nil", nil}, {"typed nil", (*Limit)(nil)}} {
			t.Run(fmt.Sprintf("%s/skip=%t", wrong.name, skip), func(t *testing.T) {
				limit := &Limit{}
				_, ok := limit.Accept(operatorVisitor{
					enter: func(n Node) (Node, bool) { return wrong.node, skip },
					leave: func(n Node) (Node, bool) { t.Fatal("invalid outer reached Leave"); return n, true },
				})
				if ok {
					t.Fatal("invalid outer replacement accepted")
				}
			})
		}
	}
	for _, slot := range []string{"count", "offset"} {
		t.Run(slot, func(t *testing.T) {
			count, offset := sharedIntLiteral("1"), sharedIntLiteral("2")
			limit := &Limit{BaseLimit: pkast.BaseLimit{Count: count, Offset: offset}}
			target := count
			if slot == "offset" {
				target = offset
			}
			_, ok := limit.Accept(operatorVisitor{
				enter: func(n Node) (Node, bool) { return n, false },
				leave: func(n Node) (Node, bool) {
					if n == target {
						return &TableName{}, true
					}
					return n, true
				},
			})
			if ok || limit.Count != count || limit.Offset != offset {
				t.Fatal("invalid child replacement must stop without changing expressions")
			}
		})
	}
}

func TestLimitRestoreRequiresCount(t *testing.T) {
	for _, tc := range []struct {
		name string
		node *Limit
	}{{"nil receiver", nil}, {"empty", &Limit{}}, {"offset only", &Limit{BaseLimit: pkast.BaseLimit{Offset: sharedIntLiteral("1")}}}} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := tc.node.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); err == nil || out.Len() != 0 {
				t.Fatalf("incomplete LIMIT must fail before writing: err=%v, output=%q", err, out.String())
			}
		})
	}
}

func TestWildcardExprRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"*", "*"},
		{"t.*", "`t`.*"},
		{"testdb.t.*", "`testdb`.`t`.*"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Items[0].Expr.(*WildcardExpr)
	}
	RunNodeRestoreTest(t, testCases, "SELECT %s", extractNodeFunc)
}

func TestSelectItemRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"*", "*"},
		{"t.*", "`t`.*"},
		{"testdb.t.*", "`testdb`.`t`.*"},
		{"col as a", "`col` AS `a`"},
		{"col + 1 a", "`col`+1 AS `a`"},
		{"1 AS ''", "1 AS ``"},
		{`1 AS ""`, "1 AS ``"},
		{"1 AS '列''别'", "1 AS `列'别`"},
		{"`db.dot`.`t``.dot`.*", "`db.dot`.`t``.dot`.*"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Items[0]
	}
	RunNodeRestoreTest(t, testCases, "SELECT %s", extractNodeFunc)
}

// A select without a projection cannot be rendered back to SQL.
func TestSelectWithoutProjectionFailsRestore(t *testing.T) {
	for _, stmt := range []Node{&SelectStmt{}, &SelectStmt{Fields: &FieldList{}}} {
		var out strings.Builder
		if err := stmt.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); err == nil {
			t.Fatalf("Restore accepted an empty projection: %q", out.String())
		}
	}
}

func TestFieldListRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"*", "*"},
		{"t.*", "`t`.*"},
		{"testdb.t.*", "`testdb`.`t`.*"},
		{"col as a", "`col` AS `a`"},
		{"`t`.*, s.col as a", "`t`.*, `s`.`col` AS `a`"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields
	}
	RunNodeRestoreTest(t, testCases, "SELECT %s", extractNodeFunc)
}

func TestTableSourceRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"tbl", "`tbl`"},
		{"tbl as t", "`tbl` AS `t`"},
		{"(select * from tbl) as t", "(SELECT * FROM `tbl`) AS `t`"},
		{"(select * from a union select * from b) as t", "(SELECT * FROM `a` UNION SELECT * FROM `b`) AS `t`"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).From.FromClauseCore().Table.(*JoinExpr).Left
	}
	RunNodeRestoreTest(t, testCases, "select * from %s", extractNodeFunc)
}

func TestOnConditionRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"on t1.a=t2.a", "ON `t1`.`a`=`t2`.`a`"},
		{"on t1.a=t2.a and t1.b=t2.b", "ON `t1`.`a`=`t2`.`a` AND `t1`.`b`=`t2`.`b`"},
	}
	for _, tc := range testCases {
		t.Run(tc.sourceSQL, func(t *testing.T) {
			stmt, err := parser.New().ParseOneStmt(fmt.Sprintf("select * from t1 join t2 %s", tc.sourceSQL))
			if err != nil {
				t.Fatal(err)
			}
			join := stmt.(*SelectStmt).From.FromClauseCore().Table.(*JoinExpr)
			var buf strings.Builder
			want := "`t1` JOIN `t2` " + tc.expectSQL
			if err := join.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &buf)); err != nil || buf.String() != want {
				t.Fatalf("join restored %q, want %q; err=%v", buf.String(), want, err)
			}
			if join.On == nil {
				t.Fatal("missing direct ON expression")
			}
			buf.Reset()
			want = strings.TrimPrefix(tc.expectSQL, "ON ")
			if err := join.On.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &buf)); err != nil || buf.String() != want {
				t.Fatalf("ON expression restored %q, want %q; err=%v", buf.String(), want, err)
			}
		})
	}
}

func TestJoinRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"t1 natural join t2", "`t1` NATURAL JOIN `t2`"},
		{"t1 natural inner join t2", "`t1` NATURAL JOIN `t2`"},
		{"t1 natural left join t2", "`t1` NATURAL LEFT JOIN `t2`"},
		{"t1 natural right outer join t2", "`t1` NATURAL RIGHT JOIN `t2`"},
		{"t1 straight_join t2", "`t1` STRAIGHT_JOIN `t2`"},
		{"t1 straight_join t2 using (a)", "`t1` STRAIGHT_JOIN `t2` USING (`a`)"},
		{"t1 straight_join t2 on t1.a>t2.a", "`t1` STRAIGHT_JOIN `t2` ON `t1`.`a`>`t2`.`a`"},
		{"t1 cross join t2", "`t1` JOIN `t2`"},
		{"t1 cross join t2 on t1.a>t2.a", "`t1` JOIN `t2` ON `t1`.`a`>`t2`.`a`"},
		{"t1 inner join t2 using (b)", "`t1` JOIN `t2` USING (`b`)"},
		{"t1 join t2 using (b,c) left join t3 on t1.a>t3.a", "(`t1` JOIN `t2` USING (`b`,`c`)) LEFT JOIN `t3` ON `t1`.`a`>`t3`.`a`"},
		{"t1 natural join t2 right outer join t3 using (b,c)", "(`t1` NATURAL JOIN `t2`) RIGHT JOIN `t3` USING (`b`,`c`)"},
		{"(a al left join b bl on al.a1 > bl.b1) join (a ar right join b br on ar.a1 > br.b1)", "(`a` AS `al` LEFT JOIN `b` AS `bl` ON `al`.`a1`>`bl`.`b1`) JOIN (`a` AS `ar` RIGHT JOIN `b` AS `br` ON `ar`.`a1`>`br`.`b1`)"},
		{"a al left join b bl on al.a1 > bl.b1, a ar right join b br on ar.a1 > br.b1", "(`a` AS `al` LEFT JOIN `b` AS `bl` ON `al`.`a1`>`bl`.`b1`) JOIN (`a` AS `ar` RIGHT JOIN `b` AS `br` ON `ar`.`a1`>`br`.`b1`)"},
		{"t1 join t2 left join t3 on t2.a=t3.a", "(`t1` JOIN `t2`) LEFT JOIN `t3` ON `t2`.`a`=`t3`.`a`"},
		{"t1 join (t2 left join t3 on t2.a=t3.a)", "`t1` JOIN (`t2` LEFT JOIN `t3` ON `t2`.`a`=`t3`.`a`)"},
		{"t1 join ((t2 join t3 on t2.a=t3.a)) left join t4 on t1.a=t4.a", "(`t1` JOIN (`t2` JOIN `t3` ON `t2`.`a`=`t3`.`a`)) LEFT JOIN `t4` ON `t1`.`a`=`t4`.`a`"},
		{"t1, t2 left join t3 on t1.a=t3.a", "(`t1`) JOIN (`t2` LEFT JOIN `t3` ON `t1`.`a`=`t3`.`a`)"},
		{"t1, t2", "(`t1`) JOIN `t2`"},
		{"t1, t2, t3", "((`t1`) JOIN `t2`) JOIN `t3`"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).From.FromClauseCore().Table.(*JoinExpr)
	}
	RunNodeRestoreTest(t, testCases, "select * from %s", extractNodeFunc)
}

func TestTableRefsClauseRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"t", "`t`"},
		{"t1 join t2", "`t1` JOIN `t2`"},
		{"t1, t2", "(`t1`) JOIN `t2`"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).From
	}
	RunNodeRestoreTest(t, testCases, "select * from %s", extractNodeFunc)
}

func TestDeleteTableListRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"t1,t2", "`t1`,`t2`"},
	}
	for _, tc := range testCases {
		for _, form := range []struct{ template, canonical string }{
			{"DELETE %s FROM t1, t2;", "DELETE `t1`,`t2` FROM (`t1`) JOIN `t2`"},
			{"DELETE FROM %s USING t1, t2;", "DELETE FROM `t1`,`t2` USING (`t1`) JOIN `t2`"},
		} {
			t.Run(form.template+tc.sourceSQL, func(t *testing.T) {
				p := parser.New()
				sourceSQL := fmt.Sprintf(form.template, tc.sourceSQL)
				stmt, err := p.ParseOneStmt(sourceSQL)
				if err != nil {
					t.Fatal(err)
				}
				del := stmt.(*DeleteStmt)
				if del.Table != nil || del.Using == nil || !del.IsMultiTable || len(del.Targets) != 2 {
					t.Fatalf("targets shape lost: %+v", del)
				}
				var list strings.Builder
				for i, target := range del.Targets {
					if target == nil || len(target.Name.Parts) != 1 || target.Name.Parts[0].Name != fmt.Sprintf("t%d", i+1) {
						t.Fatalf("target %d or its order lost", i)
					}
					if i != 0 {
						list.WriteString(",")
					}
					if err := target.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &list)); err != nil {
						t.Fatal(err)
					}
				}
				if list.String() != tc.expectSQL {
					t.Fatalf("restored targets = %q, want %q", list.String(), tc.expectSQL)
				}
				var full strings.Builder
				if err := del.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &full)); err != nil || full.String() != form.canonical {
					t.Fatalf("full restore = %q, want %q, err=%v", full.String(), form.canonical, err)
				}
				stmt2, err := p.ParseOneStmt(full.String())
				if err != nil {
					t.Fatal(err)
				}
				CleanNodeText(stmt)
				normalizeSyntaxMetadata(stmt)
				CleanNodeText(stmt2)
				normalizeSyntaxMetadata(stmt2)
				if !reflect.DeepEqual(stmt2, stmt) {
					t.Fatalf("canonical round trip: got %#v, want %#v", stmt2, stmt)
				}
			})
		}
	}
}

func TestOrderItemRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"a", "`a`"},
		{"a desc", "`a` DESC"},
		{"NULL", "NULL"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).OrderBy.OrderByClauseCore().Items[0]
	}
	RunNodeRestoreTest(t, testCases, "select * from t order by %s", extractNodeFunc)
}

func TestGroupByClauseRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"GROUP BY a,b desc", "GROUP BY `a`,`b` DESC"},
		{"GROUP BY 1 desc,b", "GROUP BY 1 DESC,`b`"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).GroupBy
	}
	RunNodeRestoreTest(t, testCases, "select * from t %s", extractNodeFunc)
}

func TestOrderByClauseRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"ORDER BY a", "ORDER BY `a`"},
		{"ORDER BY a,b", "ORDER BY `a`,`b`"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).OrderBy
	}
	RunNodeRestoreTest(t, testCases, "SELECT 1 FROM t1 %s", extractNodeFunc)

	extractNodeFromSetOprStmtFunc := func(node Node) Node {
		return node.(*SetOprStmt).Tail.OrderBy
	}
	RunNodeRestoreTest(t, testCases, "SELECT 1 FROM t1 UNION SELECT 2 FROM t2 %s", extractNodeFromSetOprStmtFunc)
}

func TestAssignmentRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"a=1", "`a`=1"},
		{"b=1+2", "`b`=1+2"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*UpdateStmt).Assignments[0]
	}
	RunNodeRestoreTest(t, testCases, "UPDATE t1 SET %s", extractNodeFunc)
}

func TestHavingClauseRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"a", "`a`"},
		{"NULL", "NULL"},
		{"a>b", "`a`>`b`"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Having
	}
	RunNodeRestoreTest(t, testCases, "select 1 from t1 group by 1 HAVING %s", extractNodeFunc)
	for _, tc := range testCases {
		sql := "select 1 from t1 group by 1 HAVING " + tc.sourceSQL
		stmt, err := parser.New().ParseOneStmt(sql)
		if err != nil {
			t.Fatal(err)
		}
		var buf strings.Builder
		want := "SELECT 1 FROM `t1` GROUP BY 1 HAVING " + tc.expectSQL
		if err := stmt.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &buf)); err != nil || buf.String() != want {
			t.Fatalf("SELECT restored %q, want %q; err=%v", buf.String(), want, err)
		}
	}
}

func TestFrameBoundRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"CURRENT ROW", "CURRENT ROW"},
		{"UNBOUNDED PRECEDING", "UNBOUNDED PRECEDING"},
		{"1 PRECEDING", "1 PRECEDING"},
		{"'2' PRECEDING", "'2' PRECEDING"},
		{"? PRECEDING", "? PRECEDING"},
		{"INTERVAL 5 DAY PRECEDING", "INTERVAL 5 DAY PRECEDING"},
		{"UNBOUNDED FOLLOWING", "UNBOUNDED FOLLOWING"},
		{"1 FOLLOWING", "1 FOLLOWING"},
		{"'3' FOLLOWING", "'3' FOLLOWING"},
		{"? FOLLOWING", "? FOLLOWING"},
		{"INTERVAL '2:30' MINUTE_SECOND FOLLOWING", "INTERVAL '2:30' MINUTE_SECOND FOLLOWING"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Items[0].Expr.(*WindowFuncExpr).Spec.Frame.(*FrameClause).Extent.Start
	}
	RunNodeRestoreTest(t, testCases, "select avg(val) over (rows between %s and current row) from t", extractNodeFunc)
}

func TestFrameBoundRestoreRejectsInvalidUnit(t *testing.T) {
	frame := &FrameBound{BaseWindowFrameBound: pkast.BaseWindowFrameBound{Type: Preceding, Expr: &LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "1"}}, Unit: lex.Ident{Name: "INVALID"}}}
	if err := frame.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &bytes.Buffer{})); err == nil {
		t.Fatal("Restore() succeeded with an invalid frame unit")
	}
}

func TestFrameClauseRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"ROWS CURRENT ROW", "ROWS BETWEEN CURRENT ROW AND CURRENT ROW"},
		{"ROWS UNBOUNDED PRECEDING", "ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW"},
		{"ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING", "ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING"},
		{"RANGE BETWEEN ? PRECEDING AND ? FOLLOWING", "RANGE BETWEEN ? PRECEDING AND ? FOLLOWING"},
		{"GROUPS CURRENT ROW", "GROUPS BETWEEN CURRENT ROW AND CURRENT ROW"},
		{"ROWS CURRENT ROW EXCLUDE CURRENT ROW", "ROWS BETWEEN CURRENT ROW AND CURRENT ROW EXCLUDE CURRENT ROW"},
		{"ROWS CURRENT ROW EXCLUDE GROUP", "ROWS BETWEEN CURRENT ROW AND CURRENT ROW EXCLUDE GROUP"},
		{"ROWS CURRENT ROW EXCLUDE TIES", "ROWS BETWEEN CURRENT ROW AND CURRENT ROW EXCLUDE TIES"},
		{"ROWS CURRENT ROW EXCLUDE NO OTHERS", "ROWS BETWEEN CURRENT ROW AND CURRENT ROW EXCLUDE NO OTHERS"},
		{"RANGE BETWEEN INTERVAL 5 DAY PRECEDING AND INTERVAL '2:30' MINUTE_SECOND FOLLOWING", "RANGE BETWEEN INTERVAL 5 DAY PRECEDING AND INTERVAL '2:30' MINUTE_SECOND FOLLOWING"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Items[0].Expr.(*WindowFuncExpr).Spec.Frame
	}
	RunNodeRestoreTest(t, testCases, "select avg(val) over (%s) from t", extractNodeFunc)
}

func TestWindowFrameRestoreRejectsUnknownValues(t *testing.T) {
	t.Parallel()
	current := &FrameBound{BaseWindowFrameBound: pkast.BaseWindowFrameBound{Type: CurrentRow}}
	for _, tc := range []struct {
		name string
		node Node
	}{
		{"zero bound", &FrameBound{}},
		{"invalid direction", &FrameBound{BaseWindowFrameBound: pkast.BaseWindowFrameBound{Type: BoundType(255)}}},
		{"unbounded unknown", &FrameBound{BaseWindowFrameBound: pkast.BaseWindowFrameBound{Unbounded: true}}},
		{"zero frame", &FrameClause{BaseWindowFrameClause: pkast.BaseWindowFrameClause{Extent: FrameExtent{Start: current, End: current}}}},
		{"invalid unit", &FrameClause{BaseWindowFrameClause: pkast.BaseWindowFrameClause{Type: FrameType(255), Extent: FrameExtent{Start: current, End: current}}}},
		{"invalid exclusion", &FrameClause{BaseWindowFrameClause: pkast.BaseWindowFrameClause{Type: Rows, Extent: FrameExtent{Start: current, End: current}, Exclusion: pkast.WindowFrameExclusion(255)}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			if err := tc.node.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); err == nil {
				t.Fatalf("Restore accepted invalid frame: %q", out.String())
			}
		})
	}
}

func TestPartitionByClauseRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"PARTITION BY a", "PARTITION BY `a`"},
		{"PARTITION BY NULL", "PARTITION BY NULL"},
		{"PARTITION BY a, b", "PARTITION BY `a`, `b`"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).Fields.Items[0].Expr.(*WindowFuncExpr).Spec.PartitionBy
	}
	RunNodeRestoreTest(t, testCases, "select avg(val) over (%s rows current row) from t", extractNodeFunc)
}

func TestWindowSpecRestore(t *testing.T) {
	testCases := []NodeRestoreTestCase{
		{"w as ()", "`w` AS ()"},
		{"w as (w1)", "`w` AS (`w1`)"},
		{"w as (w1 order by country)", "`w` AS (`w1` ORDER BY `country`)"},
		{"w as (partition by a order by b rows current row)", "`w` AS (PARTITION BY `a` ORDER BY `b` ROWS BETWEEN CURRENT ROW AND CURRENT ROW)"},
	}
	extractNodeFunc := func(node Node) Node {
		return node.(*SelectStmt).WindowSpecs[0]
	}
	RunNodeRestoreTest(t, testCases, "select rank() over w from t window %s", extractNodeFunc)

	testCases = []NodeRestoreTestCase{
		{"w", "`w`"},
		{"()", "()"},
		{"(w)", "(`w`)"},
		{"(w PARTITION BY country)", "(`w` PARTITION BY `country`)"},
		{"(PARTITION BY a ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING)", "(PARTITION BY `a` ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING)"},
	}
	extractNodeFunc = func(node Node) Node {
		return &node.(*SelectStmt).Fields.Items[0].Expr.(*WindowFuncExpr).Spec
	}
	RunNodeRestoreTest(t, testCases, "select rank() over %s from t window w as (order by a)", extractNodeFunc)
}

// TestSelectIntoStatementRestore pins the charset and filename spelling of the
// INTO OUTFILE option: the charset stays a plain string (not a QualifiedName)
// and is dropped when absent.
func TestSelectIntoStatementRestore(t *testing.T) {
	p := parser.New()
	for _, sql := range []string{
		"select * from t into outfile 'f'",
		"select * from t into outfile 'f' character set utf8mb4",
		"select * from t into dumpfile 'f'",
		"select a from t into @x, @y",
	} {
		t.Run(sql, func(t *testing.T) {
			stmt, err := p.ParseOneStmt(sql)
			if err != nil {
				t.Fatal(err)
			}
			var out strings.Builder
			if err := stmt.(*SelectStmt).Into.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out)); err != nil {
				t.Fatal(err)
			}
			restored := sql[:strings.LastIndex(sql, "into")] + out.String()
			stmt2, err := p.ParseOneStmt(restored)
			if err != nil {
				t.Fatal(err)
			}
			var a, b strings.Builder
			if err := stmt.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &a)); err != nil {
				t.Fatal(err)
			}
			if err := stmt2.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &b)); err != nil {
				t.Fatal(err)
			}
			if a.String() != b.String() {
				t.Fatalf("into-clause restore lost meaning: %q vs %q", a.String(), b.String())
			}
		})
	}

	// The zero value type is not a valid option type; Restore must reject it.
	for _, node := range []*SelectIntoStatement{{}} {
		var out strings.Builder
		err := node.Restore(dformat.NewRestoreCtx(dformat.DefaultRestoreFlags, &out))
		if err == nil || out.Len() != 0 {
			t.Fatalf("SelectIntoStatement.Restore(%+v) wrote %q, err=%v; want no output and an error", node, out.String(), err)
		}
	}
}

package ast_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

// withClauseSQLNoCycle exercises a non-recursive CTE with no CYCLE clause.
const withClauseSQLNoCycle = "WITH c AS (SELECT 1) SELECT * FROM c"

// withClauseSQLSetOpr exercises a CTE on a set-opr query.
const withClauseSQLSetOpr = "WITH c AS (SELECT 1) SELECT 1 UNION SELECT 2"

// withClauseCycleSQL is the MariaDB CYCLE extension on a recursive CTE.
const withClauseCycleSQL = "with recursive c(a) as (select 1) cycle a restrict select * from c"

func TestWithClauseRestoresCycle(t *testing.T) {
	n := dmlParse(t, withClauseCycleSQL)
	sel, ok := n.(*ast.SelectStmt)
	if !ok {
		t.Fatalf("parse = %T, want SelectStmt", n)
	}
	with := sel.With.(*ast.WithClause)
	if !with.IsRecursive || len(with.CycleColumns) != 1 {
		t.Fatalf("CYCLE clause lost: %#v", sel.With)
	}
	// The parsed column must carry Name, Raw and a source Span matching the
	// "a" in the CYCLE list.
	got := with.CycleColumns[0]
	aStart := strings.Index(withClauseCycleSQL, "cycle a") + len("cycle ")
	wantCycle := lex.Ident{Name: "a", Raw: "a", Span: loc.Span{
		Start: loc.ByteOffset(aStart),
		End:   loc.ByteOffset(aStart + 1),
	}}
	if got != wantCycle {
		t.Fatalf("CycleColumns[0] = %+v, want %+v", got, wantCycle)
	}
	var out strings.Builder
	if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "WITH RECURSIVE `c` (`a`) AS (SELECT 1) CYCLE `a` RESTRICT SELECT * FROM `c`"; got != want {
		t.Fatalf("restore = %q, want %q", got, want)
	}
}

func TestWithClauseSetOperationAttachment(t *testing.T) {
	n := dmlParse(t, withClauseSQLSetOpr)
	stmt, ok := n.(*ast.SetOprStmt)
	if !ok {
		t.Fatalf("parse = %T, want SetOprStmt", n)
	}
	if stmt.With == nil || len(stmt.With.CTEs) != 1 {
		t.Fatalf("WITH clause not attached to set-operation query: %#v", stmt.With)
	}
}

func TestWithClauseSpan(t *testing.T) {
	// The WITH clause reduction spans from the WITH keyword to the end of the
	// CYCLE list (or to the last CTE when there is no CYCLE clause).
	for _, tc := range []struct {
		name string
		sql  string
		want loc.Span
	}{
		// "WITH c AS (SELECT 1) SELECT 1": the WITH clause spans [0,20).
		{"no cycle", "WITH c AS (SELECT 1) SELECT 1", loc.Span{Start: 0, End: 20}},
		// A set-opr query: the CTE still occupies the same byte range.
		{"set opr", withClauseSQLSetOpr, loc.Span{Start: 0, End: 20}},
		// The CYCLE ... RESTRICT list is part of the WITH clause, so it spans
		// [0,50) in this input.
		{"with cycle", withClauseCycleSQL, loc.Span{Start: 0, End: 50}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := dmlParse(t, tc.sql)
			var clause *ast.WithClause
			switch st := n.(type) {
			case *ast.SelectStmt:
				clause, _ = st.With.(*ast.WithClause)
			case *ast.SetOprStmt:
				clause = st.With
			default:
				t.Fatalf("parse = %T, want SelectStmt or SetOprStmt", n)
			}
			if clause == nil {
				t.Fatal("WITH clause was lost")
			}
			if got := clause.Span(); got != tc.want {
				t.Fatalf("WITH clause span = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestWithClauseCTESpanAndVisitOrder(t *testing.T) {
	// withClauseSQLNoCycle: "WITH c AS (SELECT 1) SELECT * FROM c"
	//   The CTE "c AS (SELECT 1)" occupies byte [5,20).
	const noCycleSQL = withClauseSQLNoCycle
	const (
		cteStart = 5
		cteEnd   = 20
	)

	n := dmlParse(t, noCycleSQL)
	sel, ok := n.(*ast.SelectStmt)
	if !ok {
		t.Fatalf("parse = %T, want SelectStmt", n)
	}
	with := sel.With.(*ast.WithClause)
	if len(with.CTEs) != 1 || with.CTEs[0].CTECore().Query == nil {
		t.Fatalf("CTE query lost: %#v", with.CTEs)
	}
	// The inner CTE query is the CTE's own SELECT. Its own Span is set by
	// the independent SELECT-migration work, so we only assert that it is
	// a non-nil SelectStmt and leave the span check to that batch.
	if _, ok := with.CTEs[0].CTECore().Query.(*ast.SelectStmt); !ok {
		t.Fatalf("CTE Query = %T, want *ast.SelectStmt", with.CTEs[0].CTECore().Query)
	}

	// The CTE node's span covers "c AS (SELECT 1)" = [5,20).
	wantCTESpan := loc.Span{Start: loc.ByteOffset(cteStart), End: loc.ByteOffset(cteEnd)}
	if got := with.CTEs[0].Span(); got != wantCTESpan {
		t.Fatalf("CTE span = %+v, want %+v", got, wantCTESpan)
	}

	// Visit order: the WithClause must be entered before its CTEs, and the
	// CTE must be entered before its Query.
	var order []string
	_, ok = n.Accept(operatorVisitor{
		enter: func(node ast.Node) (ast.Node, bool) {
			switch node.(type) {
			case *ast.WithClause:
				order = append(order, "with")
			case *ast.CommonTableExpression:
				order = append(order, "cte")
			}
			return node, false
		},
		leave: func(node ast.Node) (ast.Node, bool) { return node, true },
	})
	if !ok {
		t.Fatal("traversal stopped unexpectedly")
	}
	wantOrder := []string{"with", "cte"}
	if !reflect.DeepEqual(order, wantOrder) {
		t.Fatalf("visit order = %v, want %v", order, wantOrder)
	}
}

func TestWithClauseAcceptNilCTEStops(t *testing.T) {
	n := dmlParse(t, "WITH c AS (SELECT 1) SELECT 1")
	sel := n.(*ast.SelectStmt)
	sel.With.(*ast.WithClause).CTEs[0] = nil
	left := false
	_, ok := n.Accept(operatorVisitor{
		enter: func(node ast.Node) (ast.Node, bool) { return node, false },
		leave: func(node ast.Node) (ast.Node, bool) {
			if _, isClause := node.(*ast.WithClause); isClause {
				left = true
			}
			return node, true
		},
	})
	if ok {
		t.Fatal("nil CTE must stop the traversal")
	}
	if left {
		t.Fatal("nil CTE must not reach the WithClause Leave")
	}
}

func TestWithClauseRestoreNilCTEFails(t *testing.T) {
	// A nil CTE in the CTEs list must fail Restore.
	clause := &ast.WithClause{}
	clause.CTEs = []ast.CTENode{nil}
	var out strings.Builder
	err := clause.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out))
	if err == nil {
		t.Fatal("nil CTE must fail Restore")
	}
	// The nil CTE error must carry the index of the failed CTE.
	if !strings.Contains(err.Error(), "nil CTE") || !strings.Contains(err.Error(), "CTE 0") {
		t.Fatalf("error = %v, want 'nil CTE' and 'CTE 0' context", err)
	}
}

func TestWithClauseRestoreNilClause(t *testing.T) {
	var out strings.Builder
	decl := (*ast.WithClause)(nil)
	err := decl.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out))
	if err == nil || out.Len() != 0 {
		t.Fatalf("nil clause must fail and write nothing: out=%q err=%v", out.String(), err)
	}
}

func TestWithClauseRestoreNilCycleColumns(t *testing.T) {
	// With no CYCLE columns, no CYCLE keyword is emitted.
	clause := &ast.WithClause{}
	clause.CTEs = []ast.CTENode{&ast.CommonTableExpression{BaseCommonTableExpression: pkast.BaseCommonTableExpression{
		Name:  lex.Ident{Name: "c"},
		Query: &fakeCTESelect{},
	}}}
	var out strings.Builder
	if err := clause.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); strings.Contains(got, "CYCLE") {
		t.Fatalf("no-cycle clause emitted CYCLE: %q", got)
	}
}

type fakeCTESelect struct {
	pkast.BaseQuery
}

func (n *fakeCTESelect) Accept(v ast.Visitor) (ast.Node, bool) {
	entered, _ := v.Enter(n)
	return v.Leave(entered)
}

func (n *fakeCTESelect) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("SELECT 1")
	return ctx.Err()
}

var _ ast.QueryNode = (*fakeCTESelect)(nil)

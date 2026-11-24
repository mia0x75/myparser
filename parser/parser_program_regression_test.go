package parser_test

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	pkast "github.com/mia0x75/parserkit/ast"
)

func TestProgramPanicRegressions(t *testing.T) {
	for _, src := range []string{
		"create procedure p() begin declare a int; set a=1; end",
		"create procedure p() begin declare a, b double(10,3); set a=1; end",
		"create function f() returns longblob begin declare ret longblob; return ret; end",
		"select /*+ */ 1",
		"select /*+ */ (select /*+ */ 1)",
	} {
		t.Run(src, func(t *testing.T) {
			stmt := mustParseOne(t, src)
			if strings.TrimSpace(stmt.Text()) == "" {
				t.Fatal("statement text was lost")
			}
			if _, ok := stmt.Accept(syntaxVisitor{}); !ok {
				t.Fatal("visitor aborted")
			}
			mustParseOne(t, restoreStmt(t, stmt))
		})
	}
}

func TestProgramRowRoutineRoundTrip(t *testing.T) {
	for _, src := range []string{
		"create procedure p() begin declare r row(a int,b varchar(8)) default (1,'x'); select 2 into r.a; set r.a=3; end",
		"create procedure p() begin declare r row type of db.t; select 1 into r.a; end",
		"create procedure p() begin declare a type of db.t.a default 1; end",
		"create procedure p() begin declare a type of t.a; end",
		"create procedure p() begin declare c cursor for select 1; begin declare r row type of c; fetch next from c into r; end; end",
		"create procedure p() begin declare `a.b` int; declare `@a` int; declare c cursor for select 1,2; fetch from c into `a.b`,`@a`; end",
	} {
		t.Run(src, func(t *testing.T) {
			stmt := mustParseOne(t, src)
			restored := restoreStmt(t, stmt)
			again := mustParseOne(t, restored)
			if got := restoreStmt(t, again); got != restored {
				t.Fatalf("unstable restore: %q -> %q", restored, got)
			}
			if _, ok := stmt.Accept(syntaxVisitor{}); !ok {
				t.Fatal("visitor aborted")
			}
		})
	}
}

func TestProgramFetchTargetAST(t *testing.T) {
	proc := mustParseOne(t, "create procedure p() begin declare `a.b` int; declare c cursor for select 1; fetch c into `a.b`; end").(*ast.CreateProcedureStmt)
	fetch := proc.Body.(*ast.BeginEndStmt).List[2].(*ast.FetchStmt)
	if len(fetch.Targets) != 1 || len(fetch.Targets[0].Name.Parts) != 1 || fetch.Targets[0].Name.Parts[0].Name != "a.b" || fetch.Targets[0].Kind == ast.VariableTargetUser {
		t.Fatalf("FETCH target was lost: %#v", fetch)
	}
	assertVisitorControls(t, fetch, func(n ast.Node) bool { _, ok := n.(*ast.VariableTarget); return ok })
}

func TestProgramRowDefaultAndTypeOfAST(t *testing.T) {
	proc := mustParseOne(t, "create procedure p() begin declare r row(a int,b varchar(8)) default (1,'x'); declare x type of db.t.a; declare y row type of db.t; set r.a=2; end").(*ast.CreateProcedureStmt)
	stmts := proc.Body.(*ast.BeginEndStmt).List
	row := stmts[0].(*ast.DeclareStmt)
	if len(row.RowType) != 2 || len(row.RowType[1].Type.TypeCore().Modifiers) != 1 ||
		row.RowType[1].Type.TypeCore().Modifiers[0].(*ast.LiteralExpr).Value != "8" || row.Default == nil {
		t.Fatalf("ROW type was lost: %#v", row)
	}
	ref := stmts[1].(*ast.DeclareStmt)
	if ref.TypeRef == nil || ref.RowTypeRef || len(ref.TypeRef.Parts) != 3 || ref.TypeRef.Parts[0].Name != "db" || ref.TypeRef.Parts[1].Name != "t" || ref.TypeRef.Parts[2].Name != "a" {
		t.Fatalf("TYPE OF reference was lost: %#v", ref)
	}
	if ref := stmts[2].(*ast.DeclareStmt); ref.TypeRef == nil || !ref.RowTypeRef {
		t.Fatalf("ROW TYPE OF reference was lost: %#v", ref)
	}
	assign := stmts[3].(*ast.SetStmt).Variables[0]
	if assign.Target == nil || len(assign.Target.Name.Parts) != 2 || assign.Target.Name.Parts[0].Name != "r" || assign.Target.Name.Parts[1].Name != "a" {
		t.Fatalf("ROW assignment was lost: %#v", assign)
	}
	assertVisitorControls(t, proc, func(n ast.Node) bool { _, ok := n.(*ast.ColumnDef); return ok })
}

func TestProgramLoopAndBeginEndRoundTrip(t *testing.T) {
	// These forms were also checked against the pinned MySQL 8.4.13 and
	// MariaDB 12.3.3 servers in testdata/docker-compose.yml.
	cases := []struct{ src, restored string }{
		{"for i in 1..31 do select i; end for", "FOR `i` IN 1..31 DO SELECT `i`; END FOR"},
		{"for i in 1.5..2.5 do select i; end for", "FOR `i` IN 1.5..2.5 DO SELECT `i`; END FOR"},
		{"create procedure p() begin outer_loop: for i in 1..2 do select i; end for outer_loop; end", "CREATE DEFINER = CURRENT_USER PROCEDURE `p`() BEGIN\n  `outer_loop`: FOR `i` IN 1..2 DO SELECT `i`; END FOR `outer_loop`;\nEND"},
		{"begin not atomic select 1; end", "BEGIN NOT ATOMIC\n  SELECT 1;\nEND"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			runRestoreTest(t, tc.src, tc.restored, true)
		})
	}
}

func TestProgramLoopSyntaxErrors(t *testing.T) {
	assertParseErrors(t, []string{
		"top_label: for i in 1..2 do select i; end for top_label",
	})
}

func TestProgramForBeginEndASTAndVisitors(t *testing.T) {
	stmt := mustParseOne(t, "for i in 1..2 do select 3; end for")
	loop := stmt.(*ast.ForStmt)
	if loop.Variable.Name != "i" || loop.LowerBound == nil || loop.UpperBound == nil || loop.Query != nil || len(loop.List) != 1 {
		t.Fatalf("FOR AST was not retained: %#v", loop)
	}
	values := 0
	_, ok := stmt.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch n := n.(type) {
		case *ast.LiteralExpr:
			values++
			return replaceLiteral(n, "9"), false
		}
		return n, false
	}})
	if !ok || values != 3 || restoreStmt(t, stmt) != "FOR `i` IN 9..9 DO SELECT 9; END FOR" {
		t.Fatalf("FOR visitor replacement lost: ok=%v values=%d sql=%s", ok, values, restoreStmt(t, stmt))
	}

	stmt = mustParseOne(t, "begin not atomic select 1; end")
	block := stmt.(*ast.BeginEndStmt)
	if !block.NotAtomic || len(block.List) != 1 {
		t.Fatalf("top-level compound statement AST was not retained: %#v", block)
	}
}

func TestProgramAnalyseASTAndVisitors(t *testing.T) {
	stmt := mustParseOne(t, "select 1 procedure analyse(2+3) for update")
	sel := stmt.(*ast.SelectStmt)
	if sel.ProcedureAnalyse == nil || len(sel.ProcedureAnalyse.Args) != 1 || sel.LockTp != ast.SelectLockForUpdate {
		t.Fatalf("PROCEDURE ANALYSE AST was not retained: %#v", sel)
	}
	values := 0
	_, ok := stmt.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch n := n.(type) {
		case *ast.LiteralExpr:
			values++
			return replaceLiteral(n, "9"), false
		}
		return n, false
	}})
	if !ok || values != 3 {
		t.Fatalf("PROCEDURE ANALYSE visitor: ok=%v values=%d, want 3", ok, values)
	}
	if got := restoreStmt(t, stmt); got != "SELECT 9 PROCEDURE ANALYSE(9+9) FOR UPDATE" {
		t.Fatalf("PROCEDURE ANALYSE visitor replacement lost: %s", got)
	}
}

func TestProgramCastTimeZoneASTAndVisitors(t *testing.T) {
	stmt := mustParseOne(t, "select cast(1 at time zone interval '+00:00' as datetime(3))")
	cast := stmt.(*ast.SelectStmt).Fields.Items[0].Expr.(*ast.FuncCastExpr)
	if cast.TimeZone == nil || !cast.TimeZone.Interval || cast.TimeZone.Local || cast.TimeZone.Expr == nil ||
		cast.Type.Keyword() != "DATETIME" || len(cast.Type.Modifiers) != 1 || cast.Type.Modifiers[0].(*ast.LiteralExpr).Value != "3" {
		t.Fatalf("CAST time-zone AST was not retained: %#v", cast)
	}
	_, ok := stmt.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		switch n := n.(type) {
		case *ast.TypeName:
			// Type modifiers keep their own spelling.
			return n, true
		case *ast.LiteralExpr:
			if n.Kind == pkast.LiteralString || n.Kind == pkast.LiteralBytes {
				return replaceLiteral(n, "UTC"), false
			}
			return replaceLiteral(n, "9"), false
		}
		return n, false
	}})
	if !ok {
		t.Fatal("CAST time-zone visitor aborted")
	}
	if got := restoreStmt(t, stmt); got != "SELECT CAST(9 AT TIME ZONE INTERVAL 'UTC' AS DATETIME(3))" {
		t.Fatalf("CAST time-zone visitor replacement lost: %s", got)
	}
	mustParseOne(t, restoreStmt(t, stmt))
}

func TestProgramStatementVisitors(t *testing.T) {
	for _, src := range []string{
		"call p(1)",
		"set statement x=1 for call p(2)",
		"with c as (select 1 as a) update t set a=2",
		"with c as (select 1 as a) delete from t where a=2",
	} {
		t.Run(src, func(t *testing.T) {
			stmt := mustParseOne(t, src)
			values := 0
			_, ok := stmt.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
				switch n := n.(type) {
				case *ast.LiteralExpr:
					values++
					return replaceLiteral(n, "9"), false
				}
				return n, false
			}})
			if !ok || values == 0 {
				t.Fatalf("visitor did not reach expressions: ok=%v values=%d", ok, values)
			}
			if got := restoreStmt(t, stmt); !strings.Contains(got, "9") {
				t.Fatalf("visitor replacement lost: %s", got)
			}
			_, ok = stmt.Accept(syntaxVisitor{leave: func(n ast.Node) (ast.Node, bool) {
				return n, !isValueOrLiteral(n)
			}})
			if ok {
				t.Fatal("visitor abort was not propagated")
			}
			entered := 0
			_, ok = stmt.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
				entered++
				return n, true
			}})
			if !ok || entered != 1 {
				t.Fatalf("skipChildren: ok=%v entered=%d", ok, entered)
			}
		})
	}
}

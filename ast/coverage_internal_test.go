package ast

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/model"
	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/myparser/types"
)

// nopVisitor keeps the node unchanged, so Accept methods can run without a
// real transform.
type nopVisitor struct{}

func (nopVisitor) Enter(in Node) (Node, bool) { return in, false }
func (nopVisitor) Leave(in Node) (Node, bool) { return in, true }

func restoreNode(t *testing.T, n interface {
	Restore(*format.RestoreCtx) error
}) string {
	t.Helper()
	var sb strings.Builder
	if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
		t.Fatalf("restore %T: %v", n, err)
	}
	return sb.String()
}

func ciStr(s string) model.CIStr {
	return model.NewCIStr(s)
}

func TestStatementEnumStrings(t *testing.T) {
	for algo, want := range map[AlterAlgorithm]string{
		AlterAlgorithmDefault: "DEFAULT",
		AlterAlgorithmCopy:    "COPY",
		AlterAlgorithmInplace: "INPLACE",
		AlterAlgorithmInstant: "INSTANT",
		AlterAlgorithm(99):    "DEFAULT",
	} {
		if got := algo.String(); got != want {
			t.Fatalf("AlterAlgorithm(%d).String() = %q, want %q", algo, got, want)
		}
	}
	for lock, want := range map[SelectLockType]string{
		SelectLockNone:        "none",
		SelectLockForUpdate:   "for update",
		SelectLockInShareMode: "in share mode",
		SelectLockForShare:    "for share",
		SelectLockType(99):    "unsupported select lock type",
	} {
		if got := lock.String(); got != want {
			t.Fatalf("SelectLockType(%d).String() = %q, want %q", lock, got, want)
		}
	}
}

func TestColumnNameOrigColNameAndIdentString(t *testing.T) {
	cases := []struct {
		name *ColumnName
		want string
	}{
		{&ColumnName{Name: ciStr("a")}, "a"},
		{&ColumnName{Name: ciStr("a"), Table: ciStr("t")}, "t.a"},
		{&ColumnName{Name: ciStr("a"), Table: ciStr("t"), Schema: ciStr("s")}, "s.t.a"},
	}
	for _, tt := range cases {
		if got := tt.name.OrigColName(); got != tt.want {
			t.Fatalf("OrigColName() = %q, want %q", got, tt.want)
		}
	}
	if got := (Ident{Name: ciStr("t")}).String(); got != "t" {
		t.Fatalf("Ident.String() = %q, want %q", got, "t")
	}
	if got := (Ident{Schema: ciStr("s"), Name: ciStr("t")}).String(); got != "s.t" {
		t.Fatalf("Ident.String() = %q, want %q", got, "s.t")
	}
}

func TestPositionExprFormat(t *testing.T) {
	var sb strings.Builder
	(&PositionExpr{N: 3}).Format(&sb)
	if sb.String() != "3" {
		t.Fatalf("PositionExpr.Format() = %q, want %q", sb.String(), "3")
	}
}

func TestReadOnlyChecker(t *testing.T) {
	plain := &SelectStmt{Where: &PositionExpr{N: 1}}
	if !IsReadOnly(plain) {
		t.Fatal("plain select should be read only")
	}
	writesVar := &SelectStmt{Where: &VariableExpr{Name: "x", Value: &PositionExpr{N: 1}}}
	if IsReadOnly(writesVar) {
		t.Fatal("select that assigns a user variable should not be read only")
	}
}

func TestFlagHelpers(t *testing.T) {
	e := &PositionExpr{}
	if HasWindowFlag(e) {
		t.Fatal("HasWindowFlag() = true, want false")
	}
	e.SetFlag(FlagHasWindowFunc)
	if !HasWindowFlag(e) {
		t.Fatal("HasWindowFlag() = false, want true")
	}
}

func TestPeriodDefinition(t *testing.T) {
	def := &PeriodDefinition{
		Name:  ciStr("p"),
		Start: &ColumnName{Name: ciStr("s")},
		End:   &ColumnName{Name: ciStr("e")},
	}
	want := "PERIOD FOR `p` (`s`,`e`)"
	if got := restoreNode(t, def); got != want {
		t.Fatalf("PeriodDefinition.Restore() = %q, want %q", got, want)
	}
	def.Accept(nopVisitor{})
}

func TestProcedureAnalyseClause(t *testing.T) {
	clause := &ProcedureAnalyseClause{Args: []ExprNode{&PositionExpr{N: 1}, &PositionExpr{N: 2}}}
	want := "PROCEDURE ANALYSE(1,2)"
	if got := restoreNode(t, clause); got != want {
		t.Fatalf("ProcedureAnalyseClause.Restore() = %q, want %q", got, want)
	}
	clause.Accept(nopVisitor{})
}

func TestComponentSetValue(t *testing.T) {
	value := &PositionExpr{N: 1}
	withScope := &ComponentSetValue{Scope: "SET", Name: "n", Value: value}
	if got, want := restoreNode(t, withScope), "SET `n`=1"; got != want {
		t.Fatalf("ComponentSetValue.Restore() = %q, want %q", got, want)
	}
	withoutScope := &ComponentSetValue{Name: "n", Value: value}
	if got, want := restoreNode(t, withoutScope), "`n`=1"; got != want {
		t.Fatalf("ComponentSetValue.Restore() = %q, want %q", got, want)
	}
	withScope.Accept(nopVisitor{})
}

func TestResourceOptionRestore(t *testing.T) {
	cases := map[int]string{
		MaxQueriesPerHour:     "MAX_QUERIES_PER_HOUR 5",
		MaxUpdatesPerHour:     "MAX_UPDATES_PER_HOUR 5",
		MaxConnectionsPerHour: "MAX_CONNECTIONS_PER_HOUR 5",
		MaxUserConnections:    "MAX_USER_CONNECTIONS 5",
	}
	for tp, want := range cases {
		opt := &ResourceOption{Type: tp, Count: 5}
		if got := restoreNode(t, opt); got != want {
			t.Fatalf("ResourceOption(%d).Restore() = %q, want %q", tp, got, want)
		}
	}
	var sb strings.Builder
	if err := (&ResourceOption{Type: 99}).Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err == nil {
		t.Fatal("expected error for unsupported ResourceOption.Type")
	}
}

func TestPasswordOrLockOptionRestore(t *testing.T) {
	cases := map[int]string{
		PasswordExpire:         "PASSWORD EXPIRE",
		PasswordExpireDefault:  "PASSWORD EXPIRE DEFAULT",
		PasswordExpireNever:    "PASSWORD EXPIRE NEVER",
		PasswordExpireInterval: "PASSWORD EXPIRE NEVER 5 DAY",
		Lock:                   "ACCOUNT LOCK",
		Unlock:                 "ACCOUNT UNLOCK",
	}
	for tp, want := range cases {
		opt := &PasswordOrLockOption{Type: tp, Count: 5}
		if got := restoreNode(t, opt); got != want {
			t.Fatalf("PasswordOrLockOption(%d).Restore() = %q, want %q", tp, got, want)
		}
	}
}

func TestTableOptimizerHint(t *testing.T) {
	maxExec := &TableOptimizerHint{HintName: ciStr("max_execution_time"), MaxExecutionTime: 10}
	if got, want := restoreNode(t, maxExec), "MAX_EXECUTION_TIME(10)"; got != want {
		t.Fatalf("TableOptimizerHint.Restore() = %q, want %q", got, want)
	}
	tables := &TableOptimizerHint{HintName: ciStr("bka"), Tables: []model.CIStr{ciStr("t1"), ciStr("t2")}}
	if got, want := restoreNode(t, tables), "BKA(`t1`, `t2`)"; got != want {
		t.Fatalf("TableOptimizerHint.Restore() = %q, want %q", got, want)
	}
	tables.Accept(nopVisitor{})
}

func TestAnalyzeTableStmtVariants(t *testing.T) {
	all := &AnalyzeTableStmt{
		TableNames:    []*TableName{{Name: ciStr("t")}},
		Persistent:    true,
		PersistentAll: true,
	}
	if got, want := restoreNode(t, all), "ANALYZE TABLE `t` PERSISTENT FOR ALL"; got != want {
		t.Fatalf("AnalyzeTableStmt.Restore() = %q, want %q", got, want)
	}
	named := &AnalyzeTableStmt{
		TableNames:  []*TableName{{Name: ciStr("t")}},
		Persistent:  true,
		ColumnNames: []model.CIStr{ciStr("c")},
		IndexNames:  nil,
	}
	if got, want := restoreNode(t, named), "ANALYZE TABLE `t` PERSISTENT FOR COLUMNS (`c`) INDEXES ALL"; got != want {
		t.Fatalf("AnalyzeTableStmt.Restore() = %q, want %q", got, want)
	}
	all.Accept(nopVisitor{})
}

func TestStatsStatementRestore(t *testing.T) {
	drop := &DropStatsStmt{Table: &TableName{Name: ciStr("t")}}
	if got, want := restoreNode(t, drop), "DROP STATS `t`"; got != want {
		t.Fatalf("DropStatsStmt.Restore() = %q, want %q", got, want)
	}
	drop.Accept(nopVisitor{})

	load := &LoadStatsStmt{Path: "stats.json"}
	if got, want := restoreNode(t, load), "LOAD STATS 'stats.json'"; got != want {
		t.Fatalf("LoadStatsStmt.Restore() = %q, want %q", got, want)
	}
	load.Accept(nopVisitor{})
}

func TestSpLabelSettersAndCondition(t *testing.T) {
	begin := &BeginEndStmt{}
	while := &WhileStmt{}
	repeat := &RepeatStmt{}
	loop := &LoopStmt{}
	forStmt := &ForStmt{}
	for _, s := range []LabeledStmt{begin, while, repeat, loop, forStmt} {
		s.SetLabel("lbl")
	}
	for _, label := range []string{begin.Label, while.Label, repeat.Label, loop.Label, forStmt.Label} {
		if label != "lbl" {
			t.Fatalf("SetLabel() = %q, want %q", label, "lbl")
		}
	}

	cond := NewErrnoHandlerCondition(1146)
	if cond.Type != HandlerConditionErrorCode || cond.Value != "1146" {
		t.Fatalf("NewErrnoHandlerCondition() = %+v", cond)
	}
	stmt := &DeclareConditionStmt{Name: "c", Condition: cond}
	if got, want := restoreNode(t, stmt), "DECLARE `c` CONDITION FOR 1146"; got != want {
		t.Fatalf("DeclareConditionStmt.Restore() = %q, want %q", got, want)
	}
	stmt.Accept(nopVisitor{})
}

func TestRestoreStoredProgramName(t *testing.T) {
	cases := map[string]string{
		"@var":  "@var",
		"proc":  "`proc`",
		"db.fn": "`db`.`fn`",
	}
	for name, want := range cases {
		var sb strings.Builder
		restoreStoredProgramName(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb), name)
		if got := sb.String(); got != want {
			t.Fatalf("restoreStoredProgramName(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestJSONTableRestoreAndAccept(t *testing.T) {
	expr := func(n int) ExprNode { return &PositionExpr{N: n} }

	nested := &JSONTableColumn{
		Nested:  true,
		Path:    "$.n",
		Columns: []*JSONTableColumn{{Name: ciStr("x"), Path: "$.x"}},
	}
	table := &JSONTable{
		Expr: expr(10),
		Path: "$.a",
		Columns: []*JSONTableColumn{
			{Name: ciStr("ord"), Ordinality: true},
			{Name: ciStr("v"), Tp: types.NewFieldType(mysql.TypeLong), Path: "$.v"},
			{Name: ciStr("e"), Exists: true, Path: "$.e"},
			nested,
		},
	}
	want := "JSON_TABLE(10, '$.a' COLUMNS (`ord` FOR ORDINALITY, `v` INT PATH '$.v', `e` EXISTS PATH '$.e', NESTED PATH '$.n' COLUMNS (`x` PATH '$.x')))"
	if got := restoreNode(t, table); got != want {
		t.Fatalf("JSONTable.Restore() = %q, want %q", got, want)
	}
	table.Accept(nopVisitor{})

	behaviors := []*JSONTableBehavior{
		NewJSONTableColumnBehavior(JSONTableBehaviorError, true, nil),
		NewJSONTableColumnBehavior(JSONTableBehaviorDefault, false, expr(1)),
		NewJSONTableColumnBehavior(JSONTableBehaviorValue, false, nil),
		NewJSONTableColumnBehavior(JSONTableBehaviorValue, false, expr(2)),
	}
	col := &JSONTableColumn{Name: ciStr("c"), Path: "$.c"}
	AssignJSONTableBehaviors(col, behaviors)
	AssignJSONTableBehavior(nil, behaviors[0])
	AssignJSONTableBehaviors(nil, behaviors)
	if col.OnEmpty == nil || col.OnError == nil {
		t.Fatalf("AssignJSONTableBehaviors() did not set both clauses: %+v", col)
	}
	if got := restoreNode(t, col); !strings.Contains(got, "ERROR ON EMPTY") {
		t.Fatalf("JSONTableColumn.Restore() = %q, missing ON EMPTY behavior", got)
	}
	col.Accept(nopVisitor{})
}

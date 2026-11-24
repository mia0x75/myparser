package ast

import (
	"errors"
	"io"
	"strings"
	"testing"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"

	"github.com/mia0x75/myparser/auth"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/myparser/opcode"
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

// qualName is a test-only helper to build pure name values.
func qualName(parts ...string) QualifiedName {
	q := QualifiedName{}
	for _, p := range parts {
		q.Parts = append(q.Parts, lex.Ident{Name: p, Raw: p})
	}
	return q
}

func TestColumnNameOriginalQualifiedName(t *testing.T) {
	cases := []struct {
		name QualifiedName
		want string
	}{
		{qualName("a"), "a"},
		{qualName("t", "a"), "t.a"},
		{qualName("s", "t", "a"), "s.t.a"},
	}
	for _, tt := range cases {
		if got := originalQualifiedName(tt.name); got != tt.want {
			t.Fatalf("originalQualifiedName() = %q, want %q", got, tt.want)
		}
	}
}

func TestPositionExprFormat(t *testing.T) {
	var sb strings.Builder
	(&PositionExpr{N: 3}).Format(&sb)
	if sb.String() != "3" {
		t.Fatalf("PositionExpr.Format() = %q, want %q", sb.String(), "3")
	}
}

// The parameterized position restores the parameter, not the numeric
// fallback.
func TestPositionExprRestoreParameter(t *testing.T) {
	for _, tc := range []struct {
		name string
		node *PositionExpr
		want string
	}{
		{"numeric", &PositionExpr{N: 3}, "3"},
		{"param", &PositionExpr{P: NewParamMarkerExprAt(loc.InvalidSpan(), 0)}, "?"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var sb strings.Builder
			if err := tc.node.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
				t.Fatal(err)
			}
			if sb.String() != tc.want {
				t.Fatalf("Restore() = %q, want %q", sb.String(), tc.want)
			}
		})
	}
}

func TestReadOnlyChecker(t *testing.T) {
	plain := &SelectStmt{Where: &PositionExpr{N: 1}}
	if !IsReadOnly(plain) {
		t.Fatal("plain select should be read only")
	}
	writesVar := &SelectStmt{Where: &VariableExpr{Target: &VariableTarget{Name: QualifiedName{Parts: []lex.Ident{{Name: "x"}}}, Kind: VariableTargetLocal}, Value: &PositionExpr{N: 1}}}
	if IsReadOnly(writesVar) {
		t.Fatal("select that assigns a user variable should not be read only")
	}
	writesOutput := &SelectStmt{Into: &SelectIntoStatement{}}
	if IsReadOnly(writesOutput) {
		t.Fatal("select with INTO should not be read only")
	}
}

func TestFlagHelpers(t *testing.T) {
	e := &PositionExpr{}
	if HasWindowFlag(e) {
		t.Fatal("HasWindowFlag() = true, want false")
	}
	if got := ExprFlag(e); got != FlagHasReference {
		t.Fatalf("ExprFlag(PositionExpr{}) = %d, want %d", got, FlagHasReference)
	}
}

func TestPeriodDefinition(t *testing.T) {
	def := &PeriodDefinition{
		Name:  lex.Ident{Name: "p"},
		Start: qualName("s"),
		End:   qualName("e"),
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

func TestOptimizerHint(t *testing.T) {
	maxExec := &OptimizerHint{Kind: OptimizerHintMaxExecutionTime, MaxExecutionTime: 10}
	if got, want := restoreNode(t, maxExec), "MAX_EXECUTION_TIME(10)"; got != want {
		t.Fatalf("OptimizerHint.Restore() = %q, want %q", got, want)
	}
	if err := (&OptimizerHint{}).Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, io.Discard)); err == nil {
		t.Fatal("unknown hint kind accepted")
	}
	maxExec.Accept(nopVisitor{})
}

func TestAnalyzeTableStmtVariants(t *testing.T) {
	all := &AnalyzeTableStmt{
		TableNames:    []*TableName{{BaseTableRef: pkast.BaseTableRef{Name: QualifiedName{Parts: []lex.Ident{{Name: "t"}}}}}},
		Persistent:    true,
		PersistentAll: true,
	}
	if got, want := restoreNode(t, all), "ANALYZE TABLE `t` PERSISTENT FOR ALL"; got != want {
		t.Fatalf("AnalyzeTableStmt.Restore() = %q, want %q", got, want)
	}
	named := &AnalyzeTableStmt{
		TableNames: []*TableName{{BaseTableRef: pkast.BaseTableRef{Name: QualifiedName{Parts: []lex.Ident{{Name: "t"}}}}}},
		Persistent: true,
		Columns:    []lex.Ident{{Name: "c"}},
		IndexNames: nil,
	}
	if got, want := restoreNode(t, named), "ANALYZE TABLE `t` PERSISTENT FOR COLUMNS (`c`) INDEXES ALL"; got != want {
		t.Fatalf("AnalyzeTableStmt.Restore() = %q, want %q", got, want)
	}
	all.Accept(nopVisitor{})
}

func TestStatsStatementRestore(t *testing.T) {
	drop := &DropStatsStmt{Table: &TableName{BaseTableRef: pkast.BaseTableRef{Name: QualifiedName{Parts: []lex.Ident{{Name: "t"}}}}}}
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
		s.SetLabel(lex.Ident{Name: "lbl"})
	}
	for _, label := range []lex.Ident{begin.Label, while.Label, repeat.Label, loop.Label, forStmt.Label} {
		if label.Name != "lbl" {
			t.Fatalf("SetLabel() = %q, want %q", label.Name, "lbl")
		}
	}

	cond := NewErrnoHandlerCondition(1146)
	if cond.Type != HandlerConditionErrorCode || cond.Value.Name != "1146" {
		t.Fatalf("NewErrnoHandlerCondition() = %+v", cond)
	}
	stmt := &DeclareConditionStmt{Name: lex.Ident{Name: "c"}, Condition: cond}
	if got, want := restoreNode(t, stmt), "DECLARE `c` CONDITION FOR 1146"; got != want {
		t.Fatalf("DeclareConditionStmt.Restore() = %q, want %q", got, want)
	}
	stmt.Accept(nopVisitor{})
}

func TestJSONTableRestoreAndAccept(t *testing.T) {
	expr := func(n int) ExprNode { return &PositionExpr{N: n} }

	nested := &JSONTableColumn{
		Kind:    JSONTableColumnNested,
		Name:    lex.Ident{Span: loc.InvalidSpan()},
		Path:    "$.n",
		Columns: []*JSONTableColumn{{Name: lex.Ident{Name: "x"}, Kind: JSONTableColumnPath, Type: NewTypeName("INT"), Path: "$.x"}},
	}
	table := &JSONTable{
		Expr: expr(10),
		Path: "$.a",
		Columns: []*JSONTableColumn{
			{Name: lex.Ident{Name: "ord"}, Kind: JSONTableColumnOrdinality},
			{Name: lex.Ident{Name: "v"}, Kind: JSONTableColumnPath, Type: NewTypeName("INT"), Path: "$.v"},
			{Name: lex.Ident{Name: "e"}, Kind: JSONTableColumnExists, Type: NewTypeName("INT"), Path: "$.e"},
			nested,
		},
	}
	want := "JSON_TABLE(10, '$.a' COLUMNS (`ord` FOR ORDINALITY, `v` INT PATH '$.v', `e` INT EXISTS PATH '$.e', NESTED '$.n' COLUMNS (`x` INT PATH '$.x')))"
	if got := restoreNode(t, table); got != want {
		t.Fatalf("JSONTable.Restore() = %q, want %q", got, want)
	}
	table.Accept(nopVisitor{})

	behaviors := []*JSONTableBehavior{
		{Kind: JSONTableBehaviorError, On: JSONTableBehaviorOnEmpty},
		{Kind: JSONTableBehaviorDefault, On: JSONTableBehaviorOnError, Value: expr(1)},
		{Kind: JSONTableBehaviorNull, On: JSONTableBehaviorOnEmpty},
		{Kind: JSONTableBehaviorNull, On: JSONTableBehaviorOnError},
	}
	col := &JSONTableColumn{Name: lex.Ident{Name: "c"}, Kind: JSONTableColumnPath, Type: NewTypeName("INT"), Path: "$.c", Behaviors: behaviors[:2]}
	if len(col.Behaviors) != 2 {
		t.Fatalf("behavior clauses were not retained: %+v", col)
	}
	if got := restoreNode(t, col); !strings.Contains(got, "ERROR ON EMPTY DEFAULT 1 ON ERROR") {
		t.Fatalf("JSONTableColumn.Restore() = %q, missing ordered behavior clauses", got)
	}
	col.Accept(nopVisitor{})
}

// restoreFailWriter rejects every write, so the restore context carries a
// pending error: the sentinel error, or a short write when short is set.
type restoreFailWriter struct {
	err   error
	short bool
}

func (w *restoreFailWriter) Write(p []byte) (int, error) {
	if w.short {
		return 0, nil
	}
	return 0, w.err
}

// pendingRestoreCtx returns a restore context whose writer already failed, so
// that a pending error can be observed by the restore helpers.
func pendingRestoreCtx(t *testing.T, w io.Writer) *format.RestoreCtx {
	t.Helper()
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, w)
	ctx.WritePlain("x") // Seed the pending writer error.
	if ctx.Err() == nil {
		t.Fatal("expected the restore context to carry a writer error")
	}
	return ctx
}

// TestRestorePropagatesWriterError checks the statements whose restore path
// used to swallow the writer error, because the helper returned nil although
// the context already carried a pending error.
func TestRestorePropagatesWriterError(t *testing.T) {
	writeErr := errors.New("write failed")
	statements := []struct {
		name string
		node interface {
			Restore(*format.RestoreCtx) error
		}
	}{
		{name: "create logfile group", node: &CreateLogfileGroupStmt{Name: lex.Ident{Name: "g"}}},
		{name: "alter logfile group", node: &AlterLogfileGroupStmt{Name: lex.Ident{Name: "g"}}},
		{name: "drop logfile group", node: &DropLogfileGroupStmt{Name: lex.Ident{Name: "g"}}},
		{name: "reset master", node: &ResetStmt{Tp: ResetMaster}},
		{name: "empty row", node: &RowExpr{}},
	}
	writers := []struct {
		name    string
		short   bool
		wantErr error
	}{
		{name: "writer error", wantErr: writeErr},
		{name: "short write", short: true, wantErr: io.ErrShortWrite},
	}
	for _, w := range writers {
		for _, s := range statements {
			t.Run(w.name+"/"+s.name, func(t *testing.T) {
				ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &restoreFailWriter{err: writeErr, short: w.short})
				err := s.node.Restore(ctx)
				if !errors.Is(err, w.wantErr) {
					t.Fatalf("got error %v, want %v", err, w.wantErr)
				}
				if err != ctx.Err() {
					t.Fatalf("got error %v, want ctx error %v", err, ctx.Err())
				}
			})
		}
	}
}

func TestRowRestorePropagatesChildWriterError(t *testing.T) {
	writeErr := errors.New("write failed")
	for _, tc := range []struct {
		name  string
		short bool
		want  error
	}{
		{"writer error", false, writeErr},
		{"short write", true, io.ErrShortWrite},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row := &RowExpr{}
			row.Items = []ExprNode{&LiteralExpr{BaseLiteralExpr: pkast.BaseLiteralExpr{Kind: pkast.LiteralInteger, Value: "1"}}}
			ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &restoreFailWriter{err: writeErr, short: tc.short})
			if err := row.Restore(ctx); !errors.Is(err, tc.want) {
				t.Fatalf("restore error = %v, want %v", err, tc.want)
			}
			if !errors.Is(ctx.Err(), tc.want) {
				t.Fatalf("context error = %v, want %v", ctx.Err(), tc.want)
			}
		})
	}
}

// TestRestoreHelpersPropagatePendingError checks that the restore helpers
// forward the pending context error instead of reporting success.
func TestRestoreHelpersPropagatePendingError(t *testing.T) {
	writeErr := errors.New("write failed")
	helpers := []struct {
		name    string
		restore func(ctx *format.RestoreCtx) error
	}{
		{name: "logfile group options", restore: func(ctx *format.RestoreCtx) error {
			return restoreLogfileGroupOptions(ctx, nil)
		}},
		{name: "reset option", restore: func(ctx *format.RestoreCtx) error {
			return (&ResetStmt{Tp: ResetMaster}).restoreOption(ctx)
		}},
		{name: "user identity current user", restore: func(ctx *format.RestoreCtx) error {
			return restoreUserIdentity(ctx, &auth.UserIdentity{CurrentUser: true})
		}},
		{name: "user identity", restore: func(ctx *format.RestoreCtx) error {
			return restoreUserIdentity(ctx, &auth.UserIdentity{Username: "u", Hostname: "h"})
		}},
		{name: "expr nodes", restore: func(ctx *format.RestoreCtx) error {
			return restoreExprNodes(ctx, nil)
		}},
		{name: "json table behaviors", restore: func(ctx *format.RestoreCtx) error {
			return (&JSONTableBehavior{Kind: JSONTableBehaviorNull, On: JSONTableBehaviorOnEmpty}).Restore(ctx)
		}},
		{name: "json table columns", restore: func(ctx *format.RestoreCtx) error {
			return restoreJSONTableColumns(ctx, nil)
		}},
		{name: "vcpu list", restore: func(ctx *format.RestoreCtx) error {
			return restoreVCPUList(ctx, nil)
		}},
		{name: "func params", restore: func(ctx *format.RestoreCtx) error {
			return restoreFuncParams(ctx, nil)
		}},
		{name: "stmt list", restore: func(ctx *format.RestoreCtx) error {
			return restoreStmtList(ctx, nil)
		}},
		{name: "priority no priority", restore: func(ctx *format.RestoreCtx) error {
			priority := mysql.NoPriority
			return priority.Restore(ctx)
		}},
		{name: "opcode", restore: func(ctx *format.RestoreCtx) error {
			return opcode.EQ.Restore(ctx)
		}},
	}
	for _, h := range helpers {
		t.Run(h.name, func(t *testing.T) {
			ctx := pendingRestoreCtx(t, &restoreFailWriter{err: writeErr})
			err := h.restore(ctx)
			if !errors.Is(err, writeErr) {
				t.Fatalf("got error %v, want %v", err, writeErr)
			}
			if err != ctx.Err() {
				t.Fatalf("got error %v, want ctx error %v", err, ctx.Err())
			}
		})
	}
}

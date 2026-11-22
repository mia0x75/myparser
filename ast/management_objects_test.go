package ast_test

// p59 59.5/59.6a/59.2/59.7 management objects regression tests:
// tablespace/logfile-group/resource-group/server/plugin names,
// GRANT/REVOKE segmented objects and optimizer hints.

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/auth"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/mysql"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"

	. "github.com/mia0x75/myparser/ast"
)

// --- shared helpers ---

// mgmtRestore writes n to a buffer with the default restore flags.
func mgmtRestore(t *testing.T, n Node) string {
	t.Helper()
	var out strings.Builder
	if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
		t.Fatalf("restore %T: %v", n, err)
	}
	return out.String()
}

// mgmtRestoreWithFlags restores with explicit flags.
func mgmtRestoreWithFlags(t *testing.T, n Node, flags format.RestoreFlags) string {
	t.Helper()
	var out strings.Builder
	if err := n.Restore(format.NewRestoreCtx(flags, &out)); err != nil {
		t.Fatalf("restore %T: %v", n, err)
	}
	return out.String()
}

// mgmtRestoreErr restores n, expecting an error.
func mgmtRestoreErr(t *testing.T, n Node) error {
	t.Helper()
	var out strings.Builder
	err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out))
	if err == nil {
		t.Fatalf("restore %T: expected error, got %q", n, out.String())
	}
	return err
}

// mgmtMangledWriter fails the first write, or short-writes when short is set.
type mgmtMangledWriter struct {
	err   error
	short bool
}

func (w *mgmtMangledWriter) Write(p []byte) (int, error) {
	if w.short {
		return 0, nil
	}
	return 0, w.err
}

// mgmtVisitor records entries, optional replacements and stops.
type mgmtVisitor struct {
	enter    func(n Node) (Node, bool)
	leave    func(n Node) (Node, bool)
	entered  []Node
	replaced bool
}

func (v *mgmtVisitor) Enter(n Node) (Node, bool) {
	v.entered = append(v.entered, n)
	if v.enter != nil {
		return v.enter(n)
	}
	return n, false
}

func (v *mgmtVisitor) Leave(n Node) (Node, bool) {
	// A typed-nil Enter output is a replacement failure (same-class rule).
	if n == nil {
		return n, false
	}
	if v.leave != nil {
		out, ok := v.leave(n)
		if out != n {
			v.replaced = true
		}
		return out, ok
	}
	return n, true
}

func grantUser() *UserSpec {
	return &UserSpec{User: &auth.UserIdentity{Username: "MOCK_USER", Hostname: "MOCK_HOST"}}
}

func selectPriv() *PrivElem {
	return &PrivElem{Priv: mysql.SelectPriv}
}

// --- tablespace / logfile group (59.5) ---

func TestMgmtTablespaceNamesAndEngine(t *testing.T) {
	t.Parallel()
	create := &CreateTablespaceStmt{
		Name:    lex.Ident{Name: "ts1"},
		Options: []*TablespaceOption{{Tp: TablespaceOptionEngine, Engine: lex.Ident{Name: "INNODB"}}},
	}
	// The default flags re-quote engine names; the engine stays a name, not a string payload.
	if got := mgmtRestore(t, create); got != "CREATE TABLESPACE `ts1` ENGINE = `INNODB`" {
		t.Fatalf("create restore = %q", got)
	}
	// Undo variant and a string option keep paths and comments as strings.
	undo := &CreateTablespaceStmt{
		Undo: true,
		Name: lex.Ident{Name: "uts"},
		Options: []*TablespaceOption{
			{Tp: TablespaceOptionDataFile, Str: "path/file.ibd"},
			{Tp: TablespaceOptionComment, Str: "cmt"},
		},
	}
	if got := mgmtRestore(t, undo); got != "CREATE UNDO TABLESPACE `uts` ADD DATAFILE 'path/file.ibd' COMMENT = 'cmt'" {
		t.Fatalf("undo create restore = %q", got)
	}
	alter := &AlterTablespaceStmt{Name: lex.Ident{Name: "a"}, RenameTo: lex.Ident{Name: "b"}}
	if got := mgmtRestore(t, alter); got != "ALTER TABLESPACE `a` RENAME TO `b`" {
		t.Fatalf("alter restore = %q", got)
	}
	// Quoted-empty object and rename names remain present.
	empty := &AlterTablespaceStmt{Name: lex.Ident{Raw: "``", Quote: lex.QuoteBacktick}, RenameTo: lex.Ident{Raw: "``", Quote: lex.QuoteBacktick}}
	if got := mgmtRestore(t, empty); got != "ALTER TABLESPACE `` RENAME TO ``" {
		t.Fatalf("quoted-empty restore = %q", got)
	}
	_ = mgmtRestoreErr(t, &AlterTablespaceStmt{})
	drop := &DropTablespaceStmt{
		Name:   lex.Ident{Name: "d"},
		Option: &TablespaceOption{Tp: TablespaceOptionWait},
	}
	if got := mgmtRestore(t, drop); got != "DROP TABLESPACE `d` WAIT" {
		t.Fatalf("drop restore = %q", got)
	}
	// Engine is a name, not a string payload.
	eng := &TablespaceOption{Tp: TablespaceOptionEngine, Str: "leak"}
	_ = mgmtRestoreErr(t, eng)
	_ = mgmtRestoreErr(t, &TablespaceOption{Tp: TablespaceOptionDataFile, Engine: lex.Ident{Name: "x"}})
	// Engine with no explicit name is rejected; a present empty name is
	// allowed but must still be written via WriteIdent.
	_ = mgmtRestoreErr(t, &TablespaceOption{Tp: TablespaceOptionEngine, Engine: lex.Ident{Span: loc.InvalidSpan()}})
}

func TestMgmtTablespaceVisitor(t *testing.T) {
	t.Parallel()
	// DropTablespace visits its Option; replacement role is enforced.
	drop := &DropTablespaceStmt{Name: lex.Ident{Name: "d"}}
	replacement := &TablespaceOption{Tp: TablespaceOptionNoWait}
	v := &mgmtVisitor{enter: func(n Node) (Node, bool) {
		if _, isOpt := n.(*TablespaceOption); isOpt {
			return replacement, false
		}
		return n, false
	}}
	if _, ok := drop.Accept(v); !ok {
		t.Fatal("drop without option accepted=false, want true")
	}
	withOpt := &DropTablespaceStmt{Name: lex.Ident{Name: "d"}, Option: &TablespaceOption{Tp: TablespaceOptionWait}}
	if _, ok := withOpt.Accept(&mgmtVisitor{}); !ok {
		t.Fatal("drop with option accepted=false, want true")
	}
	if _, ok := withOpt.Accept(&mgmtVisitor{enter: func(n Node) (Node, bool) {
		if _, isOpt := n.(*TablespaceOption); isOpt {
			return &OptimizerHint{}, false // wrong role
		}
		return n, false
	}}); ok {
		t.Fatal("wrong-role option replacement accepted, want rejection")
	}
	// Stop at the option and skip children of create.
	if _, ok := withOpt.Accept(&mgmtVisitor{leave: func(n Node) (Node, bool) {
		if _, isOpt := n.(*TablespaceOption); isOpt {
			return n, false
		}
		return n, true
	}}); ok {
		t.Fatal("stop at option accepted, want rejection")
	}
	create := &CreateTablespaceStmt{Name: lex.Ident{Name: "ts"}, Options: []*TablespaceOption{{Tp: TablespaceOptionWait}}}
	skipV := &mgmtVisitor{enter: func(n Node) (Node, bool) { return n, true }}
	if got, ok := create.Accept(skipV); !ok || got != create || len(skipV.entered) != 1 {
		t.Fatalf("skip: %v, %v, %d; want identity, true, 1", got, ok, len(skipV.entered))
	}
	// nil list entries are rejected.
	bad := &CreateTablespaceStmt{Name: lex.Ident{Name: "ts"}, Options: []*TablespaceOption{nil}}
	if _, ok := bad.Accept(&mgmtVisitor{}); ok {
		t.Fatal("nil option entry accepted, want rejection")
	}
	// Nested error chain: a pending ctx error surfaces with its child annotation.
	chain := &CreateTablespaceStmt{Name: lex.Ident{Name: "ts"}, Options: []*TablespaceOption{{Tp: TablespaceOptionWait}}}
	seedErr := errors.New("seed")
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &mgmtMangledWriter{err: seedErr})
	err := chain.Restore(ctx)
	if !errors.Is(err, seedErr) || !strings.Contains(err.Error(), "splicing CreateTablespaceStmt") {
		t.Fatalf("nested error = %v, want annotated seed error", err)
	}
	// Short writes and writer errors are reported.
	_ = ctx
	{
		w := &mgmtMangledWriter{short: true}
		if err := create.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, w)); !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("short write = %v, want io.ErrShortWrite", err)
		}
	}
}

func TestMgmtLogfileGroupNamesAndEngine(t *testing.T) {
	t.Parallel()
	create := &CreateLogfileGroupStmt{
		Name:     lex.Ident{Name: "lg"},
		UndoFile: "undo/1.log",
		Options:  []*LogfileGroupOption{{Tp: LogfileGroupOptionEngine, Engine: lex.Ident{Name: "InnodbEngine"}}},
	}
	if got := mgmtRestore(t, create); got != "CREATE LOGFILE GROUP `lg` ADD UNDOFILE 'undo/1.log' ENGINE = `InnodbEngine`" {
		t.Fatalf("create restore = %q", got)
	}
	alter := &AlterLogfileGroupStmt{Name: lex.Ident{Name: "lga"}, UndoFile: "u.log"}
	if got := mgmtRestore(t, alter); !strings.Contains(got, "ALTER LOGFILE GROUP `lga` ADD UNDOFILE 'u.log'") {
		t.Fatalf("alter restore = %q", got)
	}
	drop := &DropLogfileGroupStmt{Name: lex.Ident{Name: "lgd"}, Options: []*LogfileGroupOption{{Tp: LogfileGroupOptionWait}}}
	if got := mgmtRestore(t, drop); got != "DROP LOGFILE GROUP `lgd` WAIT" {
		t.Fatalf("drop restore = %q", got)
	}
	// Engine payload is exclusive for both families.
	_ = mgmtRestoreErr(t, &LogfileGroupOption{Tp: LogfileGroupOptionEngine, Str: "leak"})
	_ = mgmtRestoreErr(t, &LogfileGroupOption{Tp: LogfileGroupOptionWait, Engine: lex.Ident{Name: "x"}})
	// Nested error chain and short writes.
	stmt := &CreateLogfileGroupStmt{Name: lex.Ident{Name: "lg"}, Options: []*LogfileGroupOption{{Tp: LogfileGroupOptionWait}}}
	seedErr := errors.New("seed")
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &mgmtMangledWriter{err: seedErr})
	if err := stmt.Restore(ctx); !errors.Is(err, seedErr) || !strings.Contains(err.Error(), "splicing LogfileGroupOptions") {
		t.Fatalf("nested error = %v, want annotated seed error", err)
	}
	if err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &mgmtMangledWriter{short: true})); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short write = %v, want io.ErrShortWrite", err)
	}
	// Visitor: skip at the top of the options list; wrong-role replacement.
	skipV := &mgmtVisitor{enter: func(n Node) (Node, bool) { return n, true }}
	if got, ok := create.Accept(skipV); !ok || got != create || len(skipV.entered) != 1 {
		t.Fatalf("skip: %v, %v, %d; want identity, true, 1", got, ok, len(skipV.entered))
	}
	if _, ok := create.Accept(&mgmtVisitor{enter: func(n Node) (Node, bool) {
		if _, isOpt := n.(*LogfileGroupOption); isOpt {
			return &OptimizerHint{}, false
		}
		return n, false
	}}); ok {
		t.Fatal("wrong-role option replacement accepted, want rejection")
	}
}

// --- resource group / server / plugin (59.6) ---

func TestMgmtResourceGroupTypePreserved(t *testing.T) {
	t.Parallel()
	// Unknown type labels keep their original spelling.
	unknown := &CreateResourceGroupStmt{Name: lex.Ident{Name: "rg"}, Type: lex.Ident{Name: "custom"}}
	if got := mgmtRestore(t, unknown); got != "CREATE RESOURCE GROUP `rg` TYPE = `custom`" {
		t.Fatalf("unknown type restore = %q", got)
	}
	// Quoted type labels keep name syntax.
	quoted := &CreateResourceGroupStmt{Name: lex.Ident{Name: "rg"}, Type: lex.Ident{Name: "custom", Raw: "`custom`", Quote: lex.QuoteBacktick}}
	if got := mgmtRestore(t, quoted); got != "CREATE RESOURCE GROUP `rg` TYPE = `custom`" {
		t.Fatalf("quoted type restore = %q", got)
	}
	// USER/SYSTEM stay keywords, case-insensitively.
	for _, kw := range []string{"system", "System", "USER"} {
		stmt := &CreateResourceGroupStmt{Name: lex.Ident{Name: "rg"}, Type: lex.Ident{Name: kw}}
		if got := mgmtRestore(t, stmt); !strings.Contains(got, "TYPE = "+strings.ToUpper(kw)) {
			t.Fatalf("%s restore = %q", kw, got)
		}
	}
	// Missing type is rejected.
	_ = mgmtRestoreErr(t, &CreateResourceGroupStmt{Name: lex.Ident{Name: "rg"}})
	alter := &AlterResourceGroupStmt{Name: lex.Ident{Name: "rg"}}
	if got := mgmtRestore(t, alter); got != "ALTER RESOURCE GROUP `rg`" {
		t.Fatalf("alter restore = %q", got)
	}
	drop := &DropResourceGroupStmt{Name: lex.Ident{Name: "rg"}, Force: true}
	if got := mgmtRestore(t, drop); got != "DROP RESOURCE GROUP `rg` FORCE" {
		t.Fatalf("drop restore = %q", got)
	}
	// Visitor: wrong-role VCPU replacement rejected; skip covers only the top node.
	for _, stmt := range []Node{
		&CreateResourceGroupStmt{Name: lex.Ident{Name: "rg"}, Type: lex.Ident{Name: "system"}, VCPU: []*ResourceGroupVCPU{{From: 0, To: 1}}},
		&AlterResourceGroupStmt{Name: lex.Ident{Name: "rg"}, VCPU: []*ResourceGroupVCPU{{From: 2}}},
	} {
		if _, ok := stmt.Accept(&mgmtVisitor{enter: func(n Node) (Node, bool) {
			if _, isVCPU := n.(*ResourceGroupVCPU); isVCPU {
				return &OptimizerHint{}, false
			}
			return n, false
		}}); ok {
			t.Fatalf("%T: wrong-role VCPU replacement accepted, want rejection", stmt)
		}
	}
	if _, ok := drop.Accept(&mgmtVisitor{}); !ok {
		t.Fatal("drop visitor accepted=false, want true")
	}
}

func TestMgmtServerOptionsAndLabels(t *testing.T) {
	t.Parallel()
	// Bare DATABASE restores as a keyword; quoted labels keep name syntax.
	create := &CreateServerStmt{
		Name:    lex.Ident{Name: "srv"},
		Wrapper: "mysql",
		Options: []*ServerOption{
			{Name: lex.Ident{Name: "DATABASE"}, Value: "db1"},
			{Name: lex.Ident{Name: "DATABASE", Raw: "`DATABASE`", Quote: lex.QuoteBacktick}, Value: "db2"},
		},
	}
	got := mgmtRestore(t, create)
	want := "CREATE SERVER `srv` FOREIGN DATA WRAPPER 'mysql' OPTIONS(DATABASE 'db1', `DATABASE` 'db2')"
	if got != want {
		t.Fatalf("create server restore = %q, want %q", got, want)
	}
	alter := &AlterServerStmt{Name: lex.Ident{Name: "srv"}, Options: []*ServerOption{{Name: lex.Ident{Name: "HOST"}, Value: "h"}}}
	if got := mgmtRestore(t, alter); got != "ALTER SERVER `srv` OPTIONS(`HOST` 'h')" {
		t.Fatalf("alter server restore = %q", got)
	}
	drop := &DropServerStmt{IfExists: true, Name: lex.Ident{Name: "srv"}}
	if got := mgmtRestore(t, drop); got != "DROP SERVER IF EXISTS `srv`" {
		t.Fatalf("drop server restore = %q", got)
	}
	// Missing name / missing option label rejected; value stays a string.
	_ = mgmtRestoreErr(t, &CreateServerStmt{Wrapper: "w"})
	_ = mgmtRestoreErr(t, &ServerOption{})
	// Visitor: skip at top, stop and wrong-role option rejection.
	skipV := &mgmtVisitor{enter: func(n Node) (Node, bool) { return n, true }}
	if got, ok := create.Accept(skipV); !ok || got != create || len(skipV.entered) != 1 {
		t.Fatalf("skip: %v, %v, %d; want identity, true, 1", got, ok, len(skipV.entered))
	}
	if _, ok := create.Accept(&mgmtVisitor{leave: func(n Node) (Node, bool) {
		if _, isOpt := n.(*ServerOption); isOpt {
			return n, false
		}
		return n, true
	}}); ok {
		t.Fatal("stop at server option accepted, want rejection")
	}
	if _, ok := create.Accept(&mgmtVisitor{enter: func(n Node) (Node, bool) {
		if _, isOpt := n.(*ServerOption); isOpt {
			return &OptimizerHint{}, false
		}
		return n, false
	}}); ok {
		t.Fatal("wrong-role server option replacement accepted, want rejection")
	}
	if _, ok := create.Accept(&mgmtVisitor{enter: func(n Node) (Node, bool) {
		if _, isOpt := n.(*ServerOption); isOpt {
			return nil, false // typed nil: not a *ServerOption
		}
		return n, false
	}}); ok {
		t.Fatal("typed-nil server option replacement accepted, want rejection")
	}
}

func TestMgmtPluginNames(t *testing.T) {
	t.Parallel()
	// Quoted-empty plugin names stay present; SONAME remains a string.
	install := &InstallPluginStmt{Name: lex.Ident{Raw: "``", Quote: lex.QuoteBacktick}, Soname: "p.so"}
	if got := mgmtRestore(t, install); got != "INSTALL PLUGIN `` SONAME 'p.so'" {
		t.Fatalf("install restore = %q", got)
	}
	uninstall := &UninstallPluginStmt{Name: lex.Ident{Name: "p"}}
	if got := mgmtRestore(t, uninstall); got != "UNINSTALL PLUGIN `p`" {
		t.Fatalf("uninstall restore = %q", got)
	}
	_ = mgmtRestoreErr(t, &InstallPluginStmt{Soname: "p.so"})
	_ = mgmtRestoreErr(t, &UninstallPluginStmt{})
	// Visitor: skip covers only the top node; missing name is rejected in Restore,
	// Accept still visits both plugin statements.
	skipV := &mgmtVisitor{enter: func(n Node) (Node, bool) { return n, true }}
	if got, ok := install.Accept(skipV); !ok || got != install || len(skipV.entered) != 1 {
		t.Fatalf("skip: %v, %v, %d; want identity, true, 1", got, ok, len(skipV.entered))
	}
	if _, ok := uninstall.Accept(&mgmtVisitor{}); !ok {
		t.Fatal("uninstall visitor accepted=false, want true")
	}
}

// --- GRANT/REVOKE GrantObject (59.2) ---

func grantObject(parts ...GrantObjectPart) *GrantObject {
	obj := &GrantObject{Parts: parts}
	obj.SetSpan(loc.InvalidSpan())
	return obj
}

func grantNamedPart(name string) GrantObjectPart {
	return GrantObjectPart{Name: lex.Ident{Name: name}}
}

func grantStarPart() GrantObjectPart {
	return GrantObjectPart{Wildcard: true, WildcardSpan: loc.InvalidSpan()}
}

func TestMgmtGrantObjectShapes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		parts []GrantObjectPart
		want  string
	}{
		{"single star", []GrantObjectPart{grantStarPart()}, "*"},
		{"two stars", []GrantObjectPart{grantStarPart(), grantStarPart()}, "*.*"},
		{"named with star", []GrantObjectPart{grantNamedPart("db"), grantStarPart()}, "`db`.*"},
		{"named pair", []GrantObjectPart{grantNamedPart("db"), grantNamedPart("tbl")}, "`db`.`tbl`"},
		{"single name", []GrantObjectPart{grantNamedPart("priv")}, "`priv`"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stmt := &GrantStmt{Privs: []*PrivElem{selectPriv()}, Object: grantObject(tc.parts...), Users: []*UserSpec{grantUser()}, ObjectType: ObjectTypeNone}
			if got, want := mgmtRestore(t, stmt), "GRANT SELECT ON "+tc.want+" TO `MOCK_USER`@`MOCK_HOST`"; got != want {
				t.Fatalf("grant restore = %q, want %q", got, want)
			}
			revoke := &RevokeStmt{Privs: []*PrivElem{selectPriv()}, Object: stmt.Object, Users: []*UserSpec{grantUser()}, ObjectType: ObjectTypeNone}
			if got := mgmtRestore(t, revoke); got != "REVOKE SELECT ON "+tc.want+" FROM `MOCK_USER`@`MOCK_HOST`" {
				t.Fatalf("revoke restore = %q", got)
			}
		})
	}
	// A quoted literal star and a quoted dot are names, not wildcards.
	quoted := grantObject(
		GrantObjectPart{Name: lex.Ident{Name: "*", Raw: "`*`", Quote: lex.QuoteBacktick}},
		GrantObjectPart{Wildcard: true, WildcardSpan: loc.InvalidSpan()},
	)
	quotedGrant := &GrantStmt{ObjectType: ObjectTypeNone, Object: quoted}
	if got := mgmtRestore(t, quotedGrant); !strings.Contains(got, "ON `*`.*") {
		t.Fatalf("quoted star restore = %q", got)
	}
	empty := grantObject(
		GrantObjectPart{Name: lex.Ident{Raw: "``", Quote: lex.QuoteBacktick}},
		GrantObjectPart{Wildcard: true, WildcardSpan: loc.Span{Start: 10, End: 11}},
	)
	if got := mgmtRestore(t, empty); got != "``.*" {
		t.Fatalf("quoted-empty star restore = %q", got)
	}
	// A quoted literal dot is a name, not a wildcard.
	quotedDot := grantObject(
		GrantObjectPart{Name: lex.Ident{Name: ".", Raw: "`.`", Quote: lex.QuoteBacktick}},
		grantStarPart(),
	)
	quotedDotGrant := &GrantStmt{Privs: []*PrivElem{selectPriv()}, ObjectType: ObjectTypeNone, Object: quotedDot, Users: []*UserSpec{grantUser()}}
	if got := mgmtRestore(t, quotedDotGrant); !strings.Contains(got, "ON `.`.*") {
		t.Fatalf("quoted dot restore = %q", got)
	}
}

func TestMgmtGrantObjectInvalidCombos(t *testing.T) {
	t.Parallel()
	// Both name and wildcard set, neither set, and a named part after a
	// wildcard database are all rejected; three parts are rejected too.
	bad := []*GrantObject{
		grantObject(GrantObjectPart{Name: lex.Ident{Name: "n"}, Wildcard: true}),
		grantObject(GrantObjectPart{WildcardSpan: loc.InvalidSpan()}),
		grantObject(grantStarPart(), grantNamedPart("t")),
		grantObject(grantStarPart(), grantStarPart(), grantStarPart()),
	}
	for _, obj := range bad {
		if err := mgmtRestoreErr(t, &GrantStmt{Privs: []*PrivElem{selectPriv()}, Object: obj, Users: []*UserSpec{grantUser()}}); err == nil {
			t.Fatalf("invalid grant object accepted: %+v", obj)
		}
	}
	// The parent rejects a missing object and a wrong-role replacement.
	_ = mgmtRestoreErr(t, &GrantStmt{Privs: []*PrivElem{selectPriv()}, Users: []*UserSpec{grantUser()}})
	valid := &GrantStmt{Privs: []*PrivElem{selectPriv()}, Object: grantObject(grantNamedPart("db")), Users: []*UserSpec{grantUser()}}
	if _, ok := valid.Accept(&mgmtVisitor{enter: func(n Node) (Node, bool) {
		if _, isObj := n.(*GrantObject); isObj {
			return &OptimizerHint{}, false
		}
		return n, false
	}}); ok {
		t.Fatal("wrong-role object replacement accepted, want rejection")
	}
	// Stop and skip at the object level are honored.
	if _, ok := valid.Accept(&mgmtVisitor{leave: func(n Node) (Node, bool) {
		if _, isObj := n.(*GrantObject); isObj {
			return n, false
		}
		return n, true
	}}); ok {
		t.Fatal("stop at object accepted, want rejection")
	}
	skipV := &mgmtVisitor{enter: func(n Node) (Node, bool) {
		if _, isObj := n.(*GrantObject); isObj {
			return n, true
		}
		return n, false
	}}
	if got, ok := valid.Accept(skipV); !ok || got != valid {
		t.Fatalf("skip at object: %v, %v; want identity, true", got, ok)
	}
	// Wildcard spans survive a replacement pass (stable reuse).
	reused := grantObject(GrantObjectPart{Wildcard: true, WildcardSpan: loc.Span{Start: 3, End: 4}})
	_, _ = reused.Accept(&mgmtVisitor{})
	if reused.Parts[0].WildcardSpan != (loc.Span{Start: 3, End: 4}) {
		t.Fatalf("wildcard span changed: %+v", reused.Parts[0].WildcardSpan)
	}
}

// --- Optimizer hints (59.7) ---

func TestMgmtOptimizerHintRestore(t *testing.T) {
	t.Parallel()
	// Zero is a valid written value; keyword enum output, not a name.
	for _, val := range []uint64{0, 1 << 60} {
		hint := &OptimizerHint{Kind: OptimizerHintMaxExecutionTime, MaxExecutionTime: val}
		if got := mgmtRestore(t, hint); got != "MAX_EXECUTION_TIME("+itoa(val)+")" {
			t.Fatalf("hint %d restore = %q", val, got)
		}
	}
	// Unknown kinds and nil hints are rejected.
	_ = mgmtRestoreErr(t, &OptimizerHint{Kind: OptimizerHintUnknown, MaxExecutionTime: 7})
	_ = mgmtRestoreErr(t, &OptimizerHint{})
	var nilHint *OptimizerHint
	if _, ok := nilHint.Accept(&mgmtVisitor{}); ok {
		t.Fatal("nil hint Accept accepted, want rejection")
	}
	// Visitor: skip, stop, wrong-role replacement and nil list entries.
	valid := &OptimizerHint{Kind: OptimizerHintMaxExecutionTime}
	skipV := &mgmtVisitor{enter: func(n Node) (Node, bool) { return n, true }}
	if got, ok := valid.Accept(skipV); !ok || got != valid || len(skipV.entered) != 1 {
		t.Fatalf("skip: %v, %v, %d; want identity, true, 1", got, ok, len(skipV.entered))
	}
	if _, ok := valid.Accept(&mgmtVisitor{leave: func(n Node) (Node, bool) {
		if _, isHint := n.(*OptimizerHint); isHint {
			return n, false
		}
		return n, true
	}}); ok {
		t.Fatal("stop at hint accepted, want rejection")
	}
	if _, ok := valid.Accept(&mgmtVisitor{enter: func(n Node) (Node, bool) {
		if _, isHint := n.(*OptimizerHint); isHint {
			return &PrivElem{}, false
		}
		return n, false
	}}); ok {
		t.Fatal("wrong-role hint replacement accepted, want rejection")
	}
	// Writer failure surfaces through the select-statement restore path, which
	// annotates the child hint error.
	seedErr := errors.New("seed")
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &mgmtMangledWriter{err: seedErr})
	sel := &SelectStmt{
		SelectStmtOpts: &SelectStmtOpts{TableHints: []*OptimizerHint{valid}},
	}
	if err := sel.Restore(ctx); !errors.Is(err, seedErr) || !strings.Contains(err.Error(), "restore SelectStmt.TableHints") {
		t.Fatalf("select nested hint error = %v, want annotated seed error", err)
	}
	if err := sel.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &mgmtMangledWriter{short: true})); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short write = %v, want io.ErrShortWrite", err)
	}
}

func TestMgmtOptimizerHintsInDML(t *testing.T) {
	t.Parallel()
	hints := []*OptimizerHint{
		{Kind: OptimizerHintMaxExecutionTime, MaxExecutionTime: 0},
		{Kind: OptimizerHintMaxExecutionTime, MaxExecutionTime: 500},
	}
	// UPDATE: table role is required; hints are walked and replaced in place.
	update := &UpdateStmt{BaseUpdateStmt: pkast.BaseUpdateStmt{
		Table:       &TableName{},
		Assignments: []*Assignment{{Target: &PositionExpr{N: 1}, Expr: &PositionExpr{N: 2}}},
	}, TableHints: hints}
	if _, ok := update.Accept(&mgmtVisitor{enter: func(n Node) (Node, bool) {
		if _, isHint := n.(*OptimizerHint); isHint {
			return &PrivElem{}, false
		}
		return n, false
	}}); ok {
		t.Fatal("update wrong-role hint accepted, want rejection")
	}
	if _, ok := update.Accept(&mgmtVisitor{enter: func(n Node) (Node, bool) {
		if _, isHint := n.(*OptimizerHint); isHint {
			return &OptimizerHint{Kind: OptimizerHintMaxExecutionTime, MaxExecutionTime: 1}, false
		}
		return n, false
	}}); !ok {
		t.Fatal("update hint replacement rejected, want acceptance")
	}
	// DELETE: nil hint entries in the list are rejected.
	del := &DeleteStmt{TableHints: []*OptimizerHint{nil}}
	if _, ok := del.Accept(&mgmtVisitor{}); ok {
		t.Fatal("delete nil hint entry accepted, want rejection")
	}
	// SELECT: hints are only visited when the options block is present.
	sel := &SelectStmt{SelectStmtOpts: &SelectStmtOpts{TableHints: []*OptimizerHint{hints[0]}}}
	if _, ok := sel.Accept(&mgmtVisitor{enter: func(n Node) (Node, bool) {
		if _, isHint := n.(*OptimizerHint); isHint {
			return &PrivElem{}, false
		}
		return n, false
	}}); ok {
		t.Fatal("select wrong-role hint accepted, want rejection")
	}
	selPlain := &SelectStmt{SelectStmtOpts: &SelectStmtOpts{}}
	if _, ok := selPlain.Accept(&mgmtVisitor{}); !ok {
		t.Fatal("select without hints rejected, want acceptance")
	}
	// Writer failure surfaces through the update restore path.
	seedErr := errors.New("seed")
	upd := &UpdateStmt{BaseUpdateStmt: pkast.BaseUpdateStmt{
		Table:       &TableName{},
		Assignments: []*Assignment{{Target: &PositionExpr{N: 1}, Expr: &PositionExpr{N: 2}}},
	}, TableHints: []*OptimizerHint{{Kind: OptimizerHintMaxExecutionTime}}}
	if err := upd.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &mgmtMangledWriter{err: seedErr})); !errors.Is(err, seedErr) {
		t.Fatalf("update writer error = %v, want seed error", err)
	}
}

// --- ANSI_QUOTES-style name quoting (59.5/59.6 acceptance) ---

func TestMgmtNameQuoteStyles(t *testing.T) {
	t.Parallel()
	// Double-quote restore style re-quotes names with the dialect's name quote.
	ts := &CreateTablespaceStmt{Name: lex.Ident{Name: "T§", Raw: `"T§"`}, Options: []*TablespaceOption{}}
	if got := mgmtRestoreWithFlags(t, ts, format.DefaultRestoreFlags|format.RestoreNameDoubleQuotes); got != `CREATE TABLESPACE "T§"` {
		t.Fatalf("double-quoted restore = %q", got)
	}
	// Server names with UTF-8 spellings round-trip as quoted identifiers.
	srv := &CreateServerStmt{Name: lex.Ident{Name: "sрв"}, Options: []*ServerOption{}}
	if got := mgmtRestore(t, srv); got != "CREATE SERVER `sрв` FOREIGN DATA WRAPPER ''" {
		t.Fatalf("utf8 server name restore = %q", got)
	}
	// A dotted engine name stays a single identifier payload.
	eng := &TablespaceOption{Tp: TablespaceOptionEngine, Engine: lex.Ident{Name: "innodb.plugin"}}
	if got := mgmtRestore(t, eng); got != "ENGINE = `innodb.plugin`" {
		t.Fatalf("dotted engine restore = %q", got)
	}
}

func TestMgmtRejectHiddenOptionPayload(t *testing.T) {
	for _, tc := range []struct {
		name string
		node Node
	}{
		{"tablespace wait string", &TablespaceOption{Tp: TablespaceOptionWait, Str: "hidden"}},
		{"tablespace comment number", &TablespaceOption{Tp: TablespaceOptionComment, Val: 1}},
		{"tablespace size string", &TablespaceOption{Tp: TablespaceOptionInitialSize, Str: "hidden"}},
		{"logfile wait number", &LogfileGroupOption{Tp: LogfileGroupOptionWait, Val: 1}},
		{"logfile comment number", &LogfileGroupOption{Tp: LogfileGroupOptionComment, Val: 1}},
		{"logfile size string", &LogfileGroupOption{Tp: LogfileGroupOptionInitialSize, Str: "hidden"}},
		{"resource unknown state", &CreateResourceGroupStmt{Name: lex.Ident{Name: "r"}, Type: lex.Ident{Name: "custom"}, State: 99}},
		{"alter resource unknown state", &AlterResourceGroupStmt{Name: lex.Ident{Name: "r"}, State: -1}},
		{"grant nil user", &GrantStmt{ObjectType: ObjectTypeNone, Object: grantObject(grantStarPart()), Privs: []*PrivElem{selectPriv()}, Users: []*UserSpec{nil}}},
	} {
		t.Run(tc.name, func(t *testing.T) { _ = mgmtRestoreErr(t, tc.node) })
	}
}

// itoa keeps the hint-value check dependency-free.
func itoa(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

package ast_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

// mgmtBoundaryTable returns a TableName whose name restores as `tbl`.
func mgmtBoundaryTable() *ast.TableName {
	return &ast.TableName{
		BaseTableRef: pkast.BaseTableRef{
			Name: ast.QualifiedName{Parts: []lex.Ident{{Name: "tbl"}}},
		},
	}
}

// mgmtBoundaryRestoreErr restores n, expecting an error.
func mgmtBoundaryRestoreErr(t *testing.T, n ast.Node) error {
	t.Helper()
	var out strings.Builder
	err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out))
	if err == nil {
		t.Fatalf("restore %T: expected error, got %q", n, out.String())
	}
	return err
}

// mgmtBoundaryRestoreOK restores n, expecting success.
func mgmtBoundaryRestoreOK(t *testing.T, n ast.Node) string {
	t.Helper()
	var out strings.Builder
	if err := n.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
		t.Fatalf("restore %T: %v", n, err)
	}
	return out.String()
}

// mgmtBoundaryVisitor drives Accept with per-node Enter/Leave hooks.
type mgmtBoundaryVisitor struct {
	enter func(n ast.Node) (ast.Node, bool)
	leave func(n ast.Node) (ast.Node, bool)
}

func (v *mgmtBoundaryVisitor) Enter(n ast.Node) (ast.Node, bool) {
	if v.enter != nil {
		return v.enter(n)
	}
	return n, false
}

func (v *mgmtBoundaryVisitor) Leave(n ast.Node) (ast.Node, bool) {
	if v.leave != nil {
		return v.leave(n)
	}
	return n, true
}

// TestManagementBoundary exercises the p59 2/5/6/8 contract against the
// HANDLER / CACHE / LOAD / plugin sections.
func TestManagementBoundary(t *testing.T) {
	// --- Restore: ctx.Err checked first, then nil self and required roles.
	for _, tc := range []struct {
		name string
		node ast.Node
	}{
		{"handler nil self", (*ast.HandlerStmt)(nil)},
		{"cache entry nil self", (*ast.CacheIndexTable)(nil)},
		{"cache target nil self", (*ast.CacheIndexTarget)(nil)},
		{"cache stmt nil self", (*ast.CacheIndexStmt)(nil)},
		{"load stmt nil self", (*ast.LoadIndexStmt)(nil)},
		{"load entry nil self", (*ast.LoadIndexTable)(nil)},
		{"plugin nil self", (*ast.InstallPluginStmt)(nil)},
		{"unplugin nil self", (*ast.UninstallPluginStmt)(nil)},
	} {
		t.Run("restore/"+tc.name, func(t *testing.T) {
			_ = mgmtBoundaryRestoreErr(t, tc.node)
			if _, ok := tc.node.Accept(&mgmtBoundaryVisitor{}); ok {
				t.Fatal("nil receiver accepted")
			}
			pending := errors.New("pending writer failure")
			ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &mgmtMangledWriter{err: pending})
			ctx.WritePlain("seed")
			if err := tc.node.Restore(ctx); !errors.Is(err, pending) {
				t.Fatalf("pending error lost: %v", err)
			}
		})
	}

	// --- Restore: required-role and combination rejections.
	handlerCases := []struct {
		name string
		node *ast.HandlerStmt
	}{
		{"open limit forbidden", &ast.HandlerStmt{Tp: ast.HandlerOpen, Table: mgmtBoundaryTable(), Limit: &ast.Limit{BaseLimit: pkast.BaseLimit{Count: ast.NewParamMarkerExpr(0)}}}},
		{"open where forbidden", &ast.HandlerStmt{Tp: ast.HandlerOpen, Table: mgmtBoundaryTable(), Where: ast.NewParamMarkerExpr(0)}},
		{"close limit forbidden", &ast.HandlerStmt{Tp: ast.HandlerClose, Table: mgmtBoundaryTable(), Limit: &ast.Limit{BaseLimit: pkast.BaseLimit{Count: ast.NewParamMarkerExpr(0)}}}},
		{"read movement unknown direction", &ast.HandlerStmt{Tp: ast.HandlerRead, Table: mgmtBoundaryTable(), Direction: ast.HandlerReadUnknown}},
		{"read pk prev forbidden", &ast.HandlerStmt{Tp: ast.HandlerRead, Table: mgmtBoundaryTable(), Direction: ast.HandlerReadPrev}},
		{"read pk last forbidden", &ast.HandlerStmt{Tp: ast.HandlerRead, Table: mgmtBoundaryTable(), Direction: ast.HandlerReadLast}},
		{"movement values forbidden", &ast.HandlerStmt{Tp: ast.HandlerRead, Table: mgmtBoundaryTable(), Direction: ast.HandlerReadFirst, Values: []ast.ExprNode{ast.NewParamMarkerExpr(0)}}},
		{"typed nil value", &ast.HandlerStmt{Tp: ast.HandlerRead, Table: mgmtBoundaryTable(), Index: lex.Ident{Name: "ix"}, Op: pkast.OperatorEQ, Values: []ast.ExprNode{(*ast.LiteralExpr)(nil)}}},
		{"typed nil where", &ast.HandlerStmt{Tp: ast.HandlerRead, Table: mgmtBoundaryTable(), Direction: ast.HandlerReadFirst, Where: (*ast.LiteralExpr)(nil)}},
	}
	for _, tc := range handlerCases {
		if tc.node == nil {
			continue
		}
		t.Run("restore-handler/"+tc.name, func(t *testing.T) {
			_ = mgmtBoundaryRestoreErr(t, tc.node)
			if strings.HasPrefix(tc.name, "typed nil") {
				if _, ok := tc.node.Accept(&mgmtBoundaryVisitor{}); ok {
					t.Fatal("typed-nil child accepted")
				}
			}
		})
	}

	// Positive handler restores: movement with index (all four directions),
	// comparison with WHERE, and the no-Index FIRST regression shape.
	for _, dir := range []ast.HandlerReadDirection{ast.HandlerReadFirst, ast.HandlerReadNext, ast.HandlerReadPrev, ast.HandlerReadLast} {
		stmt := &ast.HandlerStmt{Tp: ast.HandlerRead, Table: mgmtBoundaryTable(), Index: lex.Ident{Name: "ix"}, Direction: dir}
		t.Run("restore-handler/index-direction", func(t *testing.T) {
			got := mgmtBoundaryRestoreOK(t, stmt)
			if !strings.Contains(got, "READ") {
				t.Fatalf("restore = %q, want READ", got)
			}
		})
	}
	// Without an index there must still be exactly one space before FIRST.
	{
		stmt := &ast.HandlerStmt{Tp: ast.HandlerRead, Table: mgmtBoundaryTable(), Direction: ast.HandlerReadFirst}
		got := mgmtBoundaryRestoreOK(t, stmt)
		if got != "HANDLER `tbl` READ FIRST" {
			t.Fatalf("no-index FIRST restore = %q, want READ FIRST", got)
		}
	}
	// Comparison allows WHERE (the original grammar permits it).
	{
		stmt := &ast.HandlerStmt{
			Tp: ast.HandlerRead, Table: mgmtBoundaryTable(), Index: lex.Ident{Name: "ix"},
			Op: pkast.OperatorEQ, Values: []ast.ExprNode{ast.NewParamMarkerExpr(0)},
			Where: ast.NewParamMarkerExpr(1),
		}
		got := mgmtBoundaryRestoreOK(t, stmt)
		if !strings.Contains(got, "WHERE") {
			t.Fatalf("comparison with WHERE restore = %q, want WHERE", got)
		}
	}
	// OPEN with an alias restores; OPEN is read-field free.
	{
		stmt := &ast.HandlerStmt{Tp: ast.HandlerOpen, Table: mgmtBoundaryTable(), Alias: lex.Ident{Name: "al"}}
		if got := mgmtBoundaryRestoreOK(t, stmt); !strings.Contains(got, "OPEN") {
			t.Fatalf("open alias restore = %q", got)
		}
	}

	// --- CACHE/LOAD index-name and partition-ALL mutual exclusion.
	for _, tc := range []struct {
		name string
		node ast.Node
	}{
		{"cache entry nil table", &ast.CacheIndexTable{}},
		{"cache entry missing index name", &ast.CacheIndexTable{Table: mgmtBoundaryTable(), IndexNames: []lex.Ident{{Span: lex.Ident{}.Span}}}},
		{"cache entry partition all vs named", &ast.CacheIndexTable{Table: &ast.TableName{BaseTableRef: pkast.BaseTableRef{Name: ast.QualifiedName{Parts: []lex.Ident{{Name: "tbl"}}}}, PartitionNames: []lex.Ident{{Name: "p1"}}}, PartitionAll: true}},
		{"load entry partition all vs named", &ast.LoadIndexTable{Table: &ast.TableName{BaseTableRef: pkast.BaseTableRef{Name: ast.QualifiedName{Parts: []lex.Ident{{Name: "tbl"}}}}, PartitionNames: []lex.Ident{{Name: "p1"}}}, PartitionAll: true}},
		{"cache stmt nil target", &ast.CacheIndexStmt{Tables: []*ast.CacheIndexTable{{Table: mgmtBoundaryTable()}}}},
		{"cache stmt nil entry", &ast.CacheIndexStmt{Tables: []*ast.CacheIndexTable{nil}, Target: &ast.CacheIndexTarget{Kind: ast.CacheIndexDefaultTarget}}},
		{"cache stmt empty tables", &ast.CacheIndexStmt{Target: &ast.CacheIndexTarget{Kind: ast.CacheIndexDefaultTarget}}},
		{"cache unknown target kind", &ast.CacheIndexStmt{Tables: []*ast.CacheIndexTable{{Table: mgmtBoundaryTable()}}, Target: &ast.CacheIndexTarget{Kind: ast.CacheIndexUnknownTarget}}},
		{"cache default target with name", &ast.CacheIndexStmt{Tables: []*ast.CacheIndexTable{{Table: mgmtBoundaryTable()}}, Target: &ast.CacheIndexTarget{Kind: ast.CacheIndexDefaultTarget, Name: lex.Ident{Name: "c"}}}},
		{"load stmt empty tables", &ast.LoadIndexStmt{}},
		{"load stmt nil entry", &ast.LoadIndexStmt{Tables: []*ast.LoadIndexTable{nil}}},
	} {
		t.Run("restore-mgmt/"+tc.name, func(t *testing.T) {
			_ = mgmtBoundaryRestoreErr(t, tc.node)
		})
	}
	// Quoted-empty index name is legal presence and must NOT be rejected.
	{
		entry := &ast.LoadIndexTable{Table: mgmtBoundaryTable(), IndexNames: []lex.Ident{{Raw: "``", Quote: lex.QuoteBacktick}}}
		if got := mgmtBoundaryRestoreOK(t, entry); !strings.Contains(got, "INDEX") {
			t.Fatalf("quoted-empty index restore = %q, want INDEX clause", got)
		}
	}
	// DEFAULT target without a name restores as the DEFAULT keyword.
	{
		target := &ast.CacheIndexTarget{Kind: ast.CacheIndexDefaultTarget}
		if got := mgmtBoundaryRestoreOK(t, target); !strings.Contains(got, "DEFAULT") {
			t.Fatalf("default target restore = %q, want DEFAULT", got)
		}
	}

	// --- Plugin: SONAME stays a string; missing plugin name is rejected.
	{
		stmt := &ast.InstallPluginStmt{Name: lex.Ident{Name: "p"}, Soname: "lib.so"}
		got := mgmtBoundaryRestoreOK(t, stmt)
		if !strings.Contains(got, "SONAME") || !strings.Contains(got, "lib.so") {
			t.Fatalf("plugin restore = %q, want SONAME lib.so", got)
		}
	}
	_ = mgmtBoundaryRestoreErr(t, &ast.InstallPluginStmt{Soname: "lib.so"})
	_ = mgmtBoundaryRestoreErr(t, &ast.UninstallPluginStmt{})

	// --- Accept: same-class check on skip, required children, typed-nil Leave.
	// Each root's typed-nil check targets the node's own concrete type; the
	// typed-nil Leave must be rejected by that node's Accept.
	acceptCases := []struct {
		name        string
		root        ast.Node
		typedNilFor func(n ast.Node) ast.Node
	}{
		{"handler stmt", &ast.HandlerStmt{Tp: ast.HandlerRead, Table: mgmtBoundaryTable(), Index: lex.Ident{Name: "ix"}, Direction: ast.HandlerReadFirst}, func(n ast.Node) ast.Node {
			if _, ok := n.(*ast.HandlerStmt); ok {
				return (*ast.HandlerStmt)(nil)
			}
			return n
		}},
		{"cache entry", &ast.CacheIndexTable{Table: mgmtBoundaryTable()}, func(n ast.Node) ast.Node {
			if _, ok := n.(*ast.CacheIndexTable); ok {
				return (*ast.CacheIndexTable)(nil)
			}
			return n
		}},
		{"cache target", &ast.CacheIndexTarget{Kind: ast.CacheIndexIdentifierTarget, Name: lex.Ident{Name: "c"}}, func(n ast.Node) ast.Node {
			if _, ok := n.(*ast.CacheIndexTarget); ok {
				return (*ast.CacheIndexTarget)(nil)
			}
			return n
		}},
		{"cache stmt", &ast.CacheIndexStmt{Tables: []*ast.CacheIndexTable{{Table: mgmtBoundaryTable()}}, Target: &ast.CacheIndexTarget{Kind: ast.CacheIndexDefaultTarget}}, func(n ast.Node) ast.Node {
			if _, ok := n.(*ast.CacheIndexStmt); ok {
				return (*ast.CacheIndexStmt)(nil)
			}
			return n
		}},
		{"load stmt", &ast.LoadIndexStmt{Tables: []*ast.LoadIndexTable{{Table: mgmtBoundaryTable()}}}, func(n ast.Node) ast.Node {
			if _, ok := n.(*ast.LoadIndexStmt); ok {
				return (*ast.LoadIndexStmt)(nil)
			}
			return n
		}},
		{"load entry", &ast.LoadIndexTable{Table: mgmtBoundaryTable()}, func(n ast.Node) ast.Node {
			if _, ok := n.(*ast.LoadIndexTable); ok {
				return (*ast.LoadIndexTable)(nil)
			}
			return n
		}},
		{"plugin install", &ast.InstallPluginStmt{Name: lex.Ident{Name: "p"}}, func(n ast.Node) ast.Node {
			if _, ok := n.(*ast.InstallPluginStmt); ok {
				return (*ast.InstallPluginStmt)(nil)
			}
			return n
		}},
		{"plugin uninstall", &ast.UninstallPluginStmt{Name: lex.Ident{Name: "p"}}, func(n ast.Node) ast.Node {
			if _, ok := n.(*ast.UninstallPluginStmt); ok {
				return (*ast.UninstallPluginStmt)(nil)
			}
			return n
		}},
	}
	for _, tc := range acceptCases {
		// Same-class Enter with skip=true must still validate the concrete type.
		t.Run("accept-skip-sameclass/"+tc.name, func(t *testing.T) {
			_, ok := tc.root.Accept(&mgmtBoundaryVisitor{
				enter: func(n ast.Node) (ast.Node, bool) {
					if n == tc.root {
						return n, true // skip, but the replacement is the concrete type
					}
					return n, false
				},
			})
			if !ok {
				t.Fatalf("skip with same-class replacement returned ok=false")
			}
		})

		// A typed-nil Leave return of the node's own type must be rejected.
		t.Run("accept-leave-typednil/"+tc.name, func(t *testing.T) {
			_, ok := tc.root.Accept(&mgmtBoundaryVisitor{
				enter: func(n ast.Node) (ast.Node, bool) { return n, false },
				leave: func(n ast.Node) (ast.Node, bool) {
					out := tc.typedNilFor(n)
					// Return the typed-nil only for the node's concrete type;
					// all other nodes pass through unchanged.
					if reflect.TypeOf(out) != reflect.TypeOf(n) {
						return n, true
					}
					return out, true
				},
			})
			if ok {
				t.Fatalf("typed-nil Leave returned ok=true")
			}
		})

		// A wrong-class Enter (even with skip=true) must be rejected.
		t.Run("accept-enter-wrongclass/"+tc.name, func(t *testing.T) {
			_, ok := tc.root.Accept(&mgmtBoundaryVisitor{
				enter: func(n ast.Node) (ast.Node, bool) {
					if n == tc.root {
						// A non-matching concrete type, still non-nil.
						return &ast.BeginStmt{}, true
					}
					return n, false
				},
			})
			if ok {
				t.Fatalf("wrong-class Enter returned ok=true")
			}
		})
	}

	// Handler-specific typed-nil child slots: Limit, Where, Values entries.
	// The parent's Accept must reject a typed-nil returned for a required
	// concrete child even when the child's own Accept succeeded.
	{
		handler := func(limit, where bool) *ast.HandlerStmt {
			s := &ast.HandlerStmt{
				Tp: ast.HandlerRead, Table: mgmtBoundaryTable(), Index: lex.Ident{Name: "ix"},
				Op: pkast.OperatorEQ, Values: []ast.ExprNode{ast.NewParamMarkerExpr(0)},
			}
			if limit {
				s.Limit = &ast.Limit{BaseLimit: pkast.BaseLimit{Count: ast.NewParamMarkerExpr(2)}}
			}
			if where {
				s.Where = ast.NewParamMarkerExpr(1)
			}
			return s
		}
		// typed-nil Limit with Limit present
		if _, ok := handler(true, false).Accept(&mgmtBoundaryVisitor{
			enter: func(n ast.Node) (ast.Node, bool) { return n, false },
			leave: func(n ast.Node) (ast.Node, bool) {
				if _, isLimit := n.(*ast.Limit); isLimit {
					return (*ast.Limit)(nil), true
				}
				return n, true
			},
		}); ok {
			t.Fatal("typed-nil Limit accepted with Limit present")
		}
		// typed-nil Where with Where present
		if _, ok := handler(false, true).Accept(&mgmtBoundaryVisitor{
			enter: func(n ast.Node) (ast.Node, bool) { return n, false },
			leave: func(n ast.Node) (ast.Node, bool) {
				if _, isExpr := n.(*ast.ParamMarkerExpr); isExpr {
					return (*ast.ParamMarkerExpr)(nil), true
				}
				return n, true
			},
		}); ok {
			t.Fatal("typed-nil Where accepted with Where present")
		}
		// typed-nil Values entry
		if _, ok := handler(false, false).Accept(&mgmtBoundaryVisitor{
			enter: func(n ast.Node) (ast.Node, bool) { return n, false },
			leave: func(n ast.Node) (ast.Node, bool) {
				if _, isExpr := n.(*ast.ParamMarkerExpr); isExpr {
					return (*ast.ParamMarkerExpr)(nil), true
				}
				return n, true
			},
		}); ok {
			t.Fatal("typed-nil Values entry accepted")
		}
	}

	// Required specific-child rejection: a cache entry whose Table is nil must
	// be rejected by the entry's own Accept.
	{
		if _, ok := (&ast.CacheIndexStmt{Tables: []*ast.CacheIndexTable{{}}, Target: &ast.CacheIndexTarget{Kind: ast.CacheIndexDefaultTarget}}).Accept(&mgmtBoundaryVisitor{}); ok {
			t.Fatal("cache stmt with nil entry table accepted")
		}
	}
}

func TestManagementTableEnterBoundary(t *testing.T) {
	for _, tc := range []struct {
		name string
		make func() ast.Node
	}{
		{"handler", func() ast.Node { return &ast.HandlerStmt{Tp: ast.HandlerOpen, Table: mgmtBoundaryTable()} }},
		{"cache", func() ast.Node { return &ast.CacheIndexTable{Table: mgmtBoundaryTable()} }},
		{"load", func() ast.Node { return &ast.LoadIndexTable{Table: mgmtBoundaryTable()} }},
	} {
		for _, skip := range []bool{false, true} {
			for _, replacement := range []ast.Node{&ast.BeginStmt{}, (*ast.TableName)(nil)} {
				t.Run(tc.name, func(t *testing.T) {
					defer func() {
						if r := recover(); r != nil {
							t.Fatalf("table Enter replacement panicked: %v", r)
						}
					}()
					if _, ok := tc.make().Accept(&mgmtBoundaryVisitor{enter: func(n ast.Node) (ast.Node, bool) {
						if _, ok := n.(*ast.TableName); ok {
							return replacement, skip
						}
						return n, false
					}}); ok {
						t.Fatal("invalid table Enter replacement accepted")
					}
				})
			}
		}
	}
}

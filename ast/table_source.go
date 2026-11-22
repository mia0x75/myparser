package ast

import (
	"fmt"

	"github.com/mia0x75/myparser/format"
	pkast "github.com/mia0x75/parserkit/ast"
)

// TableSourceNode exposes source syntax shared with other dialects.
type TableSourceNode = pkast.TableSourceNode

// TableSource keeps MySQL's post-alias index hints outside the shared core.
type TableSource struct {
	pkast.BaseTableSource
	IndexHints []*IndexHint
}

// TableSourceCore returns the shared data without replacing the dialect outer.
func (n *TableSource) TableSourceCore() *pkast.BaseTableSource {
	if n == nil {
		return nil
	}
	return &n.BaseTableSource
}

// Restore writes the source, its alias, and then MySQL's ordered index hints.
func (n *TableSource) Restore(ctx *format.RestoreCtx) error {
	if n != nil && len(n.IndexHints) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		table, ok := n.Source.(*TableName)
		if !ok || table == nil || n.Lateral || len(n.Columns) > 0 {
			return fmt.Errorf("restore table source: index hints require a table name without LATERAL or column aliases")
		}
	}
	if err := pkast.RestoreTableSource(ctx, n.TableSourceCore()); err != nil {
		return err
	}
	for i, hint := range n.IndexHints {
		if hint == nil {
			return fmt.Errorf("restore table source index hint %d: nil hint", i)
		}
		ctx.WritePlain(" ")
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := hint.Restore(ctx); err != nil {
			return fmt.Errorf("restore table source index hint %d: %w", i, err)
		}
	}
	return ctx.Err()
}

// Accept visits the actual MySQL outer and its source once; hints are values.
func (n *TableSource) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*TableSource)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	if outer.Source != nil {
		updated, ok := outer.Source.Accept(v)
		if !ok {
			return outer, false
		}
		source, ok := updated.(ResultSetNode)
		if !ok || source == nil {
			return outer, false
		}
		outer.Source = source
	}
	return v.Leave(outer)
}

var (
	_ ResultSetNode      = (*TableName)(nil)
	_ ResultSetNode      = (*TableSource)(nil)
	_ ResultSetNode      = (*SubqueryExpr)(nil)
	_ pkast.TableRefNode = (*TableName)(nil)
	_ TableSourceNode    = (*TableSource)(nil)
	_ QueryNode          = (*SelectStmt)(nil)
	_ QueryNode          = (*SetOprStmt)(nil)
	_ QueryNode          = (*ValuesStmt)(nil)
	_ QueryNode          = (*TableStmt)(nil)
)

package ast

import (
	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
)

var _ StmtNode = &TableMaintenanceStmt{}

// TableMaintenanceType is the type of a table maintenance statement.
type TableMaintenanceType int

// Table maintenance statement types.
const (
	TableMaintenanceCheck TableMaintenanceType = iota + 1
	TableMaintenanceRepair
	TableMaintenanceOptimize
	TableMaintenanceChecksum
)

// TableMaintenanceStmt represents CHECK/REPAIR/OPTIMIZE/CHECKSUM TABLE.
type TableMaintenanceStmt struct {
	stmtNode

	Tp      TableMaintenanceType
	Tables  []*TableName
	Options []string
}

// Restore implements Node interface.
func (n *TableMaintenanceStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore TableMaintenanceStmt: nil statement")
	}
	switch n.Tp {
	case TableMaintenanceCheck:
		ctx.WriteKeyword("CHECK TABLE ")
	case TableMaintenanceRepair:
		ctx.WriteKeyword("REPAIR TABLE ")
	case TableMaintenanceOptimize:
		ctx.WriteKeyword("OPTIMIZE TABLE ")
	case TableMaintenanceChecksum:
		ctx.WriteKeyword("CHECKSUM TABLE ")
	default:
		return errors.New("An error occurred while splicing TableMaintenanceStmt")
	}
	for i, table := range n.Tables {
		if table == nil {
			return errors.Errorf("restore TableMaintenanceStmt.Tables[%d]: missing table", i)
		}
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := table.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore TableMaintenanceStmt.Tables[%d]", i)
		}
	}
	for _, opt := range n.Options {
		ctx.WritePlain(" ")
		ctx.WriteKeyword(opt)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *TableMaintenanceStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	entered, ok := nodeAs[*TableMaintenanceStmt](newNode)
	if !ok {
		return n, false
	}
	if skipChildren {
		return v.Leave(entered)
	}
	n = entered
	for i, table := range n.Tables {
		updated, ok := acceptNode[*TableName](v, table)
		if !ok {
			return n, false
		}
		n.Tables[i] = updated
	}
	return v.Leave(n)
}

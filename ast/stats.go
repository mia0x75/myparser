package ast

import (
	"github.com/mia0x75/myparser/errors"

	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/lex"
)

var (
	_ StmtNode = &AnalyzeTableStmt{}
	_ StmtNode = &DropStatsStmt{}
	_ StmtNode = &LoadStatsStmt{}
)

// AnalyzeTableStmt is used to create table statistics.
type AnalyzeTableStmt struct {
	stmtNode

	TableNames     []*TableName
	PartitionNames []lex.Ident
	IndexNames     []lex.Ident
	MaxNumBuckets  uint64

	// IndexFlag is true when we only analyze indices for a table.
	IndexFlag bool
	// NoWriteToBinLog records LOCAL or NO_WRITE_TO_BINLOG.
	NoWriteToBinLog bool
	// Persistent selects MariaDB's engine-independent statistics.
	Persistent bool
	// PersistentAll records FOR ALL. Otherwise nil name lists mean ALL,
	// whereas non-nil empty lists mean an explicit empty list.
	PersistentAll bool
	Columns       []lex.Ident
}

// Restore implements Node interface.
func (n *AnalyzeTableStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore AnalyzeTableStmt: nil statement")
	}
	ctx.WriteKeyword("ANALYZE ")
	if n.NoWriteToBinLog {
		ctx.WriteKeyword("NO_WRITE_TO_BINLOG ")
	}
	ctx.WriteKeyword("TABLE ")
	for i, table := range n.TableNames {
		if table == nil {
			return errors.Errorf("restore AnalyzeTableStmt.TableNames[%d]: missing table", i)
		}
		if i != 0 {
			ctx.WritePlain(",")
		}
		if err := table.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore AnalyzeTableStmt.TableNames[%d]", i)
		}
	}
	if n.Persistent {
		ctx.WriteKeyword(" PERSISTENT FOR ")
		if n.PersistentAll {
			ctx.WriteKeyword("ALL")
		} else {
			ctx.WriteKeyword("COLUMNS ")
			if n.Columns == nil {
				ctx.WriteKeyword("ALL")
			} else {
				ctx.WritePlain("(")
				for i, column := range n.Columns {
					if i > 0 {
						ctx.WritePlain(",")
					}
					ctx.WriteIdent(column)
				}
				ctx.WritePlain(")")
			}
			ctx.WriteKeyword(" INDEXES ")
			restoreAnalyzeNames(ctx, n.IndexNames)
		}
		return ctx.Err()
	}
	if len(n.PartitionNames) != 0 {
		ctx.WriteKeyword(" PARTITION ")
	}
	for i, partition := range n.PartitionNames {
		if i != 0 {
			ctx.WritePlain(",")
		}
		ctx.WriteIdent(partition)
	}
	if n.IndexFlag {
		ctx.WriteKeyword(" INDEX")
	}
	for i, index := range n.IndexNames {
		if i != 0 {
			ctx.WritePlain(",")
		} else {
			ctx.WritePlain(" ")
		}
		ctx.WriteIdent(index)
	}
	if n.MaxNumBuckets != 0 {
		ctx.WriteKeyword(" WITH ")
		ctx.WritePlainf("%d", n.MaxNumBuckets)
		ctx.WriteKeyword(" BUCKETS")
	}
	return ctx.Err()
}

func restoreAnalyzeNames(ctx *format.RestoreCtx, names []lex.Ident) {
	if names == nil {
		ctx.WriteKeyword("ALL")
		return
	}
	ctx.WritePlain("(")
	for i, name := range names {
		if i > 0 {
			ctx.WritePlain(",")
		}
		ctx.WriteIdent(name)
	}
	ctx.WritePlain(")")
}

// Accept implements Node Accept interface.
func (n *AnalyzeTableStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, skipChildren := v.Enter(n)
	entered, ok := nodeAs[*AnalyzeTableStmt](newNode)
	if !ok {
		return n, false
	}
	if skipChildren {
		return v.Leave(entered)
	}
	n = entered
	for i, val := range n.TableNames {
		updated, ok := acceptNode[*TableName](v, val)
		if !ok {
			return n, false
		}
		n.TableNames[i] = updated
	}
	return v.Leave(n)
}

// DropStatsStmt is used to drop table statistics.
type DropStatsStmt struct {
	stmtNode

	Table *TableName
}

// Restore implements Node interface.
func (n *DropStatsStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Table == nil {
		return errors.New("restore DropStatsStmt: missing table")
	}
	ctx.WriteKeyword("DROP STATS ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while add table")
	}

	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *DropStatsStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, skipChildren := v.Enter(n)
	entered, ok := nodeAs[*DropStatsStmt](newNode)
	if !ok {
		return n, false
	}
	if skipChildren {
		return v.Leave(entered)
	}
	n = entered
	updated, ok := acceptNode[*TableName](v, n.Table)
	if !ok {
		return n, false
	}
	n.Table = updated
	return v.Leave(n)
}

// LoadStatsStmt is the statement node for loading statistic.
type LoadStatsStmt struct {
	stmtNode

	Path string
}

// Restore implements Node interface.
func (n *LoadStatsStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore LoadStatsStmt: nil statement")
	}
	ctx.WriteKeyword("LOAD STATS ")
	ctx.WriteString(n.Path)
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *LoadStatsStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return n, false
	}
	newNode, skipChildren := v.Enter(n)
	entered, ok := nodeAs[*LoadStatsStmt](newNode)
	if !ok {
		return n, false
	}
	if skipChildren {
		return v.Leave(entered)
	}
	return v.Leave(entered)
}

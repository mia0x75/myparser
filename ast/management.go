package ast

import (
	"io"
	"strconv"
	"strings"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"

	"github.com/mia0x75/myparser/auth"
	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
)

// ChangeMasterStmt changes a replication connection's source and configuration.
type ChangeMasterStmt struct {
	stmtNode

	Replica           bool
	Options           []*ReplicationOption
	ConnectionName    string
	HasConnectionName bool
	Channel           string
	HasChannel        bool
}

// Restore implements Node interface.
func (n *ChangeMasterStmt) Restore(ctx *format.RestoreCtx) error {
	if n.Replica {
		ctx.WriteKeyword("CHANGE REPLICATION SOURCE")
	} else {
		ctx.WriteKeyword("CHANGE MASTER")
	}
	if n.HasConnectionName || n.ConnectionName != "" {
		ctx.WritePlain(" ")
		ctx.WriteString(n.ConnectionName)
	}
	ctx.WriteKeyword(" TO ")
	for i, opt := range n.Options {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := restoreReplicationOption(ctx, opt); err != nil {
			return errors.Annotatef(err, "An error occurred while restore ChangeMasterStmt.Options[%d]", i)
		}
	}
	if n.HasChannel || n.Channel != "" {
		ctx.WriteKeyword(" FOR CHANNEL ")
		ctx.WriteString(n.Channel)
	}
	return ctx.Err()
}

// Accept implements Node Accept interface.
func (n *ChangeMasterStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*ChangeMasterStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	for i, opt := range n.Options {
		option, ok := acceptNode[*ReplicationOption](v, opt)
		if !ok {
			return n, false
		}
		n.Options[i] = option
	}
	return v.Leave(n)
}

// ReplicaSuffix holds the optional clauses while parsing START/STOP replication.
type ReplicaSuffix struct {
	Threads            []string
	UntilOptions       []*ReplicationOption
	UntilAfterGaps     bool
	ConnectionOptions  []*ReplicationOption
	ConnectionName     string
	HasConnectionName  bool
	Channel            string
	HasChannel         bool
	ChannelBeforeUntil bool
}

// ReplicaControlStmt starts or stops replication threads, optionally for a channel.
type ReplicaControlStmt struct {
	stmtNode

	Start              bool
	Replica            bool
	All                bool
	Threads            []string
	UntilOptions       []*ReplicationOption
	UntilAfterGaps     bool
	ConnectionOptions  []*ReplicationOption
	ConnectionName     string
	HasConnectionName  bool
	Channel            string
	HasChannel         bool
	ChannelBeforeUntil bool
}

// Restore implements Node interface.
func (n *ReplicaControlStmt) Restore(ctx *format.RestoreCtx) error {
	if n.Start {
		ctx.WriteKeyword("START ")
	} else {
		ctx.WriteKeyword("STOP ")
	}
	if n.All && n.Replica {
		ctx.WriteKeyword("ALL REPLICAS")
	} else if n.All {
		ctx.WriteKeyword("ALL SLAVES")
	} else if n.Replica {
		ctx.WriteKeyword("REPLICA")
	} else {
		ctx.WriteKeyword("SLAVE")
	}
	if n.HasConnectionName || n.ConnectionName != "" {
		ctx.WritePlain(" ")
		ctx.WriteString(n.ConnectionName)
	}
	for i, thread := range n.Threads {
		if i == 0 {
			ctx.WritePlain(" ")
		} else {
			ctx.WritePlain(", ")
		}
		ctx.WriteKeyword(strings.ToUpper(thread))
	}
	if n.ChannelBeforeUntil {
		n.restoreChannel(ctx)
	}
	if n.UntilAfterGaps {
		ctx.WriteKeyword(" UNTIL SQL_AFTER_MTS_GAPS")
	} else if len(n.UntilOptions) != 0 {
		ctx.WriteKeyword(" UNTIL ")
		for i, opt := range n.UntilOptions {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if err := restoreReplicationOption(ctx, opt); err != nil {
				return errors.Annotatef(err, "An error occurred while restore ReplicaControlStmt.UntilOptions[%d]", i)
			}
		}
	}
	for i, opt := range n.ConnectionOptions {
		ctx.WritePlain(" ")
		if err := restoreReplicationOption(ctx, opt); err != nil {
			return errors.Annotatef(err, "An error occurred while restore ReplicaControlStmt.ConnectionOptions[%d]", i)
		}
	}
	if !n.ChannelBeforeUntil {
		n.restoreChannel(ctx)
	}
	return ctx.Err()
}

func (n *ReplicaControlStmt) restoreChannel(ctx *format.RestoreCtx) {
	if n.HasChannel || n.Channel != "" {
		ctx.WriteKeyword(" FOR CHANNEL ")
		ctx.WriteString(n.Channel)
	}
}

// Accept implements Node Accept interface.
func (n *ReplicaControlStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*ReplicaControlStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	for i, opt := range n.UntilOptions {
		option, ok := acceptNode[*ReplicationOption](v, opt)
		if !ok {
			return n, false
		}
		n.UntilOptions[i] = option
	}
	for i, opt := range n.ConnectionOptions {
		option, ok := acceptNode[*ReplicationOption](v, opt)
		if !ok {
			return n, false
		}
		n.ConnectionOptions[i] = option
	}
	return v.Leave(n)
}

// ReplicationOptionValue represents a bare keyword, parenthesized ID list, or account.
// Exactly one field is set; an empty, non-nil Values slice represents an empty list.
// Scalar literals and DEFAULT use LiteralExpr and DefaultExpr instead.
type ReplicationOptionValue struct {
	pkast.BaseExpr

	Keyword string
	Values  []ExprNode
	User    *auth.UserIdentity
}

// Restore implements Node interface.
func (n *ReplicationOptionValue) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore ReplicationOptionValue: nil value")
	}
	switch {
	case n.Keyword != "":
		ctx.WriteKeyword(n.Keyword)
	case n.Values != nil:
		ctx.WritePlain("(")
		if err := restoreExprNodes(ctx, n.Values); err != nil {
			return errors.Annotate(err, "An error occurred while restore ReplicationOptionValue.Values")
		}
		ctx.WritePlain(")")
	case n.User != nil:
		ctx.WriteString(n.User.Username)
		ctx.WritePlain("@")
		ctx.WriteString(n.User.Hostname)
	default:
		return errors.New("Invalid replication option value")
	}
	return ctx.Err()
}

// Format implements ExprNode interface.
func (n *ReplicationOptionValue) Format(w io.Writer) {
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, w)
	switch {
	case n.Keyword != "":
		ctx.WriteKeyword(n.Keyword)
	case n.Values != nil:
		ctx.WritePlain("(")
		for i, value := range n.Values {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			FormatExpr(value, w)
		}
		ctx.WritePlain(")")
	case n.User != nil:
		ctx.WriteString(n.User.Username)
		ctx.WritePlain("@")
		ctx.WriteString(n.User.Hostname)
	}
}

// Accept implements Node Accept interface.
func (n *ReplicationOptionValue) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*ReplicationOptionValue](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	for i, value := range n.Values {
		expr, ok := acceptNode[ExprNode](v, value)
		if !ok {
			return n, false
		}
		n.Values[i] = expr
	}
	return v.Leave(n)
}

func restoreReplicationOption(ctx *format.RestoreCtx, option *ReplicationOption) error {
	if option == nil {
		return errors.New("Incomplete replication option")
	}
	return option.Restore(ctx)
}

type ResetStmtType int

const (
	ResetMaster ResetStmtType = iota
	ResetSlave
	ResetReplica
	ResetPersist
	ResetBinaryLogsAndGTIDs
	ResetQueryCache
)

type ResetSuffix struct {
	All        bool
	Channel    string
	HasChannel bool
}

type ResetStmt struct {
	stmtNode

	// Options contains the individual options of a multi-option RESET statement.
	Options     []*ResetStmt
	Tp          ResetStmtType
	Args        []ExprNode
	IfExists    bool
	All         bool
	Channel     string
	HasChannel  bool
	PersistName QualifiedName
}

func (n *ResetStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("RESET ")
	if len(n.Options) != 0 {
		for i, option := range n.Options {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if err := option.restoreOption(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore ResetStmt.Options[%d]", i)
			}
		}
		return ctx.Err()
	}
	return n.restoreOption(ctx)
}

func (n *ResetStmt) restoreOption(ctx *format.RestoreCtx) error {
	if n == nil {
		return errors.New("restore ResetStmt: nil option")
	}
	switch n.Tp {
	case ResetMaster:
		ctx.WriteKeyword("MASTER")
		if len(n.Args) != 0 {
			ctx.WriteKeyword(" TO ")
			if err := restoreExprNodes(ctx, n.Args); err != nil {
				return err
			}
		}
	case ResetSlave:
		ctx.WriteKeyword("SLAVE")
		if n.All {
			ctx.WriteKeyword(" ALL")
		}
	case ResetReplica:
		ctx.WriteKeyword("REPLICA")
		if n.All {
			ctx.WriteKeyword(" ALL")
		}
	case ResetPersist:
		ctx.WriteKeyword("PERSIST")
		if n.IfExists {
			ctx.WriteKeyword(" IF EXISTS")
		}
		if len(n.PersistName.Parts) != 0 {
			ctx.WritePlain(" ")
			for i, part := range n.PersistName.Parts {
				if i > 0 {
					ctx.WritePlain(".")
				}
				restoreStoredIdent(ctx, part)
			}
		}
	case ResetBinaryLogsAndGTIDs:
		ctx.WriteKeyword("BINARY LOGS AND GTIDS")
		if len(n.Args) != 0 {
			ctx.WriteKeyword(" TO ")
			if err := restoreExprNodes(ctx, n.Args); err != nil {
				return err
			}
		}
	case ResetQueryCache:
		ctx.WriteKeyword("QUERY CACHE")
	}
	if n.HasChannel || n.Channel != "" {
		ctx.WriteKeyword(" FOR CHANNEL ")
		ctx.WriteString(n.Channel)
	}
	return ctx.Err()
}

func (n *ResetStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*ResetStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	for i, option := range n.Options {
		updated, ok := acceptNode[*ResetStmt](v, option)
		if !ok {
			return n, false
		}
		n.Options[i] = updated
	}
	for i, arg := range n.Args {
		expr, ok := acceptNode[ExprNode](v, arg)
		if !ok {
			return n, false
		}
		n.Args[i] = expr
	}
	return v.Leave(n)
}

type ShutdownStmt struct {
	stmtNode

	WaitForAllSlaves bool
}

func (n *ShutdownStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("SHUTDOWN")
	if n.WaitForAllSlaves {
		ctx.WriteKeyword(" WAIT FOR ALL SLAVES")
	}
	return ctx.Err()
}

func (n *ShutdownStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	outer, ok := nodeAs[*ShutdownStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(outer)
}

type HandlerStmtType int

const (
	HandlerOpen HandlerStmtType = iota
	HandlerRead
	HandlerClose
)

// HandlerReadDirection is a HANDLER READ movement: first, next, prev or last.
// HandlerReadUnknown means the key was read by comparison instead of by
// movement.
type HandlerReadDirection int

const (
	HandlerReadUnknown HandlerReadDirection = iota
	HandlerReadFirst
	HandlerReadNext
	HandlerReadPrev
	HandlerReadLast
)

func handlerDirectionKeyword(direction HandlerReadDirection) (string, bool) {
	switch direction {
	case HandlerReadFirst:
		return "FIRST", true
	case HandlerReadNext:
		return "NEXT", true
	case HandlerReadPrev:
		return "PREV", true
	case HandlerReadLast:
		return "LAST", true
	}
	return "", false
}

// HandlerStmt is a HANDLER OPEN/CLOSE/READ statement.
type HandlerStmt struct {
	stmtNode

	Tp        HandlerStmtType
	Table     *TableName
	Alias     lex.Ident
	Index     lex.Ident
	Op        pkast.Operator
	Values    []ExprNode
	Direction HandlerReadDirection
	Where     ExprNode
	Limit     *Limit
}

func (n *HandlerStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("restore HandlerStmt: nil node")
	}
	// The Table is required by every HANDLER operation.
	if n.Table == nil {
		return errors.New("restore HandlerStmt: missing table")
	}
	for _, value := range n.Values {
		if !nodePresent(value) {
			return errors.New("restore HandlerStmt: missing comparison value")
		}
	}
	if n.Where != nil && !nodePresent(n.Where) {
		return errors.New("restore HandlerStmt: nil WHERE expression")
	}
	ctx.WriteKeyword("HANDLER ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore HandlerStmt.Table")
	}
	switch n.Tp {
	case HandlerOpen:
		if n.Index.Present() || n.Op != pkast.OperatorUnknown || len(n.Values) != 0 ||
			n.Direction != HandlerReadUnknown || n.Where != nil || n.Limit != nil {
			return errors.New("restore HandlerStmt: read fields are not allowed on OPEN")
		}
		ctx.WriteKeyword(" OPEN")
		if n.Alias.Present() {
			ctx.WriteKeyword(" AS ")
			ctx.WriteIdent(n.Alias)
		}
	case HandlerClose:
		// CLOSE names the opened alias, not a table index or read fields.
		if n.Alias.Present() || n.Index.Present() || n.Op != pkast.OperatorUnknown ||
			len(n.Values) != 0 || n.Direction != HandlerReadUnknown ||
			n.Where != nil || n.Limit != nil {
			return errors.New("restore HandlerStmt: aliases and read fields are not allowed on CLOSE")
		}
		ctx.WriteKeyword(" CLOSE")
	case HandlerRead:
		if n.Alias.Present() {
			return errors.New("restore HandlerStmt: aliases are not allowed on READ")
		}
		ctx.WriteKeyword(" READ")
		if n.Op != pkast.OperatorUnknown {
			// A comparison read requires an index key and at least one value.
			if !n.Index.Present() || len(n.Values) == 0 {
				return errors.New("restore HandlerStmt: comparison read requires an index and values")
			}
			if n.Direction != HandlerReadUnknown {
				return errors.New("restore HandlerStmt: direction is not allowed on a comparison read")
			}
		} else {
			if len(n.Values) != 0 {
				return errors.New("restore HandlerStmt: movement read carries comparison values")
			}
			// Movement reads must name a direction; unknown kinds are illegal
			// instead of being silently dropped. Without an index only
			// FIRST and NEXT are valid on the primary key.
			if _, hasDirection := handlerDirectionKeyword(n.Direction); !hasDirection {
				return errors.New("restore HandlerStmt: missing direction on movement read")
			}
			if !n.Index.Present() && n.Direction != HandlerReadFirst && n.Direction != HandlerReadNext {
				return errors.New("restore HandlerStmt: primary key reads only support FIRST and NEXT")
			}
		}
		if n.Index.Present() {
			ctx.WritePlain(" ")
			ctx.WriteIdent(n.Index)
		}
		if n.Op != pkast.OperatorUnknown {
			// Only the five HANDLER comparison operators are allowed; any other
			// out-of-range operator is a manually constructed illegal node.
			if n.Op != pkast.OperatorEQ && n.Op != pkast.OperatorGE &&
				n.Op != pkast.OperatorLE && n.Op != pkast.OperatorGT &&
				n.Op != pkast.OperatorLT {
				return errors.New("restore HandlerStmt: operator not supported by HANDLER READ")
			}
			ctx.WritePlain(" ")
			switch n.Op {
			case pkast.OperatorEQ:
				ctx.WritePlain("=")
			case pkast.OperatorGE:
				ctx.WritePlain(">=")
			case pkast.OperatorLE:
				ctx.WritePlain("<=")
			case pkast.OperatorGT:
				ctx.WritePlain(">")
			case pkast.OperatorLT:
				ctx.WritePlain("<")
			}
			ctx.WritePlain(" (")
			for i, value := range n.Values {
				if i != 0 {
					ctx.WritePlain(", ")
				}
				if err := value.Restore(ctx); err != nil {
					return errors.Annotatef(err, "An error occurred while restore HandlerStmt.Values[%d]", i)
				}
			}
			ctx.WritePlain(")")
		} else {
			// A movement read: direction always follows with a leading space.
			ctx.WritePlain(" ")
			keyword, _ := handlerDirectionKeyword(n.Direction)
			ctx.WriteKeyword(keyword)
		}
		if n.Where != nil {
			ctx.WriteKeyword(" WHERE ")
			if err := n.Where.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore HandlerStmt.Where")
			}
		}
	default:
		return errors.New("restore HandlerStmt: unknown statement type")
	}
	if n.Limit != nil {
		ctx.WritePlain(" ")
		if err := n.Limit.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore HandlerStmt.Limit")
		}
	}
	return ctx.Err()
}

func (n *HandlerStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	// Outer role check first: the replacement must be the concrete type.
	stmt, isStmt := newNode.(*HandlerStmt)
	if !isStmt || stmt == nil {
		return newNode, false
	}
	if skipChildren {
		// Required role: a skipped replacement must still be the concrete type.
		return v.Leave(newNode)
	}
	n = stmt
	// Required role: the concrete table of every HANDLER operation.
	if n.Table == nil {
		return n, false
	}
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	if table, isTable := node.(*TableName); !isTable || table == nil {
		return n, false
	} else {
		n.Table = table
	}
	for i, value := range n.Values {
		// Required child: each comparison value must exist.
		if !nodePresent(value) {
			return n, false
		}
		node, ok = value.Accept(v)
		if !ok {
			return n, false
		}
		if expr, isExpr := node.(ExprNode); !isExpr || !nodePresent(expr) {
			return n, false
		} else {
			n.Values[i] = expr
		}
	}
	if n.Where != nil {
		if !nodePresent(n.Where) {
			return n, false
		}
		node, ok = n.Where.Accept(v)
		if !ok {
			return n, false
		}
		if expr, isExpr := node.(ExprNode); !isExpr || !nodePresent(expr) {
			return n, false
		} else {
			n.Where = expr
		}
	}
	if n.Limit != nil {
		node, ok = n.Limit.Accept(v)
		if !ok {
			return n, false
		}
		if limit, isLimit := node.(*Limit); !isLimit || limit == nil {
			return n, false
		} else {
			n.Limit = limit
		}
	}
	updated, ok := v.Leave(n)
	if !ok {
		return n, false
	}
	if stmt, isStmt := updated.(*HandlerStmt); !isStmt || stmt == nil {
		return updated, false
	}
	return updated, true
}

type CacheIndexTargetType int

const (
	CacheIndexUnknownTarget CacheIndexTargetType = iota
	CacheIndexDefaultTarget
	CacheIndexIdentifierTarget
)

// CacheIndexStmt is the CACHE INDEX statement.
type CacheIndexStmt struct {
	stmtNode

	Tables []*CacheIndexTable
	Target *CacheIndexTarget
}

// CacheIndexTable is one table entry of a CACHE INDEX statement.
// A nil IndexNames slice omits INDEX/KEY entirely; a non-nil empty slice
// preserves an explicit INDEX ().
type CacheIndexTable struct {
	node

	Table        *TableName
	IndexNames   []lex.Ident
	PartitionAll bool
}

// CacheIndexTarget is the IN target of a CACHE INDEX statement.
type CacheIndexTarget struct {
	node

	Kind CacheIndexTargetType
	Name lex.Ident
}

func restoreCacheIndexNames(ctx *format.RestoreCtx, names []lex.Ident) {
	ctx.WriteKeyword(" INDEX (")
	for i, name := range names {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		ctx.WriteIdent(name)
	}
	ctx.WritePlain(")")
}

func (n *CacheIndexTable) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("restore CacheIndexTable: nil node")
	}
	if n.Table == nil {
		return errors.New("restore CacheIndexTable: missing table")
	}
	// Partition ALL and an explicit partition name list are mutually exclusive.
	if n.PartitionAll && len(n.Table.PartitionNames) > 0 {
		return errors.New("restore CacheIndexTable: partition ALL conflicts with named partitions")
	}
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore CacheIndexTable.Table")
	}
	if n.PartitionAll {
		ctx.WriteKeyword(" PARTITION (ALL)")
	}
	if n.IndexNames != nil {
		for _, name := range n.IndexNames {
			if !name.Present() {
				return errors.New("restore CacheIndexTable: missing index name")
			}
		}
		restoreCacheIndexNames(ctx, n.IndexNames)
	}
	return ctx.Err()
}

func (n *CacheIndexTable) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	// Outer role check first: the replacement must be the concrete type.
	entry, isEntry := newNode.(*CacheIndexTable)
	if !isEntry || entry == nil {
		return newNode, false
	}
	if skipChildren {
		// Required role: a skipped replacement must still be the concrete type.
		return v.Leave(newNode)
	}
	n = entry
	// Required role: the concrete table of the entry.
	if n.Table == nil {
		return n, false
	}
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	if table, isTable := node.(*TableName); !isTable || table == nil {
		return n, false
	} else {
		n.Table = table
	}
	updated, ok := v.Leave(n)
	if !ok {
		return n, false
	}
	if entry, isEntry := updated.(*CacheIndexTable); !isEntry || entry == nil {
		return updated, false
	}
	return updated, true
}

func (n *CacheIndexTarget) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("restore CacheIndexTarget: nil node")
	}
	switch n.Kind {
	case CacheIndexDefaultTarget:
		// A manually constructed default target must not carry a name.
		if n.Name.Present() {
			return errors.New("restore CacheIndexTarget: default target must not carry a name")
		}
		ctx.WriteKeyword("DEFAULT")
		return ctx.Err()
	case CacheIndexIdentifierTarget:
		// The quoted-default and bare-DEFAULT targets are distinct kinds; a
		// quoted empty name is still a present identifier.
		if !n.Name.Present() {
			return errors.New("restore CacheIndexTarget: identifier target requires a name")
		}
		ctx.WriteIdent(n.Name)
		return ctx.Err()
	}
	// A manually constructed unknown target kind is rejected here instead of
	// being silently restored as DEFAULT.
	return errors.New("restore CacheIndexTarget: unknown target kind")
}

func (n *CacheIndexTarget) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	// Outer role check first: the replacement must be the concrete type.
	entry, isEntry := newNode.(*CacheIndexTarget)
	if !isEntry || entry == nil {
		return newNode, false
	}
	if skipChildren {
		// Required role: a skipped replacement must still be the concrete type.
		return v.Leave(newNode)
	}
	n = entry
	updated, ok := v.Leave(n)
	if !ok {
		return n, false
	}
	if leaf, isLeaf := updated.(*CacheIndexTarget); !isLeaf || leaf == nil {
		return updated, false
	}
	return updated, true
}

func (n *CacheIndexStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("restore CacheIndexStmt: nil node")
	}
	if len(n.Tables) == 0 {
		return errors.New("restore CacheIndexStmt: missing tables")
	}
	if n.Target == nil {
		return errors.New("restore CacheIndexStmt: missing target")
	}
	for _, table := range n.Tables {
		if table == nil {
			return errors.New("restore CacheIndexStmt: missing table")
		}
	}
	ctx.WriteKeyword("CACHE INDEX ")
	for i, table := range n.Tables {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := table.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore CacheIndexStmt.Tables[%d]", i)
		}
	}
	ctx.WriteKeyword(" IN ")
	return errors.Annotate(n.Target.Restore(ctx), "An error occurred while restore CacheIndexStmt.Target")
}

func (n *CacheIndexStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	// Outer role check first: the replacement must be the concrete type.
	stmt, isStmt := newNode.(*CacheIndexStmt)
	if !isStmt || stmt == nil {
		return newNode, false
	}
	if skipChildren {
		// Required roles: a skipped replacement must keep the concrete type.
		return v.Leave(newNode)
	}
	n = stmt
	// Required roles: a non-empty table list and a concrete target.
	if len(n.Tables) == 0 {
		return n, false
	}
	if n.Target == nil {
		return n, false
	}
	for _, table := range n.Tables {
		if table == nil {
			return n, false
		}
	}
	for i, table := range n.Tables {
		node, ok := table.Accept(v)
		if !ok {
			return n, false
		}
		if entry, isEntry := node.(*CacheIndexTable); !isEntry || entry == nil {
			return n, false
		} else {
			n.Tables[i] = entry
		}
	}
	node, ok := n.Target.Accept(v)
	if !ok {
		return n, false
	}
	if target, isTarget := node.(*CacheIndexTarget); !isTarget || target == nil {
		return n, false
	} else {
		n.Target = target
	}
	updated, ok := v.Leave(n)
	if !ok {
		return n, false
	}
	if stmt, isStmt := updated.(*CacheIndexStmt); !isStmt || stmt == nil {
		return updated, false
	}
	return updated, true
}

type LoadIndexStmt struct {
	stmtNode

	Tables []*LoadIndexTable
}

func (n *LoadIndexStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("restore LoadIndexStmt: nil node")
	}
	if len(n.Tables) == 0 {
		return errors.New("restore LoadIndexStmt: missing tables")
	}
	for _, table := range n.Tables {
		if table == nil {
			return errors.New("restore LoadIndexStmt: missing table")
		}
	}
	ctx.WriteKeyword("LOAD INDEX INTO CACHE ")
	for i, table := range n.Tables {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := table.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore LoadIndexStmt.Tables[%d]", i)
		}
	}
	return ctx.Err()
}

func (n *LoadIndexStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	// Outer role check first: the replacement must be the concrete type.
	stmt, isStmt := newNode.(*LoadIndexStmt)
	if !isStmt || stmt == nil {
		return newNode, false
	}
	if skipChildren {
		// Required role: a skipped replacement must keep the concrete type.
		return v.Leave(newNode)
	}
	n = stmt
	// Required roles: a non-empty table list without nil entries.
	if len(n.Tables) == 0 {
		return n, false
	}
	for _, table := range n.Tables {
		if table == nil {
			return n, false
		}
	}
	for i, table := range n.Tables {
		node, ok := table.Accept(v)
		if !ok {
			return n, false
		}
		if entry, isEntry := node.(*LoadIndexTable); !isEntry || entry == nil {
			return n, false
		} else {
			n.Tables[i] = entry
		}
	}
	updated, ok := v.Leave(n)
	if !ok {
		return n, false
	}
	if stmt, isStmt := updated.(*LoadIndexStmt); !isStmt || stmt == nil {
		return updated, false
	}
	return updated, true
}

// LoadIndexTable is one table entry of a LOAD INDEX INTO CACHE statement.
// A nil IndexNames slice omits INDEX/KEY; a non-nil empty slice keeps INDEX ().
type LoadIndexTable struct {
	node

	Table        *TableName
	IndexNames   []lex.Ident
	PartitionAll bool
	IgnoreLeaves bool
}

func (n *LoadIndexTable) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("restore LoadIndexTable: nil node")
	}
	if n.Table == nil {
		return errors.New("restore LoadIndexTable: missing table")
	}
	// Partition ALL and an explicit partition name list are mutually exclusive.
	if n.PartitionAll && len(n.Table.PartitionNames) > 0 {
		return errors.New("restore LoadIndexTable: partition ALL conflicts with named partitions")
	}
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore LoadIndexTable.Table")
	}
	if n.PartitionAll {
		ctx.WriteKeyword(" PARTITION (ALL)")
	}
	if n.IndexNames != nil {
		for _, name := range n.IndexNames {
			if !name.Present() {
				return errors.New("restore LoadIndexTable: missing index name")
			}
		}
		restoreCacheIndexNames(ctx, n.IndexNames)
	}
	if n.IgnoreLeaves {
		ctx.WriteKeyword(" IGNORE LEAVES")
	}
	return ctx.Err()
}

func (n *LoadIndexTable) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	// Outer role check first: the replacement must be the concrete type.
	entry, isEntry := newNode.(*LoadIndexTable)
	if !isEntry || entry == nil {
		return newNode, false
	}
	if skipChildren {
		// Required role: a skipped replacement must keep the concrete type.
		return v.Leave(newNode)
	}
	n = entry
	// Required role: the concrete table of the entry.
	if n.Table == nil {
		return n, false
	}
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	if table, isTable := node.(*TableName); !isTable || table == nil {
		return n, false
	} else {
		n.Table = table
	}
	updated, ok := v.Leave(n)
	if !ok {
		return n, false
	}
	if entry, isEntry := updated.(*LoadIndexTable); !isEntry || entry == nil {
		return updated, false
	}
	return updated, true
}

type LoadXMLTagInfo struct {
	RowTag    string
	HasRowTag bool
}

type LoadXMLStmt struct {
	LoadDataStmt

	// Priority is empty, LOW_PRIORITY, or CONCURRENT.
	Priority       string
	HasIgnoreLines bool
	IgnoreRows     bool
	Charset        string
	RowTag         string
	HasRowTag      bool
}

func (n *LoadXMLStmt) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Table == nil {
		return errors.New("restore LoadXMLStmt: missing table")
	}
	ctx.WriteKeyword("LOAD XML ")
	if n.Priority != "" {
		ctx.WriteKeyword(n.Priority + " ")
	}
	if n.IsLocal {
		ctx.WriteKeyword("LOCAL ")
	}
	ctx.WriteKeyword("INFILE ")
	ctx.WriteString(n.Path)
	switch n.OnDuplicate {
	case OnDuplicateKeyHandlingReplace:
		ctx.WriteKeyword(" REPLACE")
	case OnDuplicateKeyHandlingIgnore:
		ctx.WriteKeyword(" IGNORE")
	}
	ctx.WriteKeyword(" INTO TABLE ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore LoadXMLStmt.Table")
	}
	if n.Charset != "" {
		ctx.WriteKeyword(" CHARACTER SET ")
		ctx.WriteName(n.Charset)
	}
	if n.HasRowTag {
		ctx.WriteKeyword(" ROWS IDENTIFIED BY ")
		ctx.WriteString(n.RowTag)
	}
	if n.HasIgnoreLines || n.IgnoreLines != 0 {
		ctx.WriteKeyword(" IGNORE ")
		ctx.WritePlainf("%d", n.IgnoreLines)
		if n.IgnoreRows {
			ctx.WriteKeyword(" ROWS")
		} else {
			ctx.WriteKeyword(" LINES")
		}
	}
	if len(n.ColumnsAndUserVars) != 0 {
		ctx.WritePlain(" (")
		for i, c := range n.ColumnsAndUserVars {
			if c == nil || (c.ColumnName == nil) == (c.UserVar == nil) {
				return errors.Errorf("restore LoadXMLStmt.ColumnsAndUserVars[%d]: missing or ambiguous target", i)
			}
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if c.ColumnName != nil {
				if err := restoreQualifiedName(ctx, *c.ColumnName); err != nil {
					return errors.Annotate(err, "An error occurred while restore LoadXMLStmt.ColumnsAndUserVars")
				}
			}
			if c.UserVar != nil {
				if err := c.UserVar.Restore(ctx); err != nil {
					return errors.Annotate(err, "An error occurred while restore LoadXMLStmt.ColumnsAndUserVars")
				}
			}
		}
		ctx.WritePlain(")")
	}
	if len(n.ColumnAssignments) != 0 {
		ctx.WriteKeyword(" SET")
		for i, assign := range n.ColumnAssignments {
			if assign == nil {
				return errors.Errorf("restore LoadXMLStmt.ColumnAssignments[%d]: missing assignment", i)
			}
			if i != 0 {
				ctx.WritePlain(",")
			}
			ctx.WritePlain(" ")
			if err := assign.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore LoadXMLStmt.ColumnAssignments")
			}
		}
	}
	return ctx.Err()
}

func (n *LoadXMLStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*LoadXMLStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	table, ok := acceptNode[*TableName](v, n.Table)
	if !ok {
		return n, false
	}
	n.Table = table
	for _, column := range n.ColumnsAndUserVars {
		if column == nil || (column.ColumnName == nil) == (column.UserVar == nil) {
			return n, false
		}
		if column.UserVar != nil {
			variable, ok := acceptNode[*VariableExpr](v, column.UserVar)
			if !ok {
				return n, false
			}
			column.UserVar = variable
		}
	}
	for i, assignment := range n.ColumnAssignments {
		updated, ok := acceptNode[*Assignment](v, assignment)
		if !ok {
			return n, false
		}
		n.ColumnAssignments[i] = updated
	}
	return v.Leave(n)
}

type RenameUserStmt struct {
	stmtNode

	Pairs []*RenameUserPair
}

func (n *RenameUserStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("RENAME USER ")
	for i, pair := range n.Pairs {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := pair.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore RenameUserStmt.Pairs[%d]", i)
		}
	}
	return ctx.Err()
}

func (n *RenameUserStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*RenameUserStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	for i, pair := range n.Pairs {
		updated, ok := acceptNode[*RenameUserPair](v, pair)
		if !ok {
			return n, false
		}
		n.Pairs[i] = updated
	}
	return v.Leave(n)
}

type RenameUserPair struct {
	node

	From *auth.UserIdentity
	To   *auth.UserIdentity
}

func (n *RenameUserPair) Restore(ctx *format.RestoreCtx) error {
	if err := restoreUserIdentity(ctx, n.From); err != nil {
		return errors.Annotate(err, "An error occurred while restore RenameUserPair.From")
	}
	ctx.WriteKeyword(" TO ")
	if err := restoreUserIdentity(ctx, n.To); err != nil {
		return errors.Annotate(err, "An error occurred while restore RenameUserPair.To")
	}
	return ctx.Err()
}

func (n *RenameUserPair) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	outer, ok := nodeAs[*RenameUserPair](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(outer)
}

func restoreUserIdentity(ctx *format.RestoreCtx, user *auth.UserIdentity) error {
	if user == nil {
		return errors.New("An error occurred while restore user identity")
	}
	if user.CurrentUser {
		ctx.WriteKeyword("CURRENT_USER")
		return ctx.Err()
	}
	ctx.WriteString(user.Username)
	ctx.WritePlain("@")
	ctx.WriteString(user.Hostname)
	return ctx.Err()
}

func restoreExprNodes(ctx *format.RestoreCtx, nodes []ExprNode) error {
	for i, node := range nodes {
		if !nodePresent(node) {
			return errors.Errorf("restore expression[%d]: missing expression", i)
		}
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := node.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore expression[%d]", i)
		}
	}
	return ctx.Err()
}

// ImportTableStmt is the IMPORT TABLE statement.
// See https://dev.mysql.com/doc/refman/8.0/en/import-table.html
type ImportTableStmt struct {
	stmtNode
	Paths []string
}

func (n *ImportTableStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("IMPORT TABLE FROM ")
	for i, path := range n.Paths {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		ctx.WriteString(path)
	}
	return ctx.Err()
}

func (n *ImportTableStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	outer, ok := nodeAs[*ImportTableStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(outer)
}

// InstallPluginStmt is the INSTALL PLUGIN statement.
// See https://dev.mysql.com/doc/refman/8.0/en/install-plugin.html
type InstallPluginStmt struct {
	stmtNode
	Name   lex.Ident
	Soname string
}

func (n *InstallPluginStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("restore InstallPluginStmt: nil node")
	}
	if !n.Name.Present() {
		return errors.New("restore InstallPluginStmt: missing plugin name")
	}
	ctx.WriteKeyword("INSTALL PLUGIN ")
	ctx.WriteIdent(n.Name)
	ctx.WriteKeyword(" SONAME ")
	ctx.WriteString(n.Soname)
	return ctx.Err()
}

func (n *InstallPluginStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	// The plugin name is not a node: Accept must not emit visitor events for
	// the name value, only for the statement itself.
	// Outer role check first: the replacement must be the concrete type.
	stmt, isStmt := newNode.(*InstallPluginStmt)
	if !isStmt || stmt == nil {
		return newNode, false
	}
	if skipChildren {
		// Required role: a skipped replacement must keep the concrete type.
		return v.Leave(newNode)
	}
	updated, ok := v.Leave(stmt)
	if !ok {
		return stmt, false
	}
	if leaf, isLeaf := updated.(*InstallPluginStmt); !isLeaf || leaf == nil {
		return updated, false
	}
	return updated, true
}

// UninstallPluginStmt is the UNINSTALL PLUGIN statement.
// See https://dev.mysql.com/doc/refman/8.0/en/uninstall-plugin.html
type UninstallPluginStmt struct {
	stmtNode
	Name lex.Ident
}

func (n *UninstallPluginStmt) Restore(ctx *format.RestoreCtx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil {
		return errors.New("restore UninstallPluginStmt: nil node")
	}
	if !n.Name.Present() {
		return errors.New("restore UninstallPluginStmt: missing plugin name")
	}
	ctx.WriteKeyword("UNINSTALL PLUGIN ")
	ctx.WriteIdent(n.Name)
	return ctx.Err()
}

func (n *UninstallPluginStmt) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	newNode, skipChildren := v.Enter(n)
	// The plugin name is not a node: Accept must not emit visitor events for
	// the name value, only for the statement itself.
	// Outer role check first: the replacement must be the concrete type.
	stmt, isStmt := newNode.(*UninstallPluginStmt)
	if !isStmt || stmt == nil {
		return newNode, false
	}
	if skipChildren {
		// Required role: a skipped replacement must keep the concrete type.
		return v.Leave(newNode)
	}
	updated, ok := v.Leave(stmt)
	if !ok {
		return stmt, false
	}
	if leaf, isLeaf := updated.(*UninstallPluginStmt); !isLeaf || leaf == nil {
		return updated, false
	}
	return updated, true
}

// ComponentSetValue is a component variable assignment in INSTALL COMPONENT.
type ComponentSetValue struct {
	node
	Scope string
	Name  string
	Value ExprNode
}

func (n *ComponentSetValue) Restore(ctx *format.RestoreCtx) error {
	if n == nil || !nodePresent(n.Value) {
		return errors.New("restore ComponentSetValue: missing value")
	}
	if n.Scope != "" {
		ctx.WriteKeyword(n.Scope)
		ctx.WritePlain(" ")
	}
	ctx.WriteName(n.Name)
	ctx.WritePlain("=")
	if err := n.Value.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore ComponentSetValue.Value")
	}
	return ctx.Err()
}

func (n *ComponentSetValue) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*ComponentSetValue](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	expr, ok := acceptNode[ExprNode](v, n.Value)
	if !ok {
		return n, false
	}
	n.Value = expr
	return v.Leave(n)
}

// InstallComponentStmt is the INSTALL COMPONENT statement.
// See https://dev.mysql.com/doc/refman/8.0/en/install-component.html
type InstallComponentStmt struct {
	stmtNode
	URIs      []string
	SetValues []*ComponentSetValue
}

func (n *InstallComponentStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("INSTALL COMPONENT ")
	for i, uri := range n.URIs {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		ctx.WriteString(uri)
	}
	if len(n.SetValues) != 0 {
		ctx.WriteKeyword(" SET ")
		for i, value := range n.SetValues {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if err := value.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore InstallComponentStmt.SetValues[%d]", i)
			}
		}
	}
	return ctx.Err()
}

func (n *InstallComponentStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*InstallComponentStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	for i, value := range n.SetValues {
		updated, ok := acceptNode[*ComponentSetValue](v, value)
		if !ok {
			return n, false
		}
		n.SetValues[i] = updated
	}
	return v.Leave(n)
}

// UninstallComponentStmt is the UNINSTALL COMPONENT statement.
// See https://dev.mysql.com/doc/refman/8.0/en/uninstall-component.html
type UninstallComponentStmt struct {
	stmtNode
	URIs []string
}

func (n *UninstallComponentStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("UNINSTALL COMPONENT ")
	for i, uri := range n.URIs {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		ctx.WriteString(uri)
	}
	return ctx.Err()
}

func (n *UninstallComponentStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	outer, ok := nodeAs[*UninstallComponentStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(outer)
}

// CloneOptions is the optional part of CLONE INSTANCE.
type CloneOptions struct {
	DataDir string
	SSL     string
}

// CloneStmt is the CLONE statement.
// See https://dev.mysql.com/doc/refman/8.0/en/clone.html
type CloneStmt struct {
	stmtNode
	Local    bool
	DataDir  string
	User     *auth.UserIdentity
	Port     uint64
	Password string
	SSL      string
}

func (n *CloneStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("CLONE ")
	if n.Local {
		ctx.WriteKeyword("LOCAL DATA DIRECTORY ")
		ctx.WritePlain("= ")
		ctx.WriteString(n.DataDir)
		return ctx.Err()
	}
	ctx.WriteKeyword("INSTANCE FROM ")
	if err := restoreUserIdentity(ctx, n.User); err != nil {
		return errors.Annotate(err, "An error occurred while restore CloneStmt.User")
	}
	ctx.WritePlain(":")
	ctx.WritePlain(strconv.FormatUint(n.Port, 10))
	ctx.WriteKeyword(" IDENTIFIED BY ")
	ctx.WriteString(n.Password)
	if n.DataDir != "" {
		ctx.WriteKeyword(" DATA DIRECTORY ")
		ctx.WritePlain("= ")
		ctx.WriteString(n.DataDir)
	}
	if n.SSL != "" {
		ctx.WriteKeyword(" " + n.SSL)
	}
	return ctx.Err()
}

func (n *CloneStmt) Accept(v Visitor) (Node, bool) {
	newNode, _ := v.Enter(n)
	outer, ok := nodeAs[*CloneStmt](newNode)
	if !ok {
		return newNode, false
	}
	return v.Leave(outer)
}

// DiagnosticsItemName identifies the right-hand side of a GET DIAGNOSTICS
// assignment.
type DiagnosticsItemName uint8

const (
	DiagnosticsItemNumber DiagnosticsItemName = iota + 1
	DiagnosticsItemRowCount
	DiagnosticsItemClassOrigin
	DiagnosticsItemSubclassOrigin
	DiagnosticsItemReturnedSQLState
	DiagnosticsItemMessageText
	DiagnosticsItemMySQLErrno
	DiagnosticsItemConstraintCatalog
	DiagnosticsItemConstraintSchema
	DiagnosticsItemConstraintName
	DiagnosticsItemCatalogName
	DiagnosticsItemSchemaName
	DiagnosticsItemTableName
	DiagnosticsItemColumnName
	DiagnosticsItemCursorName
	DiagnosticsItemRowNumber
)

func (n DiagnosticsItemName) String() string {
	switch n {
	case DiagnosticsItemNumber:
		return "NUMBER"
	case DiagnosticsItemRowCount:
		return "ROW_COUNT"
	case DiagnosticsItemClassOrigin:
		return "CLASS_ORIGIN"
	case DiagnosticsItemSubclassOrigin:
		return "SUBCLASS_ORIGIN"
	case DiagnosticsItemReturnedSQLState:
		return "RETURNED_SQLSTATE"
	case DiagnosticsItemMessageText:
		return "MESSAGE_TEXT"
	case DiagnosticsItemMySQLErrno:
		return "MYSQL_ERRNO"
	case DiagnosticsItemConstraintCatalog:
		return "CONSTRAINT_CATALOG"
	case DiagnosticsItemConstraintSchema:
		return "CONSTRAINT_SCHEMA"
	case DiagnosticsItemConstraintName:
		return "CONSTRAINT_NAME"
	case DiagnosticsItemCatalogName:
		return "CATALOG_NAME"
	case DiagnosticsItemSchemaName:
		return "SCHEMA_NAME"
	case DiagnosticsItemTableName:
		return "TABLE_NAME"
	case DiagnosticsItemColumnName:
		return "COLUMN_NAME"
	case DiagnosticsItemCursorName:
		return "CURSOR_NAME"
	case DiagnosticsItemRowNumber:
		return "ROW_NUMBER"
	default:
		return ""
	}
}

// DiagnosticsItem is one GET DIAGNOSTICS target assignment.
type DiagnosticsItem struct {
	node
	Target *VariableTarget
	Name   DiagnosticsItemName
}

func (n *DiagnosticsItem) Restore(ctx *format.RestoreCtx) error {
	if n == nil || n.Target == nil {
		return errors.New("ast: diagnostics item has no target")
	}
	if err := n.Target.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore DiagnosticsItem.Target")
	}
	ctx.WritePlain(" = ")
	ctx.WriteKeyword(n.Name.String())
	return ctx.Err()
}

func (n *DiagnosticsItem) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*DiagnosticsItem](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	if n.Target != nil {
		target, ok := acceptNode[*VariableTarget](v, n.Target)
		if !ok {
			return n, false
		}
		n.Target = target
	}
	return v.Leave(n)
}

// GetDiagnosticsStmt is the GET DIAGNOSTICS statement.
// See https://dev.mysql.com/doc/refman/8.0/en/get-diagnostics.html
type GetDiagnosticsStmt struct {
	stmtNode
	Stacked   bool
	Condition ExprNode
	Items     []*DiagnosticsItem
}

func (n *GetDiagnosticsStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyword("GET ")
	if n.Stacked {
		ctx.WriteKeyword("STACKED")
	} else {
		ctx.WriteKeyword("CURRENT")
	}
	ctx.WriteKeyword(" DIAGNOSTICS ")
	if n.Condition != nil {
		if !nodePresent(n.Condition) {
			return errors.New("restore GetDiagnosticsStmt.Condition: nil expression")
		}
		ctx.WriteKeyword("CONDITION ")
		if err := n.Condition.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore GetDiagnosticsStmt.Condition")
		}
		ctx.WritePlain(" ")
	}
	for i, item := range n.Items {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := item.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore GetDiagnosticsStmt.Items[%d]", i)
		}
	}
	return ctx.Err()
}

func (n *GetDiagnosticsStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	outer, ok := nodeAs[*GetDiagnosticsStmt](newNode)
	if !ok {
		return newNode, false
	}
	if skipChildren {
		return v.Leave(outer)
	}
	n = outer
	if n.Condition != nil {
		expr, ok := acceptNode[ExprNode](v, n.Condition)
		if !ok {
			return n, false
		}
		n.Condition = expr
	}
	for i, item := range n.Items {
		updated, ok := acceptNode[*DiagnosticsItem](v, item)
		if !ok {
			return n, false
		}
		n.Items[i] = updated
	}
	return v.Leave(n)
}

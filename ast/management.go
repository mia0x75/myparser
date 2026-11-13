package ast

import (
	"io"
	"strconv"
	"strings"

	"github.com/mia0x75/myparser/auth"
	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
)

// ChangeMasterStmt changes a replication connection's source and configuration.
type ChangeMasterStmt struct {
	stmtNode

	Replica           bool
	Options           []*Assignment
	ConnectionName    string
	HasConnectionName bool
	Channel           string
	HasChannel        bool
}

// Restore implements Node interface.
func (n *ChangeMasterStmt) Restore(ctx *format.RestoreCtx) error {
	if n.Replica {
		ctx.WriteKeyWord("CHANGE REPLICATION SOURCE")
	} else {
		ctx.WriteKeyWord("CHANGE MASTER")
	}
	if n.HasConnectionName || n.ConnectionName != "" {
		ctx.WritePlain(" ")
		ctx.WriteString(n.ConnectionName)
	}
	ctx.WriteKeyWord(" TO ")
	for i, opt := range n.Options {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := restoreReplicationOption(ctx, opt); err != nil {
			return errors.Annotatef(err, "An error occurred while restore ChangeMasterStmt.Options[%d]", i)
		}
	}
	if n.HasChannel || n.Channel != "" {
		ctx.WriteKeyWord(" FOR CHANNEL ")
		ctx.WriteString(n.Channel)
	}
	return nil
}

// Accept implements Node Accept interface.
func (n *ChangeMasterStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ChangeMasterStmt)
	for i, opt := range n.Options {
		node, ok := opt.Accept(v)
		if !ok {
			return n, false
		}
		n.Options[i] = node.(*Assignment)
	}
	return v.Leave(n)
}

// ReplicaSuffix holds the optional clauses while parsing START/STOP replication.
type ReplicaSuffix struct {
	Threads            []string
	UntilOptions       []*Assignment
	UntilAfterGaps     bool
	ConnectionOptions  []*Assignment
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
	UntilOptions       []*Assignment
	UntilAfterGaps     bool
	ConnectionOptions  []*Assignment
	ConnectionName     string
	HasConnectionName  bool
	Channel            string
	HasChannel         bool
	ChannelBeforeUntil bool
}

// Restore implements Node interface.
func (n *ReplicaControlStmt) Restore(ctx *format.RestoreCtx) error {
	if n.Start {
		ctx.WriteKeyWord("START ")
	} else {
		ctx.WriteKeyWord("STOP ")
	}
	if n.All && n.Replica {
		ctx.WriteKeyWord("ALL REPLICAS")
	} else if n.All {
		ctx.WriteKeyWord("ALL SLAVES")
	} else if n.Replica {
		ctx.WriteKeyWord("REPLICA")
	} else {
		ctx.WriteKeyWord("SLAVE")
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
		ctx.WriteKeyWord(strings.ToUpper(thread))
	}
	if n.ChannelBeforeUntil {
		n.restoreChannel(ctx)
	}
	if n.UntilAfterGaps {
		ctx.WriteKeyWord(" UNTIL SQL_AFTER_MTS_GAPS")
	} else if len(n.UntilOptions) != 0 {
		ctx.WriteKeyWord(" UNTIL ")
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
	return nil
}

func (n *ReplicaControlStmt) restoreChannel(ctx *format.RestoreCtx) {
	if n.HasChannel || n.Channel != "" {
		ctx.WriteKeyWord(" FOR CHANNEL ")
		ctx.WriteString(n.Channel)
	}
}

// Accept implements Node Accept interface.
func (n *ReplicaControlStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ReplicaControlStmt)
	for i, opt := range n.UntilOptions {
		node, ok := opt.Accept(v)
		if !ok {
			return n, false
		}
		n.UntilOptions[i] = node.(*Assignment)
	}
	for i, opt := range n.ConnectionOptions {
		node, ok := opt.Accept(v)
		if !ok {
			return n, false
		}
		n.ConnectionOptions[i] = node.(*Assignment)
	}
	return v.Leave(n)
}

// ReplicationOptionValue represents a bare keyword, parenthesized ID list, or account.
// Exactly one field is set; an empty, non-nil Values slice represents an empty list.
// Scalar literals and DEFAULT use ValueExpr and DefaultExpr instead.
type ReplicationOptionValue struct {
	exprNode

	Keyword string
	Values  []ExprNode
	User    *auth.UserIdentity
}

// Restore implements Node interface.
func (n *ReplicationOptionValue) Restore(ctx *format.RestoreCtx) error {
	switch {
	case n.Keyword != "":
		ctx.WriteKeyWord(n.Keyword)
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
	return nil
}

// Format implements ExprNode interface.
func (n *ReplicationOptionValue) Format(w io.Writer) {
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, w)
	switch {
	case n.Keyword != "":
		ctx.WriteKeyWord(n.Keyword)
	case n.Values != nil:
		ctx.WritePlain("(")
		for i, value := range n.Values {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			value.Format(w)
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
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ReplicationOptionValue)
	for i, value := range n.Values {
		node, ok := value.Accept(v)
		if !ok {
			return n, false
		}
		n.Values[i] = node.(ExprNode)
	}
	return v.Leave(n)
}

func restoreReplicationOption(ctx *format.RestoreCtx, option *Assignment) error {
	if option == nil || option.Column == nil || option.Expr == nil {
		return errors.New("Incomplete replication option")
	}
	ctx.WriteKeyWord(option.Column.Name.O)
	ctx.WritePlain("=")
	return option.Expr.Restore(ctx)
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
	PersistName string
}

func (n *ResetStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("RESET ")
	if len(n.Options) != 0 {
		for i, option := range n.Options {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if err := option.restoreOption(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore ResetStmt.Options[%d]", i)
			}
		}
		return nil
	}
	return n.restoreOption(ctx)
}

func (n *ResetStmt) restoreOption(ctx *format.RestoreCtx) error {
	switch n.Tp {
	case ResetMaster:
		ctx.WriteKeyWord("MASTER")
		if len(n.Args) != 0 {
			ctx.WriteKeyWord(" TO ")
			if err := restoreExprNodes(ctx, n.Args); err != nil {
				return err
			}
		}
	case ResetSlave:
		ctx.WriteKeyWord("SLAVE")
		if n.All {
			ctx.WriteKeyWord(" ALL")
		}
	case ResetReplica:
		ctx.WriteKeyWord("REPLICA")
		if n.All {
			ctx.WriteKeyWord(" ALL")
		}
	case ResetPersist:
		ctx.WriteKeyWord("PERSIST")
		if n.IfExists {
			ctx.WriteKeyWord(" IF EXISTS")
		}
		if n.PersistName != "" {
			ctx.WritePlain(" ")
			ctx.WritePlain(n.PersistName)
		}
	case ResetBinaryLogsAndGTIDs:
		ctx.WriteKeyWord("BINARY LOGS AND GTIDS")
		if len(n.Args) != 0 {
			ctx.WriteKeyWord(" TO ")
			if err := restoreExprNodes(ctx, n.Args); err != nil {
				return err
			}
		}
	case ResetQueryCache:
		ctx.WriteKeyWord("QUERY CACHE")
	}
	if n.HasChannel || n.Channel != "" {
		ctx.WriteKeyWord(" FOR CHANNEL ")
		ctx.WriteString(n.Channel)
	}
	return nil
}

func (n *ResetStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ResetStmt)
	for i, option := range n.Options {
		node, ok := option.Accept(v)
		if !ok {
			return n, false
		}
		n.Options[i] = node.(*ResetStmt)
	}
	for i, arg := range n.Args {
		node, ok := arg.Accept(v)
		if !ok {
			return n, false
		}
		n.Args[i] = node.(ExprNode)
	}
	return v.Leave(n)
}

type ShutdownStmt struct {
	stmtNode

	WaitForAllSlaves bool
}

func (n *ShutdownStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("SHUTDOWN")
	if n.WaitForAllSlaves {
		ctx.WriteKeyWord(" WAIT FOR ALL SLAVES")
	}
	return nil
}

func (n *ShutdownStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(n)
}

type HandlerStmtType int

const (
	HandlerOpen HandlerStmtType = iota
	HandlerRead
	HandlerClose
)

type HandlerStmt struct {
	stmtNode

	Tp        HandlerStmtType
	Table     *TableName
	Alias     string
	HasIndex  bool
	Index     string
	Op        string
	Values    []ExprNode
	Direction string
	Where     ExprNode
	Limit     *Limit
}

func (n *HandlerStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("HANDLER ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore HandlerStmt.Table")
	}
	switch n.Tp {
	case HandlerOpen:
		ctx.WriteKeyWord(" OPEN")
		if n.Alias != "" {
			ctx.WriteKeyWord(" AS ")
			ctx.WriteName(n.Alias)
		}
	case HandlerClose:
		ctx.WriteKeyWord(" CLOSE")
	case HandlerRead:
		ctx.WriteKeyWord(" READ")
		if n.HasIndex {
			ctx.WritePlain(" ")
			ctx.WriteName(n.Index)
		}
		if n.Op != "" {
			ctx.WritePlain(" ")
			ctx.WritePlain(n.Op)
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
		} else if n.Direction != "" {
			ctx.WritePlain(" ")
			ctx.WriteKeyWord(strings.ToUpper(n.Direction))
		}
		if n.Where != nil {
			ctx.WriteKeyWord(" WHERE ")
			if err := n.Where.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore HandlerStmt.Where")
			}
		}
	}
	if n.Limit != nil {
		ctx.WritePlain(" ")
		if err := n.Limit.Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore HandlerStmt.Limit")
		}
	}
	return nil
}

func (n *HandlerStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*HandlerStmt)
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	n.Table = node.(*TableName)
	for i, value := range n.Values {
		node, ok = value.Accept(v)
		if !ok {
			return n, false
		}
		n.Values[i] = node.(ExprNode)
	}
	if n.Where != nil {
		node, ok = n.Where.Accept(v)
		if !ok {
			return n, false
		}
		n.Where = node.(ExprNode)
	}
	if n.Limit != nil {
		node, ok = n.Limit.Accept(v)
		if !ok {
			return n, false
		}
		n.Limit = node.(*Limit)
	}
	return v.Leave(n)
}

type CacheIndexTargetType int

const (
	CacheIndexDefaultTarget CacheIndexTargetType = iota
	CacheIndexIdentifierTarget
	CacheIndexStringTarget
)

type CacheIndexStmt struct {
	stmtNode

	Tables []*TableName
	// TableIndexes is aligned with Tables. A nil entry omits INDEX; an empty
	// non-nil entry preserves an explicit INDEX ().
	TableIndexes [][]string
	PartitionAll bool
	Target       CacheIndexTargetType
	Name         string
}

func (n *CacheIndexStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("CACHE INDEX ")
	for i, table := range n.Tables {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := table.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore CacheIndexStmt.Tables[%d]", i)
		}
		if n.PartitionAll {
			ctx.WriteKeyWord(" PARTITION (ALL)")
		}
		if i < len(n.TableIndexes) && n.TableIndexes[i] != nil {
			restoreCacheIndexNames(ctx, n.TableIndexes[i])
		}
	}
	ctx.WriteKeyWord(" IN ")
	switch n.Target {
	case CacheIndexDefaultTarget:
		ctx.WriteKeyWord("DEFAULT")
	case CacheIndexIdentifierTarget:
		ctx.WriteName(n.Name)
	case CacheIndexStringTarget:
		ctx.WriteString(n.Name)
	}
	return nil
}

func (n *CacheIndexStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*CacheIndexStmt)
	for i, table := range n.Tables {
		node, ok := table.Accept(v)
		if !ok {
			return n, false
		}
		n.Tables[i] = node.(*TableName)
	}
	return v.Leave(n)
}

type LoadIndexStmt struct {
	stmtNode

	Tables []*LoadIndexTable
}

func (n *LoadIndexStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("LOAD INDEX INTO CACHE ")
	for i, table := range n.Tables {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := table.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore LoadIndexStmt.Tables[%d]", i)
		}
	}
	return nil
}

func (n *LoadIndexStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*LoadIndexStmt)
	for i, table := range n.Tables {
		node, ok := table.Accept(v)
		if !ok {
			return n, false
		}
		n.Tables[i] = node.(*LoadIndexTable)
	}
	return v.Leave(n)
}

type LoadIndexTable struct {
	node

	Table        *TableName
	IndexNames   []string
	PartitionAll bool
	IgnoreLeaves bool
}

func (n *LoadIndexTable) Restore(ctx *format.RestoreCtx) error {
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore LoadIndexTable.Table")
	}
	if n.PartitionAll {
		ctx.WriteKeyWord(" PARTITION (ALL)")
	}
	if n.IndexNames != nil {
		restoreCacheIndexNames(ctx, n.IndexNames)
	}
	if n.IgnoreLeaves {
		ctx.WriteKeyWord(" IGNORE LEAVES")
	}
	return nil
}

func restoreCacheIndexNames(ctx *format.RestoreCtx, names []string) {
	ctx.WriteKeyWord(" INDEX (")
	for i, name := range names {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		ctx.WriteName(name)
	}
	ctx.WritePlain(")")
}

func (n *LoadIndexTable) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*LoadIndexTable)
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	n.Table = node.(*TableName)
	return v.Leave(n)
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
	ctx.WriteKeyWord("LOAD XML ")
	if n.Priority != "" {
		ctx.WriteKeyWord(n.Priority + " ")
	}
	if n.IsLocal {
		ctx.WriteKeyWord("LOCAL ")
	}
	ctx.WriteKeyWord("INFILE ")
	ctx.WriteString(n.Path)
	switch n.OnDuplicate {
	case OnDuplicateKeyHandlingReplace:
		ctx.WriteKeyWord(" REPLACE")
	case OnDuplicateKeyHandlingIgnore:
		ctx.WriteKeyWord(" IGNORE")
	}
	ctx.WriteKeyWord(" INTO TABLE ")
	if err := n.Table.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore LoadXMLStmt.Table")
	}
	if n.Charset != "" {
		ctx.WriteKeyWord(" CHARACTER SET ")
		ctx.WriteName(n.Charset)
	}
	if n.HasRowTag {
		ctx.WriteKeyWord(" ROWS IDENTIFIED BY ")
		ctx.WriteString(n.RowTag)
	}
	if n.HasIgnoreLines || n.IgnoreLines != 0 {
		ctx.WriteKeyWord(" IGNORE ")
		ctx.WritePlainf("%d", n.IgnoreLines)
		if n.IgnoreRows {
			ctx.WriteKeyWord(" ROWS")
		} else {
			ctx.WriteKeyWord(" LINES")
		}
	}
	if len(n.ColumnsAndUserVars) != 0 {
		ctx.WritePlain(" (")
		for i, c := range n.ColumnsAndUserVars {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if c.ColumnName != nil {
				if err := c.ColumnName.Restore(ctx); err != nil {
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
		ctx.WriteKeyWord(" SET")
		for i, assign := range n.ColumnAssignments {
			if i != 0 {
				ctx.WritePlain(",")
			}
			ctx.WritePlain(" ")
			if err := assign.Restore(ctx); err != nil {
				return errors.Annotate(err, "An error occurred while restore LoadXMLStmt.ColumnAssignments")
			}
		}
	}
	return nil
}

func (n *LoadXMLStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*LoadXMLStmt)
	node, ok := n.Table.Accept(v)
	if !ok {
		return n, false
	}
	n.Table = node.(*TableName)
	for _, column := range n.ColumnsAndUserVars {
		if column.ColumnName != nil {
			node, ok = column.ColumnName.Accept(v)
			if !ok {
				return n, false
			}
			column.ColumnName = node.(*ColumnName)
		}
		if column.UserVar != nil {
			node, ok = column.UserVar.Accept(v)
			if !ok {
				return n, false
			}
			column.UserVar = node.(*VariableExpr)
		}
	}
	for i, assignment := range n.ColumnAssignments {
		node, ok = assignment.Accept(v)
		if !ok {
			return n, false
		}
		n.ColumnAssignments[i] = node.(*Assignment)
	}
	return v.Leave(n)
}

type RenameUserStmt struct {
	stmtNode

	Pairs []*RenameUserPair
}

func (n *RenameUserStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("RENAME USER ")
	for i, pair := range n.Pairs {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := pair.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore RenameUserStmt.Pairs[%d]", i)
		}
	}
	return nil
}

func (n *RenameUserStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*RenameUserStmt)
	for i, pair := range n.Pairs {
		node, ok := pair.Accept(v)
		if !ok {
			return n, false
		}
		n.Pairs[i] = node.(*RenameUserPair)
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
	ctx.WriteKeyWord(" TO ")
	if err := restoreUserIdentity(ctx, n.To); err != nil {
		return errors.Annotate(err, "An error occurred while restore RenameUserPair.To")
	}
	return nil
}

func (n *RenameUserPair) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(n)
}

func restoreUserIdentity(ctx *format.RestoreCtx, user *auth.UserIdentity) error {
	if user == nil {
		return errors.New("An error occurred while restore user identity")
	}
	if user.CurrentUser {
		ctx.WriteKeyWord("CURRENT_USER")
		return nil
	}
	ctx.WriteString(user.Username)
	ctx.WritePlain("@")
	ctx.WriteString(user.Hostname)
	return nil
}

func restoreExprNodes(ctx *format.RestoreCtx, nodes []ExprNode) error {
	for i, node := range nodes {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		if err := node.Restore(ctx); err != nil {
			return errors.Annotatef(err, "An error occurred while restore expression[%d]", i)
		}
	}
	return nil
}

// ImportTableStmt is the IMPORT TABLE statement.
// See https://dev.mysql.com/doc/refman/8.0/en/import-table.html
type ImportTableStmt struct {
	stmtNode
	Paths []string
}

func (n *ImportTableStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("IMPORT TABLE FROM ")
	for i, path := range n.Paths {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		ctx.WriteString(path)
	}
	return nil
}

func (n *ImportTableStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(newNode)
}

// InstallPluginStmt is the INSTALL PLUGIN statement.
// See https://dev.mysql.com/doc/refman/8.0/en/install-plugin.html
type InstallPluginStmt struct {
	stmtNode
	Name   string
	Soname string
}

func (n *InstallPluginStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("INSTALL PLUGIN ")
	ctx.WriteName(n.Name)
	ctx.WriteKeyWord(" SONAME ")
	ctx.WriteString(n.Soname)
	return nil
}

func (n *InstallPluginStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(newNode)
}

// UninstallPluginStmt is the UNINSTALL PLUGIN statement.
// See https://dev.mysql.com/doc/refman/8.0/en/uninstall-plugin.html
type UninstallPluginStmt struct {
	stmtNode
	Name string
}

func (n *UninstallPluginStmt) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteKeyWord("UNINSTALL PLUGIN ")
	ctx.WriteName(n.Name)
	return nil
}

func (n *UninstallPluginStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(newNode)
}

// ComponentSetValue is a component variable assignment in INSTALL COMPONENT.
type ComponentSetValue struct {
	node
	Scope string
	Name  string
	Value ExprNode
}

func (n *ComponentSetValue) Restore(ctx *format.RestoreCtx) error {
	if n.Scope != "" {
		ctx.WriteKeyWord(n.Scope)
		ctx.WritePlain(" ")
	}
	ctx.WriteName(n.Name)
	ctx.WritePlain("=")
	if err := n.Value.Restore(ctx); err != nil {
		return errors.Annotate(err, "An error occurred while restore ComponentSetValue.Value")
	}
	return nil
}

func (n *ComponentSetValue) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*ComponentSetValue)
	node, ok := n.Value.Accept(v)
	if !ok {
		return n, false
	}
	n.Value = node.(ExprNode)
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
	ctx.WriteKeyWord("INSTALL COMPONENT ")
	for i, uri := range n.URIs {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		ctx.WriteString(uri)
	}
	if len(n.SetValues) != 0 {
		ctx.WriteKeyWord(" SET ")
		for i, value := range n.SetValues {
			if i != 0 {
				ctx.WritePlain(", ")
			}
			if err := value.Restore(ctx); err != nil {
				return errors.Annotatef(err, "An error occurred while restore InstallComponentStmt.SetValues[%d]", i)
			}
		}
	}
	return nil
}

func (n *InstallComponentStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*InstallComponentStmt)
	for i, value := range n.SetValues {
		node, ok := value.Accept(v)
		if !ok {
			return n, false
		}
		n.SetValues[i] = node.(*ComponentSetValue)
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
	ctx.WriteKeyWord("UNINSTALL COMPONENT ")
	for i, uri := range n.URIs {
		if i != 0 {
			ctx.WritePlain(", ")
		}
		ctx.WriteString(uri)
	}
	return nil
}

func (n *UninstallComponentStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(newNode)
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
	ctx.WriteKeyWord("CLONE ")
	if n.Local {
		ctx.WriteKeyWord("LOCAL DATA DIRECTORY ")
		ctx.WritePlain("= ")
		ctx.WriteString(n.DataDir)
		return nil
	}
	ctx.WriteKeyWord("INSTANCE FROM ")
	if err := restoreUserIdentity(ctx, n.User); err != nil {
		return errors.Annotate(err, "An error occurred while restore CloneStmt.User")
	}
	ctx.WritePlain(":")
	ctx.WritePlain(strconv.FormatUint(n.Port, 10))
	ctx.WriteKeyWord(" IDENTIFIED BY ")
	ctx.WriteString(n.Password)
	if n.DataDir != "" {
		ctx.WriteKeyWord(" DATA DIRECTORY ")
		ctx.WritePlain("= ")
		ctx.WriteString(n.DataDir)
	}
	if n.SSL != "" {
		ctx.WriteKeyWord(" " + n.SSL)
	}
	return nil
}

func (n *CloneStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(newNode)
}

// DiagnosticsItem is one GET DIAGNOSTICS target assignment.
type DiagnosticsItem struct {
	node
	Target    string
	IsUserVar bool
	Name      string
}

func (n *DiagnosticsItem) Restore(ctx *format.RestoreCtx) error {
	if n.IsUserVar {
		ctx.WritePlain("@")
	}
	ctx.WriteName(n.Target)
	ctx.WritePlain(" = ")
	ctx.WriteKeyWord(n.Name)
	return nil
}

func (n *DiagnosticsItem) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	return v.Leave(newNode)
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
	ctx.WriteKeyWord("GET ")
	if n.Stacked {
		ctx.WriteKeyWord("STACKED")
	} else {
		ctx.WriteKeyWord("CURRENT")
	}
	ctx.WriteKeyWord(" DIAGNOSTICS ")
	if n.Condition != nil {
		ctx.WriteKeyWord("CONDITION ")
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
	return nil
}

func (n *GetDiagnosticsStmt) Accept(v Visitor) (Node, bool) {
	newNode, skipChildren := v.Enter(n)
	if skipChildren {
		return v.Leave(newNode)
	}
	n = newNode.(*GetDiagnosticsStmt)
	if n.Condition != nil {
		node, ok := n.Condition.Accept(v)
		if !ok {
			return n, false
		}
		n.Condition = node.(ExprNode)
	}
	for i, item := range n.Items {
		node, ok := item.Accept(v)
		if !ok {
			return n, false
		}
		n.Items[i] = node.(*DiagnosticsItem)
	}
	return v.Leave(n)
}

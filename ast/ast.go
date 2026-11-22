// Package ast is the abstract syntax tree parsed from a SQL statement.
// It can be analysed and transformed by optimizer.
package ast

import (
	"reflect"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
)

// Node is the basic element of the AST.
// Interfaces embed Node should have 'Node' name suffix.
type Node = pkast.Node

func nodePresent(node Node) bool {
	if node == nil {
		return false
	}
	value := reflect.ValueOf(node)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return !value.IsNil()
	default:
		return true
	}
}

func nodeAs[T Node](node Node) (T, bool) {
	var zero T
	if !nodePresent(node) {
		return zero, false
	}
	value, ok := node.(T)
	if !ok || !nodePresent(value) {
		return zero, false
	}
	return value, true
}

func acceptNode[T Node](visitor Visitor, node Node) (T, bool) {
	var zero T
	if !nodePresent(node) {
		return zero, false
	}
	updated, ok := node.Accept(visitor)
	if !ok {
		return zero, false
	}
	return nodeAs[T](updated)
}

// Flags indicates whether an expression contains certain types of expression.
const (
	FlagConstant       uint64 = 0
	FlagHasParamMarker uint64 = 1 << iota
	FlagHasFunc
	FlagHasReference
	FlagHasAggregateFunc
	FlagHasSubquery
	FlagHasVariable
	FlagHasDefault
	FlagPreEvaluated
	FlagHasWindowFunc
)

// ExprNode is a node that can be evaluated.
// Name of implementations should have 'Expr' suffix.
type ExprNode = pkast.ExprNode

// OptBinary is used for parser.
type OptBinary struct {
	IsBinary bool
	Charset  lex.Ident
}

// FuncNode represents function call expression node.
type FuncNode interface {
	ExprNode
	functionExpression()
}

// StmtNode represents statement node.
// Name of implementations should have 'Stmt' suffix.
type StmtNode = pkast.StmtNode

// DDLNode represents DDL statement node.
type DDLNode = pkast.DDLNode

// DMLNode represents DML statement node.
type DMLNode = pkast.DMLNode

// ResultSetNode represents a Node that returns a result set.
// Implementations include SelectStmt, SubqueryExpr, TableSource, TableName and JoinExpr.
type ResultSetNode = pkast.ResultSetNode

// QueryNode classifies a statement that produces a result set (Stmt + ResultSet).
type QueryNode = pkast.QueryNode

// SensitiveStmtNode overloads StmtNode and provides a SecureText method.
type SensitiveStmtNode interface {
	StmtNode
	// SecureText is different from Text that it hide password information.
	SecureText() string
}

// Visitor visits a Node.
type Visitor = pkast.Visitor

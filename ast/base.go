package ast

import (
	pkast "github.com/mia0x75/parserkit/ast"
)

// node shares the common syntax data; outer nodes implement Accept and Restore.
// Node implementations should embed it in.
type node = pkast.BaseNode

// stmtNode shares the common statement marker; outer nodes implement Accept and Restore.
// Statement implementations should embed it in.
type stmtNode = pkast.BaseStmt

// ddlNode shares the common DDL marker; outer nodes implement Accept and Restore.
// DDL implementations should embed it in.
type ddlNode = pkast.BaseDDL

// dmlNode shares the common DML marker; outer nodes implement Accept and Restore.
// DML implementations should embed it in.
type dmlNode = pkast.BaseDML

// resultSetNode shares the common result-set marker; outer nodes implement Accept and Restore.
// Result-set implementations should embed it in.
type resultSetNode = pkast.BaseResultSet

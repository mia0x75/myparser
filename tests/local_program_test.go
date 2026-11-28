package parser_test

import (
	"math"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/model"
	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"
	"github.com/mia0x75/parserkit/loc"
)

// TestLocalPartitionASTAndVisitors covers PARTITION BY clauses on CREATE and
// ALTER, partition option values, and the partition-count extremes.
func TestLocalPartitionASTAndVisitors(t *testing.T) {
	for _, prefix := range []string{"create table t(a int)", "alter table t"} {
		t.Run(prefix, func(t *testing.T) {
			stmt := mustParseOne(t, prefix+" partition by range(a) partitions 2 subpartition by key(a) subpartitions 3 (partition p values less than (10,20) engine=InnoDB tablespace ts comment '')")
			var part *ast.PartitionOptions
			switch x := stmt.(type) {
			case *ast.CreateTableStmt:
				part = x.Partition
			case *ast.AlterTableStmt:
				part = x.Specs[0].Partition
			}
			if part == nil || part.Tp != model.PartitionTypeRange || part.Num != 2 || !part.HasNum ||
				part.Sub == nil || part.Sub.Tp != model.PartitionTypeKey || part.Sub.Num != 3 || !part.Sub.IsSubPartition ||
				len(part.Sub.ColumnNames) != 1 || len(part.Definitions) != 1 {
				t.Fatalf("partition clauses were lost: %#v", part)
			}
			options := part.Definitions[0].Options
			if len(options) != 3 || options[0].Tp != ast.TableOptionEngine || options[0].StrValue != "InnoDB" ||
				options[1].Tp != ast.TableOptionTablespace || options[1].StrValue != "ts" ||
				options[2].Tp != ast.TableOptionComment || options[2].StrValue != "" {
				t.Fatalf("partition options were lost: %#v", options)
			}
			parts, values := 0, 0
			_, ok := stmt.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
				switch x := n.(type) {
				case *ast.PartitionOptions:
					parts++
					copy := *x
					copy.Num = 9
					return &copy, false
				case *ast.LiteralExpr:
					values++
					return replaceLiteral(x, "8"), false
				}
				return n, false
			}})
			restored := restoreStmt(t, stmt)
			if !ok || parts != 2 || values != 2 || !strings.Contains(restored, "PARTITIONS 9 SUBPARTITION") ||
				!strings.Contains(restored, "SUBPARTITIONS 9") || !strings.Contains(restored, "LESS THAN (8, 8)") {
				t.Fatalf("partition visitor replacement was lost: ok=%v parts=%d values=%d sql=%s", ok, parts, values, restored)
			}
			assertVisitorControls(t, stmt, func(n ast.Node) bool { _, ok := n.(*ast.PartitionOptions); return ok })
			assertVisitorControls(t, part, func(n ast.Node) bool { _, ok := n.(*ast.PartitionDefinition); return ok })
			assertVisitorControls(t, part.Sub, func(n ast.Node) bool { _, ok := n.(*ast.PartitionOptions); return ok })
			assertVisitorControls(t, part.Definitions[0], isValueOrLiteral)
		})
	}
	for _, action := range []string{"add partition", "reorganize partition p into"} {
		t.Run(action, func(t *testing.T) {
			stmt := mustParseOne(t, "alter table t "+action+" (partition q values less than (1))").(*ast.AlterTableStmt)
			_, ok := stmt.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
				if def, ok := n.(*ast.PartitionDefinition); ok {
					copy := *def
					copy.Name = lex.Ident{Name: "renamed", Span: loc.InvalidSpan()}
					return &copy, false
				}
				return n, false
			}})
			if !ok || stmt.Specs[0].PartDefinitions[0].Name.Name != "renamed" {
				t.Fatal("ALTER partition definition replacement was lost")
			}
			assertVisitorControls(t, stmt, func(n ast.Node) bool { _, ok := n.(*ast.PartitionDefinition); return ok })
		})
	}
	for _, num := range []uint64{0, math.MaxUint64} {
		sql := "create table t(a int) partition by key(a) partitions "
		if num == 0 {
			sql += "0"
		} else {
			sql += "18446744073709551615"
		}
		stmt := mustParseOne(t, sql).(*ast.CreateTableStmt)
		if !stmt.Partition.HasNum || stmt.Partition.Num != num {
			t.Fatalf("partition count was lost: %#v", stmt.Partition)
		}
		localRunRestoreTest(t, sql, restoreStmt(t, stmt), true)
	}
}

// TestLocalRowReturnAndIndexVisitors covers ROW TYPE OF return types on
// CREATE FUNCTION and expression index key parts.
func TestLocalRowReturnAndIndexVisitors(t *testing.T) {
	fn := mustParseOne(t, "create function f() returns row type of db.t return 1").(*ast.CreateFunctionStmt)
	if fn.ReturnType != nil || fn.ReturnsRowType == nil ||
		len(fn.ReturnsRowType.Name.Parts) != 2 ||
		fn.ReturnsRowType.Name.Parts[0].Name != "db" ||
		fn.ReturnsRowType.Name.Parts[len(fn.ReturnsRowType.Name.Parts)-1].Name != "t" {
		t.Fatalf("ROW TYPE OF return was lost: %#v", fn)
	}
	_, ok := fn.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
		if n == fn.ReturnsRowType {
			// Mirror the production syntaxTableName: a BaseTableRef with the
			// name and the name's span, so the replaced node stays span-identical.
			name := ast.QualifiedName{Parts: []lex.Ident{
				{Name: "db", Span: loc.InvalidSpan()},
				{Name: "u", Span: loc.InvalidSpan()},
			}}
			name.Span = loc.InvalidSpan()
			return &ast.TableName{BaseTableRef: pkast.BaseTableRef{Name: name}}, false
		}
		return n, false
	}})
	if !ok || fn.ReturnsRowType.Name.Parts[len(fn.ReturnsRowType.Name.Parts)-1].Name != "u" ||
		!strings.Contains(restoreStmt(t, fn), "ROW TYPE OF `db`.`u`") {
		t.Fatal("ROW TYPE OF visitor replacement was lost")
	}
	assertVisitorControls(t, fn, func(n ast.Node) bool { return n == fn.ReturnsRowType })
	for _, src := range []string{
		"create table t(a int,key k((a+1)))",
		"create unique index k on t((a+1))",
		"alter table t add key k((a+1))",
	} {
		t.Run(src, func(t *testing.T) {
			stmt := mustParseOne(t, src)
			keys, values := 0, 0
			_, ok := stmt.Accept(syntaxVisitor{enter: func(n ast.Node) (ast.Node, bool) {
				switch x := n.(type) {
				case *ast.IndexKeyPart:
					keys++
					if x.Expr == nil || len(x.Column.Parts) != 0 {
						t.Fatalf("expression index key part was lost: %#v", x)
					}
				case *ast.LiteralExpr:
					values++
					return replaceLiteral(x, "9"), false
				}
				return n, false
			}})
			if !ok || keys != 1 || values != 1 || !strings.Contains(restoreStmt(t, stmt), "((`a`+9))") {
				t.Fatalf("index visitor: ok=%v keys=%d values=%d", ok, keys, values)
			}
			assertVisitorControls(t, stmt, isValueOrLiteral)
		})
	}
}

package parser_test

import (
	"testing"

	"github.com/mia0x75/myparser/parser"
)

func TestParseAlterIdentityDialectBoundary(t *testing.T) {
	for _, column := range []string{"COLUMN ", ""} {
		for _, mode := range []string{"ALWAYS", "BY DEFAULT"} {
			for _, options := range []string{"", " ()", " (START WITH 3 INCREMENT BY 2)"} {
				sql := "ALTER TABLE alignment_t ALTER " + column + "id ADD GENERATED " + mode + " AS IDENTITY" + options
				t.Run(sql, func(t *testing.T) {
					if _, err := parser.New().ParseOneStmt(sql); err == nil {
						t.Fatal("accepted PostgreSQL identity syntax")
					}
				})
			}
		}
	}
}

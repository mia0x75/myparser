package parser_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/mia0x75/myparser/parser"
)

func TestParse(t *testing.T) {
	stmts, err := parser.New().Parse("SELECT 1")
	if err != nil || len(stmts) != 1 {
		t.Fatalf("Parse returned %d statements, error %v", len(stmts), err)
	}
}

func TestParseResultNotOverwrittenByNextCall(t *testing.T) {
	shared := parser.New()
	first, err := shared.Parse("SELECT 1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := shared.Parse("SELECT 2")
	if err != nil {
		t.Fatal(err)
	}
	if got := first[0].Text(); got != "SELECT 1" {
		t.Fatalf("first result = %q, want %q", got, "SELECT 1")
	}
	if got := second[0].Text(); got != "SELECT 2" {
		t.Fatalf("second result = %q, want %q", got, "SELECT 2")
	}
}

func TestParserConcurrentUse(t *testing.T) {
	const workers, iterations = 8, 20
	shared := parser.New()
	var group sync.WaitGroup
	group.Add(workers)
	for worker := range workers {
		go func(worker int) {
			defer group.Done()
			for iteration := range iterations {
				sql := fmt.Sprintf("SELECT %d FROM t%d", worker, iteration)
				stmts, err := shared.Parse(sql)
				if err != nil {
					t.Errorf("parse %q: %v", sql, err)
					return
				}
				if len(stmts) != 1 {
					t.Errorf("parse %q: got %d statements", sql, len(stmts))
					return
				}
				if got := stmts[0].Text(); got != sql {
					t.Errorf("parse %q: got statement %q", sql, got)
					return
				}
			}
		}(worker)
	}
	group.Wait()
}

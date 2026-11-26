package parser

import "testing"

func BenchmarkFunctionNames(b *testing.B) {
	for _, tc := range []struct{ name, sql string }{
		{"ordinary", "SELECT mock_func(1)"},
		{"quoted", "SELECT `mock.func`(1)"},
		{"aggregate", "SELECT SUM(x), COUNT(*) FROM mock_table"},
		{"window", "SELECT ROW_NUMBER() OVER ()"},
		{"many", "SELECT ABS(1), ABS(2), ABS(3), ABS(4), ABS(5), ABS(6), ABS(7), ABS(8)"},
	} {
		b.Run(tc.name, func(b *testing.B) {
			p := New()
			p.EnableWindowFunc(true)
			b.ReportAllocs()
			for b.Loop() {
				if _, err := p.ParseOneStmt(tc.sql); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

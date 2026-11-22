package ast

import (
	"testing"

	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/parserkit/loc"
)

type presenceSlice []int

func (n presenceSlice) Accept(Visitor) (Node, bool)    { return n, true }
func (presenceSlice) Restore(*format.RestoreCtx) error { return nil }
func (presenceSlice) Text() string                     { return "" }
func (presenceSlice) SetText(string)                   {}
func (presenceSlice) Span() loc.Span                   { return loc.InvalidSpan() }
func (presenceSlice) SetSpan(loc.Span)                 {}

func TestNodePresent(t *testing.T) {
	for _, tc := range []struct {
		name string
		node Node
		want bool
	}{
		{name: "nil interface"},
		{name: "nil pointer", node: (*UseStmt)(nil)},
		{name: "nil slice", node: presenceSlice(nil)},
		{name: "empty slice", node: presenceSlice{}, want: true},
		{name: "pointer", node: &UseStmt{}, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := nodePresent(tc.node); got != tc.want {
				t.Fatalf("nodePresent(%T)=%v, want %v", tc.node, got, tc.want)
			}
		})
	}
}

package driver

import (
	"strings"
	"testing"

	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/myparser/types"
	"github.com/mia0x75/myparser/types/json"
)

func TestValueExprRestore(t *testing.T) {
	testCases := []struct {
		datum  types.Datum
		expect string
	}{
		{types.NewDatum(nil), "NULL"},
		{types.NewIntDatum(1), "1"},
		{types.NewIntDatum(-1), "-1"},
		{types.NewUintDatum(1), "1"},
		{types.NewFloat32Datum(1.1), "1.1e+00"},
		{types.NewFloat64Datum(1.1), "1.1e+00"},
		{types.NewStringDatum("test `s't\"r."), "'test `s''t\"r.'"},
		{types.NewBytesDatum([]byte("test `s't\"r.")), "'test `s''t\"r.'"},
		{types.NewBinaryLiteralDatum([]byte("test `s't\"r.")), "b'11101000110010101110011011101000010000001100000011100110010011101110100001000100111001000101110'"},
		{types.NewDecimalDatum(types.NewDecFromInt(321)), "321"},
		// MysqlTime (date/datetime) restores as a quoted string.
		{
			datum: func() types.Datum {
				var d types.Datum
				d.SetMysqlTime(types.Time{Time: types.FromDate(2020, 1, 2, 3, 4, 5, 0), Type: mysql.TypeDate})
				return d
			}(),
			expect: "'2020-01-02'",
		},
		// MysqlDuration (time) restores as a quoted string.
		{
			datum:  types.NewDurationDatum(types.Duration{Duration: 3723000000000, Fsp: 0}), // 01:02:03
			expect: "'01:02:03'",
		},
		// MysqlEnum and MysqlSet restore as quoted strings.
		{
			datum:  func() types.Datum { var d types.Datum; d.SetMysqlEnum(types.Enum{Name: "a"}); return d }(),
			expect: "'a'",
		},
		{
			datum:  func() types.Datum { var d types.Datum; d.SetMysqlSet(types.Set{Name: "a,b"}); return d }(),
			expect: "'a,b'",
		},
		// MysqlJSON restores as unquoted JSON text.
		{
			datum: func() types.Datum {
				d := types.Datum{}
				j, _ := json.ParseBinaryFromString(`{"a":1}`)
				d.SetMysqlJSON(j)
				return d
			}(),
			expect: `{"a": 1}`,
		},
	}
	// Run Test
	var sb strings.Builder
	for _, testCase := range testCases {
		sb.Reset()
		expr := &ValueExpr{Datum: testCase.datum}
		err := expr.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb))
		if err != nil {
			t.Fatal(err)
		}
		if sb.String() != testCase.expect {
			t.Fatalf("got %#v, want %#v (Datum: %#v)", sb.String(), testCase.expect, testCase.datum)
		}
	}
}

func TestValueExprFormat(t *testing.T) {
	js, err := json.ParseBinaryFromString(`{"a": 1}`)
	if err != nil {
		t.Fatal(err)
	}
	datum := func(f func(*types.Datum)) types.Datum {
		var d types.Datum
		f(&d)
		return d
	}
	testCases := []struct {
		datum  types.Datum
		expect string
	}{
		{datum(func(d *types.Datum) { d.SetMysqlDuration(types.Duration{}) }), "00:00:00"},
		{datum(func(d *types.Datum) { d.SetMysqlEnum(types.Enum{Name: "a"}) }), "a"},
		{datum(func(d *types.Datum) { d.SetMysqlSet(types.Set{Name: "a,b"}) }), "a,b"},
		{types.NewMysqlBitDatum(types.BinaryLiteral("ab")), "ab"},
		{datum(func(d *types.Datum) {
			d.SetMysqlTime(types.Time{Time: types.FromDate(2020, 1, 2, 3, 4, 5, 0), Type: mysql.TypeDate})
		}), "2020-01-02"},
		{datum(func(d *types.Datum) { d.SetMysqlJSON(js) }), `{"a": 1}`},
	}
	for _, testCase := range testCases {
		var sb strings.Builder
		(&ValueExpr{Datum: testCase.datum}).Format(&sb)
		if sb.String() != testCase.expect {
			t.Fatalf("got %#v, want %#v (Datum: %#v)", sb.String(), testCase.expect, testCase.datum)
		}
	}
}

func TestValueExprFormatSpecialKindsDoesNotPanic(t *testing.T) {
	// Format cannot report errors, so these sentinel/internal kinds must not panic.
	datums := []types.Datum{
		types.MinNotNullDatum(),
		types.MaxValueDatum(),
		func() types.Datum {
			var d types.Datum
			d.SetRaw([]byte("raw"))
			return d
		}(),
		func() types.Datum {
			var d types.Datum
			d.SetInterface(42)
			return d
		}(),
	}
	for _, d := range datums {
		var sb strings.Builder
		(&ValueExpr{Datum: d}).Format(&sb)
	}
}

func TestParamMarkerExprFormat(t *testing.T) {
	var sb strings.Builder
	(&ParamMarkerExpr{}).Format(&sb)
	if got := sb.String(); got != "?" {
		t.Fatalf("got %#v, want %q", got, "?")
	}
}

type testVisitor struct {
	entered int
	left    int
}

func (v *testVisitor) Enter(n ast.Node) (ast.Node, bool) {
	v.entered++
	return n, false
}

func (v *testVisitor) Leave(n ast.Node) (ast.Node, bool) {
	v.left++
	return n, true
}

func TestParamMarkerExprRestoreAcceptAndOrder(t *testing.T) {
	var sb strings.Builder
	pm := &ParamMarkerExpr{Offset: 3}
	if err := pm.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &sb)); err != nil {
		t.Fatal(err)
	}
	if got := sb.String(); got != "?" {
		t.Fatalf("Restore: got %#v, want ?", got)
	}

	visitor := &testVisitor{}
	if _, ok := pm.Accept(visitor); !ok {
		t.Fatal("Accept returned ok=false")
	}
	if visitor.entered != 1 || visitor.left != 1 {
		t.Fatalf("Accept counts: entered=%d left=%d, want 1/1", visitor.entered, visitor.left)
	}

	pm.SetOrder(2)
	if pm.Order != 2 {
		t.Fatalf("Order: got %d, want 2", pm.Order)
	}
}

func TestValueExprProjectionOffsetAndGetDatumString(t *testing.T) {
	v := &ValueExpr{Datum: types.NewStringDatum("abc")}
	v.SetProjectionOffset(7)
	if got := v.GetProjectionOffset(); got != 7 {
		t.Fatalf("GetProjectionOffset: got %d, want 7", got)
	}
	if got := v.GetDatumString(); got != "abc" {
		t.Fatalf("GetDatumString: got %#v, want %q", got, "abc")
	}
}

func TestValueExprAcceptVisitorsChildren(t *testing.T) {
	v := &ValueExpr{Datum: types.NewIntDatum(1)}
	visitor := &testVisitor{}
	if _, ok := v.Accept(visitor); !ok {
		t.Fatal("Accept returned ok=false")
	}
	if visitor.entered != 1 || visitor.left != 1 {
		t.Fatalf("Accept counts: entered=%d left=%d, want 1/1", visitor.entered, visitor.left)
	}
}

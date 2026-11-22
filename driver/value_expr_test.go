package driver

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/myparser/types"
	"github.com/mia0x75/myparser/types/json"
	pkast "github.com/mia0x75/parserkit/ast"
)

// TestValueExprFormat verifies the driver-only Format method output.
// Format is a standalone diagnostic method: it is not part of the AST
// Node/Restore/Visitor contract and must not be called via ast.ExprNode.
func TestValueExprFormat(t *testing.T) {
	js, err := json.ParseBinaryFromString(`{"a": 1}`)
	if err != nil {
		t.Fatal(err)
	}
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
		{types.NewStringDatum("test `s't\"r."), strconv.Quote("test `s't\"r.")},
		{types.NewBytesDatum([]byte("test `s't\"r.")), strconv.Quote("test `s't\"r.")},
		{types.NewBinaryLiteralDatum([]byte("test `s't\"r.")), "b'11101000110010101110011011101000010000001100000011100110010011101110100001000100111001000101110'"},
		{types.NewDecimalDatum(types.NewDecFromInt(321)), "321"},
		// MysqlTime (date/datetime) falls through to ToString, unquoted.
		{
			datum: func() types.Datum {
				var d types.Datum
				d.SetMysqlTime(types.Time{Time: types.FromDate(2020, 1, 2, 3, 4, 5, 0), Type: mysql.TypeDate})
				return d
			}(),
			expect: "2020-01-02",
		},
		// MysqlDuration (time) falls through to ToString, unquoted.
		{
			datum:  types.NewDurationDatum(types.Duration{Duration: 3723000000000, Fsp: 0}), // 01:02:03
			expect: "01:02:03",
		},
		// MysqlEnum and MysqlSet fall through to ToString, unquoted.
		{
			datum:  func() types.Datum { var d types.Datum; d.SetMysqlEnum(types.Enum{Name: "a"}); return d }(),
			expect: "a",
		},
		{
			datum:  func() types.Datum { var d types.Datum; d.SetMysqlSet(types.Set{Name: "a,b"}); return d }(),
			expect: "a,b",
		},
		// MysqlJSON falls through to ToString, unquoted.
		{
			datum: func() types.Datum {
				d := types.Datum{}
				d.SetMysqlJSON(js)
				return d
			}(),
			expect: `{"a": 1}`,
		},
	}
	for i, testCase := range testCases {
		t.Run("", func(t *testing.T) {
			var sb strings.Builder
			(&ValueExpr{Datum: testCase.datum}).Format(&sb)
			if sb.String() != testCase.expect {
				t.Fatalf("case %d: Format got %q, want %q (Datum: %#v)", i, sb.String(), testCase.expect, testCase.datum)
			}
		})
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

// TestValueExprIsNotANode proves driver.ValueExpr is a runtime execution value
// that must not be placed in an AST slot. The reflect check catches any future
// regression that re-adds Node/ExprNode identity to the type.
func TestValueExprIsNotANode(t *testing.T) {
	t.Parallel()
	typ := reflect.TypeFor[*ValueExpr]()
	if typ.Implements(reflect.TypeFor[pkast.Node]()) {
		t.Error("ValueExpr implements parserkit ast.Node; it must remain outside the AST boundary")
	}
	if typ.Implements(reflect.TypeFor[pkast.ExprNode]()) {
		t.Error("ValueExpr implements parserkit ast.ExprNode; it must remain outside the AST boundary")
	}
	// Also verify it does not have the Accept method that would make it a visitor node.
	if _, ok := typ.MethodByName("Accept"); ok {
		t.Error("ValueExpr exposes Accept; it must remain outside the AST boundary")
	}
}

// TestValueExprFormatStringQuoting checks the strconv.Quote behavior for
// string and bytes kinds, which is the legacy WriteString canonical form.
func TestValueExprFormatStringQuoting(t *testing.T) {
	for _, tc := range []struct {
		input  any
		expect string
	}{
		{"plain", `"plain"`},
		{"with \"quote\"", `"with \"quote\""`},
		{"with 'single'", `"with 'single'"`},
		{"newline\n", `"newline\n"`},
	} {
		v := NewValueExpr(tc.input)
		var sb strings.Builder
		v.Format(&sb)
		if sb.String() != tc.expect {
			t.Fatalf("Format(%v) = %q, want %q", tc.input, sb.String(), tc.expect)
		}
	}
}

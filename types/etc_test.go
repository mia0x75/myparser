package types

import (
	"io"
	"testing"

	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/myparser/sqlerr"
)

func testIsTypeBlob(t *testing.T, tp byte, expect bool) {
	v := IsTypeBlob(tp)
	if v != expect {
		t.Fatalf("got %#v, want %#v", v, expect)
	}
}

func testIsTypeChar(t *testing.T, tp byte, expect bool) {
	v := IsTypeChar(tp)
	if v != expect {
		t.Fatalf("got %#v, want %#v", v, expect)
	}
}

func TestIsType(t *testing.T) {
	testIsTypeBlob(t, mysql.TypeTinyBlob, true)
	testIsTypeBlob(t, mysql.TypeMediumBlob, true)
	testIsTypeBlob(t, mysql.TypeBlob, true)
	testIsTypeBlob(t, mysql.TypeLongBlob, true)
	testIsTypeBlob(t, mysql.TypeInt24, false)

	testIsTypeChar(t, mysql.TypeString, true)
	testIsTypeChar(t, mysql.TypeVarchar, true)
	testIsTypeChar(t, mysql.TypeLong, false)
}

func testTypeStr(t *testing.T, tp byte, expect string) {
	v := TypeStr(tp)
	if v != expect {
		t.Fatalf("got %#v, want %#v", v, expect)
	}
}

func testTypeToStr(t *testing.T, tp byte, charset string, expect string) {
	v := TypeToStr(tp, charset)
	if v != expect {
		t.Fatalf("got %#v, want %#v", v, expect)
	}
}

func TestTypeToStr(t *testing.T) {
	testTypeStr(t, mysql.TypeYear, "year")
	testTypeStr(t, 0xdd, "")

	testTypeToStr(t, mysql.TypeBlob, "utf8", "text")
	testTypeToStr(t, mysql.TypeLongBlob, "utf8", "longtext")
	testTypeToStr(t, mysql.TypeTinyBlob, "utf8", "tinytext")
	testTypeToStr(t, mysql.TypeMediumBlob, "utf8", "mediumtext")
	testTypeToStr(t, mysql.TypeVarchar, "binary", "varbinary")
	testTypeToStr(t, mysql.TypeString, "binary", "binary")
	testTypeToStr(t, mysql.TypeTiny, "binary", "tinyint")
	testTypeToStr(t, mysql.TypeBlob, "binary", "blob")
	testTypeToStr(t, mysql.TypeLongBlob, "binary", "longblob")
	testTypeToStr(t, mysql.TypeTinyBlob, "binary", "tinyblob")
	testTypeToStr(t, mysql.TypeMediumBlob, "binary", "mediumblob")
	testTypeToStr(t, mysql.TypeVarchar, "utf8", "varchar")
	testTypeToStr(t, mysql.TypeString, "utf8", "char")
	testTypeToStr(t, mysql.TypeShort, "binary", "smallint")
	testTypeToStr(t, mysql.TypeInt24, "binary", "mediumint")
	testTypeToStr(t, mysql.TypeLong, "binary", "int")
	testTypeToStr(t, mysql.TypeLonglong, "binary", "bigint")
	testTypeToStr(t, mysql.TypeFloat, "binary", "float")
	testTypeToStr(t, mysql.TypeDouble, "binary", "double")
	testTypeToStr(t, mysql.TypeYear, "binary", "year")
	testTypeToStr(t, mysql.TypeDuration, "binary", "time")
	testTypeToStr(t, mysql.TypeDatetime, "binary", "datetime")
	testTypeToStr(t, mysql.TypeDate, "binary", "date")
	testTypeToStr(t, mysql.TypeTimestamp, "binary", "timestamp")
	testTypeToStr(t, mysql.TypeNewDecimal, "binary", "decimal")
	testTypeToStr(t, mysql.TypeUnspecified, "binary", "unspecified")
	testTypeToStr(t, 0xdd, "binary", "")
	testTypeToStr(t, mysql.TypeBit, "binary", "bit")
	testTypeToStr(t, mysql.TypeEnum, "binary", "enum")
	testTypeToStr(t, mysql.TypeSet, "binary", "set")
}

func TestEOFAsNil(t *testing.T) {
	err := EOFAsNil(io.EOF)
	if err != nil {
		t.Fatalf("got %#v, want nil", err)
	}
}

func TestMaxFloat(t *testing.T) {
	tbl := []struct {
		Flen    int
		Decimal int
		Expect  float64
	}{
		{3, 2, 9.99},
		{5, 2, 999.99},
		{10, 1, 999999999.9},
		{5, 5, 0.99999},
	}

	for _, tt := range tbl {
		f := GetMaxFloat(tt.Flen, tt.Decimal)
		if f != tt.Expect {
			t.Fatalf("got %#v, want %#v", f, tt.Expect)
		}
	}
}

func TestRoundFloat(t *testing.T) {
	tbl := []struct {
		Input  float64
		Expect float64
	}{
		{2.5, 3},
		{1.5, 2},
		{0.5, 1},
		{0.49999999999999997, 0},
		{0, 0},
		{-0.49999999999999997, 0},
		{-0.5, -1},
		{-2.5, -3},
		{-1.5, -2},
	}

	for _, tt := range tbl {
		f := RoundFloat(tt.Input)
		if f != tt.Expect {
			t.Fatalf("got %#v, want %#v", f, tt.Expect)
		}
	}
}

func TestRound(t *testing.T) {
	tbl := []struct {
		Input  float64
		Dec    int
		Expect float64
	}{
		{-1.23, 0, -1},
		{-1.58, 0, -2},
		{1.58, 0, 2},
		{1.298, 1, 1.3},
		{1.298, 0, 1},
		{23.298, -1, 20},
	}

	for _, tt := range tbl {
		f := Round(tt.Input, tt.Dec)
		if f != tt.Expect {
			t.Fatalf("got %#v, want %#v", f, tt.Expect)
		}
	}
}

func TestTruncate(t *testing.T) {
	tbl := []struct {
		Input   float64
		Flen    int
		Decimal int
		Expect  float64
		Err     error
	}{
		{100.114, 10, 2, 100.11, nil},
		{100.115, 10, 2, 100.12, nil},
		{100.1156, 10, 3, 100.116, nil},
		{100.1156, 3, 1, 99.9, ErrOverflow},
		{1.36, 10, 2, 1.36, nil},
	}

	for _, tt := range tbl {
		f, err := TruncateFloat(tt.Input, tt.Flen, tt.Decimal)
		if f != tt.Expect {
			t.Fatalf("got %#v, want %#v", f, tt.Expect)
		}
		if !sqlerr.ErrorEqual(err, tt.Err) {
			t.Fatalf("err %v", err)
		}
	}
}

func TestCompareFunctions(t *testing.T) {
	if CompareInt64(1, 2) != -1 || CompareInt64(2, 1) != 1 || CompareInt64(1, 1) != 0 {
		t.Fatal("CompareInt64")
	}
	if CompareUint64(1, 2) != -1 || CompareUint64(2, 1) != 1 || CompareUint64(1, 1) != 0 {
		t.Fatal("CompareUint64")
	}
	if CompareFloat64(1.1, 2.2) != -1 || CompareFloat64(2.2, 1.1) != 1 || CompareFloat64(1.1, 1.1) != 0 {
		t.Fatal("CompareFloat64")
	}
	if CompareString("a", "b") != -1 || CompareString("b", "a") != 1 || CompareString("a", "a") != 0 {
		t.Fatal("CompareString")
	}
}

func TestToString(t *testing.T) {
	testCases := []struct {
		input any
		want  string
	}{
		{true, "1"},
		{false, "0"},
		{42, "42"},
		{int64(42), "42"},
		{uint64(42), "42"},
		{3.14, "3.14"},
		{float32(1.5), "1.5"},
		{"hello", "hello"},
		{[]byte("world"), "world"},
	}
	for _, tc := range testCases {
		got, err := ToString(tc.input)
		if err != nil {
			t.Fatalf("ToString(%v): %v", tc.input, err)
		}
		if got != tc.want {
			t.Fatalf("ToString(%v): got %#v, want %#v", tc.input, got, tc.want)
		}
	}
	// Unconvertible type must error.
	if _, err := ToString(struct{ X int }{1}); err == nil {
		t.Fatal("ToString: expected error for struct")
	}
}

func TestConvertFunctions(t *testing.T) {
	// ConvertFloatToInt: in-range, lower-overflow, upper-exact, upper-overflow.
	if v, err := ConvertFloatToInt(3.7, 0, 10, mysql.TypeLong); err != nil || v != 4 {
		t.Fatalf("ConvertFloatToInt in-range: got %d, err %v", v, err)
	}
	if _, err := ConvertFloatToInt(-1.1, 0, 10, mysql.TypeLong); err == nil {
		t.Fatal("ConvertFloatToInt: expected lower-bound overflow error")
	}
	if v, err := ConvertFloatToInt(10.0, 0, 10, mysql.TypeLong); err != nil || v != 10 {
		t.Fatalf("ConvertFloatToInt exact upper: got %d, err %v", v, err)
	}
	if _, err := ConvertFloatToInt(11.4, 0, 10, mysql.TypeLong); err == nil {
		t.Fatal("ConvertFloatToInt: expected upper-bound overflow error")
	}

	// ConvertIntToInt: in-range, lower, upper.
	if v, err := ConvertIntToInt(5, 0, 10, mysql.TypeLong); err != nil || v != 5 {
		t.Fatalf("ConvertIntToInt in-range: got %d, err %v", v, err)
	}
	if _, err := ConvertIntToInt(-1, 0, 10, mysql.TypeLong); err == nil {
		t.Fatal("ConvertIntToInt: expected lower-bound error")
	}
	if _, err := ConvertIntToInt(11, 0, 10, mysql.TypeLong); err == nil {
		t.Fatal("ConvertIntToInt: expected upper-bound error")
	}

	// ConvertUintToInt: in-range and overflow.
	if v, err := ConvertUintToInt(5, 10, mysql.TypeLong); err != nil || v != 5 {
		t.Fatalf("ConvertUintToInt in-range: got %d, err %v", v, err)
	}
	if _, err := ConvertUintToInt(11, 10, mysql.TypeLong); err == nil {
		t.Fatal("ConvertUintToInt: expected overflow")
	}

	// ConvertUintToUint: in-range and overflow.
	if v, err := ConvertUintToUint(5, 10, mysql.TypeLong); err != nil || v != 5 {
		t.Fatalf("ConvertUintToUint in-range: got %d, err %v", v, err)
	}
	if _, err := ConvertUintToUint(11, 10, mysql.TypeLong); err == nil {
		t.Fatal("ConvertUintToUint: expected overflow")
	}
}

func TestBinaryLiteralToString(t *testing.T) {
	b := BinaryLiteral("ab")
	if got := b.ToString(); got != "ab" {
		t.Fatalf("ToString: got %#v, want %q", got, "ab")
	}
}

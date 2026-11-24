package myparser_test

import (
	"strings"

	"github.com/mia0x75/myparser"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"SELECT 1", "select ?"},
		{"select * from b where id = 1", "select * from b where id = ?"},
		{"select 1 from b where id in (1, 3, '3', 1, 2, 3, 4)", "select ? from b where id in ( ... )"},
		{"select 1 from b where id in (1, a, 4)", "select ? from b where id in ( ? , a , ? )"},
		{"select 1 from b order by 2", "select ? from b order by 2"},
		{"select /*+ a hint */ 1", "select ?"},
		{"select /* a hint */ 1", "select ?"},
		{"select truncate(1, 2)", "select truncate ( ... )"},
		{"select -1 + - 2 + b - c + 0.2 + (-2) from c where d in (1, -2, +3)", "select ? + ? + b - c + ? + ( ? ) from c where d in ( ... )"},
		{"select * from t where a <= -1 and b < -2 and c = -3 and c > -4 and c >= -5 and e is 1", "select * from t where a <= ? and b < ? and c = ? and c > ? and c >= ? and e is ?"},
		{"select count(a), b from t group by 2", "select count ( a ) , b from t group by 2"},
		{"select count(a), b, c from t group by 2, 3", "select count ( a ) , b , c from t group by 2 , 3"},
		{"select count(a), b, c from t group by (2, 3)", "select count ( a ) , b , c from t group by ( 2 , 3 )"},
		{"select a, b from t order by 1, 2", "select a , b from t order by 1 , 2"},
		{"select count(*) from t", "select count ( ? ) from t"},
		{"select * from t Force Index(kk)", "select * from t"},
		{"select * from t USE Index(kk)", "select * from t"},
		{"select * from t Ignore Index(kk)", "select * from t"},
		{"select * from t1 straight_join t2 on t1.id=t2.id", "select * from t1 join t2 on t1 . id = t2 . id"},
		// test syntax error, it will be checked by parser, but it should not make normalize dead loop.
		{"select * from t ignore index(", "select * from t ignore index"},
		{"select /*+ ", "select "},
		{"select * from 🥳", "select * from 🥳"},
	}
	for _, test := range tests {
		actual := myparser.Normalize(test.input)
		if actual != test.expect {
			t.Fatalf("got %#v, want %#v", actual, test.expect)
		}
	}
}

func TestNormalizeDigest(t *testing.T) {
	tests := []struct {
		sql        string
		normalized string
		digest     string
	}{
		{"select 1 from b where id in (1, 3, '3', 1, 2, 3, 4)", "select ? from b where id in ( ... )", "f36161eef94dbfbd5e2f6b9a2f498a4c7facc6860621fbeb8084f63898275016"},
	}
	for _, test := range tests {
		normalized, digest := myparser.NormalizeDigest(test.sql)
		if normalized != test.normalized {
			t.Fatalf("got %#v, want %#v", normalized, test.normalized)
		}
		if digest != test.digest {
			t.Fatalf("got %#v, want %#v", digest, test.digest)
		}
	}
}

func TestDigestHashReturnsSHA256(t *testing.T) {
	_, want := myparser.NormalizeDigest("select 1")

	got := myparser.DigestHash("select 1")
	if got != want {
		t.Fatalf("DigestHash mismatch: got %#v, want %#v", got, want)
	}
	if got != "e1c71d1661ae46e09b7aaec1c390957f0d6260410df4e4bc71b9c8d681021471" {
		t.Fatalf("DigestHash has unexpected value: %#v", got)
	}
	if len(got) != 64 || strings.ToLower(got) != got {
		t.Fatalf("DigestHash is not a 64-character lowercase hex digest: %#v", got)
	}
}

func TestDigestHashEqForSimpleSQL(t *testing.T) {
	sqlGroups := [][]string{
		{"select * from b where id = 1", "select * from b where id = '1'", "select * from b where id =2"},
		{"select 2 from b, c where c.id > 1", "select 4 from b, c where c.id > 23"},
		{"Select 3", "select 1"},
	}
	for _, sqlGroup := range sqlGroups {
		var d string
		for _, sql := range sqlGroup {
			dig := myparser.DigestHash(sql)
			if d == "" {
				d = dig
				continue
			}
			if d != dig {
				t.Fatalf("got %#v, want %#v", d, dig)
			}
		}
	}
}

func TestDigestHashNotEqForSimpleSQL(t *testing.T) {
	sqlGroups := [][]string{
		{"select * from b where id = 1", "select a from b where id = 1", "select * from d where bid =1"},
	}
	for _, sqlGroup := range sqlGroups {
		var d string
		for _, sql := range sqlGroup {
			dig := myparser.DigestHash(sql)
			if d == "" {
				d = dig
				continue
			}
			if d == dig {
				t.Fatalf("got %#v, want not %#v", d, dig)
			}
		}
	}
}

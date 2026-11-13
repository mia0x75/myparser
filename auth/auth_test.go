package auth

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mia0x75/myparser/format"
)

func TestEncodePassword(t *testing.T) {
	pwd := "123"
	if got, want := EncodePassword(pwd), "*23AE809DDACAF96AF0FD78ED04B6A265E05AA257"; got != want {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestDecodePassword(t *testing.T) {
	want := Sha1Hash(Sha1Hash([]byte("123")))
	for _, tt := range []struct {
		name string
		pwd  string
	}{
		{name: "prefixed", pwd: EncodePassword("123")},
		{name: "without prefix", pwd: EncodePassword("123")[1:]},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodePassword(tt.pwd)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %#v, want %#v", got, want)
			}
		})
	}
}

func TestDecodePasswordRejectsMalformedInput(t *testing.T) {
	for _, tt := range []struct {
		name string
		pwd  string
	}{
		{name: "empty", pwd: ""},
		{name: "wrong length", pwd: "*00"},
		{name: "invalid hex", pwd: "*23AE809DDACAF96AF0FD78ED04B6A265E05AA25Z"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := DecodePassword(tt.pwd); err == nil {
				t.Fatal("expected malformed password to be rejected")
			}
		})
	}
}

func TestCheckScramble(t *testing.T) {
	pwd := "abc"
	salt := []byte{85, 92, 45, 22, 58, 79, 107, 6, 122, 125, 58, 80, 12, 90, 103, 32, 90, 10, 74, 82}
	auth := []byte{24, 180, 183, 225, 166, 6, 81, 102, 70, 248, 199, 143, 91, 204, 169, 9, 161, 171, 203, 33}
	encodepwd := EncodePassword(pwd)
	hpwd, err := DecodePassword(encodepwd)
	if err != nil {
		t.Fatal(err)
	}

	res := CheckScrambledPassword(salt, hpwd, auth)
	if !res {
		t.Fatal("CheckScrambledPassword returned false")
	}
}

func TestCheckScrambledPasswordRejectsShortInput(t *testing.T) {
	if CheckScrambledPassword(nil, nil, nil) {
		t.Fatal("short scramble input must be rejected")
	}
}

func TestUserIdentityString(t *testing.T) {
	testCases := []struct {
		identity UserIdentity
		want     string
	}{
		{UserIdentity{Username: "root", Hostname: "localhost"}, "`root`@`localhost`"},
		{UserIdentity{Username: "u@v", Hostname: "h@i"}, "`u@v`@`h@i`"},
		{},
	}
	for _, tc := range testCases {
		var u *UserIdentity
		if tc.identity.Username != "" {
			u = &tc.identity
		}
		if u == nil {
			if got := u.String(); got != "" {
				t.Fatalf("nil identity: got %#v, want empty string", got)
			}
			continue
		}
		if got := u.String(); got != tc.want {
			t.Fatalf("got %#v, want %#v", got, tc.want)
		}
	}
}

func TestIdentityRestore(t *testing.T) {
	testCases := []struct {
		name string
		run  func(sb *strings.Builder) error
		want string
	}{
		{
			name: "UserIdentity current user",
			run: func(sb *strings.Builder) error {
				return (&UserIdentity{CurrentUser: true}).Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, sb))
			},
			want: "CURRENT_USER",
		},
		{
			name: "UserIdentity with hostname",
			run: func(sb *strings.Builder) error {
				return (&UserIdentity{Username: "u", Hostname: "h"}).Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, sb))
			},
			want: "`u`@`h`",
		},
		{
			name: "UserIdentity without hostname",
			run: func(sb *strings.Builder) error {
				return (&UserIdentity{Username: "u"}).Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, sb))
			},
			want: "`u`",
		},
		{
			name: "RoleIdentity with hostname",
			run: func(sb *strings.Builder) error {
				return (&RoleIdentity{Username: "r", Hostname: "h"}).Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, sb))
			},
			want: "`r`@`h`",
		},
		{
			name: "RoleIdentity without hostname",
			run: func(sb *strings.Builder) error {
				return (&RoleIdentity{Username: "r"}).Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, sb))
			},
			want: "`r`",
		},
	}
	for _, tc := range testCases {
		var sb strings.Builder
		if err := tc.run(&sb); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got := sb.String(); got != tc.want {
			t.Fatalf("%s: got %#v, want %#v", tc.name, got, tc.want)
		}
	}
}

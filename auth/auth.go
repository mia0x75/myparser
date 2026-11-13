package auth

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	"github.com/mia0x75/myparser/errors"

	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/sqlerr"
)

// UserIdentity represents username and hostname.
type UserIdentity struct {
	Username     string
	Hostname     string
	CurrentUser  bool
	AuthUsername string // Username matched in privileges system
	AuthHostname string // Match in privs system (i.e. could be a wildcard)
}

// Restore implements Node interface.
func (user *UserIdentity) Restore(ctx *format.RestoreCtx) error {
	if user.CurrentUser {
		ctx.WriteKeyWord("CURRENT_USER")
	} else {
		ctx.WriteName(user.Username)
		if user.Hostname != "" {
			ctx.WritePlain("@")
			ctx.WriteName(user.Hostname)
		}
	}
	return nil
}

// String returns user@host with backtick-quoted components.
// Used for diagnostic output where SQL injection is not a concern but
// special characters must not break the user@host format.
func (user *UserIdentity) String() string {
	if user == nil {
		return ""
	}
	return fmt.Sprintf("`%s`@`%s`", user.Username, user.Hostname)
}

type RoleIdentity struct {
	Username string
	Hostname string
}

func (role *RoleIdentity) Restore(ctx *format.RestoreCtx) error {
	ctx.WriteName(role.Username)
	if role.Hostname != "" {
		ctx.WritePlain("@")
		ctx.WriteName(role.Hostname)
	}
	return nil
}

// CheckScrambledPassword check scrambled password received from client.
// The new authentication is performed in following manner:
//
//	SERVER:  public_seed=create_random_string()
//	         send(public_seed)
//	CLIENT:  recv(public_seed)
//	         hash_stage1=sha1("password")
//	         hash_stage2=sha1(hash_stage1)
//	         reply=xor(hash_stage1, sha1(public_seed,hash_stage2)
//	         // this three steps are done in scramble()
//	         send(reply)
//	SERVER:  recv(reply)
//	         hash_stage1=xor(reply, sha1(public_seed,hash_stage2))
//	         candidate_hash2=sha1(hash_stage1)
//	         check(candidate_hash2==hash_stage2)
//	         // this three steps are done in check_scramble()
//
// CheckScrambledPassword verifies the legacy MySQL SHA1 password scramble.
// SHA1 is kept for wire-protocol compatibility, not general hashing; do not
// reuse these helpers outside this emulation.
func CheckScrambledPassword(salt, hpwd, auth []byte) bool {
	if len(auth) != sha1.Size || len(hpwd) != sha1.Size {
		return false
	}

	crypt := sha1.New()
	_, err := crypt.Write(salt)
	sqlerr.Log(errors.Trace(err))
	_, err = crypt.Write(hpwd)
	sqlerr.Log(errors.Trace(err))
	hash := crypt.Sum(nil)
	// token = scrambleHash XOR stage1Hash
	for i := range hash {
		hash[i] ^= auth[i]
	}

	return bytes.Equal(hpwd, Sha1Hash(hash))
}

// Sha1Hash is an util function to calculate sha1 hash.
// Protocol-mandated for MySQL auth emulation; not general-purpose.
func Sha1Hash(bs []byte) []byte {
	crypt := sha1.New()
	_, err := crypt.Write(bs)
	sqlerr.Log(errors.Trace(err))
	return crypt.Sum(nil)
}

// EncodePassword converts plaintext password to hashed hex string.
func EncodePassword(pwd string) string {
	if len(pwd) == 0 {
		return ""
	}
	hash1 := Sha1Hash([]byte(pwd))
	hash2 := Sha1Hash(hash1)

	return fmt.Sprintf("*%X", hash2)
}

// DecodePassword converts a MySQL password hash to byte array.
func DecodePassword(pwd string) ([]byte, error) {
	if len(pwd) == 1+sha1.Size*2 && pwd[0] == '*' {
		pwd = pwd[1:]
	}
	if len(pwd) != sha1.Size*2 {
		return nil, errors.New("invalid MySQL password hash")
	}

	x, err := hex.DecodeString(pwd)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return x, nil
}

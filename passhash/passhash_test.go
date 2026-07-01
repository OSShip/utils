package passhash

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashAndVerifyPassword(t *testing.T) {
	salt, hash, err := HashPasswordPair("password123")
	if err != nil {
		t.Fatal(err)
	}
	if salt == "" || hash == "" {
		t.Fatal("expected non-empty salt and hash")
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("expected argon2id hash, got %q", hash[:min(16, len(hash))])
	}
	if !VerifyPassword("password123", salt, hash) {
		t.Fatal("expected password to verify")
	}
	if VerifyPassword("wrong", salt, hash) {
		t.Fatal("expected wrong password to fail")
	}
}

func TestUniqueSaltsPerHash(t *testing.T) {
	salt1, _, _ := HashPasswordPair("same-password")
	salt2, _, _ := HashPasswordPair("same-password")
	if salt1 == salt2 {
		t.Fatal("expected unique salts for each hash")
	}
}

func TestHashOAuthLengthPassword(t *testing.T) {
	// OAuth users get a random UUID password; with per-user salt this exceeds bcrypt's 72-byte input limit.
	oauthPass := "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	salt, hash, err := HashPasswordPair(oauthPass)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(oauthPass, salt, hash) {
		t.Fatal("expected oauth-length password to verify")
	}
}

func TestVerifyLegacyBcryptOnly(t *testing.T) {
	legacy, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword("password123", "", string(legacy)) {
		t.Fatal("expected legacy bcrypt verification")
	}
}

func TestVerifyLegacyBcryptWithSalt(t *testing.T) {
	salt := "legacy-salt-value"
	legacy, err := bcrypt.GenerateFromPassword([]byte(salt+"password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword("password123", salt, string(legacy)) {
		t.Fatal("expected legacy salted bcrypt verification")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

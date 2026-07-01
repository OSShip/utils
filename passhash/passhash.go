package passhash

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/alexedwards/argon2id"
	"golang.org/x/crypto/bcrypt"
)

const saltBytes = 32

// HashPassword hashes password combined with the per-user salt using Argon2id.
func HashPassword(password, salt string) (string, error) {
	input := password
	if salt != "" {
		input = salt + password
	}
	hash, err := argon2id.CreateHash(input, argon2id.DefaultParams)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return hash, nil
}

// GenerateSalt returns a cryptographically random salt encoded as base64.
func GenerateSalt() (string, error) {
	b := make([]byte, saltBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	return base64.RawStdEncoding.EncodeToString(b), nil
}

// HashPasswordPair generates a new salt and returns salt + hash for storage.
func HashPasswordPair(password string) (salt, hash string, err error) {
	salt, err = GenerateSalt()
	if err != nil {
		return "", "", err
	}
	hash, err = HashPassword(password, salt)
	if err != nil {
		return "", "", err
	}
	return salt, hash, nil
}

// VerifyPassword checks a password against stored hash and salt.
// Argon2id hashes are verified when hash starts with "$argon2id$".
// Legacy bcrypt hashes (with or without per-user salt) remain supported.
func VerifyPassword(password, salt, hash string) bool {
	input := password
	if salt != "" {
		input = salt + password
	}

	if strings.HasPrefix(hash, "$argon2id$") {
		match, err := argon2id.ComparePasswordAndHash(input, hash)
		return err == nil && match
	}

	if isBcryptHash(hash) {
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(input)) == nil
	}

	return false
}

func isBcryptHash(hash string) bool {
	return strings.HasPrefix(hash, "$2a$") ||
		strings.HasPrefix(hash, "$2b$") ||
		strings.HasPrefix(hash, "$2y$")
}

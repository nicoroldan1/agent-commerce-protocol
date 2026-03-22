package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// GenerateToken creates a new registry token with "rgt_" prefix.
func GenerateToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return "rgt_" + hex.EncodeToString(b)
}

// HashToken returns a SHA-256 hash of the token for storage.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// ValidateToken checks a raw token against a stored hash.
func ValidateToken(token, storedHash string) bool {
	return HashToken(token) == storedHash
}

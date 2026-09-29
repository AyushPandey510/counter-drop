package store

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"counter-drop/api/internal/domain"
)

// IDs, secrets and PINs. Only hashes are ever stored: ticket secrets, session tokens and setup
// links as SHA-256, PINs with PBKDF2.

func NewID(prefix string) string {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s_%x%s", prefix, time.Now().UTC().Unix(), hex.EncodeToString(b))
}

// NewSecret returns 32 random bytes as hex (ticket secrets, session tokens).
func NewSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func HashSecret(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func EqualHash(rawSecret, storedHash string) bool {
	if rawSecret == "" || storedHash == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(HashSecret(rawSecret)), []byte(storedHash)) == 1
}

func NewLinkToken() string {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func CheckNewPIN(pin string) error {
	if !pinPattern.MatchString(pin) {
		return fmt.Errorf("%w: PIN must be 4 digits", domain.ErrValidation)
	}
	if domain.WeakPIN(pin) {
		return ErrWeakPIN
	}
	return nil
}

var pinPattern = regexp.MustCompile(`^\d{4}$`)

func HashPIN(pin string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	const iter = 120_000
	key, err := pbkdf2.Key(sha256.New, pin, salt, iter, 32)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", iter, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

func VerifyPIN(pin, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}
	salt, err1 := base64.RawStdEncoding.DecodeString(parts[2])
	want, err2 := base64.RawStdEncoding.DecodeString(parts[3])
	if err1 != nil || err2 != nil {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, pin, salt, iter, len(want))
	return err == nil && subtle.ConstantTimeCompare(got, want) == 1
}

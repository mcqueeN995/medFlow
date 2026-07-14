package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	memory      = uint32(64 * 1024) // 64 MB
	iterations  = uint32(3)
	parallelism = uint8(4)
	saltLength  = uint32(16)
	keyLength   = uint32(32)

	ErrInvalidHash  = errors.New("invalid hash format")
	ErrIncompatible = errors.New("incompatible hash version")
)

// Hash генерирует argon2id хеш пароля
// Формат: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
func Hash(password string) (string, error) {
	// 1. Генерируем случайную соль
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLength)

	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		memory, iterations, parallelism, saltB64, hashB64)

	return encoded, nil
}

// Compare сравнивает пароль с хешем
// Возвращает true, если пароль совпадает
func Compare(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, ErrInvalidHash
	}

	if parts[1] != "argon2id" {
		return false, ErrIncompatible
	}

	if parts[2] != "v=19" {
		return false, ErrIncompatible
	}

	var (
		mem  uint32
		iter uint32
		par  uint8
	)
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &iter, &par)
	if err != nil {
		return false, fmt.Errorf("parse parameters: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("decode salt: %w", err)
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("decode hash: %w", err)
	}

	computedHash := argon2.IDKey([]byte(password), salt, iter, mem, par, uint32(len(expectedHash)))

	if subtle.ConstantTimeCompare(computedHash, expectedHash) != 1 {
		return false, nil
	}

	return true, nil
}

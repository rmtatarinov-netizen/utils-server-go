package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"

	"golang.org/x/crypto/argon2"
)

const (
	saltSize  = 16
	timeCost  = 1
	memory    = 64 * 1024
	threads   = 4
	keyLength = 32
)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) Hash(value string) (string, error) {
	if value == "" {
		return "", errors.New("password must be a non-empty string")
	}

	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(value), salt, timeCost, memory, threads, keyLength)
	encoded := base64.RawStdEncoding.EncodeToString(append(salt, hash...))
	return encoded, nil
}

func (s *Service) Verify(encoded, value string) (bool, error) {
	if encoded == "" {
		return false, nil
	}

	decoded, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil || len(decoded) < saltSize {
		return false, nil
	}

	salt := decoded[:saltSize]
	hash := decoded[saltSize:]
	expected := argon2.IDKey([]byte(value), salt, timeCost, memory, threads, keyLength)

	return subtle.ConstantTimeCompare(hash, expected) == 1, nil
}

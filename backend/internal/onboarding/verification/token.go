package verification

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func GenerateInviteToken() (token string, prefix string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("read random token bytes: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	if len(token) < 8 {
		return "", "", fmt.Errorf("generated token too short")
	}
	return token, token[:8], nil
}

func HashInviteToken(token string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash invite token: %w", err)
	}
	return string(hash), nil
}

func VerifyInviteToken(hash, token string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(token)) == nil
}

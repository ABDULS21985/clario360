package verification

import (
	"crypto/rand"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func GenerateOTP(length int) (string, error) {
	if length < 4 || length > 12 {
		return "", fmt.Errorf("otp length must be between 4 and 12")
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	var out strings.Builder
	out.Grow(length)
	for _, b := range buf {
		out.WriteByte('0' + (b % 10))
	}
	return out.String(), nil
}

func HashOTP(otp string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash otp: %w", err)
	}
	return string(hash), nil
}

func VerifyOTP(hash, otp string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(otp)) == nil
}

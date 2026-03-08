package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
)

type ConfigEncryptor struct {
	key   []byte
	keyID string
}

func NewConfigEncryptor(rawKey string, keyID string) (*ConfigEncryptor, error) {
	key, err := normalizeKey(rawKey)
	if err != nil {
		return nil, err
	}
	if keyID == "" {
		keyID = "local-aes256gcm"
	}
	return &ConfigEncryptor{key: key, keyID: keyID}, nil
}

func normalizeKey(rawKey string) ([]byte, error) {
	if len(rawKey) == 32 {
		return []byte(rawKey), nil
	}
	if decoded, err := base64.StdEncoding.DecodeString(rawKey); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(rawKey); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if decoded, err := hex.DecodeString(rawKey); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	return nil, fmt.Errorf("integration encryption key must decode to 32 bytes")
}

func (e *ConfigEncryptor) KeyID() string {
	return e.keyID
}

func (e *ConfigEncryptor) Encrypt(config map[string]any) ([]byte, []byte, string, error) {
	plaintext, err := json.Marshal(config)
	if err != nil {
		return nil, nil, "", fmt.Errorf("marshal config: %w", err)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, nil, "", fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, "", fmt.Errorf("new gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, e.keyID, nil
}

func (e *ConfigEncryptor) Decrypt(ciphertext, nonce []byte) (map[string]any, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("invalid nonce size")
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt config: %w", err)
	}

	var config map[string]any
	if err := json.Unmarshal(plaintext, &config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if config == nil {
		config = map[string]any{}
	}
	return config, nil
}

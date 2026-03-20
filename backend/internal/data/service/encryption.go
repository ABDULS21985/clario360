package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

type ConfigEncryptor struct {
	masterKey []byte
	keyID     string
}

func NewConfigEncryptor(masterKeyBase64 string) (*ConfigEncryptor, error) {
	key, err := base64.StdEncoding.DecodeString(masterKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode encryption key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be exactly 32 bytes, got %d", len(key))
	}
	hash := sha256.Sum256(key)
	return &ConfigEncryptor{
		masterKey: key,
		keyID:     fmt.Sprintf("%x", hash[:])[:8],
	}, nil
}

func NewConfigEncryptorFromBytes(key []byte) (*ConfigEncryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be exactly 32 bytes, got %d", len(key))
	}
	hash := sha256.Sum256(key)
	copyKey := make([]byte, len(key))
	copy(copyKey, key)
	return &ConfigEncryptor{
		masterKey: copyKey,
		keyID:     fmt.Sprintf("%x", hash[:])[:8],
	}, nil
}

func (e *ConfigEncryptor) KeyID() string {
	return e.keyID
}

func (e *ConfigEncryptor) Encrypt(plaintext []byte) ([]byte, string, error) {
	defer zeroBytes(plaintext)

	block, err := aes.NewCipher(e.masterKey)
	if err != nil {
		return nil, "", fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, "", fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, e.keyID, nil
}

func (e *ConfigEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.masterKey)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce := ciphertext[:nonceSize]
	encrypted := ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt config: %w", err)
	}
	return plaintext, nil
}

func zeroBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
}

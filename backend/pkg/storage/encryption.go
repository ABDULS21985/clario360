package storage

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// EncryptionMetadata stores the encrypted DEK and nonce for a file.
type EncryptionMetadata struct {
	Algorithm    string `json:"algorithm"`     // "AES-256-GCM"
	EncryptedDEK string `json:"encrypted_dek"` // base64-encoded encrypted DEK
	Nonce        string `json:"nonce"`          // base64-encoded nonce for DEK encryption
	KeyID        string `json:"key_id"`         // KEK identifier for key rotation
}

// Encryptor performs AES-256-GCM envelope encryption.
type Encryptor struct {
	kek   []byte // 32-byte Key Encryption Key
	keyID string
}

// NewEncryptor creates an encryptor with the given master key (KEK).
// The key must be exactly 32 bytes for AES-256.
func NewEncryptor(masterKey []byte, keyID string) (*Encryptor, error) {
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("encryption: master key must be exactly 32 bytes, got %d", len(masterKey))
	}
	keyCopy := make([]byte, 32)
	copy(keyCopy, masterKey)
	return &Encryptor{kek: keyCopy, keyID: keyID}, nil
}

// Encrypt encrypts content using envelope encryption.
// Returns the ciphertext as bytes, its size, and the encryption metadata.
// The DEK is zeroed after use.
func (e *Encryptor) Encrypt(content io.Reader) ([]byte, int64, *EncryptionMetadata, error) {
	// Read all plaintext
	plaintext, err := io.ReadAll(content)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("encryption: reading content: %w", err)
	}

	// Generate random DEK (32 bytes from crypto/rand)
	dek := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, 0, nil, fmt.Errorf("encryption: generating DEK: %w", err)
	}
	// Ensure DEK is zeroed after use
	defer func() {
		for i := range dek {
			dek[i] = 0
		}
	}()

	// Encrypt content with DEK
	ciphertext, err := aesGCMEncrypt(dek, plaintext)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("encryption: encrypting content: %w", err)
	}

	// Encrypt DEK with KEK (envelope)
	encryptedDEK, dekNonce, err := aesGCMEncryptDEK(e.kek, dek)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("encryption: encrypting DEK: %w", err)
	}

	meta := &EncryptionMetadata{
		Algorithm:    "AES-256-GCM",
		EncryptedDEK: base64.StdEncoding.EncodeToString(encryptedDEK),
		Nonce:        base64.StdEncoding.EncodeToString(dekNonce),
		KeyID:        e.keyID,
	}

	return ciphertext, int64(len(ciphertext)), meta, nil
}

// Decrypt decrypts content using envelope encryption metadata.
// Returns the plaintext as a reader.
func (e *Encryptor) Decrypt(content io.Reader, meta *EncryptionMetadata) (io.Reader, error) {
	if meta == nil {
		return nil, errors.New("encryption: metadata is nil")
	}
	if meta.Algorithm != "AES-256-GCM" {
		return nil, fmt.Errorf("encryption: unsupported algorithm %q", meta.Algorithm)
	}

	encryptedDEK, err := base64.StdEncoding.DecodeString(meta.EncryptedDEK)
	if err != nil {
		return nil, fmt.Errorf("encryption: decoding encrypted DEK: %w", err)
	}
	dekNonce, err := base64.StdEncoding.DecodeString(meta.Nonce)
	if err != nil {
		return nil, fmt.Errorf("encryption: decoding nonce: %w", err)
	}

	// Decrypt DEK with KEK
	dek, err := aesGCMDecryptDEK(e.kek, encryptedDEK, dekNonce)
	if err != nil {
		return nil, fmt.Errorf("encryption: decrypting DEK: %w", err)
	}
	defer func() {
		for i := range dek {
			dek[i] = 0
		}
	}()

	// Read ciphertext
	ciphertext, err := io.ReadAll(content)
	if err != nil {
		return nil, fmt.Errorf("encryption: reading ciphertext: %w", err)
	}

	// Decrypt content with DEK
	plaintext, err := aesGCMDecrypt(dek, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("encryption: decrypting content: %w", err)
	}

	return bytes.NewReader(plaintext), nil
}

// aesGCMEncrypt encrypts plaintext with AES-256-GCM.
// Returns nonce prepended to ciphertext.
func aesGCMEncrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	// nonce is prepended to ciphertext
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// aesGCMDecrypt decrypts AES-256-GCM ciphertext with prepended nonce.
func aesGCMDecrypt(key, ciphertextWithNonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertextWithNonce) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce := ciphertextWithNonce[:nonceSize]
	ciphertext := ciphertextWithNonce[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// aesGCMEncryptDEK encrypts the DEK with the KEK, returning ciphertext and nonce separately.
func aesGCMEncryptDEK(kek, dek []byte) (encryptedDEK, nonce []byte, err error) {
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	encryptedDEK = gcm.Seal(nil, nonce, dek, nil)
	return encryptedDEK, nonce, nil
}

// aesGCMDecryptDEK decrypts the DEK using the KEK with the provided nonce.
func aesGCMDecryptDEK(kek, encryptedDEK, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, encryptedDEK, nil)
}

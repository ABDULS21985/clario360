package storage

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"io"
	"testing"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		t.Fatalf("generating test key: %v", err)
	}
	return key
}

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	key := testKey(t)
	enc, err := NewEncryptor(key, "test-key-1")
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}

	plaintext := []byte("hello, envelope encryption!")
	ciphertext, size, meta, err := enc.Encrypt(bytes.NewReader(plaintext))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if size != int64(len(ciphertext)) {
		t.Fatalf("size mismatch: got %d, want %d", size, len(ciphertext))
	}
	if meta.Algorithm != "AES-256-GCM" {
		t.Fatalf("unexpected algorithm: %s", meta.Algorithm)
	}
	if meta.KeyID != "test-key-1" {
		t.Fatalf("unexpected key ID: %s", meta.KeyID)
	}

	reader, err := enc.Decrypt(bytes.NewReader(ciphertext), meta)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	decrypted, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading decrypted: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted content mismatch:\n  got:  %q\n  want: %q", decrypted, plaintext)
	}
}

func TestEncryptDecrypt_DifferentKeys_Fails(t *testing.T) {
	key1 := testKey(t)
	key2 := testKey(t)

	enc1, err := NewEncryptor(key1, "key-1")
	if err != nil {
		t.Fatalf("NewEncryptor key1: %v", err)
	}
	enc2, err := NewEncryptor(key2, "key-2")
	if err != nil {
		t.Fatalf("NewEncryptor key2: %v", err)
	}

	plaintext := []byte("secret data")
	ciphertext, _, meta, err := enc1.Encrypt(bytes.NewReader(plaintext))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Decrypt with different key should fail
	_, err = enc2.Decrypt(bytes.NewReader(ciphertext), meta)
	if err == nil {
		t.Fatal("expected error when decrypting with different key, got nil")
	}
}

func TestEncryptDecrypt_TamperedCiphertext_Fails(t *testing.T) {
	key := testKey(t)
	enc, err := NewEncryptor(key, "key-1")
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}

	plaintext := []byte("tamper-proof data")
	ciphertext, _, meta, err := enc.Encrypt(bytes.NewReader(plaintext))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Tamper with the ciphertext (flip a byte in the middle)
	tampered := make([]byte, len(ciphertext))
	copy(tampered, ciphertext)
	tampered[len(tampered)/2] ^= 0xff

	_, err = enc.Decrypt(bytes.NewReader(tampered), meta)
	if err == nil {
		t.Fatal("expected error when decrypting tampered ciphertext, got nil")
	}
}

func TestEncryptDecrypt_TamperedDEK_Fails(t *testing.T) {
	key := testKey(t)
	enc, err := NewEncryptor(key, "key-1")
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}

	plaintext := []byte("dek-tamper test")
	ciphertext, _, meta, err := enc.Encrypt(bytes.NewReader(plaintext))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Tamper with the encrypted DEK
	dekBytes, err := base64.StdEncoding.DecodeString(meta.EncryptedDEK)
	if err != nil {
		t.Fatalf("decoding encrypted DEK: %v", err)
	}
	dekBytes[0] ^= 0xff
	meta.EncryptedDEK = base64.StdEncoding.EncodeToString(dekBytes)

	_, err = enc.Decrypt(bytes.NewReader(ciphertext), meta)
	if err == nil {
		t.Fatal("expected error when decrypting with tampered DEK, got nil")
	}
}

func TestEncryptDecrypt_InvalidKeyLength(t *testing.T) {
	tests := []struct {
		name   string
		keyLen int
	}{
		{"too short (16 bytes)", 16},
		{"too long (64 bytes)", 64},
		{"empty", 0},
		{"one byte", 1},
		{"31 bytes", 31},
		{"33 bytes", 33},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keyLen)
			_, err := NewEncryptor(key, "bad-key")
			if err == nil {
				t.Fatalf("expected error for key length %d, got nil", tt.keyLen)
			}
		})
	}
}

func TestEncryptDecrypt_EmptyContent(t *testing.T) {
	key := testKey(t)
	enc, err := NewEncryptor(key, "key-1")
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}

	plaintext := []byte{}
	ciphertext, _, meta, err := enc.Encrypt(bytes.NewReader(plaintext))
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}

	reader, err := enc.Decrypt(bytes.NewReader(ciphertext), meta)
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}
	decrypted, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading decrypted empty: %v", err)
	}
	if len(decrypted) != 0 {
		t.Fatalf("expected empty decrypted content, got %d bytes", len(decrypted))
	}
}

func TestEncryptDecrypt_LargeContent(t *testing.T) {
	key := testKey(t)
	enc, err := NewEncryptor(key, "key-1")
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}

	// 1 MB of random data
	plaintext := make([]byte, 1024*1024)
	if _, err := io.ReadFull(rand.Reader, plaintext); err != nil {
		t.Fatalf("generating random content: %v", err)
	}

	ciphertext, size, meta, err := enc.Encrypt(bytes.NewReader(plaintext))
	if err != nil {
		t.Fatalf("Encrypt 1MB: %v", err)
	}
	if size != int64(len(ciphertext)) {
		t.Fatalf("size mismatch: got %d, want %d", size, len(ciphertext))
	}

	reader, err := enc.Decrypt(bytes.NewReader(ciphertext), meta)
	if err != nil {
		t.Fatalf("Decrypt 1MB: %v", err)
	}
	decrypted, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading decrypted 1MB: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatal("1MB decrypted content does not match original")
	}
}

func TestEncryptDecrypt_NonceUniqueness(t *testing.T) {
	key := testKey(t)
	enc, err := NewEncryptor(key, "key-1")
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}

	plaintext := []byte("same content both times")

	ct1, _, meta1, err := enc.Encrypt(bytes.NewReader(plaintext))
	if err != nil {
		t.Fatalf("first Encrypt: %v", err)
	}

	ct2, _, meta2, err := enc.Encrypt(bytes.NewReader(plaintext))
	if err != nil {
		t.Fatalf("second Encrypt: %v", err)
	}

	// The ciphertexts should differ because the nonce/DEK are random each time
	if bytes.Equal(ct1, ct2) {
		t.Fatal("two encryptions of the same content produced identical ciphertext")
	}

	// The encrypted DEKs should also differ
	if meta1.EncryptedDEK == meta2.EncryptedDEK {
		t.Fatal("two encryptions produced the same encrypted DEK")
	}

	// Both should still decrypt correctly
	for i, pair := range []struct {
		ct   []byte
		meta *EncryptionMetadata
	}{
		{ct1, meta1},
		{ct2, meta2},
	} {
		reader, err := enc.Decrypt(bytes.NewReader(pair.ct), pair.meta)
		if err != nil {
			t.Fatalf("Decrypt #%d: %v", i+1, err)
		}
		decrypted, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("reading decrypted #%d: %v", i+1, err)
		}
		if !bytes.Equal(decrypted, plaintext) {
			t.Fatalf("decrypted #%d does not match original", i+1)
		}
	}
}

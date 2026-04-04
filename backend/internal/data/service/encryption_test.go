package service

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	encryptor := mustEncryptor(t, bytes.Repeat([]byte{0x42}, 32))
	plaintext := []byte(`{"host":"db.internal","password":"super-secret"}`)
	original := append([]byte(nil), plaintext...)

	ciphertext, keyID, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if keyID != encryptor.KeyID() {
		t.Fatalf("Encrypt() keyID = %q, want %q", keyID, encryptor.KeyID())
	}
	decrypted, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if !bytes.Equal(decrypted, original) {
		t.Fatalf("Decrypt() = %q, want %q", decrypted, original)
	}
}

func TestEncrypt_DifferentNonces(t *testing.T) {
	encryptor := mustEncryptor(t, bytes.Repeat([]byte{0x24}, 32))

	plaintextA := []byte(`{"token":"a"}`)
	plaintextB := []byte(`{"token":"a"}`)

	cipherA, _, err := encryptor.Encrypt(plaintextA)
	if err != nil {
		t.Fatalf("Encrypt() first error = %v", err)
	}
	cipherB, _, err := encryptor.Encrypt(plaintextB)
	if err != nil {
		t.Fatalf("Encrypt() second error = %v", err)
	}
	if bytes.Equal(cipherA, cipherB) {
		t.Fatal("Encrypt() produced identical ciphertext for identical plaintext")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	encryptorA := mustEncryptor(t, bytes.Repeat([]byte{0x11}, 32))
	encryptorB := mustEncryptor(t, bytes.Repeat([]byte{0x22}, 32))

	ciphertext, _, err := encryptorA.Encrypt([]byte(`{"api_key":"secret"}`))
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if _, err := encryptorB.Decrypt(ciphertext); err == nil {
		t.Fatal("Decrypt() with wrong key returned nil error")
	}
}

func TestDecrypt_Corrupted(t *testing.T) {
	encryptor := mustEncryptor(t, bytes.Repeat([]byte{0x33}, 32))

	ciphertext, _, err := encryptor.Encrypt([]byte(`{"password":"secret"}`))
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	ciphertext[len(ciphertext)-1] ^= 0xFF

	if _, err := encryptor.Decrypt(ciphertext); err == nil {
		t.Fatal("Decrypt() on corrupted ciphertext returned nil error")
	}
}

func TestEncrypt_ZerosPlaintext(t *testing.T) {
	encryptor := mustEncryptor(t, bytes.Repeat([]byte{0x44}, 32))
	plaintext := []byte(`{"username":"alice","password":"secret"}`)

	if _, _, err := encryptor.Encrypt(plaintext); err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	for i, b := range plaintext {
		if b != 0 {
			t.Fatalf("plaintext byte %d = %d, want 0", i, b)
		}
	}
}

func TestKeyID_Consistent(t *testing.T) {
	key := bytes.Repeat([]byte{0x55}, 32)
	first := mustEncryptor(t, key)
	second := mustEncryptorBase64(t, base64.StdEncoding.EncodeToString(key))

	if first.KeyID() != second.KeyID() {
		t.Fatalf("KeyID mismatch: %q != %q", first.KeyID(), second.KeyID())
	}
}

func TestKeyID_DifferentKeys(t *testing.T) {
	first := mustEncryptor(t, bytes.Repeat([]byte{0x66}, 32))
	second := mustEncryptor(t, bytes.Repeat([]byte{0x77}, 32))

	if first.KeyID() == second.KeyID() {
		t.Fatalf("different keys produced same key id %q", first.KeyID())
	}
}

func mustEncryptor(t *testing.T, key []byte) *ConfigEncryptor {
	t.Helper()
	encryptor, err := NewConfigEncryptorFromBytes(key)
	if err != nil {
		t.Fatalf("NewConfigEncryptorFromBytes() error = %v", err)
	}
	return encryptor
}

func mustEncryptorBase64(t *testing.T, key string) *ConfigEncryptor {
	t.Helper()
	encryptor, err := NewConfigEncryptor(key)
	if err != nil {
		t.Fatalf("NewConfigEncryptor() error = %v", err)
	}
	return encryptor
}

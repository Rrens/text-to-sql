package security_test

import (
	"testing"

	"github.com/rensmac/text-to-sql/internal/security"
)

func TestEncryptor_EncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	encryptor, err := security.NewEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{"empty", ""},
		{"short", "hello"},
		{"medium", "this is a medium length string for testing"},
		{"long", "this is a much longer string that contains more data and should still work correctly with the encryption and decryption process"},
		{"special", "special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"unicode", "unicode: 日本語 中文 한국어 🎉"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := encryptor.Encrypt([]byte(tt.plaintext))
			if err != nil {
				t.Fatalf("encrypt failed: %v", err)
			}

			decrypted, err := encryptor.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("decrypt failed: %v", err)
			}

			if string(decrypted) != tt.plaintext {
				t.Errorf("decrypted text does not match: got %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptor_EncryptString(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	encryptor, err := security.NewEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	plaintext := "password123"
	ciphertext, err := encryptor.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("encrypt string failed: %v", err)
	}

	// Ciphertext should be base64 encoded
	if len(ciphertext) == 0 {
		t.Error("ciphertext is empty")
	}

	decrypted, err := encryptor.DecryptString(ciphertext)
	if err != nil {
		t.Fatalf("decrypt string failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("decrypted text does not match: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptor_InvalidKeyLength(t *testing.T) {
	invalidKeys := [][]byte{
		make([]byte, 0),
		make([]byte, 15),
		make([]byte, 17),
		make([]byte, 31),
		make([]byte, 33),
	}

	for _, key := range invalidKeys {
		_, err := security.NewEncryptor(key)
		if err == nil {
			t.Errorf("expected error for key length %d, got nil", len(key))
		}
	}
}

func TestEncryptor_ValidKeyLengths(t *testing.T) {
	validKeys := []int{16, 24, 32}

	for _, keyLen := range validKeys {
		key := make([]byte, keyLen)
		_, err := security.NewEncryptor(key)
		if err != nil {
			t.Errorf("unexpected error for key length %d: %v", keyLen, err)
		}
	}
}

func TestEncryptor_DifferentCiphertexts(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	encryptor, _ := security.NewEncryptor(key)
	plaintext := []byte("same plaintext")

	ciphertext1, _ := encryptor.Encrypt(plaintext)
	ciphertext2, _ := encryptor.Encrypt(plaintext)

	// Same plaintext should produce different ciphertexts (due to random nonce)
	if string(ciphertext1) == string(ciphertext2) {
		t.Error("expected different ciphertexts for same plaintext")
	}
}

func TestGenerateKey(t *testing.T) {
	key1, err := security.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	if len(key1) != 32 {
		t.Errorf("expected key length 32, got %d", len(key1))
	}

	key2, _ := security.GenerateKey()

	// Keys should be different
	if string(key1) == string(key2) {
		t.Error("expected different keys")
	}
}

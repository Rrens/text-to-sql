package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

// Encryptor handles credential encryption/decryption
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new encryptor with the given key
// Key must be 16, 24, or 32 bytes for AES-128, AES-192, or AES-256
func NewEncryptor(key []byte) (*Encryptor, error) {
	keyLen := len(key)
	if keyLen != 16 && keyLen != 24 && keyLen != 32 {
		return nil, fmt.Errorf("invalid key length: %d (must be 16, 24, or 32)", keyLen)
	}
	return &Encryptor{key: key}, nil
}

// NewEncryptorFromBase64 creates an encryptor from a base64-encoded key
func NewEncryptorFromBase64(keyBase64 string) (*Encryptor, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key: %w", err)
	}
	return NewEncryptor(key)
}

// GenerateKey generates a new random encryption key
func GenerateKey() ([]byte, error) {
	key := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// Encrypt encrypts data using AES-GCM
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-GCM
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptJSON encrypts a struct as JSON
func (e *Encryptor) EncryptJSON(v any) ([]byte, error) {
	plaintext, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return e.Encrypt(plaintext)
}

// DecryptJSON decrypts and unmarshals JSON into a struct
func (e *Encryptor) DecryptJSON(ciphertext []byte, v any) error {
	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return err
	}
	return json.Unmarshal(plaintext, v)
}

// EncryptString encrypts and returns base64-encoded ciphertext
func (e *Encryptor) EncryptString(plaintext string) (string, error) {
	ciphertext, err := e.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString decrypts base64-encoded ciphertext
func (e *Encryptor) DecryptString(ciphertextBase64 string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

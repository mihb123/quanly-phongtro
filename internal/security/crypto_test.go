package security_test

import (
	"crypto/rand"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/security"
)

func TestEncryptDecrypt(t *testing.T) {
	// Generate random 32 byte key
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	plaintext := "my-secret-bot-token-123456"

	// Encrypt
	ciphertext, err := security.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Make sure ciphertext isn't empty or equal to plaintext
	if ciphertext == "" || ciphertext == plaintext {
		t.Fatalf("Ciphertext is invalid: %v", ciphertext)
	}

	// Encrypt again to ensure nonce makes it different
	ciphertext2, err := security.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt 2 failed: %v", err)
	}
	if ciphertext == ciphertext2 {
		t.Fatalf("Expected different ciphertexts due to random nonce")
	}

	// Decrypt
	decrypted, err := security.Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("Expected %v, got %v", plaintext, decrypted)
	}
}

func TestDecrypt_InvalidKey(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)
	
	plaintext := "test"
	ciphertext, _ := security.Encrypt(plaintext, key)

	wrongKey := make([]byte, 32)
	rand.Read(wrongKey)

	_, err := security.Decrypt(ciphertext, wrongKey)
	if err == nil {
		t.Fatalf("Expected error when decrypting with wrong key")
	}
}

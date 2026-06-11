package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

var (
	ErrInvalidKeySize = errors.New("invalid key size: must be 32 bytes for AES-256")
	ErrCiphertextTooShort = errors.New("ciphertext too short")
)

// Encrypt encrypts a plaintext string using AES-256-GCM and returns a base64 encoded string.
func Encrypt(plaintext string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", ErrInvalidKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64 encoded ciphertext string using AES-256-GCM and returns the plaintext.
func Decrypt(encodedCiphertext string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", ErrInvalidKeySize
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encodedCiphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", ErrCiphertextTooShort
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintextBytes, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintextBytes), nil
}

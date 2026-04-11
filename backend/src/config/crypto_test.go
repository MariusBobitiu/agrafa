package config

import (
	"strings"
	"testing"
)

func TestEncryptorEncryptsAndDecrypts(t *testing.T) {
	t.Parallel()

	encryptor, err := NewEncryptor("super-secret-root")
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}

	ciphertext, err := encryptor.Encrypt("re_test_secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if ciphertext == "re_test_secret" {
		t.Fatal("Encrypt() returned plaintext")
	}
	if strings.Contains(ciphertext, "re_test_secret") {
		t.Fatalf("ciphertext unexpectedly contains plaintext: %q", ciphertext)
	}

	plaintext, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if plaintext != "re_test_secret" {
		t.Fatalf("Decrypt() = %q, want re_test_secret", plaintext)
	}
}

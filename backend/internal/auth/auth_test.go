package auth

import (
	"strings"
	"testing"
)

// --- Password hashing tests ---

func TestHashPassword_RoundTrip(t *testing.T) {
	hash, err := HashPassword("test-password-123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	ok, err := VerifyPassword("test-password-123", hash)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Fatal("expected password to verify")
	}
}

func TestHashPassword_WrongPassword(t *testing.T) {
	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	ok, err := VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if ok {
		t.Fatal("expected wrong password to fail")
	}
}

func TestHashPassword_Empty(t *testing.T) {
	_, err := HashPassword("")
	if err == nil {
		t.Fatal("expected error for empty password")
	}
}

func TestVerifyPassword_EmptyInputs(t *testing.T) {
	_, err := VerifyPassword("", "some-hash")
	if err == nil {
		t.Fatal("expected error for empty password")
	}
	_, err = VerifyPassword("password", "")
	if err == nil {
		t.Fatal("expected error for empty hash")
	}
}

func TestVerifyPassword_InvalidFormat(t *testing.T) {
	_, err := VerifyPassword("password", "not-a-valid-hash")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestHashPassword_UniquePerCall(t *testing.T) {
	h1, _ := HashPassword("same-password")
	h2, _ := HashPassword("same-password")
	if h1 == h2 {
		t.Fatal("expected different salts to produce different hashes")
	}
}

func TestHashPassword_FormatPrefix(t *testing.T) {
	hash, _ := HashPassword("password")
	if !strings.HasPrefix(hash, "argon2id$") {
		t.Fatalf("expected argon2id$ prefix, got %q", hash[:20])
	}
}

// --- API key generation tests ---

func TestGenerateAPIKey_Format(t *testing.T) {
	prefix, secret, token, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prefix) != apiKeyPrefixLength {
		t.Fatalf("prefix length: expected %d, got %d", apiKeyPrefixLength, len(prefix))
	}
	if len(secret) != apiKeySecretLength {
		t.Fatalf("secret length: expected %d, got %d", apiKeySecretLength, len(secret))
	}
	expectedToken := "sk-" + prefix + "." + secret
	if token != expectedToken {
		t.Fatalf("token format mismatch: got %q", token)
	}
}

func TestGenerateAPIKey_Unique(t *testing.T) {
	_, _, token1, _ := GenerateAPIKey()
	_, _, token2, _ := GenerateAPIKey()
	if token1 == token2 {
		t.Fatal("expected unique tokens")
	}
}

func TestGenerateAPIKey_AlphabetOnly(t *testing.T) {
	for i := 0; i < 10; i++ {
		prefix, secret, _, _ := GenerateAPIKey()
		for _, c := range prefix + secret {
			if !strings.ContainsRune(alphabet, c) {
				t.Fatalf("unexpected char %q in key", string(c))
			}
		}
	}
}

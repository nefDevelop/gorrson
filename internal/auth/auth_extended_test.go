package auth

import (
	"os"
	"strings"
	"testing"
)

func TestPasswordEmpty(t *testing.T) {
	hash, err := HashPassword("")
	if err != nil {
		t.Fatalf("HashPassword with empty string failed: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("Expected argon2id prefix for empty password, got %s", hash)
	}
	if !CheckPasswordHash("", hash) {
		t.Error("CheckPasswordHash should work for empty password")
	}
	if CheckPasswordHash("not-empty", hash) {
		t.Error("CheckPasswordHash should fail for non-matching password")
	}
}

func TestPasswordUnicode(t *testing.T) {
	hash, err := HashPassword("contraseña-café-🔑")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if !CheckPasswordHash("contraseña-café-🔑", hash) {
		t.Error("CheckPasswordHash failed for unicode password")
	}
	if CheckPasswordHash("wrong", hash) {
		t.Error("CheckPasswordHash should fail for wrong password")
	}
}

func TestRefreshTokenUniqueness(t *testing.T) {
	tokens := make(map[string]bool)
	for i := 0; i < 3; i++ {
		ref, err := GenerateRefreshToken()
		if err != nil {
			t.Fatalf("GenerateRefreshToken failed: %v", err)
		}
		if tokens[ref] {
			t.Errorf("Duplicate refresh token generated: %s", ref)
		}
		tokens[ref] = true
	}
}

func TestGetSecretEnvOverride(t *testing.T) {
	origEnv := os.Getenv("GORRSON_JWT_SECRET")
	defer func() {
		if origEnv == "" {
			os.Unsetenv("GORRSON_JWT_SECRET")
		} else {
			os.Setenv("GORRSON_JWT_SECRET", origEnv)
		}
	}()

	customSecret := "my-custom-secret-key-for-testing"
	os.Setenv("GORRSON_JWT_SECRET", customSecret)

	secret := GetSecret()
	if string(secret) != customSecret {
		t.Errorf("GetSecret() = %s, want %s", string(secret), customSecret)
	}

	token, err := GenerateToken("env-user")
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	username, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if username != "env-user" {
		t.Errorf("Username = %s, want env-user", username)
	}
}

func TestCheckPasswordHashInvalidFormat(t *testing.T) {
	if CheckPasswordHash("pass", "$argon2id$short") {
		t.Error("Should return false for invalid hash format")
	}
	if CheckPasswordHash("pass", "plain-text-hash") {
		t.Error("Should return false for non-argon2 hash")
	}
}

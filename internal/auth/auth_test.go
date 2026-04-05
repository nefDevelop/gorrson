package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestArgon2idHashing(t *testing.T) {
	password := "my-secret-password"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("Expected argon2id prefix, got %s", hash)
	}

	if !CheckPasswordHash(password, hash) {
		t.Error("CheckPasswordHash failed for correct password")
	}

	if CheckPasswordHash("wrong-password", hash) {
		t.Error("CheckPasswordHash succeeded for wrong password")
	}
}

func TestTokens(t *testing.T) {
	username := "testuser"
	
	// Test Access Token
	token, err := GenerateToken(username)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	validatedUser, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if validatedUser != username {
		t.Errorf("Expected username %s, got %s", username, validatedUser)
	}

	// Verify expiration (should be around 24 hours)
	parsedToken, _ := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return GetSecret(), nil
	})
	claims := parsedToken.Claims.(jwt.MapClaims)
	exp := time.Unix(int64(claims["exp"].(float64)), 0)
	duration := time.Until(exp)
	
	if duration < 23*time.Hour || duration > 25*time.Hour {
		t.Errorf("Expected ~24h expiration, got %v", duration)
	}

	// Test Refresh Token
	ref, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}
	if len(ref) < 32 {
		t.Errorf("Refresh token too short: %s", ref)
	}
}

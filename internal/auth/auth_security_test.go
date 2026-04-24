package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestTokenExpirationFail(t *testing.T) {
	// Generate a token that is already expired
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": "testuser",
		"exp":      time.Now().Add(-time.Hour).Unix(), // 1 hour ago
	})

	tokenString, err := token.SignedString(GetSecret())
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	_, err = ValidateToken(tokenString)
	if err == nil {
		t.Error("Expected error for expired token, got nil")
	}
}

func TestTokenTampering(t *testing.T) {
	username := "testuser"
	tokenString, _ := GenerateToken(username)

	// Tamper with the token (e.g., change a character in the payload)
	// JWT is header.payload.signature
	parts := 3
	// We'll just change the last character of the payload part if possible
	// or just append something to the signature
	tampered := tokenString + "extra"

	_, err := ValidateToken(tampered)
	if err == nil {
		t.Error("Expected error for tampered token, got nil")
	}
}

func TestInvalidSigningMethod(t *testing.T) {
	// Try to use None algorithm (unsecured)
	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"username": "testuser",
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	
	// jwt.SigningMethodNone.Sign doesn't actually sign, it just returns empty string for signature
	tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	_, err := ValidateToken(tokenString)
	if err == nil {
		t.Error("Expected error for 'None' signing method, got nil")
	}
}

func TestEmptyToken(t *testing.T) {
	_, err := ValidateToken("")
	if err == nil {
		t.Error("Expected error for empty token, got nil")
	}
}

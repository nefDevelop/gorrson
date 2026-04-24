package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	s := NewServer("test_data", true)
	handler := s.securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("Missing X-Content-Type-Options: nosniff")
	}
	if rr.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Error("Missing X-Frame-Options: SAMEORIGIN")
	}
	if !strings.Contains(rr.Header().Get("Content-Security-Policy"), "default-src 'self'") {
		t.Error("Content-Security-Policy is not strict enough")
	}
}

func TestProxySecurityHeaders(t *testing.T) {
	s := NewServer("test_data", true)
	handler := s.securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/proxy/page?url=http://example.com", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Proxy should have slightly different CSP but still SAMEORIGIN
	if rr.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Errorf("Proxy should still be SAMEORIGIN, got %s", rr.Header().Get("X-Frame-Options"))
	}
	if !strings.Contains(rr.Header().Get("Content-Security-Policy"), "unsafe-inline") {
		t.Error("Proxy CSP should allow unsafe-inline for compatibility")
	}
}

func TestAuthMiddlewareRejection(t *testing.T) {
	s := NewServer("test_data", true)
	handler := s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/news", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized for missing token, got %d", rr.Code)
	}
}

func TestJSONBodyLimit(t *testing.T) {
	s := NewServer("test_data", true)

	// Create a large payload (> 1MB)
	largeBody := strings.Repeat("a", (1024*1024)+100)
	reqBody := `{"username": "test", "password": "` + largeBody + `"}`

	req := httptest.NewRequest("POST", "/login", strings.NewReader(reqBody))
	rr := httptest.NewRecorder()

	s.handleLogin(rr, req)

	// Should fail due to body limit in decodeJSONBody
	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected error for oversized JSON body, got %d", rr.Code)
	}
}

func TestIsValidUsernameIntegration(t *testing.T) {
	// Although tested in sanitizer, we verify how it would be used in a simulated request if needed.
	// This is a placeholder for where integration logic would go.
}

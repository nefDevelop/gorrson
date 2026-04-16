package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestSecurity_PathTraversal(t *testing.T) {
	dataDir := "./test_security"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)

	// Try to register/login with suspicious username
	evilUser := "../../etc/passwd"
	loginBody := `{"username":"` + evilUser + `", "password":"password"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(loginBody))
	rr := httptest.NewRecorder()

	srv.router.ServeHTTP(rr, req)

	// Should be rejected by NewStore/IsValidUsername
	if rr.Code != http.StatusNotFound && rr.Code != http.StatusBadRequest {
		t.Errorf("Expected rejection for evil username, got %d", rr.Code)
	}

	// Verify no file was created outside dataDir
	if _, err := os.Stat("../../etc/passwd.db"); err == nil {
		t.Error("Vulnerability found: evil database file created at root!")
		_ = os.Remove("../../etc/passwd.db")
	}
}

func TestSecurity_Headers(t *testing.T) {
	dataDir := "./test_headers"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)
	req := httptest.NewRequest("GET", "/login", nil) // Login is public
	rr := httptest.NewRecorder()

	srv.router.ServeHTTP(rr, req)

	headers := []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"X-XSS-Protection",
		"Referrer-Policy",
		"Content-Security-Policy",
	}

	for _, h := range headers {
		if rr.Header().Get(h) == "" {
			t.Errorf("Security header %s missing", h)
		}
	}
}

func TestSecurity_SSRF(t *testing.T) {
	// This test verifies that the local IP ranges are blocked
	dataDir := "./test_ssrf"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)

	// We use the proxy endpoint which uses SecureClient
	// http://localhost:8080/proxy?url=http://127.0.0.1:22
	req := httptest.NewRequest("GET", "/proxy?url=http://127.0.0.1:22", nil)
	rr := httptest.NewRecorder()

	srv.router.ServeHTTP(rr, req)

	// Should be blocked
	if rr.Code == http.StatusOK {
		t.Error("SSRF vulnerability: access to 127.0.0.1 should be blocked")
	}

	if !strings.Contains(rr.Body.String(), "blocked") && rr.Code != http.StatusInternalServerError {
		t.Logf("Response body: %s", rr.Body.String())
	}
}

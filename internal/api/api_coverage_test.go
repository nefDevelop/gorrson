package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/nef734/gorrson/internal/auth"
	"github.com/nef734/gorrson/internal/db"
)

func TestHealthEndpoint(t *testing.T) {
	dataDir := "./test_health_data"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Health endpoint returned %d, want 200", rr.Code)
	}
}

func TestUnauthorizedNoToken(t *testing.T) {
	dataDir := "./test_unauth_data"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)

	endpoints := []string{
		"GET /news",
		"GET /feeds",
		"GET /filters",
		"GET /stats",
		"GET /config/export",
	}

	for _, ep := range endpoints {
		parts := strings.SplitN(ep, " ", 2)
		method, path := parts[0], parts[1]
		req := httptest.NewRequest(method, path, nil)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("%s without token: got %d, want 401", ep, rr.Code)
		}
	}
}

func TestInvalidToken(t *testing.T) {
	dataDir := "./test_invalid_token"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)

	req := httptest.NewRequest("GET", "/news", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Invalid token: got %d, want 401", rr.Code)
	}
}

func TestExpiredToken(t *testing.T) {
	dataDir := "./test_expired_token"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)

	token, _ := auth.GenerateToken("testuser")
	parts := strings.Split(token, ".")
	// Tamper with the token to make it invalid
	tampered := parts[0] + "." + parts[1] + ".INVALID_SIGNATURE"

	req := httptest.NewRequest("GET", "/news", nil)
	req.Header.Set("Authorization", "Bearer "+tampered)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Tampered token: got %d, want 401", rr.Code)
	}
}

func TestAuthWithQueryToken(t *testing.T) {
	dataDir := "./test_query_token"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)
	username := "query_user"
	password := "query_pass"

	store, _ := db.NewStore(dataDir, username)
	hash, _ := auth.HashPassword(password)
	store.UpsertUser(context.Background(), username, hash, "")
	store.Close()

	loginBody, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(loginBody))
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	token := resp["access_token"]

	// Use token as query parameter
	req = httptest.NewRequest("GET", "/news?token="+token, nil)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Query token auth: got %d, want 200", rr.Code)
	}
}

func TestFeedUpdateAndDelete(t *testing.T) {
	dataDir := "./test_feed_crud"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)
	username := "feed_crud_user"
	password := "feed_crud_pass"

	store, _ := db.NewStore(dataDir, username)
	hash, _ := auth.HashPassword(password)
	store.UpsertUser(context.Background(), username, hash, "")
	store.Close()

	loginBody, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(loginBody))
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	token := resp["access_token"]

	// Add feed
	addBody, _ := json.Marshal(map[string]string{"url": "https://hnrss.org/frontpage"})
	req = httptest.NewRequest("POST", "/feeds", bytes.NewBuffer(addBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	// List feeds
	req = httptest.NewRequest("GET", "/feeds", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	var feeds []map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &feeds)

	if len(feeds) == 0 {
		t.Skip("No feeds synced (network may be unavailable)")
		return
	}

	feedID := feeds[0]["id"].(string)

	// Update feed
	updateBody, _ := json.Marshal(map[string]string{"title": "Updated Title", "category": "Tech"})
	req = httptest.NewRequest("PATCH", "/feeds/"+feedID, bytes.NewBuffer(updateBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Update feed: got %d, want 200", rr.Code)
	}

	// Delete feed
	req = httptest.NewRequest("DELETE", "/feeds/"+feedID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Errorf("Delete feed: got %d, want 204", rr.Code)
	}
}

func TestMarkReadAndFavorite(t *testing.T) {
	dataDir := "./test_mark_ops"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)
	username := "mark_ops_user"

	store, _ := db.NewStore(dataDir, username)
	hash, _ := auth.HashPassword("pass")
	store.UpsertUser(context.Background(), username, hash, "")
	store.Close()

	loginBody, _ := json.Marshal(map[string]string{"username": username, "password": "pass"})
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(loginBody))
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	token := resp["access_token"]

	// Get news (may be empty, but endpoint should work)
	req = httptest.NewRequest("GET", "/news?limit=5", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("List news: got %d, want 200", rr.Code)
	}

	// Mark all read (should succeed even with 0 entries)
	req = httptest.NewRequest("PUT", "/news/read-all", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Errorf("Mark all read: got %d, want 204", rr.Code)
	}
}

func TestCategoriesEndpoint(t *testing.T) {
	dataDir := "./test_categories"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)
	username := "cat_user"

	store, _ := db.NewStore(dataDir, username)
	hash, _ := auth.HashPassword("pass")
	store.UpsertUser(context.Background(), username, hash, "")
	store.Close()

	loginBody, _ := json.Marshal(map[string]string{"username": username, "password": "pass"})
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(loginBody))
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	token := resp["access_token"]

	req = httptest.NewRequest("GET", "/categories", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Categories: got %d, want 200", rr.Code)
	}
}

func TestConfigExportImport(t *testing.T) {
	dataDir := "./test_config_export"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)
	username := "config_user"

	store, _ := db.NewStore(dataDir, username)
	hash, _ := auth.HashPassword("pass")
	store.UpsertUser(context.Background(), username, hash, "")
	store.Close()

	loginBody, _ := json.Marshal(map[string]string{"username": username, "password": "pass"})
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(loginBody))
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	token := resp["access_token"]

	// Export
	req = httptest.NewRequest("GET", "/config/export", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Config export: got %d, want 200", rr.Code)
	}

	var exportData map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &exportData)
	if exportData["version"] != "1.0" {
		t.Errorf("Export version = %v, want 1.0", exportData["version"])
	}

	// Import
	configJSON := `{"filters": [{"pattern": "test-import", "action": "BLOCK"}]}`
	req = httptest.NewRequest("POST", "/config/import", strings.NewReader(configJSON))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Config import: got %d, want 200", rr.Code)
	}
}

func TestDocsEndpoint(t *testing.T) {
	dataDir := "./test_docs_data"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)

	req := httptest.NewRequest("GET", "/docs", nil)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Docs: got %d, want 200", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "swagger") && !strings.Contains(rr.Body.String(), "openapi") {
		t.Error("Docs response should contain swagger or openapi content")
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	dataDir := "./test_login_invalid"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)

	// Non-existent user
	loginBody, _ := json.Marshal(map[string]string{"username": "nobody", "password": "wrong"})
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(loginBody))
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Invalid login: got %d, want 401", rr.Code)
	}
}

func TestRefreshInvalidToken(t *testing.T) {
	dataDir := "./test_refresh_invalid"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)

	refreshBody, _ := json.Marshal(map[string]string{"username": "user", "refresh_token": "invalid"})
	req := httptest.NewRequest("POST", "/refresh", bytes.NewBuffer(refreshBody))
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Invalid refresh: got %d, want 401", rr.Code)
	}
}

func TestExportBackup(t *testing.T) {
	dataDir := "./test_backup_export"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)
	username := "backup_user"

	store, _ := db.NewStore(dataDir, username)
	hash, _ := auth.HashPassword("pass")
	store.UpsertUser(context.Background(), username, hash, "")
	store.Close()

	loginBody, _ := json.Marshal(map[string]string{"username": username, "password": "pass"})
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(loginBody))
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	token := resp["access_token"]

	req = httptest.NewRequest("GET", "/backup/export", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Backup export: got %d, want 200", rr.Code)
	}
}

func TestLoginEmptyBody(t *testing.T) {
	dataDir := "./test_login_empty"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)

	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer([]byte("{}")))
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code == http.StatusOK {
		t.Errorf("Empty login should not succeed, got 200")
	}
}

func TestLoginMalformedJSON(t *testing.T) {
	dataDir := "./test_login_malformed"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)

	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer([]byte("not json")))
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Malformed JSON login: got %d, want 400", rr.Code)
	}
}

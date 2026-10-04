package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nef734/gorrson/internal/auth"
	"github.com/nef734/gorrson/internal/db"
	"github.com/nef734/gorrson/internal/models"
)

func TestAPIServer(t *testing.T) {
	dataDir := "./test_api_data"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)
	username := "api_tester"
	password := "api_secret"

	// 0. Manual setup of user in DB
	store, _ := db.NewStore(dataDir, username)
	hash, _ := auth.HashPassword(password)
	store.UpsertUser(context.Background(), username, hash, "")
	store.Close()

	// 1. Test Login
	loginBody, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(loginBody))
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Login failed: status %d. Body: %s", rr.Code, rr.Body.String())
	}

	var loginResp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &loginResp)
	accessToken := loginResp["access_token"]
	refreshToken := loginResp["refresh_token"]
	if accessToken == "" || refreshToken == "" {
		t.Fatal("Tokens missing from login response")
	}

	// 2. Test Refresh Token
	refreshBody, _ := json.Marshal(map[string]string{
		"username":      username,
		"refresh_token": refreshToken,
	})
	req = httptest.NewRequest("POST", "/refresh", bytes.NewBuffer(refreshBody))
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Refresh failed: status %d. Body: %s", rr.Code, rr.Body.String())
	}

	var refreshResp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &refreshResp)
	if refreshResp["access_token"] == "" {
		t.Error("New access token missing from refresh response")
	}

	// 3. Test Add Feed (using access token)
	addFeedBody, _ := json.Marshal(map[string]string{
		"url": "https://news.ycombinator.com/rss",
	})
	req = httptest.NewRequest("POST", "/feeds", bytes.NewBuffer(addFeedBody))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated && rr.Code != http.StatusInternalServerError && rr.Code != http.StatusBadRequest {
		t.Errorf("AddFeed failed: status %d. Body: %s", rr.Code, rr.Body.String())
	}

	// 4. Test List Feeds
	req = httptest.NewRequest("GET", "/feeds", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ListFeeds failed: status %d", rr.Code)
	}

	var feeds []models.Feed
	json.Unmarshal(rr.Body.Bytes(), &feeds)

	// 5. Test Update Feed
	if len(feeds) > 0 {
		feedID := feeds[0].ID
		updateBody, _ := json.Marshal(map[string]string{
			"title":    "Updated Title",
			"category": "Tech",
		})
		req = httptest.NewRequest("PATCH", "/feeds/"+feedID, bytes.NewBuffer(updateBody))
		req.Header.Set("Authorization", "Bearer "+accessToken)
		rr = httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("UpdateFeed failed: status %d", rr.Code)
		}
	}

	// 6. Test List News
	req = httptest.NewRequest("GET", "/news?limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ListNews failed: status %d", rr.Code)
	}

	var entries []models.Entry
	json.Unmarshal(rr.Body.Bytes(), &entries)

	// 7. Test Mark Read & Favorite
	if len(entries) > 0 {
		entryID := entries[0].ID
		// Mark Read
		req = httptest.NewRequest("PUT", "/news/"+entryID+"/read", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		rr = httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Errorf("MarkRead failed: status %d", rr.Code)
		}

		// Mark Favorite
		req = httptest.NewRequest("PUT", "/news/"+entryID+"/favorite?favorite=true", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		rr = httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Errorf("MarkFavorite failed: status %d", rr.Code)
		}
	}

	// 8. Test Import OPML (Minimal)
	opmlData := `<?xml version="1.0" encoding="UTF-8"?><opml version="1.0"><body><outline text="HN" xmlUrl="https://news.ycombinator.com/rss" /></body></opml>`
	req = httptest.NewRequest("POST", "/feeds/import", bytes.NewBufferString(opmlData))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/xml")
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ImportOPML failed: status %d", rr.Code)
	}

	// 9. Test Delete Feed
	if len(feeds) > 0 {
		feedID := feeds[0].ID
		req = httptest.NewRequest("DELETE", "/feeds/"+feedID, nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		rr = httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Errorf("DeleteFeed failed: status %d", rr.Code)
		}
	}

	// 10. Test Docs
	req = httptest.NewRequest("GET", "/docs", nil)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Docs failed: status %d", rr.Code)
	}

	// 11. Test Filters API
	// Add Filter
	filterBody, _ := json.Marshal(map[string]string{
		"pattern": "TestPattern",
		"action":  "BLOCK",
		"target":  "title",
	})
	req = httptest.NewRequest("POST", "/filters", bytes.NewBuffer(filterBody))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("AddFilter API failed: status %d", rr.Code)
	}

	var addedFilter models.Filter
	json.Unmarshal(rr.Body.Bytes(), &addedFilter)

	// List Filters
	req = httptest.NewRequest("GET", "/filters", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("ListFilters API failed: status %d", rr.Code)
	}

	// Delete Filter
	req = httptest.NewRequest("DELETE", "/filters/"+addedFilter.ID, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Errorf("DeleteFilter API failed: status %d", rr.Code)
	}

	// 12. Test Logout
	req = httptest.NewRequest("POST", "/logout", bytes.NewBuffer(refreshBody))
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("Logout failed: status %d", rr.Code)
	}

	// Verify logout by trying to refresh again
	req = httptest.NewRequest("POST", "/refresh", bytes.NewBuffer(refreshBody))
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Refresh should have failed after logout, got status %d", rr.Code)
	}
}

func TestImageProxy(t *testing.T) {
	t.Setenv("GORRSON_TEST", "true")

	dataDir := "./test_proxy_data"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)

	// Setup user
	username := "proxy_tester"
	store, _ := db.NewStore(dataDir, username)
	store.Close()
	token, _ := auth.GenerateToken(username)

	// Mock image server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("fake-image-binary-data"))
	}))
	defer ts.Close()

	// Request through our proxy
	req := httptest.NewRequest("GET", "/proxy?url="+ts.URL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Proxy failed: status %d. Body: %s", rr.Code, rr.Body.String())
	}

	if rr.Body.String() != "fake-image-binary-data" {
		t.Errorf("Proxy data mismatch: got %s", rr.Body.String())
	}

	// Verify it was cached by checking the directory.
	// Se ignoran los ficheros .type: el Content-Type resuelto se guarda aparte
	// porque http.DetectContentType no reconoce SVG ni AVIF, y en la segunda visita
	// ya no está la cabecera del origen.
	cacheDir := filepath.Join(dataDir, "cache", "images")
	files, _ := os.ReadDir(cacheDir)
	imagenes := 0
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".type") {
			imagenes++
		}
	}
	if imagenes != 1 {
		t.Errorf("Expected 1 image in cache, got %d (ficheros totales: %d)", imagenes, len(files))
	}
}
func TestRenameCategoryAPI(t *testing.T) {
	dataDir := "./test_rename_data"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)
	username := "rename_tester"
	password := "rename_secret"

	store, _ := db.NewStore(dataDir, username)
	hash, _ := auth.HashPassword(password)
	store.UpsertUser(context.Background(), username, hash, "")

	// Pre-seed a feed with category
	feedID := "feed-1"
	store.AddFeed(context.Background(), &models.Feed{
		ID:       feedID,
		Title:    "Test Feed",
		FeedURL:  "https://test.com/rss",
		Category: "OldCat",
	})
	store.Close()

	// Login to get token
	loginBody, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(loginBody))
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	var loginResp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &loginResp)
	token := loginResp["access_token"]

	// Test Rename
	renameBody, _ := json.Marshal(map[string]string{"name": "NewCat"})
	req = httptest.NewRequest("PUT", "/categories/OldCat", bytes.NewBuffer(renameBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("Rename API failed: status %d. Body: %s", rr.Code, rr.Body.String())
	}

	// Verify in DB
	store, _ = db.NewStore(dataDir, username)
	defer store.Close()
	feed, _ := store.GetFeedByID(context.Background(), feedID)
	if feed.Category != "NewCat" {
		t.Errorf("Expected category NewCat, got %s", feed.Category)
	}
}

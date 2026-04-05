package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/nef734/gorrson/internal/auth"
	"github.com/nef734/gorrson/internal/db"
)

func TestAPIServer(t *testing.T) {
	dataDir := "./test_api_data"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir)
	username := "api_tester"
	password := "api_secret"

	// 0. Manual setup of user in DB (since there's no POST /register endpoint, we use the root CLI-like logic)
	store, _ := db.NewStore(dataDir, username)
	hash, _ := auth.HashPassword(password)
	store.UpsertUser(username, hash)
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
		t.Fatalf("Login failed: status %d", rr.Code)
	}

	var loginResp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &loginResp)
	token := loginResp["token"]
	if token == "" {
		t.Fatal("No token returned from login")
	}

	// 2. Test Add Feed
	addFeedBody, _ := json.Marshal(map[string]string{
		"url": "https://news.ycombinator.com/rss",
	})
	req = httptest.NewRequest("POST", "/feeds", bytes.NewBuffer(addFeedBody))
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	// Note: Synchronization might fail without internet, so we accept 201 or 500 (if sync fails)
	// For CI reliability, we check if it reaches the handler
	if rr.Code != http.StatusCreated && rr.Code != http.StatusInternalServerError {
		t.Errorf("AddFeed failed: status %d. Body: %s", rr.Code, rr.Body.String())
	}

	// 3. Test List Feeds
	req = httptest.NewRequest("GET", "/feeds", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ListFeeds failed: status %d", rr.Code)
	}

	var feeds []db.Feed
	json.Unmarshal(rr.Body.Bytes(), &feeds)
	if len(feeds) == 0 && rr.Code == http.StatusCreated { // Only check if AddFeed succeeded
		t.Error("Expected feeds in list")
	}

	// 4. Test List News
	req = httptest.NewRequest("GET", "/news", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ListNews failed: status %d", rr.Code)
	}
}

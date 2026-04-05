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
	"github.com/nef734/gorrson/internal/models"
)

func TestAPIServer(t *testing.T) {
	dataDir := "./test_api_data"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir)
	username := "api_tester"
	password := "api_secret"

	// 0. Manual setup of user in DB
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

	if rr.Code != http.StatusCreated && rr.Code != http.StatusInternalServerError {
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
	
	// 5. Test Logout
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

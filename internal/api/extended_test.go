package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/nef734/gorrson/internal/auth"
	"github.com/nef734/gorrson/internal/db"
)

func TestFeverAPI(t *testing.T) {
	dataDir := "./test_fever_data"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir)
	username := "fever_user"
	password := "fever_pass"

	// 0. Setup user with Fever Key
	store, _ := db.NewStore(dataDir, username)
	hash, _ := auth.HashPassword(password)
	
	// Fever Key: bcrypt(md5(user:pass))
	
	// Let's use real hashing for auth.CheckPasswordHash to pass
	feverApiKey := "fever_md5_secret" 
	feverBcrypt, _ := auth.HashPassword(feverApiKey)
	
	store.UpsertUser(username, hash, feverBcrypt)
	store.Close()

	// 1. Test Fever Auth
	req := httptest.NewRequest("POST", "/api/fever?api_key="+feverApiKey, nil)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Fever auth failed: %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["auth"].(float64) != 1 {
		t.Errorf("Expected auth: 1, got %v", resp["auth"])
	}

	// 2. Test Fever Groups
	req = httptest.NewRequest("GET", "/api/fever?api_key="+feverApiKey+"&groups", nil)
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Fever groups failed: %d", rr.Code)
	}
}

func TestConfigAPI(t *testing.T) {
	dataDir := "./test_config_data"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir)
	username := "config_user"
	password := "config_pass"

	store, _ := db.NewStore(dataDir, username)
	hash, _ := auth.HashPassword(password)
	store.UpsertUser(username, hash, "")
	store.Close()

	token, _ := auth.GenerateToken(username)

	// 1. Export Config
	req := httptest.NewRequest("GET", "/config/export", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Export config failed: %d", rr.Code)
	}

	// 2. Import Config
	configJson := `{"filters": [{"pattern": "test", "action": "BLOCK"}]}`
	req = httptest.NewRequest("POST", "/config/import", strings.NewReader(configJson))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Import config failed: %d", rr.Code)
	}
}

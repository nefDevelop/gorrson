package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nef734/gorrson/internal/models"
)

func TestFeedJSONStructure(t *testing.T) {
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	feed := models.Feed{
		ID:          "test-id",
		FeverID:     123,
		Title:       "Test Feed",
		SiteURL:     "https://example.com",
		FeedURL:     "https://example.com/rss",
		LastUpdated: now,
		UnreadCount: 5,
		// Username and Password should be omitted if empty
	}

	data, err := json.Marshal(feed)
	if err != nil {
		t.Fatalf("Failed to marshal feed: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Failed to unmarshal feed: %v", err)
	}

	// Verify types and keys
	if m["id"] != "test-id" {
		t.Errorf("Expected id 'test-id', got %v", m["id"])
	}
	if m["fever_id"] != float64(123) { // JSON numbers are floats
		t.Errorf("Expected fever_id 123, got %v", m["fever_id"])
	}
	
	// Verify omitempty
	if _, ok := m["username"]; ok {
		t.Error("username should be omitted when empty")
	}
	
	// Verify date format (should be RFC3339)
	expectedDate := "2024-01-01T12:00:00Z"
	if m["last_updated"] != expectedDate {
		t.Errorf("Expected date %s, got %s", expectedDate, m["last_updated"])
	}
}

func TestEntryJSONStructure(t *testing.T) {
	entry := models.Entry{
		ID:      "entry-1",
		Title:   "Entry Title",
		IsRead:  false,
		Content: "<p>Hello</p>",
	}

	data, _ := json.Marshal(entry)
	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if m["is_read"] != false {
		t.Errorf("Expected is_read false, got %v", m["is_read"])
	}
	if _, ok := m["matched_action"]; ok {
		t.Error("matched_action should be omitted when empty")
	}
}

func TestFeverAPIResponseJSON(t *testing.T) {
	// Fever API expects a specific root object structure
	type feverResponse struct {
		APILevel int `json:"api_level"`
		Auth     int `json:"auth"`
		// ... other fields
	}

	resp := feverResponse{
		APILevel: 3,
		Auth:     1,
	}

	data, _ := json.Marshal(resp)
	if !strings.Contains(string(data), "\"api_level\":3") {
		t.Errorf("Fever response missing api_level or wrong format: %s", string(data))
	}
}

func TestErrorResponseJSON(t *testing.T) {
	// Simulated handler for testing error JSON
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_request"})
	}

	req := httptest.NewRequest("GET", "/error", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", rr.Header().Get("Content-Type"))
	}

	var m map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("Failed to parse error JSON: %v", err)
	}

	if m["error"] != "invalid_request" {
		t.Errorf("Expected error 'invalid_request', got %s", m["error"])
	}
}

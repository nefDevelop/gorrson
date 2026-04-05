package db

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nef734/gorrson/internal/models"
)

func TestStoreFull(t *testing.T) {
	dataDir := "./test_data"
	username := "test_db_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	// 1. Test Sessions
	token := "refresh-token-123"
	sess := &Session{
		ID:           uuid.New().String(),
		Username:     username,
		RefreshToken: token,
		ExpiresAt:    time.Now().Add(time.Hour),
	}
	if err := s.CreateSession(sess); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	found, err := s.GetSession(token)
	if err != nil || found.Username != username {
		t.Errorf("GetSession failed or username mismatch: %v", err)
	}

	if err := s.DeleteSession(token); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}
	_, err = s.GetSession(token)
	if err == nil {
		t.Error("Expected error for deleted session")
	}

	// 2. Test Extended Entries (Images, Summary, Favorites)
	entry := &models.Entry{
		ID:          "e1",
		FeedID:      "f1",
		Title:       "News",
		Summary:     "A brief summary",
		Content:     "<p>Full content</p>",
		URL:         "https://example.com/news",
		ImageURL:    "https://example.com/image.jpg",
		PublishedAt: time.Now(),
		IsFavorite:  true,
	}

	if err := s.AddEntry(entry); err != nil {
		t.Fatalf("AddEntry failed: %v", err)
	}

	list, _ := s.ListEntries(false, "favorite")
	if len(list) != 1 || list[0].ImageURL != entry.ImageURL {
		t.Errorf("Expected 1 favorite entry with correct image, got %v", list)
	}

	if err := s.ToggleFavorite("e1", false); err != nil {
		t.Fatalf("ToggleFavorite failed: %v", err)
	}
	list, _ = s.ListEntries(false, "favorite")
	if len(list) != 0 {
		t.Error("Expected 0 favorite entries after toggle")
	}
}

package db

import (
	"context"
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
	if err := s.CreateSession(context.Background(), sess); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	found, err := s.GetSession(context.Background(), token)
	if err != nil || found.Username != username {
		t.Errorf("GetSession failed or username mismatch: %v", err)
	}

	if err := s.DeleteSession(context.Background(), token); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}
	_, err = s.GetSession(context.Background(), token)
	if err == nil {
		t.Error("Expected error for deleted session")
	}

	// 2. Test Extended Entries (Images, Summary, Favorites)
	_ = s.AddFeed(context.Background(), &models.Feed{
		ID:      "f1",
		Title:   "Test Feed",
		FeedURL: "https://example.com/feed",
	})

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

	if err := s.AddEntry(context.Background(), entry); err != nil {
		t.Fatalf("AddEntry failed: %v", err)
	}

	list, _ := s.ListEntries(context.Background(), false, "favorite", 10, 0, "", "")
	if len(list) != 1 || list[0].ImageURL != entry.ImageURL {
		t.Errorf("Expected 1 favorite entry with correct image, got %v", list)
	}

	if err := s.ToggleFavorite(context.Background(), "e1", false); err != nil {
		t.Fatalf("ToggleFavorite failed: %v", err)
	}
	list, _ = s.ListEntries(context.Background(), false, "favorite", 10, 0, "", "")
	if len(list) != 0 {
		t.Error("Expected 0 favorite entries after toggle")
	}
}

func TestListEntriesFilters(t *testing.T) {
	dataDir := "./test_data_filters"
	username := "filter_tester"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	_ = s.AddFeed(context.Background(), &models.Feed{ID: "f1", Title: "Tech Feed", FeedURL: "http://f1", Category: "Tech"})
	_ = s.AddFeed(context.Background(), &models.Feed{ID: "f2", Title: "News Feed", FeedURL: "http://f2", Category: "News"})

	_ = s.AddEntry(context.Background(), &models.Entry{ID: "e1", FeedID: "f1", Title: "T1", URL: "u1"})
	_ = s.AddEntry(context.Background(), &models.Entry{ID: "e2", FeedID: "f1", Title: "T2", URL: "u2"})
	_ = s.AddEntry(context.Background(), &models.Entry{ID: "e3", FeedID: "f2", Title: "T3", URL: "u3"})

	// Test case-insensitive category filtering
	list, _ := s.ListEntries(context.Background(), false, "newest", 10, 0, "tech", "")
	if len(list) != 2 {
		t.Errorf("Expected 2 tech entries, got %d", len(list))
	}

	// Test feedID filtering
	list2, _ := s.ListEntries(context.Background(), false, "newest", 10, 0, "", "f2")
	if len(list2) != 1 || list2[0].ID != "e3" {
		t.Errorf("Expected 1 news entry (e3), got %v", list2)
	}
}

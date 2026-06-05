package db

import (
	"context"
	"os"
	"testing"

	"github.com/nef734/gorrson/internal/models"
)

func TestMarkAllRead(t *testing.T) {
	dataDir := "./test_data_mark_all"
	username := "mark_all_tester"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Setup: 2 feeds with different categories
	_ = s.AddFeed(ctx, &models.Feed{ID: "f1", Title: "Tech", FeedURL: "http://f1", Category: "Tech"})
	_ = s.AddFeed(ctx, &models.Feed{ID: "f2", Title: "News", FeedURL: "http://f2", Category: "News"})

	// Setup: 5 unread entries
	_ = s.AddEntry(ctx, &models.Entry{ID: "e1", FeedID: "f1", Title: "T1", URL: "u1"})
	_ = s.AddEntry(ctx, &models.Entry{ID: "e2", FeedID: "f1", Title: "T2", URL: "u2"})
	_ = s.AddEntry(ctx, &models.Entry{ID: "e3", FeedID: "f1", Title: "T3", URL: "u3"})
	_ = s.AddEntry(ctx, &models.Entry{ID: "e4", FeedID: "f2", Title: "N1", URL: "u4"})
	_ = s.AddEntry(ctx, &models.Entry{ID: "e5", FeedID: "f2", Title: "N2", URL: "u5"})

	t.Run("MarkLoadedInCategory", func(t *testing.T) {
		// Action: Mark only Tech (limit 2)
		err := s.MarkAllRead(ctx, "Tech", "", "", 2)
		if err != nil {
			t.Fatalf("MarkAllRead failed: %v", err)
		}

		// Verify: 2 from f1 should be read, 1 from f1 remains unread, all from f2 remain unread
		unread, _ := s.ListEntries(ctx, true, "newest", 10, 0, "", "")
		if len(unread) != 3 {
			t.Errorf("Expected 3 unread entries total, got %d", len(unread))
		}

		techUnread, _ := s.ListEntries(ctx, true, "newest", 10, 0, "Tech", "")
		if len(techUnread) != 1 {
			t.Errorf("Expected 1 unread Tech entry, got %d", len(techUnread))
		}
	})

	t.Run("MarkGlobal", func(t *testing.T) {
		// Action: Mark EVERYTHING (isGlobal mode: empty category, feed, search, limit 0)
		err := s.MarkAllRead(ctx, "", "", "", 0)
		if err != nil {
			t.Fatalf("MarkAllRead global failed: %v", err)
		}

		// Verify: 0 unread entries remaining
		unread, _ := s.ListEntries(ctx, true, "newest", 10, 0, "", "")
		if len(unread) != 0 {
			t.Errorf("Expected 0 unread entries total, got %d", len(unread))
		}
	})
}

package db

import (
	"os"
	"testing"
	"time"
)

func TestStore(t *testing.T) {
	dataDir := "./test_data"
	username := "testuser"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	// Test AddFeed
	feed := &Feed{
		ID:          "f1",
		Title:       "Hacker News",
		FeedURL:     "https://news.ycombinator.com/rss",
		LastUpdated: time.Now(),
	}
	if err := s.AddFeed(feed); err != nil {
		t.Errorf("AddFeed failed: %v", err)
	}

	// Test GetFeedByURL
	f, err := s.GetFeedByURL(feed.FeedURL)
	if err != nil {
		t.Errorf("GetFeedByURL failed: %v", err)
	} else if f.Title != feed.Title {
		t.Errorf("Expected title %s, got %s", feed.Title, f.Title)
	}

	// Test AddEntry
	entry := &Entry{
		ID:          "e1",
		FeedID:      "f1",
		Title:       "The end of coding?",
		URL:         "https://example.com/end-of-coding",
		PublishedAt: time.Now(),
	}
	if err := s.AddEntry(entry); err != nil {
		t.Errorf("AddEntry failed: %v", err)
	}

	// Test ListEntries
	entries, err := s.ListEntries(false, "newest")
	if err != nil {
		t.Errorf("ListEntries failed: %v", err)
	} else if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	// Test MarkAsRead
	if err := s.MarkAsRead("e1"); err != nil {
		t.Errorf("MarkAsRead failed: %v", err)
	}

	// Test Search (FTS5)
	results, err := s.SearchEntries("coding")
	if err != nil {
		t.Errorf("SearchEntries failed: %v", err)
	} else if len(results) != 1 {
		t.Errorf("Search should return 1 result, got %d", len(results))
	}
}

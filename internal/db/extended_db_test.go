package db

import (
	"os"
	"testing"
	"time"

	"github.com/nef734/gorrson/internal/models"
)

func TestDBSearches(t *testing.T) {
	dataDir := "./test_db_search"
	username := "search_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, _ := NewStore(dataDir, username)
	defer s.Close()

	s.AddFeed(&models.Feed{ID: "f1", Title: "Tech", FeedURL: "u1"})
	s.AddEntry(&models.Entry{ID: "e1", FeedID: "f1", Title: "Golang coding", Content: "Go is great", URL: "u1", PublishedAt: time.Now()})
	s.AddEntry(&models.Entry{ID: "e2", FeedID: "f1", Title: "Python programming", Content: "Python is slow", URL: "u2", PublishedAt: time.Now()})

	// Wait a bit for triggers? No, SQLite triggers are synchronous.
	
	// Test Search
	results, err := s.SearchEntries("Golang")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 || results[0].ID != "e1" {
		t.Errorf("Expected e1, got %v", results)
	}

	results, _ = s.SearchEntries("is") // Both have "is"
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestDBCleanup(t *testing.T) {
	dataDir := "./test_db_cleanup"
	username := "cleanup_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, _ := NewStore(dataDir, username)
	defer s.Close()

	s.AddFeed(&models.Feed{ID: "f1", Title: "T1", FeedURL: "u1"})
	
	// Old entry
	old := time.Now().AddDate(-2, 0, 0)
	s.AddEntry(&models.Entry{ID: "old", FeedID: "f1", Title: "Old", URL: "u_old", PublishedAt: old})
	
	// Recent entry
	s.AddEntry(&models.Entry{ID: "new", FeedID: "f1", Title: "New", URL: "u_new", PublishedAt: time.Now()})

	count, err := s.CleanupOldEntries()
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 entry cleaned, got %d", count)
	}

	list, _ := s.ListEntries(false, "newest", 10, 0, "", "")
	if len(list) != 1 || list[0].ID != "new" {
		t.Errorf("Expected only 'new' entry, got %v", list)
	}
}

func TestDBFilters(t *testing.T) {
	dataDir := "./test_db_filters"
	username := "filter_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, _ := NewStore(dataDir, username)
	defer s.Close()

	s.AddFeed(&models.Feed{ID: "f1", Title: "T1", FeedURL: "u1"})
	
	f1 := &models.Filter{ID: "filt1", FeedID: "f1", Pattern: "spam", Action: "BLOCK", Target: "title"}
	f2 := &models.Filter{ID: "filt2", Pattern: "global", Action: "REPLACE", Target: "content"}

	s.AddFilter(f1)
	s.AddFilter(f2)

	filts, _ := s.GetFiltersForFeed("f1")
	if len(filts) != 2 {
		t.Errorf("Expected 2 filters for f1 (1 specific + 1 global), got %d", len(filts))
	}

	all, _ := s.ListFilters()
	if len(all) != 2 {
		t.Errorf("Expected 2 total filters, got %d", len(all))
	}

	s.DeleteFilter("filt1")
	all, _ = s.ListFilters()
	if len(all) != 1 {
		t.Errorf("Expected 1 filter after delete, got %d", len(all))
	}
}

package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/nef734/gorrson/internal/models"
)

func TestListEntriesPagination(t *testing.T) {
	dataDir := "./test_data_pagination"
	username := "pagination_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.AddFeed(ctx, &models.Feed{ID: "f1", Title: "Feed", FeedURL: "http://feed1"})

	for i := 0; i < 20; i++ {
		_ = s.AddEntry(ctx, &models.Entry{
			ID:          "e" + string(rune('a'+i)),
			FeedID:      "f1",
			Title:       "Entry " + string(rune('a'+i)),
			URL:         "http://entry" + string(rune('a'+i)),
			PublishedAt: time.Now().Add(-time.Duration(i) * time.Minute),
		})
	}

	// Test limit
	page1, err := s.ListEntries(ctx, false, "newest", 5, 0, "", "")
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(page1) != 5 {
		t.Errorf("Page 1: expected 5 entries, got %d", len(page1))
	}

	// Test offset
	page2, err := s.ListEntries(ctx, false, "newest", 5, 5, "", "")
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(page2) != 5 {
		t.Errorf("Page 2: expected 5 entries, got %d", len(page2))
	}

	// Verify no overlap
	for _, e1 := range page1 {
		for _, e2 := range page2 {
			if e1.ID == e2.ID {
				t.Errorf("Duplicate entry %s across pages", e1.ID)
			}
		}
	}

	// Test offset beyond data
	pageEnd, err := s.ListEntries(ctx, false, "newest", 5, 100, "", "")
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(pageEnd) != 0 {
		t.Errorf("Expected 0 entries at offset 100, got %d", len(pageEnd))
	}
}

func TestListEntriesSortOldest(t *testing.T) {
	dataDir := "./test_data_sort"
	username := "sort_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.AddFeed(ctx, &models.Feed{ID: "f1", Title: "Feed", FeedURL: "http://feed1"})

	_ = s.AddEntry(ctx, &models.Entry{ID: "e1", FeedID: "f1", Title: "First", URL: "u1", PublishedAt: time.Now().Add(-3 * time.Hour)})
	_ = s.AddEntry(ctx, &models.Entry{ID: "e2", FeedID: "f1", Title: "Second", URL: "u2", PublishedAt: time.Now().Add(-1 * time.Hour)})
	_ = s.AddEntry(ctx, &models.Entry{ID: "e3", FeedID: "f1", Title: "Third", URL: "u3", PublishedAt: time.Now()})

	entries, err := s.ListEntries(ctx, false, "oldest", 10, 0, "", "")
	if err != nil {
		t.Fatalf("ListEntries oldest failed: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(entries))
	}
	if entries[0].ID != "e1" || entries[2].ID != "e3" {
		t.Errorf("Sort oldest wrong: first=%s, last=%s", entries[0].ID, entries[2].ID)
	}
}

func TestListEntriesUnreadOnly(t *testing.T) {
	dataDir := "./test_data_unread"
	username := "unread_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.AddFeed(ctx, &models.Feed{ID: "f1", Title: "Feed", FeedURL: "http://feed1"})

	_ = s.AddEntry(ctx, &models.Entry{ID: "e1", FeedID: "f1", Title: "Read", URL: "u1", IsRead: true})
	_ = s.AddEntry(ctx, &models.Entry{ID: "e2", FeedID: "f1", Title: "Unread", URL: "u2", IsRead: false})

	entries, err := s.ListEntries(ctx, true, "newest", 10, 0, "", "")
	if err != nil {
		t.Fatalf("ListEntries unread failed: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != "e2" {
		t.Errorf("Expected only unread entry e2, got %v", entries)
	}
}

func TestListEntriesPriorityOrder(t *testing.T) {
	dataDir := "./test_data_priority"
	username := "priority_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.AddFeed(ctx, &models.Feed{ID: "f1", Title: "Feed", FeedURL: "http://feed1"})

	_ = s.AddEntry(ctx, &models.Entry{ID: "e1", FeedID: "f1", Title: "Normal", URL: "u1", PublishedAt: time.Now(), IsPriority: false})
	_ = s.AddEntry(ctx, &models.Entry{ID: "e2", FeedID: "f1", Title: "Important", URL: "u2", PublishedAt: time.Now().Add(-time.Hour), IsPriority: true})

	entries, err := s.ListEntries(ctx, false, "newest", 10, 0, "", "")
	if err != nil {
		t.Fatalf("ListEntries priority failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(entries))
	}
	if entries[0].ID != "e2" {
		t.Errorf("Priority entry should be first, got %s", entries[0].ID)
	}
}

func TestGetEntryByID(t *testing.T) {
	dataDir := "./test_data_get_entry"
	username := "get_entry_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.AddFeed(ctx, &models.Feed{ID: "f1", Title: "My Feed", FeedURL: "http://feed1"})
	_ = s.AddEntry(ctx, &models.Entry{
		ID:          "e1",
		FeedID:      "f1",
		Title:       "Article",
		Content:     "<p>Content</p>",
		URL:         "http://article",
		ImageURL:    "http://img.jpg",
		PublishedAt: time.Now(),
	})

	entry, err := s.GetEntryByID(ctx, "e1")
	if err != nil {
		t.Fatalf("GetEntryByID failed: %v", err)
	}
	if entry.Title != "Article" {
		t.Errorf("Title = %s, want Article", entry.Title)
	}
	if entry.Content != "<p>Content</p>" {
		t.Errorf("Content = %s", entry.Content)
	}
	if entry.ImageURL != "http://img.jpg" {
		t.Errorf("ImageURL = %s", entry.ImageURL)
	}
	if entry.Source != "My Feed" {
		t.Errorf("Source (feed title) = %s, want My Feed", entry.Source)
	}

	// Non-existent entry
	_, err = s.GetEntryByID(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent entry")
	}
}

func TestUpdateEntryContent(t *testing.T) {
	dataDir := "./test_data_update_content"
	username := "update_content_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.AddFeed(ctx, &models.Feed{ID: "f1", Title: "Feed", FeedURL: "http://feed1"})
	_ = s.AddEntry(ctx, &models.Entry{ID: "e1", FeedID: "f1", Title: "Art", URL: "http://art", Content: "short"})

	err = s.UpdateEntryContent(ctx, "e1", "<p>Full content here</p>")
	if err != nil {
		t.Fatalf("UpdateEntryContent failed: %v", err)
	}

	entry, _ := s.GetEntryByID(ctx, "e1")
	if entry.Content != "<p>Full content here</p>" {
		t.Errorf("Content not updated: %s", entry.Content)
	}
	if !entry.IsFullContent {
		t.Error("is_full_content should be true after update")
	}
}

func TestMarkAsReadByFeverID(t *testing.T) {
	dataDir := "./test_data_fever_read"
	username := "fever_read_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.AddFeed(ctx, &models.Feed{ID: "f1", Title: "Feed", FeedURL: "http://feed1"})
	_ = s.AddEntry(ctx, &models.Entry{ID: "e1", FeedID: "f1", Title: "Art", URL: "http://art", FeverID: 999})

	err = s.MarkAsReadByFeverID(ctx, 999)
	if err != nil {
		t.Fatalf("MarkAsReadByFeverID failed: %v", err)
	}

	entry, _ := s.GetEntryByID(ctx, "e1")
	if !entry.IsRead {
		t.Error("Entry should be marked as read by fever ID")
	}
}

func TestToggleFavoriteByFeverID(t *testing.T) {
	dataDir := "./test_data_fever_fav"
	username := "fever_fav_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.AddFeed(ctx, &models.Feed{ID: "f1", Title: "Feed", FeedURL: "http://feed1"})
	_ = s.AddEntry(ctx, &models.Entry{ID: "e1", FeedID: "f1", Title: "Art", URL: "http://art", FeverID: 888})

	// Favorite by fever ID
	err = s.ToggleFavoriteByFeverID(ctx, 888, true)
	if err != nil {
		t.Fatalf("ToggleFavoriteByFeverID failed: %v", err)
	}

	entry, _ := s.GetEntryByID(ctx, "e1")
	if !entry.IsFavorite {
		t.Error("Entry should be favorited by fever ID")
	}

	// Unfavorite
	err = s.ToggleFavoriteByFeverID(ctx, 888, false)
	if err != nil {
		t.Fatalf("ToggleFavoriteByFeverID unfavorite failed: %v", err)
	}

	entry, _ = s.GetEntryByID(ctx, "e1")
	if entry.IsFavorite {
		t.Error("Entry should be unfavorited")
	}
}

func TestMarkAllReadByFeed(t *testing.T) {
	dataDir := "./test_data_mark_feed"
	username := "mark_feed_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.AddFeed(ctx, &models.Feed{ID: "f1", Title: "Feed1", FeedURL: "http://f1"})
	_ = s.AddFeed(ctx, &models.Feed{ID: "f2", Title: "Feed2", FeedURL: "http://f2"})

	_ = s.AddEntry(ctx, &models.Entry{ID: "e1", FeedID: "f1", Title: "F1 Art", URL: "u1"})
	_ = s.AddEntry(ctx, &models.Entry{ID: "e2", FeedID: "f1", Title: "F1 Art2", URL: "u2"})
	_ = s.AddEntry(ctx, &models.Entry{ID: "e3", FeedID: "f2", Title: "F2 Art", URL: "u3"})

	err = s.MarkAllRead(ctx, "", "f1", "", 0)
	if err != nil {
		t.Fatalf("MarkAllRead by feed failed: %v", err)
	}

	unread, _ := s.ListEntries(ctx, true, "newest", 10, 0, "", "")
	if len(unread) != 1 || unread[0].ID != "e3" {
		t.Errorf("Expected only e3 unread, got %v", unread)
	}
}

func TestMarkAllReadBySearch(t *testing.T) {
	dataDir := "./test_data_mark_search"
	username := "mark_search_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.AddFeed(ctx, &models.Feed{ID: "f1", Title: "Feed", FeedURL: "http://f1"})

	_ = s.AddEntry(ctx, &models.Entry{ID: "e1", FeedID: "f1", Title: "Golang Tutorial", Content: "Learn Go", URL: "u1"})
	_ = s.AddEntry(ctx, &models.Entry{ID: "e2", FeedID: "f1", Title: "Python Guide", Content: "Learn Python", URL: "u2"})
	_ = s.AddEntry(ctx, &models.Entry{ID: "e3", FeedID: "f1", Title: "Golang Advanced", Content: "Advanced Go", URL: "u3"})

	err = s.MarkAllRead(ctx, "", "", "Golang", 0)
	if err != nil {
		t.Fatalf("MarkAllRead by search failed: %v", err)
	}

	unread, _ := s.ListEntries(ctx, true, "newest", 10, 0, "", "")
	if len(unread) != 1 || unread[0].ID != "e2" {
		t.Errorf("Expected only Python entry unread, got %v", unread)
	}
}

func TestMarkAllReadWithLimit(t *testing.T) {
	dataDir := "./test_data_mark_limit"
	username := "mark_limit_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.AddFeed(ctx, &models.Feed{ID: "f1", Title: "Feed", FeedURL: "http://f1"})

	for i := 0; i < 10; i++ {
		_ = s.AddEntry(ctx, &models.Entry{ID: "e" + string(rune('a'+i)), FeedID: "f1", Title: "Art", URL: "u" + string(rune('a'+i))})
	}

	err = s.MarkAllRead(ctx, "", "", "", 3)
	if err != nil {
		t.Fatalf("MarkAllRead with limit failed: %v", err)
	}

	unread, _ := s.ListEntries(ctx, true, "newest", 10, 0, "", "")
	if len(unread) != 7 {
		t.Errorf("Expected 7 unread after marking 3, got %d", len(unread))
	}
}

func TestSessionExpiredCleanup(t *testing.T) {
	dataDir := "./test_data_session_cleanup"
	username := "session_cleanup_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create expired session
	expired := &Session{
		ID:           "expired-id",
		Username:     username,
		RefreshToken: "expired-token",
		ExpiresAt:    time.Now().Add(-time.Hour),
	}
	_ = s.CreateSession(ctx, expired)

	// Create valid session
	valid := &Session{
		ID:           "valid-id",
		Username:     username,
		RefreshToken: "valid-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}
	_ = s.CreateSession(ctx, valid)

	err = s.DeleteExpiredSessions(ctx)
	if err != nil {
		t.Fatalf("DeleteExpiredSessions failed: %v", err)
	}

	// Expired should be gone
	_, err = s.GetSession(ctx, "expired-token")
	if err == nil {
		t.Error("Expired session should have been deleted")
	}

	// Valid should remain
	sess, err := s.GetSession(ctx, "valid-token")
	if err != nil || sess.ID != "valid-id" {
		t.Error("Valid session should still exist")
	}
}

func TestAPIKeyCRUD(t *testing.T) {
	dataDir := "./test_data_apikeys"
	username := "apikey_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Add
	err = s.AddAPIKey(ctx, "key-1", "My Key", "hash123")
	if err != nil {
		t.Fatalf("AddAPIKey failed: %v", err)
	}

	// Validate
	if !s.ValidateAPIKey(ctx, "hash123") {
		t.Error("ValidateAPIKey should return true for valid key")
	}
	if s.ValidateAPIKey(ctx, "wrong-hash") {
		t.Error("ValidateAPIKey should return false for invalid key")
	}

	// List
	keys, err := s.ListAPIKeys(ctx)
	if err != nil {
		t.Fatalf("ListAPIKeys failed: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("Expected 1 API key, got %d", len(keys))
	}

	// Delete
	err = s.DeleteAPIKey(ctx, "key-1")
	if err != nil {
		t.Fatalf("DeleteAPIKey failed: %v", err)
	}

	if s.ValidateAPIKey(ctx, "hash123") {
		t.Error("ValidateAPIKey should return false after deletion")
	}
}

func TestGetStats(t *testing.T) {
	dataDir := "./test_data_stats"
	username := "stats_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.AddFeed(ctx, &models.Feed{ID: "f1", Title: "Tech Feed", FeedURL: "http://f1", Category: "Tech"})
	_ = s.AddFeed(ctx, &models.Feed{ID: "f2", Title: "News Feed", FeedURL: "http://f2", Category: "News"})

	_ = s.AddEntry(ctx, &models.Entry{ID: "e1", FeedID: "f1", Title: "T1", URL: "u1", IsRead: false})
	_ = s.AddEntry(ctx, &models.Entry{ID: "e2", FeedID: "f1", Title: "T2", URL: "u2", IsRead: true})
	_ = s.AddEntry(ctx, &models.Entry{ID: "e3", FeedID: "f2", Title: "N1", URL: "u3", IsRead: false})

	stats, err := s.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.TotalEntries != 3 {
		t.Errorf("TotalEntries = %d, want 3", stats.TotalEntries)
	}
	if stats.UnreadEntries != 2 {
		t.Errorf("UnreadEntries = %d, want 2", stats.UnreadEntries)
	}
	if stats.TotalFeeds != 2 {
		t.Errorf("TotalFeeds = %d, want 2", stats.TotalFeeds)
	}
	if len(stats.CategoryStats) != 2 {
		t.Errorf("CategoryStats count = %d, want 2", len(stats.CategoryStats))
	}
	if len(stats.FeedStats) != 2 {
		t.Errorf("FeedStats count = %d, want 2", len(stats.FeedStats))
	}
}

func TestGetUserHashAndFeverKey(t *testing.T) {
	dataDir := "./test_data_user_hash"
	username := "hash_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.UpsertUser(ctx, username, "hash-value", "fever-key-value")

	hash, err := s.GetUserHash(ctx, username)
	if err != nil {
		t.Fatalf("GetUserHash failed: %v", err)
	}
	if hash != "hash-value" {
		t.Errorf("Hash = %s, want hash-value", hash)
	}

	feverKey, err := s.GetUserFeverKey(ctx, username)
	if err != nil {
		t.Fatalf("GetUserFeverKey failed: %v", err)
	}
	if feverKey != "fever-key-value" {
		t.Errorf("FeverKey = %s, want fever-key-value", feverKey)
	}

	// Non-existent user
	_, err = s.GetUserHash(ctx, "nobody")
	if err == nil {
		t.Error("Expected error for nonexistent user hash")
	}
}

func TestRenameCategory(t *testing.T) {
	dataDir := "./test_data_rename_cat"
	username := "rename_cat_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.AddFeed(ctx, &models.Feed{ID: "f1", Title: "F1", FeedURL: "http://f1", Category: "OldCat"})
	_ = s.AddFeed(ctx, &models.Feed{ID: "f2", Title: "F2", FeedURL: "http://f2", Category: "OldCat"})
	_ = s.AddFeed(ctx, &models.Feed{ID: "f3", Title: "F3", FeedURL: "http://f3", Category: "Other"})

	err = s.RenameCategory(ctx, "OldCat", "NewCat")
	if err != nil {
		t.Fatalf("RenameCategory failed: %v", err)
	}

	f1, _ := s.GetFeedByID(ctx, "f1")
	f2, _ := s.GetFeedByID(ctx, "f2")
	f3, _ := s.GetFeedByID(ctx, "f3")

	if f1.Category != "NewCat" || f2.Category != "NewCat" {
		t.Errorf("Category not renamed: f1=%s, f2=%s", f1.Category, f2.Category)
	}
	if f3.Category != "Other" {
		t.Errorf("Unrelated feed category changed: f3=%s", f3.Category)
	}
}

func TestFeedAuthCredentials(t *testing.T) {
	dataDir := "./test_data_feed_auth"
	username := "feed_auth_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.AddFeed(ctx, &models.Feed{
		ID:        "f1",
		Title:     "Protected Feed",
		FeedURL:   "http://protected.com/rss",
		UserAgent: "CustomBot/2.0",
		Username:  "api_user",
		Password:  "secret_pass",
	})

	feed, err := s.GetFeedByURL(ctx, "http://protected.com/rss")
	if err != nil {
		t.Fatalf("GetFeedByURL failed: %v", err)
	}
	if feed.UserAgent != "CustomBot/2.0" {
		t.Errorf("UserAgent = %s, want CustomBot/2.0", feed.UserAgent)
	}
	if feed.Username != "api_user" {
		t.Errorf("Username = %s, want api_user", feed.Username)
	}
	if feed.Password != "secret_pass" {
		t.Errorf("Password = %s, want secret_pass", feed.Password)
	}
}

func TestAddEntryDuplicateURL(t *testing.T) {
	dataDir := "./test_data_dup_url"
	username := "dup_url_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.AddFeed(ctx, &models.Feed{ID: "f1", Title: "Feed", FeedURL: "http://feed1"})

	_ = s.AddEntry(ctx, &models.Entry{ID: "e1", FeedID: "f1", Title: "First", URL: "http://same-url"})
	err = s.AddEntry(ctx, &models.Entry{ID: "e2", FeedID: "f1", Title: "Duplicate", URL: "http://same-url"})
	if err != nil {
		t.Fatalf("AddEntry duplicate should not error: %v", err)
	}

	entries, _ := s.ListEntries(ctx, false, "newest", 10, 0, "", "")
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry (duplicate ignored), got %d", len(entries))
	}
}

func TestCleanupPreservesFavorites(t *testing.T) {
	dataDir := "./test_data_cleanup_fav"
	username := "cleanup_fav_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.AddFeed(ctx, &models.Feed{ID: "f1", Title: "Feed", FeedURL: "http://f1"})

	old := time.Now().AddDate(-2, 0, 0)
	_ = s.AddEntry(ctx, &models.Entry{ID: "old-fav", FeedID: "f1", Title: "Old Fav", URL: "u1", PublishedAt: old, IsFavorite: true, IsRead: true})
	_ = s.AddEntry(ctx, &models.Entry{ID: "old-read", FeedID: "f1", Title: "Old Read", URL: "u2", PublishedAt: old, IsFavorite: false, IsRead: true})

	count, _ := s.CleanupOldEntries(ctx)
	if count != 1 {
		t.Errorf("Expected 1 cleaned (old-read), got %d", count)
	}

	entries, _ := s.ListEntries(ctx, false, "newest", 10, 0, "", "")
	if len(entries) != 1 || entries[0].ID != "old-fav" {
		t.Errorf("Favorite entry should be preserved, got %v", entries)
	}
}

func TestGetFeedByURLNotFound(t *testing.T) {
	dataDir := "./test_data_notfound"
	username := "notfound_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	_, err = s.GetFeedByURL(context.Background(), "http://nonexistent.com")
	if err == nil {
		t.Error("Expected error for nonexistent feed URL")
	}
}

func TestDeleteFeedCascade(t *testing.T) {
	dataDir := "./test_data_cascade"
	username := "cascade_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.AddFeed(ctx, &models.Feed{ID: "f1", Title: "Feed", FeedURL: "http://f1"})
	_ = s.AddEntry(ctx, &models.Entry{ID: "e1", FeedID: "f1", Title: "Art", URL: "http://art"})
	_ = s.AddFilter(ctx, &models.Filter{ID: "filt1", FeedID: "f1", Pattern: "test", Action: "BLOCK", Target: "title"})

	err = s.DeleteFeed(ctx, "f1")
	if err != nil {
		t.Fatalf("DeleteFeed failed: %v", err)
	}

	feeds, _ := s.ListFeeds(ctx)
	if len(feeds) != 0 {
		t.Errorf("Feed should be deleted, got %d feeds", len(feeds))
	}

	entries, _ := s.ListEntries(ctx, false, "newest", 10, 0, "", "")
	if len(entries) != 0 {
		t.Errorf("Entries should cascade delete, got %d", len(entries))
	}

	filters, _ := s.ListFilters(ctx)
	if len(filters) != 0 {
		t.Errorf("Filters should cascade delete, got %d", len(filters))
	}
}

func TestBeginTxCommit(t *testing.T) {
	dataDir := "./test_data_tx"
	username := "tx_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.AddFeed(ctx, &models.Feed{ID: "f1", Title: "Feed", FeedURL: "http://f1"})

	tx, err := s.Begin()
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	entry := &models.Entry{
		ID:     "e-tx",
		FeedID: "f1",
		Title:  "TX Entry",
		URL:    "http://tx",
	}
	if err := s.AddEntryTx(ctx, tx, entry); err != nil {
		t.Fatalf("AddEntryTx failed: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	entries, _ := s.ListEntries(ctx, false, "newest", 10, 0, "", "")
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry after commit, got %d", len(entries))
	}
}

func TestBeginTxRollback(t *testing.T) {
	dataDir := "./test_data_tx_rb"
	username := "tx_rb_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	_ = s.AddFeed(ctx, &models.Feed{ID: "f1", Title: "Feed", FeedURL: "http://f1"})

	tx, err := s.Begin()
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	entry := &models.Entry{
		ID:     "e-rollback",
		FeedID: "f1",
		Title:  "Rollback",
		URL:    "http://rb",
	}
	_ = s.AddEntryTx(ctx, tx, entry)
	_ = tx.Rollback()

	entries, _ := s.ListEntries(ctx, false, "newest", 10, 0, "", "")
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries after rollback, got %d", len(entries))
	}
}

func TestVerifySQLiteFile(t *testing.T) {
	dataDir := "./test_data_verify"
	username := "verify_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	s.Close()

	dbPath := dataDir + "/" + username + ".db"
	err = VerifySQLiteFile(context.Background(), dbPath)
	if err != nil {
		t.Errorf("VerifySQLiteFile failed for valid DB: %v", err)
	}

	// Invalid file
	err = VerifySQLiteFile(context.Background(), "/nonexistent/file.db")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

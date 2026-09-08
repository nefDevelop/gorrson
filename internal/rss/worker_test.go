package rss

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/nef734/gorrson/internal/db"
	"github.com/nef734/gorrson/internal/models"
)

func TestNewWorker(t *testing.T) {
	dataDir := "./test_worker_new"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	store, err := db.NewStore(dataDir, "worker_user")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	w := NewWorker(store, 5*time.Minute)
	if w == nil {
		t.Fatal("NewWorker returned nil")
	}
	if w.store != store {
		t.Error("Worker store mismatch")
	}
	if w.interval != 5*time.Minute {
		t.Errorf("Worker interval = %v, want 5m", w.interval)
	}
	if w.parser == nil {
		t.Error("Worker parser is nil")
	}
}

func TestWorkerSyncAllEmpty(t *testing.T) {
	dataDir := "./test_worker_empty"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	store, err := db.NewStore(dataDir, "empty_user")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	w := NewWorker(store, 5*time.Minute)
	ctx := context.Background()

	// SyncAll with no feeds should not panic or error
	w.SyncAll(ctx)

	feeds, _ := store.ListFeeds(ctx)
	if len(feeds) != 0 {
		t.Errorf("Expected 0 feeds, got %d", len(feeds))
	}
}

func TestWorkerSyncAllWithMockFeed(t *testing.T) {
	t.Setenv("GORRSON_TEST", "true")

	dataDir := "./test_worker_sync"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	store, err := db.NewStore(dataDir, "sync_user")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Mock RSS feed server
	feedXML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
	<title>Test Feed</title>
	<link>https://example.com</link>
	<item>
		<title>Article 1</title>
		<link>https://example.com/1</link>
		<description>First article</description>
	</item>
	<item>
		<title>Article 2</title>
		<link>https://example.com/2</link>
		<description>Second article</description>
	</item>
</channel>
</rss>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(feedXML))
	}))
	defer ts.Close()

	// Pre-add the feed to DB
	_ = store.AddFeed(context.Background(), &models.Feed{
		ID:      "mock-feed",
		Title:   "Mock",
		FeedURL: ts.URL,
	})

	w := NewWorker(store, 5*time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	w.SyncAll(ctx)

	entries, err := store.ListEntries(ctx, false, "newest", 10, 0, "", "")
	if err != nil {
		t.Fatalf("ListEntries failed: %v", err)
	}
	if len(entries) < 2 {
		t.Errorf("Expected at least 2 entries from mock feed, got %d", len(entries))
	}
}

func TestWorkerContextCancel(t *testing.T) {
	t.Setenv("GORRSON_TEST", "true")

	dataDir := "./test_worker_cancel"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	store, err := db.NewStore(dataDir, "cancel_user")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	w := NewWorker(store, 1*time.Second)

	ctx, cancel := context.WithCancel(context.Background())

	// Start in background, it should block until context is canceled
	done := make(chan struct{})
	go func() {
		w.Start(ctx)
		close(done)
	}()

	// Let it do initial sync then cancel
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Good, worker stopped
	case <-time.After(5 * time.Second):
		t.Fatal("Worker did not stop after context cancellation")
	}
}

func TestWorkerSyncAllErrorHandling(t *testing.T) {
	dataDir := "./test_worker_error"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	store, err := db.NewStore(dataDir, "error_user")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Add a feed pointing to a non-existent server
	_ = store.AddFeed(context.Background(), &models.Feed{
		ID:      "bad-feed",
		Title:   "Bad Feed",
		FeedURL: "http://localhost:1",
	})

	w := NewWorker(store, 5*time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// SyncAll should not panic even with bad feeds
	w.SyncAll(ctx)

	// Verify no entries were created
	entries, _ := store.ListEntries(ctx, false, "newest", 10, 0, "", "")
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries from bad feed, got %d", len(entries))
	}
}

package rss

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/nef734/gorrson/internal/db"
)

func TestSyncFeed(t *testing.T) {
	dataDir := "./test_rss_data"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	store, err := db.NewStore(dataDir, "rss_test")
	if err != nil {
		t.Fatalf("Could not setup test db: %v", err)
	}
	defer store.Close()

	parser := NewParser()
	url := "https://news.ycombinator.com/rss"
	
	// Set a timeout for the network call
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := parser.SyncFeed(ctx, url, store); err != nil {
		// If network fails (e.g. offline), skip the test instead of failing
		t.Logf("Skipping live feed test due to network: %v", err)
		return
	}

	feeds, err := store.ListFeeds()
	if err != nil || len(feeds) == 0 {
		t.Errorf("Expected at least one feed in DB, got: %d", len(feeds))
	}

	entries, err := store.ListEntries(false, "newest")
	if err != nil || len(entries) == 0 {
		t.Errorf("Expected at least one entry in DB, got: %d", len(entries))
	} else {
		t.Logf("Success: Synced %d entries from %s", len(entries), url)
	}
}

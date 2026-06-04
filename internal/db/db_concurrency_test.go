package db

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nef734/gorrson/internal/models"
)

func TestStoreConcurrency(t *testing.T) {
	dataDir := "./test_concurrency"
	username := "concurrent_user"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s, err := NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer s.Close()

	// Add a base feed
	feedID := "f1"
	err = s.AddFeed(context.Background(), &models.Feed{
		ID:      feedID,
		Title:   "Concurrent Feed",
		FeedURL: "http://example.com/concurrent",
	})
	if err != nil {
		t.Fatalf("Failed to add base feed: %v", err)
	}

	var wg sync.WaitGroup
	numWorkers := 20
	iterations := 50

	// Writer goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				entry := &models.Entry{
					ID:          uuid.New().String(),
					FeedID:      feedID,
					Title:       fmt.Sprintf("Title %d-%d", workerID, j),
					URL:         fmt.Sprintf("http://example.com/entry/%d/%d", workerID, j),
					PublishedAt: time.Now(),
				}
				err := s.AddEntry(context.Background(), entry)
				if err != nil {
					t.Errorf("Worker %d failed to add entry: %v", workerID, err)
				}
			}
		}(i)
	}

	// Reader goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, err := s.ListEntries(context.Background(), false, "newest", 10, 0, "", "")
				if err != nil {
					t.Errorf("Worker %d failed to read entries: %v", workerID, err)
				}
				time.Sleep(time.Millisecond) // Yield slightly
			}
		}(i)
	}

	wg.Wait()

	// Verify total inserted count
	entries, err := s.ListEntries(context.Background(), false, "newest", 0, 0, "", "")
	if err != nil {
		t.Fatalf("Failed to list entries at the end: %v", err)
	}

	expectedCount := numWorkers * iterations
	if len(entries) != expectedCount {
		t.Errorf("Expected %d entries, found %d", expectedCount, len(entries))
	}
}

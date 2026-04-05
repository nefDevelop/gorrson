package db

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/nef734/gorrson/internal/models"
)

func BenchmarkSearch(b *testing.B) {
	dataDir := "./test_data_bench"
	os.MkdirAll(dataDir, 0755)
	defer os.RemoveAll(dataDir)

	store, _ := NewStore(dataDir, "bench_user")
	defer store.Close()

	// Seed with 1000 items
	feedID := uuid.New().String()
	store.AddFeed(&models.Feed{ID: feedID, Title: "Bench Feed", FeedURL: "http://bench.com/rss"})

	for i := 0; i < 1000; i++ {
		store.AddEntry(&models.Entry{
			ID:      uuid.New().String(),
			FeedID:  feedID,
			Title:   fmt.Sprintf("Item %d with keyword gorrson", i),
			Content: "This is some content for benchmarking the search engine speed.",
			URL:     fmt.Sprintf("http://bench.com/%d", i),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.SearchEntries("gorrson")
	}
}

func BenchmarkListNews(b *testing.B) {
	dataDir := "./test_data_bench_list"
	os.MkdirAll(dataDir, 0755)
	defer os.RemoveAll(dataDir)

	store, _ := NewStore(dataDir, "bench_user_list")
	defer store.Close()

	feedID := uuid.New().String()
	store.AddFeed(&models.Feed{ID: feedID, Title: "Bench Feed", FeedURL: "http://bench.com/rss"})

	for i := 0; i < 5000; i++ {
		store.AddEntry(&models.Entry{
			ID:      uuid.New().String(),
			FeedID:  feedID,
			Title:   fmt.Sprintf("Item %d", i),
			URL:     fmt.Sprintf("http://bench.com/%d", i),
			IsRead:  false,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.ListEntries(true, "newest", 50, 0, "")
	}
}

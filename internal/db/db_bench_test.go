package db

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/nef734/gorrson/internal/models"
)

func BenchmarkSearch(b *testing.B) {
	dataDir := "./test_data_bench"
	_ = os.MkdirAll(dataDir, 0755)
	defer func() { _ = os.RemoveAll(dataDir) }()

	store, _ := NewStore(dataDir, "bench_user")
	defer func() { _ = store.Close() }()

	feedID := uuid.New().String()
	_ = store.AddFeed(context.Background(), &models.Feed{ID: feedID, Title: "Bench Feed", FeedURL: "http://bench.com/rss"})

	for i := 0; i < 1000; i++ {
		_ = store.AddEntry(context.Background(), &models.Entry{
			ID:      uuid.New().String(),
			FeedID:  feedID,
			Title:   fmt.Sprintf("Item %d with keyword gorrson", i),
			Content: "This is some content for benchmarking the search engine speed.",
			URL:     fmt.Sprintf("http://bench.com/%d", i),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.SearchEntries(context.Background(), "gorrson", 50, 0)
	}
}

func BenchmarkListNews(b *testing.B) {
	dataDir := "./test_data_bench_list"
	_ = os.MkdirAll(dataDir, 0755)
	defer func() { _ = os.RemoveAll(dataDir) }()

	store, _ := NewStore(dataDir, "bench_user_list")
	defer func() { _ = store.Close() }()

	feedID := uuid.New().String()
	_ = store.AddFeed(context.Background(), &models.Feed{ID: feedID, Title: "Bench Feed", FeedURL: "http://bench.com/rss"})

	for i := 0; i < 5000; i++ {
		_ = store.AddEntry(context.Background(), &models.Entry{
			ID:      uuid.New().String(),
			FeedID:  feedID,
			Title:   fmt.Sprintf("Item %d", i),
			URL:     fmt.Sprintf("http://bench.com/%d", i),
			IsRead:  false,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.ListEntries(context.Background(), true, "newest", 50, 0, "", "")
	}
}

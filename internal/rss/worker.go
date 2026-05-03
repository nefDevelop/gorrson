package rss

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/nef734/gorrson/internal/db"
)

// Worker handles the periodic synchronization of RSS feeds for a given database.
type Worker struct {
	store    *db.Store
	interval time.Duration
	parser   *Parser
}

func NewWorker(store *db.Store, interval time.Duration) *Worker {
	return &Worker{
		store:    store,
		interval: interval,
		parser:   NewParser(),
	}
}

// Global semaphore to limit concurrent user sync cycles across all goroutines.
var syncSemaphore = make(chan struct{}, 10)

// Start runs the sync loop until context is canceled.
func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Initial sync
	w.SyncAll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.SyncAll(ctx)
		}
	}
}

// SyncAll syncs all feeds currently in the database in parallel.
func (w *Worker) SyncAll(ctx context.Context) {
	// Acquire semaphore to limit concurrent users
	syncSemaphore <- struct{}{}
	defer func() { <-syncSemaphore }()

	start := time.Now()
	feeds, err := w.store.ListFeeds()
	if err != nil {
		slog.Error("Worker error listing feeds", slog.Any("error", err))
		return
	}

	user := w.store.GetUsername()
	slog.Info("Starting parallel sync cycle", slog.String("user", user), slog.Int("count", len(feeds)))

	// Limit concurrency for individual feeds within this user's cycle
	feedSem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	for _, f := range feeds {
		wg.Add(1)
		feedSem <- struct{}{}
		go func(feedURL string) {
			defer wg.Done()
			defer func() { <-feedSem }()
			
			// Use a sub-context with timeout for each individual feed sync
			feedCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()
			
			if err := w.parser.SyncFeed(feedCtx, feedURL, w.store); err != nil {
				slog.Error("Error syncing feed", slog.String("url", feedURL), slog.Any("error", err), slog.String("user", user))
			}
		}(f.FeedURL)
	}

	wg.Wait()

	// Retention policy: Cleanup old entries after sync
	if count, err := w.store.CleanupOldEntries(); err == nil && count > 0 {
		slog.Info("Cleaned up old entries", slog.Int64("count", count), slog.String("user", user))
	}
	slog.Info("Parallel sync cycle complete", slog.String("user", user), slog.Duration("duration", time.Since(start)))
}

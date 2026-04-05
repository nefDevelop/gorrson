package rss

import (
	"context"
	"log/slog"
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

// SyncAll syncs all feeds currently in the database.
func (w *Worker) SyncAll(ctx context.Context) {
	start := time.Now()
	feeds, err := w.store.ListFeeds()
	if err != nil {
		slog.Error("Worker error listing feeds", slog.Any("error", err))
		return
	}

	user := w.store.GetUsername()
	slog.Info("Starting sync cycle", slog.String("user", user), slog.Int("count", len(feeds)))

	for _, f := range feeds {
		// Log sync activity but continue on failure for a single feed
		if err := w.parser.SyncFeed(ctx, f.FeedURL, w.store); err != nil {
			slog.Error("Error syncing feed", slog.String("url", f.FeedURL), slog.Any("error", err), slog.String("user", user))
		}
	}

	// Retention policy: Cleanup old entries after sync
	if count, err := w.store.CleanupOldEntries(); err == nil && count > 0 {
		slog.Info("Cleaned up old entries", slog.Int64("count", count), slog.String("user", user))
	}
	slog.Info("Sync cycle complete", slog.String("user", user), slog.Duration("duration", time.Since(start)))
}

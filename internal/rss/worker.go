package rss

import (
	"context"
	"log"
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
	feeds, err := w.store.ListFeeds()
	if err != nil {
		log.Printf("[Worker] Error listing feeds: %v", err)
		return
	}

	for _, f := range feeds {
		// Log sync activity but continue on failure for a single feed
		if err := w.parser.SyncFeed(ctx, f.FeedURL, w.store); err != nil {
			log.Printf("[Worker] Error syncing %s: %v", f.FeedURL, err)
		}
	}
}

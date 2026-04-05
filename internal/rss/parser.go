package rss

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
	"github.com/nef734/gorrson/internal/db"
	"github.com/nef734/gorrson/internal/models"
	"github.com/nef734/gorrson/internal/sanitizer"
)

type Parser struct {
	fp *gofeed.Parser
}

func NewParser() *Parser {
	return &Parser{
		fp: gofeed.NewParser(),
	}
}

// SyncFeed parses a feed and saves it and its entries to the database.
func (p *Parser) SyncFeed(ctx context.Context, url string, store *db.Store) error {
	feed, err := p.fp.ParseURLWithContext(url, ctx)
	if err != nil {
		return fmt.Errorf("could not parse feed at %s: %w", url, err)
	}

	// Check if feed exists to maintain the ID
	existing, _ := store.GetFeedByURL(url)
	feedID := uuid.New().String()
	if existing != nil {
		feedID = existing.ID
	}

	f := &models.Feed{
		ID:          feedID,
		Title:       feed.Title,
		SiteURL:     feed.Link,
		FeedURL:     url,
		LastUpdated: time.Now(),
	}

	if err := store.AddFeed(f); err != nil {
		return fmt.Errorf("could not save feed metadata: %w", err)
	}

	for _, item := range feed.Items {
		content := item.Content
		if content == "" {
			content = item.Description
		}

		pubDate := time.Now()
		if item.PublishedParsed != nil {
			pubDate = *item.PublishedParsed
		}

		cleanContent := sanitizer.Clean(content)

		// Generate summary (plain text, first 200 chars)
		summary := item.Description
		if len(summary) > 200 {
			summary = summary[:197] + "..."
		}

		// Try to find an image
		imageURL := ""
		if item.Image != nil {
			imageURL = item.Image.URL
		} else if len(item.Enclosures) > 0 {
			for _, e := range item.Enclosures {
				if strings.Contains(e.Type, "image") {
					imageURL = e.URL
					break
				}
			}
		}

		entry := &models.Entry{
			ID:          uuid.New().String(),
			FeedID:      feedID,
			Title:       item.Title,
			Summary:     summary,
			Content:     cleanContent,
			URL:         item.Link,
			ImageURL:    imageURL,
			PublishedAt: pubDate,
			IsRead:      false,
			IsFavorite:  false,
		}

		if err := store.AddEntry(entry); err != nil {
			// Skip single entry errors (e.g., duplicated link)
			continue
		}
	}

	return nil
}

}

package rss

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
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
	slog.Info("SyncFeed starting", "url", url, "user", store.GetUsername())

	feed, err := p.fp.ParseURLWithContext(url, ctx)
	if err != nil {
		// Try discovery if initial parse fails
		slog.Info("Direct parse failed, trying discovery", "url", url, "err", err)
		discoveredURL, discErr := DiscoverFeed(ctx, url)
		if discErr == nil && discoveredURL != url {
			url = discoveredURL
			feed, err = p.fp.ParseURLWithContext(url, ctx)
		}
		
		if err != nil {
			slog.Error("SyncFeed failed", "url", url, "err", err)
			return fmt.Errorf("could not parse feed at %s (tried discovery): %w", url, err)
		}
	}

	slog.Info("Feed parsed", "title", feed.Title, "items", len(feed.Items), "url", url)

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
		slog.Error("Failed to save feed", "err", err)
		return fmt.Errorf("could not save feed metadata: %w", err)
	}
	slog.Info("Feed saved to DB", "title", feed.Title, "id", feedID)

	filters, _ := store.GetFiltersForFeed(feedID)

	type compiledFilter struct {
		f  models.Filter
		re *regexp.Regexp
	}
	var readyFilters []compiledFilter
	for _, filter := range filters {
		re, err := regexp.Compile("(?i)" + filter.Pattern)
		if err == nil {
			readyFilters = append(readyFilters, compiledFilter{f: filter, re: re})
		}
	}

	saved := 0
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
		title := item.Title

		// Apply Filters
		skip := false
		for _, rf := range readyFilters {
			filter := rf.f
			re := rf.re

			matchTitle := filter.Target == "title" || filter.Target == "both"
			matchContent := filter.Target == "content" || filter.Target == "both"

			if (matchTitle && re.MatchString(title)) || (matchContent && re.MatchString(cleanContent)) {
				if filter.Action == "BLOCK" {
					skip = true
					break
				}
				if filter.Action == "REPLACE" {
					if matchTitle {
						title = re.ReplaceAllString(title, filter.Replacement)
					}
					if matchContent {
						cleanContent = re.ReplaceAllString(cleanContent, filter.Replacement)
					}
				}
			}
		}

		if skip {
			continue
		}

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
			Title:       title,
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
		saved++
	}

	slog.Info("SyncFeed complete", "feed", feed.Title, "total_items", len(feed.Items), "saved_entries", saved)
	return nil
}

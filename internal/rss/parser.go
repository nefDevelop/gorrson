package rss

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
	"github.com/nef734/gorrson/internal/db"
	"github.com/nef734/gorrson/internal/models"
	"github.com/nef734/gorrson/internal/sanitizer"
)

type transport struct {
	rt   http.RoundTripper
	ua   string
	user string
	pass string
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.ua != "" {
		req.Header.Set("User-Agent", t.ua)
	}
	if t.user != "" || t.pass != "" {
		req.SetBasicAuth(t.user, t.pass)
	}
	return t.rt.RoundTrip(req)
}

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

	// Get feed settings if it exists
	existing, _ := store.GetFeedByURL(url)
	
	ua := "Gorrson/1.2 (+https://github.com/nef734/gorrson)"
	user := ""
	pass := ""

	if existing != nil {
		if existing.UserAgent != "" {
			ua = existing.UserAgent
		}
		user = existing.Username
		pass = existing.Password
	}

	// Automatic User-Agent for known social domains to avoid blocking
	if (existing == nil || existing.UserAgent == "") && 
	   (strings.Contains(url, "reddit.com") || strings.Contains(url, "mastodon") || strings.Contains(url, "bsky.app")) {
		ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36"
	}

	// Create a local parser to be thread-safe with custom client
	fp := gofeed.NewParser()
	fp.Client = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &transport{
			rt:   http.DefaultTransport,
			ua:   ua,
			user: user,
			pass: pass,
		},
	}

	feed, err := fp.ParseURLWithContext(url, ctx)
	if err != nil {
		// Try discovery if initial parse fails
		slog.Info("Direct parse failed, trying discovery", "url", url, "err", err)
		discoveredURL, discErr := DiscoverFeed(ctx, url)
		if discErr == nil && discoveredURL != url {
			url = discoveredURL
			feed, err = fp.ParseURLWithContext(url, ctx)
		}
		
		if err != nil {
			slog.Error("SyncFeed failed", "url", url, "err", err)
			return fmt.Errorf("could not parse feed at %s (tried discovery): %w", url, err)
		}
	}

	slog.Info("Feed parsed", "title", feed.Title, "items", len(feed.Items), "url", url)

	// Check if feed exists to maintain the ID
	existing, _ = store.GetFeedByURL(url)
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
		if cleanContent == "" && title != "" {
			slog.Debug("Entry content is empty, only title available", "title", title, "link", item.Link)
		}

		// Apply Filters
		matchedAction := ""
		skip := false
		for _, rf := range readyFilters {
			filter := rf.f
			re := rf.re

			matchTitle := filter.Target == "title" || filter.Target == "both"
			matchContent := filter.Target == "content" || filter.Target == "both"

			if (matchTitle && re.MatchString(title)) || (matchContent && re.MatchString(cleanContent)) {
				if filter.Action == "BLOCK" || filter.Action == "HIDE" {
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
				if filter.Action == "SILENT" {
					matchedAction = "SILENT"
				}
				if filter.Action == "HIGHLIGHT" {
					matchedAction = "HIGHLIGHT"
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

		// Special handling for Reddit: extract clean link and image from HTML content
		isReddit := strings.Contains(url, "reddit.com")
		if isReddit {
			reLink := regexp.MustCompile(`<span><a href="([^"]+)">\[link\]</a></span>`)
			if matches := reLink.FindStringSubmatch(content); len(matches) > 1 {
				item.Link = matches[1]
			}
			
			// Extract image if missing
			if imageURL == "" {
				reImg := regexp.MustCompile(`<img src="([^"]+)"`)
				if matches := reImg.FindStringSubmatch(content); len(matches) > 1 {
					imageURL = matches[1]
				}
			}
		}

		// Special handling for Mastodon & Bluesky
		isMastodon := strings.Contains(url, "/@") || strings.Contains(feed.Generator, "Mastodon")
		isBluesky := strings.Contains(url, "bsky.app")
		if (isMastodon || isBluesky) && imageURL == "" && feed.Image != nil {
			imageURL = feed.Image.URL
		}

		entry := &models.Entry{
			ID:             uuid.New().String(),
			FeedID:         feedID,
			Title:          title,
			Summary:        summary,
			Content:        cleanContent,
			URL:            item.Link,
			ImageURL:       imageURL,
			PublishedAt:    pubDate,
			IsRead:         false,
			IsFavorite:     false,
			IsFullContent:  false,
			MatchedAction:  matchedAction,
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

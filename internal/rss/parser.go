package rss

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
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

// generateFeverID converts a string ID to a 32-bit integer
func generateFeverID(id string) int {
	h := fnv.New32a()
	h.Write([]byte(id))
	return int(h.Sum32())
}

// SyncFeed parses a feed and saves it and its entries to the database.
func (p *Parser) SyncFeed(ctx context.Context, url string, store *db.Store) error {
	slog.Info("SyncFeed starting", "url", url, "user", store.GetUsername())

	// Get feed settings if it exists
	existing, _ := store.GetFeedByURL(url)

	// Default to a common browser user-agent to avoid blocking.
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36"
	user := ""
	pass := ""

	if existing != nil {
		if existing.UserAgent != "" {
			ua = existing.UserAgent
		}
		user = existing.Username
		pass = existing.Password
	}

	// Use the shared parser but with a per-request client configuration if needed
	p.fp.Client = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &transport{
			rt:   http.DefaultTransport,
			ua:   ua,
			user: user,
			pass: pass,
		},
	}

	feed, err := p.fp.ParseURLWithContext(url, ctx)
	if err != nil {
		// If the URL doesn't obviously look like a feed, try discovery.
		isFeedURL := strings.HasSuffix(url, ".rss") || strings.HasSuffix(url, ".xml") || strings.HasSuffix(url, ".atom") || strings.HasSuffix(url, "/feed")

		if !isFeedURL {
			slog.Info("Direct parse failed, trying discovery as URL is not a typical feed endpoint", "url", url, "err", err)
			discoveredURL, discErr := DiscoverFeed(ctx, url)
			if discErr == nil && discoveredURL != url {
				slog.Info("Discovered new feed URL", "original", url, "discovered", discoveredURL)
				url = discoveredURL // Update the URL we are working with
				feed, err = p.fp.ParseURLWithContext(url, ctx)
			}
		}

		if err != nil {
			slog.Error("SyncFeed failed", "url", url, "err", err)
			return fmt.Errorf("could not parse feed at %s: %w", url, err)
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
		FeverID:     generateFeverID(feedID),
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

	// Use a transaction for batch insertion of entries
	tx, err := store.Begin()
	if err != nil {
		slog.Error("Failed to start transaction", "err", err)
		return fmt.Errorf("could not start transaction: %w", err)
	}
	defer tx.Rollback()

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

		// Identify social media feeds
		isBluesky := strings.Contains(url, "bsky.app")
		isMastodon := strings.Contains(url, "/@") || strings.Contains(feed.Generator, "Mastodon")

		// Improve titles for social media if they are generic or missing
		if isBluesky || isMastodon || title == "" {
			// Check if title is generic or just a handle
			isGeneric := strings.HasPrefix(title, "Post by ") ||
				strings.HasPrefix(title, "A post from ") ||
				strings.HasPrefix(title, "@") ||
				len(title) < 3

			if title == "" || isGeneric {
				// Use a snippet of the content as a better title
				textOnly := sanitizer.StripHTML(cleanContent)
				if textOnly != "" {
					lines := strings.Split(strings.TrimSpace(textOnly), "\n")
					firstLine := lines[0]
					if len(firstLine) > 100 {
						firstLine = firstLine[:97] + "..."
					}
					title = firstLine
				} else if title == "" {
					// Hard fallback if everything is empty
					title = "Post from " + feed.Title
				}
			}
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
				if filter.Action == "IMPORTANT" || filter.Action == "FIRST" {
					matchedAction = "IMPORTANT"
				}
			}
		}

		if skip {
			continue
		}

		// Generate summary
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

		// Try media:content extension
		if imageURL == "" {
			if media, ok := item.Extensions["media"]; ok {
				if contents, ok := media["content"]; ok && len(contents) > 0 {
					for _, c := range contents {
						if strings.Contains(c.Attrs["type"], "image") || c.Attrs["medium"] == "image" {
							imageURL = c.Attrs["url"]
							break
						}
					}
				}
			}
		}

		if imageURL == "" {
			imageURL = extractFirstImageFromHTML(content)
		}

		if (isBluesky || isMastodon) && imageURL == "" && item.Link != "" {
			imageURL = fetchOGImage(ctx, item.Link)
		}

		normalizedURL := normalizeURL(item.Link)

		entry := &models.Entry{
			ID:            uuid.New().String(),
			FeedID:        feedID,
			Title:         title,
			Summary:       summary,
			Content:       cleanContent,
			URL:           normalizedURL,
			ImageURL:      imageURL,
			PublishedAt:   pubDate,
			IsRead:        false,
			IsFavorite:    false,
			IsFullContent: false,
			IsPriority:    matchedAction == "IMPORTANT",
			MatchedAction: matchedAction,
		}
		entry.FeverID = generateFeverID(entry.ID)

		if err := store.AddEntryTx(tx, entry); err != nil {
			continue
		}
		saved++
	}

	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit transaction", "err", err)
		return fmt.Errorf("could not commit entries: %w", err)
	}

	slog.Info("SyncFeed complete", "feed", feed.Title, "total_items", len(feed.Items), "saved_entries", saved)
	return nil
}

func normalizeURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.Fragment = ""
	q := u.Query()
	trackingParams := []string{"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content", "ref", "fbclid", "gclid"}
	for _, p := range trackingParams {
		q.Del(p)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func extractFirstImageFromHTML(html string) string {
	if html == "" {
		return ""
	}
	re := regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+\.(jpg|jpeg|png|webp|gif|svg)[^"']*)["']`)
	matches := re.FindAllStringSubmatch(html, -1)
	for _, match := range matches {
		if len(match) > 1 {
			return match[1]
		}
	}
	return ""
}

func fetchOGImage(ctx context.Context, pageURL string) string {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	sc := NewSecureClient("")
	body, err := sc.Fetch(ctx, pageURL)
	if err != nil {
		return ""
	}
	defer body.Close()
	limitedBody := io.LimitReader(body, 500*1024)
	doc, err := goquery.NewDocumentFromReader(limitedBody)
	if err != nil {
		return ""
	}
	var img string
	doc.Find("meta[property='og:image'], meta[name='twitter:image']").Each(func(i int, s *goquery.Selection) {
		if content, ok := s.Attr("content"); ok && img == "" {
			img = content
		}
	})
	if img != "" && !strings.HasPrefix(img, "http") {
		base, err := url.Parse(pageURL)
		if err == nil {
			ref, err := url.Parse(img)
			if err == nil {
				img = base.ResolveReference(ref).String()
			}
		}
	}
	return img
}

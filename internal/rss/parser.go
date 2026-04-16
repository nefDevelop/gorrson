package rss

import (
	"context"
	"fmt"
	"hash/fnv"
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
		// If the URL doesn't obviously look like a feed, try discovery.
		isFeedURL := strings.HasSuffix(url, ".rss") || strings.HasSuffix(url, ".xml") || strings.HasSuffix(url, ".atom") || strings.HasSuffix(url, "/feed")

		if !isFeedURL {
			slog.Info("Direct parse failed, trying discovery as URL is not a typical feed endpoint", "url", url, "err", err)
			discoveredURL, discErr := DiscoverFeed(ctx, url)
			if discErr == nil && discoveredURL != url {
				slog.Info("Discovered new feed URL", "original", url, "discovered", discoveredURL)
				url = discoveredURL // Update the URL we are working with
				feed, err = fp.ParseURLWithContext(url, ctx)
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
				if filter.Action == "IMPORTANT" || filter.Action == "FIRST" {
					matchedAction = "IMPORTANT"
				}
				if filter.Action == "NOTIFY" {
					matchedAction = "NOTIFY"
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

		// Try media:content extension (Media RSS)
		if imageURL == "" {
			if media, ok := item.Extensions["media"]; ok {
				if contents, ok := media["content"]; ok && len(contents) > 0 {
					// Take the first image content
					for _, c := range contents {
						if strings.Contains(c.Attrs["type"], "image") || c.Attrs["medium"] == "image" {
							imageURL = c.Attrs["url"]
							break
						}
					}
					// Fallback to first one if no type specified
					if imageURL == "" && len(contents) > 0 {
						imageURL = contents[0].Attrs["url"]
					}
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
		}

		// Extract image from content if still missing
		if imageURL == "" {
			imageURL = extractFirstImageFromHTML(content)
		}

		// Special handling for Mastodon & Bluesky

		// If STILL no image and it's a social feed, try to fetch OG image from the page
		// because official feeds often have limited media in RSS.
		if (isBluesky || isMastodon) && imageURL == "" && item.Link != "" {
			imageURL = fetchOGImage(ctx, item.Link)
		}

		// Last resort: fallback to feed image (avatar) for social platforms
		if (isMastodon || isBluesky) && imageURL == "" && feed.Image != nil {
			imageURL = feed.Image.URL
		}

		// Normalize URL to prevent duplicates (remove UTM and tracking params)
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

		if err := store.AddEntry(entry); err != nil {
			// Skip single entry errors (e.g., duplicated link)
			continue
		}
		saved++
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

	// Remove fragment
	u.Fragment = ""

	// Common tracking parameters to remove
	q := u.Query()
	trackingParams := []string{
		"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content",
		"ref", "referrer", "source", "tracker",
		"fbclid", "gclid", "msclkid",
		"_hsenc", "_hsmi", "mc_cid", "mc_eid",
	}

	changed := false
	for _, p := range trackingParams {
		if q.Get(p) != "" {
			q.Del(p)
			changed = true
		}
	}

	if changed {
		u.RawQuery = q.Encode()
	}

	return u.String()
}

func extractFirstImageFromHTML(html string) string {
	if html == "" {
		return ""
	}
	// Simple regex for speed, checking for common image extensions
	re := regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+\.(jpg|jpeg|png|webp|gif|svg)[^"']*)["']`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		src := matches[1]
		// Filter out common tracking pixels or icons
		lower := strings.ToLower(src)
		if strings.Contains(lower, "pixel") || strings.Contains(lower, "spacer") || strings.Contains(lower, "1x1") || strings.Contains(lower, "tracking") {
			return ""
		}
		return src
	}
	return ""
}

func fetchOGImage(ctx context.Context, pageURL string) string {
	// Don't spend too much time on this
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	sc := NewSecureClient("")
	body, err := sc.Fetch(ctx, pageURL)
	if err != nil {
		return ""
	}
	defer body.Close()

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return ""
	}

	var img string
	// Try various OG and Twitter tags
	doc.Find("meta[property='og:image'], meta[name='twitter:image'], meta[property='og:image:url']").Each(func(i int, s *goquery.Selection) {
		if content, ok := s.Attr("content"); ok && img == "" {
			img = content
		}
	})

	// Resolve relative URLs
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

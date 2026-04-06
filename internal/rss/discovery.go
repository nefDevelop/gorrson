package rss

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// DiscoverFeed attempts to find a feed URL in an HTML page.
func DiscoverFeed(ctx context.Context, pageURL string) (string, error) {
	slog.Info("DiscoverFeed: fetching page", "url", pageURL)
	sc := NewSecureClient()
	body, err := sc.Fetch(ctx, pageURL)
	if err != nil {
		slog.Warn("DiscoverFeed: fetch failed", "url", pageURL, "err", err)
		return "", err
	}
	defer body.Close()

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return "", err
	}

	base, err := url.Parse(pageURL)
	if err != nil {
		return "", fmt.Errorf("invalid page URL: %w", err)
	}

	var feedURL string
	doc.Find("link[rel='alternate']").Each(func(i int, s *goquery.Selection) {
		if feedURL != "" {
			return // Already found one
		}

		t, _ := s.Attr("type")
		if strings.Contains(t, "rss") || strings.Contains(t, "atom") {
			href, _ := s.Attr("href")
			if href != "" {
				ref, err := url.Parse(href)
				if err != nil {
					return
				}
				// Correctly resolve relative, protocol-relative, and absolute URLs
				feedURL = base.ResolveReference(ref).String()
			}
		}
	})

	if feedURL == "" {
		slog.Warn("DiscoverFeed: no feed link found on page", "url", pageURL)
		return "", fmt.Errorf("no feed found on page")
	}

	slog.Info("DiscoverFeed: found feed", "page", pageURL, "feed", feedURL)
	return feedURL, nil
}

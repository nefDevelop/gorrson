package rss

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// DiscoverFeed attempts to find a feed URL in an HTML page.
func DiscoverFeed(ctx context.Context, pageURL string) (string, error) {
	sc := NewSecureClient()
	body, err := sc.Fetch(ctx, pageURL)
	if err != nil {
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
		return "", fmt.Errorf("no feed found on page")
	}

	return feedURL, nil
}

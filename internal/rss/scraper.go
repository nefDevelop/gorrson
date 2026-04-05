package rss

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/nef734/gorrson/internal/sanitizer"
)

// ScrapeFullContent attempts to extract the main article text from a URL.
func ScrapeFullContent(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) gorrson/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	// Remove noise
	doc.Find("script, style, noscript, iframe, nav, footer, header, aside, .ads, .sidebar").Remove()

	// Heuristic: Try to find the main content
	var mainContent *goquery.Selection
	
	// 1. Try common tags
	if sel := doc.Find("article"); sel.Length() > 0 {
		mainContent = sel.First()
	} else if sel := doc.Find("main"); sel.Length() > 0 {
		mainContent = sel.First()
	} else if sel := doc.Find("[role='main']"); sel.Length() > 0 {
		mainContent = sel.First()
	} else {
		// 2. Fallback: Find the div with the most paragraphs
		maxScore := 0
		doc.Find("div").Each(func(i int, s *goquery.Selection) {
			pCount := s.Find("p").Length()
			if pCount > maxScore {
				maxScore = pCount
				mainContent = s
			}
		})
	}

	if mainContent == nil || mainContent.Length() == 0 {
		// Final fallback: Use body
		mainContent = doc.Find("body")
	}

	html, err := mainContent.Html()
	if err != nil {
		return "", err
	}

	// Final clean with our sanitizer
	return sanitizer.Clean(html), nil
}

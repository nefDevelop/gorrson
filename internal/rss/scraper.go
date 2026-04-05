package rss

import (
	"context"
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/nef734/gorrson/internal/sanitizer"
	"golang.org/x/net/html/charset"
)

// ScrapeFullContent attempts to extract the main article text from a URL.
func ScrapeFullContent(ctx context.Context, url string) (string, error) {
	sc := NewSecureClient()
	body, err := sc.Fetch(ctx, url)
	if err != nil {
		return "", err
	}
	defer body.Close()

	// Detect and convert charset to UTF-8
	utf8Body, err := decodeToUTF8(body)
	if err != nil {
		return "", err
	}

	doc, err := goquery.NewDocumentFromReader(utf8Body)
	if err != nil {
		return "", err
	}

	// 1. Remove obvious noise
	doc.Find("script, style, noscript, iframe, nav, footer, header, aside").Remove()
	doc.Find(".ads, .sidebar, .comments, .share, .related, .menu, .nav, .footer").Remove()
	doc.Find("#sidebar, #footer, #header, #comments, #menu").Remove()

	// 2. Heuristic: Try to find the main content
	var mainContent *goquery.Selection

	// Priority 1: Semantic tags
	if sel := doc.Find("article"); sel.Length() > 0 {
		mainContent = sel.First()
	} else if sel := doc.Find("main"); sel.Length() > 0 {
		mainContent = sel.First()
	} else if sel := doc.Find("[role='main']"); sel.Length() > 0 {
		mainContent = sel.First()
	} else {
		// Priority 2: Common article-related classes
		selectors := []string{
			".post-content", ".entry-content", ".article-body", 
			".content-body", ".post-text", ".story-content",
		}
		for _, s := range selectors {
			if sel := doc.Find(s); sel.Length() > 0 {
				mainContent = sel.First()
				break
			}
		}
	}

	// Priority 3: Fallback - Find the div/section with the most paragraphs
	if mainContent == nil {
		maxScore := 0
		doc.Find("div, section").Each(func(i int, s *goquery.Selection) {
			// Scoring: count paragraphs, but weigh them by length
			score := 0
			s.Find("p").Each(func(j int, p *goquery.Selection) {
				textLen := len(strings.TrimSpace(p.Text()))
				if textLen > 20 {
					score += 1 + (textLen / 50)
				}
			})

			// Penalty for elements with suspicious IDs/classes that weren't removed
			id, _ := s.Attr("id")
			class, _ := s.Attr("class")
			lowcase := strings.ToLower(id + " " + class)
			if strings.Contains(lowcase, "comment") || strings.Contains(lowcase, "sidebar") {
				score /= 2
			}

			if score > maxScore {
				maxScore = score
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

	// Final clean with our sanitizer (which uses bluemonday)
	return sanitizer.Clean(html), nil
}

// decodeToUTF8 detects the encoding and converts the reader to UTF-8 if necessary.
func decodeToUTF8(r io.Reader) (io.Reader, error) {
	// We need to peek at the content to detect the charset
	// Since we are using a limited reader already, we'll just read what we have
	// and use the charset detector.
	
	// goquery.NewDocumentFromReader expects an io.Reader, but if we read all now,
	// we'd need to provide a new reader.
	
	// A better way is using charset.NewReader which handles the conversion.
	// But it needs an initial hint from the Content-Type header usually.
	// For simplicity and robustness, we'll let it sniff.
	
	utf8Reader, err := charset.NewReader(r, "")
	if err != nil {
		return nil, err
	}
	return utf8Reader, nil
}

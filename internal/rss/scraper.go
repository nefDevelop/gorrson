package rss

import (
	"context"
	"io"
	"strings"

	"net/url"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-shiori/go-readability"
	"github.com/nef734/gorrson/internal/sanitizer"
	"golang.org/x/net/html/charset"
)

// ScrapeFullContent attempts to extract the main article text from a URL.
func ScrapeFullContent(ctx context.Context, targetURL string, originalTitle string, originalImage string) (string, error) {
	sc := NewSecureClient()
	body, err := sc.Fetch(ctx, targetURL)
	if err != nil {
		return "", err
	}
	defer func() { _ = body.Close() }()

	// Detect and convert charset to UTF-8
	utf8Body, err := decodeToUTF8(body)
	if err != nil {
		return "", err
	}

	// Parse URL for readability
	parsedURL, _ := url.Parse(targetURL)

	// Use go-readability (Mozilla's algorithm) for high-performance extraction
	article, err := readability.FromReader(utf8Body, parsedURL)
	if err != nil {
		return "", err
	}

	// Remove duplicate titles and hero images already shown in UI
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(article.Content))
	if err == nil {
		// Clean duplicate titles (checks h1, h2, h3)
		doc.Find("h1, h2, h3").Each(func(i int, s *goquery.Selection) {
			if strings.TrimSpace(s.Text()) == strings.TrimSpace(originalTitle) {
				s.Remove()
			}
		})

		// Clean duplicate hero image
		if originalImage != "" {
			doc.Find("img").Each(func(i int, s *goquery.Selection) {
				src, _ := s.Attr("src")
				if src == originalImage {
					s.Remove()
				}
			})
		}
		
		article.Content, _ = doc.Html()
	}

	// Final clean with our sanitizer (which uses bluemonday)
	return sanitizer.Clean(article.Content), nil
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

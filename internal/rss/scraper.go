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
func ScrapeFullContent(ctx context.Context, url string, originalTitle string, originalImage string) (string, error) {
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
	doc.Find("script, style, noscript, iframe, nav, footer, header, aside, form").Remove()
	doc.Find(".ads, .sidebar, .comments, .share, .related, .menu, .nav, .footer").Remove()
	doc.Find("#sidebar, #footer, #header, #comments, #menu").Remove()
	doc.Find(".social-share, .share-box, .newsletter, .signup, .promo, .follow-us, .tags-list, .author-bio, .share-buttons, .social-platforms, .share-story").Remove()
	doc.Find(".entry-footer, .post-footer, .article-footer, .related-posts, .bbc-sport-related, [data-component='topic-list']").Remove()

	// Text-based filtering for common "noise" phrases
	doc.Find("div, section, p, h3, h4, span").Each(func(i int, s *goquery.Selection) {
		txt := strings.ToLower(s.Text())
		noise := []string{
			// English
			"share this story", "subscribe to our", "sign up for", "follow us on",
			"read more:", "related:", "privacy policy", "terms of use", "account if you don't already have one",
			"related topics", "more on this story", "follow your club with", "share on",
			// Spanish
			"comparte en", "suscríbete a", "síguenos en", "más información",
			"política de privacidad", "términos de uso", "leer más", "noticias relacionadas",
			"regístrate para", "iniciar sesión", "crear cuenta", "temas relacionados", "compartir en",
		}
		for _, phrase := range noise {
			if strings.Contains(txt, phrase) && len(txt) < 300 { // Small blocks only to avoid false positives
				s.Remove()
				break
			}
		}
	})

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

	// Remove headings that match the original title
	if originalTitle != "" {
		normalizedTitle := strings.ToLower(strings.TrimSpace(originalTitle))
		mainContent.Find("h1, h2, h3").Each(func(i int, s *goquery.Selection) {
			headingText := strings.ToLower(strings.TrimSpace(s.Text()))
			if headingText == normalizedTitle || strings.Contains(headingText, normalizedTitle) || strings.Contains(normalizedTitle, headingText) {
				s.Remove()
			}
		})
	}

	// Remove duplicate main images
	if originalImage != "" {
		mainContent.Find("img").Each(func(i int, s *goquery.Selection) {
			src, _ := s.Attr("src")
			if src != "" {
				// Simple comparison first
				if strings.Contains(src, originalImage) || strings.Contains(originalImage, src) {
					s.Remove()
				}
			}
		})
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

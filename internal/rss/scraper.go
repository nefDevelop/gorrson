package rss

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"net/url"

	"github.com/PuerkitoBio/goquery"
	"github.com/antchfx/htmlquery"
	"github.com/go-shiori/go-readability"
	"github.com/nef734/gorrson/internal/sanitizer"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

const DebugMode = false

func debugLog(format string, a ...interface{}) {
	if DebugMode {
		fmt.Printf(format, a...)
	}
}

// Global semaphore to limit concurrent high-memory scraping operations
var scrapeSemaphore = make(chan struct{}, 5)

// ScrapeFullContent attempts to extract the main article text from a URL.
func ScrapeFullContent(ctx context.Context, targetURL string, originalTitle string, originalImage string, userAgent string) (string, error) {
	// Limit concurrent scraping
	scrapeSemaphore <- struct{}{}
	defer func() { <-scrapeSemaphore }()

	sc := NewSecureClient(userAgent)
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

	// Capture the raw HTML tree
	root, err := html.Parse(utf8Body)
	if err != nil {
		return "", err
	}

	var extractedNode *html.Node
	var ruleInfo = "None (Auto)"

	// 1. Try Site-specific Recipe
	recipe, err := FindRecipe(targetURL)
	if err == nil && recipe != nil {
		slog.Debug("Applying recipe", "url", targetURL, "source", recipe.Source, "body_rules", len(recipe.Body))
		extractedNode, err = ApplyRecipe(root, recipe)
		if err != nil {
			slog.Debug("Recipe extraction failed", "url", targetURL, "error", err)
		} else {
			sourceName := "System"
			if recipe.Source == "custom" {
				sourceName = "Custom"
			}
			ruleInfo = fmt.Sprintf("%s Recipe: %d body, %d strip rules",
				sourceName, len(recipe.Body), len(recipe.Strip))
		}
	}

	// 2. Fallback: Use go-readability if recipe failed or didn't exist
	if extractedNode == nil {
		// Parse URL for readability
		parsedURL, _ := url.Parse(targetURL)

		// Streaming render to readability to avoid large string allocations
		pr, pw := io.Pipe()
		go func() {
			err := html.Render(pw, root)
			_ = pw.CloseWithError(err)
		}()

		article, err := readability.FromReader(pr, parsedURL)
		if err != nil {
			fmt.Printf("✘ Extraction Failed: %s | Error: %v\n", targetURL, err)
			return "", err
		}

		// Convert readability result to node
		extractedNode, err = htmlquery.Parse(strings.NewReader(article.Content))
		if err != nil {
			return article.Content, nil
		}
	}

	// 3. Post-processing with goquery (Unified DOM operations)
	doc := goquery.NewDocumentFromNode(extractedNode)
	
	// Remove duplicate titles and hero images
	cleanNoise(doc, originalTitle, originalImage)
	
	// Resolve relative URLs in-place
	resolveRelativeURLs(doc.Selection, targetURL)

	// Final stringification (only once!)
	finalContent, err := doc.Html()
	if err != nil {
		return "", err
	}

	fmt.Printf("➜ Extraction [%s]: %s | Final: %dB\n",
		ruleInfo, targetURL, len(finalContent))

	// Final clean with our sanitizer (which uses bluemonday)
	return sanitizer.Clean(finalContent), nil
}

func resolveRelativeURLs(selection *goquery.Selection, baseURL string) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return
	}

	selection.Find("img, a, source, iframe").Each(func(i int, s *goquery.Selection) {
		// Handle Lazy Loading: Move data-src to src if src is missing or a placeholder
		if s.Is("img") {
			lazyAttrs := []string{"data-src", "data-lazy-src", "data-original", "data-actualsrc"}
			src, _ := s.Attr("src")
			isPlaceholder := src == "" || strings.Contains(src, "data:image") || strings.Contains(src, "placeholder") || strings.Contains(src, "transparent")

			if isPlaceholder {
				for _, la := range lazyAttrs {
					if val, ok := s.Attr(la); ok && val != "" {
						s.SetAttr("src", val)
						break
					}
				}
			}
		}

		attrs := []string{"src", "href", "srcset"}
		for _, attr := range attrs {
			if val, ok := s.Attr(attr); ok {
				// Special handling for srcset
				if attr == "srcset" {
					parts := strings.Split(val, ",")
					for j, p := range parts {
						p = strings.TrimSpace(p)
						bits := strings.SplitN(p, " ", 2)
						if len(bits) > 0 {
							if u, err := url.Parse(bits[0]); err == nil {
								bits[0] = base.ResolveReference(u).String()
								parts[j] = strings.Join(bits, " ")
							}
						}
					}
					s.SetAttr(attr, strings.Join(parts, ", "))
					continue
				}

				if u, err := url.Parse(val); err == nil {
					s.SetAttr(attr, base.ResolveReference(u).String())
				}
			}
		}
	})

	// Wrap tables for responsiveness
	selection.Find("table").Each(func(i int, s *goquery.Selection) {
		s.WrapHtml("<div class=\"table-container\" style=\"overflow-x: auto; max-width: 100%; margin-bottom: 1em;\"></div>")
	})
}

func cleanNoise(doc *goquery.Document, originalTitle string, originalImage string) {
	// Clean duplicate titles (checks h1, h2, h3)
	doc.Find("h1, h2, h3").Each(func(i int, s *goquery.Selection) {
		if strings.TrimSpace(s.Text()) == strings.TrimSpace(originalTitle) {
			s.Remove()
		}
	})

	// Clean common "noise" elements that readability might miss
	noiseSelectors := []string{
		".byline", ".author", ".date", ".published-at",
		".social-share", ".social-buttons", ".share-links",
		".ad-unit", ".advertisement", ".sponsored",
		".photo-credit", ".credit", ".caption", ".p-caption",
		".newsletter-signup", ".login-bundle", ".login-promo",
		".registration-promo", ".subscribe-promo",
		"aside", "nav", "footer",
		".c-byline", ".c-entry-header__labels", ".c-entry-stat-list",
	}
	for _, selector := range noiseSelectors {
		doc.Find(selector).Remove()
	}

	// Clean duplicate hero image
	if originalImage != "" {
		doc.Find("img").Each(func(i int, s *goquery.Selection) {
			src, _ := s.Attr("src")
			if src == originalImage {
				s.Remove()
			}
		})
	}
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

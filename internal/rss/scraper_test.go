package rss

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestScrapeFullContent(t *testing.T) {
	// Mock server that returns a complex HTML with noise and charset
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `
		<html>
			<head><title>Test Article</title></head>
			<body>
				<header>Nav Menu</header>
				<aside class="sidebar">Ads and stuff</aside>
				<article>
					<h1>Real Title</h1>
					<p>This is the important content of the article.</p>
					<p>Multiple paragraphs make it better.</p>
				</article>
				<footer id="footer">Copyright 2026</footer>
			</body>
		</html>`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	}))
	defer ts.Close()

	ctx := context.Background()
	content, err := ScrapeFullContent(ctx, ts.URL, "", "")
	if err != nil {
		t.Fatalf("ScrapeFullContent failed: %v", err)
	}

	// Verify noise is removed
	if strings.Contains(content, "Nav Menu") || strings.Contains(content, "Ads and stuff") {
		t.Errorf("Content contains noise: %s", content)
	}

	// Verify content is extracted
	if !strings.Contains(content, "important content") {
		t.Errorf("Content missing expected text: %s", content)
	}

	// Verify semantic extraction (article tag)
	// go-readability often converts h1 to h2
	if !strings.Contains(content, "<h2>Real Title</h2>") && !strings.Contains(content, "<h1>Real Title</h1>") {
		t.Errorf("Content missing article title: %s", content)
	}
}

func TestScrapeHeuristics(t *testing.T) {
	// Mock server without semantic tags, using div with most paragraphs
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `
		<html>
			<body>
				<div class="menu">Menu</div>
				<div class="content">
					<p>Paragraph 1 of the main story.</p>
					<p>Paragraph 2 of the main story is longer and better for scoring.</p>
					<p>Paragraph 3 of the main story is also here.</p>
				</div>
				<div class="div">Short comment</div>
			</body>
		</html>`
		w.Write([]byte(html))
	}))
	defer ts.Close()

	ctx := context.Background()
	content, err := ScrapeFullContent(ctx, ts.URL, "", "")
	if err != nil {
		t.Fatalf("ScrapeFullContent failed: %v", err)
	}

	if !strings.Contains(content, "main story") {
		t.Errorf("Heuristics failed to find main story: %s", content)
	}
	
	if strings.Contains(content, "Menu") {
		t.Error("Heuristics included menu div")
	}
}

func TestDuplicateRemoval(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `
		<html>
			<body>
				<article>
					<h1>Exclusive title match</h1>
					<img src="https://example.com/hero.jpg" class="main-hero" />
					<p>Actual news content starts here.</p>
					<img src="https://example.com/other.jpg" />
				</article>
			</body>
		</html>`
		w.Write([]byte(html))
	}))
	defer ts.Close()

	ctx := context.Background()
	title := "Exclusive title match"
	hero := "https://example.com/hero.jpg"

	content, err := ScrapeFullContent(ctx, ts.URL, title, hero)
	if err != nil {
		t.Fatalf("ScrapeFullContent failed: %v", err)
	}

	// Title should be removed (cleaner UI)
	if strings.Contains(content, "Exclusive title match") {
		t.Errorf("Duplicate title was NOT removed: %s", content)
	}

	// Hero image should be removed
	if strings.Contains(content, "hero.jpg") {
		t.Errorf("Hero image was NOT removed: %s", content)
	}

	// Other image and content should remain
	if !strings.Contains(content, "other.jpg") {
		t.Errorf("Other image was accidentally removed")
	}
	if !strings.Contains(content, "Actual news content") {
		t.Errorf("Main content was removed")
	}
}

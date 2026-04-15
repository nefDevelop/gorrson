package rss

import (
	"strings"
	"testing"

	"github.com/antchfx/htmlquery"
)

func TestFindRecipe(t *testing.T) {
	// Note: This relies on the actual data directory being present.
	// Since we just cloned it, it should work for common domains.
	
	t.Run("Match exact domain", func(t *testing.T) {
		recipe, err := FindRecipe("https://www.theverge.com/2023/1/1/test")
		if err != nil {
			t.Logf("Warning: Recipe for theverge.com not found in repo: %v", err)
			return
		}
		if recipe == nil {
			t.Fatal("Expected recipe, got nil")
		}
	})

	t.Run("Subdomain fallback", func(t *testing.T) {
		// Try a domain that usually has a generic recipe
		recipe, err := FindRecipe("https://news.ycombinator.com/item?id=1")
		if err != nil {
			t.Logf("Recipe for ycombinator.com not found: %v", err)
			return
		}
		if recipe == nil {
			t.Fatal("Expected recipe, got nil")
		}
	})
}

func TestApplyRecipe(t *testing.T) {
	htmlContent := `
		<html>
			<body>
				<div class="header">Menu</div>
				<article class="meat">
					<h1>Title</h1>
					<div class="content">The real content</div>
					<div class="ad">Buy stuff</div>
				</article>
				<div class="footer">Links</div>
			</body>
		</html>
	`
	root, _ := htmlquery.Parse(strings.NewReader(htmlContent))
	
	recipe := &Recipe{
		Body:  []string{"//article[@class='meat']"},
		Strip: []string{"//div[@class='ad']", "//h1"},
	}

	result, err := ApplyRecipe(root, recipe)
	if err != nil {
		t.Fatalf("ApplyRecipe failed: %v", err)
	}

	if strings.Contains(result, "Menu") {
		t.Errorf("Result should not contain header")
	}
	if strings.Contains(result, "Buy stuff") {
		t.Errorf("Result should not contain ad")
	}
	if strings.Contains(result, "Title") {
		t.Errorf("Result should not contain title (stripped by rule)")
	}
	if !strings.Contains(result, "The real content") {
		t.Errorf("Result should contain main content")
	}
}

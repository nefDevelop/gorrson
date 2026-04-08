package opml

import (
	"strings"
	"testing"
)

func TestOPML(t *testing.T) {
	opmlData := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
  <head><title>Gorrson Export</title></head>
  <body>
    <outline text="Tech" title="Tech">
      <outline text="Go Blog" xmlUrl="https://go.dev/blog/feed.xml" htmlUrl="https://go.dev/blog" title="Tech" />
      <outline text="Hacker News" xmlUrl="https://news.ycombinator.com/rss" htmlUrl="https://news.ycombinator.com" title="Tech" />
    </outline>
    <outline text="News" title="News">
      <outline text="BBC" xmlUrl="http://feeds.bbci.co.uk/news/rss.xml" title="News" />
    </outline>
  </body>
</opml>`

	outlines, err := Parse(strings.NewReader(opmlData))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(outlines) != 3 {
		t.Errorf("Expected 3 feeds, got %d", len(outlines))
	}

	// Verify one
	found := false
	for _, o := range outlines {
		if o.Text == "Go Blog" {
			found = true
			if o.XMLURL != "https://go.dev/blog/feed.xml" {
				t.Errorf("Wrong XMLURL: %s", o.XMLURL)
			}
			break
		}
	}
	if !found {
		t.Error("Go Blog not found in parsed outlines")
	}

	// Test Generate
	gen, err := Generate(outlines)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(string(gen), "xmlUrl=\"https://go.dev/blog/feed.xml\"") {
		t.Error("Generated OPML missing feed URL")
	}
}

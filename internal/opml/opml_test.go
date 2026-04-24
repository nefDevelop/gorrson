package opml

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseOPML(t *testing.T) {
	xmlInput := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
    <head><title>My Feeds</title></head>
    <body>
        <outline text="Tech" title="Tech">
            <outline text="Ars Technica" title="Ars Technica" type="rss" xmlUrl="https://arstechnica.com/feed/" htmlUrl="https://arstechnica.com"/>
            <outline text="Wired" title="Wired" type="rss" xmlUrl="https://www.wired.com/feed/rss" htmlUrl="https://www.wired.com"/>
        </outline>
        <outline text="Hacker News" title="Hacker News" type="rss" xmlUrl="https://news.ycombinator.com/rss" htmlUrl="https://news.ycombinator.com"/>
    </body>
</opml>`

	outlines, err := Parse(strings.NewReader(xmlInput))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(outlines) != 3 {
		t.Errorf("Expected 3 outlines, got %d", len(outlines))
	}

	foundArs := false
	for _, o := range outlines {
		if o.XMLURL == "https://arstechnica.com/feed/" {
			foundArs = true
			break
		}
	}

	if !foundArs {
		t.Error("Did not find Ars Technica feed in parsed output")
	}
}

func TestParseInvalidOPML(t *testing.T) {
	_, err := Parse(strings.NewReader("not xml"))
	if err == nil {
		t.Error("Expected error parsing invalid XML, got nil")
	}
}

func TestGenerateOPML(t *testing.T) {
	input := []Outline{
		{Text: "Feed 1", XMLURL: "http://url1.com"},
		{Text: "Feed 2", XMLURL: "http://url2.com"},
	}

	data, err := Generate(input)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !bytes.Contains(data, []byte("http://url1.com")) {
		t.Error("Generated OPML missing URL 1")
	}
	if !bytes.Contains(data, []byte("version=\"1.0\"")) {
		t.Error("Generated OPML missing version attribute")
	}
}

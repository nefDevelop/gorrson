package opml

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseNestedCategories(t *testing.T) {
	input := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
	<body>
		<outline text="Level1">
			<outline text="Level2">
				<outline text="Feed" xmlUrl="https://example.com/rss" htmlUrl="https://example.com"/>
			</outline>
		</outline>
	</body>
</opml>`

	outlines, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse nested failed: %v", err)
	}
	if len(outlines) != 1 {
		t.Errorf("Expected 1 feed from nested structure, got %d", len(outlines))
	}
	if outlines[0].XMLURL != "https://example.com/rss" {
		t.Errorf("XMLURL = %s, want https://example.com/rss", outlines[0].XMLURL)
	}
}

func TestRoundTrip(t *testing.T) {
	original := []Outline{
		{Text: "Feed1", XMLURL: "http://feed1.com", HTMLURL: "http://feed1.com"},
		{Text: "Feed2", XMLURL: "http://feed2.com", HTMLURL: "http://feed2.com"},
	}

	data, err := Generate(original)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	parsed, err := Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed) != len(original) {
		t.Errorf("Round trip: expected %d outlines, got %d", len(original), len(parsed))
	}

	for i, o := range parsed {
		if o.XMLURL != original[i].XMLURL {
			t.Errorf("Outline %d: XMLURL = %s, want %s", i, o.XMLURL, original[i].XMLURL)
		}
	}
}

func TestParseMalformedXML(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"unclosed tag", `<opml><body><outline text="test"></opml>`},
		{"invalid xml", `not xml at all`},
		{"empty", ``},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tt.input))
			if err == nil {
				t.Error("Expected error for malformed XML")
			}
		})
	}
}

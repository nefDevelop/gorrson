package sanitizer

import (
	"strings"
	"testing"
)

func TestClean(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		omits    []string
	}{
		{
			"XSS script",
			`<p>Hello <script>alert('xss')</script></p>`,
			[]string{"<p>Hello </p>"},
			[]string{"<script>", "alert"},
		},
		{
			"XSS onclick",
			`<button onclick="evil()">Click me</button>`,
			[]string{"<button>Click me</button>"},
			[]string{"onclick", "evil"},
		},
		{
			"Allowed iframe",
			`<iframe src="https://www.youtube.com/embed/123" width="560"></iframe>`,
			[]string{"<iframe", "src=\"https://www.youtube.com/embed/123\"", "width=\"560\""},
			[]string{"javascript:"},
		},
		{
			"Allowed structural tags",
			`<article><figure><img src="a.jpg"><figcaption>Caption</figcaption></figure></article>`,
			[]string{"<article", "<figure", "<figcaption", "Caption"},
			[]string{},
		},
		{
			"Data attributes for lazy loading",
			`<img data-src="real.jpg" src="placeholder.jpg">`,
			[]string{"data-src=\"real.jpg\"", "src=\"placeholder.jpg\""},
			[]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Clean(tt.input)
			for _, c := range tt.contains {
				if !strings.Contains(got, c) {
					t.Errorf("Clean() output missing %q; got %q", c, got)
				}
			}
			for _, o := range tt.omits {
				if strings.Contains(got, o) {
					t.Errorf("Clean() output contains forbidden %q; got %q", o, got)
				}
			}
		})
	}
}

func TestStripHTML(t *testing.T) {
	input := `<div><h1>Title</h1><p>Paragraph with <a href="#">link</a>.</p></div>`
	expected := "TitleParagraph with link."
	got := StripHTML(input)
	if got != expected {
		t.Errorf("StripHTML() = %q; want %q", got, expected)
	}
}

func TestIsValidUsername(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"user123", true},
		{"user_name-99", true},
		{"User-Name", true},
		{"", false},
		{"user name", false},
		{"user@name", false},
		{"<script>", false},
		{strings.Repeat("a", 51), false},
	}

	for _, tt := range tests {
		if got := IsValidUsername(tt.input); got != tt.expected {
			t.Errorf("IsValidUsername(%q) = %v; want %v", tt.input, got, tt.expected)
		}
	}
}

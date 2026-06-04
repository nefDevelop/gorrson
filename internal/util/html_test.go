package util

import (
	"strings"
	"testing"
)

func TestExtractFirstImageFromHTML(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "Valid image",
			html:     `<p>Hello</p><img src="http://example.com/image.jpg" />`,
			expected: "http://example.com/image.jpg",
		},
		{
			name:     "No image",
			html:     `<div>No images here</div>`,
			expected: "",
		},
		{
			name:     "Malicious ReDoS Payload",
			html:     `<img src="` + strings.Repeat("a", 10000) + `"`,
			expected: "",
		},
		{
			name:     "Multiple images",
			html:     `<img src="first.png"><img src="second.jpg">`,
			expected: "first.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractFirstImageFromHTML(tt.html)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestCleanHTML(t *testing.T) {
	html := `<div>Hello<script>alert("XSS")</script> World</div>`
	cleaned := CleanHTML(html)
	if strings.Contains(cleaned, "script") {
		t.Errorf("Expected script to be removed, got: %s", cleaned)
	}
	if !strings.Contains(cleaned, "Hello") || !strings.Contains(cleaned, "World") {
		t.Errorf("Expected content to remain, got: %s", cleaned)
	}
}

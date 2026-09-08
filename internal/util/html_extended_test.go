package util

import (
	"strings"
	"testing"
)

func TestExtractImageTracker(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			"tracker pixel skips to real image",
			`<img src="https://analytics.com/pixel.gif"><img src="https://example.com/real.jpg">`,
			"https://example.com/real.jpg",
		},
		{
			"tracker in path filtered",
			`<img src="https://cdn.com/tracker/image.jpg">`,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractFirstImageFromHTML(tt.html)
			if result != tt.want {
				t.Errorf("got %q, want %q", result, tt.want)
			}
		})
	}
}

func TestCleanHTMLMultipleScripts(t *testing.T) {
	input := `<p>Before</p><script>var a=1;</script><p>Middle</p><script>alert('x')</script><p>After</p>`
	result := CleanHTML(input)

	if strings.Contains(result, "<script") {
		t.Errorf("Script tags should be removed: %s", result)
	}
	if !strings.Contains(result, "Before") || !strings.Contains(result, "Middle") || !strings.Contains(result, "After") {
		t.Errorf("Content should be preserved: %s", result)
	}
}

func TestCleanHTMLCaseInsensitive(t *testing.T) {
	input := `<SCRIPT>alert(1)</SCRIPT><Script>var x;</Script>`
	result := CleanHTML(input)
	if strings.Contains(result, "<script") || strings.Contains(result, "<SCRIPT") || strings.Contains(result, "<Script") {
		t.Errorf("Script tags should be removed regardless of case: %s", result)
	}
}

package sanitizer

import (
	"strings"
	"testing"
)

func TestCleanNestedScriptInAttribute(t *testing.T) {
	input := `<div onmouseover="alert('xss')">Hover me</div>`
	result := Clean(input)
	if strings.Contains(result, "onmouseover") {
		t.Errorf("Clean should remove onmouseover: %s", result)
	}
	if !strings.Contains(result, "Hover me") {
		t.Errorf("Clean should preserve text content: %s", result)
	}
}

func TestCleanFigureFigcaption(t *testing.T) {
	input := `<figure><img src="photo.jpg"><figcaption>A nice photo</figcaption></figure>`
	result := Clean(input)

	if !strings.Contains(result, "<figure") {
		t.Errorf("figure tag should be preserved: %s", result)
	}
	if !strings.Contains(result, "<figcaption") {
		t.Errorf("figcaption tag should be preserved: %s", result)
	}
}

func TestCleanArticleSectionHeaderFooter(t *testing.T) {
	input := `<article><header>Title</header><section>Body</section><footer>Copyright</footer></article>`
	result := Clean(input)

	for _, tag := range []string{"article", "header", "section", "footer"} {
		if !strings.Contains(result, "<"+tag) {
			t.Errorf("%s tag should be preserved: %s", tag, result)
		}
	}
}

func TestCleanXSSAdvanced(t *testing.T) {
	tests := []struct {
		name  string
		input string
		omits []string
	}{
		{"event handler", `<img src=x onerror=alert(1)>`, []string{"onerror"}},
		{"javascript URI", `<a href="javascript:alert(1)">click</a>`, []string{"javascript:"}},
		{"SVG onload", `<svg onload="alert(1)"><circle/></svg>`, []string{"onload"}},
		{"meta redirect", `<meta http-equiv="refresh" content="0;url=evil.com">`, []string{"meta"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Clean(tt.input)
			for _, o := range tt.omits {
				if strings.Contains(result, o) {
					t.Errorf("Clean() output contains forbidden %q: %s", o, result)
				}
			}
		})
	}
}

func TestIsValidUsernameEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"a", true},
		{"user.name", false},
		{"user name", false},
		{"user@name", false},
		{strings.Repeat("a", 50), true},
		{strings.Repeat("a", 51), false},
		{"", false},
		{" ", false},
	}

	for _, tt := range tests {
		if got := IsValidUsername(tt.input); got != tt.expected {
			t.Errorf("IsValidUsername(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

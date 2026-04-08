package sanitizer

import (
	"strings"
	"testing"
)

func TestSanitizer(t *testing.T) {
	input := `<div><script>alert('xss')</script><p>Hello <a href="javascript:alert(1)">Link</a></p><img src="x" onerror="alert(1)"></div>`
	output := Clean(input)

	if strings.Contains(output, "<script>") {
		t.Error("Sanitizer failed to remove <script>")
	}
	if strings.Contains(output, "onerror") {
		t.Error("Sanitizer failed to remove onerror attribute")
	}
	if strings.Contains(output, "javascript:") {
		t.Error("Sanitizer failed to remove javascript: link")
	}
	if !strings.Contains(output, "<p>Hello ") {
		t.Error("Sanitizer removed safe <p> tag")
	}
}

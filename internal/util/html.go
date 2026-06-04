package util

import (
	"regexp"
	"strings"
)

// ExtractFirstImageFromHTML safely extracts the first image URL from an HTML payload.
// Uses Go's regexp which guarantees O(n) execution, mitigating ReDoS attacks.
func ExtractFirstImageFromHTML(html string) string {
	if html == "" {
		return ""
	}
	re := regexp.MustCompile(`(?i)<img[^>]+src=["']([^"']+\.(jpg|jpeg|png|webp|gif|svg)[^"']*)["']`)
	matches := re.FindAllStringSubmatch(html, -1)
	for _, match := range matches {
		if len(match) > 1 {
			src := strings.ToLower(match[1])
			if strings.Contains(src, "pixel") || strings.Contains(src, "spacer") || strings.Contains(src, "tracker") {
				continue
			}
			return match[1]
		}
	}
	return ""
}

// CleanHTML removes script tags to prevent basic XSS when showing raw excerpts.
// Note: Frontend should still sanitize.
func CleanHTML(html string) string {
	// Remover <script> tags completos
	re := regexp.MustCompile(`(?is)<script.*?>.*?</script>`)
	return re.ReplaceAllString(html, "")
}

package sanitizer

import (
	"github.com/microcosm-cc/bluemonday"
)

// Clean cleans HTML content to be safe for rendering.
func Clean(content string) string {
	p := bluemonday.UGCPolicy()

	// Allow structural tags for better article layout
	p.AllowElements("figure", "figcaption", "article", "section", "header", "footer")

	// Allow iframes from trusted video providers
	p.AllowElements("iframe")
	p.AllowAttrs("src", "width", "height", "frameborder", "allow", "allowfullscreen", "scrolling").OnElements("iframe")
	p.AllowURLSchemes("https")
	p.AllowIFramesTargeting("https://www.youtube.com/", "https://www.youtube-nocookie.com/", "https://player.vimeo.com/", "https://www.dailymotion.com/")

	// Allow data attributes often used for lazy loading or metadata (if we want to process them later)
	p.AllowAttrs("data-src", "data-srcset", "srcset", "sizes").OnElements("img")

	return p.Sanitize(content)
}

// StripHTML removes all HTML tags from a string.
func StripHTML(content string) string {
	p := bluemonday.StrictPolicy()
	return p.Sanitize(content)
}

// IsValidUsername checks if a username is safe (alphanumeric, underscores, hyphens).
func IsValidUsername(username string) bool {
	if username == "" || len(username) > 50 {
		return false
	}
	for _, char := range username {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '_' && char != '-' {
			return false
		}
	}
	return true
}

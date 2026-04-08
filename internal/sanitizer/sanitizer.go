package sanitizer

import (
	"github.com/microcosm-cc/bluemonday"
)

// Clean cleans HTML content to be safe for rendering.
func Clean(content string) string {
	p := bluemonday.UGCPolicy()
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

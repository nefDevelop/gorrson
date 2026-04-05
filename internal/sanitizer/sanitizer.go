package sanitizer

import (
	"github.com/microcosm-cc/bluemonday"
)

// Clean cleans HTML content to be safe for rendering.
func Clean(content string) string {
	p := bluemonday.UGCPolicy()
	return p.Sanitize(content)
}

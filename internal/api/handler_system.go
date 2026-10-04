package api

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	stdhtml "html"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nef734/gorrson/internal/auth"
	"github.com/nef734/gorrson/internal/db"
	"github.com/nef734/gorrson/internal/models"
	"github.com/nef734/gorrson/internal/rss"
	"github.com/nef734/gorrson/internal/sanitizer"
	"golang.org/x/net/html"
)

func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "docs/openapi.yaml")
}

func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Gorrson API Documentation</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js" charset="UTF-8"></script>
    <script>
      window.onload = () => {
        window.ui = SwaggerUIBundle({
          url: '/openapi.yaml',
          dom_id: '#swagger-ui',
          deepLinking: true,
          presets: [SwaggerUIBundle.presets.apis],
        });
      };
    </script>
  </body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (s *Server) handleFeverAPI(w http.ResponseWriter, r *http.Request) {
	// Límite de cuerpo: /api/fever era la única ruta sin MaxBytesReader.
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	// 1. Get API Key
	apiKey := r.FormValue("api_key")
	if apiKey == "" {
		// Try from POST body if not in query
		r.ParseForm()
		apiKey = r.Form.Get("api_key")
	}

	// Fever no envía el usuario: viaja hasheado dentro del api_key (md5 de
	// usuario:contraseña). Para saber a qué base de datos pertenece hay que
	// verificar la clave contra cada usuario, y cada verificación es un argon2id
	// de ~62 ms y ~67 MB. Con 20 usuarios, una sola petición No autenticada cuesta
	// 1,3 GB de RAM. Por eso esta ruta tiene su propio límite, más ajustado que
	// el de /login.
	if !s.feverLimiter.permitir(clientIP(r), s.feverLimiter.porIP) {
		slog.Warn("Fever API: rate limited", "ip", clientIP(r))
		w.Header().Set("Content-Type", "application/json")
		// auth:0 y no un error HTTP: es lo que un cliente Fever espera cuando la
		// credencial no vale. El 429 sí lo respetan, y evita el bucle de reintentos.
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"api_version": 3, "auth": 0})
		return
	}

	// 2. Buscar la base de datos del usuario. OpenExistingStore en vez de
	// NewStore: no crea ficheros inexistentes ni ejecuta las 26 sentencias de
	// migración por usuario y por petición.
	files, _ := os.ReadDir(s.dataDir)
	var userStore *db.Store

	for _, f := range files {
		if filepath.Ext(f.Name()) != ".db" || f.Name() == "global.db" {
			continue
		}
		username := strings.TrimSuffix(f.Name(), ".db")
		store, err := db.OpenExistingStore(s.dataDir, username)
		if err != nil {
			continue
		}
		feverKey, err := store.GetUserFeverKey(r.Context(), username)
		if err != nil || feverKey == "" {
			store.Close()
			continue
		}

		// Validate MD5 apiKey against the stored argon2 hash of it.
		if auth.CheckPasswordHash(apiKey, feverKey) {
			userStore = store
			break
		}
		store.Close()
	}

	if userStore == nil {
		// La clave NO se registra: es una credencial equivalente a la contraseña
		// (viaja en claro sobre HTTP) y antes acababa en gorrson.log.
		slog.Warn("Fever API: unauthorized access attempt", "ip", clientIP(r))
		// Return auth:0 instead of HTTP error to be Fever-compliant
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"api_version": 3,
			"auth":        0,
		})
		return
	}
	defer userStore.Close()

	// Fever Response Base
	resp := map[string]interface{}{
		"api_version":            3,
		"auth":                   1,
		"last_refreshed_on_time": time.Now().Unix(),
	}

	// 3. Handle Commands
	if r.URL.Query().Has("groups") {
		feeds, _ := userStore.ListFeeds(r.Context())
		categories := make(map[string]bool)
		var groups []map[string]interface{}
		for _, f := range feeds {
			if f.Category != "" && !categories[f.Category] {
				categories[f.Category] = true
				groups = append(groups, map[string]interface{}{
					"id":    uuidToInt(f.Category),
					"title": f.Category,
				})
			}
		}
		resp["groups"] = groups

		var feedsGroups []map[string]interface{}
		for cat := range categories {
			var feedIds []string
			for _, f := range feeds {
				if f.Category == cat {
					feedIds = append(feedIds, strconv.Itoa(f.FeverID))
				}
			}
			feedsGroups = append(feedsGroups, map[string]interface{}{
				"group_id": uuidToInt(cat),
				"feed_ids": strings.Join(feedIds, ","),
			})
		}
		resp["feeds_groups"] = feedsGroups
	}

	if r.URL.Query().Has("feeds") {
		feeds, _ := userStore.ListFeeds(r.Context())
		var feverFeeds []map[string]interface{}
		for _, f := range feeds {
			feverFeeds = append(feverFeeds, map[string]interface{}{
				"id":                   f.FeverID,
				"favicon_id":           0,
				"title":                f.Title,
				"url":                  f.FeedURL,
				"site_url":             f.SiteURL,
				"is_spark":             0,
				"last_updated_on_time": f.LastUpdated.Unix(),
			})
		}
		resp["feeds"] = feverFeeds
	}

	if r.URL.Query().Has("items") || r.URL.Query().Has("unread_item_ids") {
		if r.URL.Query().Has("unread_item_ids") {
			// Query propia y sin limite: la Fever API espera la lista completa
			// de ids sin leer, y truncar aqui hacia que el cliente perdiera
			// el recuento. Antes se reutilizaba ListEntries con tope 500.
			ids, _ := userStore.UnreadFeverIDs(r.Context())
			parts := make([]string, 0, len(ids))
			for _, id := range ids {
				parts = append(parts, strconv.Itoa(id))
			}
			resp["unread_item_ids"] = strings.Join(parts, ",")
		}

		if r.URL.Query().Has("items") {
			// Pagina de items, ordenada de mas reciente a mas antigua. Aqui si
			// hay tope, a proposito: son entradas completas con su HTML, y
			// volcarlas todas de golpe es lo que hace heavyweight a un
			// servidor. feverItemPageItems deja margen de sobra para la
			// ventana de sincronizacion de cualquier cliente.
			//
			// NO se implementan since_id, max_id ni with_ids, aunque el
			// protocolo los define. Requieren que los ids crezcan de forma
			// monotona y aqui no lo hacen: generateFeverID (internal/rss)
			// deriva el id con un hash FNV del UUID, asi que es un numero
			// disperso. Un cliente que pidiera desde su id mas alto se
			// saltaria para siempre las entradas cuyo hash quedara por debajo,
			// y perderia articulos en silencio. Truncar la pagina es visible;
			// perder articulos no.
			entries, _ := userStore.ListEntries(r.Context(), false, "desc", feverItemPageItems, 0, "", "")
			var items []map[string]interface{}
			for _, e := range entries {
				items = append(items, map[string]interface{}{
					"id":              e.FeverID,
					"feed_id":         uuidToInt(e.FeedID), // Using uuidToInt for consistency if FeedID is UUID
					"title":           e.Title,
					"author":          "",
					"html":            e.Content,
					"url":             e.URL,
					"is_saved":        boolToInt(e.IsFavorite),
					"is_read":         boolToInt(e.IsRead),
					"created_on_time": e.PublishedAt.Unix(),
				})
			}
			resp["items"] = items
		}
	}

	// Fever Mark Commands
	if r.FormValue("mark") == "item" {
		id, _ := strconv.Atoi(r.FormValue("id"))
		switch r.FormValue("as") {
		case "read":
			_ = userStore.MarkAsReadByFeverID(r.Context(), id)
		case "saved":
			_ = userStore.ToggleFavoriteByFeverID(r.Context(), id, true)
		case "unsaved":
			_ = userStore.ToggleFavoriteByFeverID(r.Context(), id, false)
		}
	} else if r.FormValue("mark") == "feed" {
		id, _ := strconv.Atoi(r.FormValue("id"))
		if r.FormValue("as") == "read" {
			// Find category or feed by FeverID and mark all read
			feeds, _ := userStore.ListFeeds(r.Context())
			for _, f := range feeds {
				if f.FeverID == id {
					_ = userStore.MarkAllRead(r.Context(), "", f.ID, "", 0)
					break
				}
			}
		}
	} else if r.FormValue("mark") == "group" {
		id, _ := strconv.Atoi(r.FormValue("id"))
		if r.FormValue("as") == "read" {
			// Find category by FeverID
			feeds, _ := userStore.ListFeeds(r.Context())
			for _, f := range feeds {
				if uuidToInt(f.Category) == id {
					_ = userStore.MarkAllRead(r.Context(), f.Category, "", "", 0)
					break
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleImageProxy(w http.ResponseWriter, r *http.Request) {
	imageURL := r.URL.Query().Get("url")
	if imageURL == "" {
		http.Error(w, "Missing url parameter", http.StatusBadRequest)
		return
	}

	// 1. Setup cache path
	hash := sha256.Sum256([]byte(imageURL))
	hexHash := hex.EncodeToString(hash[:])
	cacheDir := filepath.Join(s.dataDir, "cache", "images")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		http.Error(w, "Could not create cache directory", http.StatusInternalServerError)
		return
	}
	cachePath := filepath.Join(cacheDir, hexHash)

	// 2. Check if cached
	if _, err := os.Stat(cachePath); err == nil {
		s.serveCachedImage(w, r, cachePath)
		return
	}

	// 3. Not cached, download it using SecureClient
	sc := rss.NewSecureClient("")
	body, respHeader, err := sc.FetchResponse(r.Context(), imageURL)
	if err != nil {
		// El detalle va al log, no al cliente: revela la topología interna y la
		// resolución DNS (oráculo de SSRF).
		slog.Error("Proxy de imágenes: fetch fallido", "url", imageURL, "error", err)
		http.Error(w, "Failed to fetch image", http.StatusBadGateway)
		return
	}
	defer body.Close()

	// 4. Save to temp file and then move to cache to avoid partial writes
	tmpPath := cachePath + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		http.Error(w, "Could not create temporary file", http.StatusInternalServerError)
		return
	}

	// Copy content
	if _, err := io.Copy(tmpFile, body); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		http.Error(w, "Failed to save image", http.StatusInternalServerError)
		return
	}
	tmpFile.Close()

	// Antes de confirmar en la caché hay que decidir el Content-Type y comprobar que
	// lo descargado es una imagen. Si no, envenenamos la caché con HTML del atacante
	// y lo servimos después desde el origen de la app.
	//
	// La cabecera del origen solo se acepta si es image/*; ver resolveImageType.
	declarado := respHeader.Get("Content-Type")
	ct, ok := resolveImageType(declarado, tmpPath)
	if !ok {
		os.Remove(tmpPath)
		slog.Warn("Proxy de imágenes: la respuesta no es una imagen, descartada",
			"url", imageURL, "declarado", declarado)
		http.Error(w, "Remote resource is not an image", http.StatusUnsupportedMediaType)
		return
	}

	if err := os.Rename(tmpPath, cachePath); err != nil {
		http.Error(w, "Could not commit cache file", http.StatusInternalServerError)
		return
	}
	writeTypeFile(cachePath, ct)

	// 5. Serve it
	s.serveCachedImage(w, r, cachePath)
}

// serveCachedImage sirve una imagen desde la caché.
//
// El Content-Type se fija EXPLÍCITAMENTE y se añade X-Content-Type-Options:
// nosniff. Antes se usaba ServeContent con el nombre "image" (sin extensión), que
// hace DetectContentType sobre los primeros 512 bytes: un atacante que apuntara a
// una URL con HTML obtenía text/html desde el origen de la app, y como la CSP
// global permite 'unsafe-inline' el <script> se ejecutaba. Reproducido.
//
// nosniff es lo que neutraliza de verdad un cuerpo HTML: si se sirve como
// image/png con nosniff, el navegador no lo interpreta como documento. Por eso el
// Content-Type se decide aquí y no se deja que el navegador lo detecte.
func (s *Server) serveCachedImage(w http.ResponseWriter, r *http.Request, path string) {
	ct, ok := resolveImageType(readTypeFile(path), path)
	if !ok {
		// Entrada envenenada de una versión anterior (o de un descuido): se borra
		// y se trata como fallo, sin devolver el contenido.
		os.Remove(path)
		os.Remove(typeFilePath(path))
		slog.Warn("Proxy de imágenes: entrada de caché que no es una imagen, eliminada", "path", path)
		http.Error(w, "Cached resource is not an image", http.StatusUnsupportedMediaType)
		return
	}

	f, err := os.Open(path)
	if err != nil {
		http.Error(w, "Could not open cached image", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", ct)
	// Si a pesar de todo se sirviera algo que el navegador pudiera interpretar
	// como documento, esta CSP lo neutraliza: sin scripts, sin estilos, sin
	// iframes. No afecta a <img src>, que solo necesita que se pueda pintar.
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year
	// ServeContent con size evita recalcularlo y respeta el Content-Type que
	// acabamos de fijar.
	http.ServeContent(w, r, "imagen", time.Time{}, f)
}

// resolveImageType decide el Content-Type con el que se sirve un fichero.
//
// Se prefiere lo que dicen los BYTES (sniff), y si no se reconoce se recurre a lo
// que declaraba el origen. Existe ese segundo camino porque
// http.DetectContentType no reconoce SVG ni AVIF, que son miniaturas perfectly
// normales en un lector de RSS: con el sniff alone se rechazarían.
//
// Lo que NO se hace es confiar en la declaración sin más: se exige que sea
// image/*, y en cualquier caso la respuesta lleva Content-Type explícito +
// nosniff, de modo que un cuerpo HTML servido como image/png es inerte para el
// navegador.
func resolveImageType(declarado, path string) (string, bool) {
	if ct, ok := sniffImage(path); ok {
		return ct, true
	}
	declarado = strings.ToLower(strings.TrimSpace(declarado))
	if strings.HasPrefix(declarado, "image/") {
		return declarado, true
	}
	return declarado, false
}

// sniffImage lee los primeros bytes de un fichero y devuelve su Content-Type si
// DetectContentType los reconoce como imagen.
func sniffImage(path string) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return "", false
	}
	ct := http.DetectContentType(buf[:n])
	if !strings.HasPrefix(ct, "image/") {
		return ct, false
	}
	return ct, true
}

// La caché guarda el Content-Type resuelto en un fichero aparte, porque
// http.DetectContentType no reconoce SVG ni AVIF y en la segunda visita ya no se
// dispone de la cabecera original del origen.
func typeFilePath(cachePath string) string { return cachePath + ".type" }

func writeTypeFile(cachePath, ct string) {
	_ = os.WriteFile(typeFilePath(cachePath), []byte(ct), 0o600)
}

func readTypeFile(cachePath string) string {
	blob, err := os.ReadFile(typeFilePath(cachePath))
	if err != nil {
		return ""
	}
	return string(blob)
}

func (s *Server) handlePageProxy(w http.ResponseWriter, r *http.Request) {
	proxySemaphore <- struct{}{}
	defer func() { <-proxySemaphore }()

	targetURL := r.URL.Query().Get("url")
	feedID := r.URL.Query().Get("feed_id")
	username := r.Context().Value(usernameKey).(string)
	slog.Info("Page proxy request START", "url", targetURL, "feed_id", feedID, "user", username)

	// Ensure this specific endpoint is allowed to be framed by our own app
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if targetURL == "" {
		slog.Warn("Page proxy request: MISSING URL")
		http.Error(w, "Missing url parameter", http.StatusBadRequest)
		return
	}

	var customUA string
	if feedID != "" {
		store, err := db.NewStore(s.dataDir, username)
		if err == nil {
			defer store.Close()
			feed, err := store.GetFeedByID(r.Context(), feedID)
			if err == nil && feed.UserAgent != "" {
				customUA = feed.UserAgent
			}
		}
	}
	sc := rss.NewSecureClient(customUA)
	body, err := sc.Fetch(r.Context(), targetURL)
	if err != nil {
		slog.Error("Failed to fetch proxy page", "url", targetURL, "error", err)
		// El mensaje de error de url/url.Parse incluye la URL CRUDA, sin escapar. Con
		// la respuesta en text/html, eso es XSS reflejada: basta con
		// /proxy/page?url=http://a/%zz%22%3E%3Cimg%20src%3Dx%20onerror%3Dalert(1)%3E,
		// que falla al construir la petición, antes de tocar la red.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "<body style='font-family:sans-serif;padding:40px;text-align:center;background:#fff5f5;color:#c53030'>")
		fmt.Fprintf(w, "<h2>Proxy Error</h2>")
		fmt.Fprintf(w, "<p>Could not load the page. Check your internet connection or if the site blocks Gorrson.</p>")
		fmt.Fprintf(w, "<pre style='background:#fff;padding:10px;border-radius:8px;display:inline-block'>%s</pre>",
			stdhtml.EscapeString(err.Error()))
		fmt.Fprintf(w, "</body>")
		return
	}
	defer body.Close()

	// Parse HTML
	doc, err := html.Parse(body)
	if err != nil {
		http.Error(w, "Failed to parse HTML", http.StatusInternalServerError)
		return
	}

	// Rewrite URLs (href, src) to be absolute and disable scripts
	u, _ := url.Parse(targetURL)
	var rewrite func(*html.Node)
	rewrite = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Los scripts del documento se marcan como texto plano. OJO: hay que
			// SUSTITUIR el atributo type, no añadir otro. Con append(), un
			// <script type="module"> de la página queda con dos atributos type y
			// el navegador gana el primero, que es el original: el script se
			// ejecuta. Con setTypeAtLeast queda siempre un único type=text/plain.
			if n.Data == "script" {
				for i, attr := range n.Attr {
					if attr.Key == "src" {
						n.Attr[i].Val = "#disabled-by-gorrson"
					}
				}
				setSingleType(n, "text/plain")
			}

			// Disable iframes to avoid frame busting
			if n.Data == "iframe" {
				for i, attr := range n.Attr {
					if attr.Key == "src" {
						n.Attr[i].Val = "about:blank"
					}
				}
			}

			for i, attr := range n.Attr {
				if attr.Key == "href" || attr.Key == "src" || attr.Key == "srcset" {
					if attr.Key == "srcset" {
						// Simple srcset rewrite (can be improved further)
						parts := strings.Split(attr.Val, ",")
						for j, p := range parts {
							p = strings.TrimSpace(p)
							bits := strings.SplitN(p, " ", 2)
							if len(bits) > 0 {
								bits[0] = resolveURL(u, bits[0])
								parts[j] = strings.Join(bits, " ")
							}
						}
						n.Attr[i].Val = strings.Join(parts, ", ")
					} else {
						absURL := resolveURL(u, attr.Val)
						n.Attr[i].Val = absURL
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			rewrite(c)
		}
	}
	rewrite(doc)

	// El documento remoto pasa por la política estricta ANTES de que inyectemos
	// base, viewport y nuestro script de selección.
	//
	// El orden importa: al revés, el saneador se llevaría por delante el script
	// propio y el modo lector dejaría de funcionar. Con este orden, todo lo que
	// llega del tercero (scripts, onerror, onload, iframe, href="javascript:")
	// desaparece, y lo único que sobrevive es lo que gorrson añade después.
	doc, err = sanitizeRemoteDoc(doc)
	if err != nil {
		slog.Error("Failed to sanitize proxied document", "url", targetURL, "error", err)
		http.Error(w, "Render error", http.StatusInternalServerError)
		return
	}

	// Inject selection script
	// Add <base> and <viewport> tags to head
	baseNode := &html.Node{
		Type: html.ElementNode,
		Data: "base",
		Attr: []html.Attribute{{Key: "href", Val: targetURL}},
	}
	injectAtStart(doc, "head", baseNode)

	viewportNode := &html.Node{
		Type: html.ElementNode,
		Data: "meta",
		Attr: []html.Attribute{
			{Key: "name", Val: "viewport"},
			{Key: "content", Val: "width=device-width, initial-scale=1"},
		},
	}
	injectAtStart(doc, "head", viewportNode)

	// Inject selection script to body
	scriptNode := &html.Node{
		Type: html.ElementNode,
		Data: "script",
	}
	scriptNode.AppendChild(&html.Node{
		Type: html.TextNode,
		Data: `
(function() {
    let style = document.createElement('style');
    style.innerHTML = ` + "`" + `
        .gorrson-highlight { outline: 3px solid #ff5252 !important; cursor: pointer !important; opacity: 0.8 !important; }
        .gorrson-selected-body { outline: 3px solid #4caf50 !important; background: rgba(76, 175, 80, 0.1) !important; }
        .gorrson-selected-strip { outline: 3px solid #f44336 !important; background: rgba(244, 67, 54, 0.1) !important; opacity: 0.4 !important; }
        .gorrson-selected-title { outline: 3px solid #2196f3 !important; background: rgba(33, 150, 243, 0.1) !important; }
        * { visibility: visible !important; opacity: 1 !important; max-width: 100% !important; box-sizing: border-box !important; }
        body { display: block !important; background: white !important; color: black !important; overflow-x: hidden !important; width: 100% !important; }
    ` + "`" + `;
    document.head.appendChild(style);

    document.addEventListener('mouseover', e => {
        if (e.target.tagName === 'BODY' || e.target.tagName === 'HTML') return;
        e.target.classList.add('gorrson-highlight');
    }, true);
    document.addEventListener('mouseout', e => {
        e.target.classList.remove('gorrson-highlight');
    }, true);
    document.addEventListener('click', e => {
        e.preventDefault();
        e.stopPropagation();
        let xpath = getXPath(e.target);
        window.parent.postMessage({ type: 'gorrson-select', xpath: xpath }, '*');
    }, true);

    window.addEventListener('message', e => {
        if (e.data.type === 'gorrson-sync') {
            document.querySelectorAll('.gorrson-selected-body, .gorrson-selected-strip, .gorrson-selected-title')
                    .forEach(el => el.classList.remove('gorrson-selected-body', 'gorrson-selected-strip', 'gorrson-selected-title'));
            
            if (e.data.body) e.data.body.forEach(xpath => highlightByXPath(xpath, 'gorrson-selected-body'));
            if (e.data.strip) e.data.strip.forEach(xpath => highlightByXPath(xpath, 'gorrson-selected-strip'));
            if (e.data.title) highlightByXPath(e.data.title, 'gorrson-selected-title');
        }
    });

    function highlightByXPath(xpath, className) {
        try {
            let result = document.evaluate(xpath, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null);
            let node = result.singleNodeValue;
            if (node && node.classList) node.classList.add(className);
        } catch(e) {}
    }

    function getXPath(node) {
        if (node.nodeType !== 1) return '';
        if (node.id) return "//" + node.tagName.toLowerCase() + "[@id='" + node.id + "']";
        if (node === document.body) return '//body';
        
        let tag = node.tagName.toLowerCase();
        let ix = 1;
        for (let sib = node.previousSibling; sib; sib = sib.previousSibling) {
            if (sib.nodeType === 1 && sib.tagName === node.tagName) ix++;
        }

        if (node.parentNode && node.parentNode.nodeType === 1 && node.parentNode.tagName !== 'HTML') {
            return getXPath(node.parentNode) + '/' + tag + '[' + ix + ']';
        }
        return '//' + tag + '[' + ix + ']';
    }
})();
`,
	})
	injectAtStart(doc, "body", scriptNode)

	w.Header().Set("Content-Type", "text/html")
	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		slog.Error("Failed to render proxy doc", "error", err)
		http.Error(w, "Render error", http.StatusInternalServerError)
		return
	}

	slog.Info("Proxy rendered doc complete", "url", targetURL, "size", buf.Len())
	w.Write(buf.Bytes())
}

// setSingleType deja el elemento con un ÚNICO atributo type=val.
//
// Añadirlo con append() en vez de sustituirlo deja el elemento con dos atributos
// type, y según el HTML el navegador gana el primero: un <script
// type="module"> de la página se seguía ejecutando.
func setSingleType(n *html.Node, val string) {
	for i := range n.Attr {
		if n.Attr[i].Key == "type" {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: "type", Val: val})
}

// sanitizeRemoteDoc pasa el documento ya reescrito por la política estricta de
// sanitizer.CleanForProxy y lo vuelve a parsear.
//
// Se serializa y reparsea (en vez de recorrer el árbol a mano) para apoyarse en
// bluemonday, que ya está en el proyecto y es lo que protege el resto del pipeline
// de ingesta. El coste es un render + un parse más, sobre documentos que ya se
// descargan por red.
func sanitizeRemoteDoc(doc *html.Node) (*html.Node, error) {
	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return nil, err
	}
	return html.Parse(strings.NewReader(sanitizer.CleanForProxy(buf.String())))
}

func (s *Server) handleSaveRecipe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FeedID   string   `json:"feed_id"`
		BodyList []string `json:"body_list"`
		Title    string   `json:"title"`
		Strip    []string `json:"strip"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	username := r.Context().Value(usernameKey).(string)
	store, err := s.getStore(username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	feed, err := store.GetFeedByID(r.Context(), req.FeedID)
	if err != nil {
		http.Error(w, "Feed not found", http.StatusNotFound)
		return
	}

	target := feed.SiteURL
	if target == "" {
		target = feed.FeedURL
	}
	u, _ := url.Parse(target)
	domain := strings.TrimPrefix(u.Hostname(), "www.")

	customDir := filepath.Join(s.dataDir, "site-config", "custom")
	os.MkdirAll(customDir, 0755)

	path := filepath.Join(customDir, domain+".txt")
	content := fmt.Sprintf("# Custom Recipe for %s\n", domain)
	if req.Title != "" {
		content += fmt.Sprintf("title: %s\n", req.Title)
	}
	for _, b := range req.BodyList {
		content += fmt.Sprintf("body: %s\n", b)
	}
	for _, s := range req.Strip {
		content += fmt.Sprintf("strip: %s\n", s)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUpdateRecipes(w http.ResponseWriter, r *http.Request) {
	go rss.UpdateRecipes() // Run in background
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Recipe update process started in the background."})
}

func (s *Server) handleExportConfig(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(usernameKey).(string)
	store, err := s.getStore(username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	filters, _ := store.ListFilters(r.Context())
	// Evitar que un slice vacío se serialice como `null` en JSON en lugar de `[]`
	if filters == nil {
		filters = make([]models.Filter, 0)
	}

	slog.Info("Config and filters exported", "user", username, "filters_count", len(filters))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=gorrson-config.json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"version": "1.0",
		"filters": filters,
	})
}

func (s *Server) handleImportConfig(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(usernameKey).(string)
	var config struct {
		Filters []models.Filter `json:"filters"`
	}
	if err := decodeJSONBody(w, r, &config); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	store, err := s.getStore(username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	for _, f := range config.Filters {
		if f.Pattern == "" {
			continue
		}
		if f.ID == "" {
			f.ID = uuid.New().String()
		}
		store.AddFilter(r.Context(), &f)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Configuration imported successfully"})
}

func (s *Server) handleListFilters(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(usernameKey).(string)
	store, err := s.getStore(username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	filters, err := store.ListFilters(r.Context())
	if err != nil {
		http.Error(w, "Query failure", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filters)
}

func (s *Server) handleAddFilter(w http.ResponseWriter, r *http.Request) {
	var f models.Filter
	if err := decodeJSONBody(w, r, &f); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if _, err := regexp.Compile(f.Pattern); err != nil {
		http.Error(w, "Invalid regex pattern: "+err.Error(), http.StatusBadRequest)
		return
	}

	f.ID = uuid.New().String()
	username := r.Context().Value(usernameKey).(string)
	store, err := s.getStore(username)
	if err == nil {
		store.AddFilter(r.Context(), &f)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(f)
}

func (s *Server) handleDeleteFilter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	username := r.Context().Value(usernameKey).(string)
	store, err := s.getStore(username)
	if err == nil {
		store.DeleteFilter(r.Context(), id)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	username, ok := r.Context().Value(usernameKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	store, err := s.getStore(username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	keys, err := store.ListAPIKeys(r.Context())
	if err != nil {
		http.Error(w, "Query failure", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}

func (s *Server) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil || req.Name == "" {
		http.Error(w, "Bad Request: name is required", http.StatusBadRequest)
		return
	}

	username, ok := r.Context().Value(usernameKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 1. Generate random key
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	rawKey := hex.EncodeToString(b)
	fullKey := fmt.Sprintf("%s:%s", username, rawKey)

	// 2. Hash it
	hash := sha256.Sum256([]byte(rawKey))
	hashHex := hex.EncodeToString(hash[:])

	// 3. Save to DB
	id := uuid.New().String()
	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	if err := store.AddAPIKey(r.Context(), id, req.Name, hashHex); err != nil {
		http.Error(w, "Failed to save key", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":      id,
		"name":    req.Name,
		"api_key": fullKey,
	})
}

func (s *Server) handleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	username, ok := r.Context().Value(usernameKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	if err := store.DeleteAPIKey(r.Context(), id); err != nil {
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(usernameKey).(string)
	store, err := s.getStore(username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	stats, err := store.GetStats(r.Context())
	if err != nil {
		http.Error(w, "Failed to retrieve stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleExportBackup(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(usernameKey).(string)
	dbPath := filepath.Join(s.dataDir, username+".db")

	if _, err := os.Stat(dbPath); err != nil {
		http.Error(w, "Database file not found", http.StatusNotFound)
		return
	}

	slog.Info("Backup exported", "user", username, "ip", r.RemoteAddr)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=gorrson-%s-backup-%s.db", username, time.Now().Format("2006-01-02")))
	http.ServeFile(w, r, dbPath)
}

func (s *Server) handleImportBackup(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(usernameKey).(string)

	// Security: Limit size to 100MB to avoid disk filling DOS
	r.Body = http.MaxBytesReader(w, r.Body, 100<<20)

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing file in multipart form or file too large", http.StatusBadRequest)
		return
	}
	defer file.Close()

	dbPath := filepath.Join(s.dataDir, username+".db")
	tmpPath := dbPath + ".tmp"

	out, err := os.Create(tmpPath)
	if err != nil {
		http.Error(w, "Failed to create temp file", http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		os.Remove(tmpPath)
		http.Error(w, "Failed to save file. Check size limits.", http.StatusInternalServerError)
		return
	}
	out.Close()

	// Security: Verify the uploaded file is a valid SQLite DB before replacing
	if err := db.VerifySQLiteFile(r.Context(), tmpPath); err != nil {
		os.Remove(tmpPath)
		slog.Warn("Attempted to upload invalid backup file", "user", username, "err", err)
		http.Error(w, "Invalid backup file: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Atomic rename. On Linux this overwrites the existing file.
	if err := os.Rename(tmpPath, dbPath); err != nil {
		http.Error(w, "Failed to overwrite database. Is it being used?", http.StatusInternalServerError)
		return
	}

	// El store cacheado de este usuario sigue abierto sobre el INODE ANTIGUO: ya
	// no tiene nombre, pero el descriptor sigue siendo válido y SQLite sigue
	// escribiendo y leyendo ahí. Sin esto, después de restaurar un backup el
	// usuario sigue viendo sus datos antiguos y todo lo que escriba a partir de
	// ahí se pierde en un fichero sin nombre. Refrescar la página no arregla nada,
	// porque la caché está en el servidor, no en el cliente.
	s.evictStore(username)

	slog.Info("Backup restored successfully", "user", username, "ip", r.RemoteAddr)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Database restored and verified. The cached connection was dropped, " +
			"so the next request already uses the restored data.",
	})
}

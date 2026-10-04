package api

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"hash/fnv"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nef734/gorrson/internal/auth"
	"github.com/nef734/gorrson/internal/db"
	"github.com/nef734/gorrson/internal/rss"
	"golang.org/x/net/html"
)

var (
	//go:embed all:web/dist
	frontendFS embed.FS
)

type contextKey string

const usernameKey contextKey = "username"

type Server struct {
	router  *chi.Mux
	dataDir string
	parser  *rss.Parser
	silent  bool
	stores  map[string]*db.Store
	mu      sync.RWMutex

	// loginLimiter acota los intentos de autenticación: un argon2id son ~62 ms y
	// ~67 MB de RAM, así que 100 logins concurrentes desde una IP son 6,7 GB.
	// /login es público, así que es un amplificador de memoria sin autenticar.
	loginLimiter *loginLimiter

	// feverLimiter acota /api/fever, que es más cara que /login: no recibe el
	// usuario, así que tiene que verificar la clave contra TODAS las bases de
	// datos del directorio, con un argon2id (~62 ms, ~67 MB) por usuario.
	feverLimiter *loginLimiter
}

// feverItemPageItems es cuantas entradas devuelve la Fever API en una pagina
// de ?items. El protocolo propone 50 e invita a paginar con since_id, pero
// gorrson no puede usar since_id (los ids no son monotonicos, ver
// handleFeverAPI), asi que se devuelve una ventana amplia de una vez: 500
// entradas cubren de sobra la ventana de sincronizacion de cualquier cliente.
const feverItemPageItems = 500

// boolToInt converts a bool to 0 or 1 for Fever API compatibility.
// Fever espera numeros, no booleanos JSON: is_read e is_saved tienen que
// viajar como 0/1 o el cliente no los interpreta.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func NewServer(dataDir string, silent bool) *Server {
	s := &Server{
		router:  chi.NewRouter(),
		dataDir: dataDir,
		parser:  rss.NewParser(),
		silent:  silent,
		stores:  make(map[string]*db.Store),
		// 10 intentos por ventana de 15 min, contados por IP y por usuario.
		// Un usuario legítimo que se equivoca 10 veces seguidas tiene que esperar,
		// lo cual es lo que hacen los limitadores habituales; a cambio un
		// atacante no puede gastar memoria del servidor a voluntad.
		loginLimiter: newLoginLimiter(10, 15*time.Minute),
		// Más ajustado: cada petición de /api/fever puede costar N × 67 MB.
		// 20 intentos por hora son 400 argon2 en el peor caso.
		feverLimiter: newLoginLimiter(20, time.Hour),
	}
	s.routes()

	// Start background cache cleaner (Run every 24h, keep files for 15 days)
	go s.cleanupImageCache(15 * 24 * time.Hour)

	return s
}

func (s *Server) getStore(username string) (*db.Store, error) {
	s.mu.RLock()
	store, ok := s.stores[username]
	s.mu.RUnlock()
	if ok {
		return store, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check again in case another goroutine created it
	if store, ok := s.stores[username]; ok {
		return store, nil
	}

	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		return nil, err
	}
	s.stores[username] = store
	return store, nil
}

// evictStore cierra y saca de la caché el store de un usuario.
//
// Necesario cuando el fichero de la base de datos se sustituye por fuera
// (importar un backup): el *sql.DB cacheado sigue abierto sobre el inode antiguo,
// que ya no tiene nombre pero sigue siendo válido, así que las lecturas
// devuelven los datos viejos y las escrituras se pierden. La próxima llamada a
// getStore lo vuelve a abrir contra el fichero nuevo.
func (s *Server) evictStore(username string) {
	s.mu.Lock()
	store, ok := s.stores[username]
	if ok {
		delete(s.stores, username)
	}
	s.mu.Unlock()

	if ok {
		_ = store.Close()
		slog.Info("Store cache evicted", "user", username)
	}
}

func (s *Server) routes() {
	if !s.silent {
		s.router.Use(middleware.Logger)
	}
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Compress(5))
	s.router.Use(s.securityHeaders)

	s.router.Get("/openapi.yaml", s.handleOpenAPI)
	s.router.Get("/docs", s.handleDocs)
	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	s.router.Post("/login", s.handleLogin)
	s.router.Post("/refresh", s.handleRefresh)
	s.router.Post("/logout", s.handleLogout)

	// Fever API endpoint (authentication is handled inside the handler via api_key)
	s.router.Post("/api/fever", s.handleFeverAPI)
	s.router.Get("/api/fever", s.handleFeverAPI)

	s.router.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)

		r.Get("/proxy", s.handleImageProxy)
		r.Get("/proxy/page", s.handlePageProxy)

		r.Get("/news", s.handleListNews)
		r.Get("/news/{id}/content", s.handleGetFullContent)
		r.Put("/news/read-all", s.handleMarkAllRead)
		r.Put("/news/{id}/read", s.handleMarkRead)
		r.Put("/news/{id}/favorite", s.handleMarkFavorite)

		r.Get("/feeds", s.handleListFeeds)
		r.Post("/feeds", s.handleAddFeed)
		r.Post("/feeds/import", s.handleImportOPML)
		r.Get("/feeds/export", s.handleExportOPML)
		r.Patch("/feeds/{id}", s.handleUpdateFeed)
		r.Delete("/feeds/{id}", s.handleDeleteFeed)
		r.Post("/recipes/save", s.handleSaveRecipe)
		r.Post("/recipes/update", s.handleUpdateRecipes)

		r.Put("/categories/{name}", s.handleRenameCategory)

		r.Get("/config/export", s.handleExportConfig)
		r.Post("/config/import", s.handleImportConfig)

		r.Get("/filters", s.handleListFilters)
		r.Post("/filters", s.handleAddFilter)
		r.Delete("/filters/{id}", s.handleDeleteFilter)

		r.Get("/api-keys", s.handleListAPIKeys)
		r.Post("/api-keys", s.handleCreateAPIKey)
		r.Delete("/api-keys/{id}", s.handleDeleteAPIKey)

		r.Get("/stats", s.handleGetStats)

		r.Get("/backup/export", s.handleExportBackup)
		r.Post("/backup/import", s.handleImportBackup)
	})

	// Serve Svelte Frontend
	staticFS, _ := fs.Sub(frontendFS, "web/dist")
	fileServer := http.FileServer(http.FS(staticFS))

	s.router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		// Serve index.html for all non-API routes (SPA support)
		path := r.URL.Path
		if path == "/" || !strings.Contains(path, ".") {
			content, err := fs.ReadFile(staticFS, "index.html")
			if err != nil {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			w.Write(content)
			return
		}

		// Serve static assets (js, css, etc)
		fileServer.ServeHTTP(w, r)
	})
}

func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}

func (s *Server) GetRouter() http.Handler {
	return s.router
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow OPTIONS requests for preflight
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		var username string
		var token string

		// 1. Try Cookie first
		if cookie, err := r.Cookie("gorrson_auth"); err == nil {
			token = cookie.Value
			username, _ = auth.ValidateToken(token)
		}

		// 2. Try Authorization header if no valid user from cookie
		if username == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
				username, _ = auth.ValidateToken(token)
			}
		}

		// 3. Try "token" query parameter (essential for iframes/proxies)
		if username == "" {
			token = r.URL.Query().Get("token")
			if token != "" {
				username, _ = auth.ValidateToken(token)
			}
		}

		// 4. Try X-API-Key (format: "user:key") if still no user
		if username == "" {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey != "" {
				parts := strings.SplitN(apiKey, ":", 2)
				if len(parts) == 2 {
					u := parts[0]
					rawKey := parts[1]

					// Abrir SIN crear: con getStore, una cabecera
					// X-API-Key con cualquier nombre de usuario escribía una base
					// de datos nueva (~50 KB) sin haber autenticado nada, y
					// además la dejaba cacheada para siempre.
					store, serr := db.OpenExistingStore(s.dataDir, u)
					if serr == nil {
						hash := sha256.Sum256([]byte(rawKey))
						hashHex := hex.EncodeToString(hash[:])
						if store.ValidateAPIKey(r.Context(), hashHex) {
							username = u
						}
						_ = store.Close()
					}
				}
			}
		}

		if username == "" {
			slog.Warn("Auth failed: invalid token or API key", "path", r.URL.Path, "method", r.Method)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), usernameKey, username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip strict security headers for the page proxy to allow it to render correctly
		if strings.HasPrefix(r.URL.Path, "/proxy/page") {
			w.Header().Set("X-Frame-Options", "SAMEORIGIN")
			w.Header().Set("Content-Security-Policy", "default-src * 'unsafe-inline' 'unsafe-eval'; frame-ancestors 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src * 'unsafe-inline';")
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// frame-src explícito: sin él los iframes caen en default-src ('self') y
		// los embeds de las noticias (YouTube, Vimeo, Steam...) quedaban bloqueados
		// al abrir el lector. El src de los iframes ya viene saneado por el servidor
		// y solo se admiten URLs https.
		w.Header().Set("Content-Security-Policy", "default-src 'self' https://unpkg.com; frame-ancestors 'self'; frame-src https:; script-src 'self' 'unsafe-inline' https://unpkg.com; script-src-elem 'self' 'unsafe-inline' https://unpkg.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://unpkg.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data: http: https:; connect-src 'self' https://unpkg.com;")
		next.ServeHTTP(w, r)
	})
}

// uuidToInt converts a UUID string to a 32-bit integer for Fever API compatibility
func uuidToInt(id string) int {
	h := fnv.New32a()
	h.Write([]byte(id))
	return int(h.Sum32())
}

var proxySemaphore = make(chan struct{}, 10)

// resolveURL pasa una URL del documento remoto a absoluta.
//
// data: y javascript: se descartan por diseño. javascript: es ejecución de código
// en el origen de gorrson: el script de selección del modo lector llama a
// preventDefault() en los enlaces, pero eso solo cubre el clic izquierdo — clic
// central, "abrir en pestaña nueva" o el menú contextual lo esquivan sin
// problema.
func resolveURL(base *url.URL, rel string) string {
	lowered := strings.ToLower(strings.TrimSpace(rel))
	if strings.HasPrefix(lowered, "javascript:") || strings.HasPrefix(lowered, "vbscript:") ||
		strings.HasPrefix(lowered, "data:text/html") {
		return "#"
	}
	if strings.HasPrefix(lowered, "data:") {
		// data: solo se admite para imágenes en línea; el resto se descarta.
		return rel
	}
	u, err := url.Parse(rel)
	if err != nil {
		return "#"
	}
	return base.ResolveReference(u).String()
}

func injectAtStart(n *html.Node, tag string, child *html.Node) {
	var target *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == tag {
			target = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
			if target != nil {
				return
			}
		}
	}
	walk(n)

	if target != nil {
		if target.FirstChild != nil {
			target.InsertBefore(child, target.FirstChild)
		} else {
			target.AppendChild(child)
		}
	}
}

func (s *Server) cleanupImageCache(maxAge time.Duration) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	cacheDir := filepath.Join(s.dataDir, "cache", "images")

	clean := func() {
		slog.Info("Cleaning up image cache", "dir", cacheDir)
		_ = filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if time.Since(info.ModTime()) > maxAge {
				_ = os.Remove(path)
			}
			return nil
		})
	}

	// Initial clean
	clean()

	for {
		select {
		case <-ticker.C:
			clean()
		}
	}
}

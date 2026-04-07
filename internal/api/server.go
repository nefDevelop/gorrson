package api

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/nef734/gorrson/internal/auth"
	"github.com/nef734/gorrson/internal/db"
	"github.com/nef734/gorrson/internal/models"
	"github.com/nef734/gorrson/internal/opml"
	"github.com/nef734/gorrson/internal/rss"
)

var (
	//go:embed all:web/dist
	frontendFS embed.FS

	loginAttempts = make(map[string][]time.Time)
	loginMu       sync.Mutex
)

func (s *Server) loginRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/login" {
			next.ServeHTTP(w, r)
			return
		}

		ip := r.RemoteAddr
		if pos := strings.LastIndex(ip, ":"); pos != -1 {
			ip = ip[:pos] // Strip port
		}

		loginMu.Lock()
		now := time.Now()
		attempts := loginAttempts[ip]

		var valid []time.Time
		for _, t := range attempts {
			if now.Sub(t) < time.Minute {
				valid = append(valid, t)
			}
		}

		if len(valid) >= 5 {
			loginMu.Unlock()
			http.Error(w, "Too many login attempts. Try again in a minute.", http.StatusTooManyRequests)
			return
		}

		loginAttempts[ip] = append(valid, now)
		loginMu.Unlock()

		next.ServeHTTP(w, r)
	})
}

type Server struct {
	router  *chi.Mux
	dataDir string
	parser  *rss.Parser
}

func NewServer(dataDir string) *Server {
	s := &Server{
		router:  chi.NewRouter(),
		dataDir: dataDir,
		parser:  rss.NewParser(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Compress(5))

	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			next.ServeHTTP(w, r)
		})
	})

	s.router.Get("/openapi.yaml", s.handleOpenAPI)
	s.router.Get("/docs", s.handleDocs)
	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	s.router.With(s.loginRateLimit).Post("/login", s.handleLogin)
	s.router.Post("/refresh", s.handleRefresh)
	s.router.Post("/logout", s.handleLogout)
	s.router.Get("/proxy", s.handleImageProxy)

	s.router.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)

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
		
		r.Put("/categories/{name}", s.handleRenameCategory)

		r.Get("/filters", s.handleListFilters)
		r.Post("/filters", s.handleAddFilter)
		r.Delete("/filters/{id}", s.handleDeleteFilter)
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
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		username, err := auth.ValidateToken(token)
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "username", username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	store, err := db.NewStore(s.dataDir, req.Username)
	if err != nil {
		http.Error(w, "Could not open user database", http.StatusNotFound)
		return
	}
	defer store.Close()

	hash, err := store.GetUserHash(req.Username)
	if err != nil || !auth.CheckPasswordHash(req.Password, hash) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	accessToken, _ := auth.GenerateToken(req.Username)
	refreshToken, _ := auth.GenerateRefreshToken()
	session := &db.Session{
		ID:           uuid.New().String(),
		Username:     req.Username,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(time.Hour * 24 * 30),
	}
	store.CreateSession(session)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
	Username     string `json:"username"`
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	store, err := db.NewStore(s.dataDir, req.Username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	session, err := store.GetSession(req.RefreshToken)
	if err != nil || session.ExpiresAt.Before(time.Now()) {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	newAccessToken, _ := auth.GenerateToken(session.Username)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"access_token": newAccessToken})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	store, err := db.NewStore(s.dataDir, req.Username)
	if err == nil {
		defer store.Close()
		store.DeleteSession(req.RefreshToken)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListNews(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value("username").(string)
	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	unread := r.URL.Query().Get("unread") == "true"
	sort := r.URL.Query().Get("sort")
	search := r.URL.Query().Get("search")
	category := r.URL.Query().Get("category")
	feedID := r.URL.Query().Get("feed_id")
	
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50 // Default limit for mobile safety
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	// Sanitize params: frontend may send the literal string "null"
	if category == "null" || category == "undefined" {
		category = ""
	}
	if feedID == "null" || feedID == "undefined" {
		feedID = ""
	}

	var entries []models.Entry
	if search != "" {
		entries, err = store.SearchEntries(search)
	} else {
		entries, err = store.ListEntries(unread, sort, limit, offset, category, feedID)
	}

	if err != nil {
		http.Error(w, "Query failure", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func (s *Server) handleGetFullContent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	username := r.Context().Value("username").(string)
	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	entry, err := store.GetEntryByID(id)
	if err != nil {
		http.Error(w, "Entry not found", http.StatusNotFound)
		return
	}

	if entry.IsFullContent {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(entry.Content))
		return
	}

	fullContent, err := rss.ScrapeFullContent(r.Context(), entry.URL, entry.Title, entry.ImageURL)
	if err != nil {
		slog.Error("Failed to scrape full content", "url", entry.URL, "id", id, "err", err)
		http.Error(w, "Scraping failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	store.UpdateEntryContent(id, fullContent)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fullContent))
}

func (s *Server) handleMarkRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	username := r.Context().Value("username").(string)
	store, err := db.NewStore(s.dataDir, username)
	if err == nil {
		defer store.Close()
		store.MarkAsRead(id)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMarkAllRead(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value("username").(string)
	
	category := r.URL.Query().Get("category")
	feedID := r.URL.Query().Get("feed_id")
	search := r.URL.Query().Get("search")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	
	if category == "null" || category == "undefined" { category = "" }
	if feedID == "null" || feedID == "undefined" { feedID = "" }

	store, err := db.NewStore(s.dataDir, username)
	if err == nil {
		defer store.Close()
		store.MarkAllRead(category, feedID, search, limit)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMarkFavorite(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	isFavorite := r.URL.Query().Get("favorite") != "false"
	username := r.Context().Value("username").(string)
	store, err := db.NewStore(s.dataDir, username)
	if err == nil {
		defer store.Close()
		store.ToggleFavorite(id, isFavorite)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListFeeds(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value("username").(string)
	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	feeds, err := store.ListFeeds()
	if err != nil {
		http.Error(w, "Query failure", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feeds)
}

func (s *Server) handleAddFeed(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL      string `json:"url"`
		Category string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		http.Error(w, "Missing or invalid 'url' field", http.StatusBadRequest)
		return
	}
	username := r.Context().Value("username").(string)
	slog.Info("handleAddFeed: request", "user", username, "url", req.URL, "category", req.Category)

	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		http.Error(w, "Could not open DB", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	if err := s.parser.SyncFeed(r.Context(), req.URL, store); err != nil {
		slog.Error("handleAddFeed: SyncFeed failed", "url", req.URL, "err", err)
		http.Error(w, "Failed to sync feed: "+err.Error(), http.StatusBadRequest)
		return
	}

	feed, _ := store.GetFeedByURL(req.URL)
	if feed != nil && req.Category != "" {
		store.UpdateFeed(feed.ID, feed.Title, req.Category)
		feed.Category = req.Category
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if feed != nil {
		json.NewEncoder(w).Encode(feed)
	} else {
		json.NewEncoder(w).Encode(map[string]string{"url": req.URL})
	}
}

func (s *Server) handleImportOPML(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value("username").(string)
	var reader io.Reader = r.Body
	if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		file, _, err := r.FormFile("file")
		if err == nil {
			defer file.Close()
			reader = file
		}
	}

	outlines, err := opml.Parse(reader)
	if err == nil {
		go func(feeds []opml.Outline, user string) {
			st, _ := db.NewStore(s.dataDir, user)
			defer st.Close()
			for _, o := range feeds {
				if o.XMLURL != "" {
					s.parser.SyncFeed(context.Background(), o.XMLURL, st)
				}
			}
		}(outlines, username)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Import started", "count": len(outlines)})
}

func (s *Server) handleExportOPML(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value("username").(string)
	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	feeds, _ := store.ListFeeds()
	var outlines []opml.Outline
	for _, f := range feeds {
		outlines = append(outlines, opml.Outline{Text: f.Title, XMLURL: f.FeedURL, HTMLURL: f.SiteURL, Category: f.Category})
	}

	opmlBytes, _ := opml.Generate(outlines)
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", "attachment; filename=gorrson.opml")
	w.Write(opmlBytes)
}

func (s *Server) handleUpdateFeed(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct{ Title, Category string }
	json.NewDecoder(r.Body).Decode(&req)
	username := r.Context().Value("username").(string)
	store, err := db.NewStore(s.dataDir, username)
	if err == nil {
		defer store.Close()
		store.UpdateFeed(id, req.Title, req.Category)
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDeleteFeed(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	username := r.Context().Value("username").(string)
	store, err := db.NewStore(s.dataDir, username)
	if err == nil {
		defer store.Close()
		store.DeleteFeed(id)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListFilters(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value("username").(string)
	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	filters, err := store.ListFilters()
	if err != nil {
		http.Error(w, "Query failure", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filters)
}

func (s *Server) handleAddFilter(w http.ResponseWriter, r *http.Request) {
	var f models.Filter
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if _, err := regexp.Compile(f.Pattern); err != nil {
		http.Error(w, "Invalid regex pattern: "+err.Error(), http.StatusBadRequest)
		return
	}

	f.ID = uuid.New().String()
	username := r.Context().Value("username").(string)
	store, err := db.NewStore(s.dataDir, username)
	if err == nil {
		defer store.Close()
		store.AddFilter(&f)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(f)
}

func (s *Server) handleDeleteFilter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	username := r.Context().Value("username").(string)
	store, err := db.NewStore(s.dataDir, username)
	if err == nil {
		defer store.Close()
		store.DeleteFilter(id)
	}
	w.WriteHeader(http.StatusNoContent)
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
		// Serve from cache
		// We'll peek at the file to get the content-type, or just use a generic one
		// Since browser is smart, serving as octet-stream might work, 
		// but let's try to be accurate.
		f, _ := os.Open(cachePath)
		defer f.Close()
		
		// Set some cache headers for the client browser
		w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year
		http.ServeContent(w, r, "image", time.Time{}, f)
		return
	}

	// 3. Not cached, download it using SecureClient
	sc := rss.NewSecureClient()
	body, err := sc.Fetch(r.Context(), imageURL)
	if err != nil {
		http.Error(w, "Failed to fetch image: "+err.Error(), http.StatusBadGateway)
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

	if err := os.Rename(tmpPath, cachePath); err != nil {
		http.Error(w, "Could not commit cache file", http.StatusInternalServerError)
		return
	}

	// 5. Serve it
	f, _ := os.Open(cachePath)
	defer f.Close()
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	http.ServeContent(w, r, "image", time.Time{}, f)
}

func (s *Server) handleRenameCategory(w http.ResponseWriter, r *http.Request) {
	oldName := chi.URLParam(r, "name")
	var req struct {
		NewName string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	username := r.Context().Value("username").(string)
	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	if err := store.RenameCategory(oldName, req.NewName); err != nil {
		http.Error(w, "Rename failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "docs/openapi.yaml")
}

func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	html := `<!doctype html><html><head><meta charset="utf-8"><script type="module" src="https://unpkg.com/rapidoc/dist/rapidoc-min.js"></script><title>Gorrson API</title></head><body><rapi-doc spec-url="/openapi.yaml" theme="dark" show-header="false" allow-authentication="true" render-style="focused"></rapi-doc></body></html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

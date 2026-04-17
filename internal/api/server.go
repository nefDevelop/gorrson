package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"io/fs"
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
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/nef734/gorrson/internal/auth"
	"github.com/nef734/gorrson/internal/db"
	"github.com/nef734/gorrson/internal/models"
	"github.com/nef734/gorrson/internal/opml"
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
}

func NewServer(dataDir string, silent bool) *Server {
	s := &Server{
		router:  chi.NewRouter(),
		dataDir: dataDir,
		parser:  rss.NewParser(),
		silent:  silent,
	}
	s.routes()
	
	// Start background cache cleaner (Run every 24h, keep files for 15 days)
	go s.cleanupImageCache(15 * 24 * time.Hour)
	
	return s
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

					store, serr := db.NewStore(s.dataDir, u)
					if serr == nil {
						defer store.Close()
						hash := sha256.Sum256([]byte(rawKey))
						hashHex := hex.EncodeToString(hash[:])
						if store.ValidateAPIKey(hashHex) {
							username = u
						}
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
		w.Header().Set("Content-Security-Policy", "default-src 'self' https://unpkg.com; frame-ancestors 'self'; script-src 'self' 'unsafe-inline' https://unpkg.com; script-src-elem 'self' 'unsafe-inline' https://unpkg.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://unpkg.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data: http: https:; connect-src 'self' https://unpkg.com;")
		next.ServeHTTP(w, r)
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
	_ = store.CreateSession(session)

	http.SetCookie(w, &http.Cookie{
		Name:     "gorrson_auth",
		Value:    accessToken,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
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

	http.SetCookie(w, &http.Cookie{
		Name:     "gorrson_auth",
		Value:    newAccessToken,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"access_token": newAccessToken})
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
		_ = store.DeleteSession(req.RefreshToken)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "gorrson_auth",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListNews(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(usernameKey).(string)
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
		entries, err = store.SearchEntries(search, limit, offset)
	} else {
		entries, err = store.ListEntries(unread, sort, limit, offset, category, feedID)
	}

	if err != nil {
		http.Error(w, "Query failure", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entries)
}

func (s *Server) handleGetFullContent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	username := r.Context().Value(usernameKey).(string)
	slog.Debug("Full content requested", "id", id, "user", username)
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

	var customUA string
	feed, err := store.GetFeedByID(entry.FeedID)
	if err == nil {
		customUA = feed.UserAgent
	}

	// NOTE: This requires changing the signature of rss.ScrapeFullContent
	// to func ScrapeFullContent(ctx context.Context, pageURL, title, heroImageURL, userAgent string) (string, error)
	// and using the userAgent when creating the SecureClient.
	fullContent, err := rss.ScrapeFullContent(r.Context(), entry.URL, entry.Title, entry.ImageURL, customUA)
	if err != nil {
		slog.Error("Failed to scrape full content", "url", entry.URL, "id", id, "err", err)
		http.Error(w, "Content extraction failed (the site might be blocking us or require a subscription)", http.StatusServiceUnavailable)
		return
	}

	_ = store.UpdateEntryContent(id, fullContent)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fullContent))
}

func (s *Server) handleMarkRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	username := r.Context().Value(usernameKey).(string)
	store, err := db.NewStore(s.dataDir, username)
	if err == nil {
		defer store.Close()
		_ = store.MarkAsRead(id)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMarkAllRead(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(usernameKey).(string)

	category := r.URL.Query().Get("category")
	feedID := r.URL.Query().Get("feed_id")
	search := r.URL.Query().Get("search")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if category == "null" || category == "undefined" {
		category = ""
	}
	if feedID == "null" || feedID == "undefined" {
		feedID = ""
	}

	store, err := db.NewStore(s.dataDir, username)
	if err == nil {
		defer store.Close()
		_ = store.MarkAllRead(category, feedID, search, limit)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMarkFavorite(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	isFavorite := r.URL.Query().Get("favorite") != "false"
	username := r.Context().Value(usernameKey).(string)
	store, err := db.NewStore(s.dataDir, username)
	if err == nil {
		defer store.Close()
		_ = store.ToggleFavorite(id, isFavorite)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListFeeds(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(usernameKey).(string)
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

	// Flatten into a map to ensure all fields are at top level in JSON
	var enriched []map[string]interface{}
	for _, f := range feeds {
		hasRecipe := false
		target := f.SiteURL
		if target == "" {
			target = f.FeedURL
		}
		if r, ferr := rss.FindRecipe(target); ferr == nil && r != nil {
			hasRecipe = true
		}

		enriched = append(enriched, map[string]interface{}{
			"id":           f.ID,
			"title":        f.Title,
			"site_url":     f.SiteURL,
			"feed_url":     f.FeedURL,
			"category":     f.Category,
			"unread_count": f.UnreadCount,
			"has_recipe":   hasRecipe,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(enriched)
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
	username := r.Context().Value(usernameKey).(string)

	// Auto-tag social media domains if no category provided
	if req.Category == "" {
		lowerURL := strings.ToLower(req.URL)
		if strings.Contains(lowerURL, "reddit.com") {
			req.Category = "Reddit"
		} else if strings.Contains(lowerURL, "/@") || strings.Contains(lowerURL, "mastodon") {
			req.Category = "Mastodon"
		} else if strings.Contains(lowerURL, "bsky.app") {
			req.Category = "Bluesky"
		}
	}

	slog.Info("handleAddFeed: request", "user", username, "url", req.URL, "category", req.Category)

	// Update recipes in background to catch new site rules
	go rss.UpdateRecipes()

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
		store.UpdateFeed(feed.ID, feed.Title, req.Category, feed.UserAgent)
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
	username := r.Context().Value(usernameKey).(string)

	// Security: Limit size to 5MB to avoid memory DOS
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)

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
			st, err := db.NewStore(s.dataDir, user)
			if err != nil {
				slog.Error("ImportOPML goroutine: failed to open store", "user", user, "err", err)
				return
			}
			defer st.Close()

			// Use a small semaphore to throttle background syncs
			sem := make(chan struct{}, 5)
			for _, o := range feeds {
				if o.XMLURL != "" {
					sem <- struct{}{}
					go func(url string) {
						defer func() { <-sem }()
						if err := s.parser.SyncFeed(context.Background(), url, st); err != nil {
							slog.Error("ImportOPML goroutine: sync failed", "url", url, "err", err)
						}
					}(o.XMLURL)
				}
			}
			// Wait for all in this batch to finish
			for i := 0; i < cap(sem); i++ {
				sem <- struct{}{}
			}
		}(outlines, username)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Import started", "count": len(outlines)})
}

func (s *Server) handleExportOPML(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(usernameKey).(string)
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
	var req struct {
		Title    string `json:"title"`
		Category string `json:"category"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	username := r.Context().Value(usernameKey).(string)
	store, err := db.NewStore(s.dataDir, username)
	if err == nil {
		defer store.Close()
		feed, err := store.GetFeedByID(id)
		if err == nil {
			store.UpdateFeed(id, req.Title, req.Category, feed.UserAgent)
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDeleteFeed(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	username := r.Context().Value(usernameKey).(string)
	store, err := db.NewStore(s.dataDir, username)
	if err == nil {
		defer store.Close()
		store.DeleteFeed(id)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListFilters(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(usernameKey).(string)
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
	username := r.Context().Value(usernameKey).(string)
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
	username := r.Context().Value(usernameKey).(string)
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
	sc := rss.NewSecureClient("")
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

	username := r.Context().Value(usernameKey).(string)
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
func (s *Server) handleExportConfig(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(usernameKey).(string)
	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	filters, _ := store.ListFilters()
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
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	for _, f := range config.Filters {
		if f.Pattern == "" {
			continue
		}
		if f.ID == "" {
			f.ID = uuid.New().String()
		}
		store.AddFilter(&f)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Configuration imported successfully"})
}

// uuidToInt converts a UUID string to a 32-bit integer for Fever API compatibility
func uuidToInt(id string) int {
	h := fnv.New32a()
	h.Write([]byte(id))
	return int(h.Sum32())
}

func (s *Server) handleFeverAPI(w http.ResponseWriter, r *http.Request) {
	// 1. Get API Key
	apiKey := r.FormValue("api_key")
	if apiKey == "" {
		// Try from POST body if not in query
		r.ParseForm()
		apiKey = r.Form.Get("api_key")
	}

	// 2. We need a username to open the DB.
	// 2. We need a username to open the DB.
	// Fever doesn't send the username directly, it's hashed in the api_key.
	// Since Gorrson is designed for one user per DB file, we'll look at the data directory
	files, _ := os.ReadDir(s.dataDir)
	var userStore *db.Store

	for _, f := range files {
		if filepath.Ext(f.Name()) == ".db" && f.Name() != "global.db" {
			username := strings.TrimSuffix(f.Name(), ".db")
			store, err := db.NewStore(s.dataDir, username)
			if err != nil {
				continue
			}
			feverKey, err := store.GetUserFeverKey(username)
			if err != nil || feverKey == "" {
				store.Close()
				continue
			}

			// Validate MD5 apiKey against bcrypt feverKey
			if auth.CheckPasswordHash(apiKey, feverKey) {
				userStore = store
				break
			}
			store.Close()
		}
	}

	if userStore == nil {
		slog.Warn("Fever API: Unauthorized access attempt", "api_key", apiKey)
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
		feeds, _ := userStore.ListFeeds()
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
		feeds, _ := userStore.ListFeeds()
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
		unreadOnly := r.URL.Query().Has("unread_item_ids")
		entries, _ := userStore.ListEntries(unreadOnly, "desc", 500, 0, "", "")

		if r.URL.Query().Has("unread_item_ids") {
			var ids []string
			for _, e := range entries {
				ids = append(ids, strconv.Itoa(e.FeverID))
			}
			resp["unread_item_ids"] = strings.Join(ids, ",")
		}

		if r.URL.Query().Has("items") {
			var items []map[string]interface{}
			for _, e := range entries {
				items = append(items, map[string]interface{}{
					"id":              e.FeverID,
					"feed_id":         uuidToInt(e.FeedID), // Using uuidToInt for consistency if FeedID is UUID
					"title":           e.Title,
					"author":          "",
					"html":            e.Content,
					"url":             e.URL,
					"is_saved":        0,
					"is_read":         0,
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
			_ = userStore.MarkAsReadByFeverID(id)
		case "saved":
			_ = userStore.ToggleFavoriteByFeverID(id, true)
		case "unsaved":
			_ = userStore.ToggleFavoriteByFeverID(id, false)
		}
	} else if r.FormValue("mark") == "feed" {
		id, _ := strconv.Atoi(r.FormValue("id"))
		if r.FormValue("as") == "read" {
			// Find category or feed by FeverID and mark all read
			feeds, _ := userStore.ListFeeds()
			for _, f := range feeds {
				if f.FeverID == id {
					_ = userStore.MarkAllRead("", f.ID, "", 0)
					break
				}
			}
		}
	} else if r.FormValue("mark") == "group" {
		id, _ := strconv.Atoi(r.FormValue("id"))
		if r.FormValue("as") == "read" {
			// Find category by FeverID
			feeds, _ := userStore.ListFeeds()
			for _, f := range feeds {
				if uuidToInt(f.Category) == id {
					_ = userStore.MarkAllRead(f.Category, "", "", 0)
					break
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	username, ok := r.Context().Value(usernameKey).(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	keys, err := store.ListAPIKeys()
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
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

	if err := store.AddAPIKey(id, req.Name, hashHex); err != nil {
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

	if err := store.DeleteAPIKey(id); err != nil {
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	if err := db.VerifySQLiteFile(tmpPath); err != nil {
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

	slog.Info("Backup restored successfully", "user", username, "ip", r.RemoteAddr)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Database restored and verified. Page refresh recommended."})
}

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(usernameKey).(string)
	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	stats, err := store.GetStats()
	if err != nil {
		http.Error(w, "Failed to retrieve stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handlePageProxy(w http.ResponseWriter, r *http.Request) {
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
			feed, err := store.GetFeedByID(feedID)
			if err == nil && feed.UserAgent != "" {
				customUA = feed.UserAgent
			}
		}
	}
	sc := rss.NewSecureClient(customUA)
	body, err := sc.Fetch(r.Context(), targetURL)
	if err != nil {
		slog.Error("Failed to fetch proxy page", "url", targetURL, "error", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "<body style='font-family:sans-serif;padding:40px;text-align:center;background:#fff5f5;color:#c53030'>")
		fmt.Fprintf(w, "<h2>Proxy Error</h2>")
		fmt.Fprintf(w, "<p>Could not load the page. Check your internet connection or if the site blocks Gorrson.</p>")
		fmt.Fprintf(w, "<pre style='background:#fff;padding:10px;border-radius:8px;display:inline-block'>%s</pre>", err.Error())
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
			// Disable scripts by changing type
			if n.Data == "script" {
				for i, attr := range n.Attr {
					if attr.Key == "src" {
						n.Attr[i].Val = "#disabled-by-gorrson"
					}
				}
				n.Attr = append(n.Attr, html.Attribute{Key: "type", Val: "text/plain"})
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

func (s *Server) handleSaveRecipe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FeedID   string   `json:"feed_id"`
		BodyList []string `json:"body_list"`
		Title    string   `json:"title"`
		Strip    []string `json:"strip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	username := r.Context().Value(usernameKey).(string)
	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer store.Close()

	feed, err := store.GetFeedByID(req.FeedID)
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

func resolveURL(base *url.URL, rel string) string {
	if strings.HasPrefix(rel, "data:") || strings.HasPrefix(rel, "javascript:") {
		return rel
	}
	u, err := url.Parse(rel)
	if err != nil {
		return rel
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

func (s *Server) handleUpdateRecipes(w http.ResponseWriter, r *http.Request) {
	go rss.UpdateRecipes() // Run in background
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Recipe update process started in the background."})
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

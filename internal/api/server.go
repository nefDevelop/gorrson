package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nef734/gorrson/internal/auth"
	"github.com/nef734/gorrson/internal/db"
	"github.com/nef734/gorrson/internal/rss"
)

// Server defines the REST API layer for rss-cli.
type Server struct {
	router  *chi.Mux
	dataDir string
	parser  *rss.Parser
}

// NewServer initializes the chi router and routes.
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

	s.router.Post("/login", s.handleLogin)

	s.router.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)

		// News
		r.Get("/news", s.handleListNews)
		r.Put("/news/{id}/read", s.handleMarkRead)

		// Feeds (New CRUD)
		r.Get("/feeds", s.handleListFeeds)
		r.Post("/feeds", s.handleAddFeed)
		r.Patch("/feeds/{id}", s.handleUpdateFeed)
		r.Delete("/feeds/{id}", s.handleDeleteFeed)
	})
}

// Start launches the HTTP server.
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized: No Authorization header", http.StatusUnauthorized)
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
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if !auth.CheckPasswordHash(req.Password, hash) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(req.Username)
	if err != nil {
		http.Error(w, "Token generation failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (s *Server) handleListNews(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value("username").(string)
	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	unread := r.URL.Query().Get("unread") == "true"
	sort := r.URL.Query().Get("sort")
	search := r.URL.Query().Get("search")

	var entries []db.Entry
	if search != "" {
		entries, err = store.SearchEntries(search)
	} else {
		entries, err = store.ListEntries(unread, sort)
	}

	if err != nil {
		http.Error(w, "Query failure: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func (s *Server) handleMarkRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	username := r.Context().Value("username").(string)
	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	if err := store.MarkAsRead(id); err != nil {
		http.Error(w, "Update failure", http.StatusInternalServerError)
		return
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

type addFeedRequest struct {
	URL string `json:"url"`
}

func (s *Server) handleAddFeed(w http.ResponseWriter, r *http.Request) {
	var req addFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	username := r.Context().Value("username").(string)
	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	if err := s.parser.SyncFeed(r.Context(), req.URL, store); err != nil {
		http.Error(w, "Sync failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

type updateFeedRequest struct {
	Title    string `json:"title"`
	Category string `json:"category"`
}

func (s *Server) handleUpdateFeed(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	username := r.Context().Value("username").(string)
	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	if err := store.UpdateFeed(id, req.Title, req.Category); err != nil {
		http.Error(w, "Update failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDeleteFeed(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	username := r.Context().Value("username").(string)
	store, err := db.NewStore(s.dataDir, username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	if err := store.DeleteFeed(id); err != nil {
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

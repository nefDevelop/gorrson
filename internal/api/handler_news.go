package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/nef734/gorrson/internal/models"
	"github.com/nef734/gorrson/internal/rss"
)

func (s *Server) handleListNews(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(usernameKey).(string)
	store, err := s.getStore(username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

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
		entries, err = store.SearchEntries(r.Context(), search, limit, offset)
	} else {
		entries, err = store.ListEntries(r.Context(), unread, sort, limit, offset, category, feedID)
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
	store, err := s.getStore(username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	entry, err := store.GetEntryByID(r.Context(), id)
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
	feed, err := store.GetFeedByID(r.Context(), entry.FeedID)
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

	_ = store.UpdateEntryContent(r.Context(), id, fullContent)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fullContent))
}

func (s *Server) handleMarkRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	username := r.Context().Value(usernameKey).(string)
	store, err := s.getStore(username)
	if err == nil {
		_ = store.MarkAsRead(r.Context(), id)
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

	store, err := s.getStore(username)
	if err == nil {
		_ = store.MarkAllRead(r.Context(), category, feedID, search, limit)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMarkFavorite(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	isFavorite := r.URL.Query().Get("favorite") != "false"
	username := r.Context().Value(usernameKey).(string)
	store, err := s.getStore(username)
	if err == nil {
		_ = store.ToggleFavorite(r.Context(), id, isFavorite)
	}
	w.WriteHeader(http.StatusNoContent)
}

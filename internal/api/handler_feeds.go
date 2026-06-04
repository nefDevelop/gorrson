package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/nef734/gorrson/internal/db"
	"github.com/nef734/gorrson/internal/opml"
	"github.com/nef734/gorrson/internal/rss"
)

func (s *Server) handleListFeeds(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(usernameKey).(string)
	store, err := s.getStore(username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	feeds, err := store.ListFeeds(r.Context())
	if err != nil {
		http.Error(w, "Query failure", http.StatusInternalServerError)
		return
	}

	type feedResponse struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		SiteURL     string `json:"site_url"`
		FeedURL     string `json:"feed_url"`
		Category    string `json:"category"`
		UnreadCount int    `json:"unread_count"`
		HasRecipe   bool   `json:"has_recipe"`
	}
	var enriched []feedResponse
	for _, f := range feeds {
		hasRecipe := false
		target := f.SiteURL
		if target == "" {
			target = f.FeedURL
		}
		if r, ferr := rss.FindRecipe(target); ferr == nil && r != nil {
			hasRecipe = true
		}

		enriched = append(enriched, feedResponse{
			ID:          f.ID,
			Title:       f.Title,
			SiteURL:     f.SiteURL,
			FeedURL:     f.FeedURL,
			Category:    f.Category,
			UnreadCount: f.UnreadCount,
			HasRecipe:   hasRecipe,
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
	if err := decodeJSONBody(w, r, &req); err != nil || req.URL == "" {
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

	store, err := s.getStore(username)
	if err != nil {
		http.Error(w, "Could not open DB", http.StatusInternalServerError)
		return
	}

	if err := s.parser.SyncFeed(r.Context(), req.URL, store); err != nil {
		slog.Error("handleAddFeed: SyncFeed failed", "url", req.URL, "err", err)
		http.Error(w, "Failed to sync feed: "+err.Error(), http.StatusBadRequest)
		return
	}

	feed, _ := store.GetFeedByURL(r.Context(), req.URL)
	if feed != nil && req.Category != "" {
		store.UpdateFeed(r.Context(), feed.ID, feed.Title, req.Category, feed.UserAgent)
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
			st, err := s.getStore(user)
			if err != nil {
				slog.Error("ImportOPML goroutine: failed to open store", "user", user, "err", err)
				return
			}

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
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "Import started", "count": len(outlines)})
	} else {
		http.Error(w, "Invalid or too large OPML file: "+err.Error(), http.StatusBadRequest)
	}
}

func (s *Server) handleExportOPML(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(usernameKey).(string)
	store, err := s.getStore(username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	feeds, _ := store.ListFeeds(r.Context())
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
	decodeJSONBody(w, r, &req)
	username := r.Context().Value(usernameKey).(string)
	store, err := s.getStore(username)
	if err == nil {
		feed, err := store.GetFeedByID(r.Context(), id)
		if err == nil {
			store.UpdateFeed(r.Context(), id, req.Title, req.Category, feed.UserAgent)
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDeleteFeed(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	username := r.Context().Value(usernameKey).(string)
	store, err := s.getStore(username)
	if err == nil {
		store.DeleteFeed(r.Context(), id)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRenameCategory(w http.ResponseWriter, r *http.Request) {
	oldName := chi.URLParam(r, "name")
	var req struct {
		NewName string `json:"name"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
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

	if err := store.RenameCategory(r.Context(), oldName, req.NewName); err != nil {
		http.Error(w, "Rename failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/nef734/gorrson/internal/auth"
	"github.com/nef734/gorrson/internal/db"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
	Username     string `json:"username"`
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
	return json.NewDecoder(r.Body).Decode(dst)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	store, err := s.getStore(req.Username)
	if err != nil {
		http.Error(w, "Could not open user database", http.StatusNotFound)
		return
	}

	hash, err := store.GetUserHash(r.Context(), req.Username)
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
	_ = store.CreateSession(r.Context(), session)

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

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	store, err := s.getStore(req.Username)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	session, err := store.GetSession(r.Context(), req.RefreshToken)
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
	if err := decodeJSONBody(w, r, &req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	store, err := s.getStore(req.Username)
	if err == nil {
		_ = store.DeleteSession(r.Context(), req.RefreshToken)
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

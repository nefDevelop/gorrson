package api

import (
	"encoding/json"
	"errors"
	"log/slog"
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

	// El límite se comprueba ANTES de tocar nada. Cada intento cuesta un argon2id
	// (~62 ms, ~67 MB), así que acotar aquí es lo que evita el amplificador de
	// memoria: /login es público y no hace falta credencial para invocarlo.
	if ok, motivo := s.permitirLogin(r, req.Username); !ok {
		slog.Warn("Login rate limited", "path", r.URL.Path, "ip", clientIP(r))
		http.Error(w, motivo, http.StatusTooManyRequests)
		return
	}

	// Abrir SIN crear ni migrar: autenticar contra un usuario inexistente no
	// puede escribir en disco (~50 KB por intento) ni dejar un *sql.DB cacheado
	// para siempre.
	store, err := db.OpenExistingStore(s.dataDir, req.Username)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			// Mismo mensaje que una contraseña incorrecta: no se revela qué
			// usuarios están registrados.
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Could not open user database", http.StatusInternalServerError)
		return
	}
	defer store.Close()

	hash, err := store.GetUserHash(r.Context(), req.Username)
	if err != nil || !auth.CheckPasswordHash(req.Password, hash) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Autenticación correcta: el contador se libera para que un usuario legítimo
	// que se equivocó unas veces no quede bloqueado hasta que expire la ventana.
	s.loginLiberado(r, req.Username)

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

	// Igual que en login: no crear la base de datos de un usuario que aún no existe.
	store, err := db.OpenExistingStore(s.dataDir, req.Username)
	if err != nil {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}
	defer store.Close()

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

	// Idem: no crear bases de datos en un logout.
	if store, err := db.OpenExistingStore(s.dataDir, req.Username); err == nil {
		_ = store.DeleteSession(r.Context(), req.RefreshToken)
		_ = store.Close()
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

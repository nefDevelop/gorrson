package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type apiErrorResponse struct {
	Error string `json:"error"`
}

// respondWithError logs the internal error and sends a sanitized JSON response to the client
func respondWithError(w http.ResponseWriter, r *http.Request, code int, message string, err error) {
	if err != nil {
		slog.Error("API Error", "path", r.URL.Path, "status", code, "message", message, "err", err)
	} else {
		slog.Warn("API Warning", "path", r.URL.Path, "status", code, "message", message)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(apiErrorResponse{Error: message})
}

// respondWithJSON is a helper to send JSON responses
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}

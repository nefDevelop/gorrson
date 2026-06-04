package api

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestOPMLImportSizeLimit(t *testing.T) {
	dataDir := "./test_opml_security"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	s := NewServer(dataDir, true)

	// Inyectamos un usuario por defecto
	store, _ := s.getStore("testuser")
	store.UpsertUser(context.Background(), store.GetUsername(), "hash", "")

	// Crear payload de 6MB (el límite es 5MB)
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "huge.opml")
	
	// Generar 6MB de datos XML válidos pero redundantes
	largeXML := `<?xml version="1.0" encoding="UTF-8"?><opml version="2.0"><body>` + 
		strings.Repeat(`<outline text="Test" xmlUrl="http://test.com/rss" />`, 150000) + 
		`</body></opml>`
	
	io.Copy(part, strings.NewReader(largeXML))
	writer.Close()

	req := httptest.NewRequest("POST", "/feeds/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// Simulamos el auth midleware
	ctx := context.WithValue(req.Context(), usernameKey, "testuser")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	
	// Direct call to handler
	s.handleImportOPML(w, req)

	// Debería rechazar por MaxBytesReader (StatusRequestEntityTooLarge no siempre lo devuelve go-chi si el body revienta al leer)
	if w.Code == http.StatusOK {
		t.Errorf("Expected failure for payload > 5MB, got 200 OK")
	}
}

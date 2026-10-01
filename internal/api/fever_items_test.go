package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nef734/gorrson/internal/auth"
	"github.com/nef734/gorrson/internal/db"
	"github.com/nef734/gorrson/internal/models"
)

// TestFeverItemsReportaEstadoReal comprueba que el listado de items de la Fever
// API devuelve is_read e is_saved con el valor real de cada entrada.
//
// Antes de esto los dos campos iban hardcodeados a 0, con lo que un cliente
// Fever CREIA que ninguna entrada estaba leida (por muy leida que estuviera) y
// que ningun favorito existia. Marcar y desmarcar funcionaba, pero el estado
// nunca llegaba de vuelta, asi que el cliente no podia sincronizar.
func TestFeverItemsReportaEstadoReal(t *testing.T) {
	dataDir := "./test_fever_items_data"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)
	username := "fever_items"
	feverApiKey := "fever_items_secret"

	store, err := db.NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	hash, _ := auth.HashPassword("pass")
	feverBcrypt, _ := auth.HashPassword(feverApiKey)
	if err := store.UpsertUser(context.Background(), username, hash, feverBcrypt); err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}

	ctx := context.Background()
	if err := store.AddFeed(ctx, &models.Feed{
		ID: "feed-1", Title: "Feed", FeedURL: "https://feed.com/rss",
		SiteURL: "https://feed.com", FeverID: 10, Category: "General",
		LastUpdated: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("AddFeed: %v", err)
	}

	// Tres estados distintos para poder comprobar los tres casos.
	casos := []struct {
		id        string
		titulo    string
		favorita  bool
		leida     bool
	}{
		{"e-fav", "Favorita y sin leer", true, false},
		{"e-read", "Leida y no favorita", false, true},
		{"e-plain", "Sin marcar", false, false},
	}
	for _, c := range casos {
		if err := store.AddEntry(ctx, &models.Entry{
			ID: c.id, FeedID: "feed-1", Title: c.titulo,
			URL:         "https://feed.com/" + c.id,
			Content:     "contenido",
			PublishedAt: time.Now().UTC(),
		}); err != nil {
			t.Fatalf("AddEntry %s: %v", c.id, err)
		}
		if c.favorita {
			if err := store.ToggleFavorite(ctx, c.id, true); err != nil {
				t.Fatalf("ToggleFavorite %s: %v", c.id, err)
			}
		}
		if c.leida {
			if err := store.MarkAsRead(ctx, c.id); err != nil {
				t.Fatalf("MarkAsRead %s: %v", c.id, err)
			}
		}
	}
	store.Close()

	// Pedimos el listado.
	req := httptest.NewRequest("GET", "/api/fever?api_key="+feverApiKey+"&items", nil)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("Fever items fallo: %d, cuerpo: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Auth  float64 `json:"auth"`
		Items []struct {
			Title   string      `json:"title"`
			IsSaved json.Number `json:"is_saved"`
			IsRead  json.Number `json:"is_read"`
		} `json:"items"`
	}
	// UseNumber para distinguir 0 de "0": si is_read volviera como true, el
	// numero no parsearia y el fallo seria claro.
	dec := json.NewDecoder(strings.NewReader(rr.Body.String()))
	dec.UseNumber()
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("decodificar respuesta: %v, cuerpo: %s", err, rr.Body.String())
	}
	if resp.Auth != 1 {
		t.Fatalf("auth = %v, se esperaba 1", resp.Auth)
	}
	if len(resp.Items) != 3 {
		t.Fatalf("se esperaban 3 items, llegaron %d", len(resp.Items))
	}

	esperado := map[string][2]string{
		"Favorita y sin leer": {"1", "0"},
		"Leida y no favorita": {"0", "1"},
		"Sin marcar":         {"0", "0"},
	}
	for _, it := range resp.Items {
		exp, ok := esperado[it.Title]
		if !ok {
			t.Errorf("item inesperado en la respuesta: %q", it.Title)
			continue
		}
		if it.IsSaved.String() != exp[0] {
			t.Errorf("%q: is_saved = %s, se esperaba %s", it.Title, it.IsSaved, exp[0])
		}
		if it.IsRead.String() != exp[1] {
			t.Errorf("%q: is_read = %s, se esperaba %s", it.Title, it.IsRead, exp[1])
		}
	}
}

// TestFeverItemsDevuelveNumerosNoBooleanos fija el tipo de los campos. Fever
// espera 0/1; si algun dia is_read volviera como true, el cliente lo leeria
// mal. Este test falla si el tipo cambia.
func TestFeverItemsDevuelveNumerosNoBooleanos(t *testing.T) {
	dataDir := "./test_fever_tipos_data"
	os.RemoveAll(dataDir)
	defer os.RemoveAll(dataDir)

	srv := NewServer(dataDir, true)
	username := "fever_tipos"
	feverApiKey := "fever_tipos_secret"

	store, err := db.NewStore(dataDir, username)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	hash, _ := auth.HashPassword("pass")
	feverBcrypt, _ := auth.HashPassword(feverApiKey)
	store.UpsertUser(context.Background(), username, hash, feverBcrypt)
	store.AddFeed(context.Background(), &models.Feed{
		ID: "f1", Title: "Feed", FeedURL: "https://feed.com/rss",
		SiteURL: "https://feed.com", FeverID: 1, LastUpdated: time.Now().UTC(),
	})
	store.AddEntry(context.Background(), &models.Entry{
		ID: "e1", FeedID: "f1", Title: "T", URL: "https://feed.com/1",
		Content: "c", PublishedAt: time.Now().UTC(),
	})
	store.Close()

	req := httptest.NewRequest("GET", "/api/fever?api_key="+feverApiKey+"&items", nil)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	// Buscamos el texto crudo: true/false en el JSON es lo que no queremos.
	cuerpo := rr.Body.String()
	for _, campo := range []string{"is_saved", "is_read"} {
		i := indexOf(cuerpo, `"`+campo+`":`)
		if i < 0 {
			t.Fatalf("no se encontro el campo %s en la respuesta: %s", campo, cuerpo)
		}
		resto := cuerpo[i+len(campo)+3:]
		valor := ""
		for j := 0; j < len(resto) && j < 6; j++ {
			if resto[j] == ',' || resto[j] == '}' {
				break
			}
			valor += string(resto[j])
		}
		if valor != "0" && valor != "1" {
			t.Errorf("%s = %q, se esperaba 0 o 1 (Fever no acepta true/false)", campo, valor)
		}
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

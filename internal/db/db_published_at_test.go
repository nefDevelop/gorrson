package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/nef734/gorrson/internal/models"
)

// TestPublishedAtSeGuardaEnFormatoQueSQLiteEntiende comprueba que
// published_at se escribe en un formato que date() reconoce.
//
// Si se guardara el String() del time.Time ("2026-10-01 08:45:22 +0000 UTC"),
// date() devolveria NULL y la poda no borraria nada. Este test falla si eso
// vuelve a pasar.
func TestPublishedAtSeGuardaEnFormatoQueSQLiteEntiende(t *testing.T) {
	dataDir := t.TempDir()
	s, err := NewStore(dataDir, "formato")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer s.Close()
	ctx := context.Background()

	if _, err := s.db.Exec(
		`INSERT INTO feeds (id, title, feed_url) VALUES ('f1', 'Feed', 'https://f.com/rss')`); err != nil {
		t.Fatalf("insertar feed: %v", err)
	}

	entrada := &models.Entry{
		ID:          "e1",
		FeedID:      "f1",
		Title:       "Titulo",
		URL:         "https://example.com/e1",
		Content:     "contenido",
		PublishedAt: time.Date(2026, 10, 1, 8, 45, 22, 0, time.UTC),
	}
	if err := s.AddEntry(ctx, entrada); err != nil {
		t.Fatalf("AddEntry: %v", err)
	}

	// Abrimos en crudo para ver lo que hay realmente en la columna.
	raw, err := sql.Open("sqlite", filepath.Join(dataDir, "formato.db"))
	if err != nil {
		t.Fatalf("abrir bd en crudo: %v", err)
	}
	defer raw.Close()

	var guardada string
	var fecha sql.NullString
	if err := raw.QueryRow(
		`SELECT CAST(published_at AS TEXT), date(published_at) FROM entries WHERE url = ?`,
		entrada.URL).Scan(&guardada, &fecha); err != nil {
		t.Fatalf("leer published_at: %v", err)
	}

	if len(guardada) != len("2006-01-02 15:04:05") {
		t.Errorf("published_at guardado como %q (%d caracteres); se esperaba solo fecha y hora, "+
			"sin el sufijo de zona horaria, porque date() no lo reconoce", guardada, len(guardada))
	}
	if !fecha.Valid {
		t.Errorf("date(published_at) devolvio NULL para %q: la poda por fecha no funcionara", guardada)
	}
	if want := "2026-10-01"; fecha.String != want {
		t.Errorf("date(published_at) = %q, se esperaba %q", fecha.String, want)
	}

	// Y debe poder releerse como time.Time sin perder la hora.
	var leida time.Time
	if err := raw.QueryRow(`SELECT published_at FROM entries WHERE url = ?`, entrada.URL).Scan(&leida); err != nil {
		t.Fatalf("Scan a time.Time: %v", err)
	}
	if !leida.Equal(entrada.PublishedAt) {
		t.Errorf("al releer se obtuvo %v, se esperaba %v (se ha perdido precision o zona horaria)",
			leida, entrada.PublishedAt)
	}
}

// TestMigrateNormalizaPublishedAtAntiguo comprueba que una base de datos creada
// con el formato viejo se normaliza al abrirla, para que la poda no se quede
// sin hacer nada silenciosamente.
func TestMigrateNormalizaPublishedAtAntiguo(t *testing.T) {
	dataDir := t.TempDir()
	user := "migra"

	// Dejamos la base con el formato antiguo, saltandonos el helper a proposito
	// para reproducir lo que habia en disco antes de este cambio.
	s, err := NewStore(dataDir, user)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if _, err := s.db.Exec(
		`INSERT INTO feeds (id, title, feed_url) VALUES ('f1', 'Feed', 'https://f.com/rss')`); err != nil {
		t.Fatalf("insertar feed: %v", err)
	}
	if _, err := s.db.Exec(
		`INSERT INTO entries (id, feed_id, title, url, published_at) VALUES (?,?,?,?,?)`,
		"vieja", "f1", "Vieja", "https://example.com/vieja",
		"2025-01-15 10:00:00 +0000 UTC"); err != nil {
		t.Fatalf("insertar fila antigua: %v", err)
	}
	var antes string
	if err := s.db.QueryRow(
		`SELECT CAST(published_at AS TEXT) FROM entries WHERE url = ?`,
		"https://example.com/vieja").Scan(&antes); err != nil {
		t.Fatalf("leer antes: %v", err)
	}
	if antes != "2025-01-15 10:00:00 +0000 UTC" {
		t.Fatalf("la prueba no reproduce el caso: la fila se guardo como %q", antes)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("cerrar: %v", err)
	}

	// Al reabrir el store debe correr la migracion.
	s2, err := NewStore(dataDir, user)
	if err != nil {
		t.Fatalf("NewStore segunda vez: %v", err)
	}
	defer s2.Close()

	var despues string
	var fecha sql.NullString
	if err := s2.db.QueryRow(
		`SELECT CAST(published_at AS TEXT), date(published_at) FROM entries WHERE url = ?`,
		"https://example.com/vieja").Scan(&despues, &fecha); err != nil {
		t.Fatalf("leer despues: %v", err)
	}

	if despues != "2025-01-15 10:00:00" {
		t.Errorf("la migracion dejo %q, se esperaba \"2025-01-15 10:00:00\"", despues)
	}
	if want := "2025-01-15"; fecha.String != want {
		t.Errorf("date(published_at) tras migrar = %q, se esperaba %q", fecha.String, want)
	}

	// Y debe ser idempotente: reabrir otra vez no debe alterar el valor.
	if err := s2.Close(); err != nil {
		t.Fatalf("cerrar segunda vez: %v", err)
	}
	s3, err := NewStore(dataDir, user)
	if err != nil {
		t.Fatalf("NewStore tercera vez: %v", err)
	}
	defer s3.Close()
	var otra string
	if err := s3.db.QueryRow(
		`SELECT CAST(published_at AS TEXT) FROM entries WHERE url = ?`,
		"https://example.com/vieja").Scan(&otra); err != nil {
		t.Fatalf("leer tercera vez: %v", err)
	}
	if otra != despues {
		t.Errorf("la migracion no es idempotente: %q -> %q", despues, otra)
	}
}

// TestCleanupUsaRetencionDe6Meses comprueba la ventana de retencion. Con la
// fecha de referencia, una entrada de 8 meses debe caer y una de 2 meses no.
func TestCleanupUsaRetencionDe6Meses(t *testing.T) {
	dataDir := t.TempDir()
	s, err := NewStore(dataDir, "retencion")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer s.Close()
	ctx := context.Background()

	if _, err := s.db.Exec(
		`INSERT INTO feeds (id, title, feed_url) VALUES ('f1', 'Feed', 'https://f.com/rss')`); err != nil {
		t.Fatalf("insertar feed: %v", err)
	}

	ahora := time.Now().UTC()
	casos := []struct {
		nombre      string
		antiguedad  time.Duration
		debeBorrarse bool
	}{
		{"8 meses", 8 * 30 * 24 * time.Hour, true},
		{"2 meses", 2 * 30 * 24 * time.Hour, false},
	}
	for i, c := range casos {
		e := &models.Entry{
			ID:          c.nombre,
			FeedID:      "f1",
			Title:       c.nombre,
			URL:         "https://example.com/" + c.nombre,
			PublishedAt: ahora.Add(-c.antiguedad),
		}
		if err := s.AddEntry(ctx, e); err != nil {
			t.Fatalf("AddEntry %s: %v", c.nombre, err)
		}
		_ = i
	}

	borradas, err := s.CleanupOldEntries(ctx)
	if err != nil {
		t.Fatalf("CleanupOldEntries: %v", err)
	}
	if borradas != 1 {
		t.Errorf("se borraron %d entradas, se esperaba 1 (solo la de 8 meses)", borradas)
	}

	restantes, err := s.GetEntryByID(ctx, "2 meses")
	if err != nil {
		t.Errorf("la entrada de 2 meses no deberia haberse borrado: %v", err)
	}
	if restantes == nil {
		t.Error("la entrada de 2 meses desaparecio")
	}
}

package db

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/nef734/gorrson/internal/models"
)

// feverIDTestEntries es el numero de entradas que se siembran para comprobar
// que la lista de sin leer no se trunca. El antiguo tope era 500.
const feverIDTestEntries = 620

// TestOperacionesPorFeverIDIgnoranCero comprueba que un fever_id de 0 (lo que
// devuelve Atoi cuando un cliente manda un id vacio o no numerico) no toca
// nada.
//
// Sin la guarda, "UPDATE ... WHERE fever_id = 0" marcaba como leidas o como
// favoritas TODAS las entradas con ese valor, de golpe. En produccion
// generateFeverID nunca produce 0, pero el vector es un cliente de la Fever API
// mandando un id invalido, que no es raro.
func TestOperacionesPorFeverIDIgnoranCero(t *testing.T) {
	dataDir := t.TempDir()
	s, err := NewStore(dataDir, "feverid")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer s.Close()
	ctx := context.Background()

	if _, err := s.db.Exec(
		`INSERT INTO feeds (id, title, feed_url) VALUES ('f1', 'Feed', 'https://f.com/rss')`); err != nil {
		t.Fatalf("insertar feed: %v", err)
	}
	// Tres entradas con fever_id = 0, el caso que disparaba el fallo.
	for _, id := range []string{"a", "b", "c"} {
		if err := s.AddEntry(ctx, &models.Entry{
			ID: id, FeedID: "f1", Title: "T" + id, URL: "https://f.com/" + id,
			Content: "c", PublishedAt: time.Now().UTC(),
		}); err != nil {
			t.Fatalf("AddEntry %s: %v", id, err)
		}
		// Forzamos fever_id = 0 a proposito, que es lo que haria un id invalido
		// en una entrada sin id asignado.
		if _, err := s.db.Exec(`UPDATE entries SET fever_id = 0 WHERE id = ?`, id); err != nil {
			t.Fatalf("forzar fever_id=0 en %s: %v", id, err)
		}

	}

	casos := []struct {
		nombre string
		llamar func() error
	}{
		{"ToggleFavorite(true)", func() error { return s.ToggleFavoriteByFeverID(ctx, 0, true) }},
		{"ToggleFavorite(false)", func() error { return s.ToggleFavoriteByFeverID(ctx, 0, false) }},
		{"MarkAsRead", func() error { return s.MarkAsReadByFeverID(ctx, 0) }},
		{"ToggleFavorite(negativo)", func() error { return s.ToggleFavoriteByFeverID(ctx, -5, true) }},
		{"MarkAsRead(negativo)", func() error { return s.MarkAsReadByFeverID(ctx, -5) }},
	}
	for _, c := range casos {
		if err := c.llamar(); err != nil {
			t.Errorf("%s devolvio error inesperado: %v", c.nombre, err)
		}
		var fav, read int
		if err := s.db.QueryRow(
			`SELECT COALESCE(SUM(is_favorite),0), COALESCE(SUM(is_read),0) FROM entries`).Scan(&fav, &read); err != nil {
			t.Fatalf("leer estado: %v", err)
		}
		if fav != 0 || read != 0 {
			t.Errorf("%s toco las entradas: %d favoritas y %d leidas; se esperaba 0 y 0",
				c.nombre, fav, read)
		}
	}
}

// TestOperacionesPorFeverIDFuncionanConIdValido comprueba que la guarda no ha
// roto el camino bueno: un fever_id real (> 0) tiene que seguir funcionando.
func TestOperacionesPorFeverIDFuncionanConIdValido(t *testing.T) {
	dataDir := t.TempDir()
	s, err := NewStore(dataDir, "feverok")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer s.Close()
	ctx := context.Background()

	if _, err := s.db.Exec(
		`INSERT INTO feeds (id, title, feed_url) VALUES ('f1', 'Feed', 'https://f.com/rss')`); err != nil {
		t.Fatalf("insertar feed: %v", err)
	}
	if err := s.AddEntry(ctx, &models.Entry{
		ID: "e1", FeedID: "f1", Title: "T", URL: "https://f.com/e1",
		Content: "c", PublishedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	const feverID = 4242
	if _, err := s.db.Exec(`UPDATE entries SET fever_id = ? WHERE id = 'e1'`, feverID); err != nil {
		t.Fatalf("asignar fever_id: %v", err)
	}

	if err := s.ToggleFavoriteByFeverID(ctx, feverID, true); err != nil {
		t.Fatalf("ToggleFavorite: %v", err)
	}
	if err := s.MarkAsReadByFeverID(ctx, feverID); err != nil {
		t.Fatalf("MarkAsRead: %v", err)
	}
	var fav, read int
	s.db.QueryRow(`SELECT is_favorite, is_read FROM entries WHERE id = 'e1'`).Scan(&fav, &read)
	if fav != 1 {
		t.Errorf("is_favorite = %d, se esperaba 1 con un fever_id valido", fav)
	}
	if read != 1 {
		t.Errorf("is_read = %d, se esperaba 1 con un fever_id valido", read)
	}

	// Y desmarcar sigue quitando is_favorite pero conservando
	// was_ever_favorite, que es lo que protege al favorito de la poda.
	if err := s.ToggleFavoriteByFeverID(ctx, feverID, false); err != nil {
		t.Fatalf("ToggleFavorite(false): %v", err)
	}
	var isFav, everFav int
	s.db.QueryRow(`SELECT is_favorite, was_ever_favorite FROM entries WHERE id = 'e1'`).Scan(&isFav, &everFav)
	if isFav != 0 {
		t.Errorf("tras desmarcar is_favorite = %d, se esperaba 0", isFav)
	}
	if everFav != 1 {
		t.Errorf("tras desmarcar was_ever_favorite = %d, se esperaba 1 (debe sobrevivir a la poda)", everFav)
	}
}

// TestUnreadFeverIDsDevuelveTodasLasSinLeer comprueba que la lista de ids sin
// leer no se trunca. La Fever API espera la lista completa: si no, el cliente
// pierde el recuento y sus marcas de leido no cuadran.
func TestUnreadFeverIDsDevuelveTodasLasSinLeer(t *testing.T) {
	dataDir := t.TempDir()
	s, err := NewStore(dataDir, "ids")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer s.Close()
	ctx := context.Background()

	if _, err := s.db.Exec(
		`INSERT INTO feeds (id, title, feed_url) VALUES ('f1', 'Feed', 'https://f.com/rss')`); err != nil {
		t.Fatalf("insertar feed: %v", err)
	}

	// Mas entradas que el antiguo tope de 500, para que el test falle si
	// alguien vuelve a limitarlo.
	const n = feverIDTestEntries
	for i := 0; i < n; i++ {
		id := "e" + strconv.Itoa(i)
		if err := s.AddEntry(ctx, &models.Entry{
			ID: id, FeedID: "f1", Title: "T" + id,
			URL: "https://f.com/" + id, Content: "c",
			PublishedAt: time.Now().UTC(),
		}); err != nil {
			t.Fatalf("AddEntry %d: %v", i, err)
		}
		if _, err := s.db.Exec(`UPDATE entries SET fever_id = ? WHERE id = ?`, 1000+i, id); err != nil {
			t.Fatalf("asignar fever_id: %v", err)
		}
	}
	// Una de ellas se marca como leida: no debe aparecer.
	if err := s.MarkAsRead(ctx, "e7"); err != nil {
		t.Fatalf("MarkAsRead: %v", err)
	}

	ids, err := s.UnreadFeverIDs(ctx)
	if err != nil {
		t.Fatalf("UnreadFeverIDs: %v", err)
	}
	if len(ids) != n-1 {
		t.Errorf("se devolvieron %d ids sin leer, se esperaban %d (la entrada leida debe excluirse)",
			len(ids), n-1)
	}

	// Ninguno puede ser 0, que es justo el valor problematico.
	for _, id := range ids {
		if id <= 0 {
			t.Errorf("UnreadFeverIDs devolvio un id invalido: %d", id)
			break
		}
	}

	total, err := s.TotalEntries(ctx)
	if err != nil {
		t.Fatalf("TotalEntries: %v", err)
	}
	if total != n {
		t.Errorf("TotalEntries = %d, se esperaban %d", total, n)
	}
}

package db

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/nef734/gorrson/internal/models"
)

// TestQueryPlansUseIndexes es el guard que impide que ListEntries vuelva al
// escaneo completo.
//
// El fallo que corrige esto era silencioso y muy caro: el ORDER BY era
// `is_priority DESC, published_at DESC` y el único índice era
// (published_at DESC). En SQLite un índice solo cubre un ORDER BY si los términos
// iniciales coinciden, así que el plan caía a "SCAN e" + "USE TEMP B-TREE FOR
// ORDER BY": leer la tabla entera y ordenarla en memoria para devolver 50 filas.
// Medido con 100.000 entradas, 43 ms por petición de inbox y 1,19 s para /feeds.
//
// Comprobar el plan y no el tiempo es deliberado: el tiempo depende de la máquina,
// el plan no. Y no se llama a ListEntries, sino a la consulta con su forma exacta,
// porque lo que importa es el SQL que se envía.
func TestQueryPlansUseIndexes(t *testing.T) {
	if testing.Short() {
		t.Skip("requiere filas suficientes para que el planificador elija índice")
	}
	dir := t.TempDir()
	s, err := NewStore(dir, "plan_user")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	var feedIDs []string
	for i := 0; i < 6; i++ {
		f := &models.Feed{ID: uuid.New().String(), Title: fmt.Sprintf("Feed %d", i),
			FeedURL: fmt.Sprintf("https://example.invalid/%d", i), Category: "Tech"}
		if err := s.AddFeed(ctx, f); err != nil {
			t.Fatalf("AddFeed: %v", err)
		}
		feedIDs = append(feedIDs, f.ID)
	}

	// Suficientes filas para que el planificador Compare usar índice frente a
	// escaneo: con una tabla de 3 filas elige escaneo aunque los índices existan y
	// el test daría un falso verde.
	const n = 4000
	for i := 0; i < n; i++ {
		e := &models.Entry{
			ID:         uuid.New().String(),
			FeedID:     feedIDs[i%len(feedIDs)],
			Title:      fmt.Sprintf("Entrada %d", i),
			URL:        fmt.Sprintf("https://example.invalid/e/%d", i),
			IsRead:     i%3 == 0,
			IsFavorite: i%7 == 0,
		}
		if err := s.AddEntry(ctx, e); err != nil {
			t.Fatalf("AddEntry: %v", err)
		}
	}
	// ANALYZE para que el planificador tenga estadísticas; sin ellas puede elegir
	// escaneo por defecto y el test mentaría.
	if _, err := s.db.ExecContext(ctx, "ANALYZE"); err != nil {
		t.Fatalf("ANALYZE: %v", err)
	}

	const cols = `e.id, e.fever_id, e.feed_id, e.title, e.summary, e.content, e.url,
		e.image_url, e.published_at, e.is_read, e.is_favorite, e.is_full_content,
		e.is_priority, e.matched_action, f.title, f.category`
	const base = ` FROM entries e JOIN feeds f ON e.feed_id = f.id`

	cases := []struct {
		name    string
		query   string
		wantIdx string // índice que el plan debe usar
		args    []any
		forbid  []string // fragmentos que no deben aparecer en el plan
	}{
		{
			name: "inbox: unread + más recientes (la ruta más caliente)",
			query: "SELECT " + cols + base +
				" WHERE e.is_read = 0 ORDER BY e.is_priority DESC, e.published_at DESC LIMIT 50",
			wantIdx: "idx_entries_unread",
			forbid:  []string{"SCAN e", "USE TEMP B-TREE"},
		},
		{
			name: "listado sin filtros",
			query: "SELECT " + cols + base +
				" ORDER BY e.is_priority DESC, e.published_at DESC LIMIT 50",
			wantIdx: "idx_entries_timeline",
			// Un SCAN sobre el ÍNDICE es correcto aquí: se recorre en orden y para
			// a los 50. Lo que no puede aparecer es un escaneo de la TABLA.
			forbid: []string{"USE TEMP B-TREE", "SCAN e entries"},
		},
		{
			name: "favoritos",
			query: "SELECT " + cols + base +
				" WHERE e.is_favorite = 1 ORDER BY e.is_priority DESC, e.published_at DESC LIMIT 50",
			wantIdx: "idx_entries_favorites",
			forbid:  []string{"SCAN e", "USE TEMP B-TREE"},
		},
		{
			name: "más antiguos: el índice ASC es aparte, no se puede leer al revés",
			query: "SELECT " + cols + base +
				" WHERE e.is_read = 0 ORDER BY e.is_priority DESC, e.published_at ASC LIMIT 50",
			wantIdx: "idx_entries_unread_asc",
			forbid:  []string{"SCAN e", "USE TEMP B-TREE"},
		},
		{
			name: "vista de feed",
			query: "SELECT " + cols + base +
				" WHERE e.feed_id = ? AND e.is_read = 0 ORDER BY e.is_priority DESC, e.published_at DESC LIMIT 50",
			wantIdx: "idx_entries_feed_read",
			args:    []any{feedIDs[0]},
			forbid:  []string{"SCAN e", "USE TEMP B-TREE"},
		},
		{
			name:    "/feeds: conteo de no leídos por feed",
			query:   `SELECT f.id, f.title, (SELECT COUNT(*) FROM entries e WHERE e.feed_id = f.id AND e.is_read = 0) FROM feeds f`,
			wantIdx: "idx_entries_feed_read",
			forbid:  []string{"SCAN e"},
		},
		{
			// Por qué idx_entries_published sigue existiendo: hay consultas que
			// ordenan por published_at sin is_priority delante, y los índices de
			// listado (todos con is_priority) no las cubren.
			name:    "consulta que ordena por published_at en solitario",
			query:   `SELECT id FROM entries ORDER BY published_at DESC LIMIT 50`,
			wantIdx: "idx_entries_published",
			forbid:  []string{"USE TEMP B-TREE"},
		},
		{
			// MarkAllRead con búsqueda: aquí el plan lo conduce entries_fts, no
			// entries, así que ningún índice de entries puede eliminar el orden.
			// Se fija el plan para que quede documentado que el temp b-tree es
			// inherente a la búsqueda FTS y no una regresión de los índices.
			name:    "MarkAllRead con búsqueda: el plan lo conduce FTS5",
			query:   `SELECT entries.id FROM entries JOIN entries_fts ON entries.rowid = entries_fts.rowid WHERE entries_fts MATCH ? ORDER BY entries.published_at DESC LIMIT 50`,
			wantIdx: "",
			args:    []any{"Entrada"},
			forbid:  []string{"SCAN entries\n"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := queryPlan(t, s, tc.query, tc.args...)

			for _, bad := range tc.forbid {
				if strings.Contains(plan, bad) {
					t.Errorf("el plan usa %q, que es lo que costaba 43 ms a 1.193 ms "+
						"por petición.\nPlan:\n%s", bad, plan)
				}
			}
			if tc.wantIdx != "" && !strings.Contains(plan, tc.wantIdx) {
				t.Errorf("el plan no usa el índice %s.\nPlan:\n%s", tc.wantIdx, plan)
			}
			t.Logf("plan:\n%s", plan)
		})
	}
}

// queryPlan devuelve el EXPLAIN QUERY PLAN como texto.
func queryPlan(t *testing.T, s *Store, query string, args ...any) string {
	t.Helper()
	rows, err := s.db.Query("EXPLAIN QUERY PLAN "+query, args...)
	if err != nil {
		t.Fatalf("EXPLAIN QUERY PLAN: %v", err)
	}
	defer rows.Close()

	var sb strings.Builder
	for rows.Next() {
		var id, parent, notused int
		var detail string
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			t.Fatalf("scan plan: %v", err)
		}
		sb.WriteString(detail)
		sb.WriteByte('\n')
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return sb.String()
}

// TestIndicesSeCreanAlAbrir verifica que los índices de listado existen tras
// NewStore, y no solo tras un CREATE manual: la migración es idempotente y se
// ejecuta en cada arranque.
func TestIndicesSeCreanAlAbrir(t *testing.T) {
	dir := t.TempDir()
	defer os.RemoveAll(dir)

	s, err := NewStore(dir, "idx_user")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer s.Close()

	want := []string{
		"idx_entries_published",
		"idx_entries_timeline",
		"idx_entries_unread",
		"idx_entries_unread_asc",
		"idx_entries_favorites",
		"idx_entries_feed_read",
	}
	rows, err := s.db.Query(`SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='entries'`)
	if err != nil {
		t.Fatalf("sqlite_master: %v", err)
	}
	defer rows.Close()

	got := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got[name] = true
	}
	for _, w := range want {
		if !got[w] {
			t.Errorf("falta el índice %s tras NewStore", w)
		}
	}
}

// TestGetStatsAgrupaPorCategoriaYNoPorFeed fija la semántica de GetStats.
//
// Existe porque la optimización de /stats (docs/AUDIT.md §1.3) agrupa primero por
// feed para que el barrido del índice salga en streaming, y luego tiene que volver
// a agrupar por categoría. Una versión anterior de esa consulta agrupaba solo por
// feed: una categoría con 5 feeds devolvía 5 filas en vez de 1, y el total
// seguía cuadrando, así que un simple "las cifras suman lo mismo" no lo detecta.
func TestGetStatsAgrupaPorCategoriaYNoPorFeed(t *testing.T) {
	if testing.Short() {
		t.Skip("genera entradas en varias categorías")
	}
	dir := t.TempDir()
	s, err := NewStore(dir, "stats_user")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	// Dos feeds en la misma categoría y uno en otra: el caso que distingue un
	// GROUP BY por categoría de uno por feed.
	feeds := []struct{ title, category string }{
		{"A", "Tech"}, {"B", "Tech"}, {"C", "News"},
	}
	ids := make([]string, 0, len(feeds))
	for i, f := range feeds {
		fd := &models.Feed{ID: uuid.New().String(), Title: f.title,
			FeedURL: fmt.Sprintf("https://example.invalid/%d", i), Category: f.category}
		if err := s.AddFeed(ctx, fd); err != nil {
			t.Fatalf("AddFeed: %v", err)
		}
		ids = append(ids, fd.ID)
	}
	// 10 entradas en Tech (5+5) y 3 en News.
	for i := 0; i < 10; i++ {
		feed := ids[0]
		if i >= 5 {
			feed = ids[1]
		}
		if err := s.AddEntry(ctx, &models.Entry{
			ID: uuid.New().String(), FeedID: feed,
			Title: fmt.Sprintf("E%d", i), URL: fmt.Sprintf("https://example.invalid/e/%d", i),
		}); err != nil {
			t.Fatalf("AddEntry: %v", err)
		}
	}
	for i := 0; i < 3; i++ {
		if err := s.AddEntry(ctx, &models.Entry{
			ID: uuid.New().String(), FeedID: ids[2],
			Title: fmt.Sprintf("N%d", i), URL: fmt.Sprintf("https://example.invalid/n/%d", i),
		}); err != nil {
			t.Fatalf("AddEntry: %v", err)
		}
	}

	stats, err := s.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}

	// Ninguna categoría puede repetirse.
	vistos := map[string]int{}
	total := 0
	for _, c := range stats.CategoryStats {
		if c.Name == "" {
			t.Errorf("categoría con nombre vacío: %+v", c)
		}
		vistos[c.Name]++
		if vistos[c.Name] > 1 {
			t.Errorf("la categoría %q aparece %d veces; se está agrupando por feed, no por categoría",
				c.Name, vistos[c.Name])
		}
		total += c.Count
	}

	if len(stats.CategoryStats) != 2 {
		t.Errorf("categorías distintas = %d; want 2 (Tech y News)", len(stats.CategoryStats))
	}
	for _, want := range []struct {
		name  string
		count int
	}{{"Tech", 10}, {"News", 3}} {
		if got := vistos[want.name]; got != 1 {
			t.Errorf("categoría %q aparece %d veces; want 1", want.name, got)
		}
	}
	if total != 13 {
		t.Errorf("suma de categorías = %d; want 13 (10 Tech + 3 News)", total)
	}
}

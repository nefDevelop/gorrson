// seed — base de datos de pruebas para desarrollo y rendimiento.
//
// Uso:
//
//	go run ./scripts/seed                      # usuario + feeds (comportamiento original)
//	go run ./scripts/seed -n 100000            # además 100.000 entradas con índice FTS5
//
// Los valores por defecto reproducen el seed original (solo usuario + feeds) para
// no alterar `make db-recreate`. El escalado de entradas es lo que usa
// `make perf-seed SEED_N=...`.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/nef734/gorrson/internal/auth"
	"github.com/nef734/gorrson/internal/db"
	"github.com/nef734/gorrson/internal/models"

	_ "modernc.org/sqlite"
)

var (
	flagDataDir = flag.String("data-dir", "./data", "Directorio de bases de datos por usuario")
	flagUser    = flag.String("user", "user", "Usuario a crear")
	flagPass    = flag.String("pass", "user", "Contraseña del usuario")
	flagEntries = flag.Int("n", 0, "Número de entradas a generar (0 = solo feeds, como el seed original)")
	flagBatch   = flag.Int("batch", 2000, "Filas por transacción durante el sembrado masivo")
)

func main() {
	flag.Parse()

	dataDir := *flagDataDir
	username := *flagUser
	password := *flagPass

	// 1. Clean up existing DB
	dbPath := filepath.Join(dataDir, username+".db")
	_ = os.Remove(dbPath)
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")
	fmt.Printf("Cleaning up %s...\n", dbPath)

	// 2. Initialize Store (crea el esquema)
	store, err := db.NewStore(dataDir, username)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Create User
	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Fatal(err)
	}
	if err := store.UpsertUser(context.Background(), username, hash, ""); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("User '%s' created\n", username)

	// 4. Seed Feeds
	fmt.Printf("Seeding %d feeds...\n", len(feedSeeds))
	for _, s := range feedSeeds {
		f := &models.Feed{
			ID:       uuid.New().String(),
			Title:    s.Title,
			FeedURL:  s.URL,
			Category: s.Category,
		}
		if err := store.AddFeed(context.Background(), f); err != nil {
			fmt.Printf("  [Error] %s: %v\n", s.Title, err)
		} else {
			fmt.Printf("  [OK] %s (%s)\n", s.Title, s.Category)
		}
	}

	// AddFeed no devuelve el ID generado, así que se releen los feeds para poder
	// asociarles entradas. Sin esto las entradas quedarían sin feed_id y el JOIN de
	// ListEntries y SearchEntries las descartaría: el seed parecería correcto y las
	// consultas no.
	feedRows, err := store.ListFeeds(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	if len(feedRows) != len(feedSeeds) {
		log.Fatalf("se esperaban %d feeds y hay %d", len(feedSeeds), len(feedRows))
	}

	// El Store se cierra antes del sembrado masivo: se abre un handle propio para
	// poder agrupar miles de inserts en transacciones. AddEntry va fila a fila
	// (una transacción por fila), lo que a escala 1M domina el tiempo total.
	if err := store.Close(); err != nil {
		log.Fatal(err)
	}

	// 5. Seed Entries (sólo si se piden)
	if *flagEntries > 0 {
		t0 := time.Now()
		n, err := seedEntries(dbPath, feedRows, *flagEntries, *flagBatch)
		if err != nil {
			log.Fatalf("sembrando entradas: %v", err)
		}
		fmt.Printf("  • %d entradas generadas (%.1fs)\n", n, time.Since(t0).Seconds())
	}

	printSummary(dbPath)
	fmt.Println("\nSeed complete! Run 'make run' to start sync.")
}

// seedEntries genera n entradas repartidas entre los feeds.
//
// Se escribe directamente sobre el fichero (y no con Store.AddEntry) por dos
// razones: AddEntry abre una transacción por fila, y Store no expone su *sql.DB.
// El trigger entries_ai de SQLite puebla el índice FTS5 solo, así que la búsqueda
// funciona igual que con el sync real.
func seedEntries(dbPath string, feeds []models.Feed, n, batch int) (int, error) {
	dsn := "file:" + dbPath + "?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)"
	DB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return 0, err
	}
	defer DB.Close()

	// Restricciones de WAL: un solo escritor. Se fuerzan para no obtener
	// SQLITE_BUSY intermitente que arruinaría la medición.
	DB.SetMaxOpenConns(1)

	inserted := 0
	for start := 0; start < n; start += batch {
		end := start + batch
		if end > n {
			end = n
		}
		tx, err := DB.Begin()
		if err != nil {
			return inserted, err
		}
		txStmt, err := tx.Prepare(`
			INSERT INTO entries (id, feed_id, title, summary, content, url, published_at, is_read, is_favorite, is_full_content)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, 0)
			ON CONFLICT(url) DO NOTHING`)
		if err != nil {
			_ = tx.Rollback()
			return inserted, err
		}
		for i := start; i < end; i++ {
			e := makeEntry(i, feeds[i%len(feeds)])
			if _, err := txStmt.Exec(e.ID, e.FeedID, e.Title, e.Summary, e.Content, e.URL, e.PublishedAt, e.IsRead); err != nil {
				txStmt.Close()
				_ = tx.Rollback()
				return inserted, err
			}
			inserted++
		}
		txStmt.Close()
		if err := tx.Commit(); err != nil {
			return inserted, err
		}
		if batch > 0 && (end/batch)%10 == 0 {
			fmt.Printf("    %d/%d entradas\n", end, n)
		}
	}
	return inserted, nil
}

// Títulos y contenido con vocabulario variety: la búsqueda FTS5 solo mide algo si
// los términos existen. 'gorrson' y 'benchmark' están garantizados en todos para
// que cualquier test pueda usarlos como término de búsqueda.
func makeEntry(i int, feed models.Feed) models.Entry {
	titulo := fmt.Sprintf("%s: análisis de %s en %s (%d)",
		feed.Title, topics[i%len(topics)], adjectives[(i/len(topics))%len(adjectives)], i)

	contenido := fmt.Sprintf(
		"Artículo de prueba generado para el benchmark de gorrson. Índice %d del feed %s. "+
			"Tema: %s. Este texto existe para que el índice FTS5 tenga volumen real y la "+
			"búsqueda no mida el caso degenerado de una tabla vacía. gorrson benchmark "+
			"palabra clave estable para search tests. doc%08d token único por entrada. %s",
		i, feed.Title, topics[i%len(topics)], i, filler[i%len(filler)])

	resumen := fmt.Sprintf("Resumen del artículo %d: %s. %s.", i, topics[i%len(topics)], adjectives[i%len(adjectives)])

	return models.Entry{
		ID:      uuid.New().String(),
		FeedID:  feed.ID,
		Title:   titulo,
		Summary: resumen,
		Content: contenido,
		URL:     fmt.Sprintf("https://example.invalid/%s/%d", feed.Title, i),
		// Repartir en 30 días hacia atrás: da profundidad real a
		// idx_entries_published (ORDER BY published_at DESC).
		PublishedAt: time.Now().Add(-time.Duration(i%2592000) * time.Second),
		IsRead:      i%3 == 0,
	}
}

// printSummary informa del volumen real y del tamaño en disco: sin esto no se
// puede atribuir un resultado de carga al dataset que lo produjo.
func printSummary(dbPath string) {
	fmt.Println()
	DB, err := sql.Open("sqlite", "file:"+dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  (no se pudo abrir la DB para el resumen: %v)\n", err)
		return
	}
	defer DB.Close()

	for _, t := range []string{"users", "feeds", "entries", "filters", "api_keys"} {
		var n int
		if err := DB.QueryRow("SELECT COUNT(*) FROM " + t).Scan(&n); err != nil {
			continue
		}
		fmt.Printf("    %-12s %d filas\n", t, n)
	}
	// El FTS5 es una tabla externa (content='entries'): el conteo sale de entries,
	// pero el tamaño del índice es la mitad del coste de lectura.
	if fi, err := os.Stat(dbPath); err == nil {
		fmt.Printf("    %-12s %.1f MiB\n", "db", float64(fi.Size())/(1024*1024))
	}
}

var topics = []string{
	"rendimiento", "concurrencia", "SQLite", "índices", "caché",
	"latencia", "throughput", "FTS5", "WAL", "observabilidad",
	"métricas", "profiling", "Go", "contenedores", "seguridad",
	"criptografía", "redes", "kernel", "compiladores", "algoritmos",
}

var adjectives = []string{
	"detallado", "práctico", "exhaustivo", "breve", "técnico",
	"didáctico", "actualizado", "comparativo", "profundo", "iniciativo",
}

var filler = []string{
	"Conclusiones y referencias cruzadas entre los distintos apartados.",
	"Párrafo adicional sin información relevante para la búsqueda.",
	"Ejemplo práctico con fragmentos de código que no afectan al índice.",
	"Nota al pie sobre los límites del sistema.",
	"Referencia bibliográfica externa.",
}

// Corpus de feeds reales: 40 fuentes variadas (tecnología, news, gaming,
// ciencia, cultura, deportes, negocios) para que category y feed_id filten
// contra algo realista en vez de una única categoría.
var feedSeeds = []struct {
	Title    string
	URL      string
	Category string
}{
	// TECHNOLOGY
	{"Hacker News", "https://news.ycombinator.com/rss", "Tech"},
	{"TechCrunch", "https://techcrunch.com/feed/", "Tech"},
	{"The Verge", "https://www.theverge.com/rss/index.xml", "Tech"},
	{"Wired", "https://www.wired.com/feed/rss", "Tech"},
	{"Ars Technica", "https://feeds.arstechnica.com/arstechnica/index", "Tech"},
	{"Engadget", "https://www.engadget.com/rss.xml", "Tech"},
	{"Slashdot", "http://rss.slashdot.org/Slashdot/slashdotMain", "Tech"},
	{"GitHub Blog", "https://github.blog/feed/", "Tech"},
	{"Google News Tech", "https://news.google.com/rss/search?q=technology&hl=en-US&gl=US&ceid=US:en", "Tech"},
	{"Genbeta", "https://www.genbeta.com/feedburner.xml", "Tech"},
	{"Xataka", "https://www.xataka.com/feedburner.xml", "Tech"},

	// NEWS & WORLD
	{"BBC News", "https://feeds.bbci.co.uk/news/rss.xml", "News"},
	{"Reuters World", "https://www.reutersagency.com/feed/?best-topics=world-news&post_type=best", "News"},
	{"AP News", "https://apnews.com/hub/world-news.rss", "News"},
	{"NYT World", "https://rss.nytimes.com/services/xml/rss/nyt/World.xml", "News"},
	{"The Guardian World", "https://www.theguardian.com/world/rss", "News"},
	{"Al Jazeera", "https://www.aljazeera.com/xml/rss/all.xml", "News"},
	{"El País", "https://feeds.elpais.com/mrss-s/pages/ep/site/elpais.com/portada", "News"},
	{"El Mundo", "https://e00-elmundo.uecdn.es/elmundo/rss/portada.xml", "News"},
	{"RTVE Noticias", "https://www.rtve.es/api/noticias.rss", "News"},

	// GAMING
	{"AnaitGames", "https://www.anaitgames.com/rss", "Gaming"},
	{"Eurogamer", "https://www.eurogamer.net/feed/news", "Gaming"},
	{"Kotaku", "https://kotaku.com/rss", "Gaming"},
	{"Polygon", "https://www.polygon.com/rss/index.xml", "Gaming"},
	{"IGN News", "https://feeds.feedburner.com/ign/news", "Gaming"},
	{"GameSpot", "https://www.gamespot.com/feeds/news/", "Gaming"},
	{"PC Gamer", "https://www.pcgamer.com/rss/", "Gaming"},
	{"Nintendo Life", "https://www.nintendolife.com/feeds/latest", "Gaming"},
	{"Vandal", "https://vandal.elespanol.com/xml/rss.xml", "Gaming"},

	// SCIENCE & CULTURE
	{"Nature", "https://www.nature.com/nature.rss", "Science"},
	{"Science Daily", "https://www.sciencedaily.com/rss/all.xml", "Science"},
	{"Scientific American", "https://www.scientificamerican.com/feed/", "Science"},
	{"National Geographic", "https://www.nationalgeographic.com/rss/all.xml", "Science"},
	{"The Atlantic", "https://www.theatlantic.com/feed/all/", "Culture"},
	{"New Yorker", "https://www.newyorker.com/feed/everything", "Culture"},
	{"Hyperallergic", "https://hyperallergic.com/feed/", "Art"},

	// SPORTS & BUSINESS
	{"ESPN", "https://www.espn.com/espn/rss/news", "Sports"},
	{"BBC Sport", "https://feeds.bbci.co.uk/sport/rss.xml", "Sports"},
	{"Marca", "https://e00-marca.uecdn.es/rss/portada.xml", "Sports"},
	{"AS.com", "https://as.com/rss/tags/portada.xml", "Sports"},
	{"Wall Street Journal", "https://feeds.a.dj.com/rss/RSSWorldNews.xml", "Business"},
	{"Financial Times", "https://www.ft.com/?format=rss", "Business"},
}

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/nef734/gorrson/internal/auth"
	"github.com/nef734/gorrson/internal/db"
	"github.com/nef734/gorrson/internal/models"
)

func main() {
	dataDir := "./data"
	username := "user"
	password := "user"

	// 1. Clean up existing DB
	dbPath := filepath.Join(dataDir, username+".db")
	_ = os.Remove(dbPath)
	fmt.Printf("Cleaning up %s...\n", dbPath)

	// 2. Initialize Store
	store, err := db.NewStore(dataDir, username)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	// 3. Create User
	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Fatal(err)
	}
	err = store.UpsertUser(username, hash, "")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("User '%s' created with password '%s'\n", username, password)

	// 4. Seed Feeds
	seeds := []struct {
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
		{"Vandal", "https://vandal.elespanol.com/noticias.xml", "Gaming"},

		// SCIENCE & CULTURE
		{"Nature", "https://www.nature.com/nature.rss", "Science"},
		{"Science Daily", "https://www.sciencedaily.com/rss/all.xml", "Science"},
		{"Scientific American", "https://www.scientificamerican.com/section/reuters/rss/", "Science"},
		{"National Geographic", "https://www.nationalgeographic.com/rss/index.xml", "Science"},
		{"The Atlantic", "https://www.theatlantic.com/feed/all/", "Culture"},
		{"New Yorker", "https://www.newyorker.com/feed/everything", "Culture"},
		{"Hyperallergic", "https://hyperallergic.com/feed/", "Art"},

		// SPORTS & BUSINESS
		{"ESPN", "https://www.espn.com/espn/rss/news", "Sports"},
		{"BBC Sport", "https://feeds.bbci.co.uk/sport/rss.xml", "Sports"},
		{"Marca", "https://e00-marca.uecdn.es/rss/portada.xml", "Sports"},
		{"AS.com", "https://as.com/rss/portada.xml", "Sports"},
		{"Wall Street Journal", "https://feeds.a.dj.com/rss/RSSWorldNews.xml", "Business"},
		{"Financial Times", "https://www.ft.com/?format=rss", "Business"},
	}

	fmt.Printf("Seeding %d feeds...\n", len(seeds))
	for _, s := range seeds {
		f := &models.Feed{
			ID:       uuid.New().String(),
			Title:    s.Title,
			FeedURL:  s.URL,
			Category: s.Category,
		}
		if err := store.AddFeed(f); err != nil {
			fmt.Printf("  [Error] %s: %v\n", s.Title, err)
		} else {
			fmt.Printf("  [OK] %s (%s)\n", s.Title, s.Category)
		}
	}

	fmt.Println("\nSeed complete! Run 'make run' to start sync.")
}

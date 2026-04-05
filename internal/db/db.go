package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type Feed struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	SiteURL     string    `json:"site_url"`
	FeedURL     string    `json:"feed_url"`
	Category    string    `json:"category"`
	LastUpdated time.Time `json:"last_updated"`
}

type Entry struct {
	ID          string    `json:"id"`
	FeedID      string    `json:"feed_id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	URL         string    `json:"url"`
	PublishedAt time.Time `json:"published_at"`
	IsRead      bool      `json:"is_read"`
}

func NewStore(dataDir, username string) (*Store, error) {
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, fmt.Errorf("could not create data directory: %w", err)
		}
	}
	dbPath := filepath.Join(dataDir, username+".db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS feeds (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			site_url TEXT,
			feed_url TEXT UNIQUE NOT NULL,
			category TEXT,
			last_updated DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS users (
			username TEXT PRIMARY KEY,
			password_hash TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS entries (
			id TEXT PRIMARY KEY,
			feed_id TEXT NOT NULL,
			title TEXT NOT NULL,
			content TEXT,
			url TEXT UNIQUE NOT NULL,
			published_at DATETIME,
			is_read BOOLEAN DEFAULT 0,
			FOREIGN KEY(feed_id) REFERENCES feeds(id) ON DELETE CASCADE
		);`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS entries_fts USING fts5(
			title, content, content='entries', content_rowid='rowid'
		);`,
		// Triggers to keep FTS in sync - SQLite FTS5 with external content
		`CREATE TRIGGER IF NOT EXISTS entries_ai AFTER INSERT ON entries BEGIN
			INSERT INTO entries_fts(rowid, title, content) VALUES (new.rowid, new.title, new.content);
		END;`,
		`CREATE TRIGGER IF NOT EXISTS entries_ad AFTER DELETE ON entries BEGIN
			INSERT INTO entries_fts(entries_fts, rowid, title, content) VALUES('delete', old.rowid, old.title, old.content);
		END;`,
		`CREATE TRIGGER IF NOT EXISTS entries_au AFTER UPDATE ON entries BEGIN
			INSERT INTO entries_fts(entries_fts, rowid, title, content) VALUES('delete', old.rowid, old.title, old.content);
			INSERT INTO entries_fts(rowid, title, content) VALUES (new.rowid, new.title, new.content);
		END;`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) AddFeed(f *Feed) error {
	_, err := s.db.Exec(`INSERT INTO feeds (id, title, site_url, feed_url, category, last_updated) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(feed_url) DO UPDATE SET last_updated=excluded.last_updated`,
		f.ID, f.Title, f.SiteURL, f.FeedURL, f.Category, f.LastUpdated)
	return err
}

func (s *Store) AddEntry(e *Entry) error {
	_, err := s.db.Exec(`INSERT INTO entries (id, feed_id, title, content, url, published_at, is_read) VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT(url) DO NOTHING`,
		e.ID, e.FeedID, e.Title, e.Content, e.URL, e.PublishedAt, e.IsRead)
	return err
}

func (s *Store) ListFeeds() ([]Feed, error) {
	rows, err := s.db.Query("SELECT id, title, site_url, feed_url, category, last_updated FROM feeds")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []Feed
	for rows.Next() {
		var f Feed
		if err := rows.Scan(&f.ID, &f.Title, &f.SiteURL, &f.FeedURL, &f.Category, &f.LastUpdated); err != nil {
			return nil, err
		}
		feeds = append(feeds, f)
	}
	return feeds, nil
}

func (s *Store) ListEntries(unreadOnly bool, sortMode string) ([]Entry, error) {
	query := "SELECT id, feed_id, title, content, url, published_at, is_read FROM entries"
	var args []interface{}
	if unreadOnly {
		query += " WHERE is_read = 0"
	}

	if sortMode == "oldest" {
		query += " ORDER BY published_at ASC"
	} else {
		query += " ORDER BY published_at DESC"
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.FeedID, &e.Title, &e.Content, &e.URL, &e.PublishedAt, &e.IsRead); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (s *Store) MarkAsRead(entryID string) error {
	_, err := s.db.Exec("UPDATE entries SET is_read = 1 WHERE id = ?", entryID)
	return err
}

func (s *Store) SearchEntries(searchTerm string) ([]Entry, error) {
	query := `SELECT id, feed_id, title, content, url, published_at, is_read 
              FROM entries 
              JOIN entries_fts ON entries.rowid = entries_fts.rowid 
              WHERE entries_fts MATCH ? 
              ORDER BY rank`
	rows, err := s.db.Query(query, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.FeedID, &e.Title, &e.Content, &e.URL, &e.PublishedAt, &e.IsRead); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (s *Store) GetFeedByURL(url string) (*Feed, error) {
	var f Feed
	err := s.db.QueryRow("SELECT id, title, site_url, feed_url, category, last_updated FROM feeds WHERE feed_url = ?", url).
		Scan(&f.ID, &f.Title, &f.SiteURL, &f.FeedURL, &f.Category, &f.LastUpdated)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *Store) UpsertUser(username, passwordHash string) error {
	_, err := s.db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?) ON CONFLICT(username) DO UPDATE SET password_hash=excluded.password_hash", username, passwordHash)
	return err
}

func (s *Store) GetUserHash(username string) (string, error) {
	var hash string
	err := s.db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&hash)
	return hash, err
}

func (s *Store) DeleteFeed(id string) error {
	_, err := s.db.Exec("DELETE FROM feeds WHERE id = ?", id)
	return err
}

func (s *Store) UpdateFeed(id, title, category string) error {
	_, err := s.db.Exec("UPDATE feeds SET title = ?, category = ? WHERE id = ?", title, category, id)
	return err
}

func (s *Store) GetFeedByID(id string) (*Feed, error) {
	var f Feed
	err := s.db.QueryRow("SELECT id, title, site_url, feed_url, category, last_updated FROM feeds WHERE id = ?", id).
		Scan(&f.ID, &f.Title, &f.SiteURL, &f.FeedURL, &f.Category, &f.LastUpdated)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

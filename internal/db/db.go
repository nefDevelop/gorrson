package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	"github.com/nef734/gorrson/internal/models"
)

type Store struct {
	db       *sql.DB
	username string
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

	s := &Store{db: db, username: username}
	
	// Optimization: SQLite performs best with a single connection in Go to avoid lock contention.
	// WAL mode handles the concurrency at the database level.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) GetUsername() string {
	return s.username
}

func (s *Store) migrate() error {
	queries := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA synchronous=NORMAL;`,
		`PRAGMA busy_timeout=5000;`,
		`PRAGMA foreign_keys=ON;`,
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
			summary TEXT,
			content TEXT,
			url TEXT UNIQUE NOT NULL,
			image_url TEXT,
			published_at DATETIME,
			is_read BOOLEAN DEFAULT 0,
			is_favorite BOOLEAN DEFAULT 0,
			was_ever_favorite BOOLEAN DEFAULT 0,
			is_full_content BOOLEAN DEFAULT 0,
			FOREIGN KEY(feed_id) REFERENCES feeds(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS filters (
			id TEXT PRIMARY KEY,
			feed_id TEXT,
			pattern TEXT NOT NULL,
			action TEXT NOT NULL,
			target TEXT NOT NULL,
			replacement TEXT,
			FOREIGN KEY(feed_id) REFERENCES feeds(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_entries_published ON entries(published_at DESC);`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			refresh_token TEXT UNIQUE NOT NULL,
			expires_at DATETIME NOT NULL
		);`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS entries_fts USING fts5(
			title, content, content='entries', content_rowid='rowid'
		);`,
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

func (s *Store) AddFeed(f *models.Feed) error {
	_, err := s.db.Exec(`INSERT INTO feeds (id, title, site_url, feed_url, category, last_updated) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(feed_url) DO UPDATE SET last_updated=excluded.last_updated`,
		f.ID, f.Title, f.SiteURL, f.FeedURL, f.Category, f.LastUpdated)
	return err
}

func (s *Store) AddEntry(e *models.Entry) error {
	_, err := s.db.Exec(`INSERT INTO entries (id, feed_id, title, summary, content, url, image_url, published_at, is_read, is_favorite, is_full_content) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(url) DO NOTHING`,
		e.ID, e.FeedID, e.Title, e.Summary, e.Content, e.URL, e.ImageURL, e.PublishedAt, e.IsRead, e.IsFavorite, e.IsFullContent)
	return err
}

func (s *Store) ListFeeds() ([]models.Feed, error) {
	rows, err := s.db.Query(`
		SELECT f.id, f.title, f.site_url, f.feed_url, f.category, f.last_updated,
		       (SELECT COUNT(*) FROM entries e WHERE e.feed_id = f.id AND e.is_read = 0) as unread_count
		FROM feeds f`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []models.Feed
	for rows.Next() {
		var f models.Feed
		if err := rows.Scan(&f.ID, &f.Title, &f.SiteURL, &f.FeedURL, &f.Category, &f.LastUpdated, &f.UnreadCount); err != nil {
			return nil, err
		}
		feeds = append(feeds, f)
	}
	return feeds, nil
}

func (s *Store) ListEntries(unreadOnly bool, sortMode string, limit, offset int, category, feedID string) ([]models.Entry, error) {
	query := `SELECT e.id, e.feed_id, e.title, e.summary, e.content, e.url, e.image_url, e.published_at, e.is_read, e.is_favorite, e.is_full_content 
              FROM entries e 
              JOIN feeds f ON e.feed_id = f.id`
	var args []interface{}
	
	conditions := []string{}
	if unreadOnly {
		conditions = append(conditions, "e.is_read = 0")
	}

	if sortMode == "favorite" {
		conditions = append(conditions, "e.is_favorite = 1")
	}

	if category != "" {
		conditions = append(conditions, "f.category COLLATE NOCASE = ?")
		args = append(args, category)
	}

	if feedID != "" {
		conditions = append(conditions, "e.feed_id = ?")
		args = append(args, feedID)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	if sortMode == "oldest" {
		query += " ORDER BY e.published_at ASC"
	} else {
		query += " ORDER BY e.published_at DESC"
	}

	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.Entry
	for rows.Next() {
		var e models.Entry
		if err := rows.Scan(&e.ID, &e.FeedID, &e.Title, &e.Summary, &e.Content, &e.URL, &e.ImageURL, &e.PublishedAt, &e.IsRead, &e.IsFavorite, &e.IsFullContent); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (s *Store) GetEntryByID(id string) (*models.Entry, error) {
	var e models.Entry
	err := s.db.QueryRow(`SELECT id, feed_id, title, summary, content, url, image_url, published_at, is_read, is_favorite, is_full_content FROM entries WHERE id = ?`, id).
		Scan(&e.ID, &e.FeedID, &e.Title, &e.Summary, &e.Content, &e.URL, &e.ImageURL, &e.PublishedAt, &e.IsRead, &e.IsFavorite, &e.IsFullContent)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *Store) UpdateEntryContent(id, content string) error {
	_, err := s.db.Exec("UPDATE entries SET content = ?, is_full_content = 1 WHERE id = ?", content, id)
	return err
}

func (s *Store) MarkAsRead(entryID string) error {
	_, err := s.db.Exec("UPDATE entries SET is_read = 1 WHERE id = ?", entryID)
	return err
}

func (s *Store) MarkAllRead(category, feedID, search string, limit int) error {
	query := "UPDATE entries SET is_read = 1"
	var args []interface{}

	if search != "" {
		// For search, mark entries that match the FTS search
		query = `UPDATE entries SET is_read = 1 WHERE id IN (
			SELECT entries.id FROM entries 
			JOIN entries_fts ON entries.rowid = entries_fts.rowid 
			WHERE entries_fts MATCH ?
			ORDER BY entries.published_at DESC`
		if limit > 0 {
			query += " LIMIT ?"
			args = append(args, search, limit)
		} else {
			args = append(args, search)
		}
		query += ")"
	} else {
		// Handle category and feed_id
		query = `UPDATE entries SET is_read = 1 WHERE id IN (
			SELECT e.id FROM entries e
			JOIN feeds f ON e.feed_id = f.id
			WHERE 1=1`
		if category != "" {
			query += " AND f.category COLLATE NOCASE = ?"
			args = append(args, category)
		}
		if feedID != "" {
			query += " AND e.feed_id = ?"
			args = append(args, feedID)
		}
		query += " ORDER BY e.published_at DESC"
		if limit > 0 {
			query += " LIMIT ?"
			args = append(args, limit)
		}
		query += ")"
	}

	_, err := s.db.Exec(query, args...)
	return err
}

func (s *Store) ToggleFavorite(entryID string, isFavorite bool) error {
	if isFavorite {
		_, err := s.db.Exec("UPDATE entries SET is_favorite = 1, was_ever_favorite = 1 WHERE id = ?", entryID)
		return err
	}
	_, err := s.db.Exec("UPDATE entries SET is_favorite = 0 WHERE id = ?", entryID)
	return err
}

func (s *Store) CleanupOldEntries() (int64, error) {
	res, err := s.db.Exec(`
		DELETE FROM entries 
		WHERE published_at < date('now', '-1 year') 
		AND is_favorite = 0 
		AND was_ever_favorite = 0`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) SearchEntries(searchTerm string) ([]models.Entry, error) {
	query := `SELECT id, feed_id, title, summary, content, url, image_url, published_at, is_read, is_favorite, is_full_content 
              FROM entries 
              JOIN entries_fts ON entries.rowid = entries_fts.rowid 
              WHERE entries_fts MATCH ? 
              ORDER BY rank`
	rows, err := s.db.Query(query, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.Entry
	for rows.Next() {
		var e models.Entry
		if err := rows.Scan(&e.ID, &e.FeedID, &e.Title, &e.Summary, &e.Content, &e.URL, &e.ImageURL, &e.PublishedAt, &e.IsRead, &e.IsFavorite, &e.IsFullContent); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (s *Store) GetFeedByURL(url string) (*models.Feed, error) {
	var f models.Feed
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

func (s *Store) RenameCategory(oldName, newName string) error {
	_, err := s.db.Exec("UPDATE feeds SET category = ? WHERE category = ?", newName, oldName)
	return err
}

func (s *Store) GetFeedByID(id string) (*models.Feed, error) {
	var f models.Feed
	err := s.db.QueryRow("SELECT id, title, site_url, feed_url, category, last_updated FROM feeds WHERE id = ?", id).
		Scan(&f.ID, &f.Title, &f.SiteURL, &f.FeedURL, &f.Category, &f.LastUpdated)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *Store) AddFilter(f *models.Filter) error {
	var feedID sql.NullString
	if f.FeedID != "" {
		feedID = sql.NullString{String: f.FeedID, Valid: true}
	}
	_, err := s.db.Exec(`INSERT INTO filters (id, feed_id, pattern, action, target, replacement) VALUES (?, ?, ?, ?, ?, ?)`,
		f.ID, feedID, f.Pattern, f.Action, f.Target, f.Replacement)
	return err
}

func (s *Store) ListFilters() ([]models.Filter, error) {
	rows, err := s.db.Query("SELECT id, feed_id, pattern, action, target, replacement FROM filters")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var filters []models.Filter
	for rows.Next() {
		var f models.Filter
		var feedID sql.NullString
		if err := rows.Scan(&f.ID, &feedID, &f.Pattern, &f.Action, &f.Target, &f.Replacement); err != nil {
			return nil, err
		}
		if feedID.Valid {
			f.FeedID = feedID.String
		}
		filters = append(filters, f)
	}
	return filters, nil
}

func (s *Store) GetFiltersForFeed(feedID string) ([]models.Filter, error) {
	rows, err := s.db.Query("SELECT id, feed_id, pattern, action, target, replacement FROM filters WHERE feed_id IS NULL OR feed_id = '' OR feed_id = ?", feedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var filters []models.Filter
	for rows.Next() {
		var f models.Filter
		var fID sql.NullString
		if err := rows.Scan(&f.ID, &fID, &f.Pattern, &f.Action, &f.Target, &f.Replacement); err != nil {
			return nil, err
		}
		if fID.Valid {
			f.FeedID = fID.String
		}
		filters = append(filters, f)
	}
	return filters, nil
}

func (s *Store) DeleteFilter(id string) error {
	_, err := s.db.Exec("DELETE FROM filters WHERE id = ?", id)
	return err
}

type Session struct {
	ID           string
	Username     string
	RefreshToken string
	ExpiresAt    time.Time
}

func (s *Store) CreateSession(sess *Session) error {
	_, err := s.db.Exec("INSERT INTO sessions (id, username, refresh_token, expires_at) VALUES (?, ?, ?, ?)",
		sess.ID, sess.Username, sess.RefreshToken, sess.ExpiresAt)
	return err
}

func (s *Store) GetSession(token string) (*Session, error) {
	var sess Session
	err := s.db.QueryRow("SELECT id, username, refresh_token, expires_at FROM sessions WHERE refresh_token = ?", token).
		Scan(&sess.ID, &sess.Username, &sess.RefreshToken, &sess.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Store) DeleteSession(token string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE refresh_token = ?", token)
	return err
}

func (s *Store) DeleteExpiredSessions() error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now())
	return err
}

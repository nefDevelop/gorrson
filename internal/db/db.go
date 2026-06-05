package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nef734/gorrson/internal/models"
	"github.com/nef734/gorrson/internal/sanitizer"
	_ "modernc.org/sqlite"
)

type Store struct {
	db       *sql.DB
	username string
}

func NewStore(dataDir, username string) (*Store, error) {
	if !sanitizer.IsValidUsername(username) {
		return nil, fmt.Errorf("invalid username: %s", username)
	}

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, fmt.Errorf("could not create data directory: %w", err)
		}
	}
	dbPath := filepath.Join(dataDir, username+".db")
	connStr := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)", dbPath)
	db, err := sql.Open("sqlite", connStr)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}

	s := &Store{db: db, username: username}

	// Important to limit SQLite connections if busy_timeout acts up, but 1 is safest for writers.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err := s.migrate(context.Background()); err != nil {
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

// Begin starts a new transaction.
func (s *Store) Begin() (*sql.Tx, error) {
	return s.db.Begin()
}

// BeginTx starts a new transaction with context.
func (s *Store) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return s.db.BeginTx(ctx, opts)
}

// VerifySQLiteFile checks if a file is a valid SQLite database and passes a quick integrity check.
func VerifySQLiteFile(ctx context.Context, path string) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()

	_, _ = db.ExecContext(ctx, "PRAGMA busy_timeout=2000")

	var result string
	err = db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&result)
	if err != nil {
		return fmt.Errorf("not a valid sqlite file or corrupted: %w", err)
	}
	if result != "ok" {
		return fmt.Errorf("sqlite integrity check failed: %s", result)
	}

	var name string
	err = db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='feeds'").Scan(&name)
	if err != nil {
		return fmt.Errorf("database schema looks incorrect: missing 'feeds' table")
	}

	return nil
}

func (s *Store) migrate(ctx context.Context) error {
	queries := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA synchronous=NORMAL;`,
		`PRAGMA busy_timeout=5000;`,
		`PRAGMA cache_size=-1000;`,
		`PRAGMA mmap_size=0;`,
		`PRAGMA temp_store=MEMORY;`,
		`PRAGMA foreign_keys=ON;`,
		`CREATE TABLE IF NOT EXISTS feeds (
			id TEXT PRIMARY KEY,
			fever_id INTEGER DEFAULT 0,
			title TEXT NOT NULL,
			site_url TEXT,
			feed_url TEXT UNIQUE NOT NULL,
			category TEXT,
			last_updated DATETIME,
			user_agent TEXT,
			username TEXT,
			password TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS users (
			username TEXT PRIMARY KEY,
			password_hash TEXT NOT NULL,
			fever_key TEXT DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS entries (
			id TEXT PRIMARY KEY,
			fever_id INTEGER DEFAULT 0,
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
			is_priority BOOLEAN DEFAULT 0,
			matched_action TEXT,
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
		`CREATE TABLE IF NOT EXISTS api_keys (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			key_hash TEXT UNIQUE NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
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
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return err
		}
	}

	_, _ = s.db.ExecContext(ctx, "ALTER TABLE entries ADD COLUMN matched_action TEXT")
	_, _ = s.db.ExecContext(ctx, "ALTER TABLE entries ADD COLUMN is_priority BOOLEAN DEFAULT 0")
	_, _ = s.db.ExecContext(ctx, "ALTER TABLE feeds ADD COLUMN user_agent TEXT")
	_, _ = s.db.ExecContext(ctx, "ALTER TABLE feeds ADD COLUMN username TEXT")
	_, _ = s.db.ExecContext(ctx, "ALTER TABLE feeds ADD COLUMN password TEXT")
	_, _ = s.db.ExecContext(ctx, "ALTER TABLE feeds ADD COLUMN fever_id INTEGER DEFAULT 0")
	_, _ = s.db.ExecContext(ctx, "ALTER TABLE entries ADD COLUMN fever_id INTEGER DEFAULT 0")
	_, _ = s.db.ExecContext(ctx, "ALTER TABLE users ADD COLUMN fever_key TEXT DEFAULT ''")

	return nil
}

func (s *Store) AddFeed(ctx context.Context, f *models.Feed) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO feeds (id, fever_id, title, site_url, feed_url, category, last_updated, user_agent, username, password) 
                         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) 
                         ON CONFLICT(feed_url) 
                         DO UPDATE SET last_updated=excluded.last_updated, user_agent=excluded.user_agent, username=excluded.username, password=excluded.password, fever_id=excluded.fever_id`,
		f.ID, f.FeverID, f.Title, f.SiteURL, f.FeedURL, f.Category, f.LastUpdated, f.UserAgent, f.Username, f.Password)
	return err
}

func (s *Store) AddEntry(ctx context.Context, e *models.Entry) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO entries (id, fever_id, feed_id, title, summary, content, url, image_url, published_at, is_read, is_favorite, is_full_content, is_priority, matched_action) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(url) DO NOTHING`,
		e.ID, e.FeverID, e.FeedID, e.Title, e.Summary, e.Content, e.URL, e.ImageURL, e.PublishedAt, e.IsRead, e.IsFavorite, e.IsFullContent, e.IsPriority, e.MatchedAction)
	return err
}

func (s *Store) AddEntryTx(ctx context.Context, tx *sql.Tx, e *models.Entry) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO entries (id, fever_id, feed_id, title, summary, content, url, image_url, published_at, is_read, is_favorite, is_full_content, is_priority, matched_action) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(url) DO NOTHING`,
		e.ID, e.FeverID, e.FeedID, e.Title, e.Summary, e.Content, e.URL, e.ImageURL, e.PublishedAt, e.IsRead, e.IsFavorite, e.IsFullContent, e.IsPriority, e.MatchedAction)
	return err
}

func (s *Store) ListFeeds(ctx context.Context) ([]models.Feed, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT f.id, f.fever_id, f.title, f.site_url, f.feed_url, f.category, f.last_updated, f.user_agent, f.username, f.password,
		       (SELECT COUNT(*) FROM entries e WHERE e.feed_id = f.id AND e.is_read = 0) as unread_count
		FROM feeds f`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var feeds []models.Feed
	for rows.Next() {
		var f models.Feed
		var ua, user, pass sql.NullString
		if err := rows.Scan(&f.ID, &f.FeverID, &f.Title, &f.SiteURL, &f.FeedURL, &f.Category, &f.LastUpdated, &ua, &user, &pass, &f.UnreadCount); err != nil {
			return nil, err
		}
		f.UserAgent = ua.String
		f.Username = user.String
		f.Password = pass.String
		feeds = append(feeds, f)
	}
	return feeds, nil
}

func (s *Store) ListEntries(ctx context.Context, unreadOnly bool, sortMode string, limit, offset int, category, feedID string) ([]models.Entry, error) {
	query := `SELECT e.id, e.fever_id, e.feed_id, e.title, e.summary, e.content, e.url, e.image_url, e.published_at, e.is_read, e.is_favorite, e.is_full_content, e.is_priority, e.matched_action, f.title, f.category 
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
		if strings.ToLower(category) == "general" {
			conditions = append(conditions, "(f.category IS NULL OR f.category = '' OR f.category COLLATE NOCASE = 'general')")
		} else {
			conditions = append(conditions, "f.category COLLATE NOCASE = ?")
			args = append(args, category)
		}
	}

	if feedID != "" {
		conditions = append(conditions, "e.feed_id = ?")
		args = append(args, feedID)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	if sortMode == "oldest" {
		query += " ORDER BY e.is_priority DESC, e.published_at ASC"
	} else {
		query += " ORDER BY e.is_priority DESC, e.published_at DESC"
	}

	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var entries []models.Entry
	for rows.Next() {
		var e models.Entry
		var summary, content, imageURL, matchedAction, category sql.NullString
		if err := rows.Scan(&e.ID, &e.FeverID, &e.FeedID, &e.Title, &summary, &content, &e.URL, &imageURL, &e.PublishedAt, &e.IsRead, &e.IsFavorite, &e.IsFullContent, &e.IsPriority, &matchedAction, &e.Source, &category); err != nil {
			return nil, err
		}
		e.Summary = summary.String
		e.Content = content.String
		e.ImageURL = imageURL.String
		e.MatchedAction = matchedAction.String
		e.Category = category.String
		entries = append(entries, e)
	}
	return entries, nil
}

func (s *Store) GetEntryByID(ctx context.Context, id string) (*models.Entry, error) {
	var e models.Entry
	var summary, content, imageURL, matchedAction, category sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT e.id, e.fever_id, e.feed_id, e.title, e.summary, e.content, e.url, e.image_url, e.published_at, e.is_read, e.is_favorite, e.is_full_content, e.is_priority, e.matched_action, f.title, f.category 
		FROM entries e
		JOIN feeds f ON e.feed_id = f.id
		WHERE e.id = ?`, id).
		Scan(&e.ID, &e.FeverID, &e.FeedID, &e.Title, &summary, &content, &e.URL, &imageURL, &e.PublishedAt, &e.IsRead, &e.IsFavorite, &e.IsFullContent, &e.IsPriority, &matchedAction, &e.Source, &category)
	if err != nil {
		return nil, err
	}
	e.Summary = summary.String
	e.Content = content.String
	e.ImageURL = imageURL.String
	e.MatchedAction = matchedAction.String
	e.Category = category.String
	return &e, nil
}

func (s *Store) UpdateEntryContent(ctx context.Context, id, content string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE entries SET content = ?, is_full_content = 1 WHERE id = ?", content, id)
	return err
}

func (s *Store) MarkAsRead(ctx context.Context, entryID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE entries SET is_read = 1 WHERE id = ?", entryID)
	return err
}

func (s *Store) MarkAsReadByFeverID(ctx context.Context, feverID int) error {
	_, err := s.db.ExecContext(ctx, "UPDATE entries SET is_read = 1 WHERE fever_id = ?", feverID)
	return err
}

func (s *Store) ToggleFavoriteByFeverID(ctx context.Context, feverID int, isFavorite bool) error {
	if isFavorite {
		_, err := s.db.ExecContext(ctx, "UPDATE entries SET is_favorite = 1, was_ever_favorite = 1 WHERE fever_id = ?", feverID)
		return err
	}
	_, err := s.db.ExecContext(ctx, "UPDATE entries SET is_favorite = 0 WHERE fever_id = ?", feverID)
	return err
}

func (s *Store) MarkAllRead(ctx context.Context, category, feedID, search string, limit int) error {
	var query string
	var args []interface{}

	if search != "" {
		query = `UPDATE entries SET is_read = 1 WHERE is_read = 0 AND id IN (
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
		query = `UPDATE entries SET is_read = 1 WHERE is_read = 0 AND id IN (
			SELECT e.id FROM entries e
			JOIN feeds f ON e.feed_id = f.id
			WHERE e.is_read = 0`
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

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}
func (s *Store) ToggleFavorite(ctx context.Context, entryID string, isFavorite bool) error {
	if isFavorite {
		_, err := s.db.ExecContext(ctx, "UPDATE entries SET is_favorite = 1, was_ever_favorite = 1 WHERE id = ?", entryID)
		return err
	}
	_, err := s.db.ExecContext(ctx, "UPDATE entries SET is_favorite = 0 WHERE id = ?", entryID)
	return err
}

func (s *Store) CleanupOldEntries(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM entries 
		WHERE published_at < date('now', '-1 year') 
		AND is_favorite = 0 
		AND was_ever_favorite = 0`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) SearchEntries(ctx context.Context, searchTerm string, limit, offset int) ([]models.Entry, error) {
	query := `SELECT e.id, e.fever_id, e.feed_id, e.title, e.summary, e.content, e.url, e.image_url, e.published_at, e.is_read, e.is_favorite, e.is_full_content, e.is_priority, e.matched_action, f.title, f.category 
              FROM entries e
              JOIN feeds f ON e.feed_id = f.id
              JOIN entries_fts ON e.rowid = entries_fts.rowid 
              WHERE entries_fts MATCH ? 
              ORDER BY rank`
	
	var args []interface{}
	args = append(args, searchTerm)

	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var entries []models.Entry
	for rows.Next() {
		var e models.Entry
		var summary, content, imageURL, matchedAction, category sql.NullString
		if err := rows.Scan(&e.ID, &e.FeverID, &e.FeedID, &e.Title, &summary, &content, &e.URL, &imageURL, &e.PublishedAt, &e.IsRead, &e.IsFavorite, &e.IsFullContent, &e.IsPriority, &matchedAction, &e.Source, &category); err != nil {
			return nil, err
		}
		e.Summary = summary.String
		e.Content = content.String
		e.ImageURL = imageURL.String
		e.MatchedAction = matchedAction.String
		e.Category = category.String
		entries = append(entries, e)
	}
	return entries, nil
}

func (s *Store) GetFeedByURL(ctx context.Context, url string) (*models.Feed, error) {
	var f models.Feed
	var ua, user, pass sql.NullString
	err := s.db.QueryRowContext(ctx, "SELECT id, title, site_url, feed_url, category, last_updated, user_agent, username, password FROM feeds WHERE feed_url = ?", url).
		Scan(&f.ID, &f.Title, &f.SiteURL, &f.FeedURL, &f.Category, &f.LastUpdated, &ua, &user, &pass)
	if err != nil {
		return nil, err
	}
	f.UserAgent = ua.String
	f.Username = user.String
	f.Password = pass.String
	return &f, nil
}

func (s *Store) UpsertUser(ctx context.Context, username, passwordHash, feverKey string) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO users (username, password_hash, fever_key) VALUES (?, ?, ?) ON CONFLICT(username) DO UPDATE SET password_hash=excluded.password_hash, fever_key=excluded.fever_key", username, passwordHash, feverKey)
	return err
}

func (s *Store) GetUserHash(ctx context.Context, username string) (string, error) {
	var hash string
	err := s.db.QueryRowContext(ctx, "SELECT password_hash FROM users WHERE username = ?", username).Scan(&hash)
	return hash, err
}

func (s *Store) GetUserFeverKey(ctx context.Context, username string) (string, error) {
	var key string
	err := s.db.QueryRowContext(ctx, "SELECT fever_key FROM users WHERE username = ?", username).Scan(&key)
	return key, err
}

func (s *Store) DeleteFeed(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM feeds WHERE id = ?", id)
	return err
}

func (s *Store) UpdateFeed(ctx context.Context, id, title, category, userAgent string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE feeds SET title = ?, category = ?, user_agent = ? WHERE id = ?", title, category, userAgent, id)
	return err
}

func (s *Store) RenameCategory(ctx context.Context, oldName, newName string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE feeds SET category = ? WHERE category = ?", newName, oldName)
	return err
}

func (s *Store) GetFeedByID(ctx context.Context, id string) (*models.Feed, error) {
	var f models.Feed
	var ua, user, pass sql.NullString
	err := s.db.QueryRowContext(ctx, "SELECT id, title, site_url, feed_url, category, last_updated, user_agent, username, password FROM feeds WHERE id = ?", id).
		Scan(&f.ID, &f.Title, &f.SiteURL, &f.FeedURL, &f.Category, &f.LastUpdated, &ua, &user, &pass)
	if err != nil {
		return nil, err
	}
	f.UserAgent = ua.String
	f.Username = user.String
	f.Password = pass.String
	return &f, nil
}

func (s *Store) AddFilter(ctx context.Context, f *models.Filter) error {
	var feedID sql.NullString
	if f.FeedID != "" {
		feedID = sql.NullString{String: f.FeedID, Valid: true}
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO filters (id, feed_id, pattern, action, target, replacement) VALUES (?, ?, ?, ?, ?, ?)`,
		f.ID, feedID, f.Pattern, f.Action, f.Target, f.Replacement)
	return err
}

func (s *Store) ListFilters(ctx context.Context) ([]models.Filter, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, feed_id, pattern, action, target, replacement FROM filters")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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

func (s *Store) GetFiltersForFeed(ctx context.Context, feedID string) ([]models.Filter, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, feed_id, pattern, action, target, replacement FROM filters WHERE feed_id IS NULL OR feed_id = '' OR feed_id = ?", feedID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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

func (s *Store) DeleteFilter(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM filters WHERE id = ?", id)
	return err
}

type Session struct {
	ID           string
	Username     string
	RefreshToken string
	ExpiresAt    time.Time
}

func (s *Store) CreateSession(ctx context.Context, sess *Session) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO sessions (id, username, refresh_token, expires_at) VALUES (?, ?, ?, ?)",
		sess.ID, sess.Username, sess.RefreshToken, sess.ExpiresAt)
	return err
}

func (s *Store) GetSession(ctx context.Context, token string) (*Session, error) {
	var sess Session
	err := s.db.QueryRowContext(ctx, "SELECT id, username, refresh_token, expires_at FROM sessions WHERE refresh_token = ?", token).
		Scan(&sess.ID, &sess.Username, &sess.RefreshToken, &sess.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE refresh_token = ?", token)
	return err
}

func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < ?", time.Now())
	return err
}

func (s *Store) AddAPIKey(ctx context.Context, id, name, hash string) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO api_keys (id, name, key_hash) VALUES (?, ?, ?)", id, name, hash)
	return err
}

func (s *Store) ListAPIKeys(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, name, created_at FROM api_keys ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []map[string]interface{}
	for rows.Next() {
		var id, name string
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &createdAt); err != nil {
			return nil, err
		}
		keys = append(keys, map[string]interface{}{
			"id":         id,
			"name":       name,
			"created_at": createdAt,
		})
	}
	return keys, nil
}

func (s *Store) DeleteAPIKey(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM api_keys WHERE id = ?", id)
	return err
}

func (s *Store) ValidateAPIKey(ctx context.Context, hash string) bool {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM api_keys WHERE key_hash = ?", hash).Scan(&count)
	return err == nil && count > 0
}

func (s *Store) GetStats(ctx context.Context) (*models.Stats, error) {
	var stats models.Stats

	// Total Entries
	_ = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM entries").Scan(&stats.TotalEntries)
	// Unread Entries
	_ = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM entries WHERE is_read = 0").Scan(&stats.UnreadEntries)
	// Total Feeds
	_ = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM feeds").Scan(&stats.TotalFeeds)

	// Category Stats
	rows, err := s.db.QueryContext(ctx, "SELECT f.category, COUNT(e.id) FROM feeds f JOIN entries e ON f.id = e.feed_id GROUP BY f.category")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cat sql.NullString
			var count int
			if err := rows.Scan(&cat, &count); err == nil {
				name := "Uncategorized"
				if cat.Valid && cat.String != "" {
					name = cat.String
				}
				stats.CategoryStats = append(stats.CategoryStats, models.CategoryStat{Name: name, Count: count})
			}
		}
	}

	// Top 10 Feeds by volume
	rows, err = s.db.QueryContext(ctx, "SELECT f.title, COUNT(e.id) as c FROM feeds f JOIN entries e ON f.id = e.feed_id GROUP BY f.id ORDER BY c DESC LIMIT 10")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var title string
			var count int
			if err := rows.Scan(&title, &count); err == nil {
				stats.FeedStats = append(stats.FeedStats, models.FeedStat{Title: title, Count: count})
			}
		}
	}

	return &stats, nil
}

package models

import "time"

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
	Summary     string    `json:"summary"`
	Content     string    `json:"content"`
	URL         string    `json:"url"`
	ImageURL    string    `json:"image_url"`
	PublishedAt time.Time `json:"published_at"`
	IsRead      bool      `json:"is_read"`
	IsFavorite  bool      `json:"is_favorite"`
}

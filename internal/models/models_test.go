package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFeedJSONRoundTrip(t *testing.T) {
	original := Feed{
		ID:          "rt-feed",
		FeverID:     99,
		Title:       "Round Trip",
		SiteURL:     "https://example.com",
		FeedURL:     "https://example.com/rss",
		Category:    "News",
		LastUpdated: time.Now().Truncate(time.Second),
		UnreadCount: 10,
		UserAgent:   "CustomBot/1.0",
		Username:    "user",
		Password:    "pass",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Feed
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID = %s, want %s", decoded.ID, original.ID)
	}
	if decoded.Title != original.Title {
		t.Errorf("Title = %s, want %s", decoded.Title, original.Title)
	}
	if decoded.UserAgent != original.UserAgent {
		t.Errorf("UserAgent = %s, want %s", decoded.UserAgent, original.UserAgent)
	}
	if decoded.Username != original.Username {
		t.Errorf("Username = %s, want %s", decoded.Username, original.Username)
	}
}

func TestEntryJSONRoundTrip(t *testing.T) {
	original := Entry{
		ID:            "rt-entry",
		FeverID:       12345,
		FeedID:        "feed-rt",
		Title:         "Round Trip Entry",
		Summary:       "A summary",
		Content:       "<p>Content</p>",
		URL:           "https://example.com/rt",
		ImageURL:      "https://example.com/img.png",
		PublishedAt:   time.Now().Truncate(time.Second),
		IsRead:        true,
		IsFavorite:    true,
		IsFullContent: true,
		IsPriority:    false,
		MatchedAction: "REPLACE",
		Source:        "test",
		Category:      "Testing",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Entry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID = %s, want %s", decoded.ID, original.ID)
	}
	if decoded.FeverID != original.FeverID {
		t.Errorf("FeverID = %d, want %d", decoded.FeverID, original.FeverID)
	}
	if decoded.MatchedAction != original.MatchedAction {
		t.Errorf("MatchedAction = %s, want %s", decoded.MatchedAction, original.MatchedAction)
	}
}

func TestStatsJSONMarshal(t *testing.T) {
	stats := Stats{
		TotalEntries:  100,
		UnreadEntries: 25,
		TotalFeeds:    5,
		CategoryStats: []CategoryStat{
			{Name: "Tech", Count: 30},
			{Name: "News", Count: 20},
		},
		FeedStats: []FeedStat{
			{Title: "Hacker News", Count: 15},
			{Title: "Ars Technica", Count: 10},
		},
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if m["total_entries"] != float64(100) {
		t.Errorf("total_entries = %v, want 100", m["total_entries"])
	}
	if m["total_feeds"] != float64(5) {
		t.Errorf("total_feeds = %v, want 5", m["total_feeds"])
	}

	categories := m["category_stats"].([]interface{})
	if len(categories) != 2 {
		t.Errorf("category_stats length = %d, want 2", len(categories))
	}
}

func TestFilterGlobalVsSpecific(t *testing.T) {
	globalFilter := Filter{ID: "g1", Pattern: ".*", Action: "BLOCK", Target: "both"}
	specificFilter := Filter{ID: "s1", FeedID: "feed-1", Pattern: "spam", Action: "HIDE", Target: "title"}

	globalData, _ := json.Marshal(globalFilter)
	specificData, _ := json.Marshal(specificFilter)

	var globalMap, specificMap map[string]interface{}
	json.Unmarshal(globalData, &globalMap)
	json.Unmarshal(specificData, &specificMap)

	if _, ok := globalMap["feed_id"]; ok {
		t.Error("Global filter should not have feed_id in JSON")
	}
	if _, ok := specificMap["feed_id"]; !ok {
		t.Error("Specific filter should have feed_id in JSON")
	}
}

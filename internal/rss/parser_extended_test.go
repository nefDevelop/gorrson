package rss

import (
	"testing"
)

func TestNormalizeURLEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"multiple tracking params",
			"https://example.com/page?utm_source=tw&utm_medium=social&utm_campaign=launch&fbclid=123&valid=yes",
			"https://example.com/page?valid=yes",
		},
		{
			"only tracking params",
			"https://example.com/page?utm_source=x&fbclid=y",
			"https://example.com/page",
		},
		{
			"empty string",
			"",
			"",
		},
		{
			"preserve non-tracking",
			"https://example.com/page?id=123&sort=date",
			"https://example.com/page?id=123&sort=date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeURL(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGenerateFeverIDDifferentInputs(t *testing.T) {
	ids := []string{"feed-1", "feed-2", "https://example.com/rss", "https://other.com/feed", ""}

	seen := make(map[int]bool)
	for _, id := range ids {
		result := generateFeverID(id)
		if seen[result] {
			t.Errorf("FeverID collision for id %q: %d", id, result)
		}
		seen[result] = true
	}
}

func TestNewParser(t *testing.T) {
	p := NewParser()
	if p == nil {
		t.Fatal("NewParser returned nil")
	}
	if p.fp == nil {
		t.Fatal("Parser.fp is nil")
	}
}

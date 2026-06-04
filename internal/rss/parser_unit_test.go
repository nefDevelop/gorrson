package rss

import (
	"testing"

	"github.com/nef734/gorrson/internal/util"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com/page?utm_source=twitter&ref=rss", "https://example.com/page"},
		{"https://example.com/page#fragment", "https://example.com/page"},
		{"https://example.com/page?q=search&fbclid=123", "https://example.com/page?q=search"},
		{"https://example.com/page?utm_campaign=winter&utm_medium=email", "https://example.com/page"},
		{"https://example.com/page?valid=param", "https://example.com/page?valid=param"},
		{"", ""},
	}

	for _, tt := range tests {
		got := normalizeURL(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeURL(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestExtractFirstImageFromHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"simple img",
			`<div><img src="https://example.com/img.jpg"></div>`,
			"https://example.com/img.jpg",
		},
		{
			"ignore tracking pixel",
			`<div><img src="https://tracking.com/pixel.gif"><img src="https://example.com/real.png"></div>`,
			"https://example.com/real.png",
		},
		{
			"ignore spacer",
			`<div><img src="spacer.gif"><img src="content.jpg"></div>`,
			"content.jpg",
		},
		{
			"no image",
			`<div>no image here</div>`,
			"",
		},
		{
			"single quotes",
			`<img src='https://example.com/img.webp'>`,
			"https://example.com/img.webp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := util.ExtractFirstImageFromHTML(tt.input)
			if got != tt.expected {
				t.Errorf("util.ExtractFirstImageFromHTML() = %q; want %q", got, tt.expected)
			}
		})
	}
}

func TestGenerateFeverID(t *testing.T) {
	id1 := "some-uuid-1"
	id2 := "some-uuid-2"

	res1 := generateFeverID(id1)
	res2 := generateFeverID(id1)
	res3 := generateFeverID(id2)

	if res1 != res2 {
		t.Error("generateFeverID should be deterministic")
	}
	if res1 == res3 {
		t.Error("generateFeverID should produce different IDs for different inputs")
	}
}

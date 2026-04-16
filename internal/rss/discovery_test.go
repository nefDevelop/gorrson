package rss

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscoverFeed(t *testing.T) {
	t.Setenv("GORRSON_TEST", "true")

	// Mock HTML server with feed links
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `<html><head>
			<link rel="alternate" type="application/rss+xml" href="/rss.xml" title="RSS">
		</head></html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}))
	defer ts.Close()

	ctx := context.Background()
	found, err := DiscoverFeed(ctx, ts.URL)
	if err != nil {
		t.Fatalf("DiscoverFeed failed: %v", err)
	}

	expected := ts.URL + "/rss.xml"
	if found != expected {
		t.Errorf("Expected %s, got %s", expected, found)
	}
}

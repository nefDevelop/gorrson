package rss

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	MaxResponseSize = 5 * 1024 * 1024 // 5MB
	UserAgent       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	DefaultTimeout  = 30 * time.Second
)

type SecureClient struct {
	client *http.Client
}

func NewSecureClient() *SecureClient {
	return &SecureClient{
		client: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

// Fetch returns an io.ReadCloser with a limited reader to prevent OOM.
func (sc *SecureClient) Fetch(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := sc.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	// Wrap the body in a LimitReader
	return &limitedReadCloser{
		R: io.LimitReader(resp.Body, MaxResponseSize),
		C: resp.Body,
	}, nil
}

type limitedReadCloser struct {
	R io.Reader
	C io.Closer
}

func (l *limitedReadCloser) Read(p []byte) (n int, err error) {
	return l.R.Read(p)
}

func (l *limitedReadCloser) Close() error {
	return l.C.Close()
}

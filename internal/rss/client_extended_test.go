package rss

import (
	"net"
	"strings"
	"testing"
)

func TestIsPrivateIPExtended(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"127.0.0.1", true},
		{"::1", true},
		{"169.254.1.1", true},
		{"fe80::1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.0.1", true},
		{"0.0.0.0", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"142.250.80.46", false},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if ip == nil {
			t.Errorf("Failed to parse IP: %s", tt.ip)
			continue
		}
		if got := isPrivateIP(ip); got != tt.expected {
			t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, got, tt.expected)
		}
	}
}

func TestLimitedReadCloserClose(t *testing.T) {
	reader := strings.NewReader("hello")
	closed := false

	lrc := &limitedReadCloser{
		R: reader,
		C: &boolCloser{closed: &closed},
	}

	buf := make([]byte, 100)
	lrc.Read(buf)
	lrc.Close()

	if !closed {
		t.Error("Closer was not called")
	}
}

type boolCloser struct {
	closed *bool
}

func (b *boolCloser) Close() error {
	*b.closed = true
	return nil
}

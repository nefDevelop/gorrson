package rss

import (
	"io"
	"net"
	"strings"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"127.0.0.1", true},
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"::1", true},
		{"fe80::1", true},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if got := isPrivateIP(ip); got != tt.expected {
			t.Errorf("isPrivateIP(%s) = %v; want %v", tt.ip, got, tt.expected)
		}
	}
}

func TestLimitedReadCloser(t *testing.T) {
	content := "this is a long string that should be truncated"
	limit := 10
	
	reader := strings.NewReader(content)
	// We don't need a real closer for this test, just something that implements io.Closer
	lrc := &limitedReadCloser{
		R: io.LimitReader(reader, int64(limit)),
		C: io.NopCloser(reader),
	}
	defer lrc.Close()

	buf := make([]byte, 100)
	n, err := lrc.Read(buf)
	
	if n != limit {
		t.Errorf("Expected to read %d bytes, read %d", limit, n)
	}
	if err != nil && err != io.EOF {
		t.Errorf("Unexpected error: %v", err)
	}

	// Next read should be EOF
	n, err = lrc.Read(buf)
	if n != 0 || err != io.EOF {
		t.Errorf("Expected EOF, got n=%d, err=%v", n, err)
	}
}

func TestSecureClientUA(t *testing.T) {
	customUA := "Custom-Bot/1.0"
	sc := NewSecureClient(customUA)
	if sc.ua != customUA {
		t.Errorf("Expected UA %s, got %s", customUA, sc.ua)
	}

	scDefault := NewSecureClient("")
	if scDefault.ua != UserAgent {
		t.Errorf("Expected default UA %s, got %s", UserAgent, scDefault.ua)
	}
}

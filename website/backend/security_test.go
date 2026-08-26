package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIgnoresForwardedHeaderByDefault(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/auth/login", nil)
	r.RemoteAddr = "203.0.113.10:5555"
	r.Header.Set("X-Forwarded-For", "1.2.3.4")

	if got := clientIP(r); got != "203.0.113.10" {
		t.Errorf("clientIP() = %q; want %q (spoofed header harus diabaikan)", got, "203.0.113.10")
	}
}

func TestClientUsesForwardedHeaderWhenProxyTrusted(t *testing.T) {
	t.Setenv("TRUST_PROXY", "1")

	r := httptest.NewRequest(http.MethodGet, "/api/auth/login", nil)
	r.RemoteAddr = "10.0.0.1:4444"
	r.Header.Set("X-Forwarded-For", "198.51.100.7, 10.0.0.2")

	if got := clientIP(r); got != "198.51.100.7" {
		t.Errorf("clientIP() = %q; want %q (proxy tepercaya)", got, "198.51.100.7")
	}
}

func TestParseBatchIDs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"id valid", "123,456", []string{"123", "456"}},
		{"id tidak valid dibuang", "123,abc,789", []string{"123", "789"}},
		{"injeksi path dibuang", "../../private,123?x=1,42", []string{"42"}},
		{"kosong", "", nil},
		{"semua tidak valid", "a,b,c", nil},
	}

	for _, tt := range tests {
		got := parseBatchIDs(tt.input)
		if len(got) != len(tt.expected) {
			t.Errorf("%s: parseBatchIDs(%q) = %v; want %v", tt.name, tt.input, got, tt.expected)
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("%s: parseBatchIDs(%q)[%d] = %q; want %q", tt.name, tt.input, i, got[i], tt.expected[i])
			}
		}
	}
}

func TestParseBatchIDsCapsCount(t *testing.T) {
	input := ""
	for i := 1; i <= 50; i++ {
		if i > 1 {
			input += ","
		}
		input += string(rune('0' + i%10))
	}

	got := parseBatchIDs(input)
	if len(got) > maxBatchIDs {
		t.Errorf("parseBatchIDs returned %d ids; want max %d", len(got), maxBatchIDs)
	}
}

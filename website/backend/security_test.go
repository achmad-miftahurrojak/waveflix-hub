package main

import (
	"encoding/base64"
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
	t.Setenv("TRUSTED_PROXY_IPS", "10.0.0.1/32")

	r := httptest.NewRequest(http.MethodGet, "/api/auth/login", nil)
	r.RemoteAddr = "10.0.0.1:4444"
	r.Header.Set("X-Forwarded-For", "198.51.100.7, 10.0.0.2")

	if got := clientIP(r); got != "198.51.100.7" {
		t.Errorf("clientIP() = %q; want %q (proxy tepercaya)", got, "198.51.100.7")
	}
}

func TestClientIgnoresForwardedHeaderFromUntrustedProxy(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_IPS", "10.0.0.1/32")

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "203.0.113.10:4444"
	r.Header.Set("X-Forwarded-For", "198.51.100.7")

	if got := clientIP(r); got != "203.0.113.10" {
		t.Errorf("clientIP() = %q; want untrusted proxy address", got)
	}
}

func TestValidateImageDataRejectsSpoofedMime(t *testing.T) {
	data := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("not an image"))
	if _, _, err := validateImageData(data); err == nil {
		t.Fatal("validateImageData accepted non-image bytes")
	}
}

func TestWriteAuthSetsHttpOnlyCookie(t *testing.T) {
	t.Setenv("JWT_SECRET", "super-secret-key-minimum-32-characters-long")
	t.Setenv("AUTH_COOKIE_SECURE", "1")
	initJWTSecret()
	recorder := httptest.NewRecorder()
	writeAuth(recorder, 7, "user@example.com", "user", "", "", "id")
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != authCookieName || !cookies[0].HttpOnly || !cookies[0].Secure {
		t.Fatalf("auth cookie missing secure attributes: %#v", cookies)
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

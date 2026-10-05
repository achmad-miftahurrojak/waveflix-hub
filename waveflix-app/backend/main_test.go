package main

import (
	"net/url"
	"strings"
	"testing"
)

func TestNormalizeMedia(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"tv", "tv"},
		{"movie", "movie"},
		{"other", "movie"},
		{"", "movie"},
	}

	for _, tt := range tests {
		res := normalizeMedia(tt.input)
		if res != tt.expected {
			t.Errorf("normalizeMedia(%q) = %q; want %q", tt.input, res, tt.expected)
		}
	}
}

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		inputs   []string
		expected string
	}{
		{[]string{"", "hello", "world"}, "hello"},
		{[]string{"", ""}, ""},
		{[]string{"a"}, "a"},
	}

	for _, tt := range tests {
		res := firstNonEmpty(tt.inputs...)
		if res != tt.expected {
			t.Errorf("firstNonEmpty(%v) = %q; want %q", tt.inputs, res, tt.expected)
		}
	}
}

func TestJWT(t *testing.T) {
	testSecret := strings.Repeat("t", 32)
	t.Setenv("JWT_SECRET", testSecret)
	initJWTSecret()

	secret := jwtSecret()
	if string(secret) != testSecret {
		t.Errorf("jwtSecret() = %q; want %q", string(secret), testSecret)
	}

	token, err := signToken(123)
	if err != nil {
		t.Fatalf("signToken failed: %v", err)
	}
	if token == "" {
		t.Error("signToken returned empty token")
	}
}

func TestGetIDParam(t *testing.T) {
	tests := []struct {
		urlQuery string
		expected string
	}{
		{"id=123", "123"},
		{"tmdb_id=456", "456"},
		{"id=123&tmdb_id=456", "123"},
		{"id=abc", ""},
		{"id=123a", ""},
		{"id=12.3", ""},
		{"id=-123", ""},
		{"", ""},
	}

	for _, tt := range tests {
		u, _ := url.ParseQuery(tt.urlQuery)
		res := getIDParam(u)
		if res != tt.expected {
			t.Errorf("getIDParam(%q) = %q; want %q", tt.urlQuery, res, tt.expected)
		}
	}
}

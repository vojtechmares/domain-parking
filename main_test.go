package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsRegistrableApex(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"example.com", true},
		{"EXAMPLE.COM", true},
		{"www.example.com", false},
		{"example.co.uk", true},
		{"sub.example.co.uk", false},
		{"localhost", false},
		{"com", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := isRegistrableApex(tt.host); got != tt.want {
			t.Errorf("isRegistrableApex(%q) = %v, want %v", tt.host, got, tt.want)
		}
	}
}

func TestCheckHandler(t *testing.T) {
	tests := []struct {
		target string
		want   int
	}{
		{"/check?domain=example.com", http.StatusOK},
		{"/check?domain=www.example.com", http.StatusForbidden},
		{"/check", http.StatusForbidden},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, tt.target, nil)
		rec := httptest.NewRecorder()

		checkHandler(rec, req)

		if rec.Code != tt.want {
			t.Errorf("checkHandler(%q) status = %d, want %d", tt.target, rec.Code, tt.want)
		}
	}
}

func TestHandler(t *testing.T) {
	// The parking handler is permissive: it renders for any host.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "deploy-check"
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("handler status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "deploy-check") {
		t.Errorf("handler body does not contain host %q", "deploy-check")
	}
}

func TestHandlerPrefersForwardedHost(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "127.0.0.1:8080"
	req.Header.Set("X-Forwarded-Host", "example.com")
	rec := httptest.NewRecorder()

	handler(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "example.com") {
		t.Errorf("handler body does not contain forwarded host %q", "example.com")
	}
	if strings.Contains(body, "127.0.0.1") {
		t.Errorf("handler body should prefer X-Forwarded-Host over r.Host")
	}
}

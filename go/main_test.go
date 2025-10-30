// main_test.go
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- Handlers ---

func TestHomeHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	// Use the same middleware the server uses
	handler := chain(http.HandlerFunc(homeHandler), withSecurityHeaders(), withLogging())
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("home: status=%d want=%d", rr.Code, http.StatusOK)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("home: content-type=%q does not include text/html", ct)
	}
	// Optional: verify some branded text, if present in static/index.html
	if !strings.Contains(rr.Body.String(), "Harness Demo App") &&
		!strings.Contains(rr.Body.String(), "Shipt") {
		t.Log("home: skipping strict body check (branding text not found)")
	}
}

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	handler := chain(http.HandlerFunc(healthHandler), withSecurityHeaders(), withLogging())
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("healthz: status=%d want=%d", rr.Code, http.StatusOK)
	}
	if body := strings.TrimSpace(rr.Body.String()); body != `{"status":"healthy"}` {
		t.Fatalf("healthz: body=%q", body)
	}
}

func TestReadyz_Warmup(t *testing.T) {
	// Force warming state
	oldStart := startTime
	oldReadyAfter := readyAfter
	startTime = time.Now()
	readyAfter = 10 * time.Second
	defer func() {
		startTime = oldStart
		readyAfter = oldReadyAfter
	}()

	req := httptest.NewRequest("GET", "/readyz", nil)
	rr := httptest.NewRecorder()
	handler := chain(http.HandlerFunc(readyHandler), withSecurityHeaders(), withLogging())
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz(warm): status=%d want=%d", rr.Code, http.StatusServiceUnavailable)
	}
	if body := strings.TrimSpace(rr.Body.String()); body != `{"status":"warming"}` {
		t.Fatalf("readyz(warm): body=%q", body)
	}
}

func TestReadyz_Ready(t *testing.T) {
	// Make the app appear "ready"
	oldStart := startTime
	startTime = time.Now().Add(-5 * time.Second)
	defer func() { startTime = oldStart }()

	req := httptest.NewRequest("GET", "/readyz", nil)
	rr := httptest.NewRecorder()
	handler := chain(http.HandlerFunc(readyHandler), withSecurityHeaders(), withLogging())
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("readyz(ready): status=%d want=%d", rr.Code, http.StatusOK)
	}
	if body := strings.TrimSpace(rr.Body.String()); body != `{"status":"ready"}` {
		t.Fatalf("readyz(ready): body=%q", body)
	}
}

func TestVersionHandler(t *testing.T) {
	// Override metadata to deterministic values
	oldVersion, oldCommit, oldBuild := version, commit, buildTime
	version, commit, buildTime = "vTest", "abc123", "2025-01-01T00:00:00Z"
	defer func() {
		version, commit, buildTime = oldVersion, oldCommit, oldBuild
	}()

	req := httptest.NewRequest("GET", "/version", nil)
	rr := httptest.NewRecorder()
	handler := chain(http.HandlerFunc(versionHandler), withSecurityHeaders(), withLogging())
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("version: status=%d want=%d", rr.Code, http.StatusOK)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("version: content-type=%q", ct)
	}

	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("version: invalid json: %v", err)
	}
	for k, want := range map[string]string{
		"version":   "vTest",
		"commit":    "abc123",
		"buildTime": "2025-01-01T00:00:00Z",
	} {
		if got[k] != want {
			t.Fatalf("version: %s=%v want=%v", k, got[k], want)
		}
	}
}

func TestSecurityHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	handler := chain(http.HandlerFunc(healthHandler), withSecurityHeaders())
	handler.ServeHTTP(rr, req)

	check := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "no-referrer",
		"Content-Security-Policy": "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'",
	}
	for k, v := range check {
		if rr.Header().Get(k) != v {
			t.Fatalf("security header %s=%q want=%q", k, rr.Header().Get(k), v)
		}
	}
}

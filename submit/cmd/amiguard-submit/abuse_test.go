package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTrustedClientIPUsesForwardedForOnlyFromLoopback(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", nil)
	req.RemoteAddr = "203.0.113.10:12345"
	req.Header.Set("X-Forwarded-For", "198.51.100.20")
	if got := trustedClientIP(req); got != "203.0.113.10" {
		t.Fatalf("untrusted proxy header used: %q", got)
	}

	req.RemoteAddr = "127.0.0.1:12345"
	if got := trustedClientIP(req); got != "198.51.100.20" {
		t.Fatalf("trusted proxy header ignored: %q", got)
	}
}

func TestUploadAbuseGuardRateLimit(t *testing.T) {
	guard := newUploadAbuseGuard()
	guard.maxRequests = 1
	guard.window = time.Minute

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", nil)
	req.RemoteAddr = "198.51.100.30:12345"

	first := httptest.NewRecorder()
	if !guard.begin(first, req) {
		t.Fatalf("first request unexpectedly rejected: %d", first.Code)
	}
	guard.done()

	second := httptest.NewRecorder()
	if guard.begin(second, req) {
		guard.done()
		t.Fatal("second request unexpectedly allowed")
	}
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("got status %d", second.Code)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After not set")
	}
}

func TestUploadAbuseGuardConcurrencyLimit(t *testing.T) {
	guard := newUploadAbuseGuard()
	guard.slots = make(chan struct{}, 1)

	firstReq := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", nil)
	firstReq.RemoteAddr = "198.51.100.40:1111"
	first := httptest.NewRecorder()
	if !guard.begin(first, firstReq) {
		t.Fatal("first concurrent request unexpectedly rejected")
	}
	defer guard.done()

	secondReq := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", nil)
	secondReq.RemoteAddr = "198.51.100.41:2222"
	second := httptest.NewRecorder()
	if guard.begin(second, secondReq) {
		guard.done()
		t.Fatal("second concurrent request unexpectedly allowed")
	}
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("got status %d", second.Code)
	}
}

func TestUploadAbuseGuardGlobalByteBudgetAcrossClients(t *testing.T) {
	guard := newUploadAbuseGuard()
	guard.maxGlobalBytes = 100
	guard.maxRequestCharge = 100
	guard.byteWindow = time.Hour

	firstReq := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", nil)
	firstReq.RemoteAddr = "198.51.100.50:1111"
	firstReq.ContentLength = 60
	first := httptest.NewRecorder()
	if !guard.begin(first, firstReq) {
		t.Fatalf("first request unexpectedly rejected: %d", first.Code)
	}
	guard.done()

	// A different client IP must share the same global budget.
	secondReq := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", nil)
	secondReq.RemoteAddr = "203.0.113.60:2222"
	secondReq.ContentLength = 50
	second := httptest.NewRecorder()
	if guard.begin(second, secondReq) {
		guard.done()
		t.Fatal("request exceeding global byte budget unexpectedly allowed")
	}
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("got status %d", second.Code)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Fatal("Retry-After not set")
	}
}

func TestUploadAbuseGuardUnknownLengthChargedConservatively(t *testing.T) {
	guard := newUploadAbuseGuard()
	guard.maxGlobalBytes = 100
	guard.maxRequestCharge = 80

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions", nil)
	req.RemoteAddr = "198.51.100.70:3333"
	req.ContentLength = -1
	first := httptest.NewRecorder()
	if !guard.begin(first, req) {
		t.Fatalf("unknown-length request unexpectedly rejected: %d", first.Code)
	}
	guard.done()
	if guard.globalBytes != 80 {
		t.Fatalf("unknown-length charge = %d, want 80", guard.globalBytes)
	}
}

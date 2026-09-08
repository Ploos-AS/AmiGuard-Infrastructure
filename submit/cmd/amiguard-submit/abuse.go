package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultUploadRateLimitRequests = 6
	defaultUploadRateLimitWindow   = time.Minute
	defaultMaxConcurrentUploads    = 2
	defaultGlobalUploadBytes       = int64(256 << 20)
	defaultGlobalUploadByteWindow  = time.Hour
	defaultUploadBudgetCharge      = int64((16 << 20) + (128 << 10))
	maxConfigGlobalUploadBytes     = int64(8 << 30)
)

type uploadRateBucket struct {
	started time.Time
	count   int
}

type uploadAbuseGuard struct {
	mu                 sync.Mutex
	buckets            map[string]uploadRateBucket
	window             time.Duration
	maxRequests        int
	slots              chan struct{}
	byteWindow         time.Duration
	maxGlobalBytes     int64
	maxRequestCharge   int64
	globalBytes        int64
	globalBytesStarted time.Time
}

func newUploadAbuseGuard() *uploadAbuseGuard {
	maxBytes, byteWindow, err := globalUploadBudgetConfig()
	if err != nil {
		// Invalid abuse-control configuration must never silently weaken intake.
		// A zero budget fails closed with HTTP 429 until configuration is fixed.
		maxBytes = 0
		byteWindow = defaultGlobalUploadByteWindow
	}
	return &uploadAbuseGuard{
		buckets:          make(map[string]uploadRateBucket),
		window:           defaultUploadRateLimitWindow,
		maxRequests:      defaultUploadRateLimitRequests,
		slots:            make(chan struct{}, defaultMaxConcurrentUploads),
		byteWindow:       byteWindow,
		maxGlobalBytes:   maxBytes,
		maxRequestCharge: defaultUploadBudgetCharge,
	}
}

func globalUploadBudgetConfig() (int64, time.Duration, error) {
	maxBytes := defaultGlobalUploadBytes
	window := defaultGlobalUploadByteWindow

	if raw := os.Getenv("AMIGUARD_GLOBAL_UPLOAD_BYTES"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < defaultUploadBudgetCharge || value > maxConfigGlobalUploadBytes {
			return 0, 0, fmt.Errorf("AMIGUARD_GLOBAL_UPLOAD_BYTES must be between %d and %d", defaultUploadBudgetCharge, maxConfigGlobalUploadBytes)
		}
		maxBytes = value
	}
	if raw := os.Getenv("AMIGUARD_GLOBAL_UPLOAD_WINDOW_SECONDS"); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < 60 || value > 86400 {
			return 0, 0, fmt.Errorf("AMIGUARD_GLOBAL_UPLOAD_WINDOW_SECONDS must be between 60 and 86400")
		}
		window = time.Duration(value) * time.Second
	}
	return maxBytes, window, nil
}

func (g *uploadAbuseGuard) begin(w http.ResponseWriter, r *http.Request) bool {
	now := time.Now()
	client := trustedClientIP(r)

	g.mu.Lock()
	for key, bucket := range g.buckets {
		if now.Sub(bucket.started) >= 2*g.window {
			delete(g.buckets, key)
		}
	}
	bucket := g.buckets[client]
	if bucket.started.IsZero() || now.Sub(bucket.started) >= g.window {
		bucket = uploadRateBucket{started: now}
	}
	if bucket.count >= g.maxRequests {
		retryAfter := int(g.window.Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}
		g.mu.Unlock()
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		http.Error(w, "upload rate limit exceeded", http.StatusTooManyRequests)
		return false
	}

	// The byte budget is global rather than per-client, so rotating source IPs
	// cannot bypass it. Charge the declared request size (including multipart
	// overhead). Unknown/chunked sizes are conservatively charged at the
	// maximum request allowance. Rejected/invalid requests are intentionally
	// not refunded: this is an abuse budget, not accounting.
	if g.globalBytesStarted.IsZero() || now.Sub(g.globalBytesStarted) >= g.byteWindow {
		g.globalBytesStarted = now
		g.globalBytes = 0
	}
	charge := r.ContentLength
	if charge <= 0 || charge > g.maxRequestCharge {
		charge = g.maxRequestCharge
	}
	if g.maxGlobalBytes <= 0 || g.globalBytes > g.maxGlobalBytes || charge > g.maxGlobalBytes-g.globalBytes {
		retryAfter := int(g.byteWindow.Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}
		g.mu.Unlock()
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		http.Error(w, "global upload byte budget exceeded", http.StatusTooManyRequests)
		return false
	}
	g.globalBytes += charge
	bucket.count++
	g.buckets[client] = bucket
	g.mu.Unlock()

	select {
	case g.slots <- struct{}{}:
		return true
	default:
		w.Header().Set("Retry-After", "1")
		http.Error(w, "too many concurrent uploads", http.StatusTooManyRequests)
		return false
	}
}

func (g *uploadAbuseGuard) done() {
	<-g.slots
}

func trustedClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	remoteIP := net.ParseIP(host)
	if remoteIP == nil || !remoteIP.IsLoopback() {
		return host
	}

	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded == "" {
		return host
	}
	first := strings.TrimSpace(strings.Split(forwarded, ",")[0])
	if ip := net.ParseIP(first); ip != nil {
		return ip.String()
	}
	return host
}

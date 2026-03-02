package httpx

import (
	"context"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type contextKey string

const UserIDContextKey contextKey = "userID"

// SecurityHeaders applies baseline browser hardening headers to every response.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; base-uri 'self';")
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		}
		next.ServeHTTP(w, r)
	})
}

// CORS allows browser calls only from configured frontend origin.
func CORS(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && origin == allowedOrigin {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Recover catches panics so one crash does not bring down the server.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic recovered: %v", rec)
				Error(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RequestLogger prints method/path/duration for basic observability.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// Timeout enforces max processing duration for each request.
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, d, "request timeout")
	}
}

// RateLimit is a simple fixed-window per-IP limiter for abuse control.
func RateLimit(maxPerMinute int) func(http.Handler) http.Handler {
	type bucket struct {
		count       int
		windowStart time.Time
	}
	var (
		mu      sync.Mutex
		buckets = make(map[string]*bucket)
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := clientIP(r)
			if key == "" {
				key = "unknown"
			}

			now := time.Now()
			mu.Lock()
			b, ok := buckets[key]
			if !ok {
				b = &bucket{count: 0, windowStart: now}
				buckets[key] = b
			}
			// New window starts every minute. Counter resets to 0.
			if now.Sub(b.windowStart) >= time.Minute {
				b.count = 0
				b.windowStart = now
			}
			// Reject traffic if caller exceeded limit in current minute window.
			if b.count >= maxPerMinute {
				mu.Unlock()
				Error(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			b.count++
			mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

// WithUserID writes authenticated user ID into request context.
func WithUserID(r *http.Request, userID string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), UserIDContextKey, userID))
}

// UserIDFromContext retrieves user ID set by auth middleware.
func UserIDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(UserIDContextKey)
	userID, ok := v.(string)
	return userID, ok
}

// clientIP reads X-Forwarded-For first (for proxy setups), then falls back to RemoteAddr.
func clientIP(r *http.Request) string {
	if xfwd := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xfwd != "" {
		return strings.Split(xfwd, ",")[0]
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ClientIP returns the best-effort client IP (proxy-aware).
func ClientIP(r *http.Request) string {
	return clientIP(r)
}

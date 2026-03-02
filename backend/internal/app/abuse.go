package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type ipWindowLimiter struct {
	window  time.Duration
	maxHits int
	mu      sync.Mutex
	hits    map[string]*windowCounter
}

type windowCounter struct {
	windowStart time.Time
	count       int
}

type accessCodeAbuseGuard struct {
	window         time.Duration
	maxFailures    int
	lockout        time.Duration
	mu             sync.Mutex
	failureRecords map[string]*failureRecord
}

type failureRecord struct {
	windowStart time.Time
	count       int
	lockedUntil time.Time
}

func newIPWindowLimiter(window time.Duration, maxHits int) *ipWindowLimiter {
	return &ipWindowLimiter{
		window:  window,
		maxHits: maxHits,
		hits:    make(map[string]*windowCounter),
	}
}

func newAccessCodeAbuseGuard(window time.Duration, maxFailures int, lockout time.Duration) *accessCodeAbuseGuard {
	return &accessCodeAbuseGuard{
		window:         window,
		maxFailures:    maxFailures,
		lockout:        lockout,
		failureRecords: make(map[string]*failureRecord),
	}
}

func (l *ipWindowLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	item, ok := l.hits[key]
	if !ok {
		l.hits[key] = &windowCounter{windowStart: now, count: 1}
		return true
	}
	if now.Sub(item.windowStart) >= l.window {
		item.windowStart = now
		item.count = 1
		return true
	}
	if item.count >= l.maxHits {
		return false
	}
	item.count++
	return true
}

func (g *accessCodeAbuseGuard) isLocked(key string, now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	record, ok := g.failureRecords[key]
	if !ok {
		return false
	}
	return now.Before(record.lockedUntil)
}

func (g *accessCodeAbuseGuard) registerFailure(key string, now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	record, ok := g.failureRecords[key]
	if !ok {
		g.failureRecords[key] = &failureRecord{windowStart: now, count: 1}
		return false
	}
	if now.Before(record.lockedUntil) {
		return true
	}
	if now.Sub(record.windowStart) >= g.window {
		record.windowStart = now
		record.count = 1
		record.lockedUntil = time.Time{}
		return false
	}
	record.count++
	if record.count >= g.maxFailures {
		record.lockedUntil = now.Add(g.lockout)
		return true
	}
	return false
}

func (g *accessCodeAbuseGuard) clearFailures(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.failureRecords, key)
}

type turnstileVerifyResponse struct {
	Success bool `json:"success"`
}

func verifyTurnstile(ctx context.Context, secret, token, remoteIP string) (bool, error) {
	form := url.Values{}
	form.Set("secret", secret)
	form.Set("response", token)
	if strings.TrimSpace(remoteIP) != "" {
		form.Set("remoteip", strings.TrimSpace(remoteIP))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://challenges.cloudflare.com/turnstile/v0/siteverify", strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var payload turnstileVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return false, err
	}
	return payload.Success, nil
}

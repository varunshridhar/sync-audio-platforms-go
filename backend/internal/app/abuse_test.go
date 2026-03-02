package app

import (
	"testing"
	"time"
)

func TestAccessCodeAbuseGuard_LockAndClear(t *testing.T) {
	guard := newAccessCodeAbuseGuard(time.Hour, 2, 15*time.Minute)
	key := "127.0.0.1|user@example.com"
	now := time.Now().UTC()

	if guard.isLocked(key, now) {
		t.Fatalf("expected key to be initially unlocked")
	}
	if guard.registerFailure(key, now) {
		t.Fatalf("first failure should not lock")
	}
	if !guard.registerFailure(key, now.Add(time.Minute)) {
		t.Fatalf("second failure should lock")
	}
	if !guard.isLocked(key, now.Add(2*time.Minute)) {
		t.Fatalf("expected key to be locked")
	}

	guard.clearFailures(key)
	if guard.isLocked(key, now.Add(2*time.Minute)) {
		t.Fatalf("expected key to be unlocked after clear")
	}
}

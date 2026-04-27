package transport

import (
	"testing"
	"time"
)

func TestRateLimiterAllowsUnderQuota(t *testing.T) {
	rl := NewRateLimiter(3, 1)
	rl.now = func() time.Time { return time.Unix(0, 0) }

	for i := 0; i < 3; i++ {
		if !rl.Allow("hash-a", BucketRead) {
			t.Fatalf("expected hit %d to be allowed", i)
		}
	}
}

func TestRateLimiterDeniesOverQuota(t *testing.T) {
	rl := NewRateLimiter(2, 1)
	rl.now = func() time.Time { return time.Unix(0, 0) }
	rl.Allow("hash-a", BucketRead)
	rl.Allow("hash-a", BucketRead)
	if rl.Allow("hash-a", BucketRead) {
		t.Fatalf("expected over-quota to be denied")
	}
}

func TestRateLimiterIsolatesKeys(t *testing.T) {
	rl := NewRateLimiter(1, 1)
	rl.now = func() time.Time { return time.Unix(0, 0) }
	rl.Allow("hash-a", BucketRead)
	if !rl.Allow("hash-b", BucketRead) {
		t.Fatalf("limiter must not bleed across keys")
	}
}

func TestRateLimiterIsolatesBuckets(t *testing.T) {
	rl := NewRateLimiter(1, 1)
	rl.now = func() time.Time { return time.Unix(0, 0) }
	rl.Allow("hash-a", BucketRead)
	if !rl.Allow("hash-a", BucketMutate) {
		t.Fatalf("read bucket must not exhaust mutate quota")
	}
}

func TestRateLimiterSlidingWindow(t *testing.T) {
	current := time.Unix(0, 0)
	rl := NewRateLimiter(2, 2)
	rl.now = func() time.Time { return current }

	rl.Allow("hash-a", BucketRead)
	rl.Allow("hash-a", BucketRead)
	if rl.Allow("hash-a", BucketRead) {
		t.Fatalf("expected denial within the window")
	}

	current = current.Add(61 * time.Second)
	if !rl.Allow("hash-a", BucketRead) {
		t.Fatalf("expected allow after window expiry")
	}
}

func TestRateLimiterDisabledOnNonPositiveLimit(t *testing.T) {
	rl := NewRateLimiter(0, 0)
	for i := 0; i < 100; i++ {
		if !rl.Allow("hash-a", BucketRead) {
			t.Fatalf("limiter must be disabled when limit is 0")
		}
	}
}

func TestRateLimiterNilSafe(t *testing.T) {
	var rl *RateLimiter
	if !rl.Allow("hash-a", BucketRead) {
		t.Fatalf("nil limiter must allow everything")
	}
}

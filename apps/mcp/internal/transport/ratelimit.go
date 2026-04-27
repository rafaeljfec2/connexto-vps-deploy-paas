package transport

import (
	"sync"
	"time"
)

// RateLimiter implements a per-key sliding window limiter with two distinct
// quotas: one for read calls and one for mutation calls. The HTTP transport
// keys it by the SHA-256 hash of the bearer PAT so quotas survive across
// sessions for the same token without ever storing the raw secret.
//
// Implementation is intentionally simple (in-memory, sliding window) so the
// MCP can run as a single binary without Redis. Multi-replica deployments
// should front this with a CDN/Traefik plugin or migrate to a distributed
// limiter.
type RateLimiter struct {
	mu           sync.Mutex
	readPerMin   int
	mutatePerMin int
	now          func() time.Time
	buckets      map[string]*bucket
	// pruneEvery tunes how frequently Allow performs an opportunistic GC pass
	// over the buckets map. Tests may set it through NewRateLimiter; in
	// production we run a sweep every pruneEvery Allow calls to amortize cost.
	pruneEvery int
	hits       int
}

// Bucket tags the quota that should be applied to a request.
type Bucket string

const (
	BucketRead   Bucket = "read"
	BucketMutate Bucket = "mutate"
)

type bucket struct {
	read   []time.Time
	mutate []time.Time
}

// NewRateLimiter constructs a limiter that allows up to readPerMin "read"
// hits and mutatePerMin "mutate" hits per rolling minute, per key.
func NewRateLimiter(readPerMin, mutatePerMin int) *RateLimiter {
	return &RateLimiter{
		readPerMin:   readPerMin,
		mutatePerMin: mutatePerMin,
		now:          time.Now,
		buckets:      make(map[string]*bucket),
		pruneEvery:   1024,
	}
}

// Allow records a hit for the given key+bucket and returns whether the request
// is within quota. Limits of <= 0 disable that bucket (always allow).
func (r *RateLimiter) Allow(key string, kind Bucket) bool {
	if r == nil {
		return true
	}
	limit := r.readPerMin
	if kind == BucketMutate {
		limit = r.mutatePerMin
	}
	if limit <= 0 {
		return true
	}

	now := r.now()
	cutoff := now.Add(-time.Minute)

	r.mu.Lock()
	defer r.mu.Unlock()

	b, ok := r.buckets[key]
	if !ok {
		b = &bucket{}
		r.buckets[key] = b
	}

	// Increment GC counter BEFORE the quota check so a flood of denied
	// requests still amortizes the prune sweep. Otherwise a busy attacker
	// hammering an over-quota token would never trigger gc and the bucket
	// map would grow unbounded.
	r.hits++
	if r.pruneEvery > 0 && r.hits%r.pruneEvery == 0 {
		r.gc(cutoff)
	}

	hits := r.window(b, kind)
	*hits = pruneOlderThan(*hits, cutoff)
	if len(*hits) >= limit {
		return false
	}
	*hits = append(*hits, now)
	return true
}

// gc drops bucket entries whose read AND mutate windows are both empty after
// pruning. Caller MUST hold r.mu. The sweep is O(n) but only runs once every
// pruneEvery Allow calls so amortized cost is constant.
func (r *RateLimiter) gc(cutoff time.Time) {
	for key, b := range r.buckets {
		b.read = pruneOlderThan(b.read, cutoff)
		b.mutate = pruneOlderThan(b.mutate, cutoff)
		if len(b.read) == 0 && len(b.mutate) == 0 {
			delete(r.buckets, key)
		}
	}
}

// Reset drops accumulated state for tests.
func (r *RateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buckets = make(map[string]*bucket)
}

func (r *RateLimiter) window(b *bucket, kind Bucket) *[]time.Time {
	if kind == BucketMutate {
		return &b.mutate
	}
	return &b.read
}

func pruneOlderThan(hits []time.Time, cutoff time.Time) []time.Time {
	idx := 0
	for idx < len(hits) && hits[idx].Before(cutoff) {
		idx++
	}
	if idx == 0 {
		return hits
	}
	return append([]time.Time{}, hits[idx:]...)
}

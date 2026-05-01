package tools

import (
	"testing"
	"time"
)

func TestNormaliseSince_EmptyReturnsEmpty(t *testing.T) {
	got, err := normaliseSince("", time.Now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestNormaliseSince_RFC3339IsCanonicalisedToUTC(t *testing.T) {
	got, err := normaliseSince("2026-04-30T16:00:00-03:00", time.Now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "2026-04-30T19:00:00Z"
	if got != want {
		t.Fatalf("want %q got %q", want, got)
	}
}

func TestNormaliseSince_DurationShorthandComputesAgainstClock(t *testing.T) {
	now := time.Date(2026, 4, 30, 19, 0, 0, 0, time.UTC)
	got, err := normaliseSince("1h", func() time.Time { return now })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "2026-04-30T18:00:00Z"
	if got != want {
		t.Fatalf("want %q got %q", want, got)
	}
}

func TestNormaliseSince_DurationCompound(t *testing.T) {
	now := time.Date(2026, 4, 30, 19, 0, 0, 0, time.UTC)
	got, err := normaliseSince("1h30m", func() time.Time { return now })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "2026-04-30T17:30:00Z"
	if got != want {
		t.Fatalf("want %q got %q", want, got)
	}
}

func TestNormaliseSince_RejectsInvalid(t *testing.T) {
	cases := []string{"yesterday", "2026-04-30", "not-a-time", "1d", "1m1d"}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			if _, err := normaliseSince(raw, time.Now); err == nil {
				t.Fatalf("expected error for %q", raw)
			}
		})
	}
}

func TestNormaliseSince_RejectsNonPositiveDuration(t *testing.T) {
	cases := []string{"0s", "-1h"}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			if _, err := normaliseSince(raw, time.Now); err == nil {
				t.Fatalf("expected error for %q", raw)
			}
		})
	}
}

package health

import (
	"testing"
	"time"
)

func TestProbeDeadline_UsesStartPeriodWhenLargerThanMinimum(t *testing.T) {
	now := time.Date(2026, 10, 27, 18, 4, 7, 0, time.UTC)
	got := ProbeDeadline(now, "120s", 15*time.Second, 2*time.Minute)
	want := now.Add(120*time.Second + 2*time.Minute)
	if !got.Equal(want) {
		t.Fatalf("deadline = %s, want %s", got, want)
	}
}

func TestProbeDeadline_KeepsMinimumGraceWhenStartPeriodIsShorter(t *testing.T) {
	now := time.Date(2026, 10, 27, 18, 4, 7, 0, time.UTC)
	got := ProbeDeadline(now, "10s", 15*time.Second, 2*time.Minute)
	want := now.Add(15*time.Second + 2*time.Minute)
	if !got.Equal(want) {
		t.Fatalf("deadline = %s, want %s", got, want)
	}
}

func TestProbeDeadline_IgnoresUnparseableStartPeriod(t *testing.T) {
	now := time.Date(2026, 10, 27, 18, 4, 7, 0, time.UTC)
	got := ProbeDeadline(now, "2 minutes", 15*time.Second, 2*time.Minute)
	want := now.Add(15*time.Second + 2*time.Minute)
	if !got.Equal(want) {
		t.Fatalf("deadline = %s, want %s", got, want)
	}
}

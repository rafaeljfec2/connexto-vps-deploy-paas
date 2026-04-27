package toolkit

import (
	"strings"
	"testing"
)

func TestEnsureDestructiveCommitDryRunAcceptsAnyReason(t *testing.T) {
	if err := EnsureDestructiveCommit(DryRunOptions{Commit: false, Reason: ""}); err != nil {
		t.Fatalf("dry-run with empty reason should be accepted, got %v", err)
	}
	if err := EnsureDestructiveCommit(DryRunOptions{Commit: false, Reason: "x"}); err != nil {
		t.Fatalf("dry-run with short reason should be accepted, got %v", err)
	}
}

func TestEnsureDestructiveCommitRejectsTooShort(t *testing.T) {
	for _, reason := range []string{"", "1234567", "       "} {
		if err := EnsureDestructiveCommit(DryRunOptions{Commit: true, Reason: reason}); err == nil {
			t.Fatalf("expected reason %q to be rejected", reason)
		}
	}
}

func TestEnsureDestructiveCommitAcceptsBoundaries(t *testing.T) {
	if err := EnsureDestructiveCommit(DryRunOptions{Commit: true, Reason: "12345678"}); err != nil {
		t.Fatalf("8-rune reason must be accepted, got %v", err)
	}
	max := strings.Repeat("a", 500)
	if err := EnsureDestructiveCommit(DryRunOptions{Commit: true, Reason: max}); err != nil {
		t.Fatalf("500-rune reason must be accepted, got %v", err)
	}
}

func TestEnsureDestructiveCommitRejectsTooLong(t *testing.T) {
	tooLong := strings.Repeat("a", 501)
	if err := EnsureDestructiveCommit(DryRunOptions{Commit: true, Reason: tooLong}); err == nil {
		t.Fatalf("501-rune reason must be rejected")
	}
}

func TestEnsureDestructiveCommitTrimsWhitespace(t *testing.T) {
	if err := EnsureDestructiveCommit(DryRunOptions{Commit: true, Reason: "        "}); err == nil {
		t.Fatalf("whitespace-only reason must be rejected after trimming")
	}
	if err := EnsureDestructiveCommit(DryRunOptions{Commit: true, Reason: "   12345678   "}); err != nil {
		t.Fatalf("padded 8-rune reason must be accepted, got %v", err)
	}
}

func TestEnsureDestructiveCommitCountsRunesNotBytes(t *testing.T) {
	reason := "açãoção"
	if err := EnsureDestructiveCommit(DryRunOptions{Commit: true, Reason: reason}); err == nil {
		t.Fatalf("7-rune reason (multi-byte) must be rejected")
	}
	reason8 := "açãoçãot"
	if err := EnsureDestructiveCommit(DryRunOptions{Commit: true, Reason: reason8}); err != nil {
		t.Fatalf("8-rune reason (multi-byte) must be accepted, got %v", err)
	}
}

func TestDestructiveHeadersOmitsReasonOnDryRun(t *testing.T) {
	headers := DestructiveHeaders(DryRunOptions{Commit: false, Reason: ""})
	if got := headers[HeaderDryRun]; got != "true" {
		t.Errorf("expected X-Dry-Run=true on dry-run, got %q", got)
	}
	if _, ok := headers[HeaderActionReason]; ok {
		t.Errorf("X-Action-Reason must be absent on empty reason")
	}
}

func TestDestructiveHeadersTrimsReason(t *testing.T) {
	headers := DestructiveHeaders(DryRunOptions{Commit: true, Reason: "   manual cleanup   "})
	if _, ok := headers[HeaderDryRun]; ok {
		t.Errorf("X-Dry-Run must be absent on commit")
	}
	if got := headers[HeaderActionReason]; got != "manual cleanup" {
		t.Errorf("expected trimmed reason, got %q", got)
	}
}

package docker

import (
	"errors"
	"strings"
	"testing"
)

func TestBlocksHealthProbe_RestartingExitedAndDead(t *testing.T) {
	for _, status := range []string{"restarting", "RESTARTING", "exited", "dead"} {
		if !BlocksHealthProbe(status) {
			t.Fatalf("expected %q to block the health probe", status)
		}
	}
}

func TestBlocksHealthProbe_RunningDoesNotBlock(t *testing.T) {
	for _, status := range []string{"running", "created", "not_found", ""} {
		if BlocksHealthProbe(status) {
			t.Fatalf("expected %q to keep probing", status)
		}
	}
}

func TestHealthBlockedErrorReportsLogReadFailure(t *testing.T) {
	err := &HealthBlockedError{
		Name:   "api",
		Status: "exited",
		LogErr: errors.New("docker logs failed"),
	}
	got := err.Error()
	if !strings.Contains(got, "container api is exited") || !strings.Contains(got, "docker logs failed") {
		t.Fatalf("missing log read error: %s", got)
	}
}

func TestHealthBlockedErrorIncludesLogTail(t *testing.T) {
	err := &HealthBlockedError{
		Name:   "api",
		Status: "restarting",
		Logs:   "line-1\nline-2\nboom\n",
	}
	got := err.Error()
	if !strings.Contains(got, "container api is restarting") || !strings.Contains(got, "boom") {
		t.Fatalf("missing status or logs: %s", got)
	}
}

func TestTailLogLines_KeepsLastLines(t *testing.T) {
	got := TailLogLines("a\nb\nc\nd\n", 2)
	if got != "c\nd" {
		t.Fatalf("tail = %q", got)
	}
}

func TestAnnotateHealthError_LeavesBlockedErrorUntouched(t *testing.T) {
	blocked := &HealthBlockedError{Name: "api", Status: "exited", Logs: "migration failed"}
	got := (&Client{}).AnnotateHealthError(t.Context(), "api", blocked)
	if !errors.As(got, &blocked) {
		t.Fatalf("expected blocked error, got %v", got)
	}
	if !strings.Contains(got.Error(), "migration failed") {
		t.Fatalf("logs dropped: %s", got.Error())
	}
}

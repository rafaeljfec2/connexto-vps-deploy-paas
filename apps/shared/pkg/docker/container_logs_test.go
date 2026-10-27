package docker

import (
	"errors"
	"strings"
	"testing"
)

func TestMergeContainerLogStreams_KeepsStderrCrashWithErrorWord(t *testing.T) {
	stderr := "Error: Nest can't resolve dependencies of the NotificationsService"
	got := mergeContainerLogStreams("", stderr)
	if got != stderr {
		t.Fatalf("discarded crash stderr: %q", got)
	}
}

func TestMergeContainerLogStreams_OrdersByTimestamp(t *testing.T) {
	stdout := "2026-10-27T19:21:40.000000000Z ready\n2026-10-27T19:21:42.000000000Z still starting"
	stderr := "2026-10-27T19:21:41.000000000Z Error: Nest can't resolve dependencies"
	got := mergeContainerLogStreams(stdout, stderr)
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), got)
	}
	if !strings.Contains(lines[0], "ready") || !strings.Contains(lines[1], "Error: Nest") || !strings.Contains(lines[2], "still starting") {
		t.Fatalf("lines not chronological: %#v", lines)
	}
}

func TestContainerLogsFromExec_FailedCommandIsError(t *testing.T) {
	logs, err := containerLogsFromExec(
		"api",
		"",
		"Error: No such container: api",
		errors.New("exit status 1"),
	)
	if err == nil {
		t.Fatal("expected docker failure to stay an error")
	}
	if logs != "" {
		t.Fatalf("failed command must not become log text: %q", logs)
	}
	if !strings.Contains(err.Error(), "container not found: api") {
		t.Fatalf("expected not-found wrapping, got %v", err)
	}
}

func TestContainerLogsFromExec_GenericFailure(t *testing.T) {
	logs, err := containerLogsFromExec("api", "", "permission denied", errors.New("exit status 1"))
	if err == nil || logs != "" {
		t.Fatalf("expected failed get logs, got logs=%q err=%v", logs, err)
	}
	if !strings.Contains(err.Error(), "failed to get container logs") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContainerLogsFromExec_SuccessMergesCrashStderr(t *testing.T) {
	logs, err := containerLogsFromExec(
		"api",
		"2026-10-27T19:21:39.000000000Z boot",
		"2026-10-27T19:21:40.000000000Z Error: Nest can't resolve dependencies",
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(logs, "Error: Nest can't resolve dependencies") || !strings.Contains(logs, "boot") {
		t.Fatalf("missing merged streams: %q", logs)
	}
}

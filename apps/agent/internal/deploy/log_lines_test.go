package deploy

import (
	"strings"
	"testing"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

func TestFirstNonEmptyLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "single line", in: "container api is restarting", want: "container api is restarting"},
		{name: "multiline prefers first", in: "container api is restarting\n2026-01-01T12:00:00.1Z boom\n", want: "container api is restarting"},
		{name: "leading blank", in: "\n\nfirst real\nsecond", want: "first real"},
		{name: "empty", in: "   \n  ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := firstNonEmptyLine(tt.in)
			if got != tt.want {
				t.Fatalf("firstNonEmptyLine(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSplitErrorLogLines_StripsDockerTimestamps(t *testing.T) {
	t.Parallel()

	message := strings.Join([]string{
		"container api is restarting",
		"2026-04-30T19:58:01.123456789Z Error: Cannot find module 'express'",
		"2026-04-30T19:58:01.223456789Z     at Module._resolveFilename",
		"",
		"   ",
	}, "\n")

	got := splitErrorLogLines(message)
	want := []string{
		"container api is restarting",
		"Error: Cannot find module 'express'",
		"    at Module._resolveFilename",
	}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestEmitErrorLines_EmitsOneEntryPerLine(t *testing.T) {
	t.Parallel()

	var messages []string
	emit := func(_ pb.DeployStage, _ pb.DeployLogLevel, message string) {
		messages = append(messages, message)
	}

	emitErrorLines(
		emit,
		pb.DeployStage_DEPLOY_STAGE_HEALTH_CHECK,
		pb.DeployLogLevel_DEPLOY_LOG_LEVEL_ERROR,
		"line one\n2026-04-30T19:58:01.1Z line two\n\nline three",
	)

	want := []string{"line one", "line two", "line three"}
	if len(messages) != len(want) {
		t.Fatalf("emitted %d lines, want %d: %#v", len(messages), len(want), messages)
	}
	for i := range want {
		if messages[i] != want[i] {
			t.Errorf("line %d: got %q want %q", i, messages[i], want[i])
		}
	}
}

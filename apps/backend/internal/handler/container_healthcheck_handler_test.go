package handler

import (
	"reflect"
	"testing"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"github.com/paasdeploy/shared/pkg/docker"
)

func TestMapAgentHealthcheckResponseCopiesAllFields(t *testing.T) {
	resp := &pb.RunContainerHealthcheckResponse{
		Success:    true,
		ExitCode:   0,
		Stdout:     "ok",
		Stderr:     "",
		Command:    []string{"curl", "-f", "http://localhost"},
		DurationMs: 123,
	}

	got := mapAgentHealthcheckResponse(resp)

	if !got.Success {
		t.Fatal("expected Success=true")
	}
	if got.ExitCode != 0 {
		t.Fatalf("expected ExitCode=0, got %d", got.ExitCode)
	}
	if got.Stdout != "ok" {
		t.Fatalf("expected Stdout=ok, got %q", got.Stdout)
	}
	if !reflect.DeepEqual(got.Command, []string{"curl", "-f", "http://localhost"}) {
		t.Fatalf("unexpected Command: %v", got.Command)
	}
	if got.DurationMs != 123 {
		t.Fatalf("expected DurationMs=123, got %d", got.DurationMs)
	}
}

func TestMapAgentHealthcheckResponsePropagatesFailure(t *testing.T) {
	resp := &pb.RunContainerHealthcheckResponse{
		Success:  false,
		ExitCode: 1,
		Stderr:   "boom",
	}

	got := mapAgentHealthcheckResponse(resp)

	if got.Success {
		t.Fatal("expected Success=false")
	}
	if got.ExitCode != 1 {
		t.Fatalf("expected ExitCode=1, got %d", got.ExitCode)
	}
	if got.Stderr != "boom" {
		t.Fatalf("expected Stderr=boom, got %q", got.Stderr)
	}
}

func TestMapLocalHealthcheckResultMarksSuccessForExitZero(t *testing.T) {
	result := &docker.HealthcheckResult{
		ExitCode:   0,
		Stdout:     "pong",
		Stderr:     "",
		Command:    []string{"ping", "-c", "1"},
		DurationMs: 42,
	}

	got := mapLocalHealthcheckResult(result)

	if !got.Success {
		t.Fatal("expected Success=true for exit code 0")
	}
	if got.ExitCode != 0 {
		t.Fatalf("expected ExitCode=0, got %d", got.ExitCode)
	}
	if got.Stdout != "pong" {
		t.Fatalf("expected Stdout=pong, got %q", got.Stdout)
	}
	if got.DurationMs != 42 {
		t.Fatalf("expected DurationMs=42, got %d", got.DurationMs)
	}
}

func TestMapLocalHealthcheckResultMarksFailureForNonZeroExit(t *testing.T) {
	result := &docker.HealthcheckResult{
		ExitCode: 1,
		Stderr:   "no route to host",
	}

	got := mapLocalHealthcheckResult(result)

	if got.Success {
		t.Fatal("expected Success=false for non-zero exit code")
	}
	if got.ExitCode != 1 {
		t.Fatalf("expected ExitCode=1, got %d", got.ExitCode)
	}
	if got.Stderr != "no route to host" {
		t.Fatalf("expected Stderr=no route to host, got %q", got.Stderr)
	}
}

func TestMapLocalHealthcheckResultPreservesNegativeExitFromTimeout(t *testing.T) {
	result := &docker.HealthcheckResult{
		ExitCode: -1,
		Stderr:   "healthcheck timed out after 30s",
	}

	got := mapLocalHealthcheckResult(result)

	if got.Success {
		t.Fatal("expected Success=false for timeout")
	}
	if got.ExitCode != -1 {
		t.Fatalf("expected ExitCode=-1, got %d", got.ExitCode)
	}
}

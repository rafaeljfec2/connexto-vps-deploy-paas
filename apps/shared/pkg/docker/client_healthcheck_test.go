package docker

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestHealthcheckIsConfiguredWithCMD(t *testing.T) {
	if !healthcheckIsConfigured([]string{"CMD", "curl", "-f", "http://localhost"}) {
		t.Fatal("expected CMD healthcheck to be configured")
	}
}

func TestHealthcheckIsConfiguredWithCMDShell(t *testing.T) {
	if !healthcheckIsConfigured([]string{"CMD-SHELL", "curl -f http://localhost"}) {
		t.Fatal("expected CMD-SHELL healthcheck to be configured")
	}
}

func TestHealthcheckIsConfiguredWithEmpty(t *testing.T) {
	if healthcheckIsConfigured([]string{}) {
		t.Fatal("expected empty healthcheck to not be configured")
	}
}

func TestHealthcheckIsConfiguredWithNone(t *testing.T) {
	if healthcheckIsConfigured([]string{"NONE"}) {
		t.Fatal("expected NONE healthcheck to not be configured")
	}
}

func TestHealthcheckIsConfiguredWithLowercaseNone(t *testing.T) {
	if healthcheckIsConfigured([]string{"none"}) {
		t.Fatal("expected lowercase none healthcheck to not be configured")
	}
}

func TestBuildHealthcheckExecArgsCMD(t *testing.T) {
	args, err := buildHealthcheckExecArgs("abc123", []string{"CMD", "curl", "-f", "http://localhost"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"exec", "abc123", "curl", "-f", "http://localhost"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("expected %v, got %v", want, args)
	}
}

func TestBuildHealthcheckExecArgsCMDShell(t *testing.T) {
	args, err := buildHealthcheckExecArgs("abc123", []string{"CMD-SHELL", "curl -f http://localhost || exit 1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"exec", "abc123", "sh", "-c", "curl -f http://localhost || exit 1"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("expected %v, got %v", want, args)
	}
}

func TestBuildHealthcheckExecArgsCMDWithoutCommand(t *testing.T) {
	_, err := buildHealthcheckExecArgs("abc123", []string{"CMD"})
	if err == nil {
		t.Fatal("expected error for CMD without command")
	}
}

func TestBuildHealthcheckExecArgsCMDShellWithoutScript(t *testing.T) {
	_, err := buildHealthcheckExecArgs("abc123", []string{"CMD-SHELL"})
	if err == nil {
		t.Fatal("expected error for CMD-SHELL without script")
	}
}

func TestBuildHealthcheckExecArgsEmpty(t *testing.T) {
	_, err := buildHealthcheckExecArgs("abc123", []string{})
	if err == nil {
		t.Fatal("expected error for empty test")
	}
}

func TestBuildHealthcheckExecArgsLegacyShellForm(t *testing.T) {
	args, err := buildHealthcheckExecArgs("abc123", []string{"echo", "ok"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"exec", "abc123", "echo", "ok"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("expected %v, got %v", want, args)
	}
}

func TestHealthcheckUserCommandStripsCMD(t *testing.T) {
	got := healthcheckUserCommand([]string{"CMD", "curl", "-f"})
	want := []string{"curl", "-f"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestHealthcheckUserCommandStripsCMDShell(t *testing.T) {
	got := healthcheckUserCommand([]string{"CMD-SHELL", "curl -f"})
	want := []string{"curl -f"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestHealthcheckUserCommandReturnsLegacyAsIs(t *testing.T) {
	got := healthcheckUserCommand([]string{"echo", "ok"})
	want := []string{"echo", "ok"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestHealthcheckTimeoutUsesDefaultWhenZero(t *testing.T) {
	got := healthcheckTimeout(0)
	if got != defaultHealthTimeout {
		t.Fatalf("expected %v, got %v", defaultHealthTimeout, got)
	}
}

func TestHealthcheckTimeoutUsesProvidedDuration(t *testing.T) {
	got := healthcheckTimeout(int64(5 * time.Second))
	if got != 5*time.Second {
		t.Fatalf("expected 5s, got %v", got)
	}
}

func TestHealthcheckNotConfiguredErrorMessage(t *testing.T) {
	err := &HealthcheckNotConfiguredError{ContainerID: "abc"}
	if err.Error() == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestIsHealthcheckTimeoutFalseForNil(t *testing.T) {
	if isHealthcheckTimeout(nil) {
		t.Fatal("expected false for nil error")
	}
}

func TestIsHealthcheckTimeoutTrueForTimeoutMessage(t *testing.T) {
	err := errors.New("command timed out after 30s")
	if !isHealthcheckTimeout(err) {
		t.Fatal("expected true for timeout error")
	}
}

func TestIsHealthcheckTimeoutFalseForOtherError(t *testing.T) {
	err := errors.New("command failed with exit code 1")
	if isHealthcheckTimeout(err) {
		t.Fatal("expected false for non-timeout error")
	}
}

func TestAppendTimeoutMessageWhenStderrEmpty(t *testing.T) {
	got := appendTimeoutMessage("", 5*time.Second)
	want := "healthcheck timed out after 5s"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestAppendTimeoutMessageWhenStderrPresent(t *testing.T) {
	got := appendTimeoutMessage("partial output", 2*time.Second)
	want := "partial output\nhealthcheck timed out after 2s"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

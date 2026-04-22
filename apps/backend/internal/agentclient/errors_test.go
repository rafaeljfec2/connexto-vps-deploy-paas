package agentclient

import (
	"errors"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestWrapUnimplementedReturnsNilForNil(t *testing.T) {
	if got := wrapUnimplemented("Foo", nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestWrapUnimplementedWrapsGrpcUnimplemented(t *testing.T) {
	grpcErr := status.Error(codes.Unimplemented, "unknown method")
	wrapped := wrapUnimplemented("Foo", grpcErr)

	var unimpl *UnimplementedError
	if !errors.As(wrapped, &unimpl) {
		t.Fatalf("expected UnimplementedError, got %T", wrapped)
	}
	if unimpl.Method != "Foo" {
		t.Fatalf("expected method Foo, got %s", unimpl.Method)
	}
}

func TestWrapUnimplementedDoesNotWrapOtherCodes(t *testing.T) {
	grpcErr := status.Error(codes.NotFound, "missing")
	wrapped := wrapUnimplemented("Foo", grpcErr)

	var unimpl *UnimplementedError
	if errors.As(wrapped, &unimpl) {
		t.Fatal("expected non-Unimplemented error to pass through")
	}
}

func TestWrapUnimplementedDetectsUnimplementedThroughWrap(t *testing.T) {
	grpcErr := status.Error(codes.Unimplemented, "unknown")
	wrapped := wrapUnimplemented("Foo", fmt.Errorf("call failed: %w", grpcErr))

	if !IsUnimplemented(wrapped) {
		t.Fatal("expected IsUnimplemented to return true")
	}
}

func TestIsUnimplementedReturnsFalseForNil(t *testing.T) {
	if IsUnimplemented(nil) {
		t.Fatal("expected false for nil error")
	}
}

func TestUnimplementedErrorMessageIncludesMethod(t *testing.T) {
	err := &UnimplementedError{Method: "ConnectContainerNetwork"}
	if got := err.Error(); !contains(got, "ConnectContainerNetwork") {
		t.Fatalf("expected error message to mention method, got %q", got)
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

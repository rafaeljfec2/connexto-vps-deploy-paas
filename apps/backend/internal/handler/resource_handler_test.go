package handler

import (
	"errors"
	"testing"

	"github.com/paasdeploy/backend/internal/agentclient"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsUnimplementedAgentErrFalseForNil(t *testing.T) {
	if isUnimplementedAgentErr(nil) {
		t.Fatal("expected false for nil error")
	}
}

func TestIsUnimplementedAgentErrFalseForGenericError(t *testing.T) {
	if isUnimplementedAgentErr(errors.New("boom")) {
		t.Fatal("expected false for generic error")
	}
}

func TestIsUnimplementedAgentErrTrueForWrappedUnimplemented(t *testing.T) {
	err := &agentclient.UnimplementedError{Method: "Foo"}
	if !isUnimplementedAgentErr(err) {
		t.Fatal("expected true for UnimplementedError")
	}
}

func TestIsUnimplementedAgentErrTrueForGrpcUnimplemented(t *testing.T) {
	err := status.Error(codes.Unimplemented, "missing")
	if !isUnimplementedAgentErr(err) {
		t.Fatal("expected true for grpc Unimplemented status")
	}
}

func TestIsUnimplementedAgentErrFalseForGrpcUnknown(t *testing.T) {
	err := status.Error(codes.Unknown, "boom")
	if isUnimplementedAgentErr(err) {
		t.Fatal("expected false for non-Unimplemented grpc error")
	}
}

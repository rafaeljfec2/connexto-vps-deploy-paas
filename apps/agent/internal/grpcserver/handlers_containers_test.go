package grpcserver

import (
	"testing"
	"time"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestExtractSinceFromRequest_NilRequestReturnsNil(t *testing.T) {
	if got := extractSinceFromRequest(nil); got != nil {
		t.Fatalf("expected nil for nil request, got %v", got)
	}
}

func TestExtractSinceFromRequest_UnsetSinceReturnsNil(t *testing.T) {
	req := &pb.ContainerLogsRequest{ContainerId: "x"}
	if got := extractSinceFromRequest(req); got != nil {
		t.Fatalf("expected nil for unset Since, got %v", got)
	}
}

func TestExtractSinceFromRequest_PresentSinceReturnsSameInstant(t *testing.T) {
	want := time.Date(2026, 4, 30, 19, 0, 0, 0, time.UTC)
	req := &pb.ContainerLogsRequest{
		ContainerId: "x",
		Since:       timestamppb.New(want),
	}
	got := extractSinceFromRequest(req)
	if got == nil {
		t.Fatalf("expected non-nil time, got nil")
	}
	if !got.Equal(want) {
		t.Fatalf("instant mismatch\nwant: %s\n got: %s", want, got)
	}
	// timestamppb.AsTime always materialises in UTC; make that contract explicit
	// so a future regression that introduces a TZ shift is caught here.
	if got.Location() != time.UTC {
		t.Fatalf("expected UTC location, got %s", got.Location())
	}
}

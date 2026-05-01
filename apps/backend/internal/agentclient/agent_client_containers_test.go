package agentclient

import (
	"testing"
	"time"
)

func TestBuildContainerLogsRequest_OmitsSinceWhenNil(t *testing.T) {
	req := buildContainerLogsRequest("abc", 50, false, nil)

	if req.ContainerId != "abc" {
		t.Errorf("container id mismatch: got %q", req.ContainerId)
	}
	if req.Tail != 50 {
		t.Errorf("tail mismatch: got %d", req.Tail)
	}
	if req.Follow {
		t.Errorf("expected Follow=false")
	}
	if !req.Timestamps {
		t.Errorf("expected Timestamps=true")
	}
	if req.Since != nil {
		t.Errorf("expected Since=nil when caller passed nil, got %v", req.Since)
	}
}

func TestBuildContainerLogsRequest_PropagatesSinceAsUTC(t *testing.T) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Skipf("timezone data unavailable: %v", err)
	}
	since := time.Date(2026, 4, 30, 16, 0, 0, 0, loc)

	req := buildContainerLogsRequest("abc", 100, true, &since)

	if req.Since == nil {
		t.Fatalf("expected Since to be set")
	}
	got := req.Since.AsTime()
	want := time.Date(2026, 4, 30, 19, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("Since not normalised to UTC\nwant: %s\n got: %s", want, got)
	}
	if !req.Follow {
		t.Errorf("expected Follow=true")
	}
}

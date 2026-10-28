package engine

import (
	"testing"
	"time"
)

func TestChannelNotifier_LifecycleBlocksWhenFull(t *testing.T) {
	n := NewChannelNotifier(1)
	n.EmitLog("d", "a", "fill")

	done := make(chan struct{})
	go func() {
		n.EmitDeployQueued("deploy-1", "app-1")
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("QUEUED must block while buffer is full of telemetry")
	case <-time.After(30 * time.Millisecond):
	}

	first := <-n.Events()
	if first.Type != EventTypeLog {
		t.Fatalf("first event = %s, want LOG", first.Type)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("QUEUED did not unblock after drain")
	}

	queued := <-n.Events()
	if queued.Type != EventTypeQueued {
		t.Fatalf("second event = %s, want QUEUED", queued.Type)
	}
	if queued.DeployID != "deploy-1" || queued.AppID != "app-1" {
		t.Fatalf("queued ids = %s/%s", queued.DeployID, queued.AppID)
	}
}

func TestChannelNotifier_TelemetryDropsWhenFull(t *testing.T) {
	n := NewChannelNotifier(1)
	n.EmitLog("d", "a", "keep")
	n.EmitLog("d", "a", "drop")
	n.EmitStats("a", StatsData{CPUPercent: 1})

	if got := len(n.events); got != 1 {
		t.Fatalf("buffered events = %d, want 1", got)
	}
	ev := <-n.Events()
	if ev.Type != EventTypeLog || ev.Message != "keep" {
		t.Fatalf("kept event = %+v", ev)
	}
	select {
	case ev := <-n.Events():
		t.Fatalf("unexpected extra event: %+v", ev)
	default:
	}
}

package agentclient

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pb "github.com/paasdeploy/backend/gen/go/flowdeploy/v1"
)

type scriptedReceiver struct {
	mu    sync.Mutex
	queue []recvResult
}

type recvResult struct {
	entry *pb.DeployLogEntry
	err   error
}

func (r *scriptedReceiver) Recv() (*pb.DeployLogEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.queue) == 0 {
		return nil, io.EOF
	}
	next := r.queue[0]
	r.queue = r.queue[1:]
	return next.entry, next.err
}

func TestStreamDeployLogs_RetriesOnRecvErrorThenEOF(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var opens atomic.Int32
	ready := make(chan struct{})
	var notices []string
	var mu sync.Mutex
	onLog := func(entry *pb.DeployLogEntry) {
		mu.Lock()
		defer mu.Unlock()
		notices = append(notices, entry.Message)
	}

	open := func(context.Context, string) (logEntryReceiver, error) {
		n := opens.Add(1)
		if n == 1 {
			return &scriptedReceiver{queue: []recvResult{
				{entry: &pb.DeployLogEntry{Message: "line-1"}},
				{err: errors.New("stream reset")},
			}}, nil
		}
		return &scriptedReceiver{queue: []recvResult{
			{entry: &pb.DeployLogEntry{Message: "line-2"}},
			{err: io.EOF},
		}}, nil
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		streamDeployLogs(ctx, "dep-1", open, onLog, ready, 10*time.Millisecond)
	}()

	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("ready not signalled")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("streamDeployLogs did not finish after EOF")
	}

	if opens.Load() < 2 {
		t.Fatalf("expected reattach after Recv error, opens=%d", opens.Load())
	}

	mu.Lock()
	defer mu.Unlock()
	joined := notices
	foundInterrupt := false
	foundReconnect := false
	foundLine1 := false
	foundLine2 := false
	for _, m := range joined {
		switch m {
		case "Deploy log stream interrupted, reconnecting...":
			foundInterrupt = true
		case "Deploy log stream reconnected":
			foundReconnect = true
		case "line-1":
			foundLine1 = true
		case "line-2":
			foundLine2 = true
		}
	}
	if !foundLine1 || !foundLine2 {
		t.Fatalf("missing log lines: %v", joined)
	}
	if !foundInterrupt || !foundReconnect {
		t.Fatalf("missing reconnect notices: %v", joined)
	}
}

func TestStreamDeployLogs_OpenFailureSignalsReadyAndRetries(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var opens atomic.Int32
	ready := make(chan struct{})
	var sawFailNotice atomic.Bool
	onLog := func(entry *pb.DeployLogEntry) {
		if entry.Message == "Deploy log stream failed to open, retrying..." {
			sawFailNotice.Store(true)
		}
	}

	open := func(context.Context, string) (logEntryReceiver, error) {
		n := opens.Add(1)
		if n == 1 {
			return nil, errors.New("unavailable")
		}
		cancel() // stop after second attempt
		return nil, errors.New("unavailable")
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		streamDeployLogs(ctx, "dep-2", open, onLog, ready, 10*time.Millisecond)
	}()

	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("ready must fire even when first open fails")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("expected exit on ctx cancel")
	}

	if !sawFailNotice.Load() {
		t.Fatal("expected open-failure notice")
	}
	if opens.Load() < 2 {
		t.Fatalf("expected retries, opens=%d", opens.Load())
	}
}

func TestStreamDeployLogs_EOFDoesNotLoop(t *testing.T) {
	ctx := context.Background()
	ready := make(chan struct{})
	var opens atomic.Int32

	open := func(context.Context, string) (logEntryReceiver, error) {
		opens.Add(1)
		return &scriptedReceiver{queue: []recvResult{{err: io.EOF}}}, nil
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		streamDeployLogs(ctx, "dep-3", open, nil, ready, time.Hour)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("EOF must end the loop without retry")
	}
	if opens.Load() != 1 {
		t.Fatalf("opens=%d, want 1", opens.Load())
	}
}

package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

// captureHandler is a minimal slog.Handler that captures records in memory.
// It exists only to assert what the SSEHandler logs without depending on
// stdout. Concurrent writes are guarded by a mutex.
type captureHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())
	return nil
}

func (h *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(_ string) slog.Handler      { return h }

func (h *captureHandler) snapshot() []slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]slog.Record, len(h.records))
	copy(out, h.records)
	return out
}

func newTestSSEHandler(tb testing.TB, keepAlive time.Duration) (*SSEHandler, *captureHandler) {
	tb.Helper()
	cap := &captureHandler{}
	logger := slog.New(cap)
	h := NewSSEHandler(logger)
	h.keepAliveInterval = keepAlive
	return h, cap
}

func TestNewSSEHandler_NilLogger_FallsBackToDefault(t *testing.T) {
	h := NewSSEHandler(nil)
	if h.logger == nil {
		t.Fatal("expected logger to default to slog.Default(), got nil")
	}
	if h.keepAliveInterval != defaultKeepAliveInterval {
		t.Errorf("expected default keepAliveInterval=%v, got %v", defaultKeepAliveInterval, h.keepAliveInterval)
	}
}

func TestSSEHandler_Emit_LogsWarnWhenClientBufferFull(t *testing.T) {
	h, cap := newTestSSEHandler(t, time.Second)

	// Subscribe one client and drain nothing — channel will fill up.
	clientID := "test-client-1"
	ch := make(chan SSEEvent, sseClientBufferSize)
	h.mu.Lock()
	h.clients[clientID] = ch
	h.mu.Unlock()

	// Fill the channel exactly to capacity.
	for i := 0; i < sseClientBufferSize; i++ {
		h.Emit(SSEEvent{Type: "LOG", DeployID: "d1", AppID: "a1"})
	}
	if got := cap.snapshot(); len(got) != 0 {
		t.Fatalf("expected zero warnings while channel still has space, got %d", len(got))
	}

	// One more emit must trigger exactly one drop log.
	h.Emit(SSEEvent{Type: "RUNNING", DeployID: "d2", AppID: "a2", ServerID: "s2"})

	records := cap.snapshot()
	if len(records) != 1 {
		t.Fatalf("expected 1 warn record, got %d", len(records))
	}
	r := records[0]
	if r.Level != slog.LevelWarn {
		t.Errorf("expected Warn level, got %v", r.Level)
	}
	if !strings.Contains(r.Message, "buffer full") {
		t.Errorf("expected message to mention buffer full, got %q", r.Message)
	}
	attrs := collectAttrs(r)
	if attrs["clientID"] != clientID {
		t.Errorf("expected clientID=%s, got %v", clientID, attrs["clientID"])
	}
	if attrs["type"] != "RUNNING" {
		t.Errorf("expected type=RUNNING, got %v", attrs["type"])
	}
	if attrs["deployID"] != "d2" {
		t.Errorf("expected deployID=d2, got %v", attrs["deployID"])
	}
	if attrs["appID"] != "a2" {
		t.Errorf("expected appID=a2, got %v", attrs["appID"])
	}
	if attrs["serverID"] != "s2" {
		t.Errorf("expected serverID=s2, got %v", attrs["serverID"])
	}
	// Security: never log message body (may contain secrets).
	if _, hasMsg := attrs["message"]; hasMsg {
		t.Errorf("drop log must not include event.Message attribute (security)")
	}
}

func TestWriteKeepAlive_EmitsCommentAndFlushes(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	if !writeKeepAlive(w) {
		t.Fatal("writeKeepAlive returned false on a healthy writer")
	}
	got := buf.String()
	if got != ": keepalive\n\n" {
		t.Errorf("expected exactly %q, got %q", ": keepalive\n\n", got)
	}
}

func TestWriteSSEEvent_FormatsEventAndDataLines(t *testing.T) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	evt := SSEEvent{
		Type:     "RUNNING",
		DeployID: "d-1",
		AppID:    "a-1",
	}
	if !writeSSEEvent(w, evt) {
		t.Fatal("writeSSEEvent returned false on a healthy writer")
	}

	lines := strings.SplitN(buf.String(), "\n", 3)
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines (event/data/blank), got %q", buf.String())
	}
	if lines[0] != "event: deploy" {
		t.Errorf("first line: expected %q, got %q", "event: deploy", lines[0])
	}
	if !strings.HasPrefix(lines[1], "data: ") {
		t.Errorf("second line must start with 'data: ', got %q", lines[1])
	}

	var got SSEEvent
	jsonPart := strings.TrimPrefix(lines[1], "data: ")
	if err := json.Unmarshal([]byte(jsonPart), &got); err != nil {
		t.Fatalf("data line is not valid JSON: %v (%q)", err, jsonPart)
	}
	if got.Type != "RUNNING" || got.DeployID != "d-1" || got.AppID != "a-1" {
		t.Errorf("decoded event mismatch: %+v", got)
	}
}

func TestSSEHandler_EmitDeliversEventToSubscribedClient(t *testing.T) {
	h, _ := newTestSSEHandler(t, time.Second)

	clientID := "client-deliver"
	ch := make(chan SSEEvent, sseClientBufferSize)
	h.mu.Lock()
	h.clients[clientID] = ch
	h.mu.Unlock()

	h.EmitDeployRunning("deploy-A", "app-B")

	select {
	case got := <-ch:
		if got.Type != "RUNNING" || got.DeployID != "deploy-A" || got.AppID != "app-B" {
			t.Errorf("unexpected event delivered: %+v", got)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected event to be delivered within 250ms, got nothing")
	}
}

func TestSSEHandler_SendRecentEvents_ReplaysBufferedEvents(t *testing.T) {
	h, _ := newTestSSEHandler(t, time.Second)

	h.EmitDeployRunning("d-old-1", "a-old")
	h.EmitDeploySuccess("d-old-1", "a-old")

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	h.sendRecentEvents(w)

	body := buf.String()
	if !strings.Contains(body, "event: deploy") {
		t.Errorf("expected replay to include event: deploy, got:\n%s", body)
	}
	if !strings.Contains(body, "d-old-1") {
		t.Errorf("expected replay to include deployID d-old-1, got:\n%s", body)
	}
	if strings.Count(body, "data: ") != 2 {
		t.Errorf("expected 2 data lines (one per buffered event), got %d:\n%s",
			strings.Count(body, "data: "), body)
	}
}

func TestSSEHandler_ConfigDefaults(t *testing.T) {
	if defaultKeepAliveInterval != 15*time.Second {
		t.Errorf("defaultKeepAliveInterval should be 15s (lower than typical proxy idle timeouts), got %v",
			defaultKeepAliveInterval)
	}
}

func collectAttrs(r slog.Record) map[string]any {
	out := make(map[string]any)
	r.Attrs(func(a slog.Attr) bool {
		out[a.Key] = a.Value.Any()
		return true
	})
	return out
}


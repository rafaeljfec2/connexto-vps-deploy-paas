package handler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

const (
	sseEventBufferSize  = 100
	sseClientBufferSize = 100
)

type SSEHealthStatus struct {
	Status    string `json:"status"`
	Health    string `json:"health"`
	StartedAt string `json:"startedAt,omitempty"`
	Uptime    string `json:"uptime,omitempty"`
}

type SSEContainerStats struct {
	CPUPercent    float64 `json:"cpuPercent"`
	MemoryUsage   int64   `json:"memoryUsage"`
	MemoryLimit   int64   `json:"memoryLimit"`
	MemoryPercent float64 `json:"memoryPercent"`
	NetworkRx     int64   `json:"networkRx"`
	NetworkTx     int64   `json:"networkTx"`
	PIDs          int     `json:"pids"`
}

type SSEEvent struct {
	Type      string             `json:"type"`
	DeployID  string             `json:"deployId,omitempty"`
	AppID     string             `json:"appId,omitempty"`
	ServerID  string             `json:"serverId,omitempty"`
	Step      string             `json:"step,omitempty"`
	Status    string             `json:"status,omitempty"`
	Message   string             `json:"message,omitempty"`
	Health    *SSEHealthStatus   `json:"health,omitempty"`
	Stats     *SSEContainerStats `json:"stats,omitempty"`
	Timestamp time.Time          `json:"timestamp"`
}

type SSEHandler struct {
	clients   map[string]chan SSEEvent
	mu        sync.RWMutex
	eventBuf  []SSEEvent
	bufSize   int
	bufMu     sync.RWMutex
}

func NewSSEHandler() *SSEHandler {
	return &SSEHandler{
		clients:  make(map[string]chan SSEEvent),
		eventBuf: make([]SSEEvent, 0, sseEventBufferSize),
		bufSize:  sseEventBufferSize,
	}
}

func (h *SSEHandler) Register(app *fiber.App) {
	app.Get("/events/deploys", h.Stream)
}

func (h *SSEHandler) Stream(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	c.Set("Access-Control-Allow-Origin", "*")

	clientID := uuid.New().String()
	eventChan := h.subscribe(clientID)

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		defer h.unsubscribe(clientID)

		h.sendRecentEvents(w)

		for event := range eventChan {
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}

			eventType := "deploy"
			switch event.Type {
			case "LOG":
				eventType = "log"
			case "HEALTH":
				eventType = "health"
			case "STATS":
				eventType = "stats"
			case "PROVISION_STEP", "PROVISION_LOG", "PROVISION_COMPLETED", "PROVISION_FAILED":
				eventType = "provision"
			}

			fmt.Fprintf(w, "event: %s\n", eventType)
			fmt.Fprintf(w, "data: %s\n\n", data)

			if err := w.Flush(); err != nil {
				return
			}
		}
	}))

	return nil
}

func (h *SSEHandler) subscribe(clientID string) <-chan SSEEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan SSEEvent, sseClientBufferSize)
	h.clients[clientID] = ch
	return ch
}

func (h *SSEHandler) unsubscribe(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if ch, ok := h.clients[clientID]; ok {
		close(ch)
		delete(h.clients, clientID)
	}
}

func (h *SSEHandler) Emit(event SSEEvent) {
	event.Timestamp = time.Now().UTC()

	h.bufMu.Lock()
	if len(h.eventBuf) >= h.bufSize {
		h.eventBuf = h.eventBuf[1:]
	}
	h.eventBuf = append(h.eventBuf, event)
	h.bufMu.Unlock()

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, ch := range h.clients {
		select {
		case ch <- event:
		default:
		}
	}
}

func (h *SSEHandler) EmitDeployRunning(deployID, appID string) {
	h.Emit(SSEEvent{
		Type:     "RUNNING",
		DeployID: deployID,
		AppID:    appID,
	})
}

func (h *SSEHandler) EmitDeploySuccess(deployID, appID string) {
	h.Emit(SSEEvent{
		Type:     "SUCCESS",
		DeployID: deployID,
		AppID:    appID,
	})
}

func (h *SSEHandler) EmitDeployFailed(deployID, appID, message string) {
	h.Emit(SSEEvent{
		Type:     "FAILED",
		DeployID: deployID,
		AppID:    appID,
		Message:  message,
	})
}

func (h *SSEHandler) EmitLog(deployID, appID, message string) {
	h.Emit(SSEEvent{
		Type:     "LOG",
		DeployID: deployID,
		AppID:    appID,
		Message:  message,
	})
}

func (h *SSEHandler) EmitHealth(appID string, health SSEHealthStatus) {
	h.Emit(SSEEvent{
		Type:   "HEALTH",
		AppID:  appID,
		Health: &health,
	})
}

func (h *SSEHandler) EmitStats(appID string, stats SSEContainerStats) {
	h.Emit(SSEEvent{
		Type:  "STATS",
		AppID: appID,
		Stats: &stats,
	})
}

func (h *SSEHandler) EmitProvisionStep(serverID, step, status, message string) {
	h.Emit(SSEEvent{
		Type:     "PROVISION_STEP",
		ServerID: serverID,
		Step:     step,
		Status:   status,
		Message:  message,
	})
}

func (h *SSEHandler) EmitProvisionLog(serverID, message string) {
	h.Emit(SSEEvent{
		Type:     "PROVISION_LOG",
		ServerID: serverID,
		Message:  message,
	})
}

func (h *SSEHandler) EmitProvisionCompleted(serverID string) {
	h.Emit(SSEEvent{
		Type:     "PROVISION_COMPLETED",
		ServerID: serverID,
	})
}

func (h *SSEHandler) EmitProvisionFailed(serverID, message string) {
	h.Emit(SSEEvent{
		Type:     "PROVISION_FAILED",
		ServerID: serverID,
		Message:  message,
	})
}

func (h *SSEHandler) sendRecentEvents(w *bufio.Writer) {
	h.bufMu.RLock()
	events := make([]SSEEvent, len(h.eventBuf))
	copy(events, h.eventBuf)
	h.bufMu.RUnlock()

	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}

		eventType := "deploy"
		switch event.Type {
		case "LOG":
			eventType = "log"
		case "HEALTH":
			eventType = "health"
		case "STATS":
			eventType = "stats"
		case "PROVISION_STEP", "PROVISION_LOG", "PROVISION_COMPLETED", "PROVISION_FAILED":
			eventType = "provision"
		}

		fmt.Fprintf(w, "event: %s\n", eventType)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}
	w.Flush()
}

func (h *SSEHandler) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

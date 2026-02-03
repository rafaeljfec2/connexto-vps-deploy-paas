package grpcserver

import (
	"sync"
	"time"
)

type AgentConnection struct {
	ServerID        string
	LastHeartbeatAt time.Time
}

type AgentHub struct {
	mu    sync.RWMutex
	agents map[string]AgentConnection
}

func NewAgentHub() *AgentHub {
	return &AgentHub{agents: make(map[string]AgentConnection)}
}

func (h *AgentHub) Update(serverID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.agents[serverID] = AgentConnection{
		ServerID:        serverID,
		LastHeartbeatAt: time.Now(),
	}
}

func (h *AgentHub) Get(serverID string) (AgentConnection, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	conn, ok := h.agents[serverID]
	return conn, ok
}

func (h *AgentHub) List() []AgentConnection {
	h.mu.RLock()
	defer h.mu.RUnlock()
	agents := make([]AgentConnection, 0, len(h.agents))
	for _, conn := range h.agents {
		agents = append(agents, conn)
	}
	return agents
}

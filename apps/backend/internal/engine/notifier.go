package engine

import (
	"time"
)

type EventType string

const (
	EventTypeRunning EventType = "RUNNING"
	EventTypeSuccess EventType = "SUCCESS"
	EventTypeFailed  EventType = "FAILED"
	EventTypeLog     EventType = "LOG"
	EventTypeHealth  EventType = "HEALTH"
)

type HealthStatus struct {
	Status    string `json:"status"`
	Health    string `json:"health"`
	StartedAt string `json:"startedAt,omitempty"`
	Uptime    string `json:"uptime,omitempty"`
}

type DeployEvent struct {
	Type      EventType     `json:"type"`
	DeployID  string        `json:"deployId"`
	AppID     string        `json:"appId"`
	Message   string        `json:"message,omitempty"`
	Health    *HealthStatus `json:"health,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

type Notifier interface {
	EmitDeployRunning(deployID, appID string)
	EmitDeploySuccess(deployID, appID string)
	EmitDeployFailed(deployID, appID, message string)
	EmitLog(deployID, appID, message string)
	EmitHealth(appID string, health HealthStatus)
}

type ChannelNotifier struct {
	events chan DeployEvent
}

func NewChannelNotifier(bufferSize int) *ChannelNotifier {
	return &ChannelNotifier{
		events: make(chan DeployEvent, bufferSize),
	}
}

func (n *ChannelNotifier) Events() <-chan DeployEvent {
	return n.events
}

func (n *ChannelNotifier) emit(event DeployEvent) {
	event.Timestamp = time.Now().UTC()
	select {
	case n.events <- event:
	default:
	}
}

func (n *ChannelNotifier) EmitDeployRunning(deployID, appID string) {
	n.emit(DeployEvent{
		Type:     EventTypeRunning,
		DeployID: deployID,
		AppID:    appID,
	})
}

func (n *ChannelNotifier) EmitDeploySuccess(deployID, appID string) {
	n.emit(DeployEvent{
		Type:     EventTypeSuccess,
		DeployID: deployID,
		AppID:    appID,
	})
}

func (n *ChannelNotifier) EmitDeployFailed(deployID, appID, message string) {
	n.emit(DeployEvent{
		Type:     EventTypeFailed,
		DeployID: deployID,
		AppID:    appID,
		Message:  message,
	})
}

func (n *ChannelNotifier) EmitLog(deployID, appID, message string) {
	n.emit(DeployEvent{
		Type:     EventTypeLog,
		DeployID: deployID,
		AppID:    appID,
		Message:  message,
	})
}

func (n *ChannelNotifier) EmitHealth(appID string, health HealthStatus) {
	n.emit(DeployEvent{
		Type:   EventTypeHealth,
		AppID:  appID,
		Health: &health,
	})
}

func (n *ChannelNotifier) Close() {
	close(n.events)
}

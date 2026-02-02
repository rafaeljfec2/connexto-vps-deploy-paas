package service

import (
	"log/slog"
	"sync"
	"time"

	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/notification"
)

type NotificationService struct {
	channelRepo domain.NotificationChannelRepository
	ruleRepo    domain.NotificationRuleRepository
	appRepo     domain.AppRepository
	senders     map[domain.NotificationChannelType]notification.Sender
	logger      *slog.Logger
}

func NewNotificationService(
	channelRepo domain.NotificationChannelRepository,
	ruleRepo domain.NotificationRuleRepository,
	appRepo domain.AppRepository,
	logger *slog.Logger,
) *NotificationService {
	return &NotificationService{
		channelRepo: channelRepo,
		ruleRepo:    ruleRepo,
		appRepo:     appRepo,
		senders: map[domain.NotificationChannelType]notification.Sender{
			domain.NotificationChannelSlack:   notification.NewSlackSender(),
			domain.NotificationChannelDiscord: notification.NewDiscordSender(),
			domain.NotificationChannelEmail:  notification.NewEmailSender(),
		},
		logger: logger.With("component", "notification_service"),
	}
}

func (s *NotificationService) NotifyDeployRunning(deployID, appID string) {
	s.notify(domain.EventTypeDeployRunning, deployID, appID, "", "running", "")
}

func (s *NotificationService) NotifyDeploySuccess(deployID, appID string) {
	s.notify(domain.EventTypeDeploySuccess, deployID, appID, "", "success", "")
}

func (s *NotificationService) NotifyDeployFailed(deployID, appID, message string) {
	s.notify(domain.EventTypeDeployFailed, deployID, appID, message, "failed", "")
}

func (s *NotificationService) NotifyHealthChange(appID, status, health string) {
	eventType := domain.EventTypeHealthUnhealthy
	if status == "not_found" {
		eventType = domain.EventTypeContainerDown
	}
	s.notify(eventType, "", appID, "", status, health)
}

func (s *NotificationService) notify(eventType, deployID, appID, message, status, health string) {
	rules, err := s.ruleRepo.FindActiveByEventType(eventType, ptrOrNil(appID))
	if err != nil {
		s.logger.Error("Failed to find notification rules", "eventType", eventType, "error", err)
		return
	}
	if len(rules) == 0 {
		return
	}

	appName := ""
	if appID != "" {
		if app, err := s.appRepo.FindByID(appID); err == nil {
			appName = app.Name
		}
	}

	payload := notification.NotificationPayload{
		EventType: eventType,
		AppID:     appID,
		AppName:   appName,
		Message:   message,
		DeployID:  deployID,
		Status:    status,
		Health:    health,
		Timestamp: time.Now().UTC(),
	}

	var wg sync.WaitGroup
	for _, rule := range rules {
		channel, err := s.channelRepo.FindByID(rule.ChannelID)
		if err != nil {
			s.logger.Warn("Channel not found for rule", "ruleId", rule.ID, "channelId", rule.ChannelID)
			continue
		}

		sender, ok := s.senders[channel.Type]
		if !ok {
			s.logger.Warn("Unknown channel type", "channelId", channel.ID, "type", channel.Type)
			continue
		}

		wg.Add(1)
		go func(ch *domain.NotificationChannel) {
			defer wg.Done()
			if err := sender.Send(ch, payload); err != nil {
				s.logger.Error("Failed to send notification", "channelId", ch.ID, "error", err)
			}
		}(channel)
	}
	wg.Wait()
}

func ptrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

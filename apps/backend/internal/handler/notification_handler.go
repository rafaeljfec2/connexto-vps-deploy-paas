package handler

import (
	"encoding/json"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/response"
)

type NotificationHandler struct {
	channelRepo domain.NotificationChannelRepository
	ruleRepo    domain.NotificationRuleRepository
	appRepo     domain.AppRepository
	logger      *slog.Logger
}

func NewNotificationHandler(
	channelRepo domain.NotificationChannelRepository,
	ruleRepo domain.NotificationRuleRepository,
	appRepo domain.AppRepository,
	logger *slog.Logger,
) *NotificationHandler {
	return &NotificationHandler{
		channelRepo: channelRepo,
		ruleRepo:    ruleRepo,
		appRepo:     appRepo,
		logger:      logger.With("handler", "notification"),
	}
}

func (h *NotificationHandler) Register(app fiber.Router) {
	v1 := app.Group(APIPrefix)
	channels := v1.Group("/notifications/channels")
	channels.Get("/", h.ListChannels)
	channels.Post("/", h.CreateChannel)
	channels.Get("/:id", h.GetChannel)
	channels.Put("/:id", h.UpdateChannel)
	channels.Delete("/:id", h.DeleteChannel)
	channels.Get("/:id/rules", h.ListRulesByChannel)

	rules := v1.Group("/notifications/rules")
	rules.Get("/", h.ListRules)
	rules.Post("/", h.CreateRule)
	rules.Get("/:id", h.GetRule)
	rules.Put("/:id", h.UpdateRule)
	rules.Delete("/:id", h.DeleteRule)
}

type ChannelResponse struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Config    any    `json:"config"`
	AppID     string `json:"appId,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func toChannelResponse(ch *domain.NotificationChannel) ChannelResponse {
	resp := ChannelResponse{
		ID:        ch.ID,
		Type:      string(ch.Type),
		Name:      ch.Name,
		CreatedAt: ch.CreatedAt.Format(DateTimeFormatISO8601),
		UpdatedAt: ch.UpdatedAt.Format(DateTimeFormatISO8601),
	}
	if len(ch.Config) > 0 {
		var cfg any
		if err := json.Unmarshal(ch.Config, &cfg); err == nil {
			resp.Config = cfg
		}
	}
	if ch.AppID != nil {
		resp.AppID = *ch.AppID
	}
	return resp
}

type RuleResponse struct {
	ID        string `json:"id"`
	EventType string `json:"eventType"`
	ChannelID string `json:"channelId"`
	AppID     string `json:"appId,omitempty"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func toRuleResponse(r *domain.NotificationRule) RuleResponse {
	resp := RuleResponse{
		ID:        r.ID,
		EventType: r.EventType,
		ChannelID: r.ChannelID,
		Enabled:   r.Enabled,
		CreatedAt: r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: r.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if r.AppID != nil {
		resp.AppID = *r.AppID
	}
	return resp
}

func (h *NotificationHandler) requireChannelForUser(c *fiber.Ctx) (*domain.NotificationChannel, error) {
	user := GetUserFromContext(c)
	if user == nil {
		return nil, response.Unauthorized(c, MsgNotAuthenticated)
	}
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return nil, response.NotFound(c, MsgChannelNotFound)
	}
	ch, err := h.channelRepo.FindByIDAndUserID(id, user.ID)
	if err != nil {
		return nil, response.NotFound(c, MsgChannelNotFound)
	}
	return ch, nil
}

func (h *NotificationHandler) requireRuleForUser(c *fiber.Ctx) (*domain.NotificationRule, error) {
	user := GetUserFromContext(c)
	if user == nil {
		return nil, response.Unauthorized(c, MsgNotAuthenticated)
	}
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return nil, response.NotFound(c, MsgRuleNotFound)
	}
	rule, err := h.ruleRepo.FindByIDAndUserID(id, user.ID)
	if err != nil {
		return nil, response.NotFound(c, MsgRuleNotFound)
	}
	return rule, nil
}

func (h *NotificationHandler) ListChannels(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	channels, err := h.channelRepo.FindAllByUserID(user.ID)
	if err != nil {
		h.logger.Error("failed to list channels", "error", err)
		return response.InternalError(c)
	}

	resp := make([]ChannelResponse, len(channels))
	for i := range channels {
		resp[i] = toChannelResponse(&channels[i])
	}
	return response.OK(c, resp)
}

type CreateChannelRequest struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Config any    `json:"config"`
	AppID  string `json:"appId,omitempty"`
}

func (h *NotificationHandler) CreateChannel(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	var body CreateChannelRequest
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, MsgInvalidRequestBody)
	}
	if body.Type == "" || body.Name == "" {
		return response.BadRequest(c, "type and name are required")
	}

	chType := domain.NotificationChannelType(body.Type)
	if chType != domain.NotificationChannelSlack &&
		chType != domain.NotificationChannelDiscord &&
		chType != domain.NotificationChannelEmail {
		return response.BadRequest(c, "invalid channel type: slack, discord, or email")
	}

	configJSON, err := json.Marshal(body.Config)
	if err != nil {
		return response.BadRequest(c, "invalid config")
	}

	input := domain.CreateNotificationChannelInput{
		UserID: user.ID,
		Type:   chType,
		Name:   body.Name,
		Config: configJSON,
	}
	if body.AppID != "" {
		if err := EnsureAppOwnership(c, h.appRepo, body.AppID); err != nil {
			return err
		}
		input.AppID = &body.AppID
	}

	ch, err := h.channelRepo.Create(input)
	if err != nil {
		h.logger.Error("failed to create channel", "error", err)
		return response.InternalError(c)
	}
	return response.Created(c, toChannelResponse(ch))
}

func (h *NotificationHandler) GetChannel(c *fiber.Ctx) error {
	ch, err := h.requireChannelForUser(c)
	if err != nil {
		return err
	}
	return response.OK(c, toChannelResponse(ch))
}

type UpdateChannelRequest struct {
	Name   *string `json:"name,omitempty"`
	Config *any    `json:"config,omitempty"`
}

func (h *NotificationHandler) UpdateChannel(c *fiber.Ctx) error {
	_, err := h.requireChannelForUser(c)
	if err != nil {
		return err
	}

	id := c.Params("id")

	var body UpdateChannelRequest
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, MsgInvalidRequestBody)
	}

	input := domain.UpdateNotificationChannelInput{}
	if body.Name != nil {
		input.Name = body.Name
	}
	if body.Config != nil {
		configJSON, err := json.Marshal(*body.Config)
		if err != nil {
			return response.BadRequest(c, "invalid config")
		}
		configRaw := json.RawMessage(configJSON)
		input.Config = &configRaw
	}

	ch, err := h.channelRepo.Update(id, input)
	if err != nil {
		h.logger.Error("failed to update channel", "error", err)
		return response.InternalError(c)
	}
	return response.OK(c, toChannelResponse(ch))
}

func (h *NotificationHandler) DeleteChannel(c *fiber.Ctx) error {
	_, err := h.requireChannelForUser(c)
	if err != nil {
		return err
	}

	id := c.Params("id")

	if err := h.channelRepo.Delete(id); err != nil {
		return response.NotFound(c, MsgChannelNotFound)
	}
	return response.NoContent(c)
}

func (h *NotificationHandler) ListRulesByChannel(c *fiber.Ctx) error {
	ch, err := h.requireChannelForUser(c)
	if err != nil {
		return err
	}

	rules, err := h.ruleRepo.FindByChannelID(ch.ID)
	if err != nil {
		h.logger.Error("failed to list rules", "error", err)
		return response.InternalError(c)
	}

	resp := make([]RuleResponse, len(rules))
	for i := range rules {
		resp[i] = toRuleResponse(&rules[i])
	}
	return response.OK(c, resp)
}

func (h *NotificationHandler) ListRules(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	rules, err := h.ruleRepo.FindAllByUserID(user.ID)
	if err != nil {
		h.logger.Error("failed to list rules", "error", err)
		return response.InternalError(c)
	}

	resp := make([]RuleResponse, len(rules))
	for i := range rules {
		resp[i] = toRuleResponse(&rules[i])
	}
	return response.OK(c, resp)
}

type CreateRuleRequest struct {
	EventType string `json:"eventType"`
	ChannelID string `json:"channelId"`
	AppID     string `json:"appId,omitempty"`
	Enabled   bool   `json:"enabled"`
}

var validEventTypes = map[string]bool{
	domain.EventTypeDeployRunning:    true,
	domain.EventTypeDeploySuccess:    true,
	domain.EventTypeDeployFailed:     true,
	domain.EventTypeContainerDown:    true,
	domain.EventTypeHealthUnhealthy:  true,
}

func (h *NotificationHandler) CreateRule(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	var body CreateRuleRequest
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, MsgInvalidRequestBody)
	}
	if body.EventType == "" || body.ChannelID == "" {
		return response.BadRequest(c, "eventType and channelId are required")
	}
	if !validEventTypes[body.EventType] {
		return response.BadRequest(c, "invalid eventType")
	}

	_, err := h.channelRepo.FindByIDAndUserID(body.ChannelID, user.ID)
	if err != nil {
		return response.NotFound(c, MsgChannelNotFound)
	}

	input := domain.CreateNotificationRuleInput{
		UserID:    user.ID,
		EventType: body.EventType,
		ChannelID: body.ChannelID,
		Enabled:   body.Enabled,
	}
	if body.AppID != "" {
		if err := EnsureAppOwnership(c, h.appRepo, body.AppID); err != nil {
			return err
		}
		input.AppID = &body.AppID
	}

	rule, err := h.ruleRepo.Create(input)
	if err != nil {
		h.logger.Error("failed to create rule", "error", err)
		return response.InternalError(c)
	}
	return response.Created(c, toRuleResponse(rule))
}

func (h *NotificationHandler) GetRule(c *fiber.Ctx) error {
	rule, err := h.requireRuleForUser(c)
	if err != nil {
		return err
	}
	return response.OK(c, toRuleResponse(rule))
}

type UpdateRuleRequest struct {
	EventType *string `json:"eventType,omitempty"`
	Enabled   *bool   `json:"enabled,omitempty"`
}

func (h *NotificationHandler) UpdateRule(c *fiber.Ctx) error {
	_, err := h.requireRuleForUser(c)
	if err != nil {
		return err
	}

	id := c.Params("id")

	var body UpdateRuleRequest
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, MsgInvalidRequestBody)
	}

	input := domain.UpdateNotificationRuleInput{}
	if body.EventType != nil {
		if !validEventTypes[*body.EventType] {
			return response.BadRequest(c, "invalid eventType")
		}
		input.EventType = body.EventType
	}
	if body.Enabled != nil {
		input.Enabled = body.Enabled
	}

	rule, err := h.ruleRepo.Update(id, input)
	if err != nil {
		h.logger.Error("failed to update rule", "error", err)
		return response.InternalError(c)
	}
	return response.OK(c, toRuleResponse(rule))
}

func (h *NotificationHandler) DeleteRule(c *fiber.Ctx) error {
	_, err := h.requireRuleForUser(c)
	if err != nil {
		return err
	}

	id := c.Params("id")

	if err := h.ruleRepo.Delete(id); err != nil {
		return response.NotFound(c, MsgRuleNotFound)
	}
	return response.NoContent(c)
}

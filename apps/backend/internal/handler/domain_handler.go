package handler

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/cloudflare"
	"github.com/paasdeploy/backend/internal/crypto"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/response"
)

type DomainHandler struct {
	appRepo        domain.AppRepository
	domainRepo     domain.CustomDomainRepository
	connectionRepo domain.CloudflareConnectionRepository
	tokenEncryptor *crypto.TokenEncryptor
	serverIP       string
	logger         *slog.Logger
}

type DomainHandlerConfig struct {
	AppRepo        domain.AppRepository
	DomainRepo     domain.CustomDomainRepository
	ConnectionRepo domain.CloudflareConnectionRepository
	TokenEncryptor *crypto.TokenEncryptor
	ServerIP       string
	Logger         *slog.Logger
}

func NewDomainHandler(cfg DomainHandlerConfig) *DomainHandler {
	return &DomainHandler{
		appRepo:        cfg.AppRepo,
		domainRepo:     cfg.DomainRepo,
		connectionRepo: cfg.ConnectionRepo,
		tokenEncryptor: cfg.TokenEncryptor,
		serverIP:       cfg.ServerIP,
		logger:         cfg.Logger.With("handler", "domain"),
	}
}

func (h *DomainHandler) Register(app fiber.Router) {
	v1 := app.Group(APIPrefix)
	apps := v1.Group("/apps")
	apps.Get("/:id/domains", h.ListDomains)
	apps.Post("/:id/domains", h.AddDomain)
	apps.Delete("/:id/domains/:domainId", h.RemoveDomain)
}

type DomainResponse struct {
	ID         string `json:"id"`
	AppID      string `json:"appId"`
	Domain     string `json:"domain"`
	RecordType string `json:"recordType"`
	Status     string `json:"status"`
	CreatedAt  string `json:"createdAt"`
}

func toDomainResponse(d *domain.CustomDomain) DomainResponse {
	return DomainResponse{
		ID:         d.ID,
		AppID:      d.AppID,
		Domain:     d.Domain,
		RecordType: d.RecordType,
		Status:     d.Status,
		CreatedAt:  d.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func (h *DomainHandler) ListDomains(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	appID := c.Params("id")

	_, err := h.appRepo.FindByID(appID)
	if err != nil {
		return response.NotFound(c, MsgAppNotFound)
	}

	domains, err := h.domainRepo.FindByAppID(c.Context(), appID)
	if err != nil {
		h.logger.Error("failed to list domains", "error", err, "app_id", appID)
		return response.InternalError(c)
	}

	resp := make([]DomainResponse, len(domains))
	for i, d := range domains {
		resp[i] = toDomainResponse(&d)
	}

	return response.OK(c, resp)
}

type AddDomainRequest struct {
	Domain string `json:"domain"`
}

func (h *DomainHandler) AddDomain(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	appID := c.Params("id")

	var req AddDomainRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, MsgInvalidRequestBody)
	}

	if _, err := h.appRepo.FindByID(appID); err != nil {
		return response.NotFound(c, MsgAppNotFound)
	}

	conn, err := h.connectionRepo.FindByUserID(c.Context(), user.ID)
	if err != nil {
		return response.BadRequest(c, "Connect your Cloudflare account first")
	}

	accessToken, err := h.tokenEncryptor.Decrypt(conn.AccessTokenEncrypted)
	if err != nil {
		h.logger.Error("failed to decrypt cloudflare token", "error", err)
		return response.InternalError(c)
	}

	domainName := strings.ToLower(strings.TrimSpace(req.Domain))
	if domainName == "" {
		return response.BadRequest(c, "Domain is required")
	}

	if !isValidDomain(domainName) {
		return response.BadRequest(c, "Invalid domain format")
	}

	existing, _ := h.domainRepo.FindByDomain(c.Context(), domainName)
	if existing != nil {
		return response.BadRequest(c, "Domain already exists")
	}

	rootDomain := extractRootDomain(domainName)

	cfClient := cloudflare.NewClient(accessToken, h.logger)

	zoneID, err := cfClient.GetZoneID(c.Context(), rootDomain)
	if err != nil {
		h.logger.Error("zone not found", "domain", rootDomain, "error", err)
		return response.BadRequest(c, "Domain not found in your Cloudflare account")
	}

	recordID, err := cfClient.CreateARecord(c.Context(), zoneID, domainName, h.serverIP)
	if err != nil {
		h.logger.Error("failed to create DNS record", "domain", domainName, "error", err)
		return response.BadRequest(c, "Failed to create DNS record: "+err.Error())
	}
	recordType := "A"

	customDomain, err := h.domainRepo.Create(c.Context(), domain.CreateCustomDomainInput{
		AppID:       appID,
		Domain:      domainName,
		ZoneID:      zoneID,
		DNSRecordID: recordID,
		RecordType:  recordType,
	})
	if err != nil {
		_ = cfClient.DeleteRecord(c.Context(), zoneID, recordID)
		h.logger.Error("failed to save custom domain", "error", err)
		return response.InternalError(c)
	}

	h.logger.Info("Custom domain added",
		"app_id", appID,
		"domain", domainName,
		"record_type", recordType,
		"user_id", user.ID,
	)

	return response.OK(c, toDomainResponse(customDomain))
}

func (h *DomainHandler) RemoveDomain(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	appID := c.Params("id")
	domainID := c.Params("domainId")

	_, err := h.appRepo.FindByID(appID)
	if err != nil {
		return response.NotFound(c, MsgAppNotFound)
	}

	customDomain, err := h.domainRepo.FindByID(c.Context(), domainID)
	if err != nil {
		return response.NotFound(c, "Domain not found")
	}

	if customDomain.AppID != appID {
		return response.NotFound(c, "Domain not found")
	}

	conn, err := h.connectionRepo.FindByUserID(c.Context(), user.ID)
	if err == nil {
		accessToken, err := h.tokenEncryptor.Decrypt(conn.AccessTokenEncrypted)
		if err == nil {
			cfClient := cloudflare.NewClient(accessToken, h.logger)
			if deleteErr := cfClient.DeleteRecord(c.Context(), customDomain.ZoneID, customDomain.DNSRecordID); deleteErr != nil {
				h.logger.Warn("failed to delete DNS record from Cloudflare",
					"error", deleteErr,
					"domain", customDomain.Domain,
				)
			}
		}
	}

	if err := h.domainRepo.Delete(c.Context(), domainID); err != nil {
		h.logger.Error("failed to delete custom domain", "error", err)
		return response.InternalError(c)
	}

	h.logger.Info("Custom domain removed",
		"app_id", appID,
		"domain", customDomain.Domain,
		"user_id", user.ID,
	)

	return response.OK(c, fiber.Map{"message": "Domain removed"})
}

func extractRootDomain(domain string) string {
	parts := strings.Split(domain, ".")
	n := len(parts)

	if n < 2 {
		return domain
	}

	secondLevelTLDs := map[string]bool{
		"com.br": true, "net.br": true, "org.br": true, "gov.br": true, "edu.br": true,
		"co.uk": true, "org.uk": true, "gov.uk": true, "ac.uk": true,
		"com.au": true, "net.au": true, "org.au": true,
		"co.nz": true, "net.nz": true, "org.nz": true,
		"co.jp": true, "ne.jp": true, "or.jp": true,
		"com.mx": true, "org.mx": true, "gob.mx": true,
		"com.ar": true, "org.ar": true, "gov.ar": true,
	}

	if n >= 3 {
		possibleTLD := parts[n-2] + "." + parts[n-1]
		if secondLevelTLDs[possibleTLD] {
			return parts[n-3] + "." + possibleTLD
		}
	}

	return parts[n-2] + "." + parts[n-1]
}

func isValidDomain(domain string) bool {
	if len(domain) < 4 || len(domain) > 253 {
		return false
	}

	if !strings.Contains(domain, ".") {
		return false
	}

	parts := strings.Split(domain, ".")
	for _, part := range parts {
		if !isValidDomainPart(part) {
			return false
		}
	}

	return true
}

func isValidDomainPart(part string) bool {
	if len(part) == 0 || len(part) > 63 {
		return false
	}

	if part[0] == '-' || part[len(part)-1] == '-' {
		return false
	}

	for _, c := range part {
		if !isValidDomainChar(c) {
			return false
		}
	}

	return true
}

func isValidDomainChar(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-'
}

package handler

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/response"
	"github.com/paasdeploy/backend/internal/traefik"
)

type CertificateHandler struct {
	traefikClient *traefik.Client
	logger        *slog.Logger
}

type CertificateHandlerConfig struct {
	TraefikURL string
	Logger     *slog.Logger
}

func NewCertificateHandler(cfg CertificateHandlerConfig) *CertificateHandler {
	return &CertificateHandler{
		traefikClient: traefik.NewClient(cfg.TraefikURL),
		logger:        cfg.Logger,
	}
}

func (h *CertificateHandler) RegisterRoutes(router fiber.Router) {
	certificates := router.Group("/certificates")
	certificates.Get("/", h.ListCertificates)
	certificates.Get("/:domain", h.GetCertificateStatus)
}

func (h *CertificateHandler) ListCertificates(c *fiber.Ctx) error {
	certificates, err := h.traefikClient.GetAllCertificatesStatus(c.Context())
	if err != nil {
		h.logger.Warn("Traefik unavailable, returning empty certificates list", "error", err)
		return response.OK(c, []traefik.CertificateStatus{})
	}

	return response.OK(c, certificates)
}

func (h *CertificateHandler) GetCertificateStatus(c *fiber.Ctx) error {
	domain := c.Params("domain")
	if domain == "" {
		return response.BadRequest(c, "Domain is required")
	}

	status, err := h.traefikClient.GetCertificateStatus(c.Context(), domain)
	if err != nil {
		h.logger.Warn("Traefik unavailable for certificate status", "error", err, "domain", domain)
		return response.OK(c, &traefik.CertificateStatus{
			Domain: domain,
			Status: "unavailable",
			Error:  "Traefik API unavailable",
		})
	}

	return response.OK(c, status)
}

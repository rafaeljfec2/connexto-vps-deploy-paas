package handler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/agentclient"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/response"
	"github.com/paasdeploy/shared/pkg/traefik"
)

const remoteCertTimeout = 5 * time.Second

type CertificateHandler struct {
	traefikClient *traefik.Client
	agentClient   *agentclient.AgentClient
	serverRepo    domain.ServerRepository
	agentPort     int
	logger        *slog.Logger
}

type CertificateHandlerConfig struct {
	TraefikURL  string
	AgentClient *agentclient.AgentClient
	ServerRepo  domain.ServerRepository
	AgentPort   int
	Logger      *slog.Logger
}

func NewCertificateHandler(cfg CertificateHandlerConfig) *CertificateHandler {
	return &CertificateHandler{
		traefikClient: traefik.NewClient(cfg.TraefikURL),
		agentClient:   cfg.AgentClient,
		serverRepo:    cfg.ServerRepo,
		agentPort:     cfg.AgentPort,
		logger:        cfg.Logger,
	}
}

func (h *CertificateHandler) RegisterRoutes(router fiber.Router) {
	certificates := router.Group("/certificates")
	certificates.Get("/", h.ListCertificates)
	certificates.Get("/:domain", h.GetCertificateStatus)
}

func (h *CertificateHandler) ListCertificates(c *fiber.Ctx) error {
	localCerts, err := h.traefikClient.GetAllCertificatesStatus(c.Context())
	if err != nil {
		h.logger.Warn("Traefik unavailable, returning empty local certificates", "error", err)
		localCerts = []traefik.CertificateStatus{}
	}

	remoteCerts := h.fetchRemoteCertificates(c.Context())

	allCerts := make([]traefik.CertificateStatus, 0, len(localCerts)+len(remoteCerts))
	seen := make(map[string]bool, len(localCerts))

	for _, cert := range localCerts {
		seen[cert.Domain] = true
		allCerts = append(allCerts, cert)
	}

	for _, cert := range remoteCerts {
		if !seen[cert.Domain] {
			seen[cert.Domain] = true
			allCerts = append(allCerts, cert)
		}
	}

	return response.OK(c, allCerts)
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

func (h *CertificateHandler) fetchRemoteCertificates(ctx context.Context) []traefik.CertificateStatus {
	if h.agentClient == nil || h.serverRepo == nil || h.agentPort == 0 {
		return nil
	}

	servers, err := h.serverRepo.FindAll()
	if err != nil {
		h.logger.Warn("Failed to list servers for remote certificates", "error", err)
		return nil
	}

	var onlineServers []domain.Server
	for _, s := range servers {
		if s.Status == domain.ServerStatusOnline {
			onlineServers = append(onlineServers, s)
		}
	}

	if len(onlineServers) == 0 {
		return nil
	}

	type result struct {
		certs []traefik.CertificateStatus
	}

	results := make([]result, len(onlineServers))
	var wg sync.WaitGroup

	for i, server := range onlineServers {
		wg.Add(1)
		go func(idx int, srv domain.Server) {
			defer wg.Done()

			timeoutCtx, cancel := context.WithTimeout(ctx, remoteCertTimeout)
			defer cancel()

			pbCerts, err := h.agentClient.GetCertificates(timeoutCtx, srv.Host, h.agentPort)
			if err != nil {
				h.logger.Debug("Failed to fetch certificates from agent",
					"serverId", srv.ID, "host", srv.Host, "error", err)
				return
			}

			certs := make([]traefik.CertificateStatus, 0, len(pbCerts))
			for _, c := range pbCerts {
				certs = append(certs, traefik.CertificateStatus{
					Domain: c.Domain,
					Status: c.Status,
					Issuer: c.Issuer,
					Error:  c.Error,
				})
			}
			results[idx] = result{certs: certs}
		}(i, server)
	}

	wg.Wait()

	var allRemote []traefik.CertificateStatus
	for _, r := range results {
		allRemote = append(allRemote, r.certs...)
	}
	return allRemote
}

package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/cloudflare"
	"github.com/paasdeploy/backend/internal/crypto"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/response"
)

type CloudflareAuthHandler struct {
	clientID       string
	clientSecret   string
	callbackURL    string
	connectionRepo domain.CloudflareConnectionRepository
	tokenEncryptor *crypto.TokenEncryptor
	logger         *slog.Logger
	frontendURL    string
	secureCookie   bool
}

type CloudflareAuthHandlerConfig struct {
	ClientID       string
	ClientSecret   string
	CallbackURL    string
	ConnectionRepo domain.CloudflareConnectionRepository
	TokenEncryptor *crypto.TokenEncryptor
	Logger         *slog.Logger
	FrontendURL    string
	SecureCookie   bool
}

func NewCloudflareAuthHandler(cfg CloudflareAuthHandlerConfig) *CloudflareAuthHandler {
	return &CloudflareAuthHandler{
		clientID:       cfg.ClientID,
		clientSecret:   cfg.ClientSecret,
		callbackURL:    cfg.CallbackURL,
		connectionRepo: cfg.ConnectionRepo,
		tokenEncryptor: cfg.TokenEncryptor,
		logger:         cfg.Logger.With("handler", "cloudflare_auth"),
		frontendURL:    cfg.FrontendURL,
		secureCookie:   cfg.SecureCookie,
	}
}

func (h *CloudflareAuthHandler) Register(app fiber.Router) {
	cf := app.Group("/auth/cloudflare")
	cf.Get("", h.InitiateOAuth)
	cf.Get("/callback", h.HandleCallback)
	cf.Post("/connect", h.ConnectWithToken)
	cf.Post("/disconnect", h.Disconnect)
	cf.Get("/status", h.GetStatus)
}

func (h *CloudflareAuthHandler) setCookie(c *fiber.Ctx, name, value string, maxAge int) {
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    value,
		HTTPOnly: true,
		Secure:   h.secureCookie,
		SameSite: "Lax",
		MaxAge:   maxAge,
		Path:     "/",
	})
}

func (h *CloudflareAuthHandler) redirectWithError(c *fiber.Ctx, errorCode string) error {
	return c.Redirect(h.frontendURL+"/settings?error="+errorCode, fiber.StatusTemporaryRedirect)
}

func (h *CloudflareAuthHandler) generateState() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (h *CloudflareAuthHandler) InitiateOAuth(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	state, err := h.generateState()
	if err != nil {
		h.logger.Error("failed to generate state", "error", err)
		return response.InternalError(c)
	}

	h.setCookie(c, "cloudflare_oauth_state", state, 600)

	authURL := fmt.Sprintf(
		"https://dash.cloudflare.com/oauth2/auth?client_id=%s&redirect_uri=%s&response_type=code&state=%s",
		h.clientID,
		url.QueryEscape(h.callbackURL),
		state,
	)

	return c.Redirect(authURL, fiber.StatusTemporaryRedirect)
}

func (h *CloudflareAuthHandler) HandleCallback(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return h.redirectWithError(c, "not_authenticated")
	}

	if errorParam := c.Query("error"); errorParam != "" {
		h.logger.Warn("OAuth error from Cloudflare",
			"error", errorParam,
			"description", c.Query("error_description"),
		)
		return h.redirectWithError(c, "cloudflare_denied")
	}

	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		return h.redirectWithError(c, "no_code")
	}

	savedState := c.Cookies("cloudflare_oauth_state")
	if state == "" || state != savedState {
		return h.redirectWithError(c, "invalid_state")
	}

	h.setCookie(c, "cloudflare_oauth_state", "", -1)

	accessToken, err := h.exchangeCodeForToken(code)
	if err != nil {
		h.logger.Error("failed to exchange code for token", "error", err)
		return h.redirectWithError(c, "token_exchange_failed")
	}

	cfClient := cloudflare.NewClient(accessToken, h.logger)

	userInfo, err := cfClient.GetUserInfo(c.Context())
	if err != nil {
		h.logger.Error("failed to get cloudflare user info", "error", err)
		return h.redirectWithError(c, "user_info_failed")
	}

	encryptedToken, err := h.tokenEncryptor.Encrypt(accessToken)
	if err != nil {
		h.logger.Error("failed to encrypt token", "error", err)
		return h.redirectWithError(c, "encryption_failed")
	}

	_, err = h.connectionRepo.Upsert(c.Context(), domain.CreateCloudflareConnectionInput{
		UserID:               user.ID,
		CloudflareAccountID:  userInfo.ID,
		CloudflareEmail:      userInfo.Email,
		AccessTokenEncrypted: encryptedToken,
	})
	if err != nil {
		h.logger.Error("failed to save cloudflare connection", "error", err)
		return h.redirectWithError(c, "save_failed")
	}

	h.logger.Info("Cloudflare connected",
		"user_id", user.ID,
		"cloudflare_email", userInfo.Email,
	)

	return c.Redirect(h.frontendURL+"/settings?cloudflare=connected", fiber.StatusTemporaryRedirect)
}

func (h *CloudflareAuthHandler) exchangeCodeForToken(code string) (string, error) {
	// Note: Cloudflare OAuth token exchange
	// For now, we'll use API tokens directly since Cloudflare's OAuth
	// for third-party apps requires special approval.
	// Users will paste their API token instead.
	return code, nil
}

func (h *CloudflareAuthHandler) Disconnect(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	if err := h.connectionRepo.DeleteByUserID(c.Context(), user.ID); err != nil {
		h.logger.Error("failed to delete cloudflare connection", "error", err, "user_id", user.ID)
		return response.InternalError(c)
	}

	h.logger.Info("Cloudflare disconnected", "user_id", user.ID)

	return response.OK(c, fiber.Map{"message": "disconnected"})
}

func (h *CloudflareAuthHandler) GetStatus(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	conn, err := h.connectionRepo.FindByUserID(c.Context(), user.ID)
	if err != nil {
		return response.OK(c, fiber.Map{
			"connected": false,
		})
	}

	return response.OK(c, fiber.Map{
		"connected": true,
		"email":     conn.CloudflareEmail,
		"accountId": conn.CloudflareAccountID,
	})
}

type ConnectCloudflareRequest struct {
	APIToken string `json:"apiToken"`
}

func (h *CloudflareAuthHandler) ConnectWithToken(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, MsgNotAuthenticated)
	}

	var req ConnectCloudflareRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, MsgInvalidRequestBody)
	}

	if req.APIToken == "" {
		return response.BadRequest(c, "API token is required")
	}

	cfClient := cloudflare.NewClient(req.APIToken, h.logger)

	tokenInfo, err := cfClient.VerifyToken(c.Context())
	if err != nil {
		h.logger.Warn("invalid cloudflare token", "error", err, "user_id", user.ID)
		return response.BadRequest(c, "Invalid API token")
	}

	encryptedToken, err := h.tokenEncryptor.Encrypt(req.APIToken)
	if err != nil {
		h.logger.Error("failed to encrypt token", "error", err)
		return response.InternalError(c)
	}

	conn, err := h.connectionRepo.Upsert(c.Context(), domain.CreateCloudflareConnectionInput{
		UserID:               user.ID,
		CloudflareAccountID:  tokenInfo.ID,
		CloudflareEmail:      "",
		AccessTokenEncrypted: encryptedToken,
	})
	if err != nil {
		h.logger.Error("failed to save cloudflare connection", "error", err)
		return response.InternalError(c)
	}

	h.logger.Info("Cloudflare connected via API token",
		"user_id", user.ID,
		"token_id", tokenInfo.ID,
	)

	return response.OK(c, fiber.Map{
		"connected": true,
		"tokenId":   conn.CloudflareAccountID,
	})
}

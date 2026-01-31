package handler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/crypto"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/github"
	"github.com/paasdeploy/backend/internal/response"
)

type AuthHandler struct {
	oauthClient       *github.OAuthClient
	userRepo          domain.UserRepository
	sessionRepo       domain.SessionRepository
	tokenEncryptor    *crypto.TokenEncryptor
	logger            *slog.Logger
	sessionCookieName string
	sessionMaxAge     time.Duration
	secureCookie      bool
	frontendURL       string
}

type AuthHandlerConfig struct {
	OAuthClient       *github.OAuthClient
	UserRepo          domain.UserRepository
	SessionRepo       domain.SessionRepository
	TokenEncryptor    *crypto.TokenEncryptor
	Logger            *slog.Logger
	SessionCookieName string
	SessionMaxAge     time.Duration
	SecureCookie      bool
	FrontendURL       string
}

func NewAuthHandler(cfg AuthHandlerConfig) *AuthHandler {
	return &AuthHandler{
		oauthClient:       cfg.OAuthClient,
		userRepo:          cfg.UserRepo,
		sessionRepo:       cfg.SessionRepo,
		tokenEncryptor:    cfg.TokenEncryptor,
		logger:            cfg.Logger,
		sessionCookieName: cfg.SessionCookieName,
		sessionMaxAge:     cfg.SessionMaxAge,
		secureCookie:      cfg.SecureCookie,
		frontendURL:       cfg.FrontendURL,
	}
}

func (h *AuthHandler) Register(app *fiber.App) {
	auth := app.Group("/auth")
	auth.Get("/github", h.InitiateOAuth)
	auth.Get("/github/callback", h.HandleCallback)
}

func (h *AuthHandler) RegisterProtected(app fiber.Router) {
	app.Get("/auth/me", h.GetCurrentUser)
	app.Post("/auth/logout", h.Logout)
}

func (h *AuthHandler) setCookie(c *fiber.Ctx, name, value string, maxAge int) {
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

func (h *AuthHandler) clearCookie(c *fiber.Ctx, name string) {
	h.setCookie(c, name, "", -1)
}

func (h *AuthHandler) redirectWithError(c *fiber.Ctx, errorCode string) error {
	return c.Redirect(h.frontendURL+"/login?error="+errorCode, fiber.StatusTemporaryRedirect)
}

func (h *AuthHandler) InitiateOAuth(c *fiber.Ctx) error {
	state, err := crypto.GenerateState()
	if err != nil {
		h.logger.Error("failed to generate state", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to initiate OAuth")
	}

	h.setCookie(c, "oauth_state", state, 600)

	authURL := h.oauthClient.GetAuthorizationURL(state)
	return c.Redirect(authURL, fiber.StatusTemporaryRedirect)
}

func (h *AuthHandler) validateCallback(c *fiber.Ctx) (string, error) {
	if errorParam := c.Query("error"); errorParam != "" {
		errorDesc := c.Query("error_description")
		h.logger.Warn("OAuth error from GitHub", "error", errorParam, "description", errorDesc)
		return "", errors.New(errorParam)
	}

	code := c.Query("code")
	if code == "" {
		return "", errors.New("no_code")
	}

	state := c.Query("state")
	storedState := c.Cookies("oauth_state")
	if state != storedState {
		h.logger.Warn("Invalid OAuth state", "expected", storedState, "got", state)
		return "", errors.New("invalid_state")
	}

	return code, nil
}

type tokenData struct {
	accessTokenEncrypted  string
	refreshTokenEncrypted string
	expiresAt             *time.Time
}

func (h *AuthHandler) exchangeAndEncryptTokens(ctx context.Context, code string) (*github.TokenResponse, *tokenData, error) {
	tokenResp, err := h.oauthClient.ExchangeCodeForToken(ctx, code)
	if err != nil {
		return nil, nil, err
	}

	encryptedToken, err := h.tokenEncryptor.Encrypt(tokenResp.AccessToken)
	if err != nil {
		return nil, nil, err
	}

	data := &tokenData{accessTokenEncrypted: encryptedToken}

	if tokenResp.RefreshToken != "" {
		encryptedRefresh, err := h.tokenEncryptor.Encrypt(tokenResp.RefreshToken)
		if err != nil {
			h.logger.Error("failed to encrypt refresh token", "error", err)
		} else {
			data.refreshTokenEncrypted = encryptedRefresh
		}
	}

	if tokenResp.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		data.expiresAt = &t
	}

	return tokenResp, data, nil
}

func (h *AuthHandler) upsertUser(ctx context.Context, ghUser *github.GitHubUser, email string, tokens *tokenData) (*domain.User, error) {
	user, err := h.userRepo.FindByGitHubID(ctx, ghUser.ID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	if user == nil {
		user, err = h.userRepo.Create(ctx, domain.CreateUserInput{
			GitHubID:              ghUser.ID,
			GitHubLogin:           ghUser.Login,
			Name:                  ghUser.Name,
			Email:                 email,
			AvatarURL:             ghUser.AvatarURL,
			AccessTokenEncrypted:  tokens.accessTokenEncrypted,
			RefreshTokenEncrypted: tokens.refreshTokenEncrypted,
			TokenExpiresAt:        tokens.expiresAt,
		})
		if err != nil {
			return nil, err
		}
		h.logger.Info("new user created", "user_id", user.ID, "github_login", ghUser.Login)
	} else {
		user, err = h.userRepo.Update(ctx, user.ID, domain.UpdateUserInput{
			GitHubLogin:           &ghUser.Login,
			Name:                  &ghUser.Name,
			Email:                 &email,
			AvatarURL:             &ghUser.AvatarURL,
			AccessTokenEncrypted:  &tokens.accessTokenEncrypted,
			RefreshTokenEncrypted: &tokens.refreshTokenEncrypted,
			TokenExpiresAt:        tokens.expiresAt,
		})
		if err != nil {
			return nil, err
		}
		h.logger.Info("user updated", "user_id", user.ID, "github_login", ghUser.Login)
	}

	return user, nil
}

func (h *AuthHandler) createSession(c *fiber.Ctx, userID string) error {
	sessionToken, err := crypto.GenerateSessionToken()
	if err != nil {
		return err
	}

	tokenHash := crypto.HashSessionToken(sessionToken)
	expiresAt := time.Now().Add(h.sessionMaxAge)

	_, err = h.sessionRepo.Create(c.Context(), domain.CreateSessionInput{
		UserID:    userID,
		TokenHash: tokenHash,
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return err
	}

	h.setCookie(c, h.sessionCookieName, sessionToken, int(h.sessionMaxAge.Seconds()))
	return nil
}

func (h *AuthHandler) HandleCallback(c *fiber.Ctx) error {
	code, err := h.validateCallback(c)
	if err != nil {
		return h.redirectWithError(c, err.Error())
	}

	h.clearCookie(c, "oauth_state")

	ctx := c.Context()

	tokenResp, tokens, err := h.exchangeAndEncryptTokens(ctx, code)
	if err != nil {
		h.logger.Error("failed to exchange/encrypt tokens", "error", err)
		return h.redirectWithError(c, "token_exchange_failed")
	}

	ghUser, err := h.oauthClient.GetUser(ctx, tokenResp.AccessToken)
	if err != nil {
		h.logger.Error("failed to get GitHub user", "error", err)
		return h.redirectWithError(c, "user_fetch_failed")
	}

	email := ghUser.Email
	if email == "" {
		email, _ = h.oauthClient.GetPrimaryEmail(ctx, tokenResp.AccessToken)
	}

	user, err := h.upsertUser(ctx, ghUser, email, tokens)
	if err != nil {
		h.logger.Error("failed to upsert user", "error", err)
		return h.redirectWithError(c, "database_error")
	}

	if err := h.createSession(c, user.ID); err != nil {
		h.logger.Error("failed to create session", "error", err)
		return h.redirectWithError(c, "session_error")
	}

	return c.Redirect(h.frontendURL+"/?login=success", fiber.StatusTemporaryRedirect)
}

func (h *AuthHandler) GetCurrentUser(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, "not authenticated")
	}

	return response.OK(c, UserResponse{
		ID:          user.ID,
		GitHubID:    user.GitHubID,
		GitHubLogin: user.GitHubLogin,
		Name:        user.Name,
		Email:       user.Email,
		AvatarURL:   user.AvatarURL,
		CreatedAt:   user.CreatedAt,
	})
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	sessionToken := c.Cookies(h.sessionCookieName)
	if sessionToken != "" {
		tokenHash := crypto.HashSessionToken(sessionToken)
		session, err := h.sessionRepo.FindByTokenHash(c.Context(), tokenHash)
		if err == nil && session != nil {
			if delErr := h.sessionRepo.Delete(c.Context(), session.ID); delErr != nil {
				h.logger.Warn("failed to delete session", "error", delErr)
			}
		}
	}

	h.clearCookie(c, h.sessionCookieName)

	return response.OK(c, map[string]string{"message": "logged out"})
}

type UserResponse struct {
	ID          string    `json:"id"`
	GitHubID    int64     `json:"githubId"`
	GitHubLogin string    `json:"githubLogin"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	AvatarURL   string    `json:"avatarUrl"`
	CreatedAt   time.Time `json:"createdAt"`
}

const userContextKey = "user"

func SetUserInContext(c *fiber.Ctx, user *domain.User) {
	c.Locals(userContextKey, user)
}

func GetUserFromContext(c *fiber.Ctx) *domain.User {
	user, ok := c.Locals(userContextKey).(*domain.User)
	if !ok {
		return nil
	}
	return user
}

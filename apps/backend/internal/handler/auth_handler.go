package handler

import (
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

func (h *AuthHandler) InitiateOAuth(c *fiber.Ctx) error {
	state, err := crypto.GenerateState()
	if err != nil {
		h.logger.Error("failed to generate state", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError, "failed to initiate OAuth")
	}

	c.Cookie(&fiber.Cookie{
		Name:     "oauth_state",
		Value:    state,
		HTTPOnly: true,
		Secure:   h.secureCookie,
		SameSite: "Lax",
		MaxAge:   600,
		Path:     "/",
	})

	authURL := h.oauthClient.GetAuthorizationURL(state)
	return c.Redirect(authURL, fiber.StatusTemporaryRedirect)
}

func (h *AuthHandler) HandleCallback(c *fiber.Ctx) error {
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")

	if errorParam != "" {
		errorDesc := c.Query("error_description")
		h.logger.Warn("OAuth error from GitHub", "error", errorParam, "description", errorDesc)
		return c.Redirect(h.frontendURL+"/login?error="+errorParam, fiber.StatusTemporaryRedirect)
	}

	if code == "" {
		return c.Redirect(h.frontendURL+"/login?error=no_code", fiber.StatusTemporaryRedirect)
	}

	storedState := c.Cookies("oauth_state")
	if state != storedState {
		h.logger.Warn("Invalid OAuth state", "expected", storedState, "got", state)
		return c.Redirect(h.frontendURL+"/login?error=invalid_state", fiber.StatusTemporaryRedirect)
	}

	c.Cookie(&fiber.Cookie{
		Name:   "oauth_state",
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	})

	ctx := c.Context()

	tokenResp, err := h.oauthClient.ExchangeCodeForToken(ctx, code)
	if err != nil {
		h.logger.Error("failed to exchange code for token", "error", err)
		return c.Redirect(h.frontendURL+"/login?error=token_exchange_failed", fiber.StatusTemporaryRedirect)
	}

	ghUser, err := h.oauthClient.GetUser(ctx, tokenResp.AccessToken)
	if err != nil {
		h.logger.Error("failed to get GitHub user", "error", err)
		return c.Redirect(h.frontendURL+"/login?error=user_fetch_failed", fiber.StatusTemporaryRedirect)
	}

	email := ghUser.Email
	if email == "" {
		email, _ = h.oauthClient.GetPrimaryEmail(ctx, tokenResp.AccessToken)
	}

	encryptedToken, err := h.tokenEncryptor.Encrypt(tokenResp.AccessToken)
	if err != nil {
		h.logger.Error("failed to encrypt access token", "error", err)
		return c.Redirect(h.frontendURL+"/login?error=encryption_failed", fiber.StatusTemporaryRedirect)
	}

	var encryptedRefresh string
	if tokenResp.RefreshToken != "" {
		encryptedRefresh, err = h.tokenEncryptor.Encrypt(tokenResp.RefreshToken)
		if err != nil {
			h.logger.Error("failed to encrypt refresh token", "error", err)
		}
	}

	var tokenExpiresAt *time.Time
	if tokenResp.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		tokenExpiresAt = &t
	}

	user, err := h.userRepo.FindByGitHubID(ctx, ghUser.ID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		h.logger.Error("failed to find user", "error", err)
		return c.Redirect(h.frontendURL+"/login?error=database_error", fiber.StatusTemporaryRedirect)
	}

	if user == nil {
		user, err = h.userRepo.Create(ctx, domain.CreateUserInput{
			GitHubID:              ghUser.ID,
			GitHubLogin:           ghUser.Login,
			Name:                  ghUser.Name,
			Email:                 email,
			AvatarURL:             ghUser.AvatarURL,
			AccessTokenEncrypted:  encryptedToken,
			RefreshTokenEncrypted: encryptedRefresh,
			TokenExpiresAt:        tokenExpiresAt,
		})
		if err != nil {
			h.logger.Error("failed to create user", "error", err)
			return c.Redirect(h.frontendURL+"/login?error=user_creation_failed", fiber.StatusTemporaryRedirect)
		}
		h.logger.Info("new user created", "user_id", user.ID, "github_login", ghUser.Login)
	} else {
		user, err = h.userRepo.Update(ctx, user.ID, domain.UpdateUserInput{
			GitHubLogin:           &ghUser.Login,
			Name:                  &ghUser.Name,
			Email:                 &email,
			AvatarURL:             &ghUser.AvatarURL,
			AccessTokenEncrypted:  &encryptedToken,
			RefreshTokenEncrypted: &encryptedRefresh,
			TokenExpiresAt:        tokenExpiresAt,
		})
		if err != nil {
			h.logger.Error("failed to update user", "error", err)
			return c.Redirect(h.frontendURL+"/login?error=user_update_failed", fiber.StatusTemporaryRedirect)
		}
		h.logger.Info("user updated", "user_id", user.ID, "github_login", ghUser.Login)
	}

	sessionToken, err := crypto.GenerateSessionToken()
	if err != nil {
		h.logger.Error("failed to generate session token", "error", err)
		return c.Redirect(h.frontendURL+"/login?error=session_error", fiber.StatusTemporaryRedirect)
	}

	tokenHash := crypto.HashSessionToken(sessionToken)
	expiresAt := time.Now().Add(h.sessionMaxAge)

	_, err = h.sessionRepo.Create(ctx, domain.CreateSessionInput{
		UserID:    user.ID,
		TokenHash: tokenHash,
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
		ExpiresAt: expiresAt,
	})
	if err != nil {
		h.logger.Error("failed to create session", "error", err)
		return c.Redirect(h.frontendURL+"/login?error=session_error", fiber.StatusTemporaryRedirect)
	}

	c.Cookie(&fiber.Cookie{
		Name:     h.sessionCookieName,
		Value:    sessionToken,
		HTTPOnly: true,
		Secure:   h.secureCookie,
		SameSite: "Lax",
		MaxAge:   int(h.sessionMaxAge.Seconds()),
		Path:     "/",
	})

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
			_ = h.sessionRepo.Delete(c.Context(), session.ID)
		}
	}

	c.Cookie(&fiber.Cookie{
		Name:     h.sessionCookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HTTPOnly: true,
		Secure:   h.secureCookie,
		SameSite: "Lax",
	})

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

package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/mail"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/paasdeploy/backend/internal/crypto"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/ghclient"
	"github.com/paasdeploy/backend/internal/password"
	"github.com/paasdeploy/backend/internal/response"
	"github.com/paasdeploy/backend/internal/service"
)

type AuthHandler struct {
	oauthClient       *ghclient.OAuthClient
	userRepo          domain.UserRepository
	sessionRepo       domain.SessionRepository
	tokenEncryptor    *crypto.TokenEncryptor
	auditService      *service.AuditService
	logger            *slog.Logger
	sessionCookieName string
	sessionMaxAge     time.Duration
	secureCookie      bool
	cookieDomain      string
	frontendURL       string
}

type AuthHandlerConfig struct {
	OAuthClient       *ghclient.OAuthClient
	UserRepo          domain.UserRepository
	SessionRepo       domain.SessionRepository
	TokenEncryptor    *crypto.TokenEncryptor
	AuditService      *service.AuditService
	Logger            *slog.Logger
	SessionCookieName string
	SessionMaxAge     time.Duration
	SecureCookie      bool
	CookieDomain      string
	FrontendURL       string
}

func NewAuthHandler(cfg AuthHandlerConfig) *AuthHandler {
	return &AuthHandler{
		oauthClient:       cfg.OAuthClient,
		userRepo:          cfg.UserRepo,
		sessionRepo:       cfg.SessionRepo,
		tokenEncryptor:    cfg.TokenEncryptor,
		auditService:      cfg.AuditService,
		logger:            cfg.Logger,
		sessionCookieName: cfg.SessionCookieName,
		sessionMaxAge:     cfg.SessionMaxAge,
		secureCookie:      cfg.SecureCookie,
		cookieDomain:      cfg.CookieDomain,
		frontendURL:       cfg.FrontendURL,
	}
}

func (h *AuthHandler) Register(app *fiber.App) {
	auth := app.Group("/auth")
	auth.Post("/register", h.RegisterEmail)
	auth.Post("/login", h.LoginEmail)
	auth.Get("/github", h.InitiateOAuth)
	auth.Get("/github/callback", h.HandleCallback)
}

func (h *AuthHandler) RegisterProtected(app fiber.Router) {
	app.Get("/auth/me", h.GetCurrentUser)
	app.Post("/auth/logout", h.Logout)
	app.Post("/auth/link-github", h.LinkGitHub)
}

func (h *AuthHandler) setCookie(c *fiber.Ctx, name, value string, maxAge int) {
	sameSite := "Lax"
	if h.cookieDomain != "" {
		sameSite = "None"
	}
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    value,
		HTTPOnly: true,
		Secure:   h.secureCookie,
		SameSite: sameSite,
		MaxAge:   maxAge,
		Path:     "/",
		Domain:   h.cookieDomain,
	})
}

func (h *AuthHandler) clearCookie(c *fiber.Ctx, name string) {
	h.setCookie(c, name, "", -1)
}

func (h *AuthHandler) redirectWithError(c *fiber.Ctx, errorCode string) error {
	return c.Redirect(h.frontendURL+"/login?error="+errorCode, fiber.StatusTemporaryRedirect)
}

const (
	minPasswordLength     = 8
	errMsgSessionCreation = "failed to create session"
	errMsgInvalidCreds    = "invalid email or password"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) RegisterEmail(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if _, err := mail.ParseAddress(req.Email); err != nil {
		return response.BadRequest(c, "invalid email format")
	}

	if len(req.Password) < minPasswordLength {
		return response.BadRequest(c, "password must be at least 8 characters")
	}

	if req.Name == "" {
		return response.BadRequest(c, "name is required")
	}

	hash, err := password.Hash(req.Password)
	if err != nil {
		h.logger.Error("failed to hash password", "error", err)
		return response.InternalError(c)
	}

	existing, err := h.userRepo.FindByEmail(c.Context(), req.Email)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		h.logger.Error("failed to check existing email", "error", err)
		return response.InternalError(c)
	}

	var user *domain.User

	if existing != nil {
		if existing.PasswordHash != "" {
			return response.Conflict(c, "email already registered")
		}

		user, err = h.userRepo.SetPassword(c.Context(), existing.ID, hash)
		if err != nil {
			h.logger.Error("failed to set password on existing user", "error", err)
			return response.InternalError(c)
		}
		h.logger.Info("password added to existing account", "user_id", user.ID, "email", req.Email)
	} else {
		user, err = h.userRepo.CreateEmailUser(c.Context(), domain.CreateEmailUserInput{
			Email:        req.Email,
			Name:         req.Name,
			PasswordHash: hash,
		})
		if err != nil {
			h.logger.Error("failed to create email user", "error", err)
			return response.InternalError(c)
		}
		h.logger.Info("email user registered", "user_id", user.ID, "email", req.Email)
	}

	if err := h.createSession(c, user.ID); err != nil {
		h.logger.Error(errMsgSessionCreation, "error", err)
		return response.InternalError(c)
	}

	return response.Created(c, UserResponse{
		ID:           user.ID,
		GitHubID:     user.GitHubID,
		GitHubLogin:  user.GitHubLogin,
		Name:         user.Name,
		Email:        user.Email,
		AvatarURL:    user.AvatarURL,
		AuthProvider: user.AuthProvider,
		CreatedAt:    user.CreatedAt,
	})
}

func (h *AuthHandler) LoginEmail(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if req.Email == "" || req.Password == "" {
		return response.BadRequest(c, "email and password are required")
	}

	user, err := h.userRepo.FindByEmail(c.Context(), req.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return response.Unauthorized(c, errMsgInvalidCreds)
		}
		h.logger.Error("failed to find user by email", "error", err)
		return response.InternalError(c)
	}

	if user.PasswordHash == "" {
		return response.BadRequest(c, "This account uses GitHub sign-in. Please set a password via the Sign Up page to enable email login.")
	}

	if err := password.Verify(user.PasswordHash, req.Password); err != nil {
		return response.Unauthorized(c, errMsgInvalidCreds)
	}

	if err := h.createSession(c, user.ID); err != nil {
		h.logger.Error(errMsgSessionCreation, "error", err)
		return response.InternalError(c)
	}

	if h.auditService != nil {
		auditCtx := h.auditService.ExtractContext(c)
		auditCtx.UserID = &user.ID
		auditCtx.UserName = &user.Name
		h.auditService.LogUserLoggedIn(c.Context(), auditCtx, user.ID, user.Name)
	}

	return response.OK(c, UserResponse{
		ID:           user.ID,
		GitHubID:     user.GitHubID,
		GitHubLogin:  user.GitHubLogin,
		Name:         user.Name,
		Email:        user.Email,
		AvatarURL:    user.AvatarURL,
		AuthProvider: user.AuthProvider,
		CreatedAt:    user.CreatedAt,
	})
}

func (h *AuthHandler) LinkGitHub(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, "not authenticated")
	}

	if user.GitHubID != nil {
		return response.Conflict(c, "GitHub account already linked")
	}

	state, err := crypto.GenerateState()
	if err != nil {
		h.logger.Error("failed to generate state", "error", err)
		return response.InternalError(c)
	}

	h.setCookie(c, "oauth_state", state, 600)
	h.setCookie(c, "oauth_link_mode", "1", 600)

	authURL := h.oauthClient.GetAuthorizationURL(state)
	return response.OK(c, map[string]string{"redirectUrl": authURL})
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

func (h *AuthHandler) exchangeAndEncryptTokens(ctx context.Context, code string) (*ghclient.TokenResponse, *tokenData, error) {
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

func (h *AuthHandler) upsertUser(ctx context.Context, ghUser *ghclient.GitHubUser, email string, tokens *tokenData) (*domain.User, error) {
	user, err := h.userRepo.FindByGitHubID(ctx, ghUser.ID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	linkInput := domain.LinkGitHubInput{
		GitHubID:              ghUser.ID,
		GitHubLogin:           ghUser.Login,
		Name:                  ghUser.Name,
		Email:                 email,
		AvatarURL:             ghUser.AvatarURL,
		AccessTokenEncrypted:  tokens.accessTokenEncrypted,
		RefreshTokenEncrypted: tokens.refreshTokenEncrypted,
		TokenExpiresAt:        tokens.expiresAt,
	}

	if user == nil && email != "" {
		user, err = h.tryLinkByEmail(ctx, email, linkInput)
		if err != nil {
			return nil, err
		}
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
	} else if user.GitHubID != nil {
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

func (h *AuthHandler) tryLinkByEmail(ctx context.Context, email string, input domain.LinkGitHubInput) (*domain.User, error) {
	emailUser, err := h.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if emailUser.GitHubID != nil {
		return nil, nil
	}

	linked, err := h.userRepo.LinkGitHub(ctx, emailUser.ID, input)
	if err != nil {
		return nil, err
	}
	h.logger.Info("github linked to existing email user", "user_id", linked.ID, "github_login", input.GitHubLogin)
	return linked, nil
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

	linkMode := c.Cookies("oauth_link_mode") == "1"
	h.clearCookie(c, "oauth_link_mode")

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

	if linkMode {
		return h.handleLinkCallback(c, ghUser, email, tokens)
	}

	user, err := h.upsertUser(ctx, ghUser, email, tokens)
	if err != nil {
		h.logger.Error("failed to upsert user", "error", err)
		return h.redirectWithError(c, "database_error")
	}

	if err := h.createSession(c, user.ID); err != nil {
		h.logger.Error(errMsgSessionCreation, "error", err)
		return h.redirectWithError(c, "session_error")
	}

	if h.auditService != nil {
		auditCtx := h.auditService.ExtractContext(c)
		auditCtx.UserID = &user.ID
		auditCtx.UserName = &user.Name
		h.auditService.LogUserLoggedIn(c.Context(), auditCtx, user.ID, user.Name)
	}

	return c.Redirect(h.frontendURL+"/?login=success", fiber.StatusTemporaryRedirect)
}

func (h *AuthHandler) handleLinkCallback(c *fiber.Ctx, ghUser *ghclient.GitHubUser, email string, tokens *tokenData) error {
	currentUser := h.getLoggedInUser(c)
	if currentUser == nil {
		return h.redirectWithError(c, "not_authenticated")
	}

	ctx := c.Context()

	existingGH, err := h.userRepo.FindByGitHubID(ctx, ghUser.ID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		h.logger.Error("failed to check existing github user", "error", err)
		return h.redirectWithError(c, "database_error")
	}
	if existingGH != nil {
		return h.redirectWithError(c, "github_already_linked")
	}

	_, err = h.userRepo.LinkGitHub(ctx, currentUser.ID, domain.LinkGitHubInput{
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
		h.logger.Error("failed to link github", "error", err)
		return h.redirectWithError(c, "link_failed")
	}

	h.logger.Info("github linked to user", "user_id", currentUser.ID, "github_login", ghUser.Login)
	return c.Redirect(h.frontendURL+"/settings?github_linked=true", fiber.StatusTemporaryRedirect)
}

func (h *AuthHandler) getLoggedInUser(c *fiber.Ctx) *domain.User {
	sessionToken := c.Cookies(h.sessionCookieName)
	if sessionToken == "" {
		return nil
	}

	tokenHash := crypto.HashSessionToken(sessionToken)
	session, err := h.sessionRepo.FindByTokenHash(c.Context(), tokenHash)
	if err != nil || session == nil {
		return nil
	}

	user, err := h.userRepo.FindByID(c.Context(), session.UserID)
	if err != nil {
		return nil
	}

	return user
}

func (h *AuthHandler) GetCurrentUser(c *fiber.Ctx) error {
	user := GetUserFromContext(c)
	if user == nil {
		return response.Unauthorized(c, "not authenticated")
	}

	return response.OK(c, UserResponse{
		ID:           user.ID,
		GitHubID:     user.GitHubID,
		GitHubLogin:  user.GitHubLogin,
		Name:         user.Name,
		Email:        user.Email,
		AvatarURL:    user.AvatarURL,
		AuthProvider: user.AuthProvider,
		CreatedAt:    user.CreatedAt,
	})
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	sessionToken := c.Cookies(h.sessionCookieName)
	if sessionToken != "" {
		tokenHash := crypto.HashSessionToken(sessionToken)
		session, err := h.sessionRepo.FindByTokenHash(c.Context(), tokenHash)
		if err == nil && session != nil {
			if h.auditService != nil {
				user, userErr := h.userRepo.FindByID(c.Context(), session.UserID)
				if userErr == nil && user != nil {
					auditCtx := h.auditService.ExtractContext(c)
					h.auditService.LogUserLoggedOut(c.Context(), auditCtx, user.ID, user.Name)
				}
			}
			if delErr := h.sessionRepo.Delete(c.Context(), session.ID); delErr != nil {
				h.logger.Warn("failed to delete session", "error", delErr)
			}
		}
	}

	h.clearCookie(c, h.sessionCookieName)

	return response.OK(c, map[string]string{"message": "logged out"})
}

type UserResponse struct {
	ID           string    `json:"id"`
	GitHubID     *int64    `json:"githubId,omitempty"`
	GitHubLogin  string    `json:"githubLogin,omitempty"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	AvatarURL    string    `json:"avatarUrl,omitempty"`
	AuthProvider string    `json:"authProvider"`
	CreatedAt    time.Time `json:"createdAt"`
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

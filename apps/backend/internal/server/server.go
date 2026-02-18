package server

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/paasdeploy/backend/internal/middleware"
	"github.com/paasdeploy/backend/internal/response"
)

const (
	apiRateLimitMax     = 120
	apiRateLimitWindow  = 1 * time.Minute
	authRateLimitMax    = 10
	authRateLimitWindow = 1 * time.Minute
)

type Config struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	CorsOrigins  string
}

type Server struct {
	app    *fiber.App
	config Config
	logger *slog.Logger
}

func New(cfg Config, log *slog.Logger) *Server {
	app := fiber.New(fiber.Config{
		AppName:               "FlowDeploy API",
		ReadTimeout:           cfg.ReadTimeout,
		WriteTimeout:          cfg.WriteTimeout,
		IdleTimeout:           cfg.IdleTimeout,
		DisableStartupMessage: true,
		ErrorHandler:          customErrorHandler(log),
	})

	s := &Server{
		app:    app,
		config: cfg,
		logger: log,
	}

	s.setupMiddlewares()

	return s
}

func (s *Server) setupMiddlewares() {
	s.app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	s.app.Use(middleware.TraceID())

	s.app.Use(securityHeaders)

	corsOrigins := s.config.CorsOrigins
	if corsOrigins == "*" || corsOrigins == "" {
		s.logger.Warn("CORS_ORIGINS is wildcard or empty; in production, set explicit origins")
	}
	corsConfig := cors.Config{
		AllowOrigins:  corsOrigins,
		AllowMethods:  "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:  "Content-Type,Authorization,X-Trace-ID,X-GitHub-Event,X-Hub-Signature-256,X-GitHub-Delivery",
		ExposeHeaders: "X-Trace-ID",
	}
	if corsOrigins != "*" && corsOrigins != "" {
		corsConfig.AllowCredentials = true
	}
	s.app.Use(cors.New(corsConfig))

	s.app.Use(logger.New(logger.Config{
		Format:     "${time} | ${status} | ${latency} | ${method} ${path} | trace=${locals:traceId}\n",
		TimeFormat: "2006-01-02 15:04:05",
		Output:     nil,
		Next: func(c *fiber.Ctx) bool {
			return c.Path() == "/health"
		},
	}))

	s.app.Use(limiter.New(limiter.Config{
		Max:        apiRateLimitMax,
		Expiration: apiRateLimitWindow,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(response.Envelope{
				Success: false,
				Error: &response.ErrorInfo{
					Code:    response.ErrCodeRateLimited,
					Message: "too many requests",
				},
			})
		},
		Next: func(c *fiber.Ctx) bool {
			return c.Path() == "/health" || c.Path() == "/events/deploys"
		},
	}))
}

func AuthRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        authRateLimitMax,
		Expiration: authRateLimitWindow,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(response.Envelope{
				Success: false,
				Error: &response.ErrorInfo{
					Code:    response.ErrCodeRateLimited,
					Message: "too many authentication attempts",
				},
			})
		},
	})
}

func securityHeaders(c *fiber.Ctx) error {
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("X-Frame-Options", "DENY")
	c.Set("X-XSS-Protection", "1; mode=block")
	c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	c.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	if c.Protocol() == "https" {
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	}
	return c.Next()
}

func (s *Server) App() *fiber.App {
	return s.app
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.logger.Info("Server listening", "addr", addr)
	return s.app.Listen(addr)
}

func (s *Server) Shutdown() error {
	s.logger.Info("Shutting down server...")
	return s.app.Shutdown()
}

func customErrorHandler(log *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		errCode := response.ErrCodeInternal
		message := "internal server error"

		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
			message = e.Message

			switch code {
			case fiber.StatusBadRequest:
				errCode = response.ErrCodeInvalidPayload
			case fiber.StatusUnauthorized:
				errCode = response.ErrCodeUnauthorized
			case fiber.StatusForbidden:
				errCode = response.ErrCodeForbidden
			case fiber.StatusNotFound:
				errCode = response.ErrCodeNotFound
			case fiber.StatusConflict:
				errCode = response.ErrCodeConflict
			case fiber.StatusTooManyRequests:
				errCode = response.ErrCodeRateLimited
			}
		}

		traceID := middleware.GetTraceID(c)

		log.Error("Request error",
			"path", c.Path(),
			"method", c.Method(),
			"error", err.Error(),
			"status", code,
			"traceId", traceID,
		)

		return c.Status(code).JSON(response.Envelope{
			Success: false,
			Data:    nil,
			Error: &response.ErrorInfo{
				Code:    errCode,
				Message: message,
			},
			Meta: response.Meta{
				TraceID: traceID,
			},
		})
	}
}

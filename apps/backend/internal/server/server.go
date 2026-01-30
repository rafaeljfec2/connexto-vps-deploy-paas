package server

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/paasdeploy/backend/internal/middleware"
	"github.com/paasdeploy/backend/internal/response"
)

type Config struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
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

	s.app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Content-Type,Authorization,X-Trace-ID",
		ExposeHeaders: "X-Trace-ID",
	}))

	s.app.Use(logger.New(logger.Config{
		Format:     "${time} | ${status} | ${latency} | ${method} ${path} | trace=${locals:traceId}\n",
		TimeFormat: "2006-01-02 15:04:05",
		Output:     nil,
		Next: func(c *fiber.Ctx) bool {
			return c.Path() == "/health"
		},
	}))
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

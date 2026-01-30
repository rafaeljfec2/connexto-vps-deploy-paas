package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	TraceIDHeader = "X-Trace-ID"
	TraceIDKey    = "traceId"
)

func TraceID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		traceID := c.Get(TraceIDHeader)

		if traceID == "" {
			traceID = uuid.New().String()
		}

		c.Locals(TraceIDKey, traceID)
		c.Set(TraceIDHeader, traceID)

		return c.Next()
	}
}

func GetTraceID(c *fiber.Ctx) string {
	if traceID := c.Locals(TraceIDKey); traceID != nil {
		if id, ok := traceID.(string); ok {
			return id
		}
	}
	return ""
}

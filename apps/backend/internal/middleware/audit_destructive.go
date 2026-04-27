package middleware

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/paasdeploy/backend/internal/domain"
	"github.com/paasdeploy/backend/internal/service"
)

const (
	auditServiceLocalsKey = "_audit_service"
	auditDoneLocalsKey    = "_audit_done"
)

// WithAuditService injects the *AuditService in the request context so the
// AuditDestructive middleware can pick it up without leaking the dependency to
// every individual handler. Should be installed once at server bootstrap, after
// TraceID/Auth and before the destructive route groups.
func WithAuditService(svc *service.AuditService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals(auditServiceLocalsKey, svc)
		return c.Next()
	}
}

// AuditDestructive writes an audit log entry AFTER a destructive handler
// returns successfully (status < 400 and the request was not a dry-run). It is
// designed to wrap a handler in the destructive route registration:
//
//	apps.Delete("/:id",
//	    middleware.RequireScope(domain.ScopeDestructive),
//	    middleware.AuditDestructive(domain.EventAppDeleted, domain.ResourceApp, "id"),
//	    h.DeleteApp,
//	)
//
// The destructive handler can call MarkAuditDone(c) to signal that it already
// wrote a richer audit log itself; the middleware then skips silently to avoid
// duplicates.
func AuditDestructive(eventType domain.EventType, resourceType domain.ResourceType, paramKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := c.Next(); err != nil {
			return err
		}
		if IsDryRun(c) {
			return nil
		}
		if status := c.Response().StatusCode(); status >= 400 {
			return nil
		}
		if done, _ := c.Locals(auditDoneLocalsKey).(bool); done {
			return nil
		}
		svc, ok := c.Locals(auditServiceLocalsKey).(*service.AuditService)
		if !ok || svc == nil {
			slog.Default().Error(
				"audit log skipped: WithAuditService middleware not installed for destructive route",
				"path", c.Path(),
				"method", c.Method(),
				"event", string(eventType),
				"resource", string(resourceType),
			)
			return nil
		}

		var resourceID, resourceName *string
		if paramKey != "" {
			if v := c.Params(paramKey); v != "" {
				if _, err := uuid.Parse(v); err == nil {
					resourceID = &v
				} else {
					resourceName = &v
				}
			}
		}

		ctx := svc.ExtractContext(c)
		svc.Log(c.Context(), ctx, eventType, resourceType, resourceID, resourceName, nil)
		return nil
	}
}

// MarkAuditDone is called by handlers that already write a rich audit log
// entry on their own; AuditDestructive will then skip the generic write.
func MarkAuditDone(c *fiber.Ctx) {
	c.Locals(auditDoneLocalsKey, true)
}

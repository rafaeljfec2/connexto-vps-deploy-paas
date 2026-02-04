package response

import (
	"github.com/gofiber/fiber/v2"
)

type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   *ErrorInfo  `json:"error"`
	Meta    Meta        `json:"meta"`
}

type ErrorInfo struct {
	Code    ErrorCode   `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type Meta struct {
	TraceID    string      `json:"traceId,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
	Warnings   []string    `json:"warnings,omitempty"`
}

type Pagination struct {
	Page    int `json:"page"`
	PerPage int `json:"perPage"`
	Total   int `json:"total"`
}

type ErrorCode string

const (
	ErrCodeInvalidPayload ErrorCode = "INVALID_PAYLOAD"
	ErrCodeUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden      ErrorCode = "FORBIDDEN"
	ErrCodeNotFound       ErrorCode = "NOT_FOUND"
	ErrCodeConflict       ErrorCode = "CONFLICT"
	ErrCodeRateLimited    ErrorCode = "RATE_LIMITED"
	ErrCodeInternal       ErrorCode = "INTERNAL_ERROR"
)

func OK(c *fiber.Ctx, data interface{}) error {
	return send(c, fiber.StatusOK, data, nil)
}

func Created(c *fiber.Ctx, data interface{}) error {
	return send(c, fiber.StatusCreated, data, nil)
}

func Accepted(c *fiber.Ctx, data interface{}) error {
	return send(c, fiber.StatusAccepted, data, nil)
}

func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

func OKWithPagination(c *fiber.Ctx, data interface{}, page, perPage, total int) error {
	meta := Meta{
		TraceID: getTraceID(c),
		Pagination: &Pagination{
			Page:    page,
			PerPage: perPage,
			Total:   total,
		},
	}
	return sendWithMeta(c, fiber.StatusOK, data, nil, meta)
}

func BadRequest(c *fiber.Ctx, message string) error {
	return sendError(c, fiber.StatusBadRequest, ErrCodeInvalidPayload, message, nil)
}

func BadRequestWithDetails(c *fiber.Ctx, message string, details interface{}) error {
	return sendError(c, fiber.StatusBadRequest, ErrCodeInvalidPayload, message, details)
}

func Unauthorized(c *fiber.Ctx, message string) error {
	return sendError(c, fiber.StatusUnauthorized, ErrCodeUnauthorized, message, nil)
}

func Forbidden(c *fiber.Ctx, message string) error {
	return sendError(c, fiber.StatusForbidden, ErrCodeForbidden, message, nil)
}

func NotFound(c *fiber.Ctx, message string) error {
	return sendError(c, fiber.StatusNotFound, ErrCodeNotFound, message, nil)
}

func Conflict(c *fiber.Ctx, message string) error {
	return sendError(c, fiber.StatusConflict, ErrCodeConflict, message, nil)
}

func RateLimited(c *fiber.Ctx, message string) error {
	return sendError(c, fiber.StatusTooManyRequests, ErrCodeRateLimited, message, nil)
}

func InternalError(c *fiber.Ctx) error {
	return sendError(c, fiber.StatusInternalServerError, ErrCodeInternal, "internal server error", nil)
}

func ServerError(c *fiber.Ctx, status int, message string) error {
	return sendError(c, status, ErrCodeInternal, message, nil)
}

func send(c *fiber.Ctx, status int, data interface{}, errInfo *ErrorInfo) error {
	meta := Meta{
		TraceID: getTraceID(c),
	}
	return sendWithMeta(c, status, data, errInfo, meta)
}

func sendWithMeta(c *fiber.Ctx, status int, data interface{}, errInfo *ErrorInfo, meta Meta) error {
	if meta.TraceID == "" {
		meta.TraceID = getTraceID(c)
	}

	envelope := Envelope{
		Success: errInfo == nil,
		Data:    data,
		Error:   errInfo,
		Meta:    meta,
	}

	return c.Status(status).JSON(envelope)
}

func sendError(c *fiber.Ctx, status int, code ErrorCode, message string, details interface{}) error {
	errInfo := &ErrorInfo{
		Code:    code,
		Message: message,
		Details: details,
	}
	return send(c, status, nil, errInfo)
}

func getTraceID(c *fiber.Ctx) string {
	if traceID := c.Locals("traceId"); traceID != nil {
		if id, ok := traceID.(string); ok {
			return id
		}
	}
	return ""
}

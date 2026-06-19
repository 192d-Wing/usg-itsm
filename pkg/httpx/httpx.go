// Package httpx provides a uniform JSON error envelope and health handlers
// shared by every Fiber service.
package httpx

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

// ErrorBody is the canonical error response shape returned by all services.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail describes a single error.
type ErrorDetail struct {
	// Code is a stable, machine-readable identifier, e.g. "unauthorized".
	Code string `json:"code"`
	// Message is a human-readable explanation.
	Message string `json:"message"`
	// RequestID correlates the error with logs/traces when available.
	RequestID string `json:"request_id,omitempty"`
}

// DefaultErrorHandler is the Fiber ErrorHandler all services install. It maps
// fiber.Error (and unknown errors) onto the canonical error envelope with a
// stable code derived from the HTTP status.
func DefaultErrorHandler(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	var fe *fiber.Error
	if errors.As(err, &fe) {
		status = fe.Code
	}
	return WriteError(c, status, CodeForStatus(status), err.Error())
}

// CodeForStatus returns a stable machine-readable code for an HTTP status.
func CodeForStatus(status int) string {
	switch status {
	case fiber.StatusBadRequest:
		return "bad_request"
	case fiber.StatusUnauthorized:
		return "unauthorized"
	case fiber.StatusForbidden:
		return "forbidden"
	case fiber.StatusNotFound:
		return "not_found"
	case fiber.StatusMethodNotAllowed:
		return "method_not_allowed"
	case fiber.StatusConflict:
		return "conflict"
	case fiber.StatusUnprocessableEntity:
		return "unprocessable_entity"
	case fiber.StatusTooManyRequests:
		return "rate_limited"
	case fiber.StatusServiceUnavailable:
		return "unavailable"
	default:
		if status >= 500 {
			return "internal_error"
		}
		return "error"
	}
}

// WriteError sends a structured error response with the given HTTP status.
func WriteError(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(ErrorBody{
		Error: ErrorDetail{
			Code:      code,
			Message:   message,
			RequestID: requestID(c),
		},
	})
}

// requestID returns the request id set by Fiber's requestid middleware.
func requestID(c *fiber.Ctx) string {
	if v, ok := c.Locals("requestid").(string); ok {
		return v
	}
	return ""
}

// Health registers liveness and readiness endpoints on the app.
//
// ready may be nil, in which case readiness always reports ok.
func Health(app *fiber.App, ready func() error) {
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Get("/readyz", func(c *fiber.Ctx) error {
		if ready != nil {
			if err := ready(); err != nil {
				return WriteError(c, fiber.StatusServiceUnavailable, "not_ready", err.Error())
			}
		}
		return c.JSON(fiber.Map{"status": "ready"})
	})
}

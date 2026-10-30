package response

import (
	"errors"
	"net/http"

	"echobackend/pkg/applog"
	"echobackend/pkg/validator"

	"github.com/labstack/echo/v5"
)

var log = applog.Component("api")

// APIResponse represents the standard API response format
type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Errors  any    `json:"errors,omitempty"`
	Meta    any    `json:"meta,omitempty"`
}

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	TotalItems int `json:"total_items"`
	Offset     int `json:"offset"`
	Limit      int `json:"limit"`
	TotalPages int `json:"total_pages"`
}

// Success sends a successful response
func Success(c *echo.Context, message string, data any) error {
	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SuccessWithMeta sends a successful response with metadata
func SuccessWithMeta(c *echo.Context, message string, data any, meta any) error {
	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// Created sends a created response
func Created(c *echo.Context, message string, data any) error {
	return c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// BadRequest sends a bad request error response
func BadRequest(c *echo.Context, message string, err error) error {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	log.Warn("bad request",
		"message", message,
		"error", errorMsg,
	)

	return c.JSON(http.StatusBadRequest, APIResponse{
		Success: false,
		Message: message,
		Error:   errorMsg,
	})
}

// Unauthorized sends an unauthorized error response
func Unauthorized(c *echo.Context, message string) error {
	log.Warn("unauthorized",
		"message", message,
	)

	return c.JSON(http.StatusUnauthorized, APIResponse{
		Success: false,
		Message: message,
		Error:   "Unauthorized access",
	})
}

// Forbidden sends a forbidden error response
func Forbidden(c *echo.Context, message string) error {
	log.Warn("forbidden",
		"message", message,
	)

	return c.JSON(http.StatusForbidden, APIResponse{
		Success: false,
		Message: message,
		Error:   "Access forbidden",
	})
}

// TooManyRequests sends a 429 rate-limit response.
func TooManyRequests(c *echo.Context, message string) error {
	log.Warn("too many requests",
		"message", message,
	)

	return c.JSON(http.StatusTooManyRequests, APIResponse{
		Success: false,
		Message: message,
		Error:   "Rate limit exceeded",
	})
}

// NotFound sends a not found error response
func NotFound(c *echo.Context, message string, err error) error {
	errorMsg := "Resource not found"
	if err != nil {
		errorMsg = err.Error()
	}

	log.Warn("not found",
		"message", message,
		"error", errorMsg,
	)

	return c.JSON(http.StatusNotFound, APIResponse{
		Success: false,
		Message: message,
		Error:   errorMsg,
	})
}

// InternalServerError sends an internal server error response.
// The raw error is logged server-side only; the client receives a generic message
// to avoid leaking internal details (DSN, stack traces, etc.).
func InternalServerError(c *echo.Context, message string, err error) error {
	log.Error("internal server error",
		"message", message,
		"error", err,
	)

	return c.JSON(http.StatusInternalServerError, APIResponse{
		Success: false,
		Message: message,
		// Do NOT include err.Error() in the response — avoids leaking internal details.
	})
}

// ValidationError sends a validation error response
func ValidationError(c *echo.Context, message string, err error) error {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	log.Warn("validation error",
		"message", message,
		"error", errorMsg,
	)

	return c.JSON(http.StatusUnprocessableEntity, APIResponse{
		Success: false,
		Message: message,
		Error:   errorMsg,
	})
}

// FromValidateError maps Echo validation errors to a unified response:
// structured field errors use 422 with Errors populated; otherwise ValidationError fallback.
func FromValidateError(c *echo.Context, err error) error {
	if errs, ok := errors.AsType[validator.ValidationErrors](err); ok {
		log.Warn("validation error",
			"error", errs.Error(),
		)
		return c.JSON(http.StatusUnprocessableEntity, APIResponse{
			Success: false,
			Message: "Validation failed",
			Error:   errs.Error(),
			Errors:  errs.Errors,
		})
	}
	return ValidationError(c, "Validation failed", err)
}

// SpamRejected sends a 422 for content rejected by the anti-spam guard. The
// reason sits in Errors tagged "spam" so clients can show it next to field;
// axumbackend's AppError::spam_rejected produces the same envelope.
func SpamRejected(c *echo.Context, message, field, reason string) error {
	log.Warn("spam rejected",
		"message", message,
		"reason", reason,
	)

	return c.JSON(http.StatusUnprocessableEntity, APIResponse{
		Success: false,
		Message: "Validation failed",
		Error:   message,
		Errors: []validator.ValidationError{
			{Field: field, Message: reason, Tag: "spam"},
		},
	})
}

// Conflict sends a 409 Conflict response (e.g. duplicate resource).
func Conflict(c *echo.Context, message string, conflictError string) error {
	log.Warn("conflict",
		"message", message,
	)
	return c.JSON(http.StatusConflict, APIResponse{
		Success: false,
		Message: message,
		Error:   conflictError,
	})
}

// CalculatePaginationMeta calculates pagination metadata.
// Guards against division-by-zero when limit is 0.
func CalculatePaginationMeta(totalItems int64, offset, limit int) PaginationMeta {
	if limit <= 0 {
		limit = 10
	}

	total := int(totalItems)
	totalPages := total / limit
	if total%limit > 0 {
		totalPages++
	}

	return PaginationMeta{
		TotalItems: total,
		Offset:     offset,
		Limit:      limit,
		TotalPages: totalPages,
	}
}

// Framework-level messages. These responses come from the router and global
// middleware rather than from a handler, and must stay byte-identical across
// echobackend, fiberbackend and axumbackend.
const (
	// MsgRouteNotFound is the message for a request that matched no route (404).
	MsgRouteNotFound = "Route not found"
	// MsgMethodNotAllowed is the message for a known path hit with an unsupported method (405).
	MsgMethodNotAllowed = "Method not allowed"
	// MsgPayloadTooLarge is the message for a body over the global (10 MB) or a per-route limit (413).
	MsgPayloadTooLarge = "Request Entity Too Large"
	// MsgTooManyAttempts is the message for every 429 (global and per-route limiters).
	MsgTooManyAttempts = "Too many attempts. Please try again later."
	// MsgInternalServerError is the message for unhandled errors and recovered panics (500).
	MsgInternalServerError = "Internal server error"
)

// RouteNotFound sends the 404 envelope for a request that matched no route.
func RouteNotFound(c *echo.Context) error {
	return c.JSON(http.StatusNotFound, APIResponse{
		Success: false,
		Message: MsgRouteNotFound,
		Error:   "Resource not found",
	})
}

// MethodNotAllowed sends the 405 envelope for a path that exists under other methods.
func MethodNotAllowed(c *echo.Context) error {
	return c.JSON(http.StatusMethodNotAllowed, APIResponse{
		Success: false,
		Message: MsgMethodNotAllowed,
		Error:   "Method not allowed",
	})
}

// PayloadTooLarge sends the 413 envelope for a request body over the size limit.
func PayloadTooLarge(c *echo.Context) error {
	return c.JSON(http.StatusRequestEntityTooLarge, APIResponse{
		Success: false,
		Message: MsgPayloadTooLarge,
		Error:   "Request body too large",
	})
}

// PayloadTooLargeWithMessage sends a 413 for an upload over a route-specific
// cap (e.g. "Avatar is too large (max 5 MB)"); message is also the error.
func PayloadTooLargeWithMessage(c *echo.Context, message string) error {
	log.Warn("payload too large",
		"message", message,
	)

	return c.JSON(http.StatusRequestEntityTooLarge, APIResponse{
		Success: false,
		Message: message,
		Error:   message,
	})
}

// HTTPStatus sends the envelope for an error that reached the global error
// handler with a status code but no handler-written body (router, framework or
// middleware errors). 404/405/413/429 use their dedicated envelopes; 5xx (and
// errors without a usable code, reported as 500) carry only a generic message
// so internal details never leak; other 4xx codes keep message (or the status
// text when empty) with the status text as error.
func HTTPStatus(c *echo.Context, code int, message string) error {
	switch {
	case code == http.StatusNotFound:
		return RouteNotFound(c)
	case code == http.StatusMethodNotAllowed:
		return MethodNotAllowed(c)
	case code == http.StatusRequestEntityTooLarge:
		return PayloadTooLarge(c)
	case code == http.StatusTooManyRequests:
		return TooManyRequests(c, MsgTooManyAttempts)
	case code < http.StatusBadRequest || code == http.StatusInternalServerError:
		return c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Message: MsgInternalServerError,
		})
	case code > http.StatusInternalServerError:
		return c.JSON(code, APIResponse{
			Success: false,
			Message: http.StatusText(code),
		})
	}

	statusText := http.StatusText(code)
	if message == "" {
		message = statusText
	}
	return c.JSON(code, APIResponse{
		Success: false,
		Message: message,
		Error:   statusText,
	})
}

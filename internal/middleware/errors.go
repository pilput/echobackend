package middleware

import (
	"errors"
	"net/http"

	"echobackend/pkg/response"

	"github.com/labstack/echo/v5"
)

// HTTPErrorHandler is the app-wide Echo error handler. It replaces Echo's
// default {"message": ...} body with the shared response envelope so
// router-level errors (unknown route 404, wrong method 405), body-limit 413s,
// recovered panics and any other error a handler returns without writing a
// response all look like every other API error. fiberbackend's ErrorHandler
// and axumbackend's router fallbacks produce the same bodies.
func HTTPErrorHandler(c *echo.Context, err error) {
	if resp, _ := echo.UnwrapResponse(c.Response()); resp != nil && resp.Committed {
		return
	}

	code := echo.StatusCode(err)
	if code == 0 {
		code = http.StatusInternalServerError
	}
	message := ""
	if he, ok := errors.AsType[*echo.HTTPError](err); ok {
		message = he.Message
	}

	if code >= http.StatusInternalServerError {
		log.Error("unhandled request error",
			"method", c.Request().Method,
			"uri", c.Request().RequestURI,
			"status", code,
			"error", err,
		)
	}

	var sendErr error
	if c.Request().Method == http.MethodHead {
		sendErr = c.NoContent(code)
	} else {
		sendErr = response.HTTPStatus(c, code, message)
	}
	if sendErr != nil {
		log.Error("failed to send error response", "error", sendErr)
	}
}

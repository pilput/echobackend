package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"

	"github.com/labstack/echo/v5"
)

// RecoverWithLog returns middleware that recovers panics and logs them with slog.
func RecoverWithLog() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					if rErr, ok := r.(error); ok && errors.Is(rErr, http.ErrAbortHandler) {
						panic(r)
					}

					tmpErr, ok := r.(error)
					if !ok {
						tmpErr = fmt.Errorf("%v", r)
					}

					stack := make([]byte, 4<<10)
					length := runtime.Stack(stack, false)
					log.Error("panic recovered",
						"error", tmpErr,
						"method", c.Request().Method,
						"uri", c.Request().RequestURI,
						"stack", string(stack[:length]),
					)
					err = echo.NewHTTPError(http.StatusInternalServerError, "Internal server error")
				}
			}()
			return next(c)
		}
	}
}

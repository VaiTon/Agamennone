package agamennone

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// logs the error and returns a 500 Internal Server Error
func internalError(c echo.Context, err error) error {
	_ = c.String(http.StatusInternalServerError, "Oops! Something went wrong")
	return fmt.Errorf("internal server error: %v", err)
}

func loggingMiddleware(logger *slog.Logger) func(echo.HandlerFunc) echo.HandlerFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			timeTaken := time.Since(start)

			logger.Debug("got request",
				"ip", c.RealIP(),
				"method", c.Request().Method,
				"path", c.Path(),
				"proto", c.Request().Proto,
				"status", c.Response().Status,
				"duration", timeTaken,
			)

			return err
		}
	}
}

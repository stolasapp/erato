// Package app contains the web front-end.
package app

import (
	"embed"
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
	"github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1/eratov1connect"
	"github.com/stolasapp/erato/internal/sec"
	"github.com/stolasapp/erato/internal/storage"
)

//go:embed static
var staticFiles embed.FS

// New creates a web front-end server.
func New(
	cfg *eratov1.Config,
	logger *slog.Logger,
	users storage.Users,
	archive eratov1connect.ArchiveServiceHandler,
) *echo.Echo {
	srv := echo.New()

	srv.HideBanner = true
	srv.HidePort = true
	srv.Logger.SetLevel(log.OFF)

	if cfg.GetDevMode() {
		srv.Debug = true
		srv.Use(logRequests(logger))
	} else {
		srv.Use(
			middleware.Recover(),
			middleware.BasicAuth(func(_, _ string, c echo.Context) (bool, error) {
				ctx := c.Request().Context()
				usr, err := sec.Authenticate(ctx, c.Request(), users)
				if err != nil {
					return false, err
				}
				ctx = sec.SetAuthenticatedUser(ctx, usr)
				c.SetRequest(c.Request().WithContext(ctx))
				return true, nil
			}),
		)
	}

	srv.Use(
		middleware.Decompress(),
		middleware.Gzip(),
		middleware.Secure(),
		middleware.CSRFWithConfig(middleware.CSRFConfig{
			TokenLookup: "cookie:" + middleware.DefaultCSRFConfig.CookieName,
		}),
		middleware.RequestID(),
	)

	handler{handler: archive}.register(srv)
	staticFS := echo.MustSubFS(staticFiles, "static")
	srv.StaticFS("/static/", staticFS)
	srv.FileFS("/robots.txt", "robots.txt", staticFS)
	return srv
}

func logRequests(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			latency := time.Since(start)

			if err != nil {
				c.Error(err)
			}

			req := c.Request()
			res := c.Response()

			attrs := []slog.Attr{
				slog.String("method", req.Method),
				slog.String("uri", req.RequestURI),
				slog.String("route", c.Path()),
				slog.Duration("latency", latency),
				slog.Int("status", res.Status),
			}
			if err != nil {
				attrs = append(attrs, slog.Any("error", err))
			}
			logger.LogAttrs(
				req.Context(),
				slog.LevelDebug,
				"request handled",
				attrs...,
			)
			return err
		}
	}
}

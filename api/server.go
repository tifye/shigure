package api

import (
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
	"github.com/tifye/shigure/activity"
	"golang.org/x/time/rate"
)

type ServerDependencies struct {
	ActivityClient *activity.Client
}

func NewServer(logger *log.Logger, config *viper.Viper, deps *ServerDependencies) *http.Server {
	e := echo.New()
	server := &http.Server{
		Handler:           e,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       25 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
		ErrorLog:          logger.StandardLog(),
		MaxHeaderBytes:    1024,
	}

	rlimiterConfig := middleware.RateLimiterConfig{
		Skipper: middleware.DefaultSkipper,
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(5),
				Burst:     5,
				ExpiresIn: 3 * time.Minute,
			},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			return ctx.RealIP(), nil
		},
		DenyHandler: func(ctx echo.Context, identifier string, err error) error {
			logger.Warn("ratelimiting", "identifier", identifier, "err", err)
			return ctx.NoContent(http.StatusTooManyRequests)
		},
	}
	e.Use(middleware.RateLimiterWithConfig(rlimiterConfig))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:5173", "https://shigure.joshuadematas.me", "http://192.168.18.192:5173"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderXRequestedWith, echo.HeaderAuthorization},
	}))

	registerRoutes(e, logger, config, deps)

	return server
}

package api

import (
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
	"github.com/tifye/shigure/activity"
)

type ServerDependencies struct {
	ActivityClient *activity.Client
}

func NewServer(logger *log.Logger, deps *ServerDependencies) *http.Server {
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

	registerRoutes(e, logger, deps)

	return server
}

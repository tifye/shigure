package api

import (
	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
)

func registerRoutes(e *echo.Echo, logger *log.Logger, config *viper.Viper, deps *ServerDependencies) {
	auth := e.Group("", requireAuthMiddleware(logger, config))

	e.GET("/activity", handleGetActivity(deps.ActivityClient))
	e.GET("/activity/svg", handleGetSVG(logger, deps.ActivityClient))
	auth.POST("/activity/clear", handlePostClearActivity(deps.ActivityClient))
	auth.POST("/activity/youtube/:videoId", handlePostYoutubeActivity(logger, deps.ActivityClient))

	e.GET("/auth/key", handleGetToken(logger, config))
}

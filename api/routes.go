package api

import (
	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
)

func registerRoutes(e *echo.Echo, logger *log.Logger, deps *ServerDependencies) {
	e.GET("/activity", handleGetActivity(deps.ActivityClient))
	e.POST("/activity/clear", handlePostClearActivity(deps.ActivityClient))
	e.POST("/activity/youtube/:videoId", handlePostYoutubeActivity(logger, deps.ActivityClient))
	e.GET("/activity/svg", handleGetSVG(logger, deps.ActivityClient))
}

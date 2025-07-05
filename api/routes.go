package api

import (
	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
)

func registerRoutes(e *echo.Echo, logger *log.Logger, config *viper.Viper, deps *ServerDependencies) {
	auth := e.Group("", requireAuthMiddleware(logger, config))

	e.GET("/activity", handleGetYoutubeActivity(deps.ActivityClient))
	e.GET("/activity/svg", handleGetYoutubeActivitySVG(logger, deps.ActivityClient))
	e.GET("/youtube/activity/svg", handleGetYoutubeActivitySVG(logger, deps.ActivityClient)) // legacy
	auth.POST("/activity/clear", handlePostClearYoutubeActivity(deps.ActivityClient))
	auth.POST("/activity/youtube/:videoId", handlePostYoutubeActivity(logger, deps.ActivityClient))

	e.GET("/activity/vscode", handleGetVSCodeActivity(logger, deps.VSCodeActivityClient))
	e.GET("/activity/vscode/ws", handleGetVSCodeActivityWS(logger, deps.VSCodeActivityClient))
	auth.POST("/activity/vscode", handlePostVSCodeActivity(logger, deps.VSCodeActivityClient))

	e.GET("/auth/token", handleGetToken(logger, config))
	auth.GET("/auth/token/generate", handleGetGenerateToken(logger, config))
	e.POST("/auth/token/verify", handlePostVerifyToken(logger, config))

	e.GET("/personalsite/room", handlePersonalSiteRoom(logger, deps.RoomHub))
}

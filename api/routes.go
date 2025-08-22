package api

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
)

func registerRoutes(e *echo.Echo, logger *log.Logger, config *viper.Viper, deps *ServerDependencies) {
	e.GET("/", hello)
	e.HEAD("/health", hello)

	e.GET("/activity", handleGetYoutubeActivity(deps.YoutubeActivityClient))
	e.GET("/activity/svg", handleGetYoutubeActivitySVG(logger, deps.YoutubeActivityClient))
	e.GET("/youtube/activity/svg", handleGetYoutubeActivitySVG(logger, deps.YoutubeActivityClient)) // legacy
	e.POST("/activity/clear", handlePostClearYoutubeActivity(deps.YoutubeActivityClient), requireAuthMiddleware(logger, config))
	e.POST("/activity/youtube/:videoId", handlePostYoutubeActivity(logger, deps.YoutubeActivityClient), requireAuthMiddleware(logger, config))

	e.GET("/activity/vscode", handleGetVSCodeActivity(logger, deps.CodeActivityClient))
	e.POST("/activity/vscode", handlePostVSCodeActivity(logger, deps.CodeActivityClient), requireAuthMiddleware(logger, config))

	e.GET("/auth/token", handleGetToken(logger, config))
	e.GET("/auth/token/generate", handleGetGenerateToken(logger, config), requireAuthMiddleware(logger, config))
	e.POST("/auth/token/verify", handlePostVerifyToken(logger, config))

	e.GET("/stats/code", handleGetCodeStats(logger, deps.CodeActivityClient))

	e.GET("/ws", handleWebsocketConn(logger, deps.WebSocketMux, deps.NewSessionCookie))
}

func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Nyaa~")
}

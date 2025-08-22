package api

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
	"github.com/tifye/shigure/activity/code"
	"github.com/tifye/shigure/assert"
)

func handleGetCodeStats(logger *log.Logger, client *code.ActivityClient) echo.HandlerFunc {
	assert.AssertNotNil(logger)
	assert.AssertNotNil(client)
	return func(c echo.Context) error {
		stats, err := client.CodeStats(c.Request().Context())
		if err != nil {
			logger.Error("code stats", "err", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		return c.JSON(http.StatusOK, stats)
	}
}

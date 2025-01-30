package api

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
	"github.com/tifye/shigure/activity"
)

func handlePostYoutubeActivity(logger *log.Logger, ac *activity.Client) echo.HandlerFunc {
	type request struct {
		VideoId string `param:"videoId"`
	}
	return func(c echo.Context) error {
		var req request
		if err := c.Bind(&req); err != nil {
			return err
		}

		err := ac.SetYoutubeActivity(c.Request().Context(), req.VideoId)
		if err != nil {
			logger.Error("failed to fetch youtube video", "videoId", req.VideoId, "err", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		return c.JSON(http.StatusOK, ac.Activity())
	}
}

func handleGetActivity(ac *activity.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, ac.Activity())
	}
}

func handleGetSVG(logger *log.Logger, ac *activity.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set(echo.HeaderContentType, "image/svg+xml")
		c.Response().WriteHeader(http.StatusOK)
		c.Response().Header().Add("Cache-Control", "no-cache")
		err := ac.StreamSVG(c.Request().Context(), c.Response())
		if err != nil {
			logger.Errorf("Get SVG: %s", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		return nil
	}
}

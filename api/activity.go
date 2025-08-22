package api

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
	"github.com/tifye/shigure/activity/code"
	"github.com/tifye/shigure/activity/youtube"
)

func handlePostYoutubeActivity(logger *log.Logger, ac *youtube.ActivityClient) echo.HandlerFunc {
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

func handleGetYoutubeActivity(ac *youtube.ActivityClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, ac.Activity())
	}
}

func handleGetYoutubeActivitySVG(logger *log.Logger, ac *youtube.ActivityClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set(echo.HeaderContentType, "image/svg+xml")
		c.Response().Header().Add("Cache-Control", "no-cache")
		c.Response().WriteHeader(http.StatusOK)
		err := ac.StreamSVG(c.Request().Context(), c.Response())
		if err != nil {
			logger.Errorf("Get SVG: %s", err)
			return c.NoContent(http.StatusInternalServerError)
		}

		return nil
	}
}

func handlePostClearYoutubeActivity(ac *youtube.ActivityClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		ac.ClearActivity()
		return c.JSON(http.StatusOK, ac.Activity())
	}
}

func handlePostVSCodeActivity(_ *log.Logger, ac *code.ActivityClient) echo.HandlerFunc {
	type request code.VSCodeActivity
	return func(c echo.Context) error {
		var req request
		if err := c.Bind(&req); err != nil {
			return err
		}

		ac.SetActivity(c.Request().Context(), code.VSCodeActivity(req))
		return c.NoContent(http.StatusOK)
	}
}

func handleGetVSCodeActivity(logger *log.Logger, ac *code.ActivityClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger.Debug("get vscode activity")
		return c.JSON(http.StatusOK, ac.Activity())
	}
}

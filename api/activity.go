package api

import (
	"encoding/json"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
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

func handleGetYoutubeActivity(ac *activity.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, ac.Activity())
	}
}

func handleGetYoutubeActivitySVG(logger *log.Logger, ac *activity.Client) echo.HandlerFunc {
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

func handlePostClearYoutubeActivity(ac *activity.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		ac.ClearActivity()
		return c.JSON(http.StatusOK, ac.Activity())
	}
}

func handlePostVSCodeActivity(logger *log.Logger, ac *activity.VSCodeActivityClient) echo.HandlerFunc {
	type request activity.VSCodeActivity
	return func(c echo.Context) error {
		var req request
		if err := c.Bind(&req); err != nil {
			return err
		}

		logger.Debug("updating vscode activity", "activity", req)
		ac.SetActivity(activity.VSCodeActivity(req))

		return c.NoContent(http.StatusOK)
	}
}

func handleGetVSCodeActivity(logger *log.Logger, ac *activity.VSCodeActivityClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger.Debug("get vscode activity")
		return c.JSON(http.StatusOK, ac.Activity())
	}
}

func handleGetVSCodeActivityWS(logger *log.Logger, ac *activity.VSCodeActivityClient) echo.HandlerFunc {
	return func(c echo.Context) error {
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			logger.Error(err)
			return err
		}
		defer conn.Close()

		acch := make(chan activity.VSCodeActivity, 1)
		ac.Subscribe(acch)
		defer ac.Unsubscribe(acch)

		logger.Info("VSC activity client connected")

		for activity := range acch {
			bytes, err := json.Marshal(activity)
			if err != nil {
				logger.Error("json marshal vscode activity", "err", err)
				break
			}

			err = conn.WriteMessage(websocket.TextMessage, bytes)
			if err != nil {
				logger.Error("Write to websocket", "err", err)
				break
			}
		}

		return c.NoContent(http.StatusOK)
	}
}

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/tifye/shigure/personalsite"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type PositionData struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func handlePersonalSiteRoom(logger *log.Logger, room *personalsite.RoomHub) echo.HandlerFunc {
	type request struct {
		IsMe bool `query:"isme"`
	}
	return func(c echo.Context) error {
		var req request
		_ = c.Bind(&req)

		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			logger.Error(err)
			return err
		}
		defer conn.Close()

		wr := make(chan []byte)
		defer close(wr)
		user := &personalsite.RoomUser{
			ID:        room.NextID(),
			WriteChan: wr,
		}
		room.Register(user)
		defer room.Unregister(user)

		logger.Info("Client connected", "isme", req.IsMe)

		if !req.IsMe {
			if err = notifyDiscord(c.Request().Context()); err != nil {
				logger.Error("Failed to send Discord notification", "err", err, "isme", req.IsMe)
			}
		}

		go func() {
			for msg := range user.WriteChan {
				err := conn.WriteMessage(websocket.TextMessage, msg)
				if err != nil {
					logger.Error("Write to websocket", "id", user.ID, "err", err)
				}
			}
		}()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				logger.Error("ws read", "err", err)
				break
			}

			room.UserMessage(user, msg)
		}

		return c.NoContent(http.StatusOK)
	}
}

const discordWebhookURL = "https://discord.com/api/webhooks/1386429934603468901/q9lyRkp0812_HbHrG1ZjRNIghZILETgxVtP5yEvUEv7CugBgTLafsrkNqANP3rxSZDf9"

type webhookBody struct {
	Content string `json:"content"`
}

func notifyDiscord(ctx context.Context) error {
	body := webhookBody{Content: "Someone joined the room https://www.joshuadematas.me"}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marhsall body: %s", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", discordWebhookURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook res: %s", err)
	}
	res.Body.Close()
	return nil
}

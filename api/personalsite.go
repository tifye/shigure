package api

import (
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
	return func(c echo.Context) error {
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

		logger.Info("client connected")

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

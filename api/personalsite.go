package api

import (
	"fmt"
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func handlePersonalSiteRoom(logger *log.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			logger.Error(err)
			return err
		}
		defer conn.Close()

		logger.Info("client connected")

		for {
			// use ws.Read() to control bytes read
			_, msg, err := conn.ReadMessage()
			if err != nil {
				logger.Error("ws read", "err", err)
				break
			}
			fmt.Printf("%s\n", msg)
		}

		return c.NoContent(http.StatusOK)
	}
}

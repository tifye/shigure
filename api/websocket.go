package api

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/tifye/shigure/assert"
	"github.com/tifye/shigure/stream"
)

func handleWebsocketConn(logger *log.Logger, mux *stream.Mux) echo.HandlerFunc {
	assert.AssertNotNil(logger)
	assert.AssertNotNil(mux)

	return func(c echo.Context) error {
		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			logger.Error(err)
			return err
		}
		defer conn.Close()

		id := mux.Connect(func(id stream.ID, data []byte) {
			err := conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				logger.Error("write to websocket", "id", id, "err", err)
			}
		})
		defer func() {
			if err := mux.Disconnect(id); err != nil {
				logger.Error("websocket mux disconnect", "err", err, "id", id)
			}
		}()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				logger.Error("ws read", "err", err, "id", id)
				break
			}

			logger.Debug("ws message", "messge", string(msg))

			if err = mux.UserMessage(id, msg); err != nil {
				logger.Errorf("mux user message: %s", err)
				break
			}
		}

		return c.NoContent(http.StatusOK)
	}
}

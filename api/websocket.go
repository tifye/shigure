package api

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/tifye/shigure/assert"
	"github.com/tifye/shigure/stream"
)

const (
	streamIDSessionKey = "streamID"
)

func handleWebsocketConn(
	logger *log.Logger,
	mux *stream.Mux,
	newSessionCookie func(s *sessions.Session) (*http.Cookie, error),
) echo.HandlerFunc {
	assert.AssertNotNil(logger)
	assert.AssertNotNil(mux)

	return func(c echo.Context) error {
		var id stream.ID

		session, err := session.Get("session", c)
		if err != nil {
			logger.Error("get session", "err", err)
		} else {
			streamID, exists := session.Values[streamIDSessionKey]
			if exists {
				id, _ = streamID.(stream.ID)
			}
		}

		// trigger save to ensure session has an ID
		if err := session.Save(c.Request(), c.Response()); err != nil {
			logger.Error("save session for ID", "err", err)
		}

		responseHeader := http.Header{}
		sessionCookie, err := newSessionCookie(session)
		if err != nil {
			logger.Error("new session cookie", "err", err)
		} else {
			assert.AssertNotNil(sessionCookie)
			responseHeader.Add("Set-Cookie", sessionCookie.String())
		}

		conn, err := upgrader.Upgrade(c.Response(), c.Request(), responseHeader)
		if err != nil {
			logger.Error(err)
			return err
		}
		defer conn.Close()

		writeFunc := func(id stream.ID, data []byte) {
			err := conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				logger.Error("write to websocket", "id", id, "err", err)
			}
		}

		if id == 0 {
			id = mux.Connect(writeFunc)
		} else {
			if err := mux.Reconnect(id, writeFunc); err != nil {
				logger.Error("reconnect stream", "err", err, "id", id)
				id = mux.Connect(writeFunc)
			}
		}
		defer func() {
			if err := mux.Disconnect(id); err != nil {
				logger.Error("websocket mux disconnect", "err", err, "id", id)
			}
		}()

		if session != nil {
			session.Values[streamIDSessionKey] = id
			if err := session.Save(c.Request(), c.Response()); err != nil {
				logger.Error("save session", "err", err)
			}
			logger.Debug("session saved", "sessionName", session.ID)
		}

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				logger.Debug("ws read", "err", err, "id", id)
				break
			}

			if err = mux.UserMessage(id, msg); err != nil {
				logger.Errorf("mux user message: %s", err)
				break
			}
		}

		return c.NoContent(http.StatusOK)
	}
}

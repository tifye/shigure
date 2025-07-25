package api

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/tifye/shigure/assert"
	"github.com/tifye/shigure/mux"
)

func handleWebsocketConn(
	logger *log.Logger,
	mx *mux.Mux,
	newSessionCookie func(s *sessions.Session) (*http.Cookie, error),
) echo.HandlerFunc {
	assert.AssertNotNil(logger)
	assert.AssertNotNil(mx)

	return func(c echo.Context) error {
		session, err := session.Get("session", c)
		if err != nil {
			logger.Error("get session", "err", err)
		}

		// trigger save to ensure session has an ID
		if err := session.Save(c.Request(), c.Response()); err != nil {
			logger.Error("save session for ID", "err", err)
		}

		var sessionID mux.ID
		copy(sessionID[:], []byte(session.ID))

		responseHeader := http.Header{}
		sessionCookie, err := newSessionCookie(session)
		if err != nil {
			logger.Error("new session cookie", "err", err)
		} else {
			assert.AssertNotNil(sessionCookie)
			responseHeader.Add("Set-Cookie", sessionCookie.String())
		}

		logger.Debug("upgrading to websocket connection")

		conn, err := upgrader.Upgrade(c.Response(), c.Request(), responseHeader)
		if err != nil {
			logger.Error(err)
			return err
		}
		defer conn.Close()

		channelID := mx.Connect(sessionID, WriterFunc(func(data []byte) (n int, err error) {
			return len(data), conn.WriteMessage(websocket.TextMessage, data)
		}))
		defer mx.Disconnect(sessionID, channelID)

		logger.Debug("channel connected", "channelID", channelID, "sessionID", sessionID)

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				logger.Debug("ws read", "err", err, "id", sessionID)
				break
			}

			if err = mx.Message(sessionID, channelID, msg); err != nil {
				logger.Errorf("mux user message: %s", err)
				break
			}
		}

		return c.NoContent(http.StatusOK)
	}
}

type WriterFunc func(data []byte) (n int, err error)

func (f WriterFunc) Write(data []byte) (n int, err error) {
	return f(data)
}

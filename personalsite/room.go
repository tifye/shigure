package personalsite

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/tifye/shigure/assert"
	"github.com/tifye/shigure/mux"
)

type RoomHub struct {
	logger         *log.Logger
	mux            *mux.Mux
	muxMessageType string

	// Discord webhook URL to notify
	// users joining
	webhookURL string

	// Keeps track of which users
	// we have already notified about
	userNotifs map[mux.ID]struct{}
	notifMu    sync.RWMutex
}

func NewRoomHubV2(
	logger *log.Logger,
	mx *mux.Mux,
	messageType string,
	webhookURL string,
) *RoomHub {
	assert.AssertNotNil(logger)
	assert.AssertNotNil(mx)
	assert.AssertNotEmpty(webhookURL)
	return &RoomHub{
		logger:         logger,
		mux:            mx,
		muxMessageType: messageType,
		webhookURL:     webhookURL,
		userNotifs:     map[mux.ID]struct{}{},
	}
}

func (r *RoomHub) MessageType() string {
	return r.muxMessageType
}

func (r *RoomHub) HandleMessage(c *mux.Channel, msg []byte) error {
	var pdata userPositionData
	if err := json.Unmarshal(msg, &pdata); err != nil {
		r.logger.Debug(err)
		return fmt.Errorf("json unmarshal: %s", err)
	}

	id := c.ID()

	if pdata.Unreg {
		return r.broadcastDisconnect(id)
	}

	pdata.ID = id[:]
	msgb, err := json.Marshal(pdata)
	if err != nil {
		return fmt.Errorf("json marshal: %s", err)
	}

	r.notifMu.RLock()
	_, didNotify := r.userNotifs[id]
	r.notifMu.RUnlock()

	if !didNotify {
		r.notifMu.Lock()
		r.userNotifs[id] = struct{}{}
		r.notifMu.Unlock()

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			notifyDiscord(ctx, r.webhookURL)
		}()
	}

	return r.mux.Broadcast(r.muxMessageType, msgb, func(ch *mux.Channel) bool {
		return id == ch.ID()
	})
}

func (r *RoomHub) HandleDisconnect(c *mux.Channel, _ bool) {
	err := r.broadcastDisconnect(c.ID())
	if err != nil {
		r.logger.Error("broadcast disconnect", "type", r.muxMessageType, "id", c.ID())
	}
}

func (r *RoomHub) broadcastDisconnect(id mux.ID) error {
	msg := userUnregistered{
		ID:    id[:],
		Unreg: true,
	}

	msgb, err := json.Marshal(msg)
	if err != nil {
		r.logger.Error("ungregister json marshal", "err", err, "id", id)
	}

	return r.mux.Broadcast(r.muxMessageType, msgb, func(c *mux.Channel) bool {
		return c.ID() == id
	})
}

type positionData struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type userPositionData struct {
	ID    []byte `json:"id"`
	Unreg bool   `json:"delete,omitzero"`
	positionData
}

type userUnregistered struct {
	ID    []byte `json:"id"`
	Unreg bool   `json:"delete,omitzero"`
}

type webhookBody struct {
	Content string `json:"content"`
}

func notifyDiscord(ctx context.Context, webhookURL string) error {
	body := webhookBody{Content: "Someone joined the room https://www.joshuadematas.me"}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marhsall body: %s", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(bodyBytes))
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

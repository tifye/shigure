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
	"github.com/tifye/shigure/stream"
)

type RoomHub struct {
	logger         *log.Logger
	mux            *stream.Mux
	muxMessageType string

	// Discord webhook URL to notify
	// users joining
	webhookURL string

	// Keeps track of which users
	// we have already notified about
	userNotifs map[stream.ID]struct{}
	notifMu    sync.RWMutex
}

func NewRoomHubV2(logger *log.Logger, mux *stream.Mux, webhookURL string) *RoomHub {
	assert.AssertNotNil(logger)
	assert.AssertNotNil(mux)
	assert.AssertNotEmpty(webhookURL)
	return &RoomHub{
		logger:         logger,
		mux:            mux,
		muxMessageType: "room",
		webhookURL:     webhookURL,
		userNotifs:     map[stream.ID]struct{}{},
	}
}

func (r *RoomHub) MessageType() string {
	return r.muxMessageType
}

func (r *RoomHub) HandleMessage(id stream.ID, msg []byte) error {
	var pdata userPositionData
	if err := json.Unmarshal(msg, &pdata); err != nil {
		return fmt.Errorf("json unmarshal: %s", err)
	}

	if pdata.Unreg {
		return r.broadcastDisconnect(id)
	}

	pdata.ID = id
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

	return r.mux.Broadcast(r.muxMessageType, msgb, func(uid stream.ID) bool {
		return id != uid
	})
}

func (r *RoomHub) HandleDisconnect(id stream.ID) {
	err := r.broadcastDisconnect(id)
	if err != nil {
		r.logger.Error("broadcast disconnect", "type", r.muxMessageType, "id", id)
	}
}

func (r *RoomHub) broadcastDisconnect(id stream.ID) error {
	msg := userUnregistered{
		ID:    id,
		Unreg: true,
	}

	msgb, err := json.Marshal(msg)
	if err != nil {
		r.logger.Error("ungregister json marshal", "err", err, "id", id)
	}

	return r.mux.Broadcast(r.muxMessageType, msgb, filterUser(id))

}

func filterUser(id stream.ID) func(stream.ID) bool {
	return func(i stream.ID) bool {
		return id != i
	}
}

type positionData struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type userPositionData struct {
	ID    uint32 `json:"id"`
	Unreg bool   `json:"delete,omitzero"`
	positionData
}

type userUnregistered struct {
	ID    uint32 `json:"id"`
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

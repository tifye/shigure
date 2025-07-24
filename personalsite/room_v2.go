package personalsite

import (
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/tifye/shigure/assert"
	"github.com/tifye/shigure/stream"
)

type RoomHubV2 struct {
	logger *log.Logger
	mux    *stream.Mux
}

func NewRoomHubV2(logger *log.Logger, mux *stream.Mux) *RoomHubV2 {
	assert.AssertNotNil(logger)
	assert.AssertNotNil(mux)
	return &RoomHubV2{
		logger: logger,
		mux:    mux,
	}
}

func (r *RoomHubV2) HandleMessage(id stream.ID, msg []byte) error {
	var pdata userPositionData
	if err := json.Unmarshal(msg, &pdata); err != nil {
		return fmt.Errorf("json unmarshal: %s", err)
	}

	pdata.ID = id
	bmsg, err := json.Marshal(pdata)
	if err != nil {
		return fmt.Errorf("json marshal: %s", err)
	}

	return r.mux.Broadcast("room", bmsg, func(uid stream.ID) bool {
		return id != uid
	})
}

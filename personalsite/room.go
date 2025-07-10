package personalsite

import (
	"encoding/json"
	"fmt"
	"sync/atomic"

	"github.com/charmbracelet/log"
)

type RoomUser struct {
	ID        uint32
	WriteChan chan []byte
}

type RoomHub struct {
	logger    *log.Logger
	idCounter atomic.Uint32
	users     map[*RoomUser]struct{}
	broadcast chan broadcastMessage
	regch     chan *RoomUser
	unregch   chan *RoomUser
}

func NewRoomHub(logger *log.Logger) *RoomHub {
	return &RoomHub{
		logger:    logger,
		users:     map[*RoomUser]struct{}{},
		broadcast: make(chan broadcastMessage, 1),
		regch:     make(chan *RoomUser),
		unregch:   make(chan *RoomUser),
	}
}

func (r *RoomHub) Run() {
	for {
		select {
		case u := <-r.regch:
			r.users[u] = struct{}{}
			r.logger.Info("Registered user", "id", u.ID)
		case u := <-r.unregch:
			delete(r.users, u)
			r.logger.Info("Unregistered user", "id", u.ID)
		case msg := <-r.broadcast:
			for u := range r.users {
				if u == msg.user {
					continue
				}
				u.WriteChan <- msg.msg
			}
		}
	}
}

func (r *RoomHub) NextID() uint32 {
	return r.idCounter.Add(1)
}

func (r *RoomHub) Register(u *RoomUser) {
	r.regch <- u
}

func (r *RoomHub) Unregister(u *RoomUser) {
	r.unregch <- u
	msg, err := json.Marshal(userUnregistered{
		ID:    u.ID,
		Unreg: true,
	})
	if err != nil {
		r.logger.Error("Failed to send unreg message", "id", u.ID, "err", err)
		return
	}
	r.broadcast <- broadcastMessage{user: u, msg: msg}
}

type positionData struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type userPositionData struct {
	ID uint32 `json:"id"`
	positionData
}

type userUnregistered struct {
	ID    uint32 `json:"id"`
	Unreg bool   `json:"delete,omitzero"`
}

type broadcastMessage struct {
	user *RoomUser
	msg  []byte
}

func (r *RoomHub) UserMessage(u *RoomUser, msg []byte) error {
	var pdata userPositionData
	if err := json.Unmarshal(msg, &pdata); err != nil {
		return fmt.Errorf("json unmarshal: %s", err)
	}

	pdata.ID = u.ID
	bmsg, err := json.Marshal(pdata)
	if err != nil {
		return fmt.Errorf("json marshal: %s", err)
	}

	r.broadcast <- broadcastMessage{
		user: u,
		msg:  bmsg,
	}

	return nil
}

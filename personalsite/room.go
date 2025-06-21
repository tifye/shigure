package personalsite

import (
	"encoding/json"
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
			r.logger.Debug("Broadcasting")
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
}

type positionData struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type userPositionData struct {
	ID uint32 `json:"id"`
	positionData
}

type broadcastMessage struct {
	user *RoomUser
	msg  []byte
}

func (r *RoomHub) UserMessage(u *RoomUser, msg []byte) {
	r.logger.Debug("User message", "id", u.ID, "msg", string(msg))

	var pdata userPositionData
	if err := json.Unmarshal(msg, &pdata); err != nil {
		r.logger.Error("json unmarshal", "err", err)
		return
	}

	pdata.ID = u.ID
	bmsg, err := json.Marshal(pdata)
	if err != nil {
		r.logger.Error("json marshal", "err", err)
		return
	}

	r.broadcast <- broadcastMessage{
		user: u,
		msg:  bmsg,
	}
}

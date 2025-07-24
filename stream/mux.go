package stream

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/charmbracelet/log"
	"github.com/tifye/shigure/assert"
)

const (
	MessageSizeLimit = 65_535
	MessageTypeLen   = 16
)

type ID = uint32

type Mux struct {
	logger    *log.Logger
	idCounter atomic.Uint32
	sessions  map[ID]*User
	handlers  map[MessageType]func(id ID, data []byte) error
	smu       sync.RWMutex
}

// todo: user connect hook
func NewMux() *Mux {
	return &Mux{
		logger: log.NewWithOptions(os.Stdout, log.Options{
			ReportTimestamp: false,
			Level:           log.DebugLevel,
		}),
		sessions: map[ID]*User{},
		handlers: map[MessageType]func(id ID, data []byte) error{},
	}
}

func (m *Mux) Connect(write func(id ID, data []byte)) ID {
	id := m.idCounter.Add(1)
	assert.Assert(id < 10_000, fmt.Sprintf("id counter too high: %d", id))

	sesh := &User{
		id:     id,
		writer: write,
	}

	m.smu.Lock()
	_, exists := m.sessions[id]
	assert.Assert(!exists, fmt.Sprintf("session with id %d already exists", id))
	m.sessions[id] = sesh
	m.smu.Unlock()

	return id
}

func (m *Mux) Disconnect(id ID) error {
	assert.Assert(id < 10_000, fmt.Sprintf("invalid id passed: %d", id))

	m.smu.Lock()
	delete(m.sessions, id)
	m.smu.Unlock()

	return nil
}

func (m *Mux) RegisterHandler(typ string, handler func(id ID, data []byte) error) {
	assert.AssertNotNil(handler)
	assert.Assert(len(typ) <= MessageTypeLen, "message type too long")

	mtype := [16]byte{}
	copy(mtype[:], []byte(typ)[:])

	m.smu.Lock()
	m.handlers[mtype] = handler
	m.smu.Unlock()
}

type MessageType [MessageTypeLen]byte

type Message struct {
	Type   string          `json:"type"`
	Paylod json.RawMessage `json:"payload,omitzero,omitempty"`
}

func (m *Mux) UserMessage(id ID, data []byte) error {
	assert.AssertNotNil(data)
	assert.Assert(len(data) < MessageSizeLimit, fmt.Sprintf("message too big: %d", len(data)))

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("unmarhsal message: %s", err)
	}

	if len(msg.Type) > MessageTypeLen {
		return fmt.Errorf("message type too long, expect length of %d but got %d", MessageTypeLen, len(msg.Type))
	}

	m.logger.Debug("user message", "id", id, "type", msg.Type)

	mtype := [16]byte{}
	copy(mtype[:], []byte(msg.Type)[:])
	handler, ok := m.handlers[mtype]
	if !ok {
		m.logger.Warnf("could not find handler for messsage type %s", msg.Type)
		return nil
	}

	assert.AssertNotNil(handler)
	if err := handler(id, msg.Paylod); err != nil {
		return fmt.Errorf("handler[%s]: %s", msg.Type, err)
	}

	return nil
}

func (m *Mux) Broadcast(typ string, payload []byte, filter func(id ID) bool) error {
	assert.AssertNotEmpty(typ)
	assert.Assert(len(payload) < MessageSizeLimit, fmt.Sprintf("message too big: %d", len(payload)))
	assert.Assert(len(typ) <= MessageTypeLen, "message type too long")

	msg := Message{
		Type:   typ,
		Paylod: payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("json marshal: %s", err)
	}

	m.logger.Debug("broadcasting message", "type", typ, "bytes", len(data))

	if filter == nil {
		filter = func(id ID) bool { return true }
	}

	m.smu.RLock()
	for _, sesh := range m.sessions {
		if sesh.writer == nil {
			continue
		}

		if filter(sesh.id) {
			sesh.writer(sesh.id, data)
		}
	}
	m.smu.RUnlock()

	return nil
}

type User struct {
	id     ID
	writer func(id ID, data []byte)
}

func (s *User) ID() ID {
	return s.id
}

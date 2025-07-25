package stream

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/tifye/shigure/assert"
)

const (
	MessageSizeLimit = 65_535
	MessageTypeLen   = 16
)

type ID = uint32

type Mux struct {
	logger *log.Logger

	handlers        map[MessageType]func(id ID, data []byte) error
	disconnectHooks []func(id ID)
	connectHooks    []func(id ID, sesh *User)
	handlersMu      sync.RWMutex

	users   map[ID]*User
	usersMu sync.RWMutex
}

func NewMux() *Mux {
	return &Mux{
		logger: log.NewWithOptions(os.Stdout, log.Options{
			ReportTimestamp: false,
			Level:           log.DebugLevel,
		}),
		users:           map[ID]*User{},
		handlers:        map[MessageType]func(id ID, data []byte) error{},
		connectHooks:    []func(id ID, sesh *User){},
		disconnectHooks: []func(id ID){},
	}
}

func (m *Mux) Connect(write func(id ID, data []byte)) ID {
	id := rand.Uint32()
	m.connect(id, write)
	return id
}

func (m *Mux) Reconnect(id ID, write func(id ID, data []byte)) error {
	m.usersMu.RLock()
	_, exists := m.users[id]
	m.usersMu.RUnlock()

	if exists {
		return fmt.Errorf("user already connected")
	}

	m.connect(id, write)
	return nil
}

func (m *Mux) connect(id ID, write func(id ID, data []byte)) {
	user := &User{
		id:     id,
		writer: write,
	}

	m.usersMu.Lock()
	_, exists := m.users[id]
	m.users[id] = user
	m.usersMu.Unlock()
	assert.Assert(!exists, fmt.Sprintf("session with id %d already existed", id))

	m.handlersMu.RLock()
	hooks := make([]func(ID, *User), len(m.connectHooks))
	copy(hooks, m.connectHooks)
	m.handlersMu.RUnlock()

	for _, hook := range hooks {
		assert.AssertNotNil(hook)
		hook(id, user)
	}
}

func (m *Mux) Disconnect(id ID) error {
	m.usersMu.Lock()
	delete(m.users, id)
	m.usersMu.Unlock()

	m.handlersMu.RLock()
	hooks := make([]func(ID), len(m.disconnectHooks))
	copy(hooks, m.disconnectHooks)
	m.handlersMu.RUnlock()

	for _, hook := range hooks {
		assert.AssertNotNil(hook)
		hook(id)
	}

	return nil
}

func (m *Mux) RegisterHandler(typ string, handler func(id ID, data []byte) error) {
	assert.AssertNotNil(handler)
	assert.Assert(len(typ) <= MessageTypeLen, "message type too long")

	mtype := [16]byte{}
	copy(mtype[:], []byte(typ)[:])

	m.handlersMu.Lock()
	m.handlers[mtype] = handler
	m.handlersMu.Unlock()
}

func (m *Mux) RegisterDisconnectHook(hook func(id ID)) {
	assert.AssertNotNil(hook)

	m.handlersMu.Lock()
	m.disconnectHooks = append(m.disconnectHooks, hook)
	m.handlersMu.Unlock()
}

func (m *Mux) RegisterConnectHook(hook func(id ID, user *User)) {
	assert.AssertNotNil(hook)

	m.handlersMu.Lock()
	m.connectHooks = append(m.connectHooks, hook)
	m.handlersMu.Unlock()
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

func (m *Mux) SendMessage(id ID, typ string, payload []byte) error {
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

	m.usersMu.RLock()
	user, ok := m.users[id]
	m.usersMu.RUnlock()

	if !ok {
		return fmt.Errorf("no user found with id %d", id)
	}

	if user.writer != nil {
		user.writer(id, data)
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

	m.usersMu.RLock()
	for _, user := range m.users {
		if user.writer == nil {
			continue
		}

		if filter(user.id) {
			user.writer(user.id, data)
		}
	}
	m.usersMu.RUnlock()

	return nil
}

type User struct {
	id     ID
	writer func(id ID, data []byte)
}

func (s *User) ID() ID {
	return s.id
}

package mux

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/tifye/shigure/assert"
)

var (
	idSeed = [...]byte{78, 112, 168, 197, 122, 43, 91, 0, 163, 125, 98, 19, 3, 1, 102, 17, 228, 84, 34, 216, 129, 91, 143, 122, 40, 166, 236, 206, 232, 87, 208, 244}
)

const (
	MessageSizeLimit  = 65_535
	MaxMessageTypeLen = 16

	muxMessageTypePrefix = "mux:"
	subscribeMesssage    = "subscribe"
	unsubscribeMesssage  = "unsubscribe"
)

type ID = [16]byte

type Handler interface {
	HandleMessage(channel *Channel, msg []byte) error
}

type MessageType = string

type Mux struct {
	logger *log.Logger
	rnd    *rand.ChaCha8

	mu                   sync.RWMutex
	sessions             []*Session
	channels             map[ID]*Channel
	channelSubscriptions map[MessageType][]*Channel
	handlers             map[MessageType]Handler
}

func NewMux(logger *log.Logger) *Mux {
	assert.AssertNotNil(logger)
	return &Mux{
		logger:               logger,
		rnd:                  rand.NewChaCha8(idSeed),
		sessions:             []*Session{},
		channels:             map[ID]*Channel{},
		channelSubscriptions: map[MessageType][]*Channel{},
		handlers:             map[MessageType]Handler{},
	}
}

func (m *Mux) RegisterHandler(typ MessageType, handler Handler) {
	m.mu.Lock()
	_, exists := m.handlers[typ]
	assert.Assert(!exists, "handler already registered for this MessageType")
	m.handlers[typ] = handler
	m.mu.Unlock()
}

// Connect creates a new channel in the session with
// session.ID = sessionID. The channel's ID is returned.
//
// If no session exists a new one will be created.
//
// Connect hooks are called after the channel and/or session
// is added.
func (m *Mux) Connect(sessionID ID, writer io.Writer) ID {
	session := m.Session(sessionID)
	if session == nil {
		session = newSession(sessionID)
		defer func() {
			assert.Assert(len(session.channels) > 0, "expected to add at least one channel")
		}()
	} else {
		assert.Assert(len(session.channels) > 0, "expected to have at least one channel")
	}

	channelID := ID{}
	// Concurrent calls to Read and has undefined output.
	// This is ok because we don't expect a deterministic
	// output.
	_, _ = m.rnd.Read(channelID[:])
	channel := newChannel(channelID, session, writer)
	assert.AssertNotNil(channel)
	session.addChannel(channel)

	m.mu.Lock()
	m.sessions = append(m.sessions, session)
	m.channels[channelID] = channel
	m.mu.Unlock()

	return channelID
}

// Disconnect removes a channel from a session. If
// no channel or session can be found with their respective IDs
// then Disconnect is noop.
//
// Disconnect hooks are called after the channel and/or session
// is removed.
func (m *Mux) Disconnect(sessionID, channelID ID) {
	session := m.Session(sessionID)
	if session == nil {
		return
	}

	numChannels := session.removeChannel(channelID)
	if numChannels > 0 {
		return
	}

	m.mu.Lock()
	for typ, channels := range m.channelSubscriptions {
		m.channelSubscriptions[typ] = slices.DeleteFunc(channels, func(c *Channel) bool {
			return c.ID() == channelID
		})
	}

	delete(m.channels, channelID)

	m.sessions = slices.DeleteFunc(m.sessions, func(s *Session) bool {
		return s.ID() == sessionID
	})
	m.mu.Unlock()
}

// Session returns the session with the corresponding
// sessionID or nil if none exists.
func (m *Mux) Session(sessionID ID) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.sessions {
		if s.id == sessionID {
			return s
		}
	}
	return nil
}

type Message struct {
	Type   MessageType     `json:"type"`
	Paylod json.RawMessage `json:"payload,omitzero,omitempty"`
}

func (m *Mux) Message(sessionID, channelID ID, data []byte) error {
	assert.AssertNotNil(data)

	session := m.Session(sessionID)
	if session == nil {
		return fmt.Errorf("session does not exist")
	}

	channel := session.Channel(channelID)
	if channel == nil {
		return fmt.Errorf("channel does not exist")
	}

	m.logger.Debug("message", "sesdsionID", sessionID, "channelID", channelID, "msg", string(data))

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("unmarshal message: %s", err)
	}

	if len(msg.Type) > MaxMessageTypeLen {
		return fmt.Errorf("message type too long, expect length of %d but got %d", MaxMessageTypeLen, len(msg.Type))
	}

	if strings.HasPrefix(msg.Type, string(muxMessageTypePrefix)) {
		return m.handleMuxMessage(channel, msg)
	} else {
		return m.handleMessage(channel, msg)
	}
}

type muxRegisterMessage struct {
	MessageType MessageType
}

func (m *Mux) handleMuxMessage(channel *Channel, msg Message) error {
	assert.AssertNotNil(channel)
	assert.Assert(strings.HasPrefix(msg.Type, muxMessageTypePrefix), "expected message type to have correct prefix")

	action := strings.TrimPrefix(msg.Type, muxMessageTypePrefix)
	switch action {
	case subscribeMesssage:
		var reg muxRegisterMessage
		if err := json.Unmarshal(msg.Paylod, &reg); err != nil {
			return fmt.Errorf("unmarshal subscribe message: %s", err)
		}

		if len(reg.MessageType) == 0 {
			return fmt.Errorf("no MessageType provided to subscribe to")
		}

		m.mu.RLock()
		_, handlerExists := m.handlers[reg.MessageType]
		m.mu.RUnlock()
		if !handlerExists {
			m.logger.Warn("trying to subscribe on MessageType with no registered handlers", "messageType", reg.MessageType, "sessionID", channel.session.ID(), "channelID", channel.ID())
			return nil
		}

		m.mu.Lock()
		m.channelSubscriptions[reg.MessageType] = append(m.channelSubscriptions[reg.MessageType], channel)
		channel.subscribeTo(reg.MessageType)
		m.mu.Unlock()
	case unsubscribeMesssage:
		var reg muxRegisterMessage
		if err := json.Unmarshal(msg.Paylod, &reg); err != nil {
			return fmt.Errorf("unmarshal unsubscribe message: %s", err)
		}

		if len(reg.MessageType) == 0 {
			return fmt.Errorf("no MessageType provided to unsubscribe from")
		}

		channels := m.SubscribedChannels(reg.MessageType)
		if channels == nil {
			return nil
		}

		m.mu.Lock()
		m.channelSubscriptions[reg.MessageType] = slices.DeleteFunc(m.channelSubscriptions[reg.MessageType], func(c *Channel) bool {
			return c.ID() == channel.ID()
		})
		channel.unsubscribeFrom(reg.MessageType)
		m.mu.Unlock()
	default:
		panic(fmt.Sprintf("invalid mux action: %s", action))
	}

	return nil
}

func (m *Mux) handleMessage(channel *Channel, msg Message) error {
	assert.AssertNotNil(channel)
	assert.Assert(len(msg.Type) <= MaxMessageTypeLen, "message type too long")
	assert.Assert(len(msg.Type) > 0, "no message type provided")

	m.mu.RLock()
	handler := m.handlers[msg.Type]
	m.mu.RUnlock()

	handler.HandleMessage(channel, msg.Paylod)
	return nil
}

func (m *Mux) Sessions() []*Session {
	m.mu.RLock()
	sessions := make([]*Session, len(m.sessions))
	copy(sessions, m.sessions)
	m.mu.RUnlock()
	return sessions
}

func (m *Mux) SubscribedChannels(typ MessageType) []*Channel {
	assert.Assert(len(typ) <= MaxMessageTypeLen, "message type too long")
	m.mu.RLock()
	if m.channelSubscriptions[typ] == nil {
		return nil
	}
	channels := make([]*Channel, len(m.channelSubscriptions[typ]))
	copy(channels, m.channelSubscriptions[typ])
	m.mu.RUnlock()
	return channels
}

func (m *Mux) SendSession(sessionID ID, typ MessageType, payload []byte, exclude func(c *Channel) bool) error {
	session := m.Session(sessionID)
	if session == nil {
		return fmt.Errorf("session does not exist")
	}

	return m.sendSession(session, typ, payload, exclude)
}

func (m *Mux) sendSession(session *Session, typ MessageType, payload []byte, exclude func(c *Channel) bool) error {
	assert.AssertNotEmpty(typ)
	assert.Assert(len(typ) <= MaxMessageTypeLen, "message type too long")
	assert.AssertNotNil(payload)
	assert.Assert(len(payload) <= MessageSizeLimit, "payload too long") // Does not exactly cover entire message length

	msg := Message{
		Type:   typ,
		Paylod: payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("json marshal: %s", err)
	}

	if exclude == nil {
		exclude = func(_ *Channel) bool { return false }
	}

	channels := session.Channels()
	for _, channel := range channels {
		if !channel.IsSubscribedTo(typ) || exclude(channel) {
			m.logger.Debug("meep")
			continue
		}

		_, err := channel.writer.Write(data)
		if err != nil {
			m.logger.Warn("write on channel", "channelID", channel.ID(), "sessionID", channel.session.ID())
		}
	}

	return nil
}

func (m *Mux) SendChannelSession(channelID ID, typ MessageType, payload []byte, exclude func(c *Channel) bool) error {
	channel, ok := m.channels[channelID]
	if !ok {
		return fmt.Errorf("channel does not exist")
	}
	assert.AssertNotNil(channel.session)
	return m.sendSession(channel.session, typ, payload, exclude)
}

func (m *Mux) SendChannel(channelID ID, typ MessageType, payload []byte) error {
	assert.AssertNotEmpty(typ)
	assert.Assert(len(typ) <= MaxMessageTypeLen, "message type too long")
	assert.AssertNotNil(payload)
	assert.Assert(len(payload) <= MessageSizeLimit, "payload too long") // Does not exactly cover entire message length

	channel, ok := m.channels[channelID]
	if !ok {
		return fmt.Errorf("channel does not exist")
	}

	if !channel.IsSubscribedTo(typ) {
		return nil
	}

	_, err := channel.writer.Write(payload)
	if err != nil {
		m.logger.Warn("write on channel", "channelID", channel.ID(), "sessionID", channel.session.ID())
	}

	return nil
}

func (m *Mux) Broadcast(typ MessageType, payload []byte, exclude func(c *Channel) bool) error {
	assert.AssertNotEmpty(typ)
	assert.Assert(len(typ) <= MaxMessageTypeLen, "message type too long")
	assert.AssertNotNil(payload)
	assert.Assert(len(payload) <= MessageSizeLimit, "payload too long") // Does not exactly cover entire message length

	msg := Message{
		Type:   typ,
		Paylod: payload,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("json marshal: %s", err)
	}

	if exclude == nil {
		exclude = func(_ *Channel) bool { return false }
	}

	channels := m.SubscribedChannels(typ)
	for _, channel := range channels {
		assert.AssertNotNil(channel)
		if exclude(channel) {
			continue
		}

		_, err := channel.writer.Write(data)
		if err != nil {
			m.logger.Warn("write on channel", "channelID", channel.ID(), "sessionID", channel.session.ID())
		}
	}

	return nil
}

type Session struct {
	id       ID
	mu       sync.RWMutex
	channels []*Channel
}

func newSession(id ID) *Session {
	assert.AssertNotNil(id)
	return &Session{
		id:       id,
		channels: []*Channel{},
	}
}

// Channel returns the channel with the corresponding
// channelID or nil if none exists.
func (s *Session) Channel(channelID ID) *Channel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.channels {
		if c.id == channelID {
			return c
		}
	}
	return nil
}

func (s *Session) addChannel(c *Channel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.channels = append(s.channels, c)
}

// removeChannel removes a channel from the session
// with the corresponding channelID. It returns
// the numbers of channels left in the session.
func (s *Session) removeChannel(id ID) uint {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.channels = slices.DeleteFunc(s.channels, func(c *Channel) bool {
		return c.id == id
	})
	return uint(len(s.channels))
}

func (s *Session) Channels() []*Channel {
	s.mu.RLock()
	channels := make([]*Channel, len(s.channels))
	copy(channels, s.channels)
	s.mu.RUnlock()
	return channels
}

func (s *Session) ID() ID {
	return s.id
}

type Channel struct {
	id            ID
	session       *Session
	writer        io.Writer
	subscriptions []MessageType
	mu            sync.RWMutex
}

func newChannel(id ID, session *Session, writer io.Writer) *Channel {
	assert.AssertNotNil(id)
	assert.AssertNotNil(session)
	assert.AssertNotNil(writer)
	return &Channel{
		id:            id,
		session:       session,
		writer:        writer,
		subscriptions: []MessageType{},
	}
}

func (c *Channel) ID() ID {
	return c.id
}

func (c *Channel) IsSubscribedTo(typ MessageType) bool {
	c.mu.RLock()
	ok := slices.Contains(c.subscriptions, typ)
	c.mu.RUnlock()
	return ok
}

func (c *Channel) subscribeTo(typ MessageType) {
	c.mu.Lock()
	c.subscriptions = append(c.subscriptions, typ)
	c.mu.Unlock()
}

func (c *Channel) unsubscribeFrom(typ MessageType) {
	c.mu.Lock()
	c.subscriptions = slices.DeleteFunc(c.subscriptions, func(t MessageType) bool {
		return t == typ
	})
	c.mu.Unlock()
}

func (c *Channel) Session() *Session {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.session
}

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
	*hooks
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
		hooks:                newHooks(),
	}
}

func (m *Mux) RegisterHandler(typ MessageType, handler Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, exists := m.handlers[typ]
	assert.Assert(!exists, "handler already registered for this MessageType")
	m.handlers[typ] = handler
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
	defer m.runConnectHooks(channel, len(session.channels) == 0)

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
	m.logger.Info("mux disconnect", "channelID", channelID, "sessionID", sessionID)

	session := m.Session(sessionID)
	if session == nil {
		return
	}

	channel := session.Channel(channelID)
	defer m.runDisconnectHooks(channel, len(session.channels) == 0)

	subscriptions := channel.Subscriptions()
	for _, typ := range subscriptions {
		m.unsubscribeChannel(channel, typ)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.channels, channelID)
	numChannels := session.removeChannel(channelID)
	if numChannels == 0 {
		m.sessions = slices.DeleteFunc(m.sessions, func(s *Session) bool {
			return s.ID() == sessionID
		})
	}
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
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload,omitzero,omitempty"`
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

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("unmarshal message: %s", err)
	}

	if len(msg.Type) > MaxMessageTypeLen {
		return fmt.Errorf("message type too long, expect length of %d but got %d", MaxMessageTypeLen, len(msg.Type))
	}

	var err error
	if strings.HasPrefix(msg.Type, string(muxMessageTypePrefix)) {
		err = m.handleMuxMessage(channel, msg)
	} else {
		err = m.handleMessage(channel, msg)
	}

	m.runMessageHooks(channel, msg.Type, msg.Payload)

	return err
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
		if err := json.Unmarshal(msg.Payload, &reg); err != nil {
			return fmt.Errorf("unmarshal subscribe message: %s", err)
		}

		if len(reg.MessageType) == 0 {
			return fmt.Errorf("no MessageType provided to subscribe to")
		}

		m.subscribeChannel(channel, reg.MessageType)
	case unsubscribeMesssage:
		var reg muxRegisterMessage
		if err := json.Unmarshal(msg.Payload, &reg); err != nil {
			return fmt.Errorf("unmarshal unsubscribe message: %s", err)
		}

		if len(reg.MessageType) == 0 {
			return fmt.Errorf("no MessageType provided to unsubscribe from")
		}

		m.unsubscribeChannel(channel, reg.MessageType)
	default:
		m.logger.Warn("invalid mux action", "action", action)
	}

	return nil
}

func (m *Mux) subscribeChannel(channel *Channel, typ MessageType) {
	assert.AssertNotNil(channel)
	assert.Assert(len(typ) <= MaxMessageTypeLen, "message type too long")
	assert.AssertNotEmpty(typ)

	if channel.IsSubscribedTo(typ) {
		return
	}

	m.mu.RLock()
	_, handlerExists := m.handlers[typ]
	m.mu.RUnlock()
	if !handlerExists {
		m.logger.Warn("trying to subscribe on MessageType with no registered handlers", "messageType", typ, "sessionID", channel.session.ID(), "channelID", channel.ID())
		return
	}

	m.mu.Lock()
	m.channelSubscriptions[typ] = append(m.channelSubscriptions[typ], channel)
	channel.addSubscription(typ)
	m.mu.Unlock()

	m.runSubscriptionHooks(channel, typ, true)
}

func (m *Mux) unsubscribeChannel(channel *Channel, typ MessageType) {
	assert.AssertNotNil(channel)
	assert.Assert(len(typ) <= MaxMessageTypeLen, "message type too long")
	assert.AssertNotEmpty(typ)

	channels := m.SubscribedChannels(typ)
	if channels == nil {
		return
	}

	m.mu.Lock()
	m.channelSubscriptions[typ] = slices.DeleteFunc(m.channelSubscriptions[typ], func(c *Channel) bool {
		return c.ID() == channel.ID()
	})
	channel.removeSubscription(typ)
	m.mu.Unlock()

	m.runSubscriptionHooks(channel, typ, false)
}

func (m *Mux) handleMessage(channel *Channel, msg Message) error {
	assert.AssertNotNil(channel)
	assert.Assert(len(msg.Type) <= MaxMessageTypeLen, "message type too long")
	assert.Assert(len(msg.Type) > 0, "no message type provided")

	m.mu.RLock()
	handler := m.handlers[msg.Type]
	m.mu.RUnlock()

	if handler == nil {
		return nil
	}

	return handler.HandleMessage(channel, msg.Payload)
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
	defer m.mu.RUnlock()
	if m.channelSubscriptions[typ] == nil {
		return nil
	}
	channels := make([]*Channel, len(m.channelSubscriptions[typ]))
	copy(channels, m.channelSubscriptions[typ])
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
		Type:    typ,
		Payload: payload,
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

	data, err := json.Marshal(Message{
		Type:    typ,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("marshal: %s", err)
	}

	_, err = channel.writer.Write(data)
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
		Type:    typ,
		Payload: payload,
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

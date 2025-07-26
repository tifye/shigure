package mux

import (
	"io"
	"slices"
	"sync"

	"github.com/tifye/shigure/assert"
)

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

func (c *Channel) addSubscription(typ MessageType) {
	c.mu.Lock()
	if slices.Contains(c.subscriptions, typ) {
		return
	}
	c.subscriptions = append(c.subscriptions, typ)
	c.mu.Unlock()
}

func (c *Channel) removeSubscription(typ MessageType) {
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

func (c *Channel) Subscriptions() []MessageType {
	c.mu.RLock()
	defer c.mu.RUnlock()
	subs := make([]MessageType, len(c.subscriptions))
	copy(subs, c.subscriptions)
	return subs
}

package mux

import (
	"sync"
)

// DisconnectHooks get called when a channel is
// disconnected.
//
// lastChannel is true if the channel
// was the last one in the session causing the session
// to be removed.
type DisconnectHook func(c *Channel, lastChannel bool)

// ConnectHooks get called when a channel is connected.
//
// firstChannel is true if the connection also triggered
// a new session.
type ConnectHook func(c *Channel, firstChannel bool)

type MessageHook func(c *Channel, typ MessageType, payload []byte)

type hooks struct {
	disconnect []DisconnectHook
	connect    []ConnectHook
	message    []MessageHook
	mu         sync.RWMutex
}

func newHooks() *hooks {
	return &hooks{
		connect:    []ConnectHook{},
		disconnect: []DisconnectHook{},
		message:    []MessageHook{},
	}
}

func (h *hooks) runMessageHooks(c *Channel, typ MessageType, payload []byte) {
	h.mu.RLock()
	funcs := make([]MessageHook, len(h.message))
	copy(funcs, h.message)
	h.mu.RUnlock()

	for _, f := range funcs {
		f(c, typ, payload)
	}
}
func (h *hooks) AddMessageHook(f MessageHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.message = append(h.message, f)
}

func (h *hooks) runConnectHooks(c *Channel, firstChannel bool) {
	h.mu.RLock()
	funcs := make([]ConnectHook, len(h.connect))
	copy(funcs, h.connect)
	h.mu.RUnlock()

	for _, f := range funcs {
		f(c, firstChannel)
	}
}
func (h *hooks) AddConnectHook(f ConnectHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connect = append(h.connect, f)
}

func (h *hooks) runDisconnectHooks(c *Channel, lastChannel bool) {
	h.mu.RLock()
	funcs := make([]DisconnectHook, len(h.disconnect))
	copy(funcs, h.disconnect)
	h.mu.RUnlock()

	for _, f := range funcs {
		f(c, lastChannel)
	}
}
func (h *hooks) AddDisconnectHook(f DisconnectHook) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.disconnect = append(h.disconnect, f)
}

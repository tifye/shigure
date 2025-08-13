package mux

import (
	"encoding/json"
	"io"
	"math/rand/v2"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
)

var (
	rnd = rand.NewChaCha8([32]byte{})
)

func TestMuxConnectDisconnect(t *testing.T) {
	mux := NewMux(log.New(io.Discard))

	s1ID := randomID(t)
	s1c1ID := mux.Connect(s1ID, io.Discard)
	s1c2ID := mux.Connect(s1ID, io.Discard)

	s2ID := randomID(t)
	s2c1ID := mux.Connect(s2ID, io.Discard)
	s2c2ID := mux.Connect(s2ID, io.Discard)

	if s := mux.Session(s1ID); assert.NotNil(t, s) {
		assert.NotNil(t, s.Channel(s1c1ID))
		assert.NotNil(t, s.Channel(s1c2ID))
	}

	if s := mux.Session(s2ID); assert.NotNil(t, s) {
		assert.NotNil(t, s.Channel(s2c1ID))
		assert.NotNil(t, s.Channel(s2c2ID))
	}

	mux.Disconnect(s1ID, s1c1ID)
	mux.Disconnect(s1ID, s1c2ID)
	assert.Nil(t, mux.Session(s1ID))

	if s := mux.Session(s2ID); assert.NotNil(t, s) {
		assert.NotNil(t, s.Channel(s2c1ID))
		assert.NotNil(t, s.Channel(s2c2ID))
	}

	mux.Disconnect(s2ID, s2c1ID)
	mux.Disconnect(s2ID, s2c2ID)
	assert.Nil(t, mux.Session(s2ID))
}

func TestMuxDisconnectCleanup(t *testing.T) {
	var messageType MessageType = "test"
	mux := NewMux(log.New(io.Discard))
	mux.RegisterHandler(messageType, HandlerFunc(func(c *Channel, data []byte) error { return nil }))

	didWrite := false

	sID := randomID(t)
	cID := mux.Connect(sID, WriterFunc(func(data []byte) (n int, err error) {
		didWrite = true
		return 0, nil
	}))
	err := mux.Message(sID, cID, registerMessage(t, messageType))
	assert.NoError(t, err)

	// Used to keep session alive
	_ = mux.Connect(sID, WriterFunc(func(data []byte) (n int, err error) {
		return 0, nil
	}))
	assert.NoError(t, err)

	mux.Disconnect(sID, cID)
	mux.Broadcast(messageType, []byte("{}"), nil)

	assert.False(t, didWrite)
}

func randomID(t *testing.T) ID {
	t.Helper()
	id := ID{}
	_, _ = rnd.Read(id[:])
	return id
}

func TestMuxBroadcast(t *testing.T) {
	t.Run("Channel not subscribed", func(t *testing.T) {
		var messageType MessageType = "test"
		mux := NewMux(log.New(io.Discard))
		mux.RegisterHandler(messageType, HandlerFunc(func(c *Channel, data []byte) error { return nil }))

		didWrite := false

		sID := randomID(t)
		cID := mux.Connect(sID, WriterFunc(func(data []byte) (n int, err error) {
			didWrite = true
			return 0, nil
		}))

		err := mux.Message(sID, cID, registerMessage(t, "test"))
		assert.NoError(t, err)
		err = mux.Message(sID, cID, unregisterMessage(t, "test"))
		assert.NoError(t, err)

		err = mux.Broadcast(messageType, []byte("{}"), nil)
		assert.NoError(t, err)
		assert.False(t, didWrite)
	})

	t.Run("Channel is subscribed", func(t *testing.T) {
		var messageType MessageType = "test"
		mux := NewMux(log.New(io.Discard))
		mux.RegisterHandler(messageType, HandlerFunc(func(c *Channel, data []byte) error { return nil }))

		didWrite := false

		sID := randomID(t)
		cID := mux.Connect(sID, WriterFunc(func(data []byte) (n int, err error) {
			didWrite = true
			return 0, nil
		}))

		err := mux.Message(sID, cID, registerMessage(t, "test"))
		assert.NoError(t, err)

		err = mux.Broadcast(messageType, []byte("{}"), nil)
		assert.NoError(t, err)
		assert.True(t, didWrite)
	})
}

func TestMuxSendSession(t *testing.T) {
	var messageType MessageType = "test"
	mux := NewMux(log.New(io.Discard))
	mux.RegisterHandler(messageType, HandlerFunc(func(c *Channel, data []byte) error { return nil }))

	s1DidWrite := false
	s2DidWrite := false

	s1ID := randomID(t)
	c1ID := mux.Connect(s1ID, WriterFunc(func(data []byte) (n int, err error) {
		s1DidWrite = true
		return 0, nil
	}))
	err := mux.Message(s1ID, c1ID, registerMessage(t, "test"))
	assert.NoError(t, err)

	s2ID := randomID(t)
	c2ID := mux.Connect(s2ID, WriterFunc(func(data []byte) (n int, err error) {
		s2DidWrite = true
		return 0, nil
	}))
	err = mux.Message(s2ID, c2ID, registerMessage(t, "test"))
	assert.NoError(t, err)

	err = mux.SendSession(s1ID, messageType, []byte("{}"), nil)
	assert.NoError(t, err)

	assert.True(t, s1DidWrite)
	assert.False(t, s2DidWrite)
}

type WriterFunc func(data []byte) (n int, err error)

func (w WriterFunc) Write(data []byte) (n int, err error) {
	return w(data)
}

func registerMessage(t *testing.T, typ MessageType) []byte {
	t.Helper()

	regMsg := muxRegisterMessage{
		MessageType: typ,
	}
	regMsgData, _ := json.Marshal(regMsg)

	msg := Message{
		Type:    muxMessageTypePrefix + subscribeMesssage,
		Payload: regMsgData,
	}
	msgData, _ := json.Marshal(msg)

	return msgData
}

func unregisterMessage(t *testing.T, typ MessageType) []byte {
	t.Helper()

	regMsg := muxRegisterMessage{
		MessageType: typ,
	}
	regMsgData, _ := json.Marshal(regMsg)

	msg := Message{
		Type:    muxMessageTypePrefix + unsubscribeMesssage,
		Payload: regMsgData,
	}
	msgData, _ := json.Marshal(msg)

	return msgData
}

type HandlerFunc func(c *Channel, data []byte) error

func (h HandlerFunc) HandleMessage(c *Channel, data []byte) error {
	return h(c, data)
}

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/rand/v2"

	"github.com/charmbracelet/log"
	"github.com/tifye/shigure/assert"
	"github.com/tifye/shigure/mux"
)

type userSimulator struct {
	logger *log.Logger
	rnd    *rand.Rand

	// Change out of 100 that a new user will connect
	userConnectProbability uint
	// Chance out of 100 that an existing user will disconnect
	userDisconnectProbability uint
	// Chance out of 100 that a disconnect will be called on a
	// non-existent user
	invalidDisconnectFaultProbability uint

	connectedUsers    map[user]struct{}
	disconnectedUsers map[user]struct{}

	mux *mux.Mux
}

type user struct {
	sessionID mux.ID
	channelID mux.ID
}

func newUserSimulator(logger *log.Logger, mux *mux.Mux, rnd *rand.Rand) *userSimulator {
	assert.AssertNotNil(logger)
	assert.AssertNotNil(mux)
	assert.AssertNotNil(rnd)

	return &userSimulator{
		logger: logger,
		rnd:    rnd,

		userConnectProbability:            0,
		userDisconnectProbability:         rnd.UintN(probabilityRange),
		invalidDisconnectFaultProbability: rnd.UintN(probabilityRange),

		connectedUsers:    map[user]struct{}{},
		disconnectedUsers: map[user]struct{}{},

		mux: mux,
	}
}

func (s *userSimulator) String() string {
	return fmt.Sprintf(
		`userConnectProbability: %d%%
userDisconnectProbability: %d%%
invalidDisconnectFaultProbability: %d%%
`, s.userConnectProbability, s.userDisconnectProbability, s.invalidDisconnectFaultProbability)
}

func (s *userSimulator) Step() {
	if Chance(s.rnd, s.userConnectProbability) {
		s.connectUser()
	}

	if Chance(s.rnd, s.userDisconnectProbability) {
		if Chance(s.rnd, s.invalidDisconnectFaultProbability) {
			s.invalidDisconnectUser()
		} else {
			s.disconnectUser()
		}
	}
}

func (s *userSimulator) connectUser() {
	sid := [16]byte{}
	s.generateMuxID(sid[:])
	cid := s.mux.Connect(sid, io.Discard)
	s.connectedUsers[user{sessionID: sid, channelID: cid}] = struct{}{}
	s.logger.Info("User connected", "sid", sid, "cid", cid)
}

func (s *userSimulator) disconnectUser() {
	if len(s.connectedUsers) == 0 {
		s.logger.Info("No users to disconnect")
		return
	}

	n := s.rnd.IntN(len(s.connectedUsers))
	var user user
	i := 0
	for user = range s.connectedUsers {
		if i == n {
			break
		}
		i++
	}

	s.mux.Disconnect(user.sessionID, user.channelID)

	delete(s.connectedUsers, user)
	s.disconnectedUsers[user] = struct{}{}
	s.logger.Info("User disconnected", "sid", user.sessionID, "cid", user.channelID)
}

func (s *userSimulator) invalidDisconnectUser() {
	if len(s.disconnectedUsers) == 0 {
		return
	}

	n := s.rnd.IntN(len(s.disconnectedUsers))
	var user user
	i := 0
	for user = range s.disconnectedUsers {
		if i == n {
			break
		}
		i++
	}

	s.mux.Disconnect(user.sessionID, user.channelID)
}

func (s *userSimulator) generateMuxID(b []byte) {
	binary.LittleEndian.PutUint64(b, s.rnd.Uint64())
	binary.LittleEndian.PutUint64(b[8:], s.rnd.Uint64())
}

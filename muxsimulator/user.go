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

type userSimulatorConfig struct {
	// Change out of 100 that a new user will connect
	UserConnectProbability uint
	// Chance out of 100 that an existing user will disconnect
	UserDisconnectProbability uint
	// Chance out of 100 that a disconnect will be called on a
	// non-existent user
	InvalidDisconnectFaultProbability uint
}

type userSimulator struct {
	logger *log.Logger
	rnd    *rand.Rand

	// Used for metrics
	numConnects           uint
	numDisconnects        uint
	numInvalidDisconnects uint

	config userSimulatorConfig

	connectedUsers    map[user]struct{}
	disconnectedUsers map[user]struct{}

	mux *mux.Mux
}

type user struct {
	sessionID mux.ID
	channelID mux.ID
}

func newUserSimulator(
	logger *log.Logger,
	mux *mux.Mux,
	rnd *rand.Rand,
	config userSimulatorConfig,
) *userSimulator {
	assert.AssertNotNil(logger)
	assert.AssertNotNil(mux)
	assert.AssertNotNil(rnd)

	return &userSimulator{
		logger: logger,
		rnd:    rnd,

		config: config,

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
numConnects: %d
numDisconnects: %d
numInvalidDisconnects: %d
`, s.config.UserConnectProbability,
		s.config.UserDisconnectProbability,
		s.config.InvalidDisconnectFaultProbability,
		s.numConnects,
		s.numDisconnects,
		s.numInvalidDisconnects,
	)
}

func (s *userSimulator) Step() {
	if Chance(s.rnd, s.config.UserConnectProbability) {
		s.connectUser()
	}

	if Chance(s.rnd, s.config.UserDisconnectProbability) {
		if Chance(s.rnd, s.config.InvalidDisconnectFaultProbability) {
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

	s.logger.Debug("User connected", "sid", sid, "cid", cid)
	s.numConnects += 1
}

func (s *userSimulator) disconnectUser() {
	if len(s.connectedUsers) == 0 {
		s.logger.Debug("No users to disconnect")
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

	s.logger.Debug("User disconnected", "sid", user.sessionID, "cid", user.channelID)
	s.numDisconnects += 1
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

	s.numInvalidDisconnects += 1
}

func (s *userSimulator) generateMuxID(b []byte) {
	binary.LittleEndian.PutUint64(b, s.rnd.Uint64())
	binary.LittleEndian.PutUint64(b[8:], s.rnd.Uint64())
}

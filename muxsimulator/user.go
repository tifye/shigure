package main

import (
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

	connectedUsers    map[mux.ID]struct{}
	disconnectedUsers map[mux.ID]struct{}

	mux *mux.Mux
}

func newUserSimulator(logger *log.Logger, mux *mux.Mux, rnd *rand.Rand) *userSimulator {
	assert.AssertNotNil(logger)
	assert.AssertNotNil(mux)
	assert.AssertNotNil(rnd)

	return &userSimulator{
		logger: logger,
		rnd:    rnd,

		userConnectProbability:            rnd.UintN(probabilityRange),
		userDisconnectProbability:         rnd.UintN(probabilityRange),
		invalidDisconnectFaultProbability: rnd.UintN(probabilityRange),

		connectedUsers:    map[[16]byte]struct{}{},
		disconnectedUsers: map[[16]byte]struct{}{},

		mux: mux,
	}
}

func (s *userSimulator) Step() {

}

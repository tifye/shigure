package stream

import (
	"math/rand/v2"
	"os"

	"github.com/charmbracelet/log"
	"github.com/tifye/shigure/assert"
)

const (
	maxIterations    = 100_000
	probabilityRange = 100
)

type Simulator struct {
	logger *log.Logger
	seed1  uint64
	seed2  uint64

	// Change out of 100 that a new user will connect
	userConnectProbability uint
	// Chance out of 100 that an existing user will disconnect
	userDisconnectProbability uint
	// Chance out of 100 that a disconnect will be called on a
	// non-existent user
	invalidDisconnectFaultProbability uint
	connectedUsers                    map[ID]struct{}
	disconnectedUsers                 map[ID]struct{}

	mux *Mux
	rnd *rand.Rand
}

func NewSimulator(seed1, seed2 uint64) *Simulator {
	logger := log.NewWithOptions(os.Stdout, log.Options{
		Level:           log.DebugLevel,
		ReportTimestamp: false,
	})

	rnd := rand.New(rand.NewPCG(seed1, seed2))

	return &Simulator{
		logger: logger,
		seed1:  seed1,
		seed2:  seed2,

		userConnectProbability:            rnd.UintN(probabilityRange),
		userDisconnectProbability:         rnd.UintN(probabilityRange),
		invalidDisconnectFaultProbability: rnd.UintN(probabilityRange),
		connectedUsers:                    map[ID]struct{}{},
		disconnectedUsers:                 map[ID]struct{}{},

		mux: NewMux(),
		rnd: rnd,
	}
}

func (s *Simulator) Run() {
	s.logger.Info("Simulator started",
		"seed1", s.seed1, "seed2", s.seed2,
		"userConnectProbability", s.userConnectProbability,
		"userDisconnectProbability", s.userDisconnectProbability,
	)
	defer func() {
		s.logger.Info("Simulator finished",
			"seed1", s.seed1, "seed2", s.seed2,
		)
	}()

	for range maxIterations {
		// s.logger.Info("Iteration", "iteration", i)
		s.Step()
	}
}

func (s *Simulator) Step() {
	if s.Chance(s.userConnectProbability) {
		s.connectUser()
	}

	if s.Chance(s.userDisconnectProbability) {
		s.disconnectUser()

		if s.Chance(s.invalidDisconnectFaultProbability) {
			s.invalidDiscconectUser()
		}
	}
}

func (s *Simulator) connectUser() {
	id := s.mux.Connect(nil)
	s.logger.Debug("User connect", "id", id)
	s.connectedUsers[id] = struct{}{}
}

func (s *Simulator) invalidDiscconectUser() {
	assert.Assert(len(s.disconnectedUsers) > 0, "expected to have already disconnected users")

	n := s.rnd.IntN(len(s.disconnectedUsers))
	var id ID
	i := 0
	for uid := range s.disconnectedUsers {
		if i == n {
			id = uid
			break
		}
		i++
	}

	s.logger.Debug("Invalid user disconnect", "id", id)
	_ = s.mux.Disconnect(id)
}

func (s *Simulator) disconnectUser() {
	if len(s.connectedUsers) == 0 {
		return
	}

	n := s.rnd.IntN(len(s.connectedUsers))
	var id ID
	i := 0
	for uid := range s.connectedUsers {
		if i == n {
			id = uid
			break
		}
		i++
	}

	s.logger.Debug("User disconnect", "id", id)
	_ = s.mux.Disconnect(id)

	delete(s.connectedUsers, id)
	s.disconnectedUsers[id] = struct{}{}
}

func (s *Simulator) Chance(probability uint) bool {
	return probability == s.rnd.UintN(probabilityRange)
}

// todo: implement random closures
type NoopReadWriteCloser struct{}

func (NoopReadWriteCloser) Read(p []byte) (int, error)  { return len(p), nil }
func (NoopReadWriteCloser) Write(p []byte) (int, error) { return len(p), nil }
func (NoopReadWriteCloser) Close() error                { return nil }

package main

import (
	"io"
	"math/rand/v2"

	"github.com/charmbracelet/log"
	"github.com/tifye/shigure/mux"
)

const (
	maxIterations    = 100_000
	probabilityRange = 100
)

type Simulator struct {
	logger *log.Logger
	seed1  uint64
	seed2  uint64

	mux *mux.Mux
	rnd *rand.Rand
}

func NewSimulator(seed1, seed2 uint64, w io.Writer) *Simulator {
	logger := log.NewWithOptions(w, log.Options{
		Level:           log.DebugLevel,
		ReportTimestamp: false,
	})

	rnd := rand.New(rand.NewPCG(seed1, seed2))

	return &Simulator{
		logger: logger,
		seed1:  seed1,
		seed2:  seed2,

		mux: mux.NewMux(log.New(io.Discard)),
		rnd: rnd,
	}
}

func (s *Simulator) Run() {
	s.logger.Info("Simulator started",
		"seed1", s.seed1, "seed2", s.seed2,
	)
	defer func() {
		s.logger.Info("Simulator finished",
			"seed1", s.seed1, "seed2", s.seed2,
		)
	}()

	for range maxIterations {
		s.Step()
	}
}

func (s *Simulator) Step() {
	// Do random things
	// s.logger.Info("doing something")
}

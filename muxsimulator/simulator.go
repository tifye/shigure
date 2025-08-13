package main

import (
	"io"
	"math/rand/v2"

	"github.com/charmbracelet/log"
	"github.com/tifye/shigure/mux"
)

const (
	maxIterations = 100_000
)

type Simulator struct {
	logger *log.Logger
	rnd    *rand.Rand
	seed1  uint64
	seed2  uint64

	userSimulator *userSimulator

	mux *mux.Mux
}

func NewSimulator(seed1, seed2 uint64, logger *log.Logger) *Simulator {

	rnd := rand.New(rand.NewPCG(seed1, seed2))
	mux := mux.NewMux(log.New(io.Discard))
	return &Simulator{
		logger:        logger,
		rnd:           rnd,
		seed1:         seed1,
		seed2:         seed2,
		userSimulator: newUserSimulator(logger, mux, rnd),
		mux:           mux,
	}
}

func (s *Simulator) Run() {
	s.logger.Info("Simulator started",
		"seed1", s.seed1, "seed2", s.seed2,
	)
	defer func() {
		s.logger.Info("Simulator finished",
			"seed1", s.seed1, "seed2", s.seed2,
			"userSimulator", s.userSimulator,
		)
	}()

	for range maxIterations {
		s.Step()
	}
}

func (s *Simulator) Step() {
	s.userSimulator.Step()
}

package main

import (
	"flag"
	"math/rand/v2"
	"os"

	"github.com/charmbracelet/log"
)

var (
	seed1 uint64
	seed2 uint64
	debug bool
)

func main() {
	flag.Uint64Var(&seed1, "seed1", 0, "First seed value")
	flag.Uint64Var(&seed2, "seed2", 0, "Second seed value")
	flag.BoolVar(&debug, "debug", false, "Include debug logs")
	flag.Parse()

	wasSeed1Set := false
	wasSeed2Set := false

	flag.Visit(func(f *flag.Flag) {
		if f.Name == "seed1" {
			wasSeed1Set = true
		}
		if f.Name == "seed2" {
			wasSeed2Set = true
		}
	})

	if !wasSeed1Set {
		seed1 = rand.Uint64()
	}
	if !wasSeed2Set {
		seed2 = rand.Uint64()
	}

	logLevel := log.InfoLevel
	if debug {
		logLevel = log.DebugLevel
	}

	logger := log.NewWithOptions(os.Stderr, log.Options{
		Level:           logLevel,
		ReportTimestamp: false,
	})

	sim := NewSimulator(seed1, seed2, logger)
	sim.Run()
}

package main

import (
	"context"
	"flag"
	"math/rand/v2"
	"os"
	"os/signal"

	"github.com/charmbracelet/log"
)

var (
	seed1   uint64
	seed2   uint64
	debug   bool
	times   uint
	endless bool
)

func main() {
	flag.Uint64Var(&seed1, "seed1", 0, "First seed value")
	flag.Uint64Var(&seed2, "seed2", 0, "Second seed value")

	flag.UintVar(&times, "times", 0, "Amount of times to run the simulation each time with random seeds")
	flag.BoolVar(&endless, "endless", false, "Run the simulation an endless amount of times with random seeds until stopped")

	flag.BoolVar(&debug, "debug", false, "Include debug logs")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	logLevel := log.InfoLevel
	if debug {
		logLevel = log.DebugLevel
	}

	logger := log.NewWithOptions(os.Stderr, log.Options{
		Level:           logLevel,
		ReportTimestamp: false,
	})

	switch {
	case endless:
		runEndless(ctx, logger)
	case times > 0:
		runTimes(ctx, logger)
	default:
		runSeeded(ctx, logger)
	}
}

func runTimes(ctx context.Context, logger *log.Logger) {
	for range times {
		seed1 := rand.Uint64()
		seed2 := rand.Uint64()
		sim := NewSimulator(seed1, seed2, logger)
		sim.Run(ctx)

		if err := ctx.Err(); err != nil {
			logger.Error(err)
			return
		}
	}
}

func runEndless(ctx context.Context, logger *log.Logger) {
	for {
		seed1 := rand.Uint64()
		seed2 := rand.Uint64()
		sim := NewSimulator(seed1, seed2, logger)
		sim.Run(ctx)

		if err := ctx.Err(); err != nil {
			logger.Error(err)
			return
		}
	}
}

func runSeeded(ctx context.Context, logger *log.Logger) {
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

	sim := NewSimulator(seed1, seed2, logger)
	sim.Run(ctx)

	if err := ctx.Err(); err != nil {
		logger.Error(err)
	}
}

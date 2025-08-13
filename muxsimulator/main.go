package main

import (
	"flag"
	"math/rand/v2"
	"os"
)

var (
	seed1 uint64
	seed2 uint64
)

func main() {
	flag.Uint64Var(&seed1, "seed1", 0, "First seed value")
	flag.Uint64Var(&seed2, "seed2", 0, "Second seed value")
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

	sim := NewSimulator(seed1, seed2, os.Stdout)
	sim.Run()
}

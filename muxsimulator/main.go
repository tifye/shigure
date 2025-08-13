package main

import (
	"flag"
	"fmt"
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

	fmt.Println(seed1, seed2)

	sim := NewSimulator(rand.Uint64(), rand.Uint64(), os.Stdout)
	sim.Run()
}

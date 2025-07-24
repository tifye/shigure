package stream

import (
	"math/rand/v2"
	"testing"
)

func TestRunSimulator(t *testing.T) {
	sim := NewSimulator(rand.Uint64(), rand.Uint64())
	sim.Run()
}

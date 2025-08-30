package main

import "math/rand/v2"

func Chance(rnd *rand.Rand, probability uint) bool {
	return rnd.UintN(100) < probability
}

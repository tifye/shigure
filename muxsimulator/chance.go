package main

import "math/rand/v2"

var (
	probabilityRange uint = 100
)

func Chance(rnd *rand.Rand, probability uint) bool {
	return rnd.UintN(probabilityRange) < probability
}

package main

var (
	probabilityRange uint = 100
)

func (s *Simulator) Chance(probability uint) bool {
	return probability == s.rnd.UintN(probabilityRange)
}

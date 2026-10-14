package agent

import (
	"math"
	"math/bits"
)

// rateFromCounters never treats a rollback as newly transferred bytes. A fresh
// baseline is recorded by the caller. Saturation lets existing anomaly gates
// reject unrepresentable rates without wrapping multiplication.
func rateFromCounters(current, previous, ms uint64) uint64 {
	if ms == 0 || current < previous {
		return 0
	}
	hi, lo := bits.Mul64(current-previous, 1000)
	if hi >= ms {
		return math.MaxUint64
	}
	rate, _ := bits.Div64(hi, lo, ms)
	return rate
}

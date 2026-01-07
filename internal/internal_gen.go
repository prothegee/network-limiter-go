package internal_gen

import (
	"log"
	"math/rand"
)

type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// @brief generate generic random signed number
//
// @note safe for seed concurency
func RandomNumberSign[T Number](r *rand.Rand, minRange, maxRange T) T {
	if minRange > maxRange {
		log.Printf("min range is larger than max range, swaping\n")
		minRange, maxRange = maxRange, minRange
	}

	minVal := int64(minRange)
	maxVal := int64(maxRange)

	// safeguard even after swap
	if minVal > maxVal {
		return minRange // dear generic, could it happen?
	}

	n := minVal + r.Int63n(maxVal-minVal+1)

	return T(n) // valid T
}


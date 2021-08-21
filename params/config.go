package params

import "errors"

var (
	RoToTickRation         = 100000000
	InitialBlockSubsidy    = 500 * uint64(RoToTickRation)
	SubsidyHalvingInterval = 1000

	MaxValidAmount = 100000000 * uint64(RoToTickRation)
)

var ErrInvalidValue = errors.New("invalid value")

func ValueIsValid(val uint64) bool {
	return val <= MaxValidAmount
}

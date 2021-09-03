package params

import "errors"

var (
	RoToTickRation         = 100000000
	InitialBlockSubsidy    = 500 * uint64(RoToTickRation)
	SubsidyHalvingInterval uint32 = 1000

	MaxValidAmount = 100000000 * uint64(RoToTickRation)

	MaxNumberOfTXsInBlock = 1000

	BitsSize = 4

	// FeePerByte means 1 tick per byte is used as a fee, used as placeholder for now
	FeePerByte uint64 = 1
)

var ErrInvalidValue = errors.New("invalid value")

func ValueIsValid(val uint64) bool {
	return val <= MaxValidAmount
}

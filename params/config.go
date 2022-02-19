package params

import "errors"

var (
	RoToTickRation                = 100000000
	InitialBlockSubsidy           = 500 * uint64(RoToTickRation)
	SubsidyHalvingInterval uint32 = 1000

	MaxValidAmount = 100000000 * uint64(RoToTickRation)

	MaxNumberOfTXsInBlock = 1000

	BitsSize                  int    = 4
	RetargetInterval          uint32 = 2016
	ExpectedTimePerBlockInSec uint64 = 2 * 60 // 2 minutes
	MaxDifficulty             uint32 = 0x18ffffff

	// FeePerByte means 1 tick per byte is used as a fee, used as placeholder for now
	FeePerByte uint64 = 1

	StoragePath            = "/.plairo/blocks/storage"
	UndoStoragePath        = "/.plairo/blocks/undo"
	MaxBlockFileSize int32 = 134217728 // 128Mb in bytes

	// BlockMagicBytes are the same as bitcoin
	BlockMagicBytes = []byte{0xf9, 0xbe, 0xb4, 0xd9}
)

var ErrInvalidValue = errors.New("invalid value")

func ValueIsValid(val uint64) bool {
	return val <= MaxValidAmount
}

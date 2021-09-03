package coin

import "plairo/params"

func GetTotalFeeInTicks(data []byte) uint64 {
	return params.FeePerByte * uint64(len(data))
}

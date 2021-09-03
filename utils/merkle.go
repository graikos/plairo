package utils

func ComputeMerkleRoot(hashSlice [][]byte) []byte {
	if len(hashSlice) == 1 {
		return hashSlice[0]
	}

	if len(hashSlice)%2 == 1 {
		// using the bitcoin merkle implementation, so if number of hashes in a level is odd, the last
		// one is duplicated and appended to the end
		hashSlice = append(hashSlice, hashSlice[len(hashSlice)])
	}
	var tempHashes [][]byte
	i := 0
	for ; i < len(hashSlice); i+=2 {
		// concatenating, double hashing together and appending to the next level hashes
		tempHashes = append(tempHashes, CalculateSHA256Hash(CalculateSHA256Hash(append(hashSlice[i], hashSlice[i+1]...))))
	}
	return ComputeMerkleRoot(tempHashes)
}

package utils

import "crypto/sha256"

func CalculateSHA256Hash(msg []byte) []byte {
	h := sha256.Sum256(msg)
	return h[:]
}

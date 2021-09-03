package utils

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func newBitsTestCase(bitstring, expstring string) *bitsTcase {
	bits, _ := hex.DecodeString(bitstring)
	expected, _ := hex.DecodeString(expstring)
	return &bitsTcase{bits, expected}
}

type bitsTcase struct {
	bits []byte
	expected []byte
}

func TestExpandBits(t *testing.T) {
	tcases := []*bitsTcase{
		newBitsTestCase("00000000", "0000000000000000000000000000000000000000000000000000000000000000"),
		newBitsTestCase("04000000", "0000000000000000000000000000000000000000000000000000000000000000"),
		newBitsTestCase("04aaaaaa", "00000000000000000000000000000000000000000000000000000000aaaaaa00"),
		newBitsTestCase("03aaaaaa", "0000000000000000000000000000000000000000000000000000000000aaaaaa"),
		newBitsTestCase("20aaaaaa", "aaaaaa0000000000000000000000000000000000000000000000000000000000"),
		newBitsTestCase("180696f4", "00000000000000000696f4000000000000000000000000000000000000000000"),
	}

	for _, tcase := range tcases {
		if !bytes.Equal(tcase.expected, ExpandBits(tcase.bits)) {
			t.Errorf("Error converting %x\n Exp: %x\n Got: %x\n", tcase.bits, tcase.expected, ExpandBits(tcase.bits))
		}
	}
}

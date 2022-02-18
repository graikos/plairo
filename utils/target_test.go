package utils

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
)

func newBitsTestCase(bitstring, expstring string) *bitsTcase {
	bits, _ := hex.DecodeString(bitstring)
	expected, _ := hex.DecodeString(expstring)
	return &bitsTcase{bits, expected}
}

type bitsTcase struct {
	bits     []byte
	expected []byte
}

func TestExpandBits(t *testing.T) {
	tcases := []*bitsTcase{
		newBitsTestCase("00000000", "0000000000000000000000000000000000000000000000000000000000000000"),
		newBitsTestCase("02000010", "0000000000000000000000000000000000000000000000000000000000000000"),
		newBitsTestCase("01100000", "0000000000000000000000000000000000000000000000000000000000000010"),
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

type targetTcase struct {
	coeff        float64
	prevTarget   uint32
	expNewTarget uint32
}

func TestApplyCoeffToTarget(t *testing.T) {
	cases := []*targetTcase{
		{1, 1, 1},
		{2, 1, 2},
		{4, 1, 4},
		{10, 1, 4},
		{0.25, 1, 0},
		{0.01, 1, 0},
		{1, 0x01100000, 0x01100000},
		{0.5, 0x01100000, 0x01080000},
		{0.25, 4, 1},
		{1.5, 4, 6},
		{1, 0x12121212, 0x12121212},
		{1, 0x12121212, 0x12121212},
		{1, 0x12121212, 0x12121212},
		{1, 0x05010000, 0x05010000},
		{0.5, 0x04010000, 0x04008000},
		{0.5, 0x04000001, 0x03000080},
		{2, 0x03800000, 0x04010000},
		{2, 0x03800000, 0x04010000},
		{10, 0x03800000, 0x04020000},
	}
	for i, c := range cases {
		fmt.Printf("Checking case #%d \n", i)
		if c.expNewTarget != ApplyCoeffToTarget(c.coeff, c.prevTarget) {
			t.Errorf("Invalid new target for case #%d\n", i)
		}
	}
}

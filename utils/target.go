package utils

import (
	"fmt"
	"plairo/params"
)

func ExpandBits(bits []byte) []byte {
	/*
		Bits size is currently set to be 4 bytes.
		First byte is the exponent, meaning the size of the target in bytes
		The other three bytes are the coefficient, meaning the first three bytes of the target.
		The above is padded with leading zeroes to reach 32 bytes in size, since a comparison
		with a SHA-256 hash will be needed.
		Example: 0x04aabbcc
		Exponent is 0x04
		Coefficient is 0xaabbcc
		Result should be 0x00000000000000000000000000000000000000000000000000000000aabbcc00
	*/
	res := make([]byte, 32)
	exp := int(bits[0])
	copy(res[32-exp:], bits[1:])
	return res
}

func ApplyCoeffToTarget(coeff float64, prevTarget uint32) uint32 {
	// limiting coefficient to limit the effect of a single retarget in block difficulty
	if coeff > 4 {
		coeff = 4
	} else if coeff < 0.25 {
		coeff = 0.25
	}
	fmt.Printf("Prev target is: %d\n", prevTarget)
	// prevTarget is in compressed form (0xEEXXYYZZ). To apply the coefficient, the exponent is not needed for now.
	// using a mask to drop the exponent
	rawBits := uint64(prevTarget) & 0x00ffffff
	fmt.Printf("Raw bits is: %d\n", rawBits)
	bitsExp := prevTarget & 0xff000000
	fmt.Printf("Exp is: %d\n", bitsExp)
	// shifting exponent bits to increment/decrement if needed
	bitsExp >>= 24
	fmt.Printf("Now exp is (after shifting): %d\n", bitsExp)
	// The full target has a size of 32 bytes. For arithmetic calculations with operands of this size,
	// precision of 256 bits would be needed ideally.
	// We can avoid this by taking into account the limiting factor of 4.
	// The coefficient which will be applied to the previous target cannot exceed 4 or be less than 0.25.
	// This means that the previous target (in binary) will be shifted left or right by no more than 4 bits.
	// To make sure no bits are missed when applying a coefficient < 1, we can shift left by a byte.
	rawBits <<= 8
	fmt.Printf("Now rawbits is (after shifting): %d\n", rawBits)
	// now the rawBits form is 0x00000000XXYYZZ00
	// applying the coefficient and keeping the integer part of the result
	newTarget := uint64(float64(rawBits) * coeff)
	fmt.Printf("New target is: %d\n", newTarget)
	// checking "overflow" and "underflow" bytes
	// overflow is of higher priority
	if (newTarget & 0xff00000000) != 0 {
		fmt.Printf("Overflow\n")
		// if overflow is not zero, only the first 3 bytes should be kept
		// exponent is therefore incremented and the last 2 bytes dropped
		bitsExp++
		newTarget >>= 16
	} else if (newTarget & 0xff) != 0 {
		// no overflow was found, but the last byte is non-zero
		// need to check if it's necessary or just used for precision
		if (newTarget & 0xff000000) != 0 {
			fmt.Printf("Last byte was used for precision\n")
			// last byte was only used for precision, so exponent stays the same
			// last byte is dropped
			newTarget >>= 8
		} else {
			fmt.Printf("Last byte was necessary\n")
			// last byte was necessary, decrementing the exponent by one
			if bitsExp != 0 {
				bitsExp--
			} else {
				// exponent cannot decrement, last byte should be dropped
				newTarget >>= 8
			}
			fmt.Printf("Exp is now: %d\n", bitsExp)
			// no need to shift newTarget, it's already in the desired form
		}
	} else {
		// bits remained in original form, shifting back into place
		newTarget >>= 8
	}
	// shifting exponent back to the position it should be
	bitsExp <<= 24
	// adding the exponent in front of the bits coefficient
	newTarget = uint64(bitsExp) | newTarget
	if uint32(newTarget) > params.MaxDifficulty {
		return params.MaxDifficulty
	}
	fmt.Printf("Result is: %x\n", newTarget)
	return uint32(newTarget)
}

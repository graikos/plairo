package utils

import (
	"encoding/binary"
	"math"
)

func SerializeUint16(num uint16, littleEndian bool) []byte {
	b := make([]byte, 2)
	if littleEndian {
		binary.LittleEndian.PutUint16(b, num)
	} else {
		binary.BigEndian.PutUint16(b, num)
	}
	return b
}

func SerializeUint32(num uint32, littleEndian bool) []byte {
	b := make([]byte, 4)
	if littleEndian {
		binary.LittleEndian.PutUint32(b, num)
	} else {
		binary.BigEndian.PutUint32(b, num)
	}
	return b
}

func SerializeUint64(num uint64, littleEndian bool) []byte {
	b := make([]byte, 8)
	if littleEndian {
		binary.LittleEndian.PutUint64(b, num)
	} else {
		binary.BigEndian.PutUint64(b, num)
	}
	return b
}


// SerializeToOneHot converts a bool slice to one-hot representation (resembles big endian)
func SerializeToOneHot(data []bool) []byte {
	/*
	 * Example:
	 * If the data is {true, true, false, true}, we expect a slice of 1 byte, which will be 0000 1011
	 */

	// len and cap will be number of bytes required
	noOfBytes := int(math.Ceil(float64(len(data)) / 8))
	res := make([]byte, noOfBytes)

	var tempb byte = 0x00
	packCounter := 7
	pos := noOfBytes - 1
	for _, val := range data {
		if val {
			tempb += byte(0x01) << (7 - packCounter)
		}
		if packCounter == 0 {
			packCounter = 7
			res[pos] = tempb
			pos--
			tempb = 0x00
			continue
		}
		packCounter--
	}
	// adding the last byte
	if tempb != 0x00 {
		res[pos] = tempb
	}
	return res
}

func DeserializeUint16(serial []byte, littleEndian bool) uint16 {
	if littleEndian {
		return binary.LittleEndian.Uint16(serial)
	}
	return binary.BigEndian.Uint16(serial)
}
func DeserializeUint32(serial []byte, littleEndian bool) uint32 {
	if littleEndian {
		return binary.LittleEndian.Uint32(serial)
	}
	return binary.BigEndian.Uint32(serial)
}
func DeserializeUint64(serial []byte, littleEndian bool) uint64 {
	if littleEndian {
		return binary.LittleEndian.Uint64(serial)
	}
	return binary.BigEndian.Uint64(serial)
}

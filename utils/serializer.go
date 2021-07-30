package utils

import "encoding/binary"

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

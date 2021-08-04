package utils

import (
	"bytes"
	"testing"
)

func TestSerializeToOneHot(t *testing.T) {
	data := []bool{true, false, false, true, false, true, true, false, true, true, true}
	res := SerializeToOneHot(data)
	if !bytes.Equal(res, []byte{0x07, 0x69}) {
		t.Error("Expected one hot to be 0x07, 0x69")
	}
	data = []bool{false}
	res = SerializeToOneHot(data)
	if !bytes.Equal(res, []byte{0x00}) {
		t.Error("Expected one hot to be 0x00")
	}
	data = []bool{true, true, true, true, true, true, true, true}
	res = SerializeToOneHot(data)
	if !bytes.Equal(res, []byte{0xff}) {
		t.Errorf("Expected one hot to be 0xff, got %x", res)
	}
}

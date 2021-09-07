package utils

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestComputeMerkleRoot(t *testing.T) {
	a, _ := hex.DecodeString("4bf5122f344554c53bde2ebb8cd2b7e3d1600ad631c385a5d7cce23c7785459a")
	if !bytes.Equal(ComputeMerkleRoot([][]byte{a}), a) {
		t.Errorf("Invalid merkle root:\nExp: %x\nGot: %x\n", a, ComputeMerkleRoot([][]byte{a}))
	}
	a = []byte{0x01}
	b := []byte{0x02}
	exp, _ := hex.DecodeString("76a56aced915d2513dcd84c2c378b2e8aa5cd632b5b71ca2f2ac5b0e3a649bdb")
	if !bytes.Equal(ComputeMerkleRoot([][]byte{a, b}), exp) {
		t.Errorf("Invalid merkle root:\nExp: %x\nGot: %x\n", exp, ComputeMerkleRoot([][]byte{a, b}))
	}
	c := []byte{0x03}
	exp, _ = hex.DecodeString("6e42ac8d2f8c2c68e1fbbbd3e4c31191ced29abd73cd79e0c79b7139af460557")
	if !bytes.Equal(ComputeMerkleRoot([][]byte{a, b, c}), exp) {
		t.Errorf("Invalid merkle root:\nExp: %x\nGot: %x\n", exp, ComputeMerkleRoot([][]byte{a, b, c}))
	}
	d := []byte{0x04}
	e := []byte{0x05}
	f := []byte{0x06}
	exp, _ = hex.DecodeString("3be16b93025e6836910dd66fb9ebb0badc9bc9d3aa9322435b364a6617e1dff9")
	if !bytes.Equal(ComputeMerkleRoot([][]byte{a, b, c, d, e, f}), exp) {
		t.Errorf("Invalid merkle root:\nExp: %x\nGot: %x\n", exp, ComputeMerkleRoot([][]byte{a, b, c, d, e, f}))
	}
}

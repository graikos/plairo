package core

import (
	"bytes"
	"plairo/utils"
	"testing"
)

func TestGenerateOutputID(t *testing.T) {
	tOut := NewTransactionOutput([]byte("tst"), 1, 66, []byte{0x0})
	expectedArray := []byte{0x74, 0x73, 0x74, 0x00, 0x00, 0x00, 0x01}
	expected := utils.CalculateSHA256Hash(utils.CalculateSHA256Hash(expectedArray))
	if bytes.Compare(expected, tOut.OutputID) != 0{
		t.Error("OutputID not valid.")
	}
}

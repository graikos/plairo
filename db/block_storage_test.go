package db

import (
	"bytes"
	"os"
	"path/filepath"
	"plairo/params"
	"testing"
)

var testBlockStoragePath string

type testBlock struct {
	blockHash  []byte
	serialData []byte
}
type IBlock interface {
	GetBlockHash() []byte
	Serialize() []byte
	GetNoOfTx() int
	GetBlockHeader() []byte
	GetUndoData() ([]byte, []byte)
}

func (tb *testBlock) Serialize() []byte {
	return tb.serialData
}

func (tb *testBlock) GetBlockHash() []byte {
	return tb.blockHash
}

func (tb *testBlock) GetNoOfTx() int {
	return 0
}

func (tb *testBlock) GetBlockHeader() []byte {
	return []byte("header")
}

func (tb *testBlock) GetUndoData() ([]byte, []byte) {
	return []byte("undo"), []byte("checksum")
}

func (tb *testBlock) GetExpData() []byte {
	return append(params.BlockMagicBytes, tb.Serialize()...)
}

func testBlockStorageSetup() []*testBlock {
	homedir, _ := os.UserHomeDir()
	testBlockStoragePath = filepath.Join(homedir, "/.plairo/blockstoragetest")

	tcases := []*testBlock{
		{[]byte("aa"), []byte{0xd, 0xa, 0xd, 0xa}},
		{[]byte("ab"), []byte{}},
	}
	tcases[1].serialData = make([]byte, 102500)
	tcases[1].serialData[len(tcases[1].serialData)-1] = 0x99

	return tcases
}

func TestBlockStorage_WriteBlock_GetBlockData(t *testing.T) {
	cases := testBlockStorageSetup()
	bs := NewBlockStorage(testBlockStoragePath, true)
	for i, tcase := range cases {
		if err := bs.WriteBlock(tcase, 1); err != nil {
			t.Errorf("Error writing block #%d\n", i)
		}
		got, err := bs.GetBlockData(tcase.GetBlockHash())
		if !err {
			t.Errorf("Error getting data block #%d\n", i)
		}
		if !bytes.Equal(got, tcase.GetExpData()) {
			t.Errorf("Got unexpected data for block #%d\n", i)
		}

	}
}

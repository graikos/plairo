package db

import (
	"os"
	"path/filepath"
	"plairo/core"
	"plairo/utils"
)

type BlockIndex struct {
	*DBwrapper
}

var BlockIndexPath string

func init() {
	homedir, _ := os.UserHomeDir()
	BlockIndexPath = filepath.Join(homedir, "/.plairo/blocks/index")
}

func NewBlockIndex(dbpath string, isObfuscated bool) *BlockIndex {
	return &BlockIndex{NewDBwrapper(dbpath, isObfuscated)}
}

func (bi *BlockIndex) InsertBlockIndexRecord(block core.IBlock, blockHeight uint32) error {
	/*
		Block index record structure:
		-- Block Header
		-- Block Height
		-- Number of Transactions
	*/
	res := make([]byte, 0, 32) // at least 32 bytes will be needed
	res = append(res, block.GetBlockHeader()...)
	res = append(res, utils.SerializeUint32(blockHeight, false)...)
	res = append(res, utils.SerializeUint32(uint32(block.GetNoOfTx()), false)...)
	return bi.Insert(buildKey(BlockIndexKey, block.GetBlockHash()), res)
}

func (bi *BlockIndex) InsertTXIndexRecord(txid []byte, txOffsetInBlock uint32, batchMode bool) error {
	/*
		Transaction Index record structure:
		-- Offset of transaction inside the block data (4 bytes)
	*/
	res := utils.SerializeUint32(txOffsetInBlock, false)
	if batchMode {
		bi.PutInBatch(buildKey(TxIndexKey, txid), res)
		return nil
	}
	return bi.Insert(buildKey(TxIndexKey, txid), res)
}

func (bi *BlockIndex) WriteBatchBI() error {
	return bi.WriteBatch()
}

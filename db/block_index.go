package db

import (
	"os"
	"path/filepath"
	"plairo/core"
	"plairo/utils"
)

type BlockIndex struct {
	dbwrapper *DBwrapper
}

var BlockIndexPath string

func init() {
	homedir, _ := os.UserHomeDir()
	BlockIndexPath = filepath.Join(homedir, "/.plairo/blocks/index")
}

func NewBlockIndex(dbpath string, isObfuscated bool) *BlockIndex {
	return &BlockIndex{NewDBwrapper(dbpath, isObfuscated)}
}

func (bi *BlockIndex) InsertBlockIndexRecord(block *core.Block, fileIndex, posInFile, blockHeight uint32) error {
	/*
		Block index record structure:
		-- Block Header
		-- Block Height
		-- Number of Transactions
		-- Position of block data in storage files and file index
		-- TBA: Undo records location
	*/
	res := make([]byte, 0, 64) // at least 32 bytes will be needed
	res = append(res, block.GetBlockHeader()...)
	res = append(res, utils.SerializeUint32(blockHeight, false)...)
	res = append(res, utils.SerializeUint32(uint32(len(block.AllBlockTx)), false)...)
	res = append(res, utils.SerializeUint32(fileIndex, false)...)
	res = append(res, utils.SerializeUint32(posInFile, false)...)

	return bi.dbwrapper.Insert(buildKey(BlockIndexKey, block.GetBlockHash()), res)
}

func (bi *BlockIndex) InsertFileInfoRecord(fileIndex, noOfBlocks, sizeOfPlr, lowestPlr, highestPlr uint32) error {
	/*
		File record structure:
		-- Number of blocks stored in this file (4 bytes)
		-- Size of the plr file with this file index (4 bytes)
		-- TBA: size of the undo file for this file index
		-- The lowest block height stored in the file with this file index (4 bytes)
		-- The highest block height stored in the file with this file index (4 bytes)
		-- TBA: lowesst and highest heights of undo file
	*/
	res := make([]byte, 0, 16)
	res = append(res, utils.SerializeUint32(noOfBlocks, false)...)
	res = append(res, utils.SerializeUint32(sizeOfPlr, false)...)
	res = append(res, utils.SerializeUint32(lowestPlr, false)...)
	res = append(res, utils.SerializeUint32(highestPlr, false)...)

	return bi.dbwrapper.Insert(buildKey(FileInfoKey, utils.SerializeUint32(fileIndex, false)), res)
}

func (bi *BlockIndex) InsertLastBlockFileIdx(fileIndex uint32) error {
	// Saving with key 'I' the last file index used (4 bytes)
	return bi.dbwrapper.Insert([]byte{byte(LastFileInd)}, utils.SerializeUint32(fileIndex, false))
}

func (bi *BlockIndex) InsertTXIndexRecord(txid []byte, fileIndex, blockOffset, txOffsetInBlock uint32) error {
	/*
		Transaction Index record structure:
		-- File Index of plr file which contains the block of the transaction (4 bytes)
		-- Offset of the block in the file (4 bytes)
		-- Offset of transaction inside the block data (4 bytes)
	*/
	res := make([]byte, 0, 12)
	res = append(res, utils.SerializeUint32(fileIndex, false)...)
	res = append(res, utils.SerializeUint32(blockOffset, false)...)
	res = append(res, utils.SerializeUint32(txOffsetInBlock, false)...)

	return bi.dbwrapper.Insert(buildKey(TxIndexKey, txid), res)
}

func (bi *BlockIndex) Close() {
	bi.dbwrapper.Close()
}

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

type FileInfoRecord struct {
	fileIndex  uint32
	noOfBlocks uint32
	sizeOfPlr  uint32
	sizeOfUndo uint32
	lowestPlr  uint32
	highestPlr uint32
}

func (f *FileInfoRecord) SizeOfPlr() uint32 {
	return f.sizeOfPlr
}

func (f *FileInfoRecord) SizeOfUndo() uint32 {
	return f.sizeOfUndo
}

func (f *FileInfoRecord) NoOfBlocks() uint32 {
	return f.noOfBlocks
}

func (f *FileInfoRecord) LowestPlr() uint32 {
	return f.lowestPlr
}

func (f *FileInfoRecord) HighestPlr() uint32 {
	return f.highestPlr
}

func NewBlockIndex(dbpath string, isObfuscated bool) *BlockIndex {
	return &BlockIndex{NewDBwrapper(dbpath, isObfuscated)}
}

func (bi *BlockIndex) InsertBlockIndexRecord(block *core.Block, fileIndex, posInFile, blockHeight, undoFileIndex, undoPosInFile uint32) error {
	/*
		Block index record structure:
		-- Block Header
		-- Block Height
		-- Number of Transactions
		-- File index of block data in plr file
		-- Position of block data inside this file
		-- File index of block undo data in rev file
		-- Position of block undo data in this file
	*/
	res := make([]byte, 0, 64) // at least 32 bytes will be needed
	res = append(res, block.GetBlockHeader()...)
	res = append(res, utils.SerializeUint32(blockHeight, false)...)
	res = append(res, utils.SerializeUint32(uint32(len(block.AllBlockTx())), false)...)
	res = append(res, utils.SerializeUint32(fileIndex, false)...)
	res = append(res, utils.SerializeUint32(posInFile, false)...)
	res = append(res, utils.SerializeUint32(undoFileIndex, false)...)
	res = append(res, utils.SerializeUint32(undoPosInFile, false)...)

	return bi.Insert(buildKey(BlockIndexKey, block.GetBlockHash()), res)
}

func (bi *BlockIndex) InsertFileInfoRecord(fileIndex, noOfBlocks, sizeOfPlr, sizeOfUndo, lowestPlr, highestPlr uint32) error {
	/*
		File record structure:
		-- Number of blocks stored in this file (4 bytes)
		-- Size of the plr file with this file index (4 bytes)
		-- TBA: size of the undo file for this file index
		-- The lowest block height stored in the file with this file index (4 bytes)
		-- The highest block height stored in the file with this file index (4 bytes)
	*/
	res := make([]byte, 0, 16)
	res = append(res, utils.SerializeUint32(noOfBlocks, false)...)
	res = append(res, utils.SerializeUint32(sizeOfPlr, false)...)
	res = append(res, utils.SerializeUint32(sizeOfUndo, false)...)
	res = append(res, utils.SerializeUint32(lowestPlr, false)...)
	res = append(res, utils.SerializeUint32(highestPlr, false)...)

	return bi.Insert(buildKey(FileInfoKey, utils.SerializeUint32(fileIndex, false)), res)
}

func (bi *BlockIndex) GetFileInfoRecord(fileIndex uint32) (*FileInfoRecord, error) {
	data, err := bi.Get(buildKey(FileInfoKey, utils.SerializeUint32(fileIndex, false)))
	if err != nil {
		return nil, err
	}
	// TBA: Undo files index modifications
	caret := 0
	noOfBlocks := utils.DeserializeUint32(data[caret:caret+4], false)
	caret += 4
	sizeOfPlr := utils.DeserializeUint32(data[caret:caret+4], false)
	caret += 4
	sizeOfUndo := utils.DeserializeUint32(data[caret:caret+4], false)
	caret += 4
	low := utils.DeserializeUint32(data[caret:caret+4], false)
	caret += 4
	high := utils.DeserializeUint32(data[caret:caret+4], false)
	return &FileInfoRecord{noOfBlocks: noOfBlocks, sizeOfPlr: sizeOfPlr, sizeOfUndo: sizeOfUndo, lowestPlr: low, highestPlr: high}, nil
}

func (bi *BlockIndex) InsertLastBlockFileIdx(fileIndex uint32) error {
	// Saving with key 'I' the last file index used (4 bytes)
	return bi.Insert([]byte{byte(LastFileInd)}, utils.SerializeUint32(fileIndex, false))
}

func (bi *BlockIndex) GetLastBlockFileIdx() (uint32, error) {
	res, err := bi.Get([]byte{byte(LastFileInd)})
	if err != nil {
		return 0, err
	}
	return utils.DeserializeUint32(res, false), nil
}

func (bi *BlockIndex) InsertLastUndoFileIdx(fileIndex uint32) error {
	// Saving with key 'U' the last file index used to store undo records (4 bytes)
	return bi.Insert([]byte{byte(LastUndoFileInd)}, utils.SerializeUint32(fileIndex, false))
}

func (bi *BlockIndex) GetLastUndoFileIdx() (uint32, error) {
	res, err := bi.Get([]byte{byte(LastUndoFileInd)})
	if err != nil {
		return 0, err
	}
	return utils.DeserializeUint32(res, false), nil
}

func (bi *BlockIndex) InsertTXIndexRecord(txid []byte, fileIndex, blockOffset, txOffsetInBlock uint32, batchMode bool) error {
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

	if batchMode {
		bi.PutInBatch(buildKey(TxIndexKey, txid), res)
		return nil
	}
	return bi.Insert(buildKey(TxIndexKey, txid), res)
}

func (bi *BlockIndex) WriteBatch() error {
	return bi.WriteBatch()
}

func (bi *BlockIndex) Close() {
	bi.DBwrapper.Close()
}

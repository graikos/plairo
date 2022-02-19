package db

import (
	"os"
	"path/filepath"
	"plairo/core"
	"plairo/params"
)

type BlockStorage struct {
	*DBwrapper
	maxPageSize int
}

var BlockStoragePath string
var UndoStoragePath string

func init() {
	homedir, _ := os.UserHomeDir()
	BlockStoragePath = filepath.Join(homedir, params.StoragePath)
	UndoStoragePath = filepath.Join(homedir, params.UndoStoragePath)
}

func NewBlockStorage(dbpath string, isObfuscated bool) *BlockStorage {
	// using max size of 100kb
	return &BlockStorage{NewDBwrapper(dbpath, isObfuscated), 102400}
}

func (bs *BlockStorage) WriteBlock(block core.IBlock, height uint32) error {
	/*
	 Block key structure:
	 -- Block hash (32 bytes)
	 -- Record Number (1 byte)
	 Block storage record structure:
	 -- isContinued (1 byte)
	 -- Magic bytes (first record) (4 bytes)
	 -- Block data
	*/
	key := block.GetBlockHash()

	data := params.BlockMagicBytes
	data = append(data, block.Serialize()...)

	// storing block data
	if err := bs.PageInsert(key, data, bs.maxPageSize); err != nil {
		return err
	}
	// writing undo record
	uw := newUndoStorage(UndoStoragePath, true)
	defer uw.Close()
	if err := uw.writeUndo(block); err != nil {
		return err
	}
	// inserting block index record
	bi := NewBlockIndex(BlockIndexPath, true)
	defer bi.Close()
	if err := bi.InsertBlockIndexRecord(block, height); err != nil {
		return err
	}

	return nil
}

func (bs *BlockStorage) GetBlockData(bkey []byte) ([]byte, bool) {
	return bs.PageGet(bkey, bs.maxPageSize)
}

func (bs *BlockStorage) GetUndoData(bkey []byte) ([]byte, bool) {
	uw := newUndoStorage(UndoStoragePath, true)
	defer uw.Close()
	return uw.GetUndoData(bkey)
}

func (bs *BlockStorage) Close() {
	bs.DBwrapper.Close()
}

type undoStorage struct {
	*DBwrapper
	maxPageSize int
}

func newUndoStorage(dbpath string, isObfuscated bool) *undoStorage {
	// using max size of 100kb
	return &undoStorage{NewDBwrapper(dbpath, isObfuscated), 102400}
}

func (us *undoStorage) writeUndo(block core.IBlock) error {
	/*
		Structure of undo record:
			Magic Bytes (4 bytes)
			Size of Block Undo record (8 bytes)
			Block Undo record
			Double-SHA256 checksum for Block Undo record (32 bytes)
	*/
	key := block.GetBlockHash()
	undodata := params.BlockMagicBytes
	blockUndo, checksum := block.GetUndoData()
	undodata = append(undodata, blockUndo...)
	undodata = append(undodata, checksum...)

	return us.PageInsert(key, undodata, us.maxPageSize)
}

func (us *undoStorage) GetUndoData(bkey []byte) ([]byte, bool) {
	return us.PageGet(bkey, us.maxPageSize)
}

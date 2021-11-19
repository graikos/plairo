package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"plairo/params"
	"plairo/utils"

	"github.com/syndtr/goleveldb/leveldb"
)

type CState interface {
	GetTXHeight(txid []byte) uint32
}

var cstate CState

type UndoWriter struct {
	FileIndex uint32
	path      string
	maxSize   int32
	remSize   int32
}

func newUndoWriter() *UndoWriter {
	homeDir, _ := os.UserHomeDir()
	// getting the latest file index
	lastFileIdx, err := blockIndex.GetLastUndoFileIdx()
	if errors.Is(err, leveldb.ErrNotFound) {
		// undo file index not found means this is the first time undo data is written
		if err := blockIndex.InsertLastUndoFileIdx(0); err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	}
	uMeta, err := blockIndex.GetFileInfoRecord(lastFileIdx)
	var remSize int32
	if err == nil {
		// file index record was found correctly, retrieving size
		remSize = params.MaxBlockFileSize - int32(uMeta.SizeOfUndo())
	} else if errors.Is(err, leveldb.ErrNotFound) {
		// file record not found, this is the first time writing
		remSize = params.MaxBlockFileSize
	} else {
		panic(err)
	}
	return &UndoWriter{FileIndex: lastFileIdx, path: filepath.Join(homeDir, params.StoragePath), maxSize: params.MaxBlockFileSize, remSize: remSize}
}

func (uw *UndoWriter) write(block iBlock, blockHeight uint32) (undoIdx, undoOffset uint32, err error) {
	/*
		Magic Bytes (4 bytes)
		Size of Block Undo record (8 bytes)
		Block Undo record
		Double-SHA256 checksum for Block Undo record (32 bytes)
	*/
	// building the undo data to be written
	var undoData []byte
	undoData = append(undoData, params.BlockMagicBytes...)
	blockUndo, undoChksum := block.GetUndoData()
	undoData = append(undoData, utils.SerializeUint64(uint64(len(blockUndo)), false)...)
	undoData = append(undoData, blockUndo...)
	undoData = append(undoData, undoChksum...)

	// checking if current remaining size is enough for new undo write
	if int32(len(undoData)) > uw.remSize {
		// checking if a file record already exists because of block files
		// reading its contents to avoid an overwrite
		currFR, err := blockIndex.GetFileInfoRecord(uw.FileIndex)
		if err == nil {
			// re-writing existent plr data if the record already exists
			if err := blockIndex.InsertFileInfoRecord(uw.FileIndex, currFR.NoOfBlocks(), currFR.SizeOfPlr(), uint32(uw.maxSize-uw.remSize), currFR.LowestPlr(), currFR.HighestPlr()); err != nil {
				return 0, 0, err
			}
		} else if errors.Is(err, leveldb.ErrNotFound) {
			// creating a new record because of the undo write only
			if err := blockIndex.InsertFileInfoRecord(uw.FileIndex, 0, 0, uint32(uw.maxSize-uw.remSize), 0, 0); err != nil {
				return 0, 0, err
			}
		} else {
			// unexpected error occurred
			return 0, 0, err
		}

		uw.FileIndex++
		// updating last index for undo files
		if err := blockIndex.InsertLastUndoFileIdx(uw.FileIndex); err != nil {
			return 0, 0, nil
		}
		// resetting remaining size for undo writer
		uw.remSize = uw.maxSize
	}
	// formatting appropriately (e.g. 0000000001)
	fileIndexStr := fmt.Sprintf("%010d", uw.FileIndex)
	// creating directory if it doesn't exist
	_, err = os.Stat(uw.path)
	if os.IsNotExist(err) {
		if err := os.Mkdir(uw.path, 0755); err != nil {
			return 0, 0, err
		}
	} else if err != nil {
		return 0, 0, err
	}
	// opening the file in append mode
	f, err := os.OpenFile(filepath.Join(uw.path, "rev"+fileIndexStr+".dat"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	defer f.Close()

	n, err := f.Write(undoData)
	if err != nil {
		return 0, 0, err
	}

	// getting offset of block undo data inside current file
	undoOffset = uint32(uw.maxSize - uw.remSize)
	// updating remaining size of undo
	uw.remSize -= int32(n)

	// checking again if a file record already exists because of block files
	// reading its contents to avoid an overwrite
	currFR, err := blockIndex.GetFileInfoRecord(uw.FileIndex)
	if err == nil {
		// re-writing existent plr data if the record already exists
		if err := blockIndex.InsertFileInfoRecord(uw.FileIndex, currFR.NoOfBlocks(), currFR.SizeOfPlr(), uint32(uw.maxSize-uw.remSize), currFR.LowestPlr(), currFR.HighestPlr()); err != nil {
			return 0, 0, err
		}
	} else if errors.Is(err, leveldb.ErrNotFound) {
		// creating a new record because of the undo write only
		if err := blockIndex.InsertFileInfoRecord(uw.FileIndex, 0, 0, uint32(uw.maxSize-uw.remSize), 0, 0); err != nil {
			return 0, 0, err
		}
	} else {
		// unexpected error occurred
		return 0, 0, err
	}

	return uw.FileIndex, undoOffset, nil
}

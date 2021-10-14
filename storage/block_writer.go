package storage

import (
	"errors"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"os"
	"path/filepath"
	"plairo/params"
)

type BlockWriter struct {
	FileIndex         uint32
	path              string
	maxSize           int32
	remSize           int32
	currentNoOfBlocks uint32
	firstBlockHeight  uint32
}

type ITransation interface {
	Serialize() []byte
	GetTXID() []byte
}

type IBlock interface {
	Serialize() []byte
	AllBlockTx() []ITransation
}

type IBlockIndex interface {
	GetLastBlockFileIdx() (uint32, error)
	InsertLastBlockFileIdx(uint32) error
	InsertFileInfoRecord(fileIndex, noOfBlocks, sizeOfPlr, lowestPlr, highestPlr uint32) error
	InsertBlockIndexRecord(block IBlock, fileIndex, posInFile, blockHeight uint32) error
	InsertTXIndexRecord(txid []byte, fileIndex, blockOffset, txOffsetInBlock uint32, batchMode bool) error
	WriteBatch() error
}

var blockIndex IBlockIndex

func NewBlockWriter() *BlockWriter {
	homeDir, _ := os.UserHomeDir()
	lastFileIdx, err := blockIndex.GetLastBlockFileIdx()
	if errors.Is(err, leveldb.ErrNotFound) {
		if err := blockIndex.InsertLastBlockFileIdx(0); err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	}
	return &BlockWriter{FileIndex: lastFileIdx, path: filepath.Join(homeDir, params.StoragePath), maxSize: params.MaxBlockFileSize, remSize: params.MaxBlockFileSize}
}

func (bw *BlockWriter) Write(block IBlock, blockHeight uint32, indexTX bool) (n int, err error) {
	data := block.Serialize()
	// moving to new file if storage in current file is not sufficient
	if int32(len(data)) > bw.remSize {
		// saving current file metadata
		// assuming highest block height is the previous one
		err := blockIndex.InsertFileInfoRecord(bw.FileIndex, bw.currentNoOfBlocks, uint32(bw.maxSize-bw.remSize), bw.firstBlockHeight, blockHeight-1)
		if err != nil {
			return 0, err
		}
		// updating current file index
		bw.FileIndex++
		// updating block index record for file index
		if err := blockIndex.InsertLastBlockFileIdx(bw.FileIndex); err != nil {
			return 0, err
		}
		// updating writer properties
		bw.remSize = bw.maxSize
		bw.currentNoOfBlocks = 0
		// NOTE: if the write fails later, this will be inaccurate.
		// However, if the write fails, then programm will probably panic anyway.
		bw.firstBlockHeight = blockHeight
	}
	// formatting appropriately (e.g. 0000000001)
	fileIndexStr := fmt.Sprintf("%010d", bw.FileIndex)
	// creating directory if it doesn't exist
	_, err = os.Stat(bw.path)
	if os.IsNotExist(err) {
		if err := os.Mkdir(bw.path, 0755); err != nil {
			return 0, err
		}
	} else if err != nil {
		return 0, err
	}
	// opening the file in append mode
	f, err := os.OpenFile(filepath.Join(bw.path, "plr"+fileIndexStr+".dat"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	defer f.Close()

	n, err = f.Write(data)
	if err != nil {
		return n, err
	}

	blockOffset := bw.maxSize - bw.remSize
	// remaining size is reduced by n (number of bytes written)
	bw.remSize -= int32(n)
	bw.currentNoOfBlocks++
	// TODO: Update the file index record for the current file index?

	// updating block index for the latest block written
	if err := blockIndex.InsertBlockIndexRecord(block, bw.FileIndex, uint32(blockOffset), blockHeight); err != nil {
		return n, err
	}

	// if transaction indexing option is enabled, each transaction of the block will be added to the block index
	if indexTX {
		if err := bw.indexBlockTransactions(uint32(blockOffset), block); err != nil {
			return n, err
		}
	}

	return n, nil
}

func (bw *BlockWriter) indexBlockTransactions(blockOffset uint32, block IBlock) error {
	// Block header is 32 + 32 + 8 + 4 + 4 = 80 bytes
	// There are 4 magic bytes.
	// Number of Transactions is a 4-byte number.
	// using the carret to get the offset of the TX inside the block
	carret := 88 + blockOffset
	for _, tx := range block.AllBlockTx() {
		// inserting in batch is a safe operation
		// NOTE: The offset inside the block points to the start of the 4 bytes showing the length of TX data, not the
		// data itself.
		_ = blockIndex.InsertTXIndexRecord(tx.GetTXID(), bw.FileIndex, blockOffset, carret, true)
		// for each TX in the serialized block, there is a 4-byte number for the length of the following seriliazed
		// TX data. To move the carret correctly, this should be take into account.
		carret += uint32(len(tx.Serialize())) + 4
	}
	return blockIndex.WriteBatch()
}

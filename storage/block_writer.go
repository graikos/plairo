package storage

import (
	"errors"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"os"
	"path/filepath"
	"plairo/params"
	"plairo/utils"
)

type BlockWriter struct {
	FileIndex         uint32
	path              string
	maxSize           int32
	remSize           int32
	currentNoOfBlocks uint32
	firstBlockHeight  uint32
}

type iTransation interface {
	Serialize() []byte
	GetTXID() []byte
}

// TODO: Fix the returned slice problem
type iBlock interface {
	Serialize() []byte
	IterateBlockTx(chan<- interface{})
}
type iFileInfoRecord interface {
	SizeOfPlr() uint32
}

type iBlockIndex interface {
	GetLastBlockFileIdx() (uint32, error)
	GetFileInfoRecort(uint32) (iFileInfoRecord, error)
	InsertLastBlockFileIdx(uint32) error
	InsertFileInfoRecord(fileIndex, noOfBlocks, sizeOfPlr, lowestPlr, highestPlr uint32) error
	InsertBlockIndexRecord(block iBlock, fileIndex, posInFile, blockHeight uint32) error
	InsertTXIndexRecord(txid []byte, fileIndex, blockOffset, txOffsetInBlock uint32, batchMode bool) error
	WriteBatch() error
}

var blockIndex iBlockIndex

// NewBlockWriter creates a BlockWriter for the current plr file
func NewBlockWriter() *BlockWriter {
	homeDir, _ := os.UserHomeDir()
	// getting the latest file index
	lastFileIdx, err := blockIndex.GetLastBlockFileIdx()
	if errors.Is(err, leveldb.ErrNotFound) {
		if err := blockIndex.InsertLastBlockFileIdx(0); err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	}
	// getting file metadata to initialize writer
	fMeta, err := blockIndex.GetFileInfoRecort(lastFileIdx)
	var remSize int32
	if err == nil {
		// getting the correct remaining size in current file
		remSize = params.MaxBlockFileSize - int32(fMeta.SizeOfPlr())
	} else if errors.Is(err, leveldb.ErrNotFound) {
		// no file info record found means this is the first time writing
		remSize = params.MaxBlockFileSize
	} else {
		panic(err)
	}
	// subtracting current file size to initialize "carret" correctly
	return &BlockWriter{FileIndex: lastFileIdx, path: filepath.Join(homeDir, params.StoragePath), maxSize: params.MaxBlockFileSize, remSize: remSize}
}

// TODO: Should partially validated blocks be written?
func (bw *BlockWriter) Write(block iBlock, blockHeight uint32, indexTX bool) (n int, err error) {
	// Block structure in file is:
	//  -- Magic Bytes (4 bytes)
	//  -- Size of serialized block data (4 bytes)
	//  -- Block Data

	// getting the serialized block data
	ser := block.Serialize()
	// initializing with magic bytes + size of upcoming data in bytes
	data := append(params.BlockMagicBytes, utils.SerializeUint32(uint32(len(ser)), false)...)
	// appending the serialized block data
	data = append(data, ser...)
	// moving to new file if storage in current file is not sufficient
	if int32(len(data)) > bw.remSize {
		// saving current file metadata
		// assuming highest block height is the previous one
		if err := blockIndex.InsertFileInfoRecord(bw.FileIndex, bw.currentNoOfBlocks, uint32(bw.maxSize-bw.remSize), bw.firstBlockHeight, blockHeight-1); err != nil {
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
		// However, if the write fails, the program will probably panic anyway.
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
	// updating the file index record for the current file
	// for the current file, the firstBlockHeight has been set when creating the file
	// if the current file is the first one, then the firstBlockHeight will be 0, which is correct.
	// if not, the lowest and heighest height will be the same, since only one block exists presently in the file
	// The highest block height for the current file will always be the height currently being written
	if err := blockIndex.InsertFileInfoRecord(bw.FileIndex, bw.currentNoOfBlocks, uint32(bw.maxSize-bw.remSize), bw.firstBlockHeight, blockHeight); err != nil {
		return n, nil
	}

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

func (bw *BlockWriter) indexBlockTransactions(blockOffset uint32, block iBlock) error {
	// Block header is 32 + 32 + 8 + 4 + 4 = 80 bytes
	// Size of serialized block is 4 bytes.
	// There are 4 magic bytes.
	// Number of Transactions is a 4-byte number.
	// using the carret to get the offset of the TX inside the block
	carret := 92 + blockOffset
	txChan := make(chan interface{})
	// using this method as a generator to iterate over the block transactions
	go block.IterateBlockTx(txChan)
	for t := range txChan {
		tx := t.(iTransation)
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

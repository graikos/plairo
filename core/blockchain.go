package core

import (
	"bytes"
	"errors"
	"plairo/params"
	"plairo/storage"
)

var ErrInvalidLink = errors.New("previous block hash does not match")
var ErrInvalidHeight = errors.New("invalid block height")

// BNode represents a node in the blockchain. It holds the header info of the block.
type BNode struct {
	previousBNode *BNode
	nextBNode     *BNode

	header *BlockHeader

	height       uint32
	isFork       bool
	conflictNode *BNode
}

func (bn *BNode) InitializeHeaderFromBlock(block *Block) {
	bnheader := *(block.header)
	bn.header = &bnheader
}

func createGenesisNode() *BNode {
	return &BNode{
		previousBNode: nil,
		nextBNode:     nil,
		header: &BlockHeader{
			PreviousBlockHash: []byte{},
			MerkleRoot:        params.GenesisMerkleRoot,
			Timestamp:         0,
			TargetBits:        params.GenesisTargetBits,
			Nonce:             params.GenesisNonce,
		},
		isFork:       false,
		conflictNode: nil,
	}
}

type Blockchain struct {
	chain []*BNode
	forks []*Fork
}

type Fork struct {
	forkRoot  *BNode
	maxHeight uint32
}

func CreateBlockchain() *Blockchain {
	// initializing with genesis block
	return &Blockchain{
		[]*BNode{createGenesisNode()},
		[]*Fork{},
	}
}

func (bc *Blockchain) InsertBlock(block *Block, height uint32) error {

	if height > uint32(len(bc.chain)) {
		return ErrInvalidHeight
	} else if height == uint32(len(bc.chain)) {
		// checking if link is correct
		lastnode := bc.chain[len(bc.chain)-1]
		if !bytes.Equal(block.header.PreviousBlockHash, lastnode.header.GetHash()) {
			return ErrInvalidLink
		}
		if err := ValidateBlock(block, height); err != nil {
			return err
		}
		// creating new node
		newnode := &BNode{
			previousBNode: lastnode,
			nextBNode:     nil,
			height:        height,
			isFork:        false,
			conflictNode:  nil,
		}
		newnode.InitializeHeaderFromBlock(block)

		// linking former last node to the new node appended
		lastnode.nextBNode = newnode

		// confirming block as valid will remove UTXOs used in this block
		// and add the new UTXOs created in this block to the chainstate
		if err := block.ConfirmAsValid(); err != nil {
			return err
		}
		bwriter := storage.NewBlockWriter()
		if _, err := bwriter.Write(block, height, true); err != nil {
			return err
		}
		// TODO: Remove block TX from MemPool
		return nil
	}
	// TODO: add forks

	return nil
}

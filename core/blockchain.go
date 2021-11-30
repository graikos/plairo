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

	height uint32
	isFork bool
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
		isFork: false,
	}
}

type Fork struct {
	forkRoot  *BNode
	forkHead  *BNode
	maxHeight uint32
}

// couldReach is a quick way to check if an existing fork should be considered when appending blocks
func (f *Fork) couldReach(height uint32) bool {
	return f.maxHeight == height-1
}

// couldAttach checks if the new node is compatible with this fork
func (f *Fork) couldAttach(hashLink []byte) bool {
	return bytes.Equal(f.forkHead.header.GetHash(), hashLink)
}

type Blockchain struct {
	chain []*BNode
	forks []*Fork
}

func CreateBlockchain() *Blockchain {
	// initializing with genesis block
	return &Blockchain{
		[]*BNode{createGenesisNode()},
		[]*Fork{},
	}
}

func (bc *Blockchain) InsertBlock(block *Block, height uint32) error {

	if height > uint32(len(bc.chain)) || height <= 0 {
		// handling blocks with invalid height
		return ErrInvalidHeight
	} else if height < uint32(len(bc.chain)) {
		// if this condition is true, it means that the block to be inserted belongs to a fork

		// checking new block against current block of same height
		conflictNode := bc.chain[height]
		// if they are both linked to the same previous node, then an new fork should be created
		if bytes.Equal(block.header.PreviousBlockHash, conflictNode.header.PreviousBlockHash) {

			// validating header only for new block
			if err := ValidateBlockHeader(block.GetBlockHeader()); err != nil {
				return err
			}
			// creating new node that will be the root of the new fork
			newnode := &BNode{
				previousBNode: conflictNode.previousBNode,
				nextBNode:     nil,
				height:        height,
				isFork:        true,
			}
			newnode.InitializeHeaderFromBlock(block)

			// createing new fork, head and root are the same node since only one node exists in fork
			newfork := &Fork{
				forkRoot:  newnode,
				forkHead:  newnode,
				maxHeight: height,
			}

			// appending new fork to main chain forks
			bc.forks = append(bc.forks, newfork)

			// no need to check for re-org, since a new fork could never have a height greater than the main chain
			return nil

		}

		// checking if the block belongs to one of the existing forks
		for _, f := range bc.forks {
			if !f.couldReach(height) {
				continue
			}
			if f.couldAttach(block.header.GetHash()) {
				// if block is compatible with this fork, check for valid header
				// this will check header synta/structure and if the target is reached
				if err := ValidateBlockHeader(block.GetBlockHeader()); err != nil {
					return err
				}

				newnode := &BNode{
					previousBNode: f.forkHead,
					nextBNode:     nil,
					height:        height,
					isFork:        true,
				}
				newnode.InitializeHeaderFromBlock(block)
				// updating fork properties
				f.forkHead = newnode
				f.maxHeight = height

				// TODO: Check if max height is greater than main chain height, if so, re-org

				return nil
			}
		}

		// no suitable fork was found, the block should be rejected
		return ErrInvalidLink

	}
	// handling regular block insertion at the end of the chain
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
	}
	newnode.InitializeHeaderFromBlock(block)

	// linking former last node to the new node appended
	lastnode.nextBNode = newnode

	// by now the block has been confirmed, it should be written in storage
	bwriter := storage.NewBlockWriter()
	if _, err := bwriter.Write(block, height, true); err != nil {
		return err
	}
	// confirming block as valid will remove UTXOs used in this block
	// and add the new UTXOs created in this block to the chainstate
	if err := block.ConfirmAsValid(); err != nil {
		return err
	}

	// removing transactions from the mempool
	mempool.RemoveBlock(block)
	return nil
}

func (bc *Blockchain) GetHeaderAt(index int) (*BlockHeader, bool) {
	if index >= 0 && index < len(bc.chain) {
		return bc.chain[index].header, true
	}
	return nil, false
}

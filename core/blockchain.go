package core

import "plairo/params"

// BNode represents a node in the blockchain. It holds the header info of the block.
type BNode struct {
	previousBNode *BNode

	PreviousBlockHash []byte
	MerkleRoot        []byte
	Timestamp         int64
	targetBits        uint32
	nonce             uint32
}

type Blockchain struct {
	Head  *BNode
	Index uint32
}

func InitializeBlockchain() *Blockchain {
	return &Blockchain{&BNode{
		PreviousBlockHash: []byte{},
		MerkleRoot: params.GenesisMerkleRoot,
		Timestamp: 0,
		targetBits: params.GenesisTargetBits,
		nonce: params.GenesisNonce,
	}, 0}
}

func InitializeNodeFromBlock(block *Block) *BNode {
	node := &BNode{Timestamp: block.Timestamp, targetBits: block.targetBits, nonce: block.Nonce}
	copy(node.PreviousBlockHash, block.PreviousBlockHash)
	copy(node.MerkleRoot, block.MerkleRoot)
	return node
}

func (bc *Blockchain) InsertBlock(block *Block) error {
	// validating block to-be-inserted against the blockchain index after insertion
	if err := ValidateBlock(block, bc.Index+1); err != nil {
		return err
	}
	// by this point, the block is considered valid and is ready to be inserted to the blockchain
	// the outputs each transaction references can now be removed from chainstate
	if err := block.ConfirmAsValid(); err != nil {
		return err
	}
	// TODO: If block is rejected, the transactions should be added to the mempool again

	newHead := InitializeNodeFromBlock(block)
	newHead.previousBNode = bc.Head
	bc.Head = newHead
	bc.Index++
	return nil
}

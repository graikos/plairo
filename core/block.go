package core

import (
	"encoding/hex"
	"errors"
	"plairo/utils"
)

var ErrInvalidTxInBlock = errors.New("block contains invalid transaction")

type Block struct {
	PreviousBlockHash []byte
	MerkleRoot        []byte
	Timestamp         int64
	targetBits        int32
	nonce             int32
	allBlockTx        []*Transaction
}

func (b *Block) GetBlockHeader() []byte {
	/*
		Block header consists of:
		-- Previous block hash
		-- Merkle root of this block
		-- Timestamp of this block (8 bytes - Big Endian)
		-- TargetBits used when mining the block (4 bytes - Big Endian)
		-- Nonce (4 bytes - Big Endian)
	*/
	var header []byte
	header = append(header, b.PreviousBlockHash...)
	header = append(header, b.MerkleRoot...)
	header = append(header, utils.SerializeUint64(uint64(b.Timestamp), false)...)
	header = append(header, utils.SerializeUint32(uint32(b.targetBits), false)...)
	header = append(header, utils.SerializeUint32(uint32(b.nonce), false)...)
	return header
}

func (b *Block) GetBlockHash() []byte {
	return utils.CalculateSHA256Hash(utils.CalculateSHA256Hash(b.GetBlockHeader()))
}

//ValidateBlockTx validates every TX contained in the block and removes the UTXOs referenced, if all TXs are valid
func (b *Block) ValidateBlockTx() error {
	/*
		This method will have to remove the UTXOs referenced by each TX.
		To avoid referencing UTXOs twice or removing the UTXOs before ensuring the block is valid,
		a map will be used to check for duplicates and the cleanUp method will be called after making sure
		all the transactions are indeed valid.
	*/
	dedup := make(map[string]bool)
	for _, tx := range b.allBlockTx {
		if err := tx.ValidateTransaction(); err != nil {
			return err
		}
		/*
			Keeping track of each output referenced. If it has been referenced from another TX in the block,
			the block should be rejected. OutputID uniquely characterizes an output (hash of parentTXID+Vout).
		 */
		for _, inp := range tx.inputs {
			_, ok := dedup[hex.EncodeToString(inp.OutputReferred.OutputID)]
			if !ok {
				dedup[hex.EncodeToString(inp.OutputReferred.OutputID)] = true
				continue
			}
			return ErrInvalidTxInBlock
		}
	}

	// by this point, all the TX in the block are valid and the UTXOs they reference should be removed from chainstate
	// a second iteration is necessary to prevent removing the UTXOs of transactions in an invalid block
	for _, tx := range b.allBlockTx {
		if err := tx.cleanUpOutputs(); err != nil {
			return err
		}
	}
	return nil
}

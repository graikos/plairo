package core

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"plairo/params"
	"plairo/utils"
	"time"
)

var ErrInvalidTxInBlock = errors.New("block contains invalid transaction")
var ErrExceededMaxTX = errors.New("max transaction number exceeded")
var ErrStaleBlock = errors.New("stale block")
var ErrInvalidMerkleRoot = errors.New("merkle root is invalid")
var ErrTargetNotReached = errors.New("mined block does not satisfy target")

type Block struct {
	PreviousBlockHash []byte
	MerkleRoot        []byte
	Timestamp         int64
	Nonce             uint32
	targetBits        uint32
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
	header = append(header, utils.SerializeUint32(b.targetBits, false)...)
	header = append(header, utils.SerializeUint32(b.Nonce, false)...)
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
		// TODO: since dedup happens here, if an invalid TX is found, it should also be removed from mempool
		for _, inp := range tx.inputs {
			_, ok := dedup[hex.EncodeToString(inp.OutputReferred.OutputID)]
			if !ok {
				dedup[hex.EncodeToString(inp.OutputReferred.OutputID)] = true
				continue
			}
			return ErrInvalidTxInBlock
		}
	}

	return nil
}

func (b *Block) ConfirmAsValid() error {
	// by this point, all the TX in the block are valid and the UTXOs they reference should be removed from chainstate
	// a second iteration is necessary to prevent removing the UTXOs of transactions in an invalid block
	for _, tx := range b.allBlockTx {
		if err := tx.cleanUpOutputs(); err != nil {
			return err
		}
	}
	return nil
}

func (b *Block) generateBlockMerkleRoot() []byte {
	tmpTXID := make([][]byte, len(b.allBlockTx))
	for i, tx := range b.allBlockTx {
		tmpTXID[i] = tx.TXID
	}
	return utils.ComputeMerkleRoot(tmpTXID)
}

func (b *Block) ComputeMerkleRoot() {
	b.MerkleRoot = b.generateBlockMerkleRoot()
}

func (b *Block) GetBlockFees() uint64 {
	var total uint64
	for _, tx := range b.allBlockTx {
		total += tx.GetFees()
	}
	return total
}

func (b *Block) MineBlock(currentBlockHeight uint32, minerPubKey *ecdsa.PublicKey) error {
	// timestamping the block
	b.Timestamp = time.Now().Unix()
	// initializing the nonce
	b.Nonce = 0

	if err := b.ValidateBlockTx(); err != nil {
		return err
	}

	if len(b.allBlockTx) > params.MaxNumberOfTXsInBlock {
		return ErrExceededMaxTX
	}

	// calculating number of halvings for block about to be created, not current one
	halvings := (currentBlockHeight + 1) / params.SubsidyHalvingInterval
	subsidy := params.InitialBlockSubsidy >> halvings

	fees := b.GetBlockFees()
	coinbase, err := NewCoinbaseTransaction("coinbase", subsidy+fees, minerPubKey, currentBlockHeight)
	if err != nil {
		return err
	}
	b.allBlockTx = append(b.allBlockTx, coinbase)

	b.ComputeMerkleRoot()

	var blockHash []byte
	var timeBumps uint8

	for {
		blockHash = b.GetBlockHash()

		if bytes.Compare(blockHash, utils.ExpandBits(utils.SerializeUint32(b.targetBits, false))) < 0 {
			break
		}
		b.Nonce++

		// if all nonce values have been tried and it's back again at 0, then bump timestamp if permitted
		if b.Nonce == 0 {
			if timeBumps < 30 {
				b.Timestamp++
				timeBumps++
			} else {
				return ErrStaleBlock
			}
		}
	}

	return nil
}

func ValidateCoinbase(block *Block, minedBlockHeight uint32) error {
	// coinbase transaction is always appended last to the block
	coinbaseTX := block.allBlockTx[len(block.allBlockTx)-1]
	// getting the value specified in the coinbase transaction as miner reward
	var coinbaseValue uint64
	for _, outp := range coinbaseTX.outputs {
		coinbaseValue += outp.Value
	}
	// calculating number of halvings for block received
	halvings := (minedBlockHeight) / params.SubsidyHalvingInterval
	subsidy := params.InitialBlockSubsidy >> halvings

	// checking against total block fees and current subsidy
	if coinbaseValue > subsidy+block.GetBlockFees() {
		return ErrInvalidTxInBlock
	}
	return nil
}

func ValidateBlock(block *Block, minedBlockHeight uint32) error {
	if err := block.ValidateBlockTx(); err != nil {
		return err
	}
	if err := ValidateCoinbase(block, minedBlockHeight); err != nil {
		return err
	}
	if !bytes.Equal(block.MerkleRoot, block.generateBlockMerkleRoot()) {
		return ErrInvalidMerkleRoot
	}
	// TODO: add check for appropriate target bits used in mining (TBA when target and difficulty is implemented)
	if bytes.Compare(block.GetBlockHash(), utils.ExpandBits(utils.SerializeUint32(block.targetBits, false))) >= 0 {
		return ErrTargetNotReached
	}
	return nil
}

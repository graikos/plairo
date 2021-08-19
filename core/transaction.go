package core

import (
	"crypto/ecdsa"
	"fmt"
	"math"
	"plairo/params"
	"plairo/utils"
)

type Transaction struct {
	TXID        []byte
	TXsignature []byte
	BlockHeight uint32
	IsCoinbase  bool
	inputs      []*TransactionInput
	outputs     []*TransactionOutput
}

const (
	SIGHASH_ALL = iota + 1
	SIGHASS_NONE
)

// NewTransaction generates a new non-coinbase transaction
func NewTransaction(inputs []*TransactionInput, outputs []*TransactionOutput) *Transaction {
	// copying in/output slices to prevent external changes to the slice from modifying transaction internal slice
	tempinputs := make([]*TransactionInput, len(inputs))
	copy(tempinputs, inputs)
	tempoutputs := make([]*TransactionOutput, len(outputs))
	copy(tempoutputs, outputs)
	// blockheight will be set after adding the transaction to a mined block
	// transactions created using this constructor are not coinbase
	t := &Transaction{inputs: tempinputs, outputs: tempoutputs, BlockHeight: 0, IsCoinbase: false}
	t.updateOutputs()
	return t
}

func NewCoinbaseTransaction(coinbaseMsg string, coinbaseValue uint64, minerKey *ecdsa.PublicKey, blockHeight uint32) (*Transaction, error) {
	inputSig := make([]byte, 4)
	// blockheight+1 will be the height of the block to which this coinbase TX will belong
	// embedding it as inputSig
	copy(inputSig, utils.SerializeUint32(blockHeight+1, false))
	// appending the desired message (the sig of the coinbase will not be checked either way)
	inputSig = append(inputSig, []byte(coinbaseMsg)...)
	cInput := &TransactionInput{NewTransactionOutput(make([]byte, 32), 0, 0xffffffff, []byte{}), inputSig}
	if !params.ValueIsValid(coinbaseValue) {
		return nil, params.InvalidValue
	}
	// minerKey will be used as scriptPubKey (since payToPubKey is the only locking script implemented)
	scriptPubKey, err := utils.ConvertPubKeyToBytes(minerKey)
	if err != nil {
		return nil, fmt.Errorf("converting minerkey to bytes: %v", err)
	}
	cOutput := NewTransactionOutput([]byte{}, 0, coinbaseValue, scriptPubKey)
	t := &Transaction{BlockHeight: blockHeight + 1, inputs: []*TransactionInput{cInput}, outputs: []*TransactionOutput{cOutput}, IsCoinbase: true}
	t.updateOutputs()
	return t, nil
}

func (t *Transaction) GetOutputs() []*TransactionOutput {
	return t.outputs
}

func (t *Transaction) GetInputs() []*TransactionInput {
	return t.inputs
}

func (t *Transaction) updateOutputs() {
	if len(t.TXID) == 0 {
		t.generateTXID()
	}
	for i, outp := range t.outputs {
		if len(outp.ParentTXID) == 0 {
			outp.ParentTXID = t.TXID
			outp.Vout = uint32(i)
			outp.generateOutputID()
		}
	}
}

func (t *Transaction) generateTXID() {
	t.TXID = utils.CalculateSHA256Hash(utils.CalculateSHA256Hash(t.SerializeTransaction()))
}

func (t *Transaction) SerializeTransaction() []byte {
	/*
	 * No version number will be used.
	 * Number of inputs (int 4 bytes)
	 * 	 Output parentTXID (32 bytes)
	 * 	 Output vout (int)
	 * 	 Size of signature (unsigned long) indicating no of bytes
	 * 	   Signature (signature bytes + 1 byte for SIGHASH)
	 * 	 No sequence will be used.
	 * 	 ... (more inputs)
	 *
	 * Number of outputs (int 4 bytes)
	 *   Value of output in Ko (8 bytes) (unsigned long)
	 *   Size of pubkeyScript (8 bytes) (unsigned long)
	 *     pubKeyScript (variable length)
	 *
	 *  No locktime will be used.
	 *
	 *
	 */

	// initalizing with cap 8 which is guaranteed to reach because of noOfInputs and noOfOutputs
	res := make([]byte, 0, 8)
	res = append(res, utils.SerializeUint32(uint32(len(t.inputs)), false)...)
	for _, inp := range t.inputs {
		res = append(res, inp.OutputReferred.ParentTXID...)
		res = append(res, utils.SerializeUint32(inp.OutputReferred.Vout, false)...)
		// appending length of scriptsig (this will include the sighash byte, since it is appended to this
		// field when signing the input
		res = append(res, utils.SerializeUint64(uint64(len(inp.ScriptSig)), false)...)
		res = append(res, inp.ScriptSig...)
	}

	// now appending output data
	res = append(res, utils.SerializeUint32(uint32(len(t.outputs)), false)...)
	for _, outp := range t.outputs {
		res = append(res, utils.SerializeUint64(outp.Value, false)...)
		res = append(res, utils.SerializeUint64(uint64(len(outp.ScriptPubKey)), false)...)
		res = append(res, outp.ScriptPubKey...)
	}

	// reducing slice capacity
	r := append([]byte(nil), res...)
	return r
}

func (t *Transaction) SerializeTXMetadata() []byte {
	/*
	 * Value will be:
	 * -- isCoinbase (byte)
	 * -- block height (unsigned int - 4 bytes)
	 * -- number of outputs (unsigned int - 4 bytes)
	 * -- packed vector showing unspent outputs (variable - rounded to the nearest byte)
	 * -- for each unspent txo starting from 0:
	 * ---- value in Ko (unsigned long - 8 bytes)
	 * ---- size of scriptPubKey in bytes (unsigned int - 8 bytes)
	 * ---- scriptPubKey (since my version is simplified, this will be the recipient pubkey)
	 *
	 */
	// calculating the metadata length to allocate appropriately
	metadataLen := 9
	unspent := make([]bool, len(t.outputs))
	unspentCounter := 0
	for i, outp := range t.outputs {
		// marking unspent outputs
		unspent[i] = outp.IsNotSpent
		// only unspent tx metadata will be stored, only taking these into account to calculate length
		if outp.IsNotSpent {
			// 8 bits for the value and 8 bits to indicate the size of scriptpubkey
			metadataLen += 16 + len(outp.ScriptPubKey)
			unspentCounter++
		}
	}
	metadataLen += int(math.Floor(float64(unspentCounter)/8) + 1)
	metadata := make([]byte, 0, metadataLen)
	if t.IsCoinbase {
		metadata = append(metadata, 0x01)
	} else {
		metadata = append(metadata, 0x00)
	}
	metadata = append(metadata, utils.SerializeUint32(t.BlockHeight, false)...)
	metadata = append(metadata, utils.SerializeUint32(uint32(len(t.outputs)), false)...)
	metadata = append(metadata, utils.SerializeToOneHot(unspent)...)
	for _, outp := range t.outputs {
		if !outp.IsNotSpent {
			continue
		}
		metadata = append(metadata, utils.SerializeUint64(outp.Value, false)...)
		// appending the size of scriptpubkey
		metadata = append(metadata, utils.SerializeUint64(uint64(len(outp.ScriptPubKey)), false)...)
		metadata = append(metadata, outp.ScriptPubKey...)
	}
	return metadata
}

func (t *Transaction) IsSpent() bool {
	for _, outp := range t.outputs {
		if outp.IsNotSpent {
			return false
		}
	}
	return true
}

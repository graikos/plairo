package core

import (
	"crypto/ecdsa"
	"fmt"
	"plairo/utils"
)

type Transaction struct {
	TXID            []byte
	TXsignature     []byte
	BlockHeight     uint32
	IsCoinbase      bool
	senderPubKey    *ecdsa.PublicKey
	recipientPubKey *ecdsa.PublicKey
	inputs          []*TransactionInput
	outputs         []*TransactionOutput
}

const (
	SIGHASH_ALL = iota
	SIGHASS_NONE
)

// NewTransaction generates a new non-coinbase transaction
func NewTransaction(from, to *ecdsa.PublicKey, inputs []*TransactionInput, outputs []*TransactionOutput) *Transaction {
	// copying in/output slices to prevent external changes to the slice from modifying transaction internal slice
	tempinputs := make([]*TransactionInput, len(inputs))
	copy(tempinputs, inputs)
	tempoutputs := make([]*TransactionOutput, len(outputs))
	copy(tempoutputs, outputs)
	// blockheight will be set after adding the transaction to a mined block
	// transactions created using this constructor are not coinbase
	return &Transaction{senderPubKey: from, recipientPubKey: to, inputs: tempinputs, outputs: tempoutputs, BlockHeight: 0, IsCoinbase: false}
}

func NewCoinbaseTransaction(coinbaseMsg string, coinbaseValue uint64, minerKey *ecdsa.PublicKey, blockHeight uint32) (*Transaction, error) {
	inputSig := make([]byte, 4)
	// blockheight+1 will be the height of the block to which this coinbase TX will belong
	// embedding it as inputSig
	copy(inputSig, utils.SerializeUint32(blockHeight+1, false))
	// appending the desired message (the sig of the coinbase will not be checked either way)
	inputSig = append(inputSig, []byte(coinbaseMsg)...)
	cInput := &TransactionInput{NewTransactionOutput(make([]byte, 32), 0, 0xffffffff, []byte{}), inputSig}
	// TODO: add check for valid coinbase amount
	// minerKey will be used as scriptPubKey (since payToPubKey is the only locking script implemented)
	scriptPubKey, err := utils.ConvertPubKeyToBytes(minerKey)
	if err != nil {
		return nil, fmt.Errorf("converting minerkey to bytes: %v", err)
	}
	cOutput := NewTransactionOutput([]byte{}, 0, coinbaseValue, scriptPubKey)
	t := &Transaction{senderPubKey: minerKey, recipientPubKey: minerKey, BlockHeight: blockHeight + 1, inputs: []*TransactionInput{cInput}, outputs: []*TransactionOutput{cOutput}, IsCoinbase: true}
	t.updateOutputs()
	return t, nil
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

	return res
}

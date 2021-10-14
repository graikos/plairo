package core

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"plairo/coin"
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

var ErrDuplicateInput = errors.New("duplicate input")
var ErrNonExistentUTXO = errors.New("utxo referenced does not exist")
var ErrInvalidSignatureProvided = errors.New("invalid signature provided for input")
var ErrInputOutputMismatch = errors.New("output referred does not match actual output")
var ErrInsufficientFunds = errors.New("input value does not cover output value")

// SIGHASH is a flag used to provide flexibility when signing TXs, allowing multiple pay methods
type SIGHASH byte

// for now, only the SIGHASH_ALL case is implemented, meaning every TX detail is signed and tamperproof
const (
	SIGHASH_ALL SIGHASH = iota + 1
	SIGHASH_NONE
)

type CState interface {
	GetUtxo([]byte, uint32) (*TransactionOutput, bool)
	RemoveUtxo([]byte, uint32) bool
}

// cstate will be used as a chainstate pointer with injection from the db package
var cstate CState

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

// NewCoinbaseTransaction generates a new coinbase transaction
func NewCoinbaseTransaction(coinbaseMsg string, coinbaseValue uint64, minerKey *ecdsa.PublicKey, blockHeight uint32) (*Transaction, error) {
	inputSig := make([]byte, 4)
	// blockheight+1 will be the height of the block to which this coinbase TX will belong
	// embedding it as inputSig
	copy(inputSig, utils.SerializeUint32(blockHeight+1, false))
	// appending the desired message (the sig of the coinbase will not be checked either way)
	inputSig = append(inputSig, []byte(coinbaseMsg)...)
	cInput := &TransactionInput{NewTransactionOutput(make([]byte, 32), 0, 0xffffffff, []byte{}), inputSig}
	if !params.ValueIsValid(coinbaseValue) {
		return nil, params.ErrInvalidValue
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

func (t *Transaction) GetTXID() []byte {
	return t.TXID
}


// updateOutputs ensures the TX outputs have the appropriate fields
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
	t.TXID = utils.CalculateSHA256Hash(utils.CalculateSHA256Hash(t.Serialize()))
}

func (t *Transaction) Serialize() []byte {
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
	 *   Value of output in Ticks (8 bytes) (unsigned long)
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

// SerializeTXMetadata returns TX metadata used to store info about UTXOs (in chainstate)
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

// IsSpent determines if all the outputs are spent, making the whole Transaction spent
func (t *Transaction) IsSpent() bool {
	for _, outp := range t.outputs {
		if outp.IsNotSpent {
			return false
		}
	}
	return true
}

/*
ValidateTransaction is used to validate a transaction.
1) Checking if TX has duplicate inputs
2) Checking if signature provided to unlock an output referred is valid
3) Checking if outputs referred actually exist in the chainstate and are unspent
4) Checking if the new outputs have a valid value
5) Checking if the funds provided as input are sufficient to cover the output value
6) Will check if the funds provided are sufficient to cover the fees as well
*/
func (t *Transaction) ValidateTransaction() error {
	var inputValue uint64

	// using a map to make sure no duplicate inputs were used
	dedup := make(map[string]bool)
	for _, inp := range t.inputs {
		_, ok := dedup[hex.EncodeToString(inp.OutputReferred.OutputID)]
		if !ok {
			dedup[hex.EncodeToString(inp.OutputReferred.OutputID)] = true
			continue
		}
		return ErrDuplicateInput
	}

	for i, inp := range t.inputs {
		utxo, ok := cstate.GetUtxo(inp.OutputReferred.ParentTXID, inp.OutputReferred.Vout)
		if !ok {
			return ErrNonExistentUTXO
		}
		if !utxo.Equal(inp.OutputReferred) {
			return ErrInputOutputMismatch
		}
		sighashFlag := SIGHASH(inp.ScriptSig[len(inp.ScriptSig)-1])
		rawSig := inp.ScriptSig[:len(inp.ScriptSig)-1]

		sigMsg := t.gatherSignatureDataForInput(i, sighashFlag)
		// the public key will be the ScriptPubKey of the outputs, since no script is used
		// and this is a simplified version, using only the full public key
		pubkey, err := utils.ConvertBytesToPubKey(utxo.ScriptPubKey)
		if err != nil {
			return err
		}
		if !utils.VerifySignature(sigMsg, rawSig, pubkey) {
			// output cannot be unlocked, so the TX is rejected
			return ErrInvalidSignatureProvided
		}
		inputValue += inp.OutputReferred.Value
	}

	// getting the total value of the new outputs
	var outputValue uint64
	for _, outp := range t.outputs {
		outputValue += outp.Value
		// if any of the outputs has an invalid value, the TX is rejected
		if !params.ValueIsValid(outputValue) {
			return params.ErrInvalidValue
		}
	}

	// no need to check if input value is valid, since two valid input values may amount to an invalid output value
	// no need to check if total output value is valid, only individual outputs should have valid values
	// must check if input value can also cover minimum fees required to include this TX
	if inputValue < outputValue + t.GetMinimumFees() {
		return ErrInsufficientFunds
	}

	// TODO: Implement different validation if TX is coinbase
	/*
	 * The coinbase transaction must be validated differently. Since inputs do not matter,
	 * the only condition for a coinbase TX to be valid is for the output value not to exceed the sum of the block
	 * transaction fees + the block reward. In this context, the fees cannot be calculated, so the coinbase can not
	 * be validated using this method.
	 */

	// this method is purely used to validate a transaction
	// the UTXOs referenced should not be removed in this method
	return nil
}

// cleanUpOutputs removes the UTXOs referenced by this TX. Should be called after validating the TX.
func (t *Transaction) cleanUpOutputs() error {
	for _, inp := range t.inputs {
		if ok := cstate.RemoveUtxo(inp.OutputReferred.ParentTXID, inp.OutputReferred.Vout); !ok {
			return ErrNonExistentUTXO
		}
	}
	return nil
}

// gatherSignatureDataForInput provides the message to be signed according to the SIGHASH flag provided
func (t *Transaction) gatherSignatureDataForInput(inputIndex int, sighashFlag SIGHASH) []byte {
	switch sighashFlag {
	case SIGHASH_ALL:
		// backing up the rest input signatures
		backupSigs := make([][]byte, len(t.inputs))
		// replacing the rest of the signatures
		for i, inp := range t.inputs {
			backupSigs[i] = inp.ScriptSig
			if i == inputIndex {
				t.inputs[i].ScriptSig = inp.OutputReferred.ScriptPubKey
				continue
			}
			t.inputs[i].ScriptSig = []byte{}
		}
		// serializing this new TX without signature
		customSerialTX := t.Serialize()
		// appending the SIGHASH byte to obtain message data
		customSerialTX = append(customSerialTX, byte(sighashFlag))

		// double-hashing to obtain the message which will be used to sign the input
		signatureData := utils.CalculateSHA256Hash(utils.CalculateSHA256Hash(customSerialTX))
		// restoring the rest of the input signatures
		for i := range t.inputs {
			t.inputs[i].ScriptSig = backupSigs[i]
		}
		return signatureData

	default:
		return nil

	}
}

// signInput provides a signature for a specific input, its message determined by the SIGHASH flag
func (t *Transaction) signInput(inputIndex int, privateKey *ecdsa.PrivateKey, sighashFlag SIGHASH) error {
	signatureMsg := t.gatherSignatureDataForInput(inputIndex, sighashFlag)
	signature, err := utils.GenerateSignature(signatureMsg, privateKey)
	if err != nil {
		return fmt.Errorf("signing input %d: %v", inputIndex, err)
	}
	t.inputs[inputIndex].ScriptSig = append(signature, byte(sighashFlag))
	return nil
}

// GetMinimumFees returns minimum required fee amount for this TX. Calculated using the size of serialized TX.
func (t *Transaction) GetMinimumFees() uint64 {
	return coin.GetTotalFeeInTicks(t.Serialize())
}

func (t *Transaction) GetFees() uint64 {
	var inputValue uint64
	var outputValue uint64
	for _, inp := range t.inputs {
		inputValue += inp.OutputReferred.Value
	}
	for _, outp := range t.outputs {
		outputValue += outp.Value
	}
	return inputValue - outputValue
}
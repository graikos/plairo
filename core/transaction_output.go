package core

import (
	"bytes"
	"encoding/binary"
	"plairo/utils"
)

type TransactionOutput struct {
	ParentTXID   []byte
	Vout         uint32
	Value        uint64
	ScriptPubKey []byte
	OutputID     []byte
	IsNotSpent   bool
}

type TransactionInput struct {
	OutputReferred *TransactionOutput
	ScriptSig      []byte
}

func NewTransactionOutput(parenttxid []byte, vout uint32, value uint64, scriptpubkey []byte) *TransactionOutput {
	t := &TransactionOutput{ParentTXID: parenttxid, Vout: vout, Value: value, ScriptPubKey: scriptpubkey, IsNotSpent: true}
	// generating the outputID on creation
	t.generateOutputID()
	return t
}

// generateOutputID generates the ID, which will be parentTXID+vout double hashed.
func (t *TransactionOutput) generateOutputID() {
	// concat will be length of parentTXID and the 4 bytes used to represent vout
	res := make([]byte, len(t.ParentTXID)+4)
	copy(res[0:len(t.ParentTXID)], t.ParentTXID)
	// using indexing to put binary vout at the end
	binary.BigEndian.PutUint32(res[len(t.ParentTXID):], t.Vout)
	// double hash to get the output id
	t.OutputID = utils.CalculateSHA256Hash(utils.CalculateSHA256Hash(res))
}

func (t *TransactionOutput) Equal(t2 *TransactionOutput) bool {
	if bytes.Equal(t.OutputID, t2.OutputID) && t.Value == t2.Value && t.IsNotSpent == t2.IsNotSpent && bytes.Equal(t.ScriptPubKey, t2.ScriptPubKey) {
		return true
	}
	return false
}

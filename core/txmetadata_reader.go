package core

import (
	"math"
	"plairo/utils"
)
type TxMetadataReader struct {
	txid     []byte
	metadata []byte
	/*
	   Metadata format is:
	   	 * -- isCoinbase (byte)
	   	 * -- block height (unsigned int - 4 bytes)
	   	 * -- number of outputs (unsigned int - 4 bytes)
	   	 * -- packed vector showing unspent outputs (variable - rounded to the nearest byte)
	   	 * -- for each unspent txo starting from 0:
	   	 * ---- value in Ko (unsigned long - 8 bytes)
	   	 * ---- size of scriptPubKey in bytes (unsigned int - 8 bytes)
	   	 * ---- scriptPubKey (since my version is simplified, this will be the recipient pubkey)
	*/
}

func NewTxMetadataReader(txid, txmetadata []byte) *TxMetadataReader {
	return &TxMetadataReader{txid, txmetadata}
}

func (tr *TxMetadataReader) ReadIsCoinbase() bool {
	return tr.metadata[0] == 0x01
}

func (tr *TxMetadataReader) ReadBlockHeight() uint32 {
	return utils.DeserializeUint32(tr.metadata[1:5], false)
}

func (tr *TxMetadataReader) ReadNoOfOutputs() uint32 {
	return utils.DeserializeUint32(tr.metadata[5:9], false)
}

func (tr *TxMetadataReader) ReadBitVector() ([]bool, []uint32) {
	noOfOutputs := tr.ReadNoOfOutputs()
	res := make([]bool, noOfOutputs)
	sizeInBytes := int(math.Ceil(float64(noOfOutputs) / 8))
	// indexing a slice of length equal to number of bytes used in the bitvector
	bvec := tr.metadata[9 : 9+sizeInBytes]
	var i uint32
	// saving the unspent vouts. These will be used to read the outputs appropriately
	var vouts []uint32
	for ; i < noOfOutputs; i++ {
		res[i] = bvec[(sizeInBytes-1)-int(math.Floor(float64(i)/8))]&(0x01<<(i%8)) == byte(math.Pow(2, float64(i%8)))
		if res[i] {
			vouts = append(vouts, i)
		}
	}
	return res, vouts
}
func (tr *TxMetadataReader) ReadOutputs(notSpentVec []bool, vouts []uint32) []*TransactionOutput {
	noOfOutputs := tr.ReadNoOfOutputs()
	if notSpentVec == nil || vouts == nil || uint32(len(notSpentVec)) != noOfOutputs {
		notSpentVec, vouts = tr.ReadBitVector()
	}

	res := make([]*TransactionOutput, len(vouts))
	var i uint32
	caret := 9 + uint64(math.Ceil(float64(noOfOutputs)/8))

	for ; i < uint32(len(vouts)); i++ {
		val := utils.DeserializeUint64(tr.metadata[caret:caret+8], false)
		caret += 8
		slen := utils.DeserializeUint64(tr.metadata[caret:caret+8], false)
		caret += 8
		scpub := tr.metadata[caret : caret+slen]
		caret += slen
		res[i] = NewTransactionOutput(tr.txid, vouts[i], val, scpub)
	}
	if len(res) == 0 {
		return nil
	}
	return res
}

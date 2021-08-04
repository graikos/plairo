package readers

import (
	"math"
	"plairo/core"
	"plairo/utils"
)

type TxMetadataReader struct {
	txid     []byte
	metadata []byte
	pos      int
}

func NewTxMetadataReader(txid, txmetadata []byte) *TxMetadataReader {
	return &TxMetadataReader{txid, txmetadata, 0}
}

func (tr *TxMetadataReader) ReadIsCoinbase() bool {
	tr.pos++
	if tr.metadata[0] == 0x01 {
		return true
	}
	return false
}

func (tr *TxMetadataReader) ReadBlockHeight() uint32 {
	return utils.DeserializeUint32(tr.metadata[1:5], false)
}

func (tr *TxMetadataReader) ReadNoOfOutputs() uint32 {
	return utils.DeserializeUint32(tr.metadata[5:9], false)
}

func (tr *TxMetadataReader) ReadBitVector() []bool {
	noOfOutputs := tr.ReadNoOfOutputs()
	res := make([]bool, noOfOutputs)
	sizeInBytes := int(math.Ceil(float64(noOfOutputs) / 8))
	// indexing a slice of length equal to number of bytes used in the bitvector
	bvec := tr.metadata[9 : 9+sizeInBytes]
	var i uint32
	for ; i < noOfOutputs; i++ {
		res[i] = bvec[(sizeInBytes-1)-int(math.Floor(float64(i)/8))]&(0x01<<(i%8)) == byte(math.Pow(2, float64(i%8)))
	}
	return res
}
func (tr *TxMetadataReader) ReadOutputs(notSpentVec []bool) []*core.TransactionOutput {
	noOfOutputs := tr.ReadNoOfOutputs()
	res := make([]*core.TransactionOutput, noOfOutputs)
	var i uint32
	caret := 9 + uint64(math.Ceil(float64(noOfOutputs)/8))
	if notSpentVec == nil || uint32(len(notSpentVec)) != noOfOutputs {
		notSpentVec = tr.ReadBitVector()
	}
	for ; i < noOfOutputs; i++ {
		val := utils.DeserializeUint64(tr.metadata[caret:caret+8], false)
		caret += 8
		slen := utils.DeserializeUint64(tr.metadata[caret:caret+8], false)
		caret += 8
		scpub := tr.metadata[caret : caret+slen]
		caret += slen
		res[i] = core.NewTransactionOutput(tr.txid, i, val, scpub)
		res[i].IsNotSpent = notSpentVec[i]
	}
	return res
}

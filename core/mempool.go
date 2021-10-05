package core

import (
	"encoding/hex"
	"errors"
)

var ErrDoubleSpentOutput = errors.New("output referenced twice")

type MemPoolTX struct {
	Tx  *Transaction
	Fee uint64
}

type MemPoolHeap []*MemPoolTX

type MemPool struct {
	mpHeap         *MemPoolHeap
	outsReferenced map[string]bool
}

func (mh MemPoolHeap) Len() int {
	return len(mh)
}

func (mh MemPoolHeap) Less(i, j int) bool {
	// the MemPool should be a max heap to make TX picking for candidate block easier
	// as a result, higher fee should have greater priority and considered "less" than the other TX
	return mh[i].Fee > mh[j].Fee
}

func (mh MemPoolHeap) Swap(i, j int) {
	mh[i], mh[j] = mh[j], mh[i]
}

func (mh *MemPoolHeap) Push(x interface{}) {
	*mh = append(*mh, x.(*MemPoolTX))
}

func (mh *MemPoolHeap) Pop() interface{} {
	x := (*mh)[len(*mh)-1]
	*mh = (*mh)[0 : len(*mh)-1]
	return x
}

func (mp *MemPool) AddTX(tx *Transaction) error {
	// validating TX before adding (this includes fee requirements)
	if err := tx.ValidateTransaction(); err != nil {
		return err
	}
	// checking for double-spend with other transactions in the mempool
	for _, inp := range tx.inputs {
		_, ok := mp.outsReferenced[hex.EncodeToString(inp.OutputReferred.OutputID)]
		if !ok {
			mp.outsReferenced[hex.EncodeToString(inp.OutputReferred.OutputID)] = true
			continue
		}
		return ErrDoubleSpentOutput
	}
	mp.mpHeap.Push(&MemPoolTX{tx, tx.GetFees()})
	return nil
}
// TODO: IMPORTANT Figure out a way to remove transactions from inside the heap

// PopTX returns the transaction with the greatest fee value and removes outputs referenced from mempool log
func (mp *MemPool) PopTX() *Transaction {
	// an undo would be necessary only if the candidate block is rejected when inserting to the blockchain
	tx := mp.mpHeap.Pop().(*MemPoolTX).Tx
	for _, inp := range tx.inputs {
		delete(mp.outsReferenced, hex.EncodeToString(inp.OutputReferred.OutputID))
	}
	return tx
}


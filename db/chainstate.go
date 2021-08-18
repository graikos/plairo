package db

import (
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"plairo/core"
	"plairo/core/readers"
)

type Chainstate struct {
	dbwrapper *DBwrapper
}

func NewChainstate(dbpath string, isObfuscated bool) *Chainstate {
	return &Chainstate{NewDBwrapper(dbpath, isObfuscated)}
}

func buildTXkey(txid []byte) []byte {
	tkey := make([]byte, 1+len(txid))
	tkey[0] = byte('c')
	copy(tkey[1:], txid)
	return tkey
}

func (c *Chainstate) InsertTX(tx *core.Transaction) error {
	return c.dbwrapper.Insert(buildTXkey(tx.TXID), tx.SerializeTXMetadata())
}

func (c *Chainstate) RemoveTX(txid []byte) error {
	return c.dbwrapper.Remove(buildTXkey(txid))
}

func (c *Chainstate) GetTX(txid []byte) ([]byte, error) {
	return c.dbwrapper.Get(buildTXkey(txid))
}

func (c *Chainstate) UtxoExists(txid []byte, vout uint32) bool {
	txmeta, err := c.GetTX(txid)
	if errors.Is(err, leveldb.ErrNotFound) {
		return false
	}
	tr := readers.NewTxMetadataReader(txid, txmeta)
	bitvec, _ := tr.ReadBitVector()
	if uint32(len(bitvec)) < vout {
		return false
	}
	return bitvec[vout]
}

func (c *Chainstate) GetUtxo(txid []byte, vout uint32) (*core.TransactionOutput, bool) {
	txmeta, err := c.GetTX(txid)
	if errors.Is(err, leveldb.ErrNotFound) {
		return nil, false
	}
	tr := readers.NewTxMetadataReader(txid, txmeta)
	bv, vouts := tr.ReadBitVector()
	for i := 0; i < len(vouts); i++ {
		if vouts[i] == vout {
			return tr.ReadOutputs(bv, vouts)[i], true
		}
	}
	return nil, false
}

func (c *Chainstate) RemoveUtxo(txid []byte, vout uint32) bool {
	// no need to use the getUtxo method, will perform linear search in vouts since
	// number of outputs is expected to be small
	txmeta, err := c.GetTX(txid)
	if errors.Is(err, leveldb.ErrNotFound) {
		return false
	}

	tr := readers.NewTxMetadataReader(txid, txmeta)
	// getting the slice of unspent tx and their vouts
	bv, vouts := tr.ReadBitVector()

	// if utxo to-be-removed is the last unspent output, then
	// the TX entry must be removed completely
	if len(vouts) == 1 && vouts[0] == vout {
		if c.dbwrapper.Remove(buildTXkey(txid)) != nil {
			return false
		}
		return true
	}

	if vout >= uint32(len(bv)) || !bv[vout] {
		return false
	}
	outs := tr.ReadOutputs(bv, vouts)

	// creating a new fake transaction without this utxo to serialize
	fakeouts := make([]*core.TransactionOutput, tr.ReadNoOfOutputs())

	// the length of the fakeouts equals the length of the parse bitvector
	// iterating through the bitvector and creating fake outputs for unspent vouts only
	for i, val := range bv {
		// the output to-be-removed should be treated as spent in the fake outputs
		if !val || uint32(i) == vout {
			fakeouts[i] = core.NewTransactionOutput([]byte{}, 0, 0, []byte{})
			fakeouts[i].IsNotSpent = false
		} else {
			fakeouts[i] = outs[i]
		}
	}
	newmeta := core.NewTransaction(nil, fakeouts).SerializeTXMetadata()
	err = c.dbwrapper.Insert(buildTXkey(txid), newmeta)
	if err != nil {
		return false
	}
	return true
}

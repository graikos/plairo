package db

import (
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"os"
	"path/filepath"
	"plairo/core"
)

type Chainstate struct {
	dbwrapper *DBwrapper
}

var ChainstatePath string

func init() {
	homedir, _ := os.UserHomeDir()
	ChainstatePath = filepath.Join(homedir, "/.plairo/chainstate")
	// instead of initializing the chainstate db here, it will be initialized when injecting to core objects
}

var ErrSpentTX = errors.New("TX has no unspent outputs")

func NewChainstate(dbpath string, isObfuscated bool) *Chainstate {
	return &Chainstate{NewDBwrapper(dbpath, isObfuscated)}
}

func (c *Chainstate) InsertTX(tx *core.Transaction) error {
	// checking if there are unspent outputs left before inserting
	if tx.IsSpent() {
		return ErrSpentTX
	}
	return c.dbwrapper.Insert(buildKey(TxKey, tx.TXID), tx.SerializeTXMetadata())

}

func (c *Chainstate) RemoveTX(txid []byte) error {
	return c.dbwrapper.Remove(buildKey(TxKey, txid))
}

func (c *Chainstate) GetTX(txid []byte) ([]byte, error) {
	return c.dbwrapper.Get(buildKey(TxKey, txid))
}

func (c *Chainstate) UtxoExists(txid []byte, vout uint32) bool {
	txmeta, err := c.GetTX(txid)
	if errors.Is(err, leveldb.ErrNotFound) {
		return false
	}
	tr := core.NewTxMetadataReader(txid, txmeta)
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
	tr := core.NewTxMetadataReader(txid, txmeta)
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

	tr := core.NewTxMetadataReader(txid, txmeta)
	// getting the slice of unspent tx and their vouts
	bv, vouts := tr.ReadBitVector()

	// if utxo to-be-removed is the last unspent output, then
	// the TX entry must be removed completely
	if len(vouts) == 1 && vouts[0] == vout {
		if c.dbwrapper.Remove(buildKey(TxKey, txid)) != nil {
			return false
		}
		return true
	}

	// checking if UTXO exists before removing
	if vout >= uint32(len(bv)) || !bv[vout] {
		return false
	}
	outs := tr.ReadOutputs(bv, vouts)

	// creating a new fake transaction without this utxo to serialize
	fakeouts := make([]*core.TransactionOutput, tr.ReadNoOfOutputs())

	// placing each remaining unspent output to appropriate position in the new fake outputs
	for _, outp := range outs {
		if outp.Vout == vout {
			continue
		}
		fakeouts[outp.Vout] = outp
	}
	// filling in the fake outputs with dummy spent outputs
	for i, fakeout := range fakeouts {
		if fakeout == nil {
			fakeouts[i] = core.NewTransactionOutput([]byte{}, 0, 0, []byte{})
			fakeouts[i].IsNotSpent = false
		}
	}
	newmeta := core.NewTransaction(nil, fakeouts).SerializeTXMetadata()
	err = c.dbwrapper.Insert(buildKey(TxKey, txid), newmeta)
	if err != nil {
		return false
	}
	return true
}

func (c *Chainstate) GetNoOfUTXOs(txid []byte) (int, bool) {
	txmeta, err := c.GetTX(txid)
	if errors.Is(err, leveldb.ErrNotFound) {
		return 0, false
	}
	_, vouts := core.NewTxMetadataReader(txid, txmeta).ReadBitVector()
	return len(vouts), true

}

func (c *Chainstate) Close() {
	c.dbwrapper.Close()
}

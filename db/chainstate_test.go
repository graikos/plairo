package db

import (
	"bytes"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"os"
	"path/filepath"
	"plairo/core"
	"testing"
)

var testChainstatePath string

func init() {
	homedir, _ := os.UserHomeDir()
	testChainstatePath = filepath.Join(homedir, "/.plairo/chainstatetest")
}

type cTestCase struct {
	tx   *core.Transaction
	outs []*core.TransactionOutput
}

func newChainstateTestCase(values []uint64, pubkeys []string, spents []bool) *cTestCase {
	if !(len(values) == len(pubkeys) && len(values) == len(spents)) {
		panic("invalid array lengths")
	}
	outs := make([]*core.TransactionOutput, len(values))
	for i, value := range values {
		outs[i] = core.NewTransactionOutput([]byte{}, 0, value, []byte(pubkeys[i]))
		outs[i].IsNotSpent = spents[i]
	}
	tx := core.NewTransaction(nil, outs)
	return &cTestCase{tx, outs}
}

func testChainstateSetup() []*cTestCase {
	return []*cTestCase{
		newChainstateTestCase([]uint64{66, 77, 88}, []string{"pub", "public", "pubkey"}, []bool{true, true, true}),
		newChainstateTestCase([]uint64{66, 77, 88}, []string{"pub", "public", "pubkey"}, []bool{false, false, false}),
		newChainstateTestCase([]uint64{66, 77, 88}, []string{"pub", "public", "pubkey"}, []bool{false, true, false}),
		newChainstateTestCase([]uint64{55, 55, 55, 66, 77, 88, 999, 999}, []string{"pub", "pub", "pub", "pub", "public", "pubkey", "pp", "looooooooooooongkeytest"}, []bool{true, true, true, false, true, false, true, true}),
		newChainstateTestCase([]uint64{11, 55, 55, 55, 66, 77, 88, 999, 999}, []string{"f", "pub", "pub", "pub", "pub", "public", "pubkey", "pp", "looooooooooooongkeytest"}, []bool{false, true, true, true, false, true, false, true, true}),
	}
}

func TestBuildTXKey(t *testing.T) {
	testtxid := []byte{0x01, 0x0a, 0x02, 0x0b}
	expected := []byte{0x63, 0x01, 0x0a, 0x02, 0x0b}
	if !bytes.Equal(buildKey(TxKey, testtxid), expected) {
		t.Errorf("Expected %x\nGot %x\n", expected, buildKey(TxKey, testtxid))
	}
}

func TestChainstate_InsertTXGetTxRemoveTX(t *testing.T) {
	cstate := NewChainstate(testChainstatePath, true)
	defer cstate.Close()

	for i, tcase := range testChainstateSetup() {
		if err := cstate.InsertTX(tcase.tx); err != nil {
			if errors.Is(err, ErrSpentTX) && tcase.tx.IsSpent() {
				continue
			} else {
				t.Errorf("Error inserting to chainstate TX with index: %d\n", i)
			}
		}
		val, err := cstate.GetTX(tcase.tx.TXID)
		if err != nil {
			t.Errorf("Error getting from chainstate TX with index: %d\n", i)
		}
		if !bytes.Equal(val, tcase.tx.SerializeTXMetadata()) {
			t.Errorf("Value/Metadata mismatch index %d: Expected %x\n Got%x\n", i, tcase.tx.SerializeTXMetadata(), val)
		}
		if err := cstate.RemoveTX(tcase.tx.TXID); err != nil {
			t.Errorf("Error removing TX with index %d from chainstate.\n", i)
		}
		if _, err := cstate.GetTX(tcase.tx.TXID); !errors.Is(err, leveldb.ErrNotFound) {
			t.Errorf("Expected TX with index %d not to be found in chainstate.\n", i)
		}
	}
}

func TestChainstate_UtxoExists(t *testing.T) {
	cstate := NewChainstate(testChainstatePath, true)
	defer cstate.Close()

	for i, tcase := range testChainstateSetup() {
		// inserting every TX to chainstate
		if err := cstate.InsertTX(tcase.tx); err != nil {
			if errors.Is(err, ErrSpentTX) && tcase.tx.IsSpent() {
				continue
			} else {
				t.Errorf("Error inserting to chainstate TX with index: %d\n", i)
			}
		}

		// checking every utxo
		for j, outp := range tcase.outs {
			if cstate.UtxoExists(tcase.tx.TXID, uint32(j)) != outp.IsNotSpent {
				t.Errorf("TX: %d Vout:%d Expected output isNotSpent to be %v, got %v", i, j, outp.IsNotSpent, cstate.UtxoExists(tcase.tx.TXID, uint32(j)))
			}
		}
		// cleaning up
		if err := cstate.RemoveTX(tcase.tx.TXID); err != nil {
			t.Errorf("Error removing TX with index %d from chainstate.\n", i)
		}
	}
}

func TestChainstate_GetUtxo(t *testing.T) {
	cstate := NewChainstate(testChainstatePath, true)
	defer cstate.Close()

	for i, tcase := range testChainstateSetup() {
		// inserting every TX to chainstate
		if err := cstate.InsertTX(tcase.tx); err != nil {
			if errors.Is(err, ErrSpentTX) && tcase.tx.IsSpent() {
				continue
			} else {
				t.Errorf("Error inserting to chainstate TX with index: %d\n", i)
			}
		}

		// checking each output individually
		for j, outp := range tcase.outs {
			gotout, exists := cstate.GetUtxo(tcase.tx.TXID, uint32(j))
			if exists != outp.IsNotSpent {
				t.Errorf("Expected TX %d vout %d not to exist.\n", i, j)
			}
			if !exists {
				continue
			}
			if !gotout.Equal(outp) {
				t.Errorf("TX %d vout %d: Got different output than expected.\n", i, j)
			}
		}
		// cleaning up
		if err := cstate.RemoveTX(tcase.tx.TXID); err != nil {
			t.Errorf("Error removing TX with index %d from chainstate.\n", i)
		}
	}
}

func TestChainstate_GetNoOfUTXOs(t *testing.T) {
	cstate := NewChainstate(testChainstatePath, true)
	defer cstate.Close()

	for i, tcase := range testChainstateSetup() {
		// inserting every TX to chainstate
		if err := cstate.InsertTX(tcase.tx); err != nil {
			if errors.Is(err, ErrSpentTX) && tcase.tx.IsSpent() {
				continue
			} else {
				t.Errorf("Error inserting to chainstate TX with index: %d\n", i)
			}
		}

		unspentCounter := 0
		for _, outp := range tcase.outs {
			if outp.IsNotSpent {
				unspentCounter++
			}
		}
		gotNoOfUtxos, txexists := cstate.GetNoOfUTXOs(tcase.tx.TXID)
		if !txexists || gotNoOfUtxos != unspentCounter {
			t.Errorf("UTXOs number mismatch. Expected: %d Got: %d\n", unspentCounter, gotNoOfUtxos)
		}
	}
}

func TestChainstate_RemoveUtxo(t *testing.T) {

	cstate := NewChainstate(testChainstatePath, true)
	defer cstate.Close()

	for i, tcase := range testChainstateSetup() {
		// inserting every TX to chainstate
		if err := cstate.InsertTX(tcase.tx); err != nil {
			if errors.Is(err, ErrSpentTX) && tcase.tx.IsSpent() {
				continue
			} else {
				t.Errorf("Error inserting to chainstate TX with index: %d\n", i)
			}
		}

		_, txexists := cstate.GetNoOfUTXOs(tcase.tx.TXID)
		if !txexists {
			t.Errorf("Error getting TX with index: %d\n", i)
		}

		for j, outp := range tcase.outs {
			utxoRemoved := cstate.RemoveUtxo(tcase.tx.TXID, uint32(j))
			if utxoRemoved != outp.IsNotSpent {
				t.Errorf("Expected TXID %d vout %d to be removed.\n", i, j)
			}
		}

		if val, err := cstate.GetTX(tcase.tx.TXID); !errors.Is(err, leveldb.ErrNotFound) {
			t.Errorf("Expected tx with index %d not to be found. Got %x\n", i, val)
		}
	}
}

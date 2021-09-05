package core

import (
	"bytes"
	"fmt"
	"testing"
)

type readerTestCase struct {
	tr   *TxMetadataReader
	tx   *Transaction
	outs []*TransactionOutput
}

func newReaderTestCase(values []uint64, pubkeys []string, spents []bool) *readerTestCase {
	if !(len(values) == len(pubkeys) && len(values) == len(spents)) {
		panic("invalid array lengths")
	}
	outs := make([]*TransactionOutput, len(values))
	for i, value := range values {
		outs[i] = NewTransactionOutput([]byte{}, 0, value, []byte(pubkeys[i]))
		outs[i].IsNotSpent = spents[i]
	}
	tx := NewTransaction(nil, outs)
	tr := NewTxMetadataReader(tx.TXID, tx.SerializeTXMetadata())
	return &readerTestCase{tr, tx, outs}
}

func testReaderSetup() []*readerTestCase {
	return []*readerTestCase{
		newReaderTestCase([]uint64{66, 77, 88}, []string{"pub", "public", "pubkey"}, []bool{true, true, true}),
		newReaderTestCase([]uint64{66, 77, 88}, []string{"pub", "public", "pubkey"}, []bool{false, false, false}),
		newReaderTestCase([]uint64{66, 77, 88}, []string{"pub", "public", "pubkey"}, []bool{false, true, false}),
		newReaderTestCase([]uint64{55, 55, 55, 66, 77, 88, 999, 999}, []string{"pub", "pub", "pub", "pub", "public", "pubkey", "pp", "looooooooooooongkeytest"}, []bool{true, true, true, false, true, false, true, true}),
		newReaderTestCase([]uint64{11, 55, 55, 55, 66, 77, 88, 999, 999}, []string{"f", "pub", "pub", "pub", "pub", "public", "pubkey", "pp", "looooooooooooongkeytest"}, []bool{false, true, true, true, false, true, false, true, true}),
	}
}

func TestTxMetadataReader_ReadIsCoinbase(t *testing.T) {
	for _, tcase := range testReaderSetup() {
		if tcase.tr.ReadIsCoinbase() != tcase.tx.IsCoinbase {
			t.Error("Expected tx not to be coinbase.")
		}
	}
}

func TestTxMetadataReader_ReadBlockHeight(t *testing.T) {
	for _, tcase := range testReaderSetup() {
		if tcase.tr.ReadBlockHeight() != tcase.tx.BlockHeight {
			t.Errorf("Wrong block height. Expected: %d, got %d\n", tcase.tx.BlockHeight, tcase.tr.ReadBlockHeight())
		}
	}
}

func TestTxMetadataReader_ReadNoOfOutputs(t *testing.T) {
	for _, tcase := range testReaderSetup() {
		if uint32(len(tcase.tx.GetOutputs())) != tcase.tr.ReadNoOfOutputs() {
			t.Fatalf("Output count mismatch.\n Expected: %d outputs.\n Got: %d outputs.", len(tcase.tx.GetOutputs()), tcase.tr.ReadNoOfOutputs())
		}
	}
}

func TestTxMetadataReader_ReadBitVector(t *testing.T) {
	for _, tcase := range testReaderSetup() {
		got, _ := tcase.tr.ReadBitVector()
		for i, outp := range tcase.outs {
			if got[i] != outp.IsNotSpent {
				t.Fatalf("Expected %v for vout %d, got %v\n", outp.IsNotSpent, i, got[i])
			}
		}
	}
	for _, tcase := range testReaderSetup() {
		_, gotvouts := tcase.tr.ReadBitVector()
		var actvouts []uint32
		for i, outp := range tcase.outs {
			if outp.IsNotSpent {
				actvouts = append(actvouts, uint32(i))
			}
		}
		// if both are nil then the second loop won't run, so no panic.
		//
		if len(actvouts) != len(gotvouts) {
			t.Errorf("Expected length of %d, got length of %d.", len(actvouts), len(gotvouts))
		}
		for i := 0; i < len(actvouts); i++ {
			// if this loop runs, it means that the slices are of the same length
			// this means that if gotvouts is nil, this loop won't run, so no panic when indexing gotvouts
			if actvouts[i] != gotvouts[i] {
				t.Errorf("Expected vout %d to be equal to actual vout %d.", gotvouts[i], actvouts[i])
			}
		}
	}
}

func TestTxMetadataReader_ReadOutputs(t *testing.T) {
	for _, tcase := range testReaderSetup() {
		var unspentcounter int
		for _, out := range tcase.outs {
			if out.IsNotSpent {
				unspentcounter++
			}
		}

		gotouts := tcase.tr.ReadOutputs(nil, nil)
		if len(gotouts) != unspentcounter {
			t.Fatalf("Length mismatch. Expected %d. Got %d.", unspentcounter, len(gotouts))
		}
		for i := 0; i < len(gotouts); i++ {
			if !bytes.Equal(gotouts[i].ScriptPubKey, tcase.outs[gotouts[i].Vout].ScriptPubKey) {
				fmt.Printf("Got:\n ParentTXID: %x\n Vout: %d\n Value: %d\n IsNotSpent %v\n ScriptPubKey %x\n", gotouts[i].ParentTXID, gotouts[i].Vout, gotouts[i].Value, gotouts[i].IsNotSpent, gotouts[i].ScriptPubKey)
				expectedout := tcase.outs[gotouts[i].Vout]
				fmt.Printf("Expected:\n ParentTXID: %x\n Vout: %d\n Value: %d\n IsNotSpent %v\n ScriptPubKey %x\n", expectedout.ParentTXID, expectedout.Vout, expectedout.Value, expectedout.IsNotSpent, expectedout.ScriptPubKey)
				t.Errorf("Expected ScriptPubKey to be:\n%x\nGot:\n%x\n", tcase.outs[gotouts[i].Vout].ScriptPubKey, gotouts[i].ScriptPubKey)
			}
			if !bytes.Equal(gotouts[i].OutputID, tcase.outs[gotouts[i].Vout].OutputID) {
				fmt.Printf("Got:\n ParentTXID: %x\n Vout: %d\n Value: %d\n IsNotSpent %v\n ScriptPubKey %x\n", gotouts[i].ParentTXID, gotouts[i].Vout, gotouts[i].Value, gotouts[i].IsNotSpent, gotouts[i].ScriptPubKey)
				expectedout := tcase.outs[gotouts[i].Vout]
				fmt.Printf("Expected:\n ParentTXID: %x\n Vout: %d\n Value: %d\n IsNotSpent %v\n ScriptPubKey %x\n", expectedout.ParentTXID, expectedout.Vout, expectedout.Value, expectedout.IsNotSpent, expectedout.ScriptPubKey)
				t.Errorf("Expected output id to be: %x.\n Got: %x", tcase.outs[gotouts[i].Vout].OutputID, gotouts[i].OutputID)
			}
			if gotouts[i].Value != tcase.outs[gotouts[i].Vout].Value {
				fmt.Printf("Got:\n ParentTXID: %x\n Vout: %d\n Value: %d\n IsNotSpent %v\n ScriptPubKey %x\n", gotouts[i].ParentTXID, gotouts[i].Vout, gotouts[i].Value, gotouts[i].IsNotSpent, gotouts[i].ScriptPubKey)
				t.Errorf("Expected value to be: %d.\n Got: %d", tcase.outs[gotouts[i].Vout].Value, gotouts[i].Value)
			}

			if !tcase.outs[gotouts[i].Vout].IsNotSpent {
				fmt.Printf("Got:\n ParentTXID: %x\n Vout: %d\n Value: %d\n IsNotSpent %v\n ScriptPubKey %x\n", gotouts[i].ParentTXID, gotouts[i].Vout, gotouts[i].Value, gotouts[i].IsNotSpent, gotouts[i].ScriptPubKey)
				t.Errorf("Output with vout %d. IsNotSpent returned is: %v\n", gotouts[i].Vout, gotouts[i].IsNotSpent)
			}
		}

	}
	for _, tcase := range testReaderSetup() {
		var unspentcounter int
		boolvec := make([]bool, len(tcase.outs))
		for i, out := range tcase.outs {
			if out.IsNotSpent {
				unspentcounter++
			}
			boolvec[i] = out.IsNotSpent
		}

		gotouts := tcase.tr.ReadOutputs(boolvec, nil)
		if len(gotouts) != unspentcounter {
			t.Fatalf("Length mismatch. Expected %d. Got %d.", unspentcounter, len(gotouts))
		}
		for i := 0; i < len(gotouts); i++ {
			if !bytes.Equal(gotouts[i].ScriptPubKey, tcase.outs[gotouts[i].Vout].ScriptPubKey) {
				fmt.Printf("Got:\n ParentTXID: %x\n Vout: %d\n Value: %d\n IsNotSpent %v\n ScriptPubKey %x\n", gotouts[i].ParentTXID, gotouts[i].Vout, gotouts[i].Value, gotouts[i].IsNotSpent, gotouts[i].ScriptPubKey)
				expectedout := tcase.outs[gotouts[i].Vout]
				fmt.Printf("Expected:\n ParentTXID: %x\n Vout: %d\n Value: %d\n IsNotSpent %v\n ScriptPubKey %x\n", expectedout.ParentTXID, expectedout.Vout, expectedout.Value, expectedout.IsNotSpent, expectedout.ScriptPubKey)
				t.Errorf("Expected ScriptPubKey to be:\n%x\nGot:\n%x\n", tcase.outs[gotouts[i].Vout].ScriptPubKey, gotouts[i].ScriptPubKey)
			}
			if !bytes.Equal(gotouts[i].OutputID, tcase.outs[gotouts[i].Vout].OutputID) {
				fmt.Printf("Got:\n ParentTXID: %x\n Vout: %d\n Value: %d\n IsNotSpent %v\n ScriptPubKey %x\n", gotouts[i].ParentTXID, gotouts[i].Vout, gotouts[i].Value, gotouts[i].IsNotSpent, gotouts[i].ScriptPubKey)
				expectedout := tcase.outs[gotouts[i].Vout]
				fmt.Printf("Expected:\n ParentTXID: %x\n Vout: %d\n Value: %d\n IsNotSpent %v\n ScriptPubKey %x\n", expectedout.ParentTXID, expectedout.Vout, expectedout.Value, expectedout.IsNotSpent, expectedout.ScriptPubKey)
				t.Errorf("Expected output id to be: %x.\n Got: %x", tcase.outs[gotouts[i].Vout].OutputID, gotouts[i].OutputID)
			}
			if gotouts[i].Value != tcase.outs[gotouts[i].Vout].Value {
				fmt.Printf("Got:\n ParentTXID: %x\n Vout: %d\n Value: %d\n IsNotSpent %v\n ScriptPubKey %x\n", gotouts[i].ParentTXID, gotouts[i].Vout, gotouts[i].Value, gotouts[i].IsNotSpent, gotouts[i].ScriptPubKey)
				t.Errorf("Expected value to be: %d.\n Got: %d", tcase.outs[gotouts[i].Vout].Value, gotouts[i].Value)
			}

			if !boolvec[gotouts[i].Vout] {
				fmt.Printf("Got:\n ParentTXID: %x\n Vout: %d\n Value: %d\n IsNotSpent %v\n ScriptPubKey %x\n", gotouts[i].ParentTXID, gotouts[i].Vout, gotouts[i].Value, gotouts[i].IsNotSpent, gotouts[i].ScriptPubKey)
				t.Errorf("Output with vout %d. IsNotSpent returned is: %v\n", gotouts[i].Vout, gotouts[i].IsNotSpent)
			}
		}

	}
}

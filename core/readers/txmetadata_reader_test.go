package readers

import (
	"bytes"
	"fmt"
	"plairo/core"
	"testing"
)

type readerTestCase struct {
	tr   *TxMetadataReader
	tx   *core.Transaction
	outs []*core.TransactionOutput
}

func newReaderTestCase(values []uint64, pubkeys []string, spents []bool) *readerTestCase {
	if !(len(values) == len(pubkeys) && len(values) == len(spents)) {
		panic("invalid array lengths")
	}
	outs := make([]*core.TransactionOutput, len(values))
	for i, value := range values {
		outs[i] = core.NewTransactionOutput([]byte{}, 0, value, []byte(pubkeys[i]))
		outs[i].IsNotSpent = spents[i]
	}
	tx := core.NewTransaction(nil, outs)
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

func TestTxMetadataReader_ReadBitVector(t *testing.T) {
	for _, tcase := range testReaderSetup() {
		got := tcase.tr.ReadBitVector()
		for i, outp := range tcase.outs {
			if got[i] != outp.IsNotSpent {
				fmt.Println(got)
				fmt.Println()
				t.Fatalf("Expected %v for vout %d, got %v\n", outp.IsNotSpent, i, got[i])
			}
		}
	}
}

func TestTxMetadataReader_ReadOutputs(t *testing.T) {
	for _, tcase := range testReaderSetup() {
		for i, gotoutp := range tcase.tr.ReadOutputs(nil) {
			if gotoutp.IsNotSpent != tcase.outs[i].IsNotSpent || !bytes.Equal(gotoutp.OutputID, tcase.outs[i].OutputID) || gotoutp.Value != tcase.outs[i].Value || !bytes.Equal(gotoutp.ScriptPubKey, tcase.outs[i].ScriptPubKey) {
				t.Fatalf("Got/expected mismatch for actual output with vout: %d \n Got: %v \n Exp: %v", i, gotoutp, tcase.outs[i])
			}
		}
	}
	for _, tcase := range testReaderSetup() {
		boolvec := make([]bool, len(tcase.outs))
		for i, out := range tcase.outs {
			boolvec[i] = out.IsNotSpent
		}
		for i, gotoutp := range tcase.tr.ReadOutputs(boolvec) {
			if gotoutp.IsNotSpent != tcase.outs[i].IsNotSpent || !bytes.Equal(gotoutp.OutputID, tcase.outs[i].OutputID) || gotoutp.Value != tcase.outs[i].Value || !bytes.Equal(gotoutp.ScriptPubKey, tcase.outs[i].ScriptPubKey) {
				t.Fatalf("Got/expected mismatch for actual output with vout: %d \n Got: %v \n Exp: %v", i, gotoutp, tcase.outs[i])
			}
		}
	}
}

package core

import (
	"bytes"
	"encoding/hex"
	"plairo/params"
	"plairo/utils"
	"testing"
)

func TestTransaction_SerializeTransaction(t *testing.T) {
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
	dummyPubBytes, _ := hex.DecodeString("3059301306072a8648ce3d020106082a8648ce3d03010703420004bf3c72438b5f7a931198d7ef85c5c0df44f5d9079565f25dbdae96ae498a8942af671aaa4e5b32d701eca0aac42e98ba7b3b59469d793b4696ba4644bf9ee132")
	// parent digest should be e47125968b3b71049fbc4802d1e40a71ea1359decfabacf70b34588037d4ff0c
	tout := NewTransactionOutput(utils.CalculateSHA256Hash([]byte("parent")), 2, 1000, dummyPubBytes)

	dummysig := utils.CalculateSHA256Hash([]byte("dummySignature")) // should be b0382eb48f1497f6477942a6fe9ac60a21b34f5eaa61a6f121454190bdc9c81e
	var sighash byte = 0x01
	// appending the sighash byte to the "signature"
	dummysig = append(dummysig, sighash) // size is 33 bytes now

	tin := &TransactionInput{tout, dummysig}
	tx := NewTransaction([]*TransactionInput{tin}, []*TransactionOutput{tout})
	expectedSerializedTx, _ := hex.DecodeString("00000001e47125968b3b71049fbc4802d1e40a71ea1359decfabacf70b34588037d4ff0c000000020000000000000021b0382eb48f1497f6477942a6fe9ac60a21b34f5eaa61a6f121454190bdc9c81e010000000100000000000003e8000000000000005b3059301306072a8648ce3d020106082a8648ce3d03010703420004bf3c72438b5f7a931198d7ef85c5c0df44f5d9079565f25dbdae96ae498a8942af671aaa4e5b32d701eca0aac42e98ba7b3b59469d793b4696ba4644bf9ee132")
	if !bytes.Equal(tx.SerializeTransaction(), expectedSerializedTx) {
		t.Errorf("Invalid serialized TX.\nExp: %x\nGot: %x", expectedSerializedTx, tx.SerializeTransaction())
	}
}

func TestTransaction_GetFees(t *testing.T) {
	tout := NewTransactionOutput(utils.CalculateSHA256Hash([]byte("parent")), 2, 1000, []byte("dummypub"))
	tin := &TransactionInput{tout, []byte("dummysig")}
	newtout := NewTransactionOutput(utils.CalculateSHA256Hash([]byte("parent")), 2, 700, []byte("dummypub"))

	tx := NewTransaction([]*TransactionInput{tin}, []*TransactionOutput{newtout})
	if tx.GetFees() != 300 {
		t.Errorf("Incorrect fee value. Expected %d, got %d.\n", 300, tx.GetFees())
	}
	tout2 := NewTransactionOutput(utils.CalculateSHA256Hash([]byte("parent")), 2, 50, []byte("dummypub"))
	tin2 := &TransactionInput{tout2, []byte("dummysig")}
	tx = NewTransaction([]*TransactionInput{tin, tin2}, []*TransactionOutput{newtout})
	if tx.GetFees() != 350 {
		t.Errorf("Incorrect fee value. Expected %d, got %d.\n", 350, tx.GetFees())
	}
}

func TestTransaction_GetMinimumFees(t *testing.T) {
	tout := NewTransactionOutput(utils.CalculateSHA256Hash([]byte("parent")), 2, 1000, []byte("dummypub"))
	tin := &TransactionInput{tout, []byte("dummysig")}
	newtout := NewTransactionOutput(utils.CalculateSHA256Hash([]byte("parent")), 2, 700, []byte("dummypub"))
	tx := NewTransaction([]*TransactionInput{tin}, []*TransactionOutput{newtout})
	serializedTX := tx.SerializeTransaction()
	if tx.GetMinimumFees() != params.FeePerByte*uint64(len(serializedTX)) {
		t.Errorf("Incorrect minimum fee value. Expected %d, got %d.\n", params.FeePerByte*uint64(len(serializedTX)), tx.GetMinimumFees())
	}
}

func TestTransaction_gatherDataForSignature(t *testing.T) {
	// building first input
	// parent should be a7e64b1d8f42e11ca5e984d673adb0703a164b0872d12eb2a6004616abb2b2dd
	tout1 := NewTransactionOutput(utils.CalculateSHA256Hash([]byte("parent1")), 2, 32, []byte{0x88, 0x99, 0x01})
	tin1 := &TransactionInput{tout1, []byte{}}

	// building second input
	// parent should be c8fe5d507f207a382123d83514cfc112fdf25d7f2475d37cda0efddbd730db37
	tout2 := NewTransactionOutput(utils.CalculateSHA256Hash([]byte("parent2")), 2, 48, []byte{0x88, 0x99, 0x02})
	tin2 := &TransactionInput{tout2, []byte{0x22}}

	// building first new output
	newtout1 := NewTransactionOutput([]byte("parent"), 2, 48, []byte{0x77, 0x01})
	newtout2 := NewTransactionOutput([]byte("parent"), 2, 49, []byte{0x77, 0x02})

	tx := NewTransaction([]*TransactionInput{tin1, tin2}, []*TransactionOutput{newtout1, newtout2})
	exp, _ := hex.DecodeString("1c1f2f6bc70af15807c77ce99a81547aa6d3e0a3031896b8dfebdabf8a111509")

	if !bytes.Equal(tx.gatherSignatureDataForInput(0, SIGHASH_ALL), exp) {
		t.Errorf("Wrong signature data.\n Exp: %x\n Got %x\n", exp, tx.gatherSignatureDataForInput(0, SIGHASH_ALL))
	}
}

func TestTransaction_signInput(t *testing.T) {
	/*
	This test constructs a transaction. After signing the first input, the actual signature is checked against the public
	key and using the expected, not the actual signature msg. If there is a mismatch between the message used
	in the methods and the expected, the signature will not be deemed valid and the test will fail.
	This test also ensures the signature/validation mechanism for an input works as expected.
	 */
	// building first input
	// parent should be a7e64b1d8f42e11ca5e984d673adb0703a164b0872d12eb2a6004616abb2b2dd
	tout1 := NewTransactionOutput(utils.CalculateSHA256Hash([]byte("parent1")), 2, 32, []byte{0x88, 0x99, 0x01})
	tin1 := &TransactionInput{tout1, []byte{}}

	// building second input
	// parent should be c8fe5d507f207a382123d83514cfc112fdf25d7f2475d37cda0efddbd730db37
	tout2 := NewTransactionOutput(utils.CalculateSHA256Hash([]byte("parent2")), 2, 48, []byte{0x88, 0x99, 0x02})
	tin2 := &TransactionInput{tout2, []byte{0x22}}

	// building first new output
	newtout1 := NewTransactionOutput([]byte("parent"), 2, 48, []byte{0x77, 0x01})
	newtout2 := NewTransactionOutput([]byte("parent"), 2, 49, []byte{0x77, 0x02})

	tx := NewTransaction([]*TransactionInput{tin1, tin2}, []*TransactionOutput{newtout1, newtout2})

	privkey, pubkey, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	if err := tx.signInput(0, privkey, SIGHASH_ALL); err != nil {
		t.Fatalf("%v\n", err)
	}

	expSigMsg, _ := hex.DecodeString("1c1f2f6bc70af15807c77ce99a81547aa6d3e0a3031896b8dfebdabf8a111509")
	gotSignature := tx.inputs[0].ScriptSig[0:len(tx.inputs[0].ScriptSig)-1]

	if !utils.VerifySignature(expSigMsg, gotSignature, pubkey) {
		t.Errorf("Invalid signature.\n")
	}
}
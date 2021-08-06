package core

import (
	"bytes"
	"encoding/hex"
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

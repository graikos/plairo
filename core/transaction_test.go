package core

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"plairo/params"
	"plairo/utils"
	"testing"

	"github.com/syndtr/goleveldb/leveldb"
)

func TestTransaction_SerializeTransaction(t *testing.T) {
	dummyPubBytes, _ := hex.DecodeString("3059301306072a8648ce3d020106082a8648ce3d03010703420004bf3c72438b5f7a931198d7ef85c5c0df44f5d9079565f25dbdae96ae498a8942af671aaa4e5b32d701eca0aac42e98ba7b3b59469d793b4696ba4644bf9ee132")
	// parent digest should be e47125968b3b71049fbc4802d1e40a71ea1359decfabacf70b34588037d4ff0c
	tout := NewTransactionOutput(utils.CalculateSHA256Hash([]byte("parent")), 2, 1000, dummyPubBytes)
	toutreferred := NewTransactionOutput(utils.CalculateSHA256Hash([]byte("parent")), 2, 1000, dummyPubBytes)

	dummysig := utils.CalculateSHA256Hash([]byte("dummySignature")) // should be b0382eb48f1497f6477942a6fe9ac60a21b34f5eaa61a6f121454190bdc9c81e
	var sighash byte = 0x01
	// appending the sighash byte to the "signature"
	dummysig = append(dummysig, sighash) // size is 33 bytes now

	tin := &TransactionInput{toutreferred, dummysig}
	tx := NewTransaction([]*TransactionInput{tin}, []*TransactionOutput{tout})
	expectedSerializedTx, _ := hex.DecodeString("00000001e47125968b3b71049fbc4802d1e40a71ea1359decfabacf70b34588037d4ff0c000000020000000000000021b0382eb48f1497f6477942a6fe9ac60a21b34f5eaa61a6f121454190bdc9c81e010000000100000000000003e8000000000000005b3059301306072a8648ce3d020106082a8648ce3d03010703420004bf3c72438b5f7a931198d7ef85c5c0df44f5d9079565f25dbdae96ae498a8942af671aaa4e5b32d701eca0aac42e98ba7b3b59469d793b4696ba4644bf9ee132")
	if !bytes.Equal(tx.Serialize(), expectedSerializedTx) {
		t.Errorf("Invalid serialized TX.\nExp: %x\nGot: %x", expectedSerializedTx, tx.Serialize())
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
	serializedTX := tx.Serialize()
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
	gotSignature := tx.inputs[0].ScriptSig[0 : len(tx.inputs[0].ScriptSig)-1]

	if !utils.VerifySignature(expSigMsg, gotSignature, pubkey) {
		t.Errorf("Invalid signature.\n")
	}
}

/*
============================================
Testing transaction validation methods
============================================
*/

// mocking a chainstate database
type mockChainstate struct {
	txmap     map[string]*Transaction
	utxo      map[string]*TransactionOutput
	utxocount map[string]int
}

func (mc *mockChainstate) getOutputId(txid []byte, vout uint32) []byte {
	res := make([]byte, len(txid)+4)
	copy(res[0:len(txid)], txid)
	binary.BigEndian.PutUint32(res[len(txid):], vout)
	return utils.CalculateSHA256Hash(utils.CalculateSHA256Hash(res))
}

func (mc *mockChainstate) GetUtxo(txid []byte, vout uint32) (*TransactionOutput, bool) {
	out, ok := mc.utxo[hex.EncodeToString(mc.getOutputId(txid, vout))]
	return out, ok
}

func (mc *mockChainstate) GetTX(txid []byte) ([]byte, error) {
	mt, ok := mc.txmap[hex.EncodeToString(txid)]
	if !ok {
		return nil, leveldb.ErrNotFound
	}
	// no need to update output metadata, outputs are accessed using the GetUTXO method
	return mt.SerializeTXMetadata(), nil
}

func (mc *mockChainstate) RemoveUtxo(txid []byte, vout uint32) bool {
	delete(mc.utxo, hex.EncodeToString(mc.getOutputId(txid, vout)))
	mc.utxocount[hex.EncodeToString(txid)]--
	if mc.utxocount[hex.EncodeToString(txid)] == 0 {
		delete(mc.txmap, hex.EncodeToString(txid))
	}
	return true
}
func (mc *mockChainstate) InsertBatchTX(tx *Transaction) error {
	mc.txmap[hex.EncodeToString(tx.TXID)] = tx
	count := 0
	for i, outp := range tx.outputs {
		if outp.IsNotSpent {
			count++
		}
		mc.utxo[hex.EncodeToString(mc.getOutputId(tx.TXID, uint32(i)))] = outp
	}
	mc.utxocount[hex.EncodeToString(tx.TXID)] = count
	return nil
}
func (mc *mockChainstate) WriteBatchTX() error {
	return nil
}

func initTestCState() CState {
	old := cstate
	cstate = &mockChainstate{
		make(map[string]*Transaction),
		make(map[string]*TransactionOutput),
		make(map[string]int),
	}
	return old
}
func resetTestCState(oldstate CState) {
	cstate = oldstate
}

func createTestOutputs(num int, mark byte, optTXID []byte, optPubKey *ecdsa.PublicKey) []*TransactionOutput {
	/*
		ParentTXID: [mark]0000...0000000000 incrementing
		Initial vout: i
		Value: 2*(i+1)
		ScriptPubKey : First byte 0x99 last 4 bytes are i
	*/
	res := make([]*TransactionOutput, 0, num)
	for i := 0; i < num; i++ {
		parenttxid := optTXID
		if len(optTXID) == 0 {
			parenttxid := make([]byte, 32)
			parenttxid[0] = mark
			copy(parenttxid[28:], utils.SerializeUint32(uint32(i), false))
		}
		var spk []byte
		if optPubKey == nil {
			spk := make([]byte, 8)
			copy(spk[4:], utils.SerializeUint32(uint32(i), false))
			spk[0] = 0x99
		} else {
			spk, _ = utils.ConvertPubKeyToBytes(optPubKey)
		}
		res = append(res, NewTransactionOutput(parenttxid, uint32(i), uint64(2000*(i+1)), spk))
	}
	return res
}

func createTestInputs(outs []*TransactionOutput) []*TransactionInput {
	res := make([]*TransactionInput, len(outs))
	for i, outp := range outs {
		res[i] = &TransactionInput{outp, []byte{}}
	}
	return res
}

func scriptTestOutputs(outs []*TransactionOutput, pubkey *ecdsa.PublicKey) {
	for _, outp := range outs {
		outp.ScriptPubKey, _ = utils.ConvertPubKeyToBytes(pubkey)
	}
}

func signTestInputs(tx *Transaction, privkey *ecdsa.PrivateKey) {
	for i := range tx.inputs {
		tx.signInput(i, privkey, SIGHASH_ALL)
	}
}

func TestTransaction_ValidateTransaction(t *testing.T) {
	old := initTestCState()
	defer resetTestCState(old)

	// Test Case #1: Testing a normal, vaild transaction
	privkey1, pubkey1, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generating key pair 1: %v\n", err)
	}
	basetx1 := NewTransaction(createTestInputs(createTestOutputs(20, 0x01, nil, nil)), createTestOutputs(5, 0x02, nil, pubkey1))
	cstate.InsertBatchTX(basetx1)

	tx1 := NewTransaction(createTestInputs(createTestOutputs(2, 0x02, basetx1.TXID, pubkey1)), createTestOutputs(1, 0x03, nil, nil))
	signTestInputs(tx1, privkey1)
	if err := tx1.ValidateTransaction(); err != nil {
		t.Errorf("Error validating TX #1: %v\n", err)
	}

	// Test Case #2: Using insufficient funds as inputs
	privkey2, pubkey2, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generating key pair 2: %v\n", err)
	}
	basetx2 := NewTransaction(createTestInputs(createTestOutputs(20, 0x01, nil, nil)), createTestOutputs(1, 0x02, nil, pubkey2))
	cstate.InsertBatchTX(basetx2)

	tx2 := NewTransaction(createTestInputs(createTestOutputs(1, 0x02, basetx2.TXID, pubkey2)), createTestOutputs(2, 0x03, nil, nil))
	signTestInputs(tx2, privkey2)
	if err := tx2.ValidateTransaction(); !errors.Is(err, ErrInsufficientFunds) {
		t.Errorf("Error validating TX #2: %v\n", err)
	}

	// Test Case #3: Providing invalid signature for inputs
	_, pubkey3, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generating key pair #3: %v\n", err)
	}
	basetx3 := NewTransaction(createTestInputs(createTestOutputs(20, 0x01, nil, nil)), createTestOutputs(4, 0x02, nil, pubkey3))
	cstate.InsertBatchTX(basetx3)

	tx3 := NewTransaction(createTestInputs(createTestOutputs(3, 0x02, basetx3.TXID, pubkey3)), createTestOutputs(1, 0x03, nil, nil))
	// using wrong private key to sign inputs
	signTestInputs(tx3, privkey2)
	if err := tx3.ValidateTransaction(); !errors.Is(err, ErrInvalidSignatureProvided) {
		t.Errorf("Error validating TX #3: %v\n", err)
	}

	// Test Case #4: Using duplicate inputs
	privkey4, pubkey4, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generating key pair #4: %v\n", err)
	}
	basetx4 := NewTransaction(createTestInputs(createTestOutputs(4, 0x02, nil, nil)), createTestOutputs(2, 0x02, nil, pubkey4))
	cstate.InsertBatchTX(basetx4)

	dupIns := createTestOutputs(1, 0x02, basetx4.TXID, pubkey4)
	dupIns = append(dupIns, createTestOutputs(1, 0x02, basetx4.TXID, pubkey4)...)

	tx4 := NewTransaction(createTestInputs(dupIns), createTestOutputs(1, 0x02, nil, nil))
	signTestInputs(tx4, privkey4)
	if err := tx4.ValidateTransaction(); !errors.Is(err, ErrDuplicateInput) {
		t.Errorf("Error validatign TX #4: %v\n", err)
	}

	// Test Case #5: Modifying existing UTXO to change value
	privkey5, pubkey5, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generating key pair #5: %v\n", err)
	}
	basetx5 := NewTransaction(createTestInputs(createTestOutputs(4, 0x02, nil, nil)), createTestOutputs(2, 0x02, nil, pubkey5))
	cstate.InsertBatchTX(basetx5)
	modOut := createTestOutputs(1, 0x02, basetx5.TXID, pubkey5)
	modOut[0].Value = 7777777

	tx5 := NewTransaction(createTestInputs(modOut), createTestOutputs(1, 0x02, nil, nil))
	signTestInputs(tx5, privkey5)
	if err := tx5.ValidateTransaction(); !errors.Is(err, ErrInputOutputMismatch) {
		t.Errorf("Error validating TX #5: %v\n", err)
	}

	// Test Case #6: Using UTXO that does not exist
	privkey6, pubkey6, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generating key pair #6: %v\n", err)
	}
	basetx6 := NewTransaction(createTestInputs(createTestOutputs(4, 0x02, nil, nil)), createTestOutputs(2, 0x02, nil, pubkey6))
	cstate.InsertBatchTX(basetx6)

	tx6 := NewTransaction(createTestInputs(createTestOutputs(3, 0x02, basetx6.TXID, pubkey6)), createTestOutputs(1, 0x02, nil, nil))
	signTestInputs(tx6, privkey6)
	if err := tx6.ValidateTransaction(); !errors.Is(err, ErrNonExistentUTXO) {
		t.Errorf("Error validating TX #6: %5\n", err)
	}

	// Test Case #7: Funds used not enough to cover minimum fee
	privkey7, pubkey7, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generating key pair #7: %v\n", err)
	}
	pubkey7bytes, _ := utils.ConvertPubKeyToBytes(pubkey7)
	tout7 := &TransactionOutput{
		[]byte{},
		0,
		2,
		pubkey7bytes,
		[]byte{},
		true,
	}
	basetx7 := NewTransaction(createTestInputs(createTestOutputs(4, 0x02, nil, nil)), []*TransactionOutput{tout7})
	cstate.InsertBatchTX(basetx7)

	insTestOut := &TransactionOutput{
		basetx1.TXID,
		0,
		1,
		[]byte{},
		[]byte{},
		true,
	}
	tx7 := NewTransaction([]*TransactionInput{{tout7, []byte{}}}, []*TransactionOutput{insTestOut})
	signTestInputs(tx7, privkey7)
	if err := tx7.ValidateTransaction(); !errors.Is(err, ErrInsufficientFunds) {
		t.Errorf("Error validating TX #7: %v\n", err)
	}
}

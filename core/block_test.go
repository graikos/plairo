package core

import (
	"errors"
	"plairo/utils"
	"testing"
)

func initTestMempool() *MemPool {
	old := mempool
	// creating new mempool to use for testing purposes
	mempool = &MemPool{&memTree{}, make(map[string]uint64), make(map[string]bool)}
	return old
}

func resetTestMempool(old *MemPool) {
	mempool = old
}

func TestBlock_ValidateBlockTx(t *testing.T) {
	oldcstate := initTestCState()
	defer resetTestCState(oldcstate)
	oldmp := initTestMempool()
	defer resetTestMempool(oldmp)

	// will use this tx as a dummy coinbase, will never be validated during these tests
	cb := NewTransaction([]*TransactionInput{}, []*TransactionOutput{})

	// Test Case #0: Coinbase transaction should not be validated like the other block transactions
	b0 := NewBlock([]*Transaction{cb})
	if err := b0.ValidateBlockTx(); err != nil {
		t.Errorf("Unexpected result validating block #0: %v\n", err)
	}

	// Test Case #1: One of the transactions is invalid
	privkey1, pubkey1, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Error generating key pair #1: %v\n", err)
	}
	basetx1 := NewTransaction(createTestInputs(createTestOutputs(4, 0x01, nil, nil)), createTestOutputs(2, 0x02, nil, pubkey1))
	cstate.InsertBatchTX(basetx1)

	tx1 := NewTransaction(createTestInputs(createTestOutputs(3, 0x02, basetx1.TXID, pubkey1)), createTestOutputs(1, 0x03, nil, nil))
	signTestInputs(tx1, privkey1)

	b1 := NewBlock([]*Transaction{cb, tx1})
	if err := b1.ValidateBlockTx(); !errors.Is(err, ErrNonExistentUTXO) {
		t.Errorf("Unexpected result validating block #1: %v\n", err)
	}

	// Test Case #2: Two transactions use the same UTXO as input
	privkey2, pubkey2, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Error generating key pair #2: %v\n", err)
	}
	basetx2 := NewTransaction(createTestInputs(createTestOutputs(6, 0x02, nil, nil)), createTestOutputs(5, 0x02, nil, pubkey2))
	cstate.InsertBatchTX(basetx2)

	tx2 := NewTransaction(createTestInputs(createTestOutputs(2, 0x02, basetx2.TXID, pubkey2)), createTestOutputs(1, 0x03, nil, nil))
	tx2same := NewTransaction(createTestInputs(createTestOutputs(2, 0x02, basetx2.TXID, pubkey2)), createTestOutputs(1, 0x03, nil, nil))
	signTestInputs(tx2, privkey2)
	signTestInputs(tx2same, privkey2)

	b2 := NewBlock([]*Transaction{cb, tx2})
	// block #2 with just tx2 should pass validation
	if err := b2.ValidateBlockTx(); err != nil {
		t.Errorf("Unexpected result validating block #2: %v\n", err)
	}
	b2.allBlockTx = append(b2.allBlockTx, tx2same)
	// block #2 validation should fail when adding tx2same, which references the same UTXO as tx2
	if err := b2.ValidateBlockTx(); !errors.Is(err, ErrInvalidTxInBlock) {
		t.Errorf("Unexpected result validting block 32: %v\n", err)
	}
}

func TestBlock_ConfirmAsValid(t *testing.T) {
	oldcstate := initTestCState()
	defer resetTestCState(oldcstate)
	oldmp := initTestMempool()
	defer resetTestMempool(oldmp)

	// using a dummy empty Tx as coinbase in all blocks, should not be treated as regular transaction
	cb := NewTransaction([]*TransactionInput{}, []*TransactionOutput{})

	// Test case #0: Coinbase should be added to chainstate after confirming the block as valid
	b0 := NewBlock([]*Transaction{cb})
	if err := b0.ConfirmAsValid(); err != nil {
		t.Errorf("Error confirming block #0 as valid: %v\n", err)
	}
	if _, err := cstate.GetTX(cb.TXID); err != nil {
		t.Errorf("Error getting tx of block #0 from chainstate after confirming: %v\n", err)
	}

	// Test Case #1: A regular TX added in block
	privkey1, pubkey1, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Error generating key pair #1: %v\n", err)
	}
	basetx1 := NewTransaction(createTestInputs(createTestOutputs(5, 0x01, nil, nil)), createTestOutputs(2, 0x02, nil, pubkey1))
	cstate.InsertBatchTX(basetx1)

	tx1 := NewTransaction(createTestInputs(createTestOutputs(2, 0x02, basetx1.TXID, pubkey1)), createTestOutputs(1, 0x03, nil, nil))
	signTestInputs(tx1, privkey1)

	b1 := NewBlock([]*Transaction{cb, tx1})
	if err := b1.ConfirmAsValid(); err != nil {
		t.Errorf("Error confirming block #1 as valid: %v\n", err)
	}
	for i := range basetx1.outputs {
		if _, ok := cstate.GetUtxo(basetx1.TXID, uint32(i)); ok {
			t.Errorf("Found UTXO of basetx1 in chainstate.\n")
		}
	}
	for i := range tx1.outputs {
		if _, ok := cstate.GetUtxo(tx1.TXID, uint32(i)); !ok {
			t.Errorf("UTXOs of tx1 not found in chainstate.\n")
		}
	}
}

func TestBlock_GetBlockFees(t *testing.T) {
	oldcstate := initTestCState()
	defer resetTestCState(oldcstate)
	oldmp := initTestMempool()
	defer resetTestMempool(oldmp)

	// using a dummy coinbase tx for all blocks. This coinbase tx will never be validated in this test.
	cb := NewTransaction([]*TransactionInput{}, []*TransactionOutput{})

	// Test Case #0: Checking if coinbase will be taken into account in fee calculation
	b0 := NewBlock([]*Transaction{cb})
	if b0.GetBlockFees(true) != 0 {
		t.Errorf("Unexpected fee amount calculated for block #0.\n")
	}

	// Test Case #1: Calculating fees for block with regular transactions
	privkey1, pubkey1, err := utils.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Error generating key pair #1: %v\n", err)
	}
	basetx11 := NewTransaction(createTestInputs(createTestOutputs(5, 0x01, nil, nil)), createTestOutputs(2, 0x02, nil, pubkey1))
	basetx12 := NewTransaction(createTestInputs(createTestOutputs(5, 0x01, nil, nil)), createTestOutputs(3, 0x02, nil, pubkey1))
	cstate.InsertBatchTX(basetx11)
	cstate.InsertBatchTX(basetx12)

	// input value: 6000, output value: 2000. Expected fee: 4000
	tx11 := NewTransaction(createTestInputs(createTestOutputs(2, 0x02, basetx11.TXID, pubkey1)), createTestOutputs(1, 0x03, nil, nil))
	// input value: 12000, output value: 6000. Expected fee: 6000
	tx12 := NewTransaction(createTestInputs(createTestOutputs(3, 0x02, basetx12.TXID, pubkey1)), createTestOutputs(2, 0x03, nil, nil))
	signTestInputs(tx11, privkey1)
	signTestInputs(tx12, privkey1)

	// expected block fee: 10000
	b1 := NewBlock([]*Transaction{cb, tx11, tx12})
	if b1.GetBlockFees(true) != 10000 {
		t.Errorf("For block #1, expected %d, got %d.", 10000, b1.GetBlockFees(true))
	}
}

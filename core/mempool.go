package core

import (
	"bytes"
	"encoding/hex"
	"errors"
	"plairo/utils"
)

var ErrTxRecNotFound = errors.New("transaction record not found in mempool")
var ErrDoubleSpentOutput = errors.New("output referenced twice")
var ErrTxNotInMemPool = errors.New("transaction does not exist in mempool")

type txRecord struct {
	fee  uint64
	txid []byte
}

func (txr *txRecord) equal(txr2 *txRecord) bool {
	return txr.fee == txr2.fee && bytes.Equal(txr.txid, txr2.txid)
}

type memTreeNode struct {
	key    uint64
	txRec  *txRecord
	tx     *Transaction
	left   *memTreeNode
	right  *memTreeNode
	same   *memTreeNode
	height int
}

func createMemTreeNode(tx *Transaction) *memTreeNode {
	return &memTreeNode{
		key:    tx.GetFees(),
		txRec:  &txRecord{tx.GetFees(), tx.TXID},
		tx:     tx,
		left:   nil,
		right:  nil,
		same:   nil,
		height: 1,
	}
}

type memTree struct {
	root *memTreeNode
}

func (mt *memTree) getNodeHeight(node *memTreeNode) int {
	if node == nil {
		return 0
	}
	return node.height
}

func (mt *memTree) getNodeBalanceFactor(node *memTreeNode) int {
	if node == nil {
		return 0
	}
	return mt.getNodeHeight(node.left) - mt.getNodeHeight(node.right)
}

func (mt *memTree) rightRotate(node *memTreeNode) *memTreeNode {
	x := node.left
	x_r := x.right

	x.right = node
	node.left = x_r

	node.height = 1 + utils.Max(mt.getNodeHeight(node.left), mt.getNodeHeight(node.right))
	x.height = 1 + utils.Max(mt.getNodeHeight(x.left), mt.getNodeHeight(x.right))

	return x
}

func (mt *memTree) leftRotate(node *memTreeNode) *memTreeNode {
	y := node.right
	y_l := y.left

	y.left = node
	node.right = y_l

	node.height = 1 + utils.Max(mt.getNodeHeight(node.left), mt.getNodeHeight(node.right))
	y.height = 1 + utils.Max(mt.getNodeHeight(y.left), mt.getNodeHeight(y.right))

	return y
}

func (mt *memTree) insert(tx *Transaction) {
	newNode := createMemTreeNode(tx)

	mt.root = mt.insertAtNode(mt.root, newNode)
}

func (mt *memTree) insertAtNode(node, newNode *memTreeNode) *memTreeNode {

	// a suitable place for newNode was found, returning to previous recursion level
	if node == nil {
		return newNode
	}

	// if node with same key exists, chain them
	if newNode.key == node.key {
		tmp := node
		for tmp.same != nil {
			tmp = tmp.same
		}
		tmp.same = newNode
		return node
	}

	// if key is smaller than current, go left, else go right
	if newNode.key < node.key {
		node.left = mt.insertAtNode(node.left, newNode)
	} else {
		node.right = mt.insertAtNode(node.right, newNode)
	}

	// determining height for current node since the left or right subtree was modified
	node.height = 1 + utils.Max(mt.getNodeHeight(node.left), mt.getNodeHeight(node.right))

	// getting balance factor for current node
	// rotations will be required if balanace factor is >1 or <-1
	currentBalanceFactor := mt.getNodeBalanceFactor(node)

	if currentBalanceFactor > 1 {
		// if newNode followed the left path twice, the case is "left left"
		if newNode.key < node.left.key {
			return mt.rightRotate(node)
		} else {
			// since an imbalace is certain, the only other possible case is "left right"
			node.left = mt.leftRotate(node.left)
			return mt.rightRotate(node)
		}
	} else if currentBalanceFactor < -1 {
		// if the newNode followed the right and then the left path, the case is "right left"
		if newNode.key > node.right.key {
			node.right = mt.rightRotate(node.right)
			return mt.leftRotate(node)
		} else {
			// the only other case left is "right right", fixed by a single left rotation
			return mt.leftRotate(node)
		}
	}

	// if no rotations were needed, the current node is returned to the previous recursion level
	// since the current node is still the root of this subtree
	return node
}

func (mt *memTree) getSmallestNode(subRoot *memTreeNode) *memTreeNode {
	if subRoot == nil {
		return nil
	}
	for subRoot.left != nil {
		subRoot = subRoot.left
	}
	return subRoot
}

func (mt *memTree) removeRecord(txrec *txRecord) {
	mt.root = mt.removeAtNode(mt.root, txrec)
}

func (mt *memTree) removeAtNode(node *memTreeNode, txrec *txRecord) *memTreeNode {
	if node == nil {
		return node
	}

	// determining in search of node to delete
	if txrec.fee < node.key {
		// going left if the record being searched has a fee less than the current node
		node.left = mt.removeAtNode(node.left, txrec)
	} else if node.key > txrec.fee {
		// going right if the record fee is greater
		node.right = mt.removeAtNode(node.right, txrec)
	} else {
		// Else, a node with this fee was found.
		// To ensure the correct transaction record was found, we must also check if the txid matches.
		// Since nodes with the same fee are linked in a chain, it is possible the record we're looking for is part of the chain.
		// In this case, the node will simply be deleted from the chain and no balancing is needed.
		if !txrec.equal(node.txRec) {
			if node.same == nil {
				// the node cannot be found
				// the program will panic, since the existence of the record should be guaranteed before calling the remove method
				panic(ErrTxRecNotFound)
			}
			// in this case, we must traverse the chain to find the node
			tmp := node
			for {
				if tmp.same.txRec.equal(txrec) {
					// record was found, removing from linked list
					// tmp.same is guaranteed not to be nil, if it were, the loop would have stopped by this point
					tmp.same = tmp.same.same
					// returning the node since no change to it has been made
					return node
				}
				// moving the tmp cursor across the chain
				tmp = tmp.same
				if tmp.same == nil {
					break
				}
			}
			// the end of the chain was reached without the record being found, program will panic
			panic(ErrTxRecNotFound)
		} else {
			// the correct record was found

			if node.same != nil {
				// replacing node with next in chain
				tmp := node.same
				tmp.left = node.left
				tmp.right = node.right
				node = nil
				return tmp
			}

			// checking for at most one child case
			if node.left == nil || node.right == nil {
				var tmp *memTreeNode
				if node.left == nil {
					tmp = node.right
				} else {
					tmp = node.left
				}

				if tmp == nil {
					// no children of node were found
					// this node is not the start of a chain
					// returning nil to remove it
					return nil
				} else {
					// exactly one child was found,
					*node = *tmp
				}
			} else {
				// exactly two children
				tmp := mt.getSmallestNode(node.right)
				// replacing properties of current node without removing pointers and positioning
				// this means replacing the key, the txRecord and the chain the new node may have
				node.key = tmp.key
				node.txRec = tmp.txRec
				node.same = tmp.same
				// the chain of the leaf should be brought up together with the leaf
				// it needs to be removed from the leaf
				tmp.same = nil
				// now removing this node from the right subtree
				node.right = mt.removeAtNode(node.right, tmp.txRec)
			}
		}
	}

	if node == nil {
		return node
	}

	// recalculating height for current node
	node.height = 1 + utils.Max(mt.getNodeHeight(node.left), mt.getNodeHeight(node.right))

	balanceFactor := mt.getNodeBalanceFactor(node)

	// checking for left subtree imbalance
	if balanceFactor > 1 {
		// checking for "left left" case
		if mt.getNodeBalanceFactor(node.left) >= 0 {
			return mt.rightRotate(node)
		} else {
			// "left right" case
			node.left = mt.leftRotate(node.left)
			return mt.rightRotate(node)
		}
	} else if balanceFactor < -1 {
		if mt.getNodeBalanceFactor(node.right) <= 0 {
			// "right right" case
			return mt.leftRotate(node)
		} else {
			// "right left" case
			node.right = mt.rightRotate(node.right)
			return mt.leftRotate(node)
		}
	}

	// returning the node since no balancing was needed and no changes were made to the subtree root
	return node
}

func (mt *memTree) getMaxElements(node *memTreeNode, out chan<- interface{}, remaining *int) {
	if node == nil {
		return
	}
	// order of traversal should be max to min
	// using reverse in-order (right subtree - root - left subtree)
	mt.getMaxElements(node.right, out, remaining)

	// the check for remaining should be done here, since the above call may have used up all the remaining
	if *remaining == 0 {
		return
	}

	// now checking current node
	// sending the tree node first
	out <- node.tx
	*remaining--
	// remaining nodes should be selected from the chain before moving to the next tree node
	tmp := node
	for *remaining > 0 && tmp.same != nil {
		tmp = tmp.same
		out <- tmp.tx
		*remaining--
	}
	mt.getMaxElements(node.left, out, remaining)
}

type MemPool struct {
	internalTree   *memTree
	txmap          map[string]uint64
	outsReferenced map[string]bool
}

func (mp *MemPool) AddTX(tx *Transaction) error {
	// validating TX before adding (this includes fee requirements)
	if err := tx.ValidateTransaction(); err != nil {
		return err
	}
	// checking for double-spend with other transactions in the mempool
	// two iterations will be needed again to ensure a failure at later outputs won't leave behind
	// outputs marked as seen
	for _, inp := range tx.inputs {
		_, ok := mp.outsReferenced[hex.EncodeToString(inp.OutputReferred.OutputID)]
		// if output has been referenced by a transaction already in the mempool, reject the transaction
		if ok {
			return ErrDoubleSpentOutput
		}
	}

	// second iteration to update outputs referenced
	for _, inp := range tx.inputs {
		mp.outsReferenced[hex.EncodeToString(inp.OutputReferred.OutputID)] = true
	}
	// indexing transaction in internal memory pool map
	mp.txmap[hex.EncodeToString(tx.TXID)] = tx.GetFees()
	// inserting in internal tree
	mp.internalTree.insert(tx)
	return nil
}

func (mp *MemPool) RemoveTX(tx *Transaction) error {
	// checking if transaction exists in mempool before trying to remove from internal tree
	if _, ok := mp.txmap[hex.EncodeToString(tx.TXID)]; !ok {
		return ErrTxNotInMemPool
	}
	// removing from internal tree
	mp.internalTree.removeRecord(&txRecord{tx.GetFees(), tx.TXID})
	// deleting transaction from internal map
	delete(mp.txmap, hex.EncodeToString(tx.TXID))
	// removing referenced outputs
	for _, inp := range tx.inputs {
		delete(mp.outsReferenced, hex.EncodeToString(inp.OutputReferred.OutputID))
	}
	return nil
}

func (mp *MemPool) GetMaxTXs(out chan<- interface{}, noOfTxToGet int) {
	mp.internalTree.getMaxElements(mp.internalTree.root, out, &noOfTxToGet)
}

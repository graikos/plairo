package params

import (
	"crypto/ecdsa"
)

var (
	GenesisRecipientPubKey *ecdsa.PublicKey
	GenesisMerkleRoot      []byte
	GenesisTargetBits      uint32
	GenesisNonce           uint32
)

// TODO: Initialize values with real genesis block parameters

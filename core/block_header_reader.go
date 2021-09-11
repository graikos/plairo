package core

import "plairo/utils"

type BlockHeaderReader struct {
	blockHeader []byte
}

func (br *BlockHeaderReader) ReadPreviousHash() []byte {
	// since sha256 is used, the digest size in bytes is 32
	return br.blockHeader[:32]
}

func (br *BlockHeaderReader) ReadMerkleRoot() []byte {
	// since the merkle root is a sha256 digest, the size in bytes is again 32
	return br.blockHeader[32:64]
}

func (br *BlockHeaderReader) ReadTimestamp() int64 {
	// size of int64 is 8 bytes
	return int64(utils.DeserializeUint64(br.blockHeader[64:72], false))
}

func (br *BlockHeaderReader) ReadTargetBits() uint32 {
	return utils.DeserializeUint32(br.blockHeader[72:76], false)
}

func (br *BlockHeaderReader) ReadNonce() uint32 {
	return utils.DeserializeUint32(br.blockHeader[76:80], false)
}
package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
)

func GenerateKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	privkey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generating key pair: %v", err)
	}
	pubkey := &privkey.PublicKey
	return privkey, pubkey, nil
}

func GenerateSignature(msg []byte, privkey *ecdsa.PrivateKey) ([]byte, error) {
	hsh := CalculateSHA256Hash(msg)
	signature, err := ecdsa.SignASN1(rand.Reader, privkey, hsh)
	if err != nil {
		return nil, fmt.Errorf("generating signature: %v", err)
	}
	return signature, nil
}

func VerifySignature(msg, signature []byte, pubkey *ecdsa.PublicKey) bool {
	hsh := CalculateSHA256Hash(msg)
	return ecdsa.VerifyASN1(pubkey, hsh, signature)
}

package utils

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
)

func GenerateSignature(msg []byte, privkey *ecdsa.PrivateKey) ([]byte, error) {
	hsh := CalculateSHA256Hash(msg)
	signature, err := ecdsa.SignASN1(rand.Reader, privkey, hsh)
	if err != nil {
		return nil, fmt.Errorf("error generating signature: ")
	}
	return signature, nil
}

func VerifySignature(msg, signature []byte, pubkey *ecdsa.PublicKey) bool {
	hsh := CalculateSHA256Hash(msg)
	return ecdsa.VerifyASN1(pubkey, hsh, signature)
}

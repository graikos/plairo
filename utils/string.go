package utils

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
	"fmt"
)

func ConvertPubKeyToString(pubkey *ecdsa.PublicKey) (string, error) {
	encpubkey, err := x509.MarshalPKIXPublicKey(pubkey)
	if err != nil {
		return "", fmt.Errorf("could not convert pubkey to string: ")
	}
	return hex.EncodeToString(encpubkey), nil
}

func ConvertPrivKeyToString(privkey *ecdsa.PrivateKey) (string, error) {
	encprivkey, err := x509.MarshalECPrivateKey(privkey)
	if err != nil {
		return "", fmt.Errorf("could not convert privkey to string: ")
	}
	return hex.EncodeToString(encprivkey), nil
}

// TODO: converting from string to key should take into acount hex and bytes
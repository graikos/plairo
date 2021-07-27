package utils

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
)

func ConvertPubKeyToString(pubkey *ecdsa.PublicKey) (string, error) {
	encpubkey, err := x509.MarshalPKIXPublicKey(pubkey)
	if err != nil {
		return "", fmt.Errorf("marshalling public key: %v", err)
	}
	return hex.EncodeToString(encpubkey), nil
}

func ConvertPrivKeyToString(privkey *ecdsa.PrivateKey) (string, error) {
	encprivkey, err := x509.MarshalECPrivateKey(privkey)
	if err != nil {
		return "", fmt.Errorf("marshalling ecp private key: %v", err)
	}
	return hex.EncodeToString(encprivkey), nil
}


func ConvertStringToPubKey(keystr string) (*ecdsa.PublicKey, error) {
	keybytes, err := hex.DecodeString(keystr)
	if err != nil {
		return nil, fmt.Errorf("decoding key to byte slice: %v", err)
	}
	res, err := x509.ParsePKIXPublicKey(keybytes)
	if err != nil {
		return nil, fmt.Errorf("parsing public key: %v", err)
	}
	pubkey, ok := res.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("getting pubkey after parsing")
	}
	return pubkey, nil
}

func ConvertStringToPrivKey(keystr string) (*ecdsa.PrivateKey, error) {
	keybytes, err := hex.DecodeString(keystr)
	if err != nil {
		return nil, fmt.Errorf("decoding key to byte slice: %v", err)
	}
	return x509.ParseECPrivateKey(keybytes)
}

func ConvertBytesToPubKey(keybytes []byte) (*ecdsa.PublicKey, error) {
	res, err := x509.ParsePKIXPublicKey(keybytes)
	if err != nil {
		return nil, fmt.Errorf("parsing public key: %v", err)
	}
	pubkey, ok := res.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("getting pubkey after parsing")
	}
	return pubkey, nil
}

func ConvertBytesToPrivKey(keybytes []byte) (*ecdsa.PrivateKey, error) {
	return x509.ParseECPrivateKey(keybytes)
}

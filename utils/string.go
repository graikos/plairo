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

func ConvertPubKeyToBytes(pubkey *ecdsa.PublicKey) ([]byte, error) {
	encpubkey, err := x509.MarshalPKIXPublicKey(pubkey)
	if err != nil {
		return nil, fmt.Errorf("marshalling public key: %v", err)
	}
	return encpubkey, nil
}

func ConvertPrivKeyToBytes(privkey *ecdsa.PrivateKey) ([]byte, error) {
	encprivkey, err := x509.MarshalECPrivateKey(privkey)
	if err != nil {
		return nil, fmt.Errorf("marshalling ecp private key: %v", err)
	}

	return encprivkey, nil
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

func ExpandBits(bits []byte) []byte {
	/*
	Bits size is currently set to be 4 bytes.
	First byte is the exponent, meaning the size of the target in bytes
	The other three bytes are the coefficient, meaning the first three bytes of the target.
	The above is padded with leading zeroes to reach 32 bytes in size, since a comparison
	with a SHA-256 hash will be needed.
	Example: 0x04aabbcc
	Exponent is 0x04
	Coefficient is 0xaabbcc
	Result should be 0x00000000000000000000000000000000000000000000000000000000aabbcc00
	 */
	res := make([]byte, 32)
	exp := int(bits[0])
	copy(res[32-exp:], bits[1:])
	return res
}

package tests

import (
	"plairo/utils"
	"testing"
)

func TestSignAndVerify(t *testing.T) {
	privkey, pubkey, err := utils.GenerateKeyPair()
	if err != nil {
		t.Error(err)
	}
	testmsg := "test_message"
	signature, err := utils.GenerateSignature([]byte(testmsg), privkey)
	if err != nil {
		t.Error(err)
	}
	if !utils.VerifySignature([]byte(testmsg), signature, pubkey) {
		t.Error("Signature is not valid.")
	}
}

func TestKeyToStringConversions(t *testing.T) {
	privkey, pubkey, err := utils.GenerateKeyPair()
	if err != nil {
		t.Error(err)
	}
	pubkeystr, err := utils.ConvertPubKeyToString(pubkey)
	if err != nil {
		t.Error(err)
	}
	convertedPubkey, err := utils.ConvertStringToPubKey(pubkeystr)
	if err != nil {
		t.Error(err)
	}
	if !pubkey.Equal(convertedPubkey) {
		t.Error("Converted public key does not match original.")
	}

	// now testing privkey conversions
	privkeystr, err := utils.ConvertPrivKeyToString(privkey)
	if err != nil {
		t.Error(err)
	}
	convertedPrivKey, err := utils.ConvertStringToPrivKey(privkeystr)
	if err != nil {
		t.Error(err)
	}
	if !privkey.Equal(convertedPrivKey) {
		t.Error("Converted private key does not match original.")
	}
}

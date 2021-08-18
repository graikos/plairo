package db

import (
	"bytes"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"os"
	"testing"
)

var testDBWrapperPath string

func init() {
	homedir, _ := os.UserHomeDir()
	testDBWrapperPath = homedir + "/.plairo/wrappertest"
}

func TestObfKey(t *testing.T) {
	db := NewDBwrapper(testDBWrapperPath, true)
	defer db.Close()

	key, err := db.Get(constructObfKeyKey())
	if errors.Is(err, leveldb.ErrNotFound) {
		t.Errorf("Error getting obfuscation key from database.")
	}
	if len(key) != 8 {
		t.Errorf("Invalid obfuscation key length.")
	}
}

func TestDBwrapper_InsertGet(t *testing.T) {
	db := NewDBwrapper(testDBWrapperPath, true)
	defer db.Close()

	// testing insert and get methods
	err := db.Insert([]byte("testkey1"), []byte("testval1"))
	if err != nil {
		t.Error(err)
	}
	val, err := db.Get([]byte("testkey1"))
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(val, []byte("testval1")) {
		t.Errorf("Unexpected value for testkey1. Got: %x\n", val)
	}

	// testing a get call with non-existant key
	val, err = db.Get([]byte("non-existant"))
	if !errors.Is(err, leveldb.ErrNotFound) {
		t.Errorf("Expected value not to be found. Got value: %x\n", val)
	}
}

func TestObfuscation(t *testing.T) {
	db := NewDBwrapper(testDBWrapperPath, true)
	// setting a manual obfkey to predict obfuscation results
	db.obfuscationKey = []byte{0x0}
	expObfVal := []byte{0x01, 0x0a, 0x02, 0x0b, 0x03, 0x0c}
	obfval := db.obfuscateValue(expObfVal)
	// XOR-ing with 0x0, obfuscated value is expected to be equal to the original
	if !bytes.Equal(obfval, expObfVal) {
		t.Errorf("Expected obfuscation result to be: %x. Got %x\n", expObfVal, obfval)
	}
	// using length greater than value to check this case
	// will use 0xff so that the obfuscation result is the 1's complement of the original value
	db.obfuscationKey = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	obfval = db.obfuscateValue(expObfVal)
	// calculating complement
	for i, _ := range expObfVal {
		expObfVal[i] = ^expObfVal[i]
	}
	if !bytes.Equal(obfval, expObfVal) {
		t.Errorf("Expected obfuscation result to be: %x. Got %x\n", expObfVal, obfval)
	}
}

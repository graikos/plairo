package db

import (
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
	"math/rand"
)

type DBwrapper struct {
	IsObfuscated bool
	db *leveldb.DB
	obfuscationKey []byte
}

// NewDBwrapper creates a new DBwrapper struct
func NewDBwrapper(dbpath string, isObfuscated bool) *DBwrapper {
	db, err := leveldb.OpenFile(dbpath, nil)
	if err != nil {
		panic(err)
	}
	var obfkey []byte
	if isObfuscated {
		// looking up the obfuscation key in the db
		obfkey, err := db.Get(constructObfKeyKey(),nil)
		if err == leveldb.ErrNotFound {
			// if not found, generating a new one and saving in the db
			obfkey = generateObfuscationKey()
			err = db.Put(constructObfKeyKey(), obfkey, nil)
			if err != nil {
				// if the Put failed, then the db is not ok
				panic(err)
			}
		}
	}
	return &DBwrapper{IsObfuscated: isObfuscated, db: db, obfuscationKey: obfkey}
}

func (d *DBwrapper) obfuscateValue(value []byte) []byte {
	res := make([]byte, len(value))
	// XORing the value with the obfuscation key
	for i, val := range value {
		// obf key will be repeated if length exceeded
		res[i] = val ^ d.obfuscationKey[i % len(d.obfuscationKey)]
	}
	// returning an obfuscated copy to keep original value intact
	return res
}

func (d *DBwrapper) Insert(key, value []byte) error {
	if d.IsObfuscated {
		// obfuscating key and value stored
		return d.db.Put(d.obfuscateValue(key), d.obfuscateValue(value), nil)
	} else {
		return d.db.Put(key, value, nil)
	}
}

func (d *DBwrapper) Get(key []byte) ([]byte, error) {
	if d.IsObfuscated {
		val, err := d.db.Get(d.obfuscateValue(key), nil)
		// double obfuscation reveals the original content
		return d.obfuscateValue(val), err
	} else {
		return d.db.Get(key, nil)
	}
}

func (d *DBwrapper) Remove(key []byte) error {
	if d.IsObfuscated {
		return d.db.Delete(d.obfuscateValue(key), nil)
	} else {
		return d.db.Delete(key, nil)
	}
}

func (d *DBwrapper) Close() {
	d.db.Close()
}

// generateObfuscationKey generates 8 random bytes to be used as obfuscation key
func generateObfuscationKey() []byte {
	// generating a random 8 byte number
	rnd := rand.Int63()
	res := make([]byte, 8)
	// now a slice with 8 random bytes
	binary.BigEndian.PutUint64(res, uint64(rnd))
	return res
}

// constructObfKeyKey constructs the key under which the obfuscation key will be saved in the database
func constructObfKeyKey() []byte{
	b := make([]byte, 2, 2+len("obfuscate_key"))
	// will use the bitcoin identifier bits
	b[0] = 0x0e
	b[1] = 0x00
	b = append(b, []byte("obfuscate_key")...)
	return b
}
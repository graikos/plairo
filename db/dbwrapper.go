package db

import (
	"encoding/binary"
	"errors"
	"math/rand"

	"github.com/syndtr/goleveldb/leveldb"
)

type KeyType byte

const (
	TxKey         = KeyType('c')
	BlockIndexKey = KeyType('b')
	FileInfoKey   = KeyType('f')
	TxIndexKey    = KeyType('t')
)

func buildKey(keyType KeyType, data []byte) []byte {
	bkey := make([]byte, 1+len(data))
	bkey[0] = byte(keyType)
	copy(bkey[1:], data)
	return bkey
}

type DBwrapper struct {
	IsObfuscated   bool
	db             *leveldb.DB
	obfuscationKey []byte
	currentBatch   *leveldb.Batch // NOTE: This implementation is blocking if concurrent batch actions are needed
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
		obfkey, err = db.Get(constructObfKeyKey(), nil)
		if errors.Is(err, leveldb.ErrNotFound) {
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
		res[i] = val ^ d.obfuscationKey[i%len(d.obfuscationKey)]
	}
	// returning an obfuscated copy to keep original value intact
	return res
}

func (d *DBwrapper) Insert(key, value []byte) error {
	if d.IsObfuscated {
		// obfuscating key and value stored
		return d.db.Put(key, d.obfuscateValue(value), nil)
	} else {
		return d.db.Put(key, value, nil)
	}
}

func (d *DBwrapper) PutInBatch(key, value []byte) {
	// initializes a batch and adds to it
	if d.currentBatch == nil {
		d.currentBatch = new(leveldb.Batch)
	}
	d.currentBatch.Put(key, value)
}

// WriteBatch performs the batch write and resets the batch field
func (d *DBwrapper) WriteBatch() error {
	// writing the batch
	err := d.db.Write(d.currentBatch, nil)
	if err != nil {
		return err
	}
	// resetting the field
	d.currentBatch = nil
	return nil
}

func (d *DBwrapper) Get(key []byte) ([]byte, error) {
	if d.IsObfuscated {
		val, err := d.db.Get(key, nil)
		// double obfuscation reveals the original content
		return d.obfuscateValue(val), err
	} else {
		return d.db.Get(key, nil)
	}
}

func (d *DBwrapper) Remove(key []byte) error {
	return d.db.Delete(key, nil)
}

func (d *DBwrapper) PageInsert(key, value []byte, maxPageSize int) error {
	// appending page number byte
	key = append(key, 0)

	rem := len(value)
	idx := 0
	for rem > maxPageSize {
		// if page is continued, add an extra byte
		if err := d.Insert(key, append(value[idx:idx+maxPageSize], 0)); err != nil {
			return err
		}
		// increment page number in key
		key[len(key)-1]++
		// updating remaining data length and current index
		rem -= maxPageSize
		idx += maxPageSize
	}
	if err := d.Insert(key, value[idx:]); err != nil {
		return err
	}
	return nil
}

func (d *DBwrapper) PageGet(key []byte, maxPageSize int) ([]byte, bool) {
	key = append(key, 0)
	var res []byte
	for {
		data, err := d.Get(key)
		if err != nil {
			return nil, false
		}
		if len(data) != maxPageSize+1 {
			// if page is not continued, break
			res = append(res, data...)
			break
		}
		res = append(res, data[:len(data)-1]...)
		key[len(key)-1]++
	}
	return res, true
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
func constructObfKeyKey() []byte {
	b := make([]byte, 2, 2+len("obfuscate_key"))
	// will use the bitcoin identifier bits
	b[0] = 0x0e
	b[1] = 0x00
	b = append(b, []byte("obfuscate_key")...)
	return b
}

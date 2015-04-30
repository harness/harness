package bolt

import (
	"bytes"

	"github.com/boltdb/bolt"
	"github.com/youtube/vitess/go/bson"
)

func encode(v interface{}) ([]byte, error) {
	return bson.Marshal(v)
}

func decode(raw []byte, v interface{}) error {
	return bson.Unmarshal(raw, v)
}

func get(t *bolt.Tx, bucket, key []byte, v interface{}) error {
	raw := t.Bucket(bucket).Get(key)
	if raw == nil {
		return ErrKeyNotFound
	}
	return bson.Unmarshal(raw, v)
}

func raw(t *bolt.Tx, bucket, key []byte) ([]byte, error) {
	raw := t.Bucket(bucket).Get(key)
	if raw == nil {
		return nil, ErrKeyNotFound
	}
	return raw, nil
}

func update(t *bolt.Tx, bucket, key []byte, v interface{}) error {
	raw, err := encode(v)
	if err != nil {
		t.Rollback()
		return err
	}
	return t.Bucket(bucket).Put(key, raw)
}

func insert(t *bolt.Tx, bucket, key []byte, v interface{}) error {
	raw, err := encode(v)
	if err != nil {
		t.Rollback()
		return err
	}
	// verify the key does not already exists
	// in the bucket. If exists, fail
	if t.Bucket(bucket).Get(key) != nil {
		return ErrKeyExists
	}
	return t.Bucket(bucket).Put(key, raw)
}

func delete(t *bolt.Tx, bucket, key []byte) error {
	return t.Bucket(bucket).Delete(key)
}

func push(t *bolt.Tx, bucket, index, value []byte) error {
	var keys [][]byte
	err := get(t, bucket, index, &keys)
	if err != nil && err != ErrKeyNotFound {
		return err
	}
	// we shouldn't add a key that already exists
	for _, key := range keys {
		if bytes.Equal(key, value) {
			return nil
		}
	}
	keys = append(keys, value)
	return update(t, bucket, index, &keys)
}

func splice(t *bolt.Tx, bucket, index, value []byte) error {
	var keys [][]byte
	err := get(t, bucket, index, &keys)
	if err != nil && err != ErrKeyNotFound {
		return err
	}

	for i, key := range keys {
		if bytes.Equal(key, value) {
			keys = keys[:i+copy(keys[i:], keys[i+1:])]
			break
		}
	}

	return update(t, bucket, index, &keys)
}

func deleteWithPrefix(t *bolt.Tx, bucket, prefix []byte) error {
	var err error

	c := t.Bucket(bucket).Cursor()
	for k, _ := c.Seek(prefix); bytes.HasPrefix(k, prefix); k, _ = c.Next() {
		err = c.Delete()
		if err != nil {
			break
		}
	}

	// only error here is if our Tx is read-only
	return err
}

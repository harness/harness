package bolt

import (
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

package bolt

import (
	"github.com/youtube/vitess/go/bson"
	"github.com/boltdb/bolt"
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
	err = t.Bucket(bucket).Put(key, raw)
	if err != nil {
		return err
	}
	return nil
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
	err = t.Bucket(bucket).Put(key, raw)
	if err != nil {
		return err
	}
	return nil
}

func delete(t *bolt.Tx, bucket, key []byte) error {
	err := t.Bucket(bucket).Delete(key)
	if err != nil {
		return err
	}
	return nil
}

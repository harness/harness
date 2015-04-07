package bolt

import "github.com/youtube/vitess/go/bson"

func encode(v interface{}) ([]byte, error) {
	return bson.Marshal(v)
}

func decode(raw []byte, v interface{}) error {
	return bson.Unmarshal(raw, v)
}

func get(db *DB, bucket, key []byte, v interface{}) error {
	t, err := db.Begin(false)
	if err != nil {
		return err
	}
	defer t.Rollback()
	raw := t.Bucket(bucket).Get(key)
	if raw == nil {
		return ErrKeyNotFound
	}
	return bson.Unmarshal(raw, v)
}

func raw(db *DB, bucket, key []byte) ([]byte, error) {
	t, err := db.Begin(false)
	if err != nil {
		return nil, err
	}
	defer t.Rollback()
	raw := t.Bucket(bucket).Get(key)
	if raw == nil {
		return nil, ErrKeyNotFound
	}
	return raw, nil
}

func update(db *DB, bucket, key []byte, v interface{}) error {
	t, err := db.Begin(true)
	if err != nil {
		return err
	}
	raw, err := encode(v)
	if err != nil {
		t.Rollback()
		return err
	}
	err = t.Bucket(bucket).Put(key, raw)
	if err != nil {
		t.Rollback()
		return err
	}
	return t.Commit()
}

func insert(db *DB, bucket, key []byte, v interface{}) error {
	t, err := db.Begin(true)
	if err != nil {
		return err
	}
	raw, err := encode(v)
	if err != nil {
		t.Rollback()
		return err
	}
	// verify the key does not already exists
	// in the bucket. If exists, fail
	if t.Bucket(bucket).Get(key) != nil {
		t.Rollback()
		return ErrKeyExists
	}
	err = t.Bucket(bucket).Put(key, raw)
	if err != nil {
		t.Rollback()
		return err
	}
	return t.Commit()
}

func delete(db *DB, bucket, key []byte) error {
	t, err := db.Begin(true)
	if err != nil {
		return err
	}
	err = t.Bucket(bucket).Delete(key)
	if err != nil {
		t.Rollback()
		return err
	}
	return t.Commit()
}

package builtin

import (
	"github.com/boltdb/bolt"
	"github.com/drone/drone/common"
)

// Token returns the token for the given user and label.
func (db *DB) Token(user, label string) (*common.Token, error) {
	token := &common.Token{}
	key := []byte(user + "/" + label)

	err := db.View(func(t *bolt.Tx) error {
		return get(t, bucketTokens, key, token)
	})
	return token, err
}

// TokenList returns a list of all tokens for the given
// user login.
func (db *DB) TokenList(login string) ([]*common.Token, error) {
	tokens := []*common.Token{}
	userkey := []byte(login)
	err := db.Update(func(t *bolt.Tx) error {
		// get the index of user tokens and unmarshal
		// to a string array.
		var keys [][]byte
		err := get(t, bucketUserTokens, userkey, &keys)
		if err != nil && err != ErrKeyNotFound {
			return err
		}
		// for each item in the index, get the repository
		// and append to the array
		for _, key := range keys {
			token := &common.Token{}
			raw := t.Bucket(bucketTokens).Get(key)
			err = decode(raw, token)
			if err != nil {
				return err
			}
			tokens = append(tokens, token)
		}
		return nil
	})
	return tokens, err
}

// SetToken inserts a new user token in the datastore.
func (db *DB) SetToken(token *common.Token) error {
	key := []byte(token.Login + "/" + token.Label)
	return db.Update(func(t *bolt.Tx) error {
		err := push(t, bucketUserTokens, []byte(token.Login), key)
		if err != nil {
			return err
		}
		return insert(t, bucketTokens, key, token)
	})
}

// DelToken deletes the token.
func (db *DB) DelToken(token *common.Token) error {
	key := []byte(token.Login + "/" + token.Label)
	return db.Update(func(t *bolt.Tx) error {
		err := splice(t, bucketUserTokens, []byte(token.Login), key)
		if err != nil {
			return err
		}
		return delete(t, bucketTokens, key)
	})
}

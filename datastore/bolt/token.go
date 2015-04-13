package bolt

import (
	"github.com/boltdb/bolt"
	"github.com/drone/drone/common"
)

// GetToken gets a token by sha value.
func (db *DB) GetToken(user, label string) (*common.Token, error) {
	token := &common.Token{}
	key := []byte(user + "/" + label)

	err := db.View(func(t *bolt.Tx) error {
		return get(t, bucketTokens, key, token)
	})

	return token, err
}

// InsertToken inserts a new user token in the datastore.
// If the token already exists and error is returned.
func (db *DB) InsertToken(token *common.Token) error {
	key := []byte(token.Login + "/" + token.Label)
	return db.Update(func(t *bolt.Tx) error {
		return insert(t, bucketTokens, key, token)
	})
	// TODO(bradrydzewski) add token to users_token index
}

// DeleteUser deletes the token.
func (db *DB) DeleteToken(token *common.Token) error {
	key := []byte(token.Login + "/" + token.Label)
	return db.Update(func(t *bolt.Tx) error {
		return delete(t, bucketUser, key)
	})
}

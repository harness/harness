package bolt

import (
	"github.com/drone/drone/common"
)

// GetToken gets a token by sha value.
func (db *DB) GetToken(sha string) (*common.Token, error) {
	token := &common.Token{}
	key := []byte(sha)
	err := get(db, bucketTokens, key, token)
	return token, err
}

// InsertToken inserts a new user token in the datastore.
// If the token already exists and error is returned.
func (db *DB) InsertToken(token *common.Token) error {
	key := []byte(token.Sha)
	return insert(db, bucketTokens, key, token)
	// TODO(bradrydzewski) add token to users_token index
}

// DeleteUser deletes the token.
func (db *DB) DeleteToken(token *common.Token) error {
	key := []byte(token.Sha)
	return delete(db, bucketUser, key)
}

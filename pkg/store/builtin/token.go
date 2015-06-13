package builtin

import (
	"database/sql"

	"github.com/drone/drone/pkg/types"
)

type Tokenstore struct {
	*sql.DB
}

func NewTokenstore(db *sql.DB) *Tokenstore {
	return &Tokenstore{db}
}

// Token returns a token by ID.
func (db *Tokenstore) Token(id int64) (*types.Token, error) {
	return getToken(db, rebind(stmtTokenSelect), id)
}

// TokenLabel returns a token by label
func (db *Tokenstore) TokenLabel(user *types.User, label string) (*types.Token, error) {
	return getToken(db, rebind(stmtTokenSelectTokenUserLabel), user.ID, label)
}

// TokenList returns a list of all user tokens.
func (db *Tokenstore) TokenList(user *types.User) ([]*types.Token, error) {
	return getTokens(db, rebind(stmtTokenSelectTokenUserId), user.ID)
}

// AddToken inserts a new token into the datastore.
// If the token label already exists for the user
// an error is returned.
func (db *Tokenstore) AddToken(token *types.Token) error {
	return createToken(db, rebind(stmtTokenInsert), token)
}

// DelToken removes the DelToken from the datastore.
func (db *Tokenstore) DelToken(token *types.Token) error {
	var _, err = db.Exec(rebind(stmtTokenDelete), token.ID)
	return err
}

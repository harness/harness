package builtin

import (
	common "github.com/drone/drone/pkg/types"
	"github.com/russross/meddler"
)

type Tokenstore struct {
	meddler.DB
}

func NewTokenstore(db meddler.DB) *Tokenstore {
	return &Tokenstore{db}
}

// Token returns a token by ID.
func (db *Tokenstore) Token(id int64) (*common.Token, error) {
	var token = new(common.Token)
	var err = meddler.Load(db, tokenTable, token, id)
	return token, err
}

// TokenLabel returns a token by label
func (db *Tokenstore) TokenLabel(user *common.User, label string) (*common.Token, error) {
	var token = new(common.Token)
	var err = meddler.QueryRow(db, token, rebind(tokenLabelQuery), user.ID, label)
	return token, err
}

// TokenList returns a list of all user tokens.
func (db *Tokenstore) TokenList(user *common.User) ([]*common.Token, error) {
	var tokens []*common.Token
	var err = meddler.QueryAll(db, &tokens, rebind(tokenListQuery), user.ID)
	return tokens, err
}

// AddToken inserts a new token into the datastore.
// If the token label already exists for the user
// an error is returned.
func (db *Tokenstore) AddToken(token *common.Token) error {
	return meddler.Insert(db, tokenTable, token)
}

// DelToken removes the DelToken from the datastore.
func (db *Tokenstore) DelToken(token *common.Token) error {
	var _, err = db.Exec(rebind(tokenDeleteStmt), token.ID)
	return err
}

// Token table name in database.
const tokenTable = "tokens"

// SQL query to retrieve a token by label.
const tokenLabelQuery = `
SELECT *
FROM tokens
WHERE user_id     = ?
  AND token_label = ?
LIMIT 1
`

// SQL query to retrieve a list of user tokens.
const tokenListQuery = `
SELECT *
FROM tokens
WHERE user_id = ?
ORDER BY token_label ASC
`

// SQL statement to delete a Token by ID.
const tokenDeleteStmt = `
DELETE FROM tokens
WHERE token_id=?
`

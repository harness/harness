package server

import (
	"github.com/gin-gonic/gin"

	// "github.com/drone/drone/common"
)

// POST /api/user/tokens
func PostToken(c *gin.Context) {
	// 1. generate a unique, random password
	// 2. take a hash of the password, and store in the database
	// 3. return the random password to the UI and instruct the user to copy it
}

// DELETE /api/user/tokens/:sha
func DelToken(c *gin.Context) {
	store := ToDatastore(c)
	user := ToUser(c)
	hash := c.Params.ByName("hash")
	token, err := store.GetToken(hash)
	if err != nil {
		c.Fail(404, err)
	}
	err = store.DeleteToken(token)
	if err != nil {
		c.Fail(400, err)
	}

	// TODO(bradrydzewski) this should be encapsulated
	// in our database code, since this feels like a
	// database-specific implementation.
	delete(user.Tokens, token.Sha)
	err = store.UpdateUser(user)
	if err != nil {
		c.Fail(400, err)
	} else {
		c.Writer.WriteHeader(200)
	}
}

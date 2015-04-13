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

// DELETE /api/user/tokens/:label
func DelToken(c *gin.Context) {
	store := ToDatastore(c)
	user := ToUser(c)
	label := c.Params.ByName("label")
	token, err := store.GetToken(user.Login, label)
	if err != nil {
		c.Fail(404, err)
	}
	err = store.DeleteToken(token)
	if err != nil {
		c.Fail(400, err)
	}
}

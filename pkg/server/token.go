package server

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	common "github.com/drone/drone/pkg/types"
)

// POST /api/user/tokens
func PostToken(c *gin.Context) {
	sess := ToSession(c)
	store := ToDatastore(c)
	user := ToUser(c)

	in := &common.Token{}
	if !c.BindWith(in, binding.JSON) {
		return
	}

	token := &common.Token{}
	token.Label = in.Label
	token.UserID = user.ID
	// token.Repos = in.Repos
	// token.Scopes = in.Scopes
	token.Login = user.Login
	token.Kind = common.TokenUser
	token.Issued = time.Now().UTC().Unix()

	err := store.AddToken(token)
	if err != nil {
		c.Fail(500, err)
		return
	}

	jwt, err := sess.GenerateToken(token)
	if err != nil {
		c.Fail(400, err)
		return
	}

	c.JSON(200, struct {
		*common.Token
		Hash string `json:"hash"`
	}{token, jwt})
}

// DELETE /api/user/tokens/:label
func DelToken(c *gin.Context) {
	store := ToDatastore(c)
	user := ToUser(c)
	label := c.Params.ByName("label")

	token, err := store.TokenLabel(user, label)
	if err != nil {
		c.Fail(404, err)
		return
	}
	err = store.DelToken(token)
	if err != nil {
		c.Fail(400, err)
		return
	}

	c.Writer.WriteHeader(200)
}

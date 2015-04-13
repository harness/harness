package server

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/drone/drone/common"
)

// POST /api/user/tokens
func PostToken(c *gin.Context) {
	store := ToDatastore(c)
	sess := ToSession(c)
	user := ToUser(c)

	in := &common.Token{}
	if !c.BindWith(in, binding.JSON) {
		return
	}

	token := &common.Token{}
	token.Label = in.Label
	token.Repos = in.Repos
	token.Scopes = in.Scopes
	token.Login = user.Login
	token.Kind = common.TokenUser

	err := store.InsertToken(token)
	if err != nil {
		c.Fail(400, err)
	}

	jwt, err := sess.GenerateToken(token)
	if err != nil {
		c.Fail(400, err)
	}
	c.String(200, jwt)
}

// DELETE /api/user/tokens
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

	c.Writer.WriteHeader(200)
}

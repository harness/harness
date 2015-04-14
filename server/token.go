package server

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/drone/drone/common"
)

// POST /api/user/tokens
func PostToken(c *gin.Context) {
	settings := ToSettings(c)
	store := ToDatastore(c)
	sess := ToSession(c)
	user := ToUser(c)

	// if a session secret is not defined there is no way to
	// generate jwt user tokens, so we must throw an error
	if settings.Session == nil || len(settings.Session.Secret) == 0 {
		c.String(500, "User tokens are not configured")
		return
	}

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
	token.Issued = time.Now().UTC().Unix()

	err := store.InsertToken(token)
	if err != nil {
		c.Fail(400, err)
	}

	jwt, err := sess.GenerateToken(token)
	if err != nil {
		c.Fail(400, err)
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

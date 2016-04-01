package api

import (
	"github.com/drone/drone/model"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/store"

	"github.com/gin-gonic/gin"
)

func PostSecret(c *gin.Context) {
	repo := session.Repo(c)

	in := &model.Secret{}
	err := c.Bind(in)
	if err != nil {
		c.String(400, "Invalid JSON input. %s", err.Error())
		return
	}
	in.ID = 0
	in.RepoID = repo.ID

	err = store.SetSecret(c, in)
	if err != nil {
		c.String(500, "Unable to persist secret. %s", err.Error())
		return
	}

	c.String(200, "")
}

func DeleteSecret(c *gin.Context) {
	repo := session.Repo(c)
	name := c.Param("secret")

	secret, err := store.GetSecret(c, repo, name)
	if err != nil {
		c.String(404, "Cannot find secret %s.", name)
		return
	}
	err = store.DeleteSecret(c, secret)
	if err != nil {
		c.String(500, "Unable to delete secret. %s", err.Error())
		return
	}

	c.String(200, "")
}

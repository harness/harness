package server

import (
	"net/http"

	"github.com/drone/drone/model"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/store"

	"github.com/gin-gonic/gin"
)

func GetGlobalSecrets(c *gin.Context) {
	secrets, err := store.GetGlobalSecretList(c)

	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var list []*model.TeamSecret

	for _, s := range secrets {
		list = append(list, s.Clone())
	}

	c.JSON(http.StatusOK, list)
}

func PostGlobalSecret(c *gin.Context) {
	in := &model.TeamSecret{}
	err := c.Bind(in)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid JSON input. %s", err.Error())
		return
	}
	in.ID = 0

	err = store.SetGlobalSecret(c, in)
	if err != nil {
		c.String(http.StatusInternalServerError, "Unable to persist global secret. %s", err.Error())
		return
	}

	c.String(http.StatusOK, "")
}

func DeleteGlobalSecret(c *gin.Context) {
	name := c.Param("secret")

	secret, err := store.GetGlobalSecret(c, name)
	if err != nil {
		c.String(http.StatusNotFound, "Cannot find secret %s.", name)
		return
	}
	err = store.DeleteGlobalSecret(c, secret)
	if err != nil {
		c.String(http.StatusInternalServerError, "Unable to delete global secret. %s", err.Error())
		return
	}

	c.String(http.StatusOK, "")
}

func GetSecrets(c *gin.Context) {
	repo := session.Repo(c)
	secrets, err := store.GetSecretList(c, repo)

	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var list []*model.RepoSecret

	for _, s := range secrets {
		list = append(list, s.Clone())
	}

	c.JSON(http.StatusOK, list)
}

func PostSecret(c *gin.Context) {
	repo := session.Repo(c)

	in := &model.RepoSecret{}
	err := c.Bind(in)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid JSON input. %s", err.Error())
		return
	}
	in.ID = 0
	in.RepoID = repo.ID

	err = store.SetSecret(c, in)
	if err != nil {
		c.String(http.StatusInternalServerError, "Unable to persist secret. %s", err.Error())
		return
	}

	c.String(http.StatusOK, "")
}

func DeleteSecret(c *gin.Context) {
	repo := session.Repo(c)
	name := c.Param("secret")

	secret, err := store.GetSecret(c, repo, name)
	if err != nil {
		c.String(http.StatusNotFound, "Cannot find secret %s.", name)
		return
	}
	err = store.DeleteSecret(c, secret)
	if err != nil {
		c.String(http.StatusInternalServerError, "Unable to delete secret. %s", err.Error())
		return
	}

	c.String(http.StatusOK, "")
}

func GetTeamSecrets(c *gin.Context) {
	team := c.Param("team")
	secrets, err := store.GetTeamSecretList(c, team)

	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var list []*model.TeamSecret

	for _, s := range secrets {
		list = append(list, s.Clone())
	}

	c.JSON(http.StatusOK, list)
}

func PostTeamSecret(c *gin.Context) {
	team := c.Param("team")

	in := &model.TeamSecret{}
	err := c.Bind(in)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid JSON input. %s", err.Error())
		return
	}
	in.ID = 0
	in.Key = team

	err = store.SetTeamSecret(c, in)
	if err != nil {
		c.String(http.StatusInternalServerError, "Unable to persist team secret. %s", err.Error())
		return
	}

	c.String(http.StatusOK, "")
}

func DeleteTeamSecret(c *gin.Context) {
	team := c.Param("team")
	name := c.Param("secret")

	secret, err := store.GetTeamSecret(c, team, name)
	if err != nil {
		c.String(http.StatusNotFound, "Cannot find secret %s.", name)
		return
	}
	err = store.DeleteTeamSecret(c, secret)
	if err != nil {
		c.String(http.StatusInternalServerError, "Unable to delete team secret. %s", err.Error())
		return
	}

	c.String(http.StatusOK, "")
}

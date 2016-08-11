package server

import (
	"net/http"

	"github.com/drone/drone/model"
	"github.com/drone/drone/store"

	"github.com/gin-gonic/gin"
)

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

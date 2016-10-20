package server

import (
	"net/http"

	"github.com/drone/drone/model"
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

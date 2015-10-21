package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/drone/drone/model"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/shared/crypto"
	"github.com/drone/drone/store"
)

func GetUsers(c *gin.Context) {
	users, err := store.GetUserList(c)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.IndentedJSON(http.StatusOK, users)
}

func GetUser(c *gin.Context) {
	user, err := store.GetUserLogin(c, c.Param("login"))
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.IndentedJSON(http.StatusOK, user)
}

func PatchUser(c *gin.Context) {
	me := session.User(c)
	in := &model.User{}
	err := c.Bind(in)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	user, err := store.GetUserLogin(c, c.Param("login"))
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	user.Admin = in.Admin
	user.Active = in.Active

	// cannot update self
	if me.ID == user.ID {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	err = store.UpdateUser(c, user)
	if err != nil {
		c.AbortWithStatus(http.StatusConflict)
		return
	}

	c.IndentedJSON(http.StatusOK, user)
}

func PostUser(c *gin.Context) {
	in := &model.User{}
	err := c.Bind(in)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	user := &model.User{}
	user.Login = in.Login
	user.Email = in.Email
	user.Admin = in.Admin
	user.Avatar = in.Avatar
	user.Active = true
	user.Hash = crypto.Rand()

	err = store.CreateUser(c, user)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.IndentedJSON(http.StatusOK, user)
}

func DeleteUser(c *gin.Context) {
	me := session.User(c)

	user, err := store.GetUserLogin(c, c.Param("login"))
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// cannot delete self
	if me.ID == user.ID {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	err = store.DeleteUser(c, user)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Writer.WriteHeader(http.StatusNoContent)
}

package server

import (
	"encoding/base32"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/securecookie"

	"github.com/drone/drone/model"
	"github.com/drone/drone/store"
)

func GetUsers(c *gin.Context) {
	users, err := store.GetUserList(c)
	if err != nil {
		c.String(500, "Error getting user list. %s", err)
		return
	}
	c.JSON(200, users)
}

func GetUser(c *gin.Context) {
	user, err := store.GetUserLogin(c, c.Param("login"))
	if err != nil {
		c.String(404, "Cannot find user. %s", err)
		return
	}
	c.JSON(200, user)
}

func PatchUser(c *gin.Context) {
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
	user.Active = in.Active

	err = store.UpdateUser(c, user)
	if err != nil {
		c.AbortWithStatus(http.StatusConflict)
		return
	}

	c.JSON(http.StatusOK, user)
}

func PostUser(c *gin.Context) {
	in := &model.User{}
	err := c.Bind(in)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	user := &model.User{
		Active: true,
		Login:  in.Login,
		Email:  in.Email,
		Avatar: in.Avatar,
		Hash: base32.StdEncoding.EncodeToString(
			securecookie.GenerateRandomKey(32),
		),
	}
	if err = store.CreateUser(c, user); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, user)
}

func DeleteUser(c *gin.Context) {
	user, err := store.GetUserLogin(c, c.Param("login"))
	if err != nil {
		c.String(404, "Cannot find user. %s", err)
		return
	}
	if err = store.DeleteUser(c, user); err != nil {
		c.String(500, "Error deleting user. %s", err)
		return
	}
	c.String(200, "")
}

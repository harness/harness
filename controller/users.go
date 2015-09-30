package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/CiscoCloud/drone/model"
	"github.com/CiscoCloud/drone/router/middleware/context"
	"github.com/CiscoCloud/drone/router/middleware/session"
	"github.com/CiscoCloud/drone/shared/crypto"
)

func GetUsers(c *gin.Context) {
	db := context.Database(c)
	users, err := model.GetUserList(db)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.IndentedJSON(http.StatusOK, users)
}

func GetUser(c *gin.Context) {
	db := context.Database(c)
	user, err := model.GetUserLogin(db, c.Param("login"))
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.IndentedJSON(http.StatusOK, user)
}

func PatchUser(c *gin.Context) {
	me := session.User(c)
	db := context.Database(c)
	in := &model.User{}
	err := c.Bind(in)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	user, err := model.GetUserLogin(db, c.Param("login"))
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

	err = model.UpdateUser(db, user)
	if err != nil {
		c.AbortWithStatus(http.StatusConflict)
		return
	}

	c.IndentedJSON(http.StatusOK, user)
}

func PostUser(c *gin.Context) {
	db := context.Database(c)
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

	err = model.CreateUser(db, user)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.IndentedJSON(http.StatusOK, user)
}

func DeleteUser(c *gin.Context) {
	me := session.User(c)
	db := context.Database(c)

	user, err := model.GetUserLogin(db, c.Param("login"))
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// cannot delete self
	if me.ID == user.ID {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	err = model.DeleteUser(db, user)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Writer.WriteHeader(http.StatusNoContent)
}

package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/drone/drone/model"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/shared/crypto"
	"github.com/drone/drone/store"
)

// swagger:route GET /users user getUserList
//
// Get the list of all registered users.
//
//     Responses:
//       200: user
//
func GetUsers(c *gin.Context) {
	users, err := store.GetUserList(c)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Error getting user list. %s", err))
		return
	}

	c.JSON(http.StatusOK, users)
}

// swagger:route GET /users/{login} user getUserLogin
//
// Get the user with the matching login.
//
//     Responses:
//       200: user
//
func GetUser(c *gin.Context) {
	user, err := store.GetUserLogin(c, c.Param("login"))
	if err != nil {
		c.AbortWithError(http.StatusNotFound, fmt.Errorf("Cannot find user. %s", err))
		return
	}

	c.JSON(http.StatusOK, user)
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

	if session.User(c).ID == user.ID {
		c.AbortWithError(422, fmt.Errorf("Users can't deactivate or change admin rights themselfs"))
		return
	}

	user.Admin = in.Admin
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
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if in.Login == "" {
		c.AbortWithError(422, fmt.Errorf("User's login can't be empty"))
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
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// swagger:route DELETE /users/{login} user deleteUserLogin
//
// Delete the user with the matching login.
//
//     Responses:
//       200: user
//
func DeleteUser(c *gin.Context) {
	user, err := store.GetUserLogin(c, c.Param("login"))
	if err != nil {
		c.AbortWithError(http.StatusNotFound, fmt.Errorf("Cannot find user. %s", err))
		return
	}

	// User can't delete itself.
	if user.ID == session.User(c).ID {
		c.AbortWithError(422, fmt.Errorf("Users can't delete themselfs"))
		return
	}

	if err = store.DeleteUser(c, user); err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Error deleting user. %s", err))
		return
	}

	c.String(http.StatusOK, "")
}

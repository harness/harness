package server

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/ungerik/go-gravatar"

	common "github.com/drone/drone/pkg/types"
)

// GetUserCurr accepts a request to retrieve the
// currently authenticated user from the datastore
// and return in JSON format.
//
//     GET /api/user
//
func GetUserCurr(c *gin.Context) {
	c.JSON(200, ToUser(c))
}

// PutUserCurr accepts a request to update the currently
// authenticated User profile.
//
//     PUT /api/user
//
func PutUserCurr(c *gin.Context) {
	store := ToDatastore(c)
	user := ToUser(c)

	in := &common.User{}
	if !c.BindWith(in, binding.JSON) {
		return
	}
	user.Email = in.Email
	user.Gravatar = gravatar.Hash(in.Email)
	err := store.SetUser(user)
	if err != nil {
		c.Fail(400, err)
	} else {
		c.JSON(200, user)
	}
}

// GetUserRepos accepts a request to get the currently
// authenticated user's repository list from the datastore,
// encoded and returned in JSON format.
//
//     GET /api/user/repos
//
func GetUserRepos(c *gin.Context) {
	store := ToDatastore(c)
	user := ToUser(c)
	repos, err := store.RepoList(user)
	if err != nil {
		c.Fail(400, err)
	} else {
		c.JSON(200, &repos)
	}
}

// GetUserTokens accepts a request to get the currently
// authenticated user's token list from the datastore,
// encoded and returned in JSON format.
//
//     GET /api/user/tokens
//
func GetUserTokens(c *gin.Context) {
	store := ToDatastore(c)
	user := ToUser(c)
	tokens, err := store.TokenList(user)
	if err != nil {
		c.Fail(400, err)
	} else {
		c.JSON(200, &tokens)
	}
}

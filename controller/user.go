package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/drone/drone/cache"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/shared/token"
	"github.com/drone/drone/store"
)

func GetSelf(c *gin.Context) {
	c.IndentedJSON(200, session.User(c))
}

func GetFeed(c *gin.Context) {
	user := session.User(c)

	// get the repository list from the cache
	repos, err := cache.GetRepos(c, user)
	if err != nil {
		c.String(400, err.Error())
		return
	}

	feed, err := store.GetUserFeed(c, repos)
	if err != nil {
		c.String(400, err.Error())
		return
	}
	c.JSON(200, feed)
}

func GetRepos(c *gin.Context) {
	user := session.User(c)

	repos, err := cache.GetRepos(c, user)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// for each repository in the remote system we get
	// the intersection of those repostiories in Drone
	repos_, err := store.GetRepoListOf(c, repos)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.IndentedJSON(http.StatusOK, repos_)
}

func GetRemoteRepos(c *gin.Context) {
	user := session.User(c)

	repos, err := cache.GetRepos(c, user)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.IndentedJSON(http.StatusOK, repos)
}

func PostToken(c *gin.Context) {
	user := session.User(c)

	token := token.New(token.UserToken, user.Login)
	tokenstr, err := token.Sign(user.Hash)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	} else {
		c.String(http.StatusOK, tokenstr)
	}
}

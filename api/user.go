package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/drone/drone/cache"
	"github.com/drone/drone/model"
	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/shared/crypto"
	"github.com/drone/drone/shared/token"
	"github.com/drone/drone/store"
)

// swagger:route GET /user user getUser
//
// Get the currently authenticated user.
//
//     Responses:
//       200: user
//
func GetSelf(c *gin.Context) {
	c.JSON(200, session.User(c))
}

// swagger:route GET /user/feed user getUserFeed
//
// Get the currently authenticated user's build feed.
//
//     Responses:
//       200: feed
//
func GetFeed(c *gin.Context) {
	repos, err := cache.GetRepos(c, session.User(c))
	if err != nil {
		c.String(500, "Error fetching repository list. %s", err)
		return
	}

	feed, err := store.GetUserFeed(c, repos)
	if err != nil {
		c.String(500, "Error fetching feed. %s", err)
		return
	}
	c.JSON(200, feed)
}

// swagger:route GET /user/repos user getUserRepos
//
// Get the currently authenticated user's active repository list.
//
//     Responses:
//       200: repos
//
func GetRepos(c *gin.Context) {
	repos, err := cache.GetRepos(c, session.User(c))
	if err != nil {
		c.String(500, "Error fetching repository list. %s", err)
		return
	}

	repos_, err := store.GetRepoListOf(c, repos)
	if err != nil {
		c.String(500, "Error fetching repository list. %s", err)
		return
	}
	c.JSON(http.StatusOK, repos_)
}

func GetRemoteRepos(c *gin.Context) {
	repos, err := cache.GetRepos(c, session.User(c))
	if err != nil {
		c.String(500, "Error fetching repository list. %s", err)
		return
	}
	c.JSON(http.StatusOK, repos)
}

func PostToken(c *gin.Context) {
	user := session.User(c)

	token := token.New(token.UserToken, user.Login)
	tokenstr, err := token.Sign(user.Hash)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.String(http.StatusOK, tokenstr)
}

func DeleteToken(c *gin.Context) {
	user := session.User(c)
	user.Hash = crypto.Rand()
	if err := store.UpdateUser(c, user); err != nil {
		c.String(500, "Error revoking tokens. %s", err)
		return
	}

	token := token.New(token.UserToken, user.Login)
	tokenstr, err := token.Sign(user.Hash)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.String(http.StatusOK, tokenstr)
}

// swagger:response user
type userResp struct {
	// in: body
	Body model.User
}

// swagger:response users
type usersResp struct {
	// in: body
	Body []model.User
}

// swagger:response feed
type feedResp struct {
	// in: body
	Body []model.Feed
}

// swagger:response repos
type reposResp struct {
	// in: body
	Body []model.Repo
}

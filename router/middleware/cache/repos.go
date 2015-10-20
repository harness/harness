package cache

import (
	"github.com/drone/drone/cache"
	"github.com/drone/drone/model"
	"github.com/gin-gonic/gin"
)

// Repos is a middleware function that attempts to cache the
// user's list of remote repositories (ie in GitHub) to minimize
// remote calls that might be expensive, slow or rate-limited.
func Repos(c *gin.Context) {
	var user, _ = c.Get("user")

	if user == nil {
		c.Next()
		return
	}

	// if the item already exists in the cache
	// we can continue the middleware chain and
	// exit afterwards.
	v := cache.GetRepos(c, user.(*model.User))
	if v != nil {
		c.Set("repos", v)
		c.Next()
		return
	}

	// otherwise, if the item isn't cached we execute
	// the middleware chain and then cache the permissions
	// after the request is processed.
	c.Next()

	repos, ok := c.Get("repos")
	if ok {
		cache.SetRepos(c,
			user.(*model.User),
			repos.([]*model.RepoLite),
		)
	}
}

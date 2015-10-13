package cache

import (
	"fmt"

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

	key := fmt.Sprintf("repos/%s",
		user.(*model.User).Login,
	)

	// if the item already exists in the cache
	// we can continue the middleware chain and
	// exit afterwards.
	v, _ := get(key)
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
		set(key, repos, 86400) // 24 hours
	}
}

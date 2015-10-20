package cache

import (
	"github.com/drone/drone/cache"
	"github.com/drone/drone/model"
	"github.com/gin-gonic/gin"
)

const permKey = "perm"

// Perms is a middleware function that attempts to cache the
// user's remote rempository permissions (ie in GitHub) to minimize
// remote calls that might be expensive, slow or rate-limited.
func Perms(c *gin.Context) {
	var (
		owner   = c.Param("owner")
		name    = c.Param("name")
		user, _ = c.Get("user")
	)

	if user == nil {
		c.Next()
		return
	}

	// if the item already exists in the cache
	// we can continue the middleware chain and
	// exit afterwards.
	v := cache.GetPerms(c,
		user.(*model.User),
		owner,
		name,
	)
	if v != nil {
		c.Set("perm", v)
		c.Next()
		return
	}

	// otherwise, if the item isn't cached we execute
	// the middleware chain and then cache the permissions
	// after the request is processed.
	c.Next()

	perm, ok := c.Get("perm")
	if ok {
		cache.SetPerms(c,
			user.(*model.User),
			perm.(*model.Perm),
			owner,
			name,
		)
	}
}

package cache

import (
	"testing"

	"github.com/drone/drone/cache"
	"github.com/drone/drone/model"
	"github.com/franela/goblin"
	"github.com/gin-gonic/gin"
)

func TestPermCache(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Perm Cache", func() {

		var c *gin.Context
		g.BeforeEach(func() {
			c = new(gin.Context)
			cache.ToContext(c, cache.Default())
		})

		g.It("should skip when no user session", func() {
			c.Params = gin.Params{
				gin.Param{Key: "owner", Value: "octocat"},
				gin.Param{Key: "name", Value: "hello-world"},
			}

			Perms(c)

			_, ok := c.Get("perm")
			g.Assert(ok).IsFalse()
		})

		g.It("should get perms from cache", func() {
			c.Params = gin.Params{
				gin.Param{Key: "owner", Value: "octocat"},
				gin.Param{Key: "name", Value: "hello-world"},
			}
			c.Set("user", fakeUser)
			cache.SetPerms(c, fakeUser, fakePerm, "octocat", "hello-world")

			Perms(c)

			perm, ok := c.Get("perm")
			g.Assert(ok).IsTrue()
			g.Assert(perm).Equal(fakePerm)
		})

	})
}

var fakePerm = &model.Perm{
	Pull:  true,
	Push:  true,
	Admin: true,
}

var fakeUser = &model.User{
	Login: "octocat",
}

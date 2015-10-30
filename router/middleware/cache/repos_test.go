package cache

import (
	"testing"

	"github.com/drone/drone/cache"
	"github.com/drone/drone/model"
	"github.com/franela/goblin"
	"github.com/gin-gonic/gin"
)

func TestReposCache(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Repo List Cache", func() {

		var c *gin.Context
		g.BeforeEach(func() {
			c = new(gin.Context)
			cache.ToContext(c, cache.Default())
		})

		g.It("should skip when no user session", func() {
			Perms(c)

			_, ok := c.Get("perm")
			g.Assert(ok).IsFalse()
		})

		g.It("should get repos from cache", func() {
			c.Set("user", fakeUser)
			cache.SetRepos(c, fakeUser, fakeRepos)

			Repos(c)

			repos, ok := c.Get("repos")
			g.Assert(ok).IsTrue()
			g.Assert(repos).Equal(fakeRepos)
		})

	})
}

var fakeRepos = []*model.RepoLite{
	{Owner: "octocat", Name: "hello-world"},
}

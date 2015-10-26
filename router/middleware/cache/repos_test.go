package cache

import (
	"testing"

	"github.com/CiscoCloud/drone/model"
	"github.com/franela/goblin"
	"github.com/gin-gonic/gin"
)

func TestReposCache(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Repo List Cache", func() {

		g.BeforeEach(func() {
			cache.Purge()
		})

		g.It("should skip when no user session", func() {
			c := &gin.Context{}

			Perms(c)

			_, ok := c.Get("perm")
			g.Assert(ok).IsFalse()
		})

		g.It("should get repos from cache", func() {
			c := &gin.Context{}
			c.Set("user", fakeUser)
			set("repos/octocat", fakeRepos, 999)

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

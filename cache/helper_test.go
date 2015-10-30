package cache

import (
	"testing"

	"github.com/drone/drone/model"
	"github.com/franela/goblin"
	"github.com/gin-gonic/gin"
)

func TestHelper(t *testing.T) {

	g := goblin.Goblin(t)
	g.Describe("Cache helpers", func() {

		var c *gin.Context
		g.BeforeEach(func() {
			c = new(gin.Context)
			ToContext(c, Default())
		})

		g.It("Should set and get permissions", func() {
			SetPerms(c, fakeUser, fakePerm, "octocat", "Spoon-Knife")

			v := GetPerms(c, fakeUser, "octocat", "Spoon-Knife")
			g.Assert(v).Equal(fakePerm)
		})

		g.It("Should return nil if permissions if not found", func() {
			v := GetPerms(c, fakeUser, "octocat", "Spoon-Knife")
			g.Assert(v == nil).IsTrue()
		})

		g.It("Should set and get repositories", func() {
			SetRepos(c, fakeUser, fakeRepos)

			v := GetRepos(c, fakeUser)
			g.Assert(v).Equal(fakeRepos)
		})

		g.It("Should return nil if repositories not found", func() {
			v := GetRepos(c, fakeUser)
			g.Assert(v == nil).IsTrue()
		})
	})
}

var (
	fakeUser  = &model.User{Login: "octocat"}
	fakePerm  = &model.Perm{true, true, true}
	fakeRepos = []*model.RepoLite{
		{Owner: "octocat", Name: "Hello-World"},
		{Owner: "octocat", Name: "hello-world"},
		{Owner: "octocat", Name: "Spoon-Knife"},
	}
)

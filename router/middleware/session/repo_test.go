package session

import (
	"os"
	"testing"

	"github.com/drone/drone/model"
	"github.com/franela/goblin"
	"github.com/gin-gonic/gin"
)

func TestSetPerm(t *testing.T) {
	g := goblin.Goblin(t)
	g.Describe("SetPerm", func() {
		g.BeforeEach(func() {
			os.Unsetenv("PUBLIC_MODE")
		})
		g.It("Should set pull to false (private repo, user not logged in)", func() {
			c := gin.Context{}
			c.Set("repo", &model.Repo{
				IsPrivate: true,
			})
			SetPerm()(&c)
			v, ok := c.Get("perm")
			g.Assert(ok).IsTrue("perm was not set")
			p, ok := v.(*model.Perm)
			g.Assert(ok).IsTrue("perm was the wrong type")
			g.Assert(p.Pull).IsFalse("pull should be false")
		})
		g.It("Should set pull to true (private repo, user not logged in, public mode)", func() {
			os.Setenv("PUBLIC_MODE", "true")
			c := gin.Context{}
			c.Set("repo", &model.Repo{
				IsPrivate: true,
			})
			SetPerm()(&c)
			v, ok := c.Get("perm")
			g.Assert(ok).IsTrue("perm was not set")
			p, ok := v.(*model.Perm)
			g.Assert(ok).IsTrue("perm was the wrong type")
			g.Assert(p.Pull).IsTrue("pull should be true")
		})
	})
}

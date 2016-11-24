package session

import (
	"testing"

	"github.com/drone/drone/cache"
	"github.com/drone/drone/model"
	"github.com/franela/goblin"
	"github.com/gin-gonic/gin"
)

func TestTeamPerm(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("TeamPerm", func() {

		var c *gin.Context
		g.BeforeEach(func() {
			c = new(gin.Context)
			cache.ToContext(c, cache.Default())
		})

		g.It("Should set admin to false (user not logged in)", func() {
			p := TeamPerm(c)
			g.Assert(p.Admin).IsFalse("admin should be false")
		})
		g.It("Should set admin to true (user is DRONE_ADMIN)", func() {
			// Set DRONE_ADMIN user
			c.Set("user", fakeUserAdmin)

			p := TeamPerm(c)
			g.Assert(p.Admin).IsTrue("admin should be false")
		})
		g.It("Should set admin to false (user logged in, not owner of org)", func() {
			// Set fake org
			params := gin.Params{
				gin.Param{
					Key: "team",
					Value: "test_org",
				},
			}
			c.Params = params

			// Set cache to show user does not Owner/Admin
			cache.Set(c, "perms:octocat:test_org", fakeTeamPerm)

			// Set User
			c.Set("user", fakeUser)

			p := TeamPerm(c)
			g.Assert(p.Admin).IsFalse("admin should be false")
		})
		g.It("Should set admin to true (user logged in, owner of org)", func() {
			// Set fake org
			params := gin.Params{
				gin.Param{
					Key: "team",
					Value: "test_org",
				},
			}
			c.Params = params

			// Set cache to show user is Owner/Admin
			cache.Set(c, "perms:octocat:test_org", fakeTeamPermAdmin)

			// Set User
			c.Set("user", fakeUser)

			p := TeamPerm(c)
			g.Assert(p.Admin).IsTrue("admin should be true")
		})
	})
}

var (
	fakeUserAdmin = &model.User{
		Login: "octocatAdmin",
		Token: "cfcd2084",
		Admin: true,
	}

	fakeUser = &model.User{
		Login: "octocat",
		Token: "cfcd2084",
		Admin: false,
	}

	fakeTeamPermAdmin = &model.Perm{
		Admin: true,
	}

	fakeTeamPerm = &model.Perm{
		Admin: false,
	}
)

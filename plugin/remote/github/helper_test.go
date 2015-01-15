package github

import (
	"testing"

	"github.com/drone/drone/plugin/remote/github/testdata"
	"github.com/franela/goblin"
)

func Test_Helper(t *testing.T) {
	// setup a dummy github server
	var server = testdata.NewServer()
	defer server.Close()

	var client = NewClient(server.URL, "sekret", false)

	g := goblin.Goblin(t)
	g.Describe("GitHub Helper Functions", func() {

		g.It("Should Get a User")
		g.It("Should Get a User Primary Email")
		g.It("Should Get a User + Primary Email")

		g.It("Should Get a list of Orgs", func() {
			var orgs, err = GetOrgs(client)
			g.Assert(err == nil).IsTrue()
			g.Assert(len(orgs)).Equal(1)
			g.Assert(*orgs[0].Login).Equal("octocats-inc")
		})

		g.It("Should Get a list of User Repos")
		g.It("Should Get a list of Org Repos")
		g.It("Should Get a list of All Repos")
		g.It("Should Get a Repo Key")
		g.It("Should Get a Repo Hook")
		g.It("Should Create a Repo Key")
		g.It("Should Create a Repo Hook")
		g.It("Should Create or Update a Repo Key")
		g.It("Should Create or Update a Repo Hook")
		g.It("Should Get a Repo File")

		g.Describe("UserBelongsToOrg", func() {
			g.It("Should confirm user does belong to 'octocats-inc' org", func() {
				var requiredOrgs = []string{"one", "octocats-inc", "two"}
				var member, err = UserBelongsToOrg(client, requiredOrgs)
				g.Assert(err == nil).IsTrue()
				g.Assert(member).IsTrue()
			})

			g.It("Should confirm user not does belong to 'octocats-inc' org", func() {
				var requiredOrgs = []string{"one", "two"}
				var member, err = UserBelongsToOrg(client, requiredOrgs)
				g.Assert(err == nil).IsTrue()
				g.Assert(member).IsFalse()
			})
		})
	})
}
